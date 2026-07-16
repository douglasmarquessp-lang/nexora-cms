package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_LoadAll(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "p1", "1.0.0")
	createTestPlugin(t, dir, "p2", "2.0.0")

	r := NewRegistry()
	l := NewLoader(dir, r)
	plugins, err := l.LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 2 {
		t.Errorf("expected 2, got %d", len(plugins))
	}
}

func TestLoader_LoadAll_NoDir(t *testing.T) {
	r := NewRegistry()
	l := NewLoader("/nonexistent/dir", r)
	plugins, err := l.LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 0 {
		t.Errorf("expected 0, got %d", len(plugins))
	}
}

func TestLoader_Load(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "test-p", "1.0.0")

	r := NewRegistry()
	l := NewLoader(dir, r)
	p, err := l.Load("test-p")
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("expected plugin")
	}
	if p.Manifest.ID != "test-p" {
		t.Errorf("ID = %q", p.Manifest.ID)
	}
}

func TestLoader_Load_NotFound(t *testing.T) {
	r := NewRegistry()
	l := NewLoader(t.TempDir(), r)
	p, err := l.Load("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if p != nil {
		t.Error("expected nil")
	}
}

func TestLoader_Load_SkipDirs(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".hidden"), 0755)
	createTestPlugin(t, dir, "visible", "1.0.0")

	r := NewRegistry()
	l := NewLoader(dir, r)
	plugins, err := l.LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 1 {
		t.Errorf("expected 1, got %d", len(plugins))
	}
}

func TestLoader_Discover(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "found1", "1.0.0")
	createTestPlugin(t, dir, "found2", "2.0.0")
	os.MkdirAll(filepath.Join(dir, "nomanifest"), 0755)

	r := NewRegistry()
	l := NewLoader(dir, r)
	dirs, err := l.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 2 {
		t.Errorf("expected 2, got %d", len(dirs))
	}
}

func TestLoader_Discover_NoDir(t *testing.T) {
	r := NewRegistry()
	l := NewLoader("/nonexistent", r)
	dirs, err := l.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 0 {
		t.Errorf("expected 0, got %d", len(dirs))
	}
}

func TestLoader_Load_BadManifest(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "bad")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte("invalid json"), 0644)

	r := NewRegistry()
	l := NewLoader(dir, r)
	_, err := l.Load("bad")
	if err == nil {
		t.Fatal("expected error for bad manifest")
	}
}

func TestLoader_LoadAll_SkipBad(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "good", "1.0.0")
	badDir := filepath.Join(dir, "bad")
	os.MkdirAll(badDir, 0755)
	os.WriteFile(filepath.Join(badDir, "plugin.json"), []byte("not json"), 0644)

	r := NewRegistry()
	l := NewLoader(dir, r)
	_, err := l.LoadAll()
	if err == nil {
		t.Error("expected error for bad manifest in LoadAll")
	}
}
