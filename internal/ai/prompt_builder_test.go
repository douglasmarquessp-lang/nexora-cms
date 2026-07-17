package ai

import (
	"context"
	"testing"
)

func TestNewPromptBuilder(t *testing.T) {
	pb := NewPromptBuilder()
	if pb == nil {
		t.Fatal("expected non-nil prompt builder")
	}
}

func TestPromptBuilder_Defaults(t *testing.T) {
	pb := NewPromptBuilder()
	templates, err := pb.ListTemplates("")
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}
	if len(templates) == 0 {
		t.Error("expected default templates")
	}
}

func TestPromptBuilder_ListByLanguage(t *testing.T) {
	pb := NewPromptBuilder()
	enTemplates, _ := pb.ListTemplates("en")
	ptTemplates, _ := pb.ListTemplates("pt")

	if len(enTemplates) == 0 {
		t.Error("expected EN templates")
	}
	if len(ptTemplates) == 0 {
		t.Error("expected PT templates")
	}
}

func TestPromptBuilder_GetTemplate(t *testing.T) {
	pb := NewPromptBuilder()
	tmpl, err := pb.GetTemplate(PromptTypeArticle)
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}
	if tmpl.ID != PromptTypeArticle {
		t.Errorf("expected %s, got %s", PromptTypeArticle, tmpl.ID)
	}
}

