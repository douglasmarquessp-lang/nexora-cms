package autocontent

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

func (s *Service) CreateJob(ctx context.Context, siteID, userID uuid.UUID, req CreateJobRequest) (*AutocontentJob, error) {
	if req.Topic == "" {
		return nil, ErrInvalidTopic
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}
	if lang != "pt" && lang != "en" {
		return nil, ErrInvalidLanguage
	}
	priority := req.Priority
	if priority < 1 || priority > 10 {
		priority = 5
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	jobID := uuid.New()
	userIDVal := userID

	_, err = p.Exec(ctx,
		`INSERT INTO autocontent_jobs (id, site_id, user_id, topic, title, content_type, language,
		 target_language, status, priority, tone, audience, keywords, style_slug, template_id,
		 scheduled_for, max_retries, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'draft',$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$18)`,
		jobID, siteID, &userIDVal, req.Topic, req.Title, coalesceStr(req.ContentType, "article"),
		lang, req.TargetLanguage, priority, req.Tone, req.Audience, req.Keywords,
		req.StyleSlug, req.TemplateID, req.ScheduledFor, 3, &userIDVal, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create autocontent job: %w", err)
	}

	for _, step := range AllWorkflowSteps {
		stepID := uuid.New()
		deps := StepDependencies[step]
		depStrs := make([]string, len(deps))
		for i, d := range deps {
			depStrs[i] = string(d)
		}
		_, err = p.Exec(ctx,
			`INSERT INTO autocontent_steps (id, autocontent_job_id, step_name, display_name, status, depends_on, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,'pending',$5,$6,$6)`,
			stepID, jobID, string(step), StepDisplayNames[step], depStrs, now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create step %s: %w", step, err)
		}
	}

	s.addLog(ctx, p, jobID, "", "info", "autocontent job created", nil, 0)

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("autocontent.job.created"),
		EntityType: "autocontent_job",
		EntityID:   &jobID,
		Payload:    map[string]interface{}{"topic": req.Topic, "language": lang},
	})

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) GetJob(ctx context.Context, siteID, jobID uuid.UUID) (*AutocontentJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var j AutocontentJob
	err = p.QueryRow(ctx,
		`SELECT id, site_id, user_id, topic, COALESCE(title,''), COALESCE(content_type,'article'),
		        language, COALESCE(target_language,''), status, COALESCE(current_step,''),
		        COALESCE(progress,0), priority, COALESCE(word_count,0), COALESCE(tone,''),
		        COALESCE(audience,''), COALESCE(keywords,'{}'), COALESCE(style_slug,''),
		        template_id, scheduled_for, COALESCE(error_message,''), COALESCE(retry_count,0),
		        COALESCE(max_retries,3), started_at, completed_at, cancelled_at, created_by, created_at, updated_at
		 FROM autocontent_jobs WHERE id = $1 AND site_id = $2`,
		jobID, siteID,
	).Scan(&j.ID, &j.SiteID, &j.UserID, &j.Topic, &j.Title, &j.ContentType,
		&j.Language, &j.TargetLanguage, &j.Status, &j.CurrentStep,
		&j.Progress, &j.Priority, &j.WordCount, &j.Tone, &j.Audience, &j.Keywords,
		&j.StyleSlug, &j.TemplateID, &j.ScheduledFor, &j.ErrorMessage, &j.RetryCount,
		&j.MaxRetries, &j.StartedAt, &j.CompletedAt, &j.CancelledAt, &j.CreatedBy, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get autocontent job: %w", err)
	}
	return &j, nil
}

func (s *Service) GetJobDetail(ctx context.Context, siteID, jobID uuid.UUID) (*AutocontentJob, error) {
	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	steps, _ := s.GetSteps(ctx, jobID)
	job.Steps = steps

	results, _ := s.GetResults(ctx, jobID)
	job.Results = results

	return job, nil
}

