package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"nexora/internal/pkg/database"
)

func (s *Service) insertJob(ctx context.Context, p database.Pool, job *WorkflowJob) error {
	_, err := p.Exec(ctx,
		`INSERT INTO workflow_jobs (id, site_id, user_id, title, content_type, language,
		 target_language, status, priority, tone, audience, keywords, style_slug,
		 source_job_id, scheduled_for, max_retries, generate_pt, generate_en, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$20)`,
		job.ID, job.SiteID, job.UserID, job.Title, job.ContentType,
		job.Language, job.TargetLanguage, job.Status, job.Priority,
		job.Tone, job.Audience, job.Keywords, job.StyleSlug,
		job.SourceJobID, job.ScheduledFor, job.MaxRetries,
		job.GeneratePT, job.GenerateEN, job.CreatedBy, job.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert workflow job: %w", err)
	}
	return nil
}

func (s *Service) insertSteps(ctx context.Context, p database.Pool, jobID uuid.UUID, now time.Time) error {
	for _, step := range AllWorkflowSteps {
		stepID := uuid.New()
		deps := StepDependencies[step]
		depStrs := make([]string, len(deps))
		for i, d := range deps {
			depStrs[i] = string(d)
		}
		_, err := p.Exec(ctx,
			`INSERT INTO workflow_steps (id, workflow_job_id, step_name, display_name, status, depends_on, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,'pending',$5,$6,$6)`,
			stepID, jobID, string(step), StepDisplayNames[step], depStrs, now,
		)
		if err != nil {
			return fmt.Errorf("failed to create step %s: %w", step, err)
		}
	}
	return nil
}

func (s *Service) getJobByID(ctx context.Context, p database.Pool, siteID, jobID uuid.UUID) (*WorkflowJob, error) {
	var j WorkflowJob
	err := p.QueryRow(ctx,
		`SELECT id, site_id, user_id, title, COALESCE(content_type,'article'),
		        language, COALESCE(target_language,''), status, COALESCE(current_step,''),
		        COALESCE(progress,0), priority, COALESCE(word_count,0), COALESCE(tone,''),
		        COALESCE(audience,''), COALESCE(keywords,'{}'), COALESCE(style_slug,''),
		        source_job_id, scheduled_for, COALESCE(error_message,''), COALESCE(retry_count,0),
		        COALESCE(max_retries,3), generate_pt, generate_en, started_at, completed_at,
		        cancelled_at, created_by, created_at, updated_at
		 FROM workflow_jobs WHERE id = $1 AND site_id = $2`,
		jobID, siteID,
	).Scan(&j.ID, &j.SiteID, &j.UserID, &j.Title, &j.ContentType,
		&j.Language, &j.TargetLanguage, &j.Status, &j.CurrentStep,
		&j.Progress, &j.Priority, &j.WordCount, &j.Tone, &j.Audience, &j.Keywords,
		&j.StyleSlug, &j.SourceJobID, &j.ScheduledFor, &j.ErrorMessage, &j.RetryCount,
		&j.MaxRetries, &j.GeneratePT, &j.GenerateEN, &j.StartedAt, &j.CompletedAt,
		&j.CancelledAt, &j.CreatedBy, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get workflow job: %w", err)
	}
	return &j, nil
}

