package translation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"nexora/internal/ai"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func newTestService() *Service {
	cfg := &config.Config{}
	log := logger.New(cfg)
	return NewService(cfg, log, nil, nil)
}

func serviceWithAIManager(t *testing.T) *Service {
	t.Helper()
	svc := newTestService()
	m := ai.NewManager(ai.DefaultConfig(), logger.New(&config.Config{}))
	p := ai.NewMockProvider("mock", "mock-model", nil)
	if err := m.RegisterProvider(p, ai.ProviderCfg{Name: "mock", Enabled: true, Priority: 1, Weight: 10, MaxRetries: 1}); err != nil {
		t.Fatalf("failed to register mock provider: %v", err)
	}
	svc.SetAIManager(m)
	return svc
}

func TestRunTranslate_WithAIManager(t *testing.T) {
	svc := serviceWithAIManager(t)
	ctx := context.Background()

	job := &TranslationJob{
		ID:             uuid.New(),
		Title:          "Guia de SEO",
		Content:        "# Introdução\n\nEste é um conteúdo sobre SEO.",
		SourceLanguage: "pt",
		TargetLanguage: "en",
	}

	status, _, result, _, err := svc.runTranslate(ctx, job, nil)
	if err != nil {
		t.Fatalf("runTranslate failed: %v", err)
	}
	if status != StageCompleted {
		t.Errorf("expected completed, got %s", status)
	}
	if strings.TrimSpace(result.Content) == "" {
		t.Error("expected translated content")
	}
	if result.Title == "" {
		t.Error("expected translated title")
	}
}

func TestRunTranslate_NoAIManager(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	job := &TranslationJob{
		ID:             uuid.New(),
		Title:          "T",
		Content:        "Conteúdo",
		SourceLanguage: "pt",
		TargetLanguage: "en",
	}
	status, _, _, _, err := svc.runTranslate(ctx, job, nil)
	if status != StageFailed {
		t.Errorf("expected failed, got %s", status)
	}
	if !errors.Is(err, ErrAIManagerRequired) {
		t.Errorf("expected ErrAIManagerRequired, got %v", err)
	}
}

func TestRunNativeReview_Deterministic(t *testing.T) {
	svc := newTestService()
	svc.SetQualityChecker(ai.NewQualityChecker())
	ctx := context.Background()

	job := &TranslationJob{
		ID:             uuid.New(),
		Title:          "Product Review",
		Content:        "The product works well for our customers. It saves time and money every month.",
		SourceLanguage: "pt",
		TargetLanguage: "en",
	}

	s1, sc1, r1, _, err1 := svc.runNativeReview(ctx, job, nil)
	s2, sc2, _, _, err2 := svc.runNativeReview(ctx, job, nil)
	if err1 != nil || err2 != nil {
		t.Fatalf("runNativeReview failed: %v / %v", err1, err2)
	}
	if s1 != s2 {
		t.Errorf("status not deterministic: %s vs %s", s1, s2)
	}
	if sc1 == nil || sc2 == nil || *sc1 != *sc2 {
		t.Errorf("score not deterministic: %v vs %v", sc1, sc2)
	}
	if r1.Content == "" {
		t.Error("expected content in result")
	}
}

func TestWorstSection(t *testing.T) {
	text := "First paragraph is fine.\n\nThis section is in order to weird and in order to broken.\n\nLast paragraph ok."
	idx, section := worstSection(text, "en")
	if idx != 1 {
		t.Errorf("expected paragraph 1, got %d", idx)
	}
	if !strings.Contains(section, "in order to") {
		t.Errorf("unexpected section: %s", section)
	}
}

func TestWorstSection_None(t *testing.T) {
	idx, section := worstSection("All paragraphs are perfectly natural here.", "en")
	if idx != -1 || section != "" {
		t.Errorf("expected no worst section, got (%d, %q)", idx, section)
	}
}

func TestSplitParagraphs(t *testing.T) {
	paras := splitParagraphs("One.\n\nTwo.\n\nThree.")
	if len(paras) != 3 {
		t.Errorf("expected 3 paragraphs, got %d", len(paras))
	}
}

func TestGenerateSEO_Fallback(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	job := &TranslationJob{
		Title:          "Guia Completo de Marketing",
		Content:        "Conteúdo do artigo original.",
		SourceLanguage: "pt",
		TargetLanguage: "en",
	}
	native := StageResult{Title: "Complete Marketing Guide", Content: "Article content here."}

	meta := svc.generateSEO(ctx, job, native)
	if meta.Title != "Complete Marketing Guide" {
		t.Errorf("expected fallback title, got %s", meta.Title)
	}
	if meta.Slug == "" {
		t.Error("expected fallback slug")
	}
	if meta.PrimaryKeyword == "" {
		t.Error("expected fallback keyword")
	}
	if meta.MetaDescription == "" {
		t.Error("expected fallback meta description")
	}
}

func TestGenerateSEO_WithAIManager(t *testing.T) {
	svc := serviceWithAIManager(t)
	ctx := context.Background()

	job := &TranslationJob{
		Title:          "Título",
		Content:        "Conteúdo",
		SourceLanguage: "pt",
		TargetLanguage: "en",
	}
	meta := svc.generateSEO(ctx, job, StageResult{Title: "Title", Content: "Content"})
	// Mock provider returns non-JSON text: deterministic fallback kicks in.
	if meta.Title == "" || meta.Slug == "" {
		t.Error("expected usable fallback SEO metadata")
	}
}

func TestParseJSONObject(t *testing.T) {
	var out struct {
		Title string `json:"title"`
	}
	if err := parseJSONObject("here: {\"title\": \"Hello\"} done", &out); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if out.Title != "Hello" {
		t.Errorf("unexpected parse result: %s", out.Title)
	}
	if err := parseJSONObject("no json here", &out); err == nil {
		t.Error("expected error for missing JSON")
	}
}

func TestTruncateTo(t *testing.T) {
	if truncateTo("short", 10) != "short" {
		t.Error("truncateTo should not cut short strings")
	}
	if len(truncateTo(strings.Repeat("a", 200), 155)) != 155 {
		t.Error("truncateTo should cut long strings")
	}
}

func TestFinalScore_NilDB(t *testing.T) {
	svc := newTestService()
	job := &TranslationJob{ID: uuid.New(), SourceLanguage: "pt", TargetLanguage: "en"}
	if sc := svc.finalScore(context.Background(), job, nil); sc != nil {
		t.Error("expected nil score without database")
	}
}

func TestFallbackSEO(t *testing.T) {
	svc := newTestService()
	job := &TranslationJob{Title: "Guia de SEO Avançado", TargetLanguage: "en"}
	meta := svc.fallbackSEO(job, StageResult{Title: "Advanced SEO Guide", Content: "Body"})
	if meta.Title != "Advanced SEO Guide" {
		t.Errorf("unexpected title: %s", meta.Title)
	}
	if meta.Slug != "advanced-seo-guide" {
		t.Errorf("unexpected slug: %s", meta.Slug)
	}
}
