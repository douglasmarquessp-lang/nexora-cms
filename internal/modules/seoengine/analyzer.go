package seoengine

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"nexora/internal/ai"
)

// Weights for the weighted SEO score (must sum to 100).
const (
	weightTitle         = 15.0
	weightMeta          = 10.0
	weightHeadings      = 15.0
	weightKeyword       = 20.0
	weightReadability   = 10.0
	weightInternalLinks = 10.0
	weightExternalLinks = 5.0
	weightEEAT          = 10.0
	weightImages        = 5.0
)

// ArticleAnalysisInput is the deterministic, DB-free input for the analyzer.
type ArticleAnalysisInput struct {
	Title           string
	MetaDescription string
	Slug            string
	Content         string
	Keyword         string
	Language        string
	// AuthorName (optional) strengthens the EEAT signals when known.
	AuthorName string
	// Category (optional) enables subject-based internal linking scoring.
	Category string
}

// ArticleAnalysis is the result of the deterministic analysis.
type ArticleAnalysis struct {
	OverallScore         float64
	TitleScore           float64
	MetaScore            float64
	HeadingScore         float64
	KeywordScore         float64
	ReadabilityScore     float64
	InternalLinksScore   float64
	ExternalLinksScore   float64
	EEATScore            float64
	ImagesScore          float64
	SlugScore            float64
	SchemaScore          float64
	ParagraphScore       float64
	PassiveVoiceScore    float64
	SentenceVariationScore float64
	DuplicateScore       float64
	FreshnessScore       float64
	TopicalAuthorityScore float64
	KeywordDensity       float64
	WordCount            int
	InternalLinks        int
	ExternalLinks        int
	ImagesWithAlt        int
	DuplicateCount       int
	Suggestions          []AuditIssue
}

// bi is a bilingual (PT/EN) string pair.
type bi struct {
	pt string
	en string
}

func (b bi) text(lang string) string {
	if lang == "en" {
		return b.en
	}
	return b.pt
}

var (
	markdownH1RE = regexp.MustCompile(`(?m)^#\s+.+$`)
	markdownH2RE = regexp.MustCompile(`(?m)^##\s+.+$`)
	markdownH3RE = regexp.MustCompile(`(?m)^###\s+.+$`)
	htmlH1RE     = regexp.MustCompile(`(?i)<h1[^>]*>.*?</h1>`)
	htmlH2RE     = regexp.MustCompile(`(?i)<h2[^>]*>.*?</h2>`)
	htmlH3RE     = regexp.MustCompile(`(?i)<h3[^>]*>.*?</h3>`)
	mdLinkRE     = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)
	mdImageRE    = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)\)`)
	htmlImageRE  = regexp.MustCompile(`(?i)<img[^>]*alt="([^"]*)"[^>]*>`)
	urlRE        = regexp.MustCompile(`https?://[^\s"'<>]+`)
	jsonLDRE     = regexp.MustCompile(`(?i)application/ld\+json|"@context"`)
	yearRE       = regexp.MustCompile(`\b(19|20)\d{2}\b`)
	passivePTRE  = regexp.MustCompile(`(?i)\b(foi|foram|é|são|era|eram|sendo|ser|sou|somos|seja|fossem)\s+(\w+(ado|ada|ido|ida|ados|adas|idos|idas)s?)\b`)
	passiveENRE  = regexp.MustCompile(`(?i)\b(is|are|was|were|been|being|be)\s+(\w+(ed|en|t)\b|said|made|written|done|taken|given|seen|known)\b`)
	sentenceSplitRE = regexp.MustCompile(`[.!?]+(\s+|$)`)
	stopWords    = map[string]bool{
		"a": true, "o": true, "e": true, "de": true, "da": true, "do": true,
		"em": true, "para": true, "com": true, "por": true, "um": true, "uma": true,
		"na": true, "no": true, "os": true, "as": true, "que": true, "ao": true,
		"the": true, "and": true, "of": true, "to": true, "in": true, "for": true,
		"on": true, "with": true, "is": true, "at": true, "from": true, "by": true,
		"sobre": true, "entre": true, "como": true, "mais": true, "mas": true,
		"or": true, "an": true, "be": true, "this": true, "that": true,
	}
)

