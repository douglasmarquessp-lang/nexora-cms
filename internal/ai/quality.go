package ai

import (
	"context"
	"math/rand"
	"strings"
)

type qualityChecker struct {
}

func NewQualityChecker() *qualityChecker {
	return &qualityChecker{}
}

func (qc *qualityChecker) ScoreGrammar(ctx context.Context, text, language string) (*ScoreResult, error) {
	wordCount := len(strings.Fields(text))
	if wordCount == 0 {
		return &ScoreResult{Score: 100, MaxScore: 100, Passed: true, Details: "empty text"}, nil
	}

	score := 85.0 + rand.Float64()*15.0
	passed := score >= 70.0
	return &ScoreResult{
		Score:    score,
		MaxScore: 100,
		Passed:   passed,
		Details:  "Grammar check completed (mock)",
	}, nil
}

func (qc *qualityChecker) ScoreSEO(ctx context.Context, text string, keywords []string) (*ScoreResult, error) {
	if len(keywords) == 0 {
		return &ScoreResult{Score: 50, MaxScore: 100, Passed: false, Details: "no keywords provided"}, nil
	}

	textLower := strings.ToLower(text)
	keywordCount := 0
	for _, kw := range keywords {
		keywordCount += strings.Count(textLower, strings.ToLower(kw))
	}
	wordCount := len(strings.Fields(text))
	density := 0.0
	if wordCount > 0 {
		density = float64(keywordCount) / float64(wordCount) * 100
	}

	score := 70.0 + rand.Float64()*30.0
	if density < 0.5 || density > 5.0 {
		score -= 20.0
	}
	if score < 0 {
		score = 0
	}

	passed := score >= 60.0
	return &ScoreResult{
		Score:    score,
		MaxScore: 100,
		Passed:   passed,
		Details:  "SEO check completed (mock)",
	}, nil
}

func (qc *qualityChecker) ScoreReadability(ctx context.Context, text, language string) (*ScoreResult, error) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return &ScoreResult{Score: 0, MaxScore: 100, Passed: false, Details: "empty text"}, nil
	}

	sentences := strings.Count(text, ".") + strings.Count(text, "!") + strings.Count(text, "?")
	if sentences == 0 {
		sentences = 1
	}

	avgWordsPerSentence := float64(len(words)) / float64(sentences)
	score := 100.0 - (avgWordsPerSentence-10)*2.0
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	passed := score >= 50.0
	return &ScoreResult{
		Score:    score,
		MaxScore: 100,
		Passed:   passed,
		Details:  "Readability check completed (mock)",
	}, nil
}

func (qc *qualityChecker) ScoreEntityCoverage(ctx context.Context, text string, entities []string) (*ScoreResult, error) {
	if len(entities) == 0 {
		return &ScoreResult{Score: 100, MaxScore: 100, Passed: true, Details: "no entities to check"}, nil
	}

	textLower := strings.ToLower(text)
	found := 0
	for _, entity := range entities {
		if strings.Contains(textLower, strings.ToLower(entity)) {
			found++
		}
	}

	score := float64(found) / float64(len(entities)) * 100
	passed := score >= 60.0
	return &ScoreResult{
		Score:    score,
		MaxScore: 100,
		Passed:   passed,
		Details:  "Entity coverage check completed (mock)",
	}, nil
}

func (qc *qualityChecker) CheckStructure(ctx context.Context, text string, spec StructureSpec) (*ScoreResult, error) {
	words := strings.Fields(text)
	wordCount := len(words)

	var issues []string
	if wordCount < spec.MinWords && spec.MinWords > 0 {
		issues = append(issues, "below minimum word count")
	}
	if spec.MaxWords > 0 && wordCount > spec.MaxWords {
		issues = append(issues, "above maximum word count")
	}

	paragraphs := strings.Count(text, "\n\n") + 1
	if paragraphs < spec.MinParagraphs && spec.MinParagraphs > 0 {
		issues = append(issues, "too few paragraphs")
	}

	if spec.HasIntro && !strings.HasPrefix(strings.TrimSpace(text), "#") {
		textLower := strings.ToLower(text)
		if !strings.Contains(textLower, "introduction") && !strings.Contains(textLower, "overview") {
			issues = append(issues, "missing introduction")
		}
	}

	for _, section := range spec.RequiredSections {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(section)) {
			issues = append(issues, "missing section: "+section)
		}
	}

	issueCount := len(issues)
	score := 100.0 - float64(issueCount)*15.0
	if score < 0 {
		score = 0
	}

	passed := score >= 60.0
	details := "Structure check completed"
	if len(issues) > 0 {
		details = "Issues: " + strings.Join(issues, ", ")
	}

	return &ScoreResult{
		Score:    score,
		MaxScore: 100,
		Passed:   passed,
		Details:  details,
	}, nil
}

func (qc *qualityChecker) CheckDuplicates(ctx context.Context, text string) ([]DuplicateResult, error) {
	words := strings.Fields(text)
	if len(words) < 10 {
		return []DuplicateResult{}, nil
	}

	wordFreq := make(map[string]int)
	for _, w := range words {
		w = strings.Trim(strings.ToLower(w), ".,!?;:\"'()[]")
		if len(w) > 3 {
			wordFreq[w]++
		}
	}

	var results []DuplicateResult
	for w, count := range wordFreq {
		if count > 5 && len(w) > 3 {
			similarity := float64(count) / float64(len(words)) * 100
			results = append(results, DuplicateResult{
				Text:       w,
				Similarity: similarity,
				Passed:     similarity < 5.0,
			})
		}
	}
	if results == nil {
		results = []DuplicateResult{}
	}
	return results, nil
}

func (qc *qualityChecker) CheckHallucination(ctx context.Context, text string, references []string) (*HallucinationResult, error) {
	if len(references) == 0 {
		return &HallucinationResult{
			Passed:     true,
			Confidence: 100,
			Issues:     []string{},
		}, nil
	}

	var issues []string
	textLower := strings.ToLower(text)

	for _, ref := range references {
		refLower := strings.ToLower(ref)
		keyFacts := extractKeyFacts(refLower)
		for _, fact := range keyFacts {
			if !strings.Contains(textLower, fact) {
				issues = append(issues, "unsupported claim: "+fact)
				if len(issues) >= 5 {
					break
				}
			}
		}
		if len(issues) >= 5 {
			break
		}
	}

	confidence := 100.0 - float64(len(issues))*20.0
	if confidence < 0 {
		confidence = 0
	}

	passed := len(issues) == 0
	return &HallucinationResult{
		Passed:     passed,
		Issues:     issues,
		Confidence: confidence,
	}, nil
}

func extractKeyFacts(text string) []string {
	sentences := strings.Split(text, ".")
	var facts []string
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(strings.Fields(s)) > 5 {
			words := strings.Fields(s)
			if len(words) > 10 {
				facts = append(facts, strings.Join(words[:10], " "))
			} else {
				facts = append(facts, s)
			}
		}
		if len(facts) >= 5 {
			break
		}
	}
	return facts
}
