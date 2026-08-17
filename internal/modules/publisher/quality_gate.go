package publisher

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// QualityGateInput mirrors the publish funnel state the quality gate evaluates.
// It is deliberately content-only: every field is already in memory at the
// funnel entry, so the gate never needs the database.
type QualityGateInput struct {
	SiteID        uuid.UUID
	Title         string
	Content       string
	Language      string
	ContentType   string
	ResearchFacts int
}

// QualityIssue is a single gate finding. Severity "error" blocks
// publication; "warning" is informational only.
type QualityIssue struct {
	Field      string `json:"field"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	Severity   string `json:"severity"`
	Score      int    `json:"score,omitempty"`
}

// QualityGateResult is the deterministic gate verdict: a 0-100 score built
// from pure content analysis plus the structural counts behind it.
type QualityGateResult struct {
	Score     float64        `json:"score"`
	Passed    bool           `json:"passed"`
	MinScore  float64        `json:"min_score"`
	WordCount int            `json:"word_count"`
	H2Count   int            `json:"h2_count"`
	H3Count   int            `json:"h3_count"`
	Issues    []QualityIssue `json:"issues,omitempty"`
}

// ErrQualityGateBlocked is returned by the funnel when the quality gate
// rejects content. Wrapped with the score breakdown for the job error.
var ErrQualityGateBlocked = errors.New("quality gate blocked publication")

// QualityGate is the fail-open quality gate applied to auto-generated
// content before the SEO gate. Implementations must be deterministic and
// DB-free; evaluation errors must never block publication.
type QualityGate interface {
	CheckQuality(ctx context.Context, in QualityGateInput) (*QualityGateResult, error)
}

const (
	// MinQualityGateScore is the minimum overall score required to pass.
	MinQualityGateScore = 80.0

	// quality penalties (deducted from 100, floor 0)
	qPenaltyWordCount = 30
	qPenaltyTitle     = 10
	qPenaltyIntro     = 10
	qPenaltyH2        = 15
	qPenaltyH3        = 5
	qPenaltyExamples  = 10
	qPenaltyResearch  = 15
	qPenaltyGeneric   = 20
)

var (
	qMarkdownSymbolsRE = regexp.MustCompile(`[#*>_~\[\]()]`)
	qHeadingRE         = regexp.MustCompile(`(?m)^#{1,6}\s+.+$`)
	qH2RE              = regexp.MustCompile(`(?m)^##\s+.+$`)
	qH3RE              = regexp.MustCompile(`(?m)^###\s+.+$`)
	qHTMLH2RE          = regexp.MustCompile(`(?i)<h2[^>]*>.*?</h2>`)
	qHTMLH3RE          = regexp.MustCompile(`(?i)<h3[^>]*>.*?</h3>`)
	qExternalLinkRE    = regexp.MustCompile(`https?://[^\s"'<>)\]}]+`)
	qSourcesHeadingRE  = regexp.MustCompile(`(?mi)^#{1,6}\s*(sources|fontes|referencias|referências)\s*:?\s*$`)
	qBulletRE          = regexp.MustCompile(`(?m)^\s*[-*•]\s+`)
	qExampleRE         = regexp.MustCompile(`(?i)\b(exemplo:?|por exemplo|exemplos? de|example:?|for example|such as|e\.g\.)\b`)
)

// countContentWords returns the approximate word count of a markdown-ish text
// with markdown symbols stripped (headings markers, bold, links, lists).
func countContentWords(text string) int {
	plain := qMarkdownSymbolsRE.ReplaceAllString(text, " ")
	return len(strings.Fields(plain))
}

// isNewsType reports whether the content type implies a news article: news
// pieces need far fewer sections than evergreen content.
func isNewsType(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "news") || strings.Contains(ct, "noticia") || strings.Contains(ct, "notícia")
}

// requiresH3AndExamples reports whether the content type demands H3
// subsections and practical examples (how-to, list, comparison, review).
func requiresH3AndExamples(contentType string) bool {
	ct := strings.ToLower(contentType)
	for _, marker := range []string{"how", "tutorial", "guide", "list", "top", "comparison", "comparativ", "review", "comprar", "best", "melhores", "análise", "analise"} {
		if strings.Contains(ct, marker) {
			return true
		}
	}
	return false
}

