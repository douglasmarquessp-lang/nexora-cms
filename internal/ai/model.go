package ai

import "time"

type Capability string

const (
	CapGenerate   Capability = "generate"
	CapStream     Capability = "stream"
	CapEmbeddings Capability = "embeddings"
	CapSummarize  Capability = "summarize"
	CapRewrite    Capability = "rewrite"
	CapClassify   Capability = "classify"
)

type ProviderState string

const (
	ProviderHealthy   ProviderState = "healthy"
	ProviderDegraded  ProviderState = "degraded"
	ProviderUnhealthy ProviderState = "unhealthy"
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
}

type CompletionResult struct {
	Content      string        `json:"content"`
	Model        string        `json:"model"`
	ProviderName string        `json:"provider_name"`
	TotalTokens  int           `json:"total_tokens"`
	PromptTokens int           `json:"prompt_tokens"`
	Duration     time.Duration `json:"duration"`
	FinishReason string        `json:"finish_reason,omitempty"`
}

type StreamChunk struct {
	Content    string `json:"content"`
	Done       bool   `json:"done"`
	Error      error  `json:"error,omitempty"`
	Index      int    `json:"index"`
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
	Vector     []float64 `json:"vector"`
	Model      string    `json:"model"`
	Dimensions int       `json:"dimensions"`
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
	Category    string    `json:"category"`
	Confidence  float64   `json:"confidence"`
	Scores      map[string]float64 `json:"scores,omitempty"`
}

type ScoreResult struct {
	Score     float64 `json:"score"`
	MaxScore  float64 `json:"max_score"`
	Passed    bool    `json:"passed"`
	Details   string  `json:"details,omitempty"`
}

type DuplicateResult struct {
	Text     string  `json:"text"`
	Similarity float64 `json:"similarity"`
	Passed   bool    `json:"passed"`
}

type HallucinationResult struct {
	Passed       bool     `json:"passed"`
	Issues       []string `json:"issues,omitempty"`
	Confidence   float64  `json:"confidence"`
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
	Name         string       `json:"name"`
	Model        string       `json:"model"`
	Capabilities []Capability `json:"capabilities"`
	State        ProviderState `json:"state"`
	Priority     int          `json:"priority"`
	Weight       int          `json:"weight"`
	Enabled      bool         `json:"enabled"`
}

type AIMetrics struct {
	TotalRequests   int64         `json:"total_requests"`
	FailedRequests  int64         `json:"failed_requests"`
	TotalTokens     int64         `json:"total_tokens"`
	AvgLatency      time.Duration `json:"avg_latency"`
	ProviderStats   map[string]ProviderMetrics `json:"provider_stats"`
}

type ProviderMetrics struct {
	Requests      int64         `json:"requests"`
	Failed        int64         `json:"failed"`
	AvgLatency    time.Duration `json:"avg_latency"`
	TokensUsed    int64         `json:"tokens_used"`
	CircuitOpens  int64         `json:"circuit_opens"`
}

type ProviderHealthReport struct {
	Providers []HealthStatus `json:"providers"`
	Overall   ProviderState  `json:"overall"`
}

type AITestResult struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	Generate    bool   `json:"generate"`
	Stream      bool   `json:"stream,omitempty"`
	Embeddings  bool   `json:"embeddings,omitempty"`
	Summarize   bool   `json:"summarize,omitempty"`
	Rewrite     bool   `json:"rewrite,omitempty"`
	Classify    bool   `json:"classify,omitempty"`
	Error       string `json:"error,omitempty"`
}

type AIModuleConfig struct {
	Config        AIConfig
	EventBus      interface{ SetEventBus(bus interface{}) }
}

// prompt type constants
const (
	PromptTypeArticle       = "article"
	PromptTypeOutline       = "outline"
	PromptTypeSection       = "section"
	PromptTypeRevision      = "revision"
	PromptTypeFactCheck     = "fact_check"
	PromptTypeSEO           = "seo"
	PromptTypeTranslation   = "translation"
	PromptTypeSummary       = "summary"
	PromptTypeResearch      = "research"
	PromptTypeBriefing      = "briefing"
)
