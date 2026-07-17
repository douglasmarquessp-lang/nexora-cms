package ai

import "context"

type AIProvider interface {
	Generate(ctx context.Context, req CompletionRequest) (*CompletionResult, error)
	GenerateStream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
	Embeddings(ctx context.Context, input string) (*EmbeddingResult, error)
	Summarize(ctx context.Context, req SummarizeRequest) (string, error)
	Rewrite(ctx context.Context, req RewriteRequest) (string, error)
	Classify(ctx context.Context, req ClassifyRequest) (*ClassifyResult, error)
	Health(ctx context.Context) (*HealthStatus, error)
	Name() string
	Capabilities() []Capability
}

type QualityChecker interface {
	ScoreGrammar(ctx context.Context, text string, language string) (*ScoreResult, error)
	ScoreSEO(ctx context.Context, text string, keywords []string) (*ScoreResult, error)
	ScoreReadability(ctx context.Context, text string, language string) (*ScoreResult, error)
	ScoreEntityCoverage(ctx context.Context, text string, entities []string) (*ScoreResult, error)
	CheckStructure(ctx context.Context, text string, spec StructureSpec) (*ScoreResult, error)
	CheckDuplicates(ctx context.Context, text string) ([]DuplicateResult, error)
	CheckHallucination(ctx context.Context, text string, references []string) (*HallucinationResult, error)
}

type PromptBuilder interface {
	Build(ctx context.Context, templateID string, variables map[string]string) (*CompletionRequest, error)
	RegisterTemplate(template PromptTemplate) error
	ListTemplates(language string) ([]PromptTemplate, error)
	GetTemplate(id string) (*PromptTemplate, error)
}

type StreamHandler interface {
	HandleChunk(chunk StreamChunk) error
	OnComplete(result *CompletionResult) error
	OnError(err error) error
	OnProgress(progress float64) error
}

type AIManager interface {
	Provider(name string) (AIProvider, error)
	DefaultProvider() (AIProvider, error)
	Generate(ctx context.Context, req CompletionRequest) (*CompletionResult, error)
	GenerateStream(ctx context.Context, req CompletionRequest, handler StreamHandler) error
	Embeddings(ctx context.Context, input string) (*EmbeddingResult, error)
	Summarize(ctx context.Context, req SummarizeRequest) (string, error)
	Rewrite(ctx context.Context, req RewriteRequest) (string, error)
	Classify(ctx context.Context, req ClassifyRequest) (*ClassifyResult, error)
	Health(ctx context.Context) (*ProviderHealthReport, error)
	ListProviders() []ProviderInfo
	RegisterProvider(provider AIProvider, config ProviderCfg) error
	SetDefaultProvider(name string) error
	Metrics() AIMetrics
	Quality() QualityChecker
	Prompts() PromptBuilder
	Capabilities() []Capability
}