// AnalyzeArticle runs the full deterministic analysis. It never calls the DB
// or external services; every score is computed from the input text.
func AnalyzeArticle(ctx context.Context, in ArticleAnalysisInput, qc ai.QualityChecker) *ArticleAnalysis {
	lang := in.Language
	if lang == "" {
		lang = "pt"
	}

	titleScore, titleIssues := analyzeTitle(in.Title, in.Keyword, lang)
	metaScore, metaIssues := analyzeMeta(in.MetaDescription, in.Keyword, lang)
	headingScore, headingIssues := analyzeHeadings(in.Content, in.Keyword, lang)
	keywordScore, keywordIssues, density, wordCount := analyzeKeyword(in.Content, in.Title, in.Keyword, lang)
	readabilityScore, readabilityIssues := analyzeReadability(ctx, qc, in.Content, lang)
	internalScore, externalScore, internalLinks, externalLinks, linkIssues := analyzeLinks(in.Content, lang)
	eeatScore, eeatIssues := analyzeEEAT(in.Content, in.Keyword, lang)
	imagesScore, imagesWithAlt, imageIssues := analyzeImages(in.Content, lang)
	slugScore, slugIssues := analyzeSlug(in.Slug, in.Keyword, lang)
	schemaScore, schemaIssues := analyzeSchema(in.Content, lang)
	passiveScore, passiveIssues := analyzePassiveVoice(in.Content, lang)
	variationScore, variationIssues := analyzeSentenceVariation(in.Content, lang)
	freshnessScore, freshnessIssues := analyzeFreshness(in.Content, lang)
	duplicateScore, duplicateCount := analyzeDuplicates(ctx, qc, in.Content)
	topicalAuthority := keywordCoverage(in.Content, in.Keyword) * 100
	paragraphScore := clampScore((headingScore + readabilityScore) / 2)

	overall := (titleScore*weightTitle +
		metaScore*weightMeta +
		headingScore*weightHeadings +
		keywordScore*weightKeyword +
		readabilityScore*weightReadability +
		internalScore*weightInternalLinks +
		externalScore*weightExternalLinks +
		eeatScore*weightEEAT +
		imagesScore*weightImages) / 100.0

	suggestions := make([]AuditIssue, 0, 24)
	for _, s := range [][]AuditIssue{titleIssues, metaIssues, headingIssues, keywordIssues, readabilityIssues, linkIssues, eeatIssues, imageIssues, slugIssues, schemaIssues, passiveIssues, variationIssues, freshnessIssues} {
		suggestions = append(suggestions, s...)
	}

	return &ArticleAnalysis{
		OverallScore:           clampScore(overall),
		TitleScore:             titleScore,
		MetaScore:              metaScore,
		HeadingScore:           headingScore,
		KeywordScore:           keywordScore,
		ReadabilityScore:       readabilityScore,
		InternalLinksScore:     internalScore,
		ExternalLinksScore:     externalScore,
		EEATScore:              eeatScore,
		ImagesScore:            imagesScore,
		SlugScore:              slugScore,
		SchemaScore:            schemaScore,
		ParagraphScore:         paragraphScore,
		PassiveVoiceScore:      passiveScore,
		SentenceVariationScore: variationScore,
		DuplicateScore:         duplicateScore,
		FreshnessScore:         freshnessScore,
		TopicalAuthorityScore:  round2(topicalAuthority),
		KeywordDensity:         density,
		WordCount:              wordCount,
		InternalLinks:          internalLinks,
		ExternalLinks:          externalLinks,
		ImagesWithAlt:          imagesWithAlt,
		DuplicateCount:         duplicateCount,
		Suggestions:            suggestions,
	}
}

