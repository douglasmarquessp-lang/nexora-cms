package ai

import "nexora/internal/kernel"

const (
	EventAIStarted         kernel.EventType = "ai.started"
	EventAICompleted       kernel.EventType = "ai.completed"
	EventAIFailed          kernel.EventType = "ai.failed"
	EventAIStreaming       kernel.EventType = "ai.streaming"
	EventAIRetry           kernel.EventType = "ai.retry"
	EventAITimeout         kernel.EventType = "ai.timeout"
	EventAIProviderChanged kernel.EventType = "ai.provider.changed"
)
