package contentgenerator

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

// --- Job Engine ---

func (s *Service) CreateJob(ctx context.Context, siteID, userID uuid.UUID, req CreateJobRequest) (*GenerationJob, error) {
	if req.Language != "pt" && req.Language != "en" {
		return nil, ErrInvalidLanguage
	}
	if req.Priority < 1 || req.Priority > 10 {
		return nil, ErrInvalidPriority
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	jobID := uuid.New()
	priority := req.Priority
	if priority == 0 {
		priority = 5
	}

	_, err = p.Exec(ctx,
		`INSERT INTO generation_jobs (id, site_id, article_job_id, research_job_id, priority, language,
		 category, article_type, expected_size, style_slug, keywords, status, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'pending',$12,$13,$13)`,
		jobID, siteID, req.ArticleJobID, req.ResearchJobID, priority, req.Language,
		req.Category, req.ArticleType, req.ExpectedSize, req.StyleSlug, req.Keywords,
		userID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create generation job: %w", err)
	}

	for _, stage := range ValidStages {
		stageID := uuid.New()
		_, err = p.Exec(ctx,
			`INSERT INTO generation_pipeline (id, generation_job_id, stage, status, created_at, updated_at)
			 VALUES ($1,$2,$3,'pending',$4,$4)`,
			stageID, jobID, string(stage), now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create pipeline stage %s: %w", stage, err)
		}
	}

	s.addLog(ctx, p, jobID, "", "info", "generation job created", nil, 0)

	job := &GenerationJob{
		ID:           jobID,
		SiteID:       siteID,
		ArticleJobID: req.ArticleJobID,
		Priority:     priority,
		Language:     req.Language,
		Category:     req.Category,
		ArticleType:  req.ArticleType,
		ExpectedSize: req.ExpectedSize,
		StyleSlug:    req.StyleSlug,
		Keywords:     req.Keywords,
		Status:       GenStatusPending,
		CreatedBy:    &userID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("generation.job.created"),
		EntityType: "generation_job",
		EntityID:   &jobID,
		Payload:    map[string]interface{}{"language": req.Language, "priority": priority},
	})

	return job, nil
}

func (s *Service) GetJob(ctx context.Context, siteID, jobID uuid.UUID) (*GenerationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var j GenerationJob
	err = p.QueryRow(ctx,
		`SELECT id, site_id, article_job_id, research_job_id, priority, language,
		        COALESCE(category,''), COALESCE(article_type,'article'), COALESCE(expected_size,'medium'),
		        COALESCE(style_slug,''), COALESCE(keywords,'{}'), status,
		        COALESCE(progress,0), COALESCE(current_stage,''), COALESCE(error_message,''),
		        COALESCE(retry_count,0), COALESCE(max_retries,3),
		        started_at, completed_at, cancelled_at, created_by, created_at, updated_at
		 FROM generation_jobs WHERE id = $1 AND site_id = $2`,
		jobID, siteID,
	).Scan(&j.ID, &j.SiteID, &j.ArticleJobID, &j.ResearchJobID, &j.Priority, &j.Language,
		&j.Category, &j.ArticleType, &j.ExpectedSize,
		&j.StyleSlug, &j.Keywords, &j.Status,
		&j.Progress, &j.CurrentStage, &j.ErrorMessage,
		&j.RetryCount, &j.MaxRetries,
		&j.StartedAt, &j.CompletedAt, &j.CancelledAt, &j.CreatedBy, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get generation job: %w", err)
	}

	return &j, nil
}

func (s *Service) GetJobDetail(ctx context.Context, siteID, jobID uuid.UUID) (*GenerationJob, error) {
	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	pipeline, _ := s.ListPipeline(ctx, jobID)
	job.Pipeline = pipeline

	logs, _ := s.ListLogs(ctx, jobID, "", "", 0, 0)
	job.Logs = logs

	return job, nil
}

