package site

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"nexora/internal/api/rest"
	"nexora/internal/modules/auth"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

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
	if h.log != log {
		t.Error("handler logger pointer mismatch")
	}
}

func TestNewHandler_NilService(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	h := NewHandler(nil, log)

	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestHandler_Create(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("empty body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		rest.AdaptHandler(h.Create).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("name required", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(`{"slug":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		rest.AdaptHandler(h.Create).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("slug required", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(`{"name":"Test"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		rest.AdaptHandler(h.Create).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(`{"name":"Test","slug":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.Create).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(`{"name":"Test","slug":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withUserID(req)
		rest.AdaptHandler(h.Create).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_Get(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/sites/invalid", nil)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.Get).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		siteID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/sites/"+siteID, nil)
		req = withChiParams(req, map[string]string{"id": siteID})
		rest.AdaptHandler(h.Get).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_List(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("no auth", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/sites", nil)
		rest.AdaptHandler(h.List).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/sites", nil)
		req = withUserID(req)
		rest.AdaptHandler(h.List).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_Update(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/sites/invalid", strings.NewReader(`{"name":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		siteID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/sites/"+siteID, strings.NewReader(`{"name":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": siteID})
		rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
		if rec.Code == http.StatusOK {
			t.Error("expected error response")
		}
	})
}

func TestHandler_Delete(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/sites/invalid", nil)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.Delete).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		siteID := uuid.New().String()
		req := httptest.NewRequest(http.MethodDelete, "/sites/"+siteID, nil)
		req = withChiParams(req, map[string]string{"id": siteID})
		rest.AdaptHandler(h.Delete).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_AddDomain(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid site id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/sites/invalid/domains", strings.NewReader(`{"domain":"example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.AddDomain).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty domain", func(t *testing.T) {
		rec := httptest.NewRecorder()
		siteID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPost, "/sites/"+siteID+"/domains", strings.NewReader(`{"domain":""}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": siteID})
		rest.AdaptHandler(h.AddDomain).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandler_RemoveDomain(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid domain id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/sites/siteid/domains/invalid", nil)
		req = withChiParams(req, map[string]string{"domainID": "invalid"})
		rest.AdaptHandler(h.RemoveDomain).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		domainID := uuid.New().String()
		req := httptest.NewRequest(http.MethodDelete, "/sites/siteid/domains/"+domainID, nil)
		req = withChiParams(req, map[string]string{"domainID": domainID})
		rest.AdaptHandler(h.RemoveDomain).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_ListDomains(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid site id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/sites/invalid/domains", nil)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.ListDomains).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		siteID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/sites/"+siteID+"/domains", nil)
		req = withChiParams(req, map[string]string{"id": siteID})
		rest.AdaptHandler(h.ListDomains).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_SetPrimaryDomain(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid site id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/sites/invalid/domains/"+uuid.New().String()+"/primary", nil)
		req = withChiParams(req, map[string]string{"id": "invalid", "domainID": uuid.New().String()})
		rest.AdaptHandler(h.SetPrimaryDomain).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid domain id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		siteID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/sites/"+siteID+"/domains/invalid/primary", nil)
		req = withChiParams(req, map[string]string{"id": siteID, "domainID": "invalid"})
		rest.AdaptHandler(h.SetPrimaryDomain).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		siteID := uuid.New().String()
		domainID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/sites/"+siteID+"/domains/"+domainID+"/primary", nil)
		req = withChiParams(req, map[string]string{"id": siteID, "domainID": domainID})
		rest.AdaptHandler(h.SetPrimaryDomain).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_GetGlobalSetting(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/system/config/testkey", nil)
		req = withChiParams(req, map[string]string{"key": "testkey"})
		rest.AdaptHandler(h.GetGlobalSetting).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_SetGlobalSetting(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid type", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/system/config", strings.NewReader(`{"key":"test","value":"val","type":"invalid"}`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.SetGlobalSetting).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/system/config", strings.NewReader(`{"key":"test","value":"val"}`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.SetGlobalSetting).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_ListGlobalSettings(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/system/config", nil)
		rest.AdaptHandler(h.ListGlobalSettings).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_GetSiteSetting(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid site id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/sites/invalid/settings/key", nil)
		req = withChiParams(req, map[string]string{"id": "invalid", "key": "mykey"})
		rest.AdaptHandler(h.GetSiteSetting).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		siteID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/sites/"+siteID+"/settings/mykey", nil)
		req = withChiParams(req, map[string]string{"id": siteID, "key": "mykey"})
		rest.AdaptHandler(h.GetSiteSetting).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_SetSiteSetting(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid site id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/sites/invalid/settings", strings.NewReader(`{"key":"k","value":"v"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.SetSiteSetting).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("empty key validation", func(t *testing.T) {
		rec := httptest.NewRecorder()
		siteID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/sites/"+siteID+"/settings", strings.NewReader(`{"key":"","value":"v"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": siteID})
		rest.AdaptHandler(h.SetSiteSetting).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		siteID := uuid.New().String()
		req := httptest.NewRequest(http.MethodPut, "/sites/"+siteID+"/settings", strings.NewReader(`{"key":"mykey","value":"myval"}`))
		req.Header.Set("Content-Type", "application/json")
		req = withChiParams(req, map[string]string{"id": siteID})
		rest.AdaptHandler(h.SetSiteSetting).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_ListSiteSettings(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid site id", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/sites/invalid/settings", nil)
		req = withChiParams(req, map[string]string{"id": "invalid"})
		rest.AdaptHandler(h.ListSiteSettings).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("db error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		siteID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/sites/"+siteID+"/settings", nil)
		req = withChiParams(req, map[string]string{"id": siteID})
		rest.AdaptHandler(h.ListSiteSettings).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_ResponseFormat(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sites/invalid", nil)
	req = withChiParams(req, map[string]string{"id": "invalid"})
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
