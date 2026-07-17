package editorialengine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"nexora/internal/kernel"
	"nexora/internal/pkg/audit"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type Service struct {
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	eventBus *kernel.EventBus
	auditLog *audit.Logger
}

func NewService(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *Service {
	var pool database.Pool
	if db != nil {
		pool = db.Pool
	}
	return &Service{
		log:      log,
		db:       db,
		cache:    ch,
		auditLog: audit.New(pool, log),
	}
}

func (s *Service) SetEventBus(bus *kernel.EventBus) {
	s.eventBus = bus
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

// --- Pipeline ---

func (s *Service) CreatePipeline(ctx context.Context, siteID uuid.UUID, req CreatePipelineRequest) (*EditorialPipeline, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, _ := s.GetPipelineByJob(ctx, siteID, req.ArticleJobID)
	if existing != nil {
		return nil, ErrJobAlreadyInPipeline
	}

	now := time.Now()
	pipelineID := uuid.New()

	_, err = p.Exec(ctx,
		`INSERT INTO editorial_pipelines (id, article_job_id, site_id, current_stage, status, started_at, created_at, updated_at)
		 VALUES ($1,$2,$3,'research','in_progress',$4,$5,$5)`,
		pipelineID, req.ArticleJobID, siteID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create editorial pipeline: %w", err)
	}

	for _, stage := range ValidStages {
		stageID := uuid.New()
		stageStatus := "pending"
		stageStarted := (*time.Time)(nil)
		if stage == StageResearch {
			stageStatus = "in_progress"
			stageStarted = &now
		}
		_, err = p.Exec(ctx,
			`INSERT INTO pipeline_stages (id, pipeline_id, stage, status, started_at, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$6)`,
			stageID, pipelineID, string(stage), stageStatus, stageStarted, now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create pipeline stage %s: %w", stage, err)
		}
	}

	pipeline := &EditorialPipeline{
		ID:           pipelineID,
		ArticleJobID: req.ArticleJobID,
		SiteID:       siteID,
		CurrentStage: StageResearch,
		Status:       StageStatusInProgress,
		StartedAt:    &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		Action:     audit.Action("editorial.pipeline.created"),
		EntityType: "editorial_pipeline",
		EntityID:   &pipelineID,
		Payload:    map[string]interface{}{"article_job_id": req.ArticleJobID.String()},
	})

	s.fireEvent(ctx, EventEditorialStarted, map[string]interface{}{
		"pipeline_id":   pipelineID.String(),
		"article_job_id": req.ArticleJobID.String(),
		"site_id":       siteID.String(),
	}, siteID)

	return pipeline, nil
}

func (s *Service) GetPipeline(ctx context.Context, siteID, pipelineID uuid.UUID) (*PipelineDetail, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var pl EditorialPipeline
	err = p.QueryRow(ctx,
		`SELECT id, article_job_id, site_id, current_stage, status, started_at, completed_at, created_at, updated_at
		 FROM editorial_pipelines WHERE id = $1 AND site_id = $2`,
		pipelineID, siteID,
	).Scan(&pl.ID, &pl.ArticleJobID, &pl.SiteID, &pl.CurrentStage, &pl.Status,
		&pl.StartedAt, &pl.CompletedAt, &pl.CreatedAt, &pl.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPipelineNotFound
		}
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	stages, err := s.ListPipelineStages(ctx, pipelineID)
	if err != nil {
		return nil, err
	}

	return &PipelineDetail{
		EditorialPipeline: pl,
		Stages:            stages,
	}, nil
}

func (s *Service) GetPipelineByJob(ctx context.Context, siteID, articleJobID uuid.UUID) (*EditorialPipeline, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var pl EditorialPipeline
	err = p.QueryRow(ctx,
		`SELECT id, article_job_id, site_id, current_stage, status, started_at, completed_at, created_at, updated_at
		 FROM editorial_pipelines WHERE article_job_id = $1 AND site_id = $2`,
		articleJobID, siteID,
	).Scan(&pl.ID, &pl.ArticleJobID, &pl.SiteID, &pl.CurrentStage, &pl.Status,
		&pl.StartedAt, &pl.CompletedAt, &pl.CreatedAt, &pl.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPipelineNotFound
		}
		return nil, fmt.Errorf("failed to get pipeline by job: %w", err)
	}

	return &pl, nil
}

func (s *Service) ListPipelines(ctx context.Context, siteID uuid.UUID, stage, status string, limit, offset int) ([]EditorialPipeline, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2

	if stage != "" {
		where = append(where, fmt.Sprintf("current_stage = $%d", argIdx))
		args = append(args, stage)
		argIdx++
	}
	if status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT id, article_job_id, site_id, current_stage, status, started_at, completed_at, created_at, updated_at
		 FROM editorial_pipelines WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}
	defer rows.Close()

	var pipelines []EditorialPipeline
	for rows.Next() {
		var pl EditorialPipeline
		if err := rows.Scan(&pl.ID, &pl.ArticleJobID, &pl.SiteID, &pl.CurrentStage, &pl.Status,
			&pl.StartedAt, &pl.CompletedAt, &pl.CreatedAt, &pl.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan pipeline: %w", err)
		}
		pipelines = append(pipelines, pl)
	}
	if pipelines == nil {
		pipelines = []EditorialPipeline{}
	}
	return pipelines, nil
}

