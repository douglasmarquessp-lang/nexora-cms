package workflow

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "workflow"

type JobStatus string

const (
	JobStatusDraft     JobStatus = "draft"
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusPaused    JobStatus = "paused"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
	StepStatusCancelled StepStatus = "cancelled"
)

type QueueStatus string

const (
	QueueStatusPending   QueueStatus = "pending"
	QueueStatusRunning   QueueStatus = "running"
	QueueStatusPaused    QueueStatus = "paused"
	QueueStatusCompleted QueueStatus = "completed"
	QueueStatusFailed    QueueStatus = "failed"
	QueueStatusCancelled QueueStatus = "cancelled"
)

type NotificationSeverity string

const (
	SeverityInfo     NotificationSeverity = "info"
	SeverityWarning  NotificationSeverity = "warning"
	SeverityError    NotificationSeverity = "error"
	SeverityCritical NotificationSeverity = "critical"
	SeveritySuccess  NotificationSeverity = "success"
)

type WorkflowStep string

const (
	StepResearch         WorkflowStep = "research"
	StepWriter           WorkflowStep = "writer"
	StepHumanWriter      WorkflowStep = "human_writer"
	StepEditorialEngine  WorkflowStep = "editorial_engine"
	StepSEOEngine        WorkflowStep = "seo_engine"
	StepQualityCheck     WorkflowStep = "quality_check"
	StepPublisher        WorkflowStep = "publisher"
	StepFinished         WorkflowStep = "finished"
)

var AllWorkflowSteps = []WorkflowStep{
	StepResearch, StepWriter, StepHumanWriter, StepEditorialEngine,
	StepSEOEngine, StepQualityCheck, StepPublisher, StepFinished,
}

var StepDependencies = map[WorkflowStep][]WorkflowStep{
	StepResearch:        {},
	StepWriter:          {StepResearch},
	StepHumanWriter:     {StepWriter},
	StepEditorialEngine: {StepHumanWriter},
	StepSEOEngine:       {StepEditorialEngine},
	StepQualityCheck:    {StepSEOEngine},
	StepPublisher:       {StepQualityCheck},
	StepFinished:        {StepPublisher},
}

var StepDisplayNames = map[WorkflowStep]string{
	StepResearch:        "Research",
	StepWriter:          "Writer",
	StepHumanWriter:     "Human Writer",
	StepEditorialEngine: "Editorial Engine",
	StepSEOEngine:       "SEO Engine",
	StepQualityCheck:    "Quality Check",
	StepPublisher:       "Publisher",
	StepFinished:        "Finished",
}

type WorkflowJob struct {
	ID             uuid.UUID  `json:"id"`
	SiteID         uuid.UUID  `json:"site_id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	Title          string     `json:"title"`
	ContentType    string     `json:"content_type"`
	Language       string     `json:"language"`
	TargetLanguage string     `json:"target_language,omitempty"`
	Status         JobStatus  `json:"status"`
	CurrentStep    string     `json:"current_step,omitempty"`
	Progress       float64    `json:"progress"`
	Priority       int        `json:"priority"`
	WordCount      int        `json:"word_count,omitempty"`
	Tone           string     `json:"tone,omitempty"`
	Audience       string     `json:"audience,omitempty"`
	Keywords       []string   `json:"keywords,omitempty"`
	StyleSlug      string     `json:"style_slug,omitempty"`
	SourceJobID    *uuid.UUID `json:"source_job_id,omitempty"`
	ScheduledFor   *time.Time `json:"scheduled_for,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	RetryCount     int        `json:"retry_count"`
	MaxRetries     int        `json:"max_retries"`
	GeneratePT     bool       `json:"generate_pt"`
	GenerateEN     bool       `json:"generate_en"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CancelledAt    *time.Time `json:"cancelled_at,omitempty"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Steps          []Step     `json:"steps,omitempty"`
}

