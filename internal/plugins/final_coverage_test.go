package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestManagerInstall_LifecycleError(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "dep-p", "1.0.0")

	pluginDir := filepath.Join(dir, "main-p")
	os.MkdirAll(pluginDir, 0755)
	manifest := `{
		"id": "main-p",
		"name": "Main",
		"version": "1.0.0",
		"dependencies": [{"id": "missing-dep", "version": "1.0.0"}],
		"permissions": [],
		"hooks": [],
		"routes": [],
		"admin_pages": []
	}`
	os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0644)

	m := NewManager(&ManagerConfig{PluginsDir: dir}, testLogger(t), &mockEmitter{})
	m.Init(context.Background())

	_, err := m.Install(context.Background(), "main-p")
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestLoader_Load_BadJSON(t *testing.T) {
	dir := t.TempDir()
	pDir := filepath.Join(dir, "badjson")
	os.MkdirAll(pDir, 0755)
	os.WriteFile(filepath.Join(pDir, "plugin.json"), []byte("{invalid}"), 0644)

	r := NewRegistry()
	l := NewLoader(dir, r)
	_, err := l.Load("badjson")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHooks_RemovePluginFiltersOnly(t *testing.T) {
	h := NewHooks()

	h.AddFilter("test-p", HookBeforeRender, func(ctx context.Context, value interface{}, args map[string]interface{}) (interface{}, error) {
		return value, nil
	}, 10)

	h.RemovePlugin("test-p")

	result, err := h.ApplyFilter(context.Background(), HookBeforeRender, "val", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "val" {
		t.Errorf("result = %q", result)
	}
}

func TestRegistry_SetDBID_NotFound(t *testing.T) {
	r := NewRegistry()
	err := r.SetDBID("nonexistent", "uuid")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManagerInit_RegisterError(t *testing.T) {
	dir := t.TempDir()
	createTestPlugin(t, dir, "dup", "1.0.0")
	createTestPlugin(t, dir, "dup", "1.0.0")

	m := NewManager(&ManagerConfig{PluginsDir: dir}, testLogger(t), &mockEmitter{})
	if err := m.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleEventInstallHooks(t *testing.T) {
	r := NewRegistry()
	r.Register(&PluginInstance{
		Manifest: &PluginManifest{ID: "dep", Name: "Dep", Version: "1.0.0"},
		Status:   PluginStatusActive,
	})

	h := NewHooks()
	hookCalled := false
	h.AddAction("hook-p", HookBeforePluginInstall, func(ctx context.Context, args map[string]interface{}) error {
		hookCalled = true
		return nil
	}, 10)

	lc := NewLifecycle(r, h, &mockEmitter{})

	instance := &PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test",
			Version: "1.0.0",
		},
	}

	lc.Install(context.Background(), instance)
	if !hookCalled {
		t.Error("HookBeforePluginInstall not called")
	}
}
