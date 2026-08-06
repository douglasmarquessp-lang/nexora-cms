package seoengine

import (
	"context"
	"strings"
	"testing"
)

func testAnalyzer(t *testing.T) (*context.Context, *ArticleAnalysisInput) {
	t.Helper()
	ctx := context.Background()
	in := &ArticleAnalysisInput{
		Title:           "Guia Completo de Marketing de Conteúdo em 2026",
		MetaDescription: "Um guia completo sobre marketing de conteúdo: estratégias, ferramentas e exemplos práticos para sua empresa crescer em 2026.",
		Slug:            "guia-marketing-conteudo-2026",
		Content: `# Guia Completo de Marketing de Conteúdo

## Introdução
Marketing de conteúdo é a estratégia de criar material relevante para atrair clientes. Este guia de marketing de conteúdo mostra como começar em 2026.

## Estratégia
A estratégia de marketing de conteúdo exige planejamento. Veja as melhores práticas para criar um plano de marketing de conteúdo eficaz.

## Ferramentas
Existem diversas ferramentas de marketing de conteúdo no mercado. Escolha as melhores para o seu time e comece hoje.

## Conclusão
Marketing de conteúdo é um investimento de longo prazo. Fonte: estudo de mercado 2026, por autor especialista.`,
		Keyword:  "marketing de conteúdo",
		Language: "pt",
	}
	return &ctx, in
}

func TestAnalyzeArticle_Deterministic(t *testing.T) {
	ctx, in := testAnalyzer(t)
	a := AnalyzeArticle(*ctx, *in, nil)
	b := AnalyzeArticle(*ctx, *in, nil)
	if a.OverallScore != b.OverallScore {
		t.Errorf("expected deterministic score, got %f vs %f", a.OverallScore, b.OverallScore)
	}
}

func TestAnalyzeArticle_ScoresInRange(t *testing.T) {
	ctx, in := testAnalyzer(t)
	a := AnalyzeArticle(*ctx, *in, nil)
	for name, score := range map[string]float64{
		"overall":    a.OverallScore,
		"title":      a.TitleScore,
		"meta":       a.MetaScore,
		"headings":   a.HeadingScore,
		"keyword":    a.KeywordScore,
		"readability": a.ReadabilityScore,
		"slug":       a.SlugScore,
		"schema":     a.SchemaScore,
	} {
		if score < 0 || score > 100 {
			t.Errorf("%s score %f out of range [0,100]", name, score)
		}
	}
}

func TestAnalyzeArticle_HeadingsDetected(t *testing.T) {
	ctx, in := testAnalyzer(t)
	a := AnalyzeArticle(*ctx, *in, nil)
	if a.HeadingScore < 80 {
		t.Errorf("expected strong heading score for well-structured markdown, got %f", a.HeadingScore)
	}
}

func TestAnalyzeArticle_KeywordDensity(t *testing.T) {
	ctx, in := testAnalyzer(t)
	a := AnalyzeArticle(*ctx, *in, nil)
	if a.KeywordDensity <= 0 {
		t.Errorf("expected positive keyword density, got %f", a.KeywordDensity)
	}
}

func TestAnalyzeArticle_DensityInIdealBand(t *testing.T) {
	ctx := context.Background()
	content := ""
	for i := 0; i < 3; i++ {
		content += "Marketing de conteúdo é essencial. "
	}
	for i := 0; i < 12; i++ {
		content += "palavras comuns de preenchimento que não são a palavra chave aqui "
	}
	a := AnalyzeArticle(ctx, ArticleAnalysisInput{
		Title:    "Guia de Marketing de Conteúdo",
		Content:  content,
		Keyword:  "marketing de conteúdo",
		Language: "pt",
	}, nil)
	if a.KeywordDensity < 1 || a.KeywordDensity > 3 {
		t.Errorf("expected density in ideal band [1,3], got %f", a.KeywordDensity)
	}
	if a.KeywordScore < 70 {
		t.Errorf("expected strong keyword score with ideal density, got %f", a.KeywordScore)
	}
}