func (s *Service) ListJobs(ctx context.Context, siteID uuid.UUID, status, language, step string, limit, offset int) ([]AutocontentJob, error) {
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
	if step != "" {
		where = append(where, fmt.Sprintf("current_step = $%d", argIdx))
		args = append(args, step)
		argIdx++
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, user_id, topic, COALESCE(title,''), COALESCE(content_type,'article'),
		        language, COALESCE(target_language,''), status, COALESCE(current_step,''),
		        COALESCE(progress,0), priority, COALESCE(word_count,0), COALESCE(tone,''),
		        COALESCE(audience,''), COALESCE(keywords,'{}'), COALESCE(style_slug,''),
		        template_id, scheduled_for, COALESCE(error_message,''), COALESCE(retry_count,0),
		        COALESCE(max_retries,3), started_at, completed_at, cancelled_at, created_by, created_at, updated_at
		 FROM autocontent_jobs WHERE %s ORDER BY priority ASC, created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list autocontent jobs: %w", err)
	}
	defer rows.Close()

	var jobs []AutocontentJob
	for rows.Next() {
		var j AutocontentJob
		if err := rows.Scan(&j.ID, &j.SiteID, &j.UserID, &j.Topic, &j.Title, &j.ContentType,
			&j.Language, &j.TargetLanguage, &j.Status, &j.CurrentStep,
			&j.Progress, &j.Priority, &j.WordCount, &j.Tone, &j.Audience, &j.Keywords,
			&j.StyleSlug, &j.TemplateID, &j.ScheduledFor, &j.ErrorMessage, &j.RetryCount,
			&j.MaxRetries, &j.StartedAt, &j.CompletedAt, &j.CancelledAt, &j.CreatedBy, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan autocontent job: %w", err)
		}
		jobs = append(jobs, j)
	}
	if jobs == nil {
		jobs = []AutocontentJob{}
	}
	return jobs, nil
}

func (s *Service) UpdateJob(ctx context.Context, siteID, jobID uuid.UUID, req UpdateJobRequest) (*AutocontentJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}
	if existing.Status == JobStatusCompleted {
		return nil, ErrJobAlreadyCompleted
	}
	if existing.Status == JobStatusCancelled {
		return nil, ErrJobAlreadyCancelled
	}
	if existing.Status == JobStatusRunning {
		return nil, ErrJobAlreadyRunning
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Priority != nil {
		prio := *req.Priority
		if prio < 1 || prio > 10 {
			return nil, fmt.Errorf("priority must be 1-10")
		}
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, prio)
		argIdx++
	}
	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.ContentType != nil {
		setClauses = append(setClauses, fmt.Sprintf("content_type = $%d", argIdx))
		args = append(args, *req.ContentType)
		argIdx++
	}
	if req.TargetLanguage != nil {
		setClauses = append(setClauses, fmt.Sprintf("target_language = $%d", argIdx))
		args = append(args, *req.TargetLanguage)
		argIdx++
	}
	if req.WordCount != nil {
		setClauses = append(setClauses, fmt.Sprintf("word_count = $%d", argIdx))
		args = append(args, *req.WordCount)
		argIdx++
	}
	if req.Tone != nil {
		setClauses = append(setClauses, fmt.Sprintf("tone = $%d", argIdx))
		args = append(args, *req.Tone)
		argIdx++
	}
	if req.Audience != nil {
		setClauses = append(setClauses, fmt.Sprintf("audience = $%d", argIdx))
		args = append(args, *req.Audience)
		argIdx++
	}
	if req.Keywords != nil {
		setClauses = append(setClauses, fmt.Sprintf("keywords = $%d", argIdx))
		args = append(args, *req.Keywords)
		argIdx++
	}
	if req.StyleSlug != nil {
		setClauses = append(setClauses, fmt.Sprintf("style_slug = $%d", argIdx))
		args = append(args, *req.StyleSlug)
		argIdx++
	}
	if req.ScheduledFor != nil {
		setClauses = append(setClauses, fmt.Sprintf("scheduled_for = $%d", argIdx))
		args = append(args, *req.ScheduledFor)
		argIdx++
	}

	if len(setClauses) == 0 {
		return existing, nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE autocontent_jobs SET %s WHERE id = $%d AND site_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, jobID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update autocontent job: %w", err)
	}

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) DeleteJob(ctx context.Context, siteID, jobID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	tag, err := p.Exec(ctx,
		`DELETE FROM autocontent_jobs WHERE id = $1 AND site_id = $2`,
		jobID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete autocontent job: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotFound
	}

	s.fireEvent(ctx, EventAutoCancelled, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
		"reason":  "deleted",
}, siteID)
	return nil
}

// --- Workflow Engine ---

