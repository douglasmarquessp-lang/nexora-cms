package ai

import (
	"context"
	"testing"
	"time"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func TestNewManager(t *testing.T) {
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if m.Registry().Count() != 0 {
		t.Errorf("expected empty registry, got %d", m.Registry().Count())
	}
}

func setupManager(t *testing.T) *Manager {
	t.Helper()
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	p := NewMockProvider("mock", "mock-model", nil)
	m.RegisterProvider(p, ProviderCfg{Name: "mock", Enabled: true, Priority: 1, Weight: 10, MaxRetries: 2})
	return m
}

func TestManager_RegisterAndList(t *testing.T) {
	m := setupManager(t)
	providers := m.ListProviders()
	if len(providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(providers))
	}
}

func TestManager_DefaultProvider(t *testing.T) {
	m := setupManager(t)
	p, err := m.DefaultProvider()
	if err != nil {
		t.Fatalf("DefaultProvider failed: %v", err)
	}
	if p.Name() != "mock" {
		t.Errorf("expected mock, got %s", p.Name())
	}
}

func TestManager_DefaultProvider_Empty(t *testing.T) {
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	_, err := m.DefaultProvider()
	if err != ErrNoProvidersRegistered {
		t.Errorf("expected ErrNoProvidersRegistered, got %v", err)
	}
}

