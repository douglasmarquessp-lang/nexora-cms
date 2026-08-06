package research

import (
	"context"
	"strings"
	"testing"
)

func TestExtractFactBase_Versions(t *testing.T) {
	facts := ExtractFactBase("GPT-6", []SourceText{
		{Title: "GPT-6 release", Snippet: "OpenAI released GPT-6 API v2.1.0 with v1.5.0-beta support.", URL: "https://openai.com/blog"},
	})
	versions := factsOfType(facts, FactTypeVersion)
	if len(versions) == 0 {
		t.Fatalf("expected version facts, got %+v", facts)
	}
	got := map[string]bool{}
	for _, f := range versions {
		got[strings.ToLower(f.Value)] = true
	}
	if !got["v2.1.0"] || !got["v1.5.0-beta"] {
		t.Errorf("expected v2.1.0 and v1.5.0-beta, got %v", got)
	}
	if versions[0].Confidence != 80 {
		t.Errorf("version confidence should be 80, got %d", versions[0].Confidence)
	}
}

func TestExtractFactBase_VersionDedup(t *testing.T) {
	facts := ExtractFactBase("T", []SourceText{
		{Title: "A", Snippet: "Version 1.2.3 is out. Version 1.2.3 again."},
	})
	n := len(factsOfType(facts, FactTypeVersion))
	if n != 1 {
		t.Errorf("expected 1 version fact after dedup, got %d", n)
	}
}

func TestExtractFactBase_Prices(t *testing.T) {
	facts := ExtractFactBase("T", []SourceText{
		{Title: "Pricing", Snippet: "The Pro plan costs US$ 19.99 per month.", URL: "https://example.com"},
	})
	prices := factsOfType(facts, FactTypePrice)
	if len(prices) == 0 {
		t.Fatalf("expected price facts, got %+v", facts)
	}
	if prices[0].Value != "US$ 19.99" {
		t.Errorf("expected 'US$ 19.99', got %q", prices[0].Value)
	}
	if prices[0].SourceURL != "https://example.com" {
		t.Error("price fact must carry the source URL")
	}
}

func TestExtractFactBase_PricesPT(t *testing.T) {
	facts := ExtractFactBase("T", []SourceText{
		{Title: "Preço", Snippet: "O plano custa R$ 99,90 por mês."},
	})
	if len(factsOfType(facts, FactTypePrice)) == 0 {
		t.Fatalf("expected PT price fact, got %+v", facts)
	}
}

func TestExtractFactBase_Dates(t *testing.T) {
	facts := ExtractFactBase("T", []SourceText{
		{Title: "Timeline", Snippet: "Announced on 2026-03-12 and launched on March 15, 2026."},
	})
	dates := factsOfType(facts, FactTypeDate)
	if len(dates) < 2 {
		t.Fatalf("expected ISO + long dates, got %+v", facts)
	}
}

func TestExtractFactBase_DatesPT(t *testing.T) {
	facts := ExtractFactBase("T", []SourceText{
		{Title: "Datas", Snippet: "O produto foi lançado em 15 de março de 2026."},
	})
	if len(factsOfType(facts, FactTypeDate)) == 0 {
		t.Fatalf("expected PT long date, got %+v", facts)
	}
}

func TestExtractFactBase_Companies(t *testing.T) {
	facts := ExtractFactBase("T", []SourceText{
		{Title: "News", Snippet: "OpenAI and NVIDIA announced a partnership with Meta."},
	})
	companies := factsOfType(facts, FactTypeCompany)
	got := map[string]bool{}
	for _, f := range companies {
		got[strings.ToLower(f.Entity)] = true
	}
	if !got["openai"] || !got["nvidia"] {
		t.Errorf("expected openai and nvidia companies, got %v", got)
	}
}

func TestExtractFactBase_Technologies(t *testing.T) {
	facts := ExtractFactBase("T", []SourceText{
		{Title: "Tech", Snippet: "The new model uses GPT and GPU acceleration."},
	})
	techs := factsOfType(facts, FactTypeTechnology)
	if len(techs) == 0 {
		t.Fatalf("expected technology facts, got %+v", facts)
	}
}

func TestExtractFactBase_Numbers(t *testing.T) {
	facts := ExtractFactBase("T", []SourceText{
		{Title: "Stats", Snippet: "The service reached 10 million users and 85% uptime in 2026."},
	})
	numbers := factsOfType(facts, FactTypeNumber)
	if len(numbers) == 0 {
		t.Fatalf("expected number facts, got %+v", facts)
	}
	found := false
	for _, f := range numbers {
		if strings.Contains(f.Value, "million") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a million-scaled number, got %+v", numbers)
	}
}

func TestExtractFactBase_Events(t *testing.T) {
	facts := ExtractFactBase("T", []SourceText{
		{Title: "News", Snippet: "OpenAI launched a new product at the event."},
	})
	if len(factsOfType(facts, FactTypeEvent)) == 0 {
		t.Fatalf("expected event facts, got %+v", facts)
	}
}

