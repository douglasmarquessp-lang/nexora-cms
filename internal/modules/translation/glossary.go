package translation

import (
	"regexp"
	"strings"
)

// Glossary term application. Non-forbidden terms matching the job's language
// direction are replaced with word-boundary matching; forbidden terms are
// protected (never replaced, never localized) and reported.

type GlossaryApplyResult struct {
	Applied     int // glossary terms whose source term was found and replaced
	Protected   int // forbidden terms seen in the text (left untouched)
	TargetHits  int // terms whose target term is present in the output
	Applicable  int // total terms eligible for this language direction
	Items       []string
}

type termMatch struct {
	re *regexp.Regexp
	term GlossaryTerm
}

func applicableTerms(terms []GlossaryTerm, fromLang, toLang string) ([]termMatch, int) {
	var matches []termMatch
	var forbidden int
	for _, t := range terms {
		if t.SourceTerm == "" || t.TargetTerm == "" {
			continue
		}
		if t.SourceLanguage != fromLang || t.TargetLanguage != toLang {
			continue
		}
		if t.Forbidden {
			forbidden++
			continue
		}
		matches = append(matches, termMatch{
			re:   regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(t.SourceTerm) + `\b`),
			term: t,
		})
	}
	return matches, forbidden
}

// ApplyGlossary replaces glossary source terms with target terms in the
// translated text. Forbidden terms are detected but never modified.
func ApplyGlossary(text string, terms []GlossaryTerm, fromLang, toLang string) (string, GlossaryApplyResult) {
	res := GlossaryApplyResult{}

	matches, forbiddenCount := applicableTerms(terms, fromLang, toLang)
	res.Applicable = len(matches) + forbiddenCount

	// Forbidden term detection: word-boundary presence in the text.
	seenForbidden := make(map[string]bool)
	for _, t := range terms {
		if !t.Forbidden {
			continue
		}
		if t.SourceLanguage != fromLang || t.TargetLanguage != toLang {
			continue
		}
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(t.SourceTerm) + `\b`)
		if re.MatchString(text) {
			seenForbidden[t.SourceTerm] = true
			res.Protected++
			res.Items = append(res.Items, "protected (forbidden): "+t.SourceTerm)
		}
	}

	for _, m := range matches {
		if !m.re.MatchString(text) {
			continue
		}
		text = m.re.ReplaceAllString(text, m.term.TargetTerm)
		res.Applied++
		res.Items = append(res.Items, "glossary: "+m.term.SourceTerm+" -> "+m.term.TargetTerm)
	}

	// Target-hit counting: how many eligible target terms are present after apply.
	for _, m := range matches {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(m.term.TargetTerm) + `\b`)
		if re.MatchString(text) {
			res.TargetHits++
		}
	}

	return text, res
}

// GlossaryConsistency returns 0-100: share of eligible glossary terms whose
// target term is present in the final text (forbidden terms excluded).
func GlossaryConsistency(res GlossaryApplyResult) float64 {
	eligible := res.Applicable - res.Protected
	if eligible <= 0 {
		return 100
	}
	return clampScore(round2(float64(res.TargetHits) / float64(eligible) * 100))
}

func normalizeTerm(s string) string {
	return strings.TrimSpace(s)
}
