package seoengine

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

func TestCreateProject_EmptyTitle(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateProject(context.Background(), uuid.New(), uuid.New(), CreateProjectRequest{
		Title:    "",
		Language: "pt",
	})
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestCreateProject_InvalidLanguage(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateProject(context.Background(), uuid.New(), uuid.New(), CreateProjectRequest{
		Title:    "Test Project",
		Language: "fr",
	})
	if err != ErrInvalidLanguage {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestCreateCluster_EmptyName(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.CreateCluster(context.Background(), uuid.New(), CreateClusterRequest{
		Name: "",
	})
	if err == nil {
		t.Error("expected error for empty cluster name")
	}
}

func TestAddImprovement_InvalidCategory(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.AddImprovement(context.Background(), uuid.New(), uuid.New(), AddImprovementRequest{
		Category:   "invalid_category",
		Issue:      "Test issue",
		Suggestion: "Test suggestion",
	})
	if err != ErrInvalidCategory {
		t.Errorf("expected ErrInvalidCategory, got %v", err)
	}
}

func TestAddImprovement_EmptyIssue(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.AddImprovement(context.Background(), uuid.New(), uuid.New(), AddImprovementRequest{
		Category:   CategoryTitle,
		Issue:      "",
		Suggestion: "Test suggestion",
	})
	if err == nil {
		t.Error("expected error for empty issue")
	}
}

func TestAddImprovement_EmptySuggestion(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.AddImprovement(context.Background(), uuid.New(), uuid.New(), AddImprovementRequest{
		Category:   CategoryTitle,
		Issue:      "Test issue",
		Suggestion: "",
	})
	if err == nil {
		t.Error("expected error for empty suggestion")
	}
}

// --- Not Found ---

func TestGetProject_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM seo_projects WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetProject(context.Background(), uuid.New(), uuid.New())
	if err != ErrProjectNotFound {
		t.Errorf("expected ErrProjectNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestGetAudit_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM seo_audits WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetAudit(context.Background(), uuid.New(), uuid.New())
	if err != ErrAuditNotFound {
		t.Errorf("expected ErrAuditNotFound, got %v", err)
	}
}

func TestGetScores_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM seo_scores WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetScores(context.Background(), uuid.New(), uuid.New())
	if err != ErrScoreNotFound {
		t.Errorf("expected ErrScoreNotFound, got %v", err)
	}
}

func TestDeleteProject_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectExec(`DELETE FROM seo_projects WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := svc.DeleteProject(context.Background(), uuid.New(), uuid.New())
	if err != ErrProjectNotFound {
		t.Errorf("expected ErrProjectNotFound, got %v", err)
	}
}

// --- Empty Results ---

func TestListProjects_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM seo_projects WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "user_id", "title", "target_url", "post_id",
			"language", "status", "seo_score", "readability_score", "keyword_density",
			"content_quality", "technical_score", "eeat_score", "freshness_score",
			"topical_authority_score", "slug_target", "meta_title_target",
			"meta_description_target", "content_type", "recommendations",
			"checklist", "ai_suggestions",
			"started_at", "completed_at", "error_message", "created_by",
			"created_at", "updated_at",
		}))

	projects, err := svc.ListProjects(context.Background(), uuid.New(), "", "", 0, 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected empty list, got %d items", len(projects))
	}
}

func TestListClusters_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM seo_clusters WHERE`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "name", "description", "keywords",
			"article_count", "avg_score", "topical_authority_score",
			"semantic_entities", "internal_links_count",
			"content_gap_articles", "parent_cluster_id", "language",
			"created_at", "updated_at",
		}))

	clusters, err := svc.ListClusters(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(clusters) != 0 {
		t.Errorf("expected empty list, got %d items", len(clusters))
	}
}

func TestListImprovements_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM seo_improvements WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "seo_project_id", "post_id", "category", "issue",
			"suggestion", "priority", "impact_score", "effort_score", "status",
			"applied_at", "language", "created_at", "updated_at",
		}))

	improvements, err := svc.ListImprovements(context.Background(), uuid.New(), uuid.New(), "", "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(improvements) != 0 {
		t.Errorf("expected empty list, got %d items", len(improvements))
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

func TestCreateProject_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.CreateProject(context.Background(), uuid.New(), uuid.New(), CreateProjectRequest{
		Title:    "Test",
		Language: "pt",
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestGetProject_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.GetProject(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

// --- Orphan Detection ---

func TestDetectOrphanArticles_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT p\.id, COALESCE\(p\.title,''\).+FROM posts p WHERE`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "title", "slug", "incoming_links",
		}))

	orphans, err := svc.DetectOrphanArticles(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(orphans) != 0 {
		t.Errorf("expected empty list, got %d items", len(orphans))
	}
}
