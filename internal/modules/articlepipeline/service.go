package articlepipeline

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

func coalesceStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func coalesceInt(i, def int) int {
	if i == 0 {
		return def
	}
	return i
}

func toJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func parseJSON(s string) map[string]interface{} {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}

func parseChecksJSON(s string) []QualityCheck {
	var checks []QualityCheck
	if err := json.Unmarshal([]byte(s), &checks); err != nil {
		return nil
	}
	return checks
}

// --- Pipeline Job CRUD ---

func (s *Service) CreatePipeline(ctx context.Context, siteID, userID uuid.UUID, req CreatePipelineRequest) (*PipelineJob, error) {
	if req.Title == "" {
		return nil, ErrInvalidTitle
	}
	lang := coalesceStr(req.Language, "pt")
	if lang != "pt" && lang != "en" {
		return nil, ErrInvalidLanguage
	}
	priority := coalesceInt(coalesceIntPtr(req.Priority), 5)
	if priority < 1 || priority > 10 {
		return nil, ErrInvalidPriority
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	jobID := uuid.New()

	_, err = p.Exec(ctx,
		`INSERT INTO article_pipeline_jobs (id, site_id, title, topic, source_content, language,
		 target_language, content_type, status, priority, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12)`,
		jobID, siteID, req.Title, req.Topic, req.SourceContent, lang,
		req.TargetLanguage, req.ContentType, PipelineDraft, priority, userID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline job: %w", err)
	}

	for _, stage := range AllStages {
		_, err = p.Exec(ctx,
			`INSERT INTO article_pipeline_steps (id, pipeline_job_id, stage_name, display_name,
			 status, max_retries, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$7)`,
			uuid.New(), jobID, string(stage), StageDisplayNames[stage],
			StepStatusPending, 3, now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create pipeline step: %w", err)
		}
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		Action:     audit.Action("articlepipeline.created"),
		EntityType: "article_pipeline_job",
		EntityID:   &jobID,
		Payload:    map[string]interface{}{"title": req.Title, "language": lang},
	})

	s.fireEvent(ctx, EventPipelineCreated, map[string]interface{}{
		"job_id":  jobID.String(),
		"title":   req.Title,
		"site_id": siteID.String(),
	}, siteID)

	return s.GetPipelineDetail(ctx, siteID, jobID)
}

func (s *Service) GetPipeline(ctx context.Context, siteID, jobID uuid.UUID) (*PipelineJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var job PipelineJob
	err = p.QueryRow(ctx,
		`SELECT id, site_id, title, COALESCE(topic,''), COALESCE(source_content,''),
		        language, COALESCE(target_language,''), COALESCE(content_type,'article'),
		        status, COALESCE(progress,0), COALESCE(current_stage,''),
		        priority, retry_count, max_retries, COALESCE(error_message,''),
		        started_at, completed_at, cancelled_at, created_by, created_at, updated_at
		 FROM article_pipeline_jobs WHERE id = $1 AND site_id = $2`,
		jobID, siteID,
	).Scan(&job.ID, &job.SiteID, &job.Title, &job.Topic, &job.SourceContent,
		&job.Language, &job.TargetLanguage, &job.ContentType,
		&job.Status, &job.Progress, &job.CurrentStage,
		&job.Priority, &job.RetryCount, &job.MaxRetries, &job.ErrorMessage,
		&job.StartedAt, &job.CompletedAt, &job.CancelledAt, &job.CreatedBy, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get pipeline job: %w", err)
	}
	return &job, nil
}