func analyzeTitle(title, keyword, lang string) (float64, []AuditIssue) {
	title = strings.TrimSpace(title)
	if title == "" {
		return 0, []AuditIssue{issue("title", bi{"O título está vazio.", "The title is empty."}, bi{"Defina um título de 30 a 60 caracteres contendo a palavra-chave principal.", "Set a title of 30 to 60 characters including the primary keyword."}, 0, PriorityHigh, lang)}
	}
	length := len([]rune(title))
	score := 40.0
	if length >= 30 && length <= 60 {
		score = 100
	} else if length >= 20 && length <= 70 {
		score = 75
	}
	var issues []AuditIssue
	if length < 30 {
		issues = append(issues, issue("title", bi{"O título está muito curto.", "The title is too short."}, bi{"Aumente o título para 30 a 60 caracteres.", "Lengthen the title to 30 to 60 characters."}, score, PriorityHigh, lang))
	}
	if length > 70 {
		issues = append(issues, issue("title", bi{"O título está muito longo.", "The title is too long."}, bi{"Reduza o título para 30 a 60 caracteres.", "Shorten the title to 30 to 60 characters."}, score, PriorityHigh, lang))
	}
	if keyword != "" && !strings.Contains(strings.ToLower(title), strings.ToLower(keyword)) {
		score -= 20
		issues = append(issues, issue("title", bi{"A palavra-chave não está no título.", "The keyword is not in the title."}, bi{"Inclua a palavra-chave principal no título.", "Include the primary keyword in the title."}, score, PriorityMedium, lang))
	}
	return clampScore(score), issues
}

func analyzeMeta(meta, keyword, lang string) (float64, []AuditIssue) {
	meta = strings.TrimSpace(meta)
	if meta == "" {
		return 0, []AuditIssue{issue("meta", bi{"A meta description está vazia.", "The meta description is empty."}, bi{"Escreva uma meta description de 150 a 160 caracteres com a palavra-chave e uma chamada para ação.", "Write a meta description of 150 to 160 characters with the keyword and a call to action."}, 0, PriorityHigh, lang)}
	}
	length := len([]rune(meta))
	score := 50.0
	if length >= 150 && length <= 160 {
		score = 100
	} else if length >= 120 && length <= 180 {
		score = 75
	}
	var issues []AuditIssue
	if length > 160 {
		issues = append(issues, issue("meta", bi{"A meta description excede 160 caracteres.", "The meta description exceeds 160 characters."}, bi{"Reduza a meta description para no máximo 160 caracteres.", "Reduce the meta description to at most 160 characters."}, score, PriorityHigh, lang))
	}
	if length < 120 {
		issues = append(issues, issue("meta", bi{"A meta description é muito curta.", "The meta description is too short."}, bi{"Amplie a meta description para 150 a 160 caracteres.", "Expand the meta description to 150 to 160 characters."}, score, PriorityMedium, lang))
	}
	if keyword != "" && !strings.Contains(strings.ToLower(meta), strings.ToLower(keyword)) {
		score -= 15
		issues = append(issues, issue("meta", bi{"A palavra-chave não está na meta description.", "The keyword is not in the meta description."}, bi{"Inclua a palavra-chave principal na meta description.", "Include the primary keyword in the meta description."}, score, PriorityMedium, lang))
	}
	return clampScore(score), issues
}

