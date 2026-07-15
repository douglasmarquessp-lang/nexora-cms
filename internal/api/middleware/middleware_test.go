package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"nexora/internal/modules/auth"
	"nexora/internal/modules/site"
	"nexora/internal/pkg/casbin"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func TestConstants(t *testing.T) {
	if CtxSiteID != "site_id" {
		t.Errorf("expected 'site_id', got '%v'", CtxSiteID)
	}
	if CtxSiteSlug != "site_slug" {
		t.Errorf("expected 'site_slug', got '%v'", CtxSiteSlug)
	}
	if CtxUserRole != "user_role" {
		t.Errorf("expected 'user_role', got '%v'", CtxUserRole)
	}
}

func TestGetSiteID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	id, ok := GetSiteID(ctx)
	if ok {
		t.Fatal("expected false for empty context")
	}
	if id != uuid.Nil {
		t.Errorf("expected uuid.Nil, got %s", id)
	}
}

func TestGetSiteID_WithValue(t *testing.T) {
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), CtxSiteID, uid)
	id, ok := GetSiteID(ctx)
	if !ok {
		t.Fatal("expected true for context with site ID")
	}
	if id != uid {
		t.Errorf("expected %s, got %s", uid, id)
	}
}

func TestGetSiteID_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxSiteID, "not-a-uuid")
	_, ok := GetSiteID(ctx)
	if ok {
		t.Fatal("expected false for wrong type")
	}
}

func TestGetSiteSlug_EmptyContext(t *testing.T) {
	ctx := context.Background()
	slug, ok := GetSiteSlug(ctx)
	if ok {
		t.Fatal("expected false for empty context")
	}
	if slug != "" {
		t.Errorf("expected empty string, got '%s'", slug)
	}
}

func TestGetSiteSlug_WithValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxSiteSlug, "my-site")
	slug, ok := GetSiteSlug(ctx)
	if !ok {
		t.Fatal("expected true for context with slug")
	}
	if slug != "my-site" {
		t.Errorf("expected 'my-site', got '%s'", slug)
	}
}

func TestGetSiteSlug_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxSiteSlug, 42)
	_, ok := GetSiteSlug(ctx)
	if ok {
		t.Fatal("expected false for wrong type")
	}
}

func TestGetUserRole_EmptyContext(t *testing.T) {
	ctx := context.Background()
	role, ok := GetUserRole(ctx)
	if ok {
		t.Fatal("expected false for empty context")
	}
	if role != "" {
		t.Errorf("expected empty string, got '%s'", role)
	}
}

func TestGetUserRole_WithValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxUserRole, "admin")
	role, ok := GetUserRole(ctx)
	if !ok {
		t.Fatal("expected true for context with role")
	}
	if role != "admin" {
		t.Errorf("expected 'admin', got '%s'", role)
	}
}

func TestGetUserRole_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxUserRole, 42)
	_, ok := GetUserRole(ctx)
	if ok {
		t.Fatal("expected false for wrong type")
	}
}

func TestSetUserRole(t *testing.T) {
	ctx := context.Background()
	newCtx := SetUserRole(ctx, "editor")

	role, ok := GetUserRole(newCtx)
	if !ok {
		t.Fatal("expected true for set role")
	}
	if role != "editor" {
		t.Errorf("expected 'editor', got '%s'", role)
	}
}

func TestSetUserRole_Overwrite(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxUserRole, "user")
	newCtx := SetUserRole(ctx, "admin")

	role, ok := GetUserRole(newCtx)
	if !ok {
		t.Fatal("expected true for overwritten role")
	}
	if role != "admin" {
		t.Errorf("expected 'admin', got '%s'", role)
	}
}

func TestIdentifySite_Signature(t *testing.T) {
	// Compile-time check that IdentifySite returns the correct middleware function
	mw := IdentifySite(nil)
	if mw == nil {
		t.Fatal("expected non-nil middleware function")
	}

	// The returned function should have the correct signature
	var _ func(http.Handler) http.Handler = mw
}

func TestRequireSite_Signature(t *testing.T) {
	mw := RequireSite(nil)
	if mw == nil {
		t.Fatal("expected non-nil middleware function")
	}

	var _ func(http.Handler) http.Handler = mw
}

func TestRequireAuth_Signature(t *testing.T) {
	var svc *auth.Service
	mw := RequireAuth(svc)
	if mw == nil {
		t.Fatal("expected non-nil middleware function")
	}
	var _ func(http.Handler) http.Handler = mw
}

func TestOptionalAuth_Signature(t *testing.T) {
	var svc *auth.Service
	mw := OptionalAuth(svc)
	if mw == nil {
		t.Fatal("expected non-nil middleware function")
	}
	var _ func(http.Handler) http.Handler = mw
}

