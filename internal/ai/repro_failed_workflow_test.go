package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

// TestRepro_FailedGeminiCall_MasksRealCause reproduces the Railway scenario:
// a Gemini provider whose API call fails (401/404/429/5xx). The workflow's
// PipelineExecutor stage goes through the Manager, whose non-stream retry path
// returns a GENERIC "all AI providers failed" error (manager.go:414) and
// DISCARDS the real cause — so the job's error_message can never show why.
func TestRepro_FailedGeminiCall_MasksRealCause(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":404,"message":"models/gemini-2.0-flash is not found","status":"NOT_FOUND"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":401,"message":"API key not valid. Please pass a valid API key.","status":"UNAUTHENTICATED"}}`))
	}))
	defer srv.Close()

	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	g := NewGeminiProvider("gemini", "gemini-2.0-flash", "dummy-key-not-printed", srv.URL)
	if err := m.RegisterProvider(g, ProviderCfg{
		Name: "gemini", Model: "gemini-2.0-flash", APIKey: "dummy",
		BaseURL: srv.URL, Enabled: true, Priority: 1, Weight: 10,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	pe := NewPipelineExecutor(m)
	_, err := pe.ExecuteStage(context.Background(), StageDraftGen, PipelineInput{
		Title:    "The Future of AI Productivity Tools for Small Businesses in 2026",
		Topic:    "The Future of AI Productivity Tools for Small Businesses in 2026",
		Language: "en",
	})
	if err == nil {
		t.Fatal("expected stage error with failing Gemini provider")
	}

	msg := err.Error()
	if !strings.Contains(msg, "all AI providers failed") {
		t.Errorf("expected generic masked error, got: %q", msg)
	}
	if strings.Contains(msg, "API key not valid") || strings.Contains(msg, "NOT_FOUND") {
		t.Errorf("real cause leaked into masked error: %q", msg)
	}
}

// TestRepro_ManagerRetryLosesLastError asserts manager.go:414 behaviour: the
// last real provider error is NOT wrapped into ErrAllProvidersFailed for the
// non-stream Generate path (unlike the stream path at manager.go:470).
func TestRepro_ManagerRetryLosesLastError(t *testing.T) {
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	p := NewMockProvider("mock-fail", "fail-model", nil)
	p.SetFailRate(100)
	if err := m.RegisterProvider(p, ProviderCfg{Name: "mock-fail", Enabled: true, Priority: 1}); err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err := m.Generate(context.Background(), CompletionRequest{Prompt: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "all AI providers failed") {
		t.Errorf("expected masked error, got: %q", err.Error())
	}
	if strings.Contains(err.Error(), "provider not available") {
		t.Errorf("real cause should be lost (manager.go:414 returns bare ErrAllProvidersFailed), got: %q", err.Error())
	}
}
