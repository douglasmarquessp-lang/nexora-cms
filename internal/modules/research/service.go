package research

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"nexora/internal/ai"
	"nexora/internal/kernel"
	"nexora/internal/pkg/audit"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type Service struct {
	log       *logger.Logger
	db        *database.Database
	cache     *cache.Cache
	eventBus  *kernel.EventBus
	auditLog  *audit.Logger
	aiManager *ai.Manager
	cacheTTL  time.Duration
}

func (s *Service) SetAIManager(m *ai.Manager) {
	s.aiManager = m
}

// SetCacheTTL configures how long deep research results are cached
// (default 24h when unset/zero).
func (s *Service) SetCacheTTL(ttl time.Duration) {
	s.cacheTTL = ttl
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

// ExecuteGroundedResearch performs AI-powered grounded research using the configured AI provider.
// It returns research findings with source grounding metadata when available.
// Falls back gracefully when AI or grounding is unavailable, marking results as unverified.
func (s *Service) ExecuteGroundedResearch(ctx context.Context, topic, language string) (*ai.CompletionResult, error) {
	if s.aiManager == nil {
		// No AI available — return a minimal unverified result
		return &ai.CompletionResult{
			Content:      fmt.Sprintf("Research for topic '%s' (no AI configured)", topic),
			FinishReason: "unavailable",
			GroundingMetadata: &ai.GroundingMetadata{
				Unverified: true,
			},
		}, nil
	}

	// Check if any provider supports grounding
	req := ai.CompletionRequest{
		Prompt: fmt.Sprintf("Research the following topic thoroughly. Provide key facts, statistics, expert opinions, and recent developments:\n\nTopic: %s\n\nLanguage: %s", topic, language),
		System: "You are a research assistant. Provide well-researched, factual information. Cite your sources.",
	}

	// Enable grounding if the provider supports it
	providers := s.aiManager.Registry().FindByCapability(ai.CapGrounding)
	if len(providers) > 0 {
		req.Grounding = &ai.GroundingConfig{
			Enabled:    true,
			MaxSources: 10,
		}
	}

	result, err := s.aiManager.Generate(ctx, req)
	if err != nil {
		// Fallback: return unverified result rather than failing completely
		return &ai.CompletionResult{
			Content:      fmt.Sprintf("Research for topic '%s' (AI error: %s)", topic, err.Error()),
			FinishReason: "error",
			GroundingMetadata: &ai.GroundingMetadata{
				Unverified: true,
			},
		}, nil
	}

	// If grounding was requested but no grounding metadata returned, mark as unverified
	if req.Grounding != nil && req.Grounding.Enabled && result.GroundingMetadata == nil {
		result.GroundingMetadata = &ai.GroundingMetadata{
			Unverified: true,
		}
	}

	return result, nil
}

// SourcesFromGrounding converts AI grounding sources to ResearchSource models.
func (s *Service) SourcesFromGrounding(jobID uuid.UUID, gm *ai.GroundingMetadata) []ResearchSource {
	if gm == nil {
		return nil
	}

	var sources []ResearchSource
	for i, gs := range gm.Sources {
		now := time.Now()
		src := ResearchSource{
			Title:          gs.Title,
			URL:            gs.URI,
			FreshnessScore: gs.FreshnessScore,
			IsVerified:     gs.IsVerified,
			RetrievedAt:    &now,
			RelevanceScore: int(gs.FreshnessScore * 100),
			Position:       i + 1,
			GroundingMetadata: map[string]interface{}{
				"retrieved_at": now.Format(time.RFC3339),
				"domain_rank":  gs.DomainRank,
			},
		}
		if gs.PublishedAt != nil {
			src.PublishedAt = gs.PublishedAt
		}
		sources = append(sources, src)
	}
	return sources
}

// --- ArticleSource persistence ---

func (s *Service) SaveArticleSource(ctx context.Context, src ArticleSource) (*ArticleSource, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	src.ID = uuid.New()
	now := time.Now()
	src.CreatedAt = now

	var gmJSON []byte
	if src.GroundingMetadata != nil {
		gmJSON, _ = json.Marshal(src.GroundingMetadata)
	} else {
		gmJSON = []byte("{}")
	}

	_, err = p.Exec(ctx,
		`INSERT INTO article_sources
		 (id, site_id, article_id, pipeline_job_id, workflow_job_id, autocontent_job_id,
		  source_url, title, snippet, language, author, published_at, retrieved_at,
		  freshness_score, is_verified, domain_rank, relevance_score, grounding_metadata, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18::jsonb,$19)`,
		src.ID, src.SiteID, src.ArticleID, src.PipelineJobID, src.WorkflowJobID, src.AutocontentJobID,
		src.SourceURL, src.Title, src.Snippet, src.Language, src.Author,
		src.PublishedAt, src.RetrievedAt,
		src.FreshnessScore, src.IsVerified, src.DomainRank, src.RelevanceScore,
		string(gmJSON), now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save article source: %w", err)
	}

	return &src, nil
}

func (s *Service) SaveArticleSources(ctx context.Context, sources []ArticleSource) ([]ArticleSource, error) {
	var saved []ArticleSource
	for _, src := range sources {
		result, err := s.SaveArticleSource(ctx, src)
		if err != nil {
			return saved, err
		}
		saved = append(saved, *result)
	}
	if saved == nil {
		saved = []ArticleSource{}
	}
	return saved, nil
}

func (s *Service) GetArticleSources(ctx context.Context, siteID uuid.UUID, opts map[string]interface{}) ([]ArticleSource, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, site_id, article_id, pipeline_job_id, workflow_job_id, autocontent_job_id,
	                  source_url, COALESCE(title,''), COALESCE(snippet,''), COALESCE(language,''),
	                  COALESCE(author,''), published_at, retrieved_at,
	                  COALESCE(freshness_score,0), COALESCE(is_verified,false),
	                  COALESCE(domain_rank,0), COALESCE(relevance_score,0),
	                  COALESCE(grounding_metadata::text,'{}'), created_at
	           FROM article_sources WHERE site_id = $1`
	args := []interface{}{siteID}
	argIdx := 2

	if articleID, ok := opts["article_id"].(uuid.UUID); ok {
		query += fmt.Sprintf(" AND article_id = $%d", argIdx)
		args = append(args, articleID)
		argIdx++
	}
	if pipelineID, ok := opts["pipeline_job_id"].(uuid.UUID); ok {
		query += fmt.Sprintf(" AND pipeline_job_id = $%d", argIdx)
		args = append(args, pipelineID)
		argIdx++
	}
	if workflowID, ok := opts["workflow_job_id"].(uuid.UUID); ok {
		query += fmt.Sprintf(" AND workflow_job_id = $%d", argIdx)
		args = append(args, workflowID)
		argIdx++
	}
	if verified, ok := opts["is_verified"].(bool); ok {
		query += fmt.Sprintf(" AND is_verified = $%d", argIdx)
		args = append(args, verified)
		argIdx++
	}

	query += " ORDER BY relevance_score DESC, freshness_score DESC"

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list article sources: %w", err)
	}
	defer rows.Close()

	var sources []ArticleSource
	for rows.Next() {
		var s ArticleSource
		var gmStr string
		if err := rows.Scan(&s.ID, &s.SiteID, &s.ArticleID, &s.PipelineJobID, &s.WorkflowJobID, &s.AutocontentJobID,
			&s.SourceURL, &s.Title, &s.Snippet, &s.Language, &s.Author,
			&s.PublishedAt, &s.RetrievedAt, &s.FreshnessScore, &s.IsVerified,
			&s.DomainRank, &s.RelevanceScore, &gmStr, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan article source: %w", err)
		}
		if gmStr != "" && gmStr != "{}" {
			_ = json.Unmarshal([]byte(gmStr), &s.GroundingMetadata)
		}
		sources = append(sources, s)
	}
	if sources == nil {
		sources = []ArticleSource{}
	}
	return sources, nil
}

// GetFactBase returns the persisted fact base entries for a research job.
func (s *Service) GetFactBase(ctx context.Context, siteID, jobID uuid.UUID) ([]FactBaseEntry, error) {
	if siteID != uuid.Nil {
		if _, err := s.GetJob(ctx, siteID, jobID); err != nil {
			return nil, err
		}
	}
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, site_id, research_job_id, fact_type, entity, value,
		        COALESCE(source_url, ''), confidence, created_at
		 FROM research_fact_base WHERE research_job_id = $1 ORDER BY created_at, id`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list fact base: %w", err)
	}
	defer rows.Close()

	var facts []FactBaseEntry
	for rows.Next() {
		var f FactBaseEntry
		var factType string
		if err := rows.Scan(&f.ID, &f.SiteID, &f.ResearchJobID, &factType, &f.Entity, &f.Value,
			&f.SourceURL, &f.Confidence, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan fact base entry: %w", err)
		}
		f.FactType = FactType(factType)
		facts = append(facts, f)
	}
	return facts, rows.Err()
}