func (s *Service) ListJobs(ctx context.Context, siteID uuid.UUID, status, language, stage string, limit, offset int) ([]GenerationJob, error) {
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
	if stage != "" {
		where = append(where, fmt.Sprintf("current_stage = $%d", argIdx))
		args = append(args, stage)
		argIdx++
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, article_job_id, research_job_id, priority, language,
		        COALESCE(category,''), COALESCE(article_type,'article'), COALESCE(expected_size,'medium'),
		        COALESCE(style_slug,''), COALESCE(keywords,'{}'), status,
		        COALESCE(progress,0), COALESCE(current_stage,''), COALESCE(error_message,''),
		        COALESCE(retry_count,0), COALESCE(max_retries,3),
		        started_at, completed_at, cancelled_at, created_by, created_at, updated_at
		 FROM generation_jobs WHERE %s ORDER BY priority ASC, created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list generation jobs: %w", err)
	}
	defer rows.Close()

	var jobs []GenerationJob
	for rows.Next() {
		var j GenerationJob
		if err := rows.Scan(&j.ID, &j.SiteID, &j.ArticleJobID, &j.ResearchJobID, &j.Priority, &j.Language,
			&j.Category, &j.ArticleType, &j.ExpectedSize,
			&j.StyleSlug, &j.Keywords, &j.Status,
			&j.Progress, &j.CurrentStage, &j.ErrorMessage,
			&j.RetryCount, &j.MaxRetries,
			&j.StartedAt, &j.CompletedAt, &j.CancelledAt, &j.CreatedBy, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan generation job: %w", err)
		}
		jobs = append(jobs, j)
	}
	if jobs == nil {
		jobs = []GenerationJob{}
	}
	return jobs, nil
}

func (s *Service) UpdateJob(ctx context.Context, siteID, jobID uuid.UUID, req UpdateJobRequest) (*GenerationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	if existing.Status == GenStatusCompleted {
		return nil, ErrJobAlreadyCompleted
	}
	if existing.Status == GenStatusCancelled {
		return nil, ErrJobAlreadyCancelled
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Priority != nil {
		if *req.Priority < 1 || *req.Priority > 10 {
			return nil, ErrInvalidPriority
		}
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *req.Priority)
		argIdx++
	}
	if req.StyleSlug != nil {
		setClauses = append(setClauses, fmt.Sprintf("style_slug = $%d", argIdx))
		args = append(args, *req.StyleSlug)
		argIdx++
	}
	if req.Keywords != nil {
		setClauses = append(setClauses, fmt.Sprintf("keywords = $%d", argIdx))
		args = append(args, *req.Keywords)
		argIdx++
	}
	if req.ExpectedSize != nil {
		setClauses = append(setClauses, fmt.Sprintf("expected_size = $%d", argIdx))
		args = append(args, *req.ExpectedSize)
		argIdx++
	}

	if len(setClauses) == 0 {
		return existing, nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE generation_jobs SET %s WHERE id = $%d AND site_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, jobID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update generation job: %w", err)
	}

	return s.GetJob(ctx, siteID, jobID)
}

// --- Pipeline Executor ---

