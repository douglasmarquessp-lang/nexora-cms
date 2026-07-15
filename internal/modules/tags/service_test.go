package tags

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

func TestService_TagErrorConstants(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrTagNotFound, "tag not found"},
		{ErrTagSlugExists, "tag slug already exists"},
		{ErrDatabaseNotAvail, "database not available"},
	}
	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, tt.err.Error())
		}
	}
}

func TestService_TagEventConstants(t *testing.T) {
	_ = []string{
		string(EventTagCreated),
		string(EventTagUpdated),
		string(EventTagDeleted),
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

	_, err := svc.Create(context.Background(), uuid.New(), CreateTagRequest{
		Name: "Test Tag",
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_Create_EmptyName(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), CreateTagRequest{
		Name: "",
	})
	if err == nil {
		t.Fatal("expected error for empty name")
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

	_, err := svc.GetBySlug(context.Background(), uuid.New(), "test-tag")
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_List_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.List(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_Update_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), UpdateTagRequest{})
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

	svc.fireEvent(context.Background(), EventTagCreated, map[string]interface{}{
		"tag_id": uuid.New().String(),
	})
}

func TestService_FireEvent_WithoutBus(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	svc.fireEvent(context.Background(), EventTagCreated, map[string]interface{}{
		"tag_id": uuid.New().String(),
	})
}

func TestService_FireEvent_WithSubscriber(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	bus := kernel.NewEventBus(log)
	svc.SetEventBus(bus)

	received := make(chan string, 1)
	bus.Subscribe(EventTagCreated, func(ctx context.Context, event kernel.Event) error {
		if p, ok := event.Payload.(map[string]interface{}); ok {
			if name, ok := p["name"]; ok {
				received <- name.(string)
			}
		}
		return nil
	})

	svc.fireEvent(context.Background(), EventTagCreated, map[string]interface{}{
		"tag_id": uuid.New().String(),
		"name":   "Test Tag",
	})

	select {
	case name := <-received:
		if name != "Test Tag" {
			t.Errorf("expected 'Test Tag', got '%s'", name)
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
		{"My Tag", "my-tag"},
		{"  Hello  ", "hello"},
		{"Special!@#", "special"},
		{"UPPERCASE", "uppercase"},
		{"already-slug", "already-slug"},
	}
	for _, tt := range tests {
		result := generateSlug(tt.input)
		if result != tt.expected {
			t.Errorf("generateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestService_Create_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tags WHERE site_id = \$1 AND slug = \$2 AND deleted_at IS NULL`).
		WithArgs(siteID, "test-tag").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`INSERT INTO tags`).
		WithArgs(pgxmock.AnyArg(), siteID, "Test Tag", "test-tag", "#ff0000", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	tag, err := svc.Create(context.Background(), siteID, CreateTagRequest{
		Name:  "Test Tag",
		Color: "#ff0000",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if tag.Name != "Test Tag" {
		t.Errorf("expected 'Test Tag', got '%s'", tag.Name)
	}
	if tag.Slug != "test-tag" {
		t.Errorf("expected 'test-tag', got '%s'", tag.Slug)
	}
	if tag.Color != "#ff0000" {
		t.Errorf("expected '#ff0000', got '%s'", tag.Color)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Create_SlugCollision(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tags WHERE site_id = \$1 AND slug = \$2 AND deleted_at IS NULL`).
		WithArgs(siteID, "test-tag").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tags WHERE site_id = \$1 AND slug = \$2 AND deleted_at IS NULL`).
		WithArgs(siteID, "test-tag-1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`INSERT INTO tags`).
		WithArgs(pgxmock.AnyArg(), siteID, "Test Tag", "test-tag-1", "", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	tag, err := svc.Create(context.Background(), siteID, CreateTagRequest{
		Name: "Test Tag",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if tag.Slug != "test-tag-1" {
		t.Errorf("expected 'test-tag-1', got '%s'", tag.Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_GetByID_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	tagID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{"id", "site_id", "name", "slug", "color", "created_at", "updated_at", "deleted_at"}).
		AddRow(tagID, siteID, "My Tag", "my-tag", "blue", now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, name, slug, COALESCE\(color, ''\), created_at, updated_at, deleted_at`).
		WithArgs(tagID, siteID).
		WillReturnRows(rows)

	tag, err := svc.GetByID(context.Background(), siteID, tagID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if tag.ID != tagID {
		t.Errorf("expected tag ID %s, got %s", tagID, tag.ID)
	}
	if tag.Name != "My Tag" {
		t.Errorf("expected 'My Tag', got '%s'", tag.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	tagID := uuid.New()

	mock.ExpectQuery(`SELECT id, site_id, name, slug, COALESCE\(color, ''\), created_at, updated_at, deleted_at`).
		WithArgs(tagID, siteID).
		WillReturnRows(pgxmock.NewRows(nil))

	_, err := svc.GetByID(context.Background(), siteID, tagID)
	if err != ErrTagNotFound {
		t.Errorf("expected ErrTagNotFound, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_GetBySlug_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	tagID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{"id", "site_id", "name", "slug", "color", "created_at", "updated_at", "deleted_at"}).
		AddRow(tagID, siteID, "Slug Tag", "slug-tag", "", now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, name, slug, COALESCE\(color, ''\), created_at, updated_at, deleted_at`).
		WithArgs(siteID, "slug-tag").
		WillReturnRows(rows)

	tag, err := svc.GetBySlug(context.Background(), siteID, "slug-tag")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if tag.Slug != "slug-tag" {
		t.Errorf("expected 'slug-tag', got '%s'", tag.Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_List_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	tagID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{"id", "site_id", "name", "slug", "color", "created_at", "updated_at", "deleted_at"}).
		AddRow(tagID, siteID, "Tag A", "tag-a", "", now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, name, slug, COALESCE\(color, ''\), created_at, updated_at, deleted_at`).
		WithArgs(siteID).
		WillReturnRows(rows)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tags WHERE site_id = \$1 AND deleted_at IS NULL`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	resp, err := svc.List(context.Background(), siteID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(resp.Tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(resp.Tags))
	}
	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_List_Empty(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT id, site_id, name, slug, COALESCE\(color, ''\), created_at, updated_at, deleted_at`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tags WHERE site_id = \$1 AND deleted_at IS NULL`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	resp, err := svc.List(context.Background(), siteID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(resp.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(resp.Tags))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	tagID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "name", "slug", "color", "created_at", "updated_at", "deleted_at"}).
		AddRow(tagID, siteID, "Original", "original", "#000", now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, name, slug, COALESCE\(color, ''\), created_at, updated_at, deleted_at`).
		WithArgs(tagID, siteID).
		WillReturnRows(getRows)

	name := "Updated"
	color := "#fff"
	req := UpdateTagRequest{
		Name:  &name,
		Color: &color,
	}

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tags WHERE site_id = \$1 AND slug = \$2 AND id != \$3 AND deleted_at IS NULL`).
		WithArgs(siteID, "updated", tagID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`UPDATE tags SET name = \$1, slug = \$2, color = \$3, updated_at = NOW\(\) WHERE id = \$4 AND site_id = \$5 AND deleted_at IS NULL`).
		WithArgs("Updated", "updated", "#fff", tagID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	getRows2 := pgxmock.NewRows([]string{"id", "site_id", "name", "slug", "color", "created_at", "updated_at", "deleted_at"}).
		AddRow(tagID, siteID, "Updated", "updated", "#fff", now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, name, slug, COALESCE\(color, ''\), created_at, updated_at, deleted_at`).
		WithArgs(tagID, siteID).
		WillReturnRows(getRows2)

	tag, err := svc.Update(context.Background(), siteID, tagID, req)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if tag.Name != "Updated" {
		t.Errorf("expected 'Updated', got '%s'", tag.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_NoChanges(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	tagID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "name", "slug", "color", "created_at", "updated_at", "deleted_at"}).
		AddRow(tagID, siteID, "Same", "same", "", now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, name, slug, COALESCE\(color, ''\), created_at, updated_at, deleted_at`).
		WithArgs(tagID, siteID).
		WillReturnRows(getRows)

	tag, err := svc.Update(context.Background(), siteID, tagID, UpdateTagRequest{})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if tag.Name != "Same" {
		t.Errorf("expected 'Same', got '%s'", tag.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	tagID := uuid.New()

	mock.ExpectQuery(`SELECT id, site_id, name, slug, COALESCE\(color, ''\), created_at, updated_at, deleted_at`).
		WithArgs(tagID, siteID).
		WillReturnRows(pgxmock.NewRows(nil))

	name := "Updated"
	_, err := svc.Update(context.Background(), siteID, tagID, UpdateTagRequest{
		Name: &name,
	})
	if err != ErrTagNotFound {
		t.Errorf("expected ErrTagNotFound, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Delete_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	tagID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "name", "slug", "color", "created_at", "updated_at", "deleted_at"}).
		AddRow(tagID, siteID, "To Delete", "to-delete", "", now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, name, slug, COALESCE\(color, ''\), created_at, updated_at, deleted_at`).
		WithArgs(tagID, siteID).
		WillReturnRows(getRows)

	mock.ExpectExec(`UPDATE tags SET deleted_at = NOW\(\), updated_at = NOW\(\) WHERE id = \$1 AND site_id = \$2 AND deleted_at IS NULL`).
		WithArgs(tagID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := svc.Delete(context.Background(), siteID, tagID)
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
	tagID := uuid.New()

	mock.ExpectQuery(`SELECT id, site_id, name, slug, COALESCE\(color, ''\), created_at, updated_at, deleted_at`).
		WithArgs(tagID, siteID).
		WillReturnRows(pgxmock.NewRows(nil))

	err := svc.Delete(context.Background(), siteID, tagID)
	if err != ErrTagNotFound {
		t.Errorf("expected ErrTagNotFound, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_SlugCollision(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	tagID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "name", "slug", "color", "created_at", "updated_at", "deleted_at"}).
		AddRow(tagID, siteID, "Original", "original", "", now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, name, slug, COALESCE\(color, ''\), created_at, updated_at, deleted_at`).
		WithArgs(tagID, siteID).
		WillReturnRows(getRows)

	name := "New Name"
	req := UpdateTagRequest{Name: &name}

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tags WHERE site_id = \$1 AND slug = \$2 AND id != \$3 AND deleted_at IS NULL`).
		WithArgs(siteID, "new-name", tagID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tags WHERE site_id = \$1 AND slug = \$2 AND id != \$3 AND deleted_at IS NULL`).
		WithArgs(siteID, "new-name-1", tagID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`UPDATE tags SET name = \$1, slug = \$2, updated_at = NOW\(\) WHERE id = \$3 AND site_id = \$4 AND deleted_at IS NULL`).
		WithArgs("New Name", "new-name-1", tagID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	getRows2 := pgxmock.NewRows([]string{"id", "site_id", "name", "slug", "color", "created_at", "updated_at", "deleted_at"}).
		AddRow(tagID, siteID, "New Name", "new-name-1", "", now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, name, slug, COALESCE\(color, ''\), created_at, updated_at, deleted_at`).
		WithArgs(tagID, siteID).
		WillReturnRows(getRows2)

	_, err := svc.Update(context.Background(), siteID, tagID, req)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