func (s *Service) UpdatePipeline(ctx context.Context, siteID, pipelineID uuid.UUID, req UpdatePipelineRequest) (*PipelineDetail, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, err := s.GetPipeline(ctx, siteID, pipelineID)
	if err != nil {
		return nil, err
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1
	now := time.Now()

	if req.CurrentStage != nil {
		valid := false
		for _, vs := range ValidStages {
			if *req.CurrentStage == vs {
				valid = true
				break
			}
		}
		if !valid {
			return nil, ErrInvalidStage
		}
		setClauses = append(setClauses, fmt.Sprintf("current_stage = $%d", argIdx))
		args = append(args, string(*req.CurrentStage))
		argIdx++

		setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
		args = append(args, now)
		argIdx++

		_, err = p.Exec(ctx,
			`UPDATE pipeline_stages SET status = 'completed', completed_at = $1, updated_at = $1
			 WHERE pipeline_id = $2 AND stage = $3 AND status = 'in_progress'`,
			now, pipelineID, string(existing.EditorialPipeline.CurrentStage),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to complete current stage: %w", err)
		}

		_, err = p.Exec(ctx,
			`UPDATE pipeline_stages SET status = 'in_progress', started_at = $1, updated_at = $1
			 WHERE pipeline_id = $2 AND stage = $3`,
			now, pipelineID, string(*req.CurrentStage),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to start next stage: %w", err)
		}

		s.fireEvent(ctx, EventEditorialReviewed, map[string]interface{}{
			"pipeline_id": pipelineID.String(),
			"stage":       string(*req.CurrentStage),
		}, siteID)
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*req.Status))
		argIdx++
		if *req.Status == StageStatusCompleted {
			setClauses = append(setClauses, fmt.Sprintf("completed_at = $%d", argIdx))
			args = append(args, now)
			argIdx++
			s.fireEvent(ctx, EventEditorialCompleted, map[string]interface{}{
				"pipeline_id": pipelineID.String(),
			}, siteID)
		}
	}

	if len(setClauses) == 0 {
		return s.GetPipeline(ctx, siteID, pipelineID)
	}

	if !strings.Contains(strings.Join(setClauses, " "), "updated_at") {
		setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
		args = append(args, now)
		argIdx++
	}

	query := fmt.Sprintf(
		`UPDATE editorial_pipelines SET %s WHERE id = $%d AND site_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, pipelineID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update pipeline: %w", err)
	}

	return s.GetPipeline(ctx, siteID, pipelineID)
}

func (s *Service) ListPipelineStages(ctx context.Context, pipelineID uuid.UUID) ([]PipelineStageItem, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, pipeline_id, stage, status, started_at, completed_at, assigned_to,
		        COALESCE(notes,''), COALESCE(metadata::text,'{}'), created_at, updated_at
		 FROM pipeline_stages WHERE pipeline_id = $1 ORDER BY created_at ASC`,
		pipelineID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipeline stages: %w", err)
	}
	defer rows.Close()

	var stages []PipelineStageItem
	for rows.Next() {
		var s PipelineStageItem
		var metadataStr string
		if err := rows.Scan(&s.ID, &s.PipelineID, &s.Stage, &s.Status,
			&s.StartedAt, &s.CompletedAt, &s.AssignedTo, &s.Notes, &metadataStr,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan pipeline stage: %w", err)
		}
		if len(metadataStr) > 0 && metadataStr != "{}" {
			_ = json.Unmarshal([]byte(metadataStr), &s.Metadata)
		}
		if s.Metadata == nil {
			s.Metadata = make(map[string]interface{})
		}
		stages = append(stages, s)
	}
	if stages == nil {
		stages = []PipelineStageItem{}
	}
	return stages, nil
}

func (s *Service) GetPipelineStage(ctx context.Context, pipelineID, stageID uuid.UUID) (*PipelineStageItem, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var stage PipelineStageItem
	var metadataStr string
	err = p.QueryRow(ctx,
		`SELECT id, pipeline_id, stage, status, started_at, completed_at, assigned_to,
		        COALESCE(notes,''), COALESCE(metadata::text,'{}'), created_at, updated_at
		 FROM pipeline_stages WHERE id = $1 AND pipeline_id = $2`,
		stageID, pipelineID,
	).Scan(&stage.ID, &stage.PipelineID, &stage.Stage, &stage.Status,
		&stage.StartedAt, &stage.CompletedAt, &stage.AssignedTo, &stage.Notes, &metadataStr,
		&stage.CreatedAt, &stage.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrStageNotFound
		}
		return nil, fmt.Errorf("failed to get pipeline stage: %w", err)
	}
	if len(metadataStr) > 0 && metadataStr != "{}" {
		_ = json.Unmarshal([]byte(metadataStr), &stage.Metadata)
	}
	if stage.Metadata == nil {
		stage.Metadata = make(map[string]interface{})
	}
	return &stage, nil
}

func (s *Service) UpdatePipelineStage(ctx context.Context, pipelineID, stageID uuid.UUID, req UpdateStageRequest) (*PipelineStageItem, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	_, err = s.GetPipelineStage(ctx, pipelineID, stageID)
	if err != nil {
		return nil, err
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1
	now := time.Now()

	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*req.Status))
		argIdx++
		if *req.Status == StageStatusInProgress {
			setClauses = append(setClauses, fmt.Sprintf("started_at = $%d", argIdx))
			args = append(args, now)
			argIdx++
		}
		if *req.Status == StageStatusCompleted {
			setClauses = append(setClauses, fmt.Sprintf("completed_at = $%d", argIdx))
			args = append(args, now)
			argIdx++
		}
	}
	if req.AssignedTo != nil {
		setClauses = append(setClauses, fmt.Sprintf("assigned_to = $%d", argIdx))
		args = append(args, *req.AssignedTo)
		argIdx++
	}
	if req.Notes != nil {
		setClauses = append(setClauses, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *req.Notes)
		argIdx++
	}
	if req.Metadata != nil {
		data, _ := json.Marshal(*req.Metadata)
		setClauses = append(setClauses, fmt.Sprintf("metadata = $%d::jsonb", argIdx))
		args = append(args, string(data))
		argIdx++
	}

	if len(setClauses) == 0 {
		return s.GetPipelineStage(ctx, pipelineID, stageID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE pipeline_stages SET %s WHERE id = $%d AND pipeline_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, stageID, pipelineID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update pipeline stage: %w", err)
	}

	return s.GetPipelineStage(ctx, pipelineID, stageID)
}

