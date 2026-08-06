package editorial

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"

	"nexora/internal/modules/seoengine"
	"nexora/internal/pkg/database"
)

// LinkSuggestor ranks internal/external link candidates for the review
// screen. Implemented by the seoengine service (wired in cmd/api/main.go).
// A nil suggestor degrades gracefully to empty link lists.
type LinkSuggestor interface {
	SelectInternalLinks(ctx context.Context, siteID uuid.UUID, sourcePostID *uuid.UUID, sourceTitle, sourceContent, sourceKeyword, sourceCategory string, minScore, maxLinks int) ([]seoengine.InternalLinkCandidate, error)
	SelectExternalLinks(ctx context.Context, siteID uuid.UUID, topic string, minReliability, maxLinks int) ([]seoengine.ExternalLinkCandidate, error)
}

func (s *Service) SetLinkSuggestor(sg LinkSuggestor) {
	s.linkSuggestor = sg
}

// pipelineUnionSQL is a read-only UNION of every table the editorial board
// tracks. Every branch filters by site_id ($1) and emits the same column
// shape; the outer query orders by recency and caps the result ($2).
// Post rows appear in exactly one stage: published > scheduled > approval
// (latest review approved) > review (latest review needs_review) > eeat
// (any review) > seo (audited, never reviewed). Engine tables never overlap
// with posts.
const pipelineUnionSQL = `
SELECT u.id, u.title, u.slug, u.stage, u.engine, u.language,
       u.category_id, u.author_id, u.seo_score, u.eeat_score,
       u.status, u.scheduled_at, u.updated_at
FROM (
  -- idea: briefs without an outline yet
  SELECT b.id, b.topic AS title, '' AS slug, 'idea' AS stage, 'editorial_briefs' AS engine,
         b.language, NULL::uuid AS category_id, NULL::uuid AS author_id,
         NULL::numeric AS seo_score, NULL::numeric AS eeat_score,
         b.status, NULL::timestamptz AS scheduled_at, b.updated_at
  FROM editorial_briefs b
  WHERE b.site_id = $1 AND b.outline = '[]'::jsonb

  UNION ALL
  -- research: active research jobs
  SELECT r.id, r.topic AS title, '' AS slug, 'research' AS stage, 'research_jobs' AS engine,
         r.language, NULL::uuid, NULL::uuid, NULL::numeric, NULL::numeric,
         r.status, NULL::timestamptz, r.updated_at
  FROM research_jobs r
  WHERE r.site_id = $1 AND r.status NOT IN ('completed','failed')

  UNION ALL
  -- outline: briefs that already carry an outline
  SELECT b.id, b.topic AS title, '' AS slug, 'outline' AS stage, 'editorial_briefs' AS engine,
         b.language, NULL::uuid, NULL::uuid, NULL::numeric, NULL::numeric,
         b.status, NULL::timestamptz, b.updated_at
  FROM editorial_briefs b
  WHERE b.site_id = $1 AND b.outline <> '[]'::jsonb

  UNION ALL
  -- writing: every generation engine's active jobs
  SELECT id, COALESCE(NULLIF(title, ''), topic, '') AS title, '' AS slug, 'writing' AS stage,
         'autocontent_jobs' AS engine, language, NULL::uuid, NULL::uuid,
         NULL::numeric, NULL::numeric, status, NULL::timestamptz, updated_at
  FROM autocontent_jobs
  WHERE site_id = $1 AND status NOT IN ('completed','failed','cancelled')

  UNION ALL
  SELECT gj.id, COALESCE(aj.headline, '') AS title, '' AS slug, 'writing' AS stage,
         'generation_jobs' AS engine, gj.language, NULL::uuid, NULL::uuid,
         NULL::numeric, NULL::numeric, gj.status, NULL::timestamptz, gj.updated_at
  FROM generation_jobs gj
  LEFT JOIN article_jobs aj ON aj.id = gj.article_job_id
  WHERE gj.site_id = $1 AND gj.status NOT IN ('completed','failed','cancelled')

  UNION ALL
  SELECT id, COALESCE(NULLIF(title, ''), topic, '') AS title, '' AS slug, 'writing' AS stage,
         'article_pipeline_jobs' AS engine, language, NULL::uuid, NULL::uuid,
         NULL::numeric, NULL::numeric, status, NULL::timestamptz, updated_at
  FROM article_pipeline_jobs
  WHERE site_id = $1 AND status NOT IN ('completed','failed','cancelled')

  UNION ALL
  SELECT id, title, '' AS slug, 'writing' AS stage,
         'workflow_jobs' AS engine, language, NULL::uuid, NULL::uuid,
         NULL::numeric, NULL::numeric, status, NULL::timestamptz, updated_at
  FROM workflow_jobs
  WHERE site_id = $1 AND status NOT IN ('completed','failed','cancelled')

  UNION ALL
  -- translation: in-flight translation jobs
  SELECT id, title, '' AS slug, 'translation' AS stage,
         'translation_jobs' AS engine, target_language, NULL::uuid, NULL::uuid,
         NULL::numeric, NULL::numeric, status, NULL::timestamptz, updated_at
  FROM translation_jobs
  WHERE site_id = $1 AND status IN ('pending','running','waiting_review')

  UNION ALL
  -- seo: audited draft posts that never received an editorial review
  SELECT p.id, p.title, p.slug, 'seo' AS stage, 'posts' AS engine,
         COALESCE(p.post_meta->>'language', 'pt') AS language,
         (SELECT c.id FROM post_categories pc JOIN categories c ON c.id = pc.category_id
          WHERE pc.post_id = p.id LIMIT 1) AS category_id,
         p.author_id, p.seo_score, NULL::numeric AS eeat_score,
         p.status, p.scheduled_at, p.updated_at
  FROM posts p
  WHERE p.site_id = $1 AND p.status = 'draft' AND p.deleted_at IS NULL
    AND p.seo_analyzed_at IS NOT NULL
    AND NOT EXISTS (SELECT 1 FROM editorial_reviews r WHERE r.article_id = p.id)

  UNION ALL
  -- eeat: draft posts with any editorial review (context column; review and
  -- approval columns are its actionable subsets)
  SELECT p.id, p.title, p.slug, 'eeat' AS stage, 'posts' AS engine,
         COALESCE(p.post_meta->>'language', 'pt'),
         (SELECT c.id FROM post_categories pc JOIN categories c ON c.id = pc.category_id
          WHERE pc.post_id = p.id LIMIT 1),
         p.author_id, p.seo_score, e.eeat_score,
         e.decision AS status, p.scheduled_at, p.updated_at
  FROM posts p
  JOIN LATERAL (
    SELECT r.id, r.eeat_score, r.decision
    FROM editorial_reviews r
    WHERE r.article_id = p.id AND r.site_id = p.site_id
    ORDER BY r.created_at DESC LIMIT 1
  ) e ON true
  WHERE p.site_id = $1 AND p.status = 'draft' AND p.deleted_at IS NULL

  UNION ALL
  -- review: latest review flagged the article
  SELECT p.id, p.title, p.slug, 'review' AS stage, 'posts' AS engine,
         COALESCE(p.post_meta->>'language', 'pt'),
         (SELECT c.id FROM post_categories pc JOIN categories c ON c.id = pc.category_id
          WHERE pc.post_id = p.id LIMIT 1),
         p.author_id, p.seo_score, e.eeat_score,
         e.decision AS status, p.scheduled_at, p.updated_at
  FROM posts p
  JOIN LATERAL (
    SELECT r.id, r.eeat_score, r.decision
    FROM editorial_reviews r
    WHERE r.article_id = p.id AND r.site_id = p.site_id
    ORDER BY r.created_at DESC LIMIT 1
  ) e ON true
  WHERE p.site_id = $1 AND p.status = 'draft' AND p.deleted_at IS NULL
    AND e.decision = 'needs_review'

  UNION ALL
  -- approval: latest review approved the article (waiting human action)
  SELECT p.id, p.title, p.slug, 'approval' AS stage, 'posts' AS engine,
         COALESCE(p.post_meta->>'language', 'pt'),
         (SELECT c.id FROM post_categories pc JOIN categories c ON c.id = pc.category_id
          WHERE pc.post_id = p.id LIMIT 1),
         p.author_id, p.seo_score, e.eeat_score,
         e.decision AS status, p.scheduled_at, p.updated_at
  FROM posts p
  JOIN LATERAL (
    SELECT r.id, r.eeat_score, r.decision
    FROM editorial_reviews r
    WHERE r.article_id = p.id AND r.site_id = p.site_id
    ORDER BY r.created_at DESC LIMIT 1
  ) e ON true
  WHERE p.site_id = $1 AND p.status = 'draft' AND p.deleted_at IS NULL
    AND e.decision = 'approved'

  UNION ALL
  -- approval: pending human approval requests
  SELECT ar.id, COALESCE(p.title, '') AS title, COALESCE(p.slug, '') AS slug,
         'approval' AS stage, 'approval_requests' AS engine,
         COALESCE(p.post_meta->>'language', 'pt'),
         NULL::uuid AS category_id, p.author_id, p.seo_score, NULL::numeric,
         ar.status, p.scheduled_at, COALESCE(p.updated_at, ar.updated_at)
  FROM approval_requests ar
  LEFT JOIN posts p ON p.id = ar.post_id
  WHERE ar.site_id = $1 AND ar.status = 'pending'

  UNION ALL
  -- scheduled: scheduled posts (board order by scheduled_at)
  SELECT p.id, p.title, p.slug, 'scheduled' AS stage, 'posts' AS engine,
         COALESCE(p.post_meta->>'language', 'pt'),
         (SELECT c.id FROM post_categories pc JOIN categories c ON c.id = pc.category_id
          WHERE pc.post_id = p.id LIMIT 1),
         p.author_id, p.seo_score, NULL::numeric,
         p.status, p.scheduled_at, p.updated_at
  FROM posts p
  WHERE p.site_id = $1 AND p.status = 'scheduled' AND p.deleted_at IS NULL

  UNION ALL
  -- published: most recent 30 published posts (board stays focused)
  SELECT p.id, p.title, p.slug, 'published' AS stage, 'posts' AS engine,
         COALESCE(p.post_meta->>'language', 'pt'),
         (SELECT c.id FROM post_categories pc JOIN categories c ON c.id = pc.category_id
          WHERE pc.post_id = p.id LIMIT 1),
         p.author_id, p.seo_score, NULL::numeric,
         p.status, p.scheduled_at, p.updated_at
  FROM posts p
  WHERE p.site_id = $1 AND p.status = 'published' AND p.deleted_at IS NULL
  ORDER BY p.published_at DESC LIMIT 30
) u
ORDER BY u.updated_at DESC
LIMIT $2
`

