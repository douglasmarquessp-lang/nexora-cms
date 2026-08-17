package seoengine

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"nexora/internal/ai"
	"nexora/internal/kernel"
	"nexora/internal/modules/publisher"
	"nexora/internal/pkg/audit"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

func fnv64(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

type Service struct {
	log           *logger.Logger
	db            *database.Database
	cache         *cache.Cache
	eventBus      *kernel.EventBus
	auditLog       *audit.Logger
	qualityChecker ai.QualityChecker
	aiManager     *ai.Manager
	imageProvider ImageProvider

	competitorDomains          []string
	internalLinkMinScore       int
	internalLinkMax            int
	externalLinkMinReliability int
	defaultAuthor              string
	minPublishScore            float64
	minWordCount               int
	minQualityScore            float64
}

func NewService(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *Service {
	var pool database.Pool
	if db != nil {
		pool = db.Pool
	}
	s := &Service{
		log:      log,
		db:       db,
		cache:    ch,
		auditLog: audit.New(pool, log),
	}
	if cfg != nil {
		s.competitorDomains = cfg.SEO.CompetitorDomains
		s.internalLinkMinScore = cfg.SEO.InternalLinkMinScore
		s.internalLinkMax = cfg.SEO.InternalLinkMax
		s.externalLinkMinReliability = cfg.SEO.ExternalLinkMinReliability
		s.defaultAuthor = cfg.SEO.DefaultAuthor
		s.minPublishScore = cfg.SEO.MinPublishScore
		s.minWordCount = cfg.SEO.MinWordCount
	}
	if s.internalLinkMinScore == 0 {
		s.internalLinkMinScore = 40
	}
	if s.internalLinkMax == 0 {
		s.internalLinkMax = 5
	}
	if s.externalLinkMinReliability == 0 {
		s.externalLinkMinReliability = 75
	}
	if s.minWordCount == 0 {
		s.minWordCount = 1000
	}
	if s.minQualityScore == 0 {
		s.minQualityScore = publisher.MinQualityGateScore
	}
	return s
}

func (s *Service) SetEventBus(bus *kernel.EventBus) {
	s.eventBus = bus
}

func (s *Service) SetQualityChecker(qc ai.QualityChecker) {
	s.qualityChecker = qc
}

// SetImageProvider wires the optional image provider (Pexels) used by the
// content enhancer to embed a real photograph and its honest alt text into
// generated articles. Nil (default) disables image enrichment.
func (s *Service) SetImageProvider(p ImageProvider) {
	s.imageProvider = p
}

func (s *Service) SetAIManager(m *ai.Manager) {
	s.aiManager = m
}

func (s *Service) fireEvent(ctx context.Context, eventType kernel.EventType, payload interface{}, siteID uuid.UUID) {
	if s.eventBus != nil {
		s.eventBus.EmitAsync(ctx, eventType, payload, siteID.String())
	}
}

func (s *Service) pool() (database.Pool, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, ErrDatabaseNotAvail
	}
	return s.db.Pool, nil
}

// --- Project CRUD ---

func (s *Service) CreateProject(ctx context.Context, siteID, userID uuid.UUID, req CreateProjectRequest) (*SEOProject, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("project title is required")
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}
	if lang != "pt" && lang != "en" {
		return nil, ErrInvalidLanguage
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	projectID := uuid.New()
	ct := req.ContentType
	if ct == "" {
		ct = "article"
	}

	_, err = p.Exec(ctx,
		`INSERT INTO seo_projects (id, site_id, user_id, title, target_url, post_id, language, status,
		 content_type, slug_target, meta_title_target, meta_description_target, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'pending',$8,$9,$10,$11,$12,$13,$13)`,
		projectID, siteID, userID, req.Title, req.TargetURL, req.PostID, lang,
		ct, req.SlugTarget, req.MetaTitleTarget, req.MetaDescriptionTarget, userID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create seo project: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     "seoengine.project.created",
		EntityType: "seo_project",
		EntityID:   &projectID,
		Payload:    map[string]interface{}{"title": req.Title, "language": lang},
	})

	s.fireEvent(ctx, EventSEOProjectCreated, map[string]interface{}{
		"project_id": projectID.String(),
		"site_id":    siteID.String(),
		"title":      req.Title,
	}, siteID)

	return s.GetProject(ctx, siteID, projectID)
}

func (s *Service) GetProject(ctx context.Context, siteID, projectID uuid.UUID) (*SEOProject, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var proj SEOProject
	var checklistStr, aiStr string
	err = p.QueryRow(ctx,
		`SELECT id, site_id, user_id, title, COALESCE(target_url,''), post_id, language, status,
		        COALESCE(seo_score,0), COALESCE(readability_score,0), COALESCE(keyword_density,0),
		        COALESCE(content_quality,0), COALESCE(technical_score,0), COALESCE(eeat_score,0),
		        COALESCE(freshness_score,0), COALESCE(topical_authority_score,0),
		        COALESCE(slug_target,''), COALESCE(meta_title_target,''), COALESCE(meta_description_target,''),
		        COALESCE(content_type,'article'), COALESCE(recommendations,'{}'),
		        COALESCE(checklist::text,'[]'), COALESCE(ai_suggestions::text,'{}'),
		        started_at, completed_at, COALESCE(error_message,''), created_by, created_at, updated_at
		 FROM seo_projects WHERE id = $1 AND site_id = $2`,
		projectID, siteID,
	).Scan(&proj.ID, &proj.SiteID, &proj.UserID, &proj.Title, &proj.TargetURL, &proj.PostID,
		&proj.Language, &proj.Status, &proj.SEOScore, &proj.ReadabilityScore, &proj.KeywordDensity,
		&proj.ContentQuality, &proj.TechnicalScore, &proj.EEATScore, &proj.FreshnessScore,
		&proj.TopicalAuthorityScore, &proj.SlugTarget, &proj.MetaTitleTarget, &proj.MetaDescriptionTarget,
		&proj.ContentType, &proj.Recommendations, &checklistStr, &aiStr,
		&proj.StartedAt, &proj.CompletedAt, &proj.ErrorMessage, &proj.CreatedBy, &proj.CreatedAt, &proj.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get seo project: %w", err)
	}

	if len(checklistStr) > 0 {
		_ = json.Unmarshal([]byte(checklistStr), &proj.Checklist)
	}
	if proj.Checklist == nil {
		proj.Checklist = []ChecklistItem{}
	}
	if len(aiStr) > 0 {
		_ = json.Unmarshal([]byte(aiStr), &proj.AISuggestions)
	}
	if proj.AISuggestions == nil {
		proj.AISuggestions = make(map[string]interface{})
	}

	return &proj, nil
}

func (s *Service) ListProjects(ctx context.Context, siteID uuid.UUID, status, language string, limit, offset int) ([]SEOProject, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2

	if status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}
	if language != "" {
		where = append(where, fmt.Sprintf("language = $%d", argIdx))
		args = append(args, language)
		argIdx++
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, user_id, title, COALESCE(target_url,''), post_id, language, status,
		        COALESCE(seo_score,0), COALESCE(readability_score,0), COALESCE(keyword_density,0),
		        COALESCE(content_quality,0), COALESCE(technical_score,0), COALESCE(eeat_score,0),
		        COALESCE(freshness_score,0), COALESCE(topical_authority_score,0),
		        COALESCE(slug_target,''), COALESCE(meta_title_target,''), COALESCE(meta_description_target,''),
		        COALESCE(content_type,'article'), COALESCE(recommendations,'{}'),
		        COALESCE(checklist::text,'[]'), COALESCE(ai_suggestions::text,'{}'),
		        started_at, completed_at, COALESCE(error_message,''), created_by, created_at, updated_at
		 FROM seo_projects WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list seo projects: %w", err)
	}
	defer rows.Close()

	var projects []SEOProject
	for rows.Next() {
		var proj SEOProject
		var checklistStr, aiStr string
		if err := rows.Scan(&proj.ID, &proj.SiteID, &proj.UserID, &proj.Title, &proj.TargetURL, &proj.PostID,
			&proj.Language, &proj.Status, &proj.SEOScore, &proj.ReadabilityScore, &proj.KeywordDensity,
			&proj.ContentQuality, &proj.TechnicalScore, &proj.EEATScore, &proj.FreshnessScore,
			&proj.TopicalAuthorityScore, &proj.SlugTarget, &proj.MetaTitleTarget, &proj.MetaDescriptionTarget,
			&proj.ContentType, &proj.Recommendations, &checklistStr, &aiStr,
			&proj.StartedAt, &proj.CompletedAt, &proj.ErrorMessage, &proj.CreatedBy, &proj.CreatedAt, &proj.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan seo project: %w", err)
		}
		if len(checklistStr) > 0 {
			_ = json.Unmarshal([]byte(checklistStr), &proj.Checklist)
		}
		if proj.Checklist == nil {
			proj.Checklist = []ChecklistItem{}
		}
		if len(aiStr) > 0 {
			_ = json.Unmarshal([]byte(aiStr), &proj.AISuggestions)
		}
		if proj.AISuggestions == nil {
			proj.AISuggestions = make(map[string]interface{})
		}
		projects = append(projects, proj)
	}
	if projects == nil {
		projects = []SEOProject{}
	}
	return projects, nil
}

