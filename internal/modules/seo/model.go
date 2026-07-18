package seo

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "seo"

type ProjectStatus string

const (
	ProjPending    ProjectStatus = "pending"
	ProjRunning    ProjectStatus = "running"
	ProjCompleted  ProjectStatus = "completed"
	ProjFailed     ProjectStatus = "failed"
)

type KeywordType string

const (
	KWPrimary   KeywordType = "primary"
	KWSecondary KeywordType = "secondary"
	KWSemantic  KeywordType = "semantic"
)

type SearchIntent string

const (
	IntentInformational SearchIntent = "informational"
	IntentNavigational  SearchIntent = "navigational"
	IntentCommercial    SearchIntent = "commercial"
	IntentTransactional SearchIntent = "transactional"
)

type LinkType string

const (
	LinkSuggestion LinkType = "suggestion"
	LinkRelated    LinkType = "related"
	LinkOptimized  LinkType = "optimized"
)

type SEOScoreCategory string

const (
	CatKeyword    SEOScoreCategory = "keyword"
	CatContent    SEOScoreCategory = "content"
	CatTechnical  SEOScoreCategory = "technical"
	CatLinking    SEOScoreCategory = "linking"
	CatReadability SEOScoreCategory = "readability"
	CatMetadata   SEOScoreCategory = "metadata"
)

type SchemaType string

const (
	SchemaArticle    SchemaType = "Article"
	SchemaFAQ        SchemaType = "FAQPage"
	SchemaBreadcrumb SchemaType = "BreadcrumbList"
)

// --- Models ---

