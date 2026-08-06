package editorialbrain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"nexora/internal/ai"
	"nexora/internal/modules/seoengine"
	"nexora/internal/modules/publisher"
	"nexora/internal/pkg/audit"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/kernel"
	"nexora/internal/pkg/logger"
)

// ResearchProvider supplies facts and sources from the research module.
// Implementations must be nil-safe: returning empty slices is valid.
type ResearchProvider interface {
	LoadFacts(ctx context.Context, siteID uuid.UUID, jobID uuid.UUID) ([]FactEntry, error)
	LoadSources(ctx context.Context, siteID uuid.UUID, topic, language string) ([]SourceRef, error)
}

// Service is the editorial brain engine: briefs before writing, full
// editorial reviews + gate decision before publishing.
type Service struct {
	log           *logger.Logger
	db            *database.Database
	eventBus      *kernel.EventBus
	auditLog      *audit.Logger
	qualityChecker ai.QualityChecker
	research      ResearchProvider
	minFinalScore float64
}

func NewService(cfg *config.Config, log *logger.Logger, db *database.Database) *Service {
	var pool database.Pool
	if db != nil {
		pool = db.Pool
	}
	return &Service{
		log:           log,
		db:            db,
		auditLog:      audit.New(pool, log),
		minFinalScore: cfg.Editorial.MinFinalScore,
	}
}

func (s *Service) SetEventBus(bus *kernel.EventBus) { s.eventBus = bus }

// SetQualityChecker registers the deterministic AI quality checker used by
// the fluency check. Optional (nil-safe).
func (s *Service) SetQualityChecker(qc ai.QualityChecker) { s.qualityChecker = qc }

// SetResearchProvider registers the research fact/source provider. Optional:
// without it, evidence checks degrade to unverified results.
func (s *Service) SetResearchProvider(rp ResearchProvider) { s.research = rp }

// MinFinalScore returns the configured editorial gate threshold.
func (s *Service) MinFinalScore() float64 {
	if s.minFinalScore <= 0 {
		return DefaultMinFinalScore
	}
	return s.minFinalScore
}

func (s *Service) pool() (database.Pool, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, ErrDatabaseNotAvail
	}
	return s.db.Pool, nil
}

func (s *Service) fireEvent(ctx context.Context, eventType kernel.EventType, payload interface{}, siteID uuid.UUID) {
	if s.eventBus != nil {
		s.eventBus.EmitAsync(ctx, eventType, payload, siteID.String())
	}
}

// loadResearch loads facts (by research job) and sources (by topic cache).
func (s *Service) loadResearch(ctx context.Context, siteID uuid.UUID, jobID *uuid.UUID, topic, language string) ([]FactEntry, []SourceRef) {
	if s.research == nil {
		return nil, nil
	}
	var facts []FactEntry
	if jobID != nil {
		if f, err := s.research.LoadFacts(ctx, siteID, *jobID); err == nil {
			facts = f
		}
	}
	sources, err := s.research.LoadSources(ctx, siteID, topic, language)
	if err != nil {
		sources = nil
	}
	return facts, sources
}

