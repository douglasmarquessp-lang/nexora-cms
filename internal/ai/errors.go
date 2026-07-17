package ai

import "errors"

var (
	ErrProviderNotFound       = errors.New("AI provider not found")
	ErrAllProvidersFailed     = errors.New("all AI providers failed")
	ErrProviderUnavailable    = errors.New("provider not available")
	ErrCircuitBreakerOpen     = errors.New("circuit breaker open")
	ErrTimeout                = errors.New("AI request timed out")
	ErrCancelled              = errors.New("AI request cancelled")
	ErrStreamInterrupted      = errors.New("stream interrupted")
	ErrInvalidPromptTemplate  = errors.New("invalid prompt template")
	ErrInvalidModel           = errors.New("invalid model for provider")
	ErrRateLimited            = errors.New("rate limited by provider")
	ErrContextLength          = errors.New("context length exceeded")
	ErrInvalidAPIKey          = errors.New("invalid API key")
	ErrProviderConfig         = errors.New("invalid provider configuration")
	ErrEmbeddingNotSupported  = errors.New("embeddings not supported by provider")
	ErrStreamingNotSupported  = errors.New("streaming not supported by provider")
	ErrClassificationNotSupported = errors.New("classification not supported by provider")
	ErrRewriteNotSupported    = errors.New("rewrite not supported by provider")
	ErrHealthCheckFailed      = errors.New("health check failed")
	ErrNoProvidersRegistered  = errors.New("no AI providers registered")
	ErrInvalidPromptVariables = errors.New("invalid prompt template variables")
	ErrUnsupportedLanguage    = errors.New("unsupported language for prompt template")
	ErrInvalidCapability      = errors.New("invalid capability")
)