func (s *Service) pool() (database.Pool, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, ErrDatabaseNotAvail
	}
	return s.db.Pool, nil
}

func (s *Service) CreateJob(ctx context.Context, siteID, userID uuid.UUID, req CreateResearchJobRequest) (*ResearchJob, error) {
	if req.Topic == "" {
		return nil, ErrTopicRequired
	}

	lang := strings.ToLower(req.Language)
	if lang != "pt" && lang != "en" {
		return nil, ErrInvalidLanguage
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	jobID := uuid.New()

	_, err = p.Exec(ctx,
		`INSERT INTO research_jobs (id, site_id, topic, language, category, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7)`,
		jobID, siteID, req.Topic, lang, req.Category, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create research job: %w", err)
	}

	job := &ResearchJob{
		ID:       jobID,
		SiteID:   siteID,
		Topic:    req.Topic,
		Language: lang,
		Category: req.Category,
		Status:   JobStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("research.created"),
		EntityType: "research_job",
		EntityID:   &jobID,
		Payload:    map[string]interface{}{"topic": req.Topic, "language": lang},
	})

	s.fireEvent(ctx, EventResearchCreated, map[string]interface{}{
		"job_id":   jobID.String(),
		"site_id":  siteID.String(),
		"topic":    req.Topic,
		"language": lang,
	}, siteID)

	return job, nil
}

