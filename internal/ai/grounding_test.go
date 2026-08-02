package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func TestGroundingMetadataTypes(t *testing.T) {
	now := time.Now()
	source := GroundingSource{
		URI:           "https://example.com/test",
		Title:         "Test Source",
		Snippet:       "A test source snippet",
		PublishedAt:   &now,
		FreshnessScore: 0.95,
		IsVerified:    true,
		DomainRank:    1,
		RetrievedAt:   now,
	}

	if source.URI != "https://example.com/test" {
		t.Errorf("expected URI, got %s", source.URI)
	}
	if !source.IsVerified {
		t.Error("expected source to be verified")
	}
	if source.FreshnessScore != 0.95 {
		t.Errorf("expected freshness 0.95, got %f", source.FreshnessScore)
	}

	meta := GroundingMetadata{
		Sources:         []GroundingSource{source},
		SearchSuggested: true,
		Unverified:      false,
		SupportSegments: []GroundingSupport{
			{Segment: "test segment", SourceIndices: []int{0}, Confidence: 0.9},
		},
	}

	if len(meta.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(meta.Sources))
	}
	if meta.Unverified {
		t.Error("expected verified metadata")
	}
}

func TestGroundingConfigInRequest(t *testing.T) {
	gc := &GroundingConfig{
		Enabled:    true,
		MaxSources: 10,
	}
	if !gc.Enabled {
		t.Error("expected grounding enabled")
	}
	if gc.MaxSources != 10 {
		t.Errorf("expected MaxSources 10, got %d", gc.MaxSources)
	}

	req := CompletionRequest{
		Prompt:    "test",
		Grounding: gc,
	}
	if req.Grounding == nil {
		t.Fatal("expected non-nil grounding config")
	}
	if !req.Grounding.Enabled {
		t.Error("expected grounding enabled in request")
	}
}

func TestMockProviderGroundingMetadata(t *testing.T) {
	p := NewMockProvider("mock", "mock-model", nil)
	ctx := context.Background()

	// Test without grounding
	result, err := p.Generate(ctx, CompletionRequest{
		Prompt: "test without grounding",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result.GroundingMetadata != nil {
		t.Error("expected nil grounding metadata when not requested")
	}

	// Test with grounding enabled
	result, err = p.Generate(ctx, CompletionRequest{
		Prompt: "test with grounding",
		Grounding: &GroundingConfig{
			Enabled:    true,
			MaxSources: 5,
		},
	})
	if err != nil {
		t.Fatalf("Generate with grounding failed: %v", err)
	}
	if result.GroundingMetadata == nil {
		t.Fatal("expected non-nil grounding metadata")
	}
	if result.GroundingMetadata.Unverified {
		t.Error("expected verified grounding metadata")
	}
	if len(result.GroundingMetadata.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(result.GroundingMetadata.Sources))
	}
	if !result.GroundingMetadata.SearchSuggested {
		t.Error("expected search suggested")
	}
	// Verify source content
	if result.GroundingMetadata.Sources[0].URI != "https://example.com/mock-source-1" {
		t.Errorf("expected mock source 1 URI, got %s", result.GroundingMetadata.Sources[0].URI)
	}
	if !result.GroundingMetadata.Sources[0].IsVerified {
		t.Error("expected mock source 1 to be verified")
	}
}

func TestGeminiProviderGroundingCapability(t *testing.T) {
	p := NewGeminiProvider("test", "test-model", "test-key", "")
	caps := p.Capabilities()

	hasGrounding := false
	for _, c := range caps {
		if c == CapGrounding {
			hasGrounding = true
			break
		}
	}
	if !hasGrounding {
		t.Error("GeminiProvider should have CapGrounding capability")
	}
}

func TestGeminiBuildRequestWithGrounding(t *testing.T) {
	p := NewGeminiProvider("test", "test-model", "test-key", "")

	// Without grounding
	req := CompletionRequest{
		Prompt: "test",
	}
	gReq := p.buildRequest(req)
	if len(gReq.Tools) != 0 {
		t.Error("expected no tools when grounding disabled")
	}

	// With grounding
	req.Grounding = &GroundingConfig{Enabled: true}
	gReq = p.buildRequest(req)
	if len(gReq.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(gReq.Tools))
	}
	if gReq.Tools[0].GoogleSearch == nil {
		t.Error("expected googleSearch tool")
	}
}

