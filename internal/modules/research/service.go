package research

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

	_, err = p.Exec(ctx,
		`INSERT INTO research_sources (id, research_job_id, title, url, language, author, published_at, summary, main_facts, statistics, relevance_score, position, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		sourceID, jobID, source.Title, source.URL, source.Language, source.Author,
		source.PublishedAt, source.Summary, source.MainFacts, source.Statistics,
		source.RelevanceScore, source.Position, now,
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
		        COALESCE(relevance_score, 0), COALESCE(position, 0), created_at
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
		if err := rows.Scan(&s.ID, &s.ResearchJobID, &s.Title, &s.URL, &s.Language, &s.Author,
			&s.PublishedAt, &s.Summary, &s.MainFacts, &s.Statistics, &s.RelevanceScore, &s.Position, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
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
