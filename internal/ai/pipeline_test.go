package ai

import (
	"context"
	"testing"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func setupPipelineTest(t *testing.T) *PipelineExecutor {
	t.Helper()
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	p := NewMockProvider("mock", "mock-model", nil)
	m.RegisterProvider(p, ProviderCfg{Name: "mock", Enabled: true, Priority: 1, Weight: 10})
	return NewPipelineExecutor(m)
}

func TestNewPipelineExecutor(t *testing.T) {
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	pe := NewPipelineExecutor(m)
	if pe == nil {
		t.Fatal("expected non-nil pipeline executor")
	}
}

func TestPipelineExecutor_ResearchStage(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	result, err := pe.ExecuteStage(ctx, StageResearchGen, PipelineInput{
		Topic: "AI Technology",
	})
	if err != nil {
		t.Fatalf("ExecuteStage research failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
	if result.Stage != StageResearchGen {
		t.Errorf("expected StageResearchGen, got %v", result.Stage)
	}
}

func TestPipelineExecutor_BriefingStage(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	result, err := pe.ExecuteStage(ctx, StageBriefingGen, PipelineInput{
		Topic:   "Climate",
		Briefing: "Source data",
	})
	if err != nil {
		t.Fatalf("ExecuteStage briefing failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestPipelineExecutor_OutlineStage(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	result, err := pe.ExecuteStage(ctx, StageOutlineGen, PipelineInput{
		Title:     "Test Article",
		Briefing:  "Brief description",
		Keywords:  []string{"test", "golang"},
		WordCount: 500,
	})
	if err != nil {
		t.Fatalf("ExecuteStage outline failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestPipelineExecutor_DraftStage(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	result, err := pe.ExecuteStage(ctx, StageDraftGen, PipelineInput{
		Title:       "Test Article",
		ContentType: "blog",
		Language:    "en",
		Briefing:    "Topic briefing",
		Outline:     "1. Intro 2. Body",
		Keywords:    []string{"test"},
		WordCount:   500,
		Tone:        "professional",
		Audience:    "developers",
		Style:       map[string]string{"format": "markdown"},
	})
	if err != nil {
		t.Fatalf("ExecuteStage draft failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestPipelineExecutor_DraftStagePortuguese(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	result, err := pe.ExecuteStage(ctx, StageDraftGen, PipelineInput{
		Title:       "Artigo Teste",
		ContentType: "blog",
		Language:    "pt",
		Briefing:    "Briefing do tópico",
		Outline:     "1. Intro 2. Corpo",
		Keywords:    []string{"teste"},
		WordCount:   500,
		Tone:        "profissional",
		Audience:    "desenvolvedores",
	})
	if err != nil {
		t.Fatalf("ExecuteStage draft PT failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestPipelineExecutor_SEOStage(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	result, err := pe.ExecuteStage(ctx, StageSEOGen, PipelineInput{
		Briefing: "Article content for SEO",
		Keywords: []string{"seo", "optimization"},
	})
	if err != nil {
		t.Fatalf("ExecuteStage SEO failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestPipelineExecutor_QualityStage(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	result, err := pe.ExecuteStage(ctx, StageQualityCheck, PipelineInput{
		Briefing: "This is a test article for quality checking purposes. It has enough words to pass minimum requirements. We need to ensure grammar is correct. SEO keywords should be present. Readability should be good.",
		Keywords: []string{"test", "quality", "article"},
		Entities: []string{"test", "article"},
		Language: "en",
	})
	if err != nil {
		t.Fatalf("ExecuteStage quality failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestPipelineExecutor_TranslationStage(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	result, err := pe.ExecuteStage(ctx, StageTranslationGen, PipelineInput{
		Briefing:  "Content to translate",
	})
	if err != nil {
		t.Fatalf("ExecuteStage translation failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestPipelineExecutor_ReviewStage(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	result, err := pe.ExecuteStage(ctx, StageFinalReview, PipelineInput{
		Briefing: "Content to review",
	})
	if err != nil {
		t.Fatalf("ExecuteStage review failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestPipelineExecutor_UnknownStage(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	_, err := pe.ExecuteStage(ctx, PipelineStage(99), PipelineInput{})
	if err == nil {
		t.Error("expected error for unknown stage")
	}
}

func TestPipelineExecutor_FullPipeline(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	results, err := pe.ExecuteFull(ctx, PipelineInput{
		Title:       "Full Pipeline Test",
		ContentType: "article",
		Language:    "en",
		Topic:       "Test topic",
		Briefing:    "Briefing data",
		Keywords:    []string{"test"},
		WordCount:   300,
		Tone:        "professional",
		Audience:    "general",
	})
	if err != nil {
		t.Fatalf("ExecuteFull failed: %v", err)
	}
	if len(results) != 8 {
		t.Errorf("expected 8 stage results, got %d", len(results))
	}
}

func TestJoinStrings(t *testing.T) {
	if joinStrings(nil, ", ") != "" {
		t.Error("expected empty for nil input")
	}
	if joinStrings([]string{}, ", ") != "" {
		t.Error("expected empty for empty input")
	}
	if joinStrings([]string{"a"}, ", ") != "a" {
		t.Errorf("expected 'a', got '%s'", joinStrings([]string{"a"}, ", "))
	}
	if joinStrings([]string{"a", "b", "c"}, ", ") != "a, b, c" {
		t.Errorf("expected 'a, b, c', got '%s'", joinStrings([]string{"a", "b", "c"}, ", "))
	}
}
