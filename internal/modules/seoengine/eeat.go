package seoengine

import (
	"regexp"
	"strings"

	"nexora/internal/ai"
)

// EEAT weights for the final score (must sum to 1.0).
const (
	eeatWeightExperience      = 0.25
	eeatWeightExpertise       = 0.30
	eeatWeightAuthority       = 0.25
	eeatWeightTrustworthiness = 0.20
)

// EEATPillar is one of the four E-E-A-T pillars with its score and the
// explanations of why points were lost.
type EEATPillar struct {
	Name   string   `json:"name"`
	Score  float64  `json:"score"`
	Issues []string `json:"issues,omitempty"`
}

// EEATReport is the full E-E-A-T analysis: four pillars + weighted final.
type EEATReport struct {
	Experience       float64     `json:"experience"`
	Expertise        float64     `json:"expertise"`
	Authoritativeness float64    `json:"authoritativeness"`
	Trustworthiness  float64     `json:"trustworthiness"`
	Final            float64     `json:"final"`
	Pillars          []EEATPillar `json:"pillars"`
}

// eeatSignal is a deterministic signal scan over the article text. Each signal
// found adds points to its pillar; when absent, `missing` explains the loss.
type eeatSignal struct {
	match   func(lower, raw string) bool
	points  float64
	missing string
}

var (
	experienceRE = regexp.MustCompile(`(?i)\b(testamos|testei|em nossos testes|na prática|na pratica|hands-on|we tested|we've tested|in our testing|first-hand|experiência prática|experiencia pratica)\b`)
	measureRE    = regexp.MustCompile(`(?i)\b\d+(\.\d+)?\s*(%|ms|s\b|min|h\b|gb|mb|kb|hz|ghz|fps|mbps|usd|brl|eur|anos|months|weeks|dias|horas)\b`)
	casestudyRE  = regexp.MustCompile(`(?i)\b(case study|caso de uso|estudo de caso|teste real|benchmark|medimos|measured)\b`)
	expertiseRE  = regexp.MustCompile(`(?i)\b(especialista|specialist|ph\.?d|doutor|engenheiro|engineer|consultor|consultant|analista|analyst|arquiteto|architect|certificado|certified|professor|pesquisador|researcher|expert)\b`)
	credentialRE = regexp.MustCompile(`(?i)\b(mestrado|masters|bacharel|bachelor|diploma|certificação|certificacao|certificado|certified|pmp|pmbok|mba)\b`)
	definedRE    = regexp.MustCompile(`(?i)(é definido como|são definidos como|is defined as|refere-se a|refere se|refers to|significa|means)\b`)
	citeRE       = regexp.MustCompile(`(?i)\b(de acordo com|according to|segundo|estudo|study|pesquisa|research|report|relatório|relatorio|publicação|publication)\b`)
	updatedRE    = regexp.MustCompile(`(?i)\b(atualizado|updated|last modified|última atualização|ultima atualizacao)\b`)
	sourceRE     = regexp.MustCompile(`(?i)\b(fontes|referências|referencias|sources|references)\b`)
	disclaimerRE = regexp.MustCompile(`(?i)\b(isenção|isencao|disclaimer|afiliação|afiliacao|affiliate|nota do autor)\b`)
	bylineRE     = regexp.MustCompile(`(?i)\b(por |by |escrito por|written by)\s+[a-zà-úç]{3,}\b`)
	promiseRE    = regexp.MustCompile(`(?i)\b(garantimos|garantia total|100% garantido|guarantee you|promessa de resultado)\b`)
)

// regexMatch adapts a case-insensitive regex to the signal signature
// (the first argument is the lowercased text).
func regexMatch(re *regexp.Regexp) func(lower, raw string) bool {
	return func(lower, raw string) bool { return re.MatchString(lower) }
}

