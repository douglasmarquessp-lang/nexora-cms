package publisher

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

// --- Publish ---

func TestHandler_Publish(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/publisher/publish", strings.NewReader(`{"title":"Test","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		rest.AdaptHandler(h.Publish).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/publisher/publish", strings.NewReader(`{"title":"Test","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.Publish).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("empty title", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/publisher/publish", strings.NewReader(`{"title":"","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.Publish).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid language", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/publisher/publish", strings.NewReader(`{"title":"Test","language":"fr"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.Publish).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/publisher/publish", strings.NewReader(`{"title":"Test","language":"pt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.Publish).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

// --- Schedule ---

func TestHandler_Schedule(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"publication_id":"` + uuid.New().String() + `","scheduled_at":"2025-01-01T00:00:00Z"}`
		req := httptest.NewRequest(http.MethodPost, "/publisher/schedule", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		rest.AdaptHandler(h.Schedule).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"publication_id":"` + uuid.New().String() + `","scheduled_at":"2025-01-01T00:00:00Z"}`
		req := httptest.NewRequest(http.MethodPost, "/publisher/schedule", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.Schedule).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"publication_id":"` + uuid.New().String() + `","scheduled_at":"2025-01-01T00:00:00Z"}`
		req := httptest.NewRequest(http.MethodPost, "/publisher/schedule", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.Schedule).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

// --- Update ---

func TestHandler_Update(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/publisher/"+uuid.New().String(), strings.NewReader(`{"title":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/publisher/invalid", strings.NewReader(`{"title":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- GetPublication ---

func TestHandler_GetPublication(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetPublication).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher/invalid", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.GetPublication).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Delete ---

func TestHandler_DeletePublication(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/publisher/"+uuid.New().String(), nil)
		req = withUserID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.DeletePublication).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/publisher/"+uuid.New().String(), nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.DeletePublication).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})
}

// --- Unpublish ---

func TestHandler_Unpublish(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/publisher/"+uuid.New().String()+"/unpublish", nil)
		req = withUserID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.Unpublish).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Republish ---

func TestHandler_Republish(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/publisher/"+uuid.New().String()+"/republish", nil)
		req = withUserID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.Republish).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Queue ---

func TestHandler_AddToQueue(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"publication_id":"` + uuid.New().String() + `","action":"publish"}`
		req := httptest.NewRequest(http.MethodPost, "/publisher/queue", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		rest.AdaptHandler(h.AddToQueue).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"publication_id":"` + uuid.New().String() + `","action":"publish"}`
		req := httptest.NewRequest(http.MethodPost, "/publisher/queue", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.AddToQueue).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("invalid action", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"publication_id":"` + uuid.New().String() + `","action":"invalid"}`
		req := httptest.NewRequest(http.MethodPost, "/publisher/queue", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.AddToQueue).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- ListQueue ---

func TestHandler_ListQueue(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher/queue", nil)
		rest.AdaptHandler(h.ListQueue).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Schedules ---

func TestHandler_ListSchedules(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher/schedules", nil)
		rest.AdaptHandler(h.ListSchedules).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Metrics ---

func TestHandler_GetMetricsSummary(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher/metrics/summary", nil)
		rest.AdaptHandler(h.GetMetricsSummary).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Validate Slug ---

func TestHandler_ValidateSlug(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher/validate-slug?slug=test", nil)
		rest.AdaptHandler(h.ValidateSlug).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing slug", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher/validate-slug", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.ValidateSlug).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Generate Slug ---

func TestHandler_GenerateSlug(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher/generate-slug?title=Test", nil)
		rest.AdaptHandler(h.GenerateSlug).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing title", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher/generate-slug", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.GenerateSlug).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- ListPublications ---

func TestHandler_ListPublications(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher", nil)
		rest.AdaptHandler(h.ListPublications).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- CancelSchedule ---

func TestHandler_CancelSchedule(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/publisher/"+uuid.New().String()+"/schedule/"+uuid.New().String()+"/cancel", nil)
		req = withUserID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "scheduleID": uuid.New().String()})
		rest.AdaptHandler(h.CancelSchedule).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- RetryQueue ---

func TestHandler_RetryQueue(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/publisher/queue/"+uuid.New().String()+"/retry", nil)
		req = withUserID(req)
		req = withChiParams(req, map[string]string{"itemID": uuid.New().String()})
		rest.AdaptHandler(h.RetryQueue).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Get Schedule ---

func TestHandler_GetSchedule(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher/"+uuid.New().String()+"/schedule/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "scheduleID": uuid.New().String()})
		rest.AdaptHandler(h.GetSchedule).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Get History ---

func TestHandler_GetHistory(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher/"+uuid.New().String()+"/history", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetHistory).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

// --- Get Metrics ---

func TestHandler_GetMetrics(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher/"+uuid.New().String()+"/metrics", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetMetrics).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/publisher/"+uuid.New().String()+"/metrics", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetMetrics).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}