// --- Style Rules ---

func (s *Service) GetStyleRules(ctx context.Context, siteID uuid.UUID) (*EditorialStyleRules, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var rules EditorialStyleRules
	var headingStr string
	err = p.QueryRow(ctx,
		`SELECT id, site_id, COALESCE(brand_voice,''), COALESCE(tone,'neutral'),
		        COALESCE(language_level,'standard'), COALESCE(target_audience,''),
		        COALESCE(avg_word_count,800), COALESCE(heading_structure::text,'[]'),
		        COALESCE(prohibited_vocabulary,'{}'), COALESCE(required_expressions,'{}'),
		        COALESCE(personality,''), COALESCE(formality_degree,'neutral'),
		        created_at, updated_at
		 FROM editorial_style_rules WHERE site_id = $1`,
		siteID,
	).Scan(&rules.ID, &rules.SiteID, &rules.BrandVoice, &rules.Tone,
		&rules.LanguageLevel, &rules.TargetAudience,
		&rules.AvgWordCount, &headingStr,
		&rules.ProhibitedVocabulary, &rules.RequiredExpressions,
		&rules.Personality, &rules.FormalityDegree,
		&rules.CreatedAt, &rules.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrStyleRulesNotFound
		}
		return nil, fmt.Errorf("failed to get style rules: %w", err)
	}
	if len(headingStr) > 0 && headingStr != "[]" {
		_ = json.Unmarshal([]byte(headingStr), &rules.HeadingStructure)
	}
	if rules.HeadingStructure == nil {
		rules.HeadingStructure = []interface{}{}
	}
	return &rules, nil
}

