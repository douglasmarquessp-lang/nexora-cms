package translation

import (
	"context"
	"errors"
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

func jobRow(status string, currentStage *string) *pgxmock.Rows {
	rows := pgxmock.NewRows([]string{
		"id", "site_id", "project_id", "source_post_id", "target_site_id",
		"source_language", "target_language", "title", "content", "status",
		"current_stage", "translation_score", "published_post_id", "publication_id",
		"error_message", "created_by", "created_at", "updated_at", "completed_at",
	})
	now := time.Now()
	cs := ""
	if currentStage != nil {
		cs = *currentStage
	}
	return rows.AddRow(uuid.New(), uuid.New(), nil, nil, uuid.New(),
		"pt", "en", "Título", "Conteúdo", status,
		cs, nil, nil, nil, nil,
		uuid.New(), now, now, nil)
}

// --- Job validations ---

func TestCreateJob_Validation(t *testing.T) {
	svc, _ := setupMockDB(t)
	ctx := context.Background()

	if _, err := svc.CreateJob(ctx, uuid.New(), uuid.New(), CreateJobRequest{Content: "x"}); err != ErrTitleRequired {
		t.Errorf("expected ErrTitleRequired, got %v", err)
	}
	if _, err := svc.CreateJob(ctx, uuid.New(), uuid.New(), CreateJobRequest{Title: "T"}); err != ErrTargetSiteRequired {
		t.Errorf("expected ErrTargetSiteRequired, got %v", err)
	}
	_, err := svc.CreateJob(ctx, uuid.New(), uuid.New(), CreateJobRequest{
		Title: "T", TargetSiteID: uuid.New(), SourceLanguage: "pt", TargetLanguage: "fr",
	})
	if err != ErrInvalidLanguage {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestCreateJob_HappyPath(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()

	mock.ExpectExec(`INSERT INTO translation_jobs`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), "pt", "en", "Título", "Conteúdo", "pending",
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	job, err := svc.CreateJob(ctx, uuid.New(), uuid.New(), CreateJobRequest{
		Title:          "Título",
		Content:        "Conteúdo",
		SourceLanguage: "pt",
		TargetLanguage: "en",
		TargetSiteID:   uuid.New(),
	})
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}
	if job.Status != JobStatusPending {
		t.Errorf("expected pending status, got %s", job.Status)
	}
	if job.SourceLanguage != "pt" || job.TargetLanguage != "en" {
		t.Errorf("unexpected languages: %s -> %s", job.SourceLanguage, job.TargetLanguage)
	}
}

func TestCreateJob_AutoDetectLanguage(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()

	mock.ExpectExec(`INSERT INTO translation_jobs`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), "pt", "en", "Título", "Conteúdo com muitas palavras em português para detectar", "pending",
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	job, err := svc.CreateJob(ctx, uuid.New(), uuid.New(), CreateJobRequest{
		Title:          "Título",
		Content:        "Conteúdo com muitas palavras em português para detectar",
		TargetLanguage: "en",
		TargetSiteID:   uuid.New(),
	})
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}
	if job.SourceLanguage != "pt" {
		t.Errorf("expected auto-detected pt, got %s", job.SourceLanguage)
	}
}

func TestGetJob_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()

	mock.ExpectQuery(`FROM translation_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetJob(ctx, uuid.New(), uuid.New())
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}
}

func TestGetJob_HappyPath(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()

	mock.ExpectQuery(`FROM translation_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(jobRow("completed", nil))

	mock.ExpectQuery(`FROM translation_stages WHERE translation_job_id = \$1`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "translation_job_id", "stage", "status", "score", "attempt",
			"feedback", "result", "created_at", "updated_at", "completed_at",
		}))

	job, err := svc.GetJob(ctx, uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if job.Status != JobStatusCompleted {
		t.Errorf("expected completed, got %s", job.Status)
	}
}

func TestStartJob_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()

	mock.ExpectQuery(`FOR UPDATE`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.StartJob(ctx, uuid.New(), uuid.New())
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}
}

