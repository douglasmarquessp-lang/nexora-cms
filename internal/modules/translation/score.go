package translation

import (
	"context"
	"regexp"
	"strings"

	"nexora/internal/ai"
	"nexora/internal/modules/seoengine"
)

// TranslationScore computation. All components are deterministic:
//   - grammar:       ai.QualityChecker.CheckGrammarDetails (deterministic patterns)
//   - fluency:       ai.QualityChecker.ScoreReadabilityDetailed (Flesch-based)
//   - naturalness:   literal-translation marker + repetition heuristics
//   - seo:           seoengine.AnalyzeArticle on the target-language package
//   - consistency:   glossary target-term coverage
//   - localization:  cultural localization pass coverage
//   - overall:       weighted average (25/20/20/15/10/10)

type ScoreInput struct {
	Text         string
	Title        string
	MetaTitle    string
	MetaDesc     string
	Slug         string
	Keyword      string
	Language     string
	GlossaryRes  GlossaryApplyResult
	LocalizeRes  LocalizationResult
	QC           ai.QualityChecker
}

var (
	literalMarkerRe = regexp.MustCompile(`(?i)(in order to|the fact that|in terms of|there is no doubt|make a |do a |very very|as well as as well|a fim de|em termos de|no que diz respeito a|tendo em vista|fazer uma|fazer um)`)
)

// countRepeatedWords counts adjacent identical words of 4+ characters
// (deterministic; go regexp has no backreferences).
func countRepeatedWords(text string) int {
	words := tokenize(text)
	count := 0
	for i := 1; i < len(words); i++ {
		if len(words[i]) >= 4 && words[i] == words[i-1] {
			count++
		}
	}
	return count
}

func naturalnessScore(ctx context.Context, text, language string, qc ai.QualityChecker) float64 {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	score := 100.0

	literal := len(literalMarkerRe.FindAllString(text, -1))
	score -= float64(literal) * 10

	repeats := countRepeatedWords(text)
	score -= float64(repeats) * 5

	if qc != nil {
		blocks, err := qc.CheckDuplicateBlocks(ctx, text, 10)
		if err == nil {
			score -= float64(len(blocks)) * 5
		}
	}

	return clampScore(round2(score))
}

func localizationScore(res LocalizationResult, text, toLang string) float64 {
	if res.Applied > 0 {
		return 100
	}
	// Missed opportunities: target text still carrying source-locale markers.
	leftover := 0
	if toLang == "en" && strings.Contains(text, "R$") {
		leftover++
	}
	if toLang == "pt" && strings.Contains(text, "US$") {
		leftover++
	}
	if leftover > 0 {
		return 50
	}
	return 80 // nothing to localize — neutral
}

func seoScore(ctx context.Context, in ScoreInput) float64 {
	keyword := in.Keyword
	if keyword == "" {
		keyword = DeriveKeyword(in.Title, in.Language)
	}
	if strings.TrimSpace(in.Text) == "" && strings.TrimSpace(in.Title) == "" {
		return 0
	}
	analysis := seoengine.AnalyzeArticle(ctx, seoengine.ArticleAnalysisInput{
		Title:           in.Title,
		MetaDescription: in.MetaDesc,
		Slug:            in.Slug,
		Content:         in.Text,
		Keyword:         keyword,
		Language:        in.Language,
	}, in.QC)
	if analysis == nil {
		return 0
	}
	return clampScore(analysis.OverallScore)
}

// ComputeTranslationScore computes all seven score dimensions.
func ComputeTranslationScore(ctx context.Context, in ScoreInput) TranslationScore {
	grammar := 100.0
	fluency := 100.0
	if in.QC != nil {
		if g, err := in.QC.CheckGrammarDetails(ctx, in.Text, in.Language); err == nil {
			grammar = clampScore(g.OverallScore)
		}
		if r, err := in.QC.ScoreReadabilityDetailed(ctx, in.Text, in.Language); err == nil {
			fluency = clampScore(r.OverallScore)
		}
	}

	naturalness := naturalnessScore(ctx, in.Text, in.Language, in.QC)
	consistency := GlossaryConsistency(in.GlossaryRes)
	localization := localizationScore(in.LocalizeRes, in.Text, in.Language)
	seo := seoScore(ctx, in)

	overall := round2(grammar*0.25 + fluency*0.20 + naturalness*0.20 + seo*0.15 + consistency*0.10 + localization*0.10)

	return TranslationScore{
		Grammar:      round2(grammar),
		Fluency:      round2(fluency),
		Naturalness:  round2(naturalness),
		SEO:          round2(seo),
		Consistency:  round2(consistency),
		Localization: round2(localization),
		Overall:      overall,
	}
}