func (s *Service) GetJob(ctx context.Context, siteID, jobID uuid.UUID) (*ResearchJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var j ResearchJob
	err = p.QueryRow(ctx,
		`SELECT id, site_id, topic, language, COALESCE(category, ''), status,
		        COALESCE(sources_count, 0), COALESCE(error_message, ''), completed_at, created_at, updated_at
		 FROM research_jobs WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		jobID, siteID,
	).Scan(&j.ID, &j.SiteID, &j.Topic, &j.Language, &j.Category, &j.Status,
		&j.SourcesCount, &j.ErrorMessage, &j.CompletedAt, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrResearchJobNotFound
		}
		return nil, fmt.Errorf("failed to get research job: %w", err)
	}

	return &j, nil
}

func (s *Service) GetJobDetail(ctx context.Context, siteID, jobID uuid.UUID) (*ResearchJobDetail, error) {
	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	detail := &ResearchJobDetail{ResearchJob: *job}

	sources, err := s.listSources(ctx, jobID)
	if err != nil {
		return nil, err
	}
	detail.Sources = sources

	entities, err := s.listEntities(ctx, jobID)
	if err != nil {
		return nil, err
	}
	detail.Entities = entities

	briefing, err := s.GetBriefing(ctx, siteID, jobID)
	if err != nil && err != ErrBriefingNotFound {
		return nil, err
	}
	detail.Briefing = briefing

	return detail, nil
}

func (s *Service) ListJobs(ctx context.Context, siteID uuid.UUID, status JobStatus) ([]ResearchJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var rows pgx.Rows
	if status == "" {
		rows, err = p.Query(ctx,
			`SELECT id, site_id, topic, language, COALESCE(category, ''), status,
			        COALESCE(sources_count, 0), COALESCE(error_message, ''), completed_at, created_at, updated_at
			 FROM research_jobs WHERE site_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`,
			siteID,
		)
	} else {
		rows, err = p.Query(ctx,
			`SELECT id, site_id, topic, language, COALESCE(category, ''), status,
			        COALESCE(sources_count, 0), COALESCE(error_message, ''), completed_at, created_at, updated_at
			 FROM research_jobs WHERE site_id = $1 AND status = $2 AND deleted_at IS NULL ORDER BY created_at DESC`,
			siteID, string(status),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list research jobs: %w", err)
	}
	defer rows.Close()

	var jobs []ResearchJob
	for rows.Next() {
		var j ResearchJob
		if err := rows.Scan(&j.ID, &j.SiteID, &j.Topic, &j.Language, &j.Category, &j.Status,
			&j.SourcesCount, &j.ErrorMessage, &j.CompletedAt, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan research job: %w", err)
		}
		jobs = append(jobs, j)
	}
	if jobs == nil {
		jobs = []ResearchJob{}
	}
	return jobs, nil
}

func (s *Service) SearchByTopic(ctx context.Context, siteID uuid.UUID, query string) ([]ResearchJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, site_id, topic, language, COALESCE(category, ''), status,
		        COALESCE(sources_count, 0), COALESCE(error_message, ''), completed_at, created_at, updated_at
		 FROM research_jobs WHERE site_id = $1 AND deleted_at IS NULL
		 AND (topic ILIKE $2 OR category ILIKE $2)
		 ORDER BY created_at DESC`,
		siteID, "%"+query+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search research jobs: %w", err)
	}
	defer rows.Close()

	var jobs []ResearchJob
	for rows.Next() {
		var j ResearchJob
		if err := rows.Scan(&j.ID, &j.SiteID, &j.Topic, &j.Language, &j.Category, &j.Status,
			&j.SourcesCount, &j.ErrorMessage, &j.CompletedAt, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan research job: %w", err)
		}
		jobs = append(jobs, j)
	}
	if jobs == nil {
		jobs = []ResearchJob{}
	}
	return jobs, nil
}

