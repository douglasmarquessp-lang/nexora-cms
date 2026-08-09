package ai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
		Topic:    "Climate",
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
		Briefing: "Content to translate",
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
	if len(results) != 10 {
		t.Errorf("expected 10 stage results, got %d", len(results))
	}
}

// --- Research → Grounding → Quality Check Integration Tests ---

func TestPipelineQualityStageWithGroundingMetadata(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	now := time.Now()
	gm := &GroundingMetadata{
		Sources: []GroundingSource{
			{
				URI:           "https://example.com/mock-source-1",
				Title:         "Mock Source 1",
				Snippet:       "This is a mock source snippet for testing.",
				RetrievedAt:   now,
				FreshnessScore: 0.95,
				IsVerified:    true,
			},
			{
				URI:           "https://example.com/mock-source-2",
				Title:         "Mock Source 2",
				Snippet:       "Another mock source for grounding tests.",
				RetrievedAt:   now,
				FreshnessScore: 0.85,
				IsVerified:    true,
			},
		},
		SearchSuggested: true,
		Unverified:      false,
	}

	result, err := pe.ExecuteStage(ctx, StageQualityCheck, PipelineInput{
		Briefing:          "This is a mock source snippet for testing. Artificial intelligence is transforming industries worldwide. Another mock source for grounding tests.",
		Keywords:          []string{"test", "quality"},
		Entities:          []string{"test"},
		Language:          "en",
		GroundingMetadata: gm,
	})
	if err != nil {
		t.Fatalf("ExecuteStage quality with grounding failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
	if !strings.Contains(result.Analysis, "Fact Check") {
		t.Error("expected fact check results when grounding metadata provided")
	}
	if result.Content != "This is a mock source snippet for testing. Artificial intelligence is transforming industries worldwide. Another mock source for grounding tests." {
		t.Error("quality stage must preserve the article in Content, never replace it with the report")
	}
}

func TestPipelineQualityStageWithoutGrounding(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	result, err := pe.ExecuteStage(ctx, StageQualityCheck, PipelineInput{
		Briefing: "This is a test article for quality checking purposes. It has enough words to pass minimum requirements.",
		Keywords: []string{"test", "quality"},
		Entities: []string{"test", "article"},
		Language: "en",
	})
	if err != nil {
		t.Fatalf("ExecuteStage quality without grounding failed: %v", err)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
	if strings.Contains(result.Analysis, "Fact Check") {
		t.Error("expected no fact check when no grounding metadata provided")
	}
}

func TestPipelineQualityStageWithEmptyGroundingMetadata(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	gm := &GroundingMetadata{
		Unverified: true,
	}

	result, err := pe.ExecuteStage(ctx, StageQualityCheck, PipelineInput{
		Briefing:          "This content has no supporting evidence.",
		Keywords:          []string{"test"},
		Language:          "en",
		GroundingMetadata: gm,
	})
	if err != nil {
		t.Fatalf("ExecuteStage quality with empty grounding failed: %v", err)
	}
	if !strings.Contains(result.Analysis, "Fact Check") {
		t.Error("expected fact check result even with unverified/empty grounding")
	}
}

func TestPipelineResearchToQualityGroundingFlow(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	// Simulate: research stage produces grounding metadata
	researchResult, err := pe.ExecuteStage(ctx, StageResearchGen, PipelineInput{
		Topic: "AI Technology",
	})
	if err != nil {
		t.Fatalf("ExecuteStage research failed: %v", err)
	}
	if researchResult.GroundingMetadata == nil {
		t.Fatal("expected grounding metadata from research stage")
	}
	if len(researchResult.GroundingMetadata.Sources) == 0 {
		t.Fatal("expected at least one source from research stage")
	}

	// Propagate grounding metadata from research result to quality stage input
	qualityResult, err := pe.ExecuteStage(ctx, StageQualityCheck, PipelineInput{
		Briefing:          "This is a mock source snippet for testing. Artificial intelligence is transforming industries worldwide.",
		Keywords:          []string{"AI"},
		Language:          "en",
		GroundingMetadata: researchResult.GroundingMetadata,
	})
	if err != nil {
		t.Fatalf("ExecuteStage quality with propagated grounding failed: %v", err)
	}
	if qualityResult.Content == "" {
		t.Error("expected non-empty quality result")
	}
	if !strings.Contains(qualityResult.Analysis, "Fact Check") {
		t.Error("expected fact check when research grounding metadata propagated to quality")
	}
}

func TestPipelineFullWithGroundingPropagation(t *testing.T) {
	pe := setupPipelineTest(t)
	ctx := context.Background()

	results, err := pe.ExecuteFull(ctx, PipelineInput{
		Title:       "Grounded Full Pipeline",
		ContentType: "article",
		Language:    "en",
		Topic:       "AI Technology",
		Briefing:    "This is a mock source snippet for testing.",
		Keywords:    []string{"AI"},
		WordCount:   300,
		Tone:        "professional",
		Audience:    "general",
	})
	if err != nil {
		t.Fatalf("ExecuteFull failed: %v", err)
	}

	researchResult, ok := results[StageResearchGen]
	if !ok {
		t.Fatal("expected research stage result")
	}
	if researchResult.GroundingMetadata == nil {
		t.Fatal("expected grounding metadata in research stage of full pipeline")
	}
	if len(researchResult.GroundingMetadata.Sources) == 0 {
		t.Error("expected at least one grounding source")
	}
	if researchResult.GroundingMetadata.Unverified {
		t.Error("expected verified research result")
	}

	qualityResult, ok := results[StageQualityCheck]
	if !ok {
		t.Fatal("expected quality stage result")
	}
	if qualityResult.Content == "" {
		t.Error("expected non-empty quality content")
	}
}

func TestPipelineQualityFactCheckSupportedClaim(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	now := time.Now()
	gm := &GroundingMetadata{
		Sources: []GroundingSource{
			{
				URI:           "https://example.com/source1",
				Title:         "AI Research",
				Snippet:       "Artificial intelligence is transforming industries worldwide. Machine learning enables computers to learn from data.",
				RetrievedAt:   now,
				FreshnessScore: 0.95,
				IsVerified:    true,
			},
		},
		Unverified: false,
	}

	text := "Artificial intelligence is transforming industries worldwide. Machine learning enables computers to learn from data."
	report, err := qc.CheckHallucinationWithGrounding(ctx, text, nil, gm)
	if err != nil {
		t.Fatalf("CheckHallucinationWithGrounding failed: %v", err)
	}
	if !report.Grounded {
		t.Error("expected grounded report when sources provided")
	}
	if report.ClaimsChecked > 0 && report.Supported == 0 {
		t.Error("expected supported claims when text matches source content")
	}
}

func TestPipelineQualityFactCheckUnsupportedClaim(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	gm := &GroundingMetadata{
		Sources: []GroundingSource{
			{
				URI:           "https://example.com/source1",
				Title:         "Physics Research",
				Snippet:       "Quantum mechanics describes the behavior of particles at the atomic scale. The theory of relativity explains gravity.",
				IsVerified:    true,
			},
		},
		Unverified: false,
	}

	text := "The stock market reached an all-time high this year. Cryptocurrency prices surged dramatically."
	report, err := qc.CheckHallucinationWithGrounding(ctx, text, nil, gm)
	if err != nil {
		t.Fatalf("CheckHallucinationWithGrounding failed: %v", err)
	}
	if !report.Grounded {
		t.Error("expected grounded report")
	}
	if report.ClaimsChecked > 0 && report.Supported == report.ClaimsChecked {
		t.Error("expected some unsupported claims when text does not match sources")
	}
}

func TestPipelineQualityFactCheckWithoutGrounding(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	report, err := qc.CheckHallucinationWithGrounding(ctx, "Some content with no sources.", nil, nil)
	if err != nil {
		t.Fatalf("CheckHallucinationWithGrounding failed: %v", err)
	}
	if report.Grounded {
		t.Error("expected non-grounded report when no sources provided")
	}
	if report.ClaimsChecked != 0 {
		t.Errorf("expected zero claims checked with no sources, got %d", report.ClaimsChecked)
	}
}

func TestPipelineQualityFactCheckMultipleSources(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	now := time.Now()
	gm := &GroundingMetadata{
		Sources: []GroundingSource{
			{
				URI:           "https://example.com/ai",
				Title:         "AI News",
				Snippet:       "Artificial intelligence is transforming healthcare through improved diagnostics and personalized medicine.",
				RetrievedAt:   now,
				FreshnessScore: 0.98,
				IsVerified:    true,
			},
			{
				URI:           "https://example.com/tech",
				Title:         "Tech Trends",
				Snippet:       "Cloud computing continues to grow as enterprises migrate their infrastructure to the cloud.",
				RetrievedAt:   now,
				FreshnessScore: 0.90,
				IsVerified:    true,
			},
			{
				URI:           "https://example.com/science",
				Title:         "Science Daily",
				Snippet:       "Climate change research shows rising global temperatures and extreme weather events.",
				RetrievedAt:   now,
				FreshnessScore: 0.85,
				IsVerified:    false,
			},
		},
		Unverified: false,
	}

	text := "Artificial intelligence is transforming healthcare through improved diagnostics. Cloud computing continues to grow as enterprises migrate their infrastructure."
	report, err := qc.CheckHallucinationWithGrounding(ctx, text, nil, gm)
	if err != nil {
		t.Fatalf("CheckHallucinationWithGrounding failed: %v", err)
	}
	if !report.Grounded {
		t.Error("expected grounded report with multiple sources")
	}
	if report.ClaimsChecked > 0 && report.Supported == 0 {
		t.Error("expected supported claims from multiple sources")
	}
	if report.GroundingMeta == nil {
		t.Error("expected grounding meta preserved in report")
	}
	if len(report.GroundingMeta.Sources) != 3 {
		t.Errorf("expected 3 sources in report, got %d", len(report.GroundingMeta.Sources))
	}
}

func TestPipelineQualityFactCheckAIManagerNotRequired(t *testing.T) {
	// The quality checker works independently of AI manager
	qc := NewQualityChecker()
	ctx := context.Background()

	now := time.Now()
	gm := &GroundingMetadata{
		Sources: []GroundingSource{
			{
				URI:           "https://example.com/source",
				Title:         "Test Source",
				Snippet:       "Grounding works independently of AI provider.",
				RetrievedAt:   now,
				IsVerified:    true,
			},
		},
	}

	text := "Grounding works independently of AI provider."
	report, err := qc.CheckHallucinationWithGrounding(ctx, text, nil, gm)
	if err != nil {
		t.Fatalf("CheckHallucinationWithGrounding failed: %v", err)
	}
	if !report.Grounded {
		t.Error("expected grounded report")
	}
	if report.ClaimsChecked > 0 && report.Supported == 0 {
		t.Error("expected supported claims")
	}
}

func TestPipelineQualityFactCheckWithAIManagerNil(t *testing.T) {
	// Simulate AI unavailable fallback
	result := &CompletionResult{
		Content:      "Research result with no AI",
		FinishReason: "unavailable",
		GroundingMetadata: &GroundingMetadata{
			Unverified: true,
		},
	}
	if result.GroundingMetadata == nil {
		t.Fatal("expected grounding metadata")
	}
	if !result.GroundingMetadata.Unverified {
		t.Error("expected unverified flag for AI-unavailable fallback")
	}

	// Quality should still work with unverified metadata
	qc := NewQualityChecker()
	ctx := context.Background()

	report, err := qc.CheckHallucinationWithGrounding(ctx, "Some content.", nil, result.GroundingMetadata)
	if err != nil {
		t.Fatalf("CheckHallucinationWithGrounding failed: %v", err)
	}
	if report.Grounded {
		t.Error("expected non-grounded report with unverified metadata and no sources")
	}
}

func TestPipelineInputGroundingMetadataField(t *testing.T) {
	// Verify that PipelineInput.GroundingMetadata is properly serialized/deserialized
	now := time.Now()
	input := PipelineInput{
		Title:    "Test",
		Topic:    "AI",
		Language: "en",
		GroundingMetadata: &GroundingMetadata{
			Sources: []GroundingSource{
				{
					URI:           "https://example.com/test",
					Title:         "Test Source",
					Snippet:       "Test snippet.",
					RetrievedAt:   now,
					IsVerified:    true,
				},
			},
			SearchSuggested: true,
			Unverified:      false,
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal PipelineInput failed: %v", err)
	}

	var decoded PipelineInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal PipelineInput failed: %v", err)
	}

	if decoded.GroundingMetadata == nil {
		t.Fatal("expected GroundingMetadata after round-trip")
	}
	if len(decoded.GroundingMetadata.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(decoded.GroundingMetadata.Sources))
	}
	if decoded.GroundingMetadata.Sources[0].URI != "https://example.com/test" {
		t.Errorf("expected URI, got %s", decoded.GroundingMetadata.Sources[0].URI)
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
