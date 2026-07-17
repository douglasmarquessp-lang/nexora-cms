package editorial

import (
	"context"
	"encoding/json"
	"fmt"
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

func (s *Service) GetDashboard(ctx context.Context, siteID uuid.UUID) (*DashboardStats, error) {
	stats := &DashboardStats{}

	if err := s.loadStats(ctx, siteID, stats); err != nil {
		return nil, err
	}

	var err error
	stats.RecentPosts, err = s.listPostsByStatus(ctx, siteID, "", 5)
	if err != nil {
		return nil, err
	}

	stats.DraftPostsList, err = s.listPostsByStatus(ctx, siteID, "draft", 5)
	if err != nil {
		return nil, err
	}

	stats.ScheduledPostsList, err = s.listPostsByStatus(ctx, siteID, "scheduled", 5)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *Service) GetStats(ctx context.Context, siteID uuid.UUID) (*DashboardStats, error) {
	stats := &DashboardStats{}
	if err := s.loadStats(ctx, siteID, stats); err != nil {
		return nil, err
	}
	return stats, nil
}

func (s *Service) loadStats(ctx context.Context, siteID uuid.UUID, stats *DashboardStats) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	err = p.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'published' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'scheduled' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'archived' THEN 1 ELSE 0 END), 0),
			COUNT(*)
		FROM posts WHERE site_id = $1 AND deleted_at IS NULL`, siteID,
	).Scan(&stats.PublishedPosts, &stats.DraftPosts, &stats.ScheduledPosts, &stats.ArchivedPosts, &stats.TotalPosts)
	if err != nil {
		return fmt.Errorf("failed to load post stats: %w", err)
	}

	err = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM media WHERE site_id = $1 AND deleted_at IS NULL`, siteID,
	).Scan(&stats.TotalMedia)
	if err != nil {
		return fmt.Errorf("failed to load media stats: %w", err)
	}

	err = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM categories WHERE site_id = $1 AND deleted_at IS NULL`, siteID,
	).Scan(&stats.TotalCategories)
	if err != nil {
		return fmt.Errorf("failed to load category stats: %w", err)
	}

	err = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM tags WHERE site_id = $1 AND deleted_at IS NULL`, siteID,
	).Scan(&stats.TotalTags)
	if err != nil {
		return fmt.Errorf("failed to load tag stats: %w", err)
	}

	err = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM editorial_tasks WHERE site_id = $1 AND deleted_at IS NULL`, siteID,
	).Scan(&stats.TotalTasks)
	if err != nil {
		return fmt.Errorf("failed to load task stats: %w", err)
	}

	err = p.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*), 0) FROM editorial_tasks WHERE site_id = $1 AND status = 'pending' AND deleted_at IS NULL`, siteID,
	).Scan(&stats.PendingTasks)
	if err != nil {
		return fmt.Errorf("failed to load pending tasks: %w", err)
	}

	err = p.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*), 0) FROM approval_requests WHERE site_id = $1 AND status = 'pending'`, siteID,
	).Scan(&stats.PendingApprovals)
	if err != nil {
		return fmt.Errorf("failed to load pending approvals: %w", err)
	}

	return nil
}

func (s *Service) listPostsByStatus(ctx context.Context, siteID uuid.UUID, status string, limit int) ([]PostSummary, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var query string
	var args []interface{}

	if status == "" {
		query = `
			SELECT id, title, slug, status, COALESCE(excerpt, ''), published_at, created_at, updated_at
			FROM posts
			WHERE site_id = $1 AND deleted_at IS NULL
			ORDER BY updated_at DESC LIMIT $2`
		args = []interface{}{siteID, limit}
	} else {
		query = `
			SELECT id, title, slug, status, COALESCE(excerpt, ''), published_at, created_at, updated_at
			FROM posts
			WHERE site_id = $1 AND status = $2 AND deleted_at IS NULL
			ORDER BY updated_at DESC LIMIT $3`
		args = []interface{}{siteID, status, limit}
	}

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}
	defer rows.Close()

	var summaries []PostSummary
	for rows.Next() {
		var s PostSummary
		var publishedAt *time.Time
		if err := rows.Scan(&s.ID, &s.Title, &s.Slug, &s.Status, &s.Excerpt, &publishedAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		s.PublishedAt = publishedAt
		summaries = append(summaries, s)
	}
	if summaries == nil {
		summaries = []PostSummary{}
	}
	return summaries, nil
}

func (s *Service) CreateTask(ctx context.Context, siteID, userID uuid.UUID, req CreateTaskRequest) (*Task, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	status := req.Status
	if status == "" {
		status = TaskStatusPending
	}

	priority := req.Priority
	if priority == "" {
		priority = TaskPriorityMedium
	}

	now := time.Now()
	taskID := uuid.New()

	_, err = p.Exec(ctx,
		`INSERT INTO editorial_tasks (id, site_id, title, description, status, priority, assignee_id, due_date, post_id, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		taskID, siteID, req.Title, req.Description, string(status), string(priority),
		req.AssigneeID, req.DueDate, req.PostID, userID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	task := &Task{
		ID:          taskID,
		SiteID:      siteID,
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		Priority:    priority,
		AssigneeID:  req.AssigneeID,
		DueDate:     req.DueDate,
		PostID:      req.PostID,
		CreatedBy:   &userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("editorial.task.created"),
		EntityType: "editorial_task",
		EntityID:   &taskID,
		Payload:    map[string]interface{}{"title": req.Title, "priority": string(priority)},
	})

	s.fireEvent(ctx, EventTaskCreated, map[string]interface{}{
		"task_id":  taskID.String(),
		"site_id":  siteID.String(),
		"title":    req.Title,
		"priority": string(priority),
	}, siteID)

	return task, nil
}