func (s *Service) UpsertStyleRules(ctx context.Context, siteID uuid.UUID, req UpdateStyleRulesRequest) (*EditorialStyleRules, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.BrandVoice != nil {
		setClauses = append(setClauses, fmt.Sprintf("brand_voice = $%d", argIdx))
		args = append(args, *req.BrandVoice)
		argIdx++
	}
	if req.Tone != nil {
		setClauses = append(setClauses, fmt.Sprintf("tone = $%d", argIdx))
		args = append(args, *req.Tone)
		argIdx++
	}
	if req.LanguageLevel != nil {
		setClauses = append(setClauses, fmt.Sprintf("language_level = $%d", argIdx))
		args = append(args, *req.LanguageLevel)
		argIdx++
	}
	if req.TargetAudience != nil {
		setClauses = append(setClauses, fmt.Sprintf("target_audience = $%d", argIdx))
		args = append(args, *req.TargetAudience)
		argIdx++
	}
	if req.AvgWordCount != nil {
		setClauses = append(setClauses, fmt.Sprintf("avg_word_count = $%d", argIdx))
		args = append(args, *req.AvgWordCount)
		argIdx++
	}
	if req.HeadingStructure != nil {
		data, _ := json.Marshal(*req.HeadingStructure)
		setClauses = append(setClauses, fmt.Sprintf("heading_structure = $%d::jsonb", argIdx))
		args = append(args, string(data))
		argIdx++
	}
	if req.ProhibitedVocabulary != nil {
		setClauses = append(setClauses, fmt.Sprintf("prohibited_vocabulary = $%d", argIdx))
		args = append(args, *req.ProhibitedVocabulary)
		argIdx++
	}
	if req.RequiredExpressions != nil {
		setClauses = append(setClauses, fmt.Sprintf("required_expressions = $%d", argIdx))
		args = append(args, *req.RequiredExpressions)
		argIdx++
	}
	if req.Personality != nil {
		setClauses = append(setClauses, fmt.Sprintf("personality = $%d", argIdx))
		args = append(args, *req.Personality)
		argIdx++
	}
	if req.FormalityDegree != nil {
		setClauses = append(setClauses, fmt.Sprintf("formality_degree = $%d", argIdx))
		args = append(args, *req.FormalityDegree)
		argIdx++
	}

	existing, err := s.GetStyleRules(ctx, siteID)
	if err != nil && err != ErrStyleRulesNotFound {
		return nil, err
	}

	if existing != nil {
		if len(setClauses) == 0 {
			return existing, nil
		}
		setClauses = append(setClauses, "updated_at = NOW()")
		query := fmt.Sprintf(
			`UPDATE editorial_style_rules SET %s WHERE site_id = $%d`,
			strings.Join(setClauses, ", "), argIdx,
		)
		args = append(args, siteID)
		_, err = p.Exec(ctx, query, args...)
	} else {
		id := uuid.New()
		_, err = p.Exec(ctx,
			`INSERT INTO editorial_style_rules (id, site_id, brand_voice, tone, language_level,
			 target_audience, avg_word_count, heading_structure, prohibited_vocabulary,
			 required_expressions, personality, formality_degree, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13)`,
			id, siteID,
			strPtrOrDef(req.BrandVoice, ""),
			strPtrOrDef(req.Tone, "neutral"),
			strPtrOrDef(req.LanguageLevel, "standard"),
			strPtrOrDef(req.TargetAudience, ""),
			intPtrOrDef(req.AvgWordCount, 800),
			jsonOrEmpty(req.HeadingStructure),
			strSliceOrDef(req.ProhibitedVocabulary),
			strSliceOrDef(req.RequiredExpressions),
			strPtrOrDef(req.Personality, ""),
			strPtrOrDef(req.FormalityDegree, "neutral"),
			now,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to upsert style rules: %w", err)
	}

	s.fireEvent(ctx, EventStyleUpdated, map[string]interface{}{
		"site_id": siteID.String(),
	}, siteID)

	return s.GetStyleRules(ctx, siteID)
}

// --- SEO Data ---

func (s *Service) CreateSEOData(ctx context.Context, siteID, articleJobID uuid.UUID, req CreateSEODataRequest) (*SEOData, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	id := uuid.New()

	entitiesJSON, _ := json.Marshal(req.Entities)
	faqJSON, _ := json.Marshal(req.FAQ)
	schemaJSON, _ := json.Marshal(req.SchemaData)
	ogJSON, _ := json.Marshal(req.OGData)
	twitterJSON, _ := json.Marshal(req.TwitterCard)

	robots := req.Robots
	if robots == "" {
		robots = "index,follow"
	}

	_, err = p.Exec(ctx,
		`INSERT INTO editorial_seo_data (id, article_job_id, site_id, primary_keyword, secondary_keywords,
		 long_tail_keywords, entities, faq, schema_type, schema_data, meta_title, meta_description,
		 slug, canonical_url, robots, og_data, twitter_card, alt_text,
		 suggested_internal_links, suggested_external_links, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9,$10::jsonb,$11,$12,$13,$14,$15,$16::jsonb,$17::jsonb,$18,$19,$20,$21,$21)`,
		id, articleJobID, siteID, req.PrimaryKeyword, req.SecondaryKeywords,
		req.LongTailKeywords, string(entitiesJSON), string(faqJSON),
		req.SchemaType, string(schemaJSON), req.MetaTitle, req.MetaDescription,
		req.Slug, req.CanonicalURL, robots, string(ogJSON), string(twitterJSON),
		req.AltText, req.SuggestedInternalLinks, req.SuggestedExternalLinks, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create seo data: %w", err)
	}

	seo := &SEOData{
		ID:                    id,
		ArticleJobID:          articleJobID,
		SiteID:                siteID,
		PrimaryKeyword:        req.PrimaryKeyword,
		SecondaryKeywords:     req.SecondaryKeywords,
		LongTailKeywords:      req.LongTailKeywords,
		Entities:              req.Entities,
		FAQ:                   req.FAQ,
		SchemaType:            req.SchemaType,
		SchemaData:            req.SchemaData,
		MetaTitle:             req.MetaTitle,
		MetaDescription:       req.MetaDescription,
		Slug:                  req.Slug,
		CanonicalURL:          req.CanonicalURL,
		Robots:                robots,
		OGData:                req.OGData,
		TwitterCard:           req.TwitterCard,
		AltText:               req.AltText,
		SuggestedInternalLinks: req.SuggestedInternalLinks,
		SuggestedExternalLinks: req.SuggestedExternalLinks,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	s.fireEvent(ctx, EventSEOGenerated, map[string]interface{}{
		"seo_data_id":    id.String(),
		"article_job_id": articleJobID.String(),
		"site_id":        siteID.String(),
	}, siteID)

	return seo, nil
}

func (s *Service) GetSEOData(ctx context.Context, siteID, articleJobID uuid.UUID) (*SEOData, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var seo SEOData
	var entitiesStr, faqStr, schemaStr, ogStr, twitterStr string
	err = p.QueryRow(ctx,
		`SELECT id, article_job_id, site_id, COALESCE(primary_keyword,''),
		        COALESCE(secondary_keywords,'{}'), COALESCE(long_tail_keywords,'{}'),
		        COALESCE(entities::text,'[]'), COALESCE(faq::text,'[]'),
		        COALESCE(schema_type,''), COALESCE(schema_data::text,'{}'),
		        COALESCE(meta_title,''), COALESCE(meta_description,''),
		        COALESCE(slug,''), COALESCE(canonical_url,''),
		        COALESCE(robots,'index,follow'),
		        COALESCE(og_data::text,'{}'), COALESCE(twitter_card::text,'{}'),
		        COALESCE(alt_text,'{}'), COALESCE(suggested_internal_links,'{}'),
		        COALESCE(suggested_external_links,'{}'), created_at, updated_at
		 FROM editorial_seo_data WHERE article_job_id = $1 AND site_id = $2`,
		articleJobID, siteID,
	).Scan(&seo.ID, &seo.ArticleJobID, &seo.SiteID, &seo.PrimaryKeyword,
		&seo.SecondaryKeywords, &seo.LongTailKeywords,
		&entitiesStr, &faqStr,
		&seo.SchemaType, &schemaStr,
		&seo.MetaTitle, &seo.MetaDescription,
		&seo.Slug, &seo.CanonicalURL,
		&seo.Robots,
		&ogStr, &twitterStr,
		&seo.AltText, &seo.SuggestedInternalLinks, &seo.SuggestedExternalLinks,
		&seo.CreatedAt, &seo.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSEONotFound
		}
		return nil, fmt.Errorf("failed to get seo data: %w", err)
	}

	if len(entitiesStr) > 0 {
		_ = json.Unmarshal([]byte(entitiesStr), &seo.Entities)
	}
	if len(faqStr) > 0 {
		_ = json.Unmarshal([]byte(faqStr), &seo.FAQ)
	}
	if len(schemaStr) > 0 {
		_ = json.Unmarshal([]byte(schemaStr), &seo.SchemaData)
	}
	if len(ogStr) > 0 {
		_ = json.Unmarshal([]byte(ogStr), &seo.OGData)
	}
	if len(twitterStr) > 0 {
		_ = json.Unmarshal([]byte(twitterStr), &seo.TwitterCard)
	}
	if seo.Entities == nil {
		seo.Entities = []interface{}{}
	}
	if seo.FAQ == nil {
		seo.FAQ = []interface{}{}
	}
	if seo.SchemaData == nil {
		seo.SchemaData = make(map[string]interface{})
	}
	if seo.OGData == nil {
		seo.OGData = make(map[string]interface{})
	}
	if seo.TwitterCard == nil {
		seo.TwitterCard = make(map[string]interface{})
	}
	return &seo, nil
}

func (s *Service) UpdateSEOData(ctx context.Context, siteID, articleJobID uuid.UUID, req UpdateSEODataRequest) (*SEOData, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	_, err = s.GetSEOData(ctx, siteID, articleJobID)
	if err != nil {
		return nil, err
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.PrimaryKeyword != nil {
		setClauses = append(setClauses, fmt.Sprintf("primary_keyword = $%d", argIdx))
		args = append(args, *req.PrimaryKeyword)
		argIdx++
	}
	if req.SecondaryKeywords != nil {
		setClauses = append(setClauses, fmt.Sprintf("secondary_keywords = $%d", argIdx))
		args = append(args, *req.SecondaryKeywords)
		argIdx++
	}
	if req.LongTailKeywords != nil {
		setClauses = append(setClauses, fmt.Sprintf("long_tail_keywords = $%d", argIdx))
		args = append(args, *req.LongTailKeywords)
		argIdx++
	}
	if req.Entities != nil {
		data, _ := json.Marshal(*req.Entities)
		setClauses = append(setClauses, fmt.Sprintf("entities = $%d::jsonb", argIdx))
		args = append(args, string(data))
		argIdx++
	}
	if req.FAQ != nil {
		data, _ := json.Marshal(*req.FAQ)
		setClauses = append(setClauses, fmt.Sprintf("faq = $%d::jsonb", argIdx))
		args = append(args, string(data))
		argIdx++
	}
	if req.SchemaType != nil {
		setClauses = append(setClauses, fmt.Sprintf("schema_type = $%d", argIdx))
		args = append(args, *req.SchemaType)
		argIdx++
	}
	if req.SchemaData != nil {
		data, _ := json.Marshal(*req.SchemaData)
		setClauses = append(setClauses, fmt.Sprintf("schema_data = $%d::jsonb", argIdx))
		args = append(args, string(data))
		argIdx++
	}
	if req.MetaTitle != nil {
		setClauses = append(setClauses, fmt.Sprintf("meta_title = $%d", argIdx))
		args = append(args, *req.MetaTitle)
		argIdx++
	}
	if req.MetaDescription != nil {
		setClauses = append(setClauses, fmt.Sprintf("meta_description = $%d", argIdx))
		args = append(args, *req.MetaDescription)
		argIdx++
	}
	if req.Slug != nil {
		setClauses = append(setClauses, fmt.Sprintf("slug = $%d", argIdx))
		args = append(args, *req.Slug)
		argIdx++
	}
	if req.CanonicalURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("canonical_url = $%d", argIdx))
		args = append(args, *req.CanonicalURL)
		argIdx++
	}
	if req.Robots != nil {
		setClauses = append(setClauses, fmt.Sprintf("robots = $%d", argIdx))
		args = append(args, *req.Robots)
		argIdx++
	}
	if req.OGData != nil {
		data, _ := json.Marshal(*req.OGData)
		setClauses = append(setClauses, fmt.Sprintf("og_data = $%d::jsonb", argIdx))
		args = append(args, string(data))
		argIdx++
	}
	if req.TwitterCard != nil {
		data, _ := json.Marshal(*req.TwitterCard)
		setClauses = append(setClauses, fmt.Sprintf("twitter_card = $%d::jsonb", argIdx))
		args = append(args, string(data))
		argIdx++
	}
	if req.AltText != nil {
		setClauses = append(setClauses, fmt.Sprintf("alt_text = $%d", argIdx))
		args = append(args, *req.AltText)
		argIdx++
	}
	if req.SuggestedInternalLinks != nil {
		setClauses = append(setClauses, fmt.Sprintf("suggested_internal_links = $%d", argIdx))
		args = append(args, *req.SuggestedInternalLinks)
		argIdx++
	}
	if req.SuggestedExternalLinks != nil {
		setClauses = append(setClauses, fmt.Sprintf("suggested_external_links = $%d", argIdx))
		args = append(args, *req.SuggestedExternalLinks)
		argIdx++
	}

	if len(setClauses) == 0 {
		return s.GetSEOData(ctx, siteID, articleJobID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE editorial_seo_data SET %s WHERE article_job_id = $%d AND site_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, articleJobID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update seo data: %w", err)
	}

	return s.GetSEOData(ctx, siteID, articleJobID)
}

// --- Quality Scores ---

func (s *Service) CreateQualityScore(ctx context.Context, siteID, articleJobID uuid.UUID, req CreateQualityScoreRequest) (*QualityScore, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	id := uuid.New()

	dupJSON, _ := json.Marshal(req.DuplicateDetection)
	repJSON, _ := json.Marshal(req.RepetitionDetection)
	reportJSON, _ := json.Marshal(req.Report)

	overall := (req.SEOScore + req.ReadabilityScore + req.NaturalnessScore + req.EEATScore +
		req.HeadingStructure + req.InternalLinkingScore + req.ParagraphBalance) / 7.0

	_, err = p.Exec(ctx,
		`INSERT INTO editorial_quality_scores (id, article_job_id, site_id, seo_score, readability_score,
		 naturalness_score, eeat_score, keyword_density, heading_structure_score, internal_linking_score,
		 duplicate_detection, repetition_detection, passive_voice_count, avg_sentence_length,
		 paragraph_balance_score, overall_score, report, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12::jsonb,$13,$14,$15,$16,$17::jsonb,$18)`,
		id, articleJobID, siteID, req.SEOScore, req.ReadabilityScore,
		req.NaturalnessScore, req.EEATScore, req.KeywordDensity, req.HeadingStructure,
		req.InternalLinkingScore, string(dupJSON), string(repJSON),
		req.PassiveVoiceCount, req.AvgSentenceLength, req.ParagraphBalance,
		overall, string(reportJSON), now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create quality score: %w", err)
	}

	score := &QualityScore{
		ID:                  id,
		ArticleJobID:        articleJobID,
		SiteID:              siteID,
		SEOScore:            req.SEOScore,
		ReadabilityScore:    req.ReadabilityScore,
		NaturalnessScore:    req.NaturalnessScore,
		EEATScore:           req.EEATScore,
		KeywordDensity:      req.KeywordDensity,
		HeadingStructure:    req.HeadingStructure,
		InternalLinkingScore: req.InternalLinkingScore,
		DuplicateDetection:  req.DuplicateDetection,
		RepetitionDetection: req.RepetitionDetection,
		PassiveVoiceCount:   req.PassiveVoiceCount,
		AvgSentenceLength:   req.AvgSentenceLength,
		ParagraphBalance:    req.ParagraphBalance,
		OverallScore:        overall,
		Report:              req.Report,
		CreatedAt:           now,
	}

	s.fireEvent(ctx, EventQualityChecked, map[string]interface{}{
		"quality_id":     id.String(),
		"article_job_id": articleJobID.String(),
		"site_id":        siteID.String(),
		"overall_score":  overall,
	}, siteID)

	return score, nil
}

func (s *Service) GetQualityScore(ctx context.Context, siteID, articleJobID uuid.UUID) (*QualityScore, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var qs QualityScore
	var dupStr, repStr, reportStr string
	err = p.QueryRow(ctx,
		`SELECT id, article_job_id, site_id, seo_score, readability_score, naturalness_score,
		        eeat_score, keyword_density, heading_structure_score, internal_linking_score,
		        COALESCE(duplicate_detection::text,'[]'), COALESCE(repetition_detection::text,'[]'),
		        passive_voice_count, avg_sentence_length, paragraph_balance_score, overall_score,
		        COALESCE(report::text,'{}'), created_at
		 FROM editorial_quality_scores WHERE article_job_id = $1 AND site_id = $2
		 ORDER BY created_at DESC LIMIT 1`,
		articleJobID, siteID,
	).Scan(&qs.ID, &qs.ArticleJobID, &qs.SiteID, &qs.SEOScore, &qs.ReadabilityScore, &qs.NaturalnessScore,
		&qs.EEATScore, &qs.KeywordDensity, &qs.HeadingStructure, &qs.InternalLinkingScore,
		&dupStr, &repStr,
		&qs.PassiveVoiceCount, &qs.AvgSentenceLength, &qs.ParagraphBalance, &qs.OverallScore,
		&reportStr, &qs.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrQualityNotFound
		}
		return nil, fmt.Errorf("failed to get quality score: %w", err)
	}

	if len(dupStr) > 0 {
		_ = json.Unmarshal([]byte(dupStr), &qs.DuplicateDetection)
	}
	if len(repStr) > 0 {
		_ = json.Unmarshal([]byte(repStr), &qs.RepetitionDetection)
	}
	if len(reportStr) > 0 {
		_ = json.Unmarshal([]byte(reportStr), &qs.Report)
	}
	if qs.DuplicateDetection == nil {
		qs.DuplicateDetection = []interface{}{}
	}
	if qs.RepetitionDetection == nil {
		qs.RepetitionDetection = []interface{}{}
	}
	if qs.Report == nil {
		qs.Report = make(map[string]interface{})
	}
	return &qs, nil
}