// BuildBrief builds the full editorial brief: intent, persona, angle,
// audience, outline and required questions. Persisted when the DB is up.
func (s *Service) BuildBrief(ctx context.Context, siteID, userID uuid.UUID, topic, language string) (*EditorialBrief, error) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil, ErrTopicRequired
	}
	if language == "" {
		language = "pt"
	}
	if language != "pt" && language != "en" {
		return nil, ErrInvalidLanguage
	}

	intent := ClassifyIntent(topic, language)
	persona := DetectPersona(topic, language)
	outline := GenerateOutline(topic, language, intent.Intent, persona.Persona)
	questions := GenerateQuestions(topic, language, intent.Intent)

	angle := b(
		fmt.Sprintf("Foco em %s, direcionado a %s. Estrutura: %s.",
			intent.Intent.Label(language), persona.Persona.AudienceLabel(language), outline.SuggestedTitle),
		fmt.Sprintf("%s angle, targeted at %s. Structure: %s.",
			intent.Intent.Label(language), persona.Persona.AudienceLabel(language), outline.SuggestedTitle),
	).text(language)

	brief := &EditorialBrief{
		ID:                uuid.New(),
		SiteID:            siteID,
		Topic:             topic,
		TopicHash:         topicHash(topic),
		Language:          language,
		SearchIntent:      intent.Intent,
		IntentConfidence:  intent.Confidence,
		Persona:           persona.Persona,
		PersonaConfidence: persona.Confidence,
		Audience:          persona.Audience,
		Angle:             angle,
		SuggestedTitle:    outline.SuggestedTitle,
		Outline:           outline.Sections,
		Questions:         questions,
		Status:            "ready",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.saveBrief(ctx, brief); err != nil {
		return nil, err
	}
	s.fireEvent(ctx, EventBriefCreated, map[string]interface{}{
		"brief_id": brief.ID.String(), "site_id": siteID.String(),
		"topic": topic, "intent": string(intent.Intent), "persona": string(persona.Persona),
	}, siteID)
	s.auditLog.Log(ctx, audit.Entry{
		UserID: &userID, SiteID: &siteID,
		Action: audit.Action("editorial.brief_created"), EntityType: "editorial_brief",
		EntityID: &brief.ID,
		Payload:  map[string]interface{}{"topic": topic, "intent": string(intent.Intent), "persona": string(persona.Persona)},
	})
	return brief, nil
}

// ReviewRequest describes the article to review.
type ReviewRequest struct {
	BriefID       *uuid.UUID
	ArticleID     *uuid.UUID
	Title         string
	Content       string
	Language      string
	ResearchJobID *uuid.UUID
}