// tokenJaccard is the classic token-set similarity between two texts (unique
// token sets, |a∩b| / |a∪b|).
func tokenJaccard(a, b string) float64 {
	ta := strings.Fields(strings.ToLower(a))
	tb := strings.Fields(strings.ToLower(b))
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	sa := map[string]bool{}
	for _, t := range ta {
		sa[t] = true
	}
	sb := map[string]bool{}
	for _, t := range tb {
		sb[t] = true
	}
	intersection := 0
	for t := range sa {
		if sb[t] {
			intersection++
		}
	}
	union := len(sa) + len(sb) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// detectGenericContent flags templated or self-repeating text: near-identical
// consecutive sentences (template fill-ins like "X is a powerful tool that
// helps you ...") or duplicated paragraphs.
func detectGenericContent(content string) (sentence bool, paragraph bool) {
	plain := strings.TrimSpace(content)
	if plain == "" {
		return false, false
	}
	lower := strings.ToLower(plain)
	// split into sentences on punctuation + line breaks
	fields := regexp.MustCompile(`[.!?]\s+|\n+`).Split(lower, -1)
	for i := 0; i+1 < len(fields); i++ {
		if len(strings.Fields(fields[i])) < 4 {
			continue
		}
		if tokenJaccard(fields[i], fields[i+1]) >= 0.9 {
			sentence = true
			break
		}
	}
	paras := regexp.MustCompile(`\n\s*\n`).Split(lower, -1)
	seen := map[string]bool{}
	for _, p := range paras {
		p = strings.TrimSpace(p)
		if len(p) < 30 {
			continue
		}
		if seen[p] {
			paragraph = true
			break
		}
		seen[p] = true
	}
	return sentence, paragraph
}

// CheckContentQuality is the pure, deterministic quality gate. It analyzes
// depth (word count), structure (title, intro, H2/H3), substance (examples),
// research grounding (facts → citations) and originality (generic/repetitive
// text). Same input always produces the same verdict; no database, no AI, no
// randomness.
func CheckContentQuality(in QualityGateInput, minWordCount, minH2 int, minScore float64) *QualityGateResult {
	if minWordCount <= 0 {
		minWordCount = 1000
	}
	if minH2 <= 0 {
		minH2 = 3
	}
	if isNewsType(in.ContentType) && minH2 > 1 {
		minH2 = 1
	}
	if minScore <= 0 {
		minScore = MinQualityGateScore
	}

	res := &QualityGateResult{MinScore: minScore}
	res.WordCount = countContentWords(in.Content)
	res.H2Count = len(qH2RE.FindAllString(in.Content, -1)) + len(qHTMLH2RE.FindAllString(in.Content, -1))
	res.H3Count = len(qH3RE.FindAllString(in.Content, -1)) + len(qHTMLH3RE.FindAllString(in.Content, -1))

	score := 100.0
	add := func(field, message, suggestion string, penalty int) {
		sev := "error"
		issueScore := 100 - penalty
		if issueScore < 0 {
			issueScore = 0
		}
		res.Issues = append(res.Issues, QualityIssue{
			Field:      field,
			Message:    message,
			Suggestion: suggestion,
			Severity:   sev,
			Score:      issueScore,
		})
		score -= float64(penalty)
	}

	// 1. Depth: the article must meet the minimum word count.
	if res.WordCount < minWordCount {
		add("word_count",
			fmt.Sprintf("artigo curto demais: %d palavras (mínimo %d)", res.WordCount, minWordCount),
			"Expanda cada seção com dados, exemplos e detalhes reais antes de publicar.", qPenaltyWordCount)
	}

	// 2. Title.
	if strings.TrimSpace(in.Title) == "" {
		add("title", "artigo sem título", "Defina um título descritivo para o artigo.", qPenaltyTitle)
	}

	// 3. Intro: the text before the first heading must be a real
	// introduction, not a heading-first stub.
	intro := in.Content
	if loc := qHeadingRE.FindStringIndex(in.Content); loc != nil {
		intro = in.Content[:loc[0]]
	}
	if countWords(intro) < 30 {
		add("intro",
			"introdução muito curta (menos de 30 palavras antes do primeiro cabeçalho)",
			"Abra o artigo com um parágrafo que apresente o assunto e o que o leitor vai aprender.", qPenaltyIntro)
	}

	// 4. Structure: enough H2 sections to be a real article.
	if res.H2Count < minH2 {
		add("headings",
			fmt.Sprintf("apenas %d seção H2 (mínimo %d)", res.H2Count, minH2),
			"Organize o conteúdo em pelo menos 3 seções H2 com subseções H3 quando fizer sentido.", qPenaltyH2)
	}

	// 5. Substance: how-to/list/comparison/review content needs H3 and
	// practical examples — the classic AI-shallow pattern.
	needsExamples := requiresH3AndExamples(in.ContentType)
	if needsExamples && res.H3Count == 0 {
		add("subsections",
			"artigo de lista/guia/comparação sem subseções H3",
			"Use H3 para detalhar cada item/etapa com dados específicos.", qPenaltyH3)
	}
	if needsExamples {
		hasBullets := len(qBulletRE.FindAllString(in.Content, -1)) > 0
		hasExamples := qExampleRE.MatchString(in.Content)
		if !hasBullets && !hasExamples {
			add("examples",
				"artigo sem exemplos práticos (listas ou marcadores de exemplo)",
				"Inclua exemplos concretos: features, preços, prós e contras, passos numerados.", qPenaltyExamples)
		}
	}

	// 6. Research grounding: when the pipeline had verified facts, the
	// article must cite its sources (external links or a Sources section).
	if in.ResearchFacts > 0 {
		hasExternal := len(qExternalLinkRE.FindAllString(in.Content, -1)) > 0
		hasSources := qSourcesHeadingRE.MatchString(in.Content)
		if !hasExternal && !hasSources {
			add("research",
				"artigo com fatos verificados disponíveis mas sem fontes citadas",
				"Adicione uma seção de Fontes com links para as fontes usadas na pesquisa.", qPenaltyResearch)
		}
	}

	// 7. Originality: reject templated / self-repeating text.
	if sent, para := detectGenericContent(in.Content); sent || para {
		add("generic",
			"conteúdo genérico ou repetitivo detectado",
			"Reescreva com informações específicas e frases originais; evite fórmulas repetidas.", qPenaltyGeneric)
	}

	if score < 0 {
		score = 0
	}
	res.Score = score
	res.Passed = res.Score >= res.MinScore && !hasErrorIssue(res.Issues)
	return res
}

func hasErrorIssue(issues []QualityIssue) bool {
	for _, i := range issues {
		if i.Severity == "error" {
			return true
		}
	}
	return false
}
