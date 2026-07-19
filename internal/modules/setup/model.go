package setup

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "setup"

type SystemInstallation struct {
	ID          uuid.UUID  `json:"id"`
	Installed   bool       `json:"installed"`
	InstalledAt *time.Time `json:"installed_at,omitempty"`
	CmsName     string     `json:"cms_name"`
	AdminName   string     `json:"admin_name"`
	AdminEmail  string     `json:"admin_email"`
	DefaultSite string     `json:"default_site"`
	Version     string     `json:"version"`
	Locale      string     `json:"locale"`
	Timezone    string     `json:"timezone"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type InstallRequest struct {
	CmsName         string `json:"cms_name"`
	AdminName       string `json:"admin_name"`
	AdminEmail      string `json:"admin_email"`
	Password        string `json:"password"`
	SiteName        string `json:"site_name"`
	SiteDescription string `json:"site_description,omitempty"`
	Language        string `json:"language"`
	Timezone        string `json:"timezone"`
	SiteURL         string `json:"site_url"`
	AIProvider      string `json:"ai_provider,omitempty"`
	Theme           string `json:"theme,omitempty"`
	Locale          string `json:"locale,omitempty"`
}

type FinishResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type StatusResponse struct {
	Installed   bool       `json:"installed"`
	InstalledAt *time.Time `json:"installed_at,omitempty"`
	CmsName     string     `json:"cms_name,omitempty"`
	Version     string     `json:"version,omitempty"`
}

type ConfigResponse struct {
	Locales     []string `json:"locales"`
	Timezones   []string `json:"timezones"`
	Themes      []string `json:"themes"`
	AIProviders []string `json:"ai_providers"`
}

const (
	EventSetupStarted  kernel.EventType = "setup.started"
	EventSetupFinished kernel.EventType = "setup.finished"
)

var (
	ErrAlreadyInstalled = errors.New("system is already installed")
	ErrNotInstalled     = errors.New("system is not installed yet")
	ErrInvalidEmail     = errors.New("invalid email address")
	ErrWeakPassword     = errors.New("password does not meet strength requirements")
	ErrRequiredField    = errors.New("required field is missing")
	ErrDatabaseNotAvail = errors.New("database not available")
	ErrInvalidInput     = errors.New("invalid input")
)