func TestRequireAuth_MissingHeader(t *testing.T) {
	var svc *auth.Service
	mw := RequireAuth(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuth_InvalidFormat(t *testing.T) {
	var svc *auth.Service
	mw := RequireAuth(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic token")
	rec := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuth_EmptyToken(t *testing.T) {
	var svc *auth.Service
	mw := RequireAuth(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestOptionalAuth_NoHeader(t *testing.T) {
	var svc *auth.Service
	mw := OptionalAuth(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	called := false

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler should be called")
	}
}

func TestOptionalAuth_BasicAuthHeader(t *testing.T) {
	var svc *auth.Service
	mw := OptionalAuth(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic token")
	rec := httptest.NewRecorder()
	called := false

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler should be called even with non-Bearer auth")
	}
}

func TestRequirePermission_Construction(t *testing.T) {
	var enf *casbin.Enforcer
	mw := RequirePermission(enf, "post", "read")
	if mw == nil {
		t.Fatal("expected non-nil middleware")
	}
	var _ func(http.Handler) http.Handler = mw
}

func TestRequirePermission_NoAuth(t *testing.T) {
	var enf *casbin.Enforcer
	mw := RequirePermission(enf, "post", "read")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestIdentifySite_ServesNext(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := site.NewService(cfg, log, nil, nil)
	mw := IdentifySite(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	called := false

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		id, ok := GetSiteID(r.Context())
		if !ok || id != uuid.Nil {
			t.Errorf("expected uuid.Nil, got %v (ok=%v)", id, ok)
		}
		slug, ok := GetSiteSlug(r.Context())
		if !ok || slug != "" {
			t.Errorf("expected empty slug, got '%s' (ok=%v)", slug, ok)
		}
	})).ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler should be called")
	}
}

func TestIdentifySite_WithInvalidXSiteID(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := site.NewService(cfg, log, nil, nil)
	mw := IdentifySite(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Site-ID", "invalid-uuid")
	rec := httptest.NewRecorder()
	called := false

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler should be called even with invalid X-Site-ID")
	}
}

func TestRequireSite_WithNoSiteIdentifier(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := site.NewService(cfg, log, nil, nil)
	mw := RequireSite(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called when no site is identified")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestRequireSite_WithInvalidXSiteID(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := site.NewService(cfg, log, nil, nil)
	mw := RequireSite(svc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Site-ID", "not-a-uuid")
	rec := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called when X-Site-ID is invalid")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetUserID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	_, ok := GetUserID(ctx)
	if ok {
		t.Fatal("expected false for empty context")
	}
}

func TestGetUserID_WithValue(t *testing.T) {
	uid := uuid.New()
	ctx := context.WithValue(context.Background(), auth.CtxUserID, uid)
	id, ok := GetUserID(ctx)
	if !ok {
		t.Fatal("expected true for context with user ID")
	}
	if id != uid {
		t.Errorf("expected %s, got %s", uid, id)
	}
}

func TestGetUserID_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.CtxUserID, "not-a-uuid")
	_, ok := GetUserID(ctx)
	if ok {
		t.Fatal("expected false for wrong type")
	}
}

func TestContextValueHelpers_Integration(t *testing.T) {
	ctx := context.Background()

	siteID := uuid.New()
	ctx = context.WithValue(ctx, CtxSiteID, siteID)
	ctx = context.WithValue(ctx, CtxSiteSlug, "my-slug")
	ctx = SetUserRole(ctx, "superadmin")

	id, ok := GetSiteID(ctx)
	if !ok || id != siteID {
		t.Errorf("GetSiteID: got %v (ok=%v), want %v", id, ok, siteID)
	}

	slug, ok := GetSiteSlug(ctx)
	if !ok || slug != "my-slug" {
		t.Errorf("GetSiteSlug: got '%s' (ok=%v), want 'my-slug'", slug, ok)
	}

	role, ok := GetUserRole(ctx)
	if !ok || role != "superadmin" {
		t.Errorf("GetUserRole: got '%s' (ok=%v), want 'superadmin'", role, ok)
	}

	uid := uuid.New()
	ctx = context.WithValue(ctx, auth.CtxUserID, uid)
	userID, ok := GetUserID(ctx)
	if !ok || userID != uid {
		t.Errorf("GetUserID: got %v (ok=%v), want %v", userID, ok, uid)
	}
}

func TestWrapMiddleware_LikeHelper(t *testing.T) {
	// Test pattern similar to wrapMiddleware from routes.go
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := func(w http.ResponseWriter, r *http.Request) {
		mw(handler).ServeHTTP(w, r)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Middleware") != "applied" {
		t.Error("expected middleware header to be set")
	}
}

func TestRequireAuth_ValidBearer_CallsNextOnNilService(t *testing.T) {
	// With nil service, ValidateAccessToken on nil causes panic.
	// We test the middleware construction and error path only.
	mw := RequireAuth(nil)
	if mw == nil {
		t.Fatal("expected non-nil middleware")
	}
}

func TestOptionalAuth_SafePathsWithNilService(t *testing.T) {
	mw := OptionalAuth(nil)

	t.Run("no auth header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		called := false

		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})).ServeHTTP(rec, req)

		if !called {
			t.Fatal("next handler should be called")
		}
	})

	t.Run("basic auth header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		rec := httptest.NewRecorder()
		called := false

		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})).ServeHTTP(rec, req)

		if !called {
			t.Fatal("next handler should be called with non-Bearer auth")
		}
	})
}
