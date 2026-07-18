package categories

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

func TestService_CategoryErrorConstants(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrCategoryNotFound, "category not found"},
		{ErrCategorySlugExists, "category slug already exists"},
		{ErrInvalidParentCategory, "invalid parent category"},
		{ErrDatabaseNotAvail, "database not available"},
		{ErrCircularParent, "circular parent reference detected"},
	}
	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, tt.err.Error())
		}
	}
}

func TestService_CategoryEventConstants(t *testing.T) {
	_ = []string{
		string(EventCategoryCreated),
		string(EventCategoryUpdated),
		string(EventCategoryDeleted),
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

	_, err := svc.Create(context.Background(), uuid.New(), &CreateCategoryRequest{
		Name: "Test Category",
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_Create_EmptyName(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.Create(context.Background(), uuid.New(), &CreateCategoryRequest{

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

	_, err := svc.GetBySlug(context.Background(), uuid.New(), "test-category")
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

func TestService_Tree_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.Tree(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got: %v", err)
	}
}

func TestService_Update_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), UpdateCategoryRequest{})
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

	svc.fireEvent(context.Background(), EventCategoryCreated, map[string]interface{}{
		"category_id": uuid.New().String(),
	}, uuid.New())
}

func TestService_FireEvent_WithoutBus(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	svc.fireEvent(context.Background(), EventCategoryCreated, map[string]interface{}{
		"category_id": uuid.New().String(),
	}, uuid.New())
}

func TestService_FireEvent_WithSubscriber(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)

	bus := kernel.NewEventBus(log)
	svc.SetEventBus(bus)

	received := make(chan string, 1)
	bus.Subscribe(EventCategoryCreated, func(ctx context.Context, event kernel.Event) error {
		if p, ok := event.Payload.(map[string]interface{}); ok {
			if name, ok := p["name"]; ok {
				received <- name.(string)
			}
		}
		return nil
	})

	svc.fireEvent(context.Background(), EventCategoryCreated, map[string]interface{}{
		"category_id": uuid.New().String(),
		"name":        "Test Category",
	}, uuid.New())

	select {
	case name := <-received:
		if name != "Test Category" {
			t.Errorf("expected 'Test Category', got '%s'", name)
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
		{"My Category", "my-category"},
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

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM categories WHERE site_id = \$1 AND slug = \$2 AND deleted_at IS NULL`).
		WithArgs(siteID, "test-category").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`INSERT INTO categories`).
		WithArgs(pgxmock.AnyArg(), siteID, pgxmock.AnyArg(), "Test Category", "test-category", "Description", "icon-star", "#ff0000", 1, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	cat, err := svc.Create(context.Background(), siteID, &CreateCategoryRequest{
		Name:        "Test Category",
		Description: "Description",
		Icon:        "icon-star",
		Color:       "#ff0000",
		SortOrder:   1,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if cat.Name != "Test Category" {
		t.Errorf("expected 'Test Category', got '%s'", cat.Name)
	}
	if cat.Slug != "test-category" {
		t.Errorf("expected 'test-category', got '%s'", cat.Slug)
	}
	if cat.Color != "#ff0000" {
		t.Errorf("expected '#ff0000', got '%s'", cat.Color)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Create_WithParent(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	parentID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM categories WHERE site_id = \$1 AND slug = \$2 AND deleted_at IS NULL`).
		WithArgs(siteID, "child").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	parentRows := pgxmock.NewRows([]string{"id", "site_id", "parent_id", "name", "slug", "description", "icon", "color", "sort_order", "created_at", "updated_at", "deleted_at"}).
		AddRow(parentID, siteID, nil, "Parent", "parent", "", "", "", 0, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(parentID, siteID).
		WillReturnRows(parentRows)

	mock.ExpectExec(`INSERT INTO categories`).
		WithArgs(pgxmock.AnyArg(), siteID, &parentID, "Child", "child", "", "", "", 0, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	cat, err := svc.Create(context.Background(), siteID, &CreateCategoryRequest{
		Name:     "Child",
		ParentID: &parentID,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if cat.ParentID == nil || *cat.ParentID != parentID {
		t.Errorf("expected parent ID %s, got %v", parentID, cat.ParentID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Create_InvalidParent(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	parentID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM categories WHERE site_id = \$1 AND slug = \$2 AND deleted_at IS NULL`).
		WithArgs(siteID, "child").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(parentID, siteID).
		WillReturnRows(pgxmock.NewRows(nil))

	_, err := svc.Create(context.Background(), siteID, &CreateCategoryRequest{
		Name:     "Child",
		ParentID: &parentID,
	})
	if err != ErrInvalidParentCategory {
		t.Errorf("expected ErrInvalidParentCategory, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Create_SlugCollision(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM categories WHERE site_id = \$1 AND slug = \$2 AND deleted_at IS NULL`).
		WithArgs(siteID, "test-category").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM categories WHERE site_id = \$1 AND slug = \$2 AND deleted_at IS NULL`).
		WithArgs(siteID, "test-category-1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`INSERT INTO categories`).
		WithArgs(pgxmock.AnyArg(), siteID, pgxmock.AnyArg(), "Test Category", "test-category-1", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), 0, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	cat, err := svc.Create(context.Background(), siteID, &CreateCategoryRequest{
		Name: "Test Category",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if cat.Slug != "test-category-1" {
		t.Errorf("expected 'test-category-1', got '%s'", cat.Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_GetByID_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	catID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{"id", "site_id", "parent_id", "name", "slug", "description", "icon", "color", "sort_order", "created_at", "updated_at", "deleted_at"}).
		AddRow(catID, siteID, nil, "My Category", "my-category", "A desc", "icon", "blue", 0, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(catID, siteID).
		WillReturnRows(rows)

	cat, err := svc.GetByID(context.Background(), siteID, catID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if cat.ID != catID {
		t.Errorf("expected cat ID %s, got %s", catID, cat.ID)
	}
	if cat.Name != "My Category" {
		t.Errorf("expected 'My Category', got '%s'", cat.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	catID := uuid.New()

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(catID, siteID).
		WillReturnRows(pgxmock.NewRows(nil))

	_, err := svc.GetByID(context.Background(), siteID, catID)
	if err != ErrCategoryNotFound {
		t.Errorf("expected ErrCategoryNotFound, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_GetBySlug_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	catID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{"id", "site_id", "parent_id", "name", "slug", "description", "icon", "color", "sort_order", "created_at", "updated_at", "deleted_at"}).
		AddRow(catID, siteID, nil, "Slug Cat", "slug-cat", "", "", "", 0, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(siteID, "slug-cat").
		WillReturnRows(rows)

	cat, err := svc.GetBySlug(context.Background(), siteID, "slug-cat")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if cat.Slug != "slug-cat" {
		t.Errorf("expected 'slug-cat', got '%s'", cat.Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_List_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	catID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{"id", "site_id", "parent_id", "name", "slug", "description", "icon", "color", "sort_order", "created_at", "updated_at", "deleted_at"}).
		AddRow(catID, siteID, nil, "Category A", "category-a", "", "", "", 0, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(siteID).
		WillReturnRows(rows)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM categories WHERE site_id = \$1 AND deleted_at IS NULL`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	resp, err := svc.List(context.Background(), siteID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(resp.Categories) != 1 {
		t.Errorf("expected 1 category, got %d", len(resp.Categories))
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

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows(nil))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM categories WHERE site_id = \$1 AND deleted_at IS NULL`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	resp, err := svc.List(context.Background(), siteID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(resp.Categories) != 0 {
		t.Errorf("expected 0 categories, got %d", len(resp.Categories))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Tree_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{"id", "site_id", "parent_id", "name", "slug", "description", "icon", "color", "sort_order", "created_at", "updated_at", "deleted_at"}).
		AddRow(parentID, siteID, nil, "Parent", "parent", "", "", "", 0, now, now, nil).
		AddRow(childID, siteID, &parentID, "Child", "child", "", "", "", 1, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(siteID).
		WillReturnRows(rows)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM categories WHERE site_id = \$1 AND deleted_at IS NULL`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	tree, err := svc.Tree(context.Background(), siteID)
	if err != nil {
		t.Fatalf("Tree failed: %v", err)
	}
	if len(tree) != 1 {
		t.Errorf("expected 1 root, got %d", len(tree))
	}

	lookup := make(map[uuid.UUID]Category)
	var flatten func([]Category)
	flatten = func(cats []Category) {
		for _, c := range cats {
			lookup[c.ID] = c
			flatten(c.Children)
		}
	}
	flatten(tree)

	if _, ok := lookup[parentID]; !ok {
		t.Error("expected parent in tree")
	}
	if _, ok := lookup[childID]; !ok {
		t.Error("expected child in tree")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Tree_Flat(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	cat1 := uuid.New()
	cat2 := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{"id", "site_id", "parent_id", "name", "slug", "description", "icon", "color", "sort_order", "created_at", "updated_at", "deleted_at"}).
		AddRow(cat1, siteID, nil, "A", "a", "", "", "", 0, now, now, nil).
		AddRow(cat2, siteID, nil, "B", "b", "", "", "", 1, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(siteID).
		WillReturnRows(rows)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM categories WHERE site_id = \$1 AND deleted_at IS NULL`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	tree, err := svc.Tree(context.Background(), siteID)
	if err != nil {
		t.Fatalf("Tree failed: %v", err)
	}
	if len(tree) != 2 {
		t.Errorf("expected 2 roots, got %d", len(tree))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_WithMockDB(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	catID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "parent_id", "name", "slug", "description", "icon", "color", "sort_order", "created_at", "updated_at", "deleted_at"}).
		AddRow(catID, siteID, nil, "Original", "original", "Old desc", "old-icon", "#000", 0, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(catID, siteID).
		WillReturnRows(getRows)

	name := "Updated"
	desc := "New desc"
	icon := "new-icon"
	color := "#fff"
	sortOrder := 5
	req := UpdateCategoryRequest{
		Name:        &name,
		Description: &desc,
		Icon:        &icon,
		Color:       &color,
		SortOrder:   &sortOrder,
	}

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM categories WHERE site_id = \$1 AND slug = \$2 AND id != \$3 AND deleted_at IS NULL`).
		WithArgs(siteID, "updated", catID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`UPDATE categories SET name = \$1, slug = \$2, description = \$3, icon = \$4, color = \$5, sort_order = \$6, updated_at = NOW\(\) WHERE id = \$7 AND site_id = \$8 AND deleted_at IS NULL`).
		WithArgs("Updated", "updated", "New desc", "new-icon", "#fff", 5, catID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	updatedRows := pgxmock.NewRows([]string{"id", "site_id", "parent_id", "name", "slug", "description", "icon", "color", "sort_order", "created_at", "updated_at", "deleted_at"}).
		AddRow(catID, siteID, nil, "Updated", "updated", "New desc", "new-icon", "#fff", 5, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(catID, siteID).
		WillReturnRows(updatedRows)

	cat, err := svc.Update(context.Background(), siteID, catID, req)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if cat.Name != "Updated" {
		t.Errorf("expected 'Updated', got '%s'", cat.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_NoChanges(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	catID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "parent_id", "name", "slug", "description", "icon", "color", "sort_order", "created_at", "updated_at", "deleted_at"}).
		AddRow(catID, siteID, nil, "Same", "same", "", "", "", 0, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(catID, siteID).
		WillReturnRows(getRows)

	cat, err := svc.Update(context.Background(), siteID, catID, UpdateCategoryRequest{})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if cat.Name != "Same" {
		t.Errorf("expected 'Same', got '%s'", cat.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	catID := uuid.New()

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(catID, siteID).
		WillReturnRows(pgxmock.NewRows(nil))

	name := "Updated"
	_, err := svc.Update(context.Background(), siteID, catID, UpdateCategoryRequest{
		Name: &name,
	})
	if err != ErrCategoryNotFound {
		t.Errorf("expected ErrCategoryNotFound, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_ParentSelfReference(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	catID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "parent_id", "name", "slug", "description", "icon", "color", "sort_order", "created_at", "updated_at", "deleted_at"}).
		AddRow(catID, siteID, nil, "Self", "self", "", "", "", 0, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(catID, siteID).
		WillReturnRows(getRows)

	parentID := &catID
	_, err := svc.Update(context.Background(), siteID, catID, UpdateCategoryRequest{
		ParentID: &parentID,
	})
	if err != ErrCircularParent {
		t.Errorf("expected ErrCircularParent, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_SetParentNil(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	siteID := uuid.New()
	catID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "parent_id", "name", "slug", "description", "icon", "color", "sort_order", "created_at", "updated_at", "deleted_at"}).
		AddRow(catID, siteID, nil, "Cat", "cat", "", "", "", 0, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(catID, siteID).
		WillReturnRows(getRows)

	var nilPtr *uuid.UUID
	req := UpdateCategoryRequest{
		ParentID: &nilPtr,
	}

	mock.ExpectExec(`UPDATE categories SET parent_id = \$1, updated_at = NOW\(\) WHERE id = \$2 AND site_id = \$3 AND deleted_at IS NULL`).
		WithArgs(nil, catID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	updatedRows := pgxmock.NewRows([]string{"id", "site_id", "parent_id", "name", "slug", "description", "icon", "color", "sort_order", "created_at", "updated_at", "deleted_at"}).
		AddRow(catID, siteID, nil, "Cat", "cat", "", "", "", 0, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(catID, siteID).
		WillReturnRows(updatedRows)

	_, err := svc.Update(context.Background(), siteID, catID, req)
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
	catID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	getRows := pgxmock.NewRows([]string{"id", "site_id", "parent_id", "name", "slug", "description", "icon", "color", "sort_order", "created_at", "updated_at", "deleted_at"}).
		AddRow(catID, siteID, nil, "To Delete", "to-delete", "", "", "", 0, now, now, nil)

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(catID, siteID).
		WillReturnRows(getRows)

	mock.ExpectExec(`UPDATE categories SET deleted_at = NOW\(\), updated_at = NOW\(\) WHERE id = \$1 AND site_id = \$2 AND deleted_at IS NULL`).
		WithArgs(catID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := svc.Delete(context.Background(), siteID, catID)
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
	catID := uuid.New()

	mock.ExpectQuery(`SELECT id, site_id, parent_id, name, slug, COALESCE\(description, ''\), COALESCE\(icon, ''\), COALESCE\(color, ''\), sort_order`).
		WithArgs(catID, siteID).
		WillReturnRows(pgxmock.NewRows(nil))

	err := svc.Delete(context.Background(), siteID, catID)
	if err != ErrCategoryNotFound {
		t.Errorf("expected ErrCategoryNotFound, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
