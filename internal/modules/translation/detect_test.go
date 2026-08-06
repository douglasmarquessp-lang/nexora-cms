package translation

import (
	"strings"
	"testing"
)

func TestDetectLanguage_Portuguese(t *testing.T) {
	text := "O guia completo para criar conteúdo de qualidade para o seu site e alcançar mais leitores com estratégias simples."
	lang, conf := DetectLanguage(text)
	if lang != "pt" {
		t.Errorf("expected pt, got %s", lang)
	}
	if conf <= 0 || conf > 1 {
		t.Errorf("confidence out of range: %f", conf)
	}
}

func TestDetectLanguage_English(t *testing.T) {
	text := "The complete guide to creating quality content for your website and reaching more readers with simple strategies."
	lang, _ := DetectLanguage(text)
	if lang != "en" {
		t.Errorf("expected en, got %s", lang)
	}
}

func TestDetectLanguage_Empty(t *testing.T) {
	lang, conf := DetectLanguage("")
	if lang == "" {
		t.Error("expected a default language")
	}
	if conf != 0.5 {
		t.Errorf("expected neutral confidence 0.5, got %f", conf)
	}
}

func TestDetectLanguage_ShortEnglish(t *testing.T) {
	lang, _ := DetectLanguage("This is a test")
	if lang != "en" {
		t.Errorf("expected en, got %s", lang)
	}
}

func TestDetectLanguage_Deterministic(t *testing.T) {
	text := "O sistema é simples e funciona muito bem para todos os usuários da plataforma."
	l1, c1 := DetectLanguage(text)
	l2, c2 := DetectLanguage(text)
	if l1 != l2 || c1 != c2 {
		t.Errorf("detection not deterministic: (%s,%f) vs (%s,%f)", l1, c1, l2, c2)
	}
}

func TestGenerateSlug_Portuguese(t *testing.T) {
	slug := GenerateSlug("Guia Completo de Marketing de Conteúdo em 2026!", "pt")
	if slug != "guia-completo-de-marketing-de-conteudo-em-2026" {
		t.Errorf("unexpected slug: %s", slug)
	}
}

func TestGenerateSlug_English(t *testing.T) {
	slug := GenerateSlug("How to Build a Website in 2026", "en")
	if slug != "how-to-build-a-website-in-2026" {
		t.Errorf("unexpected slug: %s", slug)
	}
}

func TestGenerateSlug_Empty(t *testing.T) {
	slug := GenerateSlug("!!!", "en")
	if slug == "" {
		t.Error("expected fallback slug for empty title")
	}
}

func TestGenerateSlug_Diagonals(t *testing.T) {
	slug := GenerateSlug("Ação Coração Avião", "pt")
	if strings.Contains(slug, "ã") || strings.Contains(slug, "ç") {
		t.Errorf("slug contains diacritics: %s", slug)
	}
}

func TestDeriveKeyword(t *testing.T) {
	kw := DeriveKeyword("Guia Completo de Marketing Digital", "pt")
	if kw != "marketing" {
		t.Errorf("expected marketing, got %s", kw)
	}
	kw = DeriveKeyword("Complete Guide to Digital Marketing", "en")
	if kw != "marketing" {
		t.Errorf("expected marketing, got %s", kw)
	}
}

func TestDeriveKeyword_AllStopWords(t *testing.T) {
	kw := DeriveKeyword("The and for with", "en")
	if kw == "" {
		t.Error("expected a fallback keyword")
	}
}

func TestBlocksToText(t *testing.T) {
	blocks := []interface{}{
		map[string]interface{}{"type": "heading", "text": "Introdução"},
		map[string]interface{}{"type": "text", "text": "Primeiro parágrafo."},
		map[string]interface{}{"type": "image", "url": "x.png"},
		map[string]interface{}{"type": "quote", "text": "não renderiza"},
	}
	text := blocksToText(blocks)
	if !strings.Contains(text, "# Introdução") {
		t.Errorf("heading missing: %s", text)
	}
	if !strings.Contains(text, "Primeiro parágrafo.") {
		t.Errorf("paragraph missing: %s", text)
	}
	if strings.Contains(text, "não renderiza") {
		t.Error("non text/heading block should be skipped")
	}
}

func TestTextToBlocks(t *testing.T) {
	text := "# Title\n\nParagraph one.\n\n## Subheading\n\nParagraph two."
	blocks := textToBlocks(text)
	if len(blocks) != 4 {
		t.Fatalf("expected 4 blocks, got %d", len(blocks))
	}
	headings := 0
	for _, b := range blocks {
		m := b.(map[string]interface{})
		if m["type"] == "heading" {
			headings++
		}
	}
	if headings != 2 {
		t.Errorf("expected 2 heading blocks, got %d", headings)
	}
}

func TestTextToBlocks_Empty(t *testing.T) {
	blocks := textToBlocks("")
	if len(blocks) == 0 {
		t.Error("expected at least a placeholder block")
	}
}

func TestBlocksRoundTrip(t *testing.T) {
	original := "# Heading\n\nBody text here."
	blocks := textToBlocks(original)
	text := blocksToText(blocks)
	if !strings.Contains(text, "Heading") || !strings.Contains(text, "Body text here.") {
		t.Errorf("round trip lost content: %s", text)
	}
}

func TestClampAndRound(t *testing.T) {
	if clampScore(-5) != 0 || clampScore(150) != 100 || clampScore(50) != 50 {
		t.Error("clampScore broken")
	}
	if round2(1.234) != 1.23 || round2(1.236) != 1.24 {
		t.Errorf("round2 broken: %f %f", round2(1.234), round2(1.236))
	}
	if clamp01(-1) != 0 || clamp01(2) != 1 {
		t.Error("clamp01 broken")
	}
}

func TestNormalizeDiacritics(t *testing.T) {
	norm := normalizeDiacritics("São Paulo Ação")
	if norm != "sao paulo acao" {
		t.Errorf("unexpected normalization: %s", norm)
	}
}
