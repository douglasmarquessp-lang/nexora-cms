package editorialbrain

import (
	"fmt"
	"strings"
)

// CheckSemantic deterministically verifies semantic SEO: important entities,
// related concepts, missing terms, FAQ coverage and natural synonyms.
// `entities` and `concepts` come from the research briefing (DB-free input);
// when empty, they are derived from the topic itself.
func CheckSemantic(topic, text, language string, entities, concepts []string, faqCoverage float64) SemanticReport {
	lower := strings.ToLower(text)

	if len(entities) == 0 {
		entities = deriveEntities(topic)
	}
	if len(concepts) == 0 {
		concepts = deriveConcepts(topic)
	}

	entitiesFound := make([]string, 0)
	entitiesMissing := make([]string, 0)
	for _, e := range entities {
		if e == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(e)) {
			entitiesFound = append(entitiesFound, e)
		} else {
			entitiesMissing = append(entitiesMissing, e)
		}
	}

	conceptsMissing := make([]string, 0)
	for _, c := range concepts {
		if c == "" {
			continue
		}
		if !strings.Contains(lower, strings.ToLower(c)) {
			conceptsMissing = append(conceptsMissing, c)
		}
	}

	missingTerms := make([]string, 0)
	for _, t := range significantTokens(topic) {
		if !strings.Contains(lower, t) {
			missingTerms = append(missingTerms, t)
		}
	}

	synonymVariety := synonymVarietyScore(text)

	entityScore := 100.0
	if len(entities) > 0 {
		entityScore = clampScore(float64(len(entitiesFound)) / float64(len(entities)) * 100)
	}
	conceptScore := 100.0
	if len(concepts) > 0 {
		conceptScore = clampScore(float64(len(concepts)-len(conceptsMissing)) / float64(len(concepts)) * 100)
	}
	termsScore := 100.0
	if len(missingTerms) > 0 {
		termsScore = clampScore(100 - float64(len(missingTerms))*15)
	}

	overall := clampScore(round2(
		entityScore*0.25+conceptScore*0.25+termsScore*0.15+faqCoverage*0.20+synonymVariety*0.15,
	))

	issues := make([]SemanticIssue, 0)
	for _, e := range entitiesMissing {
		issues = append(issues, SemanticIssue{
			Kind: "entity_missing", Term: e,
			Message: fmt.Sprintf(b("Entidade importante não mencionada: %s", "Important entity not mentioned: %s").text(language), e),
		})
	}
	for _, c := range conceptsMissing {
		issues = append(issues, SemanticIssue{
			Kind: "concept_missing", Term: c,
			Message: fmt.Sprintf(b("Conceito relacionado não abordado: %s", "Related concept not covered: %s").text(language), c),
		})
	}
	for _, t := range missingTerms {
		issues = append(issues, SemanticIssue{
			Kind: "term_missing", Term: t,
			Message: fmt.Sprintf(b("Termo do assunto ausente: %s", "Topic term missing: %s").text(language), t),
		})
	}
	if faqCoverage < 100 {
		issues = append(issues, SemanticIssue{
			Kind: "faq_coverage", Term: "",
			Message: fmt.Sprintf(b("Perguntas frequentes parcialmente respondidas (%.0f%%).", "Frequently asked questions partially answered (%.0f%%).").text(language), faqCoverage),
		})
	}
	if synonymVariety < 60 {
		issues = append(issues, SemanticIssue{
			Kind: "synonym_variety", Term: "",
			Message: b("Pouca variedade de sinônimos; vocabulário repetitivo.", "Low synonym variety; repetitive vocabulary.").text(language),
		})
	}

	return SemanticReport{
		SemanticScore:   overall,
		EntitiesFound:   entitiesFound,
		EntitiesMissing: entitiesMissing,
		ConceptsMissing: conceptsMissing,
		MissingTerms:    missingTerms,
		FaqCoverage:     round2(faqCoverage),
		SynonymVariety:  round2(synonymVariety),
		Issues:          issues,
	}
}

// deriveEntities extracts candidate entities from the topic (first words,
// capitalized terms).
func deriveEntities(topic string) []string {
	out := make([]string, 0)
	seen := make(map[string]bool)
	for _, w := range strings.Fields(topic) {
		clean := strings.Trim(w, ".,!?;:()")
		if clean == "" {
			continue
		}
		if len(clean) >= 4 && (strings.ToUpper(clean[:1]) == clean[:1]) && !stopWords[strings.ToLower(clean)] {
			if !seen[clean] {
				seen[clean] = true
				out = append(out, clean)
			}
		}
	}
	if len(out) == 0 {
		for _, t := range significantTokens(topic) {
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	if len(out) > 4 {
		out = out[:4]
	}
	return out
}

// deriveConcepts derives related concepts from the topic's significant tokens.
func deriveConcepts(topic string) []string {
	tokens := significantTokens(topic)
	if len(tokens) > 3 {
		tokens = tokens[:3]
	}
	return tokens
}

// synonymVarietyScore measures vocabulary variety: the ratio of unique
// significant tokens to total significant tokens, scaled to 0-100.
func synonymVarietyScore(text string) float64 {
	tokens := significantTokens(text)
	if len(tokens) == 0 {
		return 60
	}
	set := make(map[string]bool)
	for _, t := range tokens {
		set[t] = true
	}
	ratio := float64(len(set)) / float64(len(tokens))
	return clampScore(round2(ratio * 140))
}
