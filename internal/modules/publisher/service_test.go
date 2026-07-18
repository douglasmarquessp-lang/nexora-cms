package publisher

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

func TestPublishArticle_EmptyTitle(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.PublishArticle(context.Background(), uuid.New(), uuid.New(), PublishRequest{
		Title:    "",
		Language: "pt",
	})
	if err != ErrTitleRequired {
		t.Errorf("expected ErrTitleRequired, got %v", err)
	}
}

func TestPublishArticle_InvalidLanguage(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.PublishArticle(context.Background(), uuid.New(), uuid.New(), PublishRequest{
		Title:    "Test Article",
		Language: "fr",
	})
	if err != ErrInvalidLanguage {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestPublishArticle_InvalidSlug(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.PublishArticle(context.Background(), uuid.New(), uuid.New(), PublishRequest{
		Title:    "Test Article",
		Language: "pt",
		Slug:     "a",
	})
	if err != ErrInvalidSlug {
		t.Errorf("expected ErrInvalidSlug, got %v", err)
	}
}

func TestAddToQueue_InvalidAction(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.AddToQueue(context.Background(), uuid.New(), uuid.New(), QueueRequest{
		PublicationID: uuid.New(),
		Action:        "invalid_action",
	})
	if err != ErrInvalidAction {
		t.Errorf("expected ErrInvalidAction, got %v", err)
	}
}

// --- Not Found ---

func TestGetPublication_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM publications WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetPublication(context.Background(), uuid.New(), uuid.New())
	if err != ErrPublicationNotFound {
		t.Errorf("expected ErrPublicationNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet mock expectations: %v", err)
	}
}

func TestGetSchedule_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM publication_schedule WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetSchedule(context.Background(), uuid.New(), uuid.New())
	if err != ErrScheduleNotFound {
		t.Errorf("expected ErrScheduleNotFound, got %v", err)
	}
}

func TestGetQueue_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM publication_queue WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.repo.GetQueueItem(context.Background(), uuid.New(), uuid.New())
	if err != ErrQueueItemNotFound {
		t.Errorf("expected ErrQueueItemNotFound, got %v", err)
	}
}

func TestGetMetrics_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM publication_metrics WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetPublicationMetrics(context.Background(), uuid.New(), uuid.New())
	if err != ErrMetricsNotFound {
		t.Errorf("expected ErrMetricsNotFound, got %v", err)
	}
}

// --- State Transition Validation ---

func TestUnpublish_NotPublished(t *testing.T) {
	svc, mock := setupMockDB(t)

	nowTime := now()
	mock.ExpectQuery(`SELECT .+ FROM publications WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "post_id", "title", "content", "excerpt", "slug", "url",
			"canonical_url", "language", "translations", "multilingual_urls",
			"status", "visibility", "author_id", "published_by", "published_at", "unpublished_at",
			"scheduled_at", "is_featured", "meta_title", "meta_description", "og_image",
			"featured_image_url", "tags", "categories", "word_count", "reading_time", "revision",
			"checksum", "source", "metadata", "created_by", "created_at", "updated_at",
		}).AddRow(uuid.New(), uuid.New(), nil, "test", "", "", "test-slug", "https://example.com/test-slug",
			"", "pt", "{}", "{}",
			"draft", "public", nil, nil, nil, nil,
			nil, false, "", "", "",
			"", []string{}, []string{}, 0, 0, 1,
			"", "manual", "{}", nil, nowTime, nowTime))

	_, err := svc.Unpublish(context.Background(), uuid.New(), uuid.New(), uuid.New(), "")
	if err != ErrPublicationNotPublished {
		t.Errorf("expected ErrPublicationNotPublished, got %v", err)
	}
}

func TestRepublish_AlreadyPublished(t *testing.T) {
	svc, mock := setupMockDB(t)

	nowTime := now()
	mock.ExpectQuery(`SELECT .+ FROM publications WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "post_id", "title", "content", "excerpt", "slug", "url",
			"canonical_url", "language", "translations", "multilingual_urls",
			"status", "visibility", "author_id", "published_by", "published_at", "unpublished_at",
			"scheduled_at", "is_featured", "meta_title", "meta_description", "og_image",
			"featured_image_url", "tags", "categories", "word_count", "reading_time", "revision",
			"checksum", "source", "metadata", "created_by", "created_at", "updated_at",
		}).AddRow(uuid.New(), uuid.New(), nil, "test", "", "", "test-slug", "https://example.com/test-slug",
			"", "pt", "{}", "{}",
			"published", "public", nil, nil, nil, nil,
			nil, false, "", "", "",
			"", []string{}, []string{}, 0, 0, 1,
			"", "manual", "{}", nil, nowTime, nowTime))

	_, err := svc.Republish(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err != ErrPublicationAlreadyPublished {
		t.Errorf("expected ErrPublicationAlreadyPublished, got %v", err)
	}
}