func TestStartJob_AlreadyRunning(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()
	running := "running"

	mock.ExpectQuery(`FOR UPDATE`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(jobRow(running, &running))

	_, err := svc.StartJob(ctx, uuid.New(), uuid.New())
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestCancelJob_NotRunnable(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()

	mock.ExpectQuery(`FROM translation_jobs WHERE id = \$1 AND site_id = \$2`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(jobRow("completed", nil))

	_, err := svc.CancelJob(ctx, uuid.New(), uuid.New())
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestGetScore_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT translation_score FROM translation_jobs`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetScore(ctx, uuid.New(), uuid.New())
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}
}

// --- Stage decisions ---

func TestApproveStage_NotWaiting(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT translation_job_id FROM translation_stages WHERE id = \$1`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"translation_job_id"}).AddRow(uuid.New()))

	mock.ExpectQuery(`FOR UPDATE`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(jobRow("completed", nil))

	_, err := svc.ApproveStage(ctx, uuid.New(), uuid.New())
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestRejectStage_StageNotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT translation_job_id, stage FROM translation_stages WHERE id = \$1`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.RejectStage(ctx, uuid.New(), uuid.New(), "feedback")
	if !errors.Is(err, ErrStageNotFound) {
		t.Errorf("expected ErrStageNotFound, got %v", err)
	}
}

// --- Glossary ---

func TestCreateGlossaryTerm_Validation(t *testing.T) {
	svc, _ := setupMockDB(t)
	ctx := context.Background()

	if _, err := svc.CreateGlossaryTerm(ctx, uuid.New(), uuid.New(), CreateGlossaryTermRequest{TargetTerm: "X"}); err != ErrInvalidGlossary {
		t.Errorf("expected ErrInvalidGlossary, got %v", err)
	}
	_, err := svc.CreateGlossaryTerm(ctx, uuid.New(), uuid.New(), CreateGlossaryTermRequest{
		SourceTerm: "IA", TargetTerm: "AI", SourceLanguage: "fr", TargetLanguage: "en",
	})
	if err != ErrInvalidLanguage {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestCreateGlossaryTerm_Duplicate(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	_, err := svc.CreateGlossaryTerm(ctx, uuid.New(), uuid.New(), CreateGlossaryTermRequest{
		SourceTerm: "IA", TargetTerm: "AI", SourceLanguage: "pt", TargetLanguage: "en",
	})
	if !errors.Is(err, ErrGlossaryDuplicate) {
		t.Errorf("expected ErrGlossaryDuplicate, got %v", err)
	}
}

func TestCreateGlossaryTerm_HappyPath(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectExec(`INSERT INTO glossary_terms`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), "IA", "AI",
			"pt", "en", false, "", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	term, err := svc.CreateGlossaryTerm(ctx, uuid.New(), uuid.New(), CreateGlossaryTermRequest{
		SourceTerm: "IA", TargetTerm: "AI", SourceLanguage: "pt", TargetLanguage: "en",
	})
	if err != nil {
		t.Fatalf("CreateGlossaryTerm failed: %v", err)
	}
	if term.SourceTerm != "IA" || term.TargetTerm != "AI" {
		t.Errorf("unexpected term: %s -> %s", term.SourceTerm, term.TargetTerm)
	}
}

func TestListGlossaryTerms_HappyPath(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "site_id", "project_id", "source_term", "target_term",
		"source_language", "target_language", "forbidden", "description",
		"created_by", "created_at", "updated_at",
	}).AddRow(uuid.New(), uuid.New(), nil, "IA", "AI", "pt", "en", false, nil, nil, now, now)

	mock.ExpectQuery(`FROM glossary_terms`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(rows)

	terms, err := svc.ListGlossaryTerms(ctx, uuid.New(), nil)
	if err != nil {
		t.Fatalf("ListGlossaryTerms failed: %v", err)
	}
	if len(terms) != 1 {
		t.Fatalf("expected 1 term, got %d", len(terms))
	}
	if terms[0].SourceTerm != "IA" {
		t.Errorf("unexpected source term: %s", terms[0].SourceTerm)
	}
}

func TestUpdateGlossaryTerm_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	_, err := svc.UpdateGlossaryTerm(ctx, uuid.New(), uuid.New(), UpdateGlossaryTermRequest{})
	if !errors.Is(err, ErrGlossaryNotFound) {
		t.Errorf("expected ErrGlossaryNotFound, got %v", err)
	}
}

func TestDeleteGlossaryTerm_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	ctx := context.Background()

	mock.ExpectExec(`DELETE FROM glossary_terms`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := svc.DeleteGlossaryTerm(ctx, uuid.New(), uuid.New())
	if !errors.Is(err, ErrGlossaryNotFound) {
		t.Errorf("expected ErrGlossaryNotFound, got %v", err)
	}
}
