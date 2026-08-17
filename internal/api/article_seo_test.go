package api

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	publisherModule "nexora/internal/modules/publisher"
)

func pubFixture() *publisherModule.Publication {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	return &publisherModule.Publication{
		ID:           uuid.New(),
		SiteID:       uuid.New(),
		Title:        "Guia de Inteligência Artificial",
		Slug:         "guia-ia",
		URL:          "https://meusite.com/guia-ia",
		CanonicalURL: "https://meusite.com/guia-ia",
		Language:     "pt",
		MultilingualURLs: map[string]interface{}{
			"pt": "https://meusite.com/guia-ia",
			"en": "https://meusite.com/en/guia-ia",
		},
		Status:          publisherModule.PubStatusPublished,
		MetaTitle:       "Guia de IA",
		MetaDescription: "Guia completo sobre IA.",
		OgImage:         "https://meusite.com/img.jpg",
		PublishedAt:     &now,
		UpdatedAt:       now,
		Categories:      []string{"Tecnologia"},
		Content:         "Conteúdo do guia.",
	}
}

func TestBuildArticleSEO_Hreflang(t *testing.T) {
	pub := pubFixture()
	seo := buildArticleSEO(pub, "https://meusite.com")
	if len(seo.Hreflang) != 2 {
		t.Errorf("expected 2 hreflang entries (self + en), got %d", len(seo.Hreflang))
	}
	foundEN := false
	for _, h := range seo.Hreflang {
		if h.Lang == "en" && h.URL == "https://meusite.com/en/guia-ia" {
			foundEN = true
		}
	}
	if !foundEN {
		t.Errorf("expected English alternate, got %v", seo.Hreflang)
	}
}

func TestBuildArticleSEO_SchemaScripts(t *testing.T) {
	pub := pubFixture()
	pub.Content = "## Perguntas frequentes\nQ: O que é?\nA: Uma plataforma.\nPasso a passo:\n1. Primeiro passo.\n2. Segundo passo."
	seo := buildArticleSEO(pub, "https://meusite.com")
	if len(seo.SchemaJSONLD) == 0 {
		t.Fatal("expected at least the Article schema")
	}
	joined := strings.Join(seo.SchemaJSONLD, " ")
	if !strings.Contains(joined, "Article") {
		t.Errorf("expected Article schema, got %s", joined)
	}
	if !strings.Contains(joined, "BreadcrumbList") {
		t.Errorf("expected Breadcrumb schema, got %s", joined)
	}
	// FAQ only when Q/A present.
	if !strings.Contains(joined, "FAQPage") {
		t.Errorf("expected FAQ schema, got %s", joined)
	}
	if !strings.Contains(joined, "HowTo") {
		t.Errorf("expected HowTo schema, got %s", joined)
	}
}

func TestBuildArticleSEO_NoFAQWithoutPairs(t *testing.T) {
	pub := pubFixture()
	seo := buildArticleSEO(pub, "https://meusite.com")
	joined := strings.Join(seo.SchemaJSONLD, " ")
	if strings.Contains(joined, "FAQPage") {
		t.Errorf("expected no FAQ schema without Q/A pairs, got %s", joined)
	}
}

func TestBuildArticleSEO_TwitterAndOG(t *testing.T) {
	pub := pubFixture()
	seo := buildArticleSEO(pub, "https://meusite.com")
	if seo.OgType != "article" {
		t.Errorf("expected og:type article, got %q", seo.OgType)
	}
	if seo.TwitterCard != "summary_large_image" {
		t.Errorf("expected summary_large_image, got %q", seo.TwitterCard)
	}
}

func TestBuildArticleSEO_SiteSchemas(t *testing.T) {
	pub := pubFixture()
	seo := buildArticleSEO(pub, "https://meusite.com")
	if len(seo.SiteSchemaJSONLD) != 2 {
		t.Errorf("expected Organization + WebSite schemas, got %d: %v", len(seo.SiteSchemaJSONLD), seo.SiteSchemaJSONLD)
	}
	joined := strings.Join(seo.SiteSchemaJSONLD, " ")
	if !strings.Contains(joined, "Organization") || !strings.Contains(joined, "WebSite") {
		t.Errorf("expected Organization and WebSite schemas, got %s", joined)
	}
}

