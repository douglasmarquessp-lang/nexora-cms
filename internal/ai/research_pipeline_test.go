package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestPipelineResearchStage_Preloaded(t *testing.T) {
	pe := setupPipelineTest(t)
	summary := &ResearchSummary{
		Topic:    "GPT-6",
		Language: "en",
		Briefing: "Preloaded briefing",
		Facts: []ResearchFact{
			{Type: "version", Entity: "GPT-6", Value: "2.0", Source: "openai.com", Confidence: 80},
		},
		Sources: []ResearchSourceSummary{
			{Title: "OpenAI Blog", URL: "https://openai.com", Domain: "openai.com", ReliabilityScore: 100, ReliabilityLabel: "verified"},
		},
	}

	result, err := pe.ExecuteStage(context.Background(), StageResearchGen, PipelineInput{
		Topic:    "GPT-6",
		Language: "en",
		Research: summary,
	})
	if err != nil {
		t.Fatalf("research stage failed: %v", err)
	}
	if result.Research != summary {
		t.Error("expected the preloaded summary to be carried through")
	}
	if !strings.Contains(result.Content, "Research Summary") || !strings.Contains(result.Content, "GPT-6") {
		t.Errorf("content missing summary render: %q", result.Content)
	}
}

func TestPipelineResearchStage_ResearchFn(t *testing.T) {
	pe := setupPipelineTest(t)
	called := false
	fn := func(ctx context.Context, topic, language string) (*ResearchSummary, error) {
		called = true
		return &ResearchSummary{Topic: topic, Language: language, Briefing: "from fn"}, nil
	}

	result, err := pe.ExecuteStage(context.Background(), StageResearchGen, PipelineInput{
		Topic:      "AI",
		Language:   "pt",
		ResearchFn: fn,
	})
	if err != nil {
		t.Fatalf("research stage failed: %v", err)
	}
	if !called {
		t.Error("ResearchFn was not called")
	}
	if result.Research == nil || result.Research.Briefing != "from fn" {
		t.Errorf("expected fn summary, got %+v", result.Research)
	}
}

func TestPipelineResearchStage_ResearchFnError(t *testing.T) {
	pe := setupPipelineTest(t)
	want := errors.New("research down")
	fn := func(ctx context.Context, topic, language string) (*ResearchSummary, error) {
		return nil, want
	}

	_, err := pe.ExecuteStage(context.Background(), StageResearchGen, PipelineInput{
		Topic:      "AI",
		Language:   "en",
		ResearchFn: fn,
	})
	if !errors.Is(err, want) {
		t.Errorf("expected wrapped error, got %v", err)
	}
}

func TestPipelineResearchStage_ResearchFnNilFallsBack(t *testing.T) {
	pe := setupPipelineTest(t)
	fn := func(ctx context.Context, topic, language string) (*ResearchSummary, error) {
		return nil, nil
	}

	result, err := pe.ExecuteStage(context.Background(), StageResearchGen, PipelineInput{
		Topic:      "AI",
		Language:   "en",
		ResearchFn: fn,
	})
	if err != nil {
		t.Fatalf("research stage failed: %v", err)
	}
	// Mock provider returns its standard (non-JSON) response text.
	if result.Content == "" {
		t.Error("expected fallback grounding-only content")
	}
}

func TestPipelineResearchStage_PriorityPreloadedOverFn(t *testing.T) {
	pe := setupPipelineTest(t)
	fn := func(ctx context.Context, topic, language string) (*ResearchSummary, error) {
		t.Fatal("ResearchFn must not be called when Research is preloaded")
		return nil, nil
	}

	_, err := pe.ExecuteStage(context.Background(), StageResearchGen, PipelineInput{
		Topic:      "AI",
		Language:   "en",
		Research:   &ResearchSummary{Topic: "AI", Language: "en"},
		ResearchFn: fn,
	})
	if err != nil {
		t.Fatalf("research stage failed: %v", err)
	}
}

