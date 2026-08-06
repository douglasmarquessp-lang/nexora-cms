package translation

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "translation"

// --- Job status ---

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusReview    JobStatus = "waiting_review"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// --- Pipeline stages ---

type StageType string

const (
	StageTranslate    StageType = "translate"
	StageNativeReview StageType = "native_review"
	StageSEOReview    StageType = "seo_review"
	StagePublish      StageType = "publish"
)

// StageOrder is the canonical pipeline order. A rejected stage returns the job
// to the previous stage in this list.
var StageOrder = []StageType{StageTranslate, StageNativeReview, StageSEOReview, StagePublish}

type StageStatus string

const (
	StagePending   StageStatus = "pending"
	StageRunning   StageStatus = "running"
	StageWaiting   StageStatus = "waiting_review"
	StageCompleted StageStatus = "completed"
	StageRejected  StageStatus = "rejected"
	StageFailed    StageStatus = "failed"
)

// --- Score ---

// TranslationScore holds the persisted quality score of a translation job.
// Each dimension is 0-100; Overall is the weighted average.
type TranslationScore struct {
	Grammar      float64 `json:"grammar"`
	Fluency      float64 `json:"fluency"`
	Naturalness  float64 `json:"naturalness"`
	SEO          float64 `json:"seo"`
	Consistency  float64 `json:"consistency"`
	Localization float64 `json:"localization"`
	Overall      float64 `json:"overall"`
}

// StageResult is what a stage persists in translation_stages.result (JSONB).
type StageResult struct {
	Title             string   `json:"title,omitempty"`
	Slug              string   `json:"slug,omitempty"`
	MetaTitle         string   `json:"meta_title,omitempty"`
	MetaDescription   string   `json:"meta_description,omitempty"`
	Keyword           string   `json:"keyword,omitempty"`
	SecondaryKeywords []string `json:"secondary_keywords,omitempty"`
	Content           string   `json:"content,omitempty"`
	LocalizationCount int      `json:"localization_count,omitempty"`
	GlossaryApplied   int      `json:"glossary_applied,omitempty"`
	PostID            string   `json:"post_id,omitempty"`
	PublicationID     string   `json:"publication_id,omitempty"`
}

// --- Domain models ---

