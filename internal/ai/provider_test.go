package ai

import (
	"context"
	"testing"
	"time"
)

func TestNewMockProvider(t *testing.T) {
	p := NewMockProvider("test", "test-model", nil)
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Name() != "test" {
		t.Errorf("expected name 'test', got %s", p.Name())
	}
}

func TestMockProvider_Generate(t *testing.T) {
	p := NewMockProvider("test", "test-model", nil)
	ctx := context.Background()

	result, err := p.Generate(ctx, CompletionRequest{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
	if result.ProviderName != "test" {
		t.Errorf("expected provider 'test', got %s", result.ProviderName)
	}
}

func TestMockProvider_GenerateStream(t *testing.T) {
	p := NewMockProvider("test", "test-model", nil)
	ctx := context.Background()

	ch, err := p.GenerateStream(ctx, CompletionRequest{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("GenerateStream failed: %v", err)
	}

	var chunks []StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
	if !chunks[len(chunks)-1].Done {
		t.Error("expected last chunk to be marked Done")
	}
}

func TestMockProvider_Embeddings(t *testing.T) {
	p := NewMockProvider("test", "test-model", []Capability{CapGenerate, CapEmbeddings})
	ctx := context.Background()

	result, err := p.Embeddings(ctx, "test input")
	if err != nil {
		t.Fatalf("Embeddings failed: %v", err)
	}
	if result.Dimensions != 128 {
		t.Errorf("expected 128 dimensions, got %d", result.Dimensions)
	}
}

func TestMockProvider_EmbeddingsNotSupported(t *testing.T) {
	p := NewMockProvider("test", "test-model", []Capability{CapGenerate})
	ctx := context.Background()

	_, err := p.Embeddings(ctx, "test input")
	if err != ErrEmbeddingNotSupported {
		t.Errorf("expected ErrEmbeddingNotSupported, got %v", err)
	}
}

func TestMockProvider_Summarize(t *testing.T) {
	p := NewMockProvider("test", "test-model", []Capability{CapGenerate, CapSummarize})
	ctx := context.Background()

	result, err := p.Summarize(ctx, SummarizeRequest{Text: "Long text here."})
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty summary")
	}
}

func TestMockProvider_Rewrite(t *testing.T) {
	p := NewMockProvider("test", "test-model", []Capability{CapGenerate, CapRewrite})
	ctx := context.Background()

	result, err := p.Rewrite(ctx, RewriteRequest{Text: "Original text"})
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty rewrite")
	}
}

func TestMockProvider_Classify(t *testing.T) {
	p := NewMockProvider("test", "test-model", []Capability{CapGenerate, CapClassify})
	ctx := context.Background()

	result, err := p.Classify(ctx, ClassifyRequest{
		Text:       "Sample text",
		Categories: []string{"tech", "sports", "health"},
	})
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if result.Category == "" {
		t.Error("expected a category")
	}
	if result.Confidence <= 0 {
		t.Error("expected positive confidence")
	}
}

func TestMockProvider_Health(t *testing.T) {
	p := NewMockProvider("test", "test-model", nil)
	ctx := context.Background()

	status, err := p.Health(ctx)
	if err != nil {
		t.Fatalf("Health failed: %v", err)
	}
	if status.State != ProviderHealthy {
		t.Errorf("expected healthy, got %s", status.State)
	}
}

func TestMockProvider_Unhealthy(t *testing.T) {
	p := NewMockProvider("test", "test-model", nil)
	p.SetHealthy(false)
	ctx := context.Background()

	_, err := p.Generate(ctx, CompletionRequest{Prompt: "Hello"})
	if err != ErrProviderUnavailable {
		t.Errorf("expected ErrProviderUnavailable, got %v", err)
	}
}

func TestMockProvider_Capabilities(t *testing.T) {
	caps := []Capability{CapGenerate, CapStream}
	p := NewMockProvider("test", "test-model", caps)

	got := p.Capabilities()
	if len(got) != len(caps) {
		t.Errorf("expected %d capabilities, got %d", len(caps), len(got))
	}
}

func TestMockProvider_SetLatency(t *testing.T) {
	p := NewMockProvider("test", "test-model", nil)
	p.SetLatency(5 * time.Millisecond)
	p.SetHealthy(false)

	start := time.Now()
	_, err := p.Health(context.Background())
	elapsed := time.Since(start)

	if err == nil {
		t.Error("expected error for unhealthy provider")
	}
	_ = elapsed
}

func TestMockProvider_FailRate(t *testing.T) {
	p := NewMockProvider("test", "test-model", nil)
	p.SetFailRate(1.0)
	ctx := context.Background()

	_, err := p.Generate(ctx, CompletionRequest{Prompt: "Hello"})
	if err != ErrProviderUnavailable {
		t.Errorf("expected ErrProviderUnavailable with 100%% fail rate, got %v", err)
	}
}