func (s *Service) StartJob(ctx context.Context, siteID, jobID uuid.UUID) (*AutocontentJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	switch job.Status {
	case JobStatusRunning:
		return nil, ErrJobAlreadyRunning
	case JobStatusCompleted:
		return nil, ErrJobAlreadyCompleted
	case JobStatusCancelled:
		return nil, ErrJobAlreadyCancelled
	}

	now := time.Now()
	firstStep := string(AllWorkflowSteps[0])

	_, err = p.Exec(ctx,
		`UPDATE autocontent_jobs SET status = 'running', started_at = $1, current_step = $2, updated_at = $1
		 WHERE id = $3 AND site_id = $4`,
		now, firstStep, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start autocontent job: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE autocontent_steps SET status = 'running', started_at = $1, updated_at = $1
		 WHERE autocontent_job_id = $2 AND step_name = $3`,
		now, jobID, firstStep,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start first step: %w", err)
	}

	s.addLog(ctx, p, jobID, firstStep, "info", "autocontent job started", nil, 0)
	s.fireEvent(ctx, EventAutoStarted, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
		"step":    firstStep,
}, siteID)

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) PauseJob(ctx context.Context, siteID, jobID uuid.UUID) (*AutocontentJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != JobStatusRunning {
		return nil, ErrJobNotRunning
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`UPDATE autocontent_jobs SET status = 'paused', updated_at = $1 WHERE id = $2 AND site_id = $3`,
		now, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pause autocontent job: %w", err)
	}

	s.addLog(ctx, p, jobID, job.CurrentStep, "info", "autocontent job paused", nil, 0)
	s.fireEvent(ctx, EventAutoPaused, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
		"step":    job.CurrentStep,
}, siteID)

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) ResumeJob(ctx context.Context, siteID, jobID uuid.UUID) (*AutocontentJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != JobStatusPaused {
		return nil, ErrJobPaused
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`UPDATE autocontent_jobs SET status = 'running', updated_at = $1 WHERE id = $2 AND site_id = $3`,
		now, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resume autocontent job: %w", err)
	}

	s.addLog(ctx, p, jobID, job.CurrentStep, "info", "autocontent job resumed", nil, 0)
	s.fireEvent(ctx, EventAutoResumed, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
		"step":    job.CurrentStep,
}, siteID)

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) CancelJob(ctx context.Context, siteID, jobID uuid.UUID, reason string) (*AutocontentJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status == JobStatusCompleted {
		return nil, ErrJobAlreadyCompleted
	}
	if job.Status == JobStatusCancelled {
		return nil, ErrJobAlreadyCancelled
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`UPDATE autocontent_jobs SET status = 'cancelled', cancelled_at = $1, error_message = $2, updated_at = $1
		 WHERE id = $3 AND site_id = $4`,
		now, reason, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel autocontent job: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE autocontent_steps SET status = 'cancelled', updated_at = $1
		 WHERE autocontent_job_id = $2 AND status = 'pending'`,
		now, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel pending steps: %w", err)
	}

	if job.CurrentStep != "" {
		_, err = p.Exec(ctx,
			`UPDATE autocontent_steps SET status = 'cancelled', error_message = $1, updated_at = $2
			 WHERE autocontent_job_id = $3 AND step_name = $4 AND status = 'running'`,
			reason, now, jobID, job.CurrentStep,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to cancel current step: %w", err)
		}
	}

	s.addLog(ctx, p, jobID, job.CurrentStep, "warning", fmt.Sprintf("autocontent job cancelled: %s", reason), nil, 0)
	s.fireEvent(ctx, EventAutoCancelled, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
		"reason":  reason,
		"step":    job.CurrentStep,
}, siteID)

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) RetryStep(ctx context.Context, siteID, jobID uuid.UUID, req RetryStepRequest) (*AutocontentJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	stepName := req.StepName

	step, err := s.getStepByName(ctx, p, jobID, stepName)
	if err != nil {
		return nil, err
	}

	if job.RetryCount >= job.MaxRetries {
		return nil, ErrMaxRetriesExceeded
	}
	if step.RetryCount >= step.MaxRetries {
		return nil, ErrMaxRetriesExceeded
	}

	now := time.Now()
	newRetryCount := job.RetryCount + 1

	_, err = p.Exec(ctx,
		`UPDATE autocontent_jobs SET status = 'running', retry_count = $1, error_message = '', updated_at = $2
		 WHERE id = $3 AND site_id = $4`,
		newRetryCount, now, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update retry status: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE autocontent_steps SET status = 'running', progress = 0, error_message = '',
		 retry_count = retry_count + 1, started_at = $1, completed_at = NULL, duration_ms = 0, updated_at = $1
		 WHERE autocontent_job_id = $2 AND step_name = $3`,
		now, jobID, stepName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reset step for retry: %w", err)
	}

	s.addLog(ctx, p, jobID, stepName, "info", fmt.Sprintf("step retry #%d", newRetryCount), nil, 0)
	s.fireEvent(ctx, EventAutoRetry, map[string]interface{}{
		"job_id":      jobID.String(),
		"site_id":     siteID.String(),
		"step":        stepName,
		"retry_count": newRetryCount,
}, siteID)

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) RestartJob(ctx context.Context, siteID, jobID uuid.UUID) (*AutocontentJob, error) {
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
		`UPDATE autocontent_jobs SET status = 'draft', progress = 0, current_step = '',
		 error_message = '', retry_count = 0, started_at = NULL, completed_at = NULL,
		 cancelled_at = NULL, updated_at = $1
		 WHERE id = $2 AND site_id = $3`,
		now, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to restart autocontent job: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE autocontent_steps SET status = 'pending', progress = 0, started_at = NULL,
		 completed_at = NULL, duration_ms = 0, error_message = '', retry_count = 0, updated_at = $1
		 WHERE autocontent_job_id = $2`,
		now, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reset steps: %w", err)
	}

	s.addLog(ctx, p, jobID, "", "info", "autocontent job restarted", nil, 0)

	return s.GetJob(ctx, siteID, jobID)
}

