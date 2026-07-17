package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"text/template"
)

type promptBuilder struct {
	mu        sync.RWMutex
	templates map[string]PromptTemplate
}

func NewPromptBuilder() *promptBuilder {
	pb := &promptBuilder{
		templates: make(map[string]PromptTemplate),
	}
	pb.registerDefaults()
	return pb
}

func (pb *promptBuilder) Build(ctx context.Context, templateID string, variables map[string]string) (*CompletionRequest, error) {
	pb.mu.RLock()
	tmpl, ok := pb.templates[templateID]
	pb.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidPromptTemplate, templateID)
	}

	filled := make(map[string]string)
	for _, v := range tmpl.Variables {
		if val, ok := variables[v]; ok && val != "" {
			filled[v] = val
		} else if def, ok := tmpl.Defaults[v]; ok {
			filled[v] = def
		} else {
			filled[v] = "[" + v + "]"
		}
	}

	content := tmpl.Template
	for k, v := range filled {
		content = strings.ReplaceAll(content, "{{."+k+"}}", v)
	}

	system := tmpl.System
	for k, v := range filled {
		system = strings.ReplaceAll(system, "{{."+k+"}}", v)
	}

	return &CompletionRequest{
		Prompt: content,
		System: system,
	}, nil
}

func (pb *promptBuilder) RegisterTemplate(tmpl PromptTemplate) error {
	if tmpl.ID == "" {
		return ErrInvalidPromptTemplate
	}
	if tmpl.Template == "" {
		return ErrInvalidPromptTemplate
	}

	_, err := template.New(tmpl.ID).Parse(tmpl.Template)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPromptTemplate, err)
	}

	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.templates[tmpl.ID] = tmpl
	return nil
}

func (pb *promptBuilder) ListTemplates(language string) ([]PromptTemplate, error) {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	var result []PromptTemplate
	for _, tmpl := range pb.templates {
		if language == "" || tmpl.Language == language {
			result = append(result, tmpl)
		}
	}
	if result == nil {
		result = []PromptTemplate{}
	}
	return result, nil
}

func (pb *promptBuilder) GetTemplate(id string) (*PromptTemplate, error) {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	tmpl, ok := pb.templates[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidPromptTemplate, id)
	}
	return &tmpl, nil
}

