package seoengine

import (
	"context"
	"fmt"
	"regexp"
	"strings"

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

	lang := in.Language
	if lang == "" {
		lang = "pt"
	}
	keyword := deriveKeyword(in.Title)
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
		Content:         content,
		MetaDescription: buildMetaDescription(in.Title, content, keyword, lang),
		InternalLinks:   internalI,
		ExternalLinks:   externalI,
		GapReport:       gap,
		TopicAuthority:  topicAuth,
		Suggestions:     suggestions,
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
