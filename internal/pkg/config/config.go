package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Auth      AuthConfig
	OAuth     OAuthConfig
	Storage   StorageConfig
	Cache     CacheConfig
	AI        AIConfig
	SEO       SEOConfig
	Pexels    PexelsConfig
	Research  ResearchConfig
	Freshness FreshnessConfig
	Editorial EditorialConfig
	Revalidate RevalidateConfig
	// SitesWhitelist restricts which sites are returned by GET /sites (the
	// Admin site selector). Comma-separated site UUIDs; empty = all sites.
	SitesWhitelist []string
	Debug          bool
	LogLevel       string
	LogFormat      string
	// MigrationsDir is the directory containing the SQL migration files.
	// It must remain reachable at runtime, so the deploy image ships the
	// migrations folder next to the API binary.
	MigrationsDir string
	// MigrationTimeout bounds the whole migrate-up step at startup,
	// including waiting on the advisory lock held by another instance.
	MigrationTimeout time.Duration
}

type ServerConfig struct {
	Host    string
	Port    int
	Timeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
	MaxConns int
	MinConns int
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
}

type AuthConfig struct {
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
}

type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type OAuthConfig struct {
	Google  OAuthProviderConfig
	GitHub  OAuthProviderConfig
}

type StorageConfig struct {
	Driver       string
	LocalPath    string
	S3Bucket     string
	S3Region     string
	S3Key        string
	S3Secret     string
	S3Endpoint   string
	MaxFileSize  int
}

type CacheConfig struct {
	Driver string
	TTL    time.Duration
}

type ProviderConfig struct {
	Name       string
	Model      string
	APIKey     string
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	Weight     int
	Priority   int
	Enabled    bool
}

type AIConfig struct {
	Enabled          bool
	DefaultProvider  string
	GlobalTimeout    time.Duration
	Providers        []ProviderConfig
	RetryMaxAttempts int
	RetryBaseDelay   time.Duration
	RetryMaxDelay    time.Duration
	CBFailureThreshold int
	CBRecoveryTimeout  time.Duration
	CBHalfOpenMaxReqs  int
}

// SEOConfig controls the SEO Intelligence Engine behavior.
type SEOConfig struct {
	// MinPublishScore is the minimum audit score an article must reach
	// before automatic publication is allowed. Set to 0 to disable the
	// publish gate entirely.
	MinPublishScore float64

	// CompetitorDomains are domains that must never be linked as external
	// references (direct competitors), e.g. "competitor.com".
	CompetitorDomains []string

	// InternalLinkMinScore is the minimum score an internal link candidate
	// must reach to be selected (default 40).
	InternalLinkMinScore int

	// InternalLinkMax is the maximum number of internal links inserted
	// per article (default 5).
	InternalLinkMax int

	// ExternalLinkMinReliability is the minimum reliability score (0-100)
	// an external source must have to be linked (default 75).
	ExternalLinkMinReliability int

	// DefaultAuthor is the byline attached to generated articles that carry
	// no explicit author (SEO_DEFAULT_AUTHOR). It feeds the EEAT gate
	// analysis so author presence never silently disappears between the
	// generation pipeline and the publish gate.
	DefaultAuthor string
}

// PexelsConfig controls the Pexels image provider used to enrich generated
// articles with a real photograph (featured image + in-body image). The API
// key is only read from the environment; it is never logged or exposed.
type PexelsConfig struct {
	// APIKey is the Pexels API key (PEXELS_API_KEY). Empty disables the
	// image provider entirely (articles publish without images).
	APIKey string
	// Timeout bounds each Pexels search request (default 8s).
	Timeout time.Duration
}

// ResearchConfig controls the AI Research Intelligence behavior.
type ResearchConfig struct {
	// CacheTTL is how long a deep research result is reused before a new
	// search runs for the same topic (default 24h). Set to 0 for the default.
	CacheTTL time.Duration
}

// FreshnessConfig controls the Freshness Engine + News Intelligence behavior.
type FreshnessConfig struct {
	// SweepEnabled turns the once-per-day re-evaluation sweep on/off
	// (default true).
	SweepEnabled bool

	// NewsMaxDays is the NEWS temporal window (max source age kept usable,
	// default 30).
	NewsMaxDays int

	// NewsNeverOlderDays is the absolute NEWS cutoff — sources older than
	// this are never used (default 90).
	NewsNeverOlderDays int
}

// EditorialConfig controls the AI Editorial Brain behavior.
type EditorialConfig struct {
	// MinFinalScore is the minimum final editorial note (0-100) an article
	// must reach to be published (default 90). Below it, the article returns
	// automatically to review. Set to 0 to disable the editorial gate.
	MinFinalScore float64
}

