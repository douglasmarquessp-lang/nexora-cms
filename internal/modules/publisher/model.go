package publisher

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "publisher"

type PubStatus string

const (
	PubStatusDraft     PubStatus = "draft"
	PubStatusPublished PubStatus = "published"
	PubStatusScheduled PubStatus = "scheduled"
	PubStatusUnpublished PubStatus = "unpublished"
	PubStatusArchived  PubStatus = "archived"
	PubStatusDeleted   PubStatus = "deleted"
)

type QueueStatus string

const (
	QueuePending   QueueStatus = "pending"
	QueueRunning   QueueStatus = "running"
	QueueCompleted QueueStatus = "completed"
	QueueFailed    QueueStatus = "failed"
	QueueCancelled QueueStatus = "cancelled"
)

type ScheduleStatus string

const (
	ScheduleScheduled ScheduleStatus = "scheduled"
	ScheduleRunning   ScheduleStatus = "running"
	ScheduleCompleted ScheduleStatus = "completed"
	ScheduleCancelled ScheduleStatus = "cancelled"
	ScheduleFailed    ScheduleStatus = "failed"
)

type QueueAction string

const (
	QueueActionPublish   QueueAction = "publish"
	QueueActionUnpublish QueueAction = "unpublish"
	QueueActionRepublish QueueAction = "republish"
	QueueActionUpdate    QueueAction = "update"
)

type Visibility string

const (
	VisibilityPublic   Visibility = "public"
	VisibilityPrivate  Visibility = "private"
	VisibilityPassword Visibility = "password"
)

type HistoryAction string

const (
	HistoryCreated     HistoryAction = "created"
	HistoryPublished   HistoryAction = "published"
	HistoryUpdated     HistoryAction = "updated"
	HistoryUnpublished HistoryAction = "unpublished"
	HistoryRepublished HistoryAction = "republished"
	HistoryScheduled   HistoryAction = "scheduled"
	HistoryCancelled   HistoryAction = "cancelled"
	HistoryArchived    HistoryAction = "archived"
	HistoryDeleted     HistoryAction = "deleted"
)

