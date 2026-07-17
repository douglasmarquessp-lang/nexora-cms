package research

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "research"

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

type EntityType string

const (
	EntityTypeFact       EntityType = "fact"
	EntityTypeStatistic  EntityType = "statistic"
	EntityTypeCompany    EntityType = "company"
	EntityTypePerson     EntityType = "person"
	EntityTypeProduct    EntityType = "product"
	EntityTypeKeyword    EntityType = "keyword"
)

const (
	EventResearchCreated   kernel.EventType = "research.created"
	EventResearchUpdated   kernel.EventType = "research.updated"
	EventResearchCompleted kernel.EventType = "research.completed"
	EventResearchDeleted   kernel.EventType = "research.deleted"
)

var (
	ErrResearchJobNotFound    = errors.New("research job not found")
	ErrResearchJobNotEditable = errors.New("research job is not editable")
	ErrBriefingNotFound       = errors.New("briefing not found")
	ErrDatabaseNotAvail       = errors.New("database not available")
	ErrInvalidLanguage        = errors.New("language must be 'pt' or 'en'")
	ErrTopicRequired          = errors.New("topic is required")
)

type ResearchJob struct {
	ID           uuid.UUID  `json:"id"`
	SiteID       uuid.UUID  `json:"site_id"`
	Topic        string     `json:"topic"`
	Language     string     `json:"language"`
	Category     string     `json:"category,omitempty"`
	Status       JobStatus  `json:"status"`
	SourcesCount int        `json:"sources_count"`
	ErrorMessage string     `json:"error_message,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type ResearchSource struct {
	ID             uuid.UUID  `json:"id"`
	ResearchJobID  uuid.UUID  `json:"research_job_id"`
	Title          string     `json:"title"`
	URL            string     `json:"url"`
	Language       string     `json:"language,omitempty"`
	Author         string     `json:"author,omitempty"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	Summary        string     `json:"summary,omitempty"`
	MainFacts      string     `json:"main_facts,omitempty"`
	Statistics     string     `json:"statistics,omitempty"`
	RelevanceScore int        `json:"relevance_score"`
	Position       int        `json:"position"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ResearchEntity struct {
	ID            uuid.UUID  `json:"id"`
	ResearchJobID uuid.UUID  `json:"research_job_id"`
	EntityType    EntityType `json:"entity_type"`
	Name          string     `json:"name"`
	Context       string     `json:"context,omitempty"`
	SourceURL     string     `json:"source_url,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type ResearchBriefing struct {
	ID                uuid.UUID              `json:"id"`
	ResearchJobID     uuid.UUID              `json:"research_job_id"`
	StructuredBriefing map[string]interface{} `json:"structured_briefing,omitempty"`
	Timeline          []interface{}          `json:"timeline,omitempty"`
	ConfirmedFacts    []interface{}          `json:"confirmed_facts,omitempty"`
	ConflictingInfo   []interface{}          `json:"conflicting_info,omitempty"`
	EditorialApproaches []interface{}        `json:"editorial_approaches,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

type ResearchJobDetail struct {
	ResearchJob
	Sources   []ResearchSource   `json:"sources,omitempty"`
	Entities  []ResearchEntity   `json:"entities,omitempty"`
	Briefing  *ResearchBriefing  `json:"briefing,omitempty"`
}

type CreateResearchJobRequest struct {
	Topic    string `json:"topic"`
	Language string `json:"language"`
	Category string `json:"category,omitempty"`
}

type UpdateResearchJobRequest struct {
	Status  *JobStatus `json:"status,omitempty"`
	Topic   *string    `json:"topic,omitempty"`
	Category *string   `json:"category,omitempty"`
}
