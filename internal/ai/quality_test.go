package ai

import (
	"context"
	"testing"
)

func TestNewQualityChecker(t *testing.T) {
	qc := NewQualityChecker()
	if qc == nil {
		t.Fatal("expected non-nil quality checker")
	}
}

func TestScoreGrammar(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	result, err := qc.ScoreGrammar(ctx, "This is a test sentence. It has proper grammar.", "en")
	if err != nil {
		t.Fatalf("ScoreGrammar failed: %v", err)
	}
	if result.MaxScore != 100 {
		t.Errorf("expected max score 100, got %f", result.MaxScore)
	}
	if result.Score <= 0 {
		t.Errorf("expected positive score, got %f", result.Score)
	}
}

func TestScoreGrammar_Empty(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	result, err := qc.ScoreGrammar(ctx, "", "en")
	if err != nil {
		t.Fatalf("ScoreGrammar empty failed: %v", err)
	}
	if result.Score != 100 {
		t.Errorf("expected 100 for empty text, got %f", result.Score)
	}
}

func TestScoreSEO(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	result, err := qc.ScoreSEO(ctx, "This article talks about golang programming and golang development.", []string{"golang", "programming"})
	if err != nil {
		t.Fatalf("ScoreSEO failed: %v", err)
	}
	if result.Score <= 0 {
		t.Errorf("expected positive score, got %f", result.Score)
	}
}

func TestScoreSEO_NoKeywords(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	result, err := qc.ScoreSEO(ctx, "Some text.", nil)
	if err != nil {
		t.Fatalf("ScoreSEO no keywords failed: %v", err)
	}
	if result.Passed {
		t.Error("expected not passed without keywords")
	}
}

func TestScoreReadability(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	result, err := qc.ScoreReadability(ctx, "Short sentences. Easy to read. Good flow.", "en")
	if err != nil {
		t.Fatalf("ScoreReadability failed: %v", err)
	}
	if result.Score <= 0 {
		t.Errorf("expected positive score, got %f", result.Score)
	}
}

func TestScoreReadability_Empty(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	result, err := qc.ScoreReadability(ctx, "", "en")
	if err != nil {
		t.Fatalf("ScoreReadability empty failed: %v", err)
	}
	if result.Passed {
		t.Error("expected not passed for empty text")
	}
}

func TestScoreEntityCoverage(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	result, err := qc.ScoreEntityCoverage(ctx, "Apple and Google are tech companies.", []string{"Apple", "Google"})
	if err != nil {
		t.Fatalf("ScoreEntityCoverage failed: %v", err)
	}
	if result.Score <= 0 {
		t.Errorf("expected positive score, got %f", result.Score)
	}
}

func TestScoreEntityCoverage_EmptyEntities(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	result, err := qc.ScoreEntityCoverage(ctx, "Some text.", nil)
	if err != nil {
		t.Fatalf("ScoreEntityCoverage empty failed: %v", err)
	}
	if !result.Passed {
		t.Error("expected passed with no entities to check")
	}
}

func TestCheckDuplicates(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	text := "This is a test. This is another test. This is a test. This is another test. This is a test. This is another test."
	results, err := qc.CheckDuplicates(ctx, text)
	if err != nil {
		t.Fatalf("CheckDuplicates failed: %v", err)
	}
	_ = results
}

func TestCheckDuplicates_Short(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	results, err := qc.CheckDuplicates(ctx, "Short text.")
	if err != nil {
		t.Fatalf("CheckDuplicates short failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 duplicates for short text, got %d", len(results))
	}
}

func TestCheckHallucination(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	result, err := qc.CheckHallucination(ctx, "Content to check against references.", []string{"Reference about Golang programming"})
	if err != nil {
		t.Fatalf("CheckHallucination failed: %v", err)
	}
	_ = result
}

func TestCheckHallucination_NoReferences(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	result, err := qc.CheckHallucination(ctx, "Some content.", nil)
	if err != nil {
		t.Fatalf("CheckHallucination no refs failed: %v", err)
	}
	if !result.Passed {
		t.Error("expected passed with no references")
	}
}

func TestCheckStructure(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	spec := StructureSpec{
		MinWords:         10,
		MaxWords:         500,
		MinParagraphs:    1,
		HasIntro:         false,
		HasConclusion:    false,
		RequiredSections: []string{},
	}

	result, err := qc.CheckStructure(ctx, "This is a test article with enough words to pass minimum requirements.", spec)
	if err != nil {
		t.Fatalf("CheckStructure failed: %v", err)
	}
	if result.Score <= 0 {
		t.Errorf("expected positive score, got %f", result.Score)
	}
}

func TestCheckStructure_WithMissingSections(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	spec := StructureSpec{
		MinWords:         1000,
		RequiredSections: []string{"Methodology", "Results"},
	}

	result, err := qc.CheckStructure(ctx, "Short text.", spec)
	if err != nil {
		t.Fatalf("CheckStructure failed: %v", err)
	}
	if result.Passed {
		t.Error("expected not passed for short text with missing sections")
	}
}

func TestCheckStructure_WithIntro(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	spec := StructureSpec{
		HasIntro: true,
	}

	result, err := qc.CheckStructure(ctx, "Introduction to the topic. This is the body. Conclusion here.", spec)
	if err != nil {
		t.Fatalf("CheckStructure with intro failed: %v", err)
	}
	_ = result
}

func TestExtractKeyFacts(t *testing.T) {
	facts := extractKeyFacts("This is a sentence with more than 5 words in it. This is another one with enough words. Short.")
	if len(facts) == 0 {
		t.Error("expected at least one fact")
	}
}