// GetPipeline returns the editorial board: every tracked item with its stage.
func (s *Service) GetPipeline(ctx context.Context, siteID uuid.UUID, limit int) (*PipelineResponse, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 200
	}

	rows, err := p.Query(ctx, pipelineUnionSQL, siteID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to load pipeline: %w", err)
	}
	defer rows.Close()

	items := make([]PipelineItem, 0, 32)
	for rows.Next() {
		var it PipelineItem
		var stage, engine, language, status string
		var scheduledAt *time.Time
		if err := rows.Scan(&it.ID, &it.Title, &it.Slug, &stage, &engine, &language,
			&it.CategoryID, &it.AuthorID, &it.SEOScore, &it.EEATScore,
			&status, &scheduledAt, &it.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan pipeline item: %w", err)
		}
		it.Stage = PipelineStage(stage)
		it.Engine = engine
		it.Language = language
		it.Status = status
		it.ScheduledAt = scheduledAt
		it.Actionable = it.Stage == StageReview || it.Stage == StageApproval
		items = append(items, it)
	}
	if items == nil {
		items = []PipelineItem{}
	}

	resp := &PipelineResponse{Items: items, Total: len(items)}
	byStage := map[PipelineStage]int{}
	for _, st := range PipelineStageOrder {
		byStage[st] = 0
	}
	for _, it := range items {
		byStage[it.Stage]++
	}
	for _, st := range PipelineStageOrder {
		resp.Stages = append(resp.Stages, StageCount{Stage: st, Count: byStage[st]})
	}
	return resp, nil
}

