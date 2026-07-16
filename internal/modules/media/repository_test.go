package media

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
)

func setupMockRepo(t *testing.T) (*Repository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	repo := NewRepository(mock)
	return repo, mock
}

func TestRepository_GetByID(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	mediaID := uuid.New()
	siteID := uuid.New()
	now := time.Now()
	w, h := 100, 200

	rows := mock.NewRows([]string{
		"id", "site_id", "folder_id", "filename", "original_name", "mime_type", "extension",
		"size", "width", "height", "duration", "hash", "alt_text", "caption",
		"storage_provider", "storage_key", "metadata", "created_by", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		mediaID, siteID, nil, "test.jpg", "original.jpg", "image/jpeg", "jpg",
		int64(1024), &w, &h, 0, "abc123", "", "", "local", "path/to/file.jpg",
		`{}`, uuid.New(), now, now, nil,
	)

	mock.ExpectQuery(`SELECT .+ FROM media WHERE`).
		WithArgs(mediaID, siteID).
		WillReturnRows(rows)

	m, err := repo.GetByID(context.Background(), siteID, mediaID)
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != mediaID {
		t.Errorf("ID = %v, want %v", m.ID, mediaID)
	}
	if m.Filename != "test.jpg" {
		t.Errorf("Filename = %q", m.Filename)
	}
	if m.MimeType != "image/jpeg" {
		t.Errorf("MimeType = %q", m.MimeType)
	}
	if m.Width == nil || *m.Width != 100 {
		t.Errorf("Width = %v", m.Width)
	}
	if m.Height == nil || *m.Height != 200 {
		t.Errorf("Height = %v", m.Height)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	mediaID := uuid.New()
	siteID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM media WHERE`).
		WithArgs(mediaID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, err := repo.GetByID(context.Background(), siteID, mediaID)
	if err != ErrMediaNotFound {
		t.Errorf("expected ErrMediaNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepository_GetVariants(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	mediaID := uuid.New()
	now := time.Now()

	rows := mock.NewRows([]string{
		"id", "media_id", "variant", "width", "height", "file_size", "mime_type", "storage_key", "metadata", "created_at",
	}	).AddRow(
		uuid.New(), mediaID, "thumbnail", 150, 150, int64(1024), "image/jpeg", "thumb.jpg", []byte("{}"), now,
	).AddRow(
		uuid.New(), mediaID, "small", 320, 240, int64(2048), "image/jpeg", "small.jpg", []byte("{}"), now,
	)

	mock.ExpectQuery(`SELECT .+ FROM media_variants WHERE`).
		WithArgs(mediaID).
		WillReturnRows(rows)

	variants, err := repo.GetVariants(context.Background(), mediaID)
	if err != nil {
		t.Fatal(err)
	}
	if len(variants) != 2 {
		t.Errorf("expected 2 variants, got %d", len(variants))
	}
	if variants[0].Variant != VariantThumbnail {
		t.Errorf("expected thumbnail variant, got %s", variants[0].Variant)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepository_List(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	siteID := uuid.New()
	now := time.Now()

	rows := mock.NewRows([]string{
		"id", "site_id", "folder_id", "filename", "original_name", "mime_type", "extension",
		"size", "width", "height", "duration", "hash", "alt_text", "caption",
		"storage_provider", "storage_key", "metadata", "created_by", "created_at", "updated_at", "deleted_at",
	}	).AddRow(
		uuid.New(), siteID, nil, "test.jpg", "original.jpg", "image/jpeg", "jpg",
		int64(1024), nil, nil, 0, "abc123", "", "", "local", "path/file.jpg",
		[]byte("{}"), uuid.New(), now, now, nil,
	)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media m WHERE`).
		WithArgs(siteID).
		WillReturnRows(mock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT .+ FROM media m WHERE`).
		WithArgs(siteID, 20, 0).
		WillReturnRows(rows)

	resp, err := repo.List(context.Background(), MediaListRequest{
		SiteID:  siteID,
		Page:    1,
		PerPage: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 1 {
		t.Errorf("Total = %d, want 1", resp.Total)
	}
	if len(resp.Media) != 1 {
		t.Errorf("Media count = %d, want 1", len(resp.Media))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepository_Create(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	now := time.Now()
	mediaID := uuid.New()
	siteID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec(`INSERT INTO media`).
		WithArgs(mediaID, siteID, nil, "test.jpg", "orig.jpg", "image/jpeg", "jpg",
			int64(500), (*int)(nil), (*int)(nil), 0, "hash123", "alt", "caption",
			"local", "path/file.jpg", "{}", userID, now, now).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	m := &Media{
		ID:              mediaID,
		SiteID:          siteID,
		Filename:        "test.jpg",
		OriginalName:    "orig.jpg",
		MimeType:        "image/jpeg",
		Extension:       "jpg",
		Size:            500,
		Hash:            "hash123",
		AltText:         "alt",
		Caption:         "caption",
		StorageProvider: "local",
		StorageKey:      "path/file.jpg",
		CreatedBy:       userID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Metadata:        make(map[string]interface{}),
	}

	err := repo.Create(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepository_SoftDelete(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	mediaID := uuid.New()
	siteID := uuid.New()

	mock.ExpectExec(`UPDATE media SET deleted_at`).
		WithArgs(mediaID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.SoftDelete(context.Background(), siteID, mediaID)
	if err != nil {
		t.Fatal(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepository_Restore(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	mediaID := uuid.New()
	siteID := uuid.New()

	mock.ExpectExec(`UPDATE media SET deleted_at`).
		WithArgs(mediaID, siteID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.Restore(context.Background(), siteID, mediaID)
	if err != nil {
		t.Fatal(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepository_FindByHash(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	siteID := uuid.New()
	now := time.Now()

	rows := mock.NewRows([]string{
		"id", "site_id", "folder_id", "filename", "original_name", "mime_type", "extension",
		"size", "width", "height", "duration", "hash", "alt_text", "caption",
		"storage_provider", "storage_key", "metadata", "created_by", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		uuid.New(), siteID, nil, "test.jpg", "original.jpg", "image/jpeg", "jpg",
		int64(1024), nil, nil, 0, "hash123", "", "", "local", "path/file.jpg",
		`{}`, uuid.New(), now, now, nil,
	)

	mock.ExpectQuery(`SELECT .+ FROM media WHERE`).
		WithArgs(siteID, "hash123").
		WillReturnRows(rows)

	m, err := repo.FindByHash(context.Background(), siteID, "hash123")
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("expected non-nil media")
	}
	if m.Hash != "hash123" {
		t.Errorf("Hash = %q", m.Hash)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepository_FindByHash_NotFound(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM media WHERE`).
		WithArgs(siteID, "nonexistent").
		WillReturnError(pgx.ErrNoRows)

	m, err := repo.FindByHash(context.Background(), siteID, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		t.Error("expected nil for not found")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepository_Move(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	siteID := uuid.New()
	mediaIDs := []uuid.UUID{uuid.New(), uuid.New()}
	folderID := uuid.New()

	mock.ExpectExec(`UPDATE media SET folder_id`).
		WithArgs(siteID, mediaIDs[0], mediaIDs[1], folderID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))

	err := repo.Move(context.Background(), siteID, mediaIDs, &folderID)
	if err != nil {
		t.Fatal(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepository_GetTotalSize(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	siteID := uuid.New()

	mock.ExpectQuery(`SELECT COALESCE`).
		WithArgs(siteID).
		WillReturnRows(mock.NewRows([]string{"coalesce"}).AddRow(int64(5000)))

	total, err := repo.GetTotalSize(context.Background(), siteID)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5000 {
		t.Errorf("total = %d, want 5000", total)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepository_CreateFolder(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	now := time.Now()
	folderID := uuid.New()
	siteID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec(`INSERT INTO folders`).
		WithArgs(folderID, siteID, nil, "Test Folder", "test-folder", "", 0, userID, now, now).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	f := &Folder{
		ID:        folderID,
		SiteID:    siteID,
		Name:      "Test Folder",
		Slug:      "test-folder",
		CreatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := repo.CreateFolder(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepository_ListFolders(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	siteID := uuid.New()
	now := time.Now()

	rows := mock.NewRows([]string{
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

	folders, err := repo.ListFolders(context.Background(), siteID)
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

func TestRepository_GetFolderChildCount(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	folderID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM media WHERE`).
		WithArgs(folderID).
		WillReturnRows(mock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.GetFolderChildCount(context.Background(), folderID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepository_CheckDB_Nil(t *testing.T) {
	repo := &Repository{db: nil}
	err := repo.checkDB()
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestRepository_CheckDB_OK(t *testing.T) {
	repo, mock := setupMockRepo(t)
	defer mock.Close()

	err := repo.checkDB()
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
