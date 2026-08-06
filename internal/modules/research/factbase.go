package research

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"nexora/internal/ai"
)

var (
	versionRe   = regexp.MustCompile(`\b(?i:v)?\d+\.\d+(\.\d+)?([-.]?(alpha|beta|rc|stable|latest)\d*)?\b`)
	priceRe     = regexp.MustCompile(`(?i)\b(US\$\s?\d[\d.,]*|R\$\s?\d[\d.,]*|\$\s?\d[\d.,]*|\d[\d.,]*\s?(USD|EUR|BRL|dólares|dollars|reais))\b`)
	dateISORe   = regexp.MustCompile(`\b\d{4}[-/]\d{1,2}[-/]\d{1,2}\b|\b\d{1,2}/\d{1,2}/\d{4}\b`)
	dateLongRe  = regexp.MustCompile(`(?i)\b(january|february|march|april|may|june|july|august|september|october|november|december|janeiro|fevereiro|março|abril|maio|junho|julho|agosto|setembro|outubro|novembro|dezembro)\s+\d{1,2},?\s+(de\s+)?\d{4}\b|\b\d{1,2}\s+de\s+(janeiro|fevereiro|março|abril|maio|junho|julho|agosto|setembro|outubro|novembro|dezembro)\s+de\s+\d{4}\b`)
	numberRe    = regexp.MustCompile(`(?i)\b\d[\d.,]*\s*(%|million|billion|trillion|milh[õo]es|bilh[õo]es|users|usuários|dowloads?|downloads?|devices|dispositivos|parâmetros|parameters|tokens|requests|dólares|dollars|reais|anos|years|vezes|times)?\b`)
	eventRe     = regexp.MustCompile(`(?i)\b(lançou|lançado|released|announced|launched|acquired|acquisição|adquiriu|unveiled|presented|apresentou|anunciou|introduziu|introduced|launched)\b[^.!?\n]{10,160}[.!?]?`)
	techTerms   = []string{"gpt", "gemini", "claude", "llama", "llm", "api", "gpu", "cpu", "rag", "transformer", "diffusion", "fine-tuning", "fine tuning", "multimodal", "neural", "agente", "agent"}
	companyTerms = []string{"openai", "google", "deepmind", "microsoft", "anthropic", "meta", "apple", "amazon", "aws", "nvidia", "ibm", "intel", "amd", "qualcomm", "salesforce", "oracle", "sap", "adobe", "netflix", "spotify", "uber", "airbnb", "tesla", "samsung", "sony", "mozilla", "github", "cloudflare", "databricks", "hugging face", "cohere", "mistral", "perplexity", "tiktok", "twitter", "x corp", "telegram", "petrobras", "vale", "embraer", "magalu", "americanas", "nubank", "itau", "bradesco", "caixa"}
)

// aiFactEntry mirrors the JSON contract of the fact_base prompt.
type aiFactEntry struct {
	Type       string  `json:"type"`
	Entity     string  `json:"entity"`
	Value      string  `json:"value"`
	Source     string  `json:"source"`
	Confidence float64 `json:"confidence"`
}

// sourceText is the text used for fact extraction: snippet + title per source.
func sourceText(title, snippet string) string {
	if snippet != "" {
		return snippet
	}
	return title
}

// ExtractFactBase deterministically extracts structured facts from the given
// source texts. No AI calls, no randomness: same input → same output.
func ExtractFactBase(topic string, sources []SourceText) []FactBaseEntry {
	var facts []FactBaseEntry
	known := map[string]bool{}
	for _, src := range sources {
		text := sourceText(src.Title, src.Snippet)
		if text == "" {
			continue
		}
		facts = append(facts, extractFromText(text, src.URL, known)...)
	}
	return dedupeFacts(facts)
}

// SourceText is a minimal source input for deterministic extraction.
type SourceText struct {
	Title   string
	Snippet string
	URL     string
}

func extractFromText(text, url string, known map[string]bool) []FactBaseEntry {
	var facts []FactBaseEntry

	// Versions
	for _, m := range versionRe.FindAllString(text, -1) {
		v := strings.TrimPrefix(strings.ToLower(m), "v")
		if known[v] {
			continue
		}
		facts = append(facts, FactBaseEntry{
			FactType:   FactTypeVersion,
			Entity:     versionEntity(text, m),
			Value:      m,
			SourceURL:  url,
			Confidence: 80,
		})
		known[v] = true
	}

	// Prices
	for _, m := range priceRe.FindAllString(text, -1) {
		entity := contextWords(text, m, 4)
		if entity == "" {
			entity = "product"
		}
		facts = append(facts, FactBaseEntry{
			FactType:   FactTypePrice,
			Entity:     entity,
			Value:      m,
			SourceURL:  url,
			Confidence: 70,
		})
	}

	// Dates
	for _, m := range dateISORe.FindAllString(text, -1) {
		facts = append(facts, FactBaseEntry{
			FactType:   FactTypeDate,
			Entity:     "event",
			Value:      m,
			SourceURL:  url,
			Confidence: 60,
		})
	}
	for _, m := range dateLongRe.FindAllString(text, -1) {
		facts = append(facts, FactBaseEntry{
			FactType:   FactTypeDate,
			Entity:     "event",
			Value:      m,
			SourceURL:  url,
			Confidence: 60,
		})
	}

	// Companies (known brand terms, case-insensitive)
	lower := strings.ToLower(text)
	for _, c := range companyTerms {
		if strings.Contains(lower, c) {
			facts = append(facts, FactBaseEntry{
				FactType:   FactTypeCompany,
				Entity:     titleCase(c),
				Value:      c,
				SourceURL:  url,
				Confidence: 75,
			})
		}
	}

	// Technologies (known terms)
	for _, t := range techTerms {
		if strings.Contains(lower, t) {
			facts = append(facts, FactBaseEntry{
				FactType:   FactTypeTechnology,
				Entity:     titleCase(t),
				Value:      t,
				SourceURL:  url,
				Confidence: 70,
			})
		}
	}

	// Numbers/statistics with context
	for _, m := range numberRe.FindAllString(text, -1) {
		ctx := contextWords(text, m, 5)
		if ctx == "" {
			ctx = "metric"
		}
		facts = append(facts, FactBaseEntry{
			FactType:   FactTypeNumber,
			Entity:     ctx,
			Value:      m,
			SourceURL:  url,
			Confidence: 65,
		})
	}

	// Events (sentence-level)
	for _, m := range eventRe.FindAllString(text, -1) {
		facts = append(facts, FactBaseEntry{
			FactType:   FactTypeEvent,
			Entity:     strings.TrimSpace(m),
			Value:      m,
			SourceURL:  url,
			Confidence: 70,
		})
	}

	return facts
}