func (s *Service) ListQualityScores(ctx context.Context, siteID, articleJobID uuid.UUID) ([]QualityScore, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, article_job_id, site_id, seo_score, readability_score, naturalness_score,
		        eeat_score, keyword_density, heading_structure_score, internal_linking_score,
		        COALESCE(duplicate_detection::text,'[]'), COALESCE(repetition_detection::text,'[]'),
		        passive_voice_count, avg_sentence_length, paragraph_balance_score, overall_score,
		        COALESCE(report::text,'{}'), created_at
		 FROM editorial_quality_scores WHERE article_job_id = $1 AND site_id = $2
		 ORDER BY created_at DESC`,
		articleJobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list quality scores: %w", err)
	}
	defer rows.Close()

	var scores []QualityScore
	for rows.Next() {
		var qs QualityScore
		var dupStr, repStr, reportStr string
		if err := rows.Scan(&qs.ID, &qs.ArticleJobID, &qs.SiteID, &qs.SEOScore, &qs.ReadabilityScore, &qs.NaturalnessScore,
			&qs.EEATScore, &qs.KeywordDensity, &qs.HeadingStructure, &qs.InternalLinkingScore,
			&dupStr, &repStr,
			&qs.PassiveVoiceCount, &qs.AvgSentenceLength, &qs.ParagraphBalance, &qs.OverallScore,
			&reportStr, &qs.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan quality score: %w", err)
		}
		if len(dupStr) > 0 {
			_ = json.Unmarshal([]byte(dupStr), &qs.DuplicateDetection)
		}
		if len(repStr) > 0 {
			_ = json.Unmarshal([]byte(repStr), &qs.RepetitionDetection)
		}
		if len(reportStr) > 0 {
			_ = json.Unmarshal([]byte(reportStr), &qs.Report)
		}
		if qs.DuplicateDetection == nil {
			qs.DuplicateDetection = []interface{}{}
		}
		if qs.RepetitionDetection == nil {
			qs.RepetitionDetection = []interface{}{}
		}
		if qs.Report == nil {
			qs.Report = make(map[string]interface{})
		}
		scores = append(scores, qs)
	}
	if scores == nil {
		scores = []QualityScore{}
	}
	return scores, nil
}

// --- Translation ---

func (s *Service) CreateTranslation(ctx context.Context, siteID, articleJobID uuid.UUID, req CreateTranslationRequest) (*EditorialTranslation, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	src := strings.ToLower(req.SourceLanguage)
	tgt := strings.ToLower(req.TargetLanguage)
	if src == tgt {
		return nil, ErrInvalidTranslationDir
	}
	if src != "pt" && src != "en" {
		return nil, ErrInvalidTranslationDir
	}
	if tgt != "pt" && tgt != "en" {
		return nil, ErrInvalidTranslationDir
	}

	now := time.Now()
	id := uuid.New()

	_, err = p.Exec(ctx,
		`INSERT INTO editorial_translations (id, article_job_id, site_id, source_language, target_language,
		 status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,'pending',$6,$6)`,
		id, articleJobID, siteID, src, tgt, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create translation: %w", err)
	}

	return &EditorialTranslation{
		ID:             id,
		ArticleJobID:   articleJobID,
		SiteID:         siteID,
		SourceLanguage: src,
		TargetLanguage: tgt,
		Status:         TransStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (s *Service) GetTranslation(ctx context.Context, siteID, translationID uuid.UUID) (*EditorialTranslation, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var t EditorialTranslation
	var metaStr, faqStr, entitiesStr string
	err = p.QueryRow(ctx,
		`SELECT id, article_job_id, site_id, source_language, target_language, status,
		        COALESCE(translated_slug,''), COALESCE(translated_meta::text,'{}'),
		        COALESCE(translated_faq::text,'[]'), COALESCE(translated_keywords,'{}'),
		        COALESCE(translated_entities::text,'[]'), completed_at, created_at, updated_at
		 FROM editorial_translations WHERE id = $1 AND site_id = $2`,
		translationID, siteID,
	).Scan(&t.ID, &t.ArticleJobID, &t.SiteID, &t.SourceLanguage, &t.TargetLanguage, &t.Status,
		&t.TranslatedSlug, &metaStr, &faqStr, &t.TranslatedKeywords, &entitiesStr,
		&t.CompletedAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTranslationNotFound
		}
		return nil, fmt.Errorf("failed to get translation: %w", err)
	}

	if len(metaStr) > 0 {
		_ = json.Unmarshal([]byte(metaStr), &t.TranslatedMeta)
	}
	if len(faqStr) > 0 {
		_ = json.Unmarshal([]byte(faqStr), &t.TranslatedFAQ)
	}
	if len(entitiesStr) > 0 {
		_ = json.Unmarshal([]byte(entitiesStr), &t.TranslatedEntities)
	}
	if t.TranslatedMeta == nil {
		t.TranslatedMeta = make(map[string]interface{})
	}
	if t.TranslatedFAQ == nil {
		t.TranslatedFAQ = []interface{}{}
	}
	if t.TranslatedEntities == nil {
		t.TranslatedEntities = []interface{}{}
	}
	return &t, nil
}

func (s *Service) ListTranslations(ctx context.Context, siteID, articleJobID uuid.UUID) ([]EditorialTranslation, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, article_job_id, site_id, source_language, target_language, status,
		        COALESCE(translated_slug,''), COALESCE(translated_meta::text,'{}'),
		        COALESCE(translated_faq::text,'[]'), COALESCE(translated_keywords,'{}'),
		        COALESCE(translated_entities::text,'[]'), completed_at, created_at, updated_at
		 FROM editorial_translations WHERE article_job_id = $1 AND site_id = $2
		 ORDER BY created_at DESC`,
		articleJobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list translations: %w", err)
	}
	defer rows.Close()

	var translations []EditorialTranslation
	for rows.Next() {
		var t EditorialTranslation
		var metaStr, faqStr, entitiesStr string
		if err := rows.Scan(&t.ID, &t.ArticleJobID, &t.SiteID, &t.SourceLanguage, &t.TargetLanguage, &t.Status,
			&t.TranslatedSlug, &metaStr, &faqStr, &t.TranslatedKeywords, &entitiesStr,
			&t.CompletedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan translation: %w", err)
		}
		if len(metaStr) > 0 {
			_ = json.Unmarshal([]byte(metaStr), &t.TranslatedMeta)
		}
		if len(faqStr) > 0 {
			_ = json.Unmarshal([]byte(faqStr), &t.TranslatedFAQ)
		}
		if len(entitiesStr) > 0 {
			_ = json.Unmarshal([]byte(entitiesStr), &t.TranslatedEntities)
		}
		if t.TranslatedMeta == nil {
			t.TranslatedMeta = make(map[string]interface{})
		}
		if t.TranslatedFAQ == nil {
			t.TranslatedFAQ = []interface{}{}
		}
		if t.TranslatedEntities == nil {
			t.TranslatedEntities = []interface{}{}
		}
		translations = append(translations, t)
	}
	if translations == nil {
		translations = []EditorialTranslation{}
	}
	return translations, nil
}

