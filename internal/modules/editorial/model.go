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
