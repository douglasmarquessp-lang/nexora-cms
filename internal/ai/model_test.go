package ai

import (
	"testing"

	"nexora/internal/kernel"
)

func TestCapabilityConstants(t *testing.T) {
	tests := []struct {
		cap  Capability
		name string
	}{
		{CapGenerate, "generate"},
		{CapStream, "stream"},
		{CapEmbeddings, "embeddings"},
		{CapSummarize, "summarize"},
		{CapRewrite, "rewrite"},
		{CapClassify, "classify"},
	}
	for _, tt := range tests {
		if string(tt.cap) != tt.name {
			t.Errorf("Capability(%s) = %s, want %s", tt.name, string(tt.cap), tt.name)
		}
	}
}

func TestProviderStateConstants(t *testing.T) {
	tests := []struct {
		state ProviderState
		name  string
	}{
		{ProviderHealthy, "healthy"},
		{ProviderDegraded, "degraded"},
		{ProviderUnhealthy, "unhealthy"},
		{ProviderCircuitOpen, "circuit_open"},
	}
	for _, tt := range tests {
		if string(tt.state) != tt.name {
			t.Errorf("ProviderState(%s) = %s, want %s", tt.name, string(tt.state), tt.name)
		}
	}
}

func TestPromptTypeConstants(t *testing.T) {
	tests := []struct {
		pt   string
		name string
	}{
		{PromptTypeArticle, "article"},
		{PromptTypeOutline, "outline"},
		{PromptTypeSection, "section"},
		{PromptTypeRevision, "revision"},
		{PromptTypeFactCheck, "fact_check"},
		{PromptTypeSEO, "seo"},
		{PromptTypeTranslation, "translation"},
		{PromptTypeSummary, "summary"},
		{PromptTypeResearch, "research"},
		{PromptTypeBriefing, "briefing"},
	}
	for _, tt := range tests {
		if tt.pt != tt.name {
			t.Errorf("PromptType(%s) = %s, want %s", tt.name, tt.pt, tt.name)
		}
	}
}

func TestSentinelErrors(t *testing.T) {
	errs := []error{
		ErrProviderNotFound,
		ErrAllProvidersFailed,
		ErrProviderUnavailable,
		ErrCircuitBreakerOpen,
		ErrTimeout,
		ErrCancelled,
		ErrStreamInterrupted,
		ErrInvalidPromptTemplate,
		ErrInvalidModel,
		ErrRateLimited,
		ErrContextLength,
		ErrInvalidAPIKey,
		ErrProviderConfig,
		ErrEmbeddingNotSupported,
		ErrStreamingNotSupported,
		ErrClassificationNotSupported,
		ErrRewriteNotSupported,
		ErrHealthCheckFailed,
		ErrNoProvidersRegistered,
		ErrInvalidPromptVariables,
		ErrUnsupportedLanguage,
		ErrInvalidCapability,
	}
	for _, err := range errs {
		if err == nil {
			t.Error("sentinel error should not be nil")
		}
	}
}

func TestAIEventTypes(t *testing.T) {
	events := []struct {
		event kernel.EventType
		name  string
	}{
		{EventAIStarted, "ai.started"},
		{EventAICompleted, "ai.completed"},
		{EventAIFailed, "ai.failed"},
		{EventAIStreaming, "ai.streaming"},
		{EventAIRetry, "ai.retry"},
		{EventAITimeout, "ai.timeout"},
		{EventAIProviderChanged, "ai.provider.changed"},
	}
	for _, tt := range events {
		if string(tt.event) != tt.name {
			t.Errorf("Event(%s) = %s, want %s", tt.name, string(tt.event), tt.name)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Retry.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts=3, got %d", cfg.Retry.MaxAttempts)
	}
	if cfg.CircuitBreaker.FailureThreshold != 5 {
		t.Errorf("expected FailureThreshold=5, got %d", cfg.CircuitBreaker.FailureThreshold)
	}
	if cfg.Enabled {
		t.Error("expected AI disabled by default")
	}
}

func TestPipelineStageConstants(t *testing.T) {
	stages := []struct {
		stage PipelineStage
		name  string
	}{
		{StageResearchGen, "research"},
		{StageBriefingGen, "briefing"},
		{StageOutlineGen, "outline"},
		{StageDraftGen, "draft"},
		{StageSEOGen, "seo"},
		{StageQualityCheck, "quality"},
		{StageTranslationGen, "translation"},
		{StageFinalReview, "final_review"},
	}
	for _, tt := range stages {
		if v, ok := stageNames[tt.stage]; !ok || v != tt.name {
			t.Errorf("PipelineStage(%d) name = %s, want %s", tt.stage, v, tt.name)
		}
	}
}
