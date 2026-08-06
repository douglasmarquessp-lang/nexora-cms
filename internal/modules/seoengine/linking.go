package seoengine

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"

	"nexora/internal/ai"
)

// Internal link relevance weights — deterministic scoring across four
// dimensions: keyword, subject/entity, category, intent.
const (
	linkWeightKeyword  = 0.35
	linkWeightSubject  = 0.35
	linkWeightCategory = 0.15
	linkWeightIntent   = 0.15
)

// InternalLinkCandidate is a ranked candidate for an automatic internal link.
type InternalLinkCandidate struct {
	PostID     uuid.UUID `json:"post_id"`
	Title      string    `json:"title"`
	Slug       string    `json:"slug"`
	AnchorText string    `json:"anchor_text"`
	Keyword    float64   `json:"keyword_score"`
	Subject    float64   `json:"subject_score"`
	Category   float64   `json:"category_score"`
	Intent     float64   `json:"intent_score"`
	Relevance  float64   `json:"relevance"`
}

// ExternalLinkCandidate is a vetted external link (official/high reliability).
type ExternalLinkCandidate struct {
	URL         string  `json:"url"`
	Title       string  `json:"title"`
	Domain      string  `json:"domain"`
	Reliability int     `json:"reliability"`
	Label       string  `json:"label"`
	Relevance   float64 `json:"relevance"`
}

var informationalIntentRE = regexp.MustCompile(`(?i)\b(como|what|how|why|o que|guia|guide|tutorial|aprenda|learn|dicas|tips|melhor|best|top)\b`)
var transactionalIntentRE = regexp.MustCompile(`(?i)\b(comprar|buy|preço|price|preco|contratar|hire|assinatura|subscription|download|coupon|oferta|deal)\b`)
var navigationalIntentRE = regexp.MustCompile(`(?i)\b(acessar|login|site oficial|official site|download oficial|baixar)\b`)

// detectIntent returns the dominant search intent of a title.
func detectIntent(title string) string {
	if transactionalIntentRE.MatchString(title) {
		return "transactional"
	}
	if navigationalIntentRE.MatchString(title) {
		return "navigational"
	}
	if informationalIntentRE.MatchString(title) {
		return "informational"
	}
	return "informational"
}

// ScoreInternalLinkCandidate ranks a candidate post against the source
// article across the four dimensions. All scores are 0-1; the composite is
// 0-100. Pure function, no DB, no randomness.
func ScoreInternalLinkCandidate(sourceTitle, sourceContent, sourceKeyword, sourceCategory string, candidateTitle, candidateSlug, candidateCategory string) InternalLinkCandidate {
	keyword := keywordCoverage(candidateTitle+" "+candidateSlug, sourceKeyword)

	subject := 0.0
	srcTokens := tokenize(sourceContent + " " + sourceTitle)
	canTokens := tokenize(candidateTitle)
	if len(srcTokens) > 0 && len(canTokens) > 0 {
		overlap := 0
		for _, t := range canTokens {
			if len(t) < 4 {
				continue
			}
			for _, s := range srcTokens {
				if s == t {
					overlap++
					break
				}
			}
		}
		subject = float64(overlap) / float64(len(canTokens))
	}

	category := 0.0
	if sourceCategory != "" && candidateCategory != "" {
		src := strings.ToLower(strings.TrimSpace(sourceCategory))
		can := strings.ToLower(strings.TrimSpace(candidateCategory))
		if src == can {
			category = 1.0
		} else if strings.Contains(src, can) || strings.Contains(can, src) {
			category = 0.7
		}
	}

	intent := 0.0
	if detectIntent(sourceTitle) == detectIntent(candidateTitle) {
		intent = 1.0
	}

	relevance := clampScore((keyword*linkWeightKeyword + subject*linkWeightSubject + category*linkWeightCategory + intent*linkWeightIntent) * 100)
	return InternalLinkCandidate{
		Title:      candidateTitle,
		Slug:       candidateSlug,
		AnchorText: deriveKeyword(candidateTitle),
		Keyword:    round2(keyword * 100),
		Subject:    round2(subject * 100),
		Category:   round2(category * 100),
		Intent:     round2(intent * 100),
		Relevance:  round2(relevance),
	}
}

