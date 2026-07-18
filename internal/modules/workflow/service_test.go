package workflow

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

func TestCreateJob_InvalidTitle(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateJob(context.Background(), uuid.New(), uuid.New(), CreateJobRequest{
		Title:    "",
		Language: "pt",
	})
	if err != ErrInvalidTitle {
		t.Errorf("expected ErrInvalidTitle, got %v", err)
	}
}

func TestCreateJob_InvalidLanguage(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateJob(context.Background(), uuid.New(), uuid.New(), CreateJobRequest{
		Title:    "Test Article",
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

	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
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

	mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.getStepByName(context.Background(), mock, uuid.New(), "research")
	if err != ErrStepNotFound {
		t.Errorf("expected ErrStepNotFound, got %v", err)
	}
}

func TestDeleteJob_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectExec(`DELETE FROM workflow_jobs WHERE`).
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

	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
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
	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "user_id", "title", "content_type",
			"language", "target_language", "status", "current_step", "progress",
			"priority", "word_count", "tone", "audience", "keywords", "style_slug",
			"source_job_id", "scheduled_for", "error_message", "retry_count",
			"max_retries", "generate_pt", "generate_en", "started_at", "completed_at",
			"cancelled_at", "created_by", "created_at", "updated_at",
		}).AddRow(uuid.New(), uuid.New(), nil, "test", "article", "pt", "", "draft", "", 0,
			5, 0, "", "", []string{}, "", nil, nil, "", 0, 3, false, false, nil, nil, nil, nil, nowTime, nowTime))

	_, err := svc.PauseJob(context.Background(), uuid.New(), uuid.New())
	if err != ErrJobNotRunning {
		t.Errorf("expected ErrJobNotRunning, got %v", err)
	}
}

func TestResumeJob_NotPaused(t *testing.T) {
	svc, mock := setupMockDB(t)

	nowTime := now()
	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "user_id", "title", "content_type",
			"language", "target_language", "status", "current_step", "progress",
			"priority", "word_count", "tone", "audience", "keywords", "style_slug",
			"source_job_id", "scheduled_for", "error_message", "retry_count",
			"max_retries", "generate_pt", "generate_en", "started_at", "completed_at",
			"cancelled_at", "created_by", "created_at", "updated_at",
		}).AddRow(uuid.New(), uuid.New(), nil, "test", "article", "pt", "", "draft", "", 0,
			5, 0, "", "", []string{}, "", nil, nil, "", 0, 3, false, false, nil, nil, nil, nil, nowTime, nowTime))

	_, err := svc.ResumeJob(context.Background(), uuid.New(), uuid.New())
	if err != ErrJobPaused {
		t.Errorf("expected ErrJobPaused, got %v", err)
	}
}

func TestCancelJob_StateChecks(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)

	t.Run("no db", func(t *testing.T) {
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

	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "user_id", "title", "content_type",
			"language", "target_language", "status", "current_step", "progress",
			"priority", "word_count", "tone", "audience", "keywords", "style_slug",
			"source_job_id", "scheduled_for", "error_message", "retry_count",
			"max_retries", "generate_pt", "generate_en", "started_at", "completed_at",
			"cancelled_at", "created_by", "created_at", "updated_at",
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

	mock.ExpectQuery(`SELECT .+ FROM workflow_queue WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "workflow_job_id", "title", "content", "excerpt",
			"language", "status", "priority", "scheduled_for", "is_paused",
			"retry_count", "max_retries", "meta_title", "meta_description",
			"slug", "featured_image_url", "tags", "categories",
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

	mock.ExpectQuery(`SELECT .+ FROM workflow_steps WHERE`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "workflow_job_id", "step_name", "display_name", "status",
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

// --- Validation methods ---

func TestValidateJobStatus(t *testing.T) {
	svc, _ := setupMockDB(t)

	if !svc.ValidateJobStatus(JobStatusDraft) {
		t.Error("draft should be valid")
	}
	if !svc.ValidateJobStatus(JobStatusRunning) {
		t.Error("running should be valid")
	}
	if svc.ValidateJobStatus("invalid") {
		t.Error("invalid should not be valid")
	}
}

func TestValidateLanguage(t *testing.T) {
	svc, _ := setupMockDB(t)

	if !svc.ValidateLanguage("pt") {
		t.Error("pt should be valid")
	}
	if !svc.ValidateLanguage("en") {
		t.Error("en should be valid")
	}
	if svc.ValidateLanguage("fr") {
		t.Error("fr should not be valid")
	}
}

func TestValidateAutomationAction(t *testing.T) {
	svc, _ := setupMockDB(t)

	if !svc.ValidateAutomationAction("generate_article") {
		t.Error("generate_article should be valid")
	}
	if !svc.ValidateAutomationAction("generate_pt_en") {
		t.Error("generate_pt_en should be valid")
	}
	if svc.ValidateAutomationAction("invalid_action") {
		t.Error("invalid_action should not be valid")
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
		Title:    "Test",
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

func TestExecuteAction_InvalidAction(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.ExecuteAction(context.Background(), uuid.New(), uuid.New(), AutomationAction{
		Action: "invalid_action",
	})
	if err != ErrInvalidAction {
		t.Errorf("expected ErrInvalidAction, got %v", err)
	}
}

func TestExecuteAction_GenerateArticle_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.ExecuteAction(context.Background(), uuid.New(), uuid.New(), AutomationAction{
		Action: "generate_article",
		Title:  "Test Article",
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestValidatePriority(t *testing.T) {
	svc, _ := setupMockDB(t)

	if !svc.ValidatePriority(5) {
		t.Error("5 should be valid")
	}
	if svc.ValidatePriority(0) {
		t.Error("0 should not be valid")
	}
	if svc.ValidatePriority(11) {
		t.Error("11 should not be valid")
	}
}