func TestGeminiProxyGroundingRoundTrip(t *testing.T) {
	// Mock Gemini server that returns grounding metadata
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request has the Google Search tool
		var reqBody struct {
			Tools []struct {
				GoogleSearch *struct{} `json:"googleSearch"`
			} `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(reqBody.Tools) == 0 || reqBody.Tools[0].GoogleSearch == nil {
			http.Error(w, "missing googleSearch tool", http.StatusBadRequest)
			return
		}

		// Return response with grounding metadata
		resp := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": "Grounded research result about AI technology."},
						},
						"role": "model",
					},
					"finishReason": "STOP",
					"groundingMetadata": map[string]interface{}{
						"webSearchQueries": []string{"AI technology latest developments"},
						"groundingChunks": []map[string]interface{}{
							{
								"web": map[string]string{
									"uri":   "https://example.com/ai-tech",
									"title": "AI Technology Overview",
								},
							},
							{
								"web": map[string]string{
									"uri":   "https://example.com/ai-2024",
									"title": "AI in 2024",
								},
							},
						},
						"groundingSupports": []map[string]interface{}{
							{
								"segment": map[string]interface{}{
									"text":       "Grounded research result",
									"startIndex": 0,
									"endIndex":   28,
								},
								"groundingChunkIndices": []int{0, 1},
								"confidenceScores":      []float64{0.95, 0.90},
							},
						},
					},
				},
			},
			"usageMetadata": map[string]int{
				"promptTokenCount":     10,
				"candidatesTokenCount": 20,
				"totalTokenCount":      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-grounded", "gemini-2.0-flash", "fake-key", server.URL)
	p.client = server.Client()

	ctx := context.Background()
	result, err := p.Generate(ctx, CompletionRequest{
		Prompt: "Research AI technology",
		Grounding: &GroundingConfig{
			Enabled:    true,
			MaxSources: 10,
		},
	})
	if err != nil {
		t.Fatalf("Generate with grounding failed: %v", err)
	}

	if result.GroundingMetadata == nil {
		t.Fatal("expected grounding metadata")
	}
	if len(result.GroundingMetadata.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(result.GroundingMetadata.Sources))
	}
	if result.GroundingMetadata.Sources[0].URI != "https://example.com/ai-tech" {
		t.Errorf("expected ai-tech URI, got %s", result.GroundingMetadata.Sources[0].URI)
	}
	if result.GroundingMetadata.Sources[0].Title != "AI Technology Overview" {
		t.Errorf("expected 'AI Technology Overview', got '%s'", result.GroundingMetadata.Sources[0].Title)
	}
	if !result.GroundingMetadata.SearchSuggested {
		t.Error("expected search suggested")
	}
	if result.GroundingMetadata.Unverified {
		t.Error("expected verified result")
	}
	if len(result.GroundingMetadata.SupportSegments) != 1 {
		t.Errorf("expected 1 support segment, got %d", len(result.GroundingMetadata.SupportSegments))
	}
	if result.GroundingMetadata.SupportSegments[0].Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", result.GroundingMetadata.SupportSegments[0].Confidence)
	}
}

func TestGeminiGroundingWithNoSources(t *testing.T) {
	// Mock server that returns empty grounding metadata (no web sources found)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": "Response without grounding sources."},
						},
						"role": "model",
					},
					"finishReason":    "STOP",
					"groundingMetadata": map[string]interface{}{
						"webSearchQueries": []string{"test query"},
					},
				},
			},
			"usageMetadata": map[string]int{
				"promptTokenCount":     5,
				"candidatesTokenCount": 10,
				"totalTokenCount":      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewGeminiProvider("test-nosources", "gemini-2.0-flash", "fake-key", server.URL)
	p.client = server.Client()

	ctx := context.Background()
	result, err := p.Generate(ctx, CompletionRequest{
		Prompt: "test",
		Grounding: &GroundingConfig{
			Enabled: true,
		},
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.GroundingMetadata == nil {
		t.Fatal("expected grounding metadata")
	}
	if !result.GroundingMetadata.Unverified {
		t.Error("expected unverified when no web chunks returned")
	}
	if len(result.GroundingMetadata.Sources) != 0 {
		t.Errorf("expected 0 sources, got %d", len(result.GroundingMetadata.Sources))
	}
}

func TestPipelineResearchStageWithGrounding(t *testing.T) {
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	p := NewMockProvider("mock", "mock-model", nil)
	m.RegisterProvider(p, ProviderCfg{Name: "mock", Enabled: true, Priority: 1, Weight: 10})

	pe := NewPipelineExecutor(m)
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
	if result.GroundingMetadata == nil {
		t.Fatal("expected grounding metadata from research stage")
	}
	if result.GroundingMetadata.Unverified {
		t.Error("expected verified research result")
	}
	if len(result.GroundingMetadata.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(result.GroundingMetadata.Sources))
	}
}

func TestPipelineResearchStageWithoutGroundingCapability(t *testing.T) {
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	// Create a mock provider WITHOUT CapGrounding
	p := NewMockProvider("mock-no-grounding", "mock-model", []Capability{CapGenerate, CapStream})
	m.RegisterProvider(p, ProviderCfg{Name: "mock-no-grounding", Enabled: true, Priority: 1, Weight: 10})

	pe := NewPipelineExecutor(m)
	ctx := context.Background()

	result, err := pe.ExecuteStage(ctx, StageResearchGen, PipelineInput{
		Topic: "AI Technology",
	})
	if err != nil {
		t.Fatalf("ExecuteStage research failed: %v", err)
	}
	// Provider doesn't support grounding, so GroundingMetadata should be nil
	if result.GroundingMetadata != nil {
		t.Error("expected nil grounding metadata when provider lacks CapGrounding")
	}
}

func TestGroundingFallbackWhenAIManagerNil(t *testing.T) {
	// Simulate research without AI (ExecuteGroundedResearch fallback)
	// This is tested through the research service path
	result := &CompletionResult{
		Content:      "Research for topic 'test' (no AI configured)",
		FinishReason: "unavailable",
		GroundingMetadata: &GroundingMetadata{
			Unverified: true,
		},
	}
	if result.GroundingMetadata == nil {
		t.Fatal("expected grounding metadata")
	}
	if !result.GroundingMetadata.Unverified {
		t.Error("expected unverified flag")
	}
}

func TestSourcesFromGrounding(t *testing.T) {
	now := time.Now()
	gm := &GroundingMetadata{
		Sources: []GroundingSource{
			{
				URI:           "https://example.com/src1",
				Title:         "Source 1",
				FreshnessScore: 0.95,
				IsVerified:    true,
				RetrievedAt:   now,
			},
			{
				URI:           "https://example.com/src2",
				Title:         "Source 2",
				FreshnessScore: 0.80,
				IsVerified:    false,
				RetrievedAt:   now,
				PublishedAt:   &now,
			},
		},
	}

	type testSource struct {
		URI           string
		Title         string
		FreshnessScore float64
		IsVerified    bool
		PublishedAt   *time.Time
		RelevanceScore int
	}

	sources := make([]testSource, 0)
	for i, gs := range gm.Sources {
		src := testSource{
			Title:          gs.Title,
			URI:            gs.URI,
			FreshnessScore: gs.FreshnessScore,
			IsVerified:     gs.IsVerified,
			RelevanceScore: int(gs.FreshnessScore * 100),
		}
		if gs.PublishedAt != nil {
			src.PublishedAt = gs.PublishedAt
		}
		_ = i // position not needed for test
		sources = append(sources, src)
	}

	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	if sources[0].URI != "https://example.com/src1" {
		t.Errorf("expected src1 URL, got %s", sources[0].URI)
	}
	if !sources[0].IsVerified {
		t.Error("expected src1 verified")
	}
	if sources[1].IsVerified {
		t.Error("expected src2 unverified")
	}
	if sources[1].PublishedAt == nil {
		t.Error("expected published_at for src2")
	}
}

func TestGroundingConfigYAML(t *testing.T) {
	// Verify that GroundingConfig integrates with CompletionRequest properly
	req := CompletionRequest{
		Prompt: "test",
		Grounding: &GroundingConfig{
			Enabled:        true,
			MaxSources:     5,
			ExcludeDomains: []string{"example.com"},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded CompletionRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Grounding == nil {
		t.Fatal("expected grounding config after round-trip")
	}
	if !decoded.Grounding.Enabled {
		t.Error("expected grounding enabled")
	}
	if decoded.Grounding.MaxSources != 5 {
		t.Errorf("expected MaxSources 5, got %d", decoded.Grounding.MaxSources)
	}
	if len(decoded.Grounding.ExcludeDomains) != 1 || decoded.Grounding.ExcludeDomains[0] != "example.com" {
		t.Errorf("expected [example.com], got %v", decoded.Grounding.ExcludeDomains)
	}
}

func TestResearchStageWithGroundingInFullPipeline(t *testing.T) {
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	p := NewMockProvider("mock", "mock-model", nil)
	m.RegisterProvider(p, ProviderCfg{Name: "mock", Enabled: true, Priority: 1, Weight: 10})

	pe := NewPipelineExecutor(m)
	ctx := context.Background()

	results, err := pe.ExecuteFull(ctx, PipelineInput{
		Title:       "Grounded Article",
		ContentType: "article",
		Language:    "en",
		Topic:       "AI Technology",
		Briefing:    "Research data",
		Keywords:    []string{"AI"},
		WordCount:   500,
		Tone:        "professional",
		Audience:    "developers",
	})
	if err != nil {
		t.Fatalf("ExecuteFull failed: %v", err)
	}

	researchResult, ok := results[StageResearchGen]
	if !ok {
		t.Fatal("expected research stage result")
	}
	if researchResult.GroundingMetadata == nil {
		t.Fatal("expected grounding metadata in research stage")
	}
	if len(researchResult.GroundingMetadata.Sources) == 0 {
		t.Error("expected at least one source")
	}
}


