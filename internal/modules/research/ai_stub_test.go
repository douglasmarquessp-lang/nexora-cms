package research

import (
	"context"

	"nexora/internal/ai"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

// stubAIProvider returns a fixed Generate response — used to exercise the
// AI-assisted paths (fact base, briefing) without a network or real provider.
type stubAIProvider struct {
	content string
}

func (p *stubAIProvider) Generate(_ context.Context, _ ai.CompletionRequest) (*ai.CompletionResult, error) {
	return &ai.CompletionResult{Content: p.content, FinishReason: "stop"}, nil
}

func (p *stubAIProvider) GenerateStream(_ context.Context, _ ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	ch := make(chan ai.StreamChunk)
	close(ch)
	return ch, nil
}

func (p *stubAIProvider) Embeddings(_ context.Context, _ string) (*ai.EmbeddingResult, error) {
	return &ai.EmbeddingResult{Vector: []float64{}}, nil
}

func (p *stubAIProvider) Summarize(_ context.Context, _ ai.SummarizeRequest) (string, error) {
	return p.content, nil
}

func (p *stubAIProvider) Rewrite(_ context.Context, _ ai.RewriteRequest) (string, error) {
	return p.content, nil
}

func (p *stubAIProvider) Classify(_ context.Context, _ ai.ClassifyRequest) (*ai.ClassifyResult, error) {
	return &ai.ClassifyResult{Category: "test", Confidence: 1.0}, nil
}

func (p *stubAIProvider) Health(_ context.Context) (*ai.HealthStatus, error) {
	return &ai.HealthStatus{State: ai.ProviderHealthy, Latency: 0}, nil
}

func (p *stubAIProvider) Name() string { return "stub" }

func (p *stubAIProvider) Capabilities() []ai.Capability {
	return []ai.Capability{ai.CapGenerate, ai.CapGrounding}
}

// newAIManagerReturning builds an ai.Manager whose provider always returns the
// given content, and sets it on the service.
func newAIManagerReturning(content string) *ai.Manager {
	log := logger.New(&config.Config{})
	m := ai.NewManager(ai.DefaultConfig(), log)
	_ = m.RegisterProvider(&stubAIProvider{content: content}, ai.ProviderCfg{
		Name: "stub", Enabled: true, Priority: 1, Weight: 10,
	})
	return m
}
