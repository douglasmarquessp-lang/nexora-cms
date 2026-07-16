package plugins

import (
	"testing"
)

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	p := &PluginInstance{
		Manifest: &PluginManifest{
			ID:   "test-p",
			Name: "Test",
		},
	}
	if err := r.Register(p); err != nil {
		t.Fatal(err)
	}
	if !r.Exists("test-p") {
		t.Error("expected exists")
	}
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	r := NewRegistry()
	r.Register(&PluginInstance{
		Manifest: &PluginManifest{ID: "test-p", Name: "Test"},
	})
	err := r.Register(&PluginInstance{
		Manifest: &PluginManifest{ID: "test-p", Name: "Test"},
	})
	if err == nil {
		t.Fatal("expected error for duplicate")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	r.Register(&PluginInstance{
		Manifest: &PluginManifest{ID: "test-p", Name: "Test Plugin"},
	})
	p := r.Get("test-p")
	if p == nil {
		t.Fatal("expected plugin")
	}
	if p.Manifest.Name != "Test Plugin" {
		t.Errorf("Name = %q", p.Manifest.Name)
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := NewRegistry()
	p := r.Get("nonexistent")
	if p != nil {
		t.Error("expected nil")
	}
}

func TestRegistry_GetAll(t *testing.T) {
	r := NewRegistry()
	r.Register(&PluginInstance{Manifest: &PluginManifest{ID: "p1"}})
	r.Register(&PluginInstance{Manifest: &PluginManifest{ID: "p2"}})
	r.Register(&PluginInstance{Manifest: &PluginManifest{ID: "p3"}})

	all := r.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3, got %d", len(all))
	}
}

func TestRegistry_ListByStatus(t *testing.T) {
	r := NewRegistry()
	r.Register(&PluginInstance{Manifest: &PluginManifest{ID: "p1"}, Status: PluginStatusActive})
	r.Register(&PluginInstance{Manifest: &PluginManifest{ID: "p2"}, Status: PluginStatusInactive})
	r.Register(&PluginInstance{Manifest: &PluginManifest{ID: "p3"}, Status: PluginStatusActive})

	active := r.ListByStatus(PluginStatusActive)
	if len(active) != 2 {
		t.Errorf("expected 2 active, got %d", len(active))
	}
}

func TestRegistry_Remove(t *testing.T) {
	r := NewRegistry()
	r.Register(&PluginInstance{Manifest: &PluginManifest{ID: "test-p"}})
	r.Remove("test-p")
	if r.Exists("test-p") {
		t.Error("expected removed")
	}
}

func TestRegistry_SetStatus(t *testing.T) {
	r := NewRegistry()
	r.Register(&PluginInstance{Manifest: &PluginManifest{ID: "test-p"}, Status: PluginStatusInstalled})

	if err := r.SetStatus("test-p", PluginStatusActive); err != nil {
		t.Fatal(err)
	}
	p := r.Get("test-p")
	if p.Status != PluginStatusActive {
		t.Errorf("status = %v", p.Status)
	}
}

func TestRegistry_SetStatus_NotFound(t *testing.T) {
	r := NewRegistry()
	err := r.SetStatus("nonexistent", PluginStatusActive)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistry_SetDBID(t *testing.T) {
	r := NewRegistry()
	r.Register(&PluginInstance{Manifest: &PluginManifest{ID: "test-p"}})

	if err := r.SetDBID("test-p", "db-uuid-123"); err != nil {
		t.Fatal(err)
	}
	p := r.Get("test-p")
	if p.DBID != "db-uuid-123" {
		t.Errorf("DBID = %q", p.DBID)
	}
}

func TestRegistry_Exists(t *testing.T) {
	r := NewRegistry()
	r.Register(&PluginInstance{Manifest: &PluginManifest{ID: "test-p"}})
	if !r.Exists("test-p") {
		t.Error("expected true")
	}
	if r.Exists("nonexistent") {
		t.Error("expected false")
	}
}
