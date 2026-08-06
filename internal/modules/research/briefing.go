package research

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"nexora/internal/ai"
)

// BuildBriefing deterministically assembles a structured briefing from ranked
// sources and the extracted fact base. No AI calls, no randomness.
func BuildBriefing(topic string, sources []ResearchSource, facts []FactBaseEntry) ResearchBriefingDoc {
	doc := ResearchBriefingDoc{
		Topic:   topic,
		Summary: fmt.Sprintf("Research on '%s' found %d facts across %d sources.", topic, len(facts), len(sources)),
	}

	byType := map[FactType][]FactBaseEntry{}
	for _, f := range facts {
		byType[f.FactType] = append(byType[f.FactType], f)
	}

	verified := 0
	var keyPoints []string
	for _, src := range sources {
		if src.IsVerified || src.ReliabilityScore >= 75 {
			verified++
		}
		if src.Title != "" {
			keyPoints = append(keyPoints, src.Title)
		}
	}

	// Key points: top source titles + top facts.
	for _, t := range byType[FactTypeNumber] {
		keyPoints = append(keyPoints, fmt.Sprintf("%s: %s", t.Entity, t.Value))
	}
	keyPoints = dedupeStrings(keyPoints)
	if len(keyPoints) > 12 {
		keyPoints = keyPoints[:12]
	}
	doc.KeyPoints = keyPoints

	for _, f := range byType[FactTypeNumber] {
		doc.Statistics = append(doc.Statistics, fmt.Sprintf("%s → %s (%s)", f.Entity, f.Value, f.SourceURL))
	}
	doc.Statistics = dedupeStrings(doc.Statistics)

	for _, f := range byType[FactTypeDate] {
		doc.Dates = append(doc.Dates, fmt.Sprintf("%s (%s)", f.Value, f.SourceURL))
	}
	doc.Dates = dedupeStrings(doc.Dates)

	for _, f := range byType[FactTypeCompany] {
		doc.Companies = append(doc.Companies, f.Entity)
	}
	doc.Companies = dedupeStrings(doc.Companies)

	for _, f := range byType[FactTypeProduct] {
		doc.Products = append(doc.Products, f.Entity)
	}
	doc.Products = dedupeStrings(doc.Products)

	for _, f := range byType[FactTypeEvent] {
		doc.DataFound = append(doc.DataFound, f.Value)
	}
	doc.DataFound = dedupeStrings(doc.DataFound)

	switch {
	case verified >= 3:
		doc.Conclusions = []string{
			"Findings are corroborated by multiple reliable sources (score >= 75).",
			fmt.Sprintf("%d of %d sources are verified or authoritative.", verified, len(sources)),
		}
	case verified >= 1:
		doc.Conclusions = []string{
			"At least one authoritative source confirms the core facts; cross-check remaining claims.",
		}
	default:
		doc.Conclusions = []string{
			"No authoritative sources found — treat all claims as unverified and cross-check before publishing.",
		}
	}

	return doc
}

func dedupeStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		k := strings.ToLower(strings.TrimSpace(s))
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, s)
	}
	return out
}

// BuildBriefingAI attempts AI-assisted briefing synthesis; falls back to the
// deterministic builder when AI is unavailable or returns unparsable JSON.
func (s *Service) BuildBriefingAI(ctx context.Context, topic, language string, sources []ResearchSource, facts []FactBaseEntry) ResearchBriefingDoc {
	deterministic := BuildBriefing(topic, sources, facts)
	if s.aiManager == nil {
		return deterministic
	}

	var corpus []SourceText
	for _, src := range sources {
		corpus = append(corpus, SourceText{Title: src.Title, Snippet: src.Summary, URL: src.URL})
	}

	promptID := ai.PromptTypeDeepResearch
	if language == "pt" {
		promptID = ai.PromptTypeDeepResearch + "_pt"
	}

	req, err := s.aiManager.Prompts().Build(ctx, promptID, map[string]string{
		"topic":   topic,
		"sources": buildSourceCorpus(corpus),
	})
	if err != nil {
		return deterministic
	}

	result, err := s.aiManager.Generate(ctx, *req)
	if err != nil {
		return deterministic
	}

	var parsed ResearchBriefingDoc
	if err := parseJSONObject(result.Content, &parsed); err != nil {
		return deterministic
	}
	if strings.TrimSpace(parsed.Summary) == "" {
		return deterministic
	}
	parsed.Topic = topic
	return parsed
}

func parseJSONObject(text string, out interface{}) error {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return fmt.Errorf("no JSON object found")
	}
	return json.Unmarshal([]byte(text[start:end+1]), out)
}

// rankSources sorts sources by reliability then relevance (deterministic).
func rankSources(sources []ResearchSource) []ResearchSource {
	sorted := make([]ResearchSource, len(sources))
	copy(sorted, sources)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].ReliabilityScore != sorted[j].ReliabilityScore {
			return sorted[i].ReliabilityScore > sorted[j].ReliabilityScore
		}
		return sorted[i].RelevanceScore > sorted[j].RelevanceScore
	})
	return sorted
}
