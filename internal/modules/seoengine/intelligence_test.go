package seoengine

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAnalyzeEEAT_EmptyContent(t *testing.T) {
	r := AnalyzeEEAT(ArticleAnalysisInput{Content: "", Language: "pt"})
	if r.Final >= 20 {
		t.Errorf("expected near-0 score for empty content, got %f", r.Final)
	}
	if len(r.Pillars) != 4 {
		t.Errorf("expected 4 pillars, got %d", len(r.Pillars))
	}
	for _, p := range r.Pillars {
		if len(p.Issues) == 0 {
			t.Errorf("pillar %s should have issues for empty content", p.Name)
		}
	}
}

func TestAnalyzeEEAT_WeightsSumToOne(t *testing.T) {
	sum := eeatWeightExperience + eeatWeightExpertise + eeatWeightAuthority + eeatWeightTrustworthiness
	if sum != 1.0 {
		t.Errorf("weights must sum to 1.0, got %f", sum)
	}
}

func TestAnalyzeEEAT_Deterministic(t *testing.T) {
	in := ArticleAnalysisInput{
		Content:  "Escrito por Ana Souza, especialista em segurança.\nEm nossos testes medimos 99% de eficácia.\nEstudo de caso real documentado.\nFontes: relatório anual de 2026.",
		Language: "pt",
	}
	a := AnalyzeEEAT(in)
	b := AnalyzeEEAT(in)
	if a.Final != b.Final || a.Experience != b.Experience || a.Expertise != b.Expertise {
		t.Errorf("non-deterministic result: %v vs %v", a, b)
	}
}

func TestAnalyzeEEAT_AuthorNameBoosts(t *testing.T) {
	withAuthor := AnalyzeEEAT(ArticleAnalysisInput{
		Content:    "Análise técnica detalhada com dados de 2026 e fontes oficiais verificadas.",
		Language:   "pt",
		AuthorName: "Carlos Mendes",
	})
	withoutAuthor := AnalyzeEEAT(ArticleAnalysisInput{
		Content:  "Análise técnica detalhada com dados de 2026 e fontes oficiais verificadas.",
		Language: "pt",
	})
	if withAuthor.Trustworthiness <= withoutAuthor.Trustworthiness {
		t.Errorf("expected author name to boost trust, got %f vs %f", withAuthor.Trustworthiness, withoutAuthor.Trustworthiness)
	}
	if withAuthor.Expertise <= withoutAuthor.Expertise {
		t.Errorf("expected author name to boost expertise, got %f vs %f", withAuthor.Expertise, withoutAuthor.Expertise)
	}
}

func TestAnalyzeEEAT_JSONLDSignal(t *testing.T) {
	r := AnalyzeEEAT(ArticleAnalysisInput{
		Content:  `<script type="application/ld+json">{"@context":"https://schema.org"}</script> texto com ano 2026 e citação de estudo`,
		Language: "en",
	})
	if r.Authoritativeness < 50 {
		t.Errorf("expected JSON-LD + year + citation to boost authority, got %f", r.Authoritativeness)
	}
}

func TestAnalyzeEEAT_StrongExternalAuthority(t *testing.T) {
	r := AnalyzeEEAT(ArticleAnalysisInput{
		Content:  "Veja [estudo do governo](https://www.gov.br/pesquisa). Segundo o estudo, em 2026 os dados confirmam.",
		Language: "pt",
	})
	if r.Authoritativeness < 60 {
		t.Errorf("expected gov source to boost authority, got %f", r.Authoritativeness)
	}
}

func TestAnalyzeEEAT_CompetitorDomainNotAuthority(t *testing.T) {
	// The deterministic analyzer counts reliability >= 75; a competitor-ish
	// domain (low reliability) must not count as authority.
	r := AnalyzeEEAT(ArticleAnalysisInput{
		Content:  "Veja [link](https://someunknownblog123.example.org/post) com texto de 2026.",
		Language: "en",
	})
	if r.Authoritativeness > 60 {
		t.Errorf("expected low authority without reliable sources, got %f", r.Authoritativeness)
	}
}

func TestDetectContentGaps_FullCoverage(t *testing.T) {
	text := "O produto custa R$ 99 e já está disponível para compra em 2026. " +
		"Requisitos: 4GB de RAM. Limitações: sem suporte offline. " +
		"Roadmap: versão 2.0 planejada. Compare com alternativas. " +
		"Instalação passo a passo no guia oficial, com suporte e documentação."
	r := DetectContentGaps(text)
	if r.MissingCount != 0 {
		t.Errorf("expected no missing gaps, got %v", r.Missing)
	}
	if r.CoverageRatio != 1.0 {
		t.Errorf("expected coverage 1.0, got %f", r.CoverageRatio)
	}
}

func TestDetectContentGaps_AllMissing(t *testing.T) {
	r := DetectContentGaps("apenas um texto curto sem detalhes")
	if r.MissingCount != len(gapDimensions) {
		t.Errorf("expected %d missing, got %d", len(gapDimensions), r.MissingCount)
	}
	if len(r.Suggestions) != r.MissingCount {
		t.Errorf("expected a suggestion per gap, got %d suggestions for %d gaps", len(r.Suggestions), r.MissingCount)
	}
}

