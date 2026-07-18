package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

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

func (s *Service) CreateJob(ctx context.Context, siteID, userID uuid.UUID, req CreateJobRequest) (*WorkflowJob, error) {
	if req.Title == "" {
		return nil, ErrInvalidTitle
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
	contentType := req.ContentType
	if contentType == "" {
		contentType = "article"
	}

	job := &WorkflowJob{
		ID:             jobID,
		SiteID:         siteID,
		UserID:         &userIDVal,
		Title:          req.Title,
		ContentType:    contentType,
		Language:       lang,
		TargetLanguage: req.TargetLanguage,
		Status:         JobStatusDraft,
		Priority:       priority,
		WordCount:      req.WordCount,
		Tone:           req.Tone,
		Audience:       req.Audience,
		Keywords:       req.Keywords,
		StyleSlug:      req.StyleSlug,
		SourceJobID:    req.SourceJobID,
		ScheduledFor:   req.ScheduledFor,
		MaxRetries:     3,
		GeneratePT:     req.GeneratePT,
		GenerateEN:     req.GenerateEN,
		CreatedBy:      &userIDVal,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.insertJob(ctx, p, job); err != nil {
		return nil, err
	}

	if err := s.insertSteps(ctx, p, jobID, now); err != nil {
		return nil, err
	}

	s.addLog(ctx, p, jobID, "", "info", "workflow job created", nil, 0)

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("workflow.job.created"),
		EntityType: "workflow_job",
		EntityID:   &jobID,
		Payload:    map[string]interface{}{"title": req.Title, "language": lang},
	})

	s.fireEvent(ctx, EventWorkflowCreated, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
		"title":   req.Title,
	}, siteID)

	return s.getJobByID(ctx, p, siteID, jobID)
}

func (s *Service) GetJob(ctx context.Context, siteID, jobID uuid.UUID) (*WorkflowJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	return s.getJobByID(ctx, p, siteID, jobID)
}

func (s *Service) GetJobDetail(ctx context.Context, siteID, jobID uuid.UUID) (*WorkflowJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.getJobByID(ctx, p, siteID, jobID)
	if err != nil {
		return nil, err
	}

	steps, _ := s.listSteps(ctx, p, jobID)
	job.Steps = steps

	return job, nil
}

func (s *Service) ListJobs(ctx context.Context, siteID uuid.UUID, status, language, step string, limit, offset int) ([]WorkflowJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	return s.listJobs(ctx, p, siteID, status, language, step, limit, offset)
}

func (s *Service) UpdateJob(ctx context.Context, siteID, jobID uuid.UUID, req UpdateJobRequest) (*WorkflowJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, err := s.getJobByID(ctx, p, siteID, jobID)
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
			return nil, ErrInvalidPriority
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
	if req.GeneratePT != nil {
		setClauses = append(setClauses, fmt.Sprintf("generate_pt = $%d", argIdx))
		args = append(args, *req.GeneratePT)
		argIdx++
	}
	if req.GenerateEN != nil {
		setClauses = append(setClauses, fmt.Sprintf("generate_en = $%d", argIdx))
		args = append(args, *req.GenerateEN)
	}

	if len(setClauses) == 0 {
		return existing, nil
	}

	if err := s.updateJobFields(ctx, p, jobID, siteID, setClauses, args); err != nil {
		return nil, err
	}

	return s.getJobByID(ctx, p, siteID, jobID)
}

func (s *Service) DeleteJob(ctx context.Context, siteID, jobID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	if err := s.deleteJobByID(ctx, p, siteID, jobID); err != nil {
		return err
	}

	s.fireEvent(ctx, EventWorkflowCancelled, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
		"reason":  "deleted",
	}, siteID)
	return nil
}

// --- Queue Engine ---

