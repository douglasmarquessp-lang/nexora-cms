package site

import (
	"time"

	"github.com/google/uuid"
)

type SiteStatus string

const (
	SiteStatusActive      SiteStatus = "active"
	SiteStatusInactive    SiteStatus = "inactive"
	SiteStatusSuspended   SiteStatus = "suspended"
	SiteStatusMaintenance SiteStatus = "maintenance"
)

type Site struct {
	ID           uuid.UUID              `json:"id"`
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	Description  string                 `json:"description,omitempty"`
	Status       SiteStatus             `json:"status"`
	OwnerID      uuid.UUID              `json:"owner_id"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
	FeatureFlags map[string]interface{} `json:"feature_flags,omitempty"`
	Theme        string                 `json:"theme,omitempty"`
	Locale       string                 `json:"locale,omitempty"`
	Timezone     string                 `json:"timezone,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	DeletedAt    *time.Time             `json:"deleted_at,omitempty"`
}

type SiteDomain struct {
	ID         uuid.UUID `json:"id"`
	SiteID     uuid.UUID `json:"site_id"`
	Domain     string    `json:"domain"`
	IsPrimary  bool      `json:"is_primary"`
	Verified   bool      `json:"verified"`
	SSLEnabled bool      `json:"ssl_enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type GlobalSetting struct {
	ID          uuid.UUID   `json:"id"`
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type SiteSetting struct {
	ID        uuid.UUID   `json:"id"`
	SiteID    uuid.UUID   `json:"site_id"`
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type CreateSiteRequest struct {
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	Description  string                 `json:"description,omitempty"`
	Theme        string                 `json:"theme,omitempty"`
	Locale       string                 `json:"locale,omitempty"`
	Timezone     string                 `json:"timezone,omitempty"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
	FeatureFlags map[string]interface{} `json:"feature_flags,omitempty"`
}

type UpdateSiteRequest struct {
	Name         *string                 `json:"name,omitempty"`
	Description  *string                 `json:"description,omitempty"`
	Status       *SiteStatus             `json:"status,omitempty"`
	Theme        *string                 `json:"theme,omitempty"`
	Locale       *string                 `json:"locale,omitempty"`
	Timezone     *string                 `json:"timezone,omitempty"`
	Settings     *map[string]interface{} `json:"settings,omitempty"`
	FeatureFlags *map[string]interface{} `json:"feature_flags,omitempty"`
}

type SiteResponse struct {
	*Site `json:",inline"`
}

type SiteListResponse struct {
	Sites      []Site `json:"sites"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PerPage    int    `json:"per_page"`
	TotalPages int    `json:"total_pages"`
}

type AddDomainRequest struct {
	Domain    string `json:"domain"`
	IsPrimary bool   `json:"is_primary"`
}

type UpdateGlobalSettingRequest struct {
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	Type        string      `json:"type,omitempty"`
	Description string      `json:"description,omitempty"`
}

type SetSiteSettingRequest struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}
