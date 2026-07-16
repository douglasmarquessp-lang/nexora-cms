package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
	OAuth    OAuthConfig
	Storage  StorageConfig
	Cache    CacheConfig
	Debug    bool
	LogLevel string
	LogFormat string
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
		Port:    getEnvInt("SERVER_PORT", 8080),
		Timeout: getEnvDuration("SERVER_TIMEOUT", 30*time.Second),
	}

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

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