func (s *Service) GetPipelineDetail(ctx context.Context, siteID, jobID uuid.UUID) (*PipelineJob, error) {
	job, err := s.GetPipeline(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	steps, err := s.getSteps(ctx, jobID)
	if err != nil {
		return nil, err
	}
	job.Steps = steps

	metrics, err := s.getMetrics(ctx, jobID)
	if err != nil {
		return nil, err
	}
	job.Metrics = metrics

	reports, err := s.getQualityReports(ctx, jobID)
	if err != nil {
		return nil, err
	}
	job.QualityReports = reports

	return job, nil
}

func (s *Service) ListPipelines(ctx context.Context, siteID uuid.UUID, status string, language string, limit, offset int) ([]PipelineJob, error) {
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
		`SELECT id, site_id, title, COALESCE(topic,''), language, status, COALESCE(progress,0),
		        COALESCE(current_stage,''), priority, retry_count, max_retries,
		        started_at, completed_at, cancelled_at, created_by, created_at, updated_at
		 FROM article_pipeline_jobs WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}
	defer rows.Close()

	var jobs []PipelineJob
	for rows.Next() {
		var job PipelineJob
		if err := rows.Scan(&job.ID, &job.SiteID, &job.Title, &job.Topic,
			&job.Language, &job.Status, &job.Progress, &job.CurrentStage,
			&job.Priority, &job.RetryCount, &job.MaxRetries,
			&job.StartedAt, &job.CompletedAt, &job.CancelledAt,
			&job.CreatedBy, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan pipeline job: %w", err)
		}
		jobs = append(jobs, job)
	}
	if jobs == nil {
		jobs = []PipelineJob{}
	}
	return jobs, nil
}

func (s *Service) UpdatePipeline(ctx context.Context, siteID, jobID uuid.UUID, req UpdatePipelineRequest) (*PipelineJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	_, err = s.GetPipeline(ctx, siteID, jobID)
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
	if req.Topic != nil {
		setClauses = append(setClauses, fmt.Sprintf("topic = $%d", argIdx))
		args = append(args, *req.Topic)
		argIdx++
	}
	if req.SourceContent != nil {
		setClauses = append(setClauses, fmt.Sprintf("source_content = $%d", argIdx))
		args = append(args, *req.SourceContent)
		argIdx++
	}
	if req.TargetLanguage != nil {
		setClauses = append(setClauses, fmt.Sprintf("target_language = $%d", argIdx))
		args = append(args, *req.TargetLanguage)
		argIdx++
	}
	if req.ContentType != nil {
		setClauses = append(setClauses, fmt.Sprintf("content_type = $%d", argIdx))
		args = append(args, *req.ContentType)
		argIdx++
	}
	if req.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *req.Priority)
		argIdx++
	}

	if len(setClauses) == 0 {
		return s.GetPipelineDetail(ctx, siteID, jobID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE article_pipeline_jobs SET %s WHERE id = $%d AND site_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, jobID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update pipeline job: %w", err)
	}

	return s.GetPipelineDetail(ctx, siteID, jobID)
}

func (s *Service) DeletePipeline(ctx context.Context, siteID, jobID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	tag, err := p.Exec(ctx,
		`DELETE FROM article_pipeline_jobs WHERE id = $1 AND site_id = $2`,
		jobID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete pipeline job: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotFound
	}
	return nil
}

// --- Pipeline Orchestration ---

func (s *Service) StartPipeline(ctx context.Context, siteID, jobID uuid.UUID) (*PipelineJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetPipeline(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	switch job.Status {
	case PipelineRunning:
		return nil, ErrJobAlreadyRunning
	case PipelineCompleted:
		return nil, ErrJobAlreadyCompleted
	case PipelineCancelled:
		return nil, ErrJobAlreadyCancelled
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_jobs SET status = $1, started_at = $2, current_stage = $3,
		 progress = 0, error_message = '', updated_at = $2 WHERE id = $4`,
		PipelineRunning, now, string(AllStages[0]), jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start pipeline: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_steps SET status = $1, started_at = $2, updated_at = $2
		 WHERE pipeline_job_id = $3 AND stage_name = $4`,
		StepStatusRunning, now, jobID, string(AllStages[0]),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start first stage: %w", err)
	}

	s.fireEvent(ctx, EventPipelineStarted, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
	}, siteID)

	s.fireEvent(ctx, EventStageStarted, map[string]interface{}{
		"job_id":     jobID.String(),
		"stage_name": string(AllStages[0]),
		"site_id":    siteID.String(),
	}, siteID)

	return s.GetPipelineDetail(ctx, siteID, jobID)
}

func (s *Service) PausePipeline(ctx context.Context, siteID, jobID uuid.UUID) (*PipelineJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetPipeline(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != PipelineRunning {
		return nil, ErrJobNotRunning
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_jobs SET status = $1, updated_at = $2 WHERE id = $3`,
		PipelinePaused, now, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pause pipeline: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_steps SET status = $1, updated_at = $2
		 WHERE pipeline_job_id = $3 AND status = $4`,
		StepStatusPending, now, jobID, StepStatusRunning,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pause stage: %w", err)
	}

	s.fireEvent(ctx, EventPipelinePaused, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
	}, siteID)

	return s.GetPipelineDetail(ctx, siteID, jobID)
}

func (s *Service) ResumePipeline(ctx context.Context, siteID, jobID uuid.UUID) (*PipelineJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetPipeline(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != PipelinePaused {
		return nil, ErrJobNotPaused
	}

	now := time.Now()
	currentStage := coalesceStr(job.CurrentStage, string(AllStages[0]))

	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_jobs SET status = $1, updated_at = $2 WHERE id = $3`,
		PipelineRunning, now, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resume pipeline: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_steps SET status = $1, started_at = $2, updated_at = $2
		 WHERE pipeline_job_id = $3 AND stage_name = $4`,
		StepStatusRunning, now, jobID, currentStage,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resume stage: %w", err)
	}

	s.fireEvent(ctx, EventPipelineResumed, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
	}, siteID)

	return s.GetPipelineDetail(ctx, siteID, jobID)
}

func (s *Service) CancelPipeline(ctx context.Context, siteID, jobID uuid.UUID, reason string) (*PipelineJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetPipeline(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status == PipelineCompleted {
		return nil, ErrJobAlreadyCompleted
	}
	if job.Status == PipelineCancelled {
		return nil, ErrJobAlreadyCancelled
	}

	if reason == "" {
		reason = "user requested cancellation"
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_jobs SET status = $1, cancelled_at = $2, error_message = $3,
		 updated_at = $2 WHERE id = $4`,
		PipelineCancelled, now, reason, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel pipeline: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_steps SET status = CASE
		 WHEN status = $1 THEN $2
		 WHEN status = $3 THEN $4
		 ELSE status END,
		 updated_at = $5
		 WHERE pipeline_job_id = $6`,
		StepStatusRunning, StepStatusCancelled,
		StepStatusPending, StepStatusCancelled,
		now, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel steps: %w", err)
	}

	s.fireEvent(ctx, EventPipelineCancelled, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
		"reason":  reason,
	}, siteID)

	return s.GetPipelineDetail(ctx, siteID, jobID)
}

func (s *Service) RetryStage(ctx context.Context, siteID, jobID uuid.UUID, stageName string) (*PipelineJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetPipeline(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	stage, err := s.getStage(ctx, jobID, stageName)
	if err != nil {
		return nil, err
	}

	if stage.Status != StepStatusFailed {
		return nil, fmt.Errorf("stage %s is not failed", stageName)
	}

	if job.RetryCount >= job.MaxRetries || stage.RetryCount >= stage.MaxRetries {
		return nil, ErrMaxRetriesExceeded
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_jobs SET status = $1, retry_count = retry_count + 1,
		 error_message = '', updated_at = $2 WHERE id = $3`,
		PipelineRetrying, now, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update job for retry: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_steps SET status = $1, progress = 0, error_message = '',
		 retry_count = retry_count + 1, started_at = $2, completed_at = NULL,
		 duration_ms = 0, output = '', metadata = '{}'::jsonb, updated_at = $2
		 WHERE id = $3`,
		StepStatusRunning, now, stage.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update stage for retry: %w", err)
	}

	s.fireEvent(ctx, EventPipelineRetry, map[string]interface{}{
		"job_id":     jobID.String(),
		"stage_name": stageName,
		"site_id":    siteID.String(),
	}, siteID)

	return s.GetPipelineDetail(ctx, siteID, jobID)
}

func (s *Service) RestartPipeline(ctx context.Context, siteID, jobID uuid.UUID) (*PipelineJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	_, err = s.GetPipeline(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_jobs SET status = $1, progress = 0, current_stage = '',
		 error_message = '', retry_count = 0, started_at = NULL, completed_at = NULL,
		 cancelled_at = NULL, updated_at = $2 WHERE id = $3`,
		PipelineDraft, now, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to restart pipeline: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_steps SET status = $1, progress = 0,
		 started_at = NULL, completed_at = NULL, duration_ms = 0, error_message = '',
		 retry_count = 0, output = '', metadata = '{}'::jsonb, updated_at = $2
		 WHERE pipeline_job_id = $3`,
		StepStatusPending, now, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reset steps: %w", err)
	}

	s.fireEvent(ctx, EventPipelineRestarted, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
	}, siteID)

	return s.GetPipelineDetail(ctx, siteID, jobID)
}

// --- Stage Management ---

func (s *Service) GetPipelineStages(ctx context.Context, jobID uuid.UUID) ([]Step, error) {
	return s.getSteps(ctx, jobID)
}

func (s *Service) UpdateStage(ctx context.Context, siteID, jobID uuid.UUID, stageName string, req UpdateStageRequest) (*PipelineJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	if _, err := s.GetPipeline(ctx, siteID, jobID); err != nil {
		return nil, err
	}

	stage, err := s.getStage(ctx, jobID, stageName)
	if err != nil {
		return nil, err
	}

	if stage.Status == StepStatusCompleted {
		return nil, ErrStageAlreadyCompleted
	}

	progress := stage.Progress
	if req.Progress != nil {
		progress = *req.Progress
	}

	setClauses := []string{fmt.Sprintf("status = $%d", 1)}
	args := []interface{}{req.Status}
	argIdx := 2

	setClauses = append(setClauses, fmt.Sprintf("progress = $%d", argIdx))
	args = append(args, progress)
	argIdx++

	if req.ErrorMessage != "" {
		setClauses = append(setClauses, fmt.Sprintf("error_message = $%d", argIdx))
		args = append(args, req.ErrorMessage)
		argIdx++
	}
	if req.Output != "" {
		setClauses = append(setClauses, fmt.Sprintf("output = $%d", argIdx))
		args = append(args, req.Output)
		argIdx++
	}
	if req.Metadata != nil {
		setClauses = append(setClauses, fmt.Sprintf("metadata = $%d::jsonb", argIdx))
		args = append(args, toJSON(req.Metadata))
		argIdx++
	}

	now := time.Now()

	if req.Status == StepStatusRunning {
		setClauses = append(setClauses, fmt.Sprintf("started_at = $%d", argIdx))
		args = append(args, now)
		argIdx++
	}

	if req.Status == StepStatusCompleted {
		durationMs := int64(0)
		if stage.StartedAt != nil {
			durationMs = time.Since(*stage.StartedAt).Milliseconds()
		}
		setClauses = append(setClauses, fmt.Sprintf("completed_at = $%d", argIdx))
		args = append(args, now)
		argIdx++
		setClauses = append(setClauses, fmt.Sprintf("duration_ms = $%d", argIdx))
		args = append(args, durationMs)
		argIdx++
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, now)
	argIdx++

	query := fmt.Sprintf(
		`UPDATE article_pipeline_steps SET %s WHERE id = $%d`,
		strings.Join(setClauses, ", "), argIdx,
	)
	args = append(args, stage.ID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update stage: %w", err)
	}

	switch req.Status {
	case StepStatusCompleted:
		if err := s.onStageCompleted(ctx, p, jobID, stageName, now); err != nil {
			return nil, err
		}
		s.fireEvent(ctx, EventStageCompleted, map[string]interface{}{
			"job_id":     jobID.String(),
			"stage_name": stageName,
		}, siteID)
	case StepStatusFailed:
		if err := s.onStageFailed(ctx, p, jobID, stageName, req.ErrorMessage, now); err != nil {
			return nil, err
		}
		s.fireEvent(ctx, EventStageFailed, map[string]interface{}{
			"job_id":      jobID.String(),
			"stage_name":  stageName,
			"error":       req.ErrorMessage,
		}, siteID)
	}

	return s.GetPipelineDetail(ctx, siteID, jobID)
}

func (s *Service) onStageCompleted(ctx context.Context, p database.Pool, jobID uuid.UUID, stageName string, now time.Time) error {
	nextStage := nextStageName(stageName)
	if nextStage == "" {
		_, err := p.Exec(ctx,
			`UPDATE article_pipeline_jobs SET status = $1, progress = 100,
			 current_stage = $2, completed_at = $3, updated_at = $3 WHERE id = $4`,
			PipelineCompleted, stageName, now, jobID,
		)
		if err != nil {
			return fmt.Errorf("failed to complete pipeline: %w", err)
		}
		return nil
	}

	deps := StageDependencies[StageName(nextStage)]
	allMet := true
	for _, dep := range deps {
		depStage, err := s.getStage(ctx, jobID, string(dep))
		if err != nil {
			return err
		}
		if depStage.Status != StepStatusCompleted {
			allMet = false
			break
		}
	}

	var nextStepStatus StepStatus
	if allMet {
		nextStepStatus = StepStatusPending
	} else {
		return ErrDependencyPending
	}

	_, err := p.Exec(ctx,
		`UPDATE article_pipeline_steps SET status = $1, updated_at = $2
		 WHERE pipeline_job_id = $3 AND stage_name = $4`,
		nextStepStatus, now, jobID, nextStage,
	)
	if err != nil {
		return fmt.Errorf("failed to advance to next stage: %w", err)
	}

	progress := s.calcProgress(ctx, p, jobID)
	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_jobs SET current_stage = $1, progress = $2,
		 updated_at = $3 WHERE id = $4`,
		nextStage, progress, now, jobID,
	)
	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	s.fireEvent(ctx, EventPipelineProgress, map[string]interface{}{
		"job_id":   jobID.String(),
		"stage":    nextStage,
		"progress": progress,
	}, uuid.Nil)

	return nil
}

