package editorialbrain

import (
	"strings"
)

// blockOfSentence returns the name of the block containing sentence text.
type textBlock struct {
	name     string
	content  []string
	evidence int
	official int
}

// splitBlocks splits the article into blocks by markdown/HTML headings.
// The first block without a heading is the introduction; the last is the
// conclusion when no closing heading exists.
func splitBlocks(text string, language string) []textBlock {
	blocks := make([]textBlock, 0)
	raw := strings.Split(text, "\n")
	current := textBlock{}
	flush := func() {
		if len(current.content) > 0 {
			blocks = append(blocks, current)
			current = textBlock{}
		}
	}
	for _, line := range raw {
		trimmed := strings.TrimSpace(line)
		if name, ok := headingName(trimmed); ok {
			flush()
			current = textBlock{name: name}
			continue
		}
		if trimmed != "" {
			current.content = append(current.content, trimmed)
		}
	}
	flush()

	if len(blocks) == 0 {
		blocks = append(blocks, textBlock{name: b("Artigo", "Article").text(language), content: sentences(text)})
	} else if !strings.HasPrefix(blocks[0].name, "#") && blocks[0].name != b("Introdução", "Introduction").text(language) {
		blocks[0].name = b("Introdução", "Introduction").text(language)
	}
	return blocks
}

// headingName extracts a heading name from a markdown or HTML heading line.
func headingName(line string) (string, bool) {
	if m := markdownHRE.FindStringSubmatch(line); m != nil {
		return strings.TrimSpace(m[2]), true
	}
	if m := htmlHRE.FindStringSubmatch(line); m != nil {
		name := htmlTagRE.ReplaceAllString(m[2], "")
		return strings.TrimSpace(name), true
	}
	return "", false
}

// ScoreBlocks deterministically scores each block (0-100) by evidence
// density: blocks whose claims are supported by official or verified sources
// score higher; blocks with few evidence links score low.
func ScoreBlocks(text string, evidence EvidenceReport, language string) []BlockScore {
	blocks := splitBlocks(text, language)
	claimBlock := make([][]EvidenceLink, len(blocks))
	for _, l := range evidence.Links {
		for i, blk := range blocks {
			if sentenceInBlock(l.Claim, blk.content) {
				claimBlock[i] = append(claimBlock[i], l)
				break
			}
		}
	}

	out := make([]BlockScore, 0, len(blocks))
	for i, blk := range blocks {
		links := claimBlock[i]
		verified := 0
		official := 0
		for _, l := range links {
			if l.Verified {
				verified++
			}
			if l.Note == b("Fonte oficial", "Official source").text(language) {
				official++
			}
		}
		score := 40.0
		score += float64(verified) * 15
		if official > 0 {
			score += 10
		}
		if len(blocks) == 1 {
			score += 5
		}
		score = clampScore(round2(score))
		note := b("Poucas evidências", "Few evidence").text(language)
		if verified > 0 {
			note = b("Evidências: %d", "Evidence: %d").text(language)
			note = strings.Replace(note, "%d", itoa(verified), 1)
		}
		if official > 0 {
			note = b("Fonte oficial", "Official source").text(language)
		}
		out = append(out, BlockScore{
			Block:         blk.name,
			Score:         score,
			EvidenceCount: verified,
			Note:          note,
		})
	}
	return out
}

// sentenceInBlock reports whether a claim sentence belongs to a block.
func sentenceInBlock(claim string, content []string) bool {
	needle := significantTokens(claim)
	if len(needle) == 0 {
		return false
	}
	for _, c := range content {
		hay := tokenSet(c)
		hits := 0
		for _, t := range needle {
			if hay[t] {
				hits++
			}
		}
		if float64(hits)/float64(len(needle)) >= 0.5 {
			return true
		}
	}
	return false
}

// itoa is a tiny int formatter (avoids strconv import noise).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