func (s *Service) StartJob(ctx context.Context, siteID, jobID uuid.UUID) (*GenerationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	if job.Status == GenStatusRunning {
		return nil, ErrJobAlreadyRunning
	}
	if job.Status == GenStatusCompleted {
		return nil, ErrJobAlreadyCompleted
	}
	if job.Status == GenStatusCancelled {
		return nil, ErrJobAlreadyCancelled
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`UPDATE generation_jobs SET status = 'running', started_at = $1, current_stage = $2, updated_at = $1
		 WHERE id = $3 AND site_id = $4`,
		now, string(ValidStages[0]), jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start generation job: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE generation_pipeline SET status = 'running', started_at = $1, updated_at = $1
		 WHERE generation_job_id = $2 AND stage = $3`,
		now, jobID, string(ValidStages[0]),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start pipeline stage: %w", err)
	}

	s.addLog(ctx, p, jobID, string(ValidStages[0]), "info", "generation job started", nil, 0)

	s.fireEvent(ctx, EventGenStarted, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
		"stage":   string(ValidStages[0]),
}, siteID)

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) PauseJob(ctx context.Context, siteID, jobID uuid.UUID) (*GenerationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != GenStatusRunning {
		return nil, ErrJobNotRunning
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`UPDATE generation_jobs SET status = 'paused', updated_at = $1 WHERE id = $2 AND site_id = $3`,
		now, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pause generation job: %w", err)
	}

	s.addLog(ctx, p, jobID, job.CurrentStage, "info", "generation job paused", nil, 0)

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) ResumeJob(ctx context.Context, siteID, jobID uuid.UUID) (*GenerationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != GenStatusPaused {
		return nil, ErrJobNotRunning
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`UPDATE generation_jobs SET status = 'running', updated_at = $1 WHERE id = $2 AND site_id = $3`,
		now, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resume generation job: %w", err)
	}

	s.addLog(ctx, p, jobID, job.CurrentStage, "info", "generation job resumed", nil, 0)

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) CancelJob(ctx context.Context, siteID, jobID uuid.UUID, reason string) (*GenerationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status == GenStatusCompleted {
		return nil, ErrJobAlreadyCompleted
	}
	if job.Status == GenStatusCancelled {
		return nil, ErrJobAlreadyCancelled
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`UPDATE generation_jobs SET status = 'cancelled', cancelled_at = $1, error_message = $2, updated_at = $1
		 WHERE id = $3 AND site_id = $4`,
		now, reason, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel generation job: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE generation_pipeline SET status = 'skipped', updated_at = $1
		 WHERE generation_job_id = $2 AND status = 'pending'`,
		now, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to skip pending stages: %w", err)
	}

	if job.CurrentStage != "" {
		_, err = p.Exec(ctx,
			`UPDATE generation_pipeline SET status = 'failed', error_message = $1, updated_at = $2
			 WHERE generation_job_id = $3 AND stage = $4 AND status = 'running'`,
			reason, now, jobID, job.CurrentStage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to fail current stage: %w", err)
		}
	}

	s.addLog(ctx, p, jobID, job.CurrentStage, "warning", fmt.Sprintf("generation job cancelled: %s", reason), nil, 0)

	s.fireEvent(ctx, EventGenCancelled, map[string]interface{}{
		"job_id":     jobID.String(),
		"site_id":    siteID.String(),
		"reason":     reason,
		"stage":      job.CurrentStage,
}, siteID)

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) RetryStage(ctx context.Context, siteID, jobID uuid.UUID, stage string) (*GenerationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	if job.RetryCount >= job.MaxRetries {
		return nil, ErrMaxRetriesExceeded
	}

	now := time.Now()
	newRetryCount := job.RetryCount + 1

	_, err = p.Exec(ctx,
		`UPDATE generation_jobs SET status = 'retrying', retry_count = $1, error_message = '', updated_at = $2
		 WHERE id = $3 AND site_id = $4`,
		newRetryCount, now, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update retry status: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE generation_pipeline SET status = 'pending', error_message = '', retry_count = retry_count + 1, updated_at = $1
		 WHERE generation_job_id = $2 AND stage = $3`,
		now, jobID, stage,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reset stage for retry: %w", err)
	}

	s.addLog(ctx, p, jobID, stage, "info", fmt.Sprintf("stage retry #%d", newRetryCount), nil, 0)

	s.fireEvent(ctx, EventGenRetry, map[string]interface{}{
		"job_id":      jobID.String(),
		"site_id":     siteID.String(),
		"stage":       stage,
		"retry_count": newRetryCount,
}, siteID)

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) RestartJob(ctx context.Context, siteID, jobID uuid.UUID) (*GenerationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	_, err = s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	_, err = p.Exec(ctx,
		`UPDATE generation_jobs SET status = 'pending', progress = 0, current_stage = '',
		 error_message = '', retry_count = 0, started_at = NULL, completed_at = NULL,
		 cancelled_at = NULL, updated_at = $1
		 WHERE id = $2 AND site_id = $3`,
		now, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to restart generation job: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE generation_pipeline SET status = 'pending', progress = 0, started_at = NULL,
		 completed_at = NULL, duration_ms = 0, error_message = '', retry_count = 0, updated_at = $1
		 WHERE generation_job_id = $2`,
		now, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reset pipeline: %w", err)
	}

	s.addLog(ctx, p, jobID, "", "info", "generation job restarted", nil, 0)

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) UpdateStage(ctx context.Context, jobID uuid.UUID, stage string, status StageStatus, progress float64, metadata map[string]interface{}, errorMsg string) (*GenStageItem, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	setClauses := []string{fmt.Sprintf("status = '%s'", status)}
	args := []interface{}{}
	argIdx := 1

	if status == StageStatusRunning {
		setClauses = append(setClauses, fmt.Sprintf("started_at = $%d", argIdx))
		args = append(args, now)
		argIdx++
	}
	if status == StageStatusCompleted || status == StageStatusFailed {
		setClauses = append(setClauses, fmt.Sprintf("completed_at = $%d", argIdx))
		args = append(args, now)
		argIdx++
		setClauses = append(setClauses, fmt.Sprintf("duration_ms = $%d", argIdx))
		args = append(args, int64(0))
		argIdx++
	}
	if progress > 0 {
		setClauses = append(setClauses, fmt.Sprintf("progress = $%d", argIdx))
		args = append(args, progress)
		argIdx++
	}
	if errorMsg != "" {
		setClauses = append(setClauses, fmt.Sprintf("error_message = $%d", argIdx))
		args = append(args, errorMsg)
		argIdx++
	}
	if metadata != nil {
		data, _ := json.Marshal(metadata)
		setClauses = append(setClauses, fmt.Sprintf("metadata = $%d::jsonb", argIdx))
		args = append(args, string(data))
		argIdx++
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE generation_pipeline SET %s WHERE generation_job_id = $%d AND stage = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, jobID, stage)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update pipeline stage: %w", err)
	}

	if status == StageStatusCompleted {
		nextStage := s.nextStage(stage)
		if nextStage != "" {
			_, err = p.Exec(ctx,
				`UPDATE generation_pipeline SET status = 'pending', updated_at = $1
				 WHERE generation_job_id = $2 AND stage = $3`,
				now, jobID, nextStage,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to activate next stage: %w", err)
			}

			_, err = p.Exec(ctx,
				`UPDATE generation_jobs SET current_stage = $1, updated_at = $2 WHERE id = $3`,
				nextStage, now, jobID,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to update job current stage: %w", err)
			}

			s.fireEvent(ctx, EventGenProgress, map[string]interface{}{
				"job_id":   jobID.String(),
				"stage":    nextStage,
				"progress": progress,
			}, uuid.Nil)
		} else {
			_, err = p.Exec(ctx,
				`UPDATE generation_jobs SET status = 'completed', progress = 100, completed_at = $1, updated_at = $1
				 WHERE id = $2`,
				now, jobID,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to complete job: %w", err)
			}

			s.fireEvent(ctx, EventGenCompleted, map[string]interface{}{
				"job_id": jobID.String(),
			}, uuid.Nil)
			s.fireEvent(ctx, EventGenReady, map[string]interface{}{
				"job_id": jobID.String(),
			}, uuid.Nil)
		}
	}

	if status == StageStatusFailed {
		_, err = p.Exec(ctx,
			`UPDATE generation_jobs SET status = 'failed', error_message = $1, updated_at = $2 WHERE id = $3`,
			errorMsg, now, jobID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to mark job as failed: %w", err)
		}

		s.fireEvent(ctx, EventGenFailed, map[string]interface{}{
			"job_id":  jobID.String(),
			"stage":   stage,
			"error":   errorMsg,
		}, uuid.Nil)
	}

	return s.getStageItem(ctx, jobID, stage)
}

