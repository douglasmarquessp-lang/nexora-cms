package seoengine

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"nexora/internal/ai"
)

// Content gap dimensions that a complete article should cover. Each dimension
// maps to a regex that detects coverage in the article text.
var gapDimensions = []struct {
	name  string
	regex *regexp.Regexp
	terms []string
}{
	{"price", regexp.MustCompile(`(?i)(r\$\s?\d|us\$\s?\d|usd\s?\d|\beur\s?\d|preço|preco|\bprice\b|\bcost\b|gratis|grátis|\bfree\b|assinatura|subscription)`), []string{"preço", "preco", "price", "custo", "cost"}},
	{"availability", regexp.MustCompile(`(?i)\b(disponível|disponivel|available|lançamento|lancamento|release|pre-order|pré-venda|pre-venda|quando)\b`), []string{"disponibilidade", "availability", "disponível", "disponivel", "available"}},
	{"requirements", regexp.MustCompile(`(?i)\b(requisitos|requirements|compatível|compativel|compatible|funciona|suporta|requires|necessário|necessario)\b`), []string{"requisitos", "requirements", "compatibilidade", "requirements"}},
	{"limitations", regexp.MustCompile(`(?i)\b(limitações|limitacoes|limitations|desvantagens|disadvantages|problemas|issues|não suporta|nao suporta|does not support)\b`), []string{"limitações", "limitacoes", "limitations", "desvantagens", "disadvantages"}},
	{"roadmap", regexp.MustCompile(`(?i)\b(roadmap|futuro|future|próximas|proximas|upcoming|planejado|planned|próximo|proximo)\b`), []string{"roadmap", "futuro", "future", "planejado", "planned", "upcoming"}},
	{"comparison", regexp.MustCompile(`(?i)\b(alternativa|alternative|comparado|compared|vs\.|versus|melhor que|better than|em vez de|instead of)\b`), []string{"alternativas", "alternatives", "comparação", "comparison", "vs"}},
	{"installation", regexp.MustCompile(`(?i)\b(instalação|instalacao|installation|setup|configuração|configuracao|configuration|passo a passo|step by step|guia)\b`), []string{"instalação", "instalacao", "installation", "setup", "configuração", "configuracao"}},
	{"support", regexp.MustCompile(`(?i)\b(suporte|support|ajuda|help|documentação|documentacao|documentation|comunidade|community|contato|contact)\b`), []string{"suporte", "support", "documentação", "documentacao", "documentation", "ajuda"}},
}

// ContentGapReport lists which fact-base dimensions the article covers and
// which are missing, with suggested fill content.
type ContentGapReport struct {
	Topic          string   `json:"topic"`
	Covered        []string `json:"covered"`
	Missing        []string `json:"missing"`
	MissingCount   int      `json:"missing_count"`
	CoverageRatio  float64  `json:"coverage_ratio"` // 0-1
	Suggestions    []string `json:"suggestions,omitempty"`
	AIAdditions    []string `json:"ai_additions,omitempty"`
}

// DetectContentGaps compares an article against the canonical fact-base
// dimensions. Pure and deterministic: dimension is "covered" when its regex or
// terms appear in the text.
func DetectContentGaps(text string) *ContentGapReport {
	lower := strings.ToLower(text)
	covered := []string{}
	missing := []string{}
	suggestions := []string{}
	for _, d := range gapDimensions {
		matched := d.regex.MatchString(lower)
		if !matched {
			for _, t := range d.terms {
				if strings.Contains(lower, t) {
					matched = true
					break
				}
			}
		}
		if matched {
			covered = append(covered, d.name)
		} else {
			missing = append(missing, d.name)
			suggestions = append(suggestions, suggestionForGap(d.name))
		}
	}

	total := len(gapDimensions)
	ratio := 0.0
	if total > 0 {
		ratio = float64(len(covered)) / float64(total)
	}
	return &ContentGapReport{
		Covered:       covered,
		Missing:       missing,
		MissingCount:  len(missing),
		CoverageRatio: round2(ratio),
		Suggestions:   suggestions,
	}
}

