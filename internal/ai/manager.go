package ai

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"nexora/internal/kernel"
	"nexora/internal/pkg/logger"
)

type cbState int

const (
	cbClosed   cbState = 0
	cbHalfOpen cbState = 1
	cbOpen     cbState = 2
)

type circuitBreaker struct {
	mu               sync.RWMutex
	state            cbState
	failureCount     int
	failureThreshold int
	recoveryTimeout  time.Duration
	halfOpenMaxReqs  int
	halfOpenReqs     int
	lastFailure      time.Time
	opensCount       int64
}

func newCircuitBreaker(cfg CBConfig) *circuitBreaker {
	return &circuitBreaker{
		state:            cbClosed,
		failureThreshold: cfg.FailureThreshold,
		recoveryTimeout:  cfg.RecoveryTimeout,
		halfOpenMaxReqs:  cfg.HalfOpenMaxReqs,
	}
}

func (cb *circuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case cbClosed:
		return true
	case cbOpen:
		if time.Since(cb.lastFailure) > cb.recoveryTimeout {
			cb.state = cbHalfOpen
			cb.halfOpenReqs = 0
			return true
		}
		return false
	case cbHalfOpen:
		if cb.halfOpenReqs < cb.halfOpenMaxReqs {
			cb.halfOpenReqs++
			return true
		}
		return false
	}
	return false
}

func (cb *circuitBreaker) success() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == cbHalfOpen {
		cb.state = cbClosed
		cb.failureCount = 0
		cb.halfOpenReqs = 0
	}
	cb.failureCount = 0
}

func (cb *circuitBreaker) failure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailure = time.Now()

	if cb.state == cbHalfOpen || cb.failureCount >= cb.failureThreshold {
		cb.state = cbOpen
		cb.opensCount++
	}
}

type providerStats struct {
	requests     int64
	failed       int64
	totalLatency time.Duration
	tokensUsed   int64
	circuitOpens int64
	mu           sync.Mutex
}

type Manager struct {
	config    AIConfig
	registry  *Registry
	log       *logger.Logger
	eventBus  *kernel.EventBus
	cb        *circuitBreaker
	prompts   PromptBuilder
	quality   QualityChecker
	stats     map[string]*providerStats
	statsMu   sync.Mutex
	metricsMu sync.Mutex
}

func NewManager(cfg AIConfig, log *logger.Logger) *Manager {
	cb := newCircuitBreaker(cfg.CircuitBreaker)
	return &Manager{
		config:   cfg,
		registry: NewRegistry(),
		log:      log,
		cb:       cb,
		prompts:  NewPromptBuilder(),
		quality:  NewQualityChecker(),
		stats:    make(map[string]*providerStats),
	}
}

func (m *Manager) SetEventBus(bus *kernel.EventBus) {
	m.eventBus = bus
}

func (m *Manager) Registry() *Registry {
	return m.registry
}

func (m *Manager) Provider(name string) (AIProvider, error) {
	provider, _, ok := m.registry.Get(name)
	if !ok {
		return nil, ErrProviderNotFound
	}
	return provider, nil
}

func (m *Manager) DefaultProvider() (AIProvider, error) {
	provider, _, ok := m.registry.Default()
	if !ok {
		return nil, ErrNoProvidersRegistered
	}
	return provider, nil
}

func (m *Manager) RegisterProvider(provider AIProvider, cfg ProviderCfg) error {
	return m.registry.Register(provider, cfg)
}

func (m *Manager) SetDefaultProvider(name string) error {
	return m.registry.SetDefault(name)
}

func (m *Manager) ListProviders() []ProviderInfo {
	return m.registry.List()
}

