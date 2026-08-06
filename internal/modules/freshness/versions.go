package freshness

import (
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// facetKeywords maps each diff facet to identifying keywords (EN + PT).
var facetKeywords = map[DiffFacet][]string{
	FacetPrice:     {"price", "preço", "custo", "us$", "r$", "pricing", "assinatura"},
	FacetContext:   {"context", "contexto", "context window", "janela de contexto"},
	FacetLimits:    {"limit", "limite", "max", "máximo", "rate limit", "quota"},
	FacetAPI:       {"api", "endpoint", "sdk", "key", "chave"},
	FacetBenchmark: {"benchmark", "score", "eval", "desempenho", "performance", "mmlu"},
	FacetFeatures:  {"feature", "recurso", "função", "novo", "suporte a"},
}

// NextVersion bumps a semver-ish version string deterministically (v1.2 → v1.3).
func NextVersion(v string) string {
	maj, min, ok := parseVersionSegment(v)
	if !ok {
		return "v1"
	}
	return "v" + strconv.Itoa(maj) + "." + strconv.Itoa(min+1)
}

func splitLines(text string) []string {
	var out []string
	for _, raw := range strings.Split(text, "\n") {
		s := strings.TrimSpace(raw)
		s = strings.TrimPrefix(s, "- ")
		s = strings.TrimPrefix(s, "* ")
		s = strings.TrimLeft(s, "#")
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func extractFacetLines(text string, facet DiffFacet) []string {
	var lines []string
	kw := facetKeywords[facet]
	for _, line := range splitLines(text) {
		low := strings.ToLower(line)
		for _, k := range kw {
			if strings.Contains(low, k) {
				lines = append(lines, line)
				break
			}
		}
	}
	return lines
}

// DiffVersions compares two article texts across the known facets and reports
// which dimensions changed. Deterministic.
func DiffVersions(oldText, newText string) []VersionDiff {
	order := []DiffFacet{FacetPrice, FacetContext, FacetLimits, FacetAPI, FacetBenchmark, FacetFeatures}
	var out []VersionDiff
	for _, facet := range order {
		before := strings.Join(extractFacetLines(oldText, facet), " | ")
		after := strings.Join(extractFacetLines(newText, facet), " | ")
		changed := before != after
		if changed && (before == "" || after == "") && before == "" && after == "" {
			changed = false
		}
		out = append(out, VersionDiff{Facet: facet, Before: before, After: after, Changed: changed})
	}
	return out
}

// Fingerprint builds a stable 64-bit hex fingerprint for a topic+content.
// Same subject + same day → same fingerprint → dedupe.
func Fingerprint(topic, primaryText string) string {
	norm := strings.ToLower(strings.TrimSpace(topic + " " + firstSentence(primaryText)))
	h := fnv.New64a()
	_, _ = h.Write([]byte(norm))
	return int64Hex(h.Sum64())
}

func int64Hex(v uint64) string {
	const digits = "0123456789abcdef"
	var b [16]byte
	for i := 15; i >= 0; i-- {
		b[i] = digits[v&0xf]
		v >>= 4
	}
	return string(b[:])
}

func firstSentence(s string) string {
	idx := strings.Index(s, ".")
	if idx > 0 {
		return normalizeTokens(s[:idx])
	}
	return normalizeTokens(s)
}

func normalizeTokens(s string) string {
	return strings.Join(sortedTokens(s), " ")
}

// sortedTokens lowercases and sorts unique tokens.
func sortedTokens(s string) []string {
	set := map[string]bool{}
	var fl []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			w := string(cur)
			if !set[w] {
				set[w] = true
				fl = append(fl, w)
			}
			cur = cur[:0]
		}
	}
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur = append(cur, r)
		} else {
			flush()
		}
	}
	flush()
	sort.Strings(fl)
	return fl
}

// TokenJaccard measures near-duplicate similarity between two texts (0..1).
func TokenJaccard(a, b string) float64 {
	sa := sortedTokens(a)
	sb := sortedTokens(b)
	if len(sa) == 0 && len(sb) == 0 {
		return 1
	}
	intersection := 0
	i, j := 0, 0
	for i < len(sa) && j < len(sb) {
		if sa[i] == sb[j] {
			intersection++
			i++
			j++
		} else if sa[i] < sb[j] {
			i++
		} else {
			j++
		}
	}
	union := len(sa) + len(sb) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}