// SelectInternalLinks queries the site's published posts and returns the best
// internal link candidates (composite score >= minScore, capped at maxLinks).
func (s *Service) SelectInternalLinks(ctx context.Context, siteID uuid.UUID, sourcePostID *uuid.UUID, sourceTitle, sourceContent, sourceKeyword, sourceCategory string, minScore, maxLinks int) ([]InternalLinkCandidate, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	if minScore <= 0 {
		minScore = s.internalLinkMinScore
	}
	if maxLinks <= 0 {
		maxLinks = s.internalLinkMax
	}

	exclude := "00000000-0000-0000-0000-000000000000"
	if sourcePostID != nil {
		exclude = sourcePostID.String()
	}

	rows, err := p.Query(ctx,
		`SELECT id, COALESCE(title,''), COALESCE(slug,'') FROM posts
		 WHERE site_id = $1 AND deleted_at IS NULL AND status = 'published' AND id <> $2::uuid
		 ORDER BY created_at DESC LIMIT 100`,
		siteID, exclude,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts for internal linking: %w", err)
	}
	defer rows.Close()

	candidates := []InternalLinkCandidate{}
	for rows.Next() {
		var postID uuid.UUID
		var title, slug string
		if err := rows.Scan(&postID, &title, &slug); err != nil {
			return nil, fmt.Errorf("failed to scan linking candidate: %w", err)
		}
		// Category matching is computed from title/subject heuristics when the
		// DB query does not join categories; category score stays 0 unless the
		// caller provides sourceCategory and the candidate signals it.
		scored := ScoreInternalLinkCandidate(sourceTitle, sourceContent, sourceKeyword, sourceCategory, title, slug, "")
		scored.PostID = postID
		candidates = append(candidates, scored)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Relevance > candidates[j].Relevance
	})

	selected := []InternalLinkCandidate{}
	for _, c := range candidates {
		if c.Relevance < float64(minScore) || c.Slug == "" {
			continue
		}
		selected = append(selected, c)
		if len(selected) >= maxLinks {
			break
		}
	}
	return selected, nil
}

// isCompetitorDomain reports whether the domain is in the configured blocklist.
func (s *Service) isCompetitorDomain(domain string) bool {
	for _, d := range s.competitorDomains {
		if d == "" {
			continue
		}
		if strings.EqualFold(d, domain) || strings.HasSuffix(domain, "."+d) {
			return true
		}
	}
	return false
}

// SelectExternalLinks picks external links from the site's research sources:
// only domains with reliability >= minReliability, never competitor domains.
func (s *Service) SelectExternalLinks(ctx context.Context, siteID uuid.UUID, topic string, minReliability, maxLinks int) ([]ExternalLinkCandidate, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	if minReliability <= 0 {
		minReliability = s.externalLinkMinReliability
	}
	if maxLinks <= 0 {
		maxLinks = 3
	}

	rows, err := p.Query(ctx,
		`SELECT COALESCE(url,''), COALESCE(title,''), COALESCE(domain,''), COALESCE(reliability_score,0)
		 FROM research_sources
		 WHERE site_id = $1 AND COALESCE(url,'') <> ''
		 ORDER BY COALESCE(reliability_score,0) DESC, created_at DESC LIMIT 50`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list research sources for external linking: %w", err)
	}
	defer rows.Close()

	keywords := tokenize(topic)
	candidates := []ExternalLinkCandidate{}
	seen := map[string]bool{}
	for rows.Next() {
		var url, title, domain string
		var reliability int
		if err := rows.Scan(&url, &title, &domain, &reliability); err != nil {
			return nil, fmt.Errorf("failed to scan external link source: %w", err)
		}
		if seen[url] {
			continue
		}
		seen[url] = true
		if reliability < minReliability || s.isCompetitorDomain(domain) {
			continue
		}
		relevance := 0.0
		if len(keywords) > 0 {
			hay := strings.ToLower(title + " " + domain)
			hits := 0
			for _, k := range keywords {
				if len(k) < 4 {
					continue
				}
				if strings.Contains(hay, k) {
					hits++
				}
			}
			relevance = float64(hits) / float64(len(keywords))
		}
		_, label := ai.ReliabilityOfDomain(domain)
		candidates = append(candidates, ExternalLinkCandidate{
			URL:         url,
			Title:       title,
			Domain:      domain,
			Reliability: reliability,
			Label:       label,
			Relevance:   round2(clampScore(relevance * 100)),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Reliability == candidates[j].Reliability {
			return candidates[i].Relevance > candidates[j].Relevance
		}
		return candidates[i].Reliability > candidates[j].Reliability
	})
	if len(candidates) > maxLinks {
		candidates = candidates[:maxLinks]
	}
	return candidates, nil
}