func suggestionForGap(gap string) string {
	switch gap {
	case "price":
		return "Adicione a faixa de preço ou custo (gratuito/pago, planos)."
	case "availability":
		return "Informe a disponibilidade e a data de lançamento."
	case "requirements":
		return "Liste os requisitos mínimos (hardware, software, conta)."
	case "limitations":
		return "Descreva as limitações e desvantagens conhecidas."
	case "roadmap":
		return "Mencione o roadmap e as funcionalidades planejadas."
	case "comparison":
		return "Compare com alternativas populares do mercado."
	case "installation":
		return "Adicione um passo a passo de instalação/configuração."
	case "support":
		return "Inclua canais de suporte e documentação oficial."
	}
	return "Adicione informações ausentes sobre " + gap + "."
}

// GetFactBase reads the research fact base for a topic (from research_sources
// of the site). Deterministic topic matching over titles and snippets.
func (s *Service) GetFactBase(ctx context.Context, siteID uuid.UUID, topic string, limit int) ([]string, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}

	rows, err := p.Query(ctx,
		`SELECT COALESCE(title,''), COALESCE(snippet,'') FROM research_sources
		 WHERE site_id = $1 AND COALESCE(title,'') <> ''
		 ORDER BY created_at DESC LIMIT $2`,
		siteID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list fact base sources: %w", err)
	}
	defer rows.Close()

	keywords := tokenize(topic)
	facts := []string{}
	for rows.Next() {
		var title, snippet string
		if err := rows.Scan(&title, &snippet); err != nil {
			return nil, fmt.Errorf("failed to scan fact source: %w", err)
		}
		coverage := keywordCoverage(title+" "+snippet, topic)
		if len(keywords) == 0 || coverage >= 0.15 {
			facts = append(facts, strings.TrimSpace(title+": "+snippet))
		}
	}
	if len(facts) > limit {
		facts = facts[:limit]
	}
	return facts, nil
}

// FillContentGapsAI asks the AI manager to write a short paragraph for each
// missing gap dimension. Falls back to the deterministic suggestions when AI
// is nil or returns unparsable output.
func (s *Service) FillContentGapsAI(ctx context.Context, siteID uuid.UUID, topic string, gaps *ContentGapReport) *ContentGapReport {
	if gaps == nil || len(gaps.Missing) == 0 {
		return gaps
	}
	if s.aiManager == nil {
		gaps.AIAdditions = nil
		return gaps
	}

	facts, err := s.GetFactBase(ctx, siteID, topic, 15)
	if err != nil {
		facts = nil
	}

	var additions []string
	for _, gap := range gaps.Missing {
		prompt := buildGapFillPrompt(topic, gap, facts)
		resp, err := s.aiManager.Generate(ctx, ai.CompletionRequest{
			Prompt: prompt,
		})
		if err != nil || resp == nil {
			continue
		}
		text := strings.TrimSpace(resp.Content)
		if text == "" || strings.HasPrefix(text, "Mock response") {
			continue
		}
		additions = append(additions, text)
	}
	if len(additions) > 0 {
		gaps.AIAdditions = additions
	}
	return gaps
}

// parseGapRequest decodes a ContentGapRequest body (used by the handler).
func parseGapRequest(body []byte) (topic string, ok bool) {
	var req struct {
		Topic string `json:"topic"`
	}
	if err := json.Unmarshal(body, &req); err != nil || req.Topic == "" {
		return "", false
	}
	return req.Topic, true
}

func detectArticleLanguage(s string) string {
	if strings.ContainsAny(strings.ToLower(s), "ãõçâêáéíóú") {
		return "pt"
	}
	return "en"
}

func buildGapFillPrompt(topic, gap string, facts []string) string {
	var sb strings.Builder
	sb.WriteString("Escreva um parágrafo curto (2-4 frases) em PT-BR para um artigo sobre ")
	sb.WriteString(topic)
	sb.WriteString(" cobrindo a dimensão: ")
	sb.WriteString(gap)
	if len(facts) > 0 {
		sb.WriteString("\nFatos disponíveis:\n")
		for _, f := range facts {
			sb.WriteString("- ")
			sb.WriteString(f)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\nNão invente números; use apenas os fatos fornecidos.")
	return sb.String()
}