func (m *Manager) Generate(ctx context.Context, req CompletionRequest) (*CompletionResult, error) {
	m.fireEvent(ctx, EventAIStarted, map[string]interface{}{
		"model":  req.Model,
		"stream": false,
	})

	result, err := m.executeWithRetry(ctx, req, false)
	if err != nil {
		m.fireEvent(ctx, EventAIFailed, map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	m.fireEvent(ctx, EventAICompleted, map[string]interface{}{
		"provider": result.ProviderName,
		"tokens":   result.TotalTokens,
		"duration": result.Duration.String(),
	})
	return result, nil
}

func (m *Manager) GenerateStream(ctx context.Context, req CompletionRequest, handler StreamHandler) error {
	m.fireEvent(ctx, EventAIStarted, map[string]interface{}{
		"model":  req.Model,
		"stream": true,
	})

	req.Stream = true

	err := m.executeStreamWithRetry(ctx, req, handler)
	if err != nil {
		m.fireEvent(ctx, EventAIFailed, map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	m.fireEvent(ctx, EventAICompleted, map[string]interface{}{
		"stream": true,
	})
	return nil
}

func (m *Manager) Embeddings(ctx context.Context, input string) (*EmbeddingResult, error) {
	providers := m.registry.FindByCapability(CapEmbeddings)
	if len(providers) == 0 {
		return nil, ErrEmbeddingNotSupported
	}

	var lastErr error
	for _, provider := range providers {
		result, err := provider.Embeddings(ctx, input)
		if err == nil {
			m.recordSuccess(provider.Name(), 0)
			return result, nil
		}
		lastErr = err
		m.recordFailure(provider.Name())
	}
	return nil, fmt.Errorf("%w: %v", ErrAllProvidersFailed, lastErr)
}

func (m *Manager) Summarize(ctx context.Context, req SummarizeRequest) (string, error) {
	providers := m.registry.FindByCapability(CapSummarize)
	if len(providers) == 0 {
		return "", ErrRewriteNotSupported
	}

	var lastErr error
	for _, provider := range providers {
		result, err := provider.Summarize(ctx, req)
		if err == nil {
			m.recordSuccess(provider.Name(), 0)
			return result, nil
		}
		lastErr = err
		m.recordFailure(provider.Name())
	}
	return "", fmt.Errorf("%w: %v", ErrAllProvidersFailed, lastErr)
}

func (m *Manager) Rewrite(ctx context.Context, req RewriteRequest) (string, error) {
	providers := m.registry.FindByCapability(CapRewrite)
	if len(providers) == 0 {
		return "", ErrRewriteNotSupported
	}

	var lastErr error
	for _, provider := range providers {
		result, err := provider.Rewrite(ctx, req)
		if err == nil {
			m.recordSuccess(provider.Name(), 0)
			return result, nil
		}
		lastErr = err
		m.recordFailure(provider.Name())
	}
	return "", fmt.Errorf("%w: %v", ErrAllProvidersFailed, lastErr)
}

func (m *Manager) Classify(ctx context.Context, req ClassifyRequest) (*ClassifyResult, error) {
	providers := m.registry.FindByCapability(CapClassify)
	if len(providers) == 0 {
		return nil, ErrClassificationNotSupported
	}

	var lastErr error
	for _, provider := range providers {
		result, err := provider.Classify(ctx, req)
		if err == nil {
			m.recordSuccess(provider.Name(), 0)
			return result, nil
		}
		lastErr = err
		m.recordFailure(provider.Name())
	}
	return nil, fmt.Errorf("%w: %v", ErrAllProvidersFailed, lastErr)
}

func (m *Manager) Health(ctx context.Context) (*ProviderHealthReport, error) {
	report := m.registry.HealthCheck(ctx)
	var err error
	if report.Overall != ProviderHealthy {
		err = ErrHealthCheckFailed
	}
	return report, err
}

func (m *Manager) Quality() QualityChecker {
	return m.quality
}

func (m *Manager) Prompts() PromptBuilder {
	return m.prompts
}

func (m *Manager) Capabilities() []Capability {
	var all []Capability
	for _, info := range m.ListProviders() {
		all = append(all, info.Capabilities...)
	}
	return all
}

func (m *Manager) Metrics() AIMetrics {
	m.metricsMu.Lock()
	defer m.metricsMu.Unlock()

	metrics := AIMetrics{
		ProviderStats: make(map[string]ProviderMetrics),
	}

	m.statsMu.Lock()
	for name, stats := range m.stats {
		stats.mu.Lock()
		var avgLatency time.Duration
		if stats.requests > 0 {
			avgLatency = time.Duration(int64(stats.totalLatency) / stats.requests)
		}
		metrics.ProviderStats[name] = ProviderMetrics{
			Requests:     stats.requests,
			Failed:       stats.failed,
			AvgLatency:   avgLatency,
			TokensUsed:   stats.tokensUsed,
			CircuitOpens: atomic.LoadInt64(&stats.circuitOpens),
		}
		metrics.TotalRequests += stats.requests
		metrics.FailedRequests += stats.failed
		metrics.TotalTokens += stats.tokensUsed
		stats.mu.Unlock()
	}
	m.statsMu.Unlock()

	if metrics.TotalRequests > 0 {
		var totalLatency int64
		for _, s := range metrics.ProviderStats {
			totalLatency += int64(s.AvgLatency)
		}
		metrics.AvgLatency = time.Duration(totalLatency / int64(len(metrics.ProviderStats)))
	}

	return metrics
}

func (m *Manager) executeWithRetry(ctx context.Context, req CompletionRequest, isStream bool) (*CompletionResult, error) {
	providers := m.selectProviders()

	if len(providers) == 0 {
		return nil, ErrNoProvidersRegistered
	}

	maxAttempts := m.config.Retry.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		for _, provider := range providers {
			if !m.cb.allow() {
				m.fireEvent(ctx, EventAITimeout, map[string]interface{}{
					"provider": provider.Name(),
					"reason":   "circuit_breaker_open",
				})
				continue
			}

			start := time.Now()
			result, err := provider.Generate(ctx, req)
			duration := time.Since(start)

			if err == nil {
				m.cb.success()
				m.recordSuccess(provider.Name(), result.TotalTokens)
				result.Duration = duration
				return result, nil
			}

			m.cb.failure()
			m.recordFailure(provider.Name())

			m.fireEvent(ctx, EventAIRetry, map[string]interface{}{
				"provider": provider.Name(),
				"attempt":  attempt + 1,
				"error":    err.Error(),
			})

			if attempt < maxAttempts-1 {
				delay := m.backoff(attempt)
				m.log.Debug("AI retry", "provider", provider.Name(), "attempt", attempt+1, "delay", delay)
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay):
				}
			}

			if m.shouldFailover(provider) {
				m.fireEvent(ctx, EventAIProviderChanged, map[string]interface{}{
					"from":   provider.Name(),
					"reason": "failover",
				})
			}
		}
	}

	return nil, ErrAllProvidersFailed
}

func (m *Manager) executeStreamWithRetry(ctx context.Context, req CompletionRequest, handler StreamHandler) error {
	providers := m.selectProviders()
	if len(providers) == 0 {
		return ErrNoProvidersRegistered
	}

	var lastErr error
	for _, provider := range providers {
		if !m.cb.allow() {
			continue
		}

		chunkCh, err := provider.GenerateStream(ctx, req)
		if err != nil {
			m.cb.failure()
			m.recordFailure(provider.Name())
			lastErr = err
			continue
		}

		m.cb.success()

		var result CompletionResult
		var fullContent string
		for chunk := range chunkCh {
			if chunk.Error != nil {
				_ = handler.OnError(chunk.Error)
				continue
			}
			fullContent += chunk.Content

			m.fireEvent(ctx, EventAIStreaming, map[string]interface{}{
				"chunk": chunk.Content,
				"index": chunk.Index,
				"done":  chunk.Done,
			})

			if err := handler.HandleChunk(chunk); err != nil {
				return err
			}

			if chunk.Done {
				result = CompletionResult{
					Content:      fullContent,
					ProviderName: provider.Name(),
					FinishReason: chunk.FinishReason,
				}
				_ = handler.OnComplete(&result)
				return nil
			}
		}
	}

	return fmt.Errorf("%w: %v", ErrAllProvidersFailed, lastErr)
}

func (m *Manager) selectProviders() []AIProvider {
	r := m.registry

	if r.Count() == 0 {
		return nil
	}

	list := r.List()
	var providers []AIProvider

	for _, info := range list {
		if !info.Enabled {
			continue
		}
		p, _, ok := r.Get(info.Name)
		if ok {
			providers = append(providers, p)
		}
	}

	if len(providers) == 0 {
		defaultP, _, ok := r.Default()
		if ok {
			providers = append(providers, defaultP)
		}
	}

	return providers
}

func (m *Manager) shouldFailover(provider AIProvider) bool {
	if provider == nil {
		return false
	}

	r := m.registry
	list := r.List()
	enabledCount := 0
	for _, info := range list {
		if info.Enabled {
			enabledCount++
		}
	}
	if enabledCount > 1 {
		m.fireEvent(context.Background(), EventAIProviderChanged, map[string]interface{}{
			"from": provider.Name(),
		})
	}
	return enabledCount > 1
}

func (m *Manager) backoff(attempt int) time.Duration {
	base := m.config.Retry.BaseDelay
	if base <= 0 {
		base = 100 * time.Millisecond
	}
	maxDelay := m.config.Retry.MaxDelay
	if maxDelay <= 0 {
		maxDelay = 5 * time.Second
	}

	delay := base * (1 << uint(attempt))
	jitter := time.Duration(rand.Int63n(int64(delay) / 2))

	total := delay + jitter
	if total > maxDelay {
		total = maxDelay
	}
	return total
}

func (m *Manager) recordSuccess(providerName string, tokens int) {
	m.statsMu.Lock()
	stats, ok := m.stats[providerName]
	if !ok {
		stats = &providerStats{}
		m.stats[providerName] = stats
	}
	m.statsMu.Unlock()

	stats.mu.Lock()
	stats.requests++
	stats.tokensUsed += int64(tokens)
	stats.mu.Unlock()
}

func (m *Manager) recordFailure(providerName string) {
	m.statsMu.Lock()
	stats, ok := m.stats[providerName]
	if !ok {
		stats = &providerStats{}
		m.stats[providerName] = stats
	}
	m.statsMu.Unlock()

	stats.mu.Lock()
	stats.failed++
	stats.mu.Unlock()
}

func (m *Manager) fireEvent(ctx context.Context, eventType kernel.EventType, payload interface{}) {
	if m.eventBus != nil {
		m.eventBus.EmitAsync(ctx, eventType, payload, "")
	}
}
