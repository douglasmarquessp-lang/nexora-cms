package ai

import (
	"context"
	"strings"
	"testing"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

// promptRecordingProvider records every Generate request so tests can assert
// what actually reached the model prompt.
type promptRecordingProvider struct {
	base    *MockProvider
	prompts []string
}

func (p *promptRecordingProvider) Generate(ctx context.Context, req CompletionRequest) (*CompletionResult, error) {
	p.prompts = append(p.prompts, req.Prompt)
	return p.base.Generate(ctx, req)
}

func (p *promptRecordingProvider) GenerateStream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	return p.base.GenerateStream(ctx, req)
}

func (p *promptRecordingProvider) Embeddings(ctx context.Context, input string) (*EmbeddingResult, error) {
	return p.base.Embeddings(ctx, input)
}

func (p *promptRecordingProvider) Summarize(ctx context.Context, req SummarizeRequest) (string, error) {
	return p.base.Summarize(ctx, req)
}

func (p *promptRecordingProvider) Rewrite(ctx context.Context, req RewriteRequest) (string, error) {
	return p.base.Rewrite(ctx, req)
}

func (p *promptRecordingProvider) Classify(ctx context.Context, req ClassifyRequest) (*ClassifyResult, error) {
	return p.base.Classify(ctx, req)
}

func (p *promptRecordingProvider) Health(ctx context.Context) (*HealthStatus, error) {
	return p.base.Health(ctx)
}

func (p *promptRecordingProvider) Name() string {
	return p.base.Name()
}

func (p *promptRecordingProvider) Capabilities() []Capability {
	return p.base.Capabilities()
}

func setupRecordingPipelineTest(t *testing.T) (*PipelineExecutor, *promptRecordingProvider) {
	t.Helper()
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	rp := &promptRecordingProvider{base: NewMockProvider("mock", "mock-model", nil)}
	m.RegisterProvider(rp, ProviderCfg{Name: "mock", Enabled: true, Priority: 1, Weight: 10})
	return NewPipelineExecutor(m), rp
}

func TestWordCountDefaults_ZeroNeverReachesPrompt(t *testing.T) {
	pe, rp := setupRecordingPipelineTest(t)
	ctx := context.Background()

	_, err := pe.ExecuteStage(ctx, StageDraftGen, PipelineInput{
		Title:        "Test Article",
		ContentType:  "article",
		Language:     "en",
		Briefing:     "Topic briefing",
		Outline:      "1. Intro 2. Body",
		Keywords:     []string{"test"},
		WordCount:    0, // the caller forgot to specify a target
		WordCountMin: 0,
	})
	if err != nil {
		t.Fatalf("ExecuteStage draft failed: %v", err)
	}
	if len(rp.prompts) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(rp.prompts))
	}
	prompt := rp.prompts[0]
	if !strings.Contains(prompt, "Target Length: 1200 words (minimum 1000 words") {
		t.Errorf("zero word count must default to 1200/1000, got: %.100s…", prompt)
	}
	if strings.Contains(prompt, "Target Length: 0") || strings.Contains(prompt, "minimum 0 words") {
		t.Errorf("a zero word count must never reach the prompt: %.100s…", prompt)
	}
}

func TestWordCountPreserved_WhenProvided(t *testing.T) {
	pe, rp := setupRecordingPipelineTest(t)
	ctx := context.Background()

	_, err := pe.ExecuteStage(ctx, StageDraftGen, PipelineInput{
		Title:        "Test Article",
		ContentType:  "article",
		Language:     "en",
		Briefing:     "Topic briefing",
		Outline:      "1. Intro 2. Body",
		Keywords:     []string{"test"},
		WordCount:    1500,
		WordCountMin: 800,
	})
	if err != nil {
		t.Fatalf("ExecuteStage draft failed: %v", err)
	}
	prompt := rp.prompts[0]
	if !strings.Contains(prompt, "Target Length: 1500 words (minimum 800 words") {
		t.Errorf("caller word count must be preserved, got: %.100s…", prompt)
	}
}

func TestWordCountDefaults_Outline(t *testing.T) {
	pe, rp := setupRecordingPipelineTest(t)
	ctx := context.Background()

	_, err := pe.ExecuteStage(ctx, StageOutlineGen, PipelineInput{
		Title:        "Test Article",
		Language:     "en",
		Briefing:     "Topic briefing",
		Keywords:     []string{"test"},
		WordCount:    0,
		WordCountMin: 0,
	})
	if err != nil {
		t.Fatalf("ExecuteStage outline failed: %v", err)
	}
	prompt := rp.prompts[0]
	if !strings.Contains(prompt, "Expected Size: 1200 words") {
		t.Errorf("outline must default to 1200 words, got: %.100s…", prompt)
	}
	if strings.Contains(prompt, "Expected Size: 0") {
		t.Errorf("a zero word count must never reach the outline prompt")
	}
}

func TestEffectiveWordCount(t *testing.T) {
	tests := []struct {
		in       PipelineInput
		wantWord int
		wantMin  int
	}{
		{PipelineInput{WordCount: 0, WordCountMin: 0}, 1200, 1000},
		{PipelineInput{WordCount: -5, WordCountMin: -1}, 1200, 1000},
		{PipelineInput{WordCount: 1500, WordCountMin: 800}, 1500, 800},
	}
	for _, tt := range tests {
		if got := effectiveWordCount(tt.in); got != tt.wantWord {
			t.Errorf("effectiveWordCount(%+v) = %d, want %d", tt.in, got, tt.wantWord)
		}
		if got := effectiveMinWordCount(tt.in); got != tt.wantMin {
			t.Errorf("effectiveMinWordCount(%+v) = %d, want %d", tt.in, got, tt.wantMin)
		}
	}
}
