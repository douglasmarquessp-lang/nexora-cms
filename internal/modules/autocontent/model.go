package autocontent

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "autocontent"

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
	QueuePending   QueueStatus = "pending"
	QueueApproved  QueueStatus = "approved"
	QueuePublished QueueStatus = "published"
	QueueFailed    QueueStatus = "failed"
	QueueRejected  QueueStatus = "rejected"
)

type WorkflowStep string

const (
	StepTopic             WorkflowStep = "topic"
	StepResearch          WorkflowStep = "research"
	StepBriefing          WorkflowStep = "briefing"
	StepOutline           WorkflowStep = "outline"
	StepDraft             WorkflowStep = "draft"
	StepHumanRewrite      WorkflowStep = "human_rewrite"
	StepSEOOptimization   WorkflowStep = "seo_optimization"
	StepFactCheck         WorkflowStep = "fact_check"
	StepReadability       WorkflowStep = "readability"
	StepInternalLinking   WorkflowStep = "internal_linking"
	StepMetadata          WorkflowStep = "metadata"
	StepTranslation       WorkflowStep = "translation"
	StepFeaturedImage     WorkflowStep = "featured_image"
	StepReadyForPub       WorkflowStep = "ready_for_publication"
)

var AllWorkflowSteps = []WorkflowStep{
	StepTopic, StepResearch, StepBriefing, StepOutline, StepDraft,
	StepHumanRewrite, StepSEOOptimization, StepFactCheck, StepReadability,
	StepInternalLinking, StepMetadata, StepTranslation, StepFeaturedImage,
	StepReadyForPub,
}

var StepDependencies = map[WorkflowStep][]WorkflowStep{
	StepTopic:           {},
	StepResearch:        {StepTopic},
	StepBriefing:        {StepResearch},
	StepOutline:         {StepBriefing},
	StepDraft:           {StepOutline},
	StepHumanRewrite:    {StepDraft},
	StepSEOOptimization: {StepHumanRewrite},
	StepFactCheck:       {StepSEOOptimization},
	StepReadability:     {StepFactCheck},
	StepInternalLinking: {StepReadability},
	StepMetadata:        {StepInternalLinking},
	StepTranslation:     {StepMetadata},
	StepFeaturedImage:   {StepTranslation},
	StepReadyForPub:     {StepFeaturedImage},
}

var StepDisplayNames = map[WorkflowStep]string{
	StepTopic:           "Topic Definition",
	StepResearch:        "Research",
	StepBriefing:        "Briefing Generation",
	StepOutline:         "Outline Generation",
	StepDraft:           "Draft Generation",
	StepHumanRewrite:    "Human Rewrite",
	StepSEOOptimization: "SEO Optimization",
	StepFactCheck:       "Fact Check",
	StepReadability:     "Readability Analysis",
	StepInternalLinking: "Internal Linking",
	StepMetadata:        "Metadata Generation",
	StepTranslation:     "Translation PT/EN",
	StepFeaturedImage:   "Featured Image Metadata",
	StepReadyForPub:     "Ready for Publication",
}

type AutocontentJob struct {
	ID              uuid.UUID  `json:"id"`
	SiteID          uuid.UUID  `json:"site_id"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`
	Topic           string     `json:"topic"`
	Title           string     `json:"title,omitempty"`
	ContentType     string     `json:"content_type,omitempty"`
	Language        string     `json:"language"`
	TargetLanguage  string     `json:"target_language,omitempty"`
	Status          JobStatus  `json:"status"`
	CurrentStep     string     `json:"current_step,omitempty"`
	Progress        float64    `json:"progress"`
	Priority        int        `json:"priority"`
	WordCount       int        `json:"word_count,omitempty"`
	Tone            string     `json:"tone,omitempty"`
	Audience        string     `json:"audience,omitempty"`
	Keywords        []string   `json:"keywords,omitempty"`
	StyleSlug       string     `json:"style_slug,omitempty"`
	TemplateID      *uuid.UUID `json:"template_id,omitempty"`
	ScheduledFor    *time.Time `json:"scheduled_for,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	RetryCount      int        `json:"retry_count"`
	MaxRetries      int        `json:"max_retries"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Steps           []Step     `json:"steps,omitempty"`
	Results         []Result   `json:"results,omitempty"`
}