// ReviewArticle runs the full editorial note on an article and returns the
// gate decision. Below the threshold the article returns to review instead
// of being published.
func (s *Service) ReviewArticle(ctx context.Context, siteID uuid.UUID, req ReviewRequest) (*EditorialReview, error) {
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if title == "" || content == "" {
		return nil, ErrContentRequired
	}
	language := req.Language
	if language == "" {
		language = "pt"
	}
	if language != "pt" && language != "en" {
		return nil, ErrInvalidLanguage
	}

	topic := title
	var intent SearchIntent = IntentInformational
	var persona Persona = PersonaGeneral
	var entities, concepts []string
	var questions []RequiredQuestion

	if req.BriefID != nil {
		if b, err := s.getBrief(ctx, siteID, *req.BriefID); err == nil && b != nil {
			intent = b.SearchIntent
			persona = b.Persona
			questions = b.Questions
			entities = deriveEntities(b.Topic)
			concepts = deriveConcepts(b.Topic)
			topic = b.Topic
		}
	}
	if intent == "" || len(questions) == 0 {
		ir := ClassifyIntent(topic, language)
		intent = ir.Intent
		questions = GenerateQuestions(topic, language, intent)
	}
	if persona == "" {
		persona = DetectPersona(topic, language).Persona
	}

	facts, sources := s.loadResearch(ctx, siteID, req.ResearchJobID, topic, language)

	coverage := CheckCoverage(content, language)
	fluency := CheckFluency(ctx, content, language, s.qualityChecker)
	qc := CheckQuestions(content, questions)
	evidence := LinkEvidence(content, facts, sources, language)
	blocks := ScoreBlocks(content, evidence, language)
	semantic := CheckSemantic(topic, content, language, entities, concepts, qc.AnsweredPercent)

	seoAnalysis := seoengine.AnalyzeArticle(ctx, seoengine.ArticleAnalysisInput{
		Title: title, Content: content, Keyword: deriveKeyword(topic), Language: language,
	}, s.qualityChecker)
	seoScore := 0.0
	if seoAnalysis != nil {
		seoScore = seoAnalysis.OverallScore
	}
	eeatReport := seoengine.AnalyzeEEAT(seoengine.ArticleAnalysisInput{
		Title: title, Content: content, Language: language,
	})
	eeatScore := 0.0
	if eeatReport != nil {
		eeatScore = eeatReport.Final
	}
	freshnessScore := SourcesFreshnessScore(sources)

	scores := ComputeEditorialScore(
		seoScore, eeatScore, freshnessScore, coverage.CoveragePercent,
		fluency.OverallScore, evidence.EvidenceScore, s.MinFinalScore(),
	)

	review := &EditorialReview{
		ID:           uuid.New(),
		SiteID:       siteID,
		BriefID:      req.BriefID,
		ArticleID:    req.ArticleID,
		ArticleTitle: title,
		ContentHash:  contentHash(title, content),
		Scores:       scores,
		Coverage:     coverage,
		Fluency:      fluency,
		Semantic:     semantic,
		Blocks:       blocks,
		Evidence:     evidence.Links,
		CreatedAt:    time.Now(),
	}

	if err := s.saveReview(ctx, review); err != nil {
		return nil, err
	}

	s.fireEvent(ctx, EventReviewCreated, map[string]interface{}{
		"review_id": review.ID.String(), "site_id": siteID.String(),
		"title": title, "final": scores.Final, "decision": string(scores.Decision),
	}, siteID)
	if scores.Decision == DecisionNeedsReview {
		s.fireEvent(ctx, EventScoreBlocked, map[string]interface{}{
			"review_id": review.ID.String(), "site_id": siteID.String(),
			"title": title, "final": scores.Final, "threshold": scores.Threshold,
		}, siteID)
	}
	s.auditLog.Log(ctx, audit.Entry{
		UserID: nil, SiteID: &siteID,
		Action: audit.Action("editorial.review_created"), EntityType: "editorial_review",
		EntityID: &review.ID,
		Payload:  map[string]interface{}{"title": title, "final": scores.Final, "decision": string(scores.Decision)},
	})
	return review, nil
}

// CheckEditorialScore implements the publisher editorial gate: it returns the
// final score of the most recent review for the same content. When no review
// exists the gate fails open (100).
func (s *Service) CheckEditorialScore(ctx context.Context, in publisher.EditorialGateInput) (float64, error) {
	hash := contentHash(in.Title, in.Content)
	p, err := s.pool()
	if err != nil {
		return 100, nil
	}
	var final float64
	err = p.QueryRow(ctx, `SELECT final_score FROM editorial_reviews
		WHERE site_id = $1 AND content_hash = $2 ORDER BY created_at DESC LIMIT 1`,
		in.SiteID, hash).Scan(&final)
	if err != nil {
		return 100, nil
	}
	return final, nil
}

// GetBrief loads a brief by id (site-scoped).
func (s *Service) GetBrief(ctx context.Context, siteID, briefID uuid.UUID) (*EditorialBrief, error) {
	return s.getBrief(ctx, siteID, briefID)
}

