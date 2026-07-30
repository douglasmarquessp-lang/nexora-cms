package ai

import "time"

// GroundingConfig controls whether to use web search/grounding for a completion request.
type GroundingConfig struct {
	Enabled        bool     `json:"enabled"`
	MaxSources     int      `json:"max_sources,omitempty"`
	ExcludeDomains []string `json:"exclude_domains,omitempty"`
}

// GroundingMetadata contains information about web sources used to support the generated content.
type GroundingMetadata struct {
	Sources          []GroundingSource `json:"sources,omitempty"`
	SearchSuggested  bool              `json:"search_suggested,omitempty"`
	SearchEntryPoint *SearchEntryPoint `json:"search_entry_point,omitempty"`
	SupportSegments  []GroundingSupport `json:"support_segments,omitempty"`
	Unverified       bool              `json:"unverified,omitempty"`
}

// GroundingSource represents a single web source used for grounding.
type GroundingSource struct {
	URI           string    `json:"uri"`
	Title         string    `json:"title,omitempty"`
	Snippet       string    `json:"snippet,omitempty"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	FreshnessScore float64  `json:"freshness_score,omitempty"`
	IsVerified    bool     `json:"is_verified,omitempty"`
	DomainRank    int      `json:"domain_rank,omitempty"`
	RetrievedAt   time.Time `json:"retrieved_at,omitempty"`
}

// SearchEntryPoint contains information about an entry point for the grounding search.
type SearchEntryPoint struct {
	Query  string `json:"query,omitempty"`
	URL    string `json:"url,omitempty"`
	RenderedHTML string `json:"rendered_html,omitempty"`
}

// GroundingSupport describes which parts of the generated content are supported by specific sources.
type GroundingSupport struct {
	Segment       string    `json:"segment,omitempty"`
	SourceIndices []int     `json:"source_indices,omitempty"`
	Confidence    float64   `json:"confidence,omitempty"`
}

type Capability string

const (
	CapGenerate   Capability = "generate"
	CapStream     Capability = "stream"
	CapEmbeddings Capability = "embeddings"
	CapSummarize  Capability = "summarize"
	CapRewrite    Capability = "rewrite"
	CapClassify   Capability = "classify"
	CapGrounding  Capability = "grounding"
)

type ProviderState string

const (
	ProviderHealthy     ProviderState = "healthy"
	ProviderDegraded    ProviderState = "degraded"
	ProviderUnhealthy   ProviderState = "unhealthy"
	ProviderCircuitOpen ProviderState = "circuit_open"
)

type CompletionRequest struct {
	Model       string            `json:"model,omitempty"`
	Prompt      string            `json:"prompt"`
	System      string            `json:"system,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	StopWords   []string          `json:"stop_words,omitempty"`
	Variables   map[string]string `json:"variables,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
	Grounding   *GroundingConfig  `json:"grounding,omitempty"`
}

type CompletionResult struct {
	Content           string             `json:"content"`
	Model             string             `json:"model"`
	ProviderName      string             `json:"provider_name"`
	TotalTokens       int                `json:"total_tokens"`
	PromptTokens      int                `json:"prompt_tokens"`
	Duration          time.Duration      `json:"duration"`
	FinishReason      string             `json:"finish_reason,omitempty"`
	GroundingMetadata *GroundingMetadata `json:"grounding_metadata,omitempty"`
}

type StreamChunk struct {
	Content      string `json:"content"`
	Done         bool   `json:"done"`
	Error        error  `json:"error,omitempty"`
	Index        int    `json:"index"`
	FinishReason string `json:"finish_reason,omitempty"`
}

type HealthStatus struct {
	Provider string        `json:"provider"`
	State    ProviderState `json:"state"`
	Latency  time.Duration `json:"latency"`
	Message  string        `json:"message,omitempty"`
	Model    string        `json:"model,omitempty"`
}

type EmbeddingResult struct {
	Vector     []float64     `json:"vector"`
	Model      string        `json:"model"`
	Dimensions int           `json:"dimensions"`
	Duration   time.Duration `json:"duration"`
}

type SummarizeRequest struct {
	Text     string `json:"text"`
	MaxWords int    `json:"max_words"`
	Language string `json:"language,omitempty"`
}

type RewriteRequest struct {
	Text         string `json:"text"`
	Instructions string `json:"instructions"`
	Tone         string `json:"tone,omitempty"`
	Audience     string `json:"audience,omitempty"`
}

type ClassifyRequest struct {
	Text       string   `json:"text"`
	Categories []string `json:"categories"`
}

type ClassifyResult struct {
	Category   string             `json:"category"`
	Confidence float64            `json:"confidence"`
	Scores     map[string]float64 `json:"scores,omitempty"`
}

type ScoreResult struct {
	Score    float64 `json:"score"`
	MaxScore float64 `json:"max_score"`
	Passed   bool    `json:"passed"`
	Details  string  `json:"details,omitempty"`
}

type DuplicateResult struct {
	Text       string  `json:"text"`
	Similarity float64 `json:"similarity"`
	Passed     bool    `json:"passed"`
}

type HallucinationResult struct {
	Passed     bool     `json:"passed"`
	Issues     []string `json:"issues,omitempty"`
	Confidence float64  `json:"confidence"`
}

// QualityCheckSource indicates whether a check was deterministic or AI-assisted.
type QualityCheckSource string

const (
	SourceDeterministic QualityCheckSource = "deterministic"
	SourceAI            QualityCheckSource = "ai_assisted"
	SourceHybrid        QualityCheckSource = "hybrid"
)

// QualityCheckItem represents a single finding from any quality check.
type QualityCheckItem struct {
	Category    string             `json:"category"`
	CheckName   string             `json:"check_name"`
	Severity    string             `json:"severity"` // error, warning, info
	Score       float64            `json:"score"`
	MaxScore    float64            `json:"max_score"`
	Passed      bool               `json:"passed"`
	Message     string             `json:"message,omitempty"`
	Suggestion  string             `json:"suggestion,omitempty"`
	Source      QualityCheckSource `json:"source"`
	Details     string             `json:"details,omitempty"`
}

// GrammarReport contains detailed grammar analysis results.
type GrammarReport struct {
	OverallScore float64            `json:"overall_score"`
	MaxScore     float64            `json:"max_score"`
	Passed       bool               `json:"passed"`
	Items        []QualityCheckItem `json:"items"`
	Issues       []GrammarIssue     `json:"issues"`
	AIAssisted   bool               `json:"ai_assisted,omitempty"`
}

// GrammarIssue represents a specific grammar issue found.
type GrammarIssue struct {
	Type        string `json:"type"` // capitalization, repeated_word, punctuation, spelling, syntax
	Word        string `json:"word,omitempty"`
	Position    int    `json:"position,omitempty"`
	Context     string `json:"context,omitempty"`
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion,omitempty"`
	Severity    string `json:"severity"`
}

