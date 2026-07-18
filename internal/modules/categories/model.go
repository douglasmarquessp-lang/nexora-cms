package categories

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const (
	EventCategoryCreated kernel.EventType = "category.created"
	EventCategoryUpdated kernel.EventType = "category.updated"
	EventCategoryDeleted kernel.EventType = "category.deleted"
)

var (
	ErrCategoryNotFound      = errors.New("category not found")
	ErrCategorySlugExists    = errors.New("category slug already exists")
	ErrInvalidParentCategory = errors.New("invalid parent category")
	ErrDatabaseNotAvail      = errors.New("database not available")
	ErrCircularParent        = errors.New("circular parent reference detected")
)

type Category struct {
	ID          uuid.UUID  `json:"id"`
	SiteID      uuid.UUID  `json:"site_id"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description,omitempty"`
	Icon        string     `json:"icon,omitempty"`
	Color       string     `json:"color,omitempty"`
	SortOrder   int        `json:"sort_order"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	Children    []Category `json:"children,omitempty"`
}

type CreateCategoryRequest struct {
	Name        string     `json:"name"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	Description string     `json:"description,omitempty"`
	Icon        string     `json:"icon,omitempty"`
	Color       string     `json:"color,omitempty"`
	SortOrder   int        `json:"sort_order"`
}

type UpdateCategoryRequest struct {
	Name        *string     `json:"name,omitempty"`
	ParentID    **uuid.UUID `json:"parent_id,omitempty"`
	Description *string     `json:"description,omitempty"`
	Icon        *string     `json:"icon,omitempty"`
	Color       *string     `json:"color,omitempty"`
	SortOrder   *int        `json:"sort_order,omitempty"`
}

type CategoryListResponse struct {
	Categories []Category `json:"categories"`
	Total      int        `json:"total"`
}
