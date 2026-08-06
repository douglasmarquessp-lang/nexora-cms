package translation

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func term(source, target string, forbidden bool) GlossaryTerm {
	return GlossaryTerm{
		ID:             uuid.New(),
		SourceTerm:     source,
		TargetTerm:     target,
		SourceLanguage: "pt",
		TargetLanguage: "en",
		Forbidden:      forbidden,
	}
}

func TestApplyGlossary_Basic(t *testing.T) {
	text := "O fluxo de trabalho com IA e SEO é importante."
	terms := []GlossaryTerm{term("IA", "Artificial Intelligence", false), term("SEO", "SEO", false)}
	out, res := ApplyGlossary(text, terms, "pt", "en")
	if !strings.Contains(out, "Artificial Intelligence") {
		t.Errorf("glossary not applied: %s", out)
	}
	if res.Applied != 2 {
		t.Errorf("expected 2 applied, got %d", res.Applied)
	}
	if res.TargetHits != 2 {
		t.Errorf("expected 2 target hits, got %d", res.TargetHits)
	}
}

func termDir(source, target, fromLang, toLang string, forbidden bool) GlossaryTerm {
	return GlossaryTerm{
		ID:             uuid.New(),
		SourceTerm:     source,
		TargetTerm:     target,
		SourceLanguage: fromLang,
		TargetLanguage: toLang,
		Forbidden:      forbidden,
	}
}

func TestApplyGlossary_ForbiddenTermProtected(t *testing.T) {
	text := "O workflow da equipe continua o mesmo."
	terms := []GlossaryTerm{termDir("workflow", "fluxo de trabalho", "en", "pt", true)}
	out, res := ApplyGlossary(text, terms, "en", "pt")
	if !strings.Contains(out, "workflow") {
		t.Errorf("forbidden term was replaced: %s", out)
	}
	if res.Protected != 1 {
		t.Errorf("expected 1 protected, got %d", res.Protected)
	}
	if res.Applied != 0 {
		t.Errorf("expected 0 applied, got %d", res.Applied)
	}
}

func TestApplyGlossary_TermNotPresentInText(t *testing.T) {
	text := "The AI workflow is efficient."
	terms := []GlossaryTerm{term("IA", "Artificial Intelligence", false)}
	out, res := ApplyGlossary(text, terms, "pt", "en")
	if res.Applicable != 1 {
		t.Errorf("expected 1 applicable term, got %d", res.Applicable)
	}
	if res.Applied != 0 {
		t.Errorf("source term not in text: expected 0 applied, got %d", res.Applied)
	}
	if out != text {
		t.Error("text should be unchanged when source term is absent")
	}
}

func TestApplyGlossary_WordBoundary(t *testing.T) {
	text := "IA is amazing, and so is the IA-based approach."
	terms := []GlossaryTerm{term("IA", "AI", false)}
	out, _ := ApplyGlossary(text, terms, "pt", "en")
	if strings.Contains(out, "IA-based") {
		t.Errorf("substring match happened inside another word: %s", out)
	}
}

func TestApplyGlossary_CaseInsensitive(t *testing.T) {
	text := "use seo to rank better, SEO matters."
	terms := []GlossaryTerm{term("SEO", "Search Engine Optimization", false)}
	out, _ := ApplyGlossary(text, terms, "pt", "en")
	if strings.Contains(out, "use seo ") || strings.Contains(out, " SEO ") {
		t.Errorf("case-insensitive replacement failed: %s", out)
	}
}

func TestGlossaryConsistency(t *testing.T) {
	res := GlossaryApplyResult{Applicable: 10, Protected: 4, TargetHits: 6}
	if got := GlossaryConsistency(res); got != 100 {
		t.Errorf("expected 100 consistency (6/6), got %f", got)
	}
	res = GlossaryApplyResult{Applicable: 10, Protected: 0, TargetHits: 5}
	if got := GlossaryConsistency(res); got != 50 {
		t.Errorf("expected 50 consistency, got %f", got)
	}
	res = GlossaryApplyResult{Applicable: 0, Protected: 0, TargetHits: 0}
	if got := GlossaryConsistency(res); got != 100 {
		t.Errorf("expected 100 for empty glossary, got %f", got)
	}
}

func TestApplyGlossary_MultiWordTerm(t *testing.T) {
	text := "O e-mail marketing funciona bem."
	terms := []GlossaryTerm{term("e-mail marketing", "email campaign", false)}
	out, res := ApplyGlossary(text, terms, "pt", "en")
	if !strings.Contains(out, "email campaign") {
		t.Errorf("multi-word term not applied: %s", out)
	}
	if res.Applied != 1 {
		t.Errorf("expected 1 applied, got %d", res.Applied)
	}
}

func TestNormalizeTerm(t *testing.T) {
	if normalizeTerm("  AI  ") != "AI" {
		t.Error("normalizeTerm failed")
	}
}
