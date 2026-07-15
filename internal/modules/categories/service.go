package categories

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
			query = `SELECT COUNT(*) FROM categories WHERE site_id = $1 AND slug = $2 AND id != $3 AND deleted_at IS NULL`
			args = []interface{}{siteID, candidate, *excludeID}
		} else {
			query = `SELECT COUNT(*) FROM categories WHERE site_id = $1 AND slug = $2 AND deleted_at IS NULL`
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

func (s *Service) Create(ctx context.Context, siteID uuid.UUID, req CreateCategoryRequest) (*Category, error) {
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

	if req.ParentID != nil {
		parent, err := s.GetByID(ctx, siteID, *req.ParentID)
		if err != nil {
			return nil, ErrInvalidParentCategory
		}
		if parent == nil {
			return nil, ErrInvalidParentCategory
		}
	}

	now := time.Now()
	catID := uuid.New()

	_, err = p.Exec(ctx,
		`INSERT INTO categories (id, site_id, parent_id, name, slug, description, icon, color, sort_order, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		catID, siteID, req.ParentID, req.Name, slug, req.Description, req.Icon, req.Color, req.SortOrder, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	cat := &Category{
		ID:          catID,
		SiteID:      siteID,
		ParentID:    req.ParentID,
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Icon:        req.Icon,
		Color:       req.Color,
		SortOrder:   req.SortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		EntityType: "category",
		EntityID:   &catID,
		Action:     audit.Action("category.created"),
		Payload:    map[string]interface{}{"name": req.Name, "slug": slug},
	})

	s.fireEvent(ctx, EventCategoryCreated, map[string]interface{}{
		"category_id": catID.String(),
		"site_id":     siteID.String(),
		"name":        req.Name,
		"slug":        slug,
	})

	return cat, nil
}

func (s *Service) GetByID(ctx context.Context, siteID, catID uuid.UUID) (*Category, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var cat Category
	var deletedAt *time.Time
	var parentID *uuid.UUID

	err = p.QueryRow(ctx,
		`SELECT id, site_id, parent_id, name, slug, COALESCE(description, ''), COALESCE(icon, ''), COALESCE(color, ''), sort_order,
		        created_at, updated_at, deleted_at
		 FROM categories WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		catID, siteID,
	).Scan(
		&cat.ID, &cat.SiteID, &parentID, &cat.Name, &cat.Slug,
		&cat.Description, &cat.Icon, &cat.Color, &cat.SortOrder,
		&cat.CreatedAt, &cat.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	cat.ParentID = parentID
	cat.DeletedAt = deletedAt
	return &cat, nil
}

func (s *Service) GetBySlug(ctx context.Context, siteID uuid.UUID, slug string) (*Category, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var cat Category
	var deletedAt *time.Time
	var parentID *uuid.UUID

	err = p.QueryRow(ctx,
		`SELECT id, site_id, parent_id, name, slug, COALESCE(description, ''), COALESCE(icon, ''), COALESCE(color, ''), sort_order,
		        created_at, updated_at, deleted_at
		 FROM categories WHERE site_id = $1 AND slug = $2 AND deleted_at IS NULL`,
		siteID, slug,
	).Scan(
		&cat.ID, &cat.SiteID, &parentID, &cat.Name, &cat.Slug,
		&cat.Description, &cat.Icon, &cat.Color, &cat.SortOrder,
		&cat.CreatedAt, &cat.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to get category by slug: %w", err)
	}

	cat.ParentID = parentID
	cat.DeletedAt = deletedAt
	return &cat, nil
}

func (s *Service) List(ctx context.Context, siteID uuid.UUID) (*CategoryListResponse, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, site_id, parent_id, name, slug, COALESCE(description, ''), COALESCE(icon, ''), COALESCE(color, ''), sort_order,
		        created_at, updated_at, deleted_at
		 FROM categories WHERE site_id = $1 AND deleted_at IS NULL
		 ORDER BY sort_order ASC, name ASC`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
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

	var total int
	p.QueryRow(ctx, `SELECT COUNT(*) FROM categories WHERE site_id = $1 AND deleted_at IS NULL`, siteID).Scan(&total)

	return &CategoryListResponse{
		Categories: categories,
		Total:      total,
	}, nil
}

func (s *Service) Tree(ctx context.Context, siteID uuid.UUID) ([]Category, error) {
	resp, err := s.List(ctx, siteID)
	if err != nil {
		return nil, err
	}

	catMap := make(map[uuid.UUID]*Category)
	for i := range resp.Categories {
		cat := resp.Categories[i]
		cat.Children = []Category{}
		catMap[cat.ID] = &resp.Categories[i]
	}

	for i := range resp.Categories {
		cat := &resp.Categories[i]
		if cat.ParentID != nil {
			parent, ok := catMap[*cat.ParentID]
			if ok {
				parent.Children = append(parent.Children, *cat)
			}
		}
	}

	var roots []Category
	for _, cat := range catMap {
		if cat.ParentID == nil {
			roots = append(roots, *cat)
		}
	}

	if roots == nil {
		roots = []Category{}
	}

	return roots, nil
}

func (s *Service) Update(ctx context.Context, siteID, catID uuid.UUID, req UpdateCategoryRequest) (*Category, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, err := s.GetByID(ctx, siteID, catID)
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
			uniqueSlug, err := s.ensureUniqueSlug(ctx, siteID, newSlug, &catID)
			if err != nil {
				return nil, err
			}
			setClauses = append(setClauses, fmt.Sprintf("slug = $%d", argIdx))
			args = append(args, uniqueSlug)
			argIdx++
		}
	}
	if req.ParentID != nil {
		if *req.ParentID != nil {
			parentID := **req.ParentID
			if parentID == catID {
				return nil, ErrCircularParent
			}
			parent, err := s.GetByID(ctx, siteID, parentID)
			if err != nil {
				return nil, ErrInvalidParentCategory
			}
			if s.isDescendant(ctx, p, siteID, catID, parentID) {
				return nil, ErrCircularParent
			}
			_ = parent
			setClauses = append(setClauses, fmt.Sprintf("parent_id = $%d", argIdx))
			args = append(args, parentID)
		} else {
			setClauses = append(setClauses, fmt.Sprintf("parent_id = $%d", argIdx))
			args = append(args, nil)
		}
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Icon != nil {
		setClauses = append(setClauses, fmt.Sprintf("icon = $%d", argIdx))
		args = append(args, *req.Icon)
		argIdx++
	}
	if req.Color != nil {
		setClauses = append(setClauses, fmt.Sprintf("color = $%d", argIdx))
		args = append(args, *req.Color)
		argIdx++
	}
	if req.SortOrder != nil {
		setClauses = append(setClauses, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *req.SortOrder)
		argIdx++
	}

	if len(setClauses) == 0 {
		return existing, nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf(
		`UPDATE categories SET %s WHERE id = $%d AND site_id = $%d AND deleted_at IS NULL`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, catID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	updated, err := s.GetByID(ctx, siteID, catID)
	if err != nil {
		return nil, err
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		EntityType: "category",
		EntityID:   &catID,
		Action:     audit.Action("category.updated"),
		Payload:    map[string]interface{}{"name": updated.Name},
	})

	s.fireEvent(ctx, EventCategoryUpdated, map[string]interface{}{
		"category_id": catID.String(),
		"site_id":     siteID.String(),
		"name":        updated.Name,
	})

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, siteID, catID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	existing, err := s.GetByID(ctx, siteID, catID)
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx,
		`UPDATE categories SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		catID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		EntityType: "category",
		EntityID:   &catID,
		Action:     audit.Action("category.deleted"),
		Payload:    map[string]interface{}{"name": existing.Name, "slug": existing.Slug},
	})

	s.fireEvent(ctx, EventCategoryDeleted, map[string]interface{}{
		"category_id": catID.String(),
		"site_id":     siteID.String(),
		"name":        existing.Name,
	})

	return nil
}

func (s *Service) isDescendant(ctx context.Context, p database.Pool, siteID, catID, potentialParent uuid.UUID) bool {
	current := potentialParent
	visited := make(map[uuid.UUID]bool)
	for {
		if current == catID {
			return true
		}
		if visited[current] {
			return false
		}
		visited[current] = true

		var parentID *uuid.UUID
		err := p.QueryRow(ctx,
			`SELECT parent_id FROM categories WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
			current, siteID,
		).Scan(&parentID)
		if err != nil || parentID == nil {
			return false
		}
		current = *parentID
	}
}
