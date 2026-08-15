// Package sitedomain resolves the public site context (domain and primary
// language) for a given site ID, so that publishing and public SEO URLs are
// derived from real site configuration instead of global fallbacks.
//
// Sources of truth (database only, never hardcoded per site):
//
//   - Domain:  site_domains, preferring verified primary, then any verified.
//   - Language: sites.locale, normalized (pt-BR -> pt, en-US -> en, ...).
//
// The publisher keeps its existing sitelang pin (override) as the top layer;
// this package only supplies the site-level defaults underneath it.
package sitedomain

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"nexora/internal/pkg/database"
)

// SiteContext is the resolved public context of a site.
type SiteContext struct {
	// Domain is the site's public base URL (scheme + host, no trailing
	// slash). Empty when the site has no resolvable verified domain.
	Domain string
	// PrimaryLanguage is the normalized primary language of the site
	// (e.g. "en", "pt"). Empty when the site has no locale configured.
	PrimaryLanguage string
}

// Resolver resolves the public context of a site by ID. Implementations must
// be safe for concurrent use.
type Resolver interface {
	Resolve(ctx context.Context, siteID uuid.UUID) (SiteContext, error)
}

// Store abstracts the two database lookups so the resolver is testable
// without a live database.
type Store interface {
	// SiteLocale returns the site's locale column value ("" when unset).
	SiteLocale(ctx context.Context, siteID uuid.UUID) (string, error)
	// PrimaryDomain returns the site's best verified domain (primary
	// preferred, then any verified). Returns pgx.ErrNoRows when no
	// verified domain exists.
	PrimaryDomain(ctx context.Context, siteID uuid.UUID) (string, error)
}

// DBResolver resolves site context from a Store.
type DBResolver struct {
	store Store
}

// New returns a DBResolver backed by the given store.
func New(store Store) *DBResolver {
	return &DBResolver{store: store}
}

// Resolve loads the site's locale and verified primary domain.
//
//   - Domain errors (including "no verified domain") never fail resolution:
//     an empty Domain signals the caller to use its defensive fallback.
//   - Locale errors (site missing / DB failure) are returned as errors, so a
//     real configuration problem is not silently hidden by a fallback.
func (r *DBResolver) Resolve(ctx context.Context, siteID uuid.UUID) (SiteContext, error) {
	locale, err := r.store.SiteLocale(ctx, siteID)
	if err != nil {
		return SiteContext{}, err
	}

	domain, err := r.store.PrimaryDomain(ctx, siteID)
	if err != nil && !errors.Is(err, ErrNoVerifiedDomain) {
		return SiteContext{}, err
	}

	return SiteContext{
		Domain:          strings.TrimRight(domain, "/"),
		PrimaryLanguage: NormalizeLocale(locale),
	}, nil
}

// ErrNoVerifiedDomain is returned by the store when the site has no verified
// domain; the resolver treats it as "no resolvable domain".
var ErrNoVerifiedDomain = errors.New("no verified domain for site")

// NormalizeLocale reduces a locale string to its language subtag: "pt-BR" ->
// "pt", "en-US" -> "en", "en" -> "en", "" -> "". Case and whitespace are
// normalized. Unknown language tags pass through, preserving the caller's
// own validation.
func NormalizeLocale(locale string) string {
	l := strings.ToLower(strings.TrimSpace(locale))
	if l == "" {
		return ""
	}
	if i := strings.IndexByte(l, '-'); i >= 0 {
		l = l[:i]
	}
	return l
}

// PGStore implements Store over a database.Pool.
type PGStore struct {
	pool database.Pool
}

// NewPGStore returns a Store backed by the given pool.
func NewPGStore(pool database.Pool) *PGStore {
	return &PGStore{pool: pool}
}

// SiteLocale loads the locale column of the site ("" when unset).
func (s *PGStore) SiteLocale(ctx context.Context, siteID uuid.UUID) (string, error) {
	var locale string
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(locale, '') FROM sites WHERE id = $1 AND deleted_at IS NULL`,
		siteID,
	).Scan(&locale)
	if err != nil {
		return "", err
	}
	return locale, nil
}

// PrimaryDomain returns the best verified domain: primary first, otherwise
// any verified domain (oldest first). Returns ErrNoVerifiedDomain when none.
func (s *PGStore) PrimaryDomain(ctx context.Context, siteID uuid.UUID) (string, error) {
	var domain string
	err := s.pool.QueryRow(ctx,
		`SELECT domain FROM site_domains
		 WHERE site_id = $1 AND verified = true
		 ORDER BY is_primary DESC, created_at ASC
		 LIMIT 1`,
		siteID,
	).Scan(&domain)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNoVerifiedDomain
		}
		return "", err
	}
	return domain, nil
}
