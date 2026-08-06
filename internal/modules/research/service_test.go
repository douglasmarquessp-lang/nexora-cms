package research

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/ai"
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

func TestService_CreateJob(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec(`INSERT INTO research_jobs`).
		WithArgs(pgxmock.AnyArg(), siteID, "Test Topic", "en", "tech", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	job, err := svc.CreateJob(context.Background(), siteID, userID, CreateResearchJobRequest{
		Topic:    "Test Topic",
		Language: "en",
		Category: "tech",
	})
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	if job.Topic != "Test Topic" {
		t.Errorf("expected 'Test Topic', got %q", job.Topic)
	}
	if job.Language != "en" {
		t.Errorf("expected 'en', got %q", job.Language)
	}
	if job.Status != JobStatusPending {
		t.Errorf("expected pending, got %q", job.Status)
	}
}

func TestService_CreateJob_InvalidLanguage(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateJob(context.Background(), uuid.New(), uuid.New(), CreateResearchJobRequest{
		Topic:    "Test",
		Language: "fr",
	})
	if err != ErrInvalidLanguage {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestService_CreateJob_EmptyTopic(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateJob(context.Background(), uuid.New(), uuid.New(), CreateResearchJobRequest{
		Topic:    "",
		Language: "en",
	})
	if err != ErrTopicRequired {
		t.Errorf("expected ErrTopicRequired, got %v", err)
	}
}

func TestService_GetJob_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM research_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetJob(context.Background(), siteID, jobID)
	if err != ErrResearchJobNotFound {
		t.Errorf("expected ErrResearchJobNotFound, got %v", err)
	}
}

func TestService_ListJobs_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM research_jobs WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "topic", "language", "category", "status",
			"sources_count", "error_message", "completed_at", "created_at", "updated_at",
		}))

	jobs, err := svc.ListJobs(context.Background(), siteID, "")
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}

	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestService_SearchByTopic_NoResults(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM research_jobs WHERE`).
		WithArgs(siteID, "%nonexistent%").
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "topic", "language", "category", "status",
			"sources_count", "error_message", "completed_at", "created_at", "updated_at",
		}))

	jobs, err := svc.SearchByTopic(context.Background(), siteID, "nonexistent")
	if err != nil {
		t.Fatalf("SearchByTopic failed: %v", err)
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

	mock.ExpectQuery(`SELECT .+ FROM research_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "topic", "language", "category", "status",
			"sources_count", "error_message", "completed_at", "created_at", "updated_at",
		}).AddRow(
			jobID, siteID, "Topic", "en", "", "pending", 0, "", nil, now, now,
		))

	mock.ExpectExec(`UPDATE research_jobs SET deleted_at`).
		WithArgs(jobID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := svc.DeleteJob(context.Background(), siteID, jobID)
	if err != nil {
		t.Fatalf("DeleteJob failed: %v", err)
	}
}

func TestService_AddSource(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	jobID := uuid.New()

	mock.ExpectExec(`INSERT INTO research_sources`).
		WithArgs(
			pgxmock.AnyArg(), jobID, "Source Title", "https://example.com", "example.com", 30, "en", "Author",
			pgxmock.AnyArg(), "Summary", "Facts", "Stats", 85, 1,
			float64(0), false, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectExec(`UPDATE research_jobs SET sources_count`).
		WithArgs(jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	source, err := svc.AddSource(context.Background(), jobID, ResearchSource{
		Title:            "Source Title",
		URL:              "https://example.com",
		Domain:           "example.com",
		ReliabilityScore: 30,
		Language:         "en",
		Author:           "Author",
		Summary:          "Summary",
		MainFacts:        "Facts",
		Statistics:       "Stats",
		RelevanceScore:   85,
		Position:         1,
	})
	if err != nil {
		t.Fatalf("AddSource failed: %v", err)
	}

	if source.Title != "Source Title" {
		t.Errorf("expected 'Source Title', got %q", source.Title)
	}
}

func TestService_AddEntity(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	jobID := uuid.New()

	mock.ExpectExec(`INSERT INTO research_entities`).
		WithArgs(pgxmock.AnyArg(), jobID, "company", "Acme Corp", "AI company", "https://example.com", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	entity, err := svc.AddEntity(context.Background(), jobID, ResearchEntity{
		EntityType: EntityTypeCompany,
		Name:       "Acme Corp",
		Context:    "AI company",
		SourceURL:  "https://example.com",
	})
	if err != nil {
		t.Fatalf("AddEntity failed: %v", err)
	}

	if entity.Name != "Acme Corp" {
		t.Errorf("expected 'Acme Corp', got %q", entity.Name)
	}
	if entity.EntityType != EntityTypeCompany {
		t.Errorf("expected company, got %q", entity.EntityType)
	}
}

func TestService_GetBriefing_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT .+ FROM research_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "topic", "language", "category", "status",
			"sources_count", "error_message", "completed_at", "created_at", "updated_at",
		}).AddRow(
			jobID, siteID, "Topic", "en", "", "completed", 3, "", nil, now, now,
		))

	mock.ExpectQuery(`SELECT .+ FROM research_briefings WHERE`).
		WithArgs(jobID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetBriefing(context.Background(), siteID, jobID)
	if err != ErrBriefingNotFound {
		t.Errorf("expected ErrBriefingNotFound, got %v", err)
	}
}

