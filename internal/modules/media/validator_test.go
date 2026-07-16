package media

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewValidator(t *testing.T) {
	cfg := DefaultFileValidationConfig()
	v := NewValidator(cfg)
	if v == nil {
		t.Fatal("expected non-nil validator")
	}
}

func TestValidateFolderName(t *testing.T) {
	cfg := DefaultFileValidationConfig()
	v := NewValidator(cfg)

	tests := []struct {
		name    string
		wantErr bool
	}{
		{"My Folder", false},
		{"Hello-World", false},
		{"folder_123", false},
		{"test.name", false},
		{"", true},
		{"   ", true},
		{"a", false},
		{strings.Repeat("a", 256), true},
		{"folder@name", true},
		{"folder#name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateFolderName(tt.name)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for name %q", tt.name)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for name %q: %v", tt.name, err)
			}
		})
	}
}

func TestIsAllowedMimeType(t *testing.T) {
	cfg := DefaultFileValidationConfig()
	v := NewValidator(cfg)

	tests := []struct {
		mime    string
		allowed bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"image/webp", true},
		{"image/avif", true},
		{"image/svg+xml", true},
		{"video/mp4", true},
		{"video/webm", true},
		{"audio/mpeg", true},
		{"audio/wav", true},
		{"application/pdf", true},
		{"application/msword", true},
		{"text/plain", true},
		{"text/html", false},
		{"application/x-shockwave-flash", false},
		{"image/bmp", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			got := v.isAllowedMimeType(tt.mime)
			if got != tt.allowed {
				t.Errorf("isAllowedMimeType(%q) = %v, want %v", tt.mime, got, tt.allowed)
			}
		})
	}
}

func TestIsImageType(t *testing.T) {
	tests := []struct {
		mime string
		want bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"image/webp", true},
		{"image/svg+xml", false},
		{"video/mp4", false},
		{"application/pdf", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			got := IsImageType(tt.mime)
			if got != tt.want {
				t.Errorf("IsImageType(%q) = %v, want %v", tt.mime, got, tt.want)
			}
		})
	}
}

func TestComputeHash(t *testing.T) {
	content := "test content for hashing"
	body := bytes.NewReader([]byte(content))

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=testboundary")

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, _ := w.CreateFormFile("file", "test.txt")
	fw.Write([]byte(content))
	w.Close()

	mockFile := &mockMultipartFile{data: []byte(content)}

	hash, err := ComputeHash(mockFile)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if len(hash) != 64 {
		t.Errorf("expected 64-char hash, got %d", len(hash))
	}
}

type mockMultipartFile struct {
	data   []byte
	offset int
}

func (m *mockMultipartFile) Read(p []byte) (n int, err error) {
	if m.offset >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.offset:])
	m.offset += n
	return n, nil
}

func (m *mockMultipartFile) ReadAt(p []byte, off int64) (n int, err error) {
	if int(off) >= len(m.data) {
		return 0, nil
	}
	n = copy(p, m.data[off:])
	return n, nil
}

func (m *mockMultipartFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		m.offset = int(offset)
	case 1:
		m.offset += int(offset)
	case 2:
		m.offset = len(m.data) + int(offset)
	}
	if m.offset < 0 {
		m.offset = 0
	}
	if m.offset > len(m.data) {
		m.offset = len(m.data)
	}
	return int64(m.offset), nil
}

func (m *mockMultipartFile) Close() error {
	return nil
}

var _ = strings.NewReader
