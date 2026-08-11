package seoengine

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"nexora/internal/pkg/sitelang"
	"nexora/internal/modules/publisher"
)

// EnhanceBeforePublish implements publisher.ContentEnhancer. It:
//  1. runs the content gap analysis and (optionally) AI-fills missing facts,
//  2. selects the best internal links (keyword + subject + category + intent),
//  3. selects reliable external links (never competitors),
//  4. computes topic authority coverage.
//
// Fails open: if anything goes wrong the content is returned unchanged with a
// nil error so publishing is never blocked by the enrichment layer.
func (s *Service) EnhanceBeforePublish(ctx context.Context, in publisher.ContentEnhancerInput) (*publisher.ContentEnhancement, error) {
	if s == nil {
		return nil, nil
	}

	lang := sitelang.Resolve(in.SiteID, in.Language)
	keyword := FocusKeyword(in.Title, in.Content)
	if in.Keyword != "" {
		keyword = in.Keyword
	}

	gap := DetectContentGaps(in.Content)
	s.FillContentGapsAI(ctx, in.SiteID, in.Title, gap)

	internal, err := s.SelectInternalLinks(ctx, in.SiteID, in.PostID, in.Title, in.Content, keyword, in.Category, 0, 0)
	if err != nil {
		s.log.Warn("enhance: internal link selection failed, continuing", "error", err)
		internal = nil
	}
	if len(internal) == 0 {
		// First article on a fresh site: no prior content to link to. No fake
		// links are ever generated; the analyzer scores internal_links with
		// its baseline (30/100, 10% weight) and the remaining criteria must
		// compensate for the article to pass the publish gate.
		s.log.Warn("enhance: no internal link candidates available",
			"site_id", in.SiteID, "title", in.Title,
			"impact", "internal_links_score=30; weight=10%",
		)
	}

	external, err := s.SelectExternalLinks(ctx, in.SiteID, in.Title, 0, 3)
	if err != nil {
		s.log.Warn("enhance: external link selection failed, continuing", "error", err)
		external = nil
	}

	topicAuth, err := s.TopicAuthority(ctx, in.SiteID, in.Title)
	if err != nil {
		s.log.Warn("enhance: topic authority failed, continuing", "error", err)
		topicAuth = nil
	}

	content := in.Content
	// Featured image: if the caller already knows one, embed it (honest alt
	// text derived from the title when none was given). Otherwise ask the
	// image provider (Pexels) for a real photograph — never block publishing
	// on image fetching: on any error the article proceeds without an image.
	featuredURL := strings.TrimSpace(in.FeaturedImageURL)
	featuredAlt := strings.TrimSpace(in.FeaturedImageAlt)
	if featuredURL == "" && s.imageProvider != nil {
		img, ierr := s.imageProvider.SearchImage(ctx, keyword)
		if ierr == nil && img != nil && img.URL != "" {
			featuredURL = img.URL
			featuredAlt = strings.TrimSpace(img.Alt)
			if featuredAlt == "" {
				featuredAlt = deriveImageAlt(in.Title)
			}
			content = embedFeaturedImage(content, img, featuredAlt, lang)
		} else {
			s.log.Warn("enhance: image provider failed, publishing without image", "error", ierr)
		}
	} else if featuredURL != "" {
		// Caller-provided image (e.g. media library): it must still land in
		// the ARTICLE (as a <figure> with ALT), not only in featured_image_url
		// metadata — otherwise the analyzer's analyzeImages() never sees it
		// and the images dimension sits at 30/100 at publish time.
		if featuredAlt == "" {
			featuredAlt = deriveImageAlt(in.Title)
		}
		content = embedPlainImage(content, featuredURL, featuredAlt)
	}
	if len(internal) > 0 {
		content = appendRelatedLinks(content, internal, lang)
	}
	// External sources: only append a References section when the content does
	// not already carry real https links (the AI pipeline appends research
	// sources itself) — never duplicate links across layers.
	if len(external) > 0 && !hasExternalLinks(content) {
		content = appendExternalSources(content, external, lang)
	}

	internalI := make([]interface{}, 0, len(internal))
	for i := range internal {
		internalI = append(internalI, internal[i])
	}
	externalI := make([]interface{}, 0, len(external))
	for i := range external {
		externalI = append(externalI, external[i])
	}

	suggestions := []string{}
	if gap != nil && len(gap.Missing) > 0 {
		for _, su := range gap.Suggestions {
			suggestions = append(suggestions, su)
		}
	}
	for _, c := range internal {
		suggestions = append(suggestions, fmt.Sprintf("Link interno sugerido: %s (/[%s])", c.Title, c.Slug))
	}

	return &publisher.ContentEnhancement{
		Content:          content,
		MetaDescription:  buildMetaDescription(in.Title, content, keyword, lang),
		Keyword:          keyword,
		InternalLinks:    internalI,
		ExternalLinks:    externalI,
		GapReport:        gap,
		TopicAuthority:   topicAuth,
		Suggestions:      suggestions,
		FeaturedImageURL: featuredURL,
		FeaturedImageAlt: featuredAlt,
	}, nil
}

