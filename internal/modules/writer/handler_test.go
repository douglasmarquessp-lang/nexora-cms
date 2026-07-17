package writer

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

func TestHandler_CreateJob(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer", strings.NewReader(`{"headline":"Test","language":"en"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		rest.AdaptHandler(h.CreateJob).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer", strings.NewReader(`{"headline":"Test","language":"en"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		rest.AdaptHandler(h.CreateJob).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("empty headline", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer", strings.NewReader(`{"headline":"","language":"en"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.CreateJob).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid language", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer", strings.NewReader(`{"headline":"Test","language":"fr"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.CreateJob).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer", strings.NewReader(`{invalid`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		rest.AdaptHandler(h.CreateJob).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_GetJob(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/writer/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetJob).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/writer/bad-id", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.GetJob).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListJobs(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/writer", nil)
		rest.AdaptHandler(h.ListJobs).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_UpdateJob(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/writer/"+uuid.New().String(), strings.NewReader(`{"status":"approved"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.UpdateJob).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/writer/bad-id", strings.NewReader(`{"status":"approved"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.UpdateJob).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_DeleteJob(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/writer/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.DeleteJob).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/writer/bad-id", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.DeleteJob).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_CreateOutline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer/bad-id/outline", strings.NewReader(`{"sections":[]}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.CreateOutline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListOutline(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/writer/bad-id/outline", nil)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.ListOutline).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_CreateSection(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer/bad-id/sections", strings.NewReader(`{"title":"Intro","position":0}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.CreateSection).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListSections(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/writer/bad-id/sections", nil)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.ListSections).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_GetSection(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/writer/bad-id/sections/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": "bad-id", "sectionID": uuid.New().String()})
		rest.AdaptHandler(h.GetSection).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid sectionID", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/writer/"+uuid.New().String()+"/sections/bad-id", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "sectionID": "bad-id"})
		rest.AdaptHandler(h.GetSection).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_UpdateSection(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/writer/bad-id/sections/"+uuid.New().String(), strings.NewReader(`{"title":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": "bad-id", "sectionID": uuid.New().String()})
		rest.AdaptHandler(h.UpdateSection).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid sectionID", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/writer/"+uuid.New().String()+"/sections/bad-id", strings.NewReader(`{"title":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "sectionID": "bad-id"})
		rest.AdaptHandler(h.UpdateSection).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_CreateVersion(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer/"+uuid.New().String()+"/versions", strings.NewReader(`{"change_log":"First version"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.CreateVersion).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer/"+uuid.New().String()+"/versions", strings.NewReader(`{"change_log":"First version"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.CreateVersion).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer/bad-id/versions", strings.NewReader(`{"change_log":"First version"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withSiteID(req)
		req = withUserID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.CreateVersion).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListVersions(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/writer/bad-id/versions", nil)
		req = withChiParams(req, map[string]string{"id": "bad-id"})
		rest.AdaptHandler(h.ListVersions).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_GetVersion(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/writer/bad-id/versions/"+uuid.New().String(), nil)
		req = withChiParams(req, map[string]string{"id": "bad-id", "versionID": uuid.New().String()})
		rest.AdaptHandler(h.GetVersion).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid versionID", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/writer/"+uuid.New().String()+"/versions/bad-id", nil)
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "versionID": "bad-id"})
		rest.AdaptHandler(h.GetVersion).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_RestoreVersion(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer/"+uuid.New().String()+"/versions/"+uuid.New().String()+"/restore", nil)
		req = withUserID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "versionID": uuid.New().String()})
		rest.AdaptHandler(h.RestoreVersion).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer/"+uuid.New().String()+"/versions/"+uuid.New().String()+"/restore", nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "versionID": uuid.New().String()})
		rest.AdaptHandler(h.RestoreVersion).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer/bad-id/versions/"+uuid.New().String()+"/restore", nil)
		req = withSiteID(req)
		req = withUserID(req)
		req = withChiParams(req, map[string]string{"id": "bad-id", "versionID": uuid.New().String()})
		rest.AdaptHandler(h.RestoreVersion).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid versionID", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/writer/"+uuid.New().String()+"/versions/bad-id/restore", nil)
		req = withSiteID(req)
		req = withUserID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String(), "versionID": "bad-id"})
		rest.AdaptHandler(h.RestoreVersion).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_ListStyles(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("missing site", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/writer/styles", nil)
		rest.AdaptHandler(h.ListStyles).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_GetJob_NotFound(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("no db", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/writer/"+uuid.New().String(), nil)
		req = withSiteID(req)
		req = withChiParams(req, map[string]string{"id": uuid.New().String()})
		rest.AdaptHandler(h.GetJob).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_CreateJob_NoDB(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/writer", strings.NewReader(`{"headline":"Test","language":"en"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withSiteID(req)
	req = withUserID(req)
	rest.AdaptHandler(h.CreateJob).ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandler_ListJobs_NoDB(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/writer", nil)
	req = withSiteID(req)
	rest.AdaptHandler(h.ListJobs).ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandler_NonJSONResponse(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/writer/"+uuid.New().String()+"/outline", nil)
	req = withChiParams(req, map[string]string{"id": uuid.New().String()})
	rest.AdaptHandler(h.ListOutline).ServeHTTP(rec, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Logf("non-json response (expected): %s", rec.Body.String())
	}
}