// SEOAnalysis contains detailed SEO assessment dimensions.
type SEOAnalysis struct {
	OverallScore    float64            `json:"overall_score"`
	MaxScore        float64            `json:"max_score"`
	Passed          bool               `json:"passed"`
	Items           []QualityCheckItem `json:"items"`
	TitleScore      *SEOTitleScore     `json:"title_score,omitempty"`
	HeadingsScore   *SEOHeadingsScore  `json:"headings_score,omitempty"`
	KeywordUsage    *SEOKeywordUsage   `json:"keyword_usage,omitempty"`
	MetaDescScore   *SEOMetaDescScore  `json:"meta_description_score,omitempty"`
	ContentScore    *SEOContentScore   `json:"content_structure_score,omitempty"`
	IntentScore     *SEOIntentScore    `json:"search_intent_score,omitempty"`
	AIAssisted      bool               `json:"ai_assisted,omitempty"`
}

// SEOTitleScore rates the title/heading for SEO.
type SEOTitleScore struct {
	Title          string   `json:"title"`
	Length         int      `json:"length"`
	LengthScore    float64  `json:"length_score"`    // ideal: 50-60 chars
	KeywordScore   float64  `json:"keyword_score"`   // primary keyword present?
	PositionScore  float64  `json:"position_score"`  // keyword at beginning?
	Score          float64  `json:"score"`
	MaxScore       float64  `json:"max_score"`
	Passed         bool     `json:"passed"`
	Message        string   `json:"message,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}

// SEOHeadingsScore evaluates heading structure and keyword use in headings.
type SEOHeadingsScore struct {
	H1Count        int       `json:"h1_count"`
	H2Count        int       `json:"h2_count"`
	H3Count        int       `json:"h3_count"`
	KeywordInH1    bool      `json:"keyword_in_h1"`
	KeywordInH2    int       `json:"keyword_in_h2"` // count of H2s with keyword
	HasHierarchy   bool      `json:"has_hierarchy"`  // H1 > H2 > H3 order respected
	Score          float64   `json:"score"`
	MaxScore       float64   `json:"max_score"`
	Passed         bool      `json:"passed"`
	Message        string    `json:"message,omitempty"`
	Warnings       []string  `json:"warnings,omitempty"`
}

// SEOKeywordUsage measures keyword placement and density.
type SEOKeywordUsage struct {
	Keywords        []string  `json:"keywords"`
	Density         float64   `json:"density"`          // overall keyword density %
	First100Kw      bool      `json:"first_100_words"`   // keyword appears in first 100 words
	DensityScore    float64   `json:"density_score"`     // ideal 1-3%
	PlacementScore  float64   `json:"placement_score"`
	Score           float64   `json:"score"`
	MaxScore        float64   `json:"max_score"`
	Passed          bool      `json:"passed"`
	Message         string    `json:"message,omitempty"`
	Warnings        []string  `json:"warnings,omitempty"`
}

// SEOMetaDescScore evaluates meta description quality.
type SEOMetaDescScore struct {
	HasMetaDesc     bool      `json:"has_meta_description"`
	Length          int       `json:"length"`
	LengthScore     float64   `json:"length_score"`      // ideal 150-160 chars
	KeywordPresence bool      `json:"keyword_presence"`
	Score           float64   `json:"score"`
	MaxScore        float64   `json:"max_score"`
	Passed          bool      `json:"passed"`
	Message         string    `json:"message,omitempty"`
	Warnings        []string  `json:"warnings,omitempty"`
}

// SEOContentScore evaluates content structure for SEO.
type SEOContentScore struct {
	ParagraphCount  int       `json:"paragraph_count"`
	HasLists        bool      `json:"has_lists"`
	ListCount       int       `json:"list_count,omitempty"`
	HasLinks        bool      `json:"has_links"`
	InternalLinks   int       `json:"internal_links"`
	ExternalLinks   int       `json:"external_links"`
	HasImages       bool      `json:"has_images"`
	ImageAltCount   int       `json:"image_alt_count,omitempty"`
	Score           float64   `json:"score"`
	MaxScore        float64   `json:"max_score"`
	Passed          bool      `json:"passed"`
	Message         string    `json:"message,omitempty"`
	Warnings        []string  `json:"warnings,omitempty"`
}

// SEOIntentScore evaluates search intent alignment.
type SEOIntentScore struct {
	DetectedIntent  string    `json:"detected_intent"`  // informational, navigational, commercial, transactional
	Score           float64   `json:"score"`
	MaxScore        float64   `json:"max_score"`
	Passed          bool      `json:"passed"`
	Message         string    `json:"message,omitempty"`
	AIAssisted      bool      `json:"ai_assisted,omitempty"`
}

// ReadabilityReport contains detailed readability metrics.
type ReadabilityReport struct {
	OverallScore          float64  `json:"overall_score"`
	MaxScore              float64  `json:"max_score"`
	Passed                bool     `json:"passed"`
	FleschReadingEase     float64  `json:"flesch_reading_ease"`
	FleschKincaidGrade    float64  `json:"flesch_kincaid_grade"`
	WordCount             int      `json:"word_count"`
	SentenceCount         int      `json:"sentence_count"`
	SyllableCount         int      `json:"syllable_count"`
	AvgWordsPerSentence   float64  `json:"avg_words_per_sentence"`
	AvgSyllablesPerWord   float64  `json:"avg_syllables_per_word"`
	DifficultWordCount    int      `json:"difficult_word_count"`
	DifficultWordPercent  float64  `json:"difficult_word_percent"`
	Items                 []QualityCheckItem `json:"items"`
}

// StructureReport contains detailed structure validation results.
type StructureReport struct {
	OverallScore     float64            `json:"overall_score"`
	MaxScore         float64            `json:"max_score"`
	Passed           bool               `json:"passed"`
	Items            []QualityCheckItem `json:"items"`
	HeadingIssues    []StructureIssue   `json:"heading_issues,omitempty"`
	ParagraphIssues  []StructureIssue   `json:"paragraph_issues,omitempty"`
	ListCount        int                `json:"list_count"`
	LinkCount        int                `json:"link_count"`
	BrokenLinkCount  int                `json:"broken_link_count"`
	ImageCount       int                `json:"image_count"`
	HasConclusion    bool               `json:"has_conclusion"`
	CompletenessPct  float64            `json:"completeness_pct"`
}

// StructureIssue describes a structure problem.
type StructureIssue struct {
	Type       string `json:"type"` // heading_order, missing_h1, paragraph_length, link_format, incomplete
	Element    string `json:"element,omitempty"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	Severity   string `json:"severity"`
}

