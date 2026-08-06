package translation

import (
	"regexp"
	"strings"
)

// Deterministic language detection, slug/keyword derivation, and content block
// conversion helpers. Zero randomness, zero API calls.

var ptStopWords = map[string]bool{
	"que": true, "não": true, "uma": true, "para": true, "como": true, "mais": true,
	"mas": true, "por": true, "com": true, "dos": true, "das": true, "foi": true,
	"são": true, "você": true, "está": true, "também": true, "entre": true,
	"sobre": true, "até": true, "já": true, "ainda": true, "muito": true,
	"depois": true, "antes": true, "assim": true, "aqui": true, "isso": true,
	"essa": true, "este": true, "quando": true, "onde": true, "eles": true,
	"elas": true, "seus": true, "suas": true, "pode": true, "podem": true,
	"ser": true, "tem": true, "têm": true, "era": true, "eram": true, "todo": true,
	"toda": true, "todos": true, "todas": true, "outra": true, "outro": true,
	"nossa": true, "nosso": true, "esses": true, "essas": true, "estes": true,
}

var enStopWords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "this": true, "that": true,
	"from": true, "have": true, "you": true, "are": true, "was": true, "were": true,
	"not": true, "but": true, "your": true, "will": true, "would": true, "can": true,
	"than": true, "them": true, "their": true, "there": true, "they": true, "which": true,
	"when": true, "where": true, "what": true, "our": true, "into": true, "over": true,
	"after": true, "before": true, "could": true, "should": true, "other": true,
	"more": true, "most": true, "also": true, "been": true, "being": true, "has": true,
	"had": true, "did": true, "does": true, "here": true, "then": true,
}

var ptDiacritics = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a', 'é': 'e', 'è': 'e', 'ê': 'e',
	'ẽ': 'e', 'ë': 'e', 'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i', 'ó': 'o', 'ò': 'o',
	'ô': 'o', 'õ': 'o', 'ö': 'o', 'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u', 'ç': 'c',
	'ñ': 'n', 'ý': 'y',
}

func normalizeDiacritics(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			continue
		}
		if repl, ok := ptDiacritics[r]; ok {
			b.WriteRune(repl)
			continue
		}
		b.WriteByte(' ')
	}
	return b.String()
}

// DetectLanguage returns "pt" or "en" with a 0-1 confidence, using stopword and
// diacritic frequency heuristics. Deterministic for the same input.
func DetectLanguage(text string) (string, float64) {
	words := tokenize(text)
	if len(words) == 0 {
		return "en", 0.5
	}

	var ptHits, enHits int
	for _, w := range words {
		if ptStopWords[w] {
			ptHits++
		}
		if enStopWords[w] {
			enHits++
		}
	}

	// Diacritics are a strong PT signal: ã, õ, ç, á, é, í, ó, ú appear constantly
	// in Portuguese and are nearly absent from English.
	var diacriticHits int
	for _, r := range strings.ToLower(text) {
		switch r {
		case 'ã', 'õ', 'ç', 'â', 'ê', 'ô', 'á', 'à', 'é', 'í', 'ó', 'ú', 'ü':
			diacriticHits++
		}
	}

	total := float64(ptHits + enHits + diacriticHits)
	if total == 0 {
		// No discriminating signal: tie-break to English only when the text is
		// short; otherwise neutral.
		if len(words) <= 10 {
			return "en", 0.5
		}
		return "pt", 0.5
	}

	ptScore := float64(ptHits) + float64(diacriticHits)*1.5
	enScore := float64(enHits)

	if ptScore > enScore {
		conf := (ptScore - enScore) / (ptScore + enScore)
		return "pt", clamp01(conf)
	}
	conf := (enScore - ptScore) / (ptScore + enScore)
	return "en", clamp01(conf)
}

var nonSlugRe = regexp.MustCompile(`[^a-z0-9]+`)

// GenerateSlug builds a language-aware slug: lowercase, diacritics removed,
// non-alphanumeric collapsed to hyphens, trailing hyphens trimmed.
func GenerateSlug(title string, language string) string {
	slug := nonSlugRe.ReplaceAllString(normalizeDiacritics(title), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "article"
	}
	if language == "pt" && len(slug) > 70 {
		slug = slug[:70]
		slug = strings.TrimRight(slug, "-")
	}
	return slug
}

func tokenize(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	out := words[:0]
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?\"'()[]{}")
		if w != "" {
			out = append(out, w)
		}
	}
	return out
}

func isStopWord(language, word string) bool {
	if language == "pt" {
		return ptStopWords[word]
	}
	return enStopWords[word]
}

// DeriveKeyword picks the longest non-stopword token from a title. Used as a
// deterministic primary keyword when the SEO review has no AI-generated keyword.
func DeriveKeyword(title, language string) string {
	best := ""
	for _, w := range tokenize(title) {
		if isStopWord(language, w) {
			continue
		}
		if len(w) > len(best) {
			best = w
		}
	}
	if best == "" {
		words := tokenize(title)
		if len(words) > 0 {
			return words[0]
		}
		return "article"
	}
	return best
}

// blocksToText converts posts-style JSONB content blocks to markdown-ish text.
// Only heading/text blocks are preserved; everything else is skipped.
func blocksToText(blocks []interface{}) string {
	var sb strings.Builder
	for _, b := range blocks {
		m, ok := b.(map[string]interface{})
		if !ok {
			continue
		}
		blockType, _ := m["type"].(string)
		text, _ := m["text"].(string)
		if text == "" {
			continue
		}
		switch blockType {
		case "heading":
			sb.WriteString("# ")
			sb.WriteString(text)
		case "text":
			sb.WriteString(text)
		default:
			continue
		}
		sb.WriteString("\n\n")
	}
	return strings.TrimSpace(sb.String())
}

// textToBlocks converts markdown-ish text to posts-style JSONB content blocks.
// "# " lines become heading blocks, everything else becomes text blocks.
func textToBlocks(text string) []interface{} {
	blocks := make([]interface{}, 0, 16)
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			blocks = append(blocks, map[string]interface{}{
				"type": "heading",
				"text": strings.TrimSpace(strings.TrimPrefix(trimmed, "# ")),
			})
			continue
		}
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			blocks = append(blocks, map[string]interface{}{
				"type": "heading",
				"text": strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "## "), "### ")),
			})
			continue
		}
		blocks = append(blocks, map[string]interface{}{
			"type": "text",
			"text": trimmed,
		})
	}
	if len(blocks) == 0 {
		blocks = append(blocks, map[string]interface{}{"type": "text", "text": ""})
	}
	return blocks
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
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
