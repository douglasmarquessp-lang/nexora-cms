package assets

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
	"nexora/internal/pkg/storage"
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

func setupAssetHandlerTest(t *testing.T) (*Handler, *Service) {
	t.Helper()
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)
	h := NewHandler(svc, log)
	return h, svc
}

func TestNewHandler(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	st := storage.NewLocalDriver("/tmp/test-assets", "/uploads")
	svc := NewService(cfg, log, nil, nil, st)
	h := NewHandler(svc, log)

	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.svc != svc {
		t.Error("handler service pointer mismatch")
	}
}

func TestHandler_Upload(t *testing.T) {
	h, _ := setupAssetHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/assets/upload", nil)
		req = withUserID(req)
		rest.AdaptHandler(h.Upload).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/assets/upload", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.Upload).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})
}

func TestHandler_Get(t *testing.T) {
	h, _ := setupAssetHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.Get).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/invalid", nil)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		req = withSiteID(req)
		rest.AdaptHandler(h.Get).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		assetID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/assets/"+assetID, nil)
		req = withChiParams(req, map[string]string{"id": assetID})
		req = withSiteID(req)
		rest.AdaptHandler(h.Get).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})
}

func TestHandler_List(t *testing.T) {
	h, _ := setupAssetHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets", nil)
		rest.AdaptHandler(h.List).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets", nil)
		req = withSiteID(req)
		rest.AdaptHandler(h.List).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})
}

func TestHandler_Update(t *testing.T) {
	h, _ := setupAssetHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/assets/"+uuid.New().String(), strings.NewReader(`{"alt_text":"new alt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/assets/invalid", strings.NewReader(`{"alt_text":"new alt"}`))
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
		assetID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/assets/"+assetID, strings.NewReader(`{invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": assetID})
		req = withSiteID(req)
		rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		assetID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/assets/"+assetID, strings.NewReader(`{"alt_text":"new alt"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": assetID})
		req = withSiteID(req)
		rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})
}

func TestHandler_Delete(t *testing.T) {
	h, _ := setupAssetHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/assets/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.Delete).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/assets/invalid", nil)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		req = withSiteID(req)
		rest.AdaptHandler(h.Delete).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		assetID := uuid.New().String()
		req := httptest.NewRequest(http.MethodDelete, "/assets/"+assetID, nil)
		req = withChiParams(req, map[string]string{"id": assetID})
		req = withSiteID(req)
		rest.AdaptHandler(h.Delete).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})
}

func TestHandler_LinkToPost(t *testing.T) {
	h, _ := setupAssetHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"post_id":"` + uuid.New().String() + `","asset_id":"` + uuid.New().String() + `","type":"gallery"}`
		req := httptest.NewRequest(http.MethodPost, "/assets/link", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.LinkToPost).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty post_id and asset_id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/assets/link", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.LinkToPost).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/assets/link", strings.NewReader(`{invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.LinkToPost).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"post_id":"` + uuid.New().String() + `","asset_id":"` + uuid.New().String() + `","type":"gallery"}`
		req := httptest.NewRequest(http.MethodPost, "/assets/link", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.LinkToPost).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})
}

func TestHandler_UnlinkFromPost(t *testing.T) {
	h, _ := setupAssetHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/assets/postid/link/assetid", nil)
		req = withChiParams(req, map[string]string{"postID": uuid.New().String(), "assetID": uuid.New().String()})
		rest.AdaptHandler(h.UnlinkFromPost).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid post id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/assets/invalid/link/assetid", nil)
		req = withChiParams(req, map[string]string{"postID": "invalid", "assetID": uuid.New().String()})
		req = withSiteID(req)
		rest.AdaptHandler(h.UnlinkFromPost).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid asset id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/assets/postid/link/invalid", nil)
		req = withChiParams(req, map[string]string{"postID": uuid.New().String(), "assetID": "invalid"})
		req = withSiteID(req)
		rest.AdaptHandler(h.UnlinkFromPost).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_GetPostAssets(t *testing.T) {
	h, _ := setupAssetHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/posts/postid/assets", nil)
		req = withChiParams(req, map[string]string{"postID": uuid.New().String()})
		rest.AdaptHandler(h.GetPostAssets).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid post id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/posts/invalid/assets", nil)
		req = withChiParams(req, map[string]string{"postID": "invalid"})
		req = withSiteID(req)
		rest.AdaptHandler(h.GetPostAssets).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		postID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/posts/"+postID+"/assets", nil)
		req = withChiParams(req, map[string]string{"postID": postID})
		req = withSiteID(req)
		rest.AdaptHandler(h.GetPostAssets).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", rec.Code)
		}
	})
}

func TestHandler_ResponseFormat(t *testing.T) {
	h, _ := setupAssetHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/invalid", nil)
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