// pipelineStatsSQL counts every column independently (full numbers, not the
// LIMIT-200 board slice).
const pipelineStatsSQL = `
SELECT
  (SELECT COUNT(*) FROM editorial_briefs WHERE site_id = $1 AND outline = '[]'::jsonb),
  (SELECT COUNT(*) FROM research_jobs WHERE site_id = $1 AND status NOT IN ('completed','failed')),
  (SELECT COUNT(*) FROM editorial_briefs WHERE site_id = $1 AND outline <> '[]'::jsonb),
  (SELECT COUNT(*) FROM autocontent_jobs WHERE site_id = $1 AND status NOT IN ('completed','failed','cancelled'))
    + (SELECT COUNT(*) FROM generation_jobs WHERE site_id = $1 AND status NOT IN ('completed','failed','cancelled'))
    + (SELECT COUNT(*) FROM article_pipeline_jobs WHERE site_id = $1 AND status NOT IN ('completed','failed','cancelled'))
    + (SELECT COUNT(*) FROM workflow_jobs WHERE site_id = $1 AND status NOT IN ('completed','failed','cancelled')),
  (SELECT COUNT(*) FROM posts WHERE site_id = $1 AND status = 'draft' AND deleted_at IS NULL
     AND seo_analyzed_at IS NOT NULL
     AND NOT EXISTS (SELECT 1 FROM editorial_reviews r WHERE r.article_id = posts.id)),
  (SELECT COUNT(*) FROM posts p JOIN LATERAL (
     SELECT r.decision FROM editorial_reviews r
     WHERE r.article_id = p.id AND r.site_id = p.site_id ORDER BY r.created_at DESC LIMIT 1) e ON true
   WHERE p.site_id = $1 AND p.status = 'draft' AND p.deleted_at IS NULL),
  (SELECT COUNT(*) FROM translation_jobs WHERE site_id = $1 AND status IN ('pending','running','waiting_review')),
  (SELECT COUNT(*) FROM posts p JOIN LATERAL (
     SELECT r.decision FROM editorial_reviews r
     WHERE r.article_id = p.id AND r.site_id = p.site_id ORDER BY r.created_at DESC LIMIT 1) e ON true
   WHERE p.site_id = $1 AND p.status = 'draft' AND p.deleted_at IS NULL AND e.decision = 'needs_review'),
  (SELECT COUNT(*) FROM posts p JOIN LATERAL (
     SELECT r.decision FROM editorial_reviews r
     WHERE r.article_id = p.id AND r.site_id = p.site_id ORDER BY r.created_at DESC LIMIT 1) e ON true
   WHERE p.site_id = $1 AND p.status = 'draft' AND p.deleted_at IS NULL AND e.decision = 'approved')
    + (SELECT COUNT(*) FROM approval_requests WHERE site_id = $1 AND status = 'pending'),
  (SELECT COUNT(*) FROM posts WHERE site_id = $1 AND status = 'scheduled' AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM posts WHERE site_id = $1 AND status = 'published' AND deleted_at IS NULL)
`

