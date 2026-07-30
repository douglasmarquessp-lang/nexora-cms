package ai

import (
	"context"
	"fmt"
	"time"
)

type PipelineStage int

const (
	StageResearchGen PipelineStage = iota
	StageBriefingGen
	StageOutlineGen
	StageDraftGen
	StageSEOGen
	StageQualityCheck
	StageTranslationGen
	StageFinalReview
	StageTopicGen
	StageFactCheck
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
	StageTopicGen:       "topic",
	StageFactCheck:      "fact_check",
}

type PipelineInput struct {
	Title       string            `json:"title"`
	ContentType string            `json:"content_type"`
	Language    string            `json:"language"`
	Topic       string            `json:"topic,omitempty"`
	Briefing    string            `json:"briefing,omitempty"`
	Outline     string            `json:"outline,omitempty"`
	Content     string            `json:"content,omitempty"`
	Style       map[string]string `json:"style,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	WordCount   int               `json:"word_count,omitempty"`
	Tone        string            `json:"tone,omitempty"`
	Audience    string            `json:"audience,omitempty"`
	References  []string          `json:"references,omitempty"`
	Entities    []string          `json:"entities,omitempty"`
}

type PipelineResult struct {
	Stage             PipelineStage      `json:"stage"`
	Content           string             `json:"content"`
	Error             error              `json:"error,omitempty"`
	Duration          time.Duration      `json:"duration,omitempty"`
	GroundingMetadata *GroundingMetadata `json:"grounding_metadata,omitempty"`
}

type PipelineExecutor struct {
	manager *Manager
}

func NewPipelineExecutor(manager *Manager) *PipelineExecutor {
	return &PipelineExecutor{manager: manager}
}

func (pe *PipelineExecutor) ExecuteStage(ctx context.Context, stage PipelineStage, input PipelineInput) (*PipelineResult, error) {
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
	case StageTopicGen:
		return pe.runTopic(ctx, input)
	case StageFactCheck:
		return pe.runFactCheck(ctx, input)
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
		StageTopicGen,
		StageFactCheck,
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

	// Enable grounding if any registered provider supports it
	if pe.manager.Registry().HasCapability(CapGrounding) {
		req.Grounding = &GroundingConfig{
			Enabled:    true,
			MaxSources: 10,
		}
	}

	result, err := pe.manager.Generate(ctx, *req)
	if err != nil {
		return nil, err
	}

	pipelineResult := &PipelineResult{
		Stage:   StageResearchGen,
		Content: result.Content,
	}

	// Carry grounding metadata forward so callers can persist sources
	if result.GroundingMetadata != nil {
		pipelineResult.GroundingMetadata = result.GroundingMetadata
	}

	return pipelineResult, nil
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
		"title":      input.Title,
		"briefing":   input.Briefing,
		"keywords":   joinStrings(input.Keywords, ", "),
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

func sourceText(input PipelineInput) string {
	if input.Content != "" {
		return input.Content
	}
	return input.Briefing
}

func (pe *PipelineExecutor) runSEO(ctx context.Context, input PipelineInput) (*PipelineResult, error) {
	req, err := pe.manager.Prompts().Build(ctx, PromptTypeSEO, map[string]string{
		"content":  sourceText(input),
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
	text := sourceText(input)

	// Use detailed deterministic quality checks
	grammar, _ := pe.manager.Quality().CheckGrammarDetails(ctx, text, input.Language)
	seo, _ := pe.manager.Quality().AssessSEO(ctx, text, input.Keywords)
	readability, _ := pe.manager.Quality().ScoreReadabilityDetailed(ctx, text, input.Language)
	entities, _ := pe.manager.Quality().ScoreEntityCoverage(ctx, text, input.Entities)
	duplicates, _ := pe.manager.Quality().CheckDuplicateBlocks(ctx, text, 10)
	structure, _ := pe.manager.Quality().ValidateStructure(ctx, text)

	// Run hallucination check if references or grounding available
	var factCheck *FactCheckReport
	if len(input.References) > 0 {
		// Check if research stage provided grounding metadata
		var gm *GroundingMetadata
		// Grounding metadata isn't passed between stages in PipelineInput currently,
		// so try reference-based fact checking
		factCheck, _ = pe.manager.Quality().CheckHallucinationWithGrounding(ctx, text, input.References, gm)
	}

	result := "Quality Check Results:\n"
	result += fmt.Sprintf("- Grammar: %.1f/100 (passed: %v, %d issues)\n", grammar.OverallScore, grammar.Passed, len(grammar.Issues))
	result += fmt.Sprintf("- SEO: %.1f/100 (passed: %v)\n", seo.OverallScore, seo.Passed)
	result += fmt.Sprintf("- Readability: %.1f/100 (FRE: %.0f, grade: %.1f)\n", readability.OverallScore, readability.FleschReadingEase, readability.FleschKincaidGrade)
	result += fmt.Sprintf("- Entity Coverage: %.1f/100 (passed: %v)\n", entities.Score, entities.Passed)
	result += fmt.Sprintf("- Duplicate Blocks: %d\n", len(duplicates))
	result += fmt.Sprintf("- Structure: %.1f/100 (passed: %v)\n", structure.OverallScore, structure.Passed)
	if factCheck != nil {
		result += fmt.Sprintf("- Fact Check: %.1f/100 (supported: %d, unsupported: %d)\n", factCheck.OverallScore, factCheck.Supported, factCheck.Unsupported)
	}

	allPassed := grammar.Passed && seo.Passed && readability.Passed && entities.Passed && structure.Passed
	if factCheck != nil {
		allPassed = allPassed && factCheck.Passed
	}
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
		"content":         sourceText(input),
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
		"content":      sourceText(input),
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

func (pe *PipelineExecutor) runTopic(ctx context.Context, input PipelineInput) (*PipelineResult, error) {
	req, err := pe.manager.Prompts().Build(ctx, PromptTypeTopic, map[string]string{
		"topic":       input.Topic,
		"content":     sourceText(input),
		"content_type": input.ContentType,
		"language":    input.Language,
	})
	if err != nil {
		return nil, err
	}

	result, err := pe.manager.Generate(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &PipelineResult{
		Stage:   StageTopicGen,
		Content: result.Content,
	}, nil
}

func (pe *PipelineExecutor) runFactCheck(ctx context.Context, input PipelineInput) (*PipelineResult, error) {
	req, err := pe.manager.Prompts().Build(ctx, PromptTypeFactCheck, map[string]string{
		"content":    sourceText(input),
		"references": joinStrings(input.References, "\n"),
	})
	if err != nil {
		return nil, err
	}

	result, err := pe.manager.Generate(ctx, *req)
	if err != nil {
		return nil, err
	}

	return &PipelineResult{
		Stage:   StageFactCheck,
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