func TestService_UpdateJob_NoChanges(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT .+ FROM research_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "topic", "language", "category", "status",
			"sources_count", "error_message", "completed_at", "created_at", "updated_at",
		}).AddRow(
			jobID, siteID, "Topic", "en", "", "pending", 0, "", nil, now, now,
		))

	job, err := svc.UpdateJob(context.Background(), siteID, jobID, UpdateResearchJobRequest{})
	if err != nil {
		t.Fatalf("UpdateJob failed: %v", err)
	}

	if job.Topic != "Topic" {
		t.Errorf("expected 'Topic', got %q", job.Topic)
	}
}

func TestService_CompleteJob(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	jobID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT .+ FROM research_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "topic", "language", "category", "status",
			"sources_count", "error_message", "completed_at", "created_at", "updated_at",
		}).AddRow(
			jobID, siteID, "Topic", "en", "", "pending", 0, "", nil, now, now,
		))

	mock.ExpectExec(`UPDATE research_jobs SET`).
		WithArgs("completed", pgxmock.AnyArg(), jobID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectQuery(`SELECT .+ FROM research_jobs WHERE`).
		WithArgs(jobID, siteID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "topic", "language", "category", "status",
			"sources_count", "error_message", "completed_at", "created_at", "updated_at",
		}).AddRow(
			jobID, siteID, "Topic", "en", "", "completed", 0, "", &now, now, now,
		))

	err := svc.CompleteJob(context.Background(), siteID, jobID)
	if err != nil {
		t.Fatalf("CompleteJob failed: %v", err)
	}
}

func TestService_ExecuteGroundedResearch_NoAI(t *testing.T) {
	svc, _ := setupMockDB(t)
	ctx := context.Background()

	result, err := svc.ExecuteGroundedResearch(ctx, "Test Topic", "en")
	if err != nil {
		t.Fatalf("ExecuteGroundedResearch failed: %v", err)
	}
	if result.GroundingMetadata == nil {
		t.Fatal("expected grounding metadata in fallback")
	}
	if !result.GroundingMetadata.Unverified {
		t.Error("expected unverified flag in fallback")
	}
	if result.Content == "" {
		t.Error("expected non-empty fallback content")
	}
}

func TestService_SourcesFromGrounding(t *testing.T) {
	svc, _ := setupMockDB(t)

	jobID := uuid.New()
	now := time.Now()

	gm := &ai.GroundingMetadata{
		Sources: []ai.GroundingSource{
			{
				URI:            "https://example.com/src1",
				Title:          "Source 1",
				FreshnessScore: 0.95,
				IsVerified:     true,
				RetrievedAt:    now,
			},
		},
		SearchSuggested: true,
	}

	sources := svc.SourcesFromGrounding(jobID, gm)
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].URL != "https://example.com/src1" {
		t.Errorf("expected src1 URL, got %s", sources[0].URL)
	}
	if sources[0].FreshnessScore != 0.95 {
		t.Errorf("expected freshness 0.95, got %f", sources[0].FreshnessScore)
	}
	if !sources[0].IsVerified {
		t.Error("expected source to be verified")
	}
	if sources[0].RelevanceScore != 95 {
		t.Errorf("expected relevance score 95, got %d", sources[0].RelevanceScore)
	}
}

func TestService_SourcesFromGrounding_Nil(t *testing.T) {
	svc, _ := setupMockDB(t)
	jobID := uuid.New()

	sources := svc.SourcesFromGrounding(jobID, nil)
	if sources != nil {
		t.Error("expected nil sources for nil grounding metadata")
	}
}

func TestService_ArticleSourceModel(t *testing.T) {
	siteID := uuid.New()
	now := time.Now()

	src := ArticleSource{
		SiteID:         siteID,
		SourceURL:      "https://example.com/test",
		Title:          "Test Source",
		Snippet:        "Snippet",
		FreshnessScore: 0.9,
		IsVerified:     true,
		RetrievedAt:    now,
	}

	if src.SourceURL != "https://example.com/test" {
		t.Errorf("expected URL, got %s", src.SourceURL)
	}
	if !src.IsVerified {
		t.Error("expected verified")
	}
}