// --- Empty Results ---

func TestListPublications_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM publications WHERE`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`SELECT .+ FROM publications WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "post_id", "title", "content", "excerpt", "slug", "url",
			"canonical_url", "language", "translations", "multilingual_urls",
			"status", "visibility", "author_id", "published_by", "published_at", "unpublished_at",
			"scheduled_at", "is_featured", "meta_title", "meta_description", "og_image",
			"featured_image_url", "tags", "categories", "word_count", "reading_time", "revision",
			"checksum", "source", "metadata", "created_by", "created_at", "updated_at",
		}))

	pubs, total, err := svc.ListPublications(context.Background(), uuid.New(), "", "", 0, 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(pubs) != 0 {
		t.Errorf("expected empty list, got %d items", len(pubs))
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
}

func TestListQueue_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM publication_queue WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "publication_id", "action", "status", "priority", "scheduled_for",
			"started_at", "completed_at", "error_message", "retry_count", "max_retries",
			"metadata", "created_by", "created_at", "updated_at",
		}))

	items, err := svc.ListQueue(context.Background(), uuid.New(), "", 0, 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty list, got %d items", len(items))
	}
}

func TestListSchedules_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM publication_schedule WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "site_id", "publication_id", "scheduled_at", "action", "status",
			"recurrence", "recurrence_end", "notify_on_publish", "notify_users",
			"metadata", "created_by", "cancelled_at", "cancel_reason", "created_at", "updated_at",
		}))

	schedules, err := svc.ListSchedules(context.Background(), uuid.New(), "", 0, 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(schedules) != 0 {
		t.Errorf("expected empty list, got %d items", len(schedules))
	}
}

func TestGetHistory_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT .+ FROM publication_history WHERE`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "publication_id", "site_id", "action", "previous_status", "new_status",
			"title", "slug", "changes", "reason", "performed_by", "performed_at", "created_at",
		}))

	history, err := svc.GetPublicationHistory(context.Background(), uuid.New(), uuid.New(), 0, 0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected empty list, got %d items", len(history))
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

func TestPublishArticle_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.PublishArticle(context.Background(), uuid.New(), uuid.New(), PublishRequest{
		Title:    "Test",
		Language: "pt",
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestGetPublication_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.GetPublication(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestValidateSlug_Invalid(t *testing.T) {
	svc, _ := setupMockDB(t)

	available, _, err := svc.ValidateSlug(context.Background(), uuid.New(), "")
	if err != ErrInvalidSlug {
		t.Errorf("expected ErrInvalidSlug, got %v", err)
	}
	if available {
		t.Error("expected not available for empty slug")
	}
}

func TestGenerateSlug(t *testing.T) {
	svc, mock := setupMockDB(t)

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM publications`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	slug, err := svc.GenerateSlug(context.Background(), uuid.New(), "My Test Article!")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if slug != "my-test-article" {
		t.Errorf("expected 'my-test-article', got '%s'", slug)
	}
}