func (s *Service) GetTask(ctx context.Context, siteID, taskID uuid.UUID) (*Task, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var t Task
	var deletedAt *time.Time
	err = p.QueryRow(ctx,
		`SELECT id, site_id, title, COALESCE(description, ''), status, priority, assignee_id, due_date, post_id, created_by, completed_at, created_at, updated_at
		 FROM editorial_tasks WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		taskID, siteID,
	).Scan(&t.ID, &t.SiteID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&t.AssigneeID, &t.DueDate, &t.PostID, &t.CreatedBy, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	t.DeletedAt = deletedAt
	return &t, nil
}

func (s *Service) ListTasks(ctx context.Context, siteID uuid.UUID, status TaskStatus) ([]Task, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var rows pgx.Rows
	if status == "" {
		rows, err = p.Query(ctx,
			`SELECT id, site_id, title, COALESCE(description, ''), status, priority, assignee_id, due_date, post_id, created_by, completed_at, created_at, updated_at
			 FROM editorial_tasks WHERE site_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`,
			siteID,
		)
	} else {
		rows, err = p.Query(ctx,
			`SELECT id, site_id, title, COALESCE(description, ''), status, priority, assignee_id, due_date, post_id, created_by, completed_at, created_at, updated_at
			 FROM editorial_tasks WHERE site_id = $1 AND status = $2 AND deleted_at IS NULL ORDER BY created_at DESC`,
			siteID, string(status),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var deletedAt *time.Time
		if err := rows.Scan(&t.ID, &t.SiteID, &t.Title, &t.Description, &t.Status, &t.Priority,
			&t.AssigneeID, &t.DueDate, &t.PostID, &t.CreatedBy, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		t.DeletedAt = deletedAt
		tasks = append(tasks, t)
	}
	if tasks == nil {
		tasks = []Task{}
	}
	return tasks, nil
}

func (s *Service) UpdateTask(ctx context.Context, siteID, taskID uuid.UUID, req UpdateTaskRequest) (*Task, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, err := s.GetTask(ctx, siteID, taskID)
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
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*req.Status))
		argIdx++

		if *req.Status == TaskStatusCompleted {
			now := time.Now()
			setClauses = append(setClauses, fmt.Sprintf("completed_at = $%d", argIdx))
			args = append(args, now)
			argIdx++
		}
	}
	if req.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, string(*req.Priority))
		argIdx++
	}
	if req.AssigneeID != nil {
		setClauses = append(setClauses, fmt.Sprintf("assignee_id = $%d", argIdx))
		args = append(args, *req.AssigneeID)
		argIdx++
	}
	if req.DueDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("due_date = $%d", argIdx))
		args = append(args, *req.DueDate)
		argIdx++
	}
	if req.PostID != nil {
		setClauses = append(setClauses, fmt.Sprintf("post_id = $%d", argIdx))
		args = append(args, *req.PostID)
		argIdx++
	}

	if len(setClauses) == 0 {
		return existing, nil
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = NOW()"))
	query := fmt.Sprintf(
		`UPDATE editorial_tasks SET %s WHERE id = $%d AND site_id = $%d AND deleted_at IS NULL`,
		stringsJoin(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, taskID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		EntityType: "editorial_task",
		EntityID:   &taskID,
		Action:     audit.Action("editorial.task.updated"),
		Payload:    map[string]interface{}{"title": existing.Title},
	})

	s.fireEvent(ctx, EventTaskUpdated, map[string]interface{}{
		"task_id": taskID.String(),
		"site_id": siteID.String(),
	}, siteID)

	return s.GetTask(ctx, siteID, taskID)
}

func (s *Service) DeleteTask(ctx context.Context, siteID, taskID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	task, err := s.GetTask(ctx, siteID, taskID)
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx,
		`UPDATE editorial_tasks SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		taskID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		EntityType: "editorial_task",
		EntityID:   &taskID,
		Action:     audit.Action("editorial.task.deleted"),
		Payload:    map[string]interface{}{"title": task.Title},
	})

	s.fireEvent(ctx, EventTaskDeleted, map[string]interface{}{
		"task_id": taskID.String(),
		"site_id": siteID.String(),
	}, siteID)

	return nil
}