func (s *Service) UpdateProject(ctx context.Context, siteID, projectID uuid.UUID, req UpdateProjectRequest) (*SEOProject, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, err := s.GetProject(ctx, siteID, projectID)
	if err != nil {
		return nil, err
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.TargetURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("target_url = $%d", argIdx))
		args = append(args, *req.TargetURL)
		argIdx++
	}
	if req.Language != nil {
		setClauses = append(setClauses, fmt.Sprintf("language = $%d", argIdx))
		args = append(args, *req.Language)
		argIdx++
	}
	if req.ContentType != nil {
		setClauses = append(setClauses, fmt.Sprintf("content_type = $%d", argIdx))
		args = append(args, *req.ContentType)
		argIdx++
	}
	if req.SlugTarget != nil {
		setClauses = append(setClauses, fmt.Sprintf("slug_target = $%d", argIdx))
		args = append(args, *req.SlugTarget)
		argIdx++
	}
	if req.MetaTitleTarget != nil {
		setClauses = append(setClauses, fmt.Sprintf("meta_title_target = $%d", argIdx))
		args = append(args, *req.MetaTitleTarget)
		argIdx++
	}
	if req.MetaDescriptionTarget != nil {
		setClauses = append(setClauses, fmt.Sprintf("meta_description_target = $%d", argIdx))
		args = append(args, *req.MetaDescriptionTarget)
		argIdx++
	}

	if len(setClauses) == 0 {
		return existing, nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE seo_projects SET %s WHERE id = $%d AND site_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, projectID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update seo project: %w", err)
	}

	return s.GetProject(ctx, siteID, projectID)
}

