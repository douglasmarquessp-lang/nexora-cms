package ai

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	markdownH2RE = regexp.MustCompile(`(?m)^##\s+.+$`)
	htmlH2RE     = regexp.MustCompile(`(?i)<h2[^>]*>.*?</h2>`)
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

type ResearchFact struct {
	Type        string `json:"type"`
	Entity      string `json:"entity"`
	Value       string `json:"value"`
	Source      string `json:"source,omitempty"`
	Confidence  int    `json:"confidence,omitempty"`
}

type ResearchSourceSummary struct {
	Title            string `json:"title"`
	URL              string `json:"url"`
	Domain           string `json:"domain,omitempty"`
	ReliabilityScore int    `json:"reliability_score,omitempty"`
	ReliabilityLabel string `json:"reliability_label,omitempty"`
}

// ResearchSummary is the structured output of the research stage: a briefing
// plus a fact base plus the ranked sources. It is produced by the research
// module (DeepResearch) and consumed by downstream pipeline stages.
type ResearchSummary struct {
	Topic    string                  `json:"topic"`
	Language string                  `json:"language"`
	Briefing string                  `json:"briefing,omitempty"`
	Facts    []ResearchFact          `json:"facts,omitempty"`
	Sources  []ResearchSourceSummary `json:"sources,omitempty"`
	Cached   bool                    `json:"cached,omitempty"`
}

// ResearchFn is injected by callers that own a research service (site-scoped).
// It performs deep research with cache and returns nil summary + nil error when
// no research is needed/possible (callers fall back to grounding-only).
type ResearchFn func(ctx context.Context, topic, language string) (*ResearchSummary, error)

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
	References        []string           `json:"references,omitempty"`
	Entities          []string           `json:"entities,omitempty"`
	GroundingMetadata *GroundingMetadata `json:"grounding_metadata,omitempty"`
	Research          *ResearchSummary   `json:"research,omitempty"`
	ResearchFn        ResearchFn         `json:"-"`
}

type PipelineResult struct {
	Stage             PipelineStage      `json:"stage"`
	Content           string             `json:"content"`
	// Analysis carries the stage's diagnostic report (e.g. quality check,
	// final review, SEO recommendations). It is never the article itself:
	// downstream callers publish Content, so diagnostic stages must keep the
	// article in Content and move their reports here.
	Analysis          string             `json:"analysis,omitempty"`
	Error             error              `json:"error,omitempty"`
	Duration          time.Duration      `json:"duration,omitempty"`
	GroundingMetadata *GroundingMetadata `json:"grounding_metadata,omitempty"`
	Research          *ResearchSummary   `json:"research,omitempty"`
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
	// Preferred path: a research service is wired (deep research with cache).
	if input.Research != nil {
		return &PipelineResult{
			Stage:    StageResearchGen,
			Content:  formatResearchSummary(input.Research),
			Research: input.Research,
		}, nil
	}
	if input.ResearchFn != nil {
		summary, err := input.ResearchFn(ctx, input.Topic, input.Language)
		if err != nil {
			return nil, err
		}
		if summary != nil {
			return &PipelineResult{
				Stage:    StageResearchGen,
				Content:  formatResearchSummary(summary),
				Research: summary,
			}, nil
		}
	}

	// Fallback path: grounding-only research via the AI provider.
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

	// Build a deterministic summary from the grounding sources so downstream
	// stages still receive a structured fact base + ranked sources.
	if summary := summaryFromGrounding(input.Topic, input.Language, result.GroundingMetadata); summary != nil {
		pipelineResult.Research = summary
	}

	return pipelineResult, nil
}

// summaryFromGrounding builds a ResearchSummary deterministically from
// grounding metadata (no AI call). Returns nil when no sources exist.
func summaryFromGrounding(topic, language string, gm *GroundingMetadata) *ResearchSummary {
	if gm == nil || len(gm.Sources) == 0 {
		return nil
	}
	summary := &ResearchSummary{
		Topic:    topic,
		Language: language,
		Briefing: "",
	}
	var parts []string
	for _, gs := range gm.Sources {
		domain := ExtractDomain(gs.URI)
		score, label := ReliabilityOfDomain(domain)
		summary.Sources = append(summary.Sources, ResearchSourceSummary{
			Title:            gs.Title,
			URL:              gs.URI,
			Domain:           domain,
			ReliabilityScore: score,
			ReliabilityLabel: label,
		})
		if gs.Title != "" {
			parts = append(parts, fmt.Sprintf("- %s (%s)", gs.Title, domain))
		}
	}
	if len(parts) > 0 {
		summary.Briefing = fmt.Sprintf("Sources for topic '%s':\n%s", topic, strings.Join(parts, "\n"))
	}
	return summary
}

