package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.json")
	content := `{
		"id": "test-p",
		"name": "Test Plugin",
		"version": "1.0.0",
		"author": "Test Author",
		"description": "A test plugin",
		"license": "MIT",
		"homepage": "https://example.com",
		"min_core_version": "0.1.0",
		"dependencies": [],
		"permissions": [],
		"hooks": [],
		"routes": [],
		"admin_pages": []
	}`
	os.WriteFile(path, []byte(content), 0o644)

	m, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "test-p" {
		t.Errorf("ID = %q", m.ID)
	}
	if m.Version != "1.0.0" {
		t.Errorf("Version = %q", m.Version)
	}
	if m.Name != "Test Plugin" {
		t.Errorf("Name = %q", m.Name)
	}
}

func TestLoadManifest_NotFound(t *testing.T) {
	_, err := LoadManifest("/nonexistent/plugin.json")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadManifest_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.json")
	os.WriteFile(path, []byte("not json"), 0o644)

	_, err := LoadManifest(path)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManifestValidate_Valid(t *testing.T) {
	m := &PluginManifest{
		ID:      "test-p",
		Name:    "Test",
		Version: "1.0.0",
	}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestManifestValidate_NoID(t *testing.T) {
	m := &PluginManifest{
		Name:    "Test",
		Version: "1.0.0",
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestManifestValidate_NoName(t *testing.T) {
	m := &PluginManifest{
		ID:      "test-p",
		Version: "1.0.0",
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestManifestValidate_NoVersion(t *testing.T) {
	m := &PluginManifest{
		ID:   "test-p",
		Name: "Test",
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestManifestValidate_InvalidSemver(t *testing.T) {
	m := &PluginManifest{
		ID:      "test-p",
		Name:    "Test",
		Version: "not-semver",
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for invalid semver")
	}
}

func TestManifestValidate_MinCoreVersionTooHigh(t *testing.T) {
	m := &PluginManifest{
		ID:             "test-p",
		Name:           "Test",
		Version:        "1.0.0",
		MinCoreVersion: "99.0.0",
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for min_core_version too high")
	}
}

func TestManifestValidate_InvalidDepVersion(t *testing.T) {
	m := &PluginManifest{
		ID:      "test-p",
		Name:    "Test",
		Version: "1.0.0",
		Dependencies: []PluginDep{
			{ID: "dep1", Version: "bad-version"},
		},
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for invalid dependency version")
	}
}

func TestManifestValidate_DepEmptyID(t *testing.T) {
	m := &PluginManifest{
		ID:      "test-p",
		Name:    "Test",
		Version: "1.0.0",
		Dependencies: []PluginDep{
			{ID: "", Version: "1.0.0"},
		},
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for empty dependency id")
	}
}

func TestManifestValidate_Routes(t *testing.T) {
	m := &PluginManifest{
		ID:      "test-p",
		Name:    "Test",
		Version: "1.0.0",
		Routes: []PluginRoute{
			{Path: "/api/test", Method: "GET"},
		},
	}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestManifestValidate_RouteNoPath(t *testing.T) {
	m := &PluginManifest{
		ID:      "test-p",
		Name:    "Test",
		Version: "1.0.0",
		Routes: []PluginRoute{
			{Method: "GET"},
		},
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for route with no path")
	}
}

func TestManifestValidate_RouteNoMethod(t *testing.T) {
	m := &PluginManifest{
		ID:      "test-p",
		Name:    "Test",
		Version: "1.0.0",
		Routes: []PluginRoute{
			{Path: "/api/test"},
		},
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for route with no method")
	}
}
