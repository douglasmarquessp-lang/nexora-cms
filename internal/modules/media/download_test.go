package media

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"

	"nexora/internal/api/rest"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func expectMediaGet(mock pgxmock.PgxPoolIface, mediaID, siteID uuid.UUID, storageKey string) {
	now := time.Now()
	rows := pgxmock.NewRows([]string{
		"id", "site_id", "folder_id", "filename", "original_name", "mime_type", "extension",
		"size", "width", "height", "duration", "hash", "alt_text", "caption",
		"storage_provider", "storage_key", "metadata", "created_by", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		mediaID, siteID, nil, "test.jpg", "original.jpg", "image/jpeg", "jpg",
		int64(1024), nil, nil, 0, "abc123", "", "", "local", storageKey,
		[]byte("{}"), uuid.New(), now, now, nil,
	)
	mock.ExpectQuery(`SELECT .+ FROM media WHERE`).
		WithArgs(mediaID, siteID).
		WillReturnRows(rows)
}

func expectVariants(mock pgxmock.PgxPoolIface, mediaID uuid.UUID, rows *pgxmock.Rows) {
	mock.ExpectQuery(`SELECT .+ FROM media_variants WHERE`).
		WithArgs(mediaID).
		WillReturnRows(rows)
}

func emptyVariantRows() *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"id", "media_id", "variant", "width", "height", "file_size", "mime_type",
		"storage_key", "metadata", "created_at",
	})
}

func TestService_OpenFile_Original(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	mediaID := uuid.New()
	siteID := uuid.New()

	expectMediaGet(mock, mediaID, siteID, "path/file.jpg")
	expectVariants(mock, mediaID, emptyVariantRows())

	if err := svc.storage.Upload(context.Background(), "path/file.jpg", strings.NewReader("fake-jpeg-bytes")); err != nil {
		t.Fatal(err)
	}

	rc, mimeType, err := svc.OpenFile(context.Background(), siteID, mediaID, "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake-jpeg-bytes" {
		t.Errorf("body = %q, want %q", string(data), "fake-jpeg-bytes")
	}
	if mimeType != "image/jpeg" {
		t.Errorf("mimeType = %q, want image/jpeg", mimeType)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_OpenFile_Variant(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	mediaID := uuid.New()
	siteID := uuid.New()
	now := time.Now()

	expectMediaGet(mock, mediaID, siteID, "path/file.jpg")

	variantRows := pgxmock.NewRows([]string{
		"id", "media_id", "variant", "width", "height", "file_size", "mime_type",
		"storage_key", "metadata", "created_at",
	}).AddRow(
		uuid.New(), mediaID, "thumbnail", 150, 150, int64(500), "image/webp",
		"path/thumb.webp", []byte("{}"), now,
	)
	expectVariants(mock, mediaID, variantRows)

	if err := svc.storage.Upload(context.Background(), "path/file.jpg", strings.NewReader("original-bytes")); err != nil {
		t.Fatal(err)
	}
	if err := svc.storage.Upload(context.Background(), "path/thumb.webp", strings.NewReader("thumb-bytes")); err != nil {
		t.Fatal(err)
	}

	rc, mimeType, err := svc.OpenFile(context.Background(), siteID, mediaID, "thumbnail")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "thumb-bytes" {
		t.Errorf("body = %q, want %q", string(data), "thumb-bytes")
	}
	if mimeType != "image/webp" {
		t.Errorf("mimeType = %q, want image/webp", mimeType)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_OpenFile_VariantUnknownFallsBackToOriginal(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	mediaID := uuid.New()
	siteID := uuid.New()

	expectMediaGet(mock, mediaID, siteID, "path/file.jpg")
	expectVariants(mock, mediaID, emptyVariantRows())

	if err := svc.storage.Upload(context.Background(), "path/file.jpg", strings.NewReader("original-bytes")); err != nil {
		t.Fatal(err)
	}

	rc, mimeType, err := svc.OpenFile(context.Background(), siteID, mediaID, "huge")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original-bytes" {
		t.Errorf("body = %q, want original-bytes", string(data))
	}
	if mimeType != "image/jpeg" {
		t.Errorf("mimeType = %q, want image/jpeg", mimeType)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_OpenFile_MediaNotFound(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	mediaID := uuid.New()
	siteID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM media WHERE`).
		WithArgs(mediaID, siteID).
		WillReturnError(pgx.ErrNoRows)

	_, _, err := svc.OpenFile(context.Background(), siteID, mediaID, "")
	if err != ErrMediaNotFound {
		t.Errorf("expected ErrMediaNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestService_OpenFile_StorageFileMissing(t *testing.T) {
	svc, mock := setupServiceWithMock(t)
	defer mock.Close()

	mediaID := uuid.New()
	siteID := uuid.New()

	expectMediaGet(mock, mediaID, siteID, "path/missing.jpg")
	expectVariants(mock, mediaID, emptyVariantRows())

	_, _, err := svc.OpenFile(context.Background(), siteID, mediaID, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected file-not-found error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestHandler_Download(t *testing.T) {
	log := logger.New(&config.Config{})

	t.Run("missing site", func(t *testing.T) {
		svc, mock := setupServiceWithMock(t)
		defer mock.Close()

		h := NewHandler(svc, log)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/media/"+uuid.New().String()+"/file", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.Download).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		svc, mock := setupServiceWithMock(t)
		defer mock.Close()

		h := NewHandler(svc, log)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/media/invalid/file", nil)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		req = withSiteID(req)
		rest.AdaptHandler(h.Download).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("streams file bytes", func(t *testing.T) {
		svc, mock := setupServiceWithMock(t)
		defer mock.Close()

		mediaID := uuid.New()
		siteID := uuid.New()

		expectMediaGet(mock, mediaID, siteID, "path/file.jpg")
		expectVariants(mock, mediaID, emptyVariantRows())

		if err := svc.storage.Upload(context.Background(), "path/file.jpg", strings.NewReader("fake-jpeg-bytes")); err != nil {
			t.Fatal(err)
		}

		h := NewHandler(svc, log)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/media/"+mediaID.String()+"/file", nil)
		req = withChiParams(req, map[string]string{"id": mediaID.String()})
		req = withSiteID(req, siteID)
		rest.AdaptHandler(h.Download).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
			t.Errorf("Content-Type = %q, want image/jpeg", ct)
		}
		if rec.Body.String() != "fake-jpeg-bytes" {
			t.Errorf("body = %q, want fake-jpeg-bytes", rec.Body.String())
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
	})
}
