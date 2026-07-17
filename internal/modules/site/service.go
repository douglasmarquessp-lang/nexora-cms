package site

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"nexora/internal/kernel"
	"nexora/internal/pkg/audit"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

var (
	ErrSiteNotFound          = errors.New("site not found")
	ErrSiteSlugAlreadyExists = errors.New("site slug already exists")
	ErrDomainAlreadyExists   = errors.New("domain already exists")
	ErrDomainNotFound        = errors.New("domain not found")
	ErrInvalidDomain         = errors.New("invalid domain format")
	ErrSiteNotAvailable      = errors.New("site not available")
	ErrDatabaseNotAvailable  = errors.New("database not available")
	ErrGlobalSettingNotFound = errors.New("global setting not found")
	ErrSiteSettingNotFound   = errors.New("site setting not found")
	ErrInvalidSettingType    = errors.New("invalid setting type")
)

const (
	EventSiteCreated   kernel.EventType = "site.created"
	EventSiteUpdated   kernel.EventType = "site.updated"
	EventSiteDeleted   kernel.EventType = "site.deleted"
	EventDomainAdded   kernel.EventType = "site.domain.added"
	EventDomainRemoved kernel.EventType = "site.domain.removed"
)

var domainRegex = regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

var validSettingTypes = map[string]bool{
	"string": true, "number": true, "boolean": true, "json": true, "array": true,
}

type Service struct {
	log       *logger.Logger
	db        *database.Database
	cache     *cache.Cache
	eventBus  *kernel.EventBus
	auditLog  *audit.Logger
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

func (s *Service) SetRLSContext(ctx context.Context, userID uuid.UUID, role string, siteID uuid.UUID) context.Context {
	ctx = context.WithValue(ctx, "app.current_user_id", userID.String())
	ctx = context.WithValue(ctx, "app.current_user_role", role)
	ctx = context.WithValue(ctx, "app.current_site_id", siteID.String())
	return ctx
}

func (s *Service) pool() (database.Pool, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, ErrDatabaseNotAvailable
	}
	return s.db.Pool, nil
}

func (s *Service) cacheGet(ctx context.Context, key string, dest interface{}) bool {
	if s.cache == nil {
		return false
	}
	val, ok := s.cache.Get(ctx, key)
	if !ok || val == nil {
		return false
	}
	data, err := json.Marshal(val)
	if err != nil {
		return false
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return false
	}
	return true
}

func (s *Service) cacheSet(ctx context.Context, key string, value interface{}) {
	if s.cache == nil {
		return
	}
	_ = s.cache.SetJSON(ctx, key, value, 5*time.Minute)
}

func (s *Service) cacheDel(ctx context.Context, keys ...string) {
	if s.cache == nil {
		return
	}
	for _, key := range keys {
		_ = s.cache.Delete(ctx, key)
	}
}

func (s *Service) CreateSite(ctx context.Context, userID uuid.UUID, req CreateSiteRequest) (*Site, error) {
	slug := strings.TrimSpace(strings.ToLower(req.Slug))
	if slug == "" {
		return nil, errors.New("slug is required")
	}
	if req.Name == "" {
		return nil, errors.New("name is required")
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var exists int
	err = p.QueryRow(ctx, `SELECT COUNT(*) FROM sites WHERE slug = $1 AND deleted_at IS NULL`, slug).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check slug uniqueness: %w", err)
	}
	if exists > 0 {
		return nil, ErrSiteSlugAlreadyExists
	}

	siteID := uuid.New()
	now := time.Now()

	settings := "{}"
	if req.Settings != nil {
		b, _ := json.Marshal(req.Settings)
		settings = string(b)
	}
	featureFlags := "{}"
	if req.FeatureFlags != nil {
		b, _ := json.Marshal(req.FeatureFlags)
		featureFlags = string(b)
	}

	locale := req.Locale
	if locale == "" {
		locale = "en-US"
	}
	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	_, err = p.Exec(ctx,
		`INSERT INTO sites (id, name, slug, description, status, owner_id, settings, feature_flags, theme, locale, timezone, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9, $10, $11, $12, $13)`,
		siteID, req.Name, slug, req.Description, SiteStatusActive, userID,
		settings, featureFlags, req.Theme, locale, timezone, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create site: %w", err)
	}

	site := &Site{
		ID:           siteID,
		Name:         req.Name,
		Slug:         slug,
		Description:  req.Description,
		Status:       SiteStatusActive,
		OwnerID:      userID,
		Settings:     req.Settings,
		FeatureFlags: req.FeatureFlags,
		Theme:        req.Theme,
		Locale:       locale,
		Timezone:     timezone,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if site.Settings == nil {
		site.Settings = make(map[string]interface{})
	}
	if site.FeatureFlags == nil {
		site.FeatureFlags = make(map[string]interface{})
	}

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("site.created"),
		EntityType: "site",
		EntityID:   &siteID,
		Payload:    map[string]interface{}{"slug": slug, "name": req.Name},
	})

	s.fireEvent(ctx, EventSiteCreated, map[string]interface{}{
		"site_id": siteID.String(),
		"slug":    slug,
		"name":    req.Name,
	})

	return site, nil
}