func (s *Service) nextStage(current string) string {
	for i, stage := range ValidStages {
		if string(stage) == current && i+1 < len(ValidStages) {
			return string(ValidStages[i+1])
		}
	}
	return ""
}

func (s *Service) getStageItem(ctx context.Context, jobID uuid.UUID, stage string) (*GenStageItem, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var item GenStageItem
	var metadataStr string
	err = p.QueryRow(ctx,
		`SELECT id, generation_job_id, stage, status, COALESCE(progress,0),
		        started_at, completed_at, COALESCE(duration_ms,0), COALESCE(error_message,''),
		        COALESCE(retry_count,0), COALESCE(metadata::text,'{}'), created_at, updated_at
		 FROM generation_pipeline WHERE generation_job_id = $1 AND stage = $2`,
		jobID, stage,
	).Scan(&item.ID, &item.GenerationJobID, &item.Stage, &item.Status, &item.Progress,
		&item.StartedAt, &item.CompletedAt, &item.DurationMs, &item.ErrorMessage,
		&item.RetryCount, &metadataStr, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrStageNotFound
		}
		return nil, fmt.Errorf("failed to get stage item: %w", err)
	}

	if len(metadataStr) > 0 {
		_ = json.Unmarshal([]byte(metadataStr), &item.Metadata)
	}
	if item.Metadata == nil {
		item.Metadata = make(map[string]interface{})
	}
	return &item, nil
}