func (s *Service) DeleteProject(ctx context.Context, siteID, projectID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	tag, err := p.Exec(ctx,
		`DELETE FROM seo_projects WHERE id = $1 AND site_id = $2`,
		projectID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete seo project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// --- Audit ---

func (s *Service) RunFullAudit(ctx context.Context, siteID, projectID uuid.UUID) (*SEOAudit, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	project, err := s.GetProject(ctx, siteID, projectID)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	_, err = p.Exec(ctx,
		`UPDATE seo_projects SET status = 'running', started_at = $1, updated_at = $1 WHERE id = $2`,
		now, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update project status: %w", err)
	}

	input := s.buildAnalysisInput(ctx, p, project)
	analysis := AnalyzeArticle(ctx, input, s.qualityChecker)
	issuesJSON, _ := json.Marshal(analysis.Suggestions)
	headingJSON, _ := json.Marshal(filterIssues(analysis.Suggestions, "headings", "heading"))
	imageJSON, _ := json.Marshal(filterIssues(analysis.Suggestions, "image_alt", "images"))
	schemaJSON, _ := json.Marshal(filterIssues(analysis.Suggestions, "schema"))
	eeatIssuesJSON, _ := json.Marshal(filterIssues(analysis.Suggestions, "eeat"))
	freshnessIssuesJSON, _ := json.Marshal(filterIssues(analysis.Suggestions, "freshness"))
	slugIssues := issueMessages(filterIssues(analysis.Suggestions, "slug"))
	titleIssues := issueMessages(filterIssues(analysis.Suggestions, "title"))
	metaIssues := issueMessages(filterIssues(analysis.Suggestions, "meta"))

	checklistItems := buildChecklist(analysis)
	checklistJSON, _ := json.Marshal(checklistItems)
	linkJSON, _ := json.Marshal([]LinkSuggestion{})

	auditID := uuid.New()

	_, err = p.Exec(ctx,
		`INSERT INTO seo_audits (id, site_id, seo_project_id, post_id, url,
		 title_score, meta_score, heading_score, paragraph_score, readability_score,
		 passive_voice_score, sentence_variation_score, duplicate_score, overall_score,
		 eeat_score, freshness_score, slug_score, orphan_detected, cannibalization_detected,
		 content_gap_detected, issues, heading_issues, image_alt_issues, schema_issues,
		 slug_issues, title_issues, meta_issues, link_suggestions, checklist_items,
		 eeat_issues, freshness_issues, language, audited_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,
		 $18,$19,$20,$21::jsonb,$22::jsonb,$23::jsonb,$24::jsonb,$25,$26,$27,
		 $28::jsonb,$29::jsonb,$30::jsonb,$31::jsonb,$32,$33,$34,$34)`,
		auditID, siteID, &projectID, project.PostID, project.TargetURL,
		analysis.TitleScore, analysis.MetaScore, analysis.HeadingScore, analysis.ParagraphScore, analysis.ReadabilityScore,
		analysis.PassiveVoiceScore, analysis.SentenceVariationScore, analysis.DuplicateScore, analysis.OverallScore,
		analysis.EEATScore, analysis.FreshnessScore, analysis.SlugScore,
		false, false, false,
		string(issuesJSON), string(headingJSON), string(imageJSON), string(schemaJSON),
		slugIssues, titleIssues, metaIssues,
		string(linkJSON), string(checklistJSON),
		string(eeatIssuesJSON), string(freshnessIssuesJSON),
		project.Language, now, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit: %w", err)
	}

	_, _ = p.Exec(ctx,
		`UPDATE seo_projects SET status = 'completed', seo_score = $1, readability_score = $2,
		 eeat_score = $3, freshness_score = $4, technical_score = $5,
		 checklist = $6::jsonb, completed_at = $7, updated_at = $7
		 WHERE id = $8`,
		analysis.OverallScore, analysis.ReadabilityScore, analysis.EEATScore, analysis.FreshnessScore,
		round2((analysis.TitleScore+analysis.MetaScore+analysis.SlugScore+analysis.HeadingScore+analysis.SchemaScore)/5),
		string(checklistJSON), now, projectID,
	)

	_, _ = p.Exec(ctx,
		`INSERT INTO seo_scores (id, site_id, seo_project_id, post_id, total_score,
		 keyword_score, content_score, technical_score, linking_score, readability_score,
		 metadata_score, eeat_score, freshness_score, topical_authority_score,
		 schema_score, image_score, slug_score, heading_score, multilingual_score,
		 language, scored_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$21)`,
		uuid.New(), siteID, &projectID, project.PostID, analysis.OverallScore,
		analysis.KeywordScore,
		round2((analysis.ReadabilityScore+analysis.EEATScore+analysis.FreshnessScore+analysis.KeywordScore)/4),
		round2((analysis.TitleScore+analysis.MetaScore+analysis.SlugScore+analysis.HeadingScore+analysis.SchemaScore)/5),
		round2((analysis.InternalLinksScore+analysis.ExternalLinksScore)/2),
		analysis.ReadabilityScore,
		round2((analysis.MetaScore+analysis.TitleScore)/2),
		analysis.EEATScore, analysis.FreshnessScore, analysis.TopicalAuthorityScore,
		analysis.SchemaScore, analysis.ImagesScore, analysis.SlugScore, analysis.HeadingScore,
		50, project.Language, now, now,
	)

	if project.PostID != nil {
		_, _ = p.Exec(ctx,
			`UPDATE posts SET seo_score = $1, seo_analyzed_at = $2, seo_issues = $3::jsonb, updated_at = $2
			 WHERE id = $4 AND site_id = $5`,
			analysis.OverallScore, now, string(issuesJSON), project.PostID, siteID,
		)
	}

	s.fireEvent(ctx, EventSEOAuditCompleted, map[string]interface{}{
		"project_id":    projectID.String(),
		"audit_id":      auditID.String(),
		"overall_score": analysis.OverallScore,
	}, siteID)

	s.fireEvent(ctx, EventSEOProjectCompleted, map[string]interface{}{
		"project_id": projectID.String(),
		"score":      analysis.OverallScore,
	}, siteID)

	return s.getAuditByID(ctx, p, auditID, siteID)
}

// buildAnalysisInput assembles the analyzer input from the project and, when
// available, the linked post content.
func (s *Service) buildAnalysisInput(ctx context.Context, p database.Pool, project *SEOProject) ArticleAnalysisInput {
	input := ArticleAnalysisInput{
		Title:           project.Title,
		MetaDescription: project.MetaDescriptionTarget,
		Slug:            project.SlugTarget,
		Content:         "",
		Keyword:         deriveKeyword(project.Title),
		Language:        project.Language,
	}
	if input.Language == "" {
		input.Language = "pt"
	}

	if project.PostID == nil {
		return input
	}

	var contentText string
	var slug, excerpt, metaDesc string
	err := p.QueryRow(ctx,
		`SELECT COALESCE(slug,''), COALESCE(content::text,'[]'), COALESCE(excerpt,''),
		        COALESCE(metadata->>'meta_description','')
		 FROM posts WHERE id = $1 AND site_id = $2`,
		project.PostID, project.SiteID,
	).Scan(&slug, &contentText, &excerpt, &metaDesc)
	if err != nil {
		s.log.Warn("seo audit: failed to load post content", "post_id", project.PostID, "error", err)
		return input
	}

	if input.Slug == "" {
		input.Slug = slug
	}
	if input.MetaDescription == "" {
		input.MetaDescription = excerpt
		if input.MetaDescription == "" {
			input.MetaDescription = metaDesc
		}
	}
	input.Content = extractContentText([]byte(contentText))
	return input
}

func filterIssues(issues []AuditIssue, fields ...string) []AuditIssue {
	filtered := []AuditIssue{}
	for _, iss := range issues {
		for _, f := range fields {
			if iss.Field == f {
				filtered = append(filtered, iss)
				break
			}
		}
	}
	return filtered
}

func issueMessages(issues []AuditIssue) []string {
	msgs := make([]string, 0, len(issues))
	for _, iss := range issues {
		msgs = append(msgs, iss.Issue)
	}
	return msgs
}

func buildChecklist(analysis *ArticleAnalysis) []ChecklistItem {
	items := []ChecklistItem{}
	categoryForField := map[string]ImprovementCategory{
		"title":            CategoryTitle,
		"meta":             CategoryMeta,
		"slug":             CategorySlug,
		"headings":         CategoryHeading,
		"readability":      CategoryReadability,
		"eeat":             CategoryEEAT,
		"freshness":        CategoryFreshness,
		"internal_link":    CategoryLink,
		"external_link":    CategoryLink,
		"image_alt":        CategoryImage,
		"schema":           CategorySchema,
		"passive_voice":    CategoryReadability,
		"sentence_variation": CategoryReadability,
	}
	for _, iss := range analysis.Suggestions {
		cat, ok := categoryForField[iss.Field]
		if !ok {
			cat = CategoryTitle
		}
		items = append(items, ChecklistItem{
			Category:   cat,
			Issue:      iss.Issue,
			Suggestion: iss.Suggestion,
			Priority:   ImprovementPriority(iss.Priority),
			Score:      iss.Score,
		})
	}
	return items
}

func (s *Service) getAuditByID(ctx context.Context, p database.Pool, auditID, siteID uuid.UUID) (*SEOAudit, error) {
	var a SEOAudit
	var issuesStr, headingStr, imageStr, schemaStr, slugIssues, titleIssues, metaIssues, linkStr, checklistStr, eeatIssuesStr, freshnessIssuesStr string
	err := p.QueryRow(ctx,
		`SELECT id, site_id, seo_project_id, post_id, COALESCE(url,''),
		        COALESCE(title_score,0), COALESCE(meta_score,0), COALESCE(heading_score,0),
		        COALESCE(paragraph_score,0), COALESCE(readability_score,0),
		        COALESCE(passive_voice_score,0), COALESCE(sentence_variation_score,0),
		        COALESCE(duplicate_score,0), COALESCE(overall_score,0),
		        COALESCE(eeat_score,0), COALESCE(freshness_score,0), COALESCE(slug_score,0),
		        COALESCE(orphan_detected,false), COALESCE(cannibalization_detected,false),
		        COALESCE(content_gap_detected,false),
		        COALESCE(issues::text,'[]'), COALESCE(heading_issues::text,'[]'),
		        COALESCE(image_alt_issues::text,'[]'), COALESCE(schema_issues::text,'[]'),
		        COALESCE(slug_issues,'{}'), COALESCE(title_issues,'{}'), COALESCE(meta_issues,'{}'),
		        COALESCE(link_suggestions::text,'[]'), COALESCE(checklist_items::text,'[]'),
		        COALESCE(eeat_issues::text,'[]'), COALESCE(freshness_issues::text,'[]'),
		        language, audited_at, created_at, updated_at
		 FROM seo_audits WHERE id = $1 AND site_id = $2`,
		auditID, siteID,
	).Scan(&a.ID, &a.SiteID, &a.SEOProjectID, &a.PostID, &a.URL,
		&a.TitleScore, &a.MetaScore, &a.HeadingScore,
		&a.ParagraphScore, &a.ReadabilityScore,
		&a.PassiveVoiceScore, &a.SentenceVariationScore,
		&a.DuplicateScore, &a.OverallScore,
		&a.EEATScore, &a.FreshnessScore, &a.SlugScore,
		&a.OrphanDetected, &a.CannibalizationDetected,
		&a.ContentGapDetected,
		&issuesStr, &headingStr, &imageStr, &schemaStr,
		&slugIssues, &titleIssues, &metaIssues,
		&linkStr, &checklistStr, &eeatIssuesStr, &freshnessIssuesStr,
		&a.Language, &a.AuditedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrAuditNotFound
		}
		return nil, fmt.Errorf("failed to get audit: %w", err)
	}
	unmarshalAuditJSON(&a, issuesStr, headingStr, imageStr, schemaStr, slugIssues, titleIssues, metaIssues, linkStr, checklistStr, eeatIssuesStr, freshnessIssuesStr)
	return &a, nil
}

func unmarshalAuditJSON(a *SEOAudit, issuesStr, headingStr, imageStr, schemaStr, slugIssues, titleIssues, metaIssues, linkStr, checklistStr, eeatIssuesStr, freshnessIssuesStr string) {
	if len(issuesStr) > 0 {
		_ = json.Unmarshal([]byte(issuesStr), &a.Issues)
	}
	if len(headingStr) > 0 {
		_ = json.Unmarshal([]byte(headingStr), &a.HeadingIssues)
	}
	if len(imageStr) > 0 {
		_ = json.Unmarshal([]byte(imageStr), &a.ImageAltIssues)
	}
	if len(schemaStr) > 0 {
		_ = json.Unmarshal([]byte(schemaStr), &a.SchemaIssues)
	}
	if len(linkStr) > 0 {
		_ = json.Unmarshal([]byte(linkStr), &a.LinkSuggestions)
	}
	if len(checklistStr) > 0 {
		_ = json.Unmarshal([]byte(checklistStr), &a.ChecklistItems)
	}
	if len(eeatIssuesStr) > 0 {
		_ = json.Unmarshal([]byte(eeatIssuesStr), &a.EEATIssues)
	}
	if len(freshnessIssuesStr) > 0 {
		_ = json.Unmarshal([]byte(freshnessIssuesStr), &a.FreshnessIssues)
	}
	if len(slugIssues) > 0 {
		_ = json.Unmarshal([]byte(slugIssues), &a.SlugIssues)
	}
	if len(titleIssues) > 0 {
		_ = json.Unmarshal([]byte(titleIssues), &a.TitleIssues)
	}
	if len(metaIssues) > 0 {
		_ = json.Unmarshal([]byte(metaIssues), &a.MetaIssues)
	}
	if a.Issues == nil {
		a.Issues = []AuditIssue{}
	}
	if a.HeadingIssues == nil {
		a.HeadingIssues = []AuditIssue{}
	}
	if a.ImageAltIssues == nil {
		a.ImageAltIssues = []AuditIssue{}
	}
	if a.SchemaIssues == nil {
		a.SchemaIssues = []AuditIssue{}
	}
	if a.LinkSuggestions == nil {
		a.LinkSuggestions = []LinkSuggestion{}
	}
	if a.ChecklistItems == nil {
		a.ChecklistItems = []ChecklistItem{}
	}
	if a.EEATIssues == nil {
		a.EEATIssues = []AuditIssue{}
	}
	if a.FreshnessIssues == nil {
		a.FreshnessIssues = []AuditIssue{}
	}
	if a.SlugIssues == nil {
		a.SlugIssues = []string{}
	}
	if a.TitleIssues == nil {
		a.TitleIssues = []string{}
	}
	if a.MetaIssues == nil {
		a.MetaIssues = []string{}
	}
}

func (s *Service) GetAudit(ctx context.Context, siteID, auditID uuid.UUID) (*SEOAudit, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	return s.getAuditByID(ctx, p, auditID, siteID)
}

func (s *Service) GetProjectAudits(ctx context.Context, siteID, projectID uuid.UUID) ([]SEOAudit, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, site_id, seo_project_id, post_id, COALESCE(url,''),
		        COALESCE(title_score,0), COALESCE(meta_score,0), COALESCE(heading_score,0),
		        COALESCE(paragraph_score,0), COALESCE(readability_score,0),
		        COALESCE(passive_voice_score,0), COALESCE(sentence_variation_score,0),
		        COALESCE(duplicate_score,0), COALESCE(overall_score,0),
		        COALESCE(eeat_score,0), COALESCE(freshness_score,0), COALESCE(slug_score,0),
		        COALESCE(orphan_detected,false), COALESCE(cannibalization_detected,false),
		        COALESCE(content_gap_detected,false), COALESCE(issues::text,'[]'),
		        COALESCE(heading_issues::text,'[]'), COALESCE(image_alt_issues::text,'[]'),
		        COALESCE(schema_issues::text,'[]'), COALESCE(slug_issues,'{}'),
		        COALESCE(title_issues,'{}'), COALESCE(meta_issues,'{}'),
		        COALESCE(link_suggestions::text,'[]'), COALESCE(checklist_items::text,'[]'),
		        COALESCE(eeat_issues::text,'[]'), COALESCE(freshness_issues::text,'[]'),
		        language, audited_at, created_at, updated_at
		 FROM seo_audits WHERE site_id = $1 AND seo_project_id = $2 ORDER BY created_at DESC`,
		siteID, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list audits: %w", err)
	}
	defer rows.Close()

	var audits []SEOAudit
	for rows.Next() {
		var a SEOAudit
		var issuesStr, headingStr, imageStr, schemaStr, slugIssues, titleIssues, metaIssues, linkStr, checklistStr, eeatIssuesStr, freshnessIssuesStr string
		if err := rows.Scan(&a.ID, &a.SiteID, &a.SEOProjectID, &a.PostID, &a.URL,
			&a.TitleScore, &a.MetaScore, &a.HeadingScore,
			&a.ParagraphScore, &a.ReadabilityScore,
			&a.PassiveVoiceScore, &a.SentenceVariationScore,
			&a.DuplicateScore, &a.OverallScore,
			&a.EEATScore, &a.FreshnessScore, &a.SlugScore,
			&a.OrphanDetected, &a.CannibalizationDetected,
			&a.ContentGapDetected,
			&issuesStr, &headingStr, &imageStr, &schemaStr,
			&slugIssues, &titleIssues, &metaIssues,
			&linkStr, &checklistStr, &eeatIssuesStr, &freshnessIssuesStr,
			&a.Language, &a.AuditedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan audit: %w", err)
		}
		unmarshalAuditJSON(&a, issuesStr, headingStr, imageStr, schemaStr, slugIssues, titleIssues, metaIssues, linkStr, checklistStr, eeatIssuesStr, freshnessIssuesStr)
		audits = append(audits, a)
	}
	if audits == nil {
		audits = []SEOAudit{}
	}
	return audits, nil
}

// --- Scores ---

func (s *Service) GetScores(ctx context.Context, siteID, projectID uuid.UUID) (*SEOScores, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var sc SEOScores
	err = p.QueryRow(ctx,
		`SELECT id, site_id, seo_project_id, post_id, COALESCE(total_score,0),
		        COALESCE(keyword_score,0), COALESCE(content_score,0), COALESCE(technical_score,0),
		        COALESCE(linking_score,0), COALESCE(readability_score,0), COALESCE(metadata_score,0),
		        COALESCE(eeat_score,0), COALESCE(freshness_score,0), COALESCE(topical_authority_score,0),
		        COALESCE(schema_score,0), COALESCE(image_score,0), COALESCE(slug_score,0),
		        COALESCE(heading_score,0), COALESCE(multilingual_score,0),
		        language, scored_at, created_at
		 FROM seo_scores WHERE site_id = $1 AND seo_project_id = $2 ORDER BY created_at DESC LIMIT 1`,
		siteID, projectID,
	).Scan(&sc.ID, &sc.SiteID, &sc.SEOProjectID, &sc.PostID,
		&sc.TotalScore, &sc.KeywordScore, &sc.ContentScore, &sc.TechnicalScore,
		&sc.LinkingScore, &sc.ReadabilityScore, &sc.MetadataScore,
		&sc.EEATScore, &sc.FreshnessScore, &sc.TopicalAuthorityScore,
		&sc.SchemaScore, &sc.ImageScore, &sc.SlugScore,
		&sc.HeadingScore, &sc.MultilingualScore,
		&sc.Language, &sc.ScoredAt, &sc.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrScoreNotFound
		}
		return nil, fmt.Errorf("failed to get scores: %w", err)
	}
	return &sc, nil
}

// --- Clusters ---

func (s *Service) CreateCluster(ctx context.Context, siteID uuid.UUID, req CreateClusterRequest) (*SEOCluster, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("cluster name is required")
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	clusterID := uuid.New()

	_, err = p.Exec(ctx,
		`INSERT INTO seo_clusters (id, site_id, name, description, keywords, language, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$7)`,
		clusterID, siteID, req.Name, req.Description, req.Keywords, lang, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	return &SEOCluster{
		ID:          clusterID,
		SiteID:      siteID,
		Name:        req.Name,
		Description: req.Description,
		Keywords:    req.Keywords,
		Language:    lang,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (s *Service) ListClusters(ctx context.Context, siteID uuid.UUID) ([]SEOCluster, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, site_id, name, COALESCE(description,''), COALESCE(keywords,'{}'),
		        COALESCE(article_count,0), COALESCE(avg_score,0), COALESCE(topical_authority_score,0),
		        COALESCE(semantic_entities,'{}'), COALESCE(internal_links_count,0),
		        COALESCE(content_gap_articles,'{}'), parent_cluster_id, language, created_at, updated_at
		 FROM seo_clusters WHERE site_id = $1 ORDER BY name ASC`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	defer rows.Close()

	var clusters []SEOCluster
	for rows.Next() {
		var c SEOCluster
		if err := rows.Scan(&c.ID, &c.SiteID, &c.Name, &c.Description, &c.Keywords,
			&c.ArticleCount, &c.AvgScore, &c.TopicalAuthorityScore,
			&c.SemanticEntities, &c.InternalLinksCount,
			&c.ContentGapArticles, &c.ParentClusterID, &c.Language, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan cluster: %w", err)
		}
		clusters = append(clusters, c)
	}
	if clusters == nil {
		clusters = []SEOCluster{}
	}
	return clusters, nil
}

// --- Improvements ---

func (s *Service) AddImprovement(ctx context.Context, siteID, projectID uuid.UUID, req AddImprovementRequest) (*SEOImprovement, error) {
	if req.Issue == "" {
		return nil, fmt.Errorf("issue is required")
	}
	if req.Suggestion == "" {
		return nil, fmt.Errorf("suggestion is required")
	}

	validCategory := false
	for _, c := range AllCategories {
		if req.Category == c {
			validCategory = true
			break
		}
	}
	if !validCategory {
		return nil, ErrInvalidCategory
	}

	priority := req.Priority
	if priority == "" {
		priority = PriorityMedium
	}

	lang := req.Language
	if lang == "" {
		lang = "pt"
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	improvementID := uuid.New()

	_, err = p.Exec(ctx,
		`INSERT INTO seo_improvements (id, site_id, seo_project_id, post_id, category, issue, suggestion,
		 priority, impact_score, effort_score, status, language, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'pending',$11,$12,$12)`,
		improvementID, siteID, &projectID, req.PostID, string(req.Category), req.Issue, req.Suggestion,
		string(priority), req.ImpactScore, req.EffortScore, lang, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add improvement: %w", err)
	}

	s.fireEvent(ctx, EventSEOImprovementAdded, map[string]interface{}{
		"improvement_id": improvementID.String(),
		"project_id":     projectID.String(),
		"category":       req.Category,
	}, siteID)

	return &SEOImprovement{
		ID:           improvementID,
		SiteID:       siteID,
		SEOProjectID: &projectID,
		PostID:       req.PostID,
		Category:     req.Category,
		Issue:        req.Issue,
		Suggestion:   req.Suggestion,
		Priority:     priority,
		ImpactScore:  req.ImpactScore,
		EffortScore:  req.EffortScore,
		Status:       ImprovementPending,
		Language:     lang,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (s *Service) ListImprovements(ctx context.Context, siteID, projectID uuid.UUID, category, status string) ([]SEOImprovement, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	where := []string{"site_id = $1", "seo_project_id = $2"}
	args := []interface{}{siteID, projectID}

	if category != "" {
		where = append(where, fmt.Sprintf("category = $%d", len(args)+1))
		args = append(args, category)
	}
	if status != "" {
		where = append(where, fmt.Sprintf("status = $%d", len(args)+1))
		args = append(args, status)
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, seo_project_id, post_id, category, issue, suggestion,
		        priority, COALESCE(impact_score,0), COALESCE(effort_score,0), status,
		        applied_at, language, created_at, updated_at
		 FROM seo_improvements WHERE %s ORDER BY
		   CASE priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END,
		   created_at DESC`,
		strings.Join(where, " AND "),
	)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list improvements: %w", err)
	}
	defer rows.Close()

	var improvements []SEOImprovement
	for rows.Next() {
		var imp SEOImprovement
		if err := rows.Scan(&imp.ID, &imp.SiteID, &imp.SEOProjectID, &imp.PostID,
			&imp.Category, &imp.Issue, &imp.Suggestion,
			&imp.Priority, &imp.ImpactScore, &imp.EffortScore, &imp.Status,
			&imp.AppliedAt, &imp.Language, &imp.CreatedAt, &imp.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan improvement: %w", err)
		}
		improvements = append(improvements, imp)
	}
	if improvements == nil {
		improvements = []SEOImprovement{}
	}
	return improvements, nil
}

func (s *Service) UpdateImprovement(ctx context.Context, siteID, improvementID uuid.UUID, req UpdateImprovementRequest) (*SEOImprovement, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var exists bool
	err = p.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM seo_improvements WHERE id = $1 AND site_id = $2)`,
		improvementID, siteID,
	).Scan(&exists)
	if err != nil || !exists {
		return nil, ErrImprovementNotFound
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*req.Status))
		argIdx++
		if *req.Status == ImprovementApplied {
			setClauses = append(setClauses, fmt.Sprintf("applied_at = $%d", argIdx))
			args = append(args, time.Now())
			argIdx++
		}
	}
	if req.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, string(*req.Priority))
		argIdx++
	}
	if req.ImpactScore != nil {
		setClauses = append(setClauses, fmt.Sprintf("impact_score = $%d", argIdx))
		args = append(args, *req.ImpactScore)
		argIdx++
	}
	if req.EffortScore != nil {
		setClauses = append(setClauses, fmt.Sprintf("effort_score = $%d", argIdx))
		args = append(args, *req.EffortScore)
		argIdx++
	}

	if len(setClauses) == 0 {
		return s.getImprovementByID(ctx, p, improvementID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE seo_improvements SET %s WHERE id = $%d AND site_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, improvementID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update improvement: %w", err)
	}

	return s.getImprovementByID(ctx, p, improvementID)
}

func (s *Service) getImprovementByID(ctx context.Context, p database.Pool, improvementID uuid.UUID) (*SEOImprovement, error) {
	var imp SEOImprovement
	err := p.QueryRow(ctx,
		`SELECT id, site_id, seo_project_id, post_id, category, issue, suggestion,
		        priority, COALESCE(impact_score,0), COALESCE(effort_score,0), status,
		        applied_at, language, created_at, updated_at
		 FROM seo_improvements WHERE id = $1`,
		improvementID,
	).Scan(&imp.ID, &imp.SiteID, &imp.SEOProjectID, &imp.PostID,
		&imp.Category, &imp.Issue, &imp.Suggestion,
		&imp.Priority, &imp.ImpactScore, &imp.EffortScore, &imp.Status,
		&imp.AppliedAt, &imp.Language, &imp.CreatedAt, &imp.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrImprovementNotFound
		}
		return nil, fmt.Errorf("failed to get improvement: %w", err)
	}
	return &imp, nil
}

