package contentgenerator

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "contentgenerator"

type GenStage string

const (
	GenStageResearch         GenStage = "research"
	GenStageBriefing         GenStage = "briefing"
	GenStageOutline          GenStage = "outline"
	GenStageSectionGen       GenStage = "section_generation"
	GenStageSEOOptimization  GenStage = "seo_optimization"
	GenStageQualityReview    GenStage = "quality_review"
	GenStageTranslation      GenStage = "translation"
	GenStageFinalReview      GenStage = "final_review"
	GenStagePublishReady     GenStage = "publish_ready"
)

var ValidStages = []GenStage{
	GenStageResearch, GenStageBriefing, GenStageOutline, GenStageSectionGen,
	GenStageSEOOptimization, GenStageQualityReview, GenStageTranslation,
	GenStageFinalReview, GenStagePublishReady,
}

type GenStatus string

const (
	GenStatusPending   GenStatus = "pending"
	GenStatusRunning   GenStatus = "running"
	GenStatusPaused    GenStatus = "paused"
	GenStatusCompleted GenStatus = "completed"
	GenStatusFailed    GenStatus = "failed"
	GenStatusCancelled GenStatus = "cancelled"
	GenStatusRetrying  GenStatus = "retrying"
)

type StageStatus string

const (
	StageStatusPending   StageStatus = "pending"
	StageStatusRunning   StageStatus = "running"
	StageStatusCompleted StageStatus = "completed"
	StageStatusFailed    StageStatus = "failed"
	StageStatusSkipped   StageStatus = "skipped"
)

type LogLevel string

const (
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelDebug   LogLevel = "debug"
)

const (
	EventGenStarted    kernel.EventType = "generation.started"
	EventGenProgress   kernel.EventType = "generation.progress"
	EventGenCompleted  kernel.EventType = "generation.completed"
	EventGenFailed     kernel.EventType = "generation.failed"
	EventGenRetry      kernel.EventType = "generation.retry"
	EventGenCancelled  kernel.EventType = "generation.cancelled"
	EventGenReviewed   kernel.EventType = "generation.reviewed"
	EventGenReady      kernel.EventType = "generation.ready"
)

var (
	ErrJobNotFound         = errors.New("generation job not found")
	ErrStageNotFound       = errors.New("generation stage not found")
	ErrJobAlreadyRunning   = errors.New("job is already running")
	ErrJobAlreadyCompleted = errors.New("job is already completed")
	ErrJobAlreadyCancelled = errors.New("job is already cancelled")
	ErrJobNotRunning       = errors.New("job is not running")
	ErrStageNotPending     = errors.New("stage is not pending")
	ErrInvalidPriority     = errors.New("priority must be between 1 and 10")
	ErrInvalidLanguage     = errors.New("language must be 'pt' or 'en'")
	ErrDatabaseNotAvail    = errors.New("database not available")
	ErrMaxRetriesExceeded  = errors.New("maximum retries exceeded")
	ErrDependencyFailed    = errors.New("dependency stage failed")
	ErrQualityGateFailed   = errors.New("quality gate not passed")
)

type GenerationJob struct {
	ID            uuid.UUID  `json:"id"`
	SiteID        uuid.UUID  `json:"site_id"`
	ArticleJobID  *uuid.UUID `json:"article_job_id,omitempty"`
	ResearchJobID *uuid.UUID `json:"research_job_id,omitempty"`
	Priority      int        `json:"priority"`
	Language      string     `json:"language"`
	Category      string     `json:"category,omitempty"`
	ArticleType   string     `json:"article_type,omitempty"`
	ExpectedSize  string     `json:"expected_size,omitempty"`
	StyleSlug     string     `json:"style_slug,omitempty"`
	Keywords      []string   `json:"keywords,omitempty"`
	Status        GenStatus  `json:"status"`
	Progress      float64    `json:"progress"`
	CurrentStage  string     `json:"current_stage,omitempty"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	RetryCount    int        `json:"retry_count"`
	MaxRetries    int        `json:"max_retries"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	CancelledAt   *time.Time `json:"cancelled_at,omitempty"`
	CreatedBy     *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Pipeline      []GenStageItem `json:"pipeline,omitempty"`
	Logs          []GenLogEntry  `json:"logs,omitempty"`
}

