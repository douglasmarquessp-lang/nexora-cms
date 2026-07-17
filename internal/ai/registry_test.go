package ai

import (
	"context"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if r.Count() != 0 {
		t.Errorf("expected empty registry, got %d", r.Count())
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	p := NewMockProvider("test", "test-model", nil)

	err := r.Register(p, ProviderCfg{Name: "test", Enabled: true, Priority: 1, Weight: 10})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if r.Count() != 1 {
		t.Errorf("expected 1 provider, got %d", r.Count())
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	p := NewMockProvider("test", "test-model", nil)
	r.Register(p, ProviderCfg{Name: "test", Enabled: true})

	got, _, ok := r.Get("test")
	if !ok {
		t.Fatal("expected to find test provider")
	}
	if got.Name() != "test" {
		t.Errorf("expected name 'test', got %s", got.Name())
	}

	_, _, ok = r.Get("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent provider")
	}
}

func TestRegistry_Default(t *testing.T) {
	r := NewRegistry()
	p1 := NewMockProvider("p1", "model1", nil)
	p2 := NewMockProvider("p2", "model2", nil)

	r.Register(p1, ProviderCfg{Name: "p1", Enabled: true, Priority: 2, Weight: 5})
	r.Register(p2, ProviderCfg{Name: "p2", Enabled: true, Priority: 1, Weight: 10})

	def, _, ok := r.Default()
	if !ok {
		t.Fatal("expected default provider")
	}
	if def.Name() != "p1" {
		t.Errorf("expected p1 as default (first registered), got %s", def.Name())
	}
}

func TestRegistry_SetDefault(t *testing.T) {
	r := NewRegistry()
	p1 := NewMockProvider("p1", "model1", nil)
	p2 := NewMockProvider("p2", "model2", nil)

	r.Register(p1, ProviderCfg{Name: "p1", Enabled: true})
	r.Register(p2, ProviderCfg{Name: "p2", Enabled: true})

	err := r.SetDefault("p2")
	if err != nil {
		t.Fatalf("SetDefault failed: %v", err)
	}

	def, _, _ := r.Default()
	if def.Name() != "p2" {
		t.Errorf("expected p2 as default, got %s", def.Name())
	}

	err = r.SetDefault("nonexistent")
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	p1 := NewMockProvider("p1", "model1", nil)
	p2 := NewMockProvider("p2", "model2", nil)

	r.Register(p1, ProviderCfg{Name: "p1", Enabled: true, Priority: 1})
	r.Register(p2, ProviderCfg{Name: "p2", Enabled: true, Priority: 2})

	infos := r.List()
	if len(infos) != 2 {
		t.Errorf("expected 2 providers, got %d", len(infos))
	}
}

func TestRegistry_EmptyList(t *testing.T) {
	r := NewRegistry()
	infos := r.List()
	if len(infos) != 0 {
		t.Errorf("expected 0 providers, got %d", len(infos))
	}
}

func TestRegistry_HasCapability(t *testing.T) {
	r := NewRegistry()
	p := NewMockProvider("test", "test-model", []Capability{CapGenerate, CapClassify})
	r.Register(p, ProviderCfg{Name: "test", Enabled: true})

	if !r.HasCapability(CapGenerate) {
		t.Error("expected to have generate capability")
	}
	if !r.HasCapability(CapClassify) {
		t.Error("expected to have classify capability")
	}
	if r.HasCapability(CapEmbeddings) {
		t.Error("expected not to have embeddings capability")
	}
}

func TestRegistry_FindByCapability(t *testing.T) {
	r := NewRegistry()
	p1 := NewMockProvider("p1", "model1", []Capability{CapGenerate})
	p2 := NewMockProvider("p2", "model2", []Capability{CapGenerate, CapClassify})
	p3 := NewMockProvider("p3", "model3", []Capability{CapEmbeddings})

	r.Register(p1, ProviderCfg{Name: "p1", Enabled: true, Priority: 1})
	r.Register(p2, ProviderCfg{Name: "p2", Enabled: true, Priority: 2})
	r.Register(p3, ProviderCfg{Name: "p3", Enabled: true, Priority: 3})

	providers := r.FindByCapability(CapGenerate)
	if len(providers) != 2 {
		t.Errorf("expected 2 providers with generate, got %d", len(providers))
	}

	providers = r.FindByCapability(CapEmbeddings)
	if len(providers) != 1 {
		t.Errorf("expected 1 provider with embeddings, got %d", len(providers))
	}
}

func TestRegistry_HealthCheck(t *testing.T) {
	r := NewRegistry()
	p := NewMockProvider("test", "test-model", nil)
	r.Register(p, ProviderCfg{Name: "test", Enabled: true})

	report := r.HealthCheck(context.Background())
	if report.Overall != ProviderHealthy {
		t.Errorf("expected healthy, got %s", report.Overall)
	}
	if len(report.Providers) != 1 {
		t.Errorf("expected 1 provider in report, got %d", len(report.Providers))
	}
}

func TestRegistry_HealthCheckEmpty(t *testing.T) {
	r := NewRegistry()
	report := r.HealthCheck(context.Background())
	if report.Overall != ProviderUnhealthy {
		t.Errorf("expected unhealthy for empty registry, got %s", report.Overall)
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewRegistry()
	p := NewMockProvider("test", "test-model", nil)
	r.Register(p, ProviderCfg{Name: "test", Enabled: true})

	if r.Count() != 1 {
		t.Errorf("expected 1 before unregister, got %d", r.Count())
	}

	r.Unregister("test")
	if r.Count() != 0 {
		t.Errorf("expected 0 after unregister, got %d", r.Count())
	}
}

func TestRegistry_PriorityOrdering(t *testing.T) {
	r := NewRegistry()
	r.Register(NewMockProvider("low", "l", nil), ProviderCfg{Name: "low", Enabled: true, Priority: 10, Weight: 1})
	r.Register(NewMockProvider("high", "h", nil), ProviderCfg{Name: "high", Enabled: true, Priority: 1, Weight: 10})

	infos := r.List()
	if len(infos) < 2 {
		t.Fatal("expected 2 providers")
	}
	if infos[0].Name != "high" {
		t.Errorf("expected 'high' first (priority 1), got %s", infos[0].Name)
	}
}

func TestRegistry_DisabledProviders(t *testing.T) {
	r := NewRegistry()
	r.Register(NewMockProvider("enabled", "e", nil), ProviderCfg{Name: "enabled", Enabled: true, Priority: 1})
	r.Register(NewMockProvider("disabled", "d", nil), ProviderCfg{Name: "disabled", Enabled: false, Priority: 2})

	infos := r.List()
	// disabled entry still appears in list but is marked enabled=false
	for _, info := range infos {
		if info.Name == "disabled" && info.Enabled {
			t.Error("disabled provider should be marked as not enabled")
		}
	}
}
