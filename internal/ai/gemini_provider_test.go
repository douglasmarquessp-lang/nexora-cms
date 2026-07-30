package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewGeminiProvider(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "test-key", "")
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Name() != "gemini" {
		t.Errorf("expected name 'gemini', got %s", p.Name())
	}
}

func TestGeminiProvider_Defaults(t *testing.T) {
	p := NewGeminiProvider("gemini", "", "", "")
	if p.model != "gemini-2.0-flash" {
		t.Errorf("expected default model 'gemini-2.0-flash', got %s", p.model)
	}
	if p.baseURL != "https://generativelanguage.googleapis.com/v1beta" {
		t.Errorf("expected default base URL, got %s", p.baseURL)
	}
}

func TestGeminiProvider_Generate_InvalidKey(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "invalid-key", "")
	ctx := context.Background()

	_, err := p.Generate(ctx, CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Error("expected error for invalid API key")
	}
}

func TestGeminiProvider_GenerateStream_InvalidKey(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "invalid-key", "")
	ctx := context.Background()

	_, err := p.GenerateStream(ctx, CompletionRequest{Prompt: "Hello"})
	if err == nil {
		t.Error("expected error for invalid API key")
	}
}

func TestGeminiProvider_Embeddings_InvalidKey(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "invalid-key", "")
	ctx := context.Background()

	_, err := p.Embeddings(ctx, "test input")
	if err == nil {
		t.Error("expected error for invalid API key")
	}
}

func TestGeminiProvider_Health_InvalidKey(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "invalid-key", "")
	ctx := context.Background()

	_, err := p.Health(ctx)
	if err == nil {
		t.Error("expected error for invalid API key")
	}
}

func TestGeminiProvider_Capabilities(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "test-key", "")
	caps := p.Capabilities()

	expectedCaps := []Capability{CapGenerate, CapStream, CapEmbeddings, CapSummarize, CapRewrite, CapClassify, CapGrounding}
	if len(caps) != len(expectedCaps) {
		t.Errorf("expected %d capabilities, got %d", len(expectedCaps), len(caps))
	}

	capMap := make(map[Capability]bool)
	for _, c := range caps {
		capMap[c] = true
	}
	for _, ec := range expectedCaps {
		if !capMap[ec] {
			t.Errorf("missing capability: %s", ec)
		}
	}
}

func TestGeminiProvider_Unhealthy(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "test-key", "")
	p.setUnhealthy()
	ctx := context.Background()

	_, err := p.Generate(ctx, CompletionRequest{Prompt: "Hello"})
	if err != ErrProviderUnavailable {
		t.Errorf("expected ErrProviderUnavailable, got %v", err)
	}
}

func TestGeminiProvider_Summarize(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "test-key", "")
	ctx := context.Background()

	_, err := p.Summarize(ctx, SummarizeRequest{
		Text:     "This is a long text that needs to be summarized.",
		MaxWords: 50,
		Language: "en",
	})
	if err == nil {
		t.Error("expected error (no real API), but Summarize should delegate to Generate")
	}
}

func TestGeminiProvider_Rewrite(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "test-key", "")
	ctx := context.Background()

	_, err := p.Rewrite(ctx, RewriteRequest{
		Text:         "Original text",
		Instructions: "Make it formal",
		Tone:         "formal",
		Audience:     "professionals",
	})
	if err == nil {
		t.Error("expected error (no real API), but Rewrite should delegate to Generate")
	}
}

func TestGeminiProvider_Classify(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "test-key", "")
	ctx := context.Background()

	_, err := p.Classify(ctx, ClassifyRequest{
		Text:       "This is a tech article",
		Categories: []string{"tech", "sports", "health"},
	})
	if err == nil {
		t.Error("expected error (no real API), but Classify should delegate to Generate")
	}
}

func TestGeminiProvider_Name(t *testing.T) {
	p := NewGeminiProvider("custom-name", "gemini-2.0-flash", "test-key", "")
	if p.Name() != "custom-name" {
		t.Errorf("expected 'custom-name', got %s", p.Name())
	}
}

func TestGeminiProvider_BuildRequest(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "test-key", "")

	req := p.buildRequest(CompletionRequest{
		Prompt:      "Hello",
		System:      "Be helpful",
		Temperature: 0.7,
		MaxTokens:   100,
		StopWords:   []string{"stop"},
	})

	if len(req.Contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(req.Contents))
	}
	if len(req.Contents[0].Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(req.Contents[0].Parts))
	}
	if req.Contents[0].Parts[0].Text != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", req.Contents[0].Parts[0].Text)
	}
	if req.Contents[0].Role != "user" {
		t.Errorf("expected role 'user', got '%s'", req.Contents[0].Role)
	}
	if req.SystemInstruction == nil {
		t.Fatal("expected system instruction")
	}
	if req.SystemInstruction.Parts[0].Text != "Be helpful" {
		t.Errorf("expected 'Be helpful', got '%s'", req.SystemInstruction.Parts[0].Text)
	}
	if req.GenerationConfig.Temperature != 0.7 {
		t.Errorf("expected temp 0.7, got %f", req.GenerationConfig.Temperature)
	}
	if req.GenerationConfig.MaxOutputTokens != 100 {
		t.Errorf("expected max tokens 100, got %d", req.GenerationConfig.MaxOutputTokens)
	}
	if len(req.GenerationConfig.StopSequences) != 1 || req.GenerationConfig.StopSequences[0] != "stop" {
		t.Errorf("expected stop sequence ['stop'], got %v", req.GenerationConfig.StopSequences)
	}
}

