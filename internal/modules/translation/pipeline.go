package translation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"nexora/internal/ai"
	"nexora/internal/modules/posts"
	"nexora/internal/modules/publisher"
	"nexora/internal/pkg/database"
)

// The async translation pipeline: translate -> native_review -> seo_review ->
// publish. A rejected stage returns the job to the previous stage; the job
// pauses in JobStatusReview until a human approves or rejects.

func (s *Service) executePipeline(ctx context.Context, siteID, jobID uuid.UUID) {
	p, err := s.pool()
	if err != nil {
		s.log.Error("translation pipeline: database unavailable", "job_id", jobID.String())
		return
	}

	job, err := scanJob(p.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM translation_jobs WHERE id = $1 AND site_id = $2`, jobID, siteID))
	if err != nil {
		s.log.Error("translation pipeline: job load failed", "job_id", jobID.String(), "error", err)
		return
	}
	if job.Status != JobStatusRunning {
		return // cancelled or superseded
	}

	terms, err := s.ListGlossaryTerms(ctx, siteID, job.ProjectID)
	if err != nil {
		s.failPipeline(ctx, p, job, err)
		return
	}

	stages, err := s.loadStages(ctx, p, jobID)
	if err != nil {
		s.failPipeline(ctx, p, job, err)
		return
	}

	var publishedPostID, publicationID *uuid.UUID

	for _, stageType := range StageOrder {
		st := s.findStage(stages, stageType)
		if st == nil || st.Status == StageCompleted {
			continue
		}

		st.Status = StageRunning
		st.Feedback = ""
		if err := s.updateStage(ctx, p, st); err != nil {
			s.failPipeline(ctx, p, job, err)
			return
		}
		job.CurrentStage = &stageType
		_ = s.updateJobStatus(ctx, p, jobID, siteID, JobStatusRunning, &stageType, nil, nil, nil, "")

		status, score, result, feedback, runErr := s.runStage(ctx, job, stageType, terms)

		now := time.Now()
		st.Status = status
		st.Score = score
		st.Result = result
		st.Feedback = feedback
		if status == StageCompleted || status == StageFailed {
			st.CompletedAt = &now
		}
		if err := s.updateStage(ctx, p, st); err != nil {
			s.failPipeline(ctx, p, job, err)
			return
		}

		switch status {
		case StageCompleted:
			if result.PostID != "" {
				if id, err := uuid.Parse(result.PostID); err == nil {
					publishedPostID = &id
				}
			}
			if result.PublicationID != "" {
				if id, err := uuid.Parse(result.PublicationID); err == nil {
					publicationID = &id
				}
			}
			s.fireEvent(ctx, EventTranslationStagePassed, map[string]interface{}{
				"job_id":  jobID.String(),
				"site_id": siteID.String(),
				"stage":   string(stageType),
				"score":   score,
			}, siteID)
			continue

		case StageRejected:
			_ = s.updateJobStatus(ctx, p, jobID, siteID, JobStatusReview, &stageType, nil, nil, nil, "")
			s.fireEvent(ctx, EventTranslationStageRejected, map[string]interface{}{
				"job_id":   jobID.String(),
				"site_id":  siteID.String(),
				"stage":    string(stageType),
				"feedback": feedback,
			}, siteID)
			s.log.Info("translation pipeline: stage rejected, waiting for review",
				"job_id", jobID.String(), "stage", string(stageType))
			return

		case StageFailed:
			_ = s.updateJobStatus(ctx, p, jobID, siteID, JobStatusFailed, &stageType, nil, nil, nil, runErr.Error())
			s.fireEvent(ctx, EventTranslationJobFailed, map[string]interface{}{
				"job_id":  jobID.String(),
				"site_id": siteID.String(),
				"stage":   string(stageType),
				"error":   runErr.Error(),
			}, siteID)
			s.log.Error("translation pipeline: stage failed",
				"job_id", jobID.String(), "stage", string(stageType), "error", runErr)
			return
		}
	}

	// All stages completed: persist the final deterministic TranslationScore.
	finalScore := s.finalScore(ctx, job, terms)
	if err := s.updateJobStatus(ctx, p, jobID, siteID, JobStatusCompleted, nil, finalScore,
		publishedPostID, publicationID, ""); err != nil {
		s.log.Error("translation pipeline: final update failed", "job_id", jobID.String(), "error", err)
		return
	}
	s.fireEvent(ctx, EventTranslationJobCompleted, map[string]interface{}{
		"job_id":      jobID.String(),
		"site_id":     siteID.String(),
		"score":       finalScore,
		"publication": publicationID,
	}, siteID)
	s.log.Info("translation pipeline completed", "job_id", jobID.String())
}

// failPipeline marks the job as failed and stops the pipeline.
func (s *Service) failPipeline(ctx context.Context, p database.Pool, job *TranslationJob, cause error) {
	_ = s.updateJobStatus(ctx, p, job.ID, job.SiteID, JobStatusFailed, job.CurrentStage, nil, nil, nil, cause.Error())
	s.fireEvent(ctx, EventTranslationJobFailed, map[string]interface{}{
		"job_id":  job.ID.String(),
		"site_id": job.SiteID.String(),
		"error":   cause.Error(),
	}, job.SiteID)
	s.log.Error("translation pipeline failed", "job_id", job.ID.String(), "error", cause)
}

// finalScore recomputes the full deterministic TranslationScore from the
// seo_review stage result.
func (s *Service) finalScore(ctx context.Context, job *TranslationJob, terms []GlossaryTerm) *TranslationScore {
	p, err := s.pool()
	if err != nil {
		return nil
	}
	var resultJSON string
	if err := p.QueryRow(ctx,
		`SELECT result FROM translation_stages WHERE translation_job_id = $1 AND stage = 'seo_review' ORDER BY attempt DESC LIMIT 1`,
		job.ID).Scan(&resultJSON); err != nil {
		return nil
	}
	var r StageResult
	if err := json.Unmarshal([]byte(resultJSON), &r); err != nil || r.Content == "" {
		return nil
	}
	_, gRes := ApplyGlossary(r.Content, terms, job.SourceLanguage, job.TargetLanguage)
	_, lRes := Localize(r.Content, job.SourceLanguage, job.TargetLanguage)
	sc := ComputeTranslationScore(ctx, ScoreInput{
		Text:        r.Content,
		Title:       r.Title,
		MetaTitle:   r.MetaTitle,
		MetaDesc:    r.MetaDescription,
		Slug:        r.Slug,
		Keyword:     r.Keyword,
		Language:    job.TargetLanguage,
		GlossaryRes: gRes,
		LocalizeRes: lRes,
		QC:          s.qualityChecker,
	})
	return &sc
}

// runStage executes one pipeline stage and returns its outcome.
func (s *Service) runStage(ctx context.Context, job *TranslationJob, stageType StageType, terms []GlossaryTerm) (StageStatus, *float64, StageResult, string, error) {
	switch stageType {
	case StageTranslate:
		return s.runTranslate(ctx, job, terms)
	case StageNativeReview:
		return s.runNativeReview(ctx, job, terms)
	case StageSEOReview:
		return s.runSEOReview(ctx, job, terms)
	case StagePublish:
		return s.runPublish(ctx, job)
	}
	return StageFailed, nil, StageResult{}, "unknown stage", errors.New("unknown stage " + string(stageType))
}

// loadStageResult reads the latest result JSONB of a stage.
func (s *Service) loadStageResult(ctx context.Context, jobID uuid.UUID, stage StageType) StageResult {
	p, err := s.pool()
	if err != nil {
		return StageResult{}
	}
	var resultJSON string
	if err := p.QueryRow(ctx,
		`SELECT result FROM translation_stages WHERE translation_job_id = $1 AND stage = $2 ORDER BY attempt DESC LIMIT 1`,
		jobID, string(stage)).Scan(&resultJSON); err != nil {
		return StageResult{}
	}
	var r StageResult
	if err := json.Unmarshal([]byte(resultJSON), &r); err != nil {
		return StageResult{}
	}
	return r
}

// --- Stage: Translate ---

func (s *Service) runTranslate(ctx context.Context, job *TranslationJob, terms []GlossaryTerm) (StageStatus, *float64, StageResult, string, error) {
	if s.aiManager == nil {
		return StageFailed, nil, StageResult{}, "", ErrAIManagerRequired
	}
	from, to := job.SourceLanguage, job.TargetLanguage

	// Reuse the cached research (fact base + briefing) so both language
	// versions share the same verified facts (research runs only once).
	researchCtx := s.researchContext(ctx, job)

	contentReq, err := s.aiManager.Prompts().Build(ctx, ai.PromptTypeTranslation, map[string]string{
		"source_language": langName(from),
		"target_language": langName(to),
		"content":         job.Content + researchCtx,
	})
	if err != nil {
		contentReq = &ai.CompletionRequest{
			Prompt: fmt.Sprintf("Translate the following content from %s to %s. Preserve formatting, tone, and technical terms:\n\n%s",
				langName(from), langName(to), job.Content+researchCtx),
		}
	}
	resp, err := s.aiManager.Generate(ctx, *contentReq)
	if err != nil {
		return StageFailed, nil, StageResult{}, "", fmt.Errorf("translation generation failed: %w", err)
	}
	translated := strings.TrimSpace(resp.Content)
	if translated == "" {
		return StageFailed, nil, StageResult{}, "", errors.New("AI returned empty translation")
	}

	// Translate the title separately (keeps output simple and parseable).
	title := job.Title
	titleReq := ai.CompletionRequest{
		Prompt: fmt.Sprintf("Translate the following article title from %s to %s. Return only the translated title, with no quotes or extra punctuation:\n\n%s",
			langName(from), langName(to), job.Title),
	}
	if tr, err := s.aiManager.Generate(ctx, titleReq); err == nil && strings.TrimSpace(tr.Content) != "" {
		title = strings.Trim(strings.TrimSpace(tr.Content), "\"'")
	}

	// Deterministic glossary + cultural localization pass over the translation.
	text, gRes, lRes := s.reapplyDeterministic(translated, terms, from, to)

	result := StageResult{
		Title:             title,
		Content:           text,
		GlossaryApplied:   gRes.Applied,
		LocalizationCount: lRes.Applied,
	}
	return StageCompleted, nil, result, "", nil
}

// --- Stage: Native Review ---

func (s *Service) runNativeReview(ctx context.Context, job *TranslationJob, terms []GlossaryTerm) (StageStatus, *float64, StageResult, string, error) {
	from, to := job.SourceLanguage, job.TargetLanguage
	translateResult := s.loadStageResult(ctx, job.ID, StageTranslate)
	text := translateResult.Content
	if text == "" {
		text = job.Content
	}

	text, gRes, lRes := s.reapplyDeterministic(text, terms, from, to)

	var score TranslationScore
	for attempt := 0; attempt <= MaxRewriteAttempts; attempt++ {
		score = ComputeTranslationScore(ctx, ScoreInput{
			Text:        text,
			Title:       translateResult.Title,
			Language:    to,
			GlossaryRes: gRes,
			LocalizeRes: lRes,
			QC:          s.qualityChecker,
		})
		if score.Overall >= MinNativeReviewScore || attempt == MaxRewriteAttempts {
			break
		}
		idx, section := worstSection(text, to)
		if section == "" {
			break
		}
		rewritten, err := s.rewriteSection(ctx, section, to, score)
		if err != nil || strings.TrimSpace(rewritten) == "" {
			break
		}
		paras := splitParagraphs(text)
		if idx >= len(paras) {
			break
		}
		paras[idx] = strings.TrimSpace(rewritten)
		text = strings.Join(paras, "\n\n")
		// Keep glossary + localization consistent after the rewrite.
		text, gRes, lRes = s.reapplyDeterministic(text, terms, from, to)
	}

	// Final deterministic pass + recompute (idempotent).
	score = ComputeTranslationScore(ctx, ScoreInput{
		Text:        text,
		Title:       translateResult.Title,
		Language:    to,
		GlossaryRes: gRes,
		LocalizeRes: lRes,
		QC:          s.qualityChecker,
	})
	overall := score.Overall

	result := translateResult
	result.Content = text
	result.GlossaryApplied = gRes.Applied
	result.LocalizationCount = lRes.Applied

	if overall < MinNativeReviewScore {
		return StageRejected, &overall, result,
			fmt.Sprintf("native review score %.1f below minimum %.0f (grammar %.1f, fluency %.1f, naturalness %.1f)",
				overall, MinNativeReviewScore, score.Grammar, score.Fluency, score.Naturalness), nil
	}
	return StageCompleted, &overall, result, "", nil
}

// reapplyDeterministic runs glossary + localization over a text.
// researchContext returns the cached research briefing/fact base for the
// source topic so both language versions stay fact-consistent. It never
// triggers a new research (cache-only lookup).
func (s *Service) researchContext(ctx context.Context, job *TranslationJob) string {
	if s.researchSvc == nil || job == nil || job.SiteID == uuid.Nil {
		return ""
	}
	summary, err := s.researchSvc.GetCachedSummary(ctx, job.SiteID, job.Title, job.TargetLanguage)
	if err != nil || summary == nil {
		return ""
	}
	var b strings.Builder
	if summary.Briefing != "" {
		b.WriteString("\n\nReference research (same facts must appear in the translation):\n")
		b.WriteString(summary.Briefing)
	}
	if len(summary.Facts) > 0 {
		b.WriteString("\n\nVerified facts:\n")
		for _, f := range summary.Facts {
			fmt.Fprintf(&b, "- %s: %s (%s)\n", f.Entity, f.Value, f.Type)
		}
	}
	return b.String()
}

func (s *Service) reapplyDeterministic(text string, terms []GlossaryTerm, from, to string) (string, GlossaryApplyResult, LocalizationResult) {	gText, gRes := ApplyGlossary(text, terms, from, to)
	lText, lRes := Localize(gText, from, to)
	return lText, gRes, lRes
}

// worstSection returns the index and text of the paragraph with the most
// literal-translation markers, or (-1, "") when there is none.
func worstSection(text, language string) (int, string) {
	paras := splitParagraphs(text)
	worstIdx := -1
	worstCount := 0
	for i, para := range paras {
		if para == "" {
			continue
		}
		count := len(literalMarkerRe.FindAllString(para, -1)) + countRepeatedWords(para)
		if count > worstCount {
			worstCount = count
			worstIdx = i
		}
	}
	if worstIdx < 0 {
		return -1, ""
	}
	return worstIdx, paras[worstIdx]
}

func splitParagraphs(text string) []string {
	parts := strings.Split(text, "\n\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

// rewriteSection asks the AI to rewrite one section more naturally.
func (s *Service) rewriteSection(ctx context.Context, section, to string, sc TranslationScore) (string, error) {
	if s.aiManager == nil {
		return "", ErrAIManagerRequired
	}
	feedback := fmt.Sprintf("The section reads as a literal translation. Rewrite it so it reads as if written natively in %s, preserving meaning. Current quality: grammar %.1f, fluency %.1f, naturalness %.1f.",
		langName(to), sc.Grammar, sc.Fluency, sc.Naturalness)
	req, err := s.aiManager.Prompts().Build(ctx, ai.PromptTypeRevision, map[string]string{
		"content":      section,
		"feedback":     feedback,
		"instructions": "Return only the rewritten section, preserving any markdown formatting.",
	})
	if err != nil {
		req = &ai.CompletionRequest{
			Prompt: feedback + "\n\nSection:\n" + section + "\n\nReturn only the rewritten section.",
		}
	}
	resp, err := s.aiManager.Generate(ctx, *req)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Content), nil
}

// --- Stage: SEO Review ---

func (s *Service) runSEOReview(ctx context.Context, job *TranslationJob, terms []GlossaryTerm) (StageStatus, *float64, StageResult, string, error) {
	from, to := job.SourceLanguage, job.TargetLanguage
	nativeResult := s.loadStageResult(ctx, job.ID, StageNativeReview)
	if nativeResult.Content == "" {
		nativeResult.Content = job.Content
	}

	seo := s.generateSEO(ctx, job, nativeResult)

	content := nativeResult.Content
	if seo.Title == "" {
		seo.Title = nativeResult.Title
	}
	if seo.Title == "" {
		seo.Title = job.Title
	}
	seo.Slug = GenerateSlug(seo.Slug, to)
	if seo.Slug == "" {
		seo.Slug = GenerateSlug(seo.Title, to)
	}
	if seo.PrimaryKeyword == "" {
		seo.PrimaryKeyword = DeriveKeyword(seo.Title, to)
	}
	if seo.MetaDescription == "" {
		seo.MetaDescription = truncateTo(content, 155)
	}

	_, gRes := ApplyGlossary(content, terms, from, to)
	_, lRes := Localize(content, from, to)

	sc := ComputeTranslationScore(ctx, ScoreInput{
		Text:        content,
		Title:       seo.Title,
		MetaTitle:   seo.MetaTitle,
		MetaDesc:    seo.MetaDescription,
		Slug:        seo.Slug,
		Keyword:     seo.PrimaryKeyword,
		Language:    to,
		GlossaryRes: gRes,
		LocalizeRes: lRes,
		QC:          s.qualityChecker,
	})
	overall := sc.Overall

	result := nativeResult
	result.Title = seo.Title
	result.MetaTitle = seo.MetaTitle
	result.MetaDescription = seo.MetaDescription
	result.Slug = seo.Slug
	result.Keyword = seo.PrimaryKeyword
	result.SecondaryKeywords = seo.SecondaryKeywords
	result.Content = content

	if overall < MinSEOScore {
		return StageRejected, &overall, result,
			fmt.Sprintf("international SEO score %.1f below minimum %.0f (seo %.1f, overall %.1f)",
				overall, MinSEOScore, sc.SEO, overall), nil
	}
	return StageCompleted, &overall, result, "", nil
}

type seoMetadata struct {
	Title             string   `json:"title"`
	MetaTitle         string   `json:"meta_title"`
	MetaDescription   string   `json:"meta_description"`
	Slug              string   `json:"slug"`
	PrimaryKeyword    string   `json:"primary_keyword"`
	SecondaryKeywords []string `json:"secondary_keywords"`
}

// generateSEO asks the AI for target-language SEO metadata. Never reuses
// source-language SEO data; falls back to deterministic generation.
func (s *Service) generateSEO(ctx context.Context, job *TranslationJob, nativeResult StageResult) seoMetadata {
	if s.aiManager == nil {
		return s.fallbackSEO(job, nativeResult)
	}
	prompt := fmt.Sprintf(`You are an SEO specialist writing for a %s-speaking audience. Generate original search-engine-optimized metadata for the article below, which was translated from %s.

Do NOT reuse any SEO data, keywords, titles, or slugs from the source language. Everything must be written natively in %s.

Article title: %s

Content:
%s

Return ONLY a JSON object with exactly these fields: "title" (optimized H1), "meta_title", "meta_description" (140-160 characters), "slug" (URL-safe), "primary_keyword", "secondary_keywords" (array of 2-4 strings).`,
		langName(job.TargetLanguage), langName(job.SourceLanguage), langName(job.TargetLanguage),
		nativeResult.Title, nativeResult.Content)

	resp, err := s.aiManager.Generate(ctx, ai.CompletionRequest{Prompt: prompt})
	if err != nil || strings.TrimSpace(resp.Content) == "" {
		return s.fallbackSEO(job, nativeResult)
	}

	var meta seoMetadata
	if err := parseJSONObject(resp.Content, &meta); err != nil {
		return s.fallbackSEO(job, nativeResult)
	}
	if meta.Title == "" && meta.PrimaryKeyword == "" {
		return s.fallbackSEO(job, nativeResult)
	}
	return meta
}

func (s *Service) fallbackSEO(job *TranslationJob, nativeResult StageResult) seoMetadata {
	title := nativeResult.Title
	if title == "" {
		title = job.Title
	}
	return seoMetadata{
		Title:           title,
		MetaTitle:       title,
		MetaDescription: truncateTo(nativeResult.Content, 155),
		Slug:            GenerateSlug(title, job.TargetLanguage),
		PrimaryKeyword:  DeriveKeyword(title, job.TargetLanguage),
	}
}

func parseJSONObject(text string, out interface{}) error {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return errors.New("no JSON object found")
	}
	return json.Unmarshal([]byte(text[start:end+1]), out)
}

func truncateTo(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n])
}

// --- Stage: Publish ---

func (s *Service) runPublish(ctx context.Context, job *TranslationJob) (StageStatus, *float64, StageResult, string, error) {
	if s.publisherSvc == nil {
		return StageFailed, nil, StageResult{}, "", errors.New("publisher service not configured")
	}
	seoResult := s.loadStageResult(ctx, job.ID, StageSEOReview)

	title := seoResult.Title
	if title == "" {
		title = job.Title
	}
	content := seoResult.Content
	if content == "" {
		content = job.Content
	}
	slug := seoResult.Slug
	if slug == "" {
		slug = GenerateSlug(title, job.TargetLanguage)
	}
	excerpt := seoResult.MetaDescription

	result := seoResult

	if s.postsSvc != nil && job.CreatedBy != nil && *job.CreatedBy != uuid.Nil {
		// Each language gets its own editable post on the target site.
		post, err := s.postsSvc.Create(ctx, job.TargetSiteID, *job.CreatedBy, posts.CreatePostRequest{
			Title:    title,
			Content:  textToBlocks(content),
			Excerpt:  excerpt,
			Status:   posts.PostStatusDraft,
			PostMeta: map[string]interface{}{"language": job.TargetLanguage, "translated_from": job.SourceLanguage},
		})
		if err != nil {
			return StageFailed, nil, StageResult{}, "", fmt.Errorf("failed to create translated post: %w", err)
		}
		postID := post.ID
		result.PostID = postID.String()

		pub, err := s.publisherSvc.PublishArticle(ctx, job.TargetSiteID, *job.CreatedBy, publisher.PublishRequest{
			PostID:          &postID,
			Title:           title,
			Content:         content,
			Excerpt:         excerpt,
			Slug:            slug,
			Language:        job.TargetLanguage,
			MetaTitle:       seoResult.MetaTitle,
			MetaDescription: seoResult.MetaDescription,
			Source:          "translation",
		})
		if err != nil {
			return StageFailed, nil, result, "", fmt.Errorf("failed to publish translated article: %w", err)
		}
		result.PublicationID = pub.Publication.ID.String()
	} else {
		// No valid user context: publish without a posts row.
		pub, err := s.publisherSvc.PublishGeneratedArticle(ctx, publisher.PublishGeneratedRequest{
			SiteID:          job.TargetSiteID,
			Title:           title,
			Content:         content,
			Excerpt:         excerpt,
			Slug:            slug,
			Language:        job.TargetLanguage,
			MetaTitle:       seoResult.MetaTitle,
			MetaDescription: seoResult.MetaDescription,
			Source:          "translation",
		})
		if err != nil {
			return StageFailed, nil, result, "", fmt.Errorf("failed to publish translated article: %w", err)
		}
		result.PublicationID = pub.ID.String()
	}

	s.fireEvent(ctx, EventTranslationPublished, map[string]interface{}{
		"job_id":      job.ID.String(),
		"site_id":     job.SiteID.String(),
		"target_site": job.TargetSiteID.String(),
		"publication": result.PublicationID,
		"post_id":     result.PostID,
		"language":    job.TargetLanguage,
		"slug":        slug,
	}, job.SiteID)

	return StageCompleted, nil, result, "", nil
}
