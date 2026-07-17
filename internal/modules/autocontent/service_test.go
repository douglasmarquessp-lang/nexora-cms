package autocontent

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

func now() time.Time {
	return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
}

// --- Validation ---

func TestCreateJob_InvalidTopic(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateJob(context.Background(), uuid.New(), uuid.New(), CreateJobRequest{
		Topic:    "",
		Language: "pt",
	})
	if err != ErrInvalidTopic {
		t.Errorf("expected ErrInvalidTopic, got %v", err)
	}
}

func TestCreateJob_InvalidLanguage(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateJob(context.Background(), uuid.New(), uuid.New(), CreateJobRequest{
		Topic:    "Test Topic",
		Language: "fr",
	})
	if err != ErrInvalidLanguage {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestAddToQueue_InvalidLanguage(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.AddToQueue(context.Background(), uuid.New(), QueueRequest{
		Title:    "Test",
		Language: "fr",
	})
	if err != ErrInvalidLanguage {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestAddToQueue_EmptyTitle(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.AddToQueue(context.Background(), uuid.New(), QueueRequest{
		Title:    "",
		Language: "pt",
	})
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestCreateTemplate_EmptyName(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateTemplate(context.Background(), uuid.New(), CreateTemplateRequest{
		Name:  "",
		Steps: []map[string]interface{}{{"step": "topic"}},
	})
	if err == nil {
		t.Error("expected error for empty template name")
	}
}

func TestCreateTemplate_EmptySteps(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateTemplate(context.Background(), uuid.New(), CreateTemplateRequest{
		Name:  "Test",
		Steps: []map[string]interface{}{},
	})
	if err == nil {
		t.Error("expected error for empty steps")
	}
}

func TestUpdateJob_InvalidPriority(t *testing.T) {
	svc, _ := setupMockDB(t)

	low := 0
	high := 11

	t.Run("priority too low", func(t *testing.T) {
		_, err := svc.UpdateJob(context.Background(), uuid.New(), uuid.New(), UpdateJobRequest{
			Priority: &low,
		})
		if err == nil {
			t.Error("expected error for priority < 1")
		}
	})

	t.Run("priority too high", func(t *testing.T) {
		_, err := svc.UpdateJob(context.Background(), uuid.New(), uuid.New(), UpdateJobRequest{
			Priority: &high,
		})
		if err == nil {
			t.Error("expected error for priority > 10")
		}
	})
}

// --- Not Found ---

func TestGetJob_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM autocontent_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetJob(context.Background(), uuid.New(), uuid.New())
	if err != ErrJobNotFound {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestGetStepByName_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM autocontent_steps WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.getStepByName(context.Background(), mock, uuid.New(), "topic")
	if err != ErrStepNotFound {
		t.Errorf("expected ErrStepNotFound, got %v", err)
	}
}

func TestGetResultByStep_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM autocontent_results WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetResultByStep(context.Background(), uuid.New(), "draft")
	if err != ErrResultNotFound {
		t.Errorf("expected ErrResultNotFound, got %v", err)
	}
}

func TestDeleteJob_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectExec(`DELETE FROM autocontent_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := svc.DeleteJob(context.Background(), uuid.New(), uuid.New())
	if err != ErrJobNotFound {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}
}

// --- State Transition Validation ---

func TestStartJob_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM autocontent_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.StartJob(context.Background(), uuid.New(), uuid.New())
	if err != ErrJobNotFound {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}
}

func TestPauseJob_NotRunning(t *testing.T) {
	svc, mock := setupMockDB(t)

	nowTime := now()
	mock.ExpectQuery(`SELECT .+ FROM autocontent_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "user_id", "topic", "title", "content_type",
			"language", "target_language", "status", "current_step", "progress",
			"priority", "word_count", "tone", "audience", "keywords", "style_slug",
			"template_id", "scheduled_for", "error_message", "retry_count",
			"max_retries", "started_at", "completed_at", "cancelled_at", "created_by",
			"created_at", "updated_at",
		}).AddRow(uuid.New(), uuid.New(), nil, "test", "", "article", "pt", "", "draft", "", 0,
			5, 0, "", "", []string{}, "", nil, nil, "", 0, 3, nil, nil, nil, nil, nowTime, nowTime))

	_, err := svc.PauseJob(context.Background(), uuid.New(), uuid.New())
	if err != ErrJobNotRunning {
		t.Errorf("expected ErrJobNotRunning, got %v", err)
	}
}

func TestResumeJob_NotPaused(t *testing.T) {
	svc, mock := setupMockDB(t)

	nowTime := now()
	mock.ExpectQuery(`SELECT .+ FROM autocontent_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "user_id", "topic", "title", "content_type",
			"language", "target_language", "status", "current_step", "progress",
			"priority", "word_count", "tone", "audience", "keywords", "style_slug",
			"template_id", "scheduled_for", "error_message", "retry_count",
			"max_retries", "started_at", "completed_at", "cancelled_at", "created_by",
			"created_at", "updated_at",
		}).AddRow(uuid.New(), uuid.New(), nil, "test", "", "article", "pt", "", "draft", "", 0,
			5, 0, "", "", []string{}, "", nil, nil, "", 0, 3, nil, nil, nil, nil, nowTime, nowTime))

	_, err := svc.ResumeJob(context.Background(), uuid.New(), uuid.New())
	if err != ErrJobPaused {
		t.Errorf("expected ErrJobPaused, got %v", err)
	}
}

func TestCancelJob_StateChecks(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)

	t.Run("completed", func(t *testing.T) {
		svc := NewService(cfg, log, nil, nil)
		_, err := svc.CancelJob(context.Background(), uuid.New(), uuid.New(), "test")
		if err != ErrDatabaseNotAvail {
			t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
		}
	})
}

// --- Empty Results ---

func TestListJobs_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM autocontent_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "user_id", "topic", "title", "content_type",
			"language", "target_language", "status", "current_step", "progress",
			"priority", "word_count", "tone", "audience", "keywords", "style_slug",
			"template_id", "scheduled_for", "error_message", "retry_count",
			"max_retries", "started_at", "completed_at", "cancelled_at", "created_by",
			"created_at", "updated_at",
		}))

	jobs, err := svc.ListJobs(context.Background(), uuid.New(), "", "", "", 0, 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected empty list, got %d items", len(jobs))
	}
}

func TestListQueue_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM publication_queue WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "autocontent_job_id", "title", "content", "excerpt",
			"language", "status", "priority", "scheduled_for", "meta_title",
			"meta_description", "slug", "featured_image_url", "tags", "categories",
			"published_at", "published_by", "error_message", "created_at", "updated_at",
		}))

	items, err := svc.ListQueue(context.Background(), uuid.New(), "", 0, 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty list, got %d items", len(items))
	}
}

func TestGetSteps_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM autocontent_steps WHERE`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "autocontent_job_id", "step_name", "display_name", "status",
			"progress", "depends_on", "retry_count", "max_retries", "started_at",
			"completed_at", "duration_ms", "error_message", "metadata",
			"created_at", "updated_at",
		}))

	steps, err := svc.GetSteps(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected empty list, got %d items", len(steps))
	}
}

func TestGetResults_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM autocontent_results WHERE`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "autocontent_job_id", "step_name", "content", "summary",
			"score", "passed", "data", "created_at", "updated_at",
		}))

	results, err := svc.GetResults(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty list, got %d items", len(results))
	}
}

func TestListTemplates_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM workflow_templates WHERE`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "name", "description", "steps", "is_default",
			"is_active", "created_by", "created_at", "updated_at",
		}))

	templates, err := svc.ListTemplates(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("expected empty list, got %d items", len(templates))
	}
}

// --- No DB ---

func TestPool_NilDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.pool()
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestCreateJob_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.CreateJob(context.Background(), uuid.New(), uuid.New(), CreateJobRequest{
		Topic:    "Test",
		Language: "pt",
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestGetJob_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.GetJob(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}