// --- Step Management ---

func (s *Service) GetSteps(ctx context.Context, jobID uuid.UUID) ([]Step, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	return s.listSteps(ctx, p, jobID)
}

func (s *Service) listSteps(ctx context.Context, p database.Pool, jobID uuid.UUID) ([]Step, error) {
	rows, err := p.Query(ctx,
		`SELECT id, autocontent_job_id, step_name, COALESCE(display_name,''), status,
		        COALESCE(progress,0), COALESCE(depends_on,'{}'), COALESCE(retry_count,0),
		        COALESCE(max_retries,3), started_at, completed_at, COALESCE(duration_ms,0),
		        COALESCE(error_message,''), COALESCE(metadata::text,'{}'), created_at, updated_at
		 FROM autocontent_steps WHERE autocontent_job_id = $1 ORDER BY created_at ASC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list steps: %w", err)
	}
	defer rows.Close()

	var steps []Step
	for rows.Next() {
		var s Step
		var metadataStr string
		if err := rows.Scan(&s.ID, &s.AutocontentJobID, &s.StepName, &s.DisplayName, &s.Status,
			&s.Progress, &s.DependsOn, &s.RetryCount, &s.MaxRetries,
			&s.StartedAt, &s.CompletedAt, &s.DurationMs,
			&s.ErrorMessage, &metadataStr, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan step: %w", err)
		}
		if len(metadataStr) > 0 {
			_ = json.Unmarshal([]byte(metadataStr), &s.Metadata)
		}
		if s.Metadata == nil {
			s.Metadata = make(map[string]interface{})
		}
		steps = append(steps, s)
	}
	if steps == nil {
		steps = []Step{}
	}
	return steps, nil
}

func (s *Service) getStepByName(ctx context.Context, p database.Pool, jobID uuid.UUID, stepName string) (*Step, error) {
	var step Step
	var metadataStr string
	err := p.QueryRow(ctx,
		`SELECT id, autocontent_job_id, step_name, COALESCE(display_name,''), status,
		        COALESCE(progress,0), COALESCE(depends_on,'{}'), COALESCE(retry_count,0),
		        COALESCE(max_retries,3), started_at, completed_at, COALESCE(duration_ms,0),
		        COALESCE(error_message,''), COALESCE(metadata::text,'{}'), created_at, updated_at
		 FROM autocontent_steps WHERE autocontent_job_id = $1 AND step_name = $2`,
		jobID, stepName,
	).Scan(&step.ID, &step.AutocontentJobID, &step.StepName, &step.DisplayName, &step.Status,
		&step.Progress, &step.DependsOn, &step.RetryCount, &step.MaxRetries,
		&step.StartedAt, &step.CompletedAt, &step.DurationMs,
		&step.ErrorMessage, &metadataStr, &step.CreatedAt, &step.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrStepNotFound
		}
		return nil, fmt.Errorf("failed to get step: %w", err)
	}
	if len(metadataStr) > 0 {
		_ = json.Unmarshal([]byte(metadataStr), &step.Metadata)
	}
	if step.Metadata == nil {
		step.Metadata = make(map[string]interface{})
	}
	return &step, nil
}

func (s *Service) UpdateStep(ctx context.Context, jobID uuid.UUID, stepName string, status StepStatus, progress float64, metadata map[string]interface{}, errorMsg string, durationMs int64) (*Step, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	step, err := s.getStepByName(ctx, p, jobID, stepName)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	setClauses := []string{fmt.Sprintf("status = '%s'", status)}
	args := []interface{}{}
	argIdx := 1

	if status == StepStatusRunning {
		setClauses = append(setClauses, fmt.Sprintf("started_at = $%d", argIdx))
		args = append(args, now)
		argIdx++
	}
	if status == StepStatusCompleted || status == StepStatusFailed {
		setClauses = append(setClauses, fmt.Sprintf("completed_at = $%d", argIdx))
		args = append(args, now)
		argIdx++
		setClauses = append(setClauses, fmt.Sprintf("duration_ms = $%d", argIdx))
		if durationMs > 0 {
			args = append(args, durationMs)
		} else if step.StartedAt != nil {
			args = append(args, now.Sub(*step.StartedAt).Milliseconds())
		} else {
			args = append(args, int64(0))
		}
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
		`UPDATE autocontent_steps SET %s WHERE autocontent_job_id = $%d AND step_name = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, jobID, stepName)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update step: %w", err)
	}

	if status == StepStatusCompleted {
		s.onStepCompleted(ctx, p, jobID, stepName, now, metadata)
	}
	if status == StepStatusFailed {
		s.onStepFailed(ctx, p, jobID, stepName, errorMsg, now)
	}

	return s.getStepByName(ctx, p, jobID, stepName)
}

