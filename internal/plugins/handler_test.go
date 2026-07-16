package plugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"nexora/internal/api/rest"
)

func setupHandlerTest(t *testing.T) (*Handler, *Manager) {
	t.Helper()
	m := NewManager(&ManagerConfig{
		PluginsDir: t.TempDir(),
	}, testLogger(t), &mockEmitter{})
	return NewHandler(m), m
}

func TestNewHandler(t *testing.T) {
	h, _ := setupHandlerTest(t)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestHandler_List_Empty(t *testing.T) {
	h, _ := setupHandlerTest(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plugins", nil)
	rest.AdaptHandler(h.List).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"plugins":[]`) {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestHandler_List_WithPlugins(t *testing.T) {
	h, m := setupHandlerTest(t)
	instance := &PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test Plugin",
			Version: "1.0.0",
			Author:  "Test",
		},
		Status: PluginStatusInstalled,
	}
	m.registry.Register(instance)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plugins", nil)
	rest.AdaptHandler(h.List).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "test-p") {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestHandler_Get_Found(t *testing.T) {
	h, m := setupHandlerTest(t)
	m.registry.Register(&PluginInstance{
		Manifest: &PluginManifest{
			ID:      "test-p",
			Name:    "Test Plugin",
			Version: "1.0.0",
		},
		Status: PluginStatusInstalled,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plugins/test-p", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "test-p")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rest.AdaptHandler(h.Get).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "test-p") {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestHandler_Get_NotFound(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plugins/nonexistent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rest.AdaptHandler(h.Get).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_Install(t *testing.T) {
	h, m := setupHandlerTest(t)
	createTestPlugin(t, m.cfg.PluginsDir, "new-p", "1.0.0")

	body := `{"source": "new-p"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/install", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rest.AdaptHandler(h.Install).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Install_NoSource(t *testing.T) {
	h, _ := setupHandlerTest(t)

	body := `{}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/install", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rest.AdaptHandler(h.Install).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_Install_InvalidBody(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/install", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rest.AdaptHandler(h.Install).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_Activate(t *testing.T) {
	h, m := setupHandlerTest(t)
	m.registry.Register(&PluginInstance{
		Manifest: &PluginManifest{ID: "test-p", Name: "Test", Version: "1.0.0"},
		Status:   PluginStatusInstalled,
	})

	body := `{"plugin_id": "test-p"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/activate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rest.AdaptHandler(h.Activate).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Activate_InvalidBody(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/activate", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rest.AdaptHandler(h.Activate).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_Deactivate_InvalidBody(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/deactivate", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rest.AdaptHandler(h.Deactivate).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_Activate_NoID(t *testing.T) {
	h, _ := setupHandlerTest(t)

	body := `{}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/activate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rest.AdaptHandler(h.Activate).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_Activate_NotFound(t *testing.T) {
	h, _ := setupHandlerTest(t)

	body := `{"plugin_id": "nonexistent"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/activate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rest.AdaptHandler(h.Activate).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Deactivate_NoID(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/deactivate", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rest.AdaptHandler(h.Deactivate).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Install_ManagerError(t *testing.T) {
	h, _ := setupHandlerTest(t)

	body := `{"source": "nonexistent"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/install", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rest.AdaptHandler(h.Install).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Deactivate_NotFound(t *testing.T) {
	h, _ := setupHandlerTest(t)

	body := `{"plugin_id": "nonexistent"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/deactivate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rest.AdaptHandler(h.Deactivate).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Update_NotFound(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/nonexistent/update", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Delete_NotFound(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/plugins/nonexistent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rest.AdaptHandler(h.Delete).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Update_NoID(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins//update", nil)
	rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Delete_NoID(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/plugins/", nil)
	rest.AdaptHandler(h.Delete).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Deactivate(t *testing.T) {
	h, m := setupHandlerTest(t)
	m.registry.Register(&PluginInstance{
		Manifest: &PluginManifest{ID: "test-p", Name: "Test", Version: "1.0.0"},
		Status:   PluginStatusActive,
	})

	body := `{"plugin_id": "test-p"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/deactivate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rest.AdaptHandler(h.Deactivate).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Update(t *testing.T) {
	h, m := setupHandlerTest(t)
	createTestPlugin(t, m.cfg.PluginsDir, "test-p", "1.0.0")
	m.Init(context.Background())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plugins/test-p/update", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "test-p")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rest.AdaptHandler(h.Update).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Delete(t *testing.T) {
	h, m := setupHandlerTest(t)
	m.registry.Register(&PluginInstance{
		Manifest: &PluginManifest{ID: "test-p", Name: "Test", Version: "1.0.0"},
		Status:   PluginStatusInstalled,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/plugins/test-p", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "test-p")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rest.AdaptHandler(h.Delete).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_GetSettings(t *testing.T) {
	h, m := setupHandlerTest(t)
	m.registry.Register(&PluginInstance{
		Manifest: &PluginManifest{ID: "test-p", Name: "Test", Version: "1.0.0"},
		Status:   PluginStatusInstalled,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plugins/test-p/settings", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "test-p")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rest.AdaptHandler(h.GetSettings).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_GetSettings_NotFound(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plugins/nonexistent/settings", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rest.AdaptHandler(h.GetSettings).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_UpdateSettings_InvalidBody(t *testing.T) {
	h, m := setupHandlerTest(t)
	m.registry.Register(&PluginInstance{
		Manifest: &PluginManifest{ID: "test-p", Name: "Test", Version: "1.0.0"},
		Status:   PluginStatusInstalled,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/plugins/test-p/settings", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "test-p")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rest.AdaptHandler(h.UpdateSettings).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_UpdateSettings_NotFound(t *testing.T) {
	h, _ := setupHandlerTest(t)

	body := `{"theme": "dark"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/plugins/nonexistent/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rest.AdaptHandler(h.UpdateSettings).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_UpdateSettings(t *testing.T) {
	h, m := setupHandlerTest(t)
	m.registry.Register(&PluginInstance{
		Manifest: &PluginManifest{ID: "test-p", Name: "Test", Version: "1.0.0"},
		Status:   PluginStatusInstalled,
	})

	body := `{"theme": "dark"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/plugins/test-p/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "test-p")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rest.AdaptHandler(h.UpdateSettings).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