// hasExternalLinks reports whether the content already contains markdown or
// HTML links to external (http/https) URLs.
func hasExternalLinks(content string) bool {
	for _, m := range markdownLinkRE.FindAllString(content, -1) {
		if strings.Contains(m, "http://") || strings.Contains(m, "https://") {
			return true
		}
	}
	return htmlHrefRE.MatchString(content)
}

// appendExternalSources appends a "Sources"/"Fontes" section with the reliable
// external links (markdown bullets). Deterministic: same candidates → same
// output.
func appendExternalSources(content string, links []ExternalLinkCandidate, lang string) string {
	if len(links) == 0 {
		return content
	}
	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(content))
	sb.WriteString("\n\n")
	if lang == "en" {
		sb.WriteString("## Sources\n")
	} else {
		sb.WriteString("## Fontes\n")
	}
	for _, l := range links {
		if l.URL == "" || strings.Contains(strings.ToLower(l.URL), "javascript:") {
			continue
		}
		title := l.Title
		if title == "" {
			title = l.Domain
		}
		sb.WriteString("- [")
		sb.WriteString(title)
		sb.WriteString("](")
		sb.WriteString(l.URL)
		sb.WriteString(")\n")
	}
	return strings.TrimSpace(sb.String())
}

// buildMetaDescription derives a deterministic meta description from the
// article: first keywords-bearing paragraphs, sentence-greedy up to ~155
// characters. Returns "" when no text is available (an empty meta is scored 0
// by the SEO gate — the description is never faked, just honestly derived).
func buildMetaDescription(title, content, keyword, lang string) string {
	text := stripMarkdown(content)
	if text == "" {
		text = strings.TrimSpace(title)
	}
	if text == "" {
		return ""
	}
	sentences := sentencesFrom(text)
	if len(sentences) == 0 {
		sentences = []string{text}
	}
	var sb strings.Builder
	kw := strings.ToLower(strings.TrimSpace(keyword))
	for _, s := range sentences {
		sep := ""
		if sb.Len() > 0 {
			sep = " "
		}
		line := sep + s
		if len([]rune(sb.String()+line)) > 160 {
			// The first sentence alone exceeds the limit (the markdown H1 and
			// the first paragraph join into a single run). Truncate it at a
			// word boundary instead of abandoning the description — an empty
			// meta is scored 0 by the SEO gate even though text exists.
			if sb.Len() == 0 {
				runes := []rune(line)
				cut := 160
				if cut > len(runes) {
					cut = len(runes)
				}
				chunk := string(runes[:cut])
				if cut < len(runes) {
					if idx := strings.LastIndex(chunk, " "); idx > 60 {
						chunk = chunk[:idx]
					}
				}
				sb.WriteString(chunk)
			}
			break
		}
		sb.WriteString(line)
		// prefer a description that contains the keyword when it appears in
		// an early sentence; otherwise keep collecting up to the limit.
		if kw != "" && len([]rune(sb.String())) >= 120 && strings.Contains(strings.ToLower(sb.String()), kw) {
			break
		}
	}
	result := strings.TrimSpace(sb.String())
	if len([]rune(result)) > 160 {
		runes := []rune(result)
		cut := 160
		if cut > len(runes) {
			cut = len(runes)
		}
		result = string(runes[:cut])
		if strings.HasSuffix(result, " ") {
			result = strings.TrimRight(result, " ")
		}
	}
	return result
}

var (
	markdownLinkRE = regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`)
	htmlHrefRE     = regexp.MustCompile(`(?i)<a[^>]+href=["']?https?://`)
)

// stripMarkdown removes markdown link syntax, headings, emphasis and line
// breaks, returning plain text for meta description derivation.
func stripMarkdown(content string) string {
	content = markdownLinkRE.ReplaceAllString(content, "$1")
	content = regexpHeadRE.ReplaceAllString(content, "")
	content = strings.ReplaceAll(content, "**", "")
	content = strings.ReplaceAll(content, "__", "")
	content = strings.TrimSpace(content)
	lines := strings.Split(content, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			cleaned = append(cleaned, l)
		}
	}
	return strings.Join(cleaned, " ")
}

var (
	regexpHeadRE = regexp.MustCompile(`(?m)^#{1,6}\s+`)
)

// sentencesFrom splits text into trimmed sentences on [.!?] boundaries.
func sentencesFrom(text string) []string {
	parts := sentenceSplitRE.Split(text, -1)
	sentences := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			sentences = append(sentences, p)
		}
	}
	return sentences
}

