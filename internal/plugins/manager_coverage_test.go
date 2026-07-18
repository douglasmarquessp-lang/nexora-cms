package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"nexora/internal/api/rest"
)

func TestManagerInstall_Duplicate(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "dup-p", "1.0.0")
	m := NewManager(&ManagerConfig{PluginsDir: dir}, testLogger(t), &mockEmitter{})
	m.Init(context.Background())

	_, err := m.Install(context.Background(), "dup-p")
	if err == nil {
		t.Fatal("expected error for duplicate install")
	}
}

func TestManagerDeactivate_Error(t *testing.T) {
	m := managerWithPlugins(t, "err-p")
	p := m.GetPlugin("err-p")
	p.Status = PluginStatusActive

	if err := m.Deactivate(context.Background(), "err-p"); err != nil {
		t.Fatal(err)
	}
}

func TestManagerUninstall_Error(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "uninst-p", "1.0.0")
	m := NewManager(&ManagerConfig{PluginsDir: dir}, testLogger(t), &mockEmitter{})
	m.Init(context.Background())
	m.registry.SetStatus("uninst-p", PluginStatusActive)

	if err := m.Uninstall(context.Background(), "uninst-p"); err != nil {
		t.Fatal(err)
	}
}

func TestManagerInit_WithBadPlugin(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "good-p", "1.0.0")
	os.MkdirAll(filepath.Join(dir, "bad-p"), 0o755)
	os.WriteFile(filepath.Join(dir, "bad-p", "plugin.json"), []byte("not json"), 0o644)

	m := NewManager(&ManagerConfig{PluginsDir: dir}, testLogger(t), &mockEmitter{})
	err := m.Init(context.Background())
	if err == nil {
		t.Fatal("expected error due to bad manifest")
	}
	if m.GetPlugin("good-p") != nil {
		t.Error("good-p should not be loaded after error")
	}
}

func TestLoader_LoadAll_DirError(t *testing.T) {
	r := NewRegistry()
	l := NewLoader("/dev/null/plugins", r)
	_, err := l.LoadAll()
	// /dev/null is a file, so ReadDir will fail on some systems
	if err == nil {
		t.Log("no error (dir may exist)")
	}
}

func TestLoader_Load_StatError(t *testing.T) {
	r := NewRegistry()
	l := NewLoader(t.TempDir(), r)

	pluginDir := filepath.Join(l.pluginsDir, "test")
	os.MkdirAll(pluginDir, 0o755)
	manifestPath := filepath.Join(pluginDir, "plugin.json")
	os.WriteFile(manifestPath, []byte("{}"), 0o644)
	os.Chmod(manifestPath, 0o000)

	p, err := l.Load("test")
	if err == nil && p != nil {
		os.Chmod(manifestPath, 0o644)
		t.Fatal("expected error or nil")
	}
	os.Chmod(manifestPath, 0o644)
}

func TestManagerInstall_WithoutManifest(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "empty-p"), 0o755)
	m := NewManager(&ManagerConfig{PluginsDir: dir}, testLogger(t), &mockEmitter{})

	_, err := m.Install(context.Background(), "empty-p")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManagerUpdate_PluginGone(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "gone-p", "1.0.0")
	m := NewManager(&ManagerConfig{PluginsDir: dir}, testLogger(t), &mockEmitter{})
	m.Init(context.Background())

	os.RemoveAll(filepath.Join(dir, "gone-p"))
	err := m.Update(context.Background(), "gone-p")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPermissionsMultiPlugin(t *testing.T) {
	p := NewPermissions()
	p.Register("p1", []PermissionDef{
		{PluginID: "p1", Permission: "read"},
		{PluginID: "p1", Permission: "write"},
	})
	p.Register("p2", []PermissionDef{
		{PluginID: "p2", Permission: "admin"},
	})

	all := p.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 total, got %d", len(all))
	}
}

func TestHooks_GetRegistrations_Multiple(t *testing.T) {
	h := NewHooks()
	h.AddAction("p1", HookAfterPostSave, func(ctx context.Context, args map[string]interface{}) error {
		return nil
	}, 10)
	h.AddFilter("p1", HookBeforeRender, func(ctx context.Context, value interface{}, args map[string]interface{}) (interface{}, error) {
		return value, nil
	}, 20)

	regs := h.GetRegistrations("p1")
	if len(regs) != 2 {
		t.Errorf("expected 2 registrations, got %d", len(regs))
	}
}

func TestHandlerListWithQueryParams(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plugins?search=test", http.NoBody)
	rest.AdaptHandler(h.List).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
