package posts

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/kernel"
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

func TestService_PostErrorConstants(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrPostNotFound, "post not found"},
		{ErrPostSlugExists, "post slug already exists"},
		{ErrInvalidPostStatus, "invalid post status"},
		{ErrDatabaseNotAvail, "database not available"},
		{ErrInvalidPagination, "invalid pagination parameters"},
		{ErrPostNotInSite, "post does not belong to this site"},
	}
	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, tt.err.Error())
		}
	}
}

func TestService_PostStatusConstants(t *testing.T) {
	tests := []struct {
		status   PostStatus
		expected string
	}{
		{PostStatusDraft, "draft"},
		{PostStatusPublished, "published"},
		{PostStatusScheduled, "scheduled"},
		{PostStatusArchived, "archived"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
		}
	}
}

func TestService_PostEventConstants(t *testing.T) {
	_ = []string{
		string(EventPostCreated),
		string(EventPostUpdated),
		string(EventPostDeleted),
		string(EventPostPublished),
		string(EventPostArchived),
	}
}

func TestService_Pool_NilDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.pool()
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_Pool_NilDBPool(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, &database.Database{Pool: nil}, nil)

	_, err := svc.pool()
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_Create_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), CreatePostRequest{
		Title: "Test Post",
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_Create_InvalidStatus(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), uuid.New(), CreatePostRequest{
		Title:  "Test",
		Status: PostStatus("invalid"),
	})
	if err != ErrInvalidPostStatus {
		t.Errorf("expected ErrInvalidPostStatus, got: %v", err)
	}
}

func TestService_GetByID_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_GetBySlug_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.GetBySlug(context.Background(), uuid.New(), "test-slug")
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_List_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.List(context.Background(), PostListRequest{
		SiteID: uuid.New(),
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_Update_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), UpdatePostRequest{})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_Delete_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_SetStatus_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	err := svc.SetStatus(context.Background(), uuid.New(), uuid.New(), PostStatusPublished)
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_SetStatus_InvalidStatus(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	err := svc.SetStatus(context.Background(), uuid.New(), uuid.New(), PostStatus("bogus"))
	if err != ErrInvalidPostStatus {
		t.Errorf("expected ErrInvalidPostStatus, got: %v", err)
	}
}

func TestService_SetEventBus(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	bus := kernel.NewEventBus(log)

	svc.SetEventBus(bus)
}

func TestService_SetEventBus_Nil(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	svc.SetEventBus(nil)
}

func TestService_FireEvent_WithBus(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	bus := kernel.NewEventBus(log)
	svc.SetEventBus(bus)

	svc.fireEvent(context.Background(), EventPostCreated, map[string]interface{}{
		"post_id": uuid.New().String(),
	}, uuid.New())
}

func TestService_FireEvent_WithoutBus(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	svc.fireEvent(context.Background(), EventPostCreated, map[string]interface{}{
		"post_id": uuid.New().String(),
	}, uuid.New())
}