func (s *Service) onStageFailed(ctx context.Context, p database.Pool, jobID uuid.UUID, stageName, errorMsg string, now time.Time) error {
	_, err := p.Exec(ctx,
		`UPDATE article_pipeline_jobs SET status = $1, current_stage = $2,
		 error_message = $3, updated_at = $4 WHERE id = $5`,
		PipelineFailed, stageName, errorMsg, now, jobID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark pipeline failed: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE article_pipeline_steps SET status = $1, updated_at = $2
		 WHERE pipeline_job_id = $3 AND status = $4`,
		StepStatusSkipped, now, jobID, StepStatusPending,
	)
	if err != nil {
		return fmt.Errorf("failed to skip pending stages: %w", err)
	}

	return nil
}

// --- Metrics ---

func (s *Service) RecordMetric(ctx context.Context, jobID uuid.UUID, req CreateMetricRequest) (*Metric, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	metricID := uuid.New()
	_, err = p.Exec(ctx,
		`INSERT INTO article_pipeline_metrics (id, pipeline_job_id, stage_name, metric_name,
		 metric_value, metric_unit, metadata, recorded_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,NOW())`,
		metricID, jobID, req.StageName, req.MetricName,
		req.MetricValue, req.MetricUnit, toJSON(req.Metadata),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record metric: %w", err)
	}

	var m Metric
	var metaStr string
	err = p.QueryRow(ctx,
		`SELECT id, pipeline_job_id, COALESCE(stage_name,''), metric_name, metric_value,
		        COALESCE(metric_unit,''), COALESCE(metadata::text,'{}'), recorded_at
		 FROM article_pipeline_metrics WHERE id = $1`, metricID,
	).Scan(&m.ID, &m.PipelineJobID, &m.StageName, &m.MetricName,
		&m.MetricValue, &m.MetricUnit, &metaStr, &m.RecordedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get recorded metric: %w", err)
	}
	if len(metaStr) > 0 {
		m.Metadata = parseJSON(metaStr)
	}
	return &m, nil
}

func (s *Service) GetPipelineMetrics(ctx context.Context, jobID uuid.UUID) ([]Metric, error) {
	return s.getMetrics(ctx, jobID)
}

// --- Quality Reports ---

func (s *Service) CreateQualityReport(ctx context.Context, jobID uuid.UUID, req CreateQualityReportRequest) (*QualityReport, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	reportID := uuid.New()
	qualityStatus := QualityPassed
	if req.ChecksFailed > 0 {
		qualityStatus = QualityFailed
	}

	_, err = p.Exec(ctx,
		`INSERT INTO article_quality_reports (id, pipeline_job_id, stage_name, status, score,
		 checks_passed, checks_failed, checks_total, details, summary, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,NOW())`,
		reportID, jobID, req.StageName, qualityStatus, req.Score,
		req.ChecksPassed, req.ChecksFailed, req.ChecksTotal,
		toJSON(req.Details), req.Summary,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create quality report: %w", err)
	}

	var report QualityReport
	var detailsStr string
	err = p.QueryRow(ctx,
		`SELECT id, pipeline_job_id, stage_name, status, score,
		        checks_passed, checks_failed, checks_total,
		        COALESCE(details::text,'[]'), COALESCE(summary,''), created_at
		 FROM article_quality_reports WHERE id = $1`, reportID,
	).Scan(&report.ID, &report.PipelineJobID, &report.StageName, &report.Status,
		&report.Score, &report.ChecksPassed, &report.ChecksFailed, &report.ChecksTotal,
		&detailsStr, &report.Summary, &report.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality report: %w", err)
	}
	if len(detailsStr) > 2 {
		report.Details = parseChecksJSON(detailsStr)
	}

	if qualityStatus == QualityPassed {
		s.fireEvent(ctx, EventQualityPassed, map[string]interface{}{
			"job_id":  jobID.String(),
			"stage":   req.StageName,
			"score":   req.Score,
		}, uuid.Nil)
	} else {
		s.fireEvent(ctx, EventQualityFailed, map[string]interface{}{
			"job_id":  jobID.String(),
			"stage":   req.StageName,
			"score":   req.Score,
			"failed":  req.ChecksFailed,
		}, uuid.Nil)
	}

	return &report, nil
}

func (s *Service) GetQualityReports(ctx context.Context, jobID uuid.UUID) ([]QualityReport, error) {
	return s.getQualityReports(ctx, jobID)
}

// --- Publication Candidates ---

func (s *Service) CreateCandidate(ctx context.Context, siteID, jobID uuid.UUID, req CreateCandidateRequest) (*PublicationCandidate, error) {
	if req.Title == "" {
		return nil, ErrInvalidTitle
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetPipeline(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	candidateID := uuid.New()
	wordCount := len(strings.Fields(req.Content))

	_, err = p.Exec(ctx,
		`INSERT INTO publication_candidates (id, pipeline_job_id, site_id, title, content,
		 excerpt, language, status, quality_score, seo_score, readability_score,
		 word_count, metadata, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb,NOW(),NOW())`,
		candidateID, jobID, siteID, req.Title, req.Content, req.Excerpt,
		job.Language, CandidateDraft, req.QualityScore, req.SEOScore,
		req.ReadabilityScore, wordCount, toJSON(req.Metadata),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create publication candidate: %w", err)
	}

	var cand PublicationCandidate
	var metaStr string
	err = p.QueryRow(ctx,
		`SELECT id, pipeline_job_id, site_id, title, COALESCE(content,''), COALESCE(excerpt,''),
		        language, status, COALESCE(quality_score,0), COALESCE(seo_score,0),
		        COALESCE(readability_score,0), COALESCE(word_count,0),
		        COALESCE(metadata::text,'{}'), created_at, updated_at
		 FROM publication_candidates WHERE id = $1`, candidateID,
	).Scan(&cand.ID, &cand.PipelineJobID, &cand.SiteID, &cand.Title, &cand.Content,
		&cand.Excerpt, &cand.Language, &cand.Status, &cand.QualityScore,
		&cand.SEOScore, &cand.ReadabilityScore, &cand.WordCount,
		&metaStr, &cand.CreatedAt, &cand.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get publication candidate: %w", err)
	}
	if len(metaStr) > 0 {
		cand.Metadata = parseJSON(metaStr)
	}

	s.fireEvent(ctx, EventCandidateCreated, map[string]interface{}{
		"candidate_id": candidateID.String(),
		"job_id":       jobID.String(),
		"site_id":      siteID.String(),
		"title":        req.Title,
		"quality":      req.QualityScore,
		"seo":          req.SEOScore,
	}, siteID)

	return &cand, nil
}

func (s *Service) ListCandidates(ctx context.Context, siteID uuid.UUID, status string, limit, offset int) ([]PublicationCandidate, error) {
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
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT id, pipeline_job_id, site_id, title, COALESCE(excerpt,''), language, status,
		        COALESCE(quality_score,0), COALESCE(seo_score,0), COALESCE(readability_score,0),
		        COALESCE(word_count,0), created_at, updated_at
		 FROM publication_candidates WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list candidates: %w", err)
	}
	defer rows.Close()

	var candidates []PublicationCandidate
	for rows.Next() {
		var cand PublicationCandidate
		if err := rows.Scan(&cand.ID, &cand.PipelineJobID, &cand.SiteID, &cand.Title,
			&cand.Excerpt, &cand.Language, &cand.Status,
			&cand.QualityScore, &cand.SEOScore, &cand.ReadabilityScore,
			&cand.WordCount, &cand.CreatedAt, &cand.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan candidate: %w", err)
		}
		candidates = append(candidates, cand)
	}
	if candidates == nil {
		candidates = []PublicationCandidate{}
	}
	return candidates, nil
}

// --- Stats ---

func (s *Service) GetPipelineStats(ctx context.Context, siteID uuid.UUID) (*PipelineStats, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var stats PipelineStats

	_ = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM article_pipeline_jobs WHERE site_id = $1`, siteID,
	).Scan(&stats.TotalJobs)

	_ = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM article_pipeline_jobs WHERE site_id = $1 AND status = $2`,
		siteID, PipelineRunning,
	).Scan(&stats.RunningJobs)

	_ = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM article_pipeline_jobs WHERE site_id = $1 AND status = $2`,
		siteID, PipelineCompleted,
	).Scan(&stats.CompletedJobs)

	_ = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM article_pipeline_jobs WHERE site_id = $1 AND status = $2`,
		siteID, PipelineFailed,
	).Scan(&stats.FailedJobs)

	_ = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM article_pipeline_jobs WHERE site_id = $1 AND status = $2`,
		siteID, PipelineCancelled,
	).Scan(&stats.CancelledJobs)

	if stats.CompletedJobs > 0 {
		_ = p.QueryRow(ctx,
			`SELECT COALESCE(AVG(duration_ms),0) FROM (
			 SELECT EXTRACT(EPOCH FROM (COALESCE(completed_at, NOW()) - started_at)) * 1000 AS duration_ms
			 FROM article_pipeline_jobs WHERE site_id = $1 AND status = $2 AND started_at IS NOT NULL
		 ) sub`,
			siteID, PipelineCompleted,
		).Scan(&stats.AvgDurationMs)

		_ = p.QueryRow(ctx,
			`SELECT COALESCE(AVG(quality_score),0) FROM publication_candidates WHERE site_id = $1`,
			siteID,
		).Scan(&stats.AvgQualityScore)

		_ = p.QueryRow(ctx,
			`SELECT COALESCE(AVG(seo_score),0) FROM publication_candidates WHERE site_id = $1`,
			siteID,
		).Scan(&stats.AvgSEOScore)
	}

	_ = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM publication_candidates WHERE site_id = $1`, siteID,
	).Scan(&stats.TotalCandidates)

	return &stats, nil
}

// --- Internal Helpers ---

func (s *Service) getSteps(ctx context.Context, jobID uuid.UUID) ([]Step, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, pipeline_job_id, stage_name, COALESCE(display_name,''), status,
		        COALESCE(progress,0), started_at, completed_at, COALESCE(duration_ms,0),
		        COALESCE(error_message,''), retry_count, max_retries, COALESCE(output,''),
		        COALESCE(metadata::text,'{}'), created_at, updated_at
		 FROM article_pipeline_steps WHERE pipeline_job_id = $1 ORDER BY created_at ASC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query steps: %w", err)
	}
	defer rows.Close()

	var steps []Step
	for rows.Next() {
		var step Step
		var metaStr string
		if err := rows.Scan(&step.ID, &step.PipelineJobID, &step.StageName, &step.DisplayName,
			&step.Status, &step.Progress, &step.StartedAt, &step.CompletedAt, &step.DurationMs,
			&step.ErrorMessage, &step.RetryCount, &step.MaxRetries, &step.Output,
			&metaStr, &step.CreatedAt, &step.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan step: %w", err)
		}
		if len(metaStr) > 0 {
			step.Metadata = parseJSON(metaStr)
		}
		if step.Metadata == nil {
			step.Metadata = make(map[string]interface{})
		}
		steps = append(steps, step)
	}
	if steps == nil {
		steps = []Step{}
	}
	return steps, nil
}

func (s *Service) getStage(ctx context.Context, jobID uuid.UUID, stageName string) (*Step, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var step Step
	var metaStr string
	err = p.QueryRow(ctx,
		`SELECT id, pipeline_job_id, stage_name, COALESCE(display_name,''), status,
		        COALESCE(progress,0), started_at, completed_at, COALESCE(duration_ms,0),
		        COALESCE(error_message,''), retry_count, max_retries, COALESCE(output,''),
		        COALESCE(metadata::text,'{}'), created_at, updated_at
		 FROM article_pipeline_steps WHERE pipeline_job_id = $1 AND stage_name = $2`,
		jobID, stageName,
	).Scan(&step.ID, &step.PipelineJobID, &step.StageName, &step.DisplayName,
		&step.Status, &step.Progress, &step.StartedAt, &step.CompletedAt, &step.DurationMs,
		&step.ErrorMessage, &step.RetryCount, &step.MaxRetries, &step.Output,
		&metaStr, &step.CreatedAt, &step.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrStageNotFound
		}
		return nil, fmt.Errorf("failed to get stage: %w", err)
	}
	if len(metaStr) > 0 {
		step.Metadata = parseJSON(metaStr)
	}
	if step.Metadata == nil {
		step.Metadata = make(map[string]interface{})
	}
	return &step, nil
}

func (s *Service) getMetrics(ctx context.Context, jobID uuid.UUID) ([]Metric, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, pipeline_job_id, COALESCE(stage_name,''), metric_name, metric_value,
		        COALESCE(metric_unit,''), COALESCE(metadata::text,'{}'), recorded_at
		 FROM article_pipeline_metrics WHERE pipeline_job_id = $1 ORDER BY recorded_at ASC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []Metric
	for rows.Next() {
		var m Metric
		var metaStr string
		if err := rows.Scan(&m.ID, &m.PipelineJobID, &m.StageName, &m.MetricName,
			&m.MetricValue, &m.MetricUnit, &metaStr, &m.RecordedAt); err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}
		if len(metaStr) > 0 {
			m.Metadata = parseJSON(metaStr)
		}
		metrics = append(metrics, m)
	}
	if metrics == nil {
		metrics = []Metric{}
	}
	return metrics, nil
}

func (s *Service) getQualityReports(ctx context.Context, jobID uuid.UUID) ([]QualityReport, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, pipeline_job_id, stage_name, status, score,
		        checks_passed, checks_failed, checks_total,
		        COALESCE(details::text,'[]'), COALESCE(summary,''), created_at
		 FROM article_quality_reports WHERE pipeline_job_id = $1 ORDER BY created_at ASC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query quality reports: %w", err)
	}
	defer rows.Close()

	var reports []QualityReport
	for rows.Next() {
		var r QualityReport
		var detailsStr string
		if err := rows.Scan(&r.ID, &r.PipelineJobID, &r.StageName, &r.Status,
			&r.Score, &r.ChecksPassed, &r.ChecksFailed, &r.ChecksTotal,
			&detailsStr, &r.Summary, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan quality report: %w", err)
		}
		if len(detailsStr) > 2 {
			r.Details = parseChecksJSON(detailsStr)
		}
		reports = append(reports, r)
	}
	if reports == nil {
		reports = []QualityReport{}
	}
	return reports, nil
}

func (s *Service) calcProgress(ctx context.Context, p database.Pool, jobID uuid.UUID) float64 {
	var total, completed int
	_ = p.QueryRow(ctx,
		`SELECT COUNT(*), COUNT(*) FILTER (WHERE status = $1)
		 FROM article_pipeline_steps WHERE pipeline_job_id = $2`,
		StepStatusCompleted, jobID,
	).Scan(&total, &completed)

	if total == 0 {
		return 0
	}
	return float64(completed) / float64(total) * 100
}

func nextStageName(current string) string {
	for i, stage := range AllStages {
		if string(stage) == current && i+1 < len(AllStages) {
			return string(AllStages[i+1])
		}
	}
	return ""
}

func coalesceIntPtr(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