func TestPromptBuilder_BuildArticle(t *testing.T) {
	pb := NewPromptBuilder()
	ctx := context.Background()

	req, err := pb.Build(ctx, PromptTypeArticle, map[string]string{
		"title":        "Test Article",
		"article_type": "blog",
		"keywords":     "test, article",
		"instructions": "Write a good article",
		"briefing":     "Topic about testing",
		"outline":      "1. Intro 2. Body 3. Conclusion",
		"tone":         "professional",
		"audience":     "developers",
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if req.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
	if req.System == "" {
		t.Error("expected non-empty system prompt")
	}
}

func TestPromptBuilder_BuildDefaultVariables(t *testing.T) {
	pb := NewPromptBuilder()
	ctx := context.Background()

	req, err := pb.Build(ctx, PromptTypeArticle, map[string]string{
		"title": "Test",
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if req.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestPromptBuilder_BuildPortuguese(t *testing.T) {
	pb := NewPromptBuilder()
	ctx := context.Background()

	req, err := pb.Build(ctx, PromptTypeArticle+"_pt", map[string]string{
		"title":        "Artigo Teste",
		"article_type": "blog",
		"keywords":     "teste",
		"instructions": "Escreva um bom artigo",
		"briefing":     "Tópico sobre testes",
		"outline":      "1. Intro 2. Corpo 3. Conclusão",
		"tone":         "profissional",
		"audience":     "desenvolvedores",
	})
	if err != nil {
		t.Fatalf("Build PT failed: %v", err)
	}
	if req.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestPromptBuilder_BuildOutline(t *testing.T) {
	pb := NewPromptBuilder()
	ctx := context.Background()

	req, err := pb.Build(ctx, PromptTypeOutline, map[string]string{
		"title":   "Test",
		"briefing": "Briefing text",
		"keywords": "kw1, kw2",
	})
	if err != nil {
		t.Fatalf("Build outline failed: %v", err)
	}
	if req.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestPromptBuilder_BuildSection(t *testing.T) {
	pb := NewPromptBuilder()
	ctx := context.Background()

	req, err := pb.Build(ctx, PromptTypeSection, map[string]string{
		"section_title":  "Introduction",
		"article_context": "Full article context",
		"style_guide":    "Professional tone",
	})
	if err != nil {
		t.Fatalf("Build section failed: %v", err)
	}
	if req.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestPromptBuilder_BuildRevision(t *testing.T) {
	pb := NewPromptBuilder()
	ctx := context.Background()

	req, err := pb.Build(ctx, PromptTypeRevision, map[string]string{
		"content":      "Original content",
		"feedback":     "Make it better",
		"instructions": "Revise thoroughly",
	})
	if err != nil {
		t.Fatalf("Build revision failed: %v", err)
	}
	if req.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestPromptBuilder_BuildFactCheck(t *testing.T) {
	pb := NewPromptBuilder()
	ctx := context.Background()

	req, err := pb.Build(ctx, PromptTypeFactCheck, map[string]string{
		"content":    "Content to check",
		"references": "Reference material",
	})
	if err != nil {
		t.Fatalf("Build fact check failed: %v", err)
	}
	if req.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestPromptBuilder_BuildSEO(t *testing.T) {
	pb := NewPromptBuilder()
	ctx := context.Background()

	req, err := pb.Build(ctx, PromptTypeSEO, map[string]string{
		"content":  "Article content",
		"keywords": "kw1, kw2, kw3",
		"url":      "https://example.com",
	})
	if err != nil {
		t.Fatalf("Build SEO failed: %v", err)
	}
	if req.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestPromptBuilder_BuildTranslation(t *testing.T) {
	pb := NewPromptBuilder()
	ctx := context.Background()

	req, err := pb.Build(ctx, PromptTypeTranslation, map[string]string{
		"content":         "Hello world",
		"source_language": "en",
		"target_language": "pt",
	})
	if err != nil {
		t.Fatalf("Build translation failed: %v", err)
	}
	if req.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestPromptBuilder_BuildSummary(t *testing.T) {
	pb := NewPromptBuilder()
	ctx := context.Background()

	req, err := pb.Build(ctx, PromptTypeSummary, map[string]string{
		"content": "Long content to summarize",
	})
	if err != nil {
		t.Fatalf("Build summary failed: %v", err)
	}
	if req.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestPromptBuilder_BuildResearch(t *testing.T) {
	pb := NewPromptBuilder()
	ctx := context.Background()

	req, err := pb.Build(ctx, PromptTypeResearch, map[string]string{
		"topic": "AI Technology",
	})
	if err != nil {
		t.Fatalf("Build research failed: %v", err)
	}
	if req.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestPromptBuilder_BuildBriefing(t *testing.T) {
	pb := NewPromptBuilder()
	ctx := context.Background()

	req, err := pb.Build(ctx, PromptTypeBriefing, map[string]string{
		"topic":   "Climate Change",
		"sources": "Source 1, Source 2",
	})
	if err != nil {
		t.Fatalf("Build briefing failed: %v", err)
	}
	if req.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestPromptBuilder_RegisterCustom(t *testing.T) {
	pb := NewPromptBuilder()
	err := pb.RegisterTemplate(PromptTemplate{
		ID:       "custom",
		Name:     "Custom Template",
		Template: "Write about {{.topic}}",
		System:   "You are a writer.",
		Variables: []string{"topic"},
		Language: "en",
		Version:  "1.0",
	})
	if err != nil {
		t.Fatalf("RegisterTemplate failed: %v", err)
	}

	tmpl, err := pb.GetTemplate("custom")
	if err != nil {
		t.Fatalf("GetTemplate custom failed: %v", err)
	}
	if tmpl.ID != "custom" {
		t.Errorf("expected custom, got %s", tmpl.ID)
	}
}

func TestPromptBuilder_RegisterInvalid(t *testing.T) {
	pb := NewPromptBuilder()
	err := pb.RegisterTemplate(PromptTemplate{
		ID:       "",
		Template: "write",
	})
	if err != ErrInvalidPromptTemplate {
		t.Errorf("expected ErrInvalidPromptTemplate, got %v", err)
	}

	err = pb.RegisterTemplate(PromptTemplate{
		ID:       "empty",
		Template: "",
	})
	if err != ErrInvalidPromptTemplate {
		t.Errorf("expected ErrInvalidPromptTemplate, got %v", err)
	}
}

func TestPromptBuilder_GetTemplate_NotFound(t *testing.T) {
	pb := NewPromptBuilder()
	_, err := pb.GetTemplate("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent template")
	}
}

func TestPromptBuilder_Build_TemplateNotFound(t *testing.T) {
	pb := NewPromptBuilder()
	ctx := context.Background()

	_, err := pb.Build(ctx, "nonexistent", nil)
	if err == nil {
		t.Error("expected error for nonexistent template")
	}
}
