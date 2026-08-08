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
			"source_job_id", "publication_id", "scheduled_for", "error_message", "retry_count",
			"max_retries", "generate_pt", "generate_en", "started_at", "completed_at",
			"cancelled_at", "created_by", "created_at", "updated_at",
		}).AddRow(uuid.New(), uuid.New(), nil, "test", "article", "pt", "", "draft", "", 0,
			5, 0, "", "", []string{}, "", nil, nil, nil, "", 0, 3, false, false, nil, nil, nil, nil, nowTime, nowTime))

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
			"source_job_id", "publication_id", "scheduled_for", "error_message", "retry_count",
			"max_retries", "generate_pt", "generate_en", "started_at", "completed_at",
			"cancelled_at", "created_by", "created_at", "updated_at",
		}).AddRow(uuid.New(), uuid.New(), nil, "test", "article", "pt", "", "draft", "", 0,
			5, 0, "", "", []string{}, "", nil, nil, nil, "", 0, 3, false, false, nil, nil, nil, nil, nowTime, nowTime))

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
			"source_job_id", "publication_id", "scheduled_for", "error_message", "retry_count",
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

func TestExecuteAction_GenerateArticle_AutoStart(t *testing.T) {
	svc, mock := setupMockDB(t)

	nowTime := now()
	jobID := uuid.New()
	siteID := uuid.New()
	userID := uuid.New()
	steps := AllWorkflowSteps

mock.ExpectExec(`INSERT INTO workflow_jobs`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), "Test Article 2", "article", "pt", "", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	for range steps {
		mock.ExpectExec(`INSERT INTO workflow_steps`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}
	mock.ExpectExec(`INSERT INTO workflow_logs`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	jobRow := pgxmock.NewRows([]string{
		"id", "site_id", "user_id", "title", "content_type",
		"language", "target_language", "status", "current_step", "progress",
		"priority", "word_count", "tone", "audience", "keywords", "style_slug",
		"source_job_id", "publication_id", "scheduled_for", "error_message", "retry_count",
		"max_retries", "generate_pt", "generate_en", "started_at", "completed_at",
		"cancelled_at", "created_by", "created_at", "updated_at",
	}).AddRow(jobID, siteID, &userID, "Test Article 2", "article", "pt", "", JobStatus("draft"), "", float64(0),
		5, 0, "", "", []string{}, "", nil, nil, nil, "", 0, 3, false, false, nil, nil, nil, &userID, nowTime, nowTime)

	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), siteID).
		WillReturnRows(jobRow)

	jobRowStart := pgxmock.NewRows([]string{
		"id", "site_id", "user_id", "title", "content_type",
		"language", "target_language", "status", "current_step", "progress",
		"priority", "word_count", "tone", "audience", "keywords", "style_slug",
		"source_job_id", "publication_id", "scheduled_for", "error_message", "retry_count",
		"max_retries", "generate_pt", "generate_en", "started_at", "completed_at",
		"cancelled_at", "created_by", "created_at", "updated_at",
	}).AddRow(jobID, siteID, &userID, "Test Article 2", "article", "pt", "", JobStatus("draft"), "", float64(0),
		5, 0, "", "", []string{}, "", nil, nil, nil, "", 0, 3, false, false, nil, nil, nil, &userID, nowTime, nowTime)

	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), siteID).
		WillReturnRows(jobRowStart)

	mock.ExpectExec(`UPDATE workflow_jobs SET status = 'running'`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), jobID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectExec(`UPDATE workflow_steps SET status = 'running'`).
		WithArgs(pgxmock.AnyArg(), jobID, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectExec(`INSERT INTO workflow_history`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`INSERT INTO workflow_logs`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	runningRow := pgxmock.NewRows([]string{
		"id", "site_id", "user_id", "title", "content_type",
		"language", "target_language", "status", "current_step", "progress",
		"priority", "word_count", "tone", "audience", "keywords", "style_slug",
		"source_job_id", "publication_id", "scheduled_for", "error_message", "retry_count",
		"max_retries", "generate_pt", "generate_en", "started_at", "completed_at",
		"cancelled_at", "created_by", "created_at", "updated_at",
	}).AddRow(jobID, siteID, &userID, "Test Article 2", "article", "pt", "", JobStatus("running"), "research", float64(0),
		5, 0, "", "", []string{}, "", nil, nil, nil, "", 0, 3, false, false, nowTime, nil, nil, &userID, nowTime, nowTime)
	mock.ExpectQuery(`SELECT .+ FROM workflow_jobs WHERE`).
		WithArgs(pgxmock.AnyArg(), siteID).
		WillReturnRows(runningRow)

	job, err := svc.ExecuteAction(context.Background(), siteID, userID, AutomationAction{
		Action: "generate_article",
		Title:  "Test Article 2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.Status != JobStatusRunning {
		t.Errorf("expected job to auto-start as running, got %s", job.Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
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

func TestAddLog_WritesToWorkflowLogsTable(t *testing.T) {
	svc, mock := setupMockDB(t)
	jobID := uuid.New()

	mock.ExpectExec(`INSERT INTO workflow_logs \(id, site_id, workflow_job_id, step, level, message, details, duration_ms, created_at\)`).
		WithArgs(pgxmock.AnyArg(), jobID, pgxmock.AnyArg(), "info", "workflow job created", pgxmock.AnyArg(), int64(0), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	svc.addLog(context.Background(), svc.db.Pool, jobID, "", "info", "workflow job created", nil, 0)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("addLog did not write to workflow_logs: %v", err)
	}
}
