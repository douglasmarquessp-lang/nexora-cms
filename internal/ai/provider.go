package ai

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

type MockProvider struct {
	mu           sync.RWMutex
	name         string
	model        string
	capabilities []Capability
	healthy      bool
	latency      time.Duration
	state        ProviderState
	failRate     float64
	callCount    int64
}

func NewMockProvider(name, model string, caps []Capability) *MockProvider {
	if len(caps) == 0 {
		caps = []Capability{CapGenerate, CapStream, CapEmbeddings, CapSummarize, CapRewrite, CapClassify}
	}
	return &MockProvider{
		name:         name,
		model:        model,
		capabilities: caps,
		healthy:      true,
		latency:      10 * time.Millisecond,
		state:        ProviderHealthy,
		failRate:     0.0,
	}
}

func (p *MockProvider) Name() string { return p.name }

func (p *MockProvider) Capabilities() []Capability {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.capabilities
}

func (p *MockProvider) Generate(ctx context.Context, req CompletionRequest) (*CompletionResult, error) {
	p.mu.Lock()
	p.callCount++
	p.mu.Unlock()

	if err := p.checkHealth(); err != nil {
		return nil, err
	}
	if p.shouldFail() {
		return nil, ErrProviderUnavailable
	}

	time.Sleep(p.latency)
	return &CompletionResult{
		Content:      "Mock response for: " + req.Prompt,
		Model:        p.model,
		ProviderName: p.name,
		TotalTokens:  50,
		PromptTokens: 10,
		Duration:     p.latency,
		FinishReason: "stop",
	}, nil
}

func (p *MockProvider) GenerateStream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	if err := p.checkHealth(); err != nil {
		return nil, err
	}
	if p.shouldFail() {
		return nil, ErrProviderUnavailable
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		words := []string{"Mock ", "stream ", "response ", "for: ", req.Prompt}
		for i, w := range words {
			select {
			case <-ctx.Done():
				return
			case ch <- StreamChunk{Content: w, Index: i, Done: false}:
			}
			time.Sleep(p.latency / 5)
		}
		ch <- StreamChunk{Content: "", Index: len(words), Done: true, FinishReason: "stop"}
	}()
	return ch, nil
}

func (p *MockProvider) Embeddings(ctx context.Context, input string) (*EmbeddingResult, error) {
	if err := p.checkHealth(); err != nil {
		return nil, err
	}
	if !p.hasCapability(CapEmbeddings) {
		return nil, ErrEmbeddingNotSupported
	}
	time.Sleep(p.latency)
	return &EmbeddingResult{
		Vector:     make([]float64, 128),
		Model:      p.model,
		Dimensions: 128,
		Duration:   p.latency,
	}, nil
}

func (p *MockProvider) Summarize(ctx context.Context, req SummarizeRequest) (string, error) {
	if err := p.checkHealth(); err != nil {
		return "", err
	}
	if !p.hasCapability(CapSummarize) {
		return "", ErrRewriteNotSupported
	}
	time.Sleep(p.latency)
	return "Mock summary of text.", nil
}

func (p *MockProvider) Rewrite(ctx context.Context, req RewriteRequest) (string, error) {
	if err := p.checkHealth(); err != nil {
		return "", err
	}
	if !p.hasCapability(CapRewrite) {
		return "", ErrRewriteNotSupported
	}
	time.Sleep(p.latency)
	return "Mock rewritten: " + req.Text, nil
}

func (p *MockProvider) Classify(ctx context.Context, req ClassifyRequest) (*ClassifyResult, error) {
	if err := p.checkHealth(); err != nil {
		return nil, err
	}
	if !p.hasCapability(CapClassify) {
		return nil, ErrClassificationNotSupported
	}
	time.Sleep(p.latency)
	scores := make(map[string]float64)
	var bestCat string
	var bestScore float64
	for _, c := range req.Categories {
		s := rand.Float64()
		scores[c] = s
		if s > bestScore {
			bestScore = s
			bestCat = c
		}
	}
	return &ClassifyResult{
		Category:   bestCat,
		Confidence: bestScore,
		Scores:     scores,
	}, nil
}

func (p *MockProvider) Health(ctx context.Context) (*HealthStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.healthy {
		return &HealthStatus{
			Provider: p.name,
			State:    ProviderUnhealthy,
			Latency:  0,
			Message:  "provider marked unhealthy",
		}, ErrHealthCheckFailed
	}
	return &HealthStatus{
		Provider: p.name,
		State:    p.state,
		Latency:  p.latency,
		Model:    p.model,
	}, nil
}

func (p *MockProvider) SetHealthy(h bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthy = h
	if h {
		p.state = ProviderHealthy
	} else {
		p.state = ProviderUnhealthy
	}
}

func (p *MockProvider) SetLatency(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.latency = d
}

func (p *MockProvider) SetFailRate(rate float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failRate = rate
}

func (p *MockProvider) checkHealth() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !p.healthy {
		return ErrProviderUnavailable
	}
	return nil
}

func (p *MockProvider) shouldFail() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.failRate <= 0 {
		return false
	}
	return rand.Float64() < p.failRate
}

func (p *MockProvider) hasCapability(cap Capability) bool {
	for _, c := range p.capabilities {
		if c == cap {
			return true
		}
	}
	return false
}