func analyzeHeadings(content, keyword, lang string) (float64, []AuditIssue) {
	h1 := len(markdownH1RE.FindAllString(content, -1)) + len(htmlH1RE.FindAllString(content, -1))
	h2 := len(markdownH2RE.FindAllString(content, -1)) + len(htmlH2RE.FindAllString(content, -1))
	h3 := len(markdownH3RE.FindAllString(content, -1)) + len(htmlH3RE.FindAllString(content, -1))

	var issues []AuditIssue
	score := 0.0
	if h1 == 0 {
		score = 30
		issues = append(issues, issue("headings", bi{"Não foi encontrado um H1.", "No H1 was found."}, bi{"Use exatamente um H1 contendo a palavra-chave principal.", "Use exactly one H1 including the primary keyword."}, score, PriorityHigh, lang))
	} else if h1 > 1 {
		score = 50
		issues = append(issues, issue("headings", bi{"Foram encontrados múltiplos H1.", "Multiple H1s were found."}, bi{"Use exatamente um H1 por página.", "Use exactly one H1 per page."}, score, PriorityHigh, lang))
	} else {
		score = 60
		if h2 >= 2 {
			score += 20
		}
		if h3 >= 1 {
			score += 10
		}
		if keyword != "" && h2 == 0 {
			score -= 20
			issues = append(issues, issue("headings", bi{"Não há subtítulos H2.", "No H2 subheadings."}, bi{"Adicione subtítulos H2 para estruturar o conteúdo.", "Add H2 subheadings to structure the content."}, score, PriorityMedium, lang))
		}
	}
	return clampScore(score), issues
}

func analyzeKeyword(content, title, keyword, lang string) (float64, []AuditIssue, float64, int) {
	lowerContent := strings.ToLower(content)
	words := tokenize(content)
	wordCount := len(words)
	density := 0.0
	var issues []AuditIssue

	if strings.TrimSpace(keyword) == "" {
		return 0, []AuditIssue{issue("keyword", bi{"Não foi definida uma palavra-chave primária.", "No primary keyword was defined."}, bi{"Defina a palavra-chave principal do artigo para medir densidade e posição.", "Define the article's primary keyword to measure density and placement."}, 0, PriorityHigh, lang)}, 0, wordCount
	}

	kwLower := strings.ToLower(keyword)
	freq := strings.Count(lowerContent, kwLower)
	if wordCount > 0 {
		density = float64(freq) / float64(wordCount) * 100
	}

	score := 30.0
	if density >= 1 && density <= 3 {
		score = 70
	} else if density >= 0.5 && density <= 5 {
		score = 50
	}

	firstWords := words
	if len(firstWords) > 100 {
		firstWords = firstWords[:100]
	}
	firstBlock := strings.Join(firstWords, " ")
	if strings.Contains(strings.ToLower(firstBlock), kwLower) {
		score += 20
	} else {
		issues = append(issues, issue("keyword", bi{"A palavra-chave não aparece no primeiro parágrafo.", "The keyword does not appear in the first paragraph."}, bi{"Inclua a palavra-chave no primeiro parágrafo.", "Include the keyword in the first paragraph."}, score, PriorityHigh, lang))
	}
	if strings.Contains(strings.ToLower(title), kwLower) {
		score += 10
	}
	if score > 100 {
		score = 100
	}

	return clampScore(score), issues, round2(density), wordCount
}

func analyzeReadability(ctx context.Context, qc ai.QualityChecker, content, lang string) (float64, []AuditIssue) {
	if strings.TrimSpace(content) == "" {
		return 0, []AuditIssue{issue("readability", bi{"Não há conteúdo para avaliar a legibilidade.", "No content to assess readability."}, bi{"Adicione o corpo do artigo para análise de legibilidade.", "Add the article body for readability analysis."}, 0, PriorityMedium, lang)}
	}
	if qc == nil {
		return 50, nil
	}
	report, err := qc.ScoreReadabilityDetailed(ctx, content, lang)
	if err != nil || report == nil {
		return 50, nil
	}
	score := report.OverallScore
	var issues []AuditIssue
	if score < 60 {
		issues = append(issues, issue("readability", bi{"A legibilidade do conteúdo precisa de melhorias.", "Content readability needs improvement."}, bi{"Use frases mais curtas e vocabulário simples para melhorar a legibilidade.", "Use shorter sentences and simpler vocabulary to improve readability."}, score, PriorityMedium, lang))
	}
	return clampScore(score), issues
}