func (s *Service) listJobs(ctx context.Context, p database.Pool, siteID uuid.UUID, status, language, step string, limit, offset int) ([]WorkflowJob, error) {
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
		`SELECT id, site_id, user_id, title, COALESCE(content_type,'article'),
		        language, COALESCE(target_language,''), status, COALESCE(current_step,''),
		        COALESCE(progress,0), priority, COALESCE(word_count,0), COALESCE(tone,''),
		        COALESCE(audience,''), COALESCE(keywords,'{}'), COALESCE(style_slug,''),
		        source_job_id, scheduled_for, COALESCE(error_message,''), COALESCE(retry_count,0),
		        COALESCE(max_retries,3), generate_pt, generate_en, started_at, completed_at,
		        cancelled_at, created_by, created_at, updated_at
		 FROM workflow_jobs WHERE %s ORDER BY priority ASC, created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow jobs: %w", err)
	}
	defer rows.Close()

	var jobs []WorkflowJob
	for rows.Next() {
		var j WorkflowJob
		if err := rows.Scan(&j.ID, &j.SiteID, &j.UserID, &j.Title, &j.ContentType,
			&j.Language, &j.TargetLanguage, &j.Status, &j.CurrentStep,
			&j.Progress, &j.Priority, &j.WordCount, &j.Tone, &j.Audience, &j.Keywords,
			&j.StyleSlug, &j.SourceJobID, &j.ScheduledFor, &j.ErrorMessage, &j.RetryCount,
			&j.MaxRetries, &j.GeneratePT, &j.GenerateEN, &j.StartedAt, &j.CompletedAt,
			&j.CancelledAt, &j.CreatedBy, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan workflow job: %w", err)
		}
		jobs = append(jobs, j)
	}
	if jobs == nil {
		jobs = []WorkflowJob{}
	}
	return jobs, nil
}

func (s *Service) updateJobFields(ctx context.Context, p database.Pool, jobID, siteID uuid.UUID, setClauses []string, args []interface{}) error {
	argIdx := len(args) + 1
	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE workflow_jobs SET %s WHERE id = $%d AND site_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, jobID, siteID)
	_, err := p.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update workflow job: %w", err)
	}
	return nil
}

func (s *Service) deleteJobByID(ctx context.Context, p database.Pool, siteID, jobID uuid.UUID) error {
	tag, err := p.Exec(ctx,
		`DELETE FROM workflow_jobs WHERE id = $1 AND site_id = $2`,
		jobID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete workflow job: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotFound
	}
	return nil
}

func (s *Service) listSteps(ctx context.Context, p database.Pool, jobID uuid.UUID) ([]Step, error) {
	rows, err := p.Query(ctx,
		`SELECT id, workflow_job_id, step_name, COALESCE(display_name,''), status,
		        COALESCE(progress,0), COALESCE(depends_on,'{}'), COALESCE(retry_count,0),
		        COALESCE(max_retries,3), started_at, completed_at, COALESCE(duration_ms,0),
		        COALESCE(error_message,''), COALESCE(metadata::text,'{}'), created_at, updated_at
		 FROM workflow_steps WHERE workflow_job_id = $1 ORDER BY created_at ASC`,
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
		if err := rows.Scan(&s.ID, &s.WorkflowJobID, &s.StepName, &s.DisplayName, &s.Status,
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
		`SELECT id, workflow_job_id, step_name, COALESCE(display_name,''), status,
		        COALESCE(progress,0), COALESCE(depends_on,'{}'), COALESCE(retry_count,0),
		        COALESCE(max_retries,3), started_at, completed_at, COALESCE(duration_ms,0),
		        COALESCE(error_message,''), COALESCE(metadata::text,'{}'), created_at, updated_at
		 FROM workflow_steps WHERE workflow_job_id = $1 AND step_name = $2`,
		jobID, stepName,
	).Scan(&step.ID, &step.WorkflowJobID, &step.StepName, &step.DisplayName, &step.Status,
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

func (s *Service) updateStep(ctx context.Context, p database.Pool, jobID uuid.UUID, stepName string, status StepStatus, progress float64, metadata map[string]interface{}, errorMsg string, durationMs int64) (*Step, error) {
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
		`UPDATE workflow_steps SET %s WHERE workflow_job_id = $%d AND step_name = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, jobID, stepName)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update step: %w", err)
	}

	return s.getStepByName(ctx, p, jobID, stepName)
}

func (s *Service) insertQueueItem(ctx context.Context, p database.Pool, item *QueueItem) error {
	_, err := p.Exec(ctx,
		`INSERT INTO workflow_queue (id, site_id, workflow_job_id, title, content, excerpt,
		 language, status, priority, scheduled_for, max_retries, meta_title, meta_description,
		 slug, featured_image_url, tags, categories, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$18)`,
		item.ID, item.SiteID, item.WorkflowJobID, item.Title, item.Content, item.Excerpt,
		item.Language, item.Status, item.Priority, item.ScheduledFor, item.MaxRetries,
		item.MetaTitle, item.MetaDescription, item.Slug, item.FeaturedImageURL,
		item.Tags, item.Categories, item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert queue item: %w", err)
	}
	return nil
}