type TranslationJob struct {
	ID               uuid.UUID         `json:"id"`
	SiteID           uuid.UUID         `json:"site_id"`
	ProjectID        *uuid.UUID        `json:"project_id,omitempty"`
	SourcePostID     *uuid.UUID        `json:"source_post_id,omitempty"`
	TargetSiteID     uuid.UUID         `json:"target_site_id"`
	SourceLanguage   string            `json:"source_language"`
	TargetLanguage   string            `json:"target_language"`
	Title            string            `json:"title"`
	Content          string            `json:"content"`
	Status           JobStatus         `json:"status"`
	CurrentStage     *StageType        `json:"current_stage,omitempty"`
	TranslationScore *TranslationScore `json:"translation_score,omitempty"`
	PublishedPostID  *uuid.UUID        `json:"published_post_id,omitempty"`
	PublicationID    *uuid.UUID        `json:"publication_id,omitempty"`
	ErrorMessage     string            `json:"error_message,omitempty"`
	CreatedBy        *uuid.UUID        `json:"created_by,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	Stages           []TranslationStage `json:"stages,omitempty"`
}

type TranslationStage struct {
	ID               uuid.UUID `json:"id"`
	TranslationJobID uuid.UUID `json:"translation_job_id"`
	Stage            StageType `json:"stage"`
	Status           StageStatus `json:"status"`
	Score            *float64  `json:"score,omitempty"`
	Attempt          int       `json:"attempt"`
	Feedback         string    `json:"feedback,omitempty"`
	Result           StageResult `json:"result,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

type GlossaryTerm struct {
	ID             uuid.UUID  `json:"id"`
	SiteID         uuid.UUID  `json:"site_id"`
	ProjectID      *uuid.UUID `json:"project_id,omitempty"`
	SourceTerm     string     `json:"source_term"`
	TargetTerm     string     `json:"target_term"`
	SourceLanguage string     `json:"source_language"`
	TargetLanguage string     `json:"target_language"`
	Forbidden      bool       `json:"forbidden"`
	Description    string     `json:"description,omitempty"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// --- Requests ---

type CreateJobRequest struct {
	Title          string     `json:"title"`
	Content        string     `json:"content"`
	SourceLanguage string     `json:"source_language"`
	TargetLanguage string     `json:"target_language"`
	TargetSiteID   uuid.UUID  `json:"target_site_id"`
	ProjectID      *uuid.UUID `json:"project_id,omitempty"`
	SourcePostID   *uuid.UUID `json:"source_post_id,omitempty"`
}

type CreateGlossaryTermRequest struct {
	SourceTerm     string     `json:"source_term"`
	TargetTerm     string     `json:"target_term"`
	SourceLanguage string     `json:"source_language"`
	TargetLanguage string     `json:"target_language"`
	ProjectID      *uuid.UUID `json:"project_id,omitempty"`
	Forbidden      bool       `json:"forbidden"`
	Description    string     `json:"description,omitempty"`
}

type UpdateGlossaryTermRequest struct {
	SourceTerm     *string    `json:"source_term,omitempty"`
	TargetTerm     *string    `json:"target_term,omitempty"`
	SourceLanguage *string    `json:"source_language,omitempty"`
	TargetLanguage *string    `json:"target_language,omitempty"`
	ProjectID      *uuid.UUID `json:"project_id,omitempty"`
	Forbidden      *bool      `json:"forbidden,omitempty"`
	Description    *string    `json:"description,omitempty"`
}

type DetectLanguageResult struct {
	Language   string  `json:"language"`
	Confidence float64 `json:"confidence"`
}

// --- Events ---

const (
	EventTranslationJobCreated   kernel.EventType = "translation.job.created"
	EventTranslationStagePassed  kernel.EventType = "translation.stage.passed"
	EventTranslationStageRejected kernel.EventType = "translation.stage.rejected"
	EventTranslationJobCompleted kernel.EventType = "translation.job.completed"
	EventTranslationJobFailed    kernel.EventType = "translation.job.failed"
	EventTranslationPublished    kernel.EventType = "translation.published"
	EventGlossaryCreated         kernel.EventType = "translation.glossary.created"
	EventGlossaryUpdated         kernel.EventType = "translation.glossary.updated"
	EventGlossaryDeleted         kernel.EventType = "translation.glossary.deleted"
)

// --- Errors ---

var (
	ErrJobNotFound        = errors.New("translation job not found")
	ErrStageNotFound      = errors.New("translation stage not found")
	ErrJobNotInSite       = errors.New("translation job does not belong to this site")
	ErrTitleRequired      = errors.New("title is required")
	ErrTargetSiteRequired = errors.New("target site is required")
	ErrInvalidLanguage    = errors.New("language must be 'pt' or 'en'")
	ErrInvalidStatus      = errors.New("invalid job status for this operation")
	ErrAIManagerRequired  = errors.New("AI manager not configured")
	ErrGlossaryNotFound   = errors.New("glossary term not found")
	ErrGlossaryDuplicate  = errors.New("glossary term already exists for this scope and direction")
	ErrInvalidGlossary    = errors.New("source and target terms are required")
	ErrPublishFailed      = errors.New("failed to publish translated article")
	ErrDatabaseNotAvail   = errors.New("database not available")
)

// Pipeline quality thresholds (deterministic, module-level defaults).
const (
	MinNativeReviewScore = 70.0 // native review rejects below this (after rewrite attempts)
	MinSEOScore          = 60.0 // international SEO review minimum
	MaxRewriteAttempts   = 2    // rewrite-section loop limit in native review
)