// AnalyzeEEAT deterministically scores the four E-E-A-T pillars of an article.
// No AI calls, no randomness: same input → same output. Each pillar explains
// exactly which signals were missing and the points lost.
func AnalyzeEEAT(in ArticleAnalysisInput) *EEATReport {
	lower := strings.ToLower(in.Content)
	raw := in.Content
	wordCount := len(tokenize(in.Content))

	expScore, expIssues := analyzeEeatPillar([]eeatSignal{
		{regexMatch(experienceRE), 25, "Sem evidência de experiência prática (testes, uso real, hands-on)."},
		{regexMatch(measureRE), 25, "Sem medições ou números específicos (%, ms, GB, anos, etc.)."},
		{regexMatch(casestudyRE), 25, "Sem estudos de caso, benchmarks ou testes reais."},
		{func(l, r string) bool { return countImages(r) >= 1 }, 25, "Sem imagens/capturas que comprovem o uso."},
	}, lower, raw)

	expertiseScore, expertiseIssues := analyzeEeatPillar([]eeatSignal{
		{func(l, r string) bool { return in.AuthorName != "" || bylineRE.MatchString(l) }, 20, "Autor não identificado (sem assinatura ou autoria declarada)."},
		{regexMatch(expertiseRE), 20, "Sem sinais de qualificação (especialista, engenheiro, PhD, consultor, etc.)."},
		{regexMatch(credentialRE), 15, "Sem credenciais formais (certificações, graus, MBA)."},
		{regexMatch(definedRE), 15, "Sem definições precisas de conceitos-chave."},
		{func(l, r string) bool { return wordCount >= 800 }, 15, "Conteúdo raso (menos de 800 palavras)."},
		{func(l, r string) bool { return techTermCount(r) >= 2 }, 15, "Pouco vocabulário técnico específico do tema."},
	}, lower, raw)

	authScore, authIssues := analyzeEeatPillar([]eeatSignal{
		{func(l, r string) bool { return authoritativeExternalLinks(r) >= 1 }, 30, "Nenhum link externo para fontes de alta autoridade (rel. >= 75)."},
		{regexMatch(citeRE), 25, "Sem citações de estudos, relatórios ou fontes oficiais."},
		{regexMatch(jsonLDRE), 25, "Sem dados estruturados (JSON-LD) na página."},
		{regexMatch(yearRE), 20, "Sem referências temporais (anos) que situem as afirmações."},
	}, lower, raw)

	trustScore, trustIssues := analyzeEeatPillar([]eeatSignal{
		{regexMatch(updatedRE), 20, "Sem indicação de última atualização."},
		{func(l, r string) bool { return in.AuthorName != "" && yearRE.MatchString(raw) }, 20, "Autor e data não estão presentes juntos."},
		{regexMatch(sourceRE), 20, "Sem seção de fontes/referências."},
		{regexMatch(disclaimerRE), 15, "Sem declaração de isenção/afiliação quando aplicável."},
		{func(l, r string) bool { return !promiseRE.MatchString(l) }, 15, "Linguagem de promessa exagerada detectada."},
	}, lower, raw)

	final := eeatWeightExperience*expScore +
		eeatWeightExpertise*expertiseScore +
		eeatWeightAuthority*authScore +
		eeatWeightTrustworthiness*trustScore

	return &EEATReport{
		Experience:        round2(expScore),
		Expertise:         round2(expertiseScore),
		Authoritativeness: round2(authScore),
		Trustworthiness:   round2(trustScore),
		Final:             round2(clampScore(final)),
		Pillars: []EEATPillar{
			{Name: "Experience", Score: round2(expScore), Issues: expIssues},
			{Name: "Expertise", Score: round2(expertiseScore), Issues: expertiseIssues},
			{Name: "Authoritativeness", Score: round2(authScore), Issues: authIssues},
			{Name: "Trustworthiness", Score: round2(trustScore), Issues: trustIssues},
		},
	}
}

// analyzeEeatPillar scores one pillar from its signals: present signals add
// points (capped at 100); missing signals become the explanation list.
func analyzeEeatPillar(signals []eeatSignal, lower, raw string) (float64, []string) {
	score := 0.0
	var missing []string
	for _, sig := range signals {
		if sig.match(lower, raw) {
			score += sig.points
		} else {
			missing = append(missing, sig.missing)
		}
	}
	return clampScore(score), missing
}

func countImages(text string) int {
	n := len(mdImageRE.FindAllString(text, -1))
	n += len(htmlImageRE.FindAllString(text, -1))
	return n
}

// techTerms are the vocabulary terms that indicate technical depth.
var techTerms = []string{
	"api", "gpu", "cpu", "rag", "llm", "transformer", "diffusion", "multimodal",
	"neural", "framework", "sdk", "cli", "protocol", "latency", "throughput",
	"benchmark", "architecture", "deploy", "container", "kubernetes", "docker",
	"database", "cache", "concurrency", "authentication", "encryption",
}

func techTermCount(text string) int {
	lower := strings.ToLower(text)
	count := 0
	for _, t := range techTerms {
		if strings.Contains(lower, t) {
			count++
		}
	}
	return count
}

func aiExtractDomain(u string) string {
	return ai.ExtractDomain(u)
}

func reliabilityOfDomain(domain string) (int, string) {
	return ai.ReliabilityOfDomain(domain)
}

// authoritativeExternalLinks counts external links whose domain has a
// reliability score of at least 75 (verified/official sources).
func authoritativeExternalLinks(text string) int {
	count := 0
	seen := map[string]bool{}
	for _, m := range mdLinkRE.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		u := m[1]
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			continue
		}
		domain := aiExtractDomain(u)
		score, _ := reliabilityOfDomain(domain)
		if score >= 75 && !seen[domain] {
			count++
			seen[domain] = true
		}
	}
	bare := mdLinkRE.ReplaceAllString(text, "")
	bare = mdImageRE.ReplaceAllString(bare, "")
	for _, m := range urlRE.FindAllString(bare, -1) {
		domain := aiExtractDomain(m)
		score, _ := reliabilityOfDomain(domain)
		if score >= 75 && !seen[domain] {
			count++
			seen[domain] = true
		}
	}
	return count
}