func (s *Service) listQueueItems(ctx context.Context, p database.Pool, siteID uuid.UUID, status string, limit, offset int) ([]QueueItem, error) {
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
		`SELECT id, site_id, workflow_job_id, title, COALESCE(content,''), COALESCE(excerpt,''),
		        language, status, priority, scheduled_for, is_paused, COALESCE(retry_count,0),
		        COALESCE(max_retries,3), COALESCE(meta_title,''), COALESCE(meta_description,''),
		        COALESCE(slug,''), COALESCE(featured_image_url,''), COALESCE(tags,'{}'),
		        COALESCE(categories,'{}'), published_at, published_by, COALESCE(error_message,''),
		        created_at, updated_at
		 FROM workflow_queue WHERE %s ORDER BY priority ASC, created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list queue: %w", err)
	}
	defer rows.Close()

	var items []QueueItem
	for rows.Next() {
		var item QueueItem
		if err := rows.Scan(&item.ID, &item.SiteID, &item.WorkflowJobID, &item.Title,
			&item.Content, &item.Excerpt, &item.Language, &item.Status, &item.Priority,
			&item.ScheduledFor, &item.IsPaused, &item.RetryCount, &item.MaxRetries,
			&item.MetaTitle, &item.MetaDescription, &item.Slug, &item.FeaturedImageURL,
			&item.Tags, &item.Categories, &item.PublishedAt, &item.PublishedBy,
			&item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan queue item: %w", err)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []QueueItem{}
	}
	return items, nil
}

func (s *Service) getQueueItemByID(ctx context.Context, p database.Pool, siteID, itemID uuid.UUID) (*QueueItem, error) {
	var item QueueItem
	err := p.QueryRow(ctx,
		`SELECT id, site_id, workflow_job_id, title, COALESCE(content,''), COALESCE(excerpt,''),
		        language, status, priority, scheduled_for, is_paused, COALESCE(retry_count,0),
		        COALESCE(max_retries,3), COALESCE(meta_title,''), COALESCE(meta_description,''),
		        COALESCE(slug,''), COALESCE(featured_image_url,''), COALESCE(tags,'{}'),
		        COALESCE(categories,'{}'), published_at, published_by, COALESCE(error_message,''),
		        created_at, updated_at
		 FROM workflow_queue WHERE id = $1 AND site_id = $2`,
		itemID, siteID,
	).Scan(&item.ID, &item.SiteID, &item.WorkflowJobID, &item.Title,
		&item.Content, &item.Excerpt, &item.Language, &item.Status, &item.Priority,
		&item.ScheduledFor, &item.IsPaused, &item.RetryCount, &item.MaxRetries,
		&item.MetaTitle, &item.MetaDescription, &item.Slug, &item.FeaturedImageURL,
		&item.Tags, &item.Categories, &item.PublishedAt, &item.PublishedBy,
		&item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrQueueItemNotFound
		}
		return nil, fmt.Errorf("failed to get queue item: %w", err)
	}
	return &item, nil
}

func (s *Service) updateQueueItemFields(ctx context.Context, p database.Pool, itemID, siteID uuid.UUID, setClauses []string, args []interface{}) error {
	argIdx := len(args) + 1
	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE workflow_queue SET %s WHERE id = $%d AND site_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, itemID, siteID)
	_, err := p.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update queue item: %w", err)
	}
	return nil
}

func (s *Service) insertHistory(ctx context.Context, p database.Pool, entry *HistoryEntry) error {
	detailsJSON, _ := json.Marshal(entry.Details)
	_, err := p.Exec(ctx,
		`INSERT INTO workflow_history (id, site_id, workflow_job_id, queue_id, action,
		 entity_type, entity_id, previous_status, new_status, details, error_message,
		 user_id, duration_ms, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,$11,$12,$13,$14)`,
		entry.ID, entry.SiteID, entry.WorkflowJobID, entry.QueueID,
		entry.Action, entry.EntityType, entry.EntityID, entry.PreviousStatus,
		entry.NewStatus, string(detailsJSON), entry.ErrorMessage,
		entry.UserID, entry.DurationMs, entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert history entry: %w", err)
	}
	return nil
}

func (s *Service) listHistory(ctx context.Context, p database.Pool, siteID uuid.UUID, jobID *uuid.UUID, action string, limit, offset int) ([]HistoryEntry, error) {
	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2

	if jobID != nil {
		where = append(where, fmt.Sprintf("workflow_job_id = $%d", argIdx))
		args = append(args, *jobID)
		argIdx++
	}
	if action != "" {
		where = append(where, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, action)
		argIdx++
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, workflow_job_id, queue_id, action, entity_type, entity_id,
		        COALESCE(previous_status,''), COALESCE(new_status,''), COALESCE(details::text,'{}'),
		        COALESCE(error_message,''), user_id, COALESCE(duration_ms,0), created_at
		 FROM workflow_history WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list history: %w", err)
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var detailsStr string
		if err := rows.Scan(&e.ID, &e.SiteID, &e.WorkflowJobID, &e.QueueID,
			&e.Action, &e.EntityType, &e.EntityID, &e.PreviousStatus, &e.NewStatus,
			&detailsStr, &e.ErrorMessage, &e.UserID, &e.DurationMs, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}
		if len(detailsStr) > 0 {
			_ = json.Unmarshal([]byte(detailsStr), &e.Details)
		}
		if e.Details == nil {
			e.Details = make(map[string]interface{})
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []HistoryEntry{}
	}
	return entries, nil
}

func (s *Service) insertNotification(ctx context.Context, p database.Pool, n *Notification) error {
	_, err := p.Exec(ctx,
		`INSERT INTO workflow_notifications (id, site_id, workflow_job_id, queue_id,
		 notification_type, title, message, severity, action_url, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		n.ID, n.SiteID, n.WorkflowJobID, n.QueueID,
		n.NotificationType, n.Title, n.Message, n.Severity, n.ActionURL, n.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert notification: %w", err)
	}
	return nil
}