func (s *Service) GetSite(ctx context.Context, siteID uuid.UUID) (*Site, error) {
	var site Site
	cacheKey := "site:" + siteID.String()
	if s.cacheGet(ctx, cacheKey, &site) {
		return &site, nil
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var settingsJSON, featureFlagsJSON []byte
	var description, theme, locale, timezone *string
	var deletedAt *time.Time

	err = p.QueryRow(ctx,
		`SELECT id, name, slug, COALESCE(description, ''), status, owner_id,
		        COALESCE(settings::text, '{}'), COALESCE(feature_flags::text, '{}'),
		        COALESCE(theme, ''), COALESCE(locale, ''), COALESCE(timezone, ''),
		        created_at, updated_at, deleted_at
		 FROM sites WHERE id = $1 AND deleted_at IS NULL`,
		siteID,
	).Scan(
		&site.ID, &site.Name, &site.Slug, &description, &site.Status, &site.OwnerID,
		&settingsJSON, &featureFlagsJSON,
		&theme, &locale, &timezone,
		&site.CreatedAt, &site.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSiteNotFound
		}
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	if description != nil {
		site.Description = *description
	}
	if theme != nil {
		site.Theme = *theme
	}
	if locale != nil {
		site.Locale = *locale
	}
	if timezone != nil {
		site.Timezone = *timezone
	}
	site.DeletedAt = deletedAt

	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &site.Settings); err != nil {
			s.log.Warn("failed to unmarshal site settings", "error", err, "site_id", siteID)
		}
	}
	if len(featureFlagsJSON) > 0 {
		if err := json.Unmarshal(featureFlagsJSON, &site.FeatureFlags); err != nil {
			s.log.Warn("failed to unmarshal site feature flags", "error", err, "site_id", siteID)
		}
	}
	if site.Settings == nil {
		site.Settings = make(map[string]interface{})
	}
	if site.FeatureFlags == nil {
		site.FeatureFlags = make(map[string]interface{})
	}

	s.cacheSet(ctx, cacheKey, site)
	return &site, nil
}

func (s *Service) GetSiteBySlug(ctx context.Context, slug string) (*Site, error) {
	var site Site
	cacheKey := "site:slug:" + slug
	if s.cacheGet(ctx, cacheKey, &site) {
		return &site, nil
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var settingsJSON, featureFlagsJSON []byte
	var description, theme, locale, timezone *string
	var deletedAt *time.Time

	err = p.QueryRow(ctx,
		`SELECT id, name, slug, COALESCE(description, ''), status, owner_id,
		        COALESCE(settings::text, '{}'), COALESCE(feature_flags::text, '{}'),
		        COALESCE(theme, ''), COALESCE(locale, ''), COALESCE(timezone, ''),
		        created_at, updated_at, deleted_at
		 FROM sites WHERE slug = $1 AND deleted_at IS NULL`,
		slug,
	).Scan(
		&site.ID, &site.Name, &site.Slug, &description, &site.Status, &site.OwnerID,
		&settingsJSON, &featureFlagsJSON,
		&theme, &locale, &timezone,
		&site.CreatedAt, &site.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSiteNotFound
		}
		return nil, fmt.Errorf("failed to get site by slug: %w", err)
	}

	if description != nil {
		site.Description = *description
	}
	if theme != nil {
		site.Theme = *theme
	}
	if locale != nil {
		site.Locale = *locale
	}
	if timezone != nil {
		site.Timezone = *timezone
	}
	site.DeletedAt = deletedAt

	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &site.Settings); err != nil {
			s.log.Warn("failed to unmarshal site settings", "error", err, "slug", slug)
		}
	}
	if len(featureFlagsJSON) > 0 {
		if err := json.Unmarshal(featureFlagsJSON, &site.FeatureFlags); err != nil {
			s.log.Warn("failed to unmarshal site feature flags", "error", err, "slug", slug)
		}
	}
	if site.Settings == nil {
		site.Settings = make(map[string]interface{})
	}
	if site.FeatureFlags == nil {
		site.FeatureFlags = make(map[string]interface{})
	}

	s.cacheSet(ctx, cacheKey, site)
	return &site, nil
}

