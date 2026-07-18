package seoengine

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "seoengine"

type ProjectStatus string

const (
	ProjectStatusDraft     ProjectStatus = "draft"
	ProjectStatusPending   ProjectStatus = "pending"
	ProjectStatusRunning   ProjectStatus = "running"
	ProjectStatusCompleted ProjectStatus = "completed"
	ProjectStatusFailed    ProjectStatus = "failed"
)

type ImprovementStatus string

const (
	ImprovementPending  ImprovementStatus = "pending"
	ImprovementApplied  ImprovementStatus = "applied"
	ImprovementDismissed ImprovementStatus = "dismissed"
)

type ImprovementPriority string

const (
	PriorityCritical ImprovementPriority = "critical"
	PriorityHigh     ImprovementPriority = "high"
	PriorityMedium   ImprovementPriority = "medium"
	PriorityLow      ImprovementPriority = "low"
)

type ImprovementCategory string

const (
	CategoryTitle        ImprovementCategory = "title"
	CategoryMeta         ImprovementCategory = "meta_description"
	CategorySlug         ImprovementCategory = "slug"
	CategoryHeading      ImprovementCategory = "heading"
	CategoryImage        ImprovementCategory = "image_alt"
	CategorySchema       ImprovementCategory = "schema"
	CategoryLink         ImprovementCategory = "internal_link"
	CategoryReadability  ImprovementCategory = "readability"
	CategoryEEAT         ImprovementCategory = "eeat"
	CategoryFreshness    ImprovementCategory = "freshness"
	CategoryDuplicate    ImprovementCategory = "duplicate"
	CategoryCannibalization ImprovementCategory = "cannibalization"
	CategoryGap          ImprovementCategory = "content_gap"
	CategoryOrphan       ImprovementCategory = "orphan"
)

var AllCategories = []ImprovementCategory{
	CategoryTitle, CategoryMeta, CategorySlug, CategoryHeading,
	CategoryImage, CategorySchema, CategoryLink, CategoryReadability,
	CategoryEEAT, CategoryFreshness, CategoryDuplicate,
	CategoryCannibalization, CategoryGap, CategoryOrphan,
}