func (s *Service) GetPipelineStats(ctx context.Context, siteID uuid.UUID) (*PipelineStats, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	counts := make([]int, 11)
	if err := p.QueryRow(ctx, pipelineStatsSQL, siteID).Scan(
		&counts[0], &counts[1], &counts[2], &counts[3], &counts[4], &counts[5],
		&counts[6], &counts[7], &counts[8], &counts[9], &counts[10],
	); err != nil {
		return nil, fmt.Errorf("failed to load pipeline stats: %w", err)
	}

	var avgSEO, avgEEAT *float64
	rows, err := p.Query(ctx,
		`SELECT seo_score, eeat_score FROM editorial_reviews WHERE site_id = $1`, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to load pipeline averages: %w", err)
	}
	defer rows.Close()
	var sumSEO, sumEEAT float64
	avgCount := 0
	for rows.Next() {
		var seo, eeat float64
		if err := rows.Scan(&seo, &eeat); err != nil {
			return nil, fmt.Errorf("failed to scan review averages: %w", err)
		}
		sumSEO += seo
		sumEEAT += eeat
		avgCount++
	}
	if avgCount > 0 {
		avgSEO = floatPtr(sumSEO / float64(avgCount))
		avgEEAT = floatPtr(sumEEAT / float64(avgCount))
	}

	var publishedWeek int
	if err := p.QueryRow(ctx,
		`SELECT COUNT(*) FROM posts WHERE site_id = $1 AND status = 'published' AND deleted_at IS NULL
		 AND published_at >= NOW() - INTERVAL '7 days'`, siteID,
	).Scan(&publishedWeek); err != nil {
		return nil, fmt.Errorf("failed to load weekly publications: %w", err)
	}

	stats := &PipelineStats{
		TotalItems:       sumInts(counts...),
		AvgSEOScore:      avgSEO,
		AvgEEATScore:     avgEEAT,
		PendingReviews:   counts[7],
		PendingApprovals: counts[8],
		InTranslation:    counts[6],
		PublishedWeek:    publishedWeek,
	}
	for i, st := range PipelineStageOrder {
		stats.StageCounts = append(stats.StageCounts, StageCount{Stage: st, Count: counts[i]})
	}
	return stats, nil
}

func floatPtr(v float64) *float64 {
	return &v
}

func sumInts(vals ...int) int {
	total := 0
	for _, v := range vals {
		total += v
	}
	return total
}

// GetPublishReadiness returns a pass/fail checklist for the publish funnel:
// pipeline → SEO → EEAT → Freshness → Editorial note. Missing data is
// fail-open (a missing review never blocks a manual publish).
func (s *Service) GetPublishReadiness(ctx context.Context, siteID, postID uuid.UUID) (*PublishReadiness, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	pr := &PublishReadiness{PostID: postID, Checks: []ReadinessCheck{}}
	var seoAnalyzedAt *time.Time
	var seoScore *float64
	if err := p.QueryRow(ctx,
		`SELECT title, slug, seo_score, seo_analyzed_at FROM posts
		 WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`, postID, siteID,
	).Scan(&pr.Title, &pr.Slug, &seoScore, &seoAnalyzedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to load post readiness: %w", err)
	}

	var r seoScoreRow
	hasReview := false
	err = p.QueryRow(ctx,
		`SELECT seo_score, eeat_score, freshness_score, coverage_score, naturalness_score,
		        confidence_score, final_score, decision, threshold
		 FROM editorial_reviews
		 WHERE article_id = $1 AND site_id = $2
		 ORDER BY created_at DESC LIMIT 1`, postID, siteID,
	).Scan(&r.SEO, &r.EEAT, &r.Freshness, &r.Coverage, &r.Naturalness, &r.Confidence, &r.Final, &r.Decision, &r.Threshold)
	if err == nil {
		hasReview = true
	} else if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to load editorial review: %w", err)
	}

	minSEO := s.cfg.SEO.MinPublishScore
	minFinal := s.cfg.Editorial.MinFinalScore
	pr.Checks = s.buildReadinessChecks(seoAnalyzedAt, seoScore, hasReview, &r, minSEO, minFinal)

	ready := true
	for _, c := range pr.Checks {
		if !c.Passed {
			ready = false
			if pr.Blocking == "" {
				pr.Blocking = c.Stage
			}
		}
	}
	pr.Ready = ready
	return pr, nil
}