func (s *Service) GetSiteByDomain(ctx context.Context, domain string) (*Site, error) {
	var site Site
	cacheKey := "site:domain:" + domain
	if s.cacheGet(ctx, cacheKey, &site) {
		return &site, nil
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var settingsJSON, featureFlagsJSON []byte
	var description, theme, locale, timezone *string
	var deletedAt *time.Time

	err = p.QueryRow(ctx,
		`SELECT s.id, s.name, s.slug, COALESCE(s.description, ''), s.status, s.owner_id,
		        COALESCE(s.settings::text, '{}'), COALESCE(s.feature_flags::text, '{}'),
		        COALESCE(s.theme, ''), COALESCE(s.locale, ''), COALESCE(s.timezone, ''),
		        s.created_at, s.updated_at, s.deleted_at
		 FROM sites s
		 INNER JOIN site_domains sd ON sd.site_id = s.id
		 WHERE sd.domain = $1 AND s.deleted_at IS NULL AND sd.verified = true
		 ORDER BY sd.is_primary DESC
		 LIMIT 1`,
		domain,
	).Scan(
		&site.ID, &site.Name, &site.Slug, &description, &site.Status, &site.OwnerID,
		&settingsJSON, &featureFlagsJSON,
		&theme, &locale, &timezone,
		&site.CreatedAt, &site.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSiteNotFound
		}
		return nil, fmt.Errorf("failed to get site by domain: %w", err)
	}

	if description != nil {
		site.Description = *description
	}
	if theme != nil {
		site.Theme = *theme
	}
	if locale != nil {
		site.Locale = *locale
	}
	if timezone != nil {
		site.Timezone = *timezone
	}
	site.DeletedAt = deletedAt

	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &site.Settings); err != nil {
			s.log.Warn("failed to unmarshal site settings", "error", err, "domain", domain)
		}
	}
	if len(featureFlagsJSON) > 0 {
		if err := json.Unmarshal(featureFlagsJSON, &site.FeatureFlags); err != nil {
			s.log.Warn("failed to unmarshal site feature flags", "error", err, "domain", domain)
		}
	}
	if site.Settings == nil {
		site.Settings = make(map[string]interface{})
	}
	if site.FeatureFlags == nil {
		site.FeatureFlags = make(map[string]interface{})
	}

	s.cacheSet(ctx, cacheKey, site)
	return &site, nil
}

