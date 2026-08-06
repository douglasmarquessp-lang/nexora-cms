package editorialbrain

import (
	"hash/fnv"
	"regexp"
	"sort"
	"strings"
)

// bi is a bilingual (PT/EN) string pair.
type bi struct {
	pt string
	en string
}

// b builds a bilingual pair (constructor sugar).
func b(pt, en string) bi { return bi{pt: pt, en: en} }

func (b bi) text(lang string) string {
	if lang == "en" {
		return b.en
	}
	return b.pt
}

var (
	sentenceSplitRE = regexp.MustCompile(`[.!?]+(\s+|$)`)
	markdownHRE     = regexp.MustCompile(`(?m)^(#{1,3})\s+(.+)$`)
	htmlHRE         = regexp.MustCompile(`(?i)<h([1-3])[^>]*>(.*?)</h[1-6]>`)
	htmlTagRE       = regexp.MustCompile(`<[^>]+>`)
	urlRE           = regexp.MustCompile(`https?://[^\s"'<>\)]+`)
	// versionTokenRE matches brand+version tokens ("GPT-6", "Gemini 2.5", "v4.1").
	versionTokenRE = regexp.MustCompile(`(?i)\b([a-z0-9]{2,})[\s\-]?(v?\d+(?:\.\d+){0,2})\b`)
)

// sortedSignalSet dedupes and sorts a signal list (deterministic output).
func sortedSignalSet(in []string) []string {
	set := make(map[string]bool)
	for _, s := range in {
		set[s] = true
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// normalizeHashKey lowercases and trims a string for hashing.
func normalizeHashKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// contentHash returns a stable 16-hex fingerprint of title + content.
func contentHash(title, content string) string {
	h := fnv.New64a()
	h.Write([]byte(normalizeHashKey(title) + "|" + normalizeHashKey(content)))
	return fmtHex(h.Sum64())
}

// topicHash returns a stable 16-hex fingerprint of a topic.
func topicHash(topic string) string {
	h := fnv.New64a()
	h.Write([]byte(normalizeHashKey(topic)))
	return fmtHex(h.Sum64())
}

// fmtHex formats a uint64 as a 16-char lowercase hex string.
func fmtHex(v uint64) string {
	const hexDigits = "0123456789abcdef"
	out := make([]byte, 16)
	for i := 15; i >= 0; i-- {
		out[i] = hexDigits[v&0xF]
		v >>= 4
	}
	return string(out)
}

// tokenize splits text into lowercase word tokens.
func tokenize(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') &&
			r != 'à' && r != 'á' && r != 'â' && r != 'ã' && r != 'ä' && r != 'é' && r != 'ê' &&
			r != 'ë' && r != 'í' && r != 'î' && r != 'ï' && r != 'ó' && r != 'ô' && r != 'õ' &&
			r != 'ö' && r != 'ú' && r != 'û' && r != 'ü' && r != 'ç' && r != 'À' && r != 'Á' &&
			r != 'Â' && r != 'Ã' && r != 'É' && r != 'Ê' && r != 'Í' && r != 'Ó' && r != 'Ô' &&
			r != 'Õ' && r != 'Ú' && r != 'Ç'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.ToLower(f)
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

// tokenSet returns the unique token set of a text.
func tokenSet(s string) map[string]bool {
	set := make(map[string]bool)
	for _, t := range tokenize(s) {
		set[t] = true
	}
	return set
}

// sentences splits text into sentences (non-empty, lowercased).
func sentences(text string) []string {
	raw := sentenceSplitRE.Split(text, -1)
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// clampScore clamps a score into [0,100].
func clampScore(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// round2 rounds a float to two decimals.
func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// splitParagraphs splits text into paragraphs by blank lines.
func splitParagraphs(text string) []string {
	raw := strings.Split(text, "\n\n")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// stopWords are the high-frequency words ignored by term matching.
var stopWords = map[string]bool{
	"a": true, "o": true, "e": true, "de": true, "da": true, "do": true,
	"em": true, "para": true, "com": true, "por": true, "um": true, "uma": true,
	"na": true, "no": true, "os": true, "as": true, "que": true, "ao": true,
	"the": true, "and": true, "of": true, "to": true, "in": true, "for": true,
	"on": true, "with": true, "is": true, "at": true, "from": true, "by": true,
	"sobre": true, "entre": true, "como": true, "mais": true, "mas": true,
	"or": true, "an": true, "be": true, "this": true, "that": true,
	"é": true, "são": true, "ser": true, "foi": true, "não": true, "nao": true,
	"se": true, "ou": true, "nos": true, "das": true, "dos": true, "sem": true,
	"também": true, "tambem": true, "muito": true, "pode": true, "it": true,
	"its": true, "are": true, "was": true, "were": true, "has": true, "have": true,
	"had": true, "will": true, "not": true, "but": true, "so": true, "up": true,
	"out": true, "if": true, "about": true, "what": true, "which": true,
}

// significantTokens returns the non-stopword tokens (len>=3) of a text.
func significantTokens(s string) []string {
	out := make([]string, 0)
	for _, t := range tokenize(s) {
		if len(t) >= 3 && !stopWords[t] {
			out = append(out, t)
		}
	}
	return out
}

// termOverlap returns the Jaccard overlap between two token sets.
func termOverlap(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for t := range a {
		if b[t] {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// containsAny reports whether text contains any of the substrings (case-insensitive).
func containsAny(text string, subs []string) bool {
	lower := strings.ToLower(text)
	for _, s := range subs {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}
