package seoengine

import (
	"context"
	"fmt"
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
		Content:        content,
		InternalLinks:  internalI,
		ExternalLinks:  externalI,
		GapReport:      gap,
		TopicAuthority: topicAuth,
		Suggestions:    suggestions,
	}, nil
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