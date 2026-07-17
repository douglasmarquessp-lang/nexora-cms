package ai

import (
	"context"
	"testing"

	"nexora/internal/kernel"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func TestNewAIModule(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	m := NewAIModule(cfg, log, nil, nil)
	if m == nil {
		t.Fatal("expected non-nil module")
	}
	if m.Name() != ModuleName {
		t.Errorf("expected name '%s', got '%s'", ModuleName, m.Name())
	}
}

func TestAIModule_Init(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	m := NewAIModule(cfg, log, nil, nil)

	err := m.Init(context.Background())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	svc := m.Service()
	if svc == nil {
		t.Fatal("expected non-nil service after Init")
	}
}

func TestAIModule_StartStop(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	m := NewAIModule(cfg, log, nil, nil)

	err := m.Init(context.Background())
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	err = m.Start(context.Background())
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = m.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestAIModule_SetEventBus(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	m := NewAIModule(cfg, log, nil, nil)
	m.Init(context.Background())

	// SetEventBus with nil should not panic
	m.SetEventBus(nil)
}

func TestAIModule_KernelModuleInterface(t *testing.T) {
	var _ kernel.Module = (*AIModule)(nil)
}

func TestAIModule_Init_WithProviders(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	m := NewAIModule(cfg, log, nil, nil)
	m.Init(context.Background())

	svc := m.Service()
	providers := svc.ListProviders()
	if len(providers) == 0 {
		t.Error("expected at least one registered provider after init")
	}
}

func TestAIModule_Service_Generate(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	m := NewAIModule(cfg, log, nil, nil)
	m.Init(context.Background())

	svc := m.Service()
	ctx := context.Background()

	result, err := svc.Generate(ctx, CompletionRequest{Prompt: "Hello from module"})
	if err != nil {
		t.Fatalf("Generate via module service failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}
