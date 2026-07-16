package media

import (
	"testing"
)

func TestModuleName(t *testing.T) {
	if ModuleName != "media" {
		t.Errorf("ModuleName = %q, want %q", ModuleName, "media")
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
		{"webp", "image/webp"},
		{"avif", "image/avif"},
		{"pdf", "application/pdf"},
		{"mp4", "video/mp4"},
		{"mp3", "audio/mpeg"},
		{"unknown", "application/octet-stream"},
		{"", "application/octet-stream"},
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

func TestClassifyMIMEType(t *testing.T) {
	tests := []struct {
		mime string
		want MediaType
	}{
		{"image/jpeg", MediaTypeImage},
		{"image/png", MediaTypeImage},
		{"video/mp4", MediaTypeVideo},
		{"audio/mpeg", MediaTypeAudio},
		{"audio/wav", MediaTypeAudio},
		{"application/pdf", MediaTypeDocument},
		{"text/plain", MediaTypeDocument},
		{"unknown/type", MediaTypeDocument},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			got := ClassifyMIMEType(tt.mime)
			if got != tt.want {
				t.Errorf("ClassifyMIMEType(%q) = %q, want %q", tt.mime, got, tt.want)
			}
		})
	}
}

func TestDefaultFileValidationConfig(t *testing.T) {
	cfg := DefaultFileValidationConfig()

	if cfg.MaxFileSize <= 0 {
		t.Errorf("MaxFileSize should be > 0, got %d", cfg.MaxFileSize)
	}
	if cfg.MaxStoragePerSite <= 0 {
		t.Errorf("MaxStoragePerSite should be > 0, got %d", cfg.MaxStoragePerSite)
	}

	imageMimes := cfg.AllowedMIMETypes["image"]
	if len(imageMimes) == 0 {
		t.Error("expected image MIME types")
	}

	videoMimes := cfg.AllowedMIMETypes["video"]
	if len(videoMimes) == 0 {
		t.Error("expected video MIME types")
	}
}

func TestEventTypes(t *testing.T) {
	if string(EventMediaUploaded) != "media.uploaded" {
		t.Errorf("unexpected event type: %s", EventMediaUploaded)
	}
	if string(EventMediaUpdated) != "media.updated" {
		t.Errorf("unexpected event type: %s", EventMediaUpdated)
	}
	if string(EventMediaDeleted) != "media.deleted" {
		t.Errorf("unexpected event type: %s", EventMediaDeleted)
	}
	if string(EventMediaRestored) != "media.restored" {
		t.Errorf("unexpected event type: %s", EventMediaRestored)
	}
	if string(EventFolderCreated) != "folder.created" {
		t.Errorf("unexpected event type: %s", EventFolderCreated)
	}
	if string(EventFolderDeleted) != "folder.deleted" {
		t.Errorf("unexpected event type: %s", EventFolderDeleted)
	}
}

func TestAllVariants(t *testing.T) {
	if len(AllVariants) != 5 {
		t.Errorf("expected 5 variants, got %d", len(AllVariants))
	}

	expected := []VariantType{VariantThumbnail, VariantSmall, VariantMedium, VariantLarge, VariantOriginal}
	for i, v := range expected {
		if AllVariants[i] != v {
			t.Errorf("AllVariants[%d] = %s, want %s", i, AllVariants[i], v)
		}
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"My Folder", "my-folder"},
		{"Hello World!", "hello-world"},
		{"  Spaces  ", "spaces"},
		{"Special@#$Characters", "specialcharacters"},
		{"Test Folder Name", "test-folder-name"},
		{"", "folder"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestErrorTypes(t *testing.T) {
	if ErrMediaNotFound == nil {
		t.Error("ErrMediaNotFound should not be nil")
	}
	if ErrMediaNotInSite == nil {
		t.Error("ErrMediaNotInSite should not be nil")
	}
	if ErrInvalidFileType == nil {
		t.Error("ErrInvalidFileType should not be nil")
	}
	if ErrFileTooLarge == nil {
		t.Error("ErrFileTooLarge should not be nil")
	}
	if ErrDuplicateFile == nil {
		t.Error("ErrDuplicateFile should not be nil")
	}
	if ErrStorageLimitReached == nil {
		t.Error("ErrStorageLimitReached should not be nil")
	}
	if ErrFolderNotFound == nil {
		t.Error("ErrFolderNotFound should not be nil")
	}
	if ErrFolderNotEmpty == nil {
		t.Error("ErrFolderNotEmpty should not be nil")
	}
}

func TestMediaTypeValues(t *testing.T) {
	if string(MediaTypeImage) != "image" {
		t.Errorf("MediaTypeImage = %q", MediaTypeImage)
	}
	if string(MediaTypeVideo) != "video" {
		t.Errorf("MediaTypeVideo = %q", MediaTypeVideo)
	}
	if string(MediaTypeDocument) != "document" {
		t.Errorf("MediaTypeDocument = %q", MediaTypeDocument)
	}
	if string(MediaTypeAudio) != "audio" {
		t.Errorf("MediaTypeAudio = %q", MediaTypeAudio)
	}
}

func TestVariantTypeValues(t *testing.T) {
	if string(VariantThumbnail) != "thumbnail" {
		t.Errorf("VariantThumbnail = %q", VariantThumbnail)
	}
	if string(VariantSmall) != "small" {
		t.Errorf("VariantSmall = %q", VariantSmall)
	}
	if string(VariantMedium) != "medium" {
		t.Errorf("VariantMedium = %q", VariantMedium)
	}
	if string(VariantLarge) != "large" {
		t.Errorf("VariantLarge = %q", VariantLarge)
	}
	if string(VariantOriginal) != "original" {
		t.Errorf("VariantOriginal = %q", VariantOriginal)
	}
}