func (s *Service) SaveRevision(ctx context.Context, siteID, postID, authorID uuid.UUID, req CreateRevisionRequest) (*Revision, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var currentVersion int
	err = p.QueryRow(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM post_revisions WHERE post_id = $1`, postID,
	).Scan(&currentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get current revision version: %w", err)
	}

	var title, excerpt, slug string
	var contentJSON, postMetaJSON, metadataJSON []byte
	err = p.QueryRow(ctx,
		`SELECT title, COALESCE(content::text, '[]'), COALESCE(excerpt, ''), slug,
		        COALESCE(post_meta::text, '{}'), COALESCE(metadata::text, '{}')
		 FROM posts WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		postID, siteID,
	).Scan(&title, &contentJSON, &excerpt, &slug, &postMetaJSON, &metadataJSON)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("post not found")
		}
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	revID := uuid.New()
	newVersion := currentVersion + 1

	_, err = p.Exec(ctx,
		`INSERT INTO post_revisions (id, post_id, site_id, author_id, version, title, content, excerpt, slug, post_meta, metadata, summary, change_log, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10::jsonb, $11::jsonb, $12, $13, NOW())`,
		revID, postID, siteID, authorID, newVersion, title, string(contentJSON), excerpt, slug,
		string(postMetaJSON), string(metadataJSON), req.Summary, req.ChangeLog,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save revision: %w", err)
	}

	var content []interface{}
	if len(contentJSON) > 0 {
		_ = json.Unmarshal(contentJSON, &content)
	}
	var postMeta map[string]interface{}
	if len(postMetaJSON) > 0 {
		_ = json.Unmarshal(postMetaJSON, &postMeta)
	}
	var metadata map[string]interface{}
	if len(metadataJSON) > 0 {
		_ = json.Unmarshal(metadataJSON, &metadata)
	}

	revision := &Revision{
		ID:        revID,
		PostID:    postID,
		SiteID:    siteID,
		AuthorID:  authorID,
		Version:   newVersion,
		Title:     title,
		Content:   content,
		Excerpt:   excerpt,
		Slug:      slug,
		PostMeta:  postMeta,
		Metadata:  metadata,
		Summary:   req.Summary,
		ChangeLog: req.ChangeLog,
		CreatedAt: time.Now(),
	}

	s.fireEvent(ctx, EventRevisionSaved, map[string]interface{}{
		"revision_id": revID.String(),
		"post_id":     postID.String(),
		"site_id":     siteID.String(),
		"version":     newVersion,
	}, siteID)

	return revision, nil
}

