package seoengine

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// TopicalAuthorityReport scores how much the site already covers a topic and
// identifies weak areas.
type TopicalAuthorityReport struct {
	Topic       string  `json:"topic"`
	Coverage    int     `json:"coverage"`              // number of related articles
	Authority   float64 `json:"authority"`             // 0-100 log-scale score
	RelatedArticles []string `json:"related_articles"` // titles of related posts
	Gaps        []string `json:"gaps"`                 // sub-topics with low coverage
}

// TopicAuthorityScore converts topic coverage into a 0-100 authority using a
// log scale: 1 article → ~18, 3 → ~40, 10 → ~68, 25 → ~85, 50+ → 100.
func TopicAuthorityScore(relatedArticles int) float64 {
	if relatedArticles <= 0 {
		return 0
	}
	if relatedArticles >= 50 {
		return 100
	}
	// log2 scale: log2(1+coverage) / log2(51) * 100
	score := math.Log2(float64(relatedArticles)+1) / math.Log2(51) * 100
	return round2(clampScore(score))
}

// TopicAuthority analyzes the site's existing articles about a topic. Related
// posts are matched by keyword/subject overlap with the topic title.
func (s *Service) TopicAuthority(ctx context.Context, siteID uuid.UUID, topic string) (*TopicalAuthorityReport, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	keywords := tokenize(topic)
	rows, err := p.Query(ctx,
		`SELECT COALESCE(title,''), COALESCE(slug,'') FROM posts
		 WHERE site_id = $1 AND deleted_at IS NULL AND status = 'published'
		 ORDER BY created_at DESC LIMIT 200`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts for topic authority: %w", err)
	}
	defer rows.Close()

	type post struct {
		title  string
		slug   string
		score  float64
	}
	posts := []post{}
	for rows.Next() {
		var title, slug string
		if err := rows.Scan(&title, &slug); err != nil {
			return nil, fmt.Errorf("failed to scan topic authority post: %w", err)
		}
		posts = append(posts, post{title: title, slug: slug})
	}

	related := []string{}
	for _, pst := range posts {
		coverage := keywordCoverage(pst.title+" "+pst.slug, topic)
		subject := 0.0
		tokens := tokenize(pst.title)
		if len(tokens) > 0 && len(keywords) > 0 {
			hits := 0
			for _, t := range tokens {
				if len(t) < 4 {
					continue
				}
				for _, k := range keywords {
					if t == k {
						hits++
						break
					}
				}
			}
			subject = float64(hits) / float64(len(keywords))
		}
		if coverage >= 0.3 || subject >= 0.25 {
			related = append(related, pst.title)
		}
	}

	authority := TopicAuthorityScore(len(related))

	// Gaps: the topic's own keywords that no related article mentions.
	gaps := []string{}
	for _, k := range keywords {
		if len(k) < 4 {
			continue
		}
		found := false
		for _, r := range related {
			if strings.Contains(strings.ToLower(r), k) {
				found = true
				break
			}
		}
		if !found {
			gaps = append(gaps, k)
		}
	}

	sort.Strings(gaps)
	return &TopicalAuthorityReport{
		Topic:          topic,
		Coverage:       len(related),
		Authority:      authority,
		RelatedArticles: related,
		Gaps:           gaps,
	}, nil
}

// FillTopicAuthorityGap creates a minimal gap analysis from topic terms,
// exposing which sub-topics have zero coverage. DB-free.
func FillTopicAuthorityGap(topic string, relatedTitles []string) []string {
	keywords := tokenize(topic)
	gaps := []string{}
	for _, k := range keywords {
		if len(k) < 4 {
			continue
		}
		found := false
		for _, r := range relatedTitles {
			if strings.Contains(strings.ToLower(r), k) {
				found = true
				break
			}
		}
		if !found {
			gaps = append(gaps, k)
		}
	}
	sort.Strings(gaps)
	return gaps
}