func (s *Service) UpdateTranslation(ctx context.Context, siteID, translationID uuid.UUID, req UpdateTranslationRequest) (*EditorialTranslation, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	_, err = s.GetTranslation(ctx, siteID, translationID)
	if err != nil {
		return nil, err
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1
	now := time.Now()

	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*req.Status))
		argIdx++
		if *req.Status == TransStatusCompleted {
			setClauses = append(setClauses, fmt.Sprintf("completed_at = $%d", argIdx))
			args = append(args, now)
			argIdx++
			s.fireEvent(ctx, EventEditorialTranslated, map[string]interface{}{
				"translation_id": translationID.String(),
			}, siteID)
		}
	}
	if req.TranslatedSlug != nil {
		setClauses = append(setClauses, fmt.Sprintf("translated_slug = $%d", argIdx))
		args = append(args, *req.TranslatedSlug)
		argIdx++
	}
	if req.TranslatedMeta != nil {
		data, _ := json.Marshal(*req.TranslatedMeta)
		setClauses = append(setClauses, fmt.Sprintf("translated_meta = $%d::jsonb", argIdx))
		args = append(args, string(data))
		argIdx++
	}
	if req.TranslatedFAQ != nil {
		data, _ := json.Marshal(*req.TranslatedFAQ)
		setClauses = append(setClauses, fmt.Sprintf("translated_faq = $%d::jsonb", argIdx))
		args = append(args, string(data))
		argIdx++
	}
	if req.TranslatedKeywords != nil {
		setClauses = append(setClauses, fmt.Sprintf("translated_keywords = $%d", argIdx))
		args = append(args, *req.TranslatedKeywords)
		argIdx++
	}
	if req.TranslatedEntities != nil {
		data, _ := json.Marshal(*req.TranslatedEntities)
		setClauses = append(setClauses, fmt.Sprintf("translated_entities = $%d::jsonb", argIdx))
		args = append(args, string(data))
		argIdx++
	}

	if len(setClauses) == 0 {
		return s.GetTranslation(ctx, siteID, translationID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE editorial_translations SET %s WHERE id = $%d AND site_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, translationID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update translation: %w", err)
	}

	return s.GetTranslation(ctx, siteID, translationID)
}

