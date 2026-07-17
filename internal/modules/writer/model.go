package writer

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "writer"

type JobStatus string

const (
	JobStatusDraft     JobStatus = "draft"
	JobStatusWriting   JobStatus = "writing"
	JobStatusReview    JobStatus = "review"
	JobStatusApproved  JobStatus = "approved"
	JobStatusPublished JobStatus = "published"
	JobStatusFailed    JobStatus = "failed"
)

type SectionStatus string

const (
	SectionStatusPending    SectionStatus = "pending"
	SectionStatusWriting    SectionStatus = "writing"
	SectionStatusCompleted  SectionStatus = "completed"
	SectionStatusReview     SectionStatus = "review"
)

type StyleSlug string

const (
	StyleJournalistic  StyleSlug = "journalistic"
	StyleTechnical     StyleSlug = "technical"
	StyleTutorial      StyleSlug = "tutorial"
	StyleReview        StyleSlug = "review"
	StyleComparative   StyleSlug = "comparative"
	StyleList          StyleSlug = "list"
	StyleOpinion       StyleSlug = "opinion"
	StyleCompleteGuide StyleSlug = "complete_guide"
)

type OutlineSectionType string

const (
	OutlineH1 OutlineSectionType = "h1"
	OutlineH2 OutlineSectionType = "h2"
	OutlineH3 OutlineSectionType = "h3"
)

const (
	EventWriterJobCreated         kernel.EventType = "writer.job.created"
	EventWriterJobUpdated         kernel.EventType = "writer.job.updated"
	EventWriterJobCompleted       kernel.EventType = "writer.job.completed"
	EventWriterVersionCreated     kernel.EventType = "writer.version.created"
	EventWriterVersionRestored    kernel.EventType = "writer.version.restored"
)

var (
	ErrWritingJobNotFound    = errors.New("writing job not found")
	ErrOutlineNotFound       = errors.New("outline not found")
	ErrSectionNotFound       = errors.New("section not found")
	ErrVersionNotFound       = errors.New("version not found")
	ErrStyleNotFound         = errors.New("writing style not found")
	ErrDatabaseNotAvail      = errors.New("database not available")
	ErrInvalidLanguage       = errors.New("language must be 'pt' or 'en'")
	ErrHeadlineRequired      = errors.New("headline is required")
	ErrJobNotEditable        = errors.New("job is not editable in current status")
)