func (s *Service) AddToQueue(ctx context.Context, siteID uuid.UUID, req QueueRequest) (*QueueItem, error) {
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

	item := &QueueItem{
		ID:               itemID,
		SiteID:           siteID,
		WorkflowJobID:    req.WorkflowJobID,
		Title:            req.Title,
		Content:          req.Content,
		Excerpt:          req.Excerpt,
		Language:         req.Language,
		Status:           QueueStatusPending,
		Priority:         priority,
		ScheduledFor:     req.ScheduledFor,
		MaxRetries:       3,
		MetaTitle:        req.MetaTitle,
		MetaDescription:  req.MetaDescription,
		Slug:             req.Slug,
		FeaturedImageURL: req.FeaturedImageURL,
		Tags:             req.Tags,
		Categories:       req.Categories,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.insertQueueItem(ctx, p, item); err != nil {
		return nil, err
	}

	s.fireEvent(ctx, EventWorkflowQueued, map[string]interface{}{
		"queue_item_id": itemID.String(),
		"site_id":       siteID.String(),
		"title":         req.Title,
	}, siteID)

	return item, nil
}

func (s *Service) ListQueue(ctx context.Context, siteID uuid.UUID, status string, limit, offset int) ([]QueueItem, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	return s.listQueueItems(ctx, p, siteID, status, limit, offset)
}

func (s *Service) UpdateQueueItem(ctx context.Context, siteID, itemID uuid.UUID, req UpdateQueueRequest) (*QueueItem, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	_, err = s.getQueueItemByID(ctx, p, siteID, itemID)
	if err != nil {
		return nil, err
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
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
	if req.IsPaused != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_paused = $%d", argIdx))
		args = append(args, *req.IsPaused)
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
	}

	if len(setClauses) == 0 {
		return s.getQueueItemByID(ctx, p, siteID, itemID)
	}

	if err := s.updateQueueItemFields(ctx, p, itemID, siteID, setClauses, args); err != nil {
		return nil, err
	}

	return s.getQueueItemByID(ctx, p, siteID, itemID)
}

func (s *Service) PauseQueue(ctx context.Context, siteID, itemID uuid.UUID) (*QueueItem, error) {
	paused := true
	return s.UpdateQueueItem(ctx, siteID, itemID, UpdateQueueRequest{IsPaused: &paused})
}

func (s *Service) ResumeQueue(ctx context.Context, siteID, itemID uuid.UUID) (*QueueItem, error) {
	paused := false
	return s.UpdateQueueItem(ctx, siteID, itemID, UpdateQueueRequest{IsPaused: &paused})
}

func (s *Service) CancelQueue(ctx context.Context, siteID, itemID uuid.UUID) (*QueueItem, error) {
	status := string(QueueStatusCancelled)
	return s.UpdateQueueItem(ctx, siteID, itemID, UpdateQueueRequest{Status: &status})
}

func (s *Service) ProcessQueue(ctx context.Context, siteID uuid.UUID) (*QueueItem, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	item, err := s.getNextQueueItem(ctx, p, siteID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}

	running := string(QueueStatusRunning)
	_, err = s.UpdateQueueItem(ctx, siteID, item.ID, UpdateQueueRequest{Status: &running})
	if err != nil {
		return nil, err
	}

	s.fireEvent(ctx, EventWorkflowQueueProcessed, map[string]interface{}{
		"queue_item_id": item.ID.String(),
		"site_id":       siteID.String(),
		"title":         item.Title,
	}, siteID)

	return s.getQueueItemByID(ctx, p, siteID, item.ID)
}

// --- Workflow Engine ---

func (s *Service) StartJob(ctx context.Context, siteID, jobID uuid.UUID) (*WorkflowJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.getJobByID(ctx, p, siteID, jobID)
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
		`UPDATE workflow_jobs SET status = 'running', started_at = $1, current_step = $2, updated_at = $1
		 WHERE id = $3 AND site_id = $4`,
		now, firstStep, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow job: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE workflow_steps SET status = 'running', started_at = $1, updated_at = $1
		 WHERE workflow_job_id = $2 AND step_name = $3`,
		now, jobID, firstStep,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start first step: %w", err)
	}

	s.addHistory(ctx, p, siteID, &jobID, nil, "workflow.started", "job", &jobID,
		string(job.Status), string(JobStatusRunning), nil, "", &siteID, 0)
	s.addLog(ctx, p, jobID, firstStep, "info", "workflow job started", nil, 0)
	s.fireEvent(ctx, EventWorkflowStarted, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
		"step":    firstStep,
	}, siteID)

	return s.getJobByID(ctx, p, siteID, jobID)
}

func (s *Service) PauseJob(ctx context.Context, siteID, jobID uuid.UUID) (*WorkflowJob, error) {
	return s.transitionJobStatus(ctx, siteID, jobID,
		JobStatusRunning, ErrJobNotRunning,
		"paused", "workflow.paused", EventWorkflowPaused, "pause",
	)
}

func (s *Service) ResumeJob(ctx context.Context, siteID, jobID uuid.UUID) (*WorkflowJob, error) {
	return s.transitionJobStatus(ctx, siteID, jobID,
		JobStatusPaused, ErrJobPaused,
		"running", "workflow.resumed", EventWorkflowResumed, "resume",
	)
}

func (s *Service) transitionJobStatus(ctx context.Context, siteID, jobID uuid.UUID, requiredStatus JobStatus, statusErr error, newStatus, action string, event kernel.EventType, verb string) (*WorkflowJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.getJobByID(ctx, p, siteID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != requiredStatus {
		return nil, statusErr
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`UPDATE workflow_jobs SET status = $4, updated_at = $1 WHERE id = $2 AND site_id = $3`,
		now, jobID, siteID, newStatus,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to %s workflow job: %w", verb, err)
	}

	s.addHistory(ctx, p, siteID, &jobID, nil, action, "job", &jobID,
		string(requiredStatus), newStatus, nil, "", &siteID, 0)
	s.addLog(ctx, p, jobID, job.CurrentStep, "info", "workflow job "+verb+"d", nil, 0)
	s.fireEvent(ctx, event, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
		"step":    job.CurrentStep,
	}, siteID)

	return s.getJobByID(ctx, p, siteID, jobID)
}

func (s *Service) CancelJob(ctx context.Context, siteID, jobID uuid.UUID, reason string) (*WorkflowJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.getJobByID(ctx, p, siteID, jobID)
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
		`UPDATE workflow_jobs SET status = 'cancelled', cancelled_at = $1, error_message = $2, updated_at = $1
		 WHERE id = $3 AND site_id = $4`,
		now, reason, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel workflow job: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE workflow_steps SET status = 'cancelled', updated_at = $1
		 WHERE workflow_job_id = $2 AND status = 'pending'`,
		now, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel pending steps: %w", err)
	}

	if job.CurrentStep != "" {
		_, err = p.Exec(ctx,
			`UPDATE workflow_steps SET status = 'cancelled', error_message = $1, updated_at = $2
			 WHERE workflow_job_id = $3 AND step_name = $4 AND status = 'running'`,
			reason, now, jobID, job.CurrentStep,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to cancel current step: %w", err)
		}
	}

	s.addHistory(ctx, p, siteID, &jobID, nil, "workflow.cancelled", "job", &jobID,
		string(job.Status), string(JobStatusCancelled), nil, reason, &siteID, 0)
	s.addLog(ctx, p, jobID, job.CurrentStep, "warning", fmt.Sprintf("workflow job cancelled: %s", reason), nil, 0)
	s.fireEvent(ctx, EventWorkflowCancelled, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
		"reason":  reason,
		"step":    job.CurrentStep,
	}, siteID)

	return s.getJobByID(ctx, p, siteID, jobID)
}