// --- Prompt Data ---

func (s *Service) GetPromptData(ctx context.Context, siteID, articleJobID uuid.UUID) (*PromptData, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var pd PromptData
	var briefingStr, styleStr, seoStr, outlineStr, entitiesStr string
	err = p.QueryRow(ctx,
		`SELECT id, article_job_id, site_id,
		        COALESCE(briefing::text,'{}'), COALESCE(style_rules::text,'{}'),
		        COALESCE(seo_rules::text,'{}'), COALESCE(tone,''),
		        COALESCE(outline::text,'[]'), COALESCE(entities::text,'[]'),
		        COALESCE(target_language,''), COALESCE(audience,''),
		        COALESCE(word_count,0), COALESCE(internal_links,'{}'),
		        COALESCE(constraints,'{}'), created_at, updated_at
		 FROM editorial_prompt_data WHERE article_job_id = $1 AND site_id = $2`,
		articleJobID, siteID,
	).Scan(&pd.ID, &pd.ArticleJobID, &pd.SiteID,
		&briefingStr, &styleStr, &seoStr, &pd.Tone,
		&outlineStr, &entitiesStr,
		&pd.TargetLanguage, &pd.Audience,
		&pd.WordCount, &pd.InternalLinks,
		&pd.Constraints, &pd.CreatedAt, &pd.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPromptDataNotFound
		}
		return nil, fmt.Errorf("failed to get prompt data: %w", err)
	}

	if len(briefingStr) > 0 {
		_ = json.Unmarshal([]byte(briefingStr), &pd.Briefing)
	}
	if len(styleStr) > 0 {
		_ = json.Unmarshal([]byte(styleStr), &pd.StyleRules)
	}
	if len(seoStr) > 0 {
		_ = json.Unmarshal([]byte(seoStr), &pd.SEORules)
	}
	if len(outlineStr) > 0 {
		_ = json.Unmarshal([]byte(outlineStr), &pd.Outline)
	}
	if len(entitiesStr) > 0 {
		_ = json.Unmarshal([]byte(entitiesStr), &pd.Entities)
	}
	if pd.Briefing == nil {
		pd.Briefing = make(map[string]interface{})
	}
	if pd.StyleRules == nil {
		pd.StyleRules = make(map[string]interface{})
	}
	if pd.SEORules == nil {
		pd.SEORules = make(map[string]interface{})
	}
	if pd.Outline == nil {
		pd.Outline = []interface{}{}
	}
	if pd.Entities == nil {
		pd.Entities = []interface{}{}
	}
	return &pd, nil
}

