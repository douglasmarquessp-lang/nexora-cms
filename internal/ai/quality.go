package ai

import (
	"context"
	"math"
	"regexp"
	"strings"
)

// ---------- static patterns ----------

var (
	repeatedWordsRE = regexp.MustCompile(`(?i)\b(\w{3,})\s+\1\b`)
	multipleSpacesRE = regexp.MustCompile(`[ ]{2,}`)
	leadingCapRE     = regexp.MustCompile(`^[a-z]`)
	ellipsisRE       = regexp.MustCompile(`\.{4,}`)
	questionMarkRE   = regexp.MustCompile(`\?{2,}`)
	exclamationRE    = regexp.MustCompile(`!{2,}`)
	headingRE        = regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
	h1RE             = regexp.MustCompile(`(?m)^#\s+(.+)$`)
	h2RE             = regexp.MustCompile(`(?m)^##\s+(.+)$`)
	h3RE             = regexp.MustCompile(`(?m)^###\s+(.+)$`)
	linkRE           = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	imageRE          = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	listRE           = regexp.MustCompile(`(?m)^[\s]*[-*+]\s+`)
	orderedListRE    = regexp.MustCompile(`(?m)^[\s]*\d+\.\s+`)
	paragraphSplitRE = regexp.MustCompile(`\n\s*\n`)
	sentenceSplitRE  = regexp.MustCompile(`[.!?](\s|$)`)
	wordRE           = regexp.MustCompile(`\w+('\w+)?`)
	metaDescRE       = regexp.MustCompile(`(?i)<meta\s+[^>]*name\s*=\s*["']description["'][^>]*content\s*=\s*["']([^"']*)["']`)
)

// ---------- helpers ----------

func textWords(text string) []string {
	return wordRE.FindAllString(text, -1)
}

func textSentences(text string) []string {
	parts := sentenceSplitRE.Split(text, -1)
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// countSyllables estimates syllable count for a word using English rules.
func countSyllables(word string) int {
	word = strings.ToLower(word)
	word = strings.Trim(word, ".,!?;:\"'()[]-")
	if len(word) <= 2 {
		return 1
	}
	vowels := "aeiouy"
	syllables := 0
	prevVowel := false
	for i, ch := range word {
		isVowel := strings.ContainsRune(vowels, ch)
		if isVowel && !prevVowel {
			syllables++
		}
		prevVowel = isVowel
		// silent e at end
		if i == len(word)-1 && ch == 'e' && syllables > 1 {
			syllables--
		}
	}
	if syllables == 0 {
		syllables = 1
	}
	return syllables
}

// countSyllablesPT estimates syllable count for Portuguese.
func countSyllablesPT(word string) int {
	word = strings.ToLower(word)
	word = strings.Trim(word, ".,!?;:\"'()[]-")
	if len(word) <= 2 {
		return 1
	}
	vowels := "aeiouyÃÁÀÂÄÉÈÊËÍÌÎÏÓÒÔÖÕÚÙÛÜ"
	syllables := 0
	prevVowel := false
	for _, ch := range word {
		isVowel := strings.ContainsRune(vowels, ch)
		if isVowel && !prevVowel {
			syllables++
		}
		prevVowel = isVowel
	}
	if syllables == 0 {
		syllables = 1
	}
	return syllables
}

func isDifficultWord(word string) bool {
	return countSyllables(word) >= 3
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ---------- qualityChecker ----------

type qualityChecker struct {
}

func NewQualityChecker() *qualityChecker {
	return &qualityChecker{}
}

// ---------- ScoreGrammar (legacy) ----------

func (qc *qualityChecker) ScoreGrammar(ctx context.Context, text, language string) (*ScoreResult, error) {
	report, err := qc.CheckGrammarDetails(ctx, text, language)
	if err != nil {
		return nil, err
	}
	return &ScoreResult{
		Score:    report.OverallScore,
		MaxScore: report.MaxScore,
		Passed:   report.Passed,
		Details:  formatGrammarSummary(report),
	}, nil
}

// ---------- ScoreSEO (legacy) ----------

func (qc *qualityChecker) ScoreSEO(ctx context.Context, text string, keywords []string) (*ScoreResult, error) {
	analysis, err := qc.AssessSEO(ctx, text, keywords)
	if err != nil {
		return nil, err
	}
	return &ScoreResult{
		Score:    analysis.OverallScore,
		MaxScore: analysis.MaxScore,
		Passed:   analysis.Passed,
		Details:  formatSEOSummary(analysis),
	}, nil
}

// ---------- ScoreReadability (legacy) ----------

func (qc *qualityChecker) ScoreReadability(ctx context.Context, text, language string) (*ScoreResult, error) {
	report, err := qc.ScoreReadabilityDetailed(ctx, text, language)
	if err != nil {
		return nil, err
	}
	return &ScoreResult{
		Score:    report.OverallScore,
		MaxScore: report.MaxScore,
		Passed:   report.Passed,
		Details:  formatReadabilitySummary(report),
	}, nil
}

// ---------- ScoreEntityCoverage (deterministic - unchanged logic) ----------

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
		Details:  formatEntityCoverageDetails(found, len(entities)),
	}, nil
}

// ---------- CheckStructure (legacy) ----------

