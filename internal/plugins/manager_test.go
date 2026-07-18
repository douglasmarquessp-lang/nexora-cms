package plugins

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"nexora/internal/pkg/logger"
)

func TestManagerInit(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "test-plugin", "1.0.0")

	m := NewManager(&ManagerConfig{PluginsDir: dir}, testLogger(t), &mockEmitter{})
	if err := m.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(m.ListPlugins()) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(m.ListPlugins()))
	}
}

func TestManagerInit_NoDir(t *testing.T) {
	m := NewManager(&ManagerConfig{PluginsDir: "/nonexistent"}, testLogger(t), &mockEmitter{})
	if err := m.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestManagerListPlugins(t *testing.T) {
	m := managerWithPlugins(t, "p1", "p2")
	plugins := m.ListPlugins()
	if len(plugins) != 2 {
		t.Errorf("expected 2, got %d", len(plugins))
	}
}

func TestManagerGetPlugin(t *testing.T) {
	m := managerWithPlugins(t, "my-plugin")
	p := m.GetPlugin("my-plugin")
	if p == nil {
		t.Fatal("expected plugin")
	}
	if p.Manifest.Name != "My Plugin" {
		t.Errorf("Name = %q", p.Manifest.Name)
	}
	notFound := m.GetPlugin("nonexistent")
	if notFound != nil {
		t.Error("expected nil")
	}
}

func TestManagerInstall(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "new-plugin", "1.0.0")
	m := NewManager(&ManagerConfig{PluginsDir: dir}, testLogger(t), &mockEmitter{})

	plugin, err := m.Install(context.Background(), "new-plugin")
	if err != nil {
		t.Fatal(err)
	}
	if plugin == nil {
		t.Fatal("expected plugin")
	}
	if plugin.Manifest.ID != "new-plugin" {
		t.Errorf("ID = %q", plugin.Manifest.ID)
	}
}

func TestManagerInstall_MissingManifest(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "noplugin"), 0o755)
	m := NewManager(&ManagerConfig{PluginsDir: dir}, testLogger(t), &mockEmitter{})

	_, err := m.Install(context.Background(), "noplugin")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManagerActivate(t *testing.T) {
	m := managerWithPlugins(t, "test-p")
	if err := m.Activate(context.Background(), "test-p"); err != nil {
		t.Fatal(err)
	}
	p := m.GetPlugin("test-p")
	if p.Status != PluginStatusActive {
		t.Errorf("status = %v", p.Status)
	}
}

func TestManagerActivate_NotFound(t *testing.T) {
	m := managerWithPlugins(t, "test-p")
	err := m.Activate(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManagerDeactivate(t *testing.T) {
	m := managerWithPlugins(t, "test-p")
	m.Activate(context.Background(), "test-p")
	if err := m.Deactivate(context.Background(), "test-p"); err != nil {
		t.Fatal(err)
	}
	p := m.GetPlugin("test-p")
	if p.Status != PluginStatusInactive {
		t.Errorf("status = %v", p.Status)
	}
}

func TestManagerDeactivate_NotFound(t *testing.T) {
	m := managerWithPlugins(t, "test-p")
	err := m.Deactivate(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManagerUninstall(t *testing.T) {
	m := managerWithPlugins(t, "test-p")
	if err := m.Uninstall(context.Background(), "test-p"); err != nil {
		t.Fatal(err)
	}
	if m.GetPlugin("test-p") != nil {
		t.Error("expected nil after uninstall")
	}
}

func TestManagerUninstall_NotFound(t *testing.T) {
	m := NewManager(&ManagerConfig{PluginsDir: t.TempDir()}, testLogger(t), &mockEmitter{})
	err := m.Uninstall(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManagerUpdate(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "updatable", "1.0.0")
	m := NewManager(&ManagerConfig{PluginsDir: dir}, testLogger(t), &mockEmitter{})

	m.Init(context.Background())
	p := m.GetPlugin("updatable")
	if p.Manifest.Version != "1.0.0" {
		t.Errorf("version = %q", p.Manifest.Version)
	}

	createTestPlugin(t, dir, "updatable", "2.0.0")
	if err := m.Update(context.Background(), "updatable"); err != nil {
		t.Fatal(err)
	}
	if p.Manifest.Version != "2.0.0" {
		t.Errorf("updated version = %q", p.Manifest.Version)
	}
}

func TestManagerUpdate_NotFound(t *testing.T) {
	m := NewManager(&ManagerConfig{PluginsDir: t.TempDir()}, testLogger(t), &mockEmitter{})
	err := m.Update(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManagerListActive(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "a1", "1.0.0")
	createTestPlugin(t, dir, "a2", "1.0.0")
	m := NewManager(&ManagerConfig{PluginsDir: dir}, testLogger(t), &mockEmitter{})
	m.Init(context.Background())
	m.Activate(context.Background(), "a1")

	active := m.ListActive()
	if len(active) != 1 {
		t.Errorf("expected 1 active, got %d", len(active))
	}
}

func TestManagerPermissions(t *testing.T) {
	m := NewManager(&ManagerConfig{PluginsDir: t.TempDir()}, testLogger(t), &mockEmitter{})
	p := m.Permissions()
	if p == nil {
		t.Fatal("expected permissions")
	}
}

func TestManagerSandbox(t *testing.T) {
	m := NewManager(&ManagerConfig{PluginsDir: t.TempDir()}, testLogger(t), &mockEmitter{})
	s := m.Sandbox()
	if s == nil {
		t.Fatal("expected sandbox")
	}
}

func TestManagerHooks(t *testing.T) {
	m := managerWithPlugins(t, "test-h")
	hooks := m.Hooks()
	if hooks == nil {
		t.Fatal("expected hooks")
	}
}

func TestManagerRegistry(t *testing.T) {
	m := managerWithPlugins(t, "test-r")
	r := m.Registry()
	if r == nil {
		t.Fatal("expected registry")
	}
	if !r.Exists("test-r") {
		t.Error("expected test-r to exist")
	}
}

func TestManagerLifecycle(t *testing.T) {
	m := managerWithPlugins(t, "test-l")
	lc := m.Lifecycle()
	if lc == nil {
		t.Fatal("expected lifecycle")
	}
}

func createTestPlugin(t *testing.T, dir, id, version string) {
	t.Helper()
	pluginDir := filepath.Join(dir, id)
	os.MkdirAll(pluginDir, 0o755)
	manifest := `{
		"id": "` + id + `",
		"name": "My Plugin",
		"version": "` + version + `",
		"author": "Test",
		"description": "Test plugin",
		"license": "MIT",
		"homepage": "",
		"min_core_version": "",
		"dependencies": [],
		"permissions": [],
		"hooks": [],
		"routes": [],
		"admin_pages": []
	}`
	os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o644)
}

func managerWithPlugins(t *testing.T, ids ...string) *Manager {
	t.Helper()
	dir := t.TempDir()
	for _, id := range ids {
		createTestPlugin(t, dir, id, "1.0.0")
	}
	m := NewManager(&ManagerConfig{PluginsDir: dir}, testLogger(t), &mockEmitter{})
	if err := m.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	return m
}

type mockEmitter struct{}

func (m *mockEmitter) Emit(ctx context.Context, eventType string, payload interface{}, siteID string) error {
	return nil
}

func testLogger(t *testing.T) *logger.Logger {
	t.Helper()
	return &logger.Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}
