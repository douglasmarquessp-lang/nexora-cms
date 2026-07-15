package site

import (
	"context"
	"testing"

	"nexora/internal/kernel"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func TestNewSiteModule(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	mod := NewSiteModule(cfg, log, nil, ch)

	if mod == nil {
		t.Fatal("expected non-nil module")
	}
	if mod.name != ModuleName {
		t.Errorf("expected name %q, got %q", ModuleName, mod.name)
	}
}

func TestSiteModule_Name(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	mod := NewSiteModule(cfg, log, nil, ch)

	if mod.Name() != "site" {
		t.Errorf("expected 'site', got '%s'", mod.Name())
	}
}

func TestSiteModule_Init(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	mod := NewSiteModule(cfg, log, nil, ch)

	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if mod.Service() == nil {
		t.Fatal("expected non-nil service after Init")
	}
}

func TestSiteModule_Start(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	mod := NewSiteModule(cfg, log, nil, ch)

	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := mod.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
}

func TestSiteModule_Stop(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	mod := NewSiteModule(cfg, log, nil, ch)

	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := mod.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestSiteModule_Service_BeforeInit(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	mod := NewSiteModule(cfg, log, nil, ch)

	if mod.Service() != nil {
		t.Fatal("expected nil service before Init")
	}
}

func TestSiteModule_Service_AfterInit(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	mod := NewSiteModule(cfg, log, nil, ch)

	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	svc := mod.Service()
	if svc == nil {
		t.Fatal("expected non-nil service after Init")
	}
}

func TestSiteModule_SetEventBus_Nil(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	mod := NewSiteModule(cfg, log, nil, ch)
	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	mod.SetEventBus(nil)
}

func TestSiteModule_SetEventBus_WithBus(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	mod := NewSiteModule(cfg, log, nil, ch)
	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	bus := kernel.NewEventBus(log)
	mod.SetEventBus(bus)

	if mod.eventBus != bus {
		t.Error("expected module to have event bus set")
	}
}

func TestSiteModule_SetEventBus_PropagatesToService(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	mod := NewSiteModule(cfg, log, nil, ch)
	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	bus := kernel.NewEventBus(log)
	mod.SetEventBus(bus)

	svc := mod.Service()
	svc.fireEvent(context.Background(), EventSiteCreated, map[string]interface{}{
		"test": "value",
	})
}

func TestSiteModule_FullLifecycle(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	mod := NewSiteModule(cfg, log, nil, ch)

	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	bus := kernel.NewEventBus(log)
	mod.SetEventBus(bus)

	if err := mod.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := mod.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestSiteModule_ImplementsModuleInterface(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	mod := NewSiteModule(cfg, log, nil, ch)

	var _ kernel.Module = mod
}

func TestSiteModule_DoubleInit(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	ch := cache.New(false)
	mod := NewSiteModule(cfg, log, nil, ch)

	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("first Init failed: %v", err)
	}
	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("second Init failed: %v", err)
	}
}
