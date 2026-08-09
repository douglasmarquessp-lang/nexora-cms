package seoengine

import (
	"context"
	"strings"
	"testing"

	"nexora/internal/modules/publisher"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func TestBuildMetaDescription_Deterministic(t *testing.T) {
	title := "Guia de Marketing de Conteúdo"
	content := "Marketing de conteúdo é o tema central. Segundo parágrafo com mais detalhes práticos e exemplos reais de uso em empresas pequenas e médias, sempre com foco no resultado."
	a := buildMetaDescription(title, content, "marketing", "pt")
	b := buildMetaDescription(title, content, "marketing", "pt")
	if a != b {
		t.Errorf("expected deterministic meta description, got %q vs %q", a, b)
	}
	if a == "" {
		t.Fatal("expected a derived meta description")
	}
	if !strings.Contains(strings.ToLower(a), "marketing") {
		t.Errorf("expected keyword inside the meta description, got %q", a)
	}
	runes := len([]rune(a))
	if runes > 160 {
		t.Errorf("meta description must not exceed 160 runes: %q (%d)", a, runes)
	}
}

func TestBuildMetaDescription_DifferentArticlesDiffer(t *testing.T) {
	a := buildMetaDescription("Guia", "primeiro artigo sobre tecnologia e inovação com muitos detalhes interessantes para o leitor final do site completo.", "tecnologia", "pt")
	b := buildMetaDescription("Guia", "segundo artigo sobre comida e receitas com muitos detalhes interessantes para o leitor final do site completo.", "comida", "pt")
	if a == b {
		t.Error("different articles must produce different meta descriptions")
	}
}

func TestBuildMetaDescription_EmptyInput(t *testing.T) {
	if got := buildMetaDescription("", "", "", ""); got != "" {
		t.Errorf("expected empty meta for empty input, got %q", got)
	}
}

func TestAppendExternalSources(t *testing.T) {
	links := []ExternalLinkCandidate{
		{URL: "https://example.gov.br/doc", Title: "Documentação oficial", Domain: "example.gov.br"},
		{URL: "javascript:alert(1)", Title: "evil", Domain: "evil.example"},
		{URL: "https://reuters.com/story", Title: "", Domain: "reuters.com"},
	}
	out := appendExternalSources("Base content.", links, "pt")
	if !strings.HasPrefix(out, "Base content.") {
		t.Error("original content must be preserved at the beginning")
	}
	if !strings.Contains(out, "## Fontes") {
		t.Error("expected PT Fontes section")
	}
	if !strings.Contains(out, "[Documentação oficial](https://example.gov.br/doc)") {
		t.Error("expected reliable link appended")
	}
	if strings.Contains(out, "javascript:") {
		t.Error("javascript: URLs must never be appended")
	}
	if !strings.Contains(out, "[reuters.com]") {
		t.Error("empty title must fall back to the domain")
	}

	en := appendExternalSources("Base.", links, "en")
	if !strings.Contains(en, "## Sources") {
		t.Error("expected EN Sources section")
	}

	noop := appendExternalSources("X.", nil, "pt")
	if noop != "X." {
		t.Errorf("no links must return content unchanged, got %q", noop)
	}
}

func TestHasExternalLinks(t *testing.T) {
	if !hasExternalLinks("see [docs](https://example.org/doc) here") {
		t.Error("markdown https link must count")
	}
	if !hasExternalLinks(`<a href="https://example.org">x</a>`) {
		t.Error("html <a href> must count")
	}
	if hasExternalLinks("only [relative](/internal) and ![img](/img.png)") {
		t.Error("relative links must not count as external")
	}
}

func TestEnhanceBeforePublish_MetaDescriptionGenerated(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	out, err := svc.EnhanceBeforePublish(context.Background(), publisher.ContentEnhancerInput{
		Title:    "Guia de Marketing de Conteúdo",
		Content:  "Marketing de conteúdo é a prática de criar e distribuir material relevante para atrair e reter um público-alvo ao longo do tempo.",
		Language: "pt",
	})
	if err != nil {
		t.Fatalf("EnhanceBeforePublish failed: %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil enhancement")
	}
	if out.MetaDescription == "" {
		t.Error("expected a generated meta description")
	}
	if out.Content == "" {
		t.Error("content must be preserved")
	}
}

var _ = logger.New
