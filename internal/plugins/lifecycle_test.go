package plugins

import (
	"context"
	"testing"
)

func TestLifecycle_Install(t *testing.T) {
	r := NewRegistry()
	h := NewHooks()
	lc := NewLifecycle(r, h, &mockEmitter{})

	r.Register(&PluginInstance{
		Manifest: &PluginManifest{
			ID:      "dep",
			Name:    "Dependency",
			Version: "1.0.0",
		},
		Status: PluginStatusActive,
	})

	instance := &PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test Plugin",
			Version: "1.0.0",
			Dependencies: []PluginDep{
				{ID: "dep", Version: "1.0.0"},
			},
		},
	}

	if err := lc.Install(context.Background(), instance); err != nil {
		t.Fatal(err)
	}
	if instance.Status != PluginStatusInstalled {
		t.Errorf("status = %v", instance.Status)
	}
}

func TestLifecycle_Install_MissingDep(t *testing.T) {
	r := NewRegistry()
	h := NewHooks()
	lc := NewLifecycle(r, h, &mockEmitter{})

	instance := &PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test Plugin",
			Version: "1.0.0",
			Dependencies: []PluginDep{
				{ID: "missing-dep", Version: "1.0.0"},
			},
		},
	}

	if err := lc.Install(context.Background(), instance); err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestLifecycle_Install_DepNotActive(t *testing.T) {
	r := NewRegistry()
	h := NewHooks()

	r.Register(&PluginInstance{
		Manifest: &PluginManifest{
			ID:      "inactive-dep",
			Name:    "Inactive Dep",
			Version: "1.0.0",
		},
		Status: PluginStatusInactive,
	})

	lc := NewLifecycle(r, h, &mockEmitter{})

	instance := &PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test Plugin",
			Version: "1.0.0",
			Dependencies: []PluginDep{
				{ID: "inactive-dep"},
			},
		},
	}

	if err := lc.Install(context.Background(), instance); err == nil {
		t.Fatal("expected error for inactive dependency")
	}
}

func TestLifecycle_Activate(t *testing.T) {
	r := NewRegistry()
	h := NewHooks()
	lc := NewLifecycle(r, h, &mockEmitter{})

	instance := &PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test",
			Version: "1.0.0",
			Hooks:   []PluginHookDef{{Hook: string(HookAfterPostSave), Priority: 10}},
		},
	}

	if err := lc.Activate(context.Background(), instance); err != nil {
		t.Fatal(err)
	}
	if instance.Status != PluginStatusActive {
		t.Errorf("status = %v", instance.Status)
	}
}

func TestLifecycle_Activate_InvalidHook(t *testing.T) {
	r := NewRegistry()
	h := NewHooks()
	lc := NewLifecycle(r, h, &mockEmitter{})

	instance := &PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test",
			Version: "1.0.0",
			Hooks:   []PluginHookDef{{Hook: "invalid_hook", Priority: 10}},
		},
	}

	if err := lc.Activate(context.Background(), instance); err == nil {
		t.Fatal("expected error for invalid hook")
	}
}

func TestLifecycle_Deactivate(t *testing.T) {
	r := NewRegistry()
	h := NewHooks()
	lc := NewLifecycle(r, h, &mockEmitter{})

	instance := &PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test",
			Version: "1.0.0",
		},
	}

	if err := lc.Deactivate(context.Background(), instance); err != nil {
		t.Fatal(err)
	}
	if instance.Status != PluginStatusInactive {
		t.Errorf("status = %v", instance.Status)
	}
}

func TestLifecycle_Update(t *testing.T) {
	r := NewRegistry()
	h := NewHooks()
	lc := NewLifecycle(r, h, &mockEmitter{})

	instance := &PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test",
			Version: "1.0.0",
		},
	}

	newManifest := &PluginManifest{
		ID:      "test-p",
		Name:    "Test Updated",
		Version: "2.0.0",
	}

	if err := lc.Update(context.Background(), instance, newManifest); err != nil {
		t.Fatal(err)
	}
	if instance.Manifest.Version != "2.0.0" {
		t.Errorf("version = %q", instance.Manifest.Version)
	}
}

