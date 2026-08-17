package editorialbrain

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/modules/publisher"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

func newTestSvc(t *testing.T, m pgxmock.PgxPoolIface) *Service {
	t.Helper()
	cfg := &config.Config{}
	cfg.Editorial = config.EditorialConfig{MinFinalScore: 90}
	log := logger.New(cfg)
	db := &database.Database{Pool: m}
	return NewService(cfg, log, db)
}

// fakeResearch is a nil-safe fake ResearchProvider.
type fakeResearch struct {
	facts   []FactEntry
	sources []SourceRef
}

func (f *fakeResearch) LoadFacts(ctx context.Context, siteID, jobID uuid.UUID) ([]FactEntry, error) {
	return f.facts, nil
}

func (f *fakeResearch) LoadSources(ctx context.Context, siteID uuid.UUID, topic, language string) ([]SourceRef, error) {
	return f.sources, nil
}

func TestMinFinalScore(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	if svc.MinFinalScore() != 90 {
		t.Errorf("expected default 90, got %v", svc.MinFinalScore())
	}
	cfg := &config.Config{}
	cfg.Editorial = config.EditorialConfig{MinFinalScore: 85}
	if NewService(cfg, logger.New(cfg), nil).MinFinalScore() != 85 {
		t.Error("expected configured 85")
	}
}

