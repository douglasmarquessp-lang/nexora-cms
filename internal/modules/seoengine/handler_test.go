package seoengine

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
	"nexora/internal/modules/auth"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func withSiteID(r *http.Request) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, middleware.CtxSiteID, uuid.New())
	return r.WithContext(ctx)
}

func withUserID(r *http.Request) *http.Request {
	uid := uuid.New()
	ctx := r.Context()
	ctx = context.WithValue(ctx, auth.CtxUserID, uid)
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

// --- Projects ---

func TestHandler_CreateProject(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/seoengine", strings.NewReader(`{"title":"Test Project","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		rest.AdaptHandler(h.CreateProject).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/seoengine", strings.NewReader(`{"title":"Test Project","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.CreateProject).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("empty title", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/seoengine", strings.NewReader(`{"title":"","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.CreateProject).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/seoengine", strings.NewReader(`{"title":"Test Project","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.CreateProject).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_GetProject(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/seoengine/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetProject).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/seoengine/invalid", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.GetProject).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no db", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/seoengine/"+uuid.New().String(), nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetProject).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_UpdateProject(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/seoengine/"+uuid.New().String(), strings.NewReader(`{"language":"en"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.UpdateProject).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no db", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/seoengine/"+uuid.New().String(), strings.NewReader(`{"language":"en"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.UpdateProject).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_DeleteProject(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/seoengine/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.DeleteProject).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no db", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/seoengine/"+uuid.New().String(), nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.DeleteProject).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

// --- Audit ---

func TestHandler_RunFullAudit(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/seoengine/"+uuid.New().String()+"/audit", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.RunFullAudit).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no db", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/seoengine/"+uuid.New().String()+"/audit", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.RunFullAudit).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

// --- Clusters ---

func TestHandler_CreateCluster(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/seoengine/clusters", strings.NewReader(`{"name":"Test Cluster"}`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.CreateCluster).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/seoengine/clusters", strings.NewReader(`{"name":""}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.CreateCluster).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Improvements ---

func TestHandler_AddImprovement(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/seoengine/"+uuid.New().String()+"/improvements", strings.NewReader(`{"category":"title","issue":"test","suggestion":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.AddImprovement).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty issue", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/seoengine/"+uuid.New().String()+"/improvements", strings.NewReader(`{"category":"title","issue":"","suggestion":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.AddImprovement).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty suggestion", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/seoengine/"+uuid.New().String()+"/improvements", strings.NewReader(`{"category":"title","issue":"test","suggestion":""}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.AddImprovement).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no db", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/seoengine/"+uuid.New().String()+"/improvements", strings.NewReader(`{"category":"title","issue":"test","suggestion":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.AddImprovement).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_UpdateImprovement(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/seoengine/"+uuid.New().String()+"/improvements/"+uuid.New().String(), strings.NewReader(`{"status":"applied"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "improvementID": uuid.New().String()})
		rest.AdaptHandler(h.UpdateImprovement).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no db", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/seoengine/"+uuid.New().String()+"/improvements/"+uuid.New().String(), strings.NewReader(`{"status":"applied"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "improvementID": uuid.New().String()})
		rest.AdaptHandler(h.UpdateImprovement).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

// --- Keyword Analysis ---

func TestHandler_AnalyzeKeywords(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/seoengine/keywords/analyze", strings.NewReader(`{"keywords":["test"]}`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.AnalyzeKeywords).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Dashboard ---

func TestHandler_GetDashboardStats(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/seoengine/stats", nil)
		rest.AdaptHandler(h.GetDashboardStats).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_GetMetrics(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/seoengine/metrics", nil)
		rest.AdaptHandler(h.GetMetrics).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Detection ---

func TestHandler_DetectOrphans(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/seoengine/orphans", nil)
		rest.AdaptHandler(h.DetectOrphans).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_DetectCannibalization(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/seoengine/cannibalization", nil)
		rest.AdaptHandler(h.DetectCannibalization).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_DetectContentGaps(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/seoengine/content-gaps", nil)
		rest.AdaptHandler(h.DetectContentGaps).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}
