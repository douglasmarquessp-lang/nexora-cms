package editorialbrain

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"nexora/internal/ai"
)

// passivePTRE / passiveENRE detect passive voice constructions.
var (
	passivePTRE = regexp.MustCompile(`(?i)\b(foi|foram|é|são|era|eram|sendo|ser|sou|somos|seja|fossem)\s+(\w+(ado|ada|ido|ida|ados|adas|idos|idas)s?)\b`)
	passiveENRE = regexp.MustCompile(`(?i)\b(is|are|was|were|been|being|be)\s+(\w+(ed|en|t)\b|said|made|written|done|taken|given|seen|known)\b`)
)

// longParagraphWords is the word threshold for "huge paragraph" warnings.
const longParagraphWords = 150

// shingles returns the 4-word shingle set of a sentence.
func shingles(s string) []string {
	tokens := tokenize(s)
	if len(tokens) < 4 {
		return nil
	}
	out := make([]string, 0, len(tokens)-3)
	for i := 0; i+3 < len(tokens); i++ {
		out = append(out, strings.Join(tokens[i:i+4], " "))
	}
	return out
}

// repeatRatio returns the fraction of sentences whose 4-word shingle
// appears in another sentence (phrase repetition).
func repeatRatio(sentences []string) (float64, int) {
	seen := make(map[string]int)
	for _, s := range sentences {
		for _, sh := range shingles(s) {
			seen[sh]++
		}
	}
	repeated := 0
	for _, c := range seen {
		if c > 1 {
			repeated++
		}
	}
	if len(seen) == 0 {
		return 0, 0
	}
	ratio := float64(repeated) / float64(len(seen))
	return ratio, repeated
}

// wordRepeatRatio returns the max single-word frequency ratio.
func wordRepeatRatio(text string) (float64, int) {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return 0, 0
	}
	counts := make(map[string]int)
	max := 0
	for _, t := range tokens {
		if stopWords[t] || len(t) < 3 {
			continue
		}
		counts[t]++
		if counts[t] > max {
			max = counts[t]
		}
	}
	if len(counts) == 0 {
		return 0, 0
	}
	return float64(max) / float64(len(tokens)), max
}

// maxRepeatedWord returns the most frequent non-stopword token.
func maxRepeatedWord(text string) string {
	tokens := tokenize(text)
	counts := make(map[string]int)
	best := ""
	bestCount := 0
	for _, t := range tokens {
		if stopWords[t] || len(t) < 3 {
			continue
		}
		counts[t]++
		if counts[t] > bestCount {
			best, bestCount = t, counts[t]
		}
	}
	return best
}

// CheckFluency deterministically scores reading fluency: phrase and word
// repetition, passive voice, huge paragraphs and tiring reading. The AI
// quality checker is optional (nil-safe) and only used for readability.
func CheckFluency(ctx context.Context, text, language string, qc ai.QualityChecker) FluencyReport {
	text = strings.TrimSpace(text)
	sents := sentences(text)
	paragraphs := splitParagraphs(text)

	repRatio, repCount := repeatRatio(sents)
	wordRatio, maxWord := wordRepeatRatio(text)
	passiveCount := len(passivePTRE.FindAllString(text, -1)) + len(passiveENRE.FindAllString(text, -1))

	longParas := 0
	maxParaWords := 0
	for _, p := range paragraphs {
		wc := len(tokenize(p))
		if wc > maxParaWords {
			maxParaWords = wc
		}
		if wc > longParagraphWords {
			longParas++
		}
	}

	readability := 70.0
	if qc != nil {
		if rep, err := qc.ScoreReadabilityDetailed(ctx, text, language); err == nil && rep != nil {
			readability = rep.OverallScore
		}
	}

	sentenceRep := clampScore(100 - repRatio*100*1.6)
	wordRep := clampScore(100 - wordRatio*320)
	passiveScore := clampScore(100 - float64(passiveCount)*8)
	paraScore := clampScore(100 - float64(longParas)*10)
	if longParas == 0 && maxParaWords < 30 {
		paraScore = 90
	}

	avgSentence := 0.0
	if len(sents) > 0 {
		avgSentence = float64(len(tokenize(text))) / float64(len(sents))
	}

	overall := clampScore(round2(
		readability*0.35+sentenceRep*0.25+wordRep*0.15+passiveScore*0.15+paraScore*0.10,
	))

	issues := make([]FluencyIssue, 0)
	if repCount > 0 {
		issues = append(issues, FluencyIssue{
			Kind: "phrase_repetition", Severity: "warning",
			Message: fmt.Sprintf(
				b("Frases repetidas detectadas (%d).", "Repeated sentences detected (%d).").text(language),
				repCount),
			Score: sentenceRep,
		})
	}
	if maxWord > 5 && wordRatio > 0.06 {
		issues = append(issues, FluencyIssue{
			Kind: "word_repetition", Severity: "warning",
			Message: fmt.Sprintf(
				b("Repetição de palavras (\"%s\" aparece muitas vezes).", "Word repetition (\"%s\" appears too many times).").text(language),
				maxRepeatedWord(text)),
			Score: wordRep,
		})
	}
	if passiveCount > 3 {
		issues = append(issues, FluencyIssue{
			Kind: "passive_voice", Severity: "info",
			Message: fmt.Sprintf(
				b("Excesso de voz passiva (%d ocorrências).", "Excessive passive voice (%d occurrences).").text(language),
				passiveCount),
			Score: passiveScore,
		})
	}
	if longParas > 0 {
		issues = append(issues, FluencyIssue{
			Kind: "long_paragraph", Severity: "warning",
			Message: fmt.Sprintf(
				b("%d parágrafos muito longos (acima de %d palavras).", "%d huge paragraphs (above %d words).").text(language),
				longParas, longParagraphWords),
			Score: paraScore,
		})
	}
	if readability < 50 {
		issues = append(issues, FluencyIssue{
			Kind: "reading_fatigue", Severity: "warning",
			Message: b("Leitura cansativa (legibilidade baixa).", "Tiring reading (low readability).").text(language),
			Score:   readability,
		})
	}

	return FluencyReport{
		OverallScore:      overall,
		ReadabilityScore:  readability,
		SentenceRepetition: sentenceRep,
		WordRepetition:    wordRep,
		PassiveVoice:      passiveScore,
		ParagraphScore:    paraScore,
		AvgSentenceLength: round2(avgSentence),
		MaxParagraphWords: maxParaWords,
		RepeatedSentences: repCount,
		PassiveCount:      passiveCount,
		LongParagraphs:    longParas,
		Issues:            issues,
	}
}