// ListBriefs lists the site's briefs, most recent first.
func (s *Service) ListBriefs(ctx context.Context, siteID uuid.UUID, limit, offset int) ([]EditorialBrief, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := p.Query(ctx, `SELECT id, site_id, topic, topic_hash, language, search_intent,
		intent_confidence, persona, persona_confidence, audience, angle, suggested_title,
		outline, questions, status, created_at, updated_at
		FROM editorial_briefs WHERE site_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		siteID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]EditorialBrief, 0)
	for rows.Next() {
		var b EditorialBrief
		var outlineJSON, questionsJSON []byte
		if err := rows.Scan(&b.ID, &b.SiteID, &b.Topic, &b.TopicHash, &b.Language,
			&b.SearchIntent, &b.IntentConfidence, &b.Persona, &b.PersonaConfidence,
			&b.Audience, &b.Angle, &b.SuggestedTitle, &outlineJSON, &questionsJSON,
			&b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(outlineJSON, &b.Outline)
		_ = json.Unmarshal(questionsJSON, &b.Questions)
		out = append(out, b)
	}
	return out, rows.Err()
}

// GetReview loads a review by id (site-scoped).
func (s *Service) GetReview(ctx context.Context, siteID, reviewID uuid.UUID) (*EditorialReview, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	var r EditorialReview
	var coverageJSON, fluencyJSON, semanticJSON []byte
	err = p.QueryRow(ctx, `SELECT id, site_id, brief_id, article_id, article_title, content_hash,
		seo_score, eeat_score, freshness_score, coverage_score, naturalness_score, confidence_score,
		final_score, decision, threshold, coverage, fluency, semantic, created_at
		FROM editorial_reviews WHERE id = $1 AND site_id = $2`, reviewID, siteID).
		Scan(&r.ID, &r.SiteID, &r.BriefID, &r.ArticleID, &r.ArticleTitle, &r.ContentHash,
			&r.Scores.SEO, &r.Scores.EEAT, &r.Scores.Freshness, &r.Scores.Coverage,
			&r.Scores.Naturalness, &r.Scores.Confidence, &r.Scores.Final, &r.Scores.Decision,
			&r.Scores.Threshold, &coverageJSON, &fluencyJSON, &semanticJSON, &r.CreatedAt)
	if err != nil {
		return nil, ErrReviewNotFound
	}
	_ = json.Unmarshal(coverageJSON, &r.Coverage)
	_ = json.Unmarshal(fluencyJSON, &r.Fluency)
	_ = json.Unmarshal(semanticJSON, &r.Semantic)

	blockRows, err := p.Query(ctx, `SELECT block, score, evidence_count, note
		FROM editorial_block_scores WHERE review_id = $1 ORDER BY created_at`, r.ID)
	if err == nil {
		defer blockRows.Close()
		for blockRows.Next() {
			var b BlockScore
			if err := blockRows.Scan(&b.Block, &b.Score, &b.EvidenceCount, &b.Note); err == nil {
				r.Blocks = append(r.Blocks, b)
			}
		}
	}
	evRows, err := p.Query(ctx, `SELECT claim, verified, source_title, source_url, confidence, note
		FROM editorial_evidence WHERE review_id = $1 ORDER BY created_at`, r.ID)
	if err == nil {
		defer evRows.Close()
		for evRows.Next() {
			var l EvidenceLink
			if err := evRows.Scan(&l.Claim, &l.Verified, &l.SourceTitle, &l.SourceURL, &l.Confidence, &l.Note); err == nil {
				r.Evidence = append(r.Evidence, l)
			}
		}
	}
	return &r, nil
}

// ListReviews lists the site's reviews, most recent first.
func (s *Service) ListReviews(ctx context.Context, siteID uuid.UUID, decision string, limit, offset int) ([]EditorialReview, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query := `SELECT id, site_id, article_title, content_hash, seo_score, eeat_score,
		freshness_score, coverage_score, naturalness_score, confidence_score, final_score,
		decision, threshold, created_at FROM editorial_reviews WHERE site_id = $1`
	args := []interface{}{siteID}
	if decision != "" {
		query += ` AND decision = $2`
		args = append(args, decision)
	}
	query += ` ORDER BY created_at DESC LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]EditorialReview, 0)
	for rows.Next() {
		var r EditorialReview
		if err := rows.Scan(&r.ID, &r.SiteID, &r.ArticleTitle, &r.ContentHash,
			&r.Scores.SEO, &r.Scores.EEAT, &r.Scores.Freshness, &r.Scores.Coverage,
			&r.Scores.Naturalness, &r.Scores.Confidence, &r.Scores.Final, &r.Scores.Decision,
			&r.Scores.Threshold, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