type WritingStyle struct {
	ID          uuid.UUID              `json:"id"`
	SiteID      uuid.UUID              `json:"site_id"`
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Description string                 `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	IsDefault   bool                   `json:"is_default"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type ArticleJob struct {
	ID             uuid.UUID  `json:"id"`
	SiteID         uuid.UUID  `json:"site_id"`
	ResearchJobID  *uuid.UUID `json:"research_job_id,omitempty"`
	StyleID        *uuid.UUID `json:"style_id,omitempty"`
	StyleName      string     `json:"style_name,omitempty"`
	Language       string     `json:"language"`
	Status         JobStatus  `json:"status"`
	Headline       string     `json:"headline,omitempty"`
	SEOTitle       string     `json:"seo_title,omitempty"`
	Slug           string     `json:"slug,omitempty"`
	MetaDescription string    `json:"meta_description,omitempty"`
	TargetAudience string     `json:"target_audience,omitempty"`
	Tone           string     `json:"tone,omitempty"`
	Formality      string     `json:"formality,omitempty"`
	SEOGoal        string     `json:"seo_goal,omitempty"`
	DesiredSize    string     `json:"desired_size,omitempty"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type ArticleOutline struct {
	ID              uuid.UUID `json:"id"`
	ArticleJobID    uuid.UUID `json:"article_job_id"`
	SectionType     string    `json:"section_type"`
	Title           string    `json:"title"`
	Level           int       `json:"level"`
	Content         string    `json:"content,omitempty"`
	Position        int       `json:"position"`
	WordCountTarget int       `json:"word_count_target,omitempty"`
	Keywords        string    `json:"keywords,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type ArticleSection struct {
	ID           uuid.UUID     `json:"id"`
	ArticleJobID uuid.UUID     `json:"article_job_id"`
	OutlineID    *uuid.UUID    `json:"outline_id,omitempty"`
	Title        string        `json:"title"`
	Content      string        `json:"content,omitempty"`
	WordCount    int           `json:"word_count"`
	Status       SectionStatus `json:"status"`
	Position     int           `json:"position"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

type ArticleVersion struct {
	ID              uuid.UUID              `json:"id"`
	ArticleJobID    uuid.UUID              `json:"article_job_id"`
	Version         int                    `json:"version"`
	Headline        string                 `json:"headline,omitempty"`
	SEOTitle        string                 `json:"seo_title,omitempty"`
	Slug            string                 `json:"slug,omitempty"`
	MetaDescription string                 `json:"meta_description,omitempty"`
	Sections        []interface{}          `json:"sections,omitempty"`
	Content         []interface{}          `json:"content,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Summary         string                 `json:"summary,omitempty"`
	ChangeLog       string                 `json:"change_log,omitempty"`
	CreatedBy       *uuid.UUID             `json:"created_by,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
}

type CreateArticleJobRequest struct {
	ResearchJobID  *uuid.UUID `json:"research_job_id,omitempty"`
	StyleSlug      string     `json:"style_slug,omitempty"`
	Language       string     `json:"language"`
	Headline       string     `json:"headline,omitempty"`
	SEOTitle       string     `json:"seo_title,omitempty"`
	Slug           string     `json:"slug,omitempty"`
	MetaDescription string    `json:"meta_description,omitempty"`
	TargetAudience string     `json:"target_audience,omitempty"`
	Tone           string     `json:"tone,omitempty"`
	Formality      string     `json:"formality,omitempty"`
	SEOGoal        string     `json:"seo_goal,omitempty"`
	DesiredSize    string     `json:"desired_size,omitempty"`
}

type UpdateArticleJobRequest struct {
	Status         *JobStatus `json:"status,omitempty"`
	Headline       *string    `json:"headline,omitempty"`
	SEOTitle       *string    `json:"seo_title,omitempty"`
	Slug           *string    `json:"slug,omitempty"`
	MetaDescription *string   `json:"meta_description,omitempty"`
	StyleSlug      *string    `json:"style_slug,omitempty"`
	Tone           *string    `json:"tone,omitempty"`
	Formality      *string    `json:"formality,omitempty"`
	SEOGoal        *string    `json:"seo_goal,omitempty"`
	DesiredSize    *string    `json:"desired_size,omitempty"`
}

type CreateOutlineRequest struct {
	Sections []CreateOutlineSection `json:"sections"`
}

type CreateOutlineSection struct {
	SectionType     string `json:"section_type"`
	Title           string `json:"title"`
	Level           int    `json:"level"`
	Content         string `json:"content,omitempty"`
	Position        int    `json:"position"`
	WordCountTarget int    `json:"word_count_target,omitempty"`
	Keywords        string `json:"keywords,omitempty"`
}

type CreateSectionRequest struct {
	OutlineID *uuid.UUID `json:"outline_id,omitempty"`
	Title     string     `json:"title"`
	Content   string     `json:"content,omitempty"`
	Position  int        `json:"position"`
}

type UpdateSectionRequest struct {
	Title     *string       `json:"title,omitempty"`
	Content   *string       `json:"content,omitempty"`
	Status    *SectionStatus `json:"status,omitempty"`
	Position  *int           `json:"position,omitempty"`
}

type ArticleJobDetail struct {
	ArticleJob
	Sections []ArticleSection `json:"sections,omitempty"`
	Outline  []ArticleOutline `json:"outline,omitempty"`
	Versions []ArticleVersion `json:"versions,omitempty"`
}