func TestAnalyzeArticle_WordCount(t *testing.T) {
	ctx, in := testAnalyzer(t)
	a := AnalyzeArticle(*ctx, *in, nil)
	if a.WordCount == 0 {
		t.Error("expected word count > 0")
	}
}

func TestAnalyzeArticle_EmptyContent(t *testing.T) {
	ctx := context.Background()
	a := AnalyzeArticle(ctx, ArticleAnalysisInput{Title: "T", Language: "pt"}, nil)
	if a.WordCount != 0 {
		t.Errorf("expected 0 words for empty content, got %d", a.WordCount)
	}
	if a.ReadabilityScore != 0 {
		t.Errorf("expected 0 readability for empty content, got %f", a.ReadabilityScore)
	}
}

func TestAnalyzeArticle_LanguageVariants(t *testing.T) {
	ctx, in := testAnalyzer(t)
	pt := AnalyzeArticle(*ctx, *in, nil)
	inEn := *in
	inEn.Language = "en"
	en := AnalyzeArticle(*ctx, inEn, nil)
	if len(pt.Suggestions) == 0 && len(en.Suggestions) == 0 {
		t.Fatal("expected suggestions for at least one language")
	}
	for _, s := range en.Suggestions {
		if s.Issue == "" || s.Suggestion == "" {
			t.Error("expected non-empty bilingual issue and suggestion")
		}
	}
}

func TestAnalyzeArticle_EnglishMessages(t *testing.T) {
	ctx := context.Background()
	a := AnalyzeArticle(ctx, ArticleAnalysisInput{Title: "X", Language: "en"}, nil)
	for _, s := range a.Suggestions {
		if s.Issue == "" {
			t.Error("expected non-empty issue in en")
		}
	}
}

func TestAnalyzeTitle_LengthBands(t *testing.T) {
	short := strings.Repeat("a", 25)
	medium := strings.Repeat("a", 65)
	long := strings.Repeat("a", 71)
	ideal := strings.Repeat("a", 45)
	cases := []struct {
		title string
		score float64
	}{
		{long, 40},
		{short, 75},
		{medium, 75},
		{ideal, 100},
	}
	for _, c := range cases {
		score, _ := analyzeTitle(c.title, "", "pt")
		if score != c.score {
			t.Errorf("title len %d: expected %f, got %f", len([]rune(c.title)), c.score, score)
		}
	}
}

func TestAnalyzeTitle_KeywordPenalty(t *testing.T) {
	score, issues := analyzeTitle("Um título qualquer", "marketing", "pt")
	if score >= 100 {
		t.Errorf("expected keyword penalty, got %f", score)
	}
	found := false
	for _, i := range issues {
		if i.Field == "title" {
			found = true
		}
	}
	if !found {
		t.Error("expected title issue for missing keyword")
	}
}

func TestAnalyzeMeta_Empty(t *testing.T) {
	score, issues := analyzeMeta("", "", "pt")
	if score != 0 {
		t.Errorf("expected 0 for empty meta, got %f", score)
	}
	if len(issues) == 0 {
		t.Error("expected issue for empty meta")
	}
}

func TestAnalyzeHeadings_NoH1(t *testing.T) {
	score, issues := analyzeHeadings("## Subtitle only", "", "pt")
	if score != 30 {
		t.Errorf("expected 30 for missing H1, got %f", score)
	}
	if len(issues) == 0 {
		t.Error("expected issue for missing H1")
	}
}

func TestAnalyzeHeadings_MultipleH1(t *testing.T) {
	score, _ := analyzeHeadings("# H1 One\n# H1 Two", "", "pt")
	if score != 50 {
		t.Errorf("expected 50 for multiple H1, got %f", score)
	}
}

func TestAnalyzeKeyword_NoKeyword(t *testing.T) {
	score, issues, _, _ := analyzeKeyword("algum conteúdo", "titulo", "", "pt")
	if score != 0 {
		t.Errorf("expected 0 without keyword, got %f", score)
	}
	if len(issues) == 0 {
		t.Error("expected issue for missing keyword")
	}
}

