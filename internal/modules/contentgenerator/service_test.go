package contentgenerator

import (
	"context"
	"testing"

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

// --- Validation ---

func TestCreateJob_InvalidLanguage(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateJob(context.Background(), uuid.New(), uuid.New(), CreateJobRequest{
		Language: "fr",
		Priority: 5,
	})
	if err != ErrInvalidLanguage {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestCreateJob_InvalidPriority(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateJob(context.Background(), uuid.New(), uuid.New(), CreateJobRequest{
		Language: "en",
		Priority: 0,
	})
	if err != ErrInvalidPriority {
		t.Errorf("expected ErrInvalidPriority, got %v", err)
	}
}

func TestCreateJob_InvalidPriorityHigh(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateJob(context.Background(), uuid.New(), uuid.New(), CreateJobRequest{
		Language: "en",
		Priority: 11,
	})
	if err != ErrInvalidPriority {
		t.Errorf("expected ErrInvalidPriority, got %v", err)
	}
}

// --- Not Found ---

func TestGetJob_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM generation_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetJob(context.Background(), siteID, jobID)
	if err != ErrJobNotFound {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}
}

// --- Empty Results ---

func TestListJobs_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM generation_jobs WHERE`).
		WithArgs(siteID, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "article_job_id", "research_job_id", "priority", "language",
			"category", "article_type", "expected_size",
			"style_slug", "keywords", "status",
			"progress", "current_stage", "error_message",
			"retry_count", "max_retries",
			"started_at", "completed_at", "cancelled_at", "created_by", "created_at", "updated_at",
		}))

	jobs, err := svc.ListJobs(context.Background(), siteID, "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestListPipeline_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM generation_pipeline WHERE`).
		WithArgs(jobID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "generation_job_id", "stage", "status", "progress",
			"started_at", "completed_at", "duration_ms", "error_message",
			"retry_count", "metadata", "created_at", "updated_at",
		}))

	pipeline, err := svc.ListPipeline(context.Background(), jobID)
	if err != nil {
		t.Fatalf("ListPipeline failed: %v", err)
	}
	if len(pipeline) != 0 {
		t.Errorf("expected 0 pipeline entries, got %d", len(pipeline))
	}
}

func TestListLogs_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM generation_pipeline_logs WHERE`).
		WithArgs(jobID, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "generation_job_id", "stage", "level", "message",
			"details", "duration_ms", "created_at",
		}))

	logs, err := svc.ListLogs(context.Background(), jobID, "", "", 0, 0)
	if err != nil {
		t.Fatalf("ListLogs failed: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}

func TestGetQualityGates_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM generation_quality_gates WHERE`).
		WithArgs(jobID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "generation_job_id", "stage", "status", "seo_score", "readability_score",
			"eeat_score", "keyword_density", "heading_score", "internal_linking_score",
			"required_content_passed", "min_size_passed", "metadata_passed", "overall_passed",
			"report", "checked_by", "checked_at", "created_at",
		}))

	gates, err := svc.GetQualityGates(context.Background(), jobID)
	if err != nil {
		t.Fatalf("GetQualityGates failed: %v", err)
	}
	if len(gates) != 0 {
		t.Errorf("expected 0 gates, got %d", len(gates))
	}
}

func TestGetStats_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM generation_stats WHERE`).
		WithArgs(siteID, pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	stats, err := svc.GetStats(context.Background(), siteID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
}

// --- No DB ---

func TestCreateJob_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.CreateJob(context.Background(), uuid.New(), uuid.New(), CreateJobRequest{
		Language: "en",
		Priority: 5,
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestListJobs_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.ListJobs(context.Background(), uuid.New(), "", "", "", 0, 0)
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestStartJob_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.StartJob(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestListPipeline_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.ListPipeline(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestListLogs_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.ListLogs(context.Background(), uuid.New(), "", "", 0, 0)
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestCheckQualityGate_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.CheckQualityGate(context.Background(), uuid.New(), uuid.New(), QualityGateRequest{})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestGetStats_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.GetStats(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestAssemblePrompt_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.AssemblePrompt(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}
