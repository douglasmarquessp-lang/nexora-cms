package articlepipeline

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "articlepipeline"

type StageName string

const (
	StageResearch        StageName = "research"
	StageOutline         StageName = "outline"
	StageDraft           StageName = "draft"
	StageHumanRewrite    StageName = "human_rewrite"
	StageSEOOptimization StageName = "seo_optimization"
	StageReadability     StageName = "readability"
	StageInternalLinking StageName = "internal_linking"
	StageMetadata        StageName = "metadata"
	StageTranslation     StageName = "translation"
	StageQualityScore    StageName = "quality_score"
	StagePubCandidate    StageName = "publication_candidate"
)

var AllStages = []StageName{
	StageResearch, StageOutline, StageDraft, StageHumanRewrite,
	StageSEOOptimization, StageReadability, StageInternalLinking,
	StageMetadata, StageTranslation, StageQualityScore, StagePubCandidate,
}

var StageDisplayNames = map[StageName]string{
	StageResearch:        "Research",
	StageOutline:         "Outline",
	StageDraft:           "Draft",
	StageHumanRewrite:    "Human Rewrite",
	StageSEOOptimization: "SEO Optimization",
	StageReadability:     "Readability",
	StageInternalLinking: "Internal Linking",
	StageMetadata:        "Metadata",
	StageTranslation:     "Translation",
	StageQualityScore:    "Quality Score",
	StagePubCandidate:    "Publication Candidate",
}

var StageDependencies = map[StageName][]StageName{
	StageResearch:        {},
	StageOutline:         {StageResearch},
	StageDraft:           {StageOutline},
	StageHumanRewrite:    {StageDraft},
	StageSEOOptimization: {StageHumanRewrite},
	StageReadability:     {StageSEOOptimization},
	StageInternalLinking: {StageReadability},
	StageMetadata:        {StageInternalLinking},
	StageTranslation:     {StageMetadata},
	StageQualityScore:    {StageTranslation},
	StagePubCandidate:    {StageQualityScore},
}

type PipelineStatus string

const (
	PipelineDraft     PipelineStatus = "draft"
	PipelinePending   PipelineStatus = "pending"
	PipelineRunning   PipelineStatus = "running"
	PipelinePaused    PipelineStatus = "paused"
	PipelineCompleted PipelineStatus = "completed"
	PipelineFailed    PipelineStatus = "failed"
	PipelineCancelled PipelineStatus = "cancelled"
	PipelineRetrying  PipelineStatus = "retrying"
)

var AllPipelineStatuses = []PipelineStatus{
	PipelineDraft, PipelinePending, PipelineRunning, PipelinePaused,
	PipelineCompleted, PipelineFailed, PipelineCancelled, PipelineRetrying,
}

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
	StepStatusCancelled StepStatus = "cancelled"
)

var AllStepStatuses = []StepStatus{
	StepStatusPending, StepStatusRunning, StepStatusCompleted,
	StepStatusFailed, StepStatusSkipped, StepStatusCancelled,
}

type QualityStatus string

const (
	QualityPending QualityStatus = "pending"
	QualityPassed  QualityStatus = "passed"
	QualityFailed  QualityStatus = "failed"
	QualityWarning QualityStatus = "warning"
)

type CandidateStatus string

const (
	CandidateDraft     CandidateStatus = "draft"
	CandidateApproved  CandidateStatus = "approved"
	CandidatePublished CandidateStatus = "published"
	CandidateRejected  CandidateStatus = "rejected"
)

