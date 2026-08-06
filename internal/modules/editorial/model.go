package editorial

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "editorial"

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
	TaskPriorityUrgent TaskPriority = "urgent"
)

type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
)

const (
	EventTaskCreated          kernel.EventType = "editorial.task.created"
	EventTaskUpdated          kernel.EventType = "editorial.task.updated"
	EventTaskDeleted          kernel.EventType = "editorial.task.deleted"
	EventRevisionSaved        kernel.EventType = "editorial.revision.saved"
	EventRevisionRestored     kernel.EventType = "editorial.revision.restored"
	EventApprovalRequested    kernel.EventType = "editorial.approval.requested"
	EventApprovalGranted      kernel.EventType = "editorial.approval.granted"
	EventApprovalRejected     kernel.EventType = "editorial.approval.rejected"
	EventCalendarEventCreated kernel.EventType = "editorial.calendar.created"
	EventCalendarEventUpdated kernel.EventType = "editorial.calendar.updated"
)

var (
	ErrTaskNotFound          = errors.New("task not found")
	ErrRevisionNotFound      = errors.New("revision not found")
	ErrApprovalNotFound      = errors.New("approval request not found")
	ErrCalendarEventNotFound = errors.New("calendar event not found")
	ErrWidgetNotFound        = errors.New("widget not found")
	ErrPostNotFound          = errors.New("post not found")
	ErrDatabaseNotAvail      = errors.New("database not available")
)

type DashboardStats struct {
	TotalPosts        int            `json:"total_posts"`
	PublishedPosts    int            `json:"published_posts"`
	DraftPosts        int            `json:"draft_posts"`
	ScheduledPosts    int            `json:"scheduled_posts"`
	ArchivedPosts     int            `json:"archived_posts"`
	TotalMedia        int            `json:"total_media"`
	TotalCategories   int            `json:"total_categories"`
	TotalTags         int            `json:"total_tags"`
	TotalTasks        int            `json:"total_tasks"`
	PendingTasks      int            `json:"pending_tasks"`
	PendingApprovals  int            `json:"pending_approvals"`
	RecentPosts       []PostSummary  `json:"recent_posts"`
	DraftPostsList    []PostSummary  `json:"draft_posts_list"`
	ScheduledPostsList []PostSummary `json:"scheduled_posts_list"`
}

type PostSummary struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Status      string     `json:"status"`
	Excerpt     string     `json:"excerpt"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Task struct {
	ID          uuid.UUID    `json:"id"`
	SiteID      uuid.UUID    `json:"site_id"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Status      TaskStatus   `json:"status"`
	Priority    TaskPriority `json:"priority"`
	AssigneeID  *uuid.UUID   `json:"assignee_id,omitempty"`
	DueDate     *time.Time   `json:"due_date,omitempty"`
	PostID      *uuid.UUID   `json:"post_id,omitempty"`
	CreatedBy   *uuid.UUID   `json:"created_by,omitempty"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	DeletedAt   *time.Time   `json:"deleted_at,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type CreateTaskRequest struct {
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Status      TaskStatus   `json:"status,omitempty"`
	Priority    TaskPriority `json:"priority,omitempty"`
	AssigneeID  *uuid.UUID   `json:"assignee_id,omitempty"`
	DueDate     *time.Time   `json:"due_date,omitempty"`
	PostID      *uuid.UUID   `json:"post_id,omitempty"`
}

type UpdateTaskRequest struct {
	Title       *string       `json:"title,omitempty"`
	Description *string       `json:"description,omitempty"`
	Status      *TaskStatus   `json:"status,omitempty"`
	Priority    *TaskPriority `json:"priority,omitempty"`
	AssigneeID  **uuid.UUID   `json:"assignee_id,omitempty"`
	DueDate     **time.Time   `json:"due_date,omitempty"`
	PostID      **uuid.UUID   `json:"post_id,omitempty"`
}