// --- Generate Checklist ---

func (s *Service) GenerateChecklist(ctx context.Context, siteID, projectID uuid.UUID) ([]ChecklistItem, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	project, err := s.GetProject(ctx, siteID, projectID)
	if err != nil {
		return nil, err
	}

	input := s.buildAnalysisInput(ctx, p, project)
	analysis := AnalyzeArticle(ctx, input, s.qualityChecker)
	items := buildChecklist(analysis)

	checklistJSON, _ := json.Marshal(items)
	_, _ = p.Exec(ctx,
		`UPDATE seo_projects SET checklist = $1::jsonb, updated_at = NOW() WHERE id = $2`,
		string(checklistJSON), projectID,
	)

	return items, nil
}

// --- Internal Linking Suggestions ---

func (s *Service) GetInternalLinkingSuggestions(ctx context.Context, siteID, projectID uuid.UUID) ([]LinkSuggestion, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	project, err := s.GetProject(ctx, siteID, projectID)
	if err != nil {
		return nil, err
	}

	keyword := deriveKeyword(project.Title)

	rows, err := p.Query(ctx,
		`SELECT id, COALESCE(title,''), COALESCE(slug,'') FROM posts
		 WHERE site_id = $1 AND deleted_at IS NULL AND id <> COALESCE($2, '00000000-0000-0000-0000-000000000000')
		 ORDER BY created_at DESC LIMIT 25`,
		siteID, project.PostID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts for linking suggestions: %w", err)
	}
	defer rows.Close()

	type candidate struct {
		postID    uuid.UUID
		title     string
		slug      string
		relevance float64
	}
	candidates := []candidate{}
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.postID, &c.title, &c.slug); err != nil {
			return nil, fmt.Errorf("failed to scan linking candidate: %w", err)
		}
		c.relevance = keywordCoverage(c.title+" "+c.slug, keyword) * 100
		candidates = append(candidates, c)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].relevance > candidates[j].relevance
	})

	suggestions := []LinkSuggestion{}
	for _, c := range candidates {
		if c.relevance < 40 || c.slug == "" {
			continue
		}
		suggestions = append(suggestions, LinkSuggestion{
			SourceURL:  "/" + c.slug,
			TargetURL:  project.TargetURL,
			AnchorText: c.title,
			Relevance:  round2(c.relevance),
		})
		if len(suggestions) >= 5 {
			break
		}
	}

	linkJSON, _ := json.Marshal(suggestions)
	if project.TargetURL != "" {
		_, err = p.Exec(ctx,
			`UPDATE seo_audits SET link_suggestions = $1::jsonb, updated_at = NOW()
			 WHERE seo_project_id = $2 AND site_id = $3`,
			string(linkJSON), projectID, siteID,
		)
		if err != nil {
			s.log.Error("failed to update link suggestions", "error", err)
		}
	}

	return suggestions, nil
}