func TestGeminiProvider_BuildRequest_NoSystem(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "test-key", "")

	req := p.buildRequest(CompletionRequest{
		Prompt: "Hello",
	})

	if req.SystemInstruction != nil {
		t.Error("expected nil system instruction when not set")
	}
}

func TestGeminiProvider_BuildRequest_DefaultConfig(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "test-key", "")

	req := p.buildRequest(CompletionRequest{
		Prompt: "Hello",
	})

	if req.GenerationConfig == nil {
		t.Fatal("expected generation config")
	}
	if req.GenerationConfig.Temperature != 0 {
		t.Errorf("expected zero temperature default, got %f", req.GenerationConfig.Temperature)
	}
}

func TestGeminiProvider_LangLabel(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "test-key", "")

	if label := p.langLabel("pt"); label != "Portuguese" {
		t.Errorf("expected 'Portuguese', got '%s'", label)
	}
	if label := p.langLabel("en"); label != "English" {
		t.Errorf("expected 'English', got '%s'", label)
	}
	if label := p.langLabel("fr"); label != "English" {
		t.Errorf("expected 'English' (default), got '%s'", label)
	}
}

func TestGeminiProvider_ParseError(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "test-key", "")

	t.Run("unauthorized", func(t *testing.T) {
		err := p.parseError(401, []byte(`{"error":{"code":401,"message":"API key not valid","status":"UNAUTHENTICATED"}}`))
		if err != ErrInvalidAPIKey {
			t.Errorf("expected ErrInvalidAPIKey, got %v", err)
		}
	})

	t.Run("rate limited", func(t *testing.T) {
		err := p.parseError(429, []byte(`{"error":{"code":429,"message":"Rate limit exceeded","status":"RESOURCE_EXHAUSTED"}}`))
		if err != ErrRateLimited {
			t.Errorf("expected ErrRateLimited, got %v", err)
		}
	})

	t.Run("server error", func(t *testing.T) {
		err := p.parseError(503, []byte(`{"error":{"code":503,"message":"Service unavailable","status":"UNAVAILABLE"}}`))
		if err != ErrProviderUnavailable {
			t.Errorf("expected ErrProviderUnavailable, got %v", err)
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		err := p.parseError(403, []byte(`{"error":{"code":403,"message":"Access denied","status":"PERMISSION_DENIED"}}`))
		if err != ErrInvalidAPIKey {
			t.Errorf("expected ErrInvalidAPIKey, got %v", err)
		}
	})

	t.Run("unknown error", func(t *testing.T) {
		err := p.parseError(400, []byte(`{"error":{"code":400,"message":"Bad request","status":"INVALID_ARGUMENT"}}`))
		if err == nil {
			t.Error("expected non-nil error")
		}
	})
}

func TestGeminiProvider_HasCapability(t *testing.T) {
	p := NewGeminiProvider("gemini", "gemini-2.0-flash", "test-key", "")

	if !p.hasCapability(CapGenerate) {
		t.Error("expected CapGenerate")
	}
	if !p.hasCapability(CapStream) {
		t.Error("expected CapStream")
	}
	if p.hasCapability(Capability("nonexistent")) {
		t.Error("unexpected capability")
	}
}

func TestGeminiProvider_AuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Goog-Api-Key") != "test-secret-key" {
			t.Errorf("expected X-Goog-Api-Key header, got '%s'", r.Header.Get("X-Goog-Api-Key"))
		}
		if strings.Contains(r.URL.RawQuery, "key=") {
			t.Error("API key must not be in URL query parameters")
		}
		if r.Method == http.MethodPost && r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type: application/json, got '%s'", r.Header.Get("Content-Type"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{{"text": "test response"}},
						"role":  "model",
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]int{
				"promptTokenCount":     1,
				"candidatesTokenCount": 2,
				"totalTokenCount":      3,
			},
		})
	}))
	defer server.Close()

	p := NewGeminiProvider("test", "gemini-2.0-flash", "test-secret-key", server.URL)
	p.client = server.Client()

	ctx := context.Background()
	_, err := p.Generate(ctx, CompletionRequest{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
}