func (s *Service) onStepCompleted(ctx context.Context, p database.Pool, jobID uuid.UUID, stepName string, now time.Time, metadata map[string]interface{}) {
	nextStep := s.nextStep(stepName)
	if nextStep != "" {
		deps := StepDependencies[WorkflowStep(nextStep)]
		allMet := true
		for _, dep := range deps {
			depStep, err := s.getStepByName(ctx, p, jobID, string(dep))
			if err != nil || depStep.Status != StepStatusCompleted {
				allMet = false
				break
			}
		}
		if allMet {
			_, err := p.Exec(ctx,
				`UPDATE autocontent_steps SET status = 'pending', updated_at = $1
				 WHERE autocontent_job_id = $2 AND step_name = $3 AND status = 'pending'`,
				now, jobID, nextStep,
			)
			if err == nil {
				_, _ = p.Exec(ctx,
					`UPDATE autocontent_jobs SET current_step = $1, updated_at = $2 WHERE id = $3`,
					nextStep, now, jobID,
				)
				s.fireEvent(ctx, EventAutoProgress, map[string]interface{}{
					"job_id":   jobID.String(),
					"step":     nextStep,
					"progress": s.calcProgress(ctx, p, jobID),
				}, uuid.Nil)
			}
		}
	} else {
		_, _ = p.Exec(ctx,
			`UPDATE autocontent_jobs SET status = 'completed', progress = 100, completed_at = $1, updated_at = $1
			 WHERE id = $2`,
			now, jobID,
		)
		s.fireEvent(ctx, EventAutoCompleted, map[string]interface{}{
			"job_id": jobID.String(),
		}, uuid.Nil)
	}

	s.addLog(ctx, p, jobID, stepName, "info", fmt.Sprintf("step completed: %s", StepDisplayNames[WorkflowStep(stepName)]), nil, 0)
}

func (s *Service) onStepFailed(ctx context.Context, p database.Pool, jobID uuid.UUID, stepName string, errorMsg string, now time.Time) {
	_, _ = p.Exec(ctx,
		`UPDATE autocontent_jobs SET status = 'failed', error_message = $1, updated_at = $2 WHERE id = $3`,
		errorMsg, now, jobID,
	)
	s.addLog(ctx, p, jobID, stepName, "error", fmt.Sprintf("step failed: %s - %s", stepName, errorMsg), nil, 0)
	s.fireEvent(ctx, EventAutoStepFailed, map[string]interface{}{
		"job_id":  jobID.String(),
		"step":    stepName,
		"error":   errorMsg,
	}, uuid.Nil)
}

func (s *Service) nextStep(current string) string {
	for i, step := range AllWorkflowSteps {
		if string(step) == current && i+1 < len(AllWorkflowSteps) {
			return string(AllWorkflowSteps[i+1])
		}
	}
	return ""
}