func (s *Service) buildReadinessChecks(seoAnalyzedAt *time.Time, seoScore *float64, hasReview bool, r *seoScoreRow, minSEO, minFinal float64) []ReadinessCheck {
	checks := []ReadinessCheck{
		{Stage: "pipeline", Label: "Pipeline concluído", Passed: seoAnalyzedAt != nil,
			Message: "Ainda não passou pela auditoria SEO" },
	}
	switch {
	case seoAnalyzedAt == nil:
		checks = append(checks, ReadinessCheck{Stage: "seo", Label: "SEO mínimo", Passed: true,
			Message: "Sem auditoria — o gate avalia na publicação"})
	case seoScore != nil && *seoScore >= minSEO:
		checks = append(checks, ReadinessCheck{Stage: "seo", Label: "SEO mínimo", Passed: true,
			Message: fmt.Sprintf("SEO %.1f ≥ %.1f", *seoScore, minSEO)})
	default:
		score := 0.0
		if seoScore != nil {
			score = *seoScore
		}
		checks = append(checks, ReadinessCheck{Stage: "seo", Label: "SEO mínimo", Passed: false,
			Message: fmt.Sprintf("SEO %.1f < %.1f", score, minSEO)})
	}
	checks = append(checks, s.reviewCheck("eeat", "E-E-A-T mínimo", hasReview, r.EEAT, minFinal)...)
	checks = append(checks, s.reviewCheck("freshness", "Freshness mínimo", hasReview, r.Freshness, minFinal)...)
	checks = append(checks, s.reviewCheck("editorial", "Nota editorial final", hasReview, r.Final, minFinal)...)
	return checks
}

// reviewCheck fail-opens when no review exists (manual publish is never
// blocked by a missing AI note).
func (s *Service) reviewCheck(stage, label string, hasReview bool, score float64, min float64) []ReadinessCheck {
	if !hasReview {
		return []ReadinessCheck{{Stage: stage, Label: label, Passed: true,
			Message: "Sem nota editorial — publish manual liberado"}}
	}
	if score >= min {
		return []ReadinessCheck{{Stage: stage, Label: label, Passed: true,
			Message: fmt.Sprintf("%.1f ≥ %.1f", score, min)}}
	}
	return []ReadinessCheck{{Stage: stage, Label: label, Passed: false,
		Message: fmt.Sprintf("%.1f < %.1f", score, min)}}
}

// seoScoreRow is the latest editorial review summary (mirrors ReviewScores).
type seoScoreRow struct {
	SEO, EEAT, Freshness, Coverage, Naturalness, Confidence, Final float64
	Decision string
	Threshold float64
}

