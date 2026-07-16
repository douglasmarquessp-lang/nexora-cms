package assets

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
	"nexora/internal/pkg/storage"
)

func TestNewService(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.storage != st {
		t.Error("service storage pointer mismatch")
	}
}

func TestService_isAllowedMimeType(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)

	tests := []struct {
		mime    string
		ext     string
		allowed bool
	}{
		{"image/jpeg", "jpg", true},
		{"image/png", "png", true},
		{"image/gif", "gif", true},
		{"image/webp", "webp", true},
		{"application/pdf", "pdf", true},
		{"video/mp4", "mp4", true},
		{"text/plain", "txt", true},
		{"text/csv", "csv", true},
		{"application/x-shockwave-flash", "swf", false},
		{"text/html", "html", false},
		{"image/jpeg", "png", false},
		{"", "jpg", false},
	}

	for _, tt := range tests {
		t.Run(tt.mime+"/"+tt.ext, func(t *testing.T) {
			got := svc.isAllowedMimeType(tt.mime, tt.ext)
			if got != tt.allowed {
				t.Errorf("isAllowedMimeType(%q, %q) = %v, want %v", tt.mime, tt.ext, got, tt.allowed)
			}
		})
	}
}

func TestService_classifyMIMEType(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)

	tests := []struct {
		mime  string
		want  string
	}{
		{"image/jpeg", "image"},
		{"image/png", "image"},
		{"video/mp4", "video"},
		{"audio/mpeg", "audio"},
		{"application/pdf", "document"},
		{"text/plain", "document"},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			got := svc.classifyMIMEType(tt.mime)
			if got != tt.want {
				t.Errorf("classifyMIMEType(%q) = %q, want %q", tt.mime, got, tt.want)
			}
		})
	}
}

func TestService_getSizeLimit(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)

	if got := svc.getSizeLimit("image"); got != 10*1024*1024 {
		t.Errorf("image limit = %d, want %d", got, 10*1024*1024)
	}
	if got := svc.getSizeLimit("video"); got != 100*1024*1024 {
		t.Errorf("video limit = %d, want %d", got, 100*1024*1024)
	}
	if got := svc.getSizeLimit("audio"); got != 50*1024*1024 {
		t.Errorf("audio limit = %d, want %d", got, 50*1024*1024)
	}
	if got := svc.getSizeLimit("document"); got != 10*1024*1024 {
		t.Errorf("document limit = %d, want %d", got, 10*1024*1024)
	}
}

func TestDefaultFileValidationConfig(t *testing.T) {
	cfg := DefaultFileValidationConfig()

	if cfg.MaxImageSize != 10*1024*1024 {
		t.Errorf("MaxImageSize = %d, want %d", cfg.MaxImageSize, 10*1024*1024)
	}
	if cfg.MaxVideoSize != 100*1024*1024 {
		t.Errorf("MaxVideoSize = %d, want %d", cfg.MaxVideoSize, 100*1024*1024)
	}

	imageMimes := cfg.AllowedMIMETypes["image"]
	if len(imageMimes) == 0 {
		t.Error("expected image MIME types")
	}
}

func TestService_List_Validation(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)

	// With nil db, should return error
	_, err := svc.List(context.Background(), AssetListRequest{
		SiteID: uuid.New(),
		Page:   0,
	})
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestService_Update_NotFound(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)

	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), UpdateAssetRequest{})
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestService_LinkToPost_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)

	_, err := svc.LinkToPost(context.Background(), uuid.New(), LinkAssetRequest{
		PostID:  uuid.New(),
		AssetID: uuid.New(),
		Type:    PostAssetGallery,
	})
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestService_UnlinkFromPost_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)

	err := svc.UnlinkFromPost(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestService_GetPostAssets_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)

	_, err := svc.GetPostAssets(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestService_Upload_NoDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)

	_, err := svc.Upload(context.Background(), uuid.New(), uuid.New(), nil, nil, UploadRequest{})
	if err == nil {
		t.Error("expected error with nil db")
	}
}

func TestModuleName(t *testing.T) {
	if ModuleName != "assets" {
		t.Errorf("ModuleName = %q, want %q", ModuleName, "assets")
	}
}

func TestDetectMimeTypeFromExtension(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{"jpg", "image/jpeg"},
		{"jpeg", "image/jpeg"},
		{"png", "image/png"},
		{"gif", "image/gif"},
		{"pdf", "application/pdf"},
		{"mp4", "video/mp4"},
		{"unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := DetectMimeTypeFromExtension(tt.ext)
			if got != tt.want {
				t.Errorf("DetectMimeTypeFromExtension(%q) = %q, want %q", tt.ext, got, tt.want)
			}
		})
	}
}
