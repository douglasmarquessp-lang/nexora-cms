package editorialbrain

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

// deriveKeyword returns the longest significant token of a title.
func deriveKeyword(title string) string {
	best := ""
	for _, w := range tokenize(title) {
		if stopWords[w] || len(w) < 4 {
			continue
		}
		if len(w) > len(best) {
			best = w
		}
	}
	return best
}

func (s *Service) saveBrief(ctx context.Context, b *EditorialBrief) error {
	p, err := s.pool()
	if err != nil {
		return nil
	}
	outlineJSON, _ := json.Marshal(b.Outline)
	questionsJSON, _ := json.Marshal(b.Questions)
	_, err = p.Exec(ctx, `INSERT INTO editorial_briefs
		(id, site_id, topic, topic_hash, language, search_intent, intent_confidence,
		 persona, persona_confidence, audience, angle, suggested_title, outline,
		 questions, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (site_id, topic_hash, language) DO UPDATE SET
			search_intent = EXCLUDED.search_intent,
			intent_confidence = EXCLUDED.intent_confidence,
			persona = EXCLUDED.persona,
			persona_confidence = EXCLUDED.persona_confidence,
			audience = EXCLUDED.audience,
			angle = EXCLUDED.angle,
			suggested_title = EXCLUDED.suggested_title,
			outline = EXCLUDED.outline,
			questions = EXCLUDED.questions,
			status = EXCLUDED.status,
			updated_at = NOW()`,
		b.ID, b.SiteID, b.Topic, b.TopicHash, b.Language, string(b.SearchIntent),
		b.IntentConfidence, string(b.Persona), b.PersonaConfidence, b.Audience,
		b.Angle, b.SuggestedTitle, outlineJSON, questionsJSON, b.Status, b.CreatedAt, b.UpdatedAt)
	return err
}

func (s *Service) getBrief(ctx context.Context, siteID, briefID uuid.UUID) (*EditorialBrief, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	var b EditorialBrief
	var outlineJSON, questionsJSON []byte
	err = p.QueryRow(ctx, `SELECT id, site_id, topic, topic_hash, language, search_intent,
		intent_confidence, persona, persona_confidence, audience, angle, suggested_title,
		outline, questions, status, created_at, updated_at
		FROM editorial_briefs WHERE id = $1 AND site_id = $2`, briefID, siteID).
		Scan(&b.ID, &b.SiteID, &b.Topic, &b.TopicHash, &b.Language, &b.SearchIntent,
			&b.IntentConfidence, &b.Persona, &b.PersonaConfidence, &b.Audience, &b.Angle,
			&b.SuggestedTitle, &outlineJSON, &questionsJSON, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, ErrBriefNotFound
	}
	_ = json.Unmarshal(outlineJSON, &b.Outline)
	_ = json.Unmarshal(questionsJSON, &b.Questions)
	return &b, nil
}

func (s *Service) saveReview(ctx context.Context, r *EditorialReview) error {
	p, err := s.pool()
	if err != nil {
		return nil
	}
	coverageJSON, _ := json.Marshal(r.Coverage)
	fluencyJSON, _ := json.Marshal(r.Fluency)
	semanticJSON, _ := json.Marshal(r.Semantic)

	_, err = p.Exec(ctx, `INSERT INTO editorial_reviews
		(id, site_id, brief_id, article_id, article_title, content_hash,
		 seo_score, eeat_score, freshness_score, coverage_score, naturalness_score,
		 confidence_score, final_score, decision, threshold, coverage, fluency, semantic, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		r.ID, r.SiteID, r.BriefID, r.ArticleID, r.ArticleTitle, r.ContentHash,
		r.Scores.SEO, r.Scores.EEAT, r.Scores.Freshness, r.Scores.Coverage,
		r.Scores.Naturalness, r.Scores.Confidence, r.Scores.Final,
		string(r.Scores.Decision), r.Scores.Threshold, coverageJSON, fluencyJSON,
		semanticJSON, r.CreatedAt)
	if err != nil {
		return err
	}
	for _, b := range r.Blocks {
		if _, err := p.Exec(ctx, `INSERT INTO editorial_block_scores
			(id, review_id, site_id, block, score, evidence_count, note, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			uuid.New(), r.ID, r.SiteID, b.Block, b.Score, b.EvidenceCount, b.Note, r.CreatedAt); err != nil {
			return err
		}
	}
	for _, l := range r.Evidence {
		if _, err := p.Exec(ctx, `INSERT INTO editorial_evidence
			(id, review_id, site_id, claim, verified, source_title, source_url, confidence, note, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			uuid.New(), r.ID, r.SiteID, l.Claim, l.Verified, l.SourceTitle,
			l.SourceURL, l.Confidence, l.Note, r.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

// IsNotFound reports whether an error is a "not found" sentinel.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrBriefNotFound) || errors.Is(err, ErrReviewNotFound)
}
