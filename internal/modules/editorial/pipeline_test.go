package editorial

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

func newEditorialTestSvc(t *testing.T, m pgxmock.PgxPoolIface) *Service {
	t.Helper()
	cfg := &config.Config{}
	cfg.SEO = config.SEOConfig{MinPublishScore: 80}
	cfg.Editorial = config.EditorialConfig{MinFinalScore: 90}
	log := logger.New(cfg)
	ch := cache.New(true)
	var pool database.Pool
	if m != nil {
		pool = m
	}
	db := &database.Database{Pool: pool}
	return NewService(cfg, log, db, ch)
}

func TestGetPipelineNoDB(t *testing.T) {
	svc := newEditorialTestSvc(t, nil)
	_, err := svc.GetPipeline(context.Background(), uuid.New(), 200)
	if !errors.Is(err, ErrDatabaseNotAvail) {
		t.Fatalf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestGetPipelineRows(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newEditorialTestSvc(t, m)
	siteID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	seo := 92.0
	author := uuid.New()

	rows := pgxmock.NewRows([]string{
		"id", "title", "slug", "stage", "engine", "language",
		"category_id", "author_id", "seo_score", "eeat_score",
		"status", "scheduled_at", "updated_at",
	}).
		AddRow(uuid.New(), "Post auditado", "post-auditado", "seo", "posts", "pt",
			nil, &author, &seo, nil, "draft", nil, now).
		AddRow(uuid.New(), "Aprovação pendente", "", "approval", "approval_requests", "pt",
			nil, nil, nil, nil, "pending", nil, now)

	m.ExpectQuery(`ORDER BY u.updated_at DESC`).
		WithArgs(siteID, 200).
		WillReturnRows(rows)

	resp, err := svc.GetPipeline(context.Background(), siteID, 200)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 2 {
		t.Fatalf("expected 2 items, got %d", resp.Total)
	}
	if resp.Items[0].Stage != StageSEO || resp.Items[0].Engine != "posts" {
		t.Errorf("unexpected first item: %+v", resp.Items[0])
	}
	if resp.Items[0].SEOScore == nil || *resp.Items[0].SEOScore != 92 {
		t.Errorf("expected seo score 92, got %v", resp.Items[0].SEOScore)
	}
	if resp.Items[0].Actionable {
		t.Error("seo item should not be actionable")
	}
	if !resp.Items[1].Actionable {
		t.Error("approval item should be actionable")
	}

	byStage := map[PipelineStage]int{}
	for _, sc := range resp.Stages {
		byStage[sc.Stage] = sc.Count
	}
	if byStage[StageSEO] != 1 || byStage[StageApproval] != 1 {
		t.Errorf("unexpected stage counts: %+v", resp.Stages)
	}
	if len(resp.Stages) != len(PipelineStageOrder) {
		t.Errorf("expected %d stage counts, got %d", len(PipelineStageOrder), len(resp.Stages))
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetPipelineStats(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newEditorialTestSvc(t, m)
	siteID := uuid.New()

	counts := pgxmock.NewRows([]string{
		"c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8", "c9", "c10", "c11",
	}).AddRow(2, 1, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	m.ExpectQuery(`SELECT\s+\(SELECT COUNT\(\*\) FROM editorial_briefs`).
		WithArgs(siteID).
		WillReturnRows(counts)

	avgs := pgxmock.NewRows([]string{"seo_score", "eeat_score"}).
		AddRow(90.0, 85.0).
		AddRow(70.0, 95.0)
	m.ExpectQuery(`SELECT seo_score, eeat_score FROM editorial_reviews`).
		WithArgs(siteID).
		WillReturnRows(avgs)

	m.ExpectQuery(`INTERVAL '7 days'`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"n"}).AddRow(3))

	stats, err := svc.GetPipelineStats(context.Background(), siteID)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalItems != 66 {
		t.Errorf("expected total 66, got %d", stats.TotalItems)
	}
	if stats.AvgSEOScore == nil || *stats.AvgSEOScore != 80 {
		t.Errorf("expected avg seo 80, got %v", stats.AvgSEOScore)
	}
	if stats.AvgEEATScore == nil || *stats.AvgEEATScore != 90 {
		t.Errorf("expected avg eeat 90, got %v", stats.AvgEEATScore)
	}
	if stats.PendingReviews != 8 || stats.PendingApprovals != 9 {
		t.Errorf("unexpected pending counts: %+v", stats)
	}
	if stats.PublishedWeek != 3 {
		t.Errorf("expected 3 weekly publications, got %d", stats.PublishedWeek)
	}
	if len(stats.StageCounts) != len(PipelineStageOrder) {
		t.Errorf("expected %d stage counts", len(PipelineStageOrder))
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetPipelineStatsEmptyAverages(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newEditorialTestSvc(t, m)
	siteID := uuid.New()

	counts := pgxmock.NewRows([]string{
		"c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8", "c9", "c10", "c11",
	}).AddRow(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	m.ExpectQuery(`SELECT\s+\(SELECT COUNT\(\*\) FROM editorial_briefs`).
		WithArgs(siteID).
		WillReturnRows(counts)

	m.ExpectQuery(`SELECT seo_score, eeat_score FROM editorial_reviews`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"seo_score", "eeat_score"}))

	m.ExpectQuery(`INTERVAL '7 days'`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"n"}).AddRow(0))

	stats, err := svc.GetPipelineStats(context.Background(), siteID)
	if err != nil {
		t.Fatal(err)
	}
	if stats.AvgSEOScore != nil || stats.AvgEEATScore != nil {
		t.Errorf("expected nil averages, got %v/%v", stats.AvgSEOScore, stats.AvgEEATScore)
	}
}

func TestGetPublishReadinessBlockedSEO(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newEditorialTestSvc(t, m)
	siteID, postID := uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	seo := 55.0

	m.ExpectQuery(`SELECT title, slug, seo_score, seo_analyzed_at FROM posts`).
		WithArgs(postID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{"title", "slug", "seo_score", "seo_analyzed_at"}).
			AddRow("Meu post", "meu-post", &seo, &now))

	m.ExpectQuery(`FROM editorial_reviews`).
		WithArgs(postID, siteID).
		WillReturnError(pgx.ErrNoRows)

	pr, err := svc.GetPublishReadiness(context.Background(), siteID, postID)
	if err != nil {
		t.Fatal(err)
	}
	if pr.Ready {
		t.Error("expected not ready with seo 55 < 80")
	}
	if pr.Blocking != "seo" {
		t.Errorf("expected blocking seo, got %q", pr.Blocking)
	}
	for _, c := range pr.Checks {
		if c.Stage == "eeat" && !c.Passed {
			t.Error("eeat should fail open without review")
		}
		if c.Stage == "editorial" && !c.Passed {
			t.Error("editorial should fail open without review")
		}
		if c.Stage == "pipeline" && !c.Passed {
			t.Error("pipeline should pass when seo analyzed")
		}
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetPublishReadinessReady(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newEditorialTestSvc(t, m)
	siteID, postID := uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	seo := 92.0

	m.ExpectQuery(`SELECT title, slug, seo_score, seo_analyzed_at FROM posts`).
		WithArgs(postID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{"title", "slug", "seo_score", "seo_analyzed_at"}).
			AddRow("Meu post", "meu-post", &seo, &now))

	m.ExpectQuery(`FROM editorial_reviews`).
		WithArgs(postID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"seo_score", "eeat_score", "freshness_score", "coverage_score", "naturalness_score",
			"confidence_score", "final_score", "decision", "threshold",
		}).AddRow(90.0, 92.0, 95.0, 85.0, 91.0, 90.0, 93.0, "approved", 90.0))

	pr, err := svc.GetPublishReadiness(context.Background(), siteID, postID)
	if err != nil {
		t.Fatal(err)
	}
	if !pr.Ready {
		t.Errorf("expected ready, got blocking=%q", pr.Blocking)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetArticleReviewNoReview(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newEditorialTestSvc(t, m)
	siteID, postID := uuid.New(), uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	seo := 92.0
	fresh := 0.9

	m.ExpectQuery(`SELECT title, slug, status, COALESCE\(post_meta->>'language', 'pt'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{"title", "slug", "status", "language", "seo_score", "seo_analyzed_at", "updated_at", "content"}).
			AddRow("Meu post", "meu-post", "draft", "pt", &seo, &now, now, `[]`))

	m.ExpectQuery(`FROM editorial_reviews`).
		WithArgs(postID, siteID).
		WillReturnError(pgx.ErrNoRows)

	m.ExpectQuery(`FROM article_sources`).
		WithArgs(siteID, postID).
		WillReturnRows(pgxmock.NewRows([]string{
			"source_url", "title", "snippet", "language", "is_verified", "freshness_score", "relevance_score", "retrieved_at",
		}).AddRow("https://exemplo.com/fonte", "Fonte A", "resumo", "pt", false, &fresh, 80, now))

	m.ExpectQuery(`SELECT COALESCE\(seo_issues::text, '\[\]'\) FROM posts`).
		WithArgs(postID).
		WillReturnError(pgx.ErrNoRows)

	m.ExpectQuery(`SELECT title, slug, seo_score, seo_analyzed_at FROM posts`).
		WithArgs(postID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{"title", "slug", "seo_score", "seo_analyzed_at"}).
			AddRow("Meu post", "meu-post", &seo, &now))

	m.ExpectQuery(`FROM editorial_reviews`).
		WithArgs(postID, siteID).
		WillReturnError(pgx.ErrNoRows)

	ar, err := svc.GetArticleReview(context.Background(), siteID, postID)
	if err != nil {
		t.Fatal(err)
	}
	if ar.Post.Title != "Meu post" {
		t.Errorf("unexpected post title: %q", ar.Post.Title)
	}
	if ar.Review != nil {
		t.Error("expected nil review")
	}
	if len(ar.Sources) != 1 || ar.Sources[0].IsVerified {
		t.Errorf("unexpected sources: %+v", ar.Sources)
	}
	found := false
	for _, pr := range ar.Problems {
		if pr.Kind == "sources" {
			found = true
		}
	}
	if !found {
		t.Error("expected a sources problem for unverified source")
	}
	if ar.Readiness == nil || !ar.Readiness.Ready {
		t.Errorf("expected fail-open readiness, got %+v", ar.Readiness)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetArticleReviewWithReview(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newEditorialTestSvc(t, m)
	siteID, postID := uuid.New(), uuid.New()
	revID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	seo := 92.0

	m.ExpectQuery(`SELECT title, slug, status, COALESCE\(post_meta->>'language', 'pt'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{"title", "slug", "status", "language", "seo_score", "seo_analyzed_at", "updated_at", "content"}).
			AddRow("Meu post", "meu-post", "draft", "pt", &seo, &now, now, `[]`))

	m.ExpectQuery(`FROM editorial_reviews`).
		WithArgs(postID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "seo_score", "eeat_score", "freshness_score", "coverage_score", "naturalness_score",
			"confidence_score", "final_score", "decision", "threshold", "created_at",
		}).AddRow(revID, 90.0, 92.0, 95.0, 85.0, 91.0, 90.0, 85.0, "needs_review", 90.0, now))

	m.ExpectQuery(`SELECT coverage::text, fluency::text, semantic::text`).
		WithArgs(revID).
		WillReturnRows(pgxmock.NewRows([]string{"coverage", "fluency", "semantic"}).
			AddRow(
				[]byte(`{"coverage_percent": 62.5, "missing": [{"facet": "price", "message": "Preço não citado"}]}`),
				[]byte(`{"issues": [{"message": "Frase muito longa"}]}`),
				[]byte(`{"missing_terms": ["tutorial"]}`)))

	m.ExpectQuery(`FROM article_sources`).
		WithArgs(siteID, postID).
		WillReturnRows(pgxmock.NewRows([]string{
			"source_url", "title", "snippet", "language", "is_verified", "freshness_score", "relevance_score", "retrieved_at",
		}).AddRow("https://exemplo.com/fonte", "Fonte A", "resumo", "pt", false, nil, 80, now))

	m.ExpectQuery(`FROM editorial_evidence`).
		WithArgs(revID).
		WillReturnRows(pgxmock.NewRows([]string{"claim", "source_title", "source_url", "confidence", "note"}))

	m.ExpectQuery(`SELECT COALESCE\(seo_issues::text, '\[\]'\) FROM posts`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows([]string{"issues"}).
			AddRow([]byte(`[{"field": "title", "issue": "Título muito longo", "priority": "high"}]`)))

	m.ExpectQuery(`SELECT title, slug, seo_score, seo_analyzed_at FROM posts`).
		WithArgs(postID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{"title", "slug", "seo_score", "seo_analyzed_at"}).
			AddRow("Meu post", "meu-post", &seo, &now))

	m.ExpectQuery(`FROM editorial_reviews`).
		WithArgs(postID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"seo_score", "eeat_score", "freshness_score", "coverage_score", "naturalness_score",
			"confidence_score", "final_score", "decision", "threshold",
		}).AddRow(90.0, 92.0, 95.0, 85.0, 91.0, 90.0, 85.0, "needs_review", 90.0))

	ar, err := svc.GetArticleReview(context.Background(), siteID, postID)
	if err != nil {
		t.Fatal(err)
	}
	if ar.Review == nil || ar.Review.Decision != "needs_review" {
		t.Fatalf("unexpected review: %+v", ar.Review)
	}
	kinds := map[string]bool{}
	for _, pr := range ar.Problems {
		kinds[pr.Kind] = true
	}
	for _, want := range []string{"seo", "coverage", "fluency", "semantic", "sources", "editorial"} {
		if kinds[want] {
			continue
		}
		t.Errorf("expected problem kind %q, got %v", want, kinds)
	}
	recs := map[string]*ReviewRecommendation{}
	for i := range ar.Recommendations {
		recs[ar.Recommendations[i].Label] = &ar.Recommendations[i]
	}
	if r, ok := recs["SEO"]; !ok || r.Status != "ok" {
		t.Errorf("expected SEO ok recommendation, got %+v", recs["SEO"])
	}
	if r, ok := recs["E-E-A-T"]; !ok || r.Status != "ok" {
		t.Errorf("expected E-E-A-T ok recommendation, got %+v", recs["E-E-A-T"])
	}
	if r, ok := recs["Fontes"]; !ok || r.Status != "warning" {
		t.Errorf("expected Fontes warning, got %+v", recs["Fontes"])
	}
	if r, ok := recs["Problemas"]; !ok || r.Status != "warning" {
		t.Errorf("expected Problemas warning, got %+v", recs["Problemas"])
	}
	if r, ok := recs["Ação recomendada"]; !ok || r.Status != "fail" {
		t.Errorf("expected fail action recommendation, got %+v", recs["Ação recomendada"])
	}
	if ar.Readiness == nil || ar.Readiness.Blocking != "editorial" {
		t.Errorf("expected editorial blocking, got %+v", ar.Readiness)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestLoadReviewEvidenceIsolated(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newEditorialTestSvc(t, m)
	revID := uuid.New()

	m.ExpectQuery(`FROM editorial_evidence`).
		WithArgs(revID).
		WillReturnRows(pgxmock.NewRows([]string{"claim", "source_title", "source_url", "confidence", "note"}).
			AddRow("O preço é R$ 99", "Site oficial", "https://oficial.com", 60.0, ""))

	ar := &ArticleReview{Sources: []ReviewSource{}, Problems: []ReviewProblem{}}
	if err := svc.loadReviewEvidence(context.Background(), m, revID, ar); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, pr := range ar.Problems {
		if pr.Kind == "evidence" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected evidence problem, got %+v", ar.Problems)
	}
	if err := m.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetArticleReviewNotFound(t *testing.T) {
	m, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	svc := newEditorialTestSvc(t, m)
	siteID, postID := uuid.New(), uuid.New()

	m.ExpectQuery(`SELECT title, slug, status`).
		WithArgs(postID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err = svc.GetArticleReview(context.Background(), siteID, postID)
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestDeriveKeyword(t *testing.T) {
	cases := []struct{ title, want string }{
		{"Como fazer um bolo de chocolate", "chocolate"},
		{"GPT-6 vs Gemini 2.5: comparativo", "comparativo"},
		{"", ""},
	}
	for _, c := range cases {
		if got := deriveKeyword(c.title); got != c.want {
			t.Errorf("deriveKeyword(%q) = %q, want %q", c.title, got, c.want)
		}
	}
}