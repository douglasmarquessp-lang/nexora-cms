package editorialbrain

import (
)

// importantClaimRE selects sentences that look like factual claims.
var importantClaimRE = map[string]bool{
	"cost": true, "price": true, "custa": true, "aumentou": true, "cresceu": true,
	"caiu": true, "lancou": true, "lançou": true, "lançado": true, "lançada": true,
	"segundo": true, "according": true, "de acordo": true, "relatório": true,
	"report": true, "estudo": true, "study": true, "versão": true, "version": true,
	"vantagem": true, "benefício": true, "beneficio": true, "atualizaç": true,
	"atualizac": true, "release": true, "integrado": true, "suporta": true,
	"suporte": true, "supports": true, "disponível": true, "disponivel": true,
	"available": true, "anunciou": true, "released": true, "novo": true, "nova": true,
}

// isImportantClaim decides whether a sentence must carry evidence.
func isImportantClaim(s string) bool {
	tokens := tokenize(s)
	hasNumber := false
	hasClaimWord := false
	for _, t := range tokens {
		if isNumeric(t) {
			hasNumber = true
		}
		if importantClaimRE[t] {
			hasClaimWord = true
		}
	}
	return hasNumber || hasClaimWord || len(tokens) > 18
}

// isNumeric reports whether a token is made only of digits.
func isNumeric(t string) bool {
	if t == "" {
		return false
	}
	for _, r := range t {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// factCorpus converts facts + sources into a searchable corpus.
func factCorpus(facts []FactEntry, sources []SourceRef) []struct {
	title   string
	text    string
	url     string
	score   int
	verified bool
} {
	out := make([]struct {
		title    string
		text     string
		url      string
		score    int
		verified bool
	}, 0, len(facts)+len(sources))
	for _, f := range facts {
		text := f.Entity + " " + f.Value
		score := f.Confidence
		if score <= 0 {
			score = 50
		}
		out = append(out, struct {
			title    string
			text     string
			url      string
			score    int
			verified bool
		}{title: f.Entity, text: text, url: f.SourceURL, score: score, verified: f.Confidence >= 80})
	}
	for _, s := range sources {
		out = append(out, struct {
			title    string
			text     string
			url      string
			score    int
			verified bool
		}{title: s.Title, text: s.Snippet, url: s.URL, score: s.ReliabilityScore, verified: s.IsVerified})
	}
	return out
}

// LinkEvidence deterministically ties important claims to research facts and
// sources. Every verified claim carries its supporting source; the evidence
// score is the average claim confidence (0-100).
func LinkEvidence(text string, facts []FactEntry, sources []SourceRef, language string) EvidenceReport {
	corpus := factCorpus(facts, sources)
	links := make([]EvidenceLink, 0)
	for _, s := range sentences(text) {
		if !isImportantClaim(s) {
			continue
		}
		claimSet := tokenSet(s)
		best := 0.0
		bestIdx := -1
		for i, c := range corpus {
			corpusSet := tokenSet(c.text + " " + c.title)
			overlap := termOverlap(claimSet, corpusSet)
			if overlap > best {
				best = overlap
				bestIdx = i
			}
		}
		verified := best >= 0.5 && bestIdx >= 0
		confidence := 45.0
		note := b("Sem evidência direta", "No direct evidence").text(language)
		title, url := "", ""
		if verified {
			c := corpus[bestIdx]
			if c.score >= 90 {
				confidence = 100
			} else if c.score >= 75 {
				confidence = 90
			} else if c.verified {
				confidence = 80
			} else {
				confidence = 70
			}
			title, url = c.title, c.url
			note = b("Suportado por fonte", "Supported by source").text(language)
			if c.score >= 90 {
				note = b("Fonte oficial", "Official source").text(language)
			}
		}
		links = append(links, EvidenceLink{
			Claim:       s,
			Verified:    verified,
			SourceTitle: title,
			SourceURL:   url,
			Confidence:  confidence,
			Note:        note,
		})
	}

	verifiedCount := 0
	total := 0.0
	for _, l := range links {
		if l.Verified {
			verifiedCount++
		}
		total += l.Confidence
	}
	score := 0.0
	if len(links) > 0 {
		score = round2(total / float64(len(links)))
	} else {
		score = 100
	}
	return EvidenceReport{
		EvidenceScore: score,
		ClaimsCount:   len(links),
		VerifiedCount: verifiedCount,
		Links:         links,
	}
}