// --- Schema Recommendations ---

func (s *Service) GetSchemaRecommendations(ctx context.Context, siteID, projectID uuid.UUID) ([]string, error) {
	_, err := s.pool()
	if err != nil {
		return nil, err
	}

	recommendations := []string{
		"Article schema for news and blog content",
		"BreadcrumbList for navigation hierarchy",
		"FAQ schema for question-based content",
		"Organization or Person schema for authorship",
		"LocalBusiness schema if location-based",
		"Product schema for ecommerce content",
	}

	return recommendations, nil
}

// --- Orphan Articles ---

func (s *Service) DetectOrphanArticles(ctx context.Context, siteID uuid.UUID) ([]OrphanArticle, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT p.id, COALESCE(p.title,''), COALESCE(p.slug,''),
		        COALESCE((SELECT COUNT(*) FROM seo_internal_links WHERE target_url LIKE '%' || p.slug || '%' AND site_id = $1),0)
		 FROM posts p WHERE p.site_id = $1 ORDER BY p.created_at DESC LIMIT 20`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to detect orphan articles: %w", err)
	}
	defer rows.Close()

	var orphans []OrphanArticle
	for rows.Next() {
		var o OrphanArticle
		if err := rows.Scan(&o.PostID, &o.Title, &o.URL, &o.IncomingLinks); err != nil {
			return nil, fmt.Errorf("failed to scan orphan article: %w", err)
		}
		if o.IncomingLinks == 0 {
			orphans = append(orphans, o)
		}
	}
	if orphans == nil {
		orphans = []OrphanArticle{}
	}
	return orphans, nil
}

// --- Duplicate Content ---

func (s *Service) DetectDuplicates(ctx context.Context, siteID, projectID uuid.UUID) ([]DuplicateContent, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	project, err := s.GetProject(ctx, siteID, projectID)
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, COALESCE(title,''), COALESCE(content::text,'[]') FROM posts
		 WHERE site_id = $1 AND deleted_at IS NULL AND id <> COALESCE($2, '00000000-0000-0000-0000-000000000000')
		 ORDER BY created_at DESC LIMIT 20`,
		siteID, project.PostID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts for duplicate detection: %w", err)
	}
	defer rows.Close()

	type postText struct {
		id   uuid.UUID
		text string
	}
	posts := []postText{}
	for rows.Next() {
		var pt postText
		if err := rows.Scan(&pt.id, &pt.text); err != nil {
			return nil, fmt.Errorf("failed to scan post for duplicates: %w", err)
		}
		pt.text = extractContentText([]byte(pt.text))
		posts = append(posts, pt)
	}

	duplicates := []DuplicateContent{}
	for i := 0; i < len(posts); i++ {
		for j := i + 1; j < len(posts); j++ {
			sim := shingleSimilarity(posts[i].text, posts[j].text)
			if sim >= 0.6 {
				duplicates = append(duplicates, DuplicateContent{
					PostID1:    posts[i].id,
					PostID2:    posts[j].id,
					Similarity: round2(sim * 100),
					Issue:      "High similarity detected between two articles about the same topic",
				})
			}
		}
	}
	if duplicates == nil {
		duplicates = []DuplicateContent{}
	}

	return duplicates, nil
}