// RevalidateConfig controls the Next.js ISR revalidation webhook. The API
// POSTs {site}/api/revalidate after every publish so the public site's ISR
// cache is refreshed immediately (fail-open: broken revalidation never
// blocks publishing).
type RevalidateConfig struct {
	// PublicURLs are the public site base URLs (comma separated, e.g.
	// "https://blog.example.com,https://blog2.example.com").
	PublicURLs []string
	// Token must match SITE_REVALIDATE_TOKEN on the public site.
	Token string
	// Enabled toggles the webhook. Defaults to true (no-op without URLs).
	Enabled bool
	// Timeout bounds each POST (default 5s).
	Timeout time.Duration
}

var errDefaultJWTSecret = fmt.Errorf("JWT_SECRET must be changed from the default value for security")

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Warn(".env file not found, using environment variables", "error", err)
	}

	cfg := &Config{}

	cfg.Debug = getEnvBool("DEBUG", true)
	cfg.LogLevel = getEnv("LOG_LEVEL", "debug")
	cfg.LogFormat = getEnv("LOG_FORMAT", "console")

	cfg.Server = ServerConfig{
		Host:    getEnv("SERVER_HOST", "0.0.0.0"),
		Port:    getEnvInt("SERVER_PORT", getEnvInt("PORT", 8080)),
		Timeout: getEnvDuration("SERVER_TIMEOUT", 30*time.Second),
	}

	cfg.MigrationsDir = getEnv("MIGRATIONS_DIR", "migrations")
	cfg.MigrationTimeout = getEnvDuration("MIGRATION_TIMEOUT", 10*time.Minute)

	cfg.Database = DatabaseConfig{
		Host:     getEnv("DATABASE_HOST", "localhost"),
		Port:     getEnvInt("DATABASE_PORT", 5432),
		User:     getEnv("DATABASE_USER", "nexora"),
		Password: getEnv("DATABASE_PASSWORD", "nexora_secret"),
		Name:     getEnv("DATABASE_NAME", "nexora"),
		SSLMode:  getEnv("DATABASE_SSLMODE", "disable"),
		MaxConns: getEnvInt("DATABASE_MAX_CONNS", 25),
		MinConns: getEnvInt("DATABASE_MIN_CONNS", 5),
	}

	cfg.Redis = RedisConfig{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnvInt("REDIS_PORT", 6379),
		Password: getEnv("REDIS_PASSWORD", ""),
	}

	cfg.Auth = AuthConfig{
		JWTSecret:     getEnv("JWT_SECRET", "change-me-to-a-random-64-char-string"),
		JWTAccessTTL:  getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL: getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
	}

	cfg.OAuth = OAuthConfig{
		Google: OAuthProviderConfig{
			ClientID:     getEnv("OAUTH_GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("OAUTH_GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("OAUTH_GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/oauth/callback"),
		},
		GitHub: OAuthProviderConfig{
			ClientID:     getEnv("OAUTH_GITHUB_CLIENT_ID", ""),
			ClientSecret: getEnv("OAUTH_GITHUB_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("OAUTH_GITHUB_REDIRECT_URL", "http://localhost:8080/api/v1/auth/oauth/callback"),
		},
	}

	cfg.Storage = StorageConfig{
		Driver:       getEnv("STORAGE_DRIVER", "local"),
		LocalPath:    getEnv("STORAGE_LOCAL_PATH", "./data/storage"),
		S3Bucket:     getEnv("STORAGE_S3_BUCKET", ""),
		S3Region:     getEnv("STORAGE_S3_REGION", ""),
		S3Key:        getEnv("STORAGE_S3_KEY", ""),
		S3Secret:     getEnv("STORAGE_S3_SECRET", ""),
		S3Endpoint:   getEnv("STORAGE_S3_ENDPOINT", ""),
		MaxFileSize:  getEnvInt("STORAGE_MAX_FILE_SIZE", 50*1024*1024),
	}

	cfg.Cache = CacheConfig{
		Driver: getEnv("CACHE_DRIVER", "memory"),
		TTL:    getEnvDuration("CACHE_TTL", 5*time.Minute),
	}

	geminiAPIKey := getEnv("AI_GEMINI_API_KEY", "")
	geminiModel := getEnv("AI_GEMINI_MODEL", "gemini-2.0-flash")
	geminiBaseURL := getEnv("AI_GEMINI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta")

	providers := []ProviderConfig{}
	if geminiAPIKey != "" {
		providers = append(providers, ProviderConfig{
			Name:       "gemini",
			Model:      geminiModel,
			APIKey:     geminiAPIKey,
			BaseURL:    geminiBaseURL,
			Timeout:    getEnvDuration("AI_GEMINI_TIMEOUT", 60*time.Second),
			MaxRetries: getEnvInt("AI_GEMINI_MAX_RETRIES", 3),
			Weight:     getEnvInt("AI_GEMINI_WEIGHT", 10),
			Priority:   getEnvInt("AI_GEMINI_PRIORITY", 1),
			Enabled:    getEnvBool("AI_GEMINI_ENABLED", true),
		})
	}

	cfg.AI = AIConfig{
		Enabled:          getEnvBool("AI_ENABLED", true),
		DefaultProvider:  getEnv("AI_DEFAULT_PROVIDER", ""),
		GlobalTimeout:    getEnvDuration("AI_GLOBAL_TIMEOUT", 60*time.Second),
		Providers:        providers,
		RetryMaxAttempts: getEnvInt("AI_RETRY_MAX_ATTEMPTS", 3),
		RetryBaseDelay:   getEnvDuration("AI_RETRY_BASE_DELAY", 100*time.Millisecond),
		RetryMaxDelay:    getEnvDuration("AI_RETRY_MAX_DELAY", 5*time.Second),
		CBFailureThreshold: getEnvInt("AI_CB_FAILURE_THRESHOLD", 5),
		CBRecoveryTimeout:  getEnvDuration("AI_CB_RECOVERY_TIMEOUT", 30*time.Second),
		CBHalfOpenMaxReqs:  getEnvInt("AI_CB_HALF_OPEN_MAX_REQS", 3),
	}

	cfg.SEO = SEOConfig{
		MinPublishScore:          getEnvFloat("SEO_MIN_PUBLISH_SCORE", 80),
		CompetitorDomains:        getEnvCSV("SEO_COMPETITOR_DOMAINS"),
		InternalLinkMinScore:     getEnvInt("SEO_INTERNAL_LINK_MIN_SCORE", 40),
		InternalLinkMax:          getEnvInt("SEO_INTERNAL_LINK_MAX", 5),
		ExternalLinkMinReliability: getEnvInt("SEO_EXTERNAL_LINK_MIN_RELIABILITY", 75),
		DefaultAuthor:        getEnv("SEO_DEFAULT_AUTHOR", ""),
	}

	cfg.Pexels = PexelsConfig{
		APIKey:  getEnv("PEXELS_API_KEY", ""),
		Timeout: getEnvDuration("PEXELS_TIMEOUT", 8*time.Second),
	}

	cfg.Research = ResearchConfig{
		CacheTTL: getEnvDuration("RESEARCH_CACHE_TTL", 24*time.Hour),
	}

	cfg.Freshness = FreshnessConfig{
		SweepEnabled:       getEnvBool("FRESHNESS_SWEEP_ENABLED", true),
		NewsMaxDays:        getEnvInt("FRESHNESS_NEWS_MAX_DAYS", 30),
		NewsNeverOlderDays: getEnvInt("FRESHNESS_NEWS_NEVER_OLDER_DAYS", 90),
	}
	cfg.Editorial = EditorialConfig{
		MinFinalScore: getEnvFloat("EDITORIAL_MIN_FINAL_SCORE", 90),
	}
	cfg.Revalidate = RevalidateConfig{
		PublicURLs: splitCSV(firstNonEmpty(os.Getenv("SITE_PUBLIC_URLS"), os.Getenv("SITE_PUBLIC_URL"))),
		Token:      getEnv("SITE_REVALIDATE_TOKEN", ""),
		Enabled:    getEnvBool("SITE_REVALIDATION_ENABLED", true),
		Timeout:    getEnvDuration("SITE_REVALIDATE_TIMEOUT", 5*time.Second),
	}

	// SITES_WHITELIST: when set, only these sites are listed by GET /sites
	// (Admin site selector). Empty = all sites (default behavior). Invalid
	// entries are dropped so a malformed value never breaks the API.
	cfg.SitesWhitelist = validSiteIDs(getEnvCSV("SITES_WHITELIST"))

	if cfg.Auth.JWTSecret == "change-me-to-a-random-64-char-string" {
		return nil, errDefaultJWTSecret
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		val = strings.ToLower(val)
		return val == "true" || val == "1" || val == "yes"
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return fallback
}

// getEnvCSV parses a comma-separated environment variable into a trimmed list.
func getEnvCSV(key string) []string {
	val := os.Getenv(key)
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, strings.ToLower(p))
		}
	}
	return out
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// validSiteIDs keeps only well-formed UUID entries from a comma-separated
// site whitelist. Malformed entries are dropped so a typo in SITES_WHITELIST
// never breaks the sites listing.
func validSiteIDs(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, id := range raw {
		if _, err := uuid.Parse(id); err == nil {
			out = append(out, id)
		}
	}
	return out
}
