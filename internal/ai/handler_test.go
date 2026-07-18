package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"nexora/internal/api/middleware"
	"nexora/internal/api/rest"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func withSiteID(r *http.Request) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, middleware.CtxSiteID, uuid.New())
	return r.WithContext(ctx)
}

func setupAIHandlerTest(t *testing.T) (*Handler, *Manager) {
	t.Helper()
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	p := NewMockProvider("mock", "mock-model", nil)
	m.RegisterProvider(p, ProviderCfg{Name: "mock", Enabled: true, Priority: 1, Weight: 10})
	h := NewHandler(m, log)
	return h, m
}

func TestNewHandler(t *testing.T) {
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	h := NewHandler(m, log)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestHandler_ListProviders(t *testing.T) {
	h, _ := setupAIHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ai/providers", nil)
	rest.AdaptHandler(h.ListProviders).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_HealthCheck_MissingSite(t *testing.T) {
	h, _ := setupAIHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ai/health", nil)
	rest.AdaptHandler(h.HealthCheck).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_HealthCheck_Success(t *testing.T) {
	h, _ := setupAIHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ai/health", nil)
	req = withSiteID(req)
	rest.AdaptHandler(h.HealthCheck).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_TestProvider_MissingSite(t *testing.T) {
	h, _ := setupAIHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ai/test", nil)
	rest.AdaptHandler(h.TestProvider).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_TestProvider_Default(t *testing.T) {
	h, _ := setupAIHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ai/test", nil)
	req = withSiteID(req)
	rest.AdaptHandler(h.TestProvider).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_TestProvider_Specific(t *testing.T) {
	h, _ := setupAIHandlerTest(t)

	rec := httptest.NewRecorder()
	body := `{"provider":"mock"}`
	req := httptest.NewRequest(http.MethodPost, "/ai/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withSiteID(req)
	rest.AdaptHandler(h.TestProvider).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_TestProvider_NotFound(t *testing.T) {
	log := logger.New(&config.Config{})
	m := NewManager(DefaultConfig(), log)
	h := NewHandler(m, log)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ai/test", nil)
	req = withSiteID(req)
	rest.AdaptHandler(h.TestProvider).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_PreviewPrompt_MissingSite(t *testing.T) {
	h, _ := setupAIHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ai/prompt", strings.NewReader(`{"template_id":"article"}`))
	req.Header.Set("Content-Type", "application/json")
	rest.AdaptHandler(h.PreviewPrompt).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_PreviewPrompt_Success(t *testing.T) {
	h, _ := setupAIHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ai/prompt", strings.NewReader(`{"template_id":"article","variables":{"title":"Test"}}`))
	req.Header.Set("Content-Type", "application/json")
	req = withSiteID(req)
	rest.AdaptHandler(h.PreviewPrompt).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_PreviewPrompt_InvalidBody(t *testing.T) {
	h, _ := setupAIHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ai/prompt", strings.NewReader(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	req = withSiteID(req)
	rest.AdaptHandler(h.PreviewPrompt).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_PreviewPrompt_MissingTemplate(t *testing.T) {
	h, _ := setupAIHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ai/prompt", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req = withSiteID(req)
	rest.AdaptHandler(h.PreviewPrompt).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_PreviewPrompt_TemplateNotFound(t *testing.T) {
	h, _ := setupAIHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ai/prompt", strings.NewReader(`{"template_id":"nonexistent"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withSiteID(req)
	rest.AdaptHandler(h.PreviewPrompt).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_GetCapabilities_MissingSite(t *testing.T) {
	h, _ := setupAIHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ai/capabilities", nil)
	rest.AdaptHandler(h.GetCapabilities).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_GetCapabilities_Success(t *testing.T) {
	h, _ := setupAIHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ai/capabilities", nil)
	req = withSiteID(req)
	rest.AdaptHandler(h.GetCapabilities).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
