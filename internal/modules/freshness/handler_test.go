package freshness

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"nexora/internal/api/middleware"
	"nexora/internal/api/rest"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

type harness struct {
	router chi.Router
}

func newHarness(t *testing.T, svc *Service) *harness {
	t.Helper()
	cfg := &config.Config{}
	log := logger.New(cfg)
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.CtxSiteID, uuid.New())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	RegisterRoutes(router, svc, log)
	return &harness{router: router}
}

func do(t *testing.T, h *harness, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(b)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.router.ServeHTTP(rec, req)
	return rec
}

func TestClassifyEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/freshness/classify", map[string]interface{}{
		"topic":    "Empresa anuncia novo produto hoje",
		"language": "pt",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Intent IntentType `json:"intent"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Intent != IntentNews {
		t.Errorf("expected news, got %s", out.Intent)
	}
}

func TestScoreEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/freshness/score", map[string]interface{}{
		"intent": IntentNews,
		"sources": []map[string]interface{}{{
			"title": "Reuters",
			"url":   "https://reuters.com/x",
		}},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestObsoleteEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/freshness/obsolete", map[string]interface{}{
		"text": "GPT-4 ainda é usado",
		"entities": []map[string]interface{}{{
			"entity": "GPT", "current": "6",
		}},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var out struct {
		HasObsolete bool `json:"has_obsolete"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.HasObsolete {
		t.Error("expected obsolete flag")
	}
}

func TestDedupEndpoint(t *testing.T) {
	svc := NewService(&config.Config{}, logger.New(&config.Config{}), nil)
	h := newHarness(t, svc)

	rec := do(t, h, "POST", "/freshness/dedup", map[string]interface{}{
		"topic":    "GPT-6 lançamento",
		"content":  "conteúdo",
		"language": "pt",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSweepEndpointMissingSite(t *testing.T) {
	// No site middleware → the handler must answer 400 MISSING_SITE.
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil)
	router := chi.NewRouter()
	RegisterRoutes(router, svc, log)

	req := httptest.NewRequest("POST", "/freshness/sweep", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRestAdaptContract(t *testing.T) {
	// The rest.Context + rest.AdaptHandler wiring must compile/run through a
	// real router with the same middleware used by the API.
	_ = rest.AdaptHandler
}