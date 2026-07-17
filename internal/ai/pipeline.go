package ai

import (
	"context"
	"fmt"
)

type PipelineStage int

const (
	StageResearchGen   PipelineStage = iota
	StageBriefingGen
	StageOutlineGen
	StageDraftGen
	StageSEOGen
	StageQualityCheck
	StageTranslationGen
	StageFinalReview
)

var stageNames = map[PipelineStage]string{
	StageResearchGen:    "research",
	StageBriefingGen:    "briefing",
	StageOutlineGen:     "outline",
	StageDraftGen:       "draft",
	StageSEOGen:         "seo",
	StageQualityCheck:   "quality",
	StageTranslationGen: "translation",
	StageFinalReview:    "final_review",
}

type PipelineInput struct {
	Title       string            `json:"title"`
	ContentType string            `json:"content_type"`
	Language    string            `json:"language"`
	Topic       string            `json:"topic,omitempty"`
	Briefing    string            `json:"briefing,omitempty"`
	Outline     string            `json:"outline,omitempty"`
	Style       map[string]string `json:"style,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	WordCount   int               `json:"word_count,omitempty"`
	Tone        string            `json:"tone,omitempty"`
	Audience    string            `json:"audience,omitempty"`
	References  []string          `json:"references,omitempty"`
	Entities    []string          `json:"entities,omitempty"`
}

type PipelineResult struct {
	Stage       PipelineStage `json:"stage"`
	Content     string        `json:"content"`
	Error       error         `json:"error,omitempty"`
	Duration    string        `json:"duration,omitempty"`
}

type PipelineExecutor struct {
	manager *Manager
}

func NewPipelineExecutor(manager *Manager) *PipelineExecutor {
	return &PipelineExecutor{manager: manager}
}

func (pe *PipelineExecutor) ExecuteStage(ctx context.Context, stage PipelineStage, input PipelineInput) (*PipelineResult, error) {
	lang := input.Language
	if lang == "" {
		lang = "en"
	}

	switch stage {
	case StageResearchGen:
		return pe.runResearch(ctx, input)
	case StageBriefingGen:
		return pe.runBriefing(ctx, input)
	case StageOutlineGen:
		return pe.runOutline(ctx, input)
	case StageDraftGen:
		return pe.runDraft(ctx, input)
	case StageSEOGen:
		return pe.runSEO(ctx, input)
	case StageQualityCheck:
		return pe.runQuality(ctx, input)
	case StageTranslationGen:
		return pe.runTranslation(ctx, input)
	case StageFinalReview:
		return pe.runReview(ctx, input)
	default:
		return nil, fmt.Errorf("unknown pipeline stage: %d", stage)
	}
}

func (pe *PipelineExecutor) ExecuteFull(ctx context.Context, input PipelineInput) (map[PipelineStage]*PipelineResult, error) {
	results := make(map[PipelineStage]*PipelineResult)
	stages := []PipelineStage{
		StageResearchGen,
		StageBriefingGen,
		StageOutlineGen,
		StageDraftGen,
		StageSEOGen,
		StageQualityCheck,
		StageTranslationGen,
		StageFinalReview,
	}

	for _, stage := range stages {
		result, err := pe.ExecuteStage(ctx, stage, input)
		if err != nil {
			return results, fmt.Errorf("stage %s failed: %w", stageNames[stage], err)
		}
		results[stage] = result
	}

	return results, nil
}

func (pe *PipelineExecutor) runResearch(ctx context.Context, input PipelineInput) (*PipelineResult, error) {
	req, err := pe.manager.Prompts().Build(ctx, PromptTypeResearch, map[string]string{
		"topic": input.Topic,
	})
	if err != nil {
		return nil, err
	}

	result, err := pe.manager.Generate(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &PipelineResult{
		Stage:   StageResearchGen,
		Content: result.Content,
	}, nil
}

func (pe *PipelineExecutor) runBriefing(ctx context.Context, input PipelineInput) (*PipelineResult, error) {
	promptID := PromptTypeBriefing
	if input.Language == "pt" {
		promptID = PromptTypeBriefing
	}

	req, err := pe.manager.Prompts().Build(ctx, promptID, map[string]string{
		"topic":   input.Topic,
		"sources": input.Briefing,
	})
	if err != nil {
		return nil, err
	}

	result, err := pe.manager.Generate(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &PipelineResult{
		Stage:   StageBriefingGen,
		Content: result.Content,
	}, nil
}

func (pe *PipelineExecutor) runOutline(ctx context.Context, input PipelineInput) (*PipelineResult, error) {
	promptID := PromptTypeOutline
	if input.Language == "pt" {
		promptID = PromptTypeOutline + "_pt"
	}

	req, err := pe.manager.Prompts().Build(ctx, promptID, map[string]string{
		"title":   input.Title,
		"briefing": input.Briefing,
		"keywords": joinStrings(input.Keywords, ", "),
		"word_count": fmt.Sprintf("%d", input.WordCount),
	})
	if err != nil {
		return nil, err
	}

	result, err := pe.manager.Generate(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &PipelineResult{
		Stage:   StageOutlineGen,
		Content: result.Content,
	}, nil
}

func (pe *PipelineExecutor) runDraft(ctx context.Context, input PipelineInput) (*PipelineResult, error) {
	promptID := PromptTypeArticle
	if input.Language == "pt" {
		promptID = PromptTypeArticle + "_pt"
	}

	keywords := joinStrings(input.Keywords, ", ")
	styleGuide := fmt.Sprintf("tone: %s, audience: %s", input.Tone, input.Audience)
	for k, v := range input.Style {
		styleGuide += fmt.Sprintf(", %s: %s", k, v)
	}

	req, err := pe.manager.Prompts().Build(ctx, promptID, map[string]string{
		"title":        input.Title,
		"article_type": input.ContentType,
		"word_count":   fmt.Sprintf("%d", input.WordCount),
		"keywords":     keywords,
		"instructions": styleGuide,
		"briefing":     input.Briefing,
		"outline":      input.Outline,
		"tone":         input.Tone,
		"audience":     input.Audience,
	})
	if err != nil {
		return nil, err
	}

	result, err := pe.manager.Generate(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &PipelineResult{
		Stage:   StageDraftGen,
		Content: result.Content,
	}, nil
}

func (pe *PipelineExecutor) runSEO(ctx context.Context, input PipelineInput) (*PipelineResult, error) {
	req, err := pe.manager.Prompts().Build(ctx, PromptTypeSEO, map[string]string{
		"content":  input.Briefing,
		"keywords": joinStrings(input.Keywords, ", "),
	})
	if err != nil {
		return nil, err
	}

	result, err := pe.manager.Generate(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &PipelineResult{
		Stage:   StageSEOGen,
		Content: result.Content,
	}, nil
}

func (pe *PipelineExecutor) runQuality(ctx context.Context, input PipelineInput) (*PipelineResult, error) {
	text := input.Briefing

	grammar, _ := pe.manager.Quality().ScoreGrammar(ctx, text, input.Language)
	seoScore, _ := pe.manager.Quality().ScoreSEO(ctx, text, input.Keywords)
	readability, _ := pe.manager.Quality().ScoreReadability(ctx, text, input.Language)
	entities, _ := pe.manager.Quality().ScoreEntityCoverage(ctx, text, input.Entities)
	duplicates, _ := pe.manager.Quality().CheckDuplicates(ctx, text)

	result := fmt.Sprintf("Quality Check Results:\n")
	result += fmt.Sprintf("- Grammar: %.1f/100 (passed: %v)\n", grammar.Score, grammar.Passed)
	result += fmt.Sprintf("- SEO: %.1f/100 (passed: %v)\n", seoScore.Score, seoScore.Passed)
	result += fmt.Sprintf("- Readability: %.1f/100 (passed: %v)\n", readability.Score, readability.Passed)
	result += fmt.Sprintf("- Entity Coverage: %.1f/100 (passed: %v)\n", entities.Score, entities.Passed)
	result += fmt.Sprintf("- Duplicates Found: %d\n", len(duplicates))

	allPassed := grammar.Passed && seoScore.Passed && readability.Passed && entities.Passed
	if !allPassed {
		result += "\nSome checks failed. Review required."
	} else {
		result += "\nAll quality checks passed."
	}

	return &PipelineResult{
		Stage:   StageQualityCheck,
		Content: result,
	}, nil
}

func (pe *PipelineExecutor) runTranslation(ctx context.Context, input PipelineInput) (*PipelineResult, error) {
	req, err := pe.manager.Prompts().Build(ctx, PromptTypeTranslation, map[string]string{
		"source_language": "en",
		"target_language": "pt",
		"content":         input.Briefing,
	})
	if err != nil {
		return nil, err
	}

	result, err := pe.manager.Generate(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &PipelineResult{
		Stage:   StageTranslationGen,
		Content: result.Content,
	}, nil
}

func (pe *PipelineExecutor) runReview(ctx context.Context, input PipelineInput) (*PipelineResult, error) {
	req, err := pe.manager.Prompts().Build(ctx, PromptTypeRevision, map[string]string{
		"content":      input.Briefing,
		"feedback":     "Review the content for quality, accuracy, and completeness.",
		"instructions": "Provide a final review report.",
	})
	if err != nil {
		return nil, err
	}

	result, err := pe.manager.Generate(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &PipelineResult{
		Stage:   StageFinalReview,
		Content: result.Content,
	}, nil
}

func joinStrings(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	result := items[0]
	for i := 1; i < len(items); i++ {
		result += sep + items[i]
	}
	return result
}