func TestSummaryFromGrounding(t *testing.T) {
	gm := &GroundingMetadata{
		Sources: []GroundingSource{
			{URI: "https://www.openai.com/blog", Title: "OpenAI Blog"},
			{URI: "https://unknown-blog.xyz/post", Title: "Random Blog"},
		},
	}
	s := summaryFromGrounding("AI", "en", gm)
	if s == nil {
		t.Fatal("expected non-nil summary")
	}
	if len(s.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(s.Sources))
	}
	if s.Sources[0].Domain != "openai.com" || s.Sources[0].ReliabilityScore != 100 {
		t.Errorf("openai source not scored: %+v", s.Sources[0])
	}
	if s.Sources[1].ReliabilityScore != 30 {
		t.Errorf("unknown source should score 30: %+v", s.Sources[1])
	}
	if !strings.Contains(s.Briefing, "OpenAI Blog") {
		t.Errorf("briefing missing titles: %q", s.Briefing)
	}
}

func TestSummaryFromGrounding_NilSources(t *testing.T) {
	if s := summaryFromGrounding("AI", "en", nil); s != nil {
		t.Errorf("expected nil summary, got %+v", s)
	}
	if s := summaryFromGrounding("AI", "en", &GroundingMetadata{}); s != nil {
		t.Errorf("expected nil summary for empty metadata, got %+v", s)
	}
}

func TestFormatResearchSummary(t *testing.T) {
	s := &ResearchSummary{
		Topic:    "T",
		Language: "en",
		Cached:   true,
		Briefing: "B",
		Facts:    []ResearchFact{{Type: "number", Entity: "users", Value: "10M"}},
		Sources:  []ResearchSourceSummary{{Title: "S", URL: "https://s.com", Domain: "s.com", ReliabilityScore: 90, ReliabilityLabel: "verified"}},
	}
	out := formatResearchSummary(s)
	for _, want := range []string{"Research Summary", "Topic: T", "Cached: true", "Briefing:", "Fact Base:", "users: 10M", "Sources:", "reliability 90 (verified)"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if formatResearchSummary(nil) != "" {
		t.Error("nil summary should render empty")
	}
}

func TestResearchContext(t *testing.T) {
	input := PipelineInput{Research: &ResearchSummary{
		Facts: []ResearchFact{
			{Type: "date", Entity: "launch", Value: "2026-01-15"},
		},
	}}
	out := researchContext(input)
	if !strings.Contains(out, "Fact Base") || !strings.Contains(out, "2026-01-15") {
		t.Errorf("fact base not rendered: %q", out)
	}
	if researchContext(PipelineInput{}) != "" {
		t.Error("empty input must render empty context")
	}
	if researchContext(PipelineInput{Research: &ResearchSummary{}}) != "" {
		t.Error("research without facts must render empty context")
	}
}

func TestResearchFactsInjectedIntoBriefing(t *testing.T) {
	pe := setupPipelineTest(t)
	input := PipelineInput{
		Topic:   "AI",
		Briefing: "base",
		Research: &ResearchSummary{
			Facts: []ResearchFact{{Type: "price", Entity: "API", Value: "US$ 5"}},
		},
	}
	result, err := pe.ExecuteStage(context.Background(), StageBriefingGen, input)
	if err != nil {
		t.Fatalf("briefing stage failed: %v", err)
	}
	// Mock provider echoes the prompt variables; the fact base must be present.
	if !strings.Contains(result.Content, "US$ 5") {
		t.Errorf("fact base missing from briefing prompt output: %q", result.Content)
	}
}

func TestResearchFactsInjectedIntoDraft(t *testing.T) {
	pe := setupPipelineTest(t)
	input := PipelineInput{
		Title:   "AI Overview",
		Topic:   "AI",
		Content: "draft seed",
		Research: &ResearchSummary{
			Facts: []ResearchFact{{Type: "number", Entity: "users", Value: "2 billion"}},
		},
	}
	result, err := pe.ExecuteStage(context.Background(), StageDraftGen, input)
	if err != nil {
		t.Fatalf("draft stage failed: %v", err)
	}
	if !strings.Contains(result.Content, "2 billion") {
		t.Errorf("fact base missing from draft prompt output: %q", result.Content)
	}
}