func TestLifecycle_Uninstall(t *testing.T) {
	r := NewRegistry()
	h := NewHooks()
	lc := NewLifecycle(r, h, &mockEmitter{})

	r.Register(&PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test",
			Version: "1.0.0",
		},
		Status: PluginStatusActive,
	})

	instance := r.Get("test-p")

	if err := lc.Uninstall(context.Background(), instance); err != nil {
		t.Fatal(err)
	}
	if r.Get("test-p") != nil {
		t.Error("expected plugin to be removed")
	}
}

func TestLifecycle_Install_EmitsEvent(t *testing.T) {
	r := NewRegistry()
	h := NewHooks()
	emitted := false
	emitter := &mockEmitterWithCheck{
		fn: func(eventType string) {
			if eventType == "plugin.installed" {
				emitted = true
			}
		},
	}

	lc := NewLifecycle(r, h, emitter)
	instance := &PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test",
			Version: "1.0.0",
		},
	}

	lc.Install(context.Background(), instance)
	if !emitted {
		t.Error("plugin.installed event not emitted")
	}
}

func TestLifecycle_Activate_EmitsEvent(t *testing.T) {
	r := NewRegistry()
	h := NewHooks()
	emitted := false
	emitter := &mockEmitterWithCheck{
		fn: func(eventType string) {
			if eventType == "plugin.activated" {
				emitted = true
			}
		},
	}

	lc := NewLifecycle(r, h, emitter)
	instance := &PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test",
			Version: "1.0.0",
		},
	}

	lc.Activate(context.Background(), instance)
	if !emitted {
		t.Error("plugin.activated event not emitted")
	}
}

func TestLifecycle_Deactivate_EmitsEvent(t *testing.T) {
	r := NewRegistry()
	h := NewHooks()
	emitted := false
	emitter := &mockEmitterWithCheck{
		fn: func(eventType string) {
			if eventType == "plugin.deactivated" {
				emitted = true
			}
		},
	}

	lc := NewLifecycle(r, h, emitter)
	instance := &PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test",
			Version: "1.0.0",
		},
	}

	lc.Deactivate(context.Background(), instance)
	if !emitted {
		t.Error("plugin.deactivated event not emitted")
	}
}

func TestLifecycle_Update_EmitsEvent(t *testing.T) {
	emitted := false
	emitter := &mockEmitterWithCheck{
		fn: func(eventType string) {
			if eventType == "plugin.updated" {
				emitted = true
			}
		},
	}

	lc := NewLifecycle(NewRegistry(), NewHooks(), emitter)
	instance := &PluginInstance{
		Manifest: &PluginManifest{ID: "test-p", Name: "Test", Version: "1.0.0"},
	}

	lc.Update(context.Background(), instance, &PluginManifest{ID: "test-p", Name: "Test", Version: "2.0.0"})
	if !emitted {
		t.Error("plugin.updated event not emitted")
	}
}

func TestLifecycle_Uninstall_EmitsEvent(t *testing.T) {
	r := NewRegistry()
	r.Register(&PluginInstance{Manifest: &PluginManifest{ID: "test-p", Name: "Test", Version: "1.0.0"}})

	emitted := false
	emitter := &mockEmitterWithCheck{
		fn: func(eventType string) {
			if eventType == "plugin.removed" {
				emitted = true
			}
		},
	}

	lc := NewLifecycle(r, NewHooks(), emitter)
	lc.Uninstall(context.Background(), r.Get("test-p"))
	if !emitted {
		t.Error("plugin.removed event not emitted")
	}
}

type mockEmitterWithCheck struct {
	fn func(eventType string)
}

func (m *mockEmitterWithCheck) Emit(ctx context.Context, eventType string, payload interface{}, siteID string) error {
	m.fn(eventType)
	return nil
}