func (s *Service) CreatePromptData(ctx context.Context, siteID, articleJobID uuid.UUID) (*PromptData, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, _ := s.GetPromptData(ctx, siteID, articleJobID)
	if existing != nil {
		return existing, nil
	}

	now := time.Now()
	id := uuid.New()

	briefing := map[string]interface{}{"type": "structured"}
	styleRules := map[string]interface{}{"source": "editorial_style_rules"}
	seoRules := map[string]interface{}{"source": "editorial_seo_data"}

	briefingJSON, _ := json.Marshal(briefing)
	styleJSON, _ := json.Marshal(styleRules)
	seoJSON, _ := json.Marshal(seoRules)

	_, err = p.Exec(ctx,
		`INSERT INTO editorial_prompt_data (id, article_job_id, site_id, briefing, style_rules, seo_rules,
		 created_at, updated_at)
		 VALUES ($1,$2,$3,$4::jsonb,$5::jsonb,$6::jsonb,$7,$7)`,
		id, articleJobID, siteID, string(briefingJSON), string(styleJSON), string(seoJSON), now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt data: %w", err)
	}

	return &PromptData{
		ID:             id,
		ArticleJobID:   articleJobID,
		SiteID:         siteID,
		Briefing:       briefing,
		StyleRules:     styleRules,
		SEORules:       seoRules,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// --- Helpers ---

func strPtrOrDef(p *string, def string) string {
	if p != nil {
		return *p
	}
	return def
}

func intPtrOrDef(p *int, def int) int {
	if p != nil {
		return *p
	}
	return def
}

func jsonOrEmpty(p *[]interface{}) string {
	if p != nil {
		data, _ := json.Marshal(*p)
		return string(data)
	}
	return "[]"
}

func strSliceOrDef(p *[]string) []string {
	if p != nil {
		return *p
	}
	return []string{}
}