// --- Cannibalization ---

func (s *Service) DetectCannibalization(ctx context.Context, siteID uuid.UUID) ([]CannibalizationIssue, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT keyword, COUNT(*) as cnt FROM seo_keywords
		 WHERE site_id = $1 AND keyword_type = 'primary'
		 GROUP BY keyword HAVING COUNT(*) > 1
		 ORDER BY cnt DESC LIMIT 20`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to detect cannibalization: %w", err)
	}
	defer rows.Close()

	var issues []CannibalizationIssue
	for rows.Next() {
		var ci CannibalizationIssue
		if err := rows.Scan(&ci.Keyword, &ci.Score); err != nil {
			return nil, fmt.Errorf("failed to scan cannibalization: %w", err)
		}
		ci.Suggestion = fmt.Sprintf("Consolidate pages targeting '%s' into a single authoritative article", ci.Keyword)
		issues = append(issues, ci)
	}
	if issues == nil {
		issues = []CannibalizationIssue{}
	}
	return issues, nil
}

// --- Content Gaps ---

func (s *Service) DetectContentGaps(ctx context.Context, siteID uuid.UUID) ([]ContentGapIssue, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT k.keyword, COALESCE(c.name,''), k.volume
		 FROM seo_keywords k
		 LEFT JOIN seo_clusters c ON k.cluster_id = c.id
		 WHERE k.site_id = $1 AND k.content_gap_score > 50
		 ORDER BY k.volume DESC LIMIT 20`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to detect content gaps: %w", err)
	}
	defer rows.Close()

	var gaps []ContentGapIssue
	for rows.Next() {
		var g ContentGapIssue
		if err := rows.Scan(&g.Topic, &g.Cluster, &g.Volume); err != nil {
			return nil, fmt.Errorf("failed to scan content gap: %w", err)
		}
		if g.Volume > 1000 {
			g.Priority = "high"
		} else {
			g.Priority = "medium"
		}
		gaps = append(gaps, g)
	}
	if gaps == nil {
		gaps = []ContentGapIssue{}
	}
	return gaps, nil
}

// --- Keyword Analysis ---

func (s *Service) AnalyzeKeywords(ctx context.Context, siteID uuid.UUID, req KeywordAnalysisRequest) (*KeywordAnalysisResult, error) {
	_, err := s.pool()
	if err != nil {
		return nil, err
	}

	keywords := []SEOKeyword{}
	for _, kw := range req.Keywords {
		if strings.TrimSpace(kw) == "" {
			continue
		}
		keywords = append(keywords, SEOKeyword{
			ID:                 uuid.New(),
			SiteID:             siteID,
			Keyword:            kw,
			KeywordType:        "primary",
			SearchIntent:       req.Intent,
			Volume:             100 + int(stableScore(kw+":volume")*9900),
			Difficulty:         round2(stableScore(kw+":difficulty") * 100),
			Density:            round2(stableScore(kw+":density") * 5),
			Frequency:          1 + int(stableScore(kw+":frequency")*19),
			Prominence:         round2(stableScore(kw+":prominence") * 100),
			CannibalizationScore: 0,
			ContentGapScore:    round2(stableScore(kw+":gap") * 100),
			TopicalRelevance:   round2(stableScore(kw+":relevance")*65 + 30),
			Language:           req.Language,
		})
	}

	clusters := []SEOCluster{
		{
			ID:                    uuid.New(),
			SiteID:                siteID,
			Name:                  req.Language + " Content Cluster",
			Keywords:              req.Keywords,
			TopicalAuthorityScore: round2(avgKeywordScore(keywords)),
			SemanticEntities:      []string{},
		},
	}

	cannibalization := []CannibalizationIssue{}
	if len(req.Keywords) > 1 {
		primary := strings.ToLower(req.Keywords[0])
		for _, kw := range req.Keywords[1:] {
			if strings.Contains(strings.ToLower(kw), primary) || strings.Contains(primary, strings.ToLower(kw)) {
				cannibalization = append(cannibalization, CannibalizationIssue{
					Keyword:    kw,
					Score:      70,
					Suggestion: fmt.Sprintf("Consolidate pages targeting '%s' into a single authoritative article", kw),
				})
			}
		}
	}
	if cannibalization == nil {
		cannibalization = []CannibalizationIssue{}
	}

	contentGaps := []ContentGapIssue{}
	for _, kw := range req.Keywords {
		if strings.TrimSpace(kw) == "" {
			continue
		}
		if stableScore(kw+":gap") > 0.6 {
			contentGaps = append(contentGaps, ContentGapIssue{
				Topic:    kw + " guide",
				Cluster:  req.Language + " Content Cluster",
				Volume:   100 + int(stableScore(kw+":gapvolume")*4900),
				Priority: "high",
			})
		}
	}
	if contentGaps == nil {
		contentGaps = []ContentGapIssue{}
	}

	return &KeywordAnalysisResult{
		Keywords:        keywords,
		Clusters:        clusters,
		Cannibalization: cannibalization,
		ContentGaps:     contentGaps,
	}, nil
}

