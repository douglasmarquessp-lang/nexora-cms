package articlepipeline

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
	ctx := context.WithValue(r.Context(), middleware.CtxSiteID, uuid.New())
	return r.WithContext(ctx)
}

func withUserID(r *http.Request) *http.Request {
	uid := uuid.New()
	ctx := context.WithValue(r.Context(), auth.CtxUserID, uid)
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

// --- CreatePipeline ---

func TestHandler_CreatePipeline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline",
			strings.NewReader(`{"title":"Test","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		rest.AdaptHandler(h.CreatePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline",
			strings.NewReader(`{"title":"Test","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.CreatePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("empty title", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline",
			strings.NewReader(`{"title":"","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.CreatePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline",
			strings.NewReader(`invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.CreatePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- GetPipeline ---

func TestHandler_GetPipeline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/articlepipeline/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetPipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/articlepipeline/invalid", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.GetPipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- ListPipelines ---

func TestHandler_ListPipelines(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/articlepipeline", nil)
		rest.AdaptHandler(h.ListPipelines).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no db returns 500", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/articlepipeline", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.ListPipelines).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

// --- UpdatePipeline ---

func TestHandler_UpdatePipeline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/articlepipeline/"+uuid.New().String(),
			strings.NewReader(`{"title":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.UpdatePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/articlepipeline/"+uuid.New().String(),
			strings.NewReader(`invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.UpdatePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- DeletePipeline ---

func TestHandler_DeletePipeline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/articlepipeline/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.DeletePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		jobID := uuid.New().String()
		req := httptest.NewRequest(http.MethodDelete, "/articlepipeline/"+jobID, nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": jobID})
		rest.AdaptHandler(h.DeletePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

// --- StartPipeline ---

func TestHandler_StartPipeline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/"+uuid.New().String()+"/start", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.StartPipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- PausePipeline ---
func TestHandler_PausePipeline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/"+uuid.New().String()+"/pause", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.PausePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- ResumePipeline ---
func TestHandler_ResumePipeline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/"+uuid.New().String()+"/resume", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.ResumePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- CancelPipeline ---
func TestHandler_CancelPipeline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/"+uuid.New().String()+"/cancel", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.CancelPipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- RetryStage ---
func TestHandler_RetryStage(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/"+uuid.New().String()+"/retry",
			strings.NewReader(`{"stage_name":"research"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.RetryStage).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty stage name", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/"+uuid.New().String()+"/retry",
			strings.NewReader(`{"stage_name":""}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.RetryStage).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/"+uuid.New().String()+"/retry",
			strings.NewReader(`invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.RetryStage).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- RestartPipeline ---
func TestHandler_RestartPipeline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/"+uuid.New().String()+"/restart", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.RestartPipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- GetPipelineStages ---
func TestHandler_GetPipelineStages(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/articlepipeline/invalid/stages", nil)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.GetPipelineStages).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- UpdateStage ---
func TestHandler_UpdateStage(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/articlepipeline/"+uuid.New().String()+"/stages/research",
			strings.NewReader(`{"status":"completed"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "stageName": "research"})
		rest.AdaptHandler(h.UpdateStage).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/articlepipeline/"+uuid.New().String()+"/stages/research",
			strings.NewReader(`invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "stageName": "research"})
		rest.AdaptHandler(h.UpdateStage).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- RecordMetric ---
func TestHandler_RecordMetric(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/invalid/metrics",
			strings.NewReader(`{"metric_name":"test","metric_value":1.0}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.RecordMetric).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing metric name", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/"+uuid.New().String()+"/metrics",
			strings.NewReader(`{"metric_name":"","metric_value":1.0}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.RecordMetric).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- GetPipelineMetrics ---
func TestHandler_GetPipelineMetrics(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/articlepipeline/invalid/metrics", nil)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.GetPipelineMetrics).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- CreateQualityReport ---
func TestHandler_CreateQualityReport(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/invalid/quality",
			strings.NewReader(`{"stage_name":"research","score":85}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.CreateQualityReport).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty stage name", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/"+uuid.New().String()+"/quality",
			strings.NewReader(`{"stage_name":"","score":85}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.CreateQualityReport).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- GetQualityReports ---
func TestHandler_GetQualityReports(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/articlepipeline/invalid/quality", nil)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.GetQualityReports).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- CreateCandidate ---
func TestHandler_CreateCandidate(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/"+uuid.New().String()+"/publish",
			strings.NewReader(`{"title":"Test","quality_score":85}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.CreateCandidate).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty title", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/"+uuid.New().String()+"/publish",
			strings.NewReader(`{"title":"","quality_score":85}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.CreateCandidate).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/articlepipeline/"+uuid.New().String()+"/publish",
			strings.NewReader(`invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.CreateCandidate).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- ListCandidates ---
func TestHandler_ListCandidates(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/articlepipeline/candidates", nil)
		rest.AdaptHandler(h.ListCandidates).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no db returns 500", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/articlepipeline/candidates", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.ListCandidates).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

// --- GetPipelineStats ---
func TestHandler_GetPipelineStats(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/articlepipeline/stats", nil)
		rest.AdaptHandler(h.GetPipelineStats).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no db returns 500", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/articlepipeline/stats", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.GetPipelineStats).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}
