package media

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
	"nexora/internal/pkg/storage"
)

func setupService(t *testing.T) *Service {
	t.Helper()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			Driver:      "local",
			LocalPath:   t.TempDir(),
			MaxFileSize: 50 * 1024 * 1024,
		},
	}
	log := logger.New(cfg)
	st := storage.NewLocalDriver(t.TempDir(), "/uploads")
	return NewService(cfg, log, nil, nil, st)
}

func TestNewService(t *testing.T) {
	svc := setupService(t)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestGetByID_NoDB(t *testing.T) {
	svc := setupService(t)
	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestList_NoDB(t *testing.T) {
	svc := setupService(t)
	_, err := svc.List(context.Background(), MediaListRequest{
		SiteID: uuid.New(),
		Page:   1,
	})
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestUpdate_NoDB(t *testing.T) {
	svc := setupService(t)
	altText := "new alt text"
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), UpdateMediaRequest{
		AltText: &altText,
	})
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestDelete_NoDB(t *testing.T) {
	svc := setupService(t)
	err := svc.Delete(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestRestore_NoDB(t *testing.T) {
	svc := setupService(t)
	err := svc.Restore(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestMove_NoDB(t *testing.T) {
	svc := setupService(t)
	err := svc.Move(context.Background(), uuid.New(), []uuid.UUID{uuid.New()}, nil)
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestSearch_NoDB(t *testing.T) {
	svc := setupService(t)
	_, err := svc.Search(context.Background(), uuid.New(), "test", 1, 20)
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestCreateFolder_NoDB(t *testing.T) {
	svc := setupService(t)
	_, err := svc.CreateFolder(context.Background(), uuid.New(), uuid.New(), CreateFolderRequest{
		Name: "Test Folder",
	})
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestGetFolderByID_NoDB(t *testing.T) {
	svc := setupService(t)
	_, err := svc.GetFolderByID(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestListFolders_NoDB(t *testing.T) {
	svc := setupService(t)
	_, err := svc.ListFolders(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestUpdateFolder_NoDB(t *testing.T) {
	svc := setupService(t)
	name := "New Name"
	_, err := svc.UpdateFolder(context.Background(), uuid.New(), uuid.New(), UpdateFolderRequest{
		Name: &name,
	})
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestDeleteFolder_NoDB(t *testing.T) {
	svc := setupService(t)
	err := svc.DeleteFolder(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestCopy_NoDB(t *testing.T) {
	svc := setupService(t)
	_, err := svc.Copy(context.Background(), uuid.New(), uuid.New(), []uuid.UUID{uuid.New()}, nil)
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestMimeTypeFromExt(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{"jpg", "image/jpeg"},
		{"jpeg", "image/jpeg"},
		{"png", "image/png"},
		{"gif", "image/gif"},
		{"webp", "image/webp"},
		{"unknown", "image/jpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := mimeTypeFromExt(tt.ext)
			if got != tt.want {
				t.Errorf("mimeTypeFromExt(%q) = %q, want %q", tt.ext, got, tt.want)
			}
		})
	}
}

func TestExtFromMimeType(t *testing.T) {
	tests := []struct {
		mime string
		want string
	}{
		{"image/jpeg", "jpg"},
		{"image/png", "png"},
		{"image/gif", "gif"},
		{"image/webp", "webp"},
		{"unknown", "jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			got := extFromMimeType(tt.mime)
			if got != tt.want {
				t.Errorf("extFromMimeType(%q) = %q, want %q", tt.mime, got, tt.want)
			}
		})
	}
}

func TestSetEventBus(t *testing.T) {
	svc := setupService(t)
	if svc.eventBus != nil {
		t.Error("expected nil event bus initially")
	}

	svc.SetEventBus(nil)
	if svc.eventBus != nil {
		t.Error("SetEventBus(nil) should set nil")
	}
}

func TestComputeHashSHA256(t *testing.T) {
	data := []byte("test data for hash")
	expected := fmt.Sprintf("%x", sha256.Sum256(data))

	mockFile := &mockMultipartFile{data: data}
	hash, err := ComputeHash(mockFile)
	if err != nil {
		t.Fatal(err)
	}
	if hash != expected {
		t.Errorf("hash = %q, want %q", hash, expected)
	}
}

func TestValidateFolderNameEdgeCases(t *testing.T) {
	cfg := DefaultFileValidationConfig()
	v := NewValidator(cfg)

	if err := v.ValidateFolderName("a"); err != nil {
		t.Errorf("single char should be valid: %v", err)
	}

	name := "valid-folder_name.with.dots and spaces 123"
	if err := v.ValidateFolderName(name); err != nil {
		t.Errorf("valid name %q should pass: %v", name, err)
	}
}

func TestCreateFolderRequest_Validation(t *testing.T) {
	svc := setupService(t)
	_, err := svc.CreateFolder(context.Background(), uuid.New(), uuid.New(), CreateFolderRequest{
		Name: "",
	})
	if err == nil {
		t.Error("expected error for empty folder name")
	}

	_, err = svc.CreateFolder(context.Background(), uuid.New(), uuid.New(), CreateFolderRequest{
		Name: "<script>alert(1)</script>",
	})
	if err == nil {
		t.Error("expected error for invalid folder name with HTML")
	}
}

func TestListRequestDefaults(t *testing.T) {
	svc := setupService(t)
	_, err := svc.List(context.Background(), MediaListRequest{
		SiteID: uuid.New(),
	})
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestUploadValidation(t *testing.T) {
	svc := setupService(t)
	mockFile := &mockMultipartFile{data: []byte("not a real image file")}

	_, err := svc.Upload(context.Background(), uuid.New(), uuid.New(), mockFile, nil, UploadRequest{})
	if err == nil {
		t.Error("expected error for nil header")
	}
}

func TestMediaListResponse(t *testing.T) {
	resp := &MediaListResponse{
		Media:   []Media{},
		Total:   0,
		Page:    1,
		PerPage: 20,
	}
	if resp.Media == nil {
		t.Error("expected non-nil Media slice")
	}
	if resp.Page != 1 {
		t.Errorf("Page = %d, want 1", resp.Page)
	}
	if resp.PerPage != 20 {
		t.Errorf("PerPage = %d, want 20", resp.PerPage)
	}
}