func TestBuildBriefValidation(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	if _, err := svc.BuildBrief(context.Background(), uuid.New(), uuid.New(), "", "pt"); !errors.Is(err, ErrTopicRequired) {
		t.Errorf("expected ErrTopicRequired, got %v", err)
	}
	if _, err := svc.BuildBrief(context.Background(), uuid.New(), uuid.New(), "gemini", "fr"); !errors.Is(err, ErrInvalidLanguage) {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestBuildBriefDBFree(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	b, err := svc.BuildBrief(context.Background(), uuid.New(), uuid.New(), "O que é o Gemini 3?", "pt")
	if err != nil {
		t.Fatal(err)
	}
	if b.SearchIntent != IntentInformational {
		t.Errorf("expected informational, got %s", b.SearchIntent)
	}
	if len(b.TopicHash) != 16 {
		t.Errorf("expected 16-hex topic hash, got %q", b.TopicHash)
	}
	if b.Status != "ready" || len(b.Outline) == 0 || len(b.Questions) == 0 {
		t.Error("brief not fully built")
	}
}

func TestBuildBriefUpserts(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()
	userID := uuid.New()

	m.ExpectExec(`INSERT INTO editorial_briefs`).
		WithArgs(pgxmock.AnyArg(), siteID, "O que é o Gemini 3?", pgxmock.AnyArg(), "pt",
			"informational", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), "ready",
			pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	m.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), "editorial.brief_created", "editorial_brief",
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	if _, err := svc.BuildBrief(context.Background(), siteID, userID, "O que é o Gemini 3?", "pt"); err != nil {
		t.Fatal(err)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReviewValidation(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	_, err := svc.ReviewArticle(context.Background(), uuid.New(), ReviewRequest{Title: "x", Content: ""})
	if !errors.Is(err, ErrContentRequired) {
		t.Errorf("expected ErrContentRequired, got %v", err)
	}
	_, err = svc.ReviewArticle(context.Background(), uuid.New(), ReviewRequest{Title: "x", Content: "y", Language: "de"})
	if !errors.Is(err, ErrInvalidLanguage) {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestReviewArticleDBFree(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	review, err := svc.ReviewArticle(context.Background(), uuid.New(), ReviewRequest{
		Title:    "O Gemini 3",
		Content:  "O Gemini 3 foi lançado em 2026.",
		Language: "pt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if review.Scores.Final <= 0 || review.Scores.Final > 100 {
		t.Errorf("final out of range: %v", review.Scores.Final)
	}
	if len(review.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(review.Blocks))
	}
	if len(review.Evidence) == 0 {
		t.Error("expected an unverified evidence link")
	}
	if review.Scores.Decision != DecisionNeedsReview {
		t.Errorf("expected needs_review, got %s", review.Scores.Decision)
	}
}

func TestReviewArticleWithResearch(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	now := time.Now()
	svc.SetResearchProvider(&fakeResearch{
		facts: []FactEntry{
			{FactType: "version", Entity: "Gemini", Value: "3", SourceURL: "https://gemini.google.com", Confidence: 80},
		},
		sources: []SourceRef{
			{Title: "Google Gemini 3", URL: "https://gemini.google.com/blog", Domain: "google.com",
				Snippet: "Gemini 3 foi lançado em 2026", ReliabilityScore: 100, Language: "pt", PublishedAt: &now, IsVerified: true},
		},
	})
	review, err := svc.ReviewArticle(context.Background(), uuid.New(), ReviewRequest{
		Title:    "O Gemini 3",
		Content:  "Gemini 3 foi lançado em 2026.",
		Language: "pt",
	})
	if err != nil {
		t.Fatal(err)
	}
	verified := false
	for _, l := range review.Evidence {
		if l.Verified {
			verified = true
		}
	}
	if !verified {
		t.Error("expected at least one verified evidence link")
	}
}

func TestReviewArticlePersists(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()

	// 1 review + 1 block + 1 evidence + 1 audit row.
	m.ExpectExec(`INSERT INTO editorial_reviews`).
		WithArgs(pgxmock.AnyArg(), siteID, pgxmock.AnyArg(), pgxmock.AnyArg(), "O Gemini 3", pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), "needs_review",
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	m.ExpectExec(`INSERT INTO editorial_block_scores`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), siteID, pgxmock.AnyArg(),
			pgxmock.AnyArg(), 0, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	m.ExpectExec(`INSERT INTO editorial_evidence`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), siteID, pgxmock.AnyArg(), false,
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	m.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), "editorial.review_created", "editorial_review",
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	if _, err := svc.ReviewArticle(context.Background(), siteID, ReviewRequest{
		Title:    "O Gemini 3",
		Content:  "O Gemini 3 foi lançado em 2026.",
		Language: "pt",
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCheckEditorialScoreFromReview(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()

	m.ExpectQuery(`SELECT final_score FROM editorial_reviews`).
		WithArgs(siteID, pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"final_score"}).AddRow(87.5))

	score, err := svc.CheckEditorialScore(context.Background(), publisher.EditorialGateInput{
		SiteID: siteID, Title: "O Gemini 3", Content: "O Gemini 3 foi lançado em 2026.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if score != 87.5 {
		t.Errorf("expected 87.5, got %v", score)
	}
}

// TestCheckEditorialScoreNoDB asserts the gate never fabricates a score: an
// infrastructure failure is a real error, not a silent 100.
func TestCheckEditorialScoreNoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	score, err := svc.CheckEditorialScore(context.Background(), publisher.EditorialGateInput{
		SiteID: uuid.New(), Title: "x", Content: "y",
	})
	if err == nil {
		t.Errorf("expected error without database, got score %v", score)
	}
	if score != 0 {
		t.Errorf("expected score 0 on error, got %v", score)
	}
}

// TestCheckEditorialScoreNoReview asserts the gate signals absence instead of
// fabricating a score: the caller (publisher) decides the disposition.
func TestCheckEditorialScoreNoReview(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	m.ExpectQuery(`SELECT final_score FROM editorial_reviews`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)
	score, err := svc.CheckEditorialScore(context.Background(), publisher.EditorialGateInput{
		SiteID: uuid.New(), Title: "x", Content: "y",
	})
	if !errors.Is(err, publisher.ErrNoEditorialReview) {
		t.Errorf("expected ErrNoEditorialReview, got %v", err)
	}
	if score != 0 {
		t.Errorf("expected score 0 without review, got %v", score)
	}
}

// TestReviewForGate asserts the auto-publish reviewer runs a full, real
// editorial review and returns its final note (never a fabricated value).
func TestReviewForGate(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()

	// ReviewArticle persists: 1 review + 1 block + 1 evidence + 1 audit row.
	m.ExpectExec(`INSERT INTO editorial_reviews`).
		WithArgs(pgxmock.AnyArg(), siteID, pgxmock.AnyArg(), pgxmock.AnyArg(), "O Gemini 3", pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), "needs_review",
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	m.ExpectExec(`INSERT INTO editorial_block_scores`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), siteID, pgxmock.AnyArg(),
			pgxmock.AnyArg(), 0, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	m.ExpectExec(`INSERT INTO editorial_evidence`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), siteID, pgxmock.AnyArg(), false,
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	m.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), "editorial.review_created", "editorial_review",
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	score, err := svc.ReviewForGate(context.Background(), publisher.EditorialGateInput{
		SiteID: siteID, Title: "O Gemini 3", Content: "O Gemini 3 foi lançado em 2026.", Language: "pt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if score <= 0 || score > 100 {
		t.Errorf("final note out of range: %v", score)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetBrief(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()
	briefID := uuid.New()

	outlineJSON, _ := json.Marshal([]OutlineSection{{Order: 1, Type: "intro", Title: "Introdução"}})
	questionsJSON, _ := json.Marshal([]RequiredQuestion{{ID: "what_is", Question: "O que é?"}})

	m.ExpectQuery(`SELECT id, site_id, topic`).
		WithArgs(briefID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "site_id", "topic", "topic_hash", "language",
			"search_intent", "intent_confidence", "persona", "persona_confidence", "audience",
			"angle", "suggested_title", "outline", "questions", "status", "created_at", "updated_at"}).
			AddRow(briefID, siteID, "O que é o Gemini 3?", "abc123", "pt", SearchIntent("informational"), 0.7,
				Persona("general"), 0.5, "Público geral", "angle", "O que é o Gemini 3?", outlineJSON,
				questionsJSON, "ready", time.Now(), time.Now()))

	b, err := svc.GetBrief(context.Background(), siteID, briefID)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Outline) != 1 || b.Outline[0].Title != "Introdução" {
		t.Errorf("outline not decoded: %+v", b.Outline)
	}
	if len(b.Questions) != 1 || b.Questions[0].Question != "O que é?" {
		t.Errorf("questions not decoded: %+v", b.Questions)
	}
}

func TestListBriefs(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()

	m.ExpectQuery(`SELECT id, site_id, topic`).
		WithArgs(siteID, 20, 0).
		WillReturnRows(pgxmock.NewRows([]string{"id", "site_id", "topic", "topic_hash", "language",
			"search_intent", "intent_confidence", "persona", "persona_confidence", "audience",
			"angle", "suggested_title", "outline", "questions", "status", "created_at", "updated_at"}).
			AddRow(uuid.New(), siteID, "t1", "h1", "pt", SearchIntent("tutorial"), 0.6, Persona("developer"), 0.5,
				"Dev", "a", "t1", []byte(`[]`), []byte(`[]`), "ready", time.Now(), time.Now()).
			AddRow(uuid.New(), siteID, "t2", "h2", "en", SearchIntent("informational"), 0.8, Persona("general"), 0.5,
				"General", "a", "t2", []byte(`[]`), []byte(`[]`), "ready", time.Now(), time.Now()))

	bs, err := svc.ListBriefs(context.Background(), siteID, 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(bs) != 2 {
		t.Errorf("expected 2 briefs, got %d", len(bs))
	}
}

func TestGetReview(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()
	reviewID := uuid.New()

	coverageJSON, _ := json.Marshal(CoverageReport{CoveragePercent: 50})
	fluencyJSON, _ := json.Marshal(FluencyReport{OverallScore: 80})
	semanticJSON, _ := json.Marshal(SemanticReport{SemanticScore: 70})

	m.ExpectQuery(`SELECT id, site_id, brief_id`).
		WithArgs(reviewID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "site_id", "brief_id", "article_id",
			"article_title", "content_hash", "seo_score", "eeat_score", "freshness_score",
			"coverage_score", "naturalness_score", "confidence_score", "final_score", "decision",
			"threshold", "coverage", "fluency", "semantic", "created_at"}).
			AddRow(reviewID, siteID, nil, nil, "O Gemini 3", "hash", float64(90), float64(90), float64(90), float64(90), float64(90), float64(90),
				float64(90), DecisionApproved, float64(90), coverageJSON, fluencyJSON, semanticJSON, time.Now()))
	m.ExpectQuery(`SELECT block, score, evidence_count, note`).
		WithArgs(reviewID).
		WillReturnRows(pgxmock.NewRows([]string{"block", "score", "evidence_count", "note"}).
			AddRow("Introdução", float64(55), 0, "Poucas evidências"))
	m.ExpectQuery(`SELECT claim, verified, source_title, source_url, confidence, note`).
		WithArgs(reviewID).
		WillReturnRows(pgxmock.NewRows([]string{"claim", "verified", "source_title", "source_url", "confidence", "note"}).
			AddRow("O Gemini 3 foi lançado em 2026", false, "", "", float64(45), "Sem evidência direta"))

	r, err := svc.GetReview(context.Background(), siteID, reviewID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Scores.Decision != DecisionApproved || r.Coverage.CoveragePercent != 50 {
		t.Errorf("review not decoded: %+v", r.Scores)
	}
	if len(r.Blocks) != 1 || len(r.Evidence) != 1 {
		t.Errorf("expected 1 block + 1 evidence, got %d + %d", len(r.Blocks), len(r.Evidence))
	}
}

func TestListReviews(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newTestSvc(t, m)
	siteID := uuid.New()

	m.ExpectQuery(`SELECT id, site_id, article_title`).
		WithArgs(siteID, "needs_review", 20, 0).
		WillReturnRows(pgxmock.NewRows([]string{"id", "site_id", "article_title", "content_hash",
			"seo_score", "eeat_score", "freshness_score", "coverage_score", "naturalness_score",
			"confidence_score", "final_score", "decision", "threshold", "created_at"}).
			AddRow(uuid.New(), siteID, "O Gemini 3", "h", float64(80), float64(80), float64(80), float64(80), float64(80), float64(80), float64(80),
				DecisionNeedsReview, float64(90), time.Now()))

	rs, err := svc.ListReviews(context.Background(), siteID, "needs_review", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 1 || rs[0].Scores.Decision != DecisionNeedsReview {
		t.Errorf("expected 1 needs_review row, got %+v", rs)
	}
}
