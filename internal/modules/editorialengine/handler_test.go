package editorialengine

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

// --- Pipeline ---

func TestHandler_CreatePipeline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/editorial-engine/pipelines", strings.NewReader(`{"article_job_id":"`+uuid.New().String()+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		rest.AdaptHandler(h.CreatePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/editorial-engine/pipelines", strings.NewReader(`{invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.CreatePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/editorial-engine/pipelines", strings.NewReader(`{"article_job_id":"`+uuid.New().String()+`"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.CreatePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_GetPipeline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/pipelines/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetPipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/pipelines/bad-id", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.GetPipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListPipelines(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/pipelines", nil)
		rest.AdaptHandler(h.ListPipelines).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/pipelines", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.ListPipelines).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_UpdatePipeline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/editorial-engine/pipelines/"+uuid.New().String(), strings.NewReader(`{"status":"completed"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.UpdatePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/editorial-engine/pipelines/bad-id", strings.NewReader(`{"status":"completed"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.UpdatePipeline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListPipelineStages(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/pipelines/bad-id/stages", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.ListPipelineStages).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_GetPipelineStage(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/pipelines/bad-id/stages/"+uuid.New().String(), nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id", "stageID": uuid.New().String()})
		rest.AdaptHandler(h.GetPipelineStage).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid stageID", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/pipelines/"+uuid.New().String()+"/stages/bad-id", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "stageID": "bad-id"})
		rest.AdaptHandler(h.GetPipelineStage).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_UpdatePipelineStage(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/editorial-engine/pipelines/bad-id/stages/"+uuid.New().String(), strings.NewReader(`{"status":"completed"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id", "stageID": uuid.New().String()})
		rest.AdaptHandler(h.UpdatePipelineStage).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Style Rules ---

func TestHandler_GetStyleRules(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/styles", nil)
		rest.AdaptHandler(h.GetStyleRules).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("success (no db)", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/styles", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.GetStyleRules).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_UpsertStyleRules(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/editorial-engine/styles", strings.NewReader(`{"tone":"formal"}`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.UpsertStyleRules).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/editorial-engine/styles", strings.NewReader(`{invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.UpsertStyleRules).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- SEO ---

func TestHandler_CreateSEOData(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/editorial-engine/jobs/"+uuid.New().String()+"/seo", strings.NewReader(`{"primary_keyword":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.CreateSEOData).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/editorial-engine/jobs/bad-id/seo", strings.NewReader(`{"primary_keyword":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.CreateSEOData).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_GetSEOData(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/jobs/"+uuid.New().String()+"/seo", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetSEOData).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/jobs/bad-id/seo", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.GetSEOData).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_UpdateSEOData(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/editorial-engine/jobs/"+uuid.New().String()+"/seo", strings.NewReader(`{"meta_title":"Test"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.UpdateSEOData).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Quality ---

func TestHandler_CreateQualityScore(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/editorial-engine/jobs/"+uuid.New().String()+"/quality", strings.NewReader(`{"seo_score":85}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.CreateQualityScore).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/editorial-engine/jobs/bad-id/quality", strings.NewReader(`{"seo_score":85}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.CreateQualityScore).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_GetQualityScore(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/jobs/"+uuid.New().String()+"/quality", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetQualityScore).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/jobs/bad-id/quality", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.GetQualityScore).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListQualityScores(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/jobs/bad-id/quality/history", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.ListQualityScores).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Translation ---

func TestHandler_CreateTranslation(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/editorial-engine/jobs/"+uuid.New().String()+"/translations", strings.NewReader(`{"source_language":"en","target_language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.CreateTranslation).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/editorial-engine/jobs/bad-id/translations", strings.NewReader(`{"source_language":"en","target_language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.CreateTranslation).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_GetTranslation(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/jobs/"+uuid.New().String()+"/translations/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "translationID": uuid.New().String()})
		rest.AdaptHandler(h.GetTranslation).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid translationID", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/jobs/"+uuid.New().String()+"/translations/bad-id", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "translationID": "bad-id"})
		rest.AdaptHandler(h.GetTranslation).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListTranslations(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/jobs/"+uuid.New().String()+"/translations", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.ListTranslations).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_UpdateTranslation(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/editorial-engine/jobs/"+uuid.New().String()+"/translations/"+uuid.New().String(), strings.NewReader(`{"status":"completed"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "translationID": uuid.New().String()})
		rest.AdaptHandler(h.UpdateTranslation).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid translationID", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/editorial-engine/jobs/"+uuid.New().String()+"/translations/bad-id", strings.NewReader(`{"status":"completed"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "translationID": "bad-id"})
		rest.AdaptHandler(h.UpdateTranslation).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Prompt ---

func TestHandler_GetPromptData(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/jobs/"+uuid.New().String()+"/prompt", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetPromptData).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/editorial-engine/jobs/bad-id/prompt", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.GetPromptData).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_CreatePromptData(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/editorial-engine/jobs/"+uuid.New().String()+"/prompt", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.CreatePromptData).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/editorial-engine/jobs/bad-id/prompt", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.CreatePromptData).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}