type GenStageItem struct {
	ID              uuid.UUID              `json:"id"`
	GenerationJobID uuid.UUID              `json:"generation_job_id"`
	Stage           string                 `json:"stage"`
	Status          StageStatus            `json:"status"`
	Progress        float64                `json:"progress"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	DurationMs      int64                  `json:"duration_ms,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	RetryCount      int                    `json:"retry_count"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type GenLogEntry struct {
	ID              uuid.UUID              `json:"id"`
	GenerationJobID uuid.UUID              `json:"generation_job_id"`
	Stage           string                 `json:"stage,omitempty"`
	Level           string                 `json:"level"`
	Message         string                 `json:"message"`
	Details         map[string]interface{} `json:"details,omitempty"`
	DurationMs      int64                  `json:"duration_ms,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
}

type QualityGate struct {
	ID                  uuid.UUID              `json:"id"`
	GenerationJobID     uuid.UUID              `json:"generation_job_id"`
	Stage               string                 `json:"stage"`
	Status              string                 `json:"status"`
	SEOScore            float64                `json:"seo_score"`
	ReadabilityScore    float64                `json:"readability_score"`
	EEATScore           float64                `json:"eeat_score"`
	KeywordDensity      float64                `json:"keyword_density"`
	HeadingScore        float64                `json:"heading_score"`
	InternalLinkingScore float64               `json:"internal_linking_score"`
	RequiredContentPassed bool                 `json:"required_content_passed"`
	MinSizePassed       bool                   `json:"min_size_passed"`
	MetadataPassed      bool                   `json:"metadata_passed"`
	OverallPassed       bool                   `json:"overall_passed"`
	Report              map[string]interface{} `json:"report,omitempty"`
	CheckedBy           *uuid.UUID             `json:"checked_by,omitempty"`
	CheckedAt           *time.Time             `json:"checked_at,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
}

type GenStats struct {
	ID             uuid.UUID `json:"id"`
	SiteID         uuid.UUID `json:"site_id"`
	Date           string    `json:"date"`
	TotalJobs      int       `json:"total_jobs"`
	CompletedJobs  int       `json:"completed_jobs"`
	FailedJobs     int       `json:"failed_jobs"`
	CancelledJobs  int       `json:"cancelled_jobs"`
	AvgDurationMs  int64     `json:"avg_duration_ms"`
	AvgSuccessRate float64   `json:"avg_success_rate"`
	TotalErrors    int       `json:"total_errors"`
	Throughput     float64   `json:"throughput"`
}

type CreateJobRequest struct {
	ArticleJobID  *uuid.UUID `json:"article_job_id,omitempty"`
	ResearchJobID *uuid.UUID `json:"research_job_id,omitempty"`
	Priority      int        `json:"priority,omitempty"`
	Language      string     `json:"language"`
	Category      string     `json:"category,omitempty"`
	ArticleType   string     `json:"article_type,omitempty"`
	ExpectedSize  string     `json:"expected_size,omitempty"`
	StyleSlug     string     `json:"style_slug,omitempty"`
	Keywords      []string   `json:"keywords,omitempty"`
}

type UpdateJobRequest struct {
	Priority     *int      `json:"priority,omitempty"`
	StyleSlug    *string   `json:"style_slug,omitempty"`
	Keywords     *[]string `json:"keywords,omitempty"`
	ExpectedSize *string   `json:"expected_size,omitempty"`
}

type RunStageRequest struct {
	Stage    string                 `json:"stage"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type QualityGateRequest struct {
	SEOScore             float64                `json:"seo_score"`
	ReadabilityScore     float64                `json:"readability_score"`
	EEATScore            float64                `json:"eeat_score"`
	KeywordDensity       float64                `json:"keyword_density"`
	HeadingScore         float64                `json:"heading_score"`
	InternalLinkingScore float64                `json:"internal_linking_score"`
	RequiredContentPassed bool                  `json:"required_content_passed"`
	MinSizePassed        bool                   `json:"min_size_passed"`
	MetadataPassed       bool                   `json:"metadata_passed"`
	OverallPassed        bool                   `json:"overall_passed"`
	Report               map[string]interface{} `json:"report,omitempty"`
}