// --- Content Analysis ---

func (s *Service) AnalyzeContent(ctx context.Context, siteID, projectID uuid.UUID) (*ContentAnalysisResult, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	project, err := s.GetProject(ctx, siteID, projectID)
	if err != nil {
		return nil, err
	}

	input := s.buildAnalysisInput(ctx, p, project)
	analysis := AnalyzeArticle(ctx, input, s.qualityChecker)

	issues := filterIssues(analysis.Suggestions, "readability", "eeat", "freshness", "passive_voice", "sentence_variation")
	if issues == nil {
		issues = []AuditIssue{}
	}

	return &ContentAnalysisResult{
		ReadabilityScore: analysis.ReadabilityScore,
		EEATScore:        analysis.EEATScore,
		FreshnessScore:   analysis.FreshnessScore,
		Issues:           issues,
		Checklist:        buildChecklist(analysis),
	}, nil
}

// --- Technical Analysis ---

func (s *Service) AnalyzeTechnical(ctx context.Context, siteID, projectID uuid.UUID) (*TechnicalAnalysisResult, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	project, err := s.GetProject(ctx, siteID, projectID)
	if err != nil {
		return nil, err
	}

	input := s.buildAnalysisInput(ctx, p, project)
	analysis := AnalyzeArticle(ctx, input, s.qualityChecker)

	issues := filterIssues(analysis.Suggestions, "title", "meta", "slug", "headings", "image_alt", "schema")
	if issues == nil {
		issues = []AuditIssue{}
	}

	return &TechnicalAnalysisResult{
		TitleScore:   analysis.TitleScore,
		MetaScore:    analysis.MetaScore,
		SlugScore:    analysis.SlugScore,
		HeadingScore: analysis.HeadingScore,
		ImageScore:   analysis.ImagesScore,
		SchemaScore:  analysis.SchemaScore,
		Issues:       issues,
	}, nil
}

// --- EEAT Analysis ---

func (s *Service) CheckEEAT(ctx context.Context, siteID, projectID uuid.UUID) (*ContentAnalysisResult, error) {
	return s.AnalyzeContent(ctx, siteID, projectID)
}

// --- Freshness ---

func (s *Service) CheckFreshness(ctx context.Context, siteID, projectID uuid.UUID) (*ContentAnalysisResult, error) {
	return s.AnalyzeContent(ctx, siteID, projectID)
}

// --- Dashboard ---

