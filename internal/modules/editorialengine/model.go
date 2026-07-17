package editorialengine

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "editorialengine"

type PipelineStage string

const (
	StageResearch    PipelineStage = "research"
	StageBriefing    PipelineStage = "briefing"
	StageOutline     PipelineStage = "outline"
	StageWriting     PipelineStage = "writing"
	StageReview      PipelineStage = "review"
	StageSEO         PipelineStage = "seo"
	StageTranslation PipelineStage = "translation"
	StagePublish     PipelineStage = "publish"
)

var ValidStages = []PipelineStage{
	StageResearch, StageBriefing, StageOutline, StageWriting,
	StageReview, StageSEO, StageTranslation, StagePublish,
}

type StageStatus string

const (
	StageStatusPending    StageStatus = "pending"
	StageStatusInProgress StageStatus = "in_progress"
	StageStatusCompleted  StageStatus = "completed"
	StageStatusFailed     StageStatus = "failed"
	StageStatusBlocked    StageStatus = "blocked"
)

type TranslationStatus string

const (
	TransStatusPending    TranslationStatus = "pending"
	TransStatusInProgress TranslationStatus = "in_progress"
	TransStatusCompleted  TranslationStatus = "completed"
	TransStatusFailed     TranslationStatus = "failed"
)

const (
	EventEditorialStarted    kernel.EventType = "editorial.started"
	EventEditorialReviewed   kernel.EventType = "editorial.reviewed"
	EventEditorialScored     kernel.EventType = "editorial.scored"
	EventEditorialTranslated kernel.EventType = "editorial.translated"
	EventEditorialCompleted  kernel.EventType = "editorial.completed"
	EventStyleUpdated        kernel.EventType = "style.updated"
	EventSEOGenerated        kernel.EventType = "seo.generated"
	EventQualityChecked      kernel.EventType = "quality.checked"
)

var (
	ErrPipelineNotFound      = errors.New("editorial pipeline not found")
	ErrStageNotFound         = errors.New("pipeline stage not found")
	ErrStyleRulesNotFound    = errors.New("editorial style rules not found")
	ErrSEONotFound           = errors.New("seo data not found")
	ErrQualityNotFound       = errors.New("quality score not found")
	ErrTranslationNotFound   = errors.New("translation not found")
	ErrPromptDataNotFound    = errors.New("prompt data not found")
	ErrDatabaseNotAvail      = errors.New("database not available")
	ErrInvalidStage          = errors.New("invalid pipeline stage")
	ErrInvalidStageStatus    = errors.New("invalid stage status")
	ErrInvalidTranslationDir = errors.New("invalid translation direction")
	ErrJobAlreadyInPipeline  = errors.New("article job already has a pipeline")
)