type Revision struct {
	ID        uuid.UUID              `json:"id"`
	PostID    uuid.UUID              `json:"post_id"`
	SiteID    uuid.UUID              `json:"site_id"`
	AuthorID  uuid.UUID              `json:"author_id"`
	Version   int                    `json:"version"`
	Title     string                 `json:"title"`
	Content   []interface{}          `json:"content"`
	Excerpt   string                 `json:"excerpt"`
	Slug      string                 `json:"slug"`
	PostMeta  map[string]interface{} `json:"post_meta,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Summary   string                 `json:"summary,omitempty"`
	ChangeLog string                 `json:"change_log,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type ApprovalRequest struct {
	ID           uuid.UUID      `json:"id"`
	SiteID       uuid.UUID      `json:"site_id"`
	PostID       uuid.UUID      `json:"post_id"`
	RequestedBy  uuid.UUID      `json:"requested_by"`
	Status       ApprovalStatus `json:"status"`
	Comments     string         `json:"comments,omitempty"`
	ReviewedBy   *uuid.UUID     `json:"reviewed_by,omitempty"`
	ReviewedAt   *time.Time     `json:"reviewed_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type CalendarEvent struct {
	ID          uuid.UUID  `json:"id"`
	SiteID      uuid.UUID  `json:"site_id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	EventDate   string     `json:"event_date"`
	EventType   string     `json:"event_type"`
	PostID      *uuid.UUID `json:"post_id,omitempty"`
	Color       string     `json:"color,omitempty"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreateCalendarEventRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	EventDate   string     `json:"event_date"`
	EventType   string     `json:"event_type,omitempty"`
	PostID      *uuid.UUID `json:"post_id,omitempty"`
	Color       string     `json:"color,omitempty"`
}

type UpdateCalendarEventRequest struct {
	Title       *string     `json:"title,omitempty"`
	Description *string     `json:"description,omitempty"`
	EventDate   *string     `json:"event_date,omitempty"`
	EventType   *string     `json:"event_type,omitempty"`
	PostID      **uuid.UUID `json:"post_id,omitempty"`
	Color       *string     `json:"color,omitempty"`
}

type Widget struct {
	ID         uuid.UUID              `json:"id"`
	SiteID     uuid.UUID              `json:"site_id"`
	WidgetType string                 `json:"widget_type"`
	Title      string                 `json:"title"`
	Config     map[string]interface{} `json:"config"`
	Position   int                    `json:"position"`
	Enabled    bool                   `json:"enabled"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

type UpdateWidgetRequest struct {
	WidgetType *string                `json:"widget_type,omitempty"`
	Title      *string                `json:"title,omitempty"`
	Config     *map[string]interface{} `json:"config,omitempty"`
	Position   *int                   `json:"position,omitempty"`
	Enabled    *bool                  `json:"enabled,omitempty"`
}

type CreateRevisionRequest struct {
	Summary   string `json:"summary,omitempty"`
	ChangeLog string `json:"change_log,omitempty"`
}

type ApprovalActionRequest struct {
	Status   ApprovalStatus `json:"status"`
	Comments string         `json:"comments,omitempty"`
}

// ============================================================
// Editorial Pipeline board (Sprint 5.12)
// ============================================================

type PipelineStage string

const (
	StageIdea        PipelineStage = "idea"
	StageResearch    PipelineStage = "research"
	StageOutline     PipelineStage = "outline"
	StageWriting     PipelineStage = "writing"
	StageSEO         PipelineStage = "seo"
	StageEEAT        PipelineStage = "eeat"
	StageTranslation PipelineStage = "translation"
	StageReview      PipelineStage = "review"
	StageApproval    PipelineStage = "approval"
	StageScheduled   PipelineStage = "scheduled"
	StagePublished   PipelineStage = "published"
)

// PipelineStageOrder is the canonical left-to-right column order of the board.
var PipelineStageOrder = []PipelineStage{
	StageIdea, StageResearch, StageOutline, StageWriting, StageSEO,
	StageEEAT, StageTranslation, StageReview, StageApproval, StageScheduled,
	StagePublished,
}

// PipelineItem is one card on the editorial board. Each row appears in exactly
// one stage: posts are assigned by (status, latest review decision, approval
// request) priority; engine tables (briefs, research, jobs) never overlap.
type PipelineItem struct {
	ID          uuid.UUID     `json:"id"`
	Title       string        `json:"title"`
	Slug        string        `json:"slug,omitempty"`
	Stage       PipelineStage `json:"stage"`
	Engine      string        `json:"engine"`
	EngineID    uuid.UUID     `json:"engine_id"`
	Language    string        `json:"language,omitempty"`
	CategoryID  *uuid.UUID    `json:"category_id,omitempty"`
	AuthorID    *uuid.UUID    `json:"author_id,omitempty"`
	SEOScore    *float64      `json:"seo_score,omitempty"`
	EEATScore   *float64      `json:"eeat_score,omitempty"`
	Status      string        `json:"status"`
	ScheduledAt *time.Time    `json:"scheduled_at,omitempty"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Actionable  bool          `json:"actionable"`
}

type StageCount struct {
	Stage PipelineStage `json:"stage"`
	Count int           `json:"count"`
}

type PipelineResponse struct {
	Items  []PipelineItem `json:"items"`
	Total  int            `json:"total"`
	Stages []StageCount   `json:"stages"`
}

type PipelineStats struct {
	StageCounts     []StageCount `json:"stage_counts"`
	TotalItems      int          `json:"total_items"`
	AvgSEOScore     *float64     `json:"avg_seo_score,omitempty"`
	AvgEEATScore    *float64     `json:"avg_eeat_score,omitempty"`
	PendingReviews  int          `json:"pending_reviews"`
	PendingApprovals int         `json:"pending_approvals"`
	InTranslation   int          `json:"in_translation"`
	PublishedWeek   int          `json:"published_this_week"`
}

// ============================================================
// Editorial review screen (Sprint 5.12)
// ============================================================

type ReviewPost struct {
	ID           uuid.UUID  `json:"id"`
	Title        string     `json:"title"`
	Slug         string     `json:"slug"`
	Status       string     `json:"status"`
	Language     string     `json:"language"`
	SEOScore     *float64   `json:"seo_score,omitempty"`
	SEOAnalyzedAt *time.Time `json:"seo_analyzed_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type ReviewScores struct {
	SEO         float64   `json:"seo"`
	EEAT        float64   `json:"eeat"`
	Freshness   float64   `json:"freshness"`
	Coverage    float64   `json:"coverage"`
	Naturalness float64   `json:"naturalness"`
	Confidence  float64   `json:"confidence"`
	Final       float64   `json:"final"`
	Decision    string    `json:"decision"`
	Threshold   float64   `json:"threshold"`
	CreatedAt   time.Time `json:"created_at"`
}

type ReviewSource struct {
	URL             string    `json:"url"`
	Title           string    `json:"title,omitempty"`
	Snippet         string    `json:"snippet,omitempty"`
	Language        string    `json:"language,omitempty"`
	IsVerified      bool      `json:"is_verified"`
	FreshnessScore  *float64  `json:"freshness_score,omitempty"`
	RelevanceScore  int       `json:"relevance_score,omitempty"`
	RetrievedAt     time.Time `json:"retrieved_at,omitempty"`
}

type ReviewLink struct {
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	AnchorText  string  `json:"anchor_text,omitempty"`
	Score       float64 `json:"score"`
	Label       string  `json:"label,omitempty"`
	Reliability int     `json:"reliability,omitempty"`
}

type ReviewProblem struct {
	Kind     string `json:"kind"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// ReviewRecommendation is one line of the deterministic "IA recomenda" block.
type ReviewRecommendation struct {
	Label   string    `json:"label"`
	Score   *float64  `json:"score,omitempty"`
	Status  string    `json:"status"` // ok | warning | fail | info
	Details []string  `json:"details,omitempty"`
}

type ReadinessCheck struct {
	Stage   string `json:"stage"`
	Label   string `json:"label"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type PublishReadiness struct {
	PostID   uuid.UUID        `json:"post_id"`
	Title    string           `json:"title"`
	Slug     string           `json:"slug"`
	Ready    bool             `json:"ready"`
	Blocking string           `json:"blocking,omitempty"`
	Checks   []ReadinessCheck `json:"checks"`
}

type ArticleReview struct {
	Post            ReviewPost           `json:"post"`
	Review          *ReviewScores        `json:"review,omitempty"`
	Sources         []ReviewSource       `json:"sources"`
	InternalLinks   []ReviewLink         `json:"internal_links"`
	ExternalLinks   []ReviewLink         `json:"external_links"`
	Problems        []ReviewProblem      `json:"problems"`
	Recommendations []ReviewRecommendation `json:"recommendations"`
	Readiness       *PublishReadiness    `json:"readiness,omitempty"`
}