func (s *Service) UpdateJob(ctx context.Context, siteID, jobID uuid.UUID, req UpdateResearchJobRequest) (*ResearchJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*req.Status))
		argIdx++

		if *req.Status == JobStatusCompleted {
			setClauses = append(setClauses, fmt.Sprintf("completed_at = $%d", argIdx))
			args = append(args, time.Now())
			argIdx++
		}
	}
	if req.Topic != nil {
		setClauses = append(setClauses, fmt.Sprintf("topic = $%d", argIdx))
		args = append(args, *req.Topic)
		argIdx++
	}
	if req.Category != nil {
		setClauses = append(setClauses, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *req.Category)
		argIdx++
	}

	if len(setClauses) == 0 {
		return existing, nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE research_jobs SET %s WHERE id = $%d AND site_id = $%d AND deleted_at IS NULL`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, jobID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update research job: %w", err)
	}

	evPayload := map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
	}
	if req.Status != nil {
		evPayload["status"] = string(*req.Status)
	}

	s.fireEvent(ctx, EventResearchUpdated, evPayload, siteID)

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) DeleteJob(ctx context.Context, siteID, jobID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	_, err = s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx,
		`UPDATE research_jobs SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		jobID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete research job: %w", err)
	}

	s.fireEvent(ctx, EventResearchDeleted, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
	}, siteID)

	return nil
}

func (s *Service) AddSource(ctx context.Context, jobID uuid.UUID, source ResearchSource) (*ResearchSource, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	sourceID := uuid.New()
	now := time.Now()

	var gmJSON []byte
	if source.GroundingMetadata != nil {
		gmJSON, _ = json.Marshal(source.GroundingMetadata)
	} else {
		gmJSON = []byte("{}")
	}

	retrievedAt := source.RetrievedAt
	if retrievedAt == nil {
		retrievedAt = &now
	}

	_, err = p.Exec(ctx,
		`INSERT INTO research_sources (id, research_job_id, title, url, domain, reliability_score,
		 language, author, published_at,
		 summary, main_facts, statistics, relevance_score, position,
		 freshness_score, is_verified, retrieved_at, grounding_metadata, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18::jsonb,$19)`,
		sourceID, jobID, source.Title, source.URL, source.Domain, source.ReliabilityScore,
		source.Language, source.Author,
		source.PublishedAt, source.Summary, source.MainFacts, source.Statistics,
		source.RelevanceScore, source.Position,
		source.FreshnessScore, source.IsVerified, retrievedAt,
		string(gmJSON), now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add source: %w", err)
	}

	source.ID = sourceID
	source.ResearchJobID = jobID
	source.CreatedAt = now

	_, _ = p.Exec(ctx,
		`UPDATE research_jobs SET sources_count = sources_count + 1, updated_at = NOW() WHERE id = $1`,
		jobID,
	)

	return &source, nil
}