type PipelineJob struct {
	ID              uuid.UUID       `json:"id"`
	SiteID          uuid.UUID       `json:"site_id"`
	Title           string          `json:"title"`
	Topic           string          `json:"topic,omitempty"`
	SourceContent   string          `json:"source_content,omitempty"`
	Language        string          `json:"language"`
	TargetLanguage  string          `json:"target_language,omitempty"`
	ContentType     string          `json:"content_type,omitempty"`
	Status          PipelineStatus  `json:"status"`
	Progress        float64         `json:"progress"`
	CurrentStage    string          `json:"current_stage,omitempty"`
	Priority        int             `json:"priority"`
	RetryCount      int             `json:"retry_count"`
	MaxRetries      int             `json:"max_retries"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	CancelledAt     *time.Time      `json:"cancelled_at,omitempty"`
	CreatedBy       *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Steps           []Step          `json:"steps,omitempty"`
	Metrics         []Metric        `json:"metrics,omitempty"`
	QualityReports  []QualityReport `json:"quality_reports,omitempty"`
}

type Step struct {
	ID            uuid.UUID              `json:"id"`
	PipelineJobID uuid.UUID              `json:"pipeline_job_id"`
	StageName     string                 `json:"stage_name"`
	DisplayName   string                 `json:"display_name,omitempty"`
	Status        StepStatus             `json:"status"`
	Progress      float64                `json:"progress"`
	StartedAt     *time.Time             `json:"started_at,omitempty"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	DurationMs    int64                  `json:"duration_ms,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	RetryCount    int                    `json:"retry_count"`
	MaxRetries    int                    `json:"max_retries"`
	Output        string                 `json:"output,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type Metric struct {
	ID           uuid.UUID              `json:"id"`
	PipelineJobID uuid.UUID             `json:"pipeline_job_id"`
	StageName    string                 `json:"stage_name,omitempty"`
	MetricName   string                 `json:"metric_name"`
	MetricValue  float64                `json:"metric_value"`
	MetricUnit   string                 `json:"metric_unit,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	RecordedAt   time.Time              `json:"recorded_at"`
}

type QualityReport struct {
	ID           uuid.UUID              `json:"id"`
	PipelineJobID uuid.UUID             `json:"pipeline_job_id"`
	StageName    string                 `json:"stage_name"`
	Status       QualityStatus          `json:"status"`
	Score        float64                `json:"score"`
	ChecksPassed int                    `json:"checks_passed"`
	ChecksFailed int                    `json:"checks_failed"`
	ChecksTotal  int                    `json:"checks_total"`
	Details      []QualityCheck         `json:"details,omitempty"`
	Summary      string                 `json:"summary,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

type QualityCheck struct {
	Name   string  `json:"name"`
	Passed bool    `json:"passed"`
	Score  float64 `json:"score,omitempty"`
	Detail string  `json:"detail,omitempty"`
}

type PublicationCandidate struct {
	ID              uuid.UUID              `json:"id"`
	PipelineJobID   uuid.UUID              `json:"pipeline_job_id"`
	SiteID          uuid.UUID              `json:"site_id"`
	Title           string                 `json:"title"`
	Content         string                 `json:"content,omitempty"`
	Excerpt         string                 `json:"excerpt,omitempty"`
	Language        string                 `json:"language"`
	Status          CandidateStatus        `json:"status"`
	QualityScore    float64                `json:"quality_score"`
	SEOScore        float64                `json:"seo_score"`
	ReadabilityScore float64               `json:"readability_score"`
	WordCount       int                    `json:"word_count"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type PipelineStats struct {
	TotalJobs       int64   `json:"total_jobs"`
	RunningJobs     int64   `json:"running_jobs"`
	CompletedJobs   int64   `json:"completed_jobs"`
	FailedJobs      int64   `json:"failed_jobs"`
	CancelledJobs   int64   `json:"cancelled_jobs"`
	AvgDurationMs   float64 `json:"avg_duration_ms"`
	AvgQualityScore float64 `json:"avg_quality_score"`
	AvgSEOScore     float64 `json:"avg_seo_score"`
	TotalCandidates int64   `json:"total_candidates"`
}

type CreatePipelineRequest struct {
	Title          string  `json:"title"`
	Topic          string  `json:"topic,omitempty"`
	SourceContent  string  `json:"source_content,omitempty"`
	Language       string  `json:"language,omitempty"`
	TargetLanguage string  `json:"target_language,omitempty"`
	ContentType    string  `json:"content_type,omitempty"`
	Priority       *int    `json:"priority,omitempty"`
}

type UpdatePipelineRequest struct {
	Title          *string `json:"title,omitempty"`
	Topic          *string `json:"topic,omitempty"`
	SourceContent  *string `json:"source_content,omitempty"`
	TargetLanguage *string `json:"target_language,omitempty"`
	ContentType    *string `json:"content_type,omitempty"`
	Priority       *int    `json:"priority,omitempty"`
}

type UpdateStageRequest struct {
	Status       StepStatus               `json:"status"`
	Progress     *float64                 `json:"progress,omitempty"`
	ErrorMessage string                   `json:"error_message,omitempty"`
	Output       string                   `json:"output,omitempty"`
	Metadata     map[string]interface{}   `json:"metadata,omitempty"`
}

type CreateMetricRequest struct {
	StageName   string                 `json:"stage_name,omitempty"`
	MetricName  string                 `json:"metric_name"`
	MetricValue float64                `json:"metric_value"`
	MetricUnit  string                 `json:"metric_unit,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type CreateQualityReportRequest struct {
	StageName    string         `json:"stage_name"`
	Score        float64        `json:"score"`
	ChecksPassed int            `json:"checks_passed"`
	ChecksFailed int            `json:"checks_failed"`
	ChecksTotal  int            `json:"checks_total"`
	Details      []QualityCheck `json:"details,omitempty"`
	Summary      string         `json:"summary,omitempty"`
}

type CreateCandidateRequest struct {
	Title           string                 `json:"title"`
	Content         string                 `json:"content,omitempty"`
	Excerpt         string                 `json:"excerpt,omitempty"`
	QualityScore    float64                `json:"quality_score"`
	SEOScore        float64                `json:"seo_score"`
	ReadabilityScore float64               `json:"readability_score"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

const (
	EventPipelineCreated   kernel.EventType = "articlepipeline.created"
	EventPipelineStarted   kernel.EventType = "articlepipeline.started"
	EventPipelineProgress  kernel.EventType = "articlepipeline.progress"
	EventPipelinePaused    kernel.EventType = "articlepipeline.paused"
	EventPipelineResumed   kernel.EventType = "articlepipeline.resumed"
	EventPipelineCompleted kernel.EventType = "articlepipeline.completed"
	EventPipelineFailed    kernel.EventType = "articlepipeline.failed"
	EventPipelineCancelled kernel.EventType = "articlepipeline.cancelled"
	EventPipelineRetry     kernel.EventType = "articlepipeline.retry"
	EventPipelineRestarted kernel.EventType = "articlepipeline.restarted"
	EventStageStarted      kernel.EventType = "articlepipeline.stage.started"
	EventStageCompleted    kernel.EventType = "articlepipeline.stage.completed"
	EventStageFailed       kernel.EventType = "articlepipeline.stage.failed"
	EventQualityPassed     kernel.EventType = "articlepipeline.quality.passed"
	EventQualityFailed     kernel.EventType = "articlepipeline.quality.failed"
	EventCandidateCreated  kernel.EventType = "articlepipeline.candidate.created"
)

var (
	ErrJobNotFound          = errors.New("pipeline job not found")
	ErrStageNotFound        = errors.New("pipeline stage not found")
	ErrJobAlreadyRunning    = errors.New("pipeline job is already running")
	ErrJobAlreadyCompleted  = errors.New("pipeline job is already completed")
	ErrJobAlreadyCancelled  = errors.New("pipeline job is already cancelled")
	ErrJobNotRunning        = errors.New("pipeline job is not running")
	ErrJobNotPaused         = errors.New("pipeline job is not paused")
	ErrStageNotPending      = errors.New("pipeline stage is not pending")
	ErrStageAlreadyCompleted = errors.New("pipeline stage already completed")
	ErrInvalidTitle         = errors.New("title is required")
	ErrInvalidLanguage      = errors.New("language must be 'pt' or 'en'")
	ErrInvalidPriority      = errors.New("priority must be between 1 and 10")
	ErrDatabaseNotAvail     = errors.New("database not available")
	ErrDependencyFailed     = errors.New("dependency stage failed")
	ErrDependencyPending    = errors.New("dependency stage not completed")
	ErrMaxRetriesExceeded   = errors.New("maximum retries exceeded")
	ErrCandidateNotFound    = errors.New("publication candidate not found")
	ErrMetricNotFound       = errors.New("metric not found")
	ErrQualityReportNotFound = errors.New("quality report not found")
)
