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
	mu              sync.RWMutex
	name            string
	model           string
	apiKey          string
	baseURL         string
	client          *http.Client
	healthy         bool
	latency         time.Duration
	lastUnhealthyAt time.Time
}

// healthRecheckInterval is the cooldown before an unhealthy provider is
// allowed to probe the API again. Without it a single transient failure
// would keep the provider unusable until the process restarts.
const healthRecheckInterval = 30 * time.Second

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiTool struct {
	GoogleSearch *geminiGoogleSearch `json:"googleSearch,omitempty"`
}

type geminiGoogleSearch struct{}

type geminiGenerateReq struct {
	Contents          []geminiContent  `json:"contents"`
	SystemInstruction *geminiContent   `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenConfig `json:"generationConfig,omitempty"`
	Tools             []geminiTool     `json:"tools,omitempty"`
}

type geminiGenConfig struct {
	Temperature     float64  `json:"temperature,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type geminiGenerateResp struct {
	Candidates []geminiCandidate `json:"candidates"`
	UsageMeta  geminiUsageMeta   `json:"usageMetadata"`
	Error      *geminiError      `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content           geminiContent            `json:"content"`
	FinishReason      string                   `json:"finishReason"`
	SafetyRatings     []interface{}            `json:"safetyRatings,omitempty"`
	GroundingMetadata *geminiGroundingMetadata `json:"groundingMetadata,omitempty"`
}

type geminiGroundingMetadata struct {
	SearchEntryPoint  *geminiSearchEntryPoint  `json:"searchEntryPoint,omitempty"`
	GroundingSupports []geminiGroundingSupport `json:"groundingSupports,omitempty"`
	GroundingChunks   []geminiGroundingChunk   `json:"groundingChunks,omitempty"`
	WebSearchQueries  []string                 `json:"webSearchQueries,omitempty"`
}

type geminiSearchEntryPoint struct {
	RenderedContent string `json:"renderedContent,omitempty"`
	SDKBlob         string `json:"sdkBlob,omitempty"`
}

type geminiGroundingChunk struct {
	Web *geminiWebChunk `json:"web,omitempty"`
}

type geminiWebChunk struct {
	URI   string `json:"uri"`
	Title string `json:"title"`
}

type geminiGroundingSupport struct {
	Segment               *geminiSegment `json:"segment,omitempty"`
	GroundingChunkIndices []int          `json:"groundingChunkIndices,omitempty"`
	ConfidenceScores      []float64      `json:"confidenceScores,omitempty"`
}

type geminiSegment struct {
	StartIndex int    `json:"startIndex"`
	EndIndex   int    `json:"endIndex"`
	Text       string `json:"text,omitempty"`
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
	return []Capability{CapGenerate, CapStream, CapEmbeddings, CapSummarize, CapRewrite, CapClassify, CapGrounding}
}

func (p *GeminiProvider) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := p.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Goog-Api-Key", p.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return p.client.Do(req)
}

func (p *GeminiProvider) Generate(ctx context.Context, req CompletionRequest) (*CompletionResult, error) {
	if err := p.checkHealth(ctx); err != nil {
		return nil, err
	}

	geminiReq := p.buildRequest(req)
	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	start := time.Now()
	resp, err := p.doRequest(ctx, http.MethodPost, fmt.Sprintf("/models/%s:generateContent", p.model), bytes.NewReader(body))
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

	result := &CompletionResult{
		Content:      content,
		Model:        p.model,
		ProviderName: p.name,
		TotalTokens:  geminiResp.UsageMeta.TotalTokenCount,
		PromptTokens: geminiResp.UsageMeta.PromptTokenCount,
		Duration:     duration,
		FinishReason: strings.ToLower(geminiResp.Candidates[0].FinishReason),
	}

	if gm := geminiResp.Candidates[0].GroundingMetadata; gm != nil {
		result.GroundingMetadata = p.toGroundingMetadata(gm)
	}

	return result, nil
}

func (p *GeminiProvider) GenerateStream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	if err := p.checkHealth(ctx); err != nil {
		return nil, err
	}

	geminiReq := p.buildRequest(req)
	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := p.doRequest(ctx, http.MethodPost, fmt.Sprintf("/models/%s:streamGenerateContent?alt=sse", p.model), bytes.NewReader(body))
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
	if err := p.checkHealth(ctx); err != nil {
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

	start := time.Now()
	resp, err := p.doRequest(ctx, http.MethodPost, fmt.Sprintf("/models/%s:embedContent", p.model), bytes.NewReader(body))
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
	lastUnhealthy := p.lastUnhealthyAt
	p.mu.RUnlock()

	// A provider marked unhealthy only reports the stale state while the
	// cooldown window is still open; afterwards Health() probes the API for
	// real, which lets a transient failure recover without a restart.
	if !healthy && time.Since(lastUnhealthy) < healthRecheckInterval {
		return &HealthStatus{
			Provider: p.name,
			State:    ProviderUnhealthy,
			Message:  "provider marked unhealthy",
		}, ErrHealthCheckFailed
	}

	start := time.Now()
	resp, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/models/%s", p.model), nil)
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

	p.setHealthy()
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

	if req.Grounding != nil && req.Grounding.Enabled {
		gReq.Tools = []geminiTool{{GoogleSearch: &geminiGoogleSearch{}}}
	}

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

func (p *GeminiProvider) checkHealth(ctx context.Context) error {
	p.mu.RLock()
	healthy := p.healthy
	lastUnhealthy := p.lastUnhealthyAt
	p.mu.RUnlock()

	if healthy {
		return nil
	}

	// Allow one real re-probe once the cooldown has elapsed, so a
	// temporarily-failed provider can recover without a restart.
	if time.Since(lastUnhealthy) < healthRecheckInterval {
		return ErrProviderUnavailable
	}

	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	resp, err := p.doRequest(probeCtx, http.MethodGet, fmt.Sprintf("/models/%s", p.model), nil)
	if err != nil {
		p.setUnhealthy()
		return ErrProviderUnavailable
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p.setUnhealthy()
		return ErrProviderUnavailable
	}

	p.setHealthy()
	return nil
}

func (p *GeminiProvider) setUnhealthy() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthy = false
	p.lastUnhealthyAt = time.Now()
}

func (p *GeminiProvider) setHealthy() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthy = true
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

func (p *GeminiProvider) toGroundingMetadata(gm *geminiGroundingMetadata) *GroundingMetadata {
	meta := &GroundingMetadata{
		SearchSuggested: len(gm.WebSearchQueries) > 0,
		Unverified:      false,
	}

	if gm.SearchEntryPoint != nil {
		meta.SearchEntryPoint = &SearchEntryPoint{
			RenderedHTML: gm.SearchEntryPoint.RenderedContent,
		}
		if gm.SearchEntryPoint.SDKBlob != "" {
			meta.SearchEntryPoint.RenderedHTML = gm.SearchEntryPoint.SDKBlob
		}
	}

	// Convert grounding chunks to sources
	for _, chunk := range gm.GroundingChunks {
		if chunk.Web != nil {
			now := time.Now()
			source := GroundingSource{
				URI:         chunk.Web.URI,
				Title:       chunk.Web.Title,
				RetrievedAt: now,
				IsVerified:  true,
			}
			meta.Sources = append(meta.Sources, source)
		}
	}

	// Convert grounding supports
	for _, support := range gm.GroundingSupports {
		gs := GroundingSupport{
			SourceIndices: support.GroundingChunkIndices,
		}
		if support.Segment != nil {
			gs.Segment = support.Segment.Text
		}
		if len(support.ConfidenceScores) > 0 {
			gs.Confidence = support.ConfidenceScores[0]
		}
		meta.SupportSegments = append(meta.SupportSegments, gs)
	}

	if len(meta.Sources) == 0 {
		meta.Unverified = true
	}

	return meta
}

func (p *GeminiProvider) langLabel(language string) string {
	if language == "pt" {
		return "Portuguese"
	}
	return "English"
}