func TestDetectContentGaps_Deterministic(t *testing.T) {
	text := "Preço a partir de R$ 50, requisitos mínimos listados na documentação."
	a := DetectContentGaps(text)
	b := DetectContentGaps(text)
	if len(a.Missing) != len(b.Missing) {
		t.Errorf("non-deterministic gap detection")
	}
}

func TestDetectContentGaps_PriceDetected(t *testing.T) {
	r := DetectContentGaps("O custo é de R$ 120 por mês.")
	found := false
	for _, c := range r.Covered {
		if c == "price" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected price gap covered, covered=%v", r.Covered)
	}
}

func TestTopicAuthorityScore_Scale(t *testing.T) {
	if TopicAuthorityScore(0) != 0 {
		t.Errorf("expected 0 for no articles")
	}
	one := TopicAuthorityScore(1)
	if one < 10 || one > 30 {
		t.Errorf("expected ~18 for 1 article, got %f", one)
	}
	three := TopicAuthorityScore(3)
	if three <= one {
		t.Errorf("expected 3 articles to score higher than 1, got %f vs %f", three, one)
	}
	if TopicAuthorityScore(50) != 100 {
		t.Errorf("expected 100 at 50 articles")
	}
	if TopicAuthorityScore(51) != 100 {
		t.Errorf("expected 100 beyond 50 articles")
	}
}

func TestTopicAuthorityScore_Deterministic(t *testing.T) {
	if TopicAuthorityScore(10) != TopicAuthorityScore(10) {
		t.Error("non-deterministic authority score")
	}
}

func TestFillTopicAuthorityGap(t *testing.T) {
	gaps := FillTopicAuthorityGap("inteligência artificial generativa para marketing", []string{"Guia de marketing digital", "IA no marketing"})
	if len(gaps) == 0 {
		t.Errorf("expected gaps for uncovered terms, got none")
	}
	full := FillTopicAuthorityGap("marketing", []string{"Marketing completo 2026"})
	for _, g := range full {
		if strings.Contains("marketing", g) {
			t.Errorf("expected no gap for covered term, got %q", g)
		}
	}
}

func TestScoreInternalLinkCandidate_Keyword(t *testing.T) {
	c := ScoreInternalLinkCandidate(
		"Guia completo de inteligência artificial", "Conteúdo sobre IA generativa", "inteligência artificial", "Tecnologia",
		"Inteligência artificial explicada", "ia-explicada", "Tecnologia",
	)
	if c.Keyword < 50 {
		t.Errorf("expected strong keyword score, got %f", c.Keyword)
	}
	if c.Relevance < 50 {
		t.Errorf("expected strong composite relevance, got %f", c.Relevance)
	}
}

func TestScoreInternalLinkCandidate_Unrelated(t *testing.T) {
	c := ScoreInternalLinkCandidate(
		"Receita de bolo", "Ingredientes e modo de preparo", "bolo", "Culinária",
		"Contrato de aluguel", "contrato-aluguel", "Imóveis",
	)
	if c.Relevance > 40 {
		t.Errorf("expected low relevance for unrelated topics, got %f", c.Relevance)
	}
}

func TestScoreInternalLinkCandidate_CategoryBoost(t *testing.T) {
	noCat := ScoreInternalLinkCandidate("Guia de SEO", "Conteúdo sobre SEO técnico", "seo", "Tecnologia", "SEO para iniciantes", "seo-iniciantes", "")
	withCat := ScoreInternalLinkCandidate("Guia de SEO", "Conteúdo sobre SEO técnico", "seo", "Tecnologia", "SEO para iniciantes", "seo-iniciantes", "Tecnologia")
	if withCat.Relevance <= noCat.Relevance {
		t.Errorf("expected category match to boost relevance, got %f vs %f", withCat.Relevance, noCat.Relevance)
	}
}

func TestScoreInternalLinkCandidate_IntentMatch(t *testing.T) {
	informational := ScoreInternalLinkCandidate("Como configurar servidor", "Guia passo a passo de configuração", "servidor", "", "Como comprar um servidor barato", "comprar-servidor", "")
	matched := ScoreInternalLinkCandidate("Como configurar servidor", "Guia passo a passo de configuração", "servidor", "", "Como otimizar servidores", "otimizar-servidor", "")
	if informational.Intent != 0 {
		t.Errorf("expected intent mismatch (how-to vs buy), got %f", informational.Intent)
	}
	if matched.Intent < 50 {
		t.Errorf("expected intent match (how-to vs how-to), got %f", matched.Intent)
	}
}

func TestDetectIntent(t *testing.T) {
	if detectIntent("Como fazer um bolo") != "informational" {
		t.Error("expected informational intent")
	}
	if detectIntent("Comprar iPhone 15 barato") != "transactional" {
		t.Error("expected transactional intent")
	}
	if detectIntent("Acessar painel de login") != "navigational" {
		t.Error("expected navigational intent")
	}
}