func TestExtractFactBase_EmptySources(t *testing.T) {
	if facts := ExtractFactBase("T", nil); len(facts) != 0 {
		t.Errorf("expected no facts for empty sources, got %+v", facts)
	}
	if facts := ExtractFactBase("T", []SourceText{{Title: "", Snippet: ""}}); len(facts) != 0 {
		t.Errorf("expected no facts for empty text, got %+v", facts)
	}
}

func TestExtractFactBase_Deterministic(t *testing.T) {
	sources := []SourceText{{Title: "A", Snippet: "Version 2.0 costs US$ 10."}}
	first := ExtractFactBase("T", sources)
	second := ExtractFactBase("T", sources)
	if len(first) != len(second) {
		t.Fatal("deterministic extraction must produce identical counts")
	}
	for i := range first {
		if first[i].Value != second[i].Value || first[i].FactType != second[i].FactType {
			t.Fatalf("extraction not deterministic: %+v vs %+v", first[i], second[i])
		}
	}
}

func TestExtractFactBase_SnippetPreferredOverTitle(t *testing.T) {
	facts := ExtractFactBase("T", []SourceText{
		{Title: "No version here", Snippet: "Version 9.9.9 shipped"},
	})
	if len(factsOfType(facts, FactTypeVersion)) == 0 {
		t.Error("snippet should be the extraction source")
	}
}

func TestExtractFactBaseAI_NilManagerFallsBack(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	sources := []SourceText{{Title: "A", Snippet: "Version 3.2.1 announced"}}
	facts := svc.ExtractFactBaseAI(context.Background(), "T", sources)
	if len(factsOfType(facts, FactTypeVersion)) == 0 {
		t.Error("nil AI manager must fall back to deterministic extraction")
	}
}

func TestExtractFactBaseAI_UnparsableJSONFallsBack(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	svc.aiManager = newAIManagerReturning("Mock response for: fact base")
	sources := []SourceText{{Title: "A", Snippet: "Version 4.0 released"}}
	facts := svc.ExtractFactBaseAI(context.Background(), "T", sources)
	if len(factsOfType(facts, FactTypeVersion)) == 0 {
		t.Error("unparsable AI output must fall back to deterministic extraction")
	}
}

func TestExtractFactBaseAI_ValidJSON(t *testing.T) {
	svc, mock := setupMockDB(t)
	defer mock.Close()

	svc.aiManager = newAIManagerReturning(`Here is the result:
[{"type":"company","entity":"Acme Corp","value":"Acme","source":"https://acme.com","confidence":90}]`)
	facts := svc.ExtractFactBaseAI(context.Background(), "T", nil)
	if len(facts) != 1 {
		t.Fatalf("expected 1 AI fact, got %+v", facts)
	}
	if facts[0].FactType != FactTypeCompany || facts[0].Confidence != 90 {
		t.Errorf("AI fact mismatch: %+v", facts[0])
	}
}

func TestExtractFactBaseAI_InvalidTypesDropped(t *testing.T) {
	svc, _ := setupMockDB(t)
	svc.aiManager = newAIManagerReturning(`[{"type":"banana","entity":"X","value":"Y","confidence":80},{"type":"date","entity":"E","value":"2026-01-01","confidence":70}]`)
	facts := svc.ExtractFactBaseAI(context.Background(), "T", nil)
	if len(facts) != 1 || facts[0].FactType != FactTypeDate {
		t.Errorf("invalid types must be dropped: %+v", facts)
	}
}

func TestExtractFactBaseAI_ConfidenceClamped(t *testing.T) {
	svc, _ := setupMockDB(t)
	svc.aiManager = newAIManagerReturning(`[{"type":"version","entity":"App","value":"1.0","confidence":500}]`)
	facts := svc.ExtractFactBaseAI(context.Background(), "T", nil)
	if len(facts) != 1 || facts[0].Confidence != 50 {
		t.Errorf("out-of-range confidence must clamp to 50: %+v", facts)
	}
}

func TestParseJSONArray(t *testing.T) {
	var out []map[string]interface{}
	if err := parseJSONArray(`prefix [{"a":1}] suffix`, &out); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 element, got %d", len(out))
	}
	if err := parseJSONArray("no brackets", &out); err == nil {
		t.Error("expected error for missing array")
	}
}

func TestBuildSourceCorpus(t *testing.T) {
	out := buildSourceCorpus([]SourceText{
		{Title: "T1", Snippet: "S1"},
		{Title: "T2", Snippet: "S2"},
	})
	for _, want := range []string{"[1] T1", "S1", "[2] T2", "S2"} {
		if !strings.Contains(out, want) {
			t.Errorf("corpus missing %q: %s", want, out)
		}
	}
}

func TestValidFactType(t *testing.T) {
	for _, ft := range []FactType{FactTypeCompany, FactTypeProduct, FactTypeVersion, FactTypePrice,
		FactTypeDate, FactTypeEvent, FactTypeTechnology, FactTypeNumber} {
		if !validFactType(ft) {
			t.Errorf("%s should be valid", ft)
		}
	}
	if validFactType(FactType("nope")) {
		t.Error("nope should be invalid")
	}
}

func factsOfType(facts []FactBaseEntry, ft FactType) []FactBaseEntry {
	var out []FactBaseEntry
	for _, f := range facts {
		if f.FactType == ft {
			out = append(out, f)
		}
	}
	return out
}