// appendRelatedLinks appends a "Related reading" section (in the article's
// language) with the selected internal links. Deterministic: same candidates →
// same output.
func appendRelatedLinks(content string, links []InternalLinkCandidate, lang string) string {
	if len(links) == 0 {
		return content
	}
	var sb strings.Builder
	sb.WriteString(content)
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		sb.WriteString("\n")
	}
	sb.WriteString("\n\n")
	if lang == "en" {
		sb.WriteString("### Related reading\n")
	} else {
		sb.WriteString("### Leia também\n")
	}
	for _, l := range links {
		if l.Slug == "" {
			continue
		}
		sb.WriteString("- [")
		sb.WriteString(l.Title)
		sb.WriteString("](/")
		sb.WriteString(l.Slug)
		sb.WriteString(")\n")
	}
	return strings.TrimSpace(sb.String())
}

var _ publisher.ContentEnhancer = (*Service)(nil)

// embedFeaturedImage inserts an HTML <figure> with the real photograph right
// after the first paragraph (or at the top when the content has no paragraph
// boundary). The public site renders article content as HTML, so a native
// <img> tag displays correctly. Attribution is always included (Pexels
// license requirement) with rel="nofollow noopener" on the links.
func embedFeaturedImage(content string, img *PexelsImage, alt, lang string) string {
	if img == nil || strings.TrimSpace(img.URL) == "" {
		return content
	}
	fig := "Photo by"
	if lang == "pt" {
		fig = "Foto de"
	}
	photo := strings.TrimSpace(img.Photographer)
	photoURL := strings.TrimSpace(img.PhotographerURL)
	var sb strings.Builder
	sb.WriteString("<figure>")
	sb.WriteString("<img src=\"")
	sb.WriteString(img.URL)
	sb.WriteString("\" alt=\"")
	sb.WriteString(sanitizeAttr(alt))
	sb.WriteString("\" loading=\"lazy\" />")
	if photo != "" {
		sb.WriteString("<figcaption>")
		sb.WriteString(fig)
		sb.WriteString(" <a href=\"")
		sb.WriteString(photoURL)
		sb.WriteString("\" rel=\"nofollow noopener\" target=\"_blank\">")
		sb.WriteString(sanitizeAttr(photo))
		sb.WriteString("</a> on <a href=\"https://www.pexels.com\" rel=\"nofollow noopener\" target=\"_blank\">Pexels</a></figcaption>")
	}
	sb.WriteString("</figure>")

	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return sb.String()
	}
	if idx := strings.Index(trimmed, "\n\n"); idx > 0 {
		head := strings.TrimSuffix(trimmed[:idx], "\n")
		rest := strings.TrimPrefix(trimmed[idx:], "\n")
		return head + "\n\n" + sb.String() + "\n\n" + strings.TrimSpace(rest)
	}
	return sb.String() + "\n\n" + trimmed
}

// embedPlainImage inserts a <figure> with the caller-provided image (no
// Pexels attribution). Placed right after the first paragraph like
// embedFeaturedImage, so the image is part of the article body and the
// analyzer's images dimension sees it.
func embedPlainImage(content, src, alt string) string {
	src = strings.TrimSpace(src)
	if src == "" {
		return content
	}
	var sb strings.Builder
	sb.WriteString("<figure><img src=\"")
	sb.WriteString(sanitizeAttr(src))
	sb.WriteString("\" alt=\"")
	sb.WriteString(sanitizeAttr(alt))
	sb.WriteString("\" loading=\"lazy\" /></figure>")

	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return sb.String()
	}
	if idx := strings.Index(trimmed, "\n\n"); idx > 0 {
		head := strings.TrimSuffix(trimmed[:idx], "\n")
		rest := strings.TrimPrefix(trimmed[idx:], "\n")
		return head + "\n\n" + sb.String() + "\n\n" + strings.TrimSpace(rest)
	}
	return sb.String() + "\n\n" + trimmed
}

// sanitizeAttr strips characters that would break an HTML attribute.
func sanitizeAttr(s string) string {
	repl := strings.NewReplacer("\"", "&#34;", "<", "&lt;", ">", "&gt;", "\n", " ", "\r", " ")
	return repl.Replace(strings.TrimSpace(s))
}

// deriveImageAlt builds an honest, keyword-free alt text from the article
// title: the cleaned title phrase (punctuation removed, capped at 10 words).
// Never keyword-stuffed — it describes the subject, exactly what the photo
// illustrates.
func deriveImageAlt(title string) string {
	words := tokenize(title)
	picked := []string{}
	for _, w := range words {
		if stopWords[w] || len(w) < 3 {
			continue
		}
		picked = append(picked, w)
		if len(picked) >= 10 {
			break
		}
	}
	if len(picked) == 0 {
		return strings.TrimSpace(title)
	}
	alt := strings.Join(picked, " ")
	if len([]rune(alt)) > 120 {
		runes := []rune(alt)
		alt = string(runes[:117]) + "..."
	}
	r := []rune(alt)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}