// DuplicateBlock represents a detected duplicate content block.
type DuplicateBlock struct {
	Text       string  `json:"text"`
	Similarity float64 `json:"similarity"`
	Offset     int     `json:"offset"`
	Length     int     `json:"length"`
	Passed     bool    `json:"passed"`
}

// FactCheckReport contains fact-checking results using grounding sources.
type FactCheckReport struct {
	Passed        bool                `json:"passed"`
	OverallScore  float64             `json:"overall_score"`
	MaxScore      float64             `json:"max_score"`
	ClaimsChecked int                 `json:"claims_checked"`
	Supported     int                 `json:"supported"`
	Unsupported   int                 `json:"unsupported"`
	Contradicted  int                 `json:"contradicted"`
	Unverifiable  int                 `json:"unverifiable"`
	Items         []FactCheckItem     `json:"items,omitempty"`
	Grounded      bool                `json:"grounded"`
	GroundingMeta *GroundingMetadata  `json:"grounding_metadata,omitempty"`
}

// FactCheckItem represents a single claim assessment.
type FactCheckItem struct {
	Claim         string  `json:"claim"`
	Verdict       string  `json:"verdict"` // supported, unsupported, contradicted, unverifiable
	Confidence    float64 `json:"confidence"`
	Source        string  `json:"source,omitempty"`
	Suggestion    string  `json:"suggestion,omitempty"`
	SourceQuality string  `json:"source_quality,omitempty"` // verified, unverified
}