func (s *Service) listNotifications(ctx context.Context, p database.Pool, siteID uuid.UUID, notifType string, unreadOnly bool, limit, offset int) ([]Notification, error) {
	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2

	if notifType != "" {
		where = append(where, fmt.Sprintf("notification_type = $%d", argIdx))
		args = append(args, notifType)
		argIdx++
	}
	if unreadOnly {
		where = append(where, "read = false")
	}

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, workflow_job_id, queue_id, notification_type, title,
		        COALESCE(message,''), severity, read, COALESCE(action_url,''), created_at
		 FROM workflow_notifications WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.SiteID, &n.WorkflowJobID, &n.QueueID,
			&n.NotificationType, &n.Title, &n.Message, &n.Severity, &n.Read,
			&n.ActionURL, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}
	if notifications == nil {
		notifications = []Notification{}
	}
	return notifications, nil
}

func (s *Service) countUnreadNotifications(ctx context.Context, p database.Pool, siteID uuid.UUID) (int64, error) {
	var count int64
	err := p.QueryRow(ctx,
		`SELECT COUNT(*) FROM workflow_notifications WHERE site_id = $1 AND read = false`,
		siteID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread: %w", err)
	}
	return count, nil
}

func (s *Service) markNotificationRead(ctx context.Context, p database.Pool, siteID, notifID uuid.UUID) error {
	tag, err := p.Exec(ctx,
		`UPDATE workflow_notifications SET read = true WHERE id = $1 AND site_id = $2`,
		notifID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark notification read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (s *Service) markAllNotificationsRead(ctx context.Context, p database.Pool, siteID uuid.UUID) error {
	_, err := p.Exec(ctx,
		`UPDATE workflow_notifications SET read = true WHERE site_id = $1 AND read = false`,
		siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications read: %w", err)
	}
	return nil
}

func (s *Service) getDashboard(ctx context.Context, p database.Pool, siteID uuid.UUID) (*Dashboard, error) {
	var d Dashboard
	err := p.QueryRow(ctx,
		`SELECT id, site_id,
		        COALESCE(total_jobs,0), COALESCE(running_jobs,0), COALESCE(completed_jobs,0),
		        COALESCE(failed_jobs,0), COALESCE(paused_jobs,0), COALESCE(queue_size,0),
		        COALESCE(stalled_queue,0), COALESCE(pending_review,0),
		        COALESCE(scheduled_publications,0), COALESCE(recent_publications,0),
		        COALESCE(avg_execution_ms,0), COALESCE(success_rate,0), COALESCE(failure_rate,0),
		        COALESCE(throughput_hourly,0), COALESCE(worker_utilization,0),
		        COALESCE(data::text,'{}'), snapshot_at, created_at, updated_at
		 FROM workflow_dashboard WHERE site_id = $1 ORDER BY snapshot_at DESC LIMIT 1`,
		siteID,
	).Scan(&d.ID, &d.SiteID,
		&d.TotalJobs, &d.RunningJobs, &d.CompletedJobs,
		&d.FailedJobs, &d.PausedJobs, &d.QueueSize,
		&d.StalledQueue, &d.PendingReview,
		&d.ScheduledPublications, &d.RecentPublications,
		&d.AvgExecutionMs, &d.SuccessRate, &d.FailureRate,
		&d.ThroughputHourly, &d.WorkerUtilization,
		&d.Data, &d.SnapshotAt, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}
	return &d, nil
}

func (s *Service) upsertDashboard(ctx context.Context, p database.Pool, d *Dashboard) error {
	dataJSON, _ := json.Marshal(d.Data)
	_, err := p.Exec(ctx,
		`INSERT INTO workflow_dashboard (id, site_id, total_jobs, running_jobs, completed_jobs,
		 failed_jobs, paused_jobs, queue_size, stalled_queue, pending_review,
		 scheduled_publications, recent_publications, avg_execution_ms, success_rate,
		 failure_rate, throughput_hourly, worker_utilization, data, snapshot_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18::jsonb,$19,$20,$20)`,
		d.ID, d.SiteID, d.TotalJobs, d.RunningJobs, d.CompletedJobs,
		d.FailedJobs, d.PausedJobs, d.QueueSize, d.StalledQueue, d.PendingReview,
		d.ScheduledPublications, d.RecentPublications, d.AvgExecutionMs, d.SuccessRate,
		d.FailureRate, d.ThroughputHourly, d.WorkerUtilization, string(dataJSON),
		d.SnapshotAt, d.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert dashboard: %w", err)
	}
	return nil
}

func (s *Service) getMetrics(ctx context.Context, p database.Pool, siteID uuid.UUID) (*WorkflowMetrics, error) {
	var m WorkflowMetrics

	err := p.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*),0),
		        COALESCE(SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN status = 'paused' THEN 1 ELSE 0 END),0),
		        COALESCE(AVG(CASE WHEN status = 'completed' THEN EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000 ELSE NULL END),0),
		        COALESCE((SELECT COUNT(*) FROM workflow_queue WHERE site_id = $1 AND status = 'pending'),0),
		        COALESCE((SELECT COUNT(*) FROM workflow_queue WHERE site_id = $1 AND status = 'pending' AND retry_count >= max_retries),0),
		        COALESCE((SELECT COUNT(*) FROM workflow_jobs WHERE site_id = $1 AND status = 'running'),0),
		        COALESCE((SELECT COUNT(*) FROM workflow_queue WHERE site_id = $1 AND scheduled_for IS NOT NULL AND scheduled_for > NOW()),0)
		 FROM workflow_jobs WHERE site_id = $1`,
		siteID,
	).Scan(&m.TotalJobs, &m.RunningJobs, &m.CompletedJobs, &m.FailedJobs, &m.PausedJobs,
		&m.AvgDuration, &m.QueueSize, &m.StalledCount, &m.PendingReview, &m.ScheduledCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	if m.TotalJobs > 0 {
		m.AvgSuccessRate = float64(m.CompletedJobs) / float64(m.TotalJobs) * 100
		m.AvgFailureRate = float64(m.FailedJobs) / float64(m.TotalJobs) * 100
	}

	notifCount, _ := s.countUnreadNotifications(ctx, p, siteID)
	m.NotificationCnt = notifCount

	return &m, nil
}

func (s *Service) getStageDurations(ctx context.Context, p database.Pool, siteID uuid.UUID) ([]StageDuration, error) {
	rows, err := p.Query(ctx,
		`SELECT ws.step_name, COALESCE(ws.display_name,''), AVG(ws.duration_ms), COUNT(*)
		 FROM workflow_steps ws
		 JOIN workflow_jobs wj ON ws.workflow_job_id = wj.id
		 WHERE wj.site_id = $1 AND ws.status = 'completed' AND ws.duration_ms > 0
		 GROUP BY ws.step_name, ws.display_name
		 ORDER BY ws.created_at ASC`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stage durations: %w", err)
	}
	defer rows.Close()

	var stages []StageDuration
	for rows.Next() {
		var s StageDuration
		if err := rows.Scan(&s.StepName, &s.DisplayName, &s.AvgDuration, &s.Count); err != nil {
			return nil, fmt.Errorf("failed to scan stage: %w", err)
		}
		stages = append(stages, s)
	}
	if stages == nil {
		stages = []StageDuration{}
	}
	return stages, nil
}

func (s *Service) getStats(ctx context.Context, p database.Pool, siteID uuid.UUID) (*WorkflowStats, error) {
	stats := &WorkflowStats{
		ByStatus:   make(map[string]int64),
		ByLanguage: make(map[string]int64),
		ByStep:     make(map[string]int64),
	}

	rows, err := p.Query(ctx,
		`SELECT status, COUNT(*) FROM workflow_jobs WHERE site_id = $1 GROUP BY status`,
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
			stats.ByStatus[status] = count
		}
	}

	rows2, err := p.Query(ctx,
		`SELECT language, COUNT(*) FROM workflow_jobs WHERE site_id = $1 GROUP BY language`,
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
		`SELECT current_step, COUNT(*) FROM workflow_jobs WHERE site_id = $1 AND current_step != '' GROUP BY current_step`,
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

	stages, err := s.getStageDurations(ctx, p, siteID)
	if err == nil {
		stats.Stages = stages
	}

	return stats, nil
}

func (s *Service) getNextQueueItem(ctx context.Context, p database.Pool, siteID uuid.UUID) (*QueueItem, error) {
	var item QueueItem
	err := p.QueryRow(ctx,
		`SELECT id, site_id, workflow_job_id, title, COALESCE(content,''), COALESCE(excerpt,''),
		        language, status, priority, scheduled_for, is_paused, COALESCE(retry_count,0),
		        COALESCE(max_retries,3), COALESCE(meta_title,''), COALESCE(meta_description,''),
		        COALESCE(slug,''), COALESCE(featured_image_url,''), COALESCE(tags,'{}'),
		        COALESCE(categories,'{}'), published_at, published_by, COALESCE(error_message,''),
		        created_at, updated_at
		 FROM workflow_queue
		 WHERE site_id = $1 AND status = 'pending' AND is_paused = false
		   AND (scheduled_for IS NULL OR scheduled_for <= NOW())
		 ORDER BY priority ASC, created_at ASC
		 LIMIT 1`,
		siteID,
	).Scan(&item.ID, &item.SiteID, &item.WorkflowJobID, &item.Title,
		&item.Content, &item.Excerpt, &item.Language, &item.Status, &item.Priority,
		&item.ScheduledFor, &item.IsPaused, &item.RetryCount, &item.MaxRetries,
		&item.MetaTitle, &item.MetaDescription, &item.Slug, &item.FeaturedImageURL,
		&item.Tags, &item.Categories, &item.PublishedAt, &item.PublishedBy,
		&item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get next queue item: %w", err)
	}
	return &item, nil
}

func (s *Service) addLog(ctx context.Context, p database.Pool, jobID uuid.UUID, step, level, message string, details map[string]interface{}, durationMs int64) {
	detailsJSON, _ := json.Marshal(details)
	_, err := p.Exec(ctx,
		`INSERT INTO generation_pipeline_logs (id, generation_job_id, stage, level, message, details, duration_ms, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8)`,
		uuid.New(), jobID, step, level, message, string(detailsJSON), durationMs, time.Now(),
	)
	if err != nil {
		s.log.Error("failed to add workflow log", "error", err)
	}
}
