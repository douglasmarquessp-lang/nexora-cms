package tags

import (
	"context"
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

func generateSlug(name string) string {
	slug := strings.ToLower(name)
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
			query = `SELECT COUNT(*) FROM tags WHERE site_id = $1 AND slug = $2 AND id != $3 AND deleted_at IS NULL`
			args = []interface{}{siteID, candidate, *excludeID}
		} else {
			query = `SELECT COUNT(*) FROM tags WHERE site_id = $1 AND slug = $2 AND deleted_at IS NULL`
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

func (s *Service) Create(ctx context.Context, siteID uuid.UUID, req CreateTagRequest) (*Tag, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	slug := generateSlug(req.Name)
	slug, err = s.ensureUniqueSlug(ctx, siteID, slug, nil)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	tagID := uuid.New()

	_, err = p.Exec(ctx,
		`INSERT INTO tags (id, site_id, name, slug, color, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		tagID, siteID, req.Name, slug, req.Color, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	tag := &Tag{
		ID:        tagID,
		SiteID:    siteID,
		Name:      req.Name,
		Slug:      slug,
		Color:     req.Color,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		EntityType: "tag",
		EntityID:   &tagID,
		Action:     audit.Action("tag.created"),
		Payload:    map[string]interface{}{"name": req.Name, "slug": slug},
	})

	s.fireEvent(ctx, EventTagCreated, map[string]interface{}{
		"tag_id":  tagID.String(),
		"site_id": siteID.String(),
		"name":    req.Name,
		"slug":    slug,
	}, siteID)

	return tag, nil
}

func (s *Service) GetByID(ctx context.Context, siteID, tagID uuid.UUID) (*Tag, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var tag Tag
	var deletedAt *time.Time

	err = p.QueryRow(ctx,
		`SELECT id, site_id, name, slug, COALESCE(color, ''), created_at, updated_at, deleted_at
		 FROM tags WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		tagID, siteID,
	).Scan(
		&tag.ID, &tag.SiteID, &tag.Name, &tag.Slug, &tag.Color,
		&tag.CreatedAt, &tag.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTagNotFound
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	tag.DeletedAt = deletedAt
	return &tag, nil
}

func (s *Service) GetBySlug(ctx context.Context, siteID uuid.UUID, slug string) (*Tag, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var tag Tag
	var deletedAt *time.Time

	err = p.QueryRow(ctx,
		`SELECT id, site_id, name, slug, COALESCE(color, ''), created_at, updated_at, deleted_at
		 FROM tags WHERE site_id = $1 AND slug = $2 AND deleted_at IS NULL`,
		siteID, slug,
	).Scan(
		&tag.ID, &tag.SiteID, &tag.Name, &tag.Slug, &tag.Color,
		&tag.CreatedAt, &tag.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTagNotFound
		}
		return nil, fmt.Errorf("failed to get tag by slug: %w", err)
	}

	tag.DeletedAt = deletedAt
	return &tag, nil
}

func (s *Service) List(ctx context.Context, siteID uuid.UUID) (*TagListResponse, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, site_id, name, slug, COALESCE(color, ''), created_at, updated_at, deleted_at
		 FROM tags WHERE site_id = $1 AND deleted_at IS NULL
		 ORDER BY name ASC`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
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

	var total int
	if err := p.QueryRow(ctx, `SELECT COUNT(*) FROM tags WHERE site_id = $1 AND deleted_at IS NULL`, siteID).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count tags: %w", err)
	}

	return &TagListResponse{
		Tags:  tags,
		Total: total,
	}, nil
}

func (s *Service) Update(ctx context.Context, siteID, tagID uuid.UUID, req UpdateTagRequest) (*Tag, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, err := s.GetByID(ctx, siteID, tagID)
	if err != nil {
		return nil, err
	}

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++

		newSlug := generateSlug(*req.Name)
		if newSlug != existing.Slug {
			uniqueSlug, err := s.ensureUniqueSlug(ctx, siteID, newSlug, &tagID)
			if err != nil {
				return nil, err
			}
			setClauses = append(setClauses, fmt.Sprintf("slug = $%d", argIdx))
			args = append(args, uniqueSlug)
			argIdx++
		}
	}
	if req.Color != nil {
		setClauses = append(setClauses, fmt.Sprintf("color = $%d", argIdx))
		args = append(args, *req.Color)
		argIdx++
	}

	if len(setClauses) == 0 {
		return existing, nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf(
		`UPDATE tags SET %s WHERE id = $%d AND site_id = $%d AND deleted_at IS NULL`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, tagID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}

	updated, err := s.GetByID(ctx, siteID, tagID)
	if err != nil {
		return nil, err
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		EntityType: "tag",
		EntityID:   &tagID,
		Action:     audit.Action("tag.updated"),
		Payload:    map[string]interface{}{"name": updated.Name},
	})

	s.fireEvent(ctx, EventTagUpdated, map[string]interface{}{
		"tag_id":  tagID.String(),
		"site_id": siteID.String(),
		"name":    updated.Name,
	}, siteID)

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, siteID, tagID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	existing, err := s.GetByID(ctx, siteID, tagID)
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx,
		`UPDATE tags SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		tagID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		EntityType: "tag",
		EntityID:   &tagID,
		Action:     audit.Action("tag.deleted"),
		Payload:    map[string]interface{}{"name": existing.Name, "slug": existing.Slug},
	})

	s.fireEvent(ctx, EventTagDeleted, map[string]interface{}{
		"tag_id":  tagID.String(),
		"site_id": siteID.String(),
		"name":    existing.Name,
	}, siteID)

	return nil
}