func (s *Service) calcProgress(ctx context.Context, p database.Pool, jobID uuid.UUID) float64 {
	steps, err := s.listSteps(ctx, p, jobID)
	if err != nil || len(steps) == 0 {
		return 0
	}
	completed := 0
	for _, st := range steps {
		if st.Status == StepStatusCompleted {
			completed++
		}
	}
	return float64(completed) / float64(len(steps)) * 100
}

// --- Results ---

func (s *Service) SaveResult(ctx context.Context, jobID uuid.UUID, stepName, content, summary string, score float64, passed bool, data map[string]interface{}) (*Result, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	resultID := uuid.New()

	dataJSON, _ := json.Marshal(data)

	_, err = p.Exec(ctx,
		`INSERT INTO autocontent_results (id, autocontent_job_id, step_name, content, summary, score, passed, data, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$9)`,
		resultID, jobID, stepName, content, summary, score, passed, string(dataJSON), now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save result: %w", err)
	}

	return &Result{
		ID:              resultID,
		AutocontentJobID: jobID,
		StepName:        stepName,
		Content:         content,
		Summary:         summary,
		Score:           score,
		Passed:          passed,
		Data:            data,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func (s *Service) GetResults(ctx context.Context, jobID uuid.UUID) ([]Result, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, autocontent_job_id, step_name, COALESCE(content,''), COALESCE(summary,''),
		        COALESCE(score,0), COALESCE(passed,false), COALESCE(data::text,'{}'), created_at, updated_at
		 FROM autocontent_results WHERE autocontent_job_id = $1 ORDER BY created_at ASC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list results: %w", err)
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var r Result
		var dataStr string
		if err := rows.Scan(&r.ID, &r.AutocontentJobID, &r.StepName, &r.Content, &r.Summary,
			&r.Score, &r.Passed, &dataStr, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		if len(dataStr) > 0 {
			_ = json.Unmarshal([]byte(dataStr), &r.Data)
		}
		if r.Data == nil {
			r.Data = make(map[string]interface{})
		}
		results = append(results, r)
	}
	if results == nil {
		results = []Result{}
	}
	return results, nil
}

func (s *Service) GetResultByStep(ctx context.Context, jobID uuid.UUID, stepName string) (*Result, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var r Result
	var dataStr string
	err = p.QueryRow(ctx,
		`SELECT id, autocontent_job_id, step_name, COALESCE(content,''), COALESCE(summary,''),
		        COALESCE(score,0), COALESCE(passed,false), COALESCE(data::text,'{}'), created_at, updated_at
		 FROM autocontent_results WHERE autocontent_job_id = $1 AND step_name = $2 ORDER BY created_at DESC LIMIT 1`,
		jobID, stepName,
	).Scan(&r.ID, &r.AutocontentJobID, &r.StepName, &r.Content, &r.Summary,
		&r.Score, &r.Passed, &dataStr, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrResultNotFound
		}
		return nil, fmt.Errorf("failed to get result: %w", err)
	}
	if len(dataStr) > 0 {
		_ = json.Unmarshal([]byte(dataStr), &r.Data)
	}
	if r.Data == nil {
		r.Data = make(map[string]interface{})
	}
	return &r, nil
}

// --- Queue Management ---

func (s *Service) AddToQueue(ctx context.Context, siteID uuid.UUID, req QueueRequest) (*PublicationItem, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if req.Language != "pt" && req.Language != "en" {
		return nil, ErrInvalidLanguage
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	itemID := uuid.New()
	priority := req.Priority
	if priority < 1 || priority > 10 {
		priority = 5
	}

	_, err = p.Exec(ctx,
		`INSERT INTO publication_queue (id, site_id, autocontent_job_id, title, content, excerpt,
		 language, status, priority, scheduled_for, meta_title, meta_description, slug,
		 featured_image_url, tags, categories, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'pending',$8,$9,$10,$11,$12,$13,$14,$15,$16,$16)`,
		itemID, siteID, req.AutocontentJobID, req.Title, req.Content, req.Excerpt,
		req.Language, priority, req.ScheduledFor, req.MetaTitle, req.MetaDescription,
		req.Slug, req.FeaturedImageURL, req.Tags, req.Categories, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add to publication queue: %w", err)
	}

	s.fireEvent(ctx, EventAutoQueued, map[string]interface{}{
		"queue_item_id": itemID.String(),
		"site_id":       siteID.String(),
		"title":         req.Title,
}, siteID)

	return &PublicationItem{
		ID:          itemID,
		SiteID:      siteID,
		Title:       req.Title,
		Content:     req.Content,
		Excerpt:     req.Excerpt,
		Language:    req.Language,
		Status:      QueuePending,
		Priority:    priority,
		ScheduledFor: req.ScheduledFor,
		MetaTitle:   req.MetaTitle,
		MetaDescription: req.MetaDescription,
		Slug:        req.Slug,
		Tags:        req.Tags,
		Categories:  req.Categories,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (s *Service) ListQueue(ctx context.Context, siteID uuid.UUID, status string, limit, offset int) ([]PublicationItem, error) {
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
		`SELECT id, site_id, autocontent_job_id, title, COALESCE(content,''), COALESCE(excerpt,''),
		        language, status, priority, scheduled_for, COALESCE(meta_title,''),
		        COALESCE(meta_description,''), COALESCE(slug,''), COALESCE(featured_image_url,''),
		        COALESCE(tags,'{}'), COALESCE(categories,'{}'),
		        published_at, published_by, COALESCE(error_message,''), created_at, updated_at
		 FROM publication_queue WHERE %s ORDER BY priority ASC, created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list publication queue: %w", err)
	}
	defer rows.Close()

	var items []PublicationItem
	for rows.Next() {
		var item PublicationItem
		if err := rows.Scan(&item.ID, &item.SiteID, &item.AutocontentJobID, &item.Title,
			&item.Content, &item.Excerpt, &item.Language, &item.Status, &item.Priority,
			&item.ScheduledFor, &item.MetaTitle, &item.MetaDescription, &item.Slug,
			&item.FeaturedImageURL, &item.Tags, &item.Categories,
			&item.PublishedAt, &item.PublishedBy, &item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan queue item: %w", err)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []PublicationItem{}
	}
	return items, nil
}

func (s *Service) UpdateQueueItem(ctx context.Context, siteID, itemID uuid.UUID, req UpdateQueueRequest) (*PublicationItem, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var exists bool
	err = p.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM publication_queue WHERE id = $1 AND site_id = $2)`,
		itemID, siteID,
	).Scan(&exists)
	if err != nil || !exists {
		return nil, ErrQueueItemNotFound
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*req.Status))
		argIdx++
	}
	if req.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, *req.Priority)
		argIdx++
	}
	if req.ScheduledFor != nil {
		setClauses = append(setClauses, fmt.Sprintf("scheduled_for = $%d", argIdx))
		args = append(args, *req.ScheduledFor)
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
	if req.FeaturedImageURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("featured_image_url = $%d", argIdx))
		args = append(args, *req.FeaturedImageURL)
		argIdx++
	}

	if len(setClauses) == 0 {
		return s.getQueueItem(ctx, p, itemID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE publication_queue SET %s WHERE id = $%d AND site_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, itemID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update queue item: %w", err)
	}

	return s.getQueueItem(ctx, p, itemID)
}

