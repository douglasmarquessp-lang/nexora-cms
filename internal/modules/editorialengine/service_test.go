package editorialengine

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

// --- Pipeline validation ---

func TestCreatePipeline_Duplicate(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	articleJobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM editorial_pipelines WHERE`).
		WithArgs(articleJobID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "article_job_id", "site_id", "current_stage", "status",
			"started_at", "completed_at", "created_at", "updated_at",
		}).AddRow(
			uuid.New(), articleJobID, siteID, "research", "in_progress",
			nil, nil, pgxmock.AnyArg(), pgxmock.AnyArg(),
		))

	_, err := svc.CreatePipeline(context.Background(), siteID, CreatePipelineRequest{
		ArticleJobID: articleJobID,
	})
	if err != ErrJobAlreadyInPipeline {
		t.Errorf("expected ErrJobAlreadyInPipeline, got %v", err)
	}
}

func TestGetPipeline_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	pipelineID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM editorial_pipelines WHERE`).
		WithArgs(pipelineID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetPipeline(context.Background(), siteID, pipelineID)
	if err != ErrPipelineNotFound {
		t.Errorf("expected ErrPipelineNotFound, got %v", err)
	}
}

func TestGetPipelineByJob_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM editorial_pipelines WHERE`).
		WithArgs(jobID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetPipelineByJob(context.Background(), siteID, jobID)
	if err != ErrPipelineNotFound {
		t.Errorf("expected ErrPipelineNotFound, got %v", err)
	}
}

func TestListPipelines_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM editorial_pipelines WHERE`).
		WithArgs(siteID, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "article_job_id", "site_id", "current_stage", "status",
			"started_at", "completed_at", "created_at", "updated_at",
		}))

	pipelines, err := svc.ListPipelines(context.Background(), siteID, "", "", 0, 0)
	if err != nil {
		t.Fatalf("ListPipelines failed: %v", err)
	}

	if len(pipelines) != 0 {
		t.Errorf("expected 0 pipelines, got %d", len(pipelines))
	}
}

func TestUpdatePipeline_InvalidStage(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	pipelineID := uuid.New()
	invalidStage := PipelineStage("invalid_stage")

	mock.ExpectQuery(`SELECT .+ FROM editorial_pipelines WHERE`).
		WithArgs(pipelineID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "article_job_id", "site_id", "current_stage", "status",
			"started_at", "completed_at", "created_at", "updated_at",
		}).AddRow(
			pipelineID, uuid.New(), siteID, "research", "in_progress",
			nil, nil, pgxmock.AnyArg(), pgxmock.AnyArg(),
		))

	mock.ExpectQuery(`SELECT .+ FROM pipeline_stages WHERE`).
		WithArgs(pipelineID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "pipeline_id", "stage", "status", "started_at", "completed_at",
			"assigned_to", "notes", "metadata", "created_at", "updated_at",
		}))

	_, err := svc.UpdatePipeline(context.Background(), siteID, pipelineID, UpdatePipelineRequest{
		CurrentStage: &invalidStage,
	})
	if err != ErrInvalidStage {
		t.Errorf("expected ErrInvalidStage, got %v", err)
	}
}

func TestGetPipelineStage_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	pipelineID := uuid.New()
	stageID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM pipeline_stages WHERE`).
		WithArgs(stageID, pipelineID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetPipelineStage(context.Background(), pipelineID, stageID)
	if err != ErrStageNotFound {
		t.Errorf("expected ErrStageNotFound, got %v", err)
	}
}

func TestListPipelineStages_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	pipelineID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM pipeline_stages WHERE`).
		WithArgs(pipelineID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "pipeline_id", "stage", "status", "started_at", "completed_at",
			"assigned_to", "notes", "metadata", "created_at", "updated_at",
		}))

	stages, err := svc.ListPipelineStages(context.Background(), pipelineID)
	if err != nil {
		t.Fatalf("ListPipelineStages failed: %v", err)
	}

	if len(stages) != 0 {
		t.Errorf("expected 0 stages, got %d", len(stages))
	}
}

// --- Style Rules ---

func TestGetStyleRules_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM editorial_style_rules WHERE`).
		WithArgs(siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetStyleRules(context.Background(), siteID)
	if err != ErrStyleRulesNotFound {
		t.Errorf("expected ErrStyleRulesNotFound, got %v", err)
	}
}

// --- SEO ---

func TestGetSEOData_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM editorial_seo_data WHERE`).
		WithArgs(jobID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetSEOData(context.Background(), siteID, jobID)
	if err != ErrSEONotFound {
		t.Errorf("expected ErrSEONotFound, got %v", err)
	}
}