type SEOProject struct {
	ID              uuid.UUID  `json:"id"`
	SiteID          uuid.UUID  `json:"site_id"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`
	Title           string     `json:"title"`
	TargetURL       string     `json:"target_url,omitempty"`
	PostID          *uuid.UUID `json:"post_id,omitempty"`
	Language        string     `json:"language"`
	Status          ProjectStatus `json:"status"`
	SEOScore        float64    `json:"seo_score"`
	ReadabilityScore float64   `json:"readability_score"`
	KeywordDensity  float64    `json:"keyword_density"`
	ContentQuality  float64    `json:"content_quality"`
	TechnicalScore  float64    `json:"technical_score"`
	Recommendations []string   `json:"recommendations,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Keywords        []Keyword  `json:"keywords,omitempty"`
	Audit           *Audit     `json:"audit,omitempty"`
	Metadata        *Metadata  `json:"metadata,omitempty"`
	Links           []InternalLink `json:"links,omitempty"`
	Score           *Score     `json:"score,omitempty"`
}

type Keyword struct {
	ID           uuid.UUID  `json:"id"`
	SiteID       uuid.UUID  `json:"site_id"`
	ProjectID    *uuid.UUID `json:"seo_project_id,omitempty"`
	Keyword      string     `json:"keyword"`
	KeywordType  KeywordType `json:"keyword_type"`
	SearchIntent SearchIntent `json:"search_intent"`
	Volume       int        `json:"volume"`
	Difficulty   float64    `json:"difficulty"`
	Density      float64    `json:"density"`
	Frequency    int        `json:"frequency"`
	Prominence   float64    `json:"prominence"`
	Entities     []string   `json:"entities,omitempty"`
	Language     string     `json:"language"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Cluster struct {
	ID           uuid.UUID `json:"id"`
	SiteID       uuid.UUID `json:"site_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Keywords     []string  `json:"keywords,omitempty"`
	ArticleCount int       `json:"article_count"`
	AvgScore     float64   `json:"avg_score"`
	Language     string    `json:"language"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Audit struct {
	ID                   uuid.UUID              `json:"id"`
	SiteID               uuid.UUID              `json:"site_id"`
	ProjectID            *uuid.UUID             `json:"seo_project_id,omitempty"`
	PostID               *uuid.UUID             `json:"post_id,omitempty"`
	URL                  string                 `json:"url,omitempty"`
	TitleScore           float64                `json:"title_score"`
	MetaScore            float64                `json:"meta_score"`
	HeadingScore         float64                `json:"heading_score"`
	ParagraphScore       float64                `json:"paragraph_score"`
	ReadabilityScore     float64                `json:"readability_score"`
	PassiveVoiceScore    float64                `json:"passive_voice_score"`
	SentenceVariationScore float64              `json:"sentence_variation_score"`
	DuplicateScore       float64                `json:"duplicate_score"`
	OverallScore         float64                `json:"overall_score"`
	Issues               []map[string]interface{} `json:"issues,omitempty"`
	Recommendations      []string               `json:"recommendations,omitempty"`
	Language             string                 `json:"language"`
	AuditedAt            time.Time              `json:"audited_at"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

type InternalLink struct {
	ID           uuid.UUID  `json:"id"`
	SiteID       uuid.UUID  `json:"site_id"`
	ProjectID    *uuid.UUID `json:"seo_project_id,omitempty"`
	SourceURL    string     `json:"source_url"`
	TargetURL    string     `json:"target_url"`
	AnchorText   string     `json:"anchor_text,omitempty"`
	LinkType     LinkType   `json:"link_type"`
	Relevance    float64    `json:"relevance"`
	IsExisting   bool       `json:"is_existing"`
	IsImplemented bool      `json:"is_implemented"`
	Language     string     `json:"language"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Metadata struct {
	ID               uuid.UUID              `json:"id"`
	SiteID           uuid.UUID              `json:"site_id"`
	ProjectID        *uuid.UUID             `json:"seo_project_id,omitempty"`
	PostID           *uuid.UUID             `json:"post_id,omitempty"`
	TitleTag         string                 `json:"title_tag,omitempty"`
	MetaDescription  string                 `json:"meta_description,omitempty"`
	CanonicalURL     string                 `json:"canonical_url,omitempty"`
	OGTitle          string                 `json:"og_title,omitempty"`
	OGDescription    string                 `json:"og_description,omitempty"`
	OGImage          string                 `json:"og_image,omitempty"`
	TwitterTitle     string                 `json:"twitter_title,omitempty"`
	TwitterDescription string               `json:"twitter_description,omitempty"`
	TwitterImage     string                 `json:"twitter_image,omitempty"`
	JSONLD           map[string]interface{} `json:"json_ld,omitempty"`
	FAQSchema        []map[string]interface{} `json:"faq_schema,omitempty"`
	BreadcrumbSchema []map[string]interface{} `json:"breadcrumb_schema,omitempty"`
	ArticleSchema    map[string]interface{} `json:"article_schema,omitempty"`
	Hreflang         []map[string]interface{} `json:"hreflang,omitempty"`
	RobotsDirectives []string               `json:"robots_directives,omitempty"`
	Language         string                 `json:"language"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type Score struct {
	ID               uuid.UUID  `json:"id"`
	SiteID           uuid.UUID  `json:"site_id"`
	ProjectID        *uuid.UUID `json:"seo_project_id,omitempty"`
	PostID           *uuid.UUID `json:"post_id,omitempty"`
	TotalScore       float64    `json:"total_score"`
	KeywordScore     float64    `json:"keyword_score"`
	ContentScore     float64    `json:"content_score"`
	TechnicalScore   float64    `json:"technical_score"`
	LinkingScore     float64    `json:"linking_score"`
	ReadabilityScore float64    `json:"readability_score"`
	MetadataScore    float64    `json:"metadata_score"`
	Language         string     `json:"language"`
	ScoredAt         time.Time  `json:"scored_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

// --- DTOs ---

type CreateProjectRequest struct {
	Title    string     `json:"title"`
	TargetURL string    `json:"target_url,omitempty"`
	PostID   *uuid.UUID `json:"post_id,omitempty"`
	Language string     `json:"language,omitempty"`
}

type UpdateProjectRequest struct {
	Title     *string `json:"title,omitempty"`
	TargetURL *string `json:"target_url,omitempty"`
}

type KeywordAnalysisRequest struct {
	Content     string   `json:"content"`
	Language    string   `json:"language"`
	PrimaryKW   string   `json:"primary_keyword,omitempty"`
	SecondaryKW []string `json:"secondary_keywords,omitempty"`
}

type SEOAnalysisRequest struct {
	Content      string               `json:"content"`
	Title        string               `json:"title,omitempty"`
	MetaDesc     string               `json:"meta_description,omitempty"`
	Language     string               `json:"language"`
	PrimaryKW    string               `json:"primary_keyword,omitempty"`
}

type LinkSuggestionRequest struct {
	Content string `json:"content"`
	SiteURL string `json:"site_url"`
	PostID  *uuid.UUID `json:"post_id,omitempty"`
	Language string `json:"language"`
}

type TechnicalSEORequest struct {
	Title      string                 `json:"title"`
	MetaDesc   string                 `json:"meta_description"`
	Content    string                 `json:"content"`
	URL        string                 `json:"url"`
	SiteName   string                 `json:"site_name"`
	ImageURL   string                 `json:"image_url,omitempty"`
	Language   string                 `json:"language"`
	AltLang    string                 `json:"alt_language,omitempty"`
	FAQs       []map[string]string    `json:"faqs,omitempty"`
	Categories []string               `json:"categories,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	Author     string                 `json:"author,omitempty"`
}

type ContentScoreResult struct {
	TotalScore      float64            `json:"total_score"`
	KeywordScore    float64            `json:"keyword_score"`
	ContentScore    float64            `json:"content_score"`
	TechnicalScore  float64            `json:"technical_score"`
	LinkingScore    float64            `json:"linking_score"`
	ReadabilityScore float64           `json:"readability_score"`
	MetadataScore   float64            `json:"metadata_score"`
	Breakdown       map[string]float64 `json:"breakdown"`
	Recommendations []string           `json:"recommendations"`
}

type DashboardResponse struct {
	Projects        []SEOProject  `json:"projects"`
	RecentAudits    []Audit       `json:"recent_audits"`
	TopIssues       []string      `json:"top_issues"`
	AvgSEOScore     float64       `json:"avg_seo_score"`
	AvgReadability  float64       `json:"avg_readability"`
	PendingProjects int           `json:"pending_projects"`
	CompletedToday  int           `json:"completed_today"`
}

type SEOMetrics struct {
	TotalProjects   int     `json:"total_projects"`
	CompletedProjects int   `json:"completed_projects"`
	AvgScore        float64 `json:"avg_score"`
	AvgKeywordScore float64 `json:"avg_keyword_score"`
	AvgReadability  float64 `json:"avg_readability"`
	AvgTechnical    float64 `json:"avg_technical"`
	TotalKeywords   int     `json:"total_keywords"`
	TotalLinks      int     `json:"total_links"`
}

// --- Events ---

const (
	EventSEOStarted    kernel.EventType = "seo.started"
	EventSEOCompleted  kernel.EventType = "seo.completed"
	EventSEOFailed     kernel.EventType = "seo.failed"
	EventSEOAnalyzed   kernel.EventType = "seo.analyzed"
	EventSEOKeywords   kernel.EventType = "seo.keywords.extracted"
	EventSEOLinks      kernel.EventType = "seo.links.suggested"
	EventSEOScored     kernel.EventType = "seo.scored"
	EventSEOMetadataGen kernel.EventType = "seo.metadata.generated"
)

// --- Errors ---

var (
	ErrProjectNotFound       = errors.New("seo project not found")
	ErrKeywordNotFound       = errors.New("keyword not found")
	ErrClusterNotFound       = errors.New("cluster not found")
	ErrAuditNotFound         = errors.New("audit not found")
	ErrLinkNotFound          = errors.New("internal link not found")
	ErrMetadataNotFound      = errors.New("metadata not found")
	ErrScoreNotFound         = errors.New("score not found")
	ErrEmptyContent          = errors.New("content is required")
	ErrInvalidLanguage       = errors.New("language must be 'pt' or 'en'")
	ErrEmptyTitle            = errors.New("title is required")
	ErrDatabaseNotAvail      = errors.New("database not available")
	ErrProjectAlreadyRunning = errors.New("project already running")
	ErrProjectAlreadyCompleted = errors.New("project already completed")
)