func (s *Service) GetDashboardStats(ctx context.Context, siteID uuid.UUID) (*DashboardStats, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	stats := &DashboardStats{
		ByLanguage: make(map[string]int),
	}

	err = p.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*),0),
		        COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END),0),
		        COALESCE(AVG(seo_score),0),
		        COALESCE(AVG(readability_score),0),
		        COALESCE(AVG(eeat_score),0)
		 FROM seo_projects WHERE site_id = $1`,
		siteID,
	).Scan(&stats.TotalProjects, &stats.CompletedProjects, &stats.AvgSEOScore, &stats.AvgReadability, &stats.AvgEEAT)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard stats: %w", err)
	}

	err = p.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*),0) FROM seo_improvements
		 WHERE site_id = $1 AND status = 'pending'`,
		siteID,
	).Scan(&stats.PendingImprovements)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending improvements: %w", err)
	}

	err = p.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*),0) FROM seo_improvements
		 WHERE site_id = $1 AND status = 'applied'`,
		siteID,
	).Scan(&stats.AppliedImprovements)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied improvements: %w", err)
	}

	err = p.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*),0) FROM seo_audits
		 WHERE site_id = $1 AND orphan_detected = true`,
		siteID,
	).Scan(&stats.OrphanArticles)
	if err != nil {
		return nil, fmt.Errorf("failed to get orphan count: %w", err)
	}

	err = p.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*),0) FROM seo_audits
		 WHERE site_id = $1 AND cannibalization_detected = true`,
		siteID,
	).Scan(&stats.CannibalizationIssues)
	if err != nil {
		return nil, fmt.Errorf("failed to get cannibalization count: %w", err)
	}

	err = p.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*),0) FROM seo_audits
		 WHERE site_id = $1 AND content_gap_detected = true`,
		siteID,
	).Scan(&stats.ContentGaps)
	if err != nil {
		return nil, fmt.Errorf("failed to get content gap count: %w", err)
	}

	err = p.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*),0) FROM seo_clusters WHERE site_id = $1`,
		siteID,
	).Scan(&stats.ClustersCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster count: %w", err)
	}

	rows, err := p.Query(ctx,
		`SELECT language, COUNT(*) FROM seo_projects WHERE site_id = $1 GROUP BY language`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get language stats: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var lang string
		var count int
		if err := rows.Scan(&lang, &count); err == nil {
			stats.ByLanguage[lang] = count
		}
	}

	return stats, nil
}

func (s *Service) GetMetrics(ctx context.Context, siteID uuid.UUID) (*SEOMetrics, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	metrics := &SEOMetrics{
		ByStatus:   make(map[ProjectStatus]int64),
		ByLanguage: make(map[string]int64),
		ByCategory: make(map[string]int64),
	}

	rows, err := p.Query(ctx,
		`SELECT status, COUNT(*) FROM seo_projects WHERE site_id = $1 GROUP BY status`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get status metrics: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err == nil {
			metrics.ByStatus[ProjectStatus(status)] = count
		}
	}

	rows2, err := p.Query(ctx,
		`SELECT language, COUNT(*) FROM seo_projects WHERE site_id = $1 GROUP BY language`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get language metrics: %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var lang string
		var count int64
		if err := rows2.Scan(&lang, &count); err == nil {
			metrics.ByLanguage[lang] = count
		}
	}

	rows3, err := p.Query(ctx,
		`SELECT category, COUNT(*) FROM seo_improvements WHERE site_id = $1 GROUP BY category`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get category metrics: %w", err)
	}
	defer rows3.Close()
	for rows3.Next() {
		var cat string
		var count int64
		if err := rows3.Scan(&cat, &count); err == nil {
			metrics.ByCategory[cat] = count
		}
	}

	return metrics, nil
}

// --- Publish Gate ---

// gateInput builds the analyzer input exactly like the publish gate evaluates
// it: page-level rendering (title becomes H1 when the body has none), keyword
// derived deterministically when absent, and the configured default author as
// the EEAT byline fallback. Deterministic and shared by the gate, the
// detailed gate, and the review report.
func (s *Service) gateInput(in publisher.PublishGateInput) ArticleAnalysisInput {
	input := ArticleAnalysisInput{
		Title:           in.Title,
		MetaDescription: in.MetaDescription,
		Content:         pageContent(in.Title, in.Content),
		Keyword:         in.Keyword,
		Language:        in.Language,
		AuthorName:      in.AuthorName,
	}
	if input.Keyword == "" {
		input.Keyword = FocusKeyword(in.Title, in.Content)
	}
	if input.AuthorName == "" {
		// Fallback: a configurable default byline so EEAT always sees an
		// author even when the pipeline did not carry one. Never fabricated
		// per-article — a single site-level value.
		input.AuthorName = s.defaultAuthor
	}
	if input.Language == "" {
		input.Language = "pt"
	}
	return input
}

// CheckPublishScore implements publisher.PublishGate. It returns the last
// stored audit score for the linked post, or evaluates the content inline
// (deterministically) when no stored audit exists.
func (s *Service) CheckPublishScore(ctx context.Context, in publisher.PublishGateInput) (float64, error) {
	if in.PostID != nil {
		p, err := s.pool()
		if err == nil {
			var score float64
			qerr := p.QueryRow(ctx,
				`SELECT seo_score FROM posts WHERE id = $1 AND site_id = $2 AND seo_analyzed_at IS NOT NULL`,
				in.PostID, in.SiteID,
			).Scan(&score)
			if qerr == nil {
				return score, nil
			}
			if qerr != pgx.ErrNoRows {
				s.log.Warn("publish gate: failed to read stored seo score", "error", qerr)
			}
		}
	}

	input := s.gateInput(in)
	analysis := AnalyzeArticle(ctx, input, s.qualityChecker)
	logScoreBreakdown(s.log, in.SiteID, in.Title, input.Keyword, analysis)
	return analysis.OverallScore, nil
}

// CheckPublishScoreWithIssues implements publisher.PublishGateDetailer: same
// evaluation as CheckPublishScore plus a compact, human-readable list of the
// audit dimensions that keep the score below the publish minimum. Deterministic.
func (s *Service) CheckPublishScoreWithIssues(ctx context.Context, in publisher.PublishGateInput) (float64, []string, error) {
	if in.PostID != nil {
		p, err := s.pool()
		if err == nil {
			var score float64
			var issuesJSON string
			qerr := p.QueryRow(ctx,
				`SELECT seo_score, COALESCE(seo_issues::text, '[]') FROM posts WHERE id = $1 AND site_id = $2 AND seo_analyzed_at IS NOT NULL`,
				in.PostID, in.SiteID,
			).Scan(&score, &issuesJSON)
			if qerr == nil {
				var stored []AuditIssue
				if json.Unmarshal([]byte(issuesJSON), &stored) == nil {
					issues := make([]string, 0, len(stored))
					for _, is := range stored {
						if is.Issue != "" {
							issues = append(issues, is.Issue)
						}
					}
					if len(issues) > 6 {
						issues = issues[:6]
					}
					return score, issues, nil
				}
				return score, nil, nil
			}
			if qerr != pgx.ErrNoRows {
				s.log.Warn("publish gate: failed to read stored seo score", "error", qerr)
			}
		}
	}

	input := s.gateInput(in)
	analysis := AnalyzeArticle(ctx, input, s.qualityChecker)
	return analysis.OverallScore, buildGateIssues(analysis), nil
}

// ReviewSEO implements publisher.PublishGateReviewer: the full per-dimension
// breakdown of the publish gate evaluation for the editorial review screen.
// Same evaluation path as the gate (page-level rendering, deterministic
// keyword, default author, same threshold), fail-open.
func (s *Service) ReviewSEO(ctx context.Context, in publisher.PublishGateInput) (*publisher.SEOReviewReport, error) {
	if s == nil {
		return nil, nil
	}
	input := s.gateInput(in)
	analysis := AnalyzeArticle(ctx, input, s.qualityChecker)
	if analysis == nil {
		return nil, nil
	}
	minScore := s.minPublishScore
	if minScore <= 0 {
		minScore = 70
	}
	return &publisher.SEOReviewReport{
		Score:          analysis.OverallScore,
		MinScore:       minScore,
		Passes:         analysis.OverallScore >= minScore,
		Title:          analysis.TitleScore,
		Meta:           analysis.MetaScore,
		Headings:       analysis.HeadingScore,
		Keyword:        analysis.KeywordScore,
		Readability:    analysis.ReadabilityScore,
		InternalLinks:  analysis.InternalLinksScore,
		ExternalLinks:  analysis.ExternalLinksScore,
		EEAT:           analysis.EEATScore,
		Images:         analysis.ImagesScore,
		KeywordDensity: analysis.KeywordDensity,
		WordCount:      analysis.WordCount,
		Issues:         buildGateIssues(analysis),
	}, nil
}

// buildGateIssues reports the weighted dimensions scoring below 60/100 (the
// likely culprits of a blocked publication), newest-failing first, capped at 6.
// Deterministic and language-neutral ("images: 30/100").
func buildGateIssues(a *ArticleAnalysis) []string {
	if a == nil {
		return nil
	}
	type dim struct {
		name  string
		score float64
	}
	all := []dim{
		{"title", a.TitleScore},
		{"meta_description", a.MetaScore},
		{"headings", a.HeadingScore},
		{"keyword", a.KeywordScore},
		{"readability", a.ReadabilityScore},
		{"internal_links", a.InternalLinksScore},
		{"external_links", a.ExternalLinksScore},
		{"eeat", a.EEATScore},
		{"images", a.ImagesScore},
		{"slug", a.SlugScore},
		{"schema", a.SchemaScore},
	}
	issues := []string{}
	for _, d := range all {
		if d.score < 60 {
			issues = append(issues, fmt.Sprintf("%s: %.0f/100", d.name, d.score))
		}
	}
	if a.DuplicateCount > 0 {
		issues = append(issues, fmt.Sprintf("duplicates: %d block(s) found", a.DuplicateCount))
	}
	if len(issues) > 6 {
		issues = issues[:6]
	}
	return issues
}

// pageContent returns the content text as the published page actually renders
// it: when the article body has no H1, the page shows the post title as the
// H1 heading — the analyzer consumes that page-equivalent text so heading
// scoring reflects the real page, not an artifact of the storage format.
func pageContent(title, content string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}
	if markdownH1RE.MatchString(content) || htmlH1RE.MatchString(content) {
		return content
	}
	return "# " + strings.TrimSpace(title) + "\n\n" + content
}

// logScoreBreakdown emits the structured per-dimension SEO score so a blocked
// publication can be diagnosed precisely (Sprint 6.8): every dimension of the
// weighted score plus the top audit issues. Same input → same output; this is
// pure observability, no behavior change.
func logScoreBreakdown(log *logger.Logger, siteID uuid.UUID, title, keyword string, a *ArticleAnalysis) {
	if log == nil || a == nil {
		return
	}
	issues := make([]string, 0, 8)
	for _, is := range a.Suggestions {
		if is.Issue != "" {
			issues = append(issues, is.Issue)
			if len(issues) >= 8 {
				break
			}
		}
	}
	log.Info("publish gate: seo score breakdown",
		"site_id", siteID,
		"title", title,
		"keyword", keyword,
		"title_score", round2(a.TitleScore),
		"meta_score", round2(a.MetaScore),
		"headings_score", round2(a.HeadingScore),
		"keyword_score", round2(a.KeywordScore),
		"readability_score", round2(a.ReadabilityScore),
		"internal_links_score", round2(a.InternalLinksScore),
		"external_links_score", round2(a.ExternalLinksScore),
		"eeat_score", round2(a.EEATScore),
		"images_score", round2(a.ImagesScore),
		"overall_score", round2(a.OverallScore),
		"word_count", a.WordCount,
		"keyword_density", a.KeywordDensity,
		"internal_links", a.InternalLinks,
		"external_links", a.ExternalLinks,
		"images_with_alt", a.ImagesWithAlt,
		"issues", issues,
	)
}

// --- Helper ---

// stableScore returns a deterministic value in [0,1] derived from the input
// string (stable pseudo-random, no math/rand).
func stableScore(key string) float64 {
	h := fnv64(key)
	return float64(h%1000) / 1000.0
}

func avgKeywordScore(keywords []SEOKeyword) float64 {
	if len(keywords) == 0 {
		return 0
	}
	sum := 0.0
	for _, k := range keywords {
		sum += k.TopicalRelevance
	}
	return sum / float64(len(keywords))
}



var _ publisher.PublishGate = (*Service)(nil)
var _ publisher.PublishGateDetailer = (*Service)(nil)
var _ publisher.PublishGateReviewer = (*Service)(nil)