func TestService_FireEvent_WithSubscriber(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	bus := kernel.NewEventBus(log)
	svc.SetEventBus(bus)

	received := make(chan string, 1)
	bus.Subscribe(EventPostCreated, func(ctx context.Context, event kernel.Event) error {
		if p, ok := event.Payload.(map[string]interface{}); ok {
			if slug, ok := p["slug"]; ok {
				received <- slug.(string)
			}
		}
		return nil
	})

	svc.fireEvent(context.Background(), EventPostCreated, map[string]interface{}{
		"post_id": uuid.New().String(),
		"slug":    "my-test-post",
	}, uuid.New())

	select {
	case slug := <-received:
		if slug != "my-test-post" {
			t.Errorf("expected 'my-test-post', got '%s'", slug)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestService_generateSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"  Spaces  ", "spaces"},
		{"Special!@#Chars", "special-chars"},
		{"UPPERCASE", "uppercase"},
		{"already-slug", "already-slug"},
		{"multiple   spaces", "multiple-spaces"},
	}
	for _, tt := range tests {
		result := generateSlug(tt.input)
		if result != tt.expected {
			t.Errorf("generateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestService_isValidStatus(t *testing.T) {
	tests := []struct {
		status PostStatus
		valid  bool
	}{
		{PostStatusDraft, true},
		{PostStatusPublished, true},
		{PostStatusScheduled, true},
		{PostStatusArchived, true},
		{PostStatus(""), false},
		{PostStatus("invalid"), false},
	}
	for _, tt := range tests {
		result := isValidStatus(tt.status)
		if result != tt.valid {
			t.Errorf("isValidStatus(%q) = %v, want %v", tt.status, result, tt.valid)
		}
	}
}

func TestService_Create_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	authorID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts WHERE site_id = \$1 AND slug = \$2 AND deleted_at IS NULL`).
		WithArgs(siteID, "test-post").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`INSERT INTO posts`).
		WithArgs(pgxmock.AnyArg(), siteID, "Test Post", "test-post", `[{"type":"text"}]`, "An excerpt", "draft", authorID, pgxmock.AnyArg(), pgxmock.AnyArg(), `{"key":"value"}`, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	post, err := svc.Create(context.Background(), siteID, authorID, CreatePostRequest{
		Title:   "Test Post",
		Content: []interface{}{map[string]interface{}{"type": "text"}},
		Excerpt: "An excerpt",
		Status:  PostStatusDraft,
		PostMeta: map[string]interface{}{
			"key": "value",
		},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if post.Title != "Test Post" {
		t.Errorf("expected 'Test Post', got '%s'", post.Title)
	}
	if post.Slug != "test-post" {
		t.Errorf("expected 'test-post', got '%s'", post.Slug)
	}
	if post.Status != PostStatusDraft {
		t.Errorf("expected 'draft', got '%s'", post.Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Create_PublishedSetsPublishedAt(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	authorID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts WHERE site_id = \$1 AND slug = \$2 AND deleted_at IS NULL`).
		WithArgs(siteID, "published-post").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`INSERT INTO posts`).
		WithArgs(pgxmock.AnyArg(), siteID, "Published Post", "published-post", "[]", "", "published", authorID, pgxmock.AnyArg(), pgxmock.AnyArg(), "{}", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	post, err := svc.Create(context.Background(), siteID, authorID, CreatePostRequest{
		Title:  "Published Post",
		Status: PostStatusPublished,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if post.PublishedAt == nil {
		t.Error("expected published_at to be set when status is published")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Create_SlugCollision(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	authorID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts WHERE site_id = \$1 AND slug = \$2 AND deleted_at IS NULL`).
		WithArgs(siteID, "test-post").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts WHERE site_id = \$1 AND slug = \$2 AND deleted_at IS NULL`).
		WithArgs(siteID, "test-post-1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`INSERT INTO posts`).
		WithArgs(pgxmock.AnyArg(), siteID, "Test Post", "test-post-1", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), authorID, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	post, err := svc.Create(context.Background(), siteID, authorID, CreatePostRequest{
		Title: "Test Post",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if post.Slug != "test-post-1" {
		t.Errorf("expected 'test-post-1', got '%s'", post.Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_GetByID_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{"id", "site_id", "title", "slug", "content", "excerpt", "status", "author_id", "published_at", "scheduled_at", "post_meta", "metadata", "created_at", "updated_at", "deleted_at"}).
		AddRow(postID, siteID, "Test Post", "test-post", `[{"type":"text"}]`, "Excerpt", "draft", uuid.New(), nil, nil, `{"key":"val"}`, `{}`, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(rows)

	mock.ExpectQuery(`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE\(c.description, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT t.id, t.site_id, t.name, t.slug, COALESCE\(t.color, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	post, err := svc.GetByID(context.Background(), siteID, postID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if post.ID != postID {
		t.Errorf("expected post ID %s, got %s", postID, post.ID)
	}
	if post.Title != "Test Post" {
		t.Errorf("expected 'Test Post', got '%s'", post.Title)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(pgxmock.NewRows(nil))

	_, err := svc.GetByID(context.Background(), siteID, postID)
	if err != ErrPostNotFound {
		t.Errorf("expected ErrPostNotFound, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_List_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts p WHERE p.deleted_at IS NULL AND p.site_id = \$1`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	rows := pgxmock.NewRows([]string{"id", "title", "slug", "excerpt", "status", "author_id", "published_at", "created_at", "updated_at", "category_count", "tag_count"}).
		AddRow(postID, "Test Post", "test-post", "Excerpt", PostStatus("published"), uuid.New(), nil, now, now, 0, 0)

	mock.ExpectQuery(`SELECT p.id, p.title, p.slug, COALESCE\(p.excerpt, ''\), p.status, p.author_id, p.published_at, p.created_at, p.updated_at`).
		WithArgs(siteID, 20, 0).
		WillReturnRows(rows)

	resp, err := svc.List(context.Background(), PostListRequest{
		SiteID: siteID,
		Page:   1,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(resp.Posts) != 1 {
		t.Errorf("expected 1 post, got %d", len(resp.Posts))
	}
	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_List_WithStatusFilter(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts p WHERE p.deleted_at IS NULL AND p.site_id = \$1 AND p.status = \$2`).
		WithArgs(siteID, "published").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`SELECT p.id, p.title, p.slug, COALESCE\(p.excerpt, ''\), p.status, p.author_id, p.published_at, p.created_at, p.updated_at`).
		WithArgs(siteID, "published", 20, 0).
		WillReturnRows(pgxmock.NewRows(nil))

	resp, err := svc.List(context.Background(), PostListRequest{
		SiteID: siteID,
		Status: PostStatusPublished,
		Page:   1,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(resp.Posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(resp.Posts))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_List_InvalidStatus(t *testing.T) {
	svc, _ := setupMockDB(t)

	_, err := svc.List(context.Background(), PostListRequest{
		SiteID: uuid.New(),
		Status: PostStatus("bogus"),
	})
	if err != ErrInvalidPostStatus {
		t.Errorf("expected ErrInvalidPostStatus, got: %v", err)
	}
}

func TestService_List_WithSearch(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts p WHERE`).
		WithArgs(siteID, "%search%").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`SELECT p.id, p.title, p.slug, COALESCE\(p.excerpt, ''\), p.status`).
		WithArgs(siteID, "%search%", 20, 0).
		WillReturnRows(pgxmock.NewRows(nil))

	_, err := svc.List(context.Background(), PostListRequest{
		SiteID: siteID,
		Search: "search",
		Page:   1,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "title", "slug", "content", "excerpt", "status", "author_id", "published_at", "scheduled_at", "post_meta", "metadata", "created_at", "updated_at", "deleted_at"}).
		AddRow(postID, siteID, "Original", "original", `[]`, "Old", "draft", uuid.New(), nil, nil, `{}`, `{}`, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(getRows)

	mock.ExpectQuery(`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE\(c.description, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT t.id, t.site_id, t.name, t.slug, COALESCE\(t.color, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	title := "Updated"
	excerpt := "New excerpt"
	req := UpdatePostRequest{
		Title:   &title,
		Excerpt: &excerpt,
	}

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts WHERE site_id = \$1 AND slug = \$2 AND id != \$3 AND deleted_at IS NULL`).
		WithArgs(siteID, "updated", postID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`UPDATE posts SET title = \$1, slug = \$2, excerpt = \$3, updated_at = NOW\(\) WHERE id = \$4 AND site_id = \$5 AND deleted_at IS NULL`).
		WithArgs("Updated", "updated", "New excerpt", postID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	getRows2 := pgxmock.NewRows([]string{"id", "site_id", "title", "slug", "content", "excerpt", "status", "author_id", "published_at", "scheduled_at", "post_meta", "metadata", "created_at", "updated_at", "deleted_at"}).
		AddRow(postID, siteID, "Updated", "updated", `[]`, "New excerpt", "draft", uuid.New(), nil, nil, `{}`, `{}`, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(getRows2)

	mock.ExpectQuery(`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE\(c.description, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT t.id, t.site_id, t.name, t.slug, COALESCE\(t.color, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	_, err := svc.Update(context.Background(), siteID, postID, req)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(pgxmock.NewRows(nil))

	title := "Updated"
	_, err := svc.Update(context.Background(), siteID, postID, UpdatePostRequest{
		Title: &title,
	})
	if err != ErrPostNotFound {
		t.Errorf("expected ErrPostNotFound, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_InvalidStatus(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "title", "slug", "content", "excerpt", "status", "author_id", "published_at", "scheduled_at", "post_meta", "metadata", "created_at", "updated_at", "deleted_at"}).
		AddRow(postID, siteID, "Title", "title", `[]`, "", "draft", uuid.New(), nil, nil, `{}`, `{}`, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(getRows)

	mock.ExpectQuery(`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE\(c.description, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT t.id, t.site_id, t.name, t.slug, COALESCE\(t.color, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	invalid := PostStatus("bogus")
	_, err := svc.Update(context.Background(), siteID, postID, UpdatePostRequest{
		Status: &invalid,
	})
	if err != ErrInvalidPostStatus {
		t.Errorf("expected ErrInvalidPostStatus, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_SlugCollision(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "title", "slug", "content", "excerpt", "status", "author_id", "published_at", "scheduled_at", "post_meta", "metadata", "created_at", "updated_at", "deleted_at"}).
		AddRow(postID, siteID, "Original", "original", `[]`, "", "draft", uuid.New(), nil, nil, `{}`, `{}`, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(getRows)

	mock.ExpectQuery(`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE\(c.description, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT t.id, t.site_id, t.name, t.slug, COALESCE\(t.color, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	title := "New Title"
	req := UpdatePostRequest{Title: &title}

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts WHERE site_id = \$1 AND slug = \$2 AND id != \$3 AND deleted_at IS NULL`).
		WithArgs(siteID, "new-title", postID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts WHERE site_id = \$1 AND slug = \$2 AND id != \$3 AND deleted_at IS NULL`).
		WithArgs(siteID, "new-title-1", postID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`UPDATE posts SET title = \$1, slug = \$2, updated_at = NOW\(\) WHERE id = \$3 AND site_id = \$4 AND deleted_at IS NULL`).
		WithArgs("New Title", "new-title-1", postID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	getRows2 := pgxmock.NewRows([]string{"id", "site_id", "title", "slug", "content", "excerpt", "status", "author_id", "published_at", "scheduled_at", "post_meta", "metadata", "created_at", "updated_at", "deleted_at"}).
		AddRow(postID, siteID, "New Title", "new-title-1", `[]`, "", "draft", uuid.New(), nil, nil, `{}`, `{}`, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(getRows2)

	mock.ExpectQuery(`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE\(c.description, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT t.id, t.site_id, t.name, t.slug, COALESCE\(t.color, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	_, err := svc.Update(context.Background(), siteID, postID, req)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_WithPostMeta(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "title", "slug", "content", "excerpt", "status", "author_id", "published_at", "scheduled_at", "post_meta", "metadata", "created_at", "updated_at", "deleted_at"}).
		AddRow(postID, siteID, "Title", "title", `[]`, "", "draft", uuid.New(), nil, nil, `{}`, `{}`, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(getRows)

	mock.ExpectQuery(`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE\(c.description, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT t.id, t.site_id, t.name, t.slug, COALESCE\(t.color, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	meta := map[string]interface{}{"views": float64(100), "featured": true}
	req := UpdatePostRequest{
		PostMeta: &meta,
	}

	mock.ExpectExec(`UPDATE posts SET post_meta = \$1::jsonb, updated_at = NOW\(\) WHERE id = \$2 AND site_id = \$3 AND deleted_at IS NULL`).
		WithArgs(`{"featured":true,"views":100}`, postID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	getRows2 := pgxmock.NewRows([]string{"id", "site_id", "title", "slug", "content", "excerpt", "status", "author_id", "published_at", "scheduled_at", "post_meta", "metadata", "created_at", "updated_at", "deleted_at"}).
		AddRow(postID, siteID, "Title", "title", `[]`, "", "draft", uuid.New(), nil, nil, `{"featured":true,"views":100}`, `{}`, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(getRows2)

	mock.ExpectQuery(`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE\(c.description, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT t.id, t.site_id, t.name, t.slug, COALESCE\(t.color, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	_, err := svc.Update(context.Background(), siteID, postID, req)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Delete_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "title", "slug", "content", "excerpt", "status", "author_id", "published_at", "scheduled_at", "post_meta", "metadata", "created_at", "updated_at", "deleted_at"}).
		AddRow(postID, siteID, "To Delete", "to-delete", `[]`, "", "draft", uuid.New(), nil, nil, `{}`, `{}`, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(getRows)

	mock.ExpectQuery(`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE\(c.description, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT t.id, t.site_id, t.name, t.slug, COALESCE\(t.color, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectExec(`UPDATE posts SET deleted_at = NOW\(\), updated_at = NOW\(\) WHERE id = \$1 AND site_id = \$2 AND deleted_at IS NULL`).
		WithArgs(postID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := svc.Delete(context.Background(), siteID, postID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(pgxmock.NewRows(nil))

	err := svc.Delete(context.Background(), siteID, postID)
	if err != ErrPostNotFound {
		t.Errorf("expected ErrPostNotFound, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_SetStatus_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "title", "slug", "content", "excerpt", "status", "author_id", "published_at", "scheduled_at", "post_meta", "metadata", "created_at", "updated_at", "deleted_at"}).
		AddRow(postID, siteID, "To Publish", "to-publish", `[]`, "", "draft", uuid.New(), nil, nil, `{}`, `{}`, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(getRows)

	mock.ExpectQuery(`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE\(c.description, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT t.id, t.site_id, t.name, t.slug, COALESCE\(t.color, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectExec(`UPDATE posts SET status = \$1, published_at = COALESCE\(\$2, published_at\), updated_at = NOW\(\) WHERE id = \$3 AND site_id = \$4 AND deleted_at IS NULL`).
		WithArgs("published", pgxmock.AnyArg(), postID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := svc.SetStatus(context.Background(), siteID, postID, PostStatusPublished)
	if err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_SetStatus_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(pgxmock.NewRows(nil))

	err := svc.SetStatus(context.Background(), siteID, postID, PostStatusArchived)
	if err != ErrPostNotFound {
		t.Errorf("expected ErrPostNotFound, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_SetStatus_ToArchived(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "title", "slug", "content", "excerpt", "status", "author_id", "published_at", "scheduled_at", "post_meta", "metadata", "created_at", "updated_at", "deleted_at"}).
		AddRow(postID, siteID, "To Archive", "to-archive", `[]`, "", "published", uuid.New(), &now, nil, `{}`, `{}`, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(getRows)

	mock.ExpectQuery(`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE\(c.description, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT t.id, t.site_id, t.name, t.slug, COALESCE\(t.color, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectExec(`UPDATE posts SET status = \$1, published_at = COALESCE\(\$2, published_at\), updated_at = NOW\(\) WHERE id = \$3 AND site_id = \$4 AND deleted_at IS NULL`).
		WithArgs("archived", pgxmock.AnyArg(), postID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := svc.SetStatus(context.Background(), siteID, postID, PostStatusArchived)
	if err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_List_PageNormalization(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts p WHERE p.deleted_at IS NULL AND p.site_id = \$1`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`SELECT p.id, p.title, p.slug, COALESCE\(p.excerpt, ''\), p.status, p.author_id, p.published_at, p.created_at, p.updated_at`).
		WithArgs(siteID, 20, 0).
		WillReturnRows(pgxmock.NewRows(nil))

	resp, err := svc.List(context.Background(), PostListRequest{
		SiteID:  siteID,
		Page:    0,
		PerPage: 0,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if resp.Page != 1 {
		t.Errorf("expected page 1, got %d", resp.Page)
	}
	if resp.PerPage != 20 {
		t.Errorf("expected per_page 20, got %d", resp.PerPage)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_List_PerPageClamp(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts p WHERE p.deleted_at IS NULL AND p.site_id = \$1`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`SELECT p.id, p.title, p.slug, COALESCE\(p.excerpt, ''\), p.status, p.author_id, p.published_at, p.created_at, p.updated_at`).
		WithArgs(siteID, 20, 0).
		WillReturnRows(pgxmock.NewRows(nil))

	_, err := svc.List(context.Background(), PostListRequest{
		SiteID:  siteID,
		Page:    1,
		PerPage: 200,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Create_WithRelations(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	authorID := uuid.New()
	catID := uuid.New()
	tagID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts WHERE site_id = \$1 AND slug = \$2 AND deleted_at IS NULL`).
		WithArgs(siteID, "test-post").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`INSERT INTO posts`).
		WithArgs(pgxmock.AnyArg(), siteID, "Test Post", "test-post", "[]", "", "draft", authorID, pgxmock.AnyArg(), pgxmock.AnyArg(), "{}", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectExec(`DELETE FROM post_categories WHERE post_id = \$1`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	mock.ExpectExec(`INSERT INTO post_categories \(post_id, category_id\) VALUES \(\$1, \$2\) ON CONFLICT DO NOTHING`).
		WithArgs(pgxmock.AnyArg(), catID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectExec(`DELETE FROM post_tags WHERE post_id = \$1`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	mock.ExpectExec(`INSERT INTO post_tags \(post_id, tag_id\) VALUES \(\$1, \$2\) ON CONFLICT DO NOTHING`).
		WithArgs(pgxmock.AnyArg(), tagID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	post, err := svc.Create(context.Background(), siteID, authorID, CreatePostRequest{
		Title:       "Test Post",
		CategoryIDs: []uuid.UUID{catID},
		TagIDs:      []uuid.UUID{tagID},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if post.Title != "Test Post" {
		t.Errorf("expected 'Test Post', got '%s'", post.Title)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_WithRelations(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	postID := uuid.New()
	catID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "title", "slug", "content", "excerpt", "status", "author_id", "published_at", "scheduled_at", "post_meta", "metadata", "created_at", "updated_at", "deleted_at"}).
		AddRow(postID, siteID, "Title", "title", `[]`, "", "draft", uuid.New(), nil, nil, `{}`, `{}`, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(getRows)

	mock.ExpectQuery(`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE\(c.description, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT t.id, t.site_id, t.name, t.slug, COALESCE\(t.color, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectExec(`DELETE FROM post_categories WHERE post_id = \$1`).
		WithArgs(postID).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	mock.ExpectExec(`INSERT INTO post_categories \(post_id, category_id\) VALUES \(\$1, \$2\) ON CONFLICT DO NOTHING`).
		WithArgs(postID, catID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	getRows2 := pgxmock.NewRows([]string{"id", "site_id", "title", "slug", "content", "excerpt", "status", "author_id", "published_at", "scheduled_at", "post_meta", "metadata", "created_at", "updated_at", "deleted_at"}).
		AddRow(postID, siteID, "Title", "title", `[]`, "", "draft", uuid.New(), nil, nil, `{}`, `{}`, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, title, slug, COALESCE\(content::text, '\[\]'\)`).
		WithArgs(postID, siteID).
		WillReturnRows(getRows2)

	mock.ExpectQuery(`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE\(c.description, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT t.id, t.site_id, t.name, t.slug, COALESCE\(t.color, ''\)`).
		WithArgs(postID).
		WillReturnRows(pgxmock.NewRows(nil))

	_, err := svc.Update(context.Background(), siteID, postID, UpdatePostRequest{
		CategoryIDs: []uuid.UUID{catID},
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
