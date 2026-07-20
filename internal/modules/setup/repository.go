package setup

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"nexora/internal/pkg/database"
)

type Repository struct {
	db *database.Database
}

func NewRepository(db *database.Database) *Repository {
	return &Repository{db: db}
}

func (r *Repository) pool() (database.Pool, error) {
	if r.db == nil || r.db.Pool == nil {
		return nil, ErrDatabaseNotAvail
	}
	return r.db.Pool, nil
}

func (r *Repository) GetInstallation(ctx context.Context) (*SystemInstallation, error) {
	p, err := r.pool()
	if err != nil {
		return nil, err
	}

	row := p.QueryRow(ctx,
		`SELECT id, installed, installed_at, cms_name, admin_name, admin_email,
		        default_site, version, locale, timezone, created_at, updated_at
		 FROM system_installation ORDER BY created_at DESC LIMIT 1`,
	)

	var inst SystemInstallation
	err = row.Scan(
		&inst.ID, &inst.Installed, &inst.InstalledAt,
		&inst.CmsName, &inst.AdminName, &inst.AdminEmail,
		&inst.DefaultSite, &inst.Version, &inst.Locale, &inst.Timezone,
		&inst.CreatedAt, &inst.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get installation: %w", err)
	}

	return &inst, nil
}

func (r *Repository) CreateInstallation(ctx context.Context, inst *SystemInstallation) error {
	p, err := r.pool()
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = p.Exec(ctx,
		`INSERT INTO system_installation (id, installed, installed_at, cms_name, admin_name,
		 admin_email, default_site, version, locale, timezone, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)`,
		inst.ID, true, now, inst.CmsName, inst.AdminName, inst.AdminEmail,
		inst.DefaultSite, inst.Version, inst.Locale, inst.Timezone, now,
	)
	if err != nil {
		return fmt.Errorf("failed to create installation: %w", err)
	}

	inst.Installed = true
	inst.InstalledAt = &now

	return nil
}

func (r *Repository) CreateAdminUser(ctx context.Context, id uuid.UUID, name, email, passwordHash string) error {
	p, err := r.pool()
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, name, role, metadata)
		 VALUES ($1, $2, $3, $4, 'super_admin', '{}')`,
		id, email, passwordHash, name,
	)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	return nil
}

func (r *Repository) CreateSite(ctx context.Context, id, ownerID uuid.UUID, name, slug, description, locale, timezone, url string) error {
	p, err := r.pool()
	if err != nil {
		return err
	}

	settings := "{}"
	if url != "" {
		settings = fmt.Sprintf(`{"url": "%s"}`, url)
	}

	_, err = p.Exec(ctx,
		`INSERT INTO sites (id, name, slug, description, status, owner_id, settings, locale, timezone)
		 VALUES ($1, $2, $3, $4, 'active', $5, $6::jsonb, $7, $8)`,
		id, name, slug, description, ownerID, settings, locale, timezone,
	)
	if err != nil {
		return fmt.Errorf("failed to create site: %w", err)
	}

	return nil
}

func (r *Repository) CreateSiteUser(ctx context.Context, userID, siteID uuid.UUID, role string) error {
	p, err := r.pool()
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx,
		`INSERT INTO site_users (user_id, site_id, role) VALUES ($1, $2, $3)`,
		userID, siteID, role,
	)
	if err != nil {
		return fmt.Errorf("failed to create site user: %w", err)
	}

	return nil
}

func (r *Repository) IsEmailTaken(ctx context.Context, email string) (bool, error) {
	p, err := r.pool()
	if err != nil {
		return false, err
	}

	var exists bool
	err = p.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email: %w", err)
	}

	return exists, nil
}