func (s *Service) ListPipeline(ctx context.Context, jobID uuid.UUID) ([]GenStageItem, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, generation_job_id, stage, status, COALESCE(progress,0),
		        started_at, completed_at, COALESCE(duration_ms,0), COALESCE(error_message,''),
		        COALESCE(retry_count,0), COALESCE(metadata::text,'{}'), created_at, updated_at
		 FROM generation_pipeline WHERE generation_job_id = $1 ORDER BY created_at ASC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipeline: %w", err)
	}
	defer rows.Close()

	var items []GenStageItem
	for rows.Next() {
		var item GenStageItem
		var metadataStr string
		if err := rows.Scan(&item.ID, &item.GenerationJobID, &item.Stage, &item.Status, &item.Progress,
			&item.StartedAt, &item.CompletedAt, &item.DurationMs, &item.ErrorMessage,
			&item.RetryCount, &metadataStr, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan pipeline stage: %w", err)
		}
		if len(metadataStr) > 0 {
			_ = json.Unmarshal([]byte(metadataStr), &item.Metadata)
		}
		if item.Metadata == nil {
			item.Metadata = make(map[string]interface{})
		}
		items = append(items, item)
	}
	if items == nil {
		items = []GenStageItem{}
	}
	return items, nil
}

// --- Logs ---

func (s *Service) addLog(ctx context.Context, p database.Pool, jobID uuid.UUID, stage, level, message string, details map[string]interface{}, durationMs int64) {
	detailsJSON, _ := json.Marshal(details)
	_, err := p.Exec(ctx,
		`INSERT INTO generation_pipeline_logs (id, generation_job_id, stage, level, message, details, duration_ms, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8)`,
		uuid.New(), jobID, stage, level, message, string(detailsJSON), durationMs, time.Now(),
	)
	if err != nil {
		s.log.Error("failed to add generation log", "error", err)
	}
}