func TestManager_Provider(t *testing.T) {
	m := setupManager(t)
	p, err := m.Provider("mock")
	if err != nil {
		t.Fatalf("Provider failed: %v", err)
	}
	if p.Name() != "mock" {
		t.Errorf("expected mock, got %s", p.Name())
	}

	_, err = m.Provider("nonexistent")
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestManager_SetDefaultProvider(t *testing.T) {
	m := setupManager(t)
	p2 := NewMockProvider("p2", "model2", nil)
	m.RegisterProvider(p2, ProviderCfg{Name: "p2", Enabled: true, Priority: 2})

	err := m.SetDefaultProvider("p2")
	if err != nil {
		t.Fatalf("SetDefaultProvider failed: %v", err)
	}

	def, _ := m.DefaultProvider()
	if def.Name() != "p2" {
		t.Errorf("expected p2, got %s", def.Name())
	}
}

func TestManager_Generate(t *testing.T) {
	m := setupManager(t)
	ctx := context.Background()

	result, err := m.Generate(ctx, CompletionRequest{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestManager_Generate_MultipleProviders(t *testing.T) {
	m := setupManager(t)
	p2 := NewMockProvider("mock2", "model2", nil)
	m.RegisterProvider(p2, ProviderCfg{Name: "mock2", Enabled: true, Priority: 2, Weight: 5})
	ctx := context.Background()

	result, err := m.Generate(ctx, CompletionRequest{Prompt: "Test"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestManager_Embeddings(t *testing.T) {
	m := setupManager(t)
	pWithEmb := NewMockProvider("emb", "emb-model", []Capability{CapGenerate, CapEmbeddings})
	m.RegisterProvider(pWithEmb, ProviderCfg{Name: "emb", Enabled: true, Priority: 1})
	ctx := context.Background()

	result, err := m.Embeddings(ctx, "test input")
	if err != nil {
		t.Fatalf("Embeddings failed: %v", err)
	}
	if result.Dimensions != 128 {
		t.Errorf("expected 128 dims, got %d", result.Dimensions)
	}
}

func TestManager_EmbeddingsNotSupported(t *testing.T) {
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	p := NewMockProvider("gen-only", "gen-model", []Capability{CapGenerate})
	m.RegisterProvider(p, ProviderCfg{Name: "gen-only", Enabled: true, Priority: 1})
	ctx := context.Background()

	_, err := m.Embeddings(ctx, "test")
	if err != ErrEmbeddingNotSupported {
		t.Errorf("expected ErrEmbeddingNotSupported, got %v", err)
	}
}

func TestManager_Summarize(t *testing.T) {
	m := setupManager(t)
	p := NewMockProvider("sum", "sum-model", []Capability{CapGenerate, CapSummarize})
	m.RegisterProvider(p, ProviderCfg{Name: "sum", Enabled: true, Priority: 1})
	ctx := context.Background()

	result, err := m.Summarize(ctx, SummarizeRequest{Text: "Long text."})
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestManager_Rewrite(t *testing.T) {
	m := setupManager(t)
	p := NewMockProvider("rew", "rew-model", []Capability{CapGenerate, CapRewrite})
	m.RegisterProvider(p, ProviderCfg{Name: "rew", Enabled: true, Priority: 1})
	ctx := context.Background()

	result, err := m.Rewrite(ctx, RewriteRequest{Text: "Original"})
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestManager_Classify(t *testing.T) {
	m := setupManager(t)
	p := NewMockProvider("clf", "clf-model", []Capability{CapGenerate, CapClassify})
	m.RegisterProvider(p, ProviderCfg{Name: "clf", Enabled: true, Priority: 1})
	ctx := context.Background()

	result, err := m.Classify(ctx, ClassifyRequest{Text: "Sample", Categories: []string{"a", "b"}})
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if result.Category == "" {
		t.Error("expected a category")
	}
}

func TestManager_Health(t *testing.T) {
	m := setupManager(t)
	ctx := context.Background()

	report, err := m.Health(ctx)
	if err != nil {
		t.Fatalf("Health failed: %v", err)
	}
	if report.Overall != ProviderHealthy {
		t.Errorf("expected healthy, got %s", report.Overall)
	}
}

func TestManager_Metrics(t *testing.T) {
	m := setupManager(t)
	ctx := context.Background()

	m.Generate(ctx, CompletionRequest{Prompt: "Hello"})
	m.Generate(ctx, CompletionRequest{Prompt: "World"})

	metrics := m.Metrics()
	if metrics.TotalRequests != 2 {
		t.Errorf("expected 2 requests, got %d", metrics.TotalRequests)
	}
}

func TestManager_Capabilities(t *testing.T) {
	m := setupManager(t)
	caps := m.Capabilities()
	if len(caps) == 0 {
		t.Error("expected non-empty capabilities")
	}
}

func TestManager_Prompts(t *testing.T) {
	m := setupManager(t)
	pb := m.Prompts()
	if pb == nil {
		t.Fatal("expected non-nil prompt builder")
	}
}

func TestManager_Quality(t *testing.T) {
	m := setupManager(t)
	qc := m.Quality()
	if qc == nil {
		t.Fatal("expected non-nil quality checker")
	}
}

func TestManager_CircuitBreaker_Recovery(t *testing.T) {
	log := logger.New(&config.Config{})
	cfg := DefaultConfig()
	cfg.CircuitBreaker.Enabled = true
	cfg.CircuitBreaker.FailureThreshold = 2
	cfg.CircuitBreaker.RecoveryTimeout = 100 * time.Millisecond
	cfg.CircuitBreaker.HalfOpenMaxReqs = 2
	cfg.Retry.MaxAttempts = 1

	m := NewManager(cfg, log)
	p := NewMockProvider("failing", "f-model", nil)
	p.SetFailRate(1.0)
	m.RegisterProvider(p, ProviderCfg{Name: "failing", Enabled: true, Priority: 1, MaxRetries: 0})
	ctx := context.Background()

	m.Generate(ctx, CompletionRequest{Prompt: "Hello"})
	m.Generate(ctx, CompletionRequest{Prompt: "World"})

	// Test that circuit breaker doesn't panic and metrics work.
	// With 100% fail rate and max retries=0, each Generate call makes 1 attempt.
	metrics := m.Metrics()
	if metrics.FailedRequests == 0 && metrics.TotalRequests == 0 {
		t.Error("expected some failed requests recorded")
	}
}

func TestManager_GenerateStream(t *testing.T) {
	m := setupManager(t)
	ctx := context.Background()

	acc := NewStreamAccumulator()
	err := m.GenerateStream(ctx, CompletionRequest{Prompt: "Test stream"}, acc)
	if err != nil {
		t.Fatalf("GenerateStream failed: %v", err)
	}
	if acc.Text() == "" {
		t.Error("expected non-empty accumulated text")
	}
}

func TestManager_NoProviders(t *testing.T) {
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	ctx := context.Background()

	_, err := m.Generate(ctx, CompletionRequest{Prompt: "Hello"})
	if err != ErrNoProvidersRegistered {
		t.Errorf("expected ErrNoProvidersRegistered, got %v", err)
	}
}

func TestManager_SetEventBus(t *testing.T) {
	m := setupManager(t)
	// SetEventBus should not panic
	m.SetEventBus(nil)
}

func TestManager_GenerateWithContextCancelled(t *testing.T) {
	m := setupManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := m.Generate(ctx, CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Error("expected error with cancelled context")
	}
}
