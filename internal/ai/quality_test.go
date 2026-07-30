package ai

import (
	"context"
	"strings"
	"testing"
)

func TestNewQualityChecker(t *testing.T) {
	qc := NewQualityChecker()
	if qc == nil {
		t.Fatal("expected non-nil quality checker")
	}
}

// ---------- Legacy API compatibility ----------

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
	if result.Score != 100 {
		t.Errorf("expected 100 for full coverage, got %f", result.Score)
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

// ---------- New deterministic tests ----------

func TestCheckGrammarDetails(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	report, err := qc.CheckGrammarDetails(ctx, "This is proper text. It has correct capitalization.", "en")
	if err != nil {
		t.Fatalf("CheckGrammarDetails failed: %v", err)
	}
	if report.OverallScore <= 0 {
		t.Errorf("expected positive score, got %f", report.OverallScore)
	}
}

func TestCheckGrammarDetails_Issues(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	// Text with multiple grammar issues
	report, err := qc.CheckGrammarDetails(ctx, "lowercase start.  double space.. repeated repeated word", "en")
	if err != nil {
		t.Fatalf("CheckGrammarDetails failed: %v", err)
	}
	if len(report.Issues) == 0 {
		t.Error("expected grammar issues for problematic text")
	}
	if report.Passed {
		t.Error("expected not passed for text with grammar issues")
	}
}

func TestCheckGrammarDetails_Capitalization(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	report, err := qc.CheckGrammarDetails(ctx, "this starts lowercase. It should be capitalized.", "en")
	if err != nil {
		t.Fatalf("CheckGrammarDetails failed: %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Type == "capitalization" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected capitalization issue for lowercase start")
	}
}

func TestCheckGrammarDetails_RepeatedWords(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	report, err := qc.CheckGrammarDetails(ctx, "This has a a repeated word in it. The the double is here.", "en")
	if err != nil {
		t.Fatalf("CheckGrammarDetails failed: %v", err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Type == "repeated_word" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected repeated word issue")
	}
}

func TestCheckGrammarDetails_Empty(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	report, err := qc.CheckGrammarDetails(ctx, "", "en")
	if err != nil {
		t.Fatalf("CheckGrammarDetails empty failed: %v", err)
	}
	if report.OverallScore != 100 {
		t.Errorf("expected 100 for empty text, got %f", report.OverallScore)
	}
}

func TestAssessSEO(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	text := "# Golang Programming Guide\n\nThis article about golang programming covers key concepts. Golang is great for developers."
	analysis, err := qc.AssessSEO(ctx, text, []string{"golang", "programming"})
	if err != nil {
		t.Fatalf("AssessSEO failed: %v", err)
	}
	if analysis.OverallScore <= 0 {
		t.Errorf("expected positive score, got %f", analysis.OverallScore)
	}
}

func TestAssessSEO_NoKeywords(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	analysis, err := qc.AssessSEO(ctx, "Some text without keywords.", nil)
	if err != nil {
		t.Fatalf("AssessSEO no keywords failed: %v", err)
	}
	if analysis.Passed {
		t.Error("expected not passed without keywords")
	}
}

func TestAssessSEO_TitleScore(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	text := "# Golang\n\nContent about golang programming."
	analysis, err := qc.AssessSEO(ctx, text, []string{"golang"})
	if err != nil {
		t.Fatalf("AssessSEO failed: %v", err)
	}
	if analysis.TitleScore == nil {
		t.Fatal("expected TitleScore")
	}
	if analysis.TitleScore.Passed != (analysis.TitleScore.Score >= 60) {
		t.Errorf("TitleScore.Passed inconsistency: score=%f, passed=%v", analysis.TitleScore.Score, analysis.TitleScore.Passed)
	}
}

func TestAssessSEO_MetaDescription(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	text := `# Test Article
<meta name="description" content="This is a meta description about golang programming covering key concepts and best practices for developers in 2023.">
Content about golang.`
	analysis, err := qc.AssessSEO(ctx, text, []string{"golang"})
	if err != nil {
		t.Fatalf("AssessSEO failed: %v", err)
	}
	if analysis.MetaDescScore == nil {
		t.Fatal("expected MetaDescScore")
	}
	if analysis.MetaDescScore.Score <= 0 {
		t.Errorf("expected positive meta desc score, got %f", analysis.MetaDescScore.Score)
	}
}

func TestScoreReadabilityDetailed(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	report, err := qc.ScoreReadabilityDetailed(ctx, "The cat sat on the mat. The dog ran in the park. Birds sing in the trees.", "en")
	if err != nil {
		t.Fatalf("ScoreReadabilityDetailed failed: %v", err)
	}
	if report.OverallScore <= 0 {
		t.Errorf("expected positive score, got %f", report.OverallScore)
	}
	if report.WordCount != 18 {
		t.Errorf("expected 18 words, got %d", report.WordCount)
	}
	if report.SentenceCount != 3 {
		t.Errorf("expected 3 sentences, got %d", report.SentenceCount)
	}
	if report.FleschReadingEase <= 0 {
		t.Errorf("expected positive FRE, got %f", report.FleschReadingEase)
	}
}

func TestScoreReadabilityDetailed_Empty(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	report, err := qc.ScoreReadabilityDetailed(ctx, "", "en")
	if err != nil {
		t.Fatalf("ScoreReadabilityDetailed empty failed: %v", err)
	}
	if report.Passed {
		t.Error("expected not passed for empty text")
	}
}

func TestScoreReadabilityDetailed_Portuguese(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	report, err := qc.ScoreReadabilityDetailed(ctx, "O gato sentou no tapete. O cachorro correu no parque. Os pássaros cantam nas árvores.", "pt")
	if err != nil {
		t.Fatalf("ScoreReadabilityDetailed PT failed: %v", err)
	}
	if report.OverallScore <= 0 {
		t.Errorf("expected positive score, got %f", report.OverallScore)
	}
}

func TestScoreReadabilityDetailed_DifficultWords(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	report, err := qc.ScoreReadabilityDetailed(ctx, "The incomprehensible juxtaposition of extraordinary philosophical considerations demonstrates the fundamental complexity of metaphysical epistemological frameworks.", "en")
	if err != nil {
		t.Fatalf("ScoreReadabilityDetailed failed: %v", err)
	}
	if report.DifficultWordCount < 3 {
		t.Errorf("expected many difficult words, got %d", report.DifficultWordCount)
	}
}

func TestCheckDuplicateBlocks(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	text := "This is a repeated phrase that appears again. This is a repeated phrase that appears again. More content here."
	blocks, err := qc.CheckDuplicateBlocks(ctx, text, 5)
	if err != nil {
		t.Fatalf("CheckDuplicateBlocks failed: %v", err)
	}
	if len(blocks) == 0 {
		t.Error("expected at least one duplicate block")
	}
}

func TestCheckDuplicateBlocks_NoDuplicates(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	text := "The quick brown fox jumps over the lazy dog near the riverbank."
	blocks, err := qc.CheckDuplicateBlocks(ctx, text, 10)
	if err != nil {
		t.Fatalf("CheckDuplicateBlocks failed: %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks for unique text, got %d", len(blocks))
	}
}

func TestCheckDuplicateBlocks_ShortText(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	blocks, err := qc.CheckDuplicateBlocks(ctx, "Short.", 10)
	if err != nil {
		t.Fatalf("CheckDuplicateBlocks short failed: %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks for short text, got %d", len(blocks))
	}
}

func TestValidateStructure(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	text := "# Main Title\n\nThis is a paragraph with enough content to be meaningful for testing purposes.\n\n## Section One\n\nMore content here.\n\n## Section Two\n\nFinal paragraph content."
	report, err := qc.ValidateStructure(ctx, text)
	if err != nil {
		t.Fatalf("ValidateStructure failed: %v", err)
	}
	if report.OverallScore <= 0 {
		t.Errorf("expected positive score, got %f", report.OverallScore)
	}
}

func TestValidateStructure_NoH1(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	report, err := qc.ValidateStructure(ctx, "## Section One\n\nSome paragraph content.\n\n### Subsection\n\nMore content.")
	if err != nil {
		t.Fatalf("ValidateStructure no H1 failed: %v", err)
	}
	if len(report.HeadingIssues) == 0 {
		t.Error("expected heading issues when no H1")
	}
	hasMissingH1 := false
	for _, issue := range report.HeadingIssues {
		if issue.Type == "missing_h1" {
			hasMissingH1 = true
			break
		}
	}
	if !hasMissingH1 {
		t.Error("expected missing_h1 issue")
	}
}

func TestValidateStructure_MultipleH1(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	report, err := qc.ValidateStructure(ctx, "# First H1\n\nContent.\n\n# Second H1\n\nMore content.")
	if err != nil {
		t.Fatalf("ValidateStructure multiple H1 failed: %v", err)
	}
	hasMultipleH1 := false
	for _, issue := range report.HeadingIssues {
		if issue.Message != "" && strings.Contains(issue.Message, "Multiple H1") {
			hasMultipleH1 = true
			break
		}
	}
	if !hasMultipleH1 {
		t.Error("expected multiple H1 issue")
	}
}

func TestValidateStructure_Images(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	text := "# Title\n\n![alt text](image.png)\n\nContent here."
	report, err := qc.ValidateStructure(ctx, text)
	if err != nil {
		t.Fatalf("ValidateStructure images failed: %v", err)
	}
	if report.ImageCount != 1 {
		t.Errorf("expected 1 image, got %d", report.ImageCount)
	}
}

func TestCheckHallucinationWithGrounding(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	text := "The theory of relativity was developed by Albert Einstein. Python is a popular programming language."
	references := []string{"Albert Einstein developed the theory of relativity in 1915", "Python is a programming language created by Guido van Rossum"}

	report, err := qc.CheckHallucinationWithGrounding(ctx, text, references, nil)
	if err != nil {
		t.Fatalf("CheckHallucinationWithGrounding failed: %v", err)
	}
	if report.ClaimsChecked == 0 {
		t.Error("expected at least one claim checked")
	}
	if report.Supported == 0 {
		t.Error("expected at least one supported claim")
	}
}

func TestCheckHallucinationWithGrounding_NoSources(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	report, err := qc.CheckHallucinationWithGrounding(ctx, "Some content.", nil, nil)
	if err != nil {
		t.Fatalf("CheckHallucinationWithGrounding failed: %v", err)
	}
	if !report.Passed {
		t.Error("expected passed with no references")
	}
}

func TestCheckHallucinationWithGrounding_GroundingMetadata(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	gm := &GroundingMetadata{
		Sources: []GroundingSource{
			{
				URI:     "https://example.com/source1",
				Title:   "AI Technology Overview",
				Snippet: "Artificial intelligence is transforming industries worldwide.",
				IsVerified: true,
			},
		},
		Unverified: false,
	}

	text := "Artificial intelligence is transforming industries worldwide."
	report, err := qc.CheckHallucinationWithGrounding(ctx, text, nil, gm)
	if err != nil {
		t.Fatalf("CheckHallucinationWithGrounding failed: %v", err)
	}
	if !report.Grounded {
		t.Error("expected grounded report")
	}
	if report.Supported == 0 && report.ClaimsChecked > 0 {
		t.Error("expected at least one supported claim with matching source")
	}
}

func TestCheckHallucinationWithGrounding_UnverifiedSource(t *testing.T) {
	qc := NewQualityChecker()
	ctx := context.Background()

	gm := &GroundingMetadata{
		Sources: []GroundingSource{
			{
				URI:     "https://example.com/source1",
				Title:   "Unverified Source",
				Snippet: "Some unverified content for testing.",
				IsVerified: false,
			},
		},
		Unverified: true,
	}

	text := "This claim is not in the source material at all and should be unsupported."
	report, err := qc.CheckHallucinationWithGrounding(ctx, text, nil, gm)
	if err != nil {
		t.Fatalf("CheckHallucinationWithGrounding failed: %v", err)
	}
	_ = report
}

// ---------- Internal helper tests ----------

func TestCountSyllables(t *testing.T) {
	tests := []struct {
		word     string
		expected int
	}{
		{"cat", 1},
		{"hello", 2},
		{"programming", 3},
		{"the", 1},
		{"simple", 2},
		{"beautiful", 4},
		{"idea", 3},
	}
	for _, tt := range tests {
		got := countSyllables(tt.word)
		if got != tt.expected {
			t.Errorf("countSyllables(%q) = %d, want %d", tt.word, got, tt.expected)
		}
	}
}

func TestCountSyllablesPT(t *testing.T) {
	tests := []struct {
		word     string
		expected int
	}{
		{"gato", 2},
		{"cachorro", 3},
		{"programação", 4},
		{"simples", 2},
		{"bonito", 3},
	}
	for _, tt := range tests {
		got := countSyllablesPT(tt.word)
		if got != tt.expected {
			t.Errorf("countSyllablesPT(%q) = %d, want %d", tt.word, got, tt.expected)
		}
	}
}

func TestTextWords(t *testing.T) {
	words := textWords("Hello, world! This is a test.")
	if len(words) != 6 {
		t.Errorf("expected 6 words, got %d", len(words))
	}
}

func TestTextSentences(t *testing.T) {
	sentences := textSentences("First sentence. Second sentence! Third?")
	if len(sentences) != 3 {
		t.Errorf("expected 3 sentences, got %d", len(sentences))
	}
}

func TestSeverityFromScore(t *testing.T) {
	if severityFromScore(90) != "info" {
		t.Error("expected info for 90")
	}
	if severityFromScore(70) != "warning" {
		t.Error("expected warning for 70")
	}
	if severityFromScore(50) != "error" {
		t.Error("expected error for 50")
	}
}

func TestExtractKeyTerms(t *testing.T) {
	terms := extractKeyTerms("The cat sat on the mat and watched the bird.")
	if len(terms) == 0 {
		t.Error("expected at least one key term")
	}
	for _, term := range terms {
		if len(term) <= 3 {
			t.Errorf("expected key terms longer than 3 chars, got %q", term)
		}
	}
}

func TestClamp(t *testing.T) {
	if clamp(150, 0, 100) != 100 {
		t.Error("clamp should cap at max")
	}
	if clamp(-10, 0, 100) != 0 {
		t.Error("clamp should floor at min")
	}
	if clamp(50, 0, 100) != 50 {
		t.Error("clamp should pass through")
	}
}

func TestCompletenessPercent(t *testing.T) {
	pct := completenessPercent("")
	if pct != 0 {
		t.Errorf("expected 0 for empty text, got %f", pct)
	}

	text := "# Title\n\nBody paragraph with enough words to be meaningful for testing purposes and completeness evaluation.\n\n## Subheading\n\nMore content with links to [example](https://example.com)."
	pct = completenessPercent(text)
	if pct <= 0 {
		t.Errorf("expected positive completeness, got %f", pct)
	}
}

func TestFormatFREScore(t *testing.T) {
	msg := formatFREScore(75.5, "en")
	if msg == "" {
		t.Error("expected non-empty FRE message")
	}
}

func TestFormatKeywordMsg(t *testing.T) {
	msg := formatKeywordMsg(2.5, 10, true)
	if msg == "" {
		t.Error("expected non-empty keyword message")
	}
}