func (qc *qualityChecker) CheckStructure(ctx context.Context, text string, spec StructureSpec) (*ScoreResult, error) {
	words := textWords(text)
	wordCount := len(words)

	var issues []string
	if wordCount < spec.MinWords && spec.MinWords > 0 {
		issues = append(issues, "below minimum word count")
	}
	if spec.MaxWords > 0 && wordCount > spec.MaxWords {
		issues = append(issues, "above maximum word count")
	}

	paragraphs := len(paragraphSplitRE.Split(text, -1))
	if paragraphs < spec.MinParagraphs && spec.MinParagraphs > 0 {
		issues = append(issues, "too few paragraphs")
	}

	if spec.HasIntro && !strings.HasPrefix(strings.TrimSpace(text), "#") {
		textLower := strings.ToLower(text)
		if !strings.Contains(textLower, "introduction") && !strings.Contains(textLower, "overview") {
			issues = append(issues, "missing introduction")
		}
	}

	if spec.HasConclusion {
		textLower := strings.ToLower(text)
		if !strings.Contains(textLower, "conclusion") && !strings.Contains(textLower, "summary") && !strings.Contains(textLower, "final thoughts") {
			issues = append(issues, "missing conclusion")
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

// ---------- CheckDuplicates (legacy) ----------

func (qc *qualityChecker) CheckDuplicates(ctx context.Context, text string) ([]DuplicateResult, error) {
	blocks, err := qc.CheckDuplicateBlocks(ctx, text, 10)
	if err != nil {
		return nil, err
	}
	results := make([]DuplicateResult, 0, len(blocks))
	for _, b := range blocks {
		if len(textWords(b.Text)) >= 3 {
			results = append(results, DuplicateResult{
				Text:       b.Text,
				Similarity: b.Similarity,
				Passed:     b.Passed,
			})
		}
	}
	return results, nil
}

// ---------- CheckHallucination (legacy) ----------

func (qc *qualityChecker) CheckHallucination(ctx context.Context, text string, references []string) (*HallucinationResult, error) {
	report, err := qc.CheckHallucinationWithGrounding(ctx, text, references, nil)
	if err != nil {
		return nil, err
	}
	issues := make([]string, 0, len(report.Items))
	for _, item := range report.Items {
		if item.Verdict == "unsupported" || item.Verdict == "contradicted" {
			issues = append(issues, item.Claim)
		}
	}
	return &HallucinationResult{
		Passed:     report.Passed,
		Issues:     issues,
		Confidence: report.OverallScore,
	}, nil
}

// ---------- CheckGrammarDetails ----------

func (qc *qualityChecker) CheckGrammarDetails(ctx context.Context, text, language string) (*GrammarReport, error) {
	items := make([]QualityCheckItem, 0)
	issues := make([]GrammarIssue, 0)
	totalScore := 100.0

	// 1. Capitalization check (first letter of text)
	trimmed := strings.TrimSpace(text)
	if trimmed != "" && leadingCapRE.MatchString(trimmed) {
		issues = append(issues, GrammarIssue{
			Type:       "capitalization",
			Message:    "Text does not start with a capital letter",
			Suggestion: "Capitalize the first letter of the text",
			Severity:   "error",
		})
		totalScore -= 10
	}

	// 2. Repeated words
	repeats := repeatedWordsRE.FindAllString(text, -1)
	if len(repeats) > 0 {
		issues = append(issues, GrammarIssue{
			Type:       "repeated_word",
			Message:    "Found repeated words: " + strings.Join(repeats, ", "),
			Suggestion: "Remove or rephrase repeated words",
			Severity:   "warning",
		})
		totalScore -= float64(len(repeats)) * 5
	}

	// 3. Multiple spaces
	multiSpaces := multipleSpacesRE.FindAllString(text, -1)
	if len(multiSpaces) > 0 {
		issues = append(issues, GrammarIssue{
			Type:       "punctuation",
			Message:    "Multiple consecutive spaces found",
			Suggestion: "Replace multiple spaces with single spaces",
			Severity:   "warning",
		})
		totalScore -= float64(len(multiSpaces)) * 2
	}

	// 4. Ellipsis overuse
	if len(ellipsisRE.FindAllString(text, -1)) > 2 {
		issues = append(issues, GrammarIssue{
			Type:       "punctuation",
			Message:    "Ellipsis overused",
			Suggestion: "Use ellipsis sparingly",
			Severity:   "info",
		})
		totalScore -= 5
	}

	// 5. Repeated punctuation
	repeatedPunc := len(questionMarkRE.FindAllString(text, -1)) + len(exclamationRE.FindAllString(text, -1))
	if repeatedPunc > 0 {
		issues = append(issues, GrammarIssue{
			Type:       "punctuation",
			Message:    "Repeated question/exclamation marks found",
			Suggestion: "Use single punctuation marks",
			Severity:   "warning",
		})
		totalScore -= float64(repeatedPunc) * 5
	}

	// 6. Space after punctuation check (heuristic: period followed by non-space non-end)
	punctNoSpaceRE := regexp.MustCompile(`[.!?][A-Za-z]`)
	punctNoSpace := punctNoSpaceRE.FindAllString(text, -1)
	if len(punctNoSpace) > 0 {
		issues = append(issues, GrammarIssue{
			Type:       "punctuation",
			Message:    "Missing space after punctuation",
			Suggestion: "Add a space after periods, question marks, and exclamation marks",
			Severity:   "error",
		})
		totalScore -= float64(len(punctNoSpace)) * 5
	}

	// 7. Capitalization after period
	capAfterPeriodRE := regexp.MustCompile(`\.\s+[a-z]`)
	capAfterPeriod := capAfterPeriodRE.FindAllString(text, -1)
	if len(capAfterPeriod) > 0 {
		issues = append(issues, GrammarIssue{
			Type:       "capitalization",
			Message:    "Sentence does not start with a capital letter after period",
			Suggestion: "Capitalize the first word after a period",
			Severity:   "error",
		})
		totalScore -= float64(len(capAfterPeriod)) * 5
	}

	score := clamp(totalScore, 0, 100)

	items = append(items, QualityCheckItem{
		Category:  "grammar",
		CheckName: "grammar_overview",
		Severity:  "info",
		Score:     score,
		MaxScore:  100,
		Passed:    score >= 70,
		Message:   formatGrammarIssuesCount(issues),
		Source:    SourceDeterministic,
	})

	return &GrammarReport{
		OverallScore: score,
		MaxScore:     100,
		Passed:       score >= 70,
		Items:        items,
		Issues:       issues,
		AIAssisted:   false,
	}, nil
}

// ---------- AssessSEO ----------

func (qc *qualityChecker) AssessSEO(ctx context.Context, text string, keywords []string) (*SEOAnalysis, error) {
	items := make([]QualityCheckItem, 0)

	titleScore := qc.assessSEOTitle(text)
	items = append(items, QualityCheckItem{
		Category:  "seo",
		CheckName: "title",
		Severity:  severityFromScore(titleScore.Score),
		Score:     titleScore.Score,
		MaxScore:  titleScore.MaxScore,
		Passed:    titleScore.Passed,
		Message:   titleScore.Message,
		Source:    SourceDeterministic,
	})

	headingsScore := qc.assessSEOHeadings(text, keywords)
	items = append(items, QualityCheckItem{
		Category:  "seo",
		CheckName: "headings",
		Severity:  severityFromScore(headingsScore.Score),
		Score:     headingsScore.Score,
		MaxScore:  headingsScore.MaxScore,
		Passed:    headingsScore.Passed,
		Message:   headingsScore.Message,
		Source:    SourceDeterministic,
	})

	kwUsage := qc.assessSEOKeywordUsage(text, keywords)
	items = append(items, QualityCheckItem{
		Category:  "seo",
		CheckName: "keyword_usage",
		Severity:  severityFromScore(kwUsage.Score),
		Score:     kwUsage.Score,
		MaxScore:  kwUsage.MaxScore,
		Passed:    kwUsage.Passed,
		Message:   kwUsage.Message,
		Source:    SourceDeterministic,
	})

	metaScore := qc.assessSEOMetaDesc(text, keywords)
	items = append(items, QualityCheckItem{
		Category:  "seo",
		CheckName: "meta_description",
		Severity:  severityFromScore(metaScore.Score),
		Score:     metaScore.Score,
		MaxScore:  metaScore.MaxScore,
		Passed:    metaScore.Passed,
		Message:   metaScore.Message,
		Source:    SourceDeterministic,
	})

	contentScore := qc.assessSEOContentStructure(text)
	items = append(items, QualityCheckItem{
		Category:  "seo",
		CheckName: "content_structure",
		Severity:  severityFromScore(contentScore.Score),
		Score:     contentScore.Score,
		MaxScore:  contentScore.MaxScore,
		Passed:    contentScore.Passed,
		Message:   contentScore.Message,
		Source:    SourceDeterministic,
	})

	intentScore := qc.assessSEOIntent(text, keywords)
	items = append(items, QualityCheckItem{
		Category:  "seo",
		CheckName: "search_intent",
		Severity:  severityFromScore(intentScore.Score),
		Score:     intentScore.Score,
		MaxScore:  intentScore.MaxScore,
		Passed:    intentScore.Passed,
		Message:   intentScore.Message,
		Source:    SourceDeterministic,
	})

	overall := (titleScore.Score + headingsScore.Score + kwUsage.Score + metaScore.Score + contentScore.Score + intentScore.Score) / 6.0

	return &SEOAnalysis{
		OverallScore:  clamp(overall, 0, 100),
		MaxScore:      100,
		Passed:        overall >= 60,
		Items:         items,
		TitleScore:    titleScore,
		HeadingsScore: headingsScore,
		KeywordUsage:  kwUsage,
		MetaDescScore: metaScore,
		ContentScore:  contentScore,
		IntentScore:   intentScore,
		AIAssisted:    false,
	}, nil
}

func (qc *qualityChecker) assessSEOTitle(text string) *SEOTitleScore {
	headings := h1RE.FindAllStringSubmatch(text, -1)
	if len(headings) == 0 {
		lines := strings.SplitN(strings.TrimSpace(text), "\n", 2)
		title := strings.TrimSpace(lines[0])
		titleLen := len([]rune(title))
		score := 50.0
		if titleLen >= 30 && titleLen <= 60 {
			score = 100
		} else if titleLen >= 20 && titleLen <= 70 {
			score = 75
		} else if titleLen >= 10 {
			score = 50
		}
		warnings := buildTitleWarnings(titleLen)
		return &SEOTitleScore{
			Title:     title,
			Length:    titleLen,
			Score:     score,
			MaxScore:  100,
			Passed:    score >= 60,
			Message:   formatTitleMsg(title, titleLen),
			Warnings:  warnings,
		}
	}

	h1Text := headings[0][1]
	titleLen := len([]rune(h1Text))
	score := 50.0
	if titleLen >= 30 && titleLen <= 60 {
		score = 100
	} else if titleLen >= 20 && titleLen <= 70 {
		score = 75
	} else if titleLen >= 10 {
		score = 50
	} else {
		score = 25
	}
	if titleLen < 20 {
		score -= 10
	}
	if titleLen > 70 {
		score -= 10
	}
	score = clamp(score, 0, 100)

	return &SEOTitleScore{
		Title:    h1Text,
		Length:   titleLen,
		Score:    score,
		MaxScore: 100,
		Passed:   score >= 60,
		Message:  formatTitleMsg(h1Text, titleLen),
	}
}

func (qc *qualityChecker) assessSEOHeadings(text string, keywords []string) *SEOHeadingsScore {
	h1s := h1RE.FindAllString(text, -1)
	h2s := h2RE.FindAllString(text, -1)
	h3s := h3RE.FindAllString(text, -1)

	warnings := make([]string, 0)
	score := 100.0

	if len(h1s) == 0 {
		warnings = append(warnings, "No H1 heading found")
		score -= 20
	} else if len(h1s) > 1 {
		warnings = append(warnings, "Multiple H1 headings (should have exactly one)")
		score -= 15
	}

	if len(h2s) == 0 && len(h3s) == 0 {
		warnings = append(warnings, "No subheadings (H2/H3) found")
		score -= 15
	}

	kwInH1 := false
	kwInH2 := 0
	for _, kw := range keywords {
		for _, h := range h1s {
			if strings.Contains(strings.ToLower(h), strings.ToLower(kw)) {
				kwInH1 = true
			}
		}
		for _, h := range h2s {
			if strings.Contains(strings.ToLower(h), strings.ToLower(kw)) {
				kwInH2++
			}
		}
	}
	if !kwInH1 && len(keywords) > 0 {
		warnings = append(warnings, "Primary keyword not found in H1")
		score -= 15
	}
	if kwInH2 == 0 && len(keywords) > 0 && len(h2s) > 0 {
		warnings = append(warnings, "Keyword not found in any H2")
		score -= 10
	}

	score = clamp(score, 0, 100)

	hasHierarchy := true
	if len(h1s) == 1 && len(h3s) > 0 && len(h2s) == 0 {
		hasHierarchy = false
		warnings = append(warnings, "Heading hierarchy skipped: H1→H3 without H2")
		score -= 5
	}

	return &SEOHeadingsScore{
		H1Count:      len(h1s),
		H2Count:      len(h2s),
		H3Count:      len(h3s),
		KeywordInH1:  kwInH1,
		KeywordInH2:  kwInH2,
		HasHierarchy: hasHierarchy,
		Score:        score,
		MaxScore:     100,
		Passed:       score >= 60,
		Message:      formatHeadingsMsg(len(h1s), len(h2s), len(h3s)),
		Warnings:     warnings,
	}
}

func (qc *qualityChecker) assessSEOKeywordUsage(text string, keywords []string) *SEOKeywordUsage {
	if len(keywords) == 0 {
		return &SEOKeywordUsage{
			Score:    50,
			MaxScore: 100,
			Passed:   false,
			Message:  "No keywords to analyze",
			Warnings: []string{"No keywords provided"},
		}
	}

	words := textWords(text)
	totalWords := len(words)
	if totalWords == 0 {
		return &SEOKeywordUsage{
			Score:    0,
			MaxScore: 100,
			Passed:   false,
			Message:  "Empty text",
			Warnings: []string{"Empty text"},
		}
	}

	textLower := strings.ToLower(text)
	kwCount := 0
	for _, kw := range keywords {
		kwCount += strings.Count(textLower, strings.ToLower(kw))
	}

	density := float64(kwCount) / float64(totalWords) * 100

	firstWords := textWords(text)
	var first100Lower string
	if len(firstWords) > 100 {
		first100Lower = strings.ToLower(strings.Join(firstWords[:100], " "))
	} else {
		first100Lower = textLower
	}
	kwInFirst100 := false
	for _, kw := range keywords {
		if strings.Contains(first100Lower, strings.ToLower(kw)) {
			kwInFirst100 = true
			break
		}
	}

	warnings := make([]string, 0)
	densityScore := 100.0
	if density < 0.5 {
		warnings = append(warnings, "Keyword density too low")
		densityScore = 30
	} else if density < 1.0 {
		densityScore = 60
	} else if density <= 3.0 {
		densityScore = 100
	} else if density <= 5.0 {
		warnings = append(warnings, "Keyword density high (potential keyword stuffing)")
		densityScore = 50
	} else {
		warnings = append(warnings, "Keyword density very high (keyword stuffing detected)")
		densityScore = 10
	}

	placementScore := 100.0
	if !kwInFirst100 {
		warnings = append(warnings, "Keyword not found in first 100 words")
		placementScore = 50
	}

	score := (densityScore + placementScore) / 2
	score = clamp(score, 0, 100)

	return &SEOKeywordUsage{
		Keywords:       keywords,
		Density:        density,
		First100Kw:     kwInFirst100,
		DensityScore:   densityScore,
		PlacementScore: placementScore,
		Score:          score,
		MaxScore:       100,
		Passed:         score >= 60,
		Message:        formatKeywordMsg(density, kwCount, kwInFirst100),
		Warnings:       warnings,
	}
}

func (qc *qualityChecker) assessSEOMetaDesc(text string, keywords []string) *SEOMetaDescScore {
	matches := metaDescRE.FindStringSubmatch(text)
	warnings := make([]string, 0)
	score := 50.0

	if len(matches) < 2 || matches[1] == "" {
		warnings = append(warnings, "No meta description found")
		return &SEOMetaDescScore{
			HasMetaDesc: false,
			Score:       20,
			MaxScore:    100,
			Passed:      false,
			Message:     "No meta description found",
			Warnings:    warnings,
		}
	}

	desc := matches[1]
	descLen := len(desc)
	kwFound := false
	for _, kw := range keywords {
		if strings.Contains(strings.ToLower(desc), strings.ToLower(kw)) {
			kwFound = true
			break
		}
	}

	if descLen >= 150 && descLen <= 160 {
		score = 100
	} else if descLen >= 120 && descLen <= 200 {
		score = 75
	} else if descLen >= 50 {
		score = 50
	} else {
		score = 20
		warnings = append(warnings, "Meta description too short")
	}

	if !kwFound && len(keywords) > 0 {
		warnings = append(warnings, "Keyword not found in meta description")
		score -= 15
	}

	score = clamp(score, 0, 100)

	return &SEOMetaDescScore{
		HasMetaDesc:     true,
		Length:          descLen,
		KeywordPresence: kwFound,
		Score:           score,
		MaxScore:        100,
		Passed:          score >= 60,
		Message:         formatMetaMsg(descLen, kwFound),
		Warnings:        warnings,
	}
}

func (qc *qualityChecker) assessSEOContentStructure(text string) *SEOContentScore {
	paragraphs := paragraphSplitRE.Split(text, -1)
	paraCount := len(paragraphs)
	links := linkRE.FindAllString(text, -1)
	images := imageRE.FindAllString(text, -1)
	lists := listRE.FindAllString(text, -1)
	orderedLists := orderedListRE.FindAllString(text, -1)

	var warnings []string
	score := 100.0

	if paraCount < 3 {
		warnings = append(warnings, "Too few paragraphs")
		score -= 20
	}
	if len(links) == 0 {
		warnings = append(warnings, "No links found")
		score -= 10
	}
	if len(images) == 0 {
		warnings = append(warnings, "No images found")
		score -= 10
	}
	if len(lists)+len(orderedLists) == 0 {
		score -= 5
	}

	longParas := 0
	for _, p := range paragraphs {
		if len(textWords(p)) > 150 {
			longParas++
		}
	}
	if longParas > 0 {
		warnings = append(warnings, "Very long paragraphs found (over 150 words)")
		score -= float64(longParas) * 5
	}

	score = clamp(score, 0, 100)
	return &SEOContentScore{
		ParagraphCount: paraCount,
		HasLists:       len(lists)+len(orderedLists) > 0,
		ListCount:      len(lists) + len(orderedLists),
		HasLinks:       len(links) > 0,
		ExternalLinks:  len(links),
		HasImages:      len(images) > 0,
		Score:          score,
		MaxScore:       100,
		Passed:         score >= 60,
		Message:        formatContentMsg(paraCount, len(links), len(images)),
		Warnings:       warnings,
	}
}

func (qc *qualityChecker) assessSEOIntent(text string, keywords []string) *SEOIntentScore {
	textLower := strings.ToLower(text)

	informational := countMatches(textLower, []string{"what is", "how to", "guide", "tutorial", "overview", "explain", "introduction", "learn", "basics", "understand"})
	commercial := countMatches(textLower, []string{"best", "top", "review", "vs", "compare", "alternative", "price", "cost", "buying guide", "worth"})
	navigational := countMatches(textLower, []string{"login", "sign up", "download", "official", "website", "homepage"})
	transactional := countMatches(textLower, []string{"buy", "purchase", "order", "subscribe", "discount", "coupon", "deal", "offer", "shop", "price"})

	detected := "informational"
	maxCount := informational
	if commercial > maxCount {
		maxCount = commercial
		detected = "commercial"
	}
	if navigational > maxCount {
		maxCount = navigational
		detected = "navigational"
	}
	if transactional > maxCount {
		maxCount = transactional
		detected = "transactional"
	}

	score := 70.0
	if maxCount > 0 {
		score = 80.0 + float64(maxCount)*2.0
	}
	if len(keywords) > 0 {
		kwLower := strings.ToLower(strings.Join(keywords, " "))
		buyWords := countMatches(kwLower, []string{"buy", "purchase", "price", "discount", "deal"})
		infoWords := countMatches(kwLower, []string{"what", "how", "guide", "tutorial"})
		compareWords := countMatches(kwLower, []string{"vs", "review", "best", "compare", "top"})
		if buyWords > 0 && detected != "transactional" {
			score -= 10
		}
		if compareWords > 0 && detected != "commercial" {
			score -= 10
		}
		if infoWords > 0 && detected != "informational" {
			score -= 5
		}
	}

	score = clamp(score, 0, 100)
	return &SEOIntentScore{
		DetectedIntent: detected,
		Score:          score,
		MaxScore:       100,
		Passed:         score >= 60,
		Message:        formatIntentMsg(detected),
		AIAssisted:     false,
	}
}

// ---------- ScoreReadabilityDetailed ----------

func (qc *qualityChecker) ScoreReadabilityDetailed(ctx context.Context, text, language string) (*ReadabilityReport, error) {
	words := textWords(text)
	wordCount := len(words)
	if wordCount == 0 {
		return &ReadabilityReport{
			OverallScore: 0, MaxScore: 100, Passed: false,
			FleschReadingEase: 0, FleschKincaidGrade: 0,
			WordCount: 0, SentenceCount: 0, SyllableCount: 0,
			Items: []QualityCheckItem{
				{Category: "readability", CheckName: "score", Severity: "error", Score: 0, MaxScore: 100, Passed: false, Message: "empty text", Source: SourceDeterministic},
			},
		}, nil
	}

	sentences := textSentences(text)
	sentenceCount := len(sentences)
	if sentenceCount == 0 {
		sentenceCount = 1
	}

	totalSyllables := 0
	difficultWords := 0
	var syllableFn func(string) int
	if language == "pt" {
		syllableFn = countSyllablesPT
	} else {
		syllableFn = countSyllables
	}

	for _, w := range words {
		s := syllableFn(w)
		totalSyllables += s
		if s >= 3 {
			difficultWords++
		}
	}

	avgWordsPerSent := float64(wordCount) / float64(sentenceCount)
	avgSylPerWord := float64(totalSyllables) / float64(wordCount)
	difficultPct := float64(difficultWords) / float64(wordCount) * 100

	// Flesch Reading Ease
	var fre float64
	if language == "pt" {
		// Portuguese-adjusted formula
		fre = 206.835 - 1.015*avgWordsPerSent - 72.0*avgSylPerWord
	} else {
		fre = 206.835 - 1.015*avgWordsPerSent - 84.6*avgSylPerWord
	}
	fre = clamp(fre, 0, 100)

	// Flesch-Kincaid Grade Level
	var fk float64
	if language == "pt" {
		fk = 0.39*avgWordsPerSent + 10.0*avgSylPerWord - 10.0
	} else {
		fk = 0.39*avgWordsPerSent + 11.8*avgSylPerWord - 15.59
	}
	if fk < 0 {
		fk = 0
	}

	// Map FRE to 0-100 score
	readabilityScore := fre

	items := []QualityCheckItem{
		{
			Category: "readability", CheckName: "flesch_reading_ease",
			Severity: severityFromScore(readabilityScore),
			Score:    readabilityScore, MaxScore: 100,
			Passed:  readabilityScore >= 50,
			Message: formatFREScore(fre, language),
			Source:  SourceDeterministic,
		},
		{
			Category: "readability", CheckName: "grade_level",
			Severity: severityFromScore(clamp(100-fk*10, 0, 100)),
			Score:    clamp(100-fk*10, 0, 100), MaxScore: 100,
			Passed:  fk <= 12,
			Message: fmtGradeLevel(fk),
			Source:  SourceDeterministic,
		},
		{
			Category: "readability", CheckName: "sentence_length",
			Severity: severityFromScore(clamp(100-avgWordsPerSent*2, 0, 100)),
			Score:    clamp(100-avgWordsPerSent*2, 0, 100), MaxScore: 100,
			Passed:  avgWordsPerSent <= 25,
			Message: fmtAvgSentenceLen(avgWordsPerSent),
			Source:  SourceDeterministic,
		},
		{
			Category: "readability", CheckName: "difficult_words",
			Severity: severityFromScore(clamp(100-difficultPct, 0, 100)),
			Score:    clamp(100-difficultPct, 0, 100), MaxScore: 100,
			Passed:  difficultPct <= 30,
			Message: fmtDifficultWords(difficultPct),
			Source:  SourceDeterministic,
		},
	}

	overall := (readabilityScore + clamp(100-fk*10, 0, 100) + clamp(100-avgWordsPerSent*2, 0, 100) + clamp(100-difficultPct, 0, 100)) / 4.0

	return &ReadabilityReport{
		OverallScore:         clamp(overall, 0, 100),
		MaxScore:             100,
		Passed:               overall >= 50,
		FleschReadingEase:    fre,
		FleschKincaidGrade:   fk,
		WordCount:            wordCount,
		SentenceCount:        sentenceCount,
		SyllableCount:        totalSyllables,
		AvgWordsPerSentence:  avgWordsPerSent,
		AvgSyllablesPerWord:  avgSylPerWord,
		DifficultWordCount:   difficultWords,
		DifficultWordPercent: difficultPct,
		Items:                items,
	}, nil
}

// ---------- CheckDuplicateBlocks ----------

func (qc *qualityChecker) CheckDuplicateBlocks(ctx context.Context, text string, minLength int) ([]DuplicateBlock, error) {
	words := textWords(text)
	if len(words) < 10 {
		return []DuplicateBlock{}, nil
	}

	// Use shingle-based detection (3-word shingles)
	shingles := make(map[string][]int)
	for i := 0; i < len(words)-2; i++ {
		shingle := strings.ToLower(words[i] + " " + words[i+1] + " " + words[i+2])
		shingles[shingle] = append(shingles[shingle], i)
	}

	var blocks []DuplicateBlock
	seen := make(map[string]bool)

	for shingle, positions := range shingles {
		if len(positions) < 2 || seen[shingle] {
			continue
		}

		// Find contiguous duplicate runs
		used := make(map[int]bool)
		for _, pos := range positions {
			if used[pos] {
				continue
			}
			// Expand forward
			end := pos + 3
			for end <= len(words)-3 {
				nextShingle := strings.ToLower(words[end] + " " + words[end+1] + " " + words[end+2])
				hasDup := false
				for _, p2 := range shingles[nextShingle] {
					if p2 != pos && !used[p2] && p2 >= end-3 && p2 <= end+3 {
						hasDup = true
						used[p2] = true
						break
					}
				}
				if !hasDup {
					break
				}
				used[end] = true
				end += 3
				if end >= len(words) {
					break
				}
			}
			blockLen := end - pos
			if blockLen >= minLength {
				blockText := strings.Join(words[pos:end], " ")
				similarity := float64(len(positions)) / float64(len(words)) * 100
				blocks = append(blocks, DuplicateBlock{
					Text:       blockText,
					Similarity: clamp(similarity, 0, 100),
					Offset:     pos,
					Length:     blockLen,
					Passed:     similarity < 5.0,
				})
			}
		}
		seen[shingle] = true
	}

	if blocks == nil {
		blocks = []DuplicateBlock{}
	}
	return blocks, nil
}

// ---------- ValidateStructure ----------

func (qc *qualityChecker) ValidateStructure(ctx context.Context, text string) (*StructureReport, error) {
	items := make([]QualityCheckItem, 0)
	headingIssues := make([]StructureIssue, 0)
	paragraphIssues := make([]StructureIssue, 0)
	score := 100.0

	// Heading analysis
	allHeadings := headingRE.FindAllStringSubmatch(text, -1)
	h1s := h1RE.FindAllString(text, -1)

	if len(h1s) == 0 {
		headingIssues = append(headingIssues, StructureIssue{
			Type: "missing_h1", Element: "h1",
			Message: "No H1 heading found - content must have exactly one H1",
			Suggestion: "Add an H1 heading at the top of the content",
			Severity: "error",
		})
		score -= 20
	} else if len(h1s) > 1 {
		headingIssues = append(headingIssues, StructureIssue{
			Type: "heading_order", Element: "h1",
			Message: "Multiple H1 headings found - use exactly one H1",
			Suggestion: "Change additional H1s to H2",
			Severity: "error",
		})
		score -= 15
	}

	// Heading hierarchy check: no skipping from H1 to H3 without H2
	for i, h := range allHeadings {
		level := len(h[0]) - len(strings.TrimLeft(h[0], "#"))
		if i > 0 {
			prev := allHeadings[i-1]
			prevLevel := len(prev[0]) - len(strings.TrimLeft(prev[0], "#"))
			if level > prevLevel+1 {
				headingIssues = append(headingIssues, StructureIssue{
					Type: "heading_order", Element: "h" + itoa(level),
					Message: "Skipped heading level: " + h[0] + " (no H" + itoa(level-1) + " before it)",
					Suggestion: "Add an H" + itoa(level-1) + " before this heading",
					Severity: "warning",
				})
				score -= 5
			}
		}
	}

	// Paragraph analysis
	paragraphs := paragraphSplitRE.Split(text, -1)
	nonEmptyParas := 0
	for _, p := range paragraphs {
		if len(strings.TrimSpace(p)) > 0 {
			nonEmptyParas++
		}
	}

	if nonEmptyParas < 2 {
		paragraphIssues = append(paragraphIssues, StructureIssue{
			Type: "paragraph_length", Element: "paragraph",
			Message: "Too few paragraphs for structured content",
			Suggestion: "Break content into at least 2-3 paragraphs",
			Severity: "warning",
		})
		score -= 10
	}

	// Very short paragraphs
	for _, p := range paragraphs {
		pWords := textWords(p)
		if len(pWords) > 0 && len(pWords) < 5 {
			paragraphIssues = append(paragraphIssues, StructureIssue{
				Type: "paragraph_length", Element: "paragraph",
				Message: "Very short paragraph: '" + truncate(p, 60) + "'",
				Suggestion: "Combine with adjacent paragraph or expand",
				Severity: "info",
			})
		}
	}

	// Lists
	lists := listRE.FindAllString(text, -1)
	orderedLists := orderedListRE.FindAllString(text, -1)
	listCount := len(lists) + len(orderedLists)

	// Links
	links := linkRE.FindAllString(text, -1)
	linkCount := len(links)

	// Images
	images := imageRE.FindAllString(text, -1)
	imageCount := len(images)

	// Check image alt text
	noAltCount := 0
	for _, img := range images {
		match := imageRE.FindStringSubmatch(img)
		if len(match) >= 2 && match[1] == "" {
			noAltCount++
		}
	}
	if noAltCount > 0 {
		headingIssues = append(headingIssues, StructureIssue{
			Type: "image_alt", Element: "img",
			Message: "Images missing alt text",
			Suggestion: "Add descriptive alt text to all images",
			Severity: "error",
		})
		score -= float64(noAltCount) * 5
	}

	// Conclusion detection
	textLower := strings.ToLower(text)
	hasConclusion := strings.Contains(textLower, "conclusion") ||
		strings.Contains(textLower, "summary") ||
		strings.Contains(textLower, "final thoughts")

	if !hasConclusion {
		score -= 5
	}

	// Completeness
	completeness := completenessPercent(text)

	score = clamp(score, 0, 100)

	items = append(items, QualityCheckItem{
		Category: "structure", CheckName: "overall",
		Severity: severityFromScore(score),
		Score:    score, MaxScore: 100,
		Passed:  score >= 60,
		Message: formatStructureSummary(headingIssues, paragraphIssues),
		Source:  SourceDeterministic,
	})

	return &StructureReport{
		OverallScore:    score,
		MaxScore:        100,
		Passed:          score >= 60,
		Items:           items,
		HeadingIssues:   headingIssues,
		ParagraphIssues: paragraphIssues,
		ListCount:       listCount,
		LinkCount:       linkCount,
		BrokenLinkCount: 0, // cannot verify without network access
		ImageCount:      imageCount,
		HasConclusion:   hasConclusion,
		CompletenessPct: completeness,
	}, nil
}

// ---------- CheckHallucinationWithGrounding ----------

func (qc *qualityChecker) CheckHallucinationWithGrounding(ctx context.Context, text string, references []string, grounding *GroundingMetadata) (*FactCheckReport, error) {
	if len(references) == 0 && (grounding == nil || len(grounding.Sources) == 0) {
		return &FactCheckReport{
			Passed: true, OverallScore: 100, MaxScore: 100,
			ClaimsChecked: 0, Supported: 0, Unsupported: 0, Contradicted: 0, Unverifiable: 0,
			Items: []FactCheckItem{}, Grounded: false,
		}, nil
	}

	items := make([]FactCheckItem, 0)
	supported := 0
	unsupported := 0
	contradicted := 0
	unverifiable := 0

	// Extract claims from text (sentences with substantive content)
	sentences := textSentences(text)

	// Build source text corpus from grounding metadata
	sourceCorpus := ""
	sourceVerified := false
	if grounding != nil {
		for _, src := range grounding.Sources {
			sourceCorpus += src.Title + ". " + src.Snippet + "\n"
			if src.IsVerified {
				sourceVerified = true
			}
		}
	}
	for _, ref := range references {
		sourceCorpus += ref + "\n"
	}

	sourceLower := strings.ToLower(sourceCorpus)

	claimsChecked := 0
	for _, sentence := range sentences {
		sLen := len(textWords(sentence))
		if sLen < 5 || sLen > 60 {
			continue
		}
		claimsChecked++

		sentenceLower := strings.ToLower(sentence)
		keyWords := extractKeyTerms(sentenceLower)

		if sourceCorpus == "" {
			// No sources: mark as unverifiable
			unverifiable++
			items = append(items, FactCheckItem{
				Claim:   truncate(sentence, 120),
				Verdict: "unverifiable",
				Confidence: 0,
				SourceQuality: "none",
			})
			continue
		}

		// Check if key terms from this sentence appear in source corpus
		supportedTerms := 0
		for _, term := range keyWords {
			if strings.Contains(sourceLower, term) {
				supportedTerms++
			}
		}

		matchRatio := 0.0
		if len(keyWords) > 0 {
			matchRatio = float64(supportedTerms) / float64(len(keyWords))
		}

		var verdict string
		var confidence float64
		if matchRatio >= 0.6 {
			verdict = "supported"
			confidence = matchRatio * 100
			supported++
		} else if matchRatio >= 0.3 {
			verdict = "unverifiable"
			confidence = matchRatio * 50
			unverifiable++
		} else {
			verdict = "unsupported"
			confidence = (1 - matchRatio) * 50
			unsupported++
		}

		items = append(items, FactCheckItem{
			Claim:         truncate(sentence, 120),
			Verdict:       verdict,
			Confidence:    clamp(confidence, 0, 100),
			SourceQuality: sourceQualityLabel(sourceVerified, grounding),
			Suggestion:    suggestionForVerdict(verdict),
		})
	}

	if claimsChecked == 0 {
		return &FactCheckReport{
			Passed: true, OverallScore: 100, MaxScore: 100,
			ClaimsChecked: 0, Supported: 0, Unsupported: 0, Contradicted: 0, Unverifiable: 0,
			Items: []FactCheckItem{}, Grounded: grounding != nil && len(grounding.Sources) > 0,
			GroundingMeta: grounding,
		}, nil
	}

	overallScore := float64(supported) / float64(claimsChecked) * 100
	passed := unsupported == 0

	return &FactCheckReport{
		Passed:        passed,
		OverallScore:  clamp(overallScore, 0, 100),
		MaxScore:      100,
		ClaimsChecked: claimsChecked,
		Supported:     supported,
		Unsupported:   unsupported,
		Contradicted:  contradicted,
		Unverifiable:  unverifiable,
		Items:         items,
		Grounded:      grounding != nil && len(grounding.Sources) > 0,
		GroundingMeta: grounding,
	}, nil
}

// ---------- utility functions ----------

func severityFromScore(score float64) string {
	if score >= 80 {
		return "info"
	}
	if score >= 60 {
		return "warning"
	}
	return "error"
}

func countMatches(text string, patterns []string) int {
	count := 0
	for _, p := range patterns {
		count += strings.Count(text, p)
	}
	return count
}

func extractKeyTerms(sentence string) []string {
	words := strings.Fields(sentence)
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "shall": true,
		"should": true, "may": true, "might": true, "can": true, "could": true,
		"i": true, "you": true, "he": true, "she": true, "it": true, "we": true, "they": true,
		"this": true, "that": true, "these": true, "those": true, "to": true, "of": true,
		"in": true, "for": true, "on": true, "with": true, "at": true, "by": true, "from": true,
		"as": true, "into": true, "through": true, "during": true, "before": true, "after": true,
		"above": true, "below": true, "between": true, "out": true, "off": true, "over": true,
		"under": true, "again": true, "further": true, "then": true, "once": true, "here": true,
		"there": true, "and": true, "but": true, "or": true, "if": true, "because": true,
		"so": true, "than": true, "too": true, "very": true, "just": true, "about": true,
		"also": true, "not": true, "no": true, "up": true, "down": true,
	}
	var terms []string
	for _, w := range words {
		w = strings.Trim(w, ".,!?;:\"'()[]-")
		w = strings.ToLower(w)
		if len(w) > 3 && !stopWords[w] {
			terms = append(terms, w)
		}
	}
	if len(terms) > 10 {
		terms = terms[:10]
	}
	return terms
}

func sourceQualityLabel(verified bool, gm *GroundingMetadata) string {
	if gm == nil || len(gm.Sources) == 0 {
		return "none"
	}
	if gm.Unverified {
		return "unverified"
	}
	if verified {
		return "verified"
	}
	return "partial"
}

func suggestionForVerdict(verdict string) string {
	switch verdict {
	case "unsupported":
		return "Add supporting evidence or remove this claim"
	case "contradicted":
		return "Correct this claim to match source data"
	case "unverifiable":
		return "Find sources to verify this claim"
	default:
		return ""
	}
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

func completenessPercent(text string) float64 {
	words := textWords(text)
	if len(words) == 0 {
		return 0
	}

	checks := 0
	passed := 0

	// Has title/H1
	if len(h1RE.FindAllString(text, -1)) > 0 {
		passed++
	}
	checks++

	// Has body content (>= 100 words)
	if len(words) >= 100 {
		passed++
	}
	checks++

	// Has subheadings
	if len(h2RE.FindAllString(text, -1)) > 0 {
		passed++
	}
	checks++

	// Has paragraphs
	nonEmptyParas := 0
	for _, p := range paragraphSplitRE.Split(text, -1) {
		if len(strings.TrimSpace(p)) > 0 {
			nonEmptyParas++
		}
	}
	if nonEmptyParas >= 2 {
		passed++
	}
	checks++

	// Has links
	if len(linkRE.FindAllString(text, -1)) > 0 {
		passed++
	}
	checks++

	if checks == 0 {
		return 0
	}
	return float64(passed) / float64(checks) * 100
}

// ---------- formatting helpers ----------

func formatGrammarIssuesCount(issues []GrammarIssue) string {
	if len(issues) == 0 {
		return "No grammar issues found"
	}
	errs := 0
	warns := 0
	for _, i := range issues {
		if i.Severity == "error" {
			errs++
		} else {
			warns++
		}
	}
	return fmtGrammarCounts(errs, warns)
}

func formatGrammarSummary(r *GrammarReport) string {
	if r.Passed {
		return "Grammar check passed (" + fmtGrammarCounts(countErrs(r.Issues), countWarns(r.Issues)) + ")"
	}
	return "Grammar check failed: " + fmtGrammarCounts(countErrs(r.Issues), countWarns(r.Issues))
}

func countErrs(issues []GrammarIssue) int {
	n := 0
	for _, i := range issues {
		if i.Severity == "error" {
			n++
		}
	}
	return n
}

func countWarns(issues []GrammarIssue) int {
	n := 0
	for _, i := range issues {
		if i.Severity == "warning" || i.Severity == "info" {
			n++
		}
	}
	return n
}

func fmtGrammarCounts(errs, warns int) string {
	parts := ""
	if errs > 0 {
		parts += itoa(errs) + " errors"
	}
	if warns > 0 {
		if parts != "" {
			parts += ", "
		}
		parts += itoa(warns) + " warnings"
	}
	if parts == "" {
		parts = "0 issues"
	}
	return parts
}

func formatSEOSummary(a *SEOAnalysis) string {
	failCount := 0
	for _, i := range a.Items {
		if !i.Passed {
			failCount++
		}
	}
	if failCount == 0 {
		return "All SEO checks passed"
	}
	return "SEO checks: " + itoa(failCount) + " dimensions need improvement"
}

func formatReadabilitySummary(r *ReadabilityReport) string {
	return formatFREScore(r.FleschReadingEase, "")

}

func formatTitleMsg(title string, length int) string {
	return "Title length: " + itoa(length) + " chars"
}

func buildTitleWarnings(length int) []string {
	var w []string
	if length < 20 {
		w = append(w, "Title too short")
	} else if length > 70 {
		w = append(w, "Title too long")
	}
	return w
}

func formatHeadingsMsg(h1, h2, h3 int) string {
	return "H1: " + itoa(h1) + ", H2: " + itoa(h2) + ", H3: " + itoa(h3)
}

func formatKeywordMsg(density float64, count int, inFirst100 bool) string {
	msg := "Density: " + fmtFloat(density, 1) + "%, count: " + itoa(count)
	if inFirst100 {
		msg += ", keyword in first 100 words: yes"
	} else {
		msg += ", keyword in first 100 words: no"
	}
	return msg
}

func formatMetaMsg(length int, kwFound bool) string {
	msg := "Meta description length: " + itoa(length) + " chars"
	if kwFound {
		msg += ", keyword found"
	} else {
		msg += ", keyword not found"
	}
	return msg
}

func formatIntentMsg(intent string) string {
	return "Detected intent: " + intent
}

func formatContentMsg(paras, links, images int) string {
	return "Paragraphs: " + itoa(paras) + ", links: " + itoa(links) + ", images: " + itoa(images)
}

func formatStructureSummary(hIssues, pIssues []StructureIssue) string {
	total := len(hIssues) + len(pIssues)
	if total == 0 {
		return "No structure issues found"
	}
	return itoa(total) + " structure issues found"
}

func formatFREScore(fre float64, language string) string {
	label := "Flesch Reading Ease: " + fmtFloat(fre, 1)
	if language == "pt" {
		label = "Flesch adaptado: " + fmtFloat(fre, 1)
	}
	if fre >= 90 {
		label += " (very easy)"
	} else if fre >= 80 {
		label += " (easy)"
	} else if fre >= 70 {
		label += " (fairly easy)"
	} else if fre >= 60 {
		label += " (standard)"
	} else if fre >= 50 {
		label += " (fairly difficult)"
	} else if fre >= 30 {
		label += " (difficult)"
	} else {
		label += " (very difficult)"
	}
	return label
}

func fmtGradeLevel(fk float64) string {
	return "Grade level: " + fmtFloat(fk, 1)
}

func fmtAvgSentenceLen(avg float64) string {
	return "Average sentence: " + fmtFloat(avg, 1) + " words"
}

func fmtDifficultWords(pct float64) string {
	return "Difficult words: " + fmtFloat(pct, 1) + "%"
}

func fmtFloat(v float64, decimals int) string {
	if decimals <= 0 {
		return itoa(int(math.Round(v)))
	}
	mult := 1.0
	for i := 0; i < decimals; i++ {
		mult *= 10
	}
	rounded := math.Round(v*mult) / mult
	intPart := int(rounded)
	frac := int(math.Round((rounded - float64(intPart)) * mult))
	fracStr := fmtInt(frac, decimals)
	return itoa(intPart) + "." + fracStr
}

func fmtInt(n, width int) string {
	s := itoa(n)
	for len(s) < width {
		s = "0" + s
	}
	if len(s) > width {
		s = s[:width]
	}
	return s
}

func formatEntityCoverageDetails(found, total int) string {
	return "Entity coverage: " + itoa(found) + "/" + itoa(total) + " entities found"
}