// --- Quality ---

func TestGetQualityScore_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM editorial_quality_scores WHERE`).
		WithArgs(jobID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetQualityScore(context.Background(), siteID, jobID)
	if err != ErrQualityNotFound {
		t.Errorf("expected ErrQualityNotFound, got %v", err)
	}
}

func TestListQualityScores_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM editorial_quality_scores WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "article_job_id", "site_id", "seo_score", "readability_score",
			"naturalness_score", "eeat_score", "keyword_density", "heading_structure_score",
			"internal_linking_score", "duplicate_detection", "repetition_detection",
			"passive_voice_count", "avg_sentence_length", "paragraph_balance_score",
			"overall_score", "report", "created_at",
		}))

	scores, err := svc.ListQualityScores(context.Background(), siteID, jobID)
	if err != nil {
		t.Fatalf("ListQualityScores failed: %v", err)
	}

	if len(scores) != 0 {
		t.Errorf("expected 0 scores, got %d", len(scores))
	}
}

// --- Translation ---

func TestCreateTranslation_InvalidDirection(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateTranslation(context.Background(), uuid.New(), uuid.New(), CreateTranslationRequest{
		SourceLanguage: "pt",
		TargetLanguage: "pt",
	})
	if err != ErrInvalidTranslationDir {
		t.Errorf("expected ErrInvalidTranslationDir, got %v", err)
	}
}

func TestCreateTranslation_InvalidLang(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateTranslation(context.Background(), uuid.New(), uuid.New(), CreateTranslationRequest{
		SourceLanguage: "pt",
		TargetLanguage: "fr",
	})
	if err != ErrInvalidTranslationDir {
		t.Errorf("expected ErrInvalidTranslationDir, got %v", err)
	}
}

func TestGetTranslation_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	translationID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM editorial_translations WHERE`).
		WithArgs(translationID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetTranslation(context.Background(), siteID, translationID)
	if err != ErrTranslationNotFound {
		t.Errorf("expected ErrTranslationNotFound, got %v", err)
	}
}

func TestListTranslations_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM editorial_translations WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "article_job_id", "site_id", "source_language", "target_language",
			"status", "translated_slug", "translated_meta", "translated_faq",
			"translated_keywords", "translated_entities", "completed_at",
			"created_at", "updated_at",
		}))

	translations, err := svc.ListTranslations(context.Background(), siteID, jobID)
	if err != nil {
		t.Fatalf("ListTranslations failed: %v", err)
	}

	if len(translations) != 0 {
		t.Errorf("expected 0 translations, got %d", len(translations))
	}
}

// --- Prompt ---

func TestGetPromptData_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM editorial_prompt_data WHERE`).
		WithArgs(jobID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetPromptData(context.Background(), siteID, jobID)
	if err != ErrPromptDataNotFound {
		t.Errorf("expected ErrPromptDataNotFound, got %v", err)
	}
}

func TestCreatePromptData_Existing(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM editorial_prompt_data WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "article_job_id", "site_id",
			"briefing", "style_rules", "seo_rules", "tone",
			"outline", "entities", "target_language", "audience",
			"word_count", "internal_links", "constraints", "created_at", "updated_at",
		}).AddRow(
			uuid.New(), jobID, siteID,
			"{}", "{}", "{}", "", "[]", "[]", "", "", 0, "{}", "{}",
			pgxmock.AnyArg(), pgxmock.AnyArg(),
		))

	pd, err := svc.CreatePromptData(context.Background(), siteID, jobID)
	if err != nil {
		t.Fatalf("CreatePromptData failed: %v", err)
	}

	if pd.ArticleJobID != jobID {
		t.Errorf("expected job %q, got %q", jobID, pd.ArticleJobID)
	}
}

// --- Not avail DB ---

func TestCreatePipeline_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.CreatePipeline(context.Background(), uuid.New(), CreatePipelineRequest{
		ArticleJobID: uuid.New(),
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestGetStyleRules_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.GetStyleRules(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestCreateSEOData_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.CreateSEOData(context.Background(), uuid.New(), uuid.New(), CreateSEODataRequest{})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestCreateQualityScore_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.CreateQualityScore(context.Background(), uuid.New(), uuid.New(), CreateQualityScoreRequest{})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestCreateTranslation_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.CreateTranslation(context.Background(), uuid.New(), uuid.New(), CreateTranslationRequest{
		SourceLanguage: "en",
		TargetLanguage: "pt",
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestGetPromptData_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.GetPromptData(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}