func TestIsNewsArticle(t *testing.T) {
	if !isNewsArticle([]string{"Notícias"}) {
		t.Error("expected PT 'Notícias' category to be news")
	}
	if !isNewsArticle([]string{"News"}) {
		t.Error("expected EN 'News' category to be news")
	}
	if isNewsArticle([]string{"Tutorial"}) {
		t.Error("expected tutorial NOT to be news")
	}
}

func TestExtractFAQPairs(t *testing.T) {
	content := "Q: O que é?\nA: É uma plataforma.\nP: Quem usa?\nR: Empresas."
	pairs := extractFAQPairs(content)
	if len(pairs) != 2 {
		t.Errorf("expected 2 FAQ pairs, got %d", len(pairs))
	}
	if pairs[0].Question != "O que é?" || pairs[0].Answer != "É uma plataforma." {
		t.Errorf("unexpected first pair: %+v", pairs[0])
	}
}

func TestExtractHowToSteps(t *testing.T) {
	steps := extractHowToSteps("Antes de começar.\n1. Baixe o arquivo.\n2. Execute o instalador.\nConclusão.")
	if len(steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Title != "Baixe o arquivo." {
		t.Errorf("unexpected first step: %+v", steps[0])
	}
}

func TestDeriveSiteDomain(t *testing.T) {
	if d := deriveSiteDomain("https://meusite.com/guia-ia", ""); d != "https://meusite.com" {
		t.Errorf("unexpected domain: %q", d)
	}
	if d := deriveSiteDomain("http://sub.example.org/path"); d != "http://sub.example.org" {
		t.Errorf("unexpected domain: %q", d)
	}
	if d := deriveSiteDomain(""); d != "" {
		t.Errorf("expected empty domain (no placeholder fallback), got %q", d)
	}
	if d := deriveSiteDomain("https://example.com/x"); d != "" {
		t.Errorf("expected empty domain for example.com, got %q", d)
	}
	if d := deriveSiteDomain("https://sub.example.com/x"); d != "" {
		t.Errorf("expected empty domain for subdomain of example.com, got %q", d)
	}
	if d := deriveSiteDomain("not a url", "https://meusite.com"); d != "https://meusite.com" {
		t.Errorf("expected real domain to win, got %q", d)
	}
}

func TestBuildArticleSEO_SuppressesPlaceholderURLs(t *testing.T) {
	// A legacy publication still carrying an example.com URL must never emit
	// it into hreflang or schema output.
	pub := pubFixture()
	pub.URL = "https://example.com/guia-ia"
	pub.CanonicalURL = "https://example.com/guia-ia"
	pub.MultilingualURLs = map[string]interface{}{
		"pt": "https://example.com/guia-ia",
		"en": "https://example.com/en/guia-ia",
	}

	seo := buildArticleSEO(pub, "")
	if len(seo.Hreflang) != 0 {
		t.Errorf("expected no hreflang for placeholder URLs, got %v", seo.Hreflang)
	}
	joined := strings.Join(append(seo.SchemaJSONLD, seo.SiteSchemaJSONLD...), " ")
	if strings.Contains(joined, "example.com") {
		t.Errorf("placeholder domain leaked into schemas: %s", joined)
	}
	if len(seo.SiteSchemaJSONLD) != 0 {
		t.Errorf("expected no site schemas without a resolved domain, got %v", seo.SiteSchemaJSONLD)
	}
}

func TestBuildArticleSEO_SuppressesPlaceholderSiteDomain(t *testing.T) {
	// Even if a caller passes an example.com site domain, it must not reach
	// the breadcrumb/site schemas.
	pub := pubFixture()
	seo := buildArticleSEO(pub, "https://example.com")
	joined := strings.Join(append(seo.SchemaJSONLD, seo.SiteSchemaJSONLD...), " ")
	if strings.Contains(joined, "example.com") {
		t.Errorf("placeholder site domain leaked into schemas: %s", joined)
	}
}

func TestSiteNameFromDomain(t *testing.T) {
	if n := siteNameFromDomain("https://meusite.com"); n != "Meusite" {
		t.Errorf("unexpected site name: %q", n)
	}
	if n := siteNameFromDomain(""); n != "Nexora" {
		t.Errorf("expected Nexora fallback, got %q", n)
	}
}