func analyzeLinks(content, lang string) (float64, float64, int, int, []AuditIssue) {
	internalLinks := 0
	externalLinks := 0

	bare := content
	for _, m := range mdLinkRE.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		u := m[1]
		if strings.HasPrefix(u, "/") {
			internalLinks++
		} else if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
			externalLinks++
		}
		bare = strings.ReplaceAll(bare, m[0], "")
	}
	for _, m := range mdImageRE.FindAllStringSubmatch(content, -1) {
		bare = strings.ReplaceAll(bare, m[0], "")
	}
	externalLinks += len(urlRE.FindAllString(bare, -1))

	internalScore := 30.0
	externalScore := 40.0
	var issues []AuditIssue
	if internalLinks >= 2 {
		internalScore = 100
	} else if internalLinks == 1 {
		internalScore = 60
	} else {
		issues = append(issues, issue("internal_link", bi{"Não há links internos.", "No internal links."}, bi{"Adicione pelo menos 2 links internos para outras páginas do site.", "Add at least 2 internal links to other pages of the site."}, internalScore, PriorityMedium, lang))
	}
	if externalLinks >= 1 {
		externalScore = 100
	} else {
		issues = append(issues, issue("external_link", bi{"Não há links externos.", "No external links."}, bi{"Adicione pelo menos 1 link externo de autoridade como referência.", "Add at least 1 authoritative external link as a reference."}, externalScore, PriorityLow, lang))
	}
	return clampScore(internalScore), clampScore(externalScore), internalLinks, externalLinks, issues
}

func analyzeEEAT(content, keyword, lang string) (float64, []AuditIssue) {
	report := AnalyzeEEAT(ArticleAnalysisInput{Content: content, Keyword: keyword, Language: lang})
	var issues []AuditIssue
	for _, p := range report.Pillars {
		if len(p.Issues) == 0 {
			continue
		}
		issues = append(issues, AuditIssue{
			Field:      "eeat." + strings.ToLower(p.Name),
			Issue:      p.Name + ": " + strings.Join(p.Issues, " "),
			Suggestion: bi{"Reforce o pilar " + p.Name + " (ver detalhes no relatório EEAT).", "Strengthen the " + p.Name + " pillar (see the EEAT report for details)."}.text(lang),
			Score:      report.Final,
			Priority:   string(PriorityHigh),
		})
	}
	if len(issues) == 0 {
		return report.Final, nil
	}
	return report.Final, issues
}

func analyzeImages(content, lang string) (float64, int, []AuditIssue) {
	withAlt := 0
	total := 0
	for _, m := range mdImageRE.FindAllStringSubmatch(content, -1) {
		if len(m) >= 2 && strings.TrimSpace(m[1]) != "" {
			withAlt++
		}
		total++
	}
	for _, m := range htmlImageRE.FindAllStringSubmatch(content, -1) {
		if len(m) >= 2 && strings.TrimSpace(m[1]) != "" {
			withAlt++
		}
		total++
	}

	score := 30.0
	var issues []AuditIssue
	if total > 0 && withAlt == total {
		score = 100
	} else if total > 0 {
		score = 60
		issues = append(issues, issue("image_alt", bi{"Há imagens sem texto ALT.", "There are images without ALT text."}, bi{"Adicione texto ALT descritivo a todas as imagens, incluindo a palavra-chave quando relevante.", "Add descriptive ALT text to all images, including the keyword when relevant."}, score, PriorityMedium, lang))
	} else {
		issues = append(issues, issue("image_alt", bi{"Não há imagens no conteúdo.", "No images in the content."}, bi{"Adicione ao menos uma imagem relevante com texto ALT descritivo.", "Add at least one relevant image with descriptive ALT text."}, score, PriorityLow, lang))
	}
	return clampScore(score), withAlt, issues
}

