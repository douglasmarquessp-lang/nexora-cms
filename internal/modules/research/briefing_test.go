package research

import (
	"context"
	"encoding/json"
	"testing"
)

func TestBuildBriefing_Sections(t *testing.T) {
	sources := []ResearchSource{
		{Title: "Reuters", URL: "https://reuters.com", ReliabilityScore: 95, IsVerified: true},
		{Title: "Wiki", URL: "https://wikipedia.org", ReliabilityScore: 70},
		{Title: "OpenAI", URL: "https://openai.com", ReliabilityScore: 100},
	}
	facts := []FactBaseEntry{
		{FactType: FactTypeNumber, Entity: "users", Value: "10 million"},
		{FactType: FactTypeDate, Entity: "event", Value: "2026-03-12"},
		{FactType: FactTypeCompany, Entity: "OpenAI"},
		{FactType: FactTypeEvent, Entity: "launch", Value: "OpenAI launched a product."},
	}

	doc := BuildBriefing("AI", sources, facts)
	if doc.Topic != "AI" {
		t.Errorf("topic mismatch: %q", doc.Topic)
	}
	if len(doc.KeyPoints) == 0 {
		t.Error("expected key points")
	}
	if len(doc.Statistics) == 0 || doc.Statistics[0] == "" {
		t.Errorf("expected statistics, got %+v", doc.Statistics)
	}
	if len(doc.Dates) == 0 || doc.Dates[0] == "" {
		t.Errorf("expected dates, got %+v", doc.Dates)
	}
	if len(doc.Companies) == 0 || doc.Companies[0] != "OpenAI" {
		t.Errorf("expected companies, got %+v", doc.Companies)
	}
	if len(doc.DataFound) == 0 {
		t.Error("expected data found from events")
	}
	if len(doc.Conclusions) == 0 {
		t.Error("expected conclusions")
	}
}

func TestBuildBriefing_ConclusionsCorroborated(t *testing.T) {
	sources := []ResearchSource{
		{Title: "A", URL: "https://reuters.com", ReliabilityScore: 95},
		{Title: "B", URL: "https://apnews.com", ReliabilityScore: 95, IsVerified: true},
		{Title: "C", URL: "https://bbc.com", ReliabilityScore: 90},
	}
	doc := BuildBriefing("T", sources, nil)
	if len(doc.Conclusions) != 2 {
		t.Fatalf("expected 2 corroborated conclusions, got %+v", doc.Conclusions)
	}
	if doc.Conclusions[0] != "Findings are corroborated by multiple reliable sources (score >= 75)." {
		t.Errorf("unexpected conclusion: %q", doc.Conclusions[0])
	}
}

func TestBuildBriefing_ConclusionsPartial(t *testing.T) {
	doc := BuildBriefing("T", []ResearchSource{
		{Title: "A", URL: "https://reuters.com", ReliabilityScore: 95},
		{Title: "B", URL: "https://unknown.xyz", ReliabilityScore: 30},
	}, nil)
	if len(doc.Conclusions) != 1 || doc.Conclusions[0] != "At least one authoritative source confirms the core facts; cross-check remaining claims." {
		t.Errorf("unexpected conclusion: %+v", doc.Conclusions)
	}
}

func TestBuildBriefing_ConclusionsUnverified(t *testing.T) {
	doc := BuildBriefing("T", []ResearchSource{
		{Title: "A", URL: "https://unknown.xyz", ReliabilityScore: 30},
	}, nil)
	if len(doc.Conclusions) != 1 ||
		doc.Conclusions[0] != "No authoritative sources found — treat all claims as unverified and cross-check before publishing." {
		t.Errorf("unexpected conclusion: %+v", doc.Conclusions)
	}
}

func TestBuildBriefing_Empty(t *testing.T) {
	doc := BuildBriefing("T", nil, nil)
	if doc.Summary == "" {
		t.Error("expected a summary even with no sources")
	}
	if len(doc.KeyPoints) != 0 {
		t.Errorf("expected no key points, got %+v", doc.KeyPoints)
	}
}

func TestBuildBriefing_Deterministic(t *testing.T) {
	sources := []ResearchSource{{Title: "A", URL: "https://reuters.com", ReliabilityScore: 95}}
	facts := []FactBaseEntry{{FactType: FactTypeNumber, Entity: "X", Value: "1"}}
	a := BuildBriefing("T", sources, facts)
	b := BuildBriefing("T", sources, facts)
	if a.Summary != b.Summary || len(a.KeyPoints) != len(b.KeyPoints) {
		t.Error("briefing must be deterministic")
	}
}