func (pb *promptBuilder) registerDefaults() {
	defaults := []PromptTemplate{
		{
			ID:          PromptTypeArticle,
			Name:        "Article Generation",
			Description: "Generate a complete article from briefing",
			Language:    "en",
			System:      "You are a professional content writer. Write a high-quality article following the instructions. Use {{.tone}} tone for {{.audience}} audience.",
			Template:    "Title: {{.title}}\n\nArticle Type: {{.article_type}}\n\nWord Count: {{.word_count}}\n\nKeywords: {{.keywords}}\n\nInstructions:\n{{.instructions}}\n\nResearch Briefing:\n{{.briefing}}\n\nOutline:\n{{.outline}}\n\nPlease write a complete, well-structured article.",
			Variables:   []string{"title", "article_type", "word_count", "keywords", "instructions", "briefing", "outline", "tone", "audience"},
			Defaults:    map[string]string{"word_count": "1000", "tone": "professional", "audience": "general"},
			Version:     "1.0",
		},
		{
			ID:          PromptTypeArticle + "_pt",
			Name:        "Geração de Artigo",
			Description: "Generate a complete article in Portuguese from briefing",
			Language:    "pt",
			System:      "Você é um redator profissional. Escreva um artigo de alta qualidade seguindo as instruções. Use tom {{.tone}} para público {{.audience}}.",
			Template:    "Título: {{.title}}\n\nTipo de Artigo: {{.article_type}}\n\nContagem de Palavras: {{.word_count}}\n\nPalavras-chave: {{.keywords}}\n\nInstruções:\n{{.instructions}}\n\nBriefing de Pesquisa:\n{{.briefing}}\n\nEsboço:\n{{.outline}}\n\nPor favor, escreva um artigo completo e bem estruturado.",
			Variables:   []string{"title", "article_type", "word_count", "keywords", "instructions", "briefing", "outline", "tone", "audience"},
			Defaults:    map[string]string{"word_count": "1000", "tone": "profissional", "audience": "geral"},
			Version:     "1.0",
		},
		{
			ID:          PromptTypeOutline,
			Name:        "Outline Generation",
			Description: "Generate article outline from title and briefing",
			Language:    "en",
			System:      "You are a professional content strategist. Create a detailed outline for an article.",
			Template:    "Title: {{.title}}\n\nBriefing: {{.briefing}}\n\nKeywords: {{.keywords}}\n\nExpected Size: {{.word_count}} words\n\nCreate a detailed outline with sections and subsections.",
			Variables:   []string{"title", "briefing", "keywords", "word_count"},
			Defaults:    map[string]string{"word_count": "1000"},
			Version:     "1.0",
		},
		{
			ID:          PromptTypeOutline + "_pt",
			Name:        "Geração de Esboço",
			Description: "Generate article outline in Portuguese",
			Language:    "pt",
			System:      "Você é um estrategista de conteúdo profissional. Crie um esboço detalhado para um artigo.",
			Template:    "Título: {{.title}}\n\nBriefing: {{.briefing}}\n\nPalavras-chave: {{.keywords}}\n\nTamanho Esperado: {{.word_count}} palavras\n\nCrie um esboço detalhado com seções e subseções.",
			Variables:   []string{"title", "briefing", "keywords", "word_count"},
			Defaults:    map[string]string{"word_count": "1000"},
			Version:     "1.0",
		},
		{
			ID:          PromptTypeSection,
			Name:        "Section Generation",
			Description: "Generate a specific section of an article",
			Language:    "en",
			System:      "You are a professional writer. Write the requested section of the article following the established style.",
			Template:    "Section Title: {{.section_title}}\n\nArticle Context:\n{{.article_context}}\n\nStyle Guide: {{.style_guide}}\n\nWord Count: {{.word_count}}\n\nWrite the content for this section.",
			Variables:   []string{"section_title", "article_context", "style_guide", "word_count"},
			Defaults:    map[string]string{"word_count": "300"},
			Version:     "1.0",
		},
		{
			ID:          PromptTypeRevision,
			Name:        "Revision Prompt",
			Description: "Revise content based on feedback",
			Language:    "en",
			System:      "You are an expert editor. Revise the content according to the feedback provided.",
			Template:    "Original Content:\n{{.content}}\n\nFeedback:\n{{.feedback}}\n\nInstructions: {{.instructions}}\n\nProvide the revised version.",
			Variables:   []string{"content", "feedback", "instructions"},
			Version:     "1.0",
		},
		{
			ID:          PromptTypeFactCheck,
			Name:        "Fact Checking",
			Description: "Fact check content against references",
			Language:    "en",
			System:      "You are a fact-checker. Verify the content against the provided references.",
			Template:    "Content to Check:\n{{.content}}\n\nReferences:\n{{.references}}\n\nReport any inaccuracies, unsupported claims, or hallucinations.",
			Variables:   []string{"content", "references"},
			Version:     "1.0",
		},
		{
			ID:          PromptTypeSEO,
			Name:        "SEO Optimization",
			Description: "Optimize content for search engines",
			Language:    "en",
			System:      "You are an SEO specialist. Optimize the content for search engines while maintaining readability.",
			Template:    "Content:\n{{.content}}\n\nTarget Keywords: {{.keywords}}\n\nCurrent URL: {{.url}}\n\nProvide SEO-optimized title, meta description, heading improvements, and keyword placement suggestions.",
			Variables:   []string{"content", "keywords", "url"},
			Version:     "1.0",
		},
		{
			ID:          PromptTypeTranslation,
			Name:        "Translation",
			Description: "Translate content between languages",
			Language:    "en",
			System:      "You are a professional translator. Translate the content accurately while preserving meaning and tone.",
			Template:    "Translate the following content from {{.source_language}} to {{.target_language}}:\n\n{{.content}}\n\nPreserve formatting, tone, and technical terms.",
			Variables:   []string{"source_language", "target_language", "content"},
			Version:     "1.0",
		},
		{
			ID:          PromptTypeSummary,
			Name:        "Content Summary",
			Description: "Summarize content to specified length",
			Language:    "en",
			System:      "Summarize the following content concisely while preserving key information.",
			Template:    "Content:\n{{.content}}\n\nMaximum Length: {{.max_words}} words\n\nProvide a clear, concise summary.",
			Variables:   []string{"content", "max_words"},
			Defaults:    map[string]string{"max_words": "200"},
			Version:     "1.0",
		},
		{
			ID:          PromptTypeResearch,
			Name:        "Research Query",
			Description: "Generate research queries from a topic",
			Language:    "en",
			System:      "You are a research assistant. Generate research queries for the given topic.",
			Template:    "Topic: {{.topic}}\n\nDepth: {{.depth}}\n\nGenerate research questions, search queries, and key areas to investigate.",
			Variables:   []string{"topic", "depth"},
			Defaults:    map[string]string{"depth": "comprehensive"},
			Version:     "1.0",
		},
		{
			ID:          PromptTypeBriefing,
			Name:        "Research Briefing",
			Description: "Generate a research briefing from sources",
			Language:    "en",
			System:      "You are a research analyst. Create a comprehensive briefing from the provided sources.",
			Template:    "Topic: {{.topic}}\n\nSources:\n{{.sources}}\n\nCreate a briefing covering key findings, statistics, expert opinions, and relevance.",
			Variables:   []string{"topic", "sources"},
			Version:     "1.0",
		},
	}

	for _, tmpl := range defaults {
		pb.templates[tmpl.ID] = tmpl
	}
}
