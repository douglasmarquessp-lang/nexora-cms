package tags

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const (
	EventTagCreated kernel.EventType = "tag.created"
	EventTagUpdated kernel.EventType = "tag.updated"
	EventTagDeleted kernel.EventType = "tag.deleted"
)

var (
	ErrTagNotFound      = errors.New("tag not found")
	ErrTagSlugExists    = errors.New("tag slug already exists")
	ErrDatabaseNotAvail = errors.New("database not available")
)

type Tag struct {
	ID        uuid.UUID  `json:"id"`
	SiteID    uuid.UUID  `json:"site_id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Color     string     `json:"color,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type CreateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

type UpdateTagRequest struct {
	Name  *string `json:"name,omitempty"`
	Color *string `json:"color,omitempty"`
}

type TagListResponse struct {
	Tags  []Tag `json:"tags"`
	Total int   `json:"total"`
}