func (s *Service) ListRevisions(ctx context.Context, siteID, postID uuid.UUID) ([]Revision, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, post_id, site_id, author_id, version, title, COALESCE(content::text, '[]'),
		        COALESCE(excerpt, ''), slug, COALESCE(post_meta::text, '{}'), COALESCE(metadata::text, '{}'),
		        COALESCE(summary, ''), COALESCE(change_log, ''), created_at
		 FROM post_revisions
		 WHERE post_id = $1 AND site_id = $2
		 ORDER BY version DESC`,
		postID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list revisions: %w", err)
	}
	defer rows.Close()

	var revisions []Revision
	for rows.Next() {
		var r Revision
		var contentJSON, postMetaJSON, metadataJSON string
		if err := rows.Scan(&r.ID, &r.PostID, &r.SiteID, &r.AuthorID, &r.Version, &r.Title,
			&contentJSON, &r.Excerpt, &r.Slug, &postMetaJSON, &metadataJSON,
			&r.Summary, &r.ChangeLog, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan revision: %w", err)
		}
		if len(contentJSON) > 0 {
			_ = json.Unmarshal([]byte(contentJSON), &r.Content)
		}
		if len(postMetaJSON) > 0 {
			_ = json.Unmarshal([]byte(postMetaJSON), &r.PostMeta)
		}
		if len(metadataJSON) > 0 {
			_ = json.Unmarshal([]byte(metadataJSON), &r.Metadata)
		}
		if r.Content == nil {
			r.Content = []interface{}{}
		}
		if r.PostMeta == nil {
			r.PostMeta = make(map[string]interface{})
		}
		if r.Metadata == nil {
			r.Metadata = make(map[string]interface{})
		}
		revisions = append(revisions, r)
	}
	if revisions == nil {
		revisions = []Revision{}
	}
	return revisions, nil
}

func (s *Service) GetRevision(ctx context.Context, siteID, postID, revisionID uuid.UUID) (*Revision, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var r Revision
	var contentJSON, postMetaJSON, metadataJSON string
	err = p.QueryRow(ctx,
		`SELECT id, post_id, site_id, author_id, version, title, COALESCE(content::text, '[]'),
		        COALESCE(excerpt, ''), slug, COALESCE(post_meta::text, '{}'), COALESCE(metadata::text, '{}'),
		        COALESCE(summary, ''), COALESCE(change_log, ''), created_at
		 FROM post_revisions
		 WHERE id = $1 AND post_id = $2 AND site_id = $3`,
		revisionID, postID, siteID,
	).Scan(&r.ID, &r.PostID, &r.SiteID, &r.AuthorID, &r.Version, &r.Title,
		&contentJSON, &r.Excerpt, &r.Slug, &postMetaJSON, &metadataJSON,
		&r.Summary, &r.ChangeLog, &r.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrRevisionNotFound
		}
		return nil, fmt.Errorf("failed to get revision: %w", err)
	}

	if len(contentJSON) > 0 {
		_ = json.Unmarshal([]byte(contentJSON), &r.Content)
	}
	if len(postMetaJSON) > 0 {
		_ = json.Unmarshal([]byte(postMetaJSON), &r.PostMeta)
	}
	if len(metadataJSON) > 0 {
		_ = json.Unmarshal([]byte(metadataJSON), &r.Metadata)
	}
	if r.Content == nil {
		r.Content = []interface{}{}
	}
	if r.PostMeta == nil {
		r.PostMeta = make(map[string]interface{})
	}
	if r.Metadata == nil {
		r.Metadata = make(map[string]interface{})
	}

	return &r, nil
}

func (s *Service) RestoreRevision(ctx context.Context, siteID, postID, revisionID, userID uuid.UUID) (*Revision, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	revision, err := s.GetRevision(ctx, siteID, postID, revisionID)
	if err != nil {
		return nil, err
	}

	contentJSON, _ := json.Marshal(revision.Content)
	postMetaJSON, _ := json.Marshal(revision.PostMeta)
	metadataJSON, _ := json.Marshal(revision.Metadata)

	_, err = p.Exec(ctx,
		`UPDATE posts SET title = $1, content = $2::jsonb, excerpt = $3, slug = $4,
		 post_meta = $5::jsonb, metadata = $6::jsonb, updated_at = NOW()
		 WHERE id = $7 AND site_id = $8 AND deleted_at IS NULL`,
		revision.Title, string(contentJSON), revision.Excerpt, revision.Slug,
		string(postMetaJSON), string(metadataJSON), postID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to restore revision: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		EntityType: "post",
		EntityID:   &postID,
		Action:     audit.Action("post.revision_restored"),
		Payload:    map[string]interface{}{"version": revision.Version},
	})

	s.fireEvent(ctx, EventRevisionRestored, map[string]interface{}{
		"post_id":     postID.String(),
		"site_id":     siteID.String(),
		"revision_id": revisionID.String(),
		"version":     revision.Version,
	}, siteID)

	return revision, nil
}

func (s *Service) RequestApproval(ctx context.Context, siteID, postID, userID uuid.UUID) (*ApprovalRequest, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	reqID := uuid.New()
	now := time.Now()

	_, err = p.Exec(ctx,
		`INSERT INTO approval_requests (id, site_id, post_id, requested_by, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'pending', $5, $6)`,
		reqID, siteID, postID, userID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create approval request: %w", err)
	}

	req := &ApprovalRequest{
		ID:          reqID,
		SiteID:      siteID,
		PostID:      postID,
		RequestedBy: userID,
		Status:      ApprovalStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("editorial.approval.requested"),
		EntityType: "approval_request",
		EntityID:   &reqID,
		Payload:    map[string]interface{}{"post_id": postID.String()},
	})

	s.fireEvent(ctx, EventApprovalRequested, map[string]interface{}{
		"approval_id": reqID.String(),
		"post_id":     postID.String(),
		"site_id":     siteID.String(),
	}, siteID)

	return req, nil
}

func (s *Service) ReviewApproval(ctx context.Context, siteID, postID, approvalID, reviewerID uuid.UUID, req ApprovalActionRequest) (*ApprovalRequest, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	if req.Status != ApprovalStatusApproved && req.Status != ApprovalStatusRejected {
		return nil, fmt.Errorf("invalid approval status: %s", req.Status)
	}

	now := time.Now()

	_, err = p.Exec(ctx,
		`UPDATE approval_requests SET status = $1, comments = $2, reviewed_by = $3, reviewed_at = $4, updated_at = $5
		 WHERE id = $6 AND post_id = $7 AND site_id = $8 AND status = 'pending'`,
		string(req.Status), req.Comments, reviewerID, now, now, approvalID, postID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to review approval: %w", err)
	}

	eventType := EventApprovalGranted
	actionStr := "editorial.approval.granted"
	if req.Status == ApprovalStatusRejected {
		eventType = EventApprovalRejected
		actionStr = "editorial.approval.rejected"
	}

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &reviewerID,
		SiteID:     &siteID,
		Action:     audit.Action(actionStr),
		EntityType: "approval_request",
		EntityID:   &approvalID,
		Payload:    map[string]interface{}{"post_id": postID.String(), "status": string(req.Status)},
	})

	s.fireEvent(ctx, eventType, map[string]interface{}{
		"approval_id": approvalID.String(),
		"post_id":     postID.String(),
		"site_id":     siteID.String(),
		"status":      string(req.Status),
	}, siteID)

	var ar ApprovalRequest
	err = p.QueryRow(ctx,
		`SELECT id, site_id, post_id, requested_by, status, COALESCE(comments, ''), reviewed_by, reviewed_at, created_at, updated_at
		 FROM approval_requests WHERE id = $1`, approvalID,
	).Scan(&ar.ID, &ar.SiteID, &ar.PostID, &ar.RequestedBy, &ar.Status, &ar.Comments, &ar.ReviewedBy, &ar.ReviewedAt, &ar.CreatedAt, &ar.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated approval: %w", err)
	}

	return &ar, nil
}

func (s *Service) ListApprovals(ctx context.Context, siteID uuid.UUID, status ApprovalStatus) ([]ApprovalRequest, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var rows pgx.Rows
	if status == "" {
		rows, err = p.Query(ctx,
			`SELECT id, site_id, post_id, requested_by, status, COALESCE(comments, ''), reviewed_by, reviewed_at, created_at, updated_at
			 FROM approval_requests WHERE site_id = $1 ORDER BY created_at DESC`,
			siteID,
		)
	} else {
		rows, err = p.Query(ctx,
			`SELECT id, site_id, post_id, requested_by, status, COALESCE(comments, ''), reviewed_by, reviewed_at, created_at, updated_at
			 FROM approval_requests WHERE site_id = $1 AND status = $2 ORDER BY created_at DESC`,
			siteID, string(status),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list approvals: %w", err)
	}
	defer rows.Close()

	var approvals []ApprovalRequest
	for rows.Next() {
		var a ApprovalRequest
		if err := rows.Scan(&a.ID, &a.SiteID, &a.PostID, &a.RequestedBy, &a.Status, &a.Comments,
			&a.ReviewedBy, &a.ReviewedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan approval: %w", err)
		}
		approvals = append(approvals, a)
	}
	if approvals == nil {
		approvals = []ApprovalRequest{}
	}
	return approvals, nil
}

func (s *Service) GetApproval(ctx context.Context, siteID, approvalID uuid.UUID) (*ApprovalRequest, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var a ApprovalRequest
	err = p.QueryRow(ctx,
		`SELECT id, site_id, post_id, requested_by, status, COALESCE(comments, ''), reviewed_by, reviewed_at, created_at, updated_at
		 FROM approval_requests WHERE id = $1 AND site_id = $2`,
		approvalID, siteID,
	).Scan(&a.ID, &a.SiteID, &a.PostID, &a.RequestedBy, &a.Status, &a.Comments,
		&a.ReviewedBy, &a.ReviewedAt, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrApprovalNotFound
		}
		return nil, fmt.Errorf("failed to get approval: %w", err)
	}
	return &a, nil
}

func (s *Service) CreateCalendarEvent(ctx context.Context, siteID, userID uuid.UUID, req CreateCalendarEventRequest) (*CalendarEvent, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	eventType := req.EventType
	if eventType == "" {
		eventType = "publication"
	}

	eventID := uuid.New()
	now := time.Now()

	_, err = p.Exec(ctx,
		`INSERT INTO editorial_calendar_events (id, site_id, title, description, event_date, event_type, post_id, color, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		eventID, siteID, req.Title, req.Description, req.EventDate, eventType, req.PostID, req.Color, userID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar event: %w", err)
	}

	event := &CalendarEvent{
		ID:          eventID,
		SiteID:      siteID,
		Title:       req.Title,
		Description: req.Description,
		EventDate:   req.EventDate,
		EventType:   eventType,
		PostID:      req.PostID,
		Color:       req.Color,
		CreatedBy:   &userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.fireEvent(ctx, EventCalendarEventCreated, map[string]interface{}{
		"event_id":   eventID.String(),
		"site_id":    siteID.String(),
		"title":      req.Title,
		"event_date": req.EventDate,
	}, siteID)

	return event, nil
}

func (s *Service) ListCalendarEvents(ctx context.Context, siteID uuid.UUID, startDate, endDate string) ([]CalendarEvent, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var rows pgx.Rows
	if startDate != "" && endDate != "" {
		rows, err = p.Query(ctx,
			`SELECT id, site_id, title, COALESCE(description, ''), event_date, event_type, post_id, COALESCE(color, ''), created_by, created_at, updated_at
			 FROM editorial_calendar_events
			 WHERE site_id = $1 AND event_date >= $2 AND event_date <= $3
			 ORDER BY event_date ASC`,
			siteID, startDate, endDate,
		)
	} else {
		rows, err = p.Query(ctx,
			`SELECT id, site_id, title, COALESCE(description, ''), event_date, event_type, post_id, COALESCE(color, ''), created_by, created_at, updated_at
			 FROM editorial_calendar_events
			 WHERE site_id = $1
			 ORDER BY event_date ASC`,
			siteID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list calendar events: %w", err)
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var e CalendarEvent
		if err := rows.Scan(&e.ID, &e.SiteID, &e.Title, &e.Description, &e.EventDate, &e.EventType,
			&e.PostID, &e.Color, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan calendar event: %w", err)
		}
		events = append(events, e)
	}
	if events == nil {
		events = []CalendarEvent{}
	}
	return events, nil
}

func (s *Service) UpdateCalendarEvent(ctx context.Context, siteID, eventID uuid.UUID, req UpdateCalendarEventRequest) (*CalendarEvent, error) {
	p, err := s.pool()
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
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.EventDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("event_date = $%d", argIdx))
		args = append(args, *req.EventDate)
		argIdx++
	}
	if req.EventType != nil {
		setClauses = append(setClauses, fmt.Sprintf("event_type = $%d", argIdx))
		args = append(args, *req.EventType)
		argIdx++
	}
	if req.PostID != nil {
		setClauses = append(setClauses, fmt.Sprintf("post_id = $%d", argIdx))
		args = append(args, *req.PostID)
		argIdx++
	}
	if req.Color != nil {
		setClauses = append(setClauses, fmt.Sprintf("color = $%d", argIdx))
		args = append(args, *req.Color)
		argIdx++
	}

	if len(setClauses) == 0 {
		return s.getCalendarEvent(ctx, siteID, eventID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE editorial_calendar_events SET %s WHERE id = $%d AND site_id = $%d`,
		stringsJoin(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, eventID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update calendar event: %w", err)
	}

	s.fireEvent(ctx, EventCalendarEventUpdated, map[string]interface{}{
		"event_id": eventID.String(),
		"site_id":  siteID.String(),
	}, siteID)

	return s.getCalendarEvent(ctx, siteID, eventID)
}

func (s *Service) DeleteCalendarEvent(ctx context.Context, siteID, eventID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx,
		`DELETE FROM editorial_calendar_events WHERE id = $1 AND site_id = $2`,
		eventID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete calendar event: %w", err)
	}

	return nil
}

func (s *Service) getCalendarEvent(ctx context.Context, siteID, eventID uuid.UUID) (*CalendarEvent, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var e CalendarEvent
	err = p.QueryRow(ctx,
		`SELECT id, site_id, title, COALESCE(description, ''), event_date, event_type, post_id, COALESCE(color, ''), created_by, created_at, updated_at
		 FROM editorial_calendar_events WHERE id = $1 AND site_id = $2`,
		eventID, siteID,
	).Scan(&e.ID, &e.SiteID, &e.Title, &e.Description, &e.EventDate, &e.EventType,
		&e.PostID, &e.Color, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrCalendarEventNotFound
		}
		return nil, fmt.Errorf("failed to get calendar event: %w", err)
	}
	return &e, nil
}

func (s *Service) ListWidgets(ctx context.Context, siteID uuid.UUID) ([]Widget, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, site_id, widget_type, title, COALESCE(config::text, '{}'), position, enabled, created_at, updated_at
		 FROM editorial_widgets WHERE site_id = $1 ORDER BY position ASC`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list widgets: %w", err)
	}
	defer rows.Close()

	var widgets []Widget
	for rows.Next() {
		var w Widget
		var configJSON string
		if err := rows.Scan(&w.ID, &w.SiteID, &w.WidgetType, &w.Title, &configJSON, &w.Position, &w.Enabled, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan widget: %w", err)
		}
		if len(configJSON) > 0 {
			_ = json.Unmarshal([]byte(configJSON), &w.Config)
		}
		if w.Config == nil {
			w.Config = make(map[string]interface{})
		}
		widgets = append(widgets, w)
	}
	if widgets == nil {
		widgets = []Widget{}
	}
	return widgets, nil
}

func (s *Service) UpdateWidget(ctx context.Context, siteID, widgetID uuid.UUID, req UpdateWidgetRequest) (*Widget, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.WidgetType != nil {
		setClauses = append(setClauses, fmt.Sprintf("widget_type = $%d", argIdx))
		args = append(args, *req.WidgetType)
		argIdx++
	}
	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.Config != nil {
		b, _ := json.Marshal(*req.Config)
		setClauses = append(setClauses, fmt.Sprintf("config = $%d::jsonb", argIdx))
		args = append(args, string(b))
		argIdx++
	}
	if req.Position != nil {
		setClauses = append(setClauses, fmt.Sprintf("position = $%d", argIdx))
		args = append(args, *req.Position)
		argIdx++
	}
	if req.Enabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("enabled = $%d", argIdx))
		args = append(args, *req.Enabled)
		argIdx++
	}

	if len(setClauses) == 0 {
		return s.getWidget(ctx, siteID, widgetID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE editorial_widgets SET %s WHERE id = $%d AND site_id = $%d`,
		stringsJoin(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, widgetID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update widget: %w", err)
	}

	return s.getWidget(ctx, siteID, widgetID)
}

func (s *Service) getWidget(ctx context.Context, siteID, widgetID uuid.UUID) (*Widget, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var w Widget
	var configJSON string
	err = p.QueryRow(ctx,
		`SELECT id, site_id, widget_type, title, COALESCE(config::text, '{}'), position, enabled, created_at, updated_at
		 FROM editorial_widgets WHERE id = $1 AND site_id = $2`,
		widgetID, siteID,
	).Scan(&w.ID, &w.SiteID, &w.WidgetType, &w.Title, &configJSON, &w.Position, &w.Enabled, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrWidgetNotFound
		}
		return nil, fmt.Errorf("failed to get widget: %w", err)
	}
	if len(configJSON) > 0 {
		_ = json.Unmarshal([]byte(configJSON), &w.Config)
	}
	if w.Config == nil {
		w.Config = make(map[string]interface{})
	}
	return &w, nil
}

func stringsJoin(elems []string, sep string) string {
	result := ""
	for i, e := range elems {
		if i > 0 {
			result += sep
		}
		result += e
	}
	return result
}