type Step struct {
	ID              uuid.UUID              `json:"id"`
	AutocontentJobID uuid.UUID             `json:"autocontent_job_id"`
	StepName        string                 `json:"step_name"`
	DisplayName     string                 `json:"display_name,omitempty"`
	Status          StepStatus             `json:"status"`
	Progress        float64                `json:"progress"`
	DependsOn       []string               `json:"depends_on,omitempty"`
	RetryCount      int                    `json:"retry_count"`
	MaxRetries      int                    `json:"max_retries"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	DurationMs      int64                  `json:"duration_ms,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type Result struct {
	ID              uuid.UUID              `json:"id"`
	AutocontentJobID uuid.UUID             `json:"autocontent_job_id"`
	StepName        string                 `json:"step_name"`
	Content         string                 `json:"content,omitempty"`
	Summary         string                 `json:"summary,omitempty"`
	Score           float64                `json:"score"`
	Passed          bool                   `json:"passed"`
	Data            map[string]interface{} `json:"data,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type PublicationItem struct {
	ID              uuid.UUID  `json:"id"`
	SiteID          uuid.UUID  `json:"site_id"`
	AutocontentJobID *uuid.UUID `json:"autocontent_job_id,omitempty"`
	Title           string     `json:"title"`
	Content         string     `json:"content,omitempty"`
	Excerpt         string     `json:"excerpt,omitempty"`
	Language        string     `json:"language"`
	Status          QueueStatus `json:"status"`
	Priority        int        `json:"priority"`
	ScheduledFor    *time.Time `json:"scheduled_for,omitempty"`
	MetaTitle       string     `json:"meta_title,omitempty"`
	MetaDescription string     `json:"meta_description,omitempty"`
	Slug            string     `json:"slug,omitempty"`
	FeaturedImageURL string    `json:"featured_image_url,omitempty"`
	Tags            []string   `json:"tags,omitempty"`
	Categories      []string   `json:"categories,omitempty"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	PublishedBy     *uuid.UUID `json:"published_by,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type WorkflowTemplate struct {
	ID          uuid.UUID                `json:"id"`
	SiteID      uuid.UUID                `json:"site_id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Steps       []map[string]interface{} `json:"steps"`
	IsDefault   bool                     `json:"is_default"`
	IsActive    bool                     `json:"is_active"`
	CreatedBy   *uuid.UUID               `json:"created_by,omitempty"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

// --- DTOs ---

type CreateJobRequest struct {
	Topic          string   `json:"topic"`
	Title          string   `json:"title,omitempty"`
	ContentType    string   `json:"content_type,omitempty"`
	Language       string   `json:"language,omitempty"`
	TargetLanguage string   `json:"target_language,omitempty"`
	Priority       int      `json:"priority,omitempty"`
	WordCount      int      `json:"word_count,omitempty"`
	Tone           string   `json:"tone,omitempty"`
	Audience       string   `json:"audience,omitempty"`
	Keywords       []string `json:"keywords,omitempty"`
	StyleSlug      string   `json:"style_slug,omitempty"`
	TemplateID     *uuid.UUID `json:"template_id,omitempty"`
	ScheduledFor   *time.Time `json:"scheduled_for,omitempty"`
}

type UpdateJobRequest struct {
	Title          *string   `json:"title,omitempty"`
	ContentType    *string   `json:"content_type,omitempty"`
	TargetLanguage *string   `json:"target_language,omitempty"`
	Priority       *int      `json:"priority,omitempty"`
	WordCount      *int      `json:"word_count,omitempty"`
	Tone           *string   `json:"tone,omitempty"`
	Audience       *string   `json:"audience,omitempty"`
	Keywords       *[]string `json:"keywords,omitempty"`
	StyleSlug      *string   `json:"style_slug,omitempty"`
	ScheduledFor   *time.Time `json:"scheduled_for,omitempty"`
}

type RetryStepRequest struct {
	StepName string `json:"step_name"`
}

type QueueRequest struct {
	AutocontentJobID *uuid.UUID `json:"autocontent_job_id,omitempty"`
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
	Status           *QueueStatus `json:"status,omitempty"`
	Priority         *int         `json:"priority,omitempty"`
	ScheduledFor     *time.Time   `json:"scheduled_for,omitempty"`
	MetaTitle        *string      `json:"meta_title,omitempty"`
	MetaDescription  *string      `json:"meta_description,omitempty"`
	Slug             *string      `json:"slug,omitempty"`
	FeaturedImageURL *string      `json:"featured_image_url,omitempty"`
}

type CreateTemplateRequest struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Steps       []map[string]interface{} `json:"steps"`
	IsDefault   bool                     `json:"is_default"`
}

type AutocontentMetrics struct {
	TotalJobs       int64   `json:"total_jobs"`
	RunningJobs     int64   `json:"running_jobs"`
	CompletedJobs   int64   `json:"completed_jobs"`
	FailedJobs      int64   `json:"failed_jobs"`
	PausedJobs      int64   `json:"paused_jobs"`
	AvgDuration     float64 `json:"avg_duration_ms"`
	AvgSuccessRate  float64 `json:"avg_success_rate"`
	QueueSize       int     `json:"queue_size"`
	TemplateCount   int     `json:"template_count"`
}

type AutocontentStats struct {
	ByStatus   map[JobStatus]int64         `json:"by_status"`
	ByLanguage map[string]int64            `json:"by_language"`
	ByStep     map[string]int64            `json:"by_step"`
	Daily      []DailyStats               `json:"daily,omitempty"`
}

type DailyStats struct {
	Date    string `json:"date"`
	Created int64  `json:"created"`
	Completed int64 `json:"completed"`
	Failed  int64  `json:"failed"`
}

// --- Events ---

const (
	EventAutoCreated    kernel.EventType = "autocontent.created"
	EventAutoStarted    kernel.EventType = "autocontent.started"
	EventAutoProgress   kernel.EventType = "autocontent.progress"
	EventAutoPaused     kernel.EventType = "autocontent.paused"
	EventAutoResumed    kernel.EventType = "autocontent.resumed"
	EventAutoCompleted  kernel.EventType = "autocontent.completed"
	EventAutoFailed     kernel.EventType = "autocontent.failed"
	EventAutoCancelled  kernel.EventType = "autocontent.cancelled"
	EventAutoRetry      kernel.EventType = "autocontent.retry"
	EventAutoStepStarted kernel.EventType = "autocontent.step.started"
	EventAutoStepCompleted kernel.EventType = "autocontent.step.completed"
	EventAutoStepFailed kernel.EventType = "autocontent.step.failed"
	EventAutoQueued     kernel.EventType = "autocontent.queued"
)

// --- Errors ---

var (
	ErrJobNotFound          = errors.New("autocontent job not found")
	ErrJobAlreadyRunning    = errors.New("job is already running")
	ErrJobAlreadyCompleted  = errors.New("job is already completed")
	ErrJobAlreadyCancelled  = errors.New("job is already cancelled")
	ErrJobNotRunning        = errors.New("job is not running")
	ErrStepNotFound         = errors.New("step not found")
	ErrStepAlreadyCompleted = errors.New("step already completed")
	ErrDependencyFailed     = errors.New("dependency step failed")
	ErrDependencyPending    = errors.New("dependency step not completed")
	ErrInvalidTopic         = errors.New("topic is required")
	ErrDatabaseNotAvail     = errors.New("database not available")
	ErrMaxRetriesExceeded   = errors.New("maximum retries exceeded")
	ErrQueueItemNotFound    = errors.New("queue item not found")
	ErrTemplateNotFound     = errors.New("workflow template not found")
	ErrResultNotFound       = errors.New("result not found")
	ErrInvalidLanguage      = errors.New("language must be 'pt' or 'en'")
	ErrInvalidStep          = errors.New("invalid workflow step")
	ErrJobPaused            = errors.New("job is paused")
)