func (s *Service) ListSites(ctx context.Context, userID uuid.UUID, page, perPage int) (*SiteListResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * perPage

	rows, err := p.Query(ctx,
		`SELECT id, name, slug, COALESCE(description, ''), status, owner_id,
		        COALESCE(settings::text, '{}'), COALESCE(feature_flags::text, '{}'),
		        COALESCE(theme, ''), COALESCE(locale, ''), COALESCE(timezone, ''),
		        created_at, updated_at
		 FROM sites
		 WHERE deleted_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`,
		perPage, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list sites: %w", err)
	}
	defer rows.Close()

	var total int
	var sites []Site

	for rows.Next() {
		var site Site
		var settingsJSON, featureFlagsJSON []byte
		var description, theme, locale, timezone *string

		err := rows.Scan(
			&site.ID, &site.Name, &site.Slug, &description, &site.Status, &site.OwnerID,
			&settingsJSON, &featureFlagsJSON,
			&theme, &locale, &timezone,
			&site.CreatedAt, &site.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan site: %w", err)
		}

		if description != nil {
			site.Description = *description
		}
		if theme != nil {
			site.Theme = *theme
		}
		if locale != nil {
			site.Locale = *locale
		}
		if timezone != nil {
			site.Timezone = *timezone
		}

		if len(settingsJSON) > 0 {
			if err := json.Unmarshal(settingsJSON, &site.Settings); err != nil {
				s.log.Warn("failed to unmarshal site settings", "error", err, "site_id", site.ID)
			}
		}
		if len(featureFlagsJSON) > 0 {
			if err := json.Unmarshal(featureFlagsJSON, &site.FeatureFlags); err != nil {
				s.log.Warn("failed to unmarshal site feature flags", "error", err, "site_id", site.ID)
			}
		}
		if site.Settings == nil {
			site.Settings = make(map[string]interface{})
		}
		if site.FeatureFlags == nil {
			site.FeatureFlags = make(map[string]interface{})
		}

		sites = append(sites, site)
	}

	if len(sites) > 0 {
		if err := p.QueryRow(ctx, `SELECT COUNT(*) FROM sites WHERE deleted_at IS NULL`).Scan(&total); err != nil {
			return nil, fmt.Errorf("failed to count sites: %w", err)
		}
	} else {
		total = 0
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	return &SiteListResponse{
		Sites:   sites,
		Total:   total,
		Page:    page,
		PerPage: perPage,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) UpdateSite(ctx context.Context, siteID uuid.UUID, req UpdateSiteRequest) (*Site, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	site, err := s.GetSite(ctx, siteID)
	if err != nil {
		return nil, err
	}

	var setClauses []string
	args := []interface{}{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
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
	}
	if req.Theme != nil {
		setClauses = append(setClauses, fmt.Sprintf("theme = $%d", argIdx))
		args = append(args, *req.Theme)
		argIdx++
	}
	if req.Locale != nil {
		setClauses = append(setClauses, fmt.Sprintf("locale = $%d", argIdx))
		args = append(args, *req.Locale)
		argIdx++
	}
	if req.Timezone != nil {
		setClauses = append(setClauses, fmt.Sprintf("timezone = $%d", argIdx))
		args = append(args, *req.Timezone)
		argIdx++
	}
	if req.Settings != nil {
		b, _ := json.Marshal(*req.Settings)
		setClauses = append(setClauses, fmt.Sprintf("settings = $%d::jsonb", argIdx))
		args = append(args, string(b))
		argIdx++
	}
	if req.FeatureFlags != nil {
		b, _ := json.Marshal(*req.FeatureFlags)
		setClauses = append(setClauses, fmt.Sprintf("feature_flags = $%d::jsonb", argIdx))
		args = append(args, string(b))
		argIdx++
	}

	if len(setClauses) == 0 {
		return site, nil
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = NOW()"))

	query := fmt.Sprintf(
		`UPDATE sites SET %s WHERE id = $%d AND deleted_at IS NULL`,
		strings.Join(setClauses, ", "),
		argIdx,
	)
	args = append(args, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update site: %w", err)
	}

	updatedSite, err := s.GetSite(ctx, siteID)
	if err != nil {
		return nil, err
	}

	s.auditLog.Log(ctx, audit.Entry{
		EntityType: "site",
		EntityID:   &siteID,
		Action:     audit.Action("site.updated"),
		Payload:    map[string]interface{}{"updated_fields": len(setClauses) - 1},
	})

	s.fireEvent(ctx, EventSiteUpdated, map[string]interface{}{
		"site_id": siteID.String(),
	})

	s.cacheDel(ctx, "site:"+siteID.String(), "site:slug:"+site.Slug)

	return updatedSite, nil
}

func (s *Service) DeleteSite(ctx context.Context, siteID uuid.UUID, userID uuid.UUID) error {
	site, err := s.GetSite(ctx, siteID)
	if err != nil {
		return err
	}

	if site.OwnerID != userID {
		var role string
		if s.db != nil && s.db.Pool != nil {
			err := s.db.Pool.QueryRow(ctx, `SELECT role FROM users WHERE id = $1`, userID).Scan(&role)
			if err != nil || role != "superadmin" {
				return ErrSiteNotAvailable
			}
		} else {
			return ErrSiteNotAvailable
		}
	}

	pgxPool, ok := s.db.Pool.(*pgxpool.Pool)
	if !ok {
		return fmt.Errorf("database pool does not support transactions")
	}

	tx, err := pgxPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	cleanup := []string{
		`DELETE FROM publication_queue WHERE site_id = $1`,
		`DELETE FROM autocontent_results WHERE autocontent_job_id IN (SELECT id FROM autocontent_jobs WHERE site_id = $1)`,
		`DELETE FROM autocontent_steps WHERE autocontent_job_id IN (SELECT id FROM autocontent_jobs WHERE site_id = $1)`,
		`DELETE FROM autocontent_jobs WHERE site_id = $1`,
		`DELETE FROM workflow_templates WHERE site_id = $1`,
		`DELETE FROM generation_quality_gates WHERE generation_job_id IN (SELECT id FROM generation_jobs WHERE site_id = $1)`,
		`DELETE FROM generation_pipeline_logs WHERE generation_job_id IN (SELECT id FROM generation_jobs WHERE site_id = $1)`,
		`DELETE FROM generation_pipeline WHERE generation_job_id IN (SELECT id FROM generation_jobs WHERE site_id = $1)`,
		`DELETE FROM generation_stats WHERE site_id = $1`,
		`DELETE FROM generation_jobs WHERE site_id = $1`,
		`DELETE FROM editorial_prompt_data WHERE site_id = $1`,
		`DELETE FROM editorial_translations WHERE site_id = $1`,
		`DELETE FROM editorial_quality_scores WHERE site_id = $1`,
		`DELETE FROM editorial_seo_data WHERE site_id = $1`,
		`DELETE FROM editorial_style_rules WHERE site_id = $1`,
		`DELETE FROM pipeline_stages WHERE pipeline_id IN (SELECT id FROM editorial_pipelines WHERE site_id = $1)`,
		`DELETE FROM editorial_pipelines WHERE site_id = $1`,
		`DELETE FROM article_versions WHERE article_job_id IN (SELECT id FROM article_jobs WHERE site_id = $1)`,
		`DELETE FROM article_sections WHERE article_job_id IN (SELECT id FROM article_jobs WHERE site_id = $1)`,
		`DELETE FROM article_outlines WHERE article_job_id IN (SELECT id FROM article_jobs WHERE site_id = $1)`,
		`DELETE FROM article_jobs WHERE site_id = $1`,
		`DELETE FROM writing_styles WHERE site_id = $1`,
		`DELETE FROM research_briefings WHERE research_job_id IN (SELECT id FROM research_jobs WHERE site_id = $1)`,
		`DELETE FROM research_entities WHERE research_job_id IN (SELECT id FROM research_jobs WHERE site_id = $1)`,
		`DELETE FROM research_sources WHERE research_job_id IN (SELECT id FROM research_jobs WHERE site_id = $1)`,
		`DELETE FROM research_jobs WHERE site_id = $1`,
		`DELETE FROM editorial_widgets WHERE site_id = $1`,
		`DELETE FROM editorial_calendar_events WHERE site_id = $1`,
		`DELETE FROM approval_requests WHERE site_id = $1`,
		`DELETE FROM post_revisions WHERE site_id = $1`,
		`DELETE FROM editorial_tasks WHERE site_id = $1`,
		`DELETE FROM post_autosaves WHERE site_id = $1`,
		`DELETE FROM post_assets WHERE post_id IN (SELECT id FROM posts WHERE site_id = $1)`,
		`DELETE FROM post_tags WHERE post_id IN (SELECT id FROM posts WHERE site_id = $1)`,
		`DELETE FROM post_categories WHERE post_id IN (SELECT id FROM posts WHERE site_id = $1)`,
		`DELETE FROM posts WHERE site_id = $1`,
		`DELETE FROM categories WHERE site_id = $1`,
		`DELETE FROM tags WHERE site_id = $1`,
		`DELETE FROM assets WHERE site_id = $1`,
		`DELETE FROM media_variants WHERE media_id IN (SELECT id FROM media WHERE site_id = $1)`,
		`DELETE FROM media WHERE site_id = $1`,
		`DELETE FROM folders WHERE site_id = $1`,
		`DELETE FROM seo_scores WHERE site_id = $1`,
		`DELETE FROM seo_metadata WHERE site_id = $1`,
		`DELETE FROM seo_internal_links WHERE site_id = $1`,
		`DELETE FROM seo_audits WHERE site_id = $1`,
		`DELETE FROM seo_clusters WHERE site_id = $1`,
		`DELETE FROM seo_keywords WHERE site_id = $1`,
		`DELETE FROM seo_projects WHERE site_id = $1`,
		`DELETE FROM site_domains WHERE site_id = $1`,
		`DELETE FROM site_settings WHERE site_id = $1`,
	}

	for _, q := range cleanup {
		if _, err := tx.Exec(ctx, q, siteID); err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}
	}

	_, err = tx.Exec(ctx,
		`UPDATE sites SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete site: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		EntityType: "site",
		EntityID:   &siteID,
		Action:     audit.Action("site.deleted"),
		Payload:    map[string]interface{}{"slug": site.Slug},
	})

	s.fireEvent(ctx, EventSiteDeleted, map[string]interface{}{
		"site_id": siteID.String(),
		"slug":    site.Slug,
	})

	s.cacheDel(ctx, "site:"+siteID.String(), "site:slug:"+site.Slug)

	return nil
}

func (s *Service) AddDomain(ctx context.Context, siteID uuid.UUID, req AddDomainRequest) (*SiteDomain, error) {
	domain := strings.TrimSpace(strings.ToLower(req.Domain))
	if domain == "" || !domainRegex.MatchString(domain) {
		return nil, ErrInvalidDomain
	}

	u, err := url.Parse("https://" + domain)
	if err != nil || u.Host != domain {
		return nil, ErrInvalidDomain
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	_, err = s.GetSite(ctx, siteID)
	if err != nil {
		return nil, err
	}

	var exists int
	err = p.QueryRow(ctx, `SELECT COUNT(*) FROM site_domains WHERE domain = $1`, domain).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check domain uniqueness: %w", err)
	}
	if exists > 0 {
		return nil, ErrDomainAlreadyExists
	}

	domainID := uuid.New()
	now := time.Now()

	_, err = p.Exec(ctx,
		`INSERT INTO site_domains (id, site_id, domain, is_primary, verified, ssl_enabled, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, false, false, $5, $6)`,
		domainID, siteID, domain, req.IsPrimary, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add domain: %w", err)
	}

	if req.IsPrimary {
		_, _ = p.Exec(ctx,
			`UPDATE site_domains SET is_primary = (id = $1) WHERE site_id = $2`,
			domainID, siteID,
		)
	}

	sd := &SiteDomain{
		ID:        domainID,
		SiteID:    siteID,
		Domain:    domain,
		IsPrimary: req.IsPrimary,
		Verified:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		EntityType: "site_domain",
		EntityID:   &domainID,
		Action:     audit.Action("site.domain.added"),
		Payload:    map[string]interface{}{"domain": domain},
	})

	s.fireEvent(ctx, EventDomainAdded, map[string]interface{}{
		"site_id": siteID.String(),
		"domain":  domain,
	})

	s.cacheDel(ctx, "site:domain:"+domain)

	return sd, nil
}

func (s *Service) RemoveDomain(ctx context.Context, domainID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	var domain, siteID string
	err = p.QueryRow(ctx,
		`SELECT domain, site_id FROM site_domains WHERE id = $1`, domainID,
	).Scan(&domain, &siteID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrDomainNotFound
		}
		return fmt.Errorf("failed to find domain: %w", err)
	}

	_, err = p.Exec(ctx, `DELETE FROM site_domains WHERE id = $1`, domainID)
	if err != nil {
		return fmt.Errorf("failed to remove domain: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		EntityType: "site_domain",
		EntityID:   &domainID,
		Action:     audit.Action("site.domain.removed"),
		Payload:    map[string]interface{}{"domain": domain},
	})

	s.fireEvent(ctx, EventDomainRemoved, map[string]interface{}{
		"domain": domain,
	})

	s.cacheDel(ctx, "site:domain:"+domain)

	return nil
}

func (s *Service) ListDomains(ctx context.Context, siteID uuid.UUID) ([]SiteDomain, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, site_id, domain, is_primary, verified, ssl_enabled, created_at, updated_at
		 FROM site_domains WHERE site_id = $1
		 ORDER BY is_primary DESC, created_at ASC`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list domains: %w", err)
	}
	defer rows.Close()

	var domains []SiteDomain
	for rows.Next() {
		var d SiteDomain
		err := rows.Scan(&d.ID, &d.SiteID, &d.Domain, &d.IsPrimary, &d.Verified, &d.SSLEnabled, &d.CreatedAt, &d.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan domain: %w", err)
		}
		domains = append(domains, d)
	}

	return domains, nil
}

func (s *Service) SetPrimaryDomain(ctx context.Context, siteID, domainID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	var count int
	err = p.QueryRow(ctx, `SELECT COUNT(*) FROM site_domains WHERE id = $1 AND site_id = $2`, domainID, siteID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify domain: %w", err)
	}
	if count == 0 {
		return ErrDomainNotFound
	}

	_, err = p.Exec(ctx,
		`UPDATE site_domains SET is_primary = (id = $1), updated_at = NOW() WHERE site_id = $2`,
		domainID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to set primary domain: %w", err)
	}

	return nil
}

func (s *Service) GetGlobalSetting(ctx context.Context, key string) (*GlobalSetting, error) {
	var gs GlobalSetting
	cacheKey := "setting:global:" + key
	if s.cacheGet(ctx, cacheKey, &gs) {
		return &gs, nil
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var valueJSON []byte
	var description *string

	err = p.QueryRow(ctx,
		`SELECT id, key, value::text, type, COALESCE(description, ''), created_at, updated_at
		 FROM global_settings WHERE key = $1`,
		key,
	).Scan(&gs.ID, &gs.Key, &valueJSON, &gs.Type, &description, &gs.CreatedAt, &gs.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGlobalSettingNotFound
		}
		return nil, fmt.Errorf("failed to get global setting: %w", err)
	}

	if description != nil {
		gs.Description = *description
	}

	if len(valueJSON) > 0 {
		if err := json.Unmarshal(valueJSON, &gs.Value); err != nil {
			s.log.Warn("failed to unmarshal global setting value", "error", err, "key", key)
		}
	}

	s.cacheSet(ctx, cacheKey, gs)
	return &gs, nil
}

func (s *Service) SetGlobalSetting(ctx context.Context, req UpdateGlobalSettingRequest) (*GlobalSetting, error) {
	if req.Type != "" && !validSettingTypes[req.Type] {
		return nil, ErrInvalidSettingType
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	settingType := req.Type
	if settingType == "" {
		settingType = s.inferType(req.Value)
	}

	valueBytes, err := json.Marshal(req.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal value: %w", err)
	}
	valueStr := string(valueBytes)

	var gs GlobalSetting
	err = p.QueryRow(ctx,
		`INSERT INTO global_settings (key, value, type, description, created_at, updated_at)
		 VALUES ($1, $2::jsonb, $3, $4, NOW(), NOW())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, type = EXCLUDED.type, description = EXCLUDED.description, updated_at = NOW()
		 RETURNING id, key, value::text, type, COALESCE(description, ''), created_at, updated_at`,
		"", valueStr, settingType, req.Description,
	).Scan(&gs.ID, &gs.Key, &valueBytes, &gs.Type, &gs.Description, &gs.CreatedAt, &gs.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert global setting: %w", err)
	}

	if len(valueBytes) > 0 {
		if err := json.Unmarshal(valueBytes, &gs.Value); err != nil {
			s.log.Warn("failed to unmarshal global setting value", "error", err, "key", gs.Key)
		}
	}

	s.cacheDel(ctx, "setting:global:"+gs.Key, "settings:global")

	return &gs, nil
}

func (s *Service) ListGlobalSettings(ctx context.Context) ([]GlobalSetting, error) {
	var settings []GlobalSetting
	cacheKey := "settings:global"
	if s.cacheGet(ctx, cacheKey, &settings) {
		return settings, nil
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, key, value::text, type, COALESCE(description, ''), created_at, updated_at
		 FROM global_settings ORDER BY key ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list global settings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var gs GlobalSetting
		var valueJSON []byte
		var description *string

		err := rows.Scan(&gs.ID, &gs.Key, &valueJSON, &gs.Type, &description, &gs.CreatedAt, &gs.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan global setting: %w", err)
		}

		if description != nil {
			gs.Description = *description
		}
		if len(valueJSON) > 0 {
			if err := json.Unmarshal(valueJSON, &gs.Value); err != nil {
				s.log.Warn("failed to unmarshal global setting value", "error", err, "key", gs.Key)
			}
		}

		settings = append(settings, gs)
	}

	s.cacheSet(ctx, cacheKey, settings)
	return settings, nil
}

func (s *Service) GetSiteSetting(ctx context.Context, siteID uuid.UUID, key string) (*SiteSetting, error) {
	var ss SiteSetting
	cacheKey := "setting:site:" + siteID.String() + ":" + key
	if s.cacheGet(ctx, cacheKey, &ss) {
		return &ss, nil
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var valueJSON []byte

	err = p.QueryRow(ctx,
		`SELECT id, site_id, key, value::text, created_at, updated_at
		 FROM site_settings WHERE site_id = $1 AND key = $2`,
		siteID, key,
	).Scan(&ss.ID, &ss.SiteID, &ss.Key, &valueJSON, &ss.CreatedAt, &ss.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSiteSettingNotFound
		}
		return nil, fmt.Errorf("failed to get site setting: %w", err)
	}

	if len(valueJSON) > 0 {
		if err := json.Unmarshal(valueJSON, &ss.Value); err != nil {
			s.log.Warn("failed to unmarshal site setting value", "error", err, "key", key, "site_id", siteID)
		}
	}

	s.cacheSet(ctx, cacheKey, ss)
	return &ss, nil
}

func (s *Service) SetSiteSetting(ctx context.Context, siteID uuid.UUID, req SetSiteSettingRequest) (*SiteSetting, error) {
	if req.Key == "" {
		return nil, errors.New("key is required")
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	valueBytes, err := json.Marshal(req.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal value: %w", err)
	}
	valueStr := string(valueBytes)

	var ss SiteSetting
	var valueJSON []byte

	err = p.QueryRow(ctx,
		`INSERT INTO site_settings (site_id, key, value, created_at, updated_at)
		 VALUES ($1, $2, $3::jsonb, NOW(), NOW())
		 ON CONFLICT (site_id, key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
		 RETURNING id, site_id, key, value::text, created_at, updated_at`,
		siteID, req.Key, valueStr,
	).Scan(&ss.ID, &ss.SiteID, &ss.Key, &valueJSON, &ss.CreatedAt, &ss.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert site setting: %w", err)
	}

	if len(valueJSON) > 0 {
		if err := json.Unmarshal(valueJSON, &ss.Value); err != nil {
			s.log.Warn("failed to unmarshal site setting value", "error", err, "key", req.Key, "site_id", siteID)
		}
	}

	s.cacheDel(ctx, "setting:site:"+siteID.String()+":"+req.Key, "settings:site:"+siteID.String())

	return &ss, nil
}

func (s *Service) ListSiteSettings(ctx context.Context, siteID uuid.UUID) ([]SiteSetting, error) {
	var settings []SiteSetting
	cacheKey := "settings:site:" + siteID.String()
	if s.cacheGet(ctx, cacheKey, &settings) {
		return settings, nil
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, site_id, key, value::text, created_at, updated_at
		 FROM site_settings WHERE site_id = $1 ORDER BY key ASC`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list site settings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ss SiteSetting
		var valueJSON []byte

		err := rows.Scan(&ss.ID, &ss.SiteID, &ss.Key, &valueJSON, &ss.CreatedAt, &ss.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan site setting: %w", err)
		}

		if len(valueJSON) > 0 {
			if err := json.Unmarshal(valueJSON, &ss.Value); err != nil {
				s.log.Warn("failed to unmarshal site setting value", "error", err, "key", ss.Key, "site_id", siteID)
			}
		}

		settings = append(settings, ss)
	}

	s.cacheSet(ctx, cacheKey, settings)
	return settings, nil
}

func (s *Service) inferType(val interface{}) string {
	if val == nil {
		return "json"
	}
	switch val.(type) {
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		return "array"
	default:
		return "json"
	}
}


