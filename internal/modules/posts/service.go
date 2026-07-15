package posts

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
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

var slugRegex = regexp.MustCompile(`[^a-z0-9-]+`)
var multiDashRegex = regexp.MustCompile(`-{2,}`)

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

func (s *Service) fireEvent(ctx context.Context, eventType kernel.EventType, payload interface{}) {
	if s.eventBus != nil {
		s.eventBus.EmitAsync(ctx, eventType, payload, "")
	}
}

func (s *Service) pool() (database.Pool, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, ErrDatabaseNotAvail
	}
	return s.db.Pool, nil
}

func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = slugRegex.ReplaceAllString(slug, "-")
	slug = multiDashRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func (s *Service) ensureUniqueSlug(ctx context.Context, siteID uuid.UUID, slug string, excludeID *uuid.UUID) (string, error) {
	p, err := s.pool()
	if err != nil {
		return "", err
	}

	candidate := slug
	suffix := 0

	for {
		var count int
		var query string
		var args []interface{}

		if excludeID != nil {
			query = `SELECT COUNT(*) FROM posts WHERE site_id = $1 AND slug = $2 AND id != $3 AND deleted_at IS NULL`
			args = []interface{}{siteID, candidate, *excludeID}
		} else {
			query = `SELECT COUNT(*) FROM posts WHERE site_id = $1 AND slug = $2 AND deleted_at IS NULL`
			args = []interface{}{siteID, candidate}
		}

		err := p.QueryRow(ctx, query, args...).Scan(&count)
		if err != nil {
			return "", fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		if count == 0 {
			return candidate, nil
		}
		suffix++
		candidate = fmt.Sprintf("%s-%d", slug, suffix)
	}
}

func (s *Service) Create(ctx context.Context, siteID, authorID uuid.UUID, req CreatePostRequest) (*Post, error) {
	if req.Status != "" && !isValidStatus(req.Status) {
		return nil, ErrInvalidPostStatus
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	slug := generateSlug(req.Title)
	slug, err = s.ensureUniqueSlug(ctx, siteID, slug, nil)
	if err != nil {
		return nil, err
	}

	status := req.Status
	if status == "" {
		status = PostStatusDraft
	}

	now := time.Now()
	postID := uuid.New()

	contentJSON := "[]"
	if len(req.Content) > 0 {
		b, _ := json.Marshal(req.Content)
		contentJSON = string(b)
	}

	postMetaJSON := "{}"
	if req.PostMeta != nil {
		b, _ := json.Marshal(req.PostMeta)
		postMetaJSON = string(b)
	}

	var publishedAt *time.Time
	if req.PublishedAt != nil {
		publishedAt = req.PublishedAt
	} else if status == PostStatusPublished {
		publishedAt = &now
	}

	_, err = p.Exec(ctx,
		`INSERT INTO posts (id, site_id, title, slug, content, excerpt, status, author_id, published_at, scheduled_at, post_meta, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10, $11::jsonb, '{}', $12, $13)`,
		postID, siteID, req.Title, slug, contentJSON, req.Excerpt, string(status), authorID,
		publishedAt, req.ScheduledAt, postMetaJSON, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	if err := s.setPostRelations(ctx, p, postID, req.CategoryIDs, req.TagIDs); err != nil {
		return nil, err
	}

	post := &Post{
		ID:          postID,
		SiteID:      siteID,
		Title:       req.Title,
		Slug:        slug,
		Content:     req.Content,
		Excerpt:     req.Excerpt,
		Status:      status,
		AuthorID:    authorID,
		PublishedAt: publishedAt,
		ScheduledAt: req.ScheduledAt,
		PostMeta:    req.PostMeta,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if post.Content == nil {
		post.Content = []interface{}{}
	}
	if post.PostMeta == nil {
		post.PostMeta = make(map[string]interface{})
	}

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &authorID,
		SiteID:     &siteID,
		Action:     audit.Action("post.created"),
		EntityType: "post",
		EntityID:   &postID,
		Payload:    map[string]interface{}{"title": req.Title, "slug": slug, "status": string(status)},
	})

	s.fireEvent(ctx, EventPostCreated, map[string]interface{}{
		"post_id": postID.String(),
		"site_id": siteID.String(),
		"title":   req.Title,
		"slug":    slug,
		"status":  string(status),
	})

	if status == PostStatusPublished {
		s.fireEvent(ctx, EventPostPublished, map[string]interface{}{
			"post_id": postID.String(),
			"site_id": siteID.String(),
			"title":   req.Title,
			"slug":    slug,
		})
	}

	return post, nil
}

func (s *Service) GetByID(ctx context.Context, siteID, postID uuid.UUID) (*Post, error) {
	return s.getPost(ctx, siteID, &postID, "")
}

func (s *Service) GetBySlug(ctx context.Context, siteID uuid.UUID, slug string) (*Post, error) {
	return s.getPost(ctx, siteID, nil, slug)
}

func (s *Service) getPost(ctx context.Context, siteID uuid.UUID, postID *uuid.UUID, slug string) (*Post, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var post Post
	var contentJSON, postMetaJSON, metadataJSON []byte
	var deletedAt *time.Time
	var publishedAt, scheduledAt *time.Time

	var query string
	var args []interface{}

	if postID != nil {
		query = `SELECT id, site_id, title, slug, COALESCE(content::text, '[]'), COALESCE(excerpt, ''), status,
		         author_id, published_at, scheduled_at, COALESCE(post_meta::text, '{}'), COALESCE(metadata::text, '{}'),
		         created_at, updated_at, deleted_at
		 FROM posts WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`
		args = []interface{}{*postID, siteID}
	} else {
		query = `SELECT id, site_id, title, slug, COALESCE(content::text, '[]'), COALESCE(excerpt, ''), status,
		         author_id, published_at, scheduled_at, COALESCE(post_meta::text, '{}'), COALESCE(metadata::text, '{}'),
		         created_at, updated_at, deleted_at
		 FROM posts WHERE slug = $1 AND site_id = $2 AND deleted_at IS NULL`
		args = []interface{}{slug, siteID}
	}

	err = p.QueryRow(ctx, query, args...).Scan(
		&post.ID, &post.SiteID, &post.Title, &post.Slug, &contentJSON, &post.Excerpt, &post.Status,
		&post.AuthorID, &publishedAt, &scheduledAt, &postMetaJSON, &metadataJSON,
		&post.CreatedAt, &post.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	post.DeletedAt = deletedAt
	post.PublishedAt = publishedAt
	post.ScheduledAt = scheduledAt

	if len(contentJSON) > 0 {
		json.Unmarshal(contentJSON, &post.Content)
	}
	if len(postMetaJSON) > 0 {
		json.Unmarshal(postMetaJSON, &post.PostMeta)
	}
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &post.Metadata)
	}
	if post.Content == nil {
		post.Content = []interface{}{}
	}
	if post.PostMeta == nil {
		post.PostMeta = make(map[string]interface{})
	}
	if post.Metadata == nil {
		post.Metadata = make(map[string]interface{})
	}

	categories, err := s.getPostCategories(ctx, p, post.ID)
	if err != nil {
		return nil, err
	}
	post.Categories = categories

	tags, err := s.getPostTags(ctx, p, post.ID)
	if err != nil {
		return nil, err
	}
	post.Tags = tags

	return &post, nil
}

func (s *Service) List(ctx context.Context, req PostListRequest) (*PostListResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	allowedSorts := map[string]bool{"created_at": true, "updated_at": true, "title": true, "published_at": true}
	sortColumn := "created_at"
	if req.Sort != "" && allowedSorts[req.Sort] {
		sortColumn = req.Sort
	}

	orderDir := "DESC"
	if strings.EqualFold(req.Order, "asc") {
		orderDir = "ASC"
	}

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	whereClauses = append(whereClauses, fmt.Sprintf("p.deleted_at IS NULL"))
	whereClauses = append(whereClauses, fmt.Sprintf("p.site_id = $%d", argIdx))
	args = append(args, req.SiteID)
	argIdx++

	if req.Status != "" {
		if !isValidStatus(req.Status) {
			return nil, ErrInvalidPostStatus
		}
		whereClauses = append(whereClauses, fmt.Sprintf("p.status = $%d", argIdx))
		args = append(args, string(req.Status))
		argIdx++
	}

	if req.AuthorID != uuid.Nil {
		whereClauses = append(whereClauses, fmt.Sprintf("p.author_id = $%d", argIdx))
		args = append(args, req.AuthorID)
		argIdx++
	}

	if req.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(p.title ILIKE $%d OR p.slug ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+req.Search+"%")
		argIdx++
	}

	if req.CategoryID != uuid.Nil {
		whereClauses = append(whereClauses, fmt.Sprintf("EXISTS (SELECT 1 FROM post_categories pc WHERE pc.post_id = p.id AND pc.category_id = $%d)", argIdx))
		args = append(args, req.CategoryID)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM posts p WHERE %s`, whereSQL)
	err = p.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count posts: %w", err)
	}

	offset := (req.Page - 1) * req.PerPage

	query := fmt.Sprintf(
		`SELECT p.id, p.title, p.slug, COALESCE(p.excerpt, ''), p.status, p.author_id,
		        p.published_at, p.created_at, p.updated_at,
		        (SELECT COUNT(*) FROM post_categories pc WHERE pc.post_id = p.id) AS category_count,
		        (SELECT COUNT(*) FROM post_tags pt WHERE pt.post_id = p.id) AS tag_count
		 FROM posts p
		 WHERE %s
		 ORDER BY p.%s %s
		 LIMIT $%d OFFSET $%d`,
		whereSQL, sortColumn, orderDir, argIdx, argIdx+1,
	)
	args = append(args, req.PerPage, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}
	defer rows.Close()

	var summaries []PostSummary
	for rows.Next() {
		var s PostSummary
		var publishedAt *time.Time

		err := rows.Scan(
			&s.ID, &s.Title, &s.Slug, &s.Excerpt, &s.Status, &s.AuthorID,
			&publishedAt, &s.CreatedAt, &s.UpdatedAt,
			&s.CategoryCount, &s.TagCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post summary: %w", err)
		}
		s.PublishedAt = publishedAt
		summaries = append(summaries, s)
	}

	if summaries == nil {
		summaries = []PostSummary{}
	}

	return &PostListResponse{
		Posts:   summaries,
		Total:   total,
		Page:    req.Page,
		PerPage: req.PerPage,
	}, nil
}

func (s *Service) Update(ctx context.Context, siteID, postID uuid.UUID, req UpdatePostRequest) (*Post, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, err := s.GetByID(ctx, siteID, postID)
	if err != nil {
		return nil, err
	}

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++

		newSlug := generateSlug(*req.Title)
		if newSlug != existing.Slug {
			uniqueSlug, err := s.ensureUniqueSlug(ctx, siteID, newSlug, &postID)
			if err != nil {
				return nil, err
			}
			setClauses = append(setClauses, fmt.Sprintf("slug = $%d", argIdx))
			args = append(args, uniqueSlug)
			argIdx++
		}
	}
	if req.Content != nil {
		b, _ := json.Marshal(*req.Content)
		setClauses = append(setClauses, fmt.Sprintf("content = $%d::jsonb", argIdx))
		args = append(args, string(b))
		argIdx++
	}
	if req.Excerpt != nil {
		setClauses = append(setClauses, fmt.Sprintf("excerpt = $%d", argIdx))
		args = append(args, *req.Excerpt)
		argIdx++
	}
	if req.Status != nil {
		if !isValidStatus(*req.Status) {
			return nil, ErrInvalidPostStatus
		}
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*req.Status))
		argIdx++

		if *req.Status == PostStatusPublished && existing.PublishedAt == nil {
			now := time.Now()
			setClauses = append(setClauses, fmt.Sprintf("published_at = $%d", argIdx))
			args = append(args, now)
			argIdx++
		}
	}
	if req.PublishedAt != nil {
		setClauses = append(setClauses, fmt.Sprintf("published_at = $%d", argIdx))
		if *req.PublishedAt != nil {
			args = append(args, **req.PublishedAt)
		} else {
			args = append(args, nil)
		}
		argIdx++
	}
	if req.ScheduledAt != nil {
		setClauses = append(setClauses, fmt.Sprintf("scheduled_at = $%d", argIdx))
		if *req.ScheduledAt != nil {
			args = append(args, **req.ScheduledAt)
		} else {
			args = append(args, nil)
		}
		argIdx++
	}
	if req.PostMeta != nil {
		b, _ := json.Marshal(*req.PostMeta)
		setClauses = append(setClauses, fmt.Sprintf("post_meta = $%d::jsonb", argIdx))
		args = append(args, string(b))
		argIdx++
	}

	if len(setClauses) > 0 || req.CategoryIDs != nil || req.TagIDs != nil {
		if len(setClauses) > 0 {
			setClauses = append(setClauses, "updated_at = NOW()")
			updateQuery := fmt.Sprintf(
				`UPDATE posts SET %s WHERE id = $%d AND site_id = $%d AND deleted_at IS NULL`,
				strings.Join(setClauses, ", "), argIdx, argIdx+1,
			)
			args = append(args, postID, siteID)

			_, err = p.Exec(ctx, updateQuery, args...)
			if err != nil {
				return nil, fmt.Errorf("failed to update post: %w", err)
			}
		}

		if req.CategoryIDs != nil || req.TagIDs != nil {
			if err := s.setPostRelations(ctx, p, postID, req.CategoryIDs, req.TagIDs); err != nil {
				return nil, err
			}
		}

		s.fireEvent(ctx, EventPostUpdated, map[string]interface{}{
			"post_id": postID.String(),
			"site_id": siteID.String(),
			"title":   existing.Title,
		})

		if req.Status != nil {
			switch *req.Status {
			case PostStatusPublished:
				s.fireEvent(ctx, EventPostPublished, map[string]interface{}{
					"post_id": postID.String(),
					"site_id": siteID.String(),
				})
			case PostStatusArchived:
				s.fireEvent(ctx, EventPostArchived, map[string]interface{}{
					"post_id": postID.String(),
					"site_id": siteID.String(),
				})
			}
		}

		s.auditLog.Log(ctx, audit.Entry{
			SiteID:     &siteID,
			EntityType: "post",
			EntityID:   &postID,
			Action:     audit.Action("post.updated"),
			Payload:    map[string]interface{}{"title": existing.Title},
		})
	}

	return s.GetByID(ctx, siteID, postID)
}

func (s *Service) Delete(ctx context.Context, siteID, postID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	post, err := s.GetByID(ctx, siteID, postID)
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx,
		`UPDATE posts SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		postID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete post: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		EntityType: "post",
		EntityID:   &postID,
		Action:     audit.Action("post.deleted"),
		Payload:    map[string]interface{}{"title": post.Title, "slug": post.Slug},
	})

	s.fireEvent(ctx, EventPostDeleted, map[string]interface{}{
		"post_id": postID.String(),
		"site_id": siteID.String(),
		"slug":    post.Slug,
	})

	return nil
}