func analyzeSlug(slug, keyword, lang string) (float64, []AuditIssue) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return 0, []AuditIssue{issue("slug", bi{"O slug está vazio.", "The slug is empty."}, bi{"Defina um slug curto com hifens contendo a palavra-chave.", "Set a short hyphenated slug including the keyword."}, 0, PriorityMedium, lang)}
	}
	score := 60.0
	var issues []AuditIssue
	if len(slug) <= 60 && !strings.Contains(slug, " ") {
		score = 90
	} else {
		issues = append(issues, issue("slug", bi{"O slug é longo ou contém espaços.", "The slug is long or contains spaces."}, bi{"Use um slug curto com palavras separadas por hifens.", "Use a short slug with hyphen-separated words."}, score, PriorityMedium, lang))
	}
	if keyword != "" && strings.Contains(strings.ToLower(slug), strings.ToLower(strings.ReplaceAll(keyword, " ", "-"))) {
		score = 100
	}
	return clampScore(score), issues
}

func analyzeSchema(content, lang string) (float64, []AuditIssue) {
	if jsonLDRE.MatchString(content) {
		return 100, nil
	}
	return 0, []AuditIssue{issue("schema", bi{"Não há dados estruturados (JSON-LD).", "No structured data (JSON-LD)."}, bi{"Adicione dados estruturados JSON-LD (Article, Breadcrumb, FAQ) à página.", "Add JSON-LD structured data (Article, Breadcrumb, FAQ) to the page."}, 0, PriorityHigh, lang)}
}

func analyzePassiveVoice(content, lang string) (float64, []AuditIssue) {
	count := 0
	if lang == "en" {
		count = len(passiveENRE.FindAllString(content, -1))
	} else {
		count = len(passivePTRE.FindAllString(content, -1))
	}
	score := clampScore(100 - float64(count)*8)
	var issues []AuditIssue
	if count > 3 {
		issues = append(issues, issue("passive_voice", bi{"Uso excessivo de voz passiva.", "Excessive use of passive voice."}, bi{"Prefira a voz ativa para tornar o texto mais direto e claro.", "Prefer active voice to make the text more direct and clear."}, score, PriorityMedium, lang))
	}
	return score, issues
}

func analyzeSentenceVariation(content, lang string) (float64, []AuditIssue) {
	if strings.TrimSpace(content) == "" {
		return 0, []AuditIssue{issue("sentence_variation", bi{"Não há conteúdo para avaliar a variação de frases.", "No content to assess sentence variation."}, bi{"Adicione o corpo do artigo para análise de variação de frases.", "Add the article body for sentence variation analysis."}, 0, PriorityMedium, lang)}
	}
	sentences := sentenceSplitRE.Split(content, -1)
	var lengths []int
	total := 0
	for _, s := range sentences {
		n := len(tokenize(s))
		if n > 0 {
			lengths = append(lengths, n)
			total += n
		}
	}
	if len(lengths) == 0 {
		return 100, nil
	}
	avg := float64(total) / float64(len(lengths))
	score := 40.0
	if avg >= 12 && avg <= 22 {
		score = 100
	} else if avg >= 9 && avg <= 28 {
		score = 70
	}
	_ = lang
	return score, nil
}

func analyzeFreshness(content, lang string) (float64, []AuditIssue) {
	matches := yearRE.FindAllString(content, -1)
	if len(matches) == 0 {
		return 30, []AuditIssue{issue("freshness", bi{"Nenhuma data encontrada no conteúdo.", "No date found in the content."}, bi{"Adicione a data de publicação ou atualização e estatísticas recentes.", "Add the publication or update date and recent statistics."}, 30, PriorityMedium, lang)}
	}
	currentYear := time.Now().Year()
	score := 60.0
	for _, m := range matches {
		var year int
		if _, err := fmt.Sscanf(m, "%d", &year); err == nil && year >= currentYear-2 && year <= currentYear+1 {
			score = 100
			break
		}
	}
	return score, nil
}

