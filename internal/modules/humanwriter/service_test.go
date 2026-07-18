package humanwriter

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func TestNewService(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewService_WithDB(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestSetEventBus_Nil(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	svc.SetEventBus(nil)
}

// --- Validation Tests ---

func TestCreateProfile_InvalidSlug(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.CreateProfile(context.Background(), uuid.New(), ProfileCreateRequest{
		Slug:    "",
		Name:    "Test Profile",
		Language: "pt",
	})
	if err != ErrInvalidSlug {
		t.Errorf("expected ErrInvalidSlug, got %v", err)
	}
}

func TestCreateProfile_EmptyName(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.CreateProfile(context.Background(), uuid.New(), ProfileCreateRequest{
		Slug:    "test",
		Name:    "",
		Language: "pt",
	})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateProfile_InvalidLanguage(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.CreateProfile(context.Background(), uuid.New(), ProfileCreateRequest{
		Slug:    "test",
		Name:    "Test",
		Language: "fr",
	})
	if err != ErrInvalidLanguage {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestCreateProfile_DBError(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.CreateProfile(context.Background(), uuid.New(), ProfileCreateRequest{
		Slug:    "test",
		Name:    "Test",
		Language: "pt",
	})
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestHumanize_EmptyText(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.Humanize(context.Background(), uuid.New(), HumanizeRequest{
		Text: "",
	})
	if err != ErrInvalidText {
		t.Errorf("expected ErrInvalidText, got %v", err)
	}
}

func TestHumanize_InvalidLanguage(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.Humanize(context.Background(), uuid.New(), HumanizeRequest{
		Text:     "Some text",
		Language: "fr",
	})
	if err != ErrInvalidLanguage {
		t.Errorf("expected ErrInvalidLanguage, got %v", err)
	}
}

func TestBatchHumanize_EmptyTexts(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.BatchHumanize(context.Background(), uuid.New(), BatchHumanizeRequest{
		Texts: []string{},
	})
	if err != ErrInvalidText {
		t.Errorf("expected ErrInvalidText, got %v", err)
	}
}

func TestAnalyzeText_EmptyText(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.AnalyzeText(context.Background(), AnalyzeRequest{
		Text: "",
	})
	if err != ErrInvalidText {
		t.Errorf("expected ErrInvalidText, got %v", err)
	}
}

// --- Humanization Engine Tests ---

func TestHumanize_NoDB_English(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	result, err := svc.Humanize(context.Background(), uuid.New(), HumanizeRequest{
		Text:     "This is a sample text. It has multiple sentences. Each one should be varied. This is quite interesting. Moreover, we need to test it.",
		Language: "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HumanizedText == "" {
		t.Error("expected non-empty humanized text")
	}
	if result.WordCountOriginal == 0 {
		t.Error("expected non-zero original word count")
	}
	if result.BurstinessScore < 0 || result.BurstinessScore > 1 {
		t.Error("burstiness score out of range")
	}
	if result.PerplexityScore < 0 || result.PerplexityScore > 1 {
		t.Error("perplexity score out of range")
	}
	if result.RepetitionScore < 0 || result.RepetitionScore > 1 {
		t.Error("repetition score out of range")
	}
	if result.PassiveVoiceScore < 0 || result.PassiveVoiceScore > 1 {
		t.Error("passive voice score out of range")
	}
	if result.RhythmScore < 0 || result.RhythmScore > 1 {
		t.Error("rhythm score out of range")
	}
	if result.FlowScore < 0 || result.FlowScore > 1 {
		t.Error("flow score out of range")
	}
	if len(result.RulesApplied) == 0 {
		t.Error("expected at least one rule applied")
	}
}

func TestHumanize_NoDB_Portuguese(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	result, err := svc.Humanize(context.Background(), uuid.New(), HumanizeRequest{
		Text:     "Este é um texto de exemplo. Ele contém múltiplas frases. Cada uma deve ser variada. É bastante interessante.",
		Language: "pt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HumanizedText == "" {
		t.Error("expected non-empty humanized text")
	}
	if len(result.RulesApplied) == 0 {
		t.Error("expected at least one rule applied")
	}
}

func TestHumanize_SingleSentence(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	result, err := svc.Humanize(context.Background(), uuid.New(), HumanizeRequest{
		Text:     "This is a single sentence.",
		Language: "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HumanizedText == "" {
		t.Error("expected non-empty humanized text")
	}
}

func TestBatchHumanize_English(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	result, err := svc.BatchHumanize(context.Background(), uuid.New(), BatchHumanizeRequest{
		Texts:    []string{"First text. It has sentences.", "Second one. Also multiple sentences here."},
		Language: "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
}

// --- Analysis Tests ---

func TestAnalyzeText_English(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	result, err := svc.AnalyzeText(context.Background(), AnalyzeRequest{
		Text:     "This is the first sentence. Here is another one. And a third for good measure.",
		Language: "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SentenceCount != 3 {
		t.Errorf("expected 3 sentences, got %d", result.SentenceCount)
	}
	if result.WordCount == 0 {
		t.Error("expected non-zero word count")
	}
	if result.AvgSentenceLength <= 0 {
		t.Error("expected positive average sentence length")
	}
}

func TestAnalyzeText_Portuguese(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	result, err := svc.AnalyzeText(context.Background(), AnalyzeRequest{
		Text:     "Esta é a primeira frase. Aqui está outra. E uma terceira para garantir.",
		Language: "pt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SentenceCount != 3 {
		t.Errorf("expected 3 sentences, got %d", result.SentenceCount)
	}
}

// --- Engine Utility Tests ---

func TestCalcBurstiness_ShortText(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	score := svc.calcBurstiness("Short text.")
	if score != 0 {
		t.Errorf("expected 0 for short text, got %f", score)
	}
}

func TestCalcBurstiness_Normal(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	score := svc.calcBurstiness("Short. A much longer sentence that goes on and on. Medium length one. Another short one. Very long sentence with many words that just keep going and going without stopping at all.")
	if score < 0 || score > 1 {
		t.Errorf("burstiness score %f out of range [0,1]", score)
	}
}

func TestCalcPerplexity_ShortText(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	score := svc.calcPerplexity("Hi", "en")
	if score != 1.0 {
		t.Errorf("expected 1.0 for very short text, got %f", score)
	}
}

func TestCalcPerplexity_Normal(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	score := svc.calcPerplexity("The quick brown fox jumps over the lazy dog near the bank of the river.", "en")
	if score < 0 || score > 1 {
		t.Errorf("perplexity score %f out of range [0,1]", score)
	}
}

func TestDetectRepetition_NoRepetition(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	score := svc.detectRepetition("The quick brown fox jumps over the lazy dog.")
	if score != 0 {
		t.Errorf("expected 0 for non-repetitive text, got %f", score)
	}
}

func TestDetectRepetition_HighRepetition(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	score := svc.detectRepetition("The the the the the the the the the the dog dog dog dog dog cat cat cat cat cat.")
	if score <= 0 {
		t.Errorf("expected positive for repetitive text, got %f", score)
	}
}

func TestDetectPassiveVoice_Portuguese(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	score := svc.detectPassiveVoice("Os relatórios foram enviados pelo gerente. As cartas foram escritas pela secretária.", "pt")
	if score <= 0 {
		t.Errorf("expected positive for passive voice in PT, got %f", score)
	}
}

func TestDetectPassiveVoice_Active(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	score := svc.detectPassiveVoice("The manager sent the report. The secretary wrote the letter.", "en")
	if score != 0 {
		t.Errorf("expected 0 for active voice, got %f", score)
	}
}

func TestAnalyzeRhythm_ShortText(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	score := svc.analyzeRhythm("Short text.")
	if score != 0.5 {
		t.Errorf("expected 0.5 for short text, got %f", score)
	}
}

func TestAnalyzeRhythm_Normal(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	score := svc.analyzeRhythm("Short sentence. A much longer sentence with many words here. Brief. Another extensive sentence that keeps going and going without stopping. Tiny. Huge lengthy sentence with many many words in it.")
	if score < 0 || score > 1 {
		t.Errorf("rhythm score %f out of range [0,1]", score)
	}
}

func TestAnalyzeFlow_SingleParagraph(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	score := svc.analyzeFlow("This is a single paragraph. It has multiple sentences. But still just one paragraph.")
	if score != 0.5 {
		t.Errorf("expected 0.5 for single paragraph, got %f", score)
	}
}

func TestAnalyzeFlow_MultipleParagraphs(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	score := svc.analyzeFlow("First paragraph here. Some more content.\n\nHowever, the second paragraph starts with a connector. This is good.\n\nFinally, the third paragraph also has a connector.")
	if score < 0 || score > 1 {
		t.Errorf("flow score %f out of range [0,1]", score)
	}
}

func TestRemoveAICliches_English(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	result := svc.removeAICliches("In today's world, technology is important. In the digital era, things change fast.", "en")
	if result == "" {
		t.Error("expected non-empty result after cliche removal")
	}
}

func TestRemoveAICliches_Portuguese(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	result := svc.removeAICliches("No mundo atual, a tecnologia é importante. Na era digital, as coisas mudam rapidamente.", "pt")
	if result == "" {
		t.Error("expected non-empty result after cliche removal")
	}
}

func TestApplyConnectorRotation_English(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	result := svc.applyConnectorRotation("Furthermore, this is a test. However, we need to check it. Therefore, it works.", "en", nil)
	if result == "" {
		t.Error("expected non-empty result after connector rotation")
	}
}

func TestApplyConnectorRotation_Portuguese(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	result := svc.applyConnectorRotation("Além disso, isto é um teste. Porém, precisamos verificar. Portanto, funciona.", "pt", nil)
	if result == "" {
		t.Error("expected non-empty result after connector rotation")
	}
}

func TestApplyVocabularyDiversity_English(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	result := svc.applyVocabularyDiversity("This is a good example. It is very interesting and important.", "en", nil)
	if result == "" {
		t.Error("expected non-empty result after vocabulary diversity")
	}
}

func TestApplyVocabularyDiversity_Portuguese(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	result := svc.applyVocabularyDiversity("Isto é um bom exemplo. É muito interessante e importante.", "pt", nil)
	if result == "" {
		t.Error("expected non-empty result after vocabulary diversity")
	}
}

func TestInsertPlaceholders_English(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	result := svc.insertPlaceholders("According to [expert], this is correct. The [statistic] confirms it.", "en")
	if result == "" {
		t.Error("expected non-empty result after placeholder insertion")
	}
}

func TestInsertPlaceholders_Portuguese(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	result := svc.insertPlaceholders("De acordo com [especialista], isto está correto. A [estatística] confirma.", "pt")
	if result == "" {
		t.Error("expected non-empty result after placeholder insertion")
	}
}

func TestApplyParagraphNormalization_NilProfile(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	result := svc.applyParagraphNormalization("Para one. With content.\n\nPara two. More content.\n\nPara three. Even more.", &WritingProfile{
		ParagraphSizeMin: 2,
		ParagraphSizeMax: 5,
	})
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestGetExpansionPhrase_Portuguese(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	for i := 0; i < 10; i++ {
		p := svc.getExpansionPhrase("pt")
		if p == "" {
			t.Error("expected non-empty expansion phrase for PT")
		}
	}
}

func TestGetExpansionPhrase_English(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	for i := 0; i < 10; i++ {
		p := svc.getExpansionPhrase("en")
		if p == "" {
			t.Error("expected non-empty expansion phrase for EN")
		}
	}
}

func TestGetCompressionPhrase(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)
	for _, lang := range []string{"pt", "en"} {
		for i := 0; i < 10; i++ {
			p := svc.getCompressionPhrase(lang)
			if p == "" {
				t.Errorf("expected non-empty compression phrase for %s", lang)
			}
		}
	}
}

func TestGetMetrics_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.GetMetrics(context.Background(), uuid.New())
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestListProfiles_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.ListProfiles(context.Background(), uuid.New(), "pt")
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestListRules_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.ListRules(context.Background(), uuid.New(), nil)
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestListPersonas_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.ListPersonas(context.Background(), uuid.New(), nil, "en")
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestListVocabularySets_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.ListVocabularySets(context.Background(), uuid.New(), "", "")
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestListTransitions_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.ListTransitions(context.Background(), uuid.New(), "", "")
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestListPatterns_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.ListPatterns(context.Background(), uuid.New(), "", "")
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestListTemplates_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.ListTemplates(context.Background(), uuid.New(), "", "")
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestListHistory_NoDB(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil, nil)

	_, err := svc.ListHistory(context.Background(), uuid.New(), nil, "", 10, 0)
	if err != ErrDatabaseNotAvail {
		t.Errorf("expected ErrDatabaseNotAvail, got %v", err)
	}
}

func TestTimeFunc(t *testing.T) {
	_ = time.Now()
}