func (s *Service) RetryStep(ctx context.Context, siteID, jobID uuid.UUID, req RetryRequest) (*WorkflowJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	job, err := s.getJobByID(ctx, p, siteID, jobID)
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
		`UPDATE workflow_jobs SET status = 'running', retry_count = $1, error_message = '', updated_at = $2
		 WHERE id = $3 AND site_id = $4`,
		newRetryCount, now, jobID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update retry status: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE workflow_steps SET status = 'running', progress = 0, error_message = '',
		 retry_count = retry_count + 1, started_at = $1, completed_at = NULL, duration_ms = 0, updated_at = $1
		 WHERE workflow_job_id = $2 AND step_name = $3`,
		now, jobID, stepName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reset step for retry: %w", err)
	}

	s.addHistory(ctx, p, siteID, &jobID, nil, "workflow.retry", "step", nil,
		string(StepStatusFailed), string(StepStatusRunning), nil, "", &siteID, 0)
	s.addLog(ctx, p, jobID, stepName, "info", fmt.Sprintf("step retry #%d", newRetryCount), nil, 0)
	s.fireEvent(ctx, EventWorkflowRetry, map[string]interface{}{
		"job_id":      jobID.String(),
		"site_id":     siteID.String(),
		"step":        stepName,
		"retry_count": newRetryCount,
	}, siteID)

	return s.getJobByID(ctx, p, siteID, jobID)
}

// --- Steps ---

func (s *Service) GetSteps(ctx context.Context, jobID uuid.UUID) ([]Step, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	return s.listSteps(ctx, p, jobID)
}

func (s *Service) AdvanceStep(ctx context.Context, jobID uuid.UUID, stepName string, status StepStatus, progress float64, metadata map[string]interface{}, errorMsg string, durationMs int64) (*Step, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	step, err := s.updateStep(ctx, p, jobID, stepName, status, progress, metadata, errorMsg, durationMs)
	if err != nil {
		return nil, err
	}

	if status == StepStatusCompleted {
		s.onStepCompleted(ctx, p, jobID, stepName, metadata)
	}
	if status == StepStatusFailed {
		s.onStepFailed(ctx, p, jobID, stepName, errorMsg)
	}

	return step, nil
}

func (s *Service) onStepCompleted(ctx context.Context, p database.Pool, jobID uuid.UUID, stepName string, metadata map[string]interface{}) {
	now := time.Now()
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
				`UPDATE workflow_steps SET status = 'pending', updated_at = $1
				 WHERE workflow_job_id = $2 AND step_name = $3 AND status = 'pending'`,
				now, jobID, nextStep,
			)
			if err == nil {
				_, _ = p.Exec(ctx,
					`UPDATE workflow_jobs SET current_step = $1, updated_at = $2 WHERE id = $3`,
					nextStep, now, jobID,
				)
				s.fireEvent(ctx, EventWorkflowProgress, map[string]interface{}{
					"job_id":   jobID.String(),
					"step":     nextStep,
					"progress": s.calcProgress(ctx, p, jobID),
				}, uuid.Nil)
			}
		}
	} else {
		_, _ = p.Exec(ctx,
			`UPDATE workflow_jobs SET status = 'completed', progress = 100, completed_at = $1, updated_at = $1
			 WHERE id = $2`,
			now, jobID,
		)
		s.addNotification(ctx, p, &jobID, nil, "job.completed", "Job Completed",
			"Workflow job completed successfully", "success", "")
		s.fireEvent(ctx, EventWorkflowCompleted, map[string]interface{}{
			"job_id": jobID.String(),
		}, uuid.Nil)
	}

	s.addLog(ctx, p, jobID, stepName, "info", fmt.Sprintf("step completed: %s", StepDisplayNames[WorkflowStep(stepName)]), nil, 0)
}

func (s *Service) onStepFailed(ctx context.Context, p database.Pool, jobID uuid.UUID, stepName string, errorMsg string) {
	now := time.Now()
	_, _ = p.Exec(ctx,
		`UPDATE workflow_jobs SET status = 'failed', error_message = $1, updated_at = $2 WHERE id = $3`,
		errorMsg, now, jobID,
	)

	s.addNotification(ctx, p, &jobID, nil, "job.failed", "Job Failed",
		fmt.Sprintf("Step %s failed: %s", stepName, errorMsg), "error", "")

	if stepName == string(StepQualityCheck) {
		s.fireEvent(ctx, EventWorkflowQualityFailed, map[string]interface{}{
			"job_id": jobID.String(),
			"step":   stepName,
			"error":  errorMsg,
		}, uuid.Nil)
	}
	if stepName == string(StepSEOEngine) {
		s.fireEvent(ctx, EventWorkflowSEOFailed, map[string]interface{}{
			"job_id": jobID.String(),
			"step":   stepName,
			"error":  errorMsg,
		}, uuid.Nil)
	}

	s.addLog(ctx, p, jobID, stepName, "error", fmt.Sprintf("step failed: %s - %s", stepName, errorMsg), nil, 0)
	s.fireEvent(ctx, EventWorkflowStepFailed, map[string]interface{}{
		"job_id": jobID.String(),
		"step":   stepName,
		"error":  errorMsg,
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

// --- Notifications ---

func (s *Service) ListNotifications(ctx context.Context, siteID uuid.UUID, notifType string, unreadOnly bool, limit, offset int) (*NotificationList, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	notifications, err := s.listNotifications(ctx, p, siteID, notifType, unreadOnly, limit, offset)
	if err != nil {
		return nil, err
	}

	unread, err := s.countUnreadNotifications(ctx, p, siteID)
	if err != nil {
		return nil, err
	}

	return &NotificationList{
		Notifications: notifications,
		Total:         int64(len(notifications)),
		Unread:        unread,
	}, nil
}

func (s *Service) MarkNotificationRead(ctx context.Context, siteID, notifID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}
	return s.markNotificationRead(ctx, p, siteID, notifID)
}

func (s *Service) MarkAllNotificationsRead(ctx context.Context, siteID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}
	return s.markAllNotificationsRead(ctx, p, siteID)
}

func (s *Service) addNotification(ctx context.Context, p database.Pool, jobID *uuid.UUID, queueID *uuid.UUID, notifType, title, message, severity, actionURL string) {
	n := &Notification{
		ID:               uuid.New(),
		SiteID:           uuid.Nil,
		WorkflowJobID:    jobID,
		QueueID:          queueID,
		NotificationType: notifType,
		Title:            title,
		Message:          message,
		Severity:         severity,
		ActionURL:        actionURL,
		CreatedAt:        time.Now(),
	}

	if jobID != nil {
		job, err := s.getJobByID(ctx, p, *jobID, *jobID)
		if err == nil {
			n.SiteID = job.SiteID
		}
	}

	if err := s.insertNotification(ctx, p, n); err != nil {
		s.log.Error("failed to add notification", "error", err)
	}
}

// --- Dashboard ---

func (s *Service) GetDashboard(ctx context.Context, siteID uuid.UUID) (*Dashboard, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	d, err := s.getDashboard(ctx, p, siteID)
	if err != nil {
		return nil, err
	}

	if d == nil {
		return s.refreshDashboard(ctx, p, siteID)
	}

	return d, nil
}

func (s *Service) RefreshDashboard(ctx context.Context, siteID uuid.UUID) (*Dashboard, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	return s.refreshDashboard(ctx, p, siteID)
}

func (s *Service) refreshDashboard(ctx context.Context, p database.Pool, siteID uuid.UUID) (*Dashboard, error) {
	now := time.Now()

	var totalJobs, runningJobs, completedJobs, failedJobs, pausedJobs int64
	_ = p.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*),0),
		        COALESCE(SUM(CASE WHEN status='running' THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN status='completed' THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN status='failed' THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN status='paused' THEN 1 ELSE 0 END),0)
		 FROM workflow_jobs WHERE site_id = $1`, siteID,
	).Scan(&totalJobs, &runningJobs, &completedJobs, &failedJobs, &pausedJobs)

	var queueSize int64
	_ = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM workflow_queue WHERE site_id = $1 AND status = 'pending'`, siteID,
	).Scan(&queueSize)

	var stalledQueue int64
	_ = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM workflow_queue WHERE site_id = $1 AND status = 'pending' AND retry_count >= max_retries`, siteID,
	).Scan(&stalledQueue)

	var pendingReview int64
	_ = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM workflow_jobs WHERE site_id = $1 AND current_step = 'quality_check' AND status = 'running'`, siteID,
	).Scan(&pendingReview)

	var scheduledPubs int64
	_ = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM workflow_queue WHERE site_id = $1 AND scheduled_for IS NOT NULL AND scheduled_for > $2`, siteID, now,
	).Scan(&scheduledPubs)

	var recentPubs int64
	_ = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM workflow_jobs WHERE site_id = $1 AND status = 'completed' AND completed_at > $2`, siteID, now.Add(-24*time.Hour),
	).Scan(&recentPubs)

	var avgExecMs float64
	_ = p.QueryRow(ctx,
		`SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000),0)
		 FROM workflow_jobs WHERE site_id = $1 AND status = 'completed' AND completed_at IS NOT NULL`, siteID,
	).Scan(&avgExecMs)

	var successRate, failureRate float64
	if totalJobs > 0 {
		successRate = float64(completedJobs) / float64(totalJobs) * 100
		failureRate = float64(failedJobs) / float64(totalJobs) * 100
	}

	d := &Dashboard{
		ID:                    uuid.New(),
		SiteID:                siteID,
		TotalJobs:             totalJobs,
		RunningJobs:           runningJobs,
		CompletedJobs:         completedJobs,
		FailedJobs:            failedJobs,
		PausedJobs:            pausedJobs,
		QueueSize:             queueSize,
		StalledQueue:          stalledQueue,
		PendingReview:         pendingReview,
		ScheduledPublications: scheduledPubs,
		RecentPublications:    recentPubs,
		AvgExecutionMs:        avgExecMs,
		SuccessRate:           successRate,
		FailureRate:           failureRate,
		ThroughputHourly:      0,
		WorkerUtilization:     0,
		Data:                  make(map[string]interface{}),
		SnapshotAt:            now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	_ = s.upsertDashboard(ctx, p, d)

	return d, nil
}

// --- Metrics ---

func (s *Service) GetMetrics(ctx context.Context, siteID uuid.UUID) (*WorkflowMetrics, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	return s.getMetrics(ctx, p, siteID)
}

// --- Stats ---

func (s *Service) GetStats(ctx context.Context, siteID uuid.UUID) (*WorkflowStats, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	return s.getStats(ctx, p, siteID)
}

// --- History ---

func (s *Service) ListHistory(ctx context.Context, siteID uuid.UUID, jobID *uuid.UUID, action string, limit, offset int) ([]HistoryEntry, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	return s.listHistory(ctx, p, siteID, jobID, action, limit, offset)
}

func (s *Service) addHistory(ctx context.Context, p database.Pool, siteID uuid.UUID, workflowJobID, queueID *uuid.UUID,
	action, entityType string, entityID *uuid.UUID, previousStatus, newStatus string,
	details map[string]interface{}, errorMessage string, userID *uuid.UUID, durationMs int64) {

	entry := &HistoryEntry{
		ID:             uuid.New(),
		SiteID:         siteID,
		WorkflowJobID:  workflowJobID,
		QueueID:        queueID,
		Action:         action,
		EntityType:     entityType,
		EntityID:       entityID,
		PreviousStatus: previousStatus,
		NewStatus:      newStatus,
		Details:        details,
		ErrorMessage:   errorMessage,
		UserID:         userID,
		DurationMs:     durationMs,
		CreatedAt:      time.Now(),
	}

	if err := s.insertHistory(ctx, p, entry); err != nil {
		s.log.Error("failed to add history entry", "error", err)
	}
}

// --- Automation ---

func (s *Service) ExecuteAction(ctx context.Context, siteID, userID uuid.UUID, action AutomationAction) (*WorkflowJob, error) {
	switch action.Action {
	case "generate_article":
		return s.CreateJob(ctx, siteID, userID, CreateJobRequest{
			Title:    coalesceStr(action.Title, "New Article"),
			Language: "pt",
		})

	case "generate_pt_en":
		return s.CreateJob(ctx, siteID, userID, CreateJobRequest{
			Title:      coalesceStr(action.Title, "New Article"),
			Language:   "pt",
			GeneratePT: true,
			GenerateEN: true,
		})

	case "publish_now":
		if action.JobID == "" {
			return nil, fmt.Errorf("job_id required for publish_now")
		}
		jobID, err := uuid.Parse(action.JobID)
		if err != nil {
			return nil, fmt.Errorf("invalid job id: %w", err)
		}
		job, err := s.GetJob(ctx, siteID, jobID)
		if err != nil {
			return nil, err
		}
		if job.Status != JobStatusCompleted {
			_, err = s.StartJob(ctx, siteID, jobID)
			if err != nil {
				return nil, err
			}
		}
		return s.GetJob(ctx, siteID, jobID)

	case "schedule":
		if action.JobID == "" {
			return nil, fmt.Errorf("job_id required for schedule")
		}
		return s.GetJob(ctx, siteID, uuid.MustParse(action.JobID))

	case "rebuild_seo":
		if action.JobID == "" {
			return nil, fmt.Errorf("job_id required for rebuild_seo")
		}
		return s.GetJob(ctx, siteID, uuid.MustParse(action.JobID))

	case "regenerate":
		if action.JobID == "" {
			return nil, fmt.Errorf("job_id required for regenerate")
		}
		jobID, err := uuid.Parse(action.JobID)
		if err != nil {
			return nil, fmt.Errorf("invalid job id: %w", err)
		}
		existing, err := s.GetJob(ctx, siteID, jobID)
		if err != nil {
			return nil, err
		}
		return s.CreateJob(ctx, siteID, userID, CreateJobRequest{
			Title:       existing.Title + " (Regenerated)",
			Language:    existing.Language,
			SourceJobID: &existing.ID,
		})

	case "duplicate":
		if action.JobID == "" {
			return nil, fmt.Errorf("job_id required for duplicate")
		}
		jobID, err := uuid.Parse(action.JobID)
		if err != nil {
			return nil, fmt.Errorf("invalid job id: %w", err)
		}
		existing, err := s.GetJob(ctx, siteID, jobID)
		if err != nil {
			return nil, err
		}
		return s.CreateJob(ctx, siteID, userID, CreateJobRequest{
			Title:       existing.Title + " (Copy)",
			Language:    existing.Language,
			SourceJobID: &existing.ID,
		})

	default:
		return nil, ErrInvalidAction
	}
}

// --- Helpers ---

func coalesceStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