type Publication struct {
	ID               uuid.UUID              `json:"id"`
	SiteID           uuid.UUID              `json:"site_id"`
	PostID           *uuid.UUID             `json:"post_id,omitempty"`
	Title            string                 `json:"title"`
	Content          string                 `json:"content,omitempty"`
	Excerpt          string                 `json:"excerpt,omitempty"`
	Slug             string                 `json:"slug"`
	URL              string                 `json:"url"`
	CanonicalURL     string                 `json:"canonical_url,omitempty"`
	Language         string                 `json:"language"`
	Translations     map[string]interface{} `json:"translations,omitempty"`
	MultilingualURLs map[string]interface{} `json:"multilingual_urls,omitempty"`
	Status           PubStatus              `json:"status"`
	Visibility       Visibility             `json:"visibility"`
	AuthorID         *uuid.UUID             `json:"author_id,omitempty"`
	PublishedBy      *uuid.UUID             `json:"published_by,omitempty"`
	PublishedAt      *time.Time             `json:"published_at,omitempty"`
	UnpublishedAt    *time.Time             `json:"unpublished_at,omitempty"`
	ScheduledAt      *time.Time             `json:"scheduled_at,omitempty"`
	IsFeatured       bool                   `json:"is_featured"`
	MetaTitle        string                 `json:"meta_title,omitempty"`
	MetaDescription  string                 `json:"meta_description,omitempty"`
	OgImage          string                 `json:"og_image,omitempty"`
	FeaturedImageURL string                 `json:"featured_image_url,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	Categories       []string               `json:"categories,omitempty"`
	WordCount        int                    `json:"word_count"`
	ReadingTime      int                    `json:"reading_time"`
	Revision         int                    `json:"revision"`
	Checksum         string                 `json:"checksum,omitempty"`
	Source           string                 `json:"source,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy        *uuid.UUID             `json:"created_by,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type PublicationHistory struct {
	ID             uuid.UUID              `json:"id"`
	PublicationID  uuid.UUID              `json:"publication_id"`
	SiteID         uuid.UUID              `json:"site_id"`
	Action         HistoryAction          `json:"action"`
	PreviousStatus string                 `json:"previous_status,omitempty"`
	NewStatus      string                 `json:"new_status,omitempty"`
	Title          string                 `json:"title,omitempty"`
	Slug           string                 `json:"slug,omitempty"`
	Changes        map[string]interface{} `json:"changes,omitempty"`
	Reason         string                 `json:"reason,omitempty"`
	PerformedBy    *uuid.UUID             `json:"performed_by,omitempty"`
	PerformedAt    time.Time              `json:"performed_at"`
	CreatedAt      time.Time              `json:"created_at"`
}

type QueueItem struct {
	ID            uuid.UUID              `json:"id"`
	SiteID        uuid.UUID              `json:"site_id"`
	PublicationID *uuid.UUID             `json:"publication_id,omitempty"`
	Action        QueueAction            `json:"action"`
	Status        QueueStatus            `json:"status"`
	Priority      int                    `json:"priority"`
	ScheduledFor  *time.Time             `json:"scheduled_for,omitempty"`
	StartedAt     *time.Time             `json:"started_at,omitempty"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	RetryCount    int                    `json:"retry_count"`
	MaxRetries    int                    `json:"max_retries"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy     *uuid.UUID             `json:"created_by,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type Schedule struct {
	ID             uuid.UUID              `json:"id"`
	SiteID         uuid.UUID              `json:"site_id"`
	PublicationID  uuid.UUID              `json:"publication_id"`
	ScheduledAt    time.Time              `json:"scheduled_at"`
	Action         string                 `json:"action"`
	Status         ScheduleStatus         `json:"status"`
	Recurrence     string                 `json:"recurrence,omitempty"`
	RecurrenceEnd  *time.Time             `json:"recurrence_end,omitempty"`
	NotifyOnPublish bool                  `json:"notify_on_publish"`
	NotifyUsers    []uuid.UUID            `json:"notify_users,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy      *uuid.UUID             `json:"created_by,omitempty"`
	CancelledAt    *time.Time             `json:"cancelled_at,omitempty"`
	CancelReason   string                 `json:"cancel_reason,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type PublicationMetrics struct {
	ID              uuid.UUID              `json:"id"`
	SiteID          uuid.UUID              `json:"site_id"`
	PublicationID   uuid.UUID              `json:"publication_id"`
	ViewCount       int64                  `json:"view_count"`
	UniqueVisitors  int64                  `json:"unique_visitors"`
	AvgTimeSeconds  float64                `json:"avg_time_seconds"`
	BounceRate      float64                `json:"bounce_rate"`
	ShareCount      int                    `json:"share_count"`
	CommentCount    int                    `json:"comment_count"`
	LikeCount       int                    `json:"like_count"`
	ClickCount      int                    `json:"click_count"`
	CTR             float64                `json:"ctr"`
	ScrollDepth     float64                `json:"scroll_depth"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	RecordedAt      time.Time              `json:"recorded_at"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// --- DTOs ---

type PublishRequest struct {
	PostID           *uuid.UUID             `json:"post_id,omitempty"`
	Title            string                 `json:"title"`
	Content          string                 `json:"content,omitempty"`
	Excerpt          string                 `json:"excerpt,omitempty"`
	Slug             string                 `json:"slug,omitempty"`
	Language         string                 `json:"language,omitempty"`
	Visibility       Visibility             `json:"visibility,omitempty"`
	AuthorID         *uuid.UUID             `json:"author_id,omitempty"`
	IsFeatured       bool                   `json:"is_featured,omitempty"`
	MetaTitle        string                 `json:"meta_title,omitempty"`
	MetaDescription  string                 `json:"meta_description,omitempty"`
	OgImage          string                 `json:"og_image,omitempty"`
	FeaturedImageURL string                 `json:"featured_image_url,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	Categories       []string               `json:"categories,omitempty"`
	CanonicalURL     string                 `json:"canonical_url,omitempty"`
	Translations     map[string]interface{} `json:"translations,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Source           string                 `json:"source,omitempty"`
}

type UpdatePublicationRequest struct {
	Title            *string                 `json:"title,omitempty"`
	Content          *string                 `json:"content,omitempty"`
	Excerpt          *string                 `json:"excerpt,omitempty"`
	Slug             *string                 `json:"slug,omitempty"`
	Language         *string                 `json:"language,omitempty"`
	Visibility       *Visibility             `json:"visibility,omitempty"`
	IsFeatured       *bool                   `json:"is_featured,omitempty"`
	MetaTitle        *string                 `json:"meta_title,omitempty"`
	MetaDescription  *string                 `json:"meta_description,omitempty"`
	OgImage          *string                 `json:"og_image,omitempty"`
	FeaturedImageURL *string                 `json:"featured_image_url,omitempty"`
	Tags             *[]string               `json:"tags,omitempty"`
	Categories       *[]string               `json:"categories,omitempty"`
	CanonicalURL     *string                 `json:"canonical_url,omitempty"`
	Translations     *map[string]interface{} `json:"translations,omitempty"`
	Metadata         *map[string]interface{} `json:"metadata,omitempty"`
}

type ScheduleRequest struct {
	PublicationID  uuid.UUID  `json:"publication_id"`
	ScheduledAt    time.Time  `json:"scheduled_at"`
	Action         string     `json:"action,omitempty"`
	Recurrence     string     `json:"recurrence,omitempty"`
	RecurrenceEnd  *time.Time `json:"recurrence_end,omitempty"`
	NotifyOnPublish bool      `json:"notify_on_publish,omitempty"`
}

type QueueRequest struct {
	PublicationID uuid.UUID  `json:"publication_id"`
	Action        string     `json:"action,omitempty"`
	ScheduledFor  *time.Time `json:"scheduled_for,omitempty"`
	Priority      int        `json:"priority,omitempty"`
}

type RetryQueueRequest struct {
	QueueItemID uuid.UUID `json:"queue_item_id"`
}

type PublishResponse struct {
	Publication *Publication `json:"publication"`
	QueueItem   *QueueItem   `json:"queue_item,omitempty"`
	Schedule    *Schedule    `json:"schedule,omitempty"`
}

type PublicationMetricsSummary struct {
	TotalPublications int64   `json:"total_publications"`
	PublishedCount    int64   `json:"published_count"`
	ScheduledCount    int64   `json:"scheduled_count"`
	DraftCount        int64   `json:"draft_count"`
	ArchivedCount     int64   `json:"archived_count"`
	TotalViews        int64   `json:"total_views"`
	AvgViews          float64 `json:"avg_views"`
	TotalShares       int64   `json:"total_shares"`
	TotalComments     int64   `json:"total_comments"`
	QueueSize         int     `json:"queue_size"`
	PendingSchedules  int     `json:"pending_schedules"`
}

type PublicationListResponse struct {
	Publications []Publication `json:"publications"`
	Total        int           `json:"total"`
}

// --- Events ---

const (
	EventPubCreated        kernel.EventType = "publisher.publication.created"
	EventPubPublished      kernel.EventType = "publisher.publication.published"
	EventPubUpdated        kernel.EventType = "publisher.publication.updated"
	EventPubUnpublished    kernel.EventType = "publisher.publication.unpublished"
	EventPubRepublished    kernel.EventType = "publisher.publication.republished"
	EventPubScheduled      kernel.EventType = "publisher.publication.scheduled"
	EventPubCancelled      kernel.EventType = "publisher.publication.schedule.cancelled"
	EventPubArchived       kernel.EventType = "publisher.publication.archived"
	EventPubDeleted        kernel.EventType = "publisher.publication.deleted"
	EventPubQueueAdded     kernel.EventType = "publisher.queue.added"
	EventPubQueueStarted   kernel.EventType = "publisher.queue.started"
	EventPubQueueCompleted kernel.EventType = "publisher.queue.completed"
	EventPubQueueFailed    kernel.EventType = "publisher.queue.failed"
	EventPubQueueRetried   kernel.EventType = "publisher.queue.retried"
	EventPubSitemapUpdate  kernel.EventType = "publisher.sitemap.update"
	EventPubRSSUpdate      kernel.EventType = "publisher.rss.update"
	EventPubRobotsRefresh  kernel.EventType = "publisher.robots.refresh"
	EventPubCachePurge     kernel.EventType = "publisher.cache.purge"
)

// --- Errors ---

var (
	ErrPublicationNotFound    = errors.New("publication not found")
	ErrDuplicateSlug          = errors.New("duplicate slug for site")
	ErrInvalidSlug            = errors.New("invalid slug format")
	ErrInvalidLanguage        = errors.New("language must be 'pt' or 'en'")
	ErrInvalidVisibility      = errors.New("visibility must be public, private, or password")
	ErrInvalidStatus          = errors.New("invalid publication status")
	ErrInvalidAction          = errors.New("invalid queue action")
	ErrInvalidRecurrence      = errors.New("invalid recurrence pattern")
	ErrTitleRequired          = errors.New("title is required")
	ErrDatabaseNotAvail       = errors.New("database not available")
	ErrQueueItemNotFound      = errors.New("queue item not found")
	ErrScheduleNotFound       = errors.New("schedule not found")
	ErrScheduleAlreadyActive  = errors.New("schedule already active for this publication")
	ErrMaxRetriesExceeded     = errors.New("maximum retries exceeded")
	ErrPublicationAlreadyPublished = errors.New("publication already published")
	ErrPublicationNotPublished     = errors.New("publication is not published")
	ErrCannotModifyPublished       = errors.New("cannot modify published publication, use update instead")
	ErrHistoryNotFound             = errors.New("history entry not found")
	ErrMetricsNotFound             = errors.New("metrics not found")
)