type StructureSpec struct {
	RequiredSections []string `json:"required_sections"`
	MinWords         int      `json:"min_words"`
	MaxWords         int      `json:"max_words"`
	MinParagraphs    int      `json:"min_paragraphs"`
	HasIntro         bool     `json:"has_intro"`
	HasConclusion    bool     `json:"has_conclusion"`
}

type PromptTemplate struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Language    string            `json:"language"`
	System      string            `json:"system"`
	Template    string            `json:"template"`
	Variables   []string          `json:"variables"`
	Defaults    map[string]string `json:"defaults,omitempty"`
	Version     string            `json:"version"`
}

type ProviderInfo struct {
	Name         string        `json:"name"`
	Model        string        `json:"model"`
	Capabilities []Capability  `json:"capabilities"`
	State        ProviderState `json:"state"`
	Priority     int           `json:"priority"`
	Weight       int           `json:"weight"`
	Enabled      bool          `json:"enabled"`
}

type AIMetrics struct {
	TotalRequests  int64                      `json:"total_requests"`
	FailedRequests int64                      `json:"failed_requests"`
	TotalTokens    int64                      `json:"total_tokens"`
	AvgLatency     time.Duration              `json:"avg_latency"`
	ProviderStats  map[string]ProviderMetrics `json:"provider_stats"`
}

type ProviderMetrics struct {
	Requests     int64         `json:"requests"`
	Failed       int64         `json:"failed"`
	AvgLatency   time.Duration `json:"avg_latency"`
	TokensUsed   int64         `json:"tokens_used"`
	CircuitOpens int64         `json:"circuit_opens"`
}

type ProviderHealthReport struct {
	Providers []HealthStatus `json:"providers"`
	Overall   ProviderState  `json:"overall"`
}

type AITestResult struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Generate   bool   `json:"generate"`
	Stream     bool   `json:"stream,omitempty"`
	Embeddings bool   `json:"embeddings,omitempty"`
	Summarize  bool   `json:"summarize,omitempty"`
	Rewrite    bool   `json:"rewrite,omitempty"`
	Classify   bool   `json:"classify,omitempty"`
	Error      string `json:"error,omitempty"`
}

type AIModuleConfig struct {
	Config   AIConfig
	EventBus interface{ SetEventBus(bus interface{}) }
}

// prompt type constants
const (
	PromptTypeArticle          = "article"
	PromptTypeOutline          = "outline"
	PromptTypeSection          = "section"
	PromptTypeRevision         = "revision"
	PromptTypeFactCheck        = "fact_check"
	PromptTypeSEO              = "seo"
	PromptTypeTranslation      = "translation"
	PromptTypeSummary          = "summary"
	PromptTypeResearch         = "research"
	PromptTypeBriefing         = "briefing"
	PromptTypeTopic            = "topic"
	PromptTypeQualityGrammar   = "quality_grammar"
	PromptTypeQualitySEO       = "quality_seo"
	PromptTypeQualityReadability = "quality_readability"
	PromptTypeQualityIntent    = "quality_intent"
)
