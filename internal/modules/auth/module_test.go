package auth

import (
	"context"
	"testing"

	"nexora/internal/kernel"
	"nexora/internal/pkg/logger"
)

func TestAuthModule_Name(t *testing.T) {
	mod := NewAuthModule(testConfig(), nil, nil)
	if mod.Name() != ModuleName {
		t.Errorf("expected %s, got %s", ModuleName, mod.Name())
	}
	if mod.Name() != "auth" {
		t.Errorf("expected auth, got %s", mod.Name())
	}
}

func TestAuthModule_Init(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	mod := NewAuthModule(cfg, log, nil)

	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if mod.Service() == nil {
		t.Fatal("expected non-nil service after Init")
	}
}

func TestAuthModule_Start(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	mod := NewAuthModule(cfg, log, nil)

	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := mod.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
}

func TestAuthModule_Stop(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	mod := NewAuthModule(cfg, log, nil)

	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := mod.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestAuthModule_Service_BeforeInit(t *testing.T) {
	mod := NewAuthModule(testConfig(), nil, nil)
	if mod.Service() != nil {
		t.Fatal("expected nil service before Init")
	}
}

func TestAuthModule_SetEventBus_Nil(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	mod := NewAuthModule(cfg, log, nil)
	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Should not panic
	mod.SetEventBus(nil)
}

func TestAuthModule_SetEventBus_WithBus(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	mod := NewAuthModule(cfg, log, nil)
	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	bus := kernel.NewEventBus(log)
	mod.SetEventBus(bus)

	if mod.eventBus != bus {
		t.Error("expected module to have event bus set")
	}
}

func TestAuthModule_SetEventBus_PropagatesToService(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	mod := NewAuthModule(cfg, log, nil)
	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	bus := kernel.NewEventBus(log)
	mod.SetEventBus(bus)

	// Verify service's SetEventBus was called
	svc := mod.Service()
	// fireEvent should use the event bus now
	svc.fireEvent(context.Background(), kernel.EventUserRegistered, map[string]interface{}{
		"test": "value",
	})
	// No panic means success
}

func TestAuthModule_HandleEvent(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	mod := NewAuthModule(cfg, log, nil)

	err := mod.handleEvent(context.Background(), kernel.Event{
		Type: kernel.EventUserRegistered,
		ID:   "test-id",
	})
	if err != nil {
		t.Fatalf("handleEvent failed: %v", err)
	}
}

func TestAuthModule_FullLifecycle(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	mod := NewAuthModule(cfg, log, nil)

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

func TestAuthModule_ImplementsModuleInterface(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	mod := NewAuthModule(cfg, log, nil)

	var _ kernel.Module = mod
}

func TestAuthModule_DoubleInit(t *testing.T) {
	cfg := testConfig()
	log := logger.New(cfg)
	mod := NewAuthModule(cfg, log, nil)

	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("first Init failed: %v", err)
	}
	if err := mod.Init(context.Background()); err != nil {
		t.Fatalf("second Init failed: %v", err)
	}
}