func TestIsCompetitorDomain(t *testing.T) {
	s := &Service{competitorDomains: []string{"rival.com"}}
	if !s.isCompetitorDomain("rival.com") {
		t.Error("expected exact match to be competitor")
	}
	if !s.isCompetitorDomain("blog.rival.com") {
		t.Error("expected subdomain to be competitor")
	}
	if s.isCompetitorDomain("rival.com.br") {
		t.Error("expected different domain NOT to be competitor")
	}
	if s.isCompetitorDomain("other.com") {
		t.Error("expected unrelated domain not to be competitor")
	}
}

func TestAppendRelatedLinks(t *testing.T) {
	links := []InternalLinkCandidate{
		{Title: "SEO para iniciantes", Slug: "seo-iniciantes", Relevance: 88},
		{Title: "SEO técnico", Slug: "seo-tecnico", Relevance: 70},
	}
	out := appendRelatedLinks("Conteúdo original", links, "pt")
	if !strings.Contains(out, "Conteúdo original") {
		t.Error("original content must be preserved")
	}
	if !strings.Contains(out, "Leia também") {
		t.Error("expected PT related heading")
	}
	if !strings.Contains(out, "[SEO para iniciantes](/seo-iniciantes)") {
		t.Error("expected markdown link with slug")
	}
}

func TestAppendRelatedLinks_English(t *testing.T) {
	out := appendRelatedLinks("Body text", []InternalLinkCandidate{{Title: "Guide", Slug: "guide", Relevance: 90}}, "en")
	if !strings.Contains(out, "Related reading") {
		t.Error("expected EN heading")
	}
}

func TestAppendRelatedLinks_NoLinks(t *testing.T) {
	if appendRelatedLinks("texto", nil, "pt") != "texto" {
		t.Error("expected unchanged content with no links")
	}
}

func TestBuildArticleSchemaJSONLD(t *testing.T) {
	out, err := BuildArticleSchemaJSONLD("https://site.com/artigo", "Título", "Título", "desc", "https://site.com/img.jpg", "João", "Site", "", "2026-01-01", "2026-02-01", "pt")
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON-LD: %v", err)
	}
	if parsed["@type"] != "Article" {
		t.Errorf("expected Article type, got %v", parsed["@type"])
	}
	if parsed["inLanguage"] != "pt" {
		t.Errorf("expected pt language, got %v", parsed["inLanguage"])
	}
	if _, ok := parsed["author"]; !ok {
		t.Error("expected author present")
	}
}

func TestBuildNewsArticleSchema(t *testing.T) {
	out, err := BuildNewsArticleSchema("https://site.com/noticia", "Título", "desc", "img", "Ana", "NewsCo", "2026-01-01", "2026-01-02")
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON-LD: %v", err)
	}
	if parsed["@type"] != "NewsArticle" {
		t.Errorf("expected NewsArticle type, got %v", parsed["@type"])
	}
}

func TestBuildFAQSchema(t *testing.T) {
	out, err := BuildFAQSchema("https://site.com/faq", []FAQQuestion{
		{Question: "O que é?", Answer: "É uma plataforma."},
		{Question: "Custa quanto?", Answer: "R$ 99."},
	})
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON-LD: %v", err)
	}
	if parsed["@type"] != "FAQPage" {
		t.Errorf("expected FAQPage type, got %v", parsed["@type"])
	}
}

func TestBuildFAQSchema_Empty(t *testing.T) {
	out, err := BuildFAQSchema("https://site.com/faq", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Errorf("expected empty output for no FAQs, got %s", out)
	}
}

func TestBuildHowToSchema(t *testing.T) {
	out, err := BuildHowToSchema("Como instalar", "Passo a passo", []HowToStep{
		{Title: "Baixe o instalador", Text: "Acesse o site oficial."},
		{Title: "Execute", Text: "Siga o assistente."},
	})
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON-LD: %v", err)
	}
	if parsed["@type"] != "HowTo" {
		t.Errorf("expected HowTo type, got %v", parsed["@type"])
	}
}

func TestBuildBreadcrumbSchema(t *testing.T) {
	out, err := BuildBreadcrumbSchema([]BreadcrumbItem{
		{Name: "Home", URL: "https://site.com/"},
		{Name: "Guia", URL: "https://site.com/guia"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "BreadcrumbList") {
		t.Error("expected BreadcrumbList schema")
	}
}

func TestBuildOrganizationSchema(t *testing.T) {
	out, err := BuildOrganizationSchema("Meu Site", "https://site.com/", "https://site.com/logo.png", "Desc", []string{"https://x.com/mysite"})
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON-LD: %v", err)
	}
	if parsed["@type"] != "Organization" {
		t.Errorf("expected Organization type, got %v", parsed["@type"])
	}
	if _, ok := parsed["sameAs"]; !ok {
		t.Error("expected sameAs present")
	}
}

func TestBuildWebSiteSchema(t *testing.T) {
	out, err := BuildWebSiteSchema("Meu Site", "https://site.com/", "https://site.com/busca?q={search_term_string}", "pt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "WebSite") {
		t.Error("expected WebSite schema")
	}
	if !strings.Contains(out, "SearchAction") {
		t.Error("expected SearchAction when searchURL provided")
	}
}