func (s *Service) SetStatus(ctx context.Context, siteID, postID uuid.UUID, status PostStatus) error {
	if !isValidStatus(status) {
		return ErrInvalidPostStatus
	}

	p, err := s.pool()
	if err != nil {
		return err
	}

	existing, err := s.GetByID(ctx, siteID, postID)
	if err != nil {
		return err
	}

	now := time.Now()
	var publishedAt *time.Time
	if status == PostStatusPublished && existing.PublishedAt == nil {
		publishedAt = &now
	}

	_, err = p.Exec(ctx,
		`UPDATE posts SET status = $1, published_at = COALESCE($2, published_at), updated_at = NOW() WHERE id = $3 AND site_id = $4 AND deleted_at IS NULL`,
		string(status), publishedAt, postID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to set post status: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		EntityType: "post",
		EntityID:   &postID,
		Action:     audit.Action("post.status_changed"),
		Payload:    map[string]interface{}{"from": string(existing.Status), "to": string(status)},
	})

	payload := map[string]interface{}{
		"post_id": postID.String(),
		"site_id": siteID.String(),
		"status":  string(status),
	}

	switch status {
	case PostStatusPublished:
		s.fireEvent(ctx, EventPostPublished, payload)
	case PostStatusArchived:
		s.fireEvent(ctx, EventPostArchived, payload)
	}

	return nil
}