func dedupeFacts(facts []FactBaseEntry) []FactBaseEntry {
	seen := map[string]bool{}
	var out []FactBaseEntry
	for _, f := range facts {
		key := string(f.FactType) + "|" + strings.ToLower(f.Entity) + "|" + strings.ToLower(f.Value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, f)
	}
	return out
}

func versionEntity(text, ver string) string {
	idx := strings.Index(text, ver)
	if idx < 0 {
		return "software"
	}
	before := strings.TrimSpace(text[:idx])
	words := strings.Fields(before)
	if len(words) == 0 {
		return "software"
	}
	// Take up to 3 words before the version, e.g. "GPT-5 API"
	start := len(words) - 3
	if start < 0 {
		start = 0
	}
	name := strings.Join(words[start:], " ")
	if len(name) > 60 {
		name = name[len(name)-60:]
	}
	return name
}

// contextWords returns up to n words surrounding a match inside text.
func contextWords(text, match string, n int) string {
	idx := strings.Index(text, match)
	if idx < 0 {
		return ""
	}
	before := strings.Fields(text[:idx])
	start := len(before) - n
	if start < 0 {
		start = 0
	}
	after := strings.Fields(text[idx+len(match):])
	words := append(before[start:], after[:min(n, len(after))]...)
	joined := strings.TrimSpace(strings.Join(words, " "))
	if len(joined) > 80 {
		joined = joined[:80]
	}
	return joined
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ExtractFactBaseAI attempts AI-assisted fact extraction; falls back to the
// deterministic extractor when AI is unavailable or returns unparsable JSON.
func (s *Service) ExtractFactBaseAI(ctx context.Context, topic string, sources []SourceText) []FactBaseEntry {
	deterministic := ExtractFactBase(topic, sources)
	if s.aiManager == nil {
		return deterministic
	}

	req, err := s.aiManager.Prompts().Build(ctx, ai.PromptTypeFactBase, map[string]string{
		"topic":   topic,
		"content": buildSourceCorpus(sources),
	})
	if err != nil {
		return deterministic
	}

	result, err := s.aiManager.Generate(ctx, *req)
	if err != nil {
		return deterministic
	}

	var parsed []aiFactEntry
	if err := parseJSONArray(result.Content, &parsed); err != nil || len(parsed) == 0 {
		return deterministic
	}

	urlByTitle := map[string]string{}
	for _, src := range sources {
		urlByTitle[strings.ToLower(src.Title)] = src.URL
	}

	var out []FactBaseEntry
	for _, f := range parsed {
		ft := FactType(strings.ToLower(strings.TrimSpace(f.Type)))
		if !validFactType(ft) || strings.TrimSpace(f.Entity) == "" {
			continue
		}
		entry := FactBaseEntry{
			FactType:   ft,
			Entity:     strings.TrimSpace(f.Entity),
			Value:      strings.TrimSpace(f.Value),
			SourceURL:  strings.TrimSpace(f.Source),
			Confidence: int(f.Confidence),
		}
		if entry.Confidence <= 0 || entry.Confidence > 100 {
			entry.Confidence = 50
		}
		if entry.SourceURL == "" {
			entry.SourceURL = urlByTitle[strings.ToLower(f.Entity)]
		}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return deterministic
	}
	return dedupeFacts(out)
}

func validFactType(t FactType) bool {
	switch t {
	case FactTypeCompany, FactTypeProduct, FactTypeVersion, FactTypePrice,
		FactTypeDate, FactTypeEvent, FactTypeTechnology, FactTypeNumber:
		return true
	}
	return false
}

func parseJSONArray(text string, out interface{}) error {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start < 0 || end < start {
		return fmt.Errorf("no JSON array found")
	}
	return json.Unmarshal([]byte(text[start:end+1]), out)
}

func buildSourceCorpus(sources []SourceText) string {
	var b strings.Builder
	for i, src := range sources {
		fmt.Fprintf(&b, "[%d] %s\n%s\n\n", i+1, src.Title, src.Snippet)
	}
	return b.String()
}