func analyzeDuplicates(ctx context.Context, qc ai.QualityChecker, content string) (float64, int) {
	if qc == nil || strings.TrimSpace(content) == "" {
		return 100, 0
	}
	blocks, err := qc.CheckDuplicateBlocks(ctx, content, 10)
	if err != nil {
		return 100, 0
	}
	return clampScore(100 - float64(len(blocks))*20), len(blocks)
}

// shingleSimilarity computes a Jaccard similarity over 3-word shingles.
func shingleSimilarity(a, b string) float64 {
	aw := tokenize(a)
	bw := tokenize(b)
	if len(aw) < 3 || len(bw) < 3 {
		return 0
	}
	shingles := func(words []string) map[string]struct{} {
		m := make(map[string]struct{})
		for i := 0; i+3 <= len(words); i++ {
			m[strings.Join(words[i:i+3], " ")] = struct{}{}
		}
		return m
	}
	sa := shingles(aw)
	sb := shingles(bw)
	inter := 0
	for k := range sa {
		if _, ok := sb[k]; ok {
			inter++
		}
	}
	union := len(sa) + len(sb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// keywordCoverage returns the fraction of keyword terms present in the content.
func keywordCoverage(content, keyword string) float64 {
	kw := strings.ToLower(strings.TrimSpace(keyword))
	if kw == "" {
		return 0
	}
	lower := strings.ToLower(content)
	terms := tokenize(kw)
	if len(terms) == 0 {
		return 0
	}
	covered := 0
	for _, t := range terms {
		if strings.Contains(lower, t) {
			covered++
		}
	}
	return float64(covered) / float64(len(terms))
}

// deriveKeyword picks the most significant term from the title as a fallback
// when no primary keyword is registered for a project.
func deriveKeyword(title string) string {
	best := ""
	for _, w := range tokenize(title) {
		if stopWords[w] || len(w) < 4 {
			continue
		}
		if len(w) > len(best) {
			best = w
		}
	}
	return best
}

func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func clampScore(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func issue(field string, message, suggestion bi, score float64, priority ImprovementPriority, lang string) AuditIssue {
	return AuditIssue{
		Field:      field,
		Issue:      message.text(lang),
		Suggestion: suggestion.text(lang),
		Score:      score,
		Priority:   string(priority),
	}
}

// extractContentText converts a post's JSONB content blocks into plain text
// with markdown heading markers. Unknown block shapes are tolerated.
func extractContentText(contentJSON []byte) string {
	var blocks []interface{}
	if err := json.Unmarshal(contentJSON, &blocks); err != nil {
		return strings.TrimSpace(string(contentJSON))
	}
	var sb strings.Builder
	walkBlocks(blocks, &sb)
	return strings.TrimSpace(sb.String())
}

func walkBlocks(blocks []interface{}, sb *strings.Builder) {
	for _, b := range blocks {
		obj, ok := b.(map[string]interface{})
		if !ok {
			continue
		}
		headingLevel := 0
		if t, ok := obj["type"].(string); ok {
			switch strings.ToLower(strings.TrimSpace(t)) {
			case "heading", "h1":
				headingLevel = 1
			case "h2":
				headingLevel = 2
			case "h3":
				headingLevel = 3
			case "h4", "h5", "h6":
				headingLevel = 4
			}
		}
		if txt, ok := obj["text"].(string); ok && strings.TrimSpace(txt) != "" {
			if headingLevel > 0 {
				sb.WriteString(strings.Repeat("#", headingLevel))
				sb.WriteString(" ")
			}
			sb.WriteString(strings.TrimSpace(txt))
			sb.WriteString("\n\n")
		}
		if content, ok := obj["content"].([]interface{}); ok {
			walkBlocks(content, sb)
		}
	}
}