func (s *Service) ListLogs(ctx context.Context, jobID uuid.UUID, stage, level string, limit, offset int) ([]GenLogEntry, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	where := []string{"generation_job_id = $1"}
	args := []interface{}{jobID}
	argIdx := 2

	if stage != "" {
		where = append(where, fmt.Sprintf("stage = $%d", argIdx))
		args = append(args, stage)
		argIdx++
	}
	if level != "" {
		where = append(where, fmt.Sprintf("level = $%d", argIdx))
		args = append(args, level)
		argIdx++
	}

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT id, generation_job_id, COALESCE(stage,''), level, message,
		        COALESCE(details::text,'{}'), COALESCE(duration_ms,0), created_at
		 FROM generation_pipeline_logs WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list logs: %w", err)
	}
	defer rows.Close()

	var logs []GenLogEntry
	for rows.Next() {
		var l GenLogEntry
		var detailsStr string
		if err := rows.Scan(&l.ID, &l.GenerationJobID, &l.Stage, &l.Level, &l.Message,
			&detailsStr, &l.DurationMs, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan log entry: %w", err)
		}
		if len(detailsStr) > 0 {
			_ = json.Unmarshal([]byte(detailsStr), &l.Details)
		}
		if l.Details == nil {
			l.Details = make(map[string]interface{})
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []GenLogEntry{}
	}
	return logs, nil
}

// --- Quality Gate ---

func (s *Service) CheckQualityGate(ctx context.Context, siteID, jobID uuid.UUID, req QualityGateRequest) (*QualityGate, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	_, err = s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	gateID := uuid.New()

	reportJSON, _ := json.Marshal(req.Report)

	_, err = p.Exec(ctx,
		`INSERT INTO generation_quality_gates (id, generation_job_id, stage, status,
		 seo_score, readability_score, eeat_score, keyword_density, heading_score,
		 internal_linking_score, required_content_passed, min_size_passed, metadata_passed,
		 overall_passed, report, checked_at, created_at)
		 VALUES ($1,$2,'final',$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::jsonb,$15,$15)`,
		gateID, jobID, gateStatus(req.OverallPassed),
		req.SEOScore, req.ReadabilityScore, req.EEATScore, req.KeywordDensity, req.HeadingScore,
		req.InternalLinkingScore, req.RequiredContentPassed, req.MinSizePassed, req.MetadataPassed,
		req.OverallPassed, string(reportJSON), now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create quality gate: %w", err)
	}

	s.fireEvent(ctx, EventGenReviewed, map[string]interface{}{
		"job_id":         jobID.String(),
		"site_id":        siteID.String(),
		"overall_passed": req.OverallPassed,
}, siteID)

	return &QualityGate{
		ID:                  gateID,
		GenerationJobID:     jobID,
		Stage:               "final",
		Status:              gateStatus(req.OverallPassed),
		SEOScore:            req.SEOScore,
		ReadabilityScore:    req.ReadabilityScore,
		EEATScore:           req.EEATScore,
		KeywordDensity:      req.KeywordDensity,
		HeadingScore:        req.HeadingScore,
		InternalLinkingScore: req.InternalLinkingScore,
		RequiredContentPassed: req.RequiredContentPassed,
		MinSizePassed:       req.MinSizePassed,
		MetadataPassed:      req.MetadataPassed,
		OverallPassed:       req.OverallPassed,
		Report:              req.Report,
		CheckedAt:           &now,
		CreatedAt:           now,
	}, nil
}

func gateStatus(passed bool) string {
	if passed {
		return "passed"
	}
	return "failed"
}

