package setup

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexora/internal/api/rest"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

func setupHandlerTest(t *testing.T) (*Handler, *Service) {
	t.Helper()
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:     "test-secret-that-is-long-enough-for-hmac-sha256",
			JWTAccessTTL:  900000000000,
			JWTRefreshTTL: 604800000000000,
		},
	}
	log := logger.New(cfg)
	repo := NewRepository(nil)
	svc := NewService(cfg, log, repo)
	h := NewHandler(svc, log)
	return h, svc
}

func TestNewHandler(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret: "test-secret-that-is-long-enough-for-hmac-sha256",
		},
	}
	log := logger.New(cfg)
	repo := NewRepository(nil)
	svc := NewService(cfg, log, repo)
	h := NewHandler(svc, log)

	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.svc != svc {
		t.Error("handler service pointer mismatch")
	}
}

func TestHandler_Status(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("no db", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/setup/status", nil)
		rest.AdaptHandler(h.Status).ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_Install(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("invalid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/setup/install", strings.NewReader(`invalid`))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.Install).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"cms_name":"","admin_name":"","admin_email":"bad","password":"weak","site_name":"","language":"","timezone":""}`
		req := httptest.NewRequest(http.MethodPost, "/setup/install", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.Install).ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no db", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := `{"cms_name":"Nexora","admin_name":"Admin","admin_email":"admin@test.com","password":"Str0ng!Pass","site_name":"My Site","language":"pt-BR","timezone":"America/Sao_Paulo"}`
		req := httptest.NewRequest(http.MethodPost, "/setup/install", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rest.AdaptHandler(h.Install).ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestHandler_Config(t *testing.T) {
	h, _ := setupHandlerTest(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/setup/config", nil)
	rest.AdaptHandler(h.Config).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_Finish(t *testing.T) {
	h, _ := setupHandlerTest(t)

	t.Run("no db", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/setup/finish", nil)
		rest.AdaptHandler(h.Finish).ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestSetupService(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret: "test-secret-that-is-long-enough-for-hmac-sha256",
		},
	}
	log := logger.New(cfg)
	repo := NewRepository(&database.Database{Pool: nil})
	svc := NewService(cfg, log, repo)

	_ = svc
}