func (s *Service) listSources(ctx context.Context, jobID uuid.UUID) ([]ResearchSource, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, research_job_id, COALESCE(title, ''), url, COALESCE(language, ''), COALESCE(author, ''),
		        published_at, COALESCE(summary, ''), COALESCE(main_facts, ''), COALESCE(statistics, ''),
		        COALESCE(relevance_score, 0), COALESCE(position, 0),
		        COALESCE(freshness_score, 0), COALESCE(is_verified, false), retrieved_at,
		        COALESCE(grounding_metadata::text, '{}'), created_at
		 FROM research_sources WHERE research_job_id = $1 ORDER BY position ASC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list sources: %w", err)
	}
	defer rows.Close()

	var sources []ResearchSource
	for rows.Next() {
		var s ResearchSource
		var gmStr string
		if err := rows.Scan(&s.ID, &s.ResearchJobID, &s.Title, &s.URL, &s.Language, &s.Author,
			&s.PublishedAt, &s.Summary, &s.MainFacts, &s.Statistics, &s.RelevanceScore, &s.Position,
			&s.FreshnessScore, &s.IsVerified, &s.RetrievedAt,
			&gmStr, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}
		if gmStr != "" && gmStr != "{}" {
			_ = json.Unmarshal([]byte(gmStr), &s.GroundingMetadata)
		}
		sources = append(sources, s)
	}
	if sources == nil {
		sources = []ResearchSource{}
	}
	return sources, nil
}

func (s *Service) AddEntity(ctx context.Context, jobID uuid.UUID, entity ResearchEntity) (*ResearchEntity, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	entityID := uuid.New()
	now := time.Now()

	_, err = p.Exec(ctx,
		`INSERT INTO research_entities (id, research_job_id, entity_type, name, context, source_url, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		entityID, jobID, string(entity.EntityType), entity.Name, entity.Context, entity.SourceURL, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add entity: %w", err)
	}

	entity.ID = entityID
	entity.ResearchJobID = jobID
	entity.CreatedAt = now

	return &entity, nil
}

func (s *Service) listEntities(ctx context.Context, jobID uuid.UUID) ([]ResearchEntity, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, research_job_id, entity_type, name, COALESCE(context, ''), COALESCE(source_url, ''), created_at
		 FROM research_entities WHERE research_job_id = $1 ORDER BY entity_type, name`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	defer rows.Close()

	var entities []ResearchEntity
	for rows.Next() {
		var e ResearchEntity
		if err := rows.Scan(&e.ID, &e.ResearchJobID, &e.EntityType, &e.Name, &e.Context, &e.SourceURL, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}
		entities = append(entities, e)
	}
	if entities == nil {
		entities = []ResearchEntity{}
	}
	return entities, nil
}

func (s *Service) SaveBriefing(ctx context.Context, jobID uuid.UUID, briefing ResearchBriefing) (*ResearchBriefing, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()

	briefingJSON, _ := json.Marshal(briefing.StructuredBriefing)
	timelineJSON, _ := json.Marshal(briefing.Timeline)
	factsJSON, _ := json.Marshal(briefing.ConfirmedFacts)
	conflictingJSON, _ := json.Marshal(briefing.ConflictingInfo)
	approachesJSON, _ := json.Marshal(briefing.EditorialApproaches)

	_, err = p.Exec(ctx,
		`INSERT INTO research_briefings (id, research_job_id, structured_briefing, timeline, confirmed_facts, conflicting_info, editorial_approaches, created_at, updated_at)
		 VALUES ($1, $2, $3::jsonb, $4::jsonb, $5::jsonb, $6::jsonb, $7::jsonb, $8, $9)
		 ON CONFLICT (research_job_id) DO UPDATE SET
		   structured_briefing = EXCLUDED.structured_briefing,
		   timeline = EXCLUDED.timeline,
		   confirmed_facts = EXCLUDED.confirmed_facts,
		   conflicting_info = EXCLUDED.conflicting_info,
		   editorial_approaches = EXCLUDED.editorial_approaches,
		   updated_at = EXCLUDED.updated_at`,
		uuid.New(), jobID, string(briefingJSON), string(timelineJSON), string(factsJSON),
		string(conflictingJSON), string(approachesJSON), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save briefing: %w", err)
	}

	return s.GetBriefingByJobID(ctx, jobID)
}

func (s *Service) GetBriefing(ctx context.Context, siteID, jobID uuid.UUID) (*ResearchBriefing, error) {
	_, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	return s.GetBriefingByJobID(ctx, jobID)
}

func (s *Service) GetBriefingByJobID(ctx context.Context, jobID uuid.UUID) (*ResearchBriefing, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var b ResearchBriefing
	var briefingStr, timelineStr, factsStr, conflictingStr, approachesStr string

	err = p.QueryRow(ctx,
		`SELECT id, research_job_id,
		        COALESCE(structured_briefing::text, '{}'),
		        COALESCE(timeline::text, '[]'),
		        COALESCE(confirmed_facts::text, '[]'),
		        COALESCE(conflicting_info::text, '[]'),
		        COALESCE(editorial_approaches::text, '[]'),
		        created_at, updated_at
		 FROM research_briefings WHERE research_job_id = $1`,
		jobID,
	).Scan(&b.ID, &b.ResearchJobID,
		&briefingStr, &timelineStr, &factsStr, &conflictingStr, &approachesStr,
		&b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrBriefingNotFound
		}
		return nil, fmt.Errorf("failed to get briefing: %w", err)
	}

	if len(briefingStr) > 0 {
		_ = json.Unmarshal([]byte(briefingStr), &b.StructuredBriefing)
	}
	if len(timelineStr) > 0 {
		_ = json.Unmarshal([]byte(timelineStr), &b.Timeline)
	}
	if len(factsStr) > 0 {
		_ = json.Unmarshal([]byte(factsStr), &b.ConfirmedFacts)
	}
	if len(conflictingStr) > 0 {
		_ = json.Unmarshal([]byte(conflictingStr), &b.ConflictingInfo)
	}
	if len(approachesStr) > 0 {
		_ = json.Unmarshal([]byte(approachesStr), &b.EditorialApproaches)
	}
	if b.StructuredBriefing == nil {
		b.StructuredBriefing = make(map[string]interface{})
	}
	if b.Timeline == nil {
		b.Timeline = []interface{}{}
	}
	if b.ConfirmedFacts == nil {
		b.ConfirmedFacts = []interface{}{}
	}
	if b.ConflictingInfo == nil {
		b.ConflictingInfo = []interface{}{}
	}
	if b.EditorialApproaches == nil {
		b.EditorialApproaches = []interface{}{}
	}

	return &b, nil
}

