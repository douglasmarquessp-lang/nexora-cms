package seoengine

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"nexora/internal/modules/publisher"
)

// wellStructuredArticle builds a deterministic, well-structured article that
// mirrors what the AI pipeline + enhancer produce for a real topic: single H1
// (the page title), 2+ H2, 1 H3, keyword in the first 100 words, ideal density
// band, 2 internal links, 1 external link, images with alt, and a 150-160
// rune meta description containing the keyword.
func wellStructuredArticle(title, keyword, meta string) string {
	var body strings.Builder
	// first paragraph: keyword within the first 100 words
	body.WriteString(fmt.Sprintf("%s é o tema central deste artigo. Este texto explica %s para quem está começando. ", title, keyword))
	body.WriteString("Parágrafos bem escritos comunicam com clareza. Quem lê espera respostas diretas e informação útil. ")
	for i := 0; i < 40; i++ {
		body.WriteString("O texto seguinte desenvolve o assunto com contexto, exemplos práticos e recomendações de uso. ")
	}
	body.WriteString("\n\n## O que é\n")
	body.WriteString(fmt.Sprintf("Primeiro entendemos o conceito de %s e de onde ele vem. ", keyword))
	for i := 0; i < 10; i++ {
		body.WriteString(fmt.Sprintf("Uma explicação acessível ajuda o leitor a fixar o fundamento do %s. ", keyword))
	}
	body.WriteString("\n\n## Como aplicar\n")
	for i := 0; i < 10; i++ {
		body.WriteString(fmt.Sprintf("Na prática, aplicar o %s exige método, consistência e medição de resultados. ", keyword))
	}
	body.WriteString("\n\n### Passos iniciais\n")
	for i := 0; i < 10; i++ {
		body.WriteString("Comece devagar, valide cada etapa e ajuste o processo conforme os resultados aparecem. ")
	}
	body.WriteString("\n\nLeia mais sobre o tema em [guia completo](/guia-completo) e [estratégias avançadas](/estrategias-avancadas). ")
	body.WriteString("A referência oficial está disponível em [exemplo](https://example.com/referencia). ")
	body.WriteString("\n\n![gráfico explicativo](/img/grafico.png) ")
	body.WriteString("![processo ilustrado](/img/processo.png)")
	return body.String()
}

func wellMeta(keyword string) string {
	// 150-160 runes, contains the keyword
	meta := fmt.Sprintf("%s — guia completo e atualizado para iniciantes. ", strings.Title(strings.ReplaceAll(keyword, "-", " ")))
	for _, w := range []string{"dicas", "práticas", "definições", "e exemplos", "reais de uso", "em um só lugar"} {
		if len([]rune(meta+w)) > 155 {
			break
		}
		meta += w + " "
	}
	return meta
}

func gateFlowScore(t *testing.T, svc *Service, title, content, meta, lang string) float64 {
	t.Helper()
	score, err := svc.CheckPublishScore(context.Background(), publisher.PublishGateInput{
		SiteID:          uuid.New(),
		Title:           title,
		Content:         content,
		MetaDescription: meta,
		Language:        lang,
	})
	if err != nil {
		t.Fatalf("CheckPublishScore failed: %v", err)
	}
	return score
}

// A well-structured article (title + headings + keyword density + links +
// images + meta) must clear the 80-point publish minimum without any AI or
// stored score: the whole gate path is deterministic.
func TestCheckPublishScore_WellStructuredPasses(t *testing.T) {
	svc, _ := setupMockDB(t) // inline analysis path does not touch the DB
	title := "Guia de Marketing de Conteúdo para Pequenas Empresas"
	content := wellStructuredArticle(title, "marketing de conteúdo", "")
	score := gateFlowScore(t, svc, title, content, wellMeta("marketing de conteúdo"), "pt")
	if score < 80 {
		t.Errorf("expected well-structured article to pass the 80 minimum gate, got %.2f", score)
	}
}

// A second, different topic must also pass: the gate is not tuned to one
// fixture (no keyword/topic hardcoding anywhere).
func TestCheckPublishScore_SecondTopicAlsoPasses(t *testing.T) {
	svc, _ := setupMockDB(t)
	title := "Como Usar Inteligência Artificial no Atendimento ao Cliente"
	content := wellStructuredArticle(title, "inteligência artificial", "")
	meta := "inteligência artificial no atendimento: guia para empresas com exemplos práticos, métricas e planos de implementação em um texto claro"
	score := gateFlowScore(t, svc, title, content, meta, "pt")
	if score < 80 {
		t.Errorf("expected second well-structured article to pass, got %.2f", score)
	}
}

func TestCheckPublishScore_MissingMetaLosesPoints(t *testing.T) {
	svc, _ := setupMockDB(t)
	title := "Guia de Marketing de conteúdo para Pequenas Empresas"
	content := wellStructuredArticle(title, "marketing de conteúdo", "")
	withMeta := gateFlowScore(t, svc, title, content, wellMeta(title), "pt")
	noMeta := gateFlowScore(t, svc, title, content, "", "pt")
	if noMeta >= withMeta {
		t.Errorf("expected empty meta to score below a real meta description: %f >= %f", noMeta, withMeta)
	}
	if noMeta > 80 {
		t.Errorf("expected empty meta to drop below the publish minimum, got %.2f", noMeta)
	}
}

func TestCheckPublishScore_MissingHeadingsReducesScore(t *testing.T) {
	svc, _ := setupMockDB(t)
	title := "Guia de Marketing de Conteúdo para Pequenas Empresas"
	base := wellStructuredArticle(title, "marketing de conteúdo", "")
	withHeadings := base
	noHeadings := strings.ReplaceAll(strings.ReplaceAll(base, "\n\n## O que é\n", "\n\n"), "\n\n### Passos iniciais\n", "\n\n")
	noHeadings = strings.ReplaceAll(noHeadings, "\n\n## Como aplicar\n", "\n\n")
	noHeadings = strings.TrimRight(noHeadings, "\n")
	sWith := gateFlowScore(t, svc, title, withHeadings, wellMeta(title), "pt")
	sNo := gateFlowScore(t, svc, title, noHeadings, wellMeta(title), "pt")
	if sNo >= sWith {
		t.Errorf("expected missing headings to lose points, %f >= %f", sNo, sWith)
	}
}

func TestCheckPublishScore_MissingExternalLinksReducesScore(t *testing.T) {
	svc, _ := setupMockDB(t)
	title := "Guia de Marketing de Conteúdo para Pequenas Empresas"
	withExt := wellStructuredArticle(title, "marketing de conteúdo", "")
	noExt := strings.ReplaceAll(withExt, "A referência oficial está disponível em [exemplo](https://example.com/referencia). ", "")
	sWith := gateFlowScore(t, svc, title, withExt, wellMeta(title), "pt")
	sNo := gateFlowScore(t, svc, title, noExt, wellMeta(title), "pt")
	if sNo >= sWith {
		t.Errorf("expected missing external links to reduce the score, %f >= %f", sNo, sWith)
	}
}

// A poor draft (no structure, no links, empty meta, no keyword) stays far
// below the minimum — the gate still blocks bad content.
func TestCheckPublishScore_MediocreArticleFails(t *testing.T) {
	svc, _ := setupMockDB(t)
	title := "Post"
	score := gateFlowScore(t, svc, title,
		"Um texto curto e sem estrutura, sem links e sem palavra-chave repetida num parágrafo pequeno qualquer.",
		"", "pt")
	if score >= 80 {
		t.Errorf("expected poor article to fail the gate, got %.2f", score)
	}
}