type SEOProject struct {
	ID                  uuid.UUID  `json:"id"`
	SiteID              uuid.UUID  `json:"site_id"`
	UserID              *uuid.UUID `json:"user_id,omitempty"`
	Title               string     `json:"title"`
	TargetURL           string     `json:"target_url,omitempty"`
	PostID              *uuid.UUID `json:"post_id,omitempty"`
	Language            string     `json:"language"`
	Status              ProjectStatus `json:"status"`
	SEOScore            float64    `json:"seo_score"`
	ReadabilityScore    float64    `json:"readability_score"`
	KeywordDensity      float64    `json:"keyword_density"`
	ContentQuality      float64    `json:"content_quality"`
	TechnicalScore      float64    `json:"technical_score"`
	EEATScore           float64    `json:"eeat_score"`
	FreshnessScore      float64    `json:"freshness_score"`
	TopicalAuthorityScore float64 `json:"topical_authority_score"`
	SlugTarget          string     `json:"slug_target,omitempty"`
	MetaTitleTarget     string     `json:"meta_title_target,omitempty"`
	MetaDescriptionTarget string  `json:"meta_description_target,omitempty"`
	ContentType         string     `json:"content_type,omitempty"`
	Recommendations     []string   `json:"recommendations,omitempty"`
	Checklist           []ChecklistItem `json:"checklist,omitempty"`
	AISuggestions       map[string]interface{} `json:"ai_suggestions,omitempty"`
	StartedAt           *time.Time `json:"started_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	ErrorMessage        string     `json:"error_message,omitempty"`
	CreatedBy           *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type ChecklistItem struct {
	Category    ImprovementCategory `json:"category"`
	Issue       string              `json:"issue"`
	Suggestion  string              `json:"suggestion"`
	Priority    ImprovementPriority `json:"priority"`
	Completed   bool                `json:"completed"`
	Score       float64             `json:"score"`
}

type SEOScores struct {
	ID                  uuid.UUID  `json:"id"`
	SiteID              uuid.UUID  `json:"site_id"`
	SEOProjectID        *uuid.UUID `json:"seo_project_id,omitempty"`
	PostID              *uuid.UUID `json:"post_id,omitempty"`
	TotalScore          float64    `json:"total_score"`
	KeywordScore        float64    `json:"keyword_score"`
	ContentScore        float64    `json:"content_score"`
	TechnicalScore      float64    `json:"technical_score"`
	LinkingScore        float64    `json:"linking_score"`
	ReadabilityScore    float64    `json:"readability_score"`
	MetadataScore       float64    `json:"metadata_score"`
	EEATScore           float64    `json:"eeat_score"`
	FreshnessScore      float64    `json:"freshness_score"`
	TopicalAuthorityScore float64  `json:"topical_authority_score"`
	SchemaScore         float64    `json:"schema_score"`
	ImageScore          float64    `json:"image_score"`
	SlugScore           float64    `json:"slug_score"`
	HeadingScore        float64    `json:"heading_score"`
	MultilingualScore   float64    `json:"multilingual_score"`
	Language            string     `json:"language"`
	ScoredAt            time.Time  `json:"scored_at"`
	CreatedAt           time.Time  `json:"created_at"`
}

type SEOKeyword struct {
	ID                 uuid.UUID `json:"id"`
	SiteID             uuid.UUID `json:"site_id"`
	SEOProjectID       *uuid.UUID `json:"seo_project_id,omitempty"`
	ClusterID          *uuid.UUID `json:"cluster_id,omitempty"`
	Keyword            string    `json:"keyword"`
	KeywordType        string    `json:"keyword_type"`
	SearchIntent       string    `json:"search_intent"`
	Volume             int       `json:"volume"`
	Difficulty         float64   `json:"difficulty"`
	Density            float64   `json:"density"`
	Frequency          int       `json:"frequency"`
	Prominence         float64   `json:"prominence"`
	Entities           []string  `json:"entities,omitempty"`
	SemanticEntities   []string  `json:"semantic_entities,omitempty"`
	CannibalizationScore float64 `json:"cannibalization_score"`
	ContentGapScore    float64   `json:"content_gap_score"`
	TopicalRelevance   float64   `json:"topical_relevance"`
	Language           string    `json:"language"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type SEOCluster struct {
	ID                   uuid.UUID `json:"id"`
	SiteID               uuid.UUID `json:"site_id"`
	Name                 string    `json:"name"`
	Description          string    `json:"description,omitempty"`
	Keywords             []string  `json:"keywords,omitempty"`
	ArticleCount         int       `json:"article_count"`
	AvgScore             float64   `json:"avg_score"`
	TopicalAuthorityScore float64  `json:"topical_authority_score"`
	SemanticEntities     []string  `json:"semantic_entities,omitempty"`
	InternalLinksCount   int       `json:"internal_links_count"`
	ContentGapArticles   []string  `json:"content_gap_articles,omitempty"`
	ParentClusterID      *uuid.UUID `json:"parent_cluster_id,omitempty"`
	Language             string    `json:"language"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type SEOAudit struct {
	ID                    uuid.UUID              `json:"id"`
	SiteID                uuid.UUID              `json:"site_id"`
	SEOProjectID          *uuid.UUID             `json:"seo_project_id,omitempty"`
	PostID                *uuid.UUID             `json:"post_id,omitempty"`
	URL                   string                 `json:"url,omitempty"`
	TitleScore            float64                `json:"title_score"`
	MetaScore             float64                `json:"meta_score"`
	HeadingScore          float64                `json:"heading_score"`
	ParagraphScore        float64                `json:"paragraph_score"`
	ReadabilityScore      float64                `json:"readability_score"`
	PassiveVoiceScore     float64                `json:"passive_voice_score"`
	SentenceVariationScore float64               `json:"sentence_variation_score"`
	DuplicateScore        float64                `json:"duplicate_score"`
	OverallScore          float64                `json:"overall_score"`
	EEATScore             float64                `json:"eeat_score"`
	FreshnessScore        float64                `json:"freshness_score"`
	SlugScore             float64                `json:"slug_score"`
	OrphanDetected        bool                   `json:"orphan_detected"`
	CannibalizationDetected bool                 `json:"cannibalization_detected"`
	ContentGapDetected    bool                   `json:"content_gap_detected"`
	Issues                []AuditIssue           `json:"issues,omitempty"`
	Recommendations       []string               `json:"recommendations,omitempty"`
	HeadingIssues         []AuditIssue           `json:"heading_issues,omitempty"`
	ImageAltIssues        []AuditIssue           `json:"image_alt_issues,omitempty"`
	SchemaIssues          []AuditIssue           `json:"schema_issues,omitempty"`
	SlugIssues            []string               `json:"slug_issues,omitempty"`
	TitleIssues           []string               `json:"title_issues,omitempty"`
	MetaIssues            []string               `json:"meta_issues,omitempty"`
	EEATIssues            []AuditIssue           `json:"eeat_issues,omitempty"`
	FreshnessIssues       []AuditIssue           `json:"freshness_issues,omitempty"`
	LinkSuggestions       []LinkSuggestion       `json:"link_suggestions,omitempty"`
	ChecklistItems        []ChecklistItem        `json:"checklist_items,omitempty"`
	Language              string                 `json:"language"`
	AuditedAt             time.Time              `json:"audited_at"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

type AuditIssue struct {
	Field       string  `json:"field"`
	Issue       string  `json:"issue"`
	Suggestion  string  `json:"suggestion"`
	Score       float64 `json:"score"`
	Priority    string  `json:"priority"`
}

type LinkSuggestion struct {
	SourceURL   string  `json:"source_url"`
	TargetURL   string  `json:"target_url"`
	AnchorText  string  `json:"anchor_text"`
	Relevance   float64 `json:"relevance"`
}

type SEOImprovement struct {
	ID           uuid.UUID            `json:"id"`
	SiteID       uuid.UUID            `json:"site_id"`
	SEOProjectID *uuid.UUID           `json:"seo_project_id,omitempty"`
	PostID       *uuid.UUID           `json:"post_id,omitempty"`
	Category     ImprovementCategory  `json:"category"`
	Issue        string               `json:"issue"`
	Suggestion   string               `json:"suggestion"`
	Priority     ImprovementPriority  `json:"priority"`
	ImpactScore  float64              `json:"impact_score"`
	EffortScore  float64              `json:"effort_score"`
	Status       ImprovementStatus    `json:"status"`
	AppliedAt    *time.Time           `json:"applied_at,omitempty"`
	Language     string               `json:"language"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// --- DTOs ---

type CreateProjectRequest struct {
	Title               string   `json:"title"`
	TargetURL           string   `json:"target_url,omitempty"`
	PostID              *uuid.UUID `json:"post_id,omitempty"`
	Language            string   `json:"language,omitempty"`
	ContentType         string   `json:"content_type,omitempty"`
	SlugTarget          string   `json:"slug_target,omitempty"`
	MetaTitleTarget     string   `json:"meta_title_target,omitempty"`
	MetaDescriptionTarget string `json:"meta_description_target,omitempty"`
}

type UpdateProjectRequest struct {
	Title               *string   `json:"title,omitempty"`
	TargetURL           *string   `json:"target_url,omitempty"`
	Language            *string   `json:"language,omitempty"`
	ContentType         *string   `json:"content_type,omitempty"`
	SlugTarget          *string   `json:"slug_target,omitempty"`
	MetaTitleTarget     *string   `json:"meta_title_target,omitempty"`
	MetaDescriptionTarget *string `json:"meta_description_target,omitempty"`
}

type AddImprovementRequest struct {
	Category    ImprovementCategory `json:"category"`
	Issue       string              `json:"issue"`
	Suggestion  string              `json:"suggestion"`
	Priority    ImprovementPriority `json:"priority,omitempty"`
	ImpactScore float64             `json:"impact_score,omitempty"`
	EffortScore float64             `json:"effort_score,omitempty"`
	PostID      *uuid.UUID          `json:"post_id,omitempty"`
	Language    string              `json:"language,omitempty"`
}

type UpdateImprovementRequest struct {
	Status      *ImprovementStatus  `json:"status,omitempty"`
	Priority    *ImprovementPriority `json:"priority,omitempty"`
	ImpactScore *float64            `json:"impact_score,omitempty"`
	EffortScore *float64            `json:"effort_score,omitempty"`
}

type CreateClusterRequest struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Keywords    []string   `json:"keywords,omitempty"`
	Language    string     `json:"language,omitempty"`
}

type KeywordAnalysisRequest struct {
	Keywords   []string `json:"keywords"`
	Language   string   `json:"language,omitempty"`
	Intent     string   `json:"intent,omitempty"`
}

// --- Analysis Results ---

type KeywordAnalysisResult struct {
	Keywords        []SEOKeyword        `json:"keywords"`
	Clusters        []SEOCluster        `json:"clusters,omitempty"`
	Cannibalization []CannibalizationIssue `json:"cannibalization,omitempty"`
	ContentGaps     []ContentGapIssue   `json:"content_gaps,omitempty"`
}

type CannibalizationIssue struct {
	Keyword    string   `json:"keyword"`
	PostIDs    []string `json:"post_ids"`
	Score      float64  `json:"score"`
	Suggestion string   `json:"suggestion"`
}

type ContentGapIssue struct {
	Topic     string   `json:"topic"`
	Cluster   string   `json:"cluster,omitempty"`
	Volume    int      `json:"volume"`
	Priority  string   `json:"priority"`
}

type ContentAnalysisResult struct {
	ReadabilityScore   float64        `json:"readability_score"`
	EEATScore          float64        `json:"eeat_score"`
	FreshnessScore     float64        `json:"freshness_score"`
	Issues             []AuditIssue   `json:"issues"`
	Checklist          []ChecklistItem `json:"checklist"`
}

type TechnicalAnalysisResult struct {
	TitleScore        float64        `json:"title_score"`
	MetaScore         float64        `json:"meta_score"`
	SlugScore         float64        `json:"slug_score"`
	HeadingScore      float64        `json:"heading_score"`
	ImageScore        float64        `json:"image_score"`
	SchemaScore       float64        `json:"schema_score"`
	Issues            []AuditIssue   `json:"issues"`
}

type OrphanArticle struct {
	PostID      uuid.UUID `json:"post_id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	IncomingLinks int     `json:"incoming_links"`
}

type DuplicateContent struct {
	PostID1    uuid.UUID `json:"post_id_1"`
	PostID2    uuid.UUID `json:"post_id_2"`
	Similarity float64   `json:"similarity"`
	Issue      string    `json:"issue"`
}

// --- Dashboard ---

type DashboardStats struct {
	TotalProjects      int            `json:"total_projects"`
	CompletedProjects  int            `json:"completed_projects"`
	AvgSEOScore        float64        `json:"avg_seo_score"`
	AvgReadability     float64        `json:"avg_readability"`
	AvgEEAT            float64        `json:"avg_eeat"`
	PendingImprovements int           `json:"pending_improvements"`
	AppliedImprovements int           `json:"applied_improvements"`
	OrphanArticles     int            `json:"orphan_articles"`
	CannibalizationIssues int         `json:"cannibalization_issues"`
	ContentGaps        int            `json:"content_gaps"`
	ClustersCount      int            `json:"clusters_count"`
	ByLanguage         map[string]int `json:"by_language"`
}

type SEOMetrics struct {
	ByStatus   map[ProjectStatus]int64            `json:"by_status"`
	ByLanguage map[string]int64                   `json:"by_language"`
	ByCategory map[string]int64                   `json:"by_category"`
}

// --- Events ---

const (
	EventSEOProjectCreated    kernel.EventType = "seoengine.project.created"
	EventSEOProjectStarted   kernel.EventType = "seoengine.project.started"
	EventSEOProjectCompleted kernel.EventType = "seoengine.project.completed"
	EventSEOProjectFailed    kernel.EventType = "seoengine.project.failed"
	EventSEOAuditCompleted   kernel.EventType = "seoengine.audit.completed"
	EventSEOImprovementAdded kernel.EventType = "seoengine.improvement.added"
	EventSEOImprovementApplied kernel.EventType = "seoengine.improvement.applied"
)

// --- Errors ---

var (
	ErrProjectNotFound         = errors.New("seo project not found")
	ErrKeywordNotFound         = errors.New("seo keyword not found")
	ErrClusterNotFound         = errors.New("seo cluster not found")
	ErrAuditNotFound           = errors.New("seo audit not found")
	ErrScoreNotFound           = errors.New("seo score not found")
	ErrImprovementNotFound     = errors.New("seo improvement not found")
	ErrDatabaseNotAvail        = errors.New("database not available")
	ErrInvalidLanguage         = errors.New("language must be 'pt' or 'en'")
	ErrInvalidCategory         = errors.New("invalid improvement category")
	ErrInvalidPriority         = errors.New("invalid improvement priority")
	ErrInvalidStatus           = errors.New("invalid improvement status")
	ErrInvalidProjectStatus    = errors.New("invalid project status")
	ErrInvalidContentType      = errors.New("invalid content type")
)
