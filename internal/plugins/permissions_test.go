package plugins

import (
	"testing"
)

func TestPermissions_Register(t *testing.T) {
	p := NewPermissions()
	p.Register("test-p", []PermissionDef{
		{PluginID: "test-p", Permission: "read", Description: "Read access"},
	})

	perms := p.GetByPlugin("test-p")
	if len(perms) != 1 {
		t.Errorf("expected 1, got %d", len(perms))
	}
}

func TestPermissions_GetByPlugin_NotFound(t *testing.T) {
	p := NewPermissions()
	perms := p.GetByPlugin("nonexistent")
	if len(perms) != 0 {
		t.Errorf("expected 0, got %d", len(perms))
	}
}

func TestPermissions_GetAll(t *testing.T) {
	p := NewPermissions()
	p.Register("p1", []PermissionDef{
		{PluginID: "p1", Permission: "read"},
	})
	p.Register("p2", []PermissionDef{
		{PluginID: "p2", Permission: "write"},
	})

	all := p.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}
}

func TestPermissions_Check(t *testing.T) {
	p := NewPermissions()
	p.Register("test-p", []PermissionDef{
		{PluginID: "test-p", Permission: "read"},
	})

	if err := p.Check("test-p", "read"); err != nil {
		t.Fatal(err)
	}
}

func TestPermissions_Check_NotFound(t *testing.T) {
	p := NewPermissions()
	if err := p.Check("nonexistent", "read"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPermissions_Check_WrongPerm(t *testing.T) {
	p := NewPermissions()
	p.Register("test-p", []PermissionDef{
		{PluginID: "test-p", Permission: "read"},
	})

	if err := p.Check("test-p", "write"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPermissions_RemovePlugin(t *testing.T) {
	p := NewPermissions()
	p.Register("test-p", []PermissionDef{
		{PluginID: "test-p", Permission: "read"},
	})

	p.RemovePlugin("test-p")
	perms := p.GetByPlugin("test-p")
	if len(perms) != 0 {
		t.Errorf("expected 0, got %d", len(perms))
	}
}