// GetArticleReview composes the full review screen: post, scores, sources,
// link suggestions, problems and the deterministic "IA recomenda" block.
func (s *Service) GetArticleReview(ctx context.Context, siteID, postID uuid.UUID) (*ArticleReview, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	ar := &ArticleReview{Sources: []ReviewSource{}, InternalLinks: []ReviewLink{}, ExternalLinks: []ReviewLink{}, Problems: []ReviewProblem{}}
	var contentText string
	if err := p.QueryRow(ctx,
		`SELECT title, slug, status, COALESCE(post_meta->>'language', 'pt'), seo_score, seo_analyzed_at, updated_at, content::text
		 FROM posts WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`, postID, siteID,
	).Scan(&ar.Post.Title, &ar.Post.Slug, &ar.Post.Status, &ar.Post.Language, &ar.Post.SEOScore, &ar.Post.SEOAnalyzedAt, &ar.Post.UpdatedAt, &contentText); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to load review post: %w", err)
	}
	ar.Post.ID = postID

	var rev ReviewScores
	revID := uuid.Nil
	var hasReview bool
	err = p.QueryRow(ctx,
		`SELECT id, seo_score, eeat_score, freshness_score, coverage_score, naturalness_score,
		        confidence_score, final_score, decision, threshold, created_at
		 FROM editorial_reviews
		 WHERE article_id = $1 AND site_id = $2
		 ORDER BY created_at DESC LIMIT 1`, postID, siteID,
	).Scan(&revID, &rev.SEO, &rev.EEAT, &rev.Freshness, &rev.Coverage, &rev.Naturalness,
		&rev.Confidence, &rev.Final, &rev.Decision, &rev.Threshold, &rev.CreatedAt)
	if err == nil {
		hasReview = true
		ar.Review = &rev
	} else if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to load review scores: %w", err)
	}

	var coverage, fluency, semantic map[string]interface{}
	if hasReview {
		var covRaw, fluRaw, semRaw []byte
		if err := p.QueryRow(ctx,
			`SELECT coverage::text, fluency::text, semantic::text FROM editorial_reviews WHERE id = $1`, revID,
		).Scan(&covRaw, &fluRaw, &semRaw); err != nil {
			return nil, fmt.Errorf("failed to load review details: %w", err)
		}
		json.Unmarshal(covRaw, &coverage)
		json.Unmarshal(fluRaw, &fluency)
		json.Unmarshal(semRaw, &semantic)
	}

	if err := s.loadReviewSources(ctx, p, siteID, postID, ar); err != nil {
		return nil, err
	}
	if err := s.loadReviewEvidence(ctx, p, revID, ar); err != nil {
		return nil, err
	}

	problems := s.collectProblems(ctx, ar, coverage, fluency, semantic)
	if hasReview {
		problems = append(problems, s.reviewProblems(ar.Review)...)
	}
	ar.Problems = problems

	s.loadLinkSuggestions(ctx, siteID, postID, ar, contentText)

	readiness, err := s.GetPublishReadiness(ctx, siteID, postID)
	if err == nil {
		ar.Readiness = readiness
	}

	ar.Recommendations = s.buildRecommendations(ar, hasReview)
	return ar, nil
}