func TestAnalyzeKeyword_DensityBand(t *testing.T) {
	content := "marketing de conteúdo é importante. O marketing de conteúdo gera resultados. Marketing de conteúdo exige constância. "
	for i := 0; i < 10; i++ {
		content += "palavras comuns de preenchimento para diluir a densidade aqui"
	}
	score, _, density, _ := analyzeKeyword(content, "titulo", "marketing de conteúdo", "pt")
	if density <= 0 {
		t.Errorf("expected positive density, got %f", density)
	}
	if score < 30 {
		t.Errorf("expected score >= 30 with density %f, got %f", density, score)
	}
}

func TestAnalyzeKeyword_FirstParagraph(t *testing.T) {
	score, issues, _, _ := analyzeKeyword("marketing de conteúdo aparece aqui no início", "titulo", "marketing de conteúdo", "pt")
	if score < 50 {
		t.Errorf("expected bonus for keyword in first paragraph, got %f", score)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestAnalyzePassiveVoice_Count(t *testing.T) {
	score, issues := analyzePassiveVoice("O relatório foi escrito pelo time. A página foi criada ontem. O sistema foi atualizado. O erro foi corrigido. O arquivo foi enviado.", "pt")
	if score >= 100 {
		t.Errorf("expected passive voice penalty, got %f", score)
	}
	if len(issues) == 0 {
		t.Error("expected issue for excessive passive voice")
	}
}

func TestAnalyzePassiveVoice_Clean(t *testing.T) {
	score, issues := analyzePassiveVoice("O time escreveu o relatório e corrigiu o erro rapidamente.", "pt")
	if score != 100 {
		t.Errorf("expected 100 for active voice, got %f", score)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestAnalyzeFreshness_NoDate(t *testing.T) {
	score, issues := analyzeFreshness("conteúdo sem nenhuma data mencionada", "pt")
	if score != 30 {
		t.Errorf("expected 30 without dates, got %f", score)
	}
	if len(issues) == 0 {
		t.Error("expected issue for missing date")
	}
}

func TestAnalyzeFreshness_RecentYear(t *testing.T) {
	score, _ := analyzeFreshness("Publicado em 2026 com dados atualizados.", "pt")
	if score != 100 {
		t.Errorf("expected 100 for recent year, got %f", score)
	}
}

func TestAnalyzeImages_MarkdownAlt(t *testing.T) {
	score, withAlt, issues := analyzeImages("![Imagem de exemplo](https://exemplo.com/img.jpg)", "pt")
	if withAlt != 1 {
		t.Errorf("expected 1 image with alt, got %d", withAlt)
	}
	if score != 100 {
		t.Errorf("expected 100 for all images with alt, got %f", score)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestAnalyzeImages_NoAlt(t *testing.T) {
	_, _, issues := analyzeImages("![](https://exemplo.com/img.jpg)", "pt")
	found := false
	for _, i := range issues {
		if i.Field == "image_alt" {
			found = true
		}
	}
	if !found {
		t.Error("expected image_alt issue for missing alt")
	}
}

func TestAnalyzeSchema_JSONLD(t *testing.T) {
	score, issues := analyzeSchema(`{"@context":"https://schema.org","@type":"Article"}`, "pt")
	if score != 100 {
		t.Errorf("expected 100 for JSON-LD, got %f", score)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestAnalyzeSchema_Missing(t *testing.T) {
	score, issues := analyzeSchema("conteúdo simples", "pt")
	if score != 0 {
		t.Errorf("expected 0 without schema, got %f", score)
	}
	if len(issues) == 0 {
		t.Error("expected schema issue")
	}
}

func TestAnalyzeLinks_InternalAndExternal(t *testing.T) {
	internal, external, ic, ec, issues := analyzeLinks("[link interno](/outra-pagina) e [externo](https://exemplo.com)", "pt")
	if ic != 1 || ec != 1 {
		t.Errorf("expected 1 internal and 1 external, got %d/%d", ic, ec)
	}
	if internal != 60 {
		t.Errorf("expected 60 internal score with 1 link, got %f", internal)
	}
	if external != 100 {
		t.Errorf("expected 100 external score, got %f", external)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestAnalyzeEEAT_Signals(t *testing.T) {
	strong := "Escrito por João Silva, especialista em engenharia de software, certificado em arquitetura de nuvem.\n" +
		"Em nossos testes medimos 99.9% de disponibilidade e 20ms de latência.\n" +
		"Nesse estudo de caso analisamos o benchmark da solução.\n" +
		"API é definido como um contrato de comunicação entre sistemas, e a latência das transações é medida por benchmark.\n" +
		"Fontes: estudo publicado pela Universidade X, acesso em 2026.\n" +
		"Visita oficial: https://www.gov.br/tecnologia\n" +
		"Atualizado em 2026. Isenção de responsabilidade aplicável.\n" +
		"<script type=\"application/ld+json\">{\"@context\":\"https://schema.org\"}</script>"
	score, _ := analyzeEEAT(strong, "software", "pt")
	if score < 80 {
		t.Errorf("expected strong EEAT signals, got %f", score)
	}
}

func TestAnalyzeEEAT_Weak(t *testing.T) {
	score, issues := analyzeEEAT("texto curto sem autor, data ou fontes", "", "pt")
	if score >= 100 {
		t.Errorf("expected low EEAT, got %f", score)
	}
	if len(issues) == 0 {
		t.Error("expected eeat issue")
	}
}

func TestAnalyzeSlug_Good(t *testing.T) {
	score, issues := analyzeSlug("guia-marketing-conteudo", "marketing", "pt")
	if score != 100 {
		t.Errorf("expected 100 for keyword-rich slug, got %f", score)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestAnalyzeSlug_Empty(t *testing.T) {
	score, issues := analyzeSlug("", "", "pt")
	if score != 0 {
		t.Errorf("expected 0 for empty slug, got %f", score)
	}
	if len(issues) == 0 {
		t.Error("expected issue for empty slug")
	}
}

func TestAnalyzeArticle_WeightsSum(t *testing.T) {
	total := weightTitle + weightMeta + weightHeadings + weightKeyword + weightReadability +
		weightInternalLinks + weightExternalLinks + weightEEAT + weightImages
	if total != 100 {
		t.Errorf("weights must sum to 100, got %f", total)
	}
}

func TestAnalyzeArticle_NilQualityChecker(t *testing.T) {
	ctx, in := testAnalyzer(t)
	a := AnalyzeArticle(*ctx, *in, nil)
	if a.ReadabilityScore <= 0 {
		t.Errorf("expected fallback readability with nil qc, got %f", a.ReadabilityScore)
	}
}

func TestKeywordCoverage(t *testing.T) {
	cov := keywordCoverage("falamos sobre marketing digital e conteúdo", "marketing digital")
	if cov != 1.0 {
		t.Errorf("expected full coverage, got %f", cov)
	}
	cov = keywordCoverage("texto sem relação nenhuma", "marketing digital")
	if cov != 0.0 {
		t.Errorf("expected no coverage, got %f", cov)
	}
}

func TestDeriveKeyword(t *testing.T) {
	kw := deriveKeyword("Guia Completo de Marketing de Conteúdo")
	if kw == "" {
		t.Error("expected derived keyword from title")
	}
	if stopWords[kw] {
		t.Errorf("derived keyword %q must not be a stop word", kw)
	}
	if len(kw) < 4 {
		t.Errorf("derived keyword %q too short", kw)
	}
}

func TestTokenize(t *testing.T) {
	words := tokenize("Olá, mundo! Teste-123.")
	want := []string{"olá", "mundo", "teste", "123"}
	if len(words) != len(want) {
		t.Fatalf("expected %d words, got %d: %v", len(want), len(words), words)
	}
	for i := range want {
		if words[i] != want[i] {
			t.Errorf("expected %q, got %q", want[i], words[i])
		}
	}
}

func TestExtractContentText(t *testing.T) {
	content := []byte(`[
		{"type":"heading","text":"Título do Artigo"},
		{"type":"text","text":"Primeiro parágrafo."},
		{"type":"paragraph","content":[{"type":"text","text":"Texto aninhado."}]}
	]`)
	out := extractContentText(content)
	if !contains(out, "# Título do Artigo") {
		t.Errorf("expected markdown heading in output, got %q", out)
	}
	if !contains(out, "Primeiro parágrafo.") {
		t.Errorf("expected text block in output, got %q", out)
	}
	if !contains(out, "Texto aninhado.") {
		t.Errorf("expected nested text in output, got %q", out)
	}
}

func TestExtractContentText_InvalidJSON(t *testing.T) {
	out := extractContentText([]byte("not json"))
	if out == "" {
		t.Error("expected raw fallback for invalid JSON")
	}
}

func TestExtractContentText_Empty(t *testing.T) {
	out := extractContentText([]byte("[]"))
	if out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestShingleSimilarity(t *testing.T) {
	a := "marketing de conteúdo é a estratégia de criar material relevante para atrair clientes e gerar vendas"
	b := "marketing de conteúdo é a estratégia de criar material relevante para atrair clientes e gerar vendas"
	sim := shingleSimilarity(a, b)
	if sim != 1.0 {
		t.Errorf("expected similarity 1.0, got %f", sim)
	}
	c := "completamente diferente sobre futebol e culinária brasileira tradicional"
	sim = shingleSimilarity(a, c)
	if sim >= 0.5 {
		t.Errorf("expected low similarity, got %f", sim)
	}
}

func TestShingleSimilarity_Short(t *testing.T) {
	if s := shingleSimilarity("um dois", "um dois"); s != 0 {
		t.Errorf("expected 0 for short texts, got %f", s)
	}
}

func TestClampScore(t *testing.T) {
	if clampScore(-5) != 0 {
		t.Error("expected clamp to 0")
	}
	if clampScore(150) != 100 {
		t.Error("expected clamp to 100")
	}
	if clampScore(50) != 50 {
		t.Error("expected unchanged value")
	}
}

func TestRound2(t *testing.T) {
	if round2(1.235) != 1.24 {
		t.Errorf("expected 1.24, got %f", round2(1.235))
	}
	if round2(1.234) != 1.23 {
		t.Errorf("expected 1.23, got %f", round2(1.234))
	}
}

func TestBuildChecklist(t *testing.T) {
	ctx, in := testAnalyzer(t)
	a := AnalyzeArticle(*ctx, *in, nil)
	items := buildChecklist(a)
	if len(items) == 0 {
		t.Fatal("expected checklist items from suggestions")
	}
	for _, it := range items {
		if it.Issue == "" || it.Suggestion == "" {
			t.Error("expected non-empty checklist issue and suggestion")
		}
		if it.Priority == "" {
			t.Error("expected checklist priority")
		}
	}
}

func TestFilterIssues(t *testing.T) {
	issues := []AuditIssue{
		{Field: "title"},
		{Field: "meta"},
		{Field: "slug"},
	}
	filtered := filterIssues(issues, "title", "slug")
	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered issues, got %d", len(filtered))
	}
}

func TestIssueMessages(t *testing.T) {
	issues := []AuditIssue{{Issue: "a"}, {Issue: "b"}}
	msgs := issueMessages(issues)
	if len(msgs) != 2 || msgs[0] != "a" {
		t.Errorf("unexpected messages: %v", msgs)
	}
}

func TestStableScore_Deterministic(t *testing.T) {
	if stableScore("keyword x") != stableScore("keyword x") {
		t.Error("expected stable score to be deterministic")
	}
	if stableScore("a") == stableScore("b") {
		t.Error("expected different inputs to usually differ")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
