package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type GeminiProvider struct {
	mu       sync.RWMutex
	name     string
	model    string
	apiKey   string
	baseURL  string
	client   *http.Client
	healthy  bool
	latency  time.Duration
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerateReq struct {
	Contents         []geminiContent    `json:"contents"`
	SystemInstruction *geminiContent    `json:"systemInstruction,omitempty"`
	GenerationConfig *geminiGenConfig   `json:"generationConfig,omitempty"`
}

type geminiGenConfig struct {
	Temperature      float64  `json:"temperature,omitempty"`
	MaxOutputTokens  int      `json:"maxOutputTokens,omitempty"`
	StopSequences    []string `json:"stopSequences,omitempty"`
}

type geminiGenerateResp struct {
	Candidates []geminiCandidate `json:"candidates"`
	UsageMeta  geminiUsageMeta   `json:"usageMetadata"`
	Error      *geminiError      `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content       geminiContent `json:"content"`
	FinishReason  string        `json:"finishReason"`
	SafetyRatings []interface{} `json:"safetyRatings,omitempty"`
}

type geminiUsageMeta struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

type geminiEmbedReq struct {
	Content geminiContent `json:"content"`
}

type geminiEmbedResp struct {
	Embedding struct {
		Values []float64 `json:"values"`
	} `json:"embedding"`
	Error *geminiError `json:"error,omitempty"`
}

func NewGeminiProvider(name, model, apiKey, baseURL string) *GeminiProvider {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &GeminiProvider{
		name:    name,
		model:   model,
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		healthy: true,
		latency: 0,
	}
}

func (p *GeminiProvider) Name() string { return p.name }

func (p *GeminiProvider) Capabilities() []Capability {
	return []Capability{CapGenerate, CapStream, CapEmbeddings, CapSummarize, CapRewrite, CapClassify}
}

func (p *GeminiProvider) Generate(ctx context.Context, req CompletionRequest) (*CompletionResult, error) {
	if err := p.checkHealth(); err != nil {
		return nil, err
	}

	geminiReq := p.buildRequest(req)
	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)

	start := time.Now()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		p.setUnhealthy()
		return nil, fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp.StatusCode, respBody)
	}

	var geminiResp geminiGenerateResp
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("gemini API error: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("gemini returned no candidates")
	}

	content := ""
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		content += part.Text
	}

	duration := time.Since(start)
	p.setLatency(duration)

	return &CompletionResult{
		Content:      content,
		Model:        p.model,
		ProviderName: p.name,
		TotalTokens:  geminiResp.UsageMeta.TotalTokenCount,
		PromptTokens: geminiResp.UsageMeta.PromptTokenCount,
		Duration:     duration,
		FinishReason: strings.ToLower(geminiResp.Candidates[0].FinishReason),
	}, nil
}

func (p *GeminiProvider) GenerateStream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	if err := p.checkHealth(); err != nil {
		return nil, err
	}

	geminiReq := p.buildRequest(req)
	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", p.baseURL, p.model, p.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		p.setUnhealthy()
		return nil, fmt.Errorf("gemini stream request failed: %w", err)
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			ch <- StreamChunk{Error: p.parseError(resp.StatusCode, respBody)}
			return
		}

		decoder := json.NewDecoder(resp.Body)
		index := 0
		for {
			var geminiResp geminiGenerateResp
			if err := decoder.Decode(&geminiResp); err != nil {
				if err == io.EOF {
					break
				}
				ch <- StreamChunk{Error: fmt.Errorf("stream decode error: %w", err)}
				return
			}

			if geminiResp.Error != nil {
				ch <- StreamChunk{Error: fmt.Errorf("gemini API error: %s", geminiResp.Error.Message)}
				return
			}

			if len(geminiResp.Candidates) > 0 {
				content := ""
				for _, part := range geminiResp.Candidates[0].Content.Parts {
					content += part.Text
				}
				finishReason := strings.ToLower(geminiResp.Candidates[0].FinishReason)
				done := finishReason != "" && finishReason != "finish_reason_unspecified"

				select {
				case <-ctx.Done():
					ch <- StreamChunk{Error: ctx.Err()}
					return
				case ch <- StreamChunk{Content: content, Index: index, Done: done, FinishReason: finishReason}:
				}
				index++
				if done {
					return
				}
			}
		}
	}()

	return ch, nil
}

func (p *GeminiProvider) Embeddings(ctx context.Context, input string) (*EmbeddingResult, error) {
	if err := p.checkHealth(); err != nil {
		return nil, err
	}
	if !p.hasCapability(CapEmbeddings) {
		return nil, ErrEmbeddingNotSupported
	}

	embedReq := geminiEmbedReq{
		Content: geminiContent{
			Parts: []geminiPart{{Text: input}},
		},
	}
	body, err := json.Marshal(embedReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:embedContent?key=%s", p.baseURL, p.model, p.apiKey)

	start := time.Now()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		p.setUnhealthy()
		return nil, fmt.Errorf("gemini embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedding response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp.StatusCode, respBody)
	}

	var embedResp geminiEmbedResp
	if err := json.Unmarshal(respBody, &embedResp); err != nil {
		return nil, fmt.Errorf("failed to parse embedding response: %w", err)
	}

	if embedResp.Error != nil {
		return nil, fmt.Errorf("gemini API error: %s", embedResp.Error.Message)
	}

	return &EmbeddingResult{
		Vector:     embedResp.Embedding.Values,
		Model:      p.model,
		Dimensions: len(embedResp.Embedding.Values),
		Duration:   time.Since(start),
	}, nil
}

func (p *GeminiProvider) Summarize(ctx context.Context, req SummarizeRequest) (string, error) {
	completionReq := CompletionRequest{
		Prompt: fmt.Sprintf("Summarize the following text in %s. Keep it under %d words:\n\n%s",
			p.langLabel(req.Language), req.MaxWords, req.Text),
		Temperature: 0.3,
	}
	result, err := p.Generate(ctx, completionReq)
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

func (p *GeminiProvider) Rewrite(ctx context.Context, req RewriteRequest) (string, error) {
	completionReq := CompletionRequest{
		Prompt: fmt.Sprintf("Rewrite the following text. Instructions: %s. Tone: %s. Audience: %s.\n\n%s",
			req.Instructions, req.Tone, req.Audience, req.Text),
		Temperature: 0.7,
	}
	result, err := p.Generate(ctx, completionReq)
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

func (p *GeminiProvider) Classify(ctx context.Context, req ClassifyRequest) (*ClassifyResult, error) {
	categories := strings.Join(req.Categories, ", ")
	completionReq := CompletionRequest{
		Prompt: fmt.Sprintf("Classify the following text into exactly one of these categories: %s. Return only the category name, nothing else.\n\n%s",
			categories, req.Text),
		Temperature: 0.1,
		MaxTokens:   50,
	}
	result, err := p.Generate(ctx, completionReq)
	if err != nil {
		return nil, err
	}

	predicted := strings.TrimSpace(result.Content)
	confidence := 1.0
	scores := make(map[string]float64)
	score := 1.0 / float64(len(req.Categories))
	for _, c := range req.Categories {
		if c == predicted {
			scores[c] = confidence
		} else {
			scores[c] = score
		}
	}

	return &ClassifyResult{
		Category:   predicted,
		Confidence: confidence,
		Scores:     scores,
	}, nil
}

func (p *GeminiProvider) Health(ctx context.Context) (*HealthStatus, error) {
	p.mu.RLock()
	healthy := p.healthy
	p.mu.RUnlock()

	if !healthy {
		return &HealthStatus{
			Provider: p.name,
			State:    ProviderUnhealthy,
			Message:  "provider marked unhealthy",
		}, ErrHealthCheckFailed
	}

	url := fmt.Sprintf("%s/models/%s?key=%s", p.baseURL, p.model, p.apiKey)

	start := time.Now()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &HealthStatus{
			Provider: p.name,
			State:    ProviderUnhealthy,
			Message:  err.Error(),
		}, ErrHealthCheckFailed
	}

	resp, err := p.client.Do(httpReq)
	latency := time.Since(start)
	if err != nil {
		p.setUnhealthy()
		return &HealthStatus{
			Provider: p.name,
			State:    ProviderUnhealthy,
			Latency:  latency,
			Message:  err.Error(),
		}, ErrHealthCheckFailed
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &HealthStatus{
			Provider: p.name,
			State:    ProviderDegraded,
			Latency:  latency,
			Model:    p.model,
			Message:  fmt.Sprintf("API returned status %d", resp.StatusCode),
		}, nil
	}

	return &HealthStatus{
		Provider: p.name,
		State:    ProviderHealthy,
		Latency:  latency,
		Model:    p.model,
	}, nil
}

func (p *GeminiProvider) buildRequest(req CompletionRequest) geminiGenerateReq {
	gReq := geminiGenerateReq{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{{Text: req.Prompt}},
				Role:  "user",
			},
		},
	}

	if req.System != "" {
		gReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.System}},
		}
	}

	genCfg := &geminiGenConfig{}
	if req.Temperature > 0 {
		genCfg.Temperature = req.Temperature
	}
	if req.MaxTokens > 0 {
		genCfg.MaxOutputTokens = req.MaxTokens
	}
	if len(req.StopWords) > 0 {
		genCfg.StopSequences = req.StopWords
	}
	gReq.GenerationConfig = genCfg

	return gReq
}

func (p *GeminiProvider) parseError(statusCode int, body []byte) error {
	var geminiErr geminiError
	if err := json.Unmarshal(body, &geminiErr); err == nil && geminiErr.Message != "" {
		switch {
		case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
			return ErrInvalidAPIKey
		case statusCode == http.StatusTooManyRequests:
			return ErrRateLimited
		case statusCode >= 500:
			return ErrProviderUnavailable
		default:
			return fmt.Errorf("%w: %s", ErrProviderConfig, geminiErr.Message)
		}
	}

	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return ErrInvalidAPIKey
	case statusCode == http.StatusTooManyRequests:
		return ErrRateLimited
	case statusCode >= 500:
		return ErrProviderUnavailable
	default:
		return fmt.Errorf("gemini API returned status %d: %s", statusCode, string(body))
	}
}

func (p *GeminiProvider) checkHealth() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !p.healthy {
		return ErrProviderUnavailable
	}
	return nil
}

func (p *GeminiProvider) setUnhealthy() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthy = false
}

func (p *GeminiProvider) setLatency(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.latency = d
}

func (p *GeminiProvider) hasCapability(capability Capability) bool {
	for _, c := range p.Capabilities() {
		if c == capability {
			return true
		}
	}
	return false
}

func (p *GeminiProvider) langLabel(language string) string {
	if language == "pt" {
		return "Portuguese"
	}
	return "English"
}