func (s *Service) getQueueItem(ctx context.Context, p database.Pool, itemID uuid.UUID) (*PublicationItem, error) {
	var item PublicationItem
	err := p.QueryRow(ctx,
		`SELECT id, site_id, autocontent_job_id, title, COALESCE(content,''), COALESCE(excerpt,''),
		        language, status, priority, scheduled_for, COALESCE(meta_title,''),
		        COALESCE(meta_description,''), COALESCE(slug,''), COALESCE(featured_image_url,''),
		        COALESCE(tags,'{}'), COALESCE(categories,'{}'),
		        published_at, published_by, COALESCE(error_message,''), created_at, updated_at
		 FROM publication_queue WHERE id = $1`,
		itemID,
	).Scan(&item.ID, &item.SiteID, &item.AutocontentJobID, &item.Title,
		&item.Content, &item.Excerpt, &item.Language, &item.Status, &item.Priority,
		&item.ScheduledFor, &item.MetaTitle, &item.MetaDescription, &item.Slug,
		&item.FeaturedImageURL, &item.Tags, &item.Categories,
		&item.PublishedAt, &item.PublishedBy, &item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrQueueItemNotFound
		}
		return nil, fmt.Errorf("failed to get queue item: %w", err)
	}
	return &item, nil
}

// --- Templates ---