func (s *Service) CompleteJob(ctx context.Context, siteID, jobID uuid.UUID) error {
	_, err := s.UpdateJob(ctx, siteID, jobID, UpdateResearchJobRequest{
		Status: jobStatusPtr(JobStatusCompleted),
	})
	if err != nil {
		return err
	}

	s.fireEvent(ctx, EventResearchCompleted, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
	}, siteID)

	return nil
}

func jobStatusPtr(s JobStatus) *JobStatus { return &s }

func ArticleSourcesFromGrounding(siteID, articleID, pipelineJobID, workflowJobID, autocontentJobID uuid.UUID, gm *ai.GroundingMetadata) []ArticleSource {
	if gm == nil {
		return nil
	}
	now := time.Now()
	var sources []ArticleSource
	for _, gs := range gm.Sources {
		src := ArticleSource{
			SiteID:        siteID,
			ArticleID:     &articleID,
			SourceURL:     gs.URI,
			Title:         gs.Title,
			Snippet:       gs.Snippet,
			PublishedAt:   gs.PublishedAt,
			RetrievedAt:   now,
			FreshnessScore: gs.FreshnessScore,
			IsVerified:    gs.IsVerified,
			DomainRank:    gs.DomainRank,
			RelevanceScore: int(gs.FreshnessScore * 100),
		}
		if pipelineJobID != uuid.Nil {
			src.PipelineJobID = &pipelineJobID
		}
		if workflowJobID != uuid.Nil {
			src.WorkflowJobID = &workflowJobID
		}
		if autocontentJobID != uuid.Nil {
			src.AutocontentJobID = &autocontentJobID
		}
		if gs.PublishedAt != nil {
			src.PublishedAt = gs.PublishedAt
		}
		sources = append(sources, src)
	}
	return sources
}