func (s *Service) loadReviewSources(ctx context.Context, p database.Pool, siteID, postID uuid.UUID, ar *ArticleReview) error {
	rows, err := p.Query(ctx,
		`SELECT source_url, COALESCE(title, ''), COALESCE(snippet, ''), COALESCE(language, ''),
		        COALESCE(is_verified, false), freshness_score, relevance_score, retrieved_at
		 FROM article_sources
		 WHERE site_id = $1 AND article_id = $2
		 ORDER BY is_verified DESC, relevance_score DESC LIMIT 20`, siteID, postID)
	if err != nil {
		return fmt.Errorf("failed to load review sources: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var src ReviewSource
		if err := rows.Scan(&src.URL, &src.Title, &src.Snippet, &src.Language,
			&src.IsVerified, &src.FreshnessScore, &src.RelevanceScore, &src.RetrievedAt); err != nil {
			return fmt.Errorf("failed to scan review source: %w", err)
		}
		ar.Sources = append(ar.Sources, src)
	}
	if ar.Sources == nil {
		ar.Sources = []ReviewSource{}
	}
	return nil
}

func (s *Service) loadReviewEvidence(ctx context.Context, p database.Pool, revID uuid.UUID, ar *ArticleReview) error {
	if revID == uuid.Nil {
		return nil
	}
	rows, err := p.Query(ctx,
		`SELECT claim, source_title, source_url, confidence, note
		 FROM editorial_evidence
		 WHERE review_id = $1 AND verified = false
		 ORDER BY created_at DESC LIMIT 10`, revID)
	if err != nil {
		return fmt.Errorf("failed to load review evidence: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var claim, srcTitle, srcURL, note string
		var confidence float64
		if err := rows.Scan(&claim, &srcTitle, &srcURL, &confidence, &note); err != nil {
			return fmt.Errorf("failed to scan review evidence: %w", err)
		}
		msg := claim
		if srcTitle != "" {
			msg = claim + " (fonte: " + srcTitle + ")"
		}
		ar.Problems = append(ar.Problems, ReviewProblem{Kind: "evidence", Message: msg, Severity: "warning"})
	}
	return nil
}

func (s *Service) collectProblems(ctx context.Context, ar *ArticleReview, coverage, fluency, semantic map[string]interface{}) []ReviewProblem {
	problems := []ReviewProblem{}

	var seoIssues []map[string]interface{}
	if err := s.loadSeoIssues(ctx, ar.Post.ID, &seoIssues); err == nil {
		for _, iss := range seoIssues {
			sev := "info"
			switch strings.ToLower(fieldString(iss, "priority")) {
			case "high", "critical":
				sev = "error"
			case "medium":
				sev = "warning"
			}
			problems = append(problems, ReviewProblem{
				Kind: "seo", Message: fieldString(iss, "issue"), Severity: sev,
			})
		}
	}

	if missing := jsonMissing(coverage, "missing"); len(missing) > 0 {
		for _, m := range missing {
			problems = append(problems, ReviewProblem{Kind: "coverage", Message: m, Severity: "warning"})
		}
	}
	if issues := jsonIssues(fluency); len(issues) > 0 {
		for _, m := range issues {
			problems = append(problems, ReviewProblem{Kind: "fluency", Message: m, Severity: "warning"})
		}
	}
	if terms := jsonStringList(semantic, "missing_terms"); len(terms) > 0 {
		for _, t := range terms {
			problems = append(problems, ReviewProblem{Kind: "semantic", Message: "Termo ausente: " + t, Severity: "warning"})
		}
	}
	unverified := 0
	for _, src := range ar.Sources {
		if !src.IsVerified {
			unverified++
		}
	}
	if unverified > 0 {
		problems = append(problems, ReviewProblem{
			Kind: "sources", Message: fmt.Sprintf("%d fonte(s) não verificada(s)", unverified), Severity: "warning",
		})
	}
	return problems
}

func (s *Service) reviewProblems(rev *ReviewScores) []ReviewProblem {
	problems := []ReviewProblem{}
	if rev.Final < rev.Threshold {
		problems = append(problems, ReviewProblem{
			Kind: "editorial", Severity: "error",
			Message: fmt.Sprintf("Nota editorial %.1f abaixo do limite %.1f", rev.Final, rev.Threshold),
		})
	}
	return problems
}

func (s *Service) loadLinkSuggestions(ctx context.Context, siteID, postID uuid.UUID, ar *ArticleReview, contentText string) {
	if s.linkSuggestor == nil {
		return
	}
	keyword := deriveKeyword(ar.Post.Title)
	if cands, err := s.linkSuggestor.SelectInternalLinks(ctx, siteID, &postID, ar.Post.Title, contentText, keyword, "", 40, 5); err == nil {
		for _, c := range cands {
			ar.InternalLinks = append(ar.InternalLinks, ReviewLink{
				Title: c.Title, URL: "/" + c.Slug, AnchorText: c.AnchorText, Score: c.Relevance,
			})
		}
	}
	minRel := s.cfg.SEO.ExternalLinkMinReliability
	if minRel == 0 {
		minRel = 75
	}
	if cands, err := s.linkSuggestor.SelectExternalLinks(ctx, siteID, ar.Post.Title, minRel, 5); err == nil {
		for _, c := range cands {
			ar.ExternalLinks = append(ar.ExternalLinks, ReviewLink{
				Title: c.Title, URL: c.URL, Score: c.Relevance, Label: c.Label, Reliability: c.Reliability,
			})
		}
	}
}

// buildRecommendations is the deterministic "IA recomenda" block.
func (s *Service) buildRecommendations(ar *ArticleReview, hasReview bool) []ReviewRecommendation {
	recs := []ReviewRecommendation{}
	minSEO := s.cfg.SEO.MinPublishScore
	minFinal := s.cfg.Editorial.MinFinalScore

	seoDetails := []string{}
	for _, pr := range ar.Problems {
		if pr.Kind == "seo" && len(seoDetails) < 3 {
			seoDetails = append(seoDetails, pr.Message)
		}
	}
	recs = append(recs, reviewRec("SEO", ar.Post.SEOScore, seoStatus(ar.Post.SEOScore, minSEO), seoDetails))

	var eeatScore, freshScore, naturalScore, finalScore float64
	if hasReview {
		eeatScore, freshScore, naturalScore, finalScore = ar.Review.EEAT, ar.Review.Freshness, ar.Review.Naturalness, ar.Review.Final
	}
	eeatDetails := []string{}
	for _, pr := range ar.Problems {
		if (pr.Kind == "coverage" || pr.Kind == "semantic") && len(eeatDetails) < 3 {
			eeatDetails = append(eeatDetails, pr.Message)
		}
	}
	recs = append(recs, reviewRec("E-E-A-T", &eeatScore, thresholdStatus(eeatScore, hasReview, minFinal), eeatDetails))
	recs = append(recs, reviewRec("Freshness", &freshScore, thresholdStatus(freshScore, hasReview, minFinal), nil))

	readDetails := []string{}
	for _, pr := range ar.Problems {
		if pr.Kind == "fluency" && len(readDetails) < 3 {
			readDetails = append(readDetails, pr.Message)
		}
	}
	natural := 0.0
	hasNatural := hasReview
	if hasReview {
		natural = naturalScore
	}
	recs = append(recs, reviewRec("Leitura", &natural, thresholdStatus(natural, hasNatural, 70), readDetails))

	verified, total := 0, 0
	for _, src := range ar.Sources {
		total++
		if src.IsVerified {
			verified++
		}
	}
	evidenceStatus := "info"
	var evidenceScore *float64
	evidenceDetails := []string{}
	if total > 0 {
		sc := float64(verified) / float64(total) * 100
		evidenceScore = &sc
		evidenceStatus = "ok"
		if sc < 50 {
			evidenceStatus = "warning"
		}
		evidenceDetails = append(evidenceDetails,
			fmt.Sprintf("%d/%d fontes verificadas", verified, total))
	}
	for _, pr := range ar.Problems {
		if pr.Kind == "evidence" {
			evidenceDetails = append(evidenceDetails, pr.Message)
		}
	}
	recs = append(recs, reviewRec("Fontes", evidenceScore, evidenceStatus, evidenceDetails))

	problemStatus := "ok"
	problemDetails := []string{}
	if len(ar.Problems) > 0 {
		problemStatus = "warning"
		for _, pr := range ar.Problems {
			if len(problemDetails) < 5 {
				problemDetails = append(problemDetails, "• "+pr.Message)
			}
		}
	}
	recs = append(recs, reviewRec("Problemas", nil, problemStatus, problemDetails))

	if hasReview && finalScore < minFinal {
		recs = append(recs, ReviewRecommendation{
			Label: "Ação recomendada", Status: "fail",
			Details: []string{"Enviar de volta para revisão antes de publicar."},
		})
	}
	return recs
}

func reviewRec(label string, score *float64, status string, details []string) ReviewRecommendation {
	return ReviewRecommendation{Label: label, Score: score, Status: status, Details: details}
}

func seoStatus(score *float64, min float64) string {
	if score == nil {
		return "info"
	}
	if *score >= min {
		return "ok"
	}
	return "warning"
}

func thresholdStatus(score float64, has bool, min float64) string {
	if !has {
		return "info"
	}
	if score >= min {
		return "ok"
	}
	return "warning"
}

// loadSeoIssues reads the post's stored seo_issues JSONB without failing the
// review screen on DB errors (best-effort).
func (s *Service) loadSeoIssues(ctx context.Context, postID uuid.UUID, out *[]map[string]interface{}) error {
	p, err := s.pool()
	if err != nil {
		return err
	}
	var raw []byte
	if err := p.QueryRow(ctx,
		`SELECT COALESCE(seo_issues::text, '[]') FROM posts WHERE id = $1`, postID,
	).Scan(&raw); err != nil {
		return err
	}
	json.Unmarshal(raw, out)
	return nil
}

func jsonMissing(m map[string]interface{}, key string) []string {
	out := []string{}
	if m == nil {
		return out
	}
	raw, ok := m[key]
	if !ok {
		return out
	}
	items, ok := raw.([]interface{})
	if !ok {
		return out
	}
	for _, it := range items {
		obj, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		if msg := fieldString(obj, "message"); msg != "" {
			out = append(out, msg)
		}
	}
	return out
}

func jsonIssues(m map[string]interface{}) []string {
	out := []string{}
	if m == nil {
		return out
	}
	raw, ok := m["issues"]
	if !ok {
		return out
	}
	items, ok := raw.([]interface{})
	if !ok {
		return out
	}
	for _, it := range items {
		obj, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		if msg := fieldString(obj, "message"); msg != "" {
			out = append(out, msg)
		}
	}
	return out
}

func jsonStringList(m map[string]interface{}, key string) []string {
	out := []string{}
	if m == nil {
		return out
	}
	raw, ok := m[key]
	if !ok {
		return out
	}
	items, ok := raw.([]interface{})
	if !ok {
		return out
	}
	for _, it := range items {
		if str, ok := it.(string); ok {
			out = append(out, str)
		}
	}
	return out
}

func fieldString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	if str, ok := v.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", v)
}

var stopWords = map[string]bool{
	"o": true, "a": true, "os": true, "as": true, "de": true, "do": true, "da": true,
	"dos": true, "das": true, "e": true, "em": true, "no": true, "na": true, "para": true,
	"com": true, "um": true, "uma": true, "por": true, "que": true, "como": true,
	"the": true, "and": true, "for": true, "with": true, "from": true, "you": true,
	"your": true, "what": true, "how": true, "when": true, "why": true, "not": true,
}

// deriveKeyword picks the longest non-stopword token of the title.
func deriveKeyword(title string) string {
	best := ""
	re := regexp.MustCompile(`[A-Za-zÀ-ÿ0-9]{4,}`)
	for _, tok := range re.FindAllString(title, -1) {
		if stopWords[strings.ToLower(tok)] {
			continue
		}
		if len(tok) > len(best) {
			best = tok
		}
	}
	return best
}