func (s *Service) CreateTemplate(ctx context.Context, siteID uuid.UUID, req CreateTemplateRequest) (*WorkflowTemplate, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("template name is required")
	}
	if len(req.Steps) == 0 {
		return nil, fmt.Errorf("at least one step is required")
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	tmplID := uuid.New()

	stepsJSON, _ := json.Marshal(req.Steps)

	if req.IsDefault {
		_, _ = p.Exec(ctx,
			`UPDATE workflow_templates SET is_default = false WHERE site_id = $1`,
			siteID,
		)
	}

	_, err = p.Exec(ctx,
		`INSERT INTO workflow_templates (id, site_id, name, description, steps, is_default, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5::jsonb,$6,$7,$7)`,
		tmplID, siteID, req.Name, req.Description, string(stepsJSON), req.IsDefault, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return &WorkflowTemplate{
		ID:          tmplID,
		SiteID:      siteID,
		Name:        req.Name,
		Description: req.Description,
		Steps:       req.Steps,
		IsDefault:   req.IsDefault,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (s *Service) ListTemplates(ctx context.Context, siteID uuid.UUID) ([]WorkflowTemplate, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, site_id, name, COALESCE(description,''), COALESCE(steps::text,'[]'), is_default, is_active, created_by, created_at, updated_at
		 FROM workflow_templates WHERE site_id = $1 ORDER BY is_default DESC, name ASC`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []WorkflowTemplate
	for rows.Next() {
		var t WorkflowTemplate
		var stepsStr string
		if err := rows.Scan(&t.ID, &t.SiteID, &t.Name, &t.Description, &stepsStr,
			&t.IsDefault, &t.IsActive, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		if len(stepsStr) > 0 {
			_ = json.Unmarshal([]byte(stepsStr), &t.Steps)
		}
		if t.Steps == nil {
			t.Steps = []map[string]interface{}{}
		}
		templates = append(templates, t)
	}
	if templates == nil {
		templates = []WorkflowTemplate{}
	}
	return templates, nil
}

// --- Metrics / Stats ---

func (s *Service) GetMetrics(ctx context.Context, siteID uuid.UUID) (*AutocontentMetrics, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var m AutocontentMetrics

	err = p.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*),0),
		        COALESCE(SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN status = 'paused' THEN 1 ELSE 0 END),0),
		        COALESCE(AVG(CASE WHEN status = 'completed' THEN EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000 ELSE NULL END),0),
		        COALESCE((SELECT COUNT(*) FROM publication_queue WHERE site_id = $1 AND status = 'pending'),0)
		 FROM autocontent_jobs WHERE site_id = $1`,
		siteID,
	).Scan(&m.TotalJobs, &m.RunningJobs, &m.CompletedJobs, &m.FailedJobs, &m.PausedJobs,
		&m.AvgDuration, &m.QueueSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	if m.TotalJobs > 0 {
		m.AvgSuccessRate = float64(m.CompletedJobs) / float64(m.TotalJobs) * 100
	}

	return &m, nil
}

func (s *Service) GetStats(ctx context.Context, siteID uuid.UUID) (*AutocontentStats, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	stats := &AutocontentStats{
		ByStatus:   make(map[JobStatus]int64),
		ByLanguage: make(map[string]int64),
		ByStep:     make(map[string]int64),
	}

	rows, err := p.Query(ctx,
		`SELECT status, COUNT(*) FROM autocontent_jobs WHERE site_id = $1 GROUP BY status`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get status stats: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err == nil {
			stats.ByStatus[JobStatus(status)] = count
		}
	}

	rows2, err := p.Query(ctx,
		`SELECT language, COUNT(*) FROM autocontent_jobs WHERE site_id = $1 GROUP BY language`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get language stats: %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var lang string
		var count int64
		if err := rows2.Scan(&lang, &count); err == nil {
			stats.ByLanguage[lang] = count
		}
	}

	rows3, err := p.Query(ctx,
		`SELECT current_step, COUNT(*) FROM autocontent_jobs WHERE site_id = $1 AND current_step != '' GROUP BY current_step`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get step stats: %w", err)
	}
	defer rows3.Close()
	for rows3.Next() {
		var step string
		var count int64
		if err := rows3.Scan(&step, &count); err == nil {
			stats.ByStep[step] = count
		}
	}

	return stats, nil
}

// --- Logs ---

func (s *Service) addLog(ctx context.Context, p database.Pool, jobID uuid.UUID, step, level, message string, details map[string]interface{}, durationMs int64) {
	detailsJSON, _ := json.Marshal(details)
	_, err := p.Exec(ctx,
		`INSERT INTO generation_pipeline_logs (id, generation_job_id, stage, level, message, details, duration_ms, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8)`,
		uuid.New(), jobID, step, level, message, string(detailsJSON), durationMs, time.Now(),
	)
	if err != nil {
		s.log.Error("failed to add autocontent log", "error", err)
	}
}

// --- Helpers ---

func coalesceStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
