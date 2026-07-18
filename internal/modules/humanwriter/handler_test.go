package humanwriter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
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

func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

func setupHandlerTest(t *testing.T) (*Handler, *Service) {
	t.Helper()
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	h := NewHandler(svc, log)
	return h, svc
}

func TestNewHandler(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := NewService(cfg, log, nil, nil)
	h := NewHandler(svc, log)

	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.svc != svc {
		t.Error("handler service pointer mismatch")
	}
}

// --- Profiles ---

func TestHandler_CreateProfile(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/profiles",
			strings.NewReader(`{"slug":"test","name":"Test","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.CreateProfile).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty slug", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/profiles",
			strings.NewReader(`{"slug":"","name":"Test","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.CreateProfile).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/profiles",
			strings.NewReader(`{"slug":"test","name":"","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.CreateProfile).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/profiles",
			strings.NewReader(`invalid json`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.CreateProfile).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_GetProfile(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/profiles/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetProfile).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/profiles/invalid", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.GetProfile).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		profileID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/profiles/"+profileID, nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": profileID})
		rest.AdaptHandler(h.GetProfile).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_ListProfiles(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/profiles", nil)
		rest.AdaptHandler(h.ListProfiles).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no DB returns 500", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/profiles", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.ListProfiles).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_UpdateProfile(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/humanwriter/profiles/"+uuid.New().String(),
			strings.NewReader(`{"name":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.UpdateProfile).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/humanwriter/profiles/"+uuid.New().String(),
			strings.NewReader(`invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.UpdateProfile).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_DeleteProfile(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/humanwriter/profiles/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.DeleteProfile).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		profileID := uuid.New().String()
		req := httptest.NewRequest(http.MethodDelete, "/humanwriter/profiles/"+profileID, nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": profileID})
		rest.AdaptHandler(h.DeleteProfile).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

// --- Rules ---

func TestHandler_ListRules(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/rules", nil)
		rest.AdaptHandler(h.ListRules).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ToggleRule(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/humanwriter/rules/"+uuid.New().String()+"/toggle",
			strings.NewReader(`{"enabled":true}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.ToggleRule).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/humanwriter/rules/"+uuid.New().String()+"/toggle",
			strings.NewReader(`invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.ToggleRule).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ruleID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/humanwriter/rules/"+ruleID+"/toggle",
			strings.NewReader(`{"enabled":true}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": ruleID})
		rest.AdaptHandler(h.ToggleRule).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

// --- Personas ---

func TestHandler_CreatePersona(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/personas",
			strings.NewReader(`{"language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.CreatePersona).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_GetPersona(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/personas/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetPersona).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListPersonas(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/personas", nil)
		rest.AdaptHandler(h.ListPersonas).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_UpdatePersona(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/humanwriter/personas/"+uuid.New().String(),
			strings.NewReader(`{"name":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.UpdatePersona).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		personaID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/humanwriter/personas/"+personaID,
			strings.NewReader(`{"name":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": personaID})
		rest.AdaptHandler(h.UpdatePersona).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_DeletePersona(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/humanwriter/personas/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.DeletePersona).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		personaID := uuid.New().String()
		req := httptest.NewRequest(http.MethodDelete, "/humanwriter/personas/"+personaID, nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": personaID})
		rest.AdaptHandler(h.DeletePersona).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

// --- Vocabulary, Transitions, Patterns, Templates ---

func TestHandler_ListVocabularySets(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/vocabulary", nil)
		rest.AdaptHandler(h.ListVocabularySets).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListTransitions(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/transitions", nil)
		rest.AdaptHandler(h.ListTransitions).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListPatterns(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/patterns", nil)
		rest.AdaptHandler(h.ListPatterns).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListTemplates(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/templates", nil)
		rest.AdaptHandler(h.ListTemplates).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Humanize ---

func TestHandler_Humanize(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/humanize",
			strings.NewReader(`{"text":"Hello world","language":"en"}`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.Humanize).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty text", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/humanize",
			strings.NewReader(`{"text":"","language":"en"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.Humanize).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/humanize",
			strings.NewReader(`invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.Humanize).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/humanize",
			strings.NewReader(`{"text":"This is a test text. It has multiple sentences. Each one should be varied.","language":"en"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.Humanize).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

func TestHandler_BatchHumanize(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/batch",
			strings.NewReader(`{"texts":["Hello world"],"language":"en"}`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.BatchHumanize).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty texts", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/batch",
			strings.NewReader(`{"texts":[],"language":"en"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.BatchHumanize).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/batch",
			strings.NewReader(`{"texts":["First text. With sentences.","Second one."],"language":"en"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.BatchHumanize).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

// --- Analyze ---

func TestHandler_Analyze(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("empty text", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/analyze",
			strings.NewReader(`{"text":""}`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.Analyze).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/analyze",
			strings.NewReader(`invalid`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.Analyze).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/humanwriter/analyze",
			strings.NewReader(`{"text":"This is a test. With multiple sentences.","language":"en"}`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.Analyze).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

// --- Metrics ---

func TestHandler_GetMetrics(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/metrics", nil)
		rest.AdaptHandler(h.GetMetrics).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- History ---

func TestHandler_ListHistory(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/humanwriter/history", nil)
		rest.AdaptHandler(h.ListHistory).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}
