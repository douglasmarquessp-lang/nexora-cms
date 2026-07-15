package kernel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"nexora/internal/pkg/logger"
)

type EventType string

const (
	EventUserRegistered EventType = "user.registered"
	EventUserLogin      EventType = "user.login"
	EventUserLogout     EventType = "user.logout"
	EventOAuthLinked    EventType = "oauth.linked"
	EventOAuthLogin     EventType = "oauth.login"
	EventTokenRefreshed EventType = "token.refreshed"
	EventPasswordChange EventType = "password.changed"
	EventMFAEnabled     EventType = "mfa.enabled"
	EventMFADisabled    EventType = "mfa.disabled"
)

type Event struct {
	ID        string
	Type      EventType
	Timestamp time.Time
	Payload   interface{}
	SiteID    string
	Context   context.Context
}

type EventHandler func(ctx context.Context, event Event) error

type EventBus struct {
	mu       sync.RWMutex
	log      *logger.Logger
	handlers map[EventType][]EventHandler
}

func NewEventBus(log *logger.Logger) *EventBus {
	return &EventBus{
		log:      log,
		handlers: make(map[EventType][]EventHandler),
	}
}

func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	eb.log.Debug("event handler subscribed", "event_type", string(eventType))
}

func (eb *EventBus) Emit(ctx context.Context, eventType EventType, payload interface{}, siteID string) error {
	eb.mu.RLock()
	handlers, exists := eb.handlers[eventType]
	eb.mu.RUnlock()

	if !exists {
		return nil
	}

	event := Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now(),
		Payload:   payload,
		SiteID:    siteID,
		Context:   ctx,
	}

	eb.log.Debug("emitting event", "event_type", string(eventType), "event_id", event.ID)

	for _, handler := range handlers {
		if err := eb.safeHandle(ctx, handler, event); err != nil {
			eb.log.Error("event handler error",
				"event_type", string(eventType),
				"error", err,
			)
		}
	}

	return nil
}

func (eb *EventBus) EmitAsync(ctx context.Context, eventType EventType, payload interface{}, siteID string) {
	go func() {
		if err := eb.Emit(ctx, eventType, payload, siteID); err != nil {
			eb.log.Error("async event error", "event_type", string(eventType), "error", err)
		}
	}()
}

func (eb *EventBus) safeHandle(ctx context.Context, handler EventHandler, event Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in event handler: %v", r)
		}
	}()

	return handler(ctx, event)
}