func (s *Service) GetQualityGates(ctx context.Context, jobID uuid.UUID) ([]QualityGate, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, generation_job_id, stage, status, seo_score, readability_score,
		        eeat_score, keyword_density, heading_score, internal_linking_score,
		        required_content_passed, min_size_passed, metadata_passed, overall_passed,
		        COALESCE(report::text,'{}'), checked_by, checked_at, created_at
		 FROM generation_quality_gates WHERE generation_job_id = $1 ORDER BY created_at DESC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list quality gates: %w", err)
	}
	defer rows.Close()

	var gates []QualityGate
	for rows.Next() {
		var g QualityGate
		var reportStr string
		if err := rows.Scan(&g.ID, &g.GenerationJobID, &g.Stage, &g.Status,
			&g.SEOScore, &g.ReadabilityScore, &g.EEATScore, &g.KeywordDensity, &g.HeadingScore,
			&g.InternalLinkingScore,
			&g.RequiredContentPassed, &g.MinSizePassed, &g.MetadataPassed, &g.OverallPassed,
			&reportStr, &g.CheckedBy, &g.CheckedAt, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan quality gate: %w", err)
		}
		if len(reportStr) > 0 {
			_ = json.Unmarshal([]byte(reportStr), &g.Report)
		}
		if g.Report == nil {
			g.Report = make(map[string]interface{})
		}
		gates = append(gates, g)
	}
	if gates == nil {
		gates = []QualityGate{}
	}
	return gates, nil
}

// --- Stats ---

func (s *Service) GetStats(ctx context.Context, siteID uuid.UUID) (*GenStats, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	dateStr := now.Format("2006-01-02")

	var stats GenStats
	err = p.QueryRow(ctx,
		`SELECT id, site_id, date::text, total_jobs, completed_jobs, failed_jobs,
		        cancelled_jobs, avg_duration_ms, avg_success_rate, total_errors, throughput
		 FROM generation_stats WHERE site_id = $1 AND date = $2::date`,
		siteID, dateStr,
	).Scan(&stats.ID, &stats.SiteID, &stats.Date, &stats.TotalJobs, &stats.CompletedJobs,
		&stats.FailedJobs, &stats.CancelledJobs, &stats.AvgDurationMs, &stats.AvgSuccessRate,
		&stats.TotalErrors, &stats.Throughput)
	if err != nil {
		if err == pgx.ErrNoRows {
			stats = GenStats{
				SiteID:    siteID,
				Date:      dateStr,
			}
		} else {
			return nil, fmt.Errorf("failed to get stats: %w", err)
		}
	}

	return &stats, nil
}

func (s *Service) GetDashboardStats(ctx context.Context, siteID uuid.UUID) (map[string]interface{}, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var running, queued, completed, failed, cancelled int
	var avgDuration int64
	var total int

	err = p.QueryRow(ctx,
		`SELECT COALESCE(SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0),
		        COALESCE(AVG(CASE WHEN status = 'completed' THEN EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000 ELSE NULL END), 0),
		        COUNT(*)
		 FROM generation_jobs WHERE site_id = $1`,
		siteID,
	).Scan(&running, &queued, &completed, &failed, &cancelled, &avgDuration, &total)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard stats: %w", err)
	}

	successRate := 0.0
	if total > 0 {
		successRate = float64(completed) / float64(total) * 100
	}

	return map[string]interface{}{
		"running":     running,
		"queued":      queued,
		"completed":   completed,
		"failed":      failed,
		"cancelled":   cancelled,
		"total":       total,
		"avg_duration_ms": avgDuration,
		"success_rate":    successRate,
	}, nil
}

// --- Prompt Assembly ---

func (s *Service) AssemblePrompt(ctx context.Context, siteID, jobID uuid.UUID) (map[string]interface{}, error) {
	_, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	prompt := map[string]interface{}{
		"job_id":         job.ID.String(),
		"language":       job.Language,
		"article_type":   job.ArticleType,
		"expected_size":  job.ExpectedSize,
		"style_slug":     job.StyleSlug,
		"keywords":       job.Keywords,
		"category":       job.Category,
		"target_audience": "",
		"tone":           "",
		"constraints":    []string{},
	}

	if job.ResearchJobID != nil {
		prompt["research_job_id"] = job.ResearchJobID.String()
	}
	if job.ArticleJobID != nil {
		prompt["article_job_id"] = job.ArticleJobID.String()
	}

	return prompt, nil
}
