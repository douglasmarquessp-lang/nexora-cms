package posts

import (
	"context"
	"encoding/json"
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

func TestHandler_Create(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(`{"title":"Test"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		rest.AdaptHandler(h.Create).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(`{"title":"Test"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.Create).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(`{invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.Create).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("title required", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(`{"status":"draft"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.Create).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader(`{"title":"Test"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.Create).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})
}

func TestHandler_Get(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/posts/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.Get).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/posts/invalid", nil)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		req = withSiteID(req)
		rest.AdaptHandler(h.Get).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		postID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/posts/"+postID, nil)
		req = withChiParams(req, map[string]string{"id": postID})
		req = withSiteID(req)
		rest.AdaptHandler(h.Get).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})
}

func TestHandler_GetBySlug(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/posts/by-slug?slug=test", nil)
		rest.AdaptHandler(h.GetBySlug).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing slug param", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/posts/by-slug", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.GetBySlug).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/posts/by-slug?slug=test", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.GetBySlug).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})
}

func TestHandler_List(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/posts", nil)
		rest.AdaptHandler(h.List).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/posts", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.List).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/posts?status=invalid", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.List).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503 (nil db), got %d", rec.Code)
		}
	})
}

func TestHandler_Update(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/posts/"+uuid.New().String(), strings.NewReader(`{"title":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/posts/invalid", strings.NewReader(`{"title":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": "invalid"})
		req = withSiteID(req)
		rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/posts/"+body, strings.NewReader(`{invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": body})
		req = withSiteID(req)
		rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		postID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/posts/"+postID, strings.NewReader(`{"title":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": postID})
		req = withSiteID(req)
		rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})
}

func TestHandler_Delete(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/posts/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.Delete).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/posts/invalid", nil)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		req = withSiteID(req)
		rest.AdaptHandler(h.Delete).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		postID := uuid.New().String()
		req := httptest.NewRequest(http.MethodDelete, "/posts/"+postID, nil)
		req = withChiParams(req, map[string]string{"id": postID})
		req = withSiteID(req)
		rest.AdaptHandler(h.Delete).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})
}

func TestHandler_SetStatus(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/posts/"+uuid.New().String()+"/status", strings.NewReader(`{"status":"published"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.SetStatus).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/posts/invalid/status", strings.NewReader(`{"status":"published"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": "invalid"})
		req = withSiteID(req)
		rest.AdaptHandler(h.SetStatus).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		postID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/posts/"+postID+"/status", strings.NewReader(`{invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": postID})
		req = withSiteID(req)
		rest.AdaptHandler(h.SetStatus).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		postID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/posts/"+postID+"/status", strings.NewReader(`{"status":""}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": postID})
		req = withSiteID(req)
		rest.AdaptHandler(h.SetStatus).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		postID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/posts/"+postID+"/status", strings.NewReader(`{"status":"published"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": postID})
		req = withSiteID(req)
		rest.AdaptHandler(h.SetStatus).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})
}

func TestHandler_ResponseFormat(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/posts/invalid", nil)
	req = withChiParams(req, map[string]string{"id": "invalid"})
	req = withSiteID(req)
	rest.AdaptHandler(h.Get).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	errObj, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error object in response")
	}
	if errObj["code"] != "INVALID_ID" {
		t.Errorf("expected code INVALID_ID, got %v", errObj["code"])
	}
}
