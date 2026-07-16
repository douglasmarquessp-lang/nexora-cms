package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLocalDriver(t *testing.T) {
	d := NewLocalDriver("/tmp/test-storage", "/uploads")
	if d.BasePath != "/tmp/test-storage" {
		t.Errorf("BasePath = %q, want %q", d.BasePath, "/tmp/test-storage")
	}
	if d.BaseURL != "/uploads" {
		t.Errorf("BaseURL = %q, want %q", d.BaseURL, "/uploads")
	}
}

func TestLocalDriver_Upload(t *testing.T) {
	dir := t.TempDir()
	d := NewLocalDriver(dir, "/uploads")

	reader := strings.NewReader("test content")
	err := d.Upload(context.Background(), "test/file.txt", reader)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	fullPath := filepath.Join(dir, "test", "file.txt")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Error("uploaded file does not exist")
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read uploaded file: %v", err)
	}
	if string(data) != "test content" {
		t.Errorf("file content = %q, want %q", string(data), "test content")
	}
}

func TestLocalDriver_Download(t *testing.T) {
	dir := t.TempDir()
	d := NewLocalDriver(dir, "/uploads")

	// Create a test file
	testDir := filepath.Join(dir, "test")
	os.MkdirAll(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("test content"), 0644)

	reader, err := d.Download(context.Background(), "test/file.txt")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	defer reader.Close()

	data := make([]byte, 100)
	n, err := reader.Read(data)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}
	if string(data[:n]) != "test content" {
		t.Errorf("content = %q, want %q", string(data[:n]), "test content")
	}
}

func TestLocalDriver_Download_NotFound(t *testing.T) {
	dir := t.TempDir()
	d := NewLocalDriver(dir, "/uploads")

	_, err := d.Download(context.Background(), "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found'", err.Error())
	}
}

func TestLocalDriver_Delete(t *testing.T) {
	dir := t.TempDir()
	d := NewLocalDriver(dir, "/uploads")

	// Create a test file
	testDir := filepath.Join(dir, "test")
	os.MkdirAll(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("test"), 0644)

	err := d.Delete(context.Background(), "test/file.txt")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(testDir, "file.txt")); !os.IsNotExist(err) {
		t.Error("file should not exist after delete")
	}
}

func TestLocalDriver_Delete_NotExist(t *testing.T) {
	dir := t.TempDir()
	d := NewLocalDriver(dir, "/uploads")

	err := d.Delete(context.Background(), "nonexistent.txt")
	if err != nil {
		t.Errorf("deleting nonexistent file should not error: %v", err)
	}
}

func TestLocalDriver_Exists(t *testing.T) {
	dir := t.TempDir()
	d := NewLocalDriver(dir, "/uploads")

	exists, err := d.Exists(context.Background(), "nonexistent.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("file should not exist")
	}

	// Create and check
	testDir := filepath.Join(dir, "test")
	os.MkdirAll(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("test"), 0644)

	exists, err = d.Exists(context.Background(), "test/file.txt")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("file should exist")
	}
}

func TestLocalDriver_URL(t *testing.T) {
	d := NewLocalDriver("/tmp", "/uploads")
	url := d.URL("test/file.txt")
	if url != "/uploads/test/file.txt" {
		t.Errorf("URL = %q, want %q", url, "/uploads/test/file.txt")
	}
}

func TestDriverInterface(t *testing.T) {
	var d Driver = NewLocalDriver("/tmp", "/uploads")
	if d == nil {
		t.Error("LocalDriver should implement Driver interface")
	}
}
