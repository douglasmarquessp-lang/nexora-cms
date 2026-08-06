package translation

import (
	"context"
	"testing"

	"nexora/internal/ai"
)

func TestComputeTranslationScore_Deterministic(t *testing.T) {
	ctx := context.Background()
	qc := ai.NewQualityChecker()
	text := "The product works well for our customers. It saves time and money every single month."
	in := ScoreInput{
		Text:        text,
		Title:       "Product Review",
		MetaDesc:    text,
		Slug:        "product-review",
		Keyword:     "product",
		Language:    "en",
		GlossaryRes: GlossaryApplyResult{Applicable: 0},
		LocalizeRes: LocalizationResult{},
		QC:          qc,
	}

	s1 := ComputeTranslationScore(ctx, in)
	s2 := ComputeTranslationScore(ctx, in)
	if s1 != s2 {
		t.Error("score not deterministic")
	}
	for name, v := range map[string]float64{
		"grammar": s1.Grammar, "fluency": s1.Fluency, "naturalness": s1.Naturalness,
		"seo": s1.SEO, "consistency": s1.Consistency, "localization": s1.Localization, "overall": s1.Overall,
	} {
		if v < 0 || v > 100 {
			t.Errorf("%s out of range: %f", name, v)
		}
	}
	if s1.Overall == 0 {
		t.Error("expected non-zero overall score")
	}
}

func TestComputeTranslationScore_NilQC(t *testing.T) {
	ctx := context.Background()
	in := ScoreInput{
		Text:        "A simple English sentence here.",
		Title:       "Simple Sentence",
		Language:    "en",
		GlossaryRes: GlossaryApplyResult{Applicable: 0},
		LocalizeRes: LocalizationResult{},
		QC:          nil,
	}
	s := ComputeTranslationScore(ctx, in)
	if s.Grammar != 100 || s.Fluency != 100 {
		t.Errorf("expected neutral grammar/fluency without QC, got %f/%f", s.Grammar, s.Fluency)
	}
}

func TestComputeTranslationScore_EmptyInput(t *testing.T) {
	ctx := context.Background()
	s := ComputeTranslationScore(ctx, ScoreInput{Text: "", Title: "", Language: "en", QC: ai.NewQualityChecker()})
	if s.SEO != 0 {
		t.Errorf("expected 0 SEO for empty input, got %f", s.SEO)
	}
}

func TestNaturalness_LiteralMarkers(t *testing.T) {
	ctx := context.Background()
	bad := "In order to achieve this, in order to win, in order to grow, we must act now."
	good := "We must act now to achieve this, win, and grow."
	qc := ai.NewQualityChecker()
	badScore := naturalnessScore(ctx, bad, "en", qc)
	goodScore := naturalnessScore(ctx, good, "en", qc)
	if badScore >= goodScore {
		t.Errorf("literal markers should penalize: bad=%f good=%f", badScore, goodScore)
	}
	if badScore > 100-30 {
		t.Errorf("expected at least 30 penalty for 3 markers, got %f", badScore)
	}
}

func TestNaturalness_RepeatedWords(t *testing.T) {
	ctx := context.Background()
	text := "This is very very important and we need to focus focus on it."
	score := naturalnessScore(ctx, text, "en", nil)
	if score > 100-15 {
		t.Errorf("expected repetition penalty, got %f", score)
	}
}

func TestNaturalness_Empty(t *testing.T) {
	if s := naturalnessScore(context.Background(), "", "en", nil); s != 0 {
		t.Errorf("expected 0 for empty text, got %f", s)
	}
}

func TestLocalizationScore_MissedOpportunity(t *testing.T) {
	text := "The price is R$ 50 for the whole package."
	if s := localizationScore(LocalizationResult{}, text, "en"); s != 50 {
		t.Errorf("expected 50 for unconverted currency, got %f", s)
	}
	if s := localizationScore(LocalizationResult{Applied: 2}, text, "en"); s != 100 {
		t.Errorf("expected 100 when localization applied, got %f", s)
	}
	if s := localizationScore(LocalizationResult{}, "Nothing to convert here.", "en"); s != 80 {
		t.Errorf("expected neutral 80, got %f", s)
	}
}

func TestScore_SEOComponent(t *testing.T) {
	ctx := context.Background()
	in := ScoreInput{
		Text:     "# Guide\n\nThis guide covers keyword research in depth with examples for beginners.",
		Title:    "Keyword Research Guide",
		MetaDesc: "Learn keyword research with practical examples and strategies.",
		Slug:     "keyword-research-guide",
		Keyword:  "keyword research",
		Language: "en",
		QC:       ai.NewQualityChecker(),
	}
	s := ComputeTranslationScore(ctx, in)
	if s.SEO <= 0 {
		t.Errorf("expected positive SEO score, got %f", s.SEO)
	}
}
