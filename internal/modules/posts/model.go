package posts

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

type PostStatus string

const (
	PostStatusDraft     PostStatus = "draft"
	PostStatusPublished PostStatus = "published"
	PostStatusScheduled PostStatus = "scheduled"
	PostStatusArchived  PostStatus = "archived"
)

const (
	EventPostCreated   kernel.EventType = "post.created"
	EventPostUpdated   kernel.EventType = "post.updated"
	EventPostDeleted   kernel.EventType = "post.deleted"
	EventPostPublished kernel.EventType = "post.published"
	EventPostArchived  kernel.EventType = "post.archived"
)

var (
	ErrPostNotFound      = errors.New("post not found")
	ErrPostSlugExists    = errors.New("post slug already exists")
	ErrInvalidPostStatus = errors.New("invalid post status")
	ErrDatabaseNotAvail  = errors.New("database not available")
	ErrInvalidPagination = errors.New("invalid pagination parameters")
	ErrPostNotInSite     = errors.New("post does not belong to this site")
)

type Post struct {
	ID          uuid.UUID              `json:"id"`
	SiteID      uuid.UUID              `json:"site_id"`
	Title       string                 `json:"title"`
	Slug        string                 `json:"slug"`
	Content     []interface{}          `json:"content"`
	Excerpt     string                 `json:"excerpt"`
	Status      PostStatus             `json:"status"`
	AuthorID    uuid.UUID              `json:"author_id"`
	PublishedAt *time.Time             `json:"published_at,omitempty"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	PostMeta    map[string]interface{} `json:"post_meta,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DeletedAt   *time.Time             `json:"deleted_at,omitempty"`
	Categories  []Category             `json:"categories,omitempty"`
	Tags        []Tag                  `json:"tags,omitempty"`
}

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

type CreatePostRequest struct {
	Title       string                 `json:"title"`
	Content     []interface{}          `json:"content"`
	Excerpt     string                 `json:"excerpt"`
	Status      PostStatus             `json:"status"`
	PublishedAt *time.Time             `json:"published_at,omitempty"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	PostMeta    map[string]interface{} `json:"post_meta,omitempty"`
	CategoryIDs []uuid.UUID            `json:"category_ids,omitempty"`
	TagIDs      []uuid.UUID            `json:"tag_ids,omitempty"`
}

type UpdatePostRequest struct {
	Title       *string                `json:"title,omitempty"`
	Content     *[]interface{}         `json:"content,omitempty"`
	Excerpt     *string                `json:"excerpt,omitempty"`
	Status      *PostStatus            `json:"status,omitempty"`
	PublishedAt **time.Time            `json:"published_at,omitempty"`
	ScheduledAt **time.Time            `json:"scheduled_at,omitempty"`
	PostMeta    *map[string]interface{} `json:"post_meta,omitempty"`
	CategoryIDs []uuid.UUID            `json:"category_ids,omitempty"`
	TagIDs      []uuid.UUID            `json:"tag_ids,omitempty"`
}

type PostListRequest struct {
	SiteID     uuid.UUID
	Status     PostStatus
	AuthorID   uuid.UUID
	CategoryID uuid.UUID
	Search     string
	Page       int
	PerPage    int
	Sort       string
	Order      string
}

type PostListResponse struct {
	Posts    []PostSummary `json:"posts"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PerPage  int           `json:"per_page"`
}

type PostSummary struct {
	ID            uuid.UUID              `json:"id"`
	Title         string                 `json:"title"`
	Slug          string                 `json:"slug"`
	Excerpt       string                 `json:"excerpt"`
	Status        PostStatus             `json:"status"`
	AuthorID      uuid.UUID              `json:"author_id"`
	PublishedAt   *time.Time             `json:"published_at,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	CategoryCount int                    `json:"category_count"`
	TagCount      int                    `json:"tag_count"`
}

type SetStatusRequest struct {
	Status PostStatus `json:"status"`
}