type Step struct {
	ID            uuid.UUID              `json:"id"`
	WorkflowJobID uuid.UUID              `json:"workflow_job_id"`
	StepName      string                 `json:"step_name"`
	DisplayName   string                 `json:"display_name,omitempty"`
	Status        StepStatus             `json:"status"`
	Progress      float64                `json:"progress"`
	DependsOn     []string               `json:"depends_on,omitempty"`
	RetryCount    int                    `json:"retry_count"`
	MaxRetries    int                    `json:"max_retries"`
	StartedAt     *time.Time             `json:"started_at,omitempty"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	DurationMs    int64                  `json:"duration_ms,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type QueueItem struct {
	ID               uuid.UUID  `json:"id"`
	SiteID           uuid.UUID  `json:"site_id"`
	WorkflowJobID    *uuid.UUID `json:"workflow_job_id,omitempty"`
	Title            string     `json:"title"`
	Content          string     `json:"content,omitempty"`
	Excerpt          string     `json:"excerpt,omitempty"`
	Language         string     `json:"language"`
	Status           QueueStatus `json:"status"`
	Priority         int        `json:"priority"`
	ScheduledFor     *time.Time `json:"scheduled_for,omitempty"`
	IsPaused         bool       `json:"is_paused"`
	RetryCount       int        `json:"retry_count"`
	MaxRetries       int        `json:"max_retries"`
	MetaTitle        string     `json:"meta_title,omitempty"`
	MetaDescription  string     `json:"meta_description,omitempty"`
	Slug             string     `json:"slug,omitempty"`
	FeaturedImageURL string     `json:"featured_image_url,omitempty"`
	Tags             []string   `json:"tags,omitempty"`
	Categories       []string   `json:"categories,omitempty"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	PublishedBy      *uuid.UUID `json:"published_by,omitempty"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type HistoryEntry struct {
	ID             uuid.UUID              `json:"id"`
	SiteID         uuid.UUID              `json:"site_id"`
	WorkflowJobID  *uuid.UUID             `json:"workflow_job_id,omitempty"`
	QueueID        *uuid.UUID             `json:"queue_id,omitempty"`
	Action         string                 `json:"action"`
	EntityType     string                 `json:"entity_type"`
	EntityID       *uuid.UUID             `json:"entity_id,omitempty"`
	PreviousStatus string                 `json:"previous_status,omitempty"`
	NewStatus      string                 `json:"new_status,omitempty"`
	Details        map[string]interface{} `json:"details,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	UserID         *uuid.UUID             `json:"user_id,omitempty"`
	DurationMs     int64                  `json:"duration_ms,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

type Notification struct {
	ID               uuid.UUID  `json:"id"`
	SiteID           uuid.UUID  `json:"site_id"`
	WorkflowJobID    *uuid.UUID `json:"workflow_job_id,omitempty"`
	QueueID          *uuid.UUID `json:"queue_id,omitempty"`
	NotificationType string     `json:"notification_type"`
	Title            string     `json:"title"`
	Message          string     `json:"message,omitempty"`
	Severity         string     `json:"severity"`
	Read             bool       `json:"read"`
	ActionURL        string     `json:"action_url,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type Dashboard struct {
	ID                   uuid.UUID              `json:"id"`
	SiteID               uuid.UUID              `json:"site_id"`
	TotalJobs            int64                  `json:"total_jobs"`
	RunningJobs          int64                  `json:"running_jobs"`
	CompletedJobs        int64                  `json:"completed_jobs"`
	FailedJobs           int64                  `json:"failed_jobs"`
	PausedJobs           int64                  `json:"paused_jobs"`
	QueueSize            int64                  `json:"queue_size"`
	StalledQueue         int64                  `json:"stalled_queue"`
	PendingReview        int64                  `json:"pending_review"`
	ScheduledPublications int64                 `json:"scheduled_publications"`
	RecentPublications   int64                  `json:"recent_publications"`
	AvgExecutionMs       float64                `json:"avg_execution_ms"`
	SuccessRate          float64                `json:"success_rate"`
	FailureRate          float64                `json:"failure_rate"`
	ThroughputHourly     float64                `json:"throughput_hourly"`
	WorkerUtilization    float64                `json:"worker_utilization"`
	Data                 map[string]interface{} `json:"data,omitempty"`
	SnapshotAt           time.Time              `json:"snapshot_at"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

type WorkflowMetrics struct {
	TotalJobs       int64   `json:"total_jobs"`
	RunningJobs     int64   `json:"running_jobs"`
	CompletedJobs   int64   `json:"completed_jobs"`
	FailedJobs      int64   `json:"failed_jobs"`
	PausedJobs      int64   `json:"paused_jobs"`
	AvgDuration     float64 `json:"avg_duration_ms"`
	AvgSuccessRate  float64 `json:"avg_success_rate"`
	AvgFailureRate  float64 `json:"avg_failure_rate"`
	QueueSize       int64   `json:"queue_size"`
	StalledCount    int64   `json:"stalled_count"`
	PendingReview   int64   `json:"pending_review"`
	ScheduledCount  int64   `json:"scheduled_count"`
	Throughput      float64 `json:"throughput_hourly"`
	WorkerUtil      float64 `json:"worker_utilization"`
	NotificationCnt int64   `json:"notification_count"`
}

type StageDuration struct {
	StepName    string  `json:"step_name"`
	DisplayName string  `json:"display_name"`
	AvgDuration float64 `json:"avg_duration_ms"`
	Count       int64   `json:"count"`
}

type WorkflowStats struct {
	ByStatus   map[string]int64 `json:"by_status"`
	ByLanguage map[string]int64 `json:"by_language"`
	ByStep     map[string]int64 `json:"by_step"`
	Stages     []StageDuration  `json:"stages,omitempty"`
}

// --- DTOs ---

type CreateJobRequest struct {
	Title          string     `json:"title"`
	ContentType    string     `json:"content_type,omitempty"`
	Language       string     `json:"language,omitempty"`
	TargetLanguage string     `json:"target_language,omitempty"`
	Priority       int        `json:"priority,omitempty"`
	WordCount      int        `json:"word_count,omitempty"`
	Tone           string     `json:"tone,omitempty"`
	Audience       string     `json:"audience,omitempty"`
	Keywords       []string   `json:"keywords,omitempty"`
	StyleSlug      string     `json:"style_slug,omitempty"`
	SourceJobID    *uuid.UUID `json:"source_job_id,omitempty"`
	ScheduledFor   *time.Time `json:"scheduled_for,omitempty"`
	GeneratePT     bool       `json:"generate_pt"`
	GenerateEN     bool       `json:"generate_en"`
}

type UpdateJobRequest struct {
	Title          *string     `json:"title,omitempty"`
	ContentType    *string     `json:"content_type,omitempty"`
	TargetLanguage *string     `json:"target_language,omitempty"`
	Priority       *int        `json:"priority,omitempty"`
	WordCount      *int        `json:"word_count,omitempty"`
	Tone           *string     `json:"tone,omitempty"`
	Audience       *string     `json:"audience,omitempty"`
	Keywords       *[]string   `json:"keywords,omitempty"`
	StyleSlug      *string     `json:"style_slug,omitempty"`
	ScheduledFor   *time.Time  `json:"scheduled_for,omitempty"`
	GeneratePT     *bool       `json:"generate_pt,omitempty"`
	GenerateEN     *bool       `json:"generate_en,omitempty"`
}

type QueueRequest struct {
	WorkflowJobID    *uuid.UUID `json:"workflow_job_id,omitempty"`
	Title            string     `json:"title"`
	Content          string     `json:"content,omitempty"`
	Excerpt          string     `json:"excerpt,omitempty"`
	Language         string     `json:"language"`
	Priority         int        `json:"priority,omitempty"`
	ScheduledFor     *time.Time `json:"scheduled_for,omitempty"`
	MetaTitle        string     `json:"meta_title,omitempty"`
	MetaDescription  string     `json:"meta_description,omitempty"`
	Slug             string     `json:"slug,omitempty"`
	FeaturedImageURL string     `json:"featured_image_url,omitempty"`
	Tags             []string   `json:"tags,omitempty"`
	Categories       []string   `json:"categories,omitempty"`
}

type UpdateQueueRequest struct {
	Status           *string    `json:"status,omitempty"`
	Priority         *int       `json:"priority,omitempty"`
	ScheduledFor     *time.Time `json:"scheduled_for,omitempty"`
	IsPaused         *bool      `json:"is_paused,omitempty"`
	MetaTitle        *string    `json:"meta_title,omitempty"`
	MetaDescription  *string    `json:"meta_description,omitempty"`
	Slug             *string    `json:"slug,omitempty"`
	FeaturedImageURL *string    `json:"featured_image_url,omitempty"`
}

type AutomationAction struct {
	Action string `json:"action"`
	JobID  string `json:"job_id,omitempty"`
	Title  string `json:"title,omitempty"`
}

type RetryRequest struct {
	StepName string `json:"step_name"`
}

type NotificationList struct {
	Notifications []Notification `json:"notifications"`
	Total         int64          `json:"total"`
	Unread        int64          `json:"unread"`
}

// --- Events ---

const (
	EventWorkflowCreated          kernel.EventType = "workflow.created"
	EventWorkflowStarted          kernel.EventType = "workflow.started"
	EventWorkflowProgress         kernel.EventType = "workflow.progress"
	EventWorkflowPaused           kernel.EventType = "workflow.paused"
	EventWorkflowResumed          kernel.EventType = "workflow.resumed"
	EventWorkflowCompleted        kernel.EventType = "workflow.completed"
	EventWorkflowFailed           kernel.EventType = "workflow.failed"
	EventWorkflowCancelled        kernel.EventType = "workflow.cancelled"
	EventWorkflowRetry            kernel.EventType = "workflow.retry"
	EventWorkflowStepStarted      kernel.EventType = "workflow.step.started"
	EventWorkflowStepCompleted    kernel.EventType = "workflow.step.completed"
	EventWorkflowStepFailed       kernel.EventType = "workflow.step.failed"
	EventWorkflowQueued           kernel.EventType = "workflow.queued"
	EventWorkflowQueueProcessed   kernel.EventType = "workflow.queue.processed"
	EventWorkflowQueueStalled     kernel.EventType = "workflow.queue.stalled"
	EventWorkflowPublicationReady kernel.EventType = "workflow.publication.ready"
	EventWorkflowQualityFailed    kernel.EventType = "workflow.quality.failed"
	EventWorkflowSEOFailed        kernel.EventType = "workflow.seo.failed"
)

// --- Errors ---

var (
	ErrJobNotFound          = errors.New("workflow job not found")
	ErrJobAlreadyRunning    = errors.New("job is already running")
	ErrJobAlreadyCompleted  = errors.New("job is already completed")
	ErrJobAlreadyCancelled  = errors.New("job is already cancelled")
	ErrJobNotRunning        = errors.New("job is not running")
	ErrJobPaused            = errors.New("job is paused")
	ErrStepNotFound         = errors.New("step not found")
	ErrStepAlreadyCompleted = errors.New("step already completed")
	ErrDependencyFailed     = errors.New("dependency step failed")
	ErrDependencyPending    = errors.New("dependency step not completed")
	ErrInvalidTitle         = errors.New("title is required")
	ErrInvalidLanguage      = errors.New("language must be 'pt' or 'en'")
	ErrInvalidPriority      = errors.New("priority must be 1-10")
	ErrDatabaseNotAvail     = errors.New("database not available")
	ErrMaxRetriesExceeded   = errors.New("maximum retries exceeded")
	ErrQueueItemNotFound    = errors.New("queue item not found")
	ErrQueueItemPaused      = errors.New("queue item is paused")
	ErrQueueItemRunning     = errors.New("queue item is already running")
	ErrNotificationNotFound = errors.New("notification not found")
	ErrInvalidAction        = errors.New("invalid automation action")
	ErrInvalidStep          = errors.New("invalid workflow step")
)