func TestBuildBriefingAI_NilManagerFallsBack(t *testing.T) {
	svc, _ := setupMockDB(t)
	doc := svc.BuildBriefingAI(context.Background(), "T", "en", []ResearchSource{
		{Title: "A", URL: "https://reuters.com", ReliabilityScore: 95},
	}, nil)
	if doc.Topic != "T" || doc.Summary == "" {
		t.Errorf("nil manager must fall back to deterministic briefing: %+v", doc)
	}
}

func TestBuildBriefingAI_ValidJSON(t *testing.T) {
	svc, _ := setupMockDB(t)
	svc.aiManager = newAIManagerReturning(`{"summary":"AI summary","key_points":["P1"],"data_found":[],"statistics":["S1"],"dates":[],"companies":[],"products":[],"conclusions":["C1"]}`)
	doc := svc.BuildBriefingAI(context.Background(), "T", "en", nil, nil)
	if doc.Summary != "AI summary" {
		t.Errorf("expected AI summary, got %+v", doc)
	}
	if len(doc.KeyPoints) != 1 || doc.KeyPoints[0] != "P1" {
		t.Errorf("expected AI key points, got %+v", doc.KeyPoints)
	}
	if doc.Topic != "T" {
		t.Error("topic must be forced to the real topic")
	}
}

func TestBuildBriefingAI_UnparsableFallsBack(t *testing.T) {
	svc, _ := setupMockDB(t)
	svc.aiManager = newAIManagerReturning("Mock response for: deep_research")
	doc := svc.BuildBriefingAI(context.Background(), "T", "pt", nil, nil)
	if doc.Summary == "" {
		t.Error("unparsable AI output must fall back to deterministic briefing")
	}
}

func TestBuildBriefingAI_EmptySummaryFallsBack(t *testing.T) {
	svc, _ := setupMockDB(t)
	svc.aiManager = newAIManagerReturning(`{"summary":"","key_points":[]}`)
	doc := svc.BuildBriefingAI(context.Background(), "T", "en", nil, nil)
	if doc.Summary == "" {
		t.Error("empty AI summary must fall back to deterministic briefing")
	}
}

func TestRankSources(t *testing.T) {
	in := []ResearchSource{
		{URL: "https://low.example", ReliabilityScore: 30, RelevanceScore: 90},
		{URL: "https://high.example", ReliabilityScore: 95, RelevanceScore: 10},
		{URL: "https://high2.example", ReliabilityScore: 95, RelevanceScore: 80},
	}
	ranked := rankSources(in)
	if len(ranked) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(ranked))
	}
	if ranked[0].URL != "https://high2.example" || ranked[1].URL != "https://high.example" {
		t.Errorf("reliability+relevance order wrong: %+v", ranked)
	}
	if ranked[2].URL != "https://low.example" {
		t.Errorf("lowest reliability must rank last: %+v", ranked)
	}
}

func TestRankSources_DoesNotMutateInput(t *testing.T) {
	in := []ResearchSource{
		{URL: "https://low.example", ReliabilityScore: 30},
		{URL: "https://high.example", ReliabilityScore: 95},
	}
	ranked := rankSources(in)
	if in[0].URL != "https://low.example" {
		t.Error("rankSources must not mutate the input slice")
	}
	_ = ranked
}

func TestDedupeStrings(t *testing.T) {
	out := dedupeStrings([]string{"A", "a", " B ", "", "A"})
	if len(out) != 2 || out[0] != "A" || out[1] != " B " {
		t.Errorf("unexpected dedupe result: %+v", out)
	}
}

func TestParseJSONObject(t *testing.T) {
	var out map[string]interface{}
	if err := parseJSONObject(`prefix {"k":"v"} suffix`, &out); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if out["k"] != "v" {
		t.Errorf("unexpected parse result: %+v", out)
	}
	if err := parseJSONObject("no braces", &out); err == nil {
		t.Error("expected error for missing object")
	}
}

func TestResearchBriefingDoc_JSONRoundTrip(t *testing.T) {
	doc := ResearchBriefingDoc{
		Topic: "T", Summary: "S", KeyPoints: []string{"K1"}, Conclusions: []string{"C1"},
		Dates: []string{"2026-01-01"},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var back ResearchBriefingDoc
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if back.Topic != "T" || len(back.KeyPoints) != 1 || back.KeyPoints[0] != "K1" || len(back.Dates) != 1 {
		t.Errorf("round-trip mismatch: %+v", back)
	}
}
