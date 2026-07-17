package writer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/pkg/audit"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

func TestNewService(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func setupMockDB(t *testing.T) (*Service, pgxmock.PgxPoolIface) {
	t.Helper()
	cfg := &config.Config{}
	log := logger.New(cfg)

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}

	svc := NewService(cfg, log, &database.Database{Pool: mock}, nil)
	svc.auditLog = audit.New(nil, log)
	return svc, mock
}

func TestService_CreateJob_EmptyHeadline(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateJob(context.Background(), uuid.New(), uuid.New(), CreateArticleJobRequest{
		Headline: "",
		Language: "en",
	})
	if err != ErrHeadlineRequired {
		t.Errorf("expected ErrHeadlineRequired, got %v", err)
	}
}

func TestService_CreateJob_InvalidLanguage(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateJob(context.Background(), uuid.New(), uuid.New(), CreateArticleJobRequest{
		Headline: "Test",
		Language: "fr",
	})
	if err != ErrInvalidLanguage {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestService_GetJob_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM article_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetJob(context.Background(), siteID, jobID)
	if err != ErrWritingJobNotFound {
		t.Errorf("expected ErrWritingJobNotFound, got %v", err)
	}
}

func TestService_ListJobs_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM article_jobs WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "research_job_id", "style_id", "style_name", "language", "status",
			"headline", "seo_title", "slug", "meta_description", "target_audience",
			"tone", "formality", "seo_goal", "desired_size",
			"created_by", "completed_at", "error_message", "created_at", "updated_at",
		}))

	jobs, err := svc.ListJobs(context.Background(), siteID, "")
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}

	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestService_DeleteJob(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT .+ FROM article_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "research_job_id", "style_id", "style_name", "language", "status",
			"headline", "seo_title", "slug", "meta_description", "target_audience",
			"tone", "formality", "seo_goal", "desired_size",
			"created_by", "completed_at", "error_message", "created_at", "updated_at",
		}).AddRow(
			jobID, siteID, nil, nil, "Journalistic", "en", "draft",
			"Test", "", "", "", "", "neutral", "neutral", "", "medium",
			uuid.New(), nil, "", now, now,
		))

	mock.ExpectExec(`UPDATE article_jobs SET deleted_at`).
		WithArgs(jobID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := svc.DeleteJob(context.Background(), siteID, jobID)
	if err != nil {
		t.Fatalf("DeleteJob failed: %v", err)
	}
}

func TestService_CreateOutline(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	jobID := uuid.New()

	mock.ExpectExec(`INSERT INTO article_outlines`).
		WithArgs(pgxmock.AnyArg(), jobID, "h1", "Title", 0, "Content", 0, 0, "", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectExec(`INSERT INTO article_outlines`).
		WithArgs(pgxmock.AnyArg(), jobID, "h2", "Section 1", 1, "", 1, 300, "keyword", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	outlines, err := svc.CreateOutline(context.Background(), jobID, CreateOutlineRequest{
		Sections: []CreateOutlineSection{
			{SectionType: "h1", Title: "Title", Level: 0, Content: "Content", Position: 0},
			{SectionType: "h2", Title: "Section 1", Level: 1, Position: 1, WordCountTarget: 300, Keywords: "keyword"},
		},
	})
	if err != nil {
		t.Fatalf("CreateOutline failed: %v", err)
	}

	if len(outlines) != 2 {
		t.Errorf("expected 2 outline sections, got %d", len(outlines))
	}
	if outlines[0].Title != "Title" {
		t.Errorf("expected 'Title', got %q", outlines[0].Title)
	}
}

func TestService_CreateSection(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	jobID := uuid.New()
	var nilOutlineID *uuid.UUID

	mock.ExpectExec(`INSERT INTO article_sections`).
		WithArgs(pgxmock.AnyArg(), jobID, nilOutlineID, "Intro", "Content", 0, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	section, err := svc.CreateSection(context.Background(), jobID, CreateSectionRequest{
		Title:    "Intro",
		Content:  "Content",
		Position: 0,
	})
	if err != nil {
		t.Fatalf("CreateSection failed: %v", err)
	}

	if section.Title != "Intro" {
		t.Errorf("expected 'Intro', got %q", section.Title)
	}
	if section.Status != SectionStatusPending {
		t.Errorf("expected pending, got %q", section.Status)
	}
}

func TestService_GetSection_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	jobID := uuid.New()
	sectionID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM article_sections WHERE`).
		WithArgs(sectionID, jobID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetSection(context.Background(), jobID, sectionID)
	if err != ErrSectionNotFound {
		t.Errorf("expected ErrSectionNotFound, got %v", err)
	}
}

func TestService_ListVersions_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM article_versions WHERE`).
		WithArgs(jobID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "article_job_id", "version", "headline", "seo_title",
			"slug", "meta_description", "sections", "content", "metadata",
			"summary", "change_log", "created_by", "created_at",
		}))

	versions, err := svc.ListVersions(context.Background(), jobID)
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}

	if len(versions) != 0 {
		t.Errorf("expected 0 versions, got %d", len(versions))
	}
}

func TestService_GetVersion_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	jobID := uuid.New()
	versionID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM article_versions WHERE`).
		WithArgs(versionID, jobID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetVersion(context.Background(), jobID, versionID)
	if err != ErrVersionNotFound {
		t.Errorf("expected ErrVersionNotFound, got %v", err)
	}
}

func TestService_ListStyles_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM writing_styles WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "name", "slug", "description", "config", "is_default", "created_at", "updated_at",
		}))

	styles, err := svc.ListStyles(context.Background(), siteID)
	if err != nil {
		t.Fatalf("ListStyles failed: %v", err)
	}

	if len(styles) != 0 {
		t.Errorf("expected 0 styles, got %d", len(styles))
	}
}

func TestService_UpdateJob_NoChanges(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM article_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.UpdateJob(context.Background(), siteID, jobID, UpdateArticleJobRequest{})
	if err != ErrWritingJobNotFound {
		t.Errorf("expected ErrWritingJobNotFound, got %v", err)
	}
}

func TestService_ListOutline_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM article_outlines WHERE`).
		WithArgs(jobID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "article_job_id", "section_type", "title", "level", "content",
			"position", "word_count_target", "keywords", "created_at",
		}))

	outlines, err := svc.ListOutline(context.Background(), jobID)
	if err != nil {
		t.Fatalf("ListOutline failed: %v", err)
	}

	if len(outlines) != 0 {
		t.Errorf("expected 0 outlines, got %d", len(outlines))
	}
}

func TestService_ListSections_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM article_sections WHERE`).
		WithArgs(jobID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "article_job_id", "outline_id", "title", "content",
			"word_count", "status", "position", "created_at", "updated_at",
		}))

	sections, err := svc.ListSections(context.Background(), jobID)
	if err != nil {
		t.Fatalf("ListSections failed: %v", err)
	}

	if len(sections) != 0 {
		t.Errorf("expected 0 sections, got %d", len(sections))
	}
}