func (s *Service) setPostRelations(ctx context.Context, p database.Pool, postID uuid.UUID, categoryIDs, tagIDs []uuid.UUID) error {
	if categoryIDs != nil {
		_, err := p.Exec(ctx, `DELETE FROM post_categories WHERE post_id = $1`, postID)
		if err != nil {
			return fmt.Errorf("failed to clear post categories: %w", err)
		}
		for _, catID := range categoryIDs {
			_, err := p.Exec(ctx,
				`INSERT INTO post_categories (post_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				postID, catID,
			)
			if err != nil {
				return fmt.Errorf("failed to link category %s: %w", catID, err)
			}
		}
	}

	if tagIDs != nil {
		_, err := p.Exec(ctx, `DELETE FROM post_tags WHERE post_id = $1`, postID)
		if err != nil {
			return fmt.Errorf("failed to clear post tags: %w", err)
		}
		for _, tagID := range tagIDs {
			_, err := p.Exec(ctx,
				`INSERT INTO post_tags (post_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				postID, tagID,
			)
			if err != nil {
				return fmt.Errorf("failed to link tag %s: %w", tagID, err)
			}
		}
	}

	return nil
}

func (s *Service) getPostCategories(ctx context.Context, p database.Pool, postID uuid.UUID) ([]Category, error) {
	rows, err := p.Query(ctx,
		`SELECT c.id, c.site_id, c.parent_id, c.name, c.slug, COALESCE(c.description, ''), COALESCE(c.icon, ''), COALESCE(c.color, ''), c.sort_order,
		        c.created_at, c.updated_at, c.deleted_at
		 FROM categories c
		 INNER JOIN post_categories pc ON pc.category_id = c.id
		 WHERE pc.post_id = $1 AND c.deleted_at IS NULL
		 ORDER BY c.sort_order ASC, c.name ASC`,
		postID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get post categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		var deletedAt *time.Time
		var parentID *uuid.UUID

		err := rows.Scan(
			&cat.ID, &cat.SiteID, &parentID, &cat.Name, &cat.Slug,
			&cat.Description, &cat.Icon, &cat.Color, &cat.SortOrder,
			&cat.CreatedAt, &cat.UpdatedAt, &deletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		cat.ParentID = parentID
		cat.DeletedAt = deletedAt
		categories = append(categories, cat)
	}
	if categories == nil {
		categories = []Category{}
	}
	return categories, nil
}

func (s *Service) getPostTags(ctx context.Context, p database.Pool, postID uuid.UUID) ([]Tag, error) {
	rows, err := p.Query(ctx,
		`SELECT t.id, t.site_id, t.name, t.slug, COALESCE(t.color, ''), t.created_at, t.updated_at, t.deleted_at
		 FROM tags t
		 INNER JOIN post_tags pt ON pt.tag_id = t.id
		 WHERE pt.post_id = $1 AND t.deleted_at IS NULL
		 ORDER BY t.name ASC`,
		postID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get post tags: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		var deletedAt *time.Time

		err := rows.Scan(
			&tag.ID, &tag.SiteID, &tag.Name, &tag.Slug, &tag.Color,
			&tag.CreatedAt, &tag.UpdatedAt, &deletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tag.DeletedAt = deletedAt
		tags = append(tags, tag)
	}
	if tags == nil {
		tags = []Tag{}
	}
	return tags, nil
}

func isValidStatus(status PostStatus) bool {
	switch status {
	case PostStatusDraft, PostStatusPublished, PostStatusScheduled, PostStatusArchived:
		return true
	default:
		return false
	}
}