type EditorialPipeline struct {
	ID           uuid.UUID    `json:"id"`
	ArticleJobID uuid.UUID    `json:"article_job_id"`
	SiteID       uuid.UUID    `json:"site_id"`
	CurrentStage PipelineStage `json:"current_stage"`
	Status       StageStatus   `json:"status"`
	StartedAt    *time.Time    `json:"started_at,omitempty"`
	CompletedAt  *time.Time    `json:"completed_at,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Stages       []PipelineStageItem `json:"stages,omitempty"`
}

type PipelineStageItem struct {
	ID         uuid.UUID    `json:"id"`
	PipelineID uuid.UUID    `json:"pipeline_id"`
	Stage      PipelineStage `json:"stage"`
	Status     StageStatus   `json:"status"`
	StartedAt  *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	AssignedTo *uuid.UUID   `json:"assigned_to,omitempty"`
	Notes      string        `json:"notes,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

type EditorialStyleRules struct {
	ID                  uuid.UUID              `json:"id"`
	SiteID              uuid.UUID              `json:"site_id"`
	BrandVoice          string                 `json:"brand_voice,omitempty"`
	Tone                string                 `json:"tone,omitempty"`
	LanguageLevel       string                 `json:"language_level,omitempty"`
	TargetAudience      string                 `json:"target_audience,omitempty"`
	AvgWordCount        int                    `json:"avg_word_count,omitempty"`
	HeadingStructure    []interface{}          `json:"heading_structure,omitempty"`
	ProhibitedVocabulary []string              `json:"prohibited_vocabulary,omitempty"`
	RequiredExpressions []string               `json:"required_expressions,omitempty"`
	Personality         string                 `json:"personality,omitempty"`
	FormalityDegree     string                 `json:"formality_degree,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

type SEOData struct {
	ID                    uuid.UUID              `json:"id"`
	ArticleJobID          uuid.UUID              `json:"article_job_id"`
	SiteID                uuid.UUID              `json:"site_id"`
	PrimaryKeyword        string                 `json:"primary_keyword,omitempty"`
	SecondaryKeywords     []string               `json:"secondary_keywords,omitempty"`
	LongTailKeywords      []string               `json:"long_tail_keywords,omitempty"`
	Entities              []interface{}          `json:"entities,omitempty"`
	FAQ                   []interface{}          `json:"faq,omitempty"`
	SchemaType            string                 `json:"schema_type,omitempty"`
	SchemaData            map[string]interface{} `json:"schema_data,omitempty"`
	MetaTitle             string                 `json:"meta_title,omitempty"`
	MetaDescription       string                 `json:"meta_description,omitempty"`
	Slug                  string                 `json:"slug,omitempty"`
	CanonicalURL          string                 `json:"canonical_url,omitempty"`
	Robots                string                 `json:"robots,omitempty"`
	OGData                map[string]interface{} `json:"og_data,omitempty"`
	TwitterCard           map[string]interface{} `json:"twitter_card,omitempty"`
	AltText               []string               `json:"alt_text,omitempty"`
	SuggestedInternalLinks []string              `json:"suggested_internal_links,omitempty"`
	SuggestedExternalLinks []string              `json:"suggested_external_links,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

type QualityScore struct {
	ID                  uuid.UUID              `json:"id"`
	ArticleJobID        uuid.UUID              `json:"article_job_id"`
	SiteID              uuid.UUID              `json:"site_id"`
	SEOScore            float64                `json:"seo_score"`
	ReadabilityScore    float64                `json:"readability_score"`
	NaturalnessScore    float64                `json:"naturalness_score"`
	EEATScore           float64                `json:"eeat_score"`
	KeywordDensity      float64                `json:"keyword_density"`
	HeadingStructure    float64                `json:"heading_structure_score"`
	InternalLinkingScore float64               `json:"internal_linking_score"`
	DuplicateDetection  []interface{}          `json:"duplicate_detection,omitempty"`
	RepetitionDetection []interface{}          `json:"repetition_detection,omitempty"`
	PassiveVoiceCount   int                    `json:"passive_voice_count"`
	AvgSentenceLength   float64                `json:"avg_sentence_length"`
	ParagraphBalance    float64                `json:"paragraph_balance_score"`
	OverallScore        float64                `json:"overall_score"`
	Report              map[string]interface{} `json:"report,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
}

type EditorialTranslation struct {
	ID               uuid.UUID              `json:"id"`
	ArticleJobID     uuid.UUID              `json:"article_job_id"`
	SiteID           uuid.UUID              `json:"site_id"`
	SourceLanguage   string                 `json:"source_language"`
	TargetLanguage   string                 `json:"target_language"`
	Status           TranslationStatus       `json:"status"`
	TranslatedSlug   string                 `json:"translated_slug,omitempty"`
	TranslatedMeta   map[string]interface{} `json:"translated_meta,omitempty"`
	TranslatedFAQ    []interface{}          `json:"translated_faq,omitempty"`
	TranslatedKeywords []string             `json:"translated_keywords,omitempty"`
	TranslatedEntities []interface{}        `json:"translated_entities,omitempty"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type PromptData struct {
	ID             uuid.UUID              `json:"id"`
	ArticleJobID   uuid.UUID              `json:"article_job_id"`
	SiteID         uuid.UUID              `json:"site_id"`
	Briefing       map[string]interface{} `json:"briefing,omitempty"`
	StyleRules     map[string]interface{} `json:"style_rules,omitempty"`
	SEORules       map[string]interface{} `json:"seo_rules,omitempty"`
	Tone           string                 `json:"tone,omitempty"`
	Outline        []interface{}          `json:"outline,omitempty"`
	Entities       []interface{}          `json:"entities,omitempty"`
	TargetLanguage string                 `json:"target_language,omitempty"`
	Audience       string                 `json:"audience,omitempty"`
	WordCount      int                    `json:"word_count,omitempty"`
	InternalLinks  []string               `json:"internal_links,omitempty"`
	Constraints    []string               `json:"constraints,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type PipelineDetail struct {
	EditorialPipeline
	Stages []PipelineStageItem `json:"stages"`
}

type CreatePipelineRequest struct {
	ArticleJobID uuid.UUID `json:"article_job_id"`
}

type UpdatePipelineRequest struct {
	CurrentStage *PipelineStage `json:"current_stage,omitempty"`
	Status       *StageStatus   `json:"status,omitempty"`
}

type UpdateStageRequest struct {
	Status     *StageStatus `json:"status,omitempty"`
	AssignedTo *uuid.UUID  `json:"assigned_to,omitempty"`
	Notes      *string      `json:"notes,omitempty"`
	Metadata   *map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateStyleRulesRequest struct {
	BrandVoice          *string    `json:"brand_voice,omitempty"`
	Tone                *string    `json:"tone,omitempty"`
	LanguageLevel       *string    `json:"language_level,omitempty"`
	TargetAudience      *string    `json:"target_audience,omitempty"`
	AvgWordCount        *int       `json:"avg_word_count,omitempty"`
	HeadingStructure    *[]interface{} `json:"heading_structure,omitempty"`
	ProhibitedVocabulary *[]string `json:"prohibited_vocabulary,omitempty"`
	RequiredExpressions *[]string  `json:"required_expressions,omitempty"`
	Personality         *string    `json:"personality,omitempty"`
	FormalityDegree     *string    `json:"formality_degree,omitempty"`
}

type CreateSEODataRequest struct {
	PrimaryKeyword        string                 `json:"primary_keyword"`
	SecondaryKeywords     []string               `json:"secondary_keywords,omitempty"`
	LongTailKeywords      []string               `json:"long_tail_keywords,omitempty"`
	Entities              []interface{}          `json:"entities,omitempty"`
	FAQ                   []interface{}          `json:"faq,omitempty"`
	SchemaType            string                 `json:"schema_type,omitempty"`
	SchemaData            map[string]interface{} `json:"schema_data,omitempty"`
	MetaTitle             string                 `json:"meta_title,omitempty"`
	MetaDescription       string                 `json:"meta_description,omitempty"`
	Slug                  string                 `json:"slug,omitempty"`
	CanonicalURL          string                 `json:"canonical_url,omitempty"`
	Robots                string                 `json:"robots,omitempty"`
	OGData                map[string]interface{} `json:"og_data,omitempty"`
	TwitterCard           map[string]interface{} `json:"twitter_card,omitempty"`
	AltText               []string               `json:"alt_text,omitempty"`
	SuggestedInternalLinks []string              `json:"suggested_internal_links,omitempty"`
	SuggestedExternalLinks []string              `json:"suggested_external_links,omitempty"`
}

type UpdateSEODataRequest struct {
	PrimaryKeyword        *string                 `json:"primary_keyword,omitempty"`
	SecondaryKeywords     *[]string               `json:"secondary_keywords,omitempty"`
	LongTailKeywords      *[]string               `json:"long_tail_keywords,omitempty"`
	Entities              *[]interface{}          `json:"entities,omitempty"`
	FAQ                   *[]interface{}          `json:"faq,omitempty"`
	SchemaType            *string                 `json:"schema_type,omitempty"`
	SchemaData            *map[string]interface{} `json:"schema_data,omitempty"`
	MetaTitle             *string                 `json:"meta_title,omitempty"`
	MetaDescription       *string                 `json:"meta_description,omitempty"`
	Slug                  *string                 `json:"slug,omitempty"`
	CanonicalURL          *string                 `json:"canonical_url,omitempty"`
	Robots                *string                 `json:"robots,omitempty"`
	OGData                *map[string]interface{} `json:"og_data,omitempty"`
	TwitterCard           *map[string]interface{} `json:"twitter_card,omitempty"`
	AltText               *[]string               `json:"alt_text,omitempty"`
	SuggestedInternalLinks *[]string              `json:"suggested_internal_links,omitempty"`
	SuggestedExternalLinks *[]string              `json:"suggested_external_links,omitempty"`
}

type CreateQualityScoreRequest struct {
	SEOScore            float64                `json:"seo_score"`
	ReadabilityScore    float64                `json:"readability_score"`
	NaturalnessScore    float64                `json:"naturalness_score"`
	EEATScore           float64                `json:"eeat_score"`
	KeywordDensity      float64                `json:"keyword_density"`
	HeadingStructure    float64                `json:"heading_structure_score"`
	InternalLinkingScore float64               `json:"internal_linking_score"`
	DuplicateDetection  []interface{}          `json:"duplicate_detection,omitempty"`
	RepetitionDetection []interface{}          `json:"repetition_detection,omitempty"`
	PassiveVoiceCount   int                    `json:"passive_voice_count"`
	AvgSentenceLength   float64                `json:"avg_sentence_length"`
	ParagraphBalance    float64                `json:"paragraph_balance_score"`
	Report              map[string]interface{} `json:"report,omitempty"`
}

type CreateTranslationRequest struct {
	SourceLanguage string `json:"source_language"`
	TargetLanguage string `json:"target_language"`
}

type UpdateTranslationRequest struct {
	Status             *TranslationStatus      `json:"status,omitempty"`
	TranslatedSlug     *string                 `json:"translated_slug,omitempty"`
	TranslatedMeta     *map[string]interface{} `json:"translated_meta,omitempty"`
	TranslatedFAQ      *[]interface{}          `json:"translated_faq,omitempty"`
	TranslatedKeywords *[]string               `json:"translated_keywords,omitempty"`
	TranslatedEntities *[]interface{}          `json:"translated_entities,omitempty"`
}