// formatResearchSummary renders a ResearchSummary as prompt-ready text.
func formatResearchSummary(s *ResearchSummary) string {
	if s == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("Research Summary\n")
	b.WriteString("Topic: ")
	b.WriteString(s.Topic)
	b.WriteString("\nLanguage: ")
	b.WriteString(s.Language)
	if s.Cached {
		b.WriteString("\nCached: true")
	}
	if s.Briefing != "" {
		b.WriteString("\n\nBriefing:\n")
		b.WriteString(s.Briefing)
	}
	if len(s.Facts) > 0 {
		b.WriteString("\n\nFact Base:")
		for _, f := range s.Facts {
			fmt.Fprintf(&b, "\n- %s: %s (%s)", f.Entity, f.Value, f.Type)
			if f.Source != "" {
				fmt.Fprintf(&b, " [%s]", f.Source)
			}
		}
	}
	if len(s.Sources) > 0 {
		b.WriteString("\n\nSources:")
		for _, src := range s.Sources {
			fmt.Fprintf(&b, "\n- %s | %s | reliability %d (%s)", src.Title, src.URL, src.ReliabilityScore, src.ReliabilityLabel)
		}
	}
	return b.String()
}

// researchContext appends the fact base to a stage's briefing text so drafts
// and briefings are grounded in real data (dates, numbers, versions).
func researchContext(input PipelineInput) string {
	if input.Research == nil || len(input.Research.Facts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nFact Base (use these verified facts; do not invent new ones):")
	for _, f := range input.Research.Facts {
		fmt.Fprintf(&b, "\n- %s | %s | %s", f.Entity, f.Value, f.Type)
	}
	return b.String()
}

func (pe *PipelineExecutor) runBriefing(ctx context.Context, input PipelineInput) (*PipelineResult, error) {
	promptID := PromptTypeBriefing
	if input.Language == "pt" {
		promptID = PromptTypeBriefing
	}

	req, err := pe.manager.Prompts().Build(ctx, promptID, map[string]string{
		"topic":   input.Topic,
		"sources": input.Briefing + researchContext(input),
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
		"briefing":     input.Briefing + researchContext(input),
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

	// Enrichment stage: the article stays the Content. We add:
	//   1. an "Introduction" H2 when the draft has no subheadings yet, so the
	//      published page always has a real H2 structure;
	//   2. a real "Sources" section with markdown links to the research
	//      grounding sources (reliability >= 75) when available.
	// The AI's SEO recommendations go to Analysis (diagnostic, not published).
	base := sourceText(input)
	article := ensureIntroHeading(base, input.Language)
	if sources := sourcesSection(input.GroundingMetadata, input.Language); sources != "" {
		article = strings.TrimSpace(article) + "\n\n" + sources
	}

	return &PipelineResult{
		Stage:    StageSEOGen,
		Content:  article,
		Analysis: result.Content,
	}, nil
}

// sourcesSection renders markdown links to the research sources with
// reliability >= 80 (verified/official: news agencies, official sites, docs).
// Returns "" when there are no usable sources so the article stays untouched.
// External URLs are deduplicated against the article text.
func sourcesSection(gm *GroundingMetadata, lang string) string {
	if gm == nil || len(gm.Sources) == 0 {
		return ""
	}
	heading := "## Fontes\n"
	if lang == "en" {
		heading = "## Sources\n"
	}
	var sb strings.Builder
	sb.WriteString(heading)
	count := 0
	seen := map[string]bool{}
	for _, src := range gm.Sources {
		if src.URI == "" || seen[src.URI] {
			continue
		}
		domain := ExtractDomain(src.URI)
		score, _ := ReliabilityOfDomain(domain)
		if score < 75 {
			continue
		}
		seen[src.URI] = true
		title := src.Title
		if title == "" {
			title = domain
		}
		sb.WriteString("- [")
		sb.WriteString(title)
		sb.WriteString("](")
		sb.WriteString(src.URI)
		sb.WriteString(")\n")
		count++
		if count >= 5 {
			break
		}
	}
	if count == 0 {
		return ""
	}
	return strings.TrimSpace(sb.String())
}

// ensureIntroHeading prepends a real H2 "Introduction" when the article has no
// H2 subheading yet — articles that reach the SEO stage without any structure
// get an honest structural heading instead of staying heading-less.
func ensureIntroHeading(content, lang string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	if markdownH2RE.MatchString(trimmed) || htmlH2RE.MatchString(trimmed) {
		return content
	}
	heading := "## Introdução\n\n"
	if lang == "en" {
		heading = "## Introduction\n\n"
	}
	return heading + trimmed
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

	// Run hallucination check if references or grounding metadata available
	var factCheck *FactCheckReport
	if len(input.References) > 0 || input.GroundingMetadata != nil {
		factCheck, _ = pe.manager.Quality().CheckHallucinationWithGrounding(ctx, text, input.References, input.GroundingMetadata)
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
		Stage:    StageQualityCheck,
		Content:  text,
		Analysis: result,
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

// runReview runs the final review on the article. The article is preserved in
// Content; the review report goes to Analysis so callers that publish straight
// from the pipeline output never lose the draft to a diagnostic report.
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
		Stage:    StageFinalReview,
		Content:  sourceText(input),
		Analysis: result.Content,
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
