package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/api/rest"
	"nexora/internal/modules/publisher"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

// --- Fakes ---

type fakeEnhancer struct {
	out   *publisher.ContentEnhancement
	err   error
	calls int
	last  publisher.ContentEnhancerInput
}

func (f *fakeEnhancer) EnhanceForReview(_ context.Context, in publisher.ContentEnhancerInput) (*publisher.ContentEnhancement, error) {
	f.calls++
	f.last = in
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

type fakeSEOReviewer struct {
	out   *publisher.SEOReviewReport
	err   error
	calls int
	last  publisher.PublishGateInput
}

func (f *fakeSEOReviewer) ReviewSEO(_ context.Context, in publisher.PublishGateInput) (*publisher.SEOReviewReport, error) {
	f.calls++
	f.last = in
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

// --- Fixtures ---

var reviewJobCols = []string{
	"id", "site_id", "user_id", "title", "content_type",
	"language", "target_language", "status", "current_step", "progress",
	"priority", "word_count", "tone", "audience", "keywords", "style_slug",
	"source_job_id", "publication_id", "scheduled_for", "error_message", "retry_count",
	"max_retries", "generate_pt", "generate_en", "started_at", "completed_at",
	"cancelled_at", "created_by", "created_at", "updated_at",
	"review_status", "revision", "approved_by", "approved_at", "rejected_by", "rejected_at", "rejection_reason",
}

func reviewJobRow(jobID, siteID uuid.UUID, review ReviewStatus, pubID *uuid.UUID) *pgxmock.Rows {
	now := time.Now()
	return pgxmock.NewRows(reviewJobCols).AddRow(
		jobID, siteID, &wfJobUserID, "Test Article", "article", "en", "", JobStatus("completed"), "", float64(100),
		5, 0, "", "", []string{}, "", nil, pubID, nil, "", 0, 3, false, false, nil, nil, nil, &wfJobUserID, now, now,
		review, 1, nil, nil, nil, nil, "",
	)
}

func reviewStepRows(jobID uuid.UUID) *pgxmock.Rows {
	meta, _ := json.Marshal(map[string]interface{}{
		"ai_content": "# Test Article\n\nThis is the generated article body with enough content to analyze.",
		"keyword":    "test article",
	})
	return pgxmock.NewRows([]string{"step_name", "metadata"}).AddRow("finished", string(meta))
}

func versionCols() []string {
	return []string{
		"id", "site_id", "workflow_job_id", "version", "title", "slug", "content",
		"meta_title", "meta_description", "keyword", "featured_image_url",
		"featured_image_alt", "language", "created_at",
	}
}

func versionRow(jobID, siteID uuid.UUID, version int, title, keyword, metaTitle, metaDesc string) *pgxmock.Rows {
	now := time.Now()
	return pgxmock.NewRows(versionCols()).AddRow(
		uuid.New(), siteID, jobID, version, title, "test-article", "# Test Article\n\nVersioned body.",
		metaTitle, metaDesc, keyword, "https://img.example.com/a.jpg", "alt text", "en", now,
	)
}

func newReviewService(t *testing.T) (*Service, pgxmock.PgxPoolIface) {
	t.Helper()
	cfg := &config.Config{}
	log := logger.New(cfg)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	svc := NewService(cfg, log, &database.Database{Pool: mock}, nil)
	return svc, mock
}

func expectJobHistoryLog(t *testing.T, mock pgxmock.PgxPoolIface) {
	t.Helper()
	mock.ExpectExec(`INSERT INTO workflow_history`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO workflow_logs`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
}

func expectSaveVersion(mock pgxmock.PgxPoolIface) {
	mock.ExpectExec(`INSERT INTO workflow_job_versions`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
}

// --- GetJobReview ---

func TestGetJobReview_Full(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()
	enh := &fakeEnhancer{out: &publisher.ContentEnhancement{
		Content:         "# Test Article\n\nThis is the generated article body with enough content to analyze.\n\n## Sources\n- [x](https://example.com)",
		MetaDescription: "A meta description derived from the article.",
		Keyword:         "test article",
	}}
	reviewer := &fakeSEOReviewer{out: &publisher.SEOReviewReport{
		Score: 82.4, MinScore: 70, Passes: true,
		Title: 100, Meta: 90, Headings: 90, Keyword: 100, Readability: 60,
		InternalLinks: 80, ExternalLinks: 90, EEAT: 70, Images: 60,
		KeywordDensity: 1.5, WordCount: 12, Issues: []string{"readability: 60/100"},
	}}
	svc.SetEnhancer(enh)
	svc.SetSEOReviewer(reviewer)

	mock.ExpectQuery(`SELECT id, site_id, user_id, title,.*FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil))
	mock.ExpectQuery(`SELECT step_name, COALESCE\(metadata::text, '{}'\)\s+FROM workflow_steps\s+WHERE workflow_job_id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewStepRows(jobID))
	mock.ExpectQuery(`SELECT id, site_id, workflow_job_id, version,.*FROM workflow_job_versions\s+WHERE workflow_job_id = \$1 AND site_id = \$2\s+ORDER BY version DESC LIMIT 1`).
		WithArgs(jobID, siteID).WillReturnRows(pgxmock.NewRows(versionCols()))

	detail, err := svc.GetJobReview(context.Background(), siteID, jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Job == nil || detail.Job.Status != JobStatusCompleted {
		t.Error("expected completed job")
	}
	if detail.Article == nil || detail.Article.Title != "Test Article" {
		t.Error("expected article with job title")
	}
	if !strings.Contains(detail.Article.Content, "## Sources") {
		t.Error("expected enhanced content in review article")
	}
	if detail.SEO == nil {
		t.Fatal("expected seo breakdown")
	}
	if detail.SEO.Score != 82.4 || !detail.SEO.Passes || detail.SEO.MinScore != 70 {
		t.Errorf("unexpected seo report: %+v", detail.SEO)
	}
	if len(detail.SEO.Issues) != 1 || detail.SEO.Issues[0] != "readability: 60/100" {
		t.Errorf("unexpected issues: %v", detail.SEO.Issues)
	}
	if detail.Version != 1 {
		t.Errorf("expected version 1, got %d", detail.Version)
	}
	if enh.calls != 1 || reviewer.calls != 1 {
		t.Errorf("expected 1 enhancer call and 1 reviewer call, got %d/%d", enh.calls, reviewer.calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetJobReview_VersionOverrides(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()
	svc.SetEnhancer(&fakeEnhancer{out: &publisher.ContentEnhancement{
		Content: "enhanced", MetaDescription: "enhanced-meta", Keyword: "enhanced-kw",
	}})

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil))
	mock.ExpectQuery(`FROM workflow_steps\s+WHERE workflow_job_id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewStepRows(jobID))
	mock.ExpectQuery(`FROM workflow_job_versions\s+WHERE workflow_job_id = \$1 AND site_id = \$2\s+ORDER BY version DESC LIMIT 1`).
		WithArgs(jobID, siteID).WillReturnRows(versionRow(jobID, siteID, 2, "Edited Title", "edited kw", "Edited Meta Title", "Edited meta description"))

	detail, err := svc.GetJobReview(context.Background(), siteID, jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Article.Title != "Edited Title" {
		t.Errorf("version title not applied: %q", detail.Article.Title)
	}
	if detail.Article.Keyword != "enhanced-kw" {
		t.Errorf("expected enhancer keyword to win, got %q", detail.Article.Keyword)
	}
	if detail.Article.FeaturedImageURL != "https://img.example.com/a.jpg" {
		t.Error("version featured image not applied")
	}
	if detail.Version != 1 {
		t.Errorf("expected job revision version, got %d", detail.Version)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetJobReview_NoEnhancerNoReviewer(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil))
	mock.ExpectQuery(`FROM workflow_steps\s+WHERE workflow_job_id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewStepRows(jobID))
	mock.ExpectQuery(`FROM workflow_job_versions\s+WHERE workflow_job_id = \$1 AND site_id = \$2\s+ORDER BY version DESC LIMIT 1`).
		WithArgs(jobID, siteID).WillReturnRows(pgxmock.NewRows(versionCols()))

	detail, err := svc.GetJobReview(context.Background(), siteID, jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.SEO != nil {
		t.Error("expected nil seo when no reviewer")
	}
	if !strings.Contains(detail.Article.Content, "generated article body") {
		t.Error("expected raw draft content")
	}
}

func TestGetJobReview_ArticleNotFound(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil))
	empty := pgxmock.NewRows([]string{"step_name", "metadata"}).AddRow("finished", `{"ai_content": ""}`)
	mock.ExpectQuery(`FROM workflow_steps\s+WHERE workflow_job_id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(empty)

	_, err := svc.GetJobReview(context.Background(), siteID, jobID)
	if !errors.Is(err, ErrReviewArticleNotFound) {
		t.Errorf("expected ErrReviewArticleNotFound, got %v", err)
	}
}

func TestGetJobReview_EnhancerFailsOpen(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()
	svc.SetEnhancer(&fakeEnhancer{err: errors.New("boom")})
	svc.SetSEOReviewer(&fakeSEOReviewer{out: &publisher.SEOReviewReport{Score: 50, MinScore: 70, Passes: false}})

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil))
	mock.ExpectQuery(`FROM workflow_steps\s+WHERE workflow_job_id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewStepRows(jobID))
	mock.ExpectQuery(`FROM workflow_job_versions\s+WHERE workflow_job_id = \$1 AND site_id = \$2\s+ORDER BY version DESC LIMIT 1`).
		WithArgs(jobID, siteID).WillReturnRows(pgxmock.NewRows(versionCols()))

	detail, err := svc.GetJobReview(context.Background(), siteID, jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(detail.Article.Content, "generated article body") {
		t.Error("expected raw draft content when enhancer fails")
	}
	if detail.SEO == nil || detail.SEO.Score != 50 {
		t.Errorf("expected seo from reviewer, got %+v", detail.SEO)
	}
}

// --- ApproveJob ---

func TestApproveJob_Success(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()
	pubID := uuid.New()
	fakePub := &fakeGeneratedPublisher{pub: &publisher.Publication{
		ID: pubID, SiteID: siteID, Title: "Test Article", Slug: "test-article", Status: "published", Language: "en",
	}}
	svc.publisherSvc = fakePub
	svc.SetEnhancer(&fakeEnhancer{out: &publisher.ContentEnhancement{
		Content: "enhanced body", MetaDescription: "enhanced meta", Keyword: "test article",
	}})

	userID := uuid.New()
	metaTitle := "Meta Title Override"
	req := ApproveReviewRequest{MetaTitle: &metaTitle}

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil))
	mock.ExpectQuery(`FROM workflow_steps\s+WHERE workflow_job_id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewStepRows(jobID))
	mock.ExpectQuery(`FROM workflow_job_versions\s+WHERE workflow_job_id = \$1 AND site_id = \$2\s+ORDER BY version DESC LIMIT 1`).
		WithArgs(jobID, siteID).WillReturnRows(pgxmock.NewRows(versionCols()))
	mock.ExpectExec(`UPDATE workflow_jobs\s+SET review_status = 'published', approved_by = \$1, approved_at = \$2,.*publication_id = \$3.*WHERE id = \$4 AND site_id = \$5`).
		WithArgs(userID, pgxmock.AnyArg(), pubID, jobID, siteID).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	expectSaveVersion(mock)
	expectJobHistoryLog(t, mock)

	result, err := svc.ApproveJob(context.Background(), siteID, jobID, userID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fakePub.calls != 1 {
		t.Errorf("expected 1 publish call, got %d", fakePub.calls)
	}
	if fakePub.lastReq.Title != "Test Article" {
		t.Errorf("expected job title, got %q", fakePub.lastReq.Title)
	}
	if fakePub.lastReq.MetaTitle != "Meta Title Override" {
		t.Errorf("approve override not applied: %q", fakePub.lastReq.MetaTitle)
	}
	if fakePub.lastReq.MetaDescription != "enhanced meta" {
		t.Errorf("enhancer meta not forwarded: %q", fakePub.lastReq.MetaDescription)
	}
	if fakePub.lastReq.Source != "workflow" || fakePub.lastReq.SourceJobID != jobID {
		t.Error("publish request must carry workflow source + job id")
	}
	if result.Publication == nil || result.Publication.ID != pubID {
		t.Error("expected publication in result")
	}
	if result.Job.ReviewStatus != ReviewStatusPublished {
		t.Errorf("expected published review status, got %s", result.Job.ReviewStatus)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestApproveJob_GateBlocks(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()
	fakePub := &fakeGeneratedPublisher{err: publisher.ErrSEOPublishBlocked}
	svc.publisherSvc = fakePub

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil))
	mock.ExpectQuery(`FROM workflow_steps\s+WHERE workflow_job_id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewStepRows(jobID))
	mock.ExpectQuery(`FROM workflow_job_versions\s+WHERE workflow_job_id = \$1 AND site_id = \$2\s+ORDER BY version DESC LIMIT 1`).
		WithArgs(jobID, siteID).WillReturnRows(pgxmock.NewRows(versionCols()))

	_, err := svc.ApproveJob(context.Background(), siteID, jobID, uuid.New(), ApproveReviewRequest{})
	if !errors.Is(err, publisher.ErrSEOPublishBlocked) {
		t.Errorf("expected ErrSEOPublishBlocked, got %v", err)
	}
	if fakePub.calls != 1 {
		t.Error("expected the publish funnel to run the gate")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestApproveJob_AlreadyPublished(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()
	pubID := uuid.New()
	svc.publisherSvc = &fakeGeneratedPublisher{pub: &publisher.Publication{ID: pubID}}

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusPublished, &pubID))

	_, err := svc.ApproveJob(context.Background(), siteID, jobID, uuid.New(), ApproveReviewRequest{})
	if !errors.Is(err, ErrJobAlreadyPublished) {
		t.Errorf("expected ErrJobAlreadyPublished, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestApproveJob_Rejected(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()
	svc.publisherSvc = &fakeGeneratedPublisher{pub: &publisher.Publication{ID: uuid.New()}}

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusRejected, nil))

	_, err := svc.ApproveJob(context.Background(), siteID, jobID, uuid.New(), ApproveReviewRequest{})
	if !errors.Is(err, ErrJobReviewRejected) {
		t.Errorf("expected ErrJobReviewRejected, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- RejectJob ---

func TestRejectJob_Success(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()
	userID := uuid.New()

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil))
	mock.ExpectExec(`UPDATE workflow_jobs\s+SET review_status = 'rejected', rejected_by = \$1, rejected_at = \$2,.*rejection_reason = \$3.*WHERE id = \$4 AND site_id = \$5`).
		WithArgs(userID, pgxmock.AnyArg(), "missing sources", jobID, siteID).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	expectJobHistoryLog(t, mock)
	rejectedRow := reviewJobRow(jobID, siteID, ReviewStatusRejected, nil)
	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(rejectedRow)

	job, err := svc.RejectJob(context.Background(), siteID, jobID, userID, "missing sources")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.ReviewStatus != ReviewStatusRejected {
		t.Errorf("expected rejected status, got %s", job.ReviewStatus)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRejectJob_EmptyReason(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil))

	_, err := svc.RejectJob(context.Background(), siteID, jobID, uuid.New(), "   ")
	if !errors.Is(err, ErrReviewReasonRequired) {
		t.Errorf("expected ErrReviewReasonRequired, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRejectJob_Published(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()
	pubID := uuid.New()

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusPublished, &pubID))

	_, err := svc.RejectJob(context.Background(), siteID, jobID, uuid.New(), "reason")
	if !errors.Is(err, ErrJobAlreadyPublished) {
		t.Errorf("expected ErrJobAlreadyPublished, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- RegenerateJob ---

func TestRegenerateJob_Success(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusRejected, nil))
	mock.ExpectQuery(`FROM workflow_steps\s+WHERE workflow_job_id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewStepRows(jobID))
	mock.ExpectQuery(`FROM workflow_job_versions\s+WHERE workflow_job_id = \$1 AND site_id = \$2\s+ORDER BY version DESC LIMIT 1`).
		WithArgs(jobID, siteID).WillReturnRows(versionRow(jobID, siteID, 2, "Old Title", "old kw", "", ""))
	expectSaveVersion(mock)
	mock.ExpectExec(`UPDATE workflow_jobs\s+SET status = 'draft', review_status = 'generated', revision = \$1,.*approved_by = NULL,.*WHERE id = \$2 AND site_id = \$3`).
		WithArgs(3, jobID, siteID).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	expectJobHistoryLog(t, mock)
	resetRow := reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil)
	resetRow = pgxmock.NewRows(reviewJobCols).AddRow(
		jobID, siteID, &wfJobUserID, "Test Article", "article", "en", "", JobStatus("draft"), "", float64(0),
		5, 0, "", "", []string{}, "", nil, nil, nil, "", 0, 3, false, false, nil, nil, nil, &wfJobUserID, time.Now(), time.Now(),
		ReviewStatusGenerated, 3, nil, nil, nil, nil, "",
	)
	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(resetRow)

	job, err := svc.RegenerateJob(context.Background(), siteID, jobID, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.Status != JobStatusDraft {
		t.Errorf("expected draft status, got %s", job.Status)
	}
	if job.Revision != 3 {
		t.Errorf("expected revision 3, got %d", job.Revision)
	}
	if job.ReviewStatus != ReviewStatusGenerated {
		t.Errorf("expected generated review status, got %s", job.ReviewStatus)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRegenerateJob_Published(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()
	pubID := uuid.New()

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusPublished, &pubID))

	_, err := svc.RegenerateJob(context.Background(), siteID, jobID, uuid.New())
	if !errors.Is(err, ErrJobAlreadyPublished) {
		t.Errorf("expected ErrJobAlreadyPublished, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- SaveDraftMeta ---

func TestSaveDraftMeta_Success(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()
	title := "Edited Title"
	kw := "new keyword"

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil))
	mock.ExpectQuery(`FROM workflow_steps\s+WHERE workflow_job_id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewStepRows(jobID))
	mock.ExpectQuery(`FROM workflow_job_versions\s+WHERE workflow_job_id = \$1 AND site_id = \$2\s+ORDER BY version DESC LIMIT 1`).
		WithArgs(jobID, siteID).WillReturnRows(pgxmock.NewRows(versionCols()))
	expectSaveVersion(mock)
	mock.ExpectExec(`INSERT INTO workflow_logs`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil))

	job, err := svc.SaveDraftMeta(context.Background(), siteID, jobID, SaveDraftRequest{Title: &title, Keyword: &kw})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job == nil {
		t.Fatal("expected job")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- ListJobVersions ---

func TestListJobVersions(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil))
	rows := versionRow(jobID, siteID, 2, "V2", "kw2", "", "")
	rows.AddRow(uuid.New(), siteID, jobID, 1, "V1", "v1", "# V1", "", "", "kw1", "", "", "en", time.Now())
	mock.ExpectQuery(`FROM workflow_job_versions\s+WHERE workflow_job_id = \$1 AND site_id = \$2\s+ORDER BY version DESC`).
		WithArgs(jobID, siteID).WillReturnRows(rows)

	versions, err := svc.ListJobVersions(context.Background(), siteID, jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if versions[0].Version != 2 || versions[1].Version != 1 {
		t.Errorf("expected desc order, got %d/%d", versions[0].Version, versions[1].Version)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestListJobVersions_Empty(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID, siteID := uuid.New(), uuid.New()

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(jobID, siteID).WillReturnRows(reviewJobRow(jobID, siteID, ReviewStatusGenerated, nil))
	mock.ExpectQuery(`FROM workflow_job_versions\s+WHERE workflow_job_id = \$1 AND site_id = \$2\s+ORDER BY version DESC`).
		WithArgs(jobID, siteID).WillReturnRows(pgxmock.NewRows(versionCols()))

	versions, err := svc.ListJobVersions(context.Background(), siteID, jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if versions == nil || len(versions) != 0 {
		t.Error("expected empty (non-nil) version list")
	}
}

// --- Handler ---

func reviewHandlerWithDB(t *testing.T, svc *Service) *Handler {
	t.Helper()
	cfg := &config.Config{}
	log := logger.New(cfg)
	return NewHandler(svc, log)
}

func TestHandler_GetJobReview(t *testing.T) {
	svc, _ := setupMockDB(t)
	h := reviewHandlerWithDB(t, svc)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/workflow/"+uuid.New().String()+"/review", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetJobReview).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/workflow/invalid/review", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.GetJobReview).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		cfg := &config.Config{}
		noDB := NewService(cfg, logger.New(cfg), nil, nil)
		h := reviewHandlerWithDB(t, noDB)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/workflow/"+uuid.New().String()+"/review", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetJobReview).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 (no db), got %d", rec.Code)
		}
	})
}

func TestHandler_ApproveJobReview_GateBlocked(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID := uuid.New()
	svc.publisherSvc = &fakeGeneratedPublisher{err: publisher.ErrSEOPublishBlocked}
	h := reviewHandlerWithDB(t, svc)

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(reviewJobRow(jobID, uuid.New(), ReviewStatusGenerated, nil))
	mock.ExpectQuery(`FROM workflow_steps\s+WHERE workflow_job_id = \$1 AND site_id = \$2`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(reviewStepRows(jobID))
	mock.ExpectQuery(`FROM workflow_job_versions\s+WHERE workflow_job_id = \$1 AND site_id = \$2\s+ORDER BY version DESC LIMIT 1`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows(versionCols()))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/workflow/"+jobID.String()+"/review/approve",
		strings.NewReader(`{}`))
	req = withSiteID(req)
	req = withUserID(req)
	req = withChiParams(req, map[string]string{"id": jobID.String()})
	rest.AdaptHandler(h.ApproveJobReview).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "SEO_SCORE_BELOW_MINIMUM") {
		t.Errorf("expected SEO_SCORE_BELOW_MINIMUM code, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestHandler_ApproveJobReview_MissingUser(t *testing.T) {
	svc, _ := setupMockDB(t)
	h := reviewHandlerWithDB(t, svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/workflow/"+uuid.New().String()+"/review/approve",
		strings.NewReader(`{}`))
	req = withSiteID(req)
	req = withChiParams(req, map[string]string{"id": uuid.New().String()})
	rest.AdaptHandler(h.ApproveJobReview).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandler_RejectJobReview_EmptyReason(t *testing.T) {
	svc, mock := newReviewService(t)
	jobID := uuid.New()
	h := reviewHandlerWithDB(t, svc)

	mock.ExpectQuery(`FROM workflow_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(reviewJobRow(jobID, uuid.New(), ReviewStatusGenerated, nil))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/workflow/"+jobID.String()+"/review/reject",
		strings.NewReader(`{"reason": ""}`))
	req = withSiteID(req)
	req = withUserID(req)
	req = withChiParams(req, map[string]string{"id": jobID.String()})
	rest.AdaptHandler(h.RejectJobReview).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "rejection reason is required") {
		t.Errorf("expected reason message, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestHandler_ListJobVersions_NoDB(t *testing.T) {
	svc, _ := setupMockDB(t)
	h := reviewHandlerWithDB(t, svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/workflow/"+uuid.New().String()+"/versions", nil)
	req = withSiteID(req)
	req = withChiParams(req, map[string]string{"id": uuid.New().String()})
	rest.AdaptHandler(h.ListJobVersions).ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 (no db), got %d", rec.Code)
	}
}
