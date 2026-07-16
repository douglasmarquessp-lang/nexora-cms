package media

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
	"nexora/internal/pkg/storage"
)

func setupServiceWithMock(t *testing.T) (*Service, pgxmock.PgxPoolIface) {
	t.Helper()
	cfg := &config.Config{}
	log := logger.New(cfg)

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}

	db := &database.Database{Pool: mock}
	ch := cache.New(true)
	st := storage.NewLocalDriver(t.TempDir(), "/uploads")

	svc := NewService(cfg, log, db, ch, st)
	return svc, mock
}

func TestService_GetByID_WithMock(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer func() { mock.Close() }()

	mediaID := uuid.New()
	siteID := uuid.New()
	now := time.Now()

	w, h := 100, 200
	rows := pgxmock.NewRows([]string{
		"id", "site_id", "folder_id", "filename", "original_name", "mime_type", "extension",
		"size", "width", "height", "duration", "hash", "alt_text", "caption",
		"storage_provider", "storage_key", "metadata", "created_by", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		mediaID, siteID, nil, "test.jpg", "original.jpg", "image/jpeg", "jpg",
		int64(1024), &w, &h, 0, "abc123", "alt", "caption", "local", "path/file.jpg",
		[]byte("{}"), uuid.New(), now, now, nil,
	)

	mock.ExpectQuery(`SELECT .+ FROM media WHERE`).
		WithArgs(mediaID, siteID).
		WillReturnRows(rows)

	m, err := svc.GetByID(context.Background(), siteID, mediaID)
	if err != nil {
		t.Fatal(err)
	}
	if m.AltText != "alt" {
		t.Errorf("AltText = %q", m.AltText)
	}
	if m.Width == nil || *m.Width != 100 {
		t.Errorf("Width = %v", m.Width)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	mediaID := uuid.New()
	siteID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM media WHERE`).
		WithArgs(mediaID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := svc.GetByID(context.Background(), siteID, mediaID)
	if err != ErrMediaNotFound {
		t.Errorf("expected ErrMediaNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Update_WithMock(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	mediaID := uuid.New()
	siteID := uuid.New()
	now := time.Now()
	folderID := uuid.New()
	w, h := 100, 200

	rows := pgxmock.NewRows([]string{
		"id", "site_id", "folder_id", "filename", "original_name", "mime_type", "extension",
		"size", "width", "height", "duration", "hash", "alt_text", "caption",
		"storage_provider", "storage_key", "metadata", "created_by", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		mediaID, siteID, &folderID, "test.jpg", "original.jpg", "image/jpeg", "jpg",
		int64(1024), &w, &h, 0, "abc123", "new alt", "new caption", "local", "path/file.jpg",
		[]byte("{}"), uuid.New(), now, now, nil,
	)

	mock.ExpectExec(`UPDATE media SET`).
		WithArgs(folderID, "new alt", "new caption", mediaID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectQuery(`SELECT .+ FROM media WHERE`).
		WithArgs(mediaID, siteID).
		WillReturnRows(rows)

	altText := "new alt"
	caption := "new caption"
	updated, err := svc.Update(context.Background(), siteID, mediaID, UpdateMediaRequest{
		AltText:  &altText,
		Caption:  &caption,
		FolderID: &folderID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.AltText != "new alt" {
		t.Errorf("AltText = %q", updated.AltText)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Delete_WithMock(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	mediaID := uuid.New()
	siteID := uuid.New()
	now := time.Now()

	getRows := pgxmock.NewRows([]string{
		"id", "site_id", "folder_id", "filename", "original_name", "mime_type", "extension",
		"size", "width", "height", "duration", "hash", "alt_text", "caption",
		"storage_provider", "storage_key", "metadata", "created_by", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		mediaID, siteID, nil, "test.jpg", "original.jpg", "image/jpeg", "jpg",
		int64(1024), nil, nil, 0, "abc123", "", "", "local", "path/file.jpg",
		[]byte("{}"), uuid.New(), now, now, nil,
	)

	mock.ExpectQuery(`SELECT .+ FROM media WHERE`).
		WithArgs(mediaID, siteID).
		WillReturnRows(getRows)

	mock.ExpectExec(`UPDATE media SET deleted_at`).
		WithArgs(mediaID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectQuery(`SELECT .+ FROM media_variants WHERE`).
		WithArgs(mediaID).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "media_id", "variant", "width", "height", "file_size", "mime_type",
			"storage_key", "metadata", "created_at",
		}))

	err := svc.Delete(context.Background(), siteID, mediaID)
	if err != nil {
		t.Fatal(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Move_WithMock(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	siteID := uuid.New()
	folderID := uuid.New()
	mediaIDs := []uuid.UUID{uuid.New()}

	folderRows := pgxmock.NewRows([]string{
		"id", "site_id", "parent_id", "name", "slug", "description", "sort_order",
		"created_by", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		folderID, siteID, nil, "Folder", "folder", "", 0, uuid.New(), time.Now(), time.Now(), nil,
	)

	mock.ExpectQuery(`SELECT .+ FROM folders WHERE`).
		WithArgs(folderID, siteID).
		WillReturnRows(folderRows)

	mock.ExpectExec(`UPDATE media SET folder_id`).
		WithArgs(siteID, mediaIDs[0], folderID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := svc.Move(context.Background(), siteID, mediaIDs, &folderID)
	if err != nil {
		t.Fatal(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_CreateFolder_WithMock(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	siteID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec(`INSERT INTO folders`).
		WithArgs(pgxmock.AnyArg(), siteID, nil, "New Folder", "new-folder", "", 0, userID, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	folder, err := svc.CreateFolder(context.Background(), siteID, userID, CreateFolderRequest{
		Name: "New Folder",
	})
	if err != nil {
		t.Fatal(err)
	}
	if folder.Name != "New Folder" {
		t.Errorf("Name = %q", folder.Name)
	}
	if folder.Slug != "new-folder" {
		t.Errorf("Slug = %q", folder.Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_List_WithMock(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	siteID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media m WHERE`).
		WithArgs(siteID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	rows := pgxmock.NewRows([]string{
		"id", "site_id", "folder_id", "filename", "original_name", "mime_type", "extension",
		"size", "width", "height", "duration", "hash", "alt_text", "caption",
		"storage_provider", "storage_key", "metadata", "created_by", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		uuid.New(), siteID, nil, "test.jpg", "original.jpg", "image/jpeg", "jpg",
		int64(1024), nil, nil, 0, "abc123", "", "", "local", "path/file.jpg",
		[]byte("{}"), uuid.New(), now, now, nil,
	)

	mock.ExpectQuery(`SELECT .+ FROM media m WHERE`).
		WithArgs(siteID, 20, 0).
		WillReturnRows(rows)

	resp, err := svc.List(context.Background(), MediaListRequest{
		SiteID: siteID,
		Page:   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 1 {
		t.Errorf("Total = %d", resp.Total)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Search_WithMock(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	siteID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media m WHERE`).
		WithArgs(siteID, "%test%").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	rows := pgxmock.NewRows([]string{
		"id", "site_id", "folder_id", "filename", "original_name", "mime_type", "extension",
		"size", "width", "height", "duration", "hash", "alt_text", "caption",
		"storage_provider", "storage_key", "metadata", "created_by", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		uuid.New(), siteID, nil, "test1.jpg", "test image 1.jpg", "image/jpeg", "jpg",
		int64(500), nil, nil, 0, "hash1", "", "", "local", "path1.jpg",
		[]byte("{}"), uuid.New(), now, now, nil,
	).AddRow(
		uuid.New(), siteID, nil, "test2.png", "test image 2.png", "image/png", "png",
		int64(700), nil, nil, 0, "hash2", "", "", "local", "path2.png",
		[]byte("{}"), uuid.New(), now, now, nil,
	)

	mock.ExpectQuery(`SELECT .+ FROM media m WHERE`).
		WithArgs(siteID, "%test%", 20, 0).
		WillReturnRows(rows)

	resp, err := svc.Search(context.Background(), siteID, "test", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 2 {
		t.Errorf("Total = %d, want 2", resp.Total)
	}
	if len(resp.Media) != 2 {
		t.Errorf("Media count = %d, want 2", len(resp.Media))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_Restore_WithMock(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	mediaID := uuid.New()
	siteID := uuid.New()

	mock.ExpectExec(`UPDATE media SET deleted_at`).
		WithArgs(mediaID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := svc.Restore(context.Background(), siteID, mediaID)
	if err != nil {
		t.Fatal(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_DeleteFolder_WithMock(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	siteID := uuid.New()
	folderID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media WHERE`).
		WithArgs(folderID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM folders WHERE`).
		WithArgs(folderID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(`UPDATE folders SET deleted_at`).
		WithArgs(folderID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := svc.DeleteFolder(context.Background(), siteID, folderID)
	if err != nil {
		t.Fatal(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_DeleteFolder_NotEmpty(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	folderID := uuid.New()
	siteID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media WHERE`).
		WithArgs(folderID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(5))

	err := svc.DeleteFolder(context.Background(), siteID, folderID)
	if err != ErrFolderNotEmpty {
		t.Errorf("expected ErrFolderNotEmpty, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_ListFolders_WithMock(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	siteID := uuid.New()
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "site_id", "parent_id", "name", "slug", "description", "sort_order",
		"created_by", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		uuid.New(), siteID, nil, "Folder 1", "folder-1", "", 0, uuid.New(), now, now, nil,
	).AddRow(
		uuid.New(), siteID, nil, "Folder 2", "folder-2", "", 1, uuid.New(), now, now, nil,
	)

	mock.ExpectQuery(`SELECT .+ FROM folders WHERE`).
		WithArgs(siteID).
		WillReturnRows(rows)

	folders, err := svc.ListFolders(context.Background(), siteID)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 2 {
		t.Errorf("expected 2 folders, got %d", len(folders))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_GetFolderByID_WithMock(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	folderID := uuid.New()
	siteID := uuid.New()
	now := time.Now()

	rows := pgxmock.NewRows([]string{
		"id", "site_id", "parent_id", "name", "slug", "description", "sort_order",
		"created_by", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		folderID, siteID, nil, "Test Folder", "test-folder", "Description", 0, uuid.New(), now, now, nil,
	)

	mock.ExpectQuery(`SELECT .+ FROM folders WHERE`).
		WithArgs(folderID, siteID).
		WillReturnRows(rows)

	f, err := svc.GetFolderByID(context.Background(), siteID, folderID)
	if err != nil {
		t.Fatal(err)
	}
	if f.Name != "Test Folder" {
		t.Errorf("Name = %q", f.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_UpdateFolder_WithMock(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	folderID := uuid.New()
	siteID := uuid.New()
	now := time.Now()

	name := "Updated Name"

	mock.ExpectExec(`UPDATE folders SET name`).
		WithArgs("Updated Name", folderID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	rows := pgxmock.NewRows([]string{
		"id", "site_id", "parent_id", "name", "slug", "description", "sort_order",
		"created_by", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		folderID, siteID, nil, "Updated Name", "updated-name", "Updated desc", 0, uuid.New(), now, now, nil,
	)

	mock.ExpectQuery(`SELECT .+ FROM folders WHERE`).
		WithArgs(folderID, siteID).
		WillReturnRows(rows)

	f, err := svc.UpdateFolder(context.Background(), siteID, folderID, UpdateFolderRequest{
		Name: &name,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.Name != "Updated Name" {
		t.Errorf("Name = %q", f.Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
