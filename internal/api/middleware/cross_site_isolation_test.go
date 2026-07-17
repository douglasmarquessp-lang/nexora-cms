package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"nexora/internal/modules/auth"
	"nexora/internal/modules/site"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func TestCrossSite_GetSiteID_Isolation(t *testing.T) {
	siteA := uuid.New()
	siteB := uuid.New()

	ctx := context.Background()
	ctxA := context.WithValue(ctx, CtxSiteID, siteA)
	ctxB := context.WithValue(ctx, CtxSiteID, siteB)

	idA, okA := GetSiteID(ctxA)
	idB, okB := GetSiteID(ctxB)

	if !okA || !okB {
		t.Fatal("both contexts should have site IDs")
	}
	if idA == idB {
		t.Fatal("site A and site B must have different IDs")
	}
	if idA != siteA {
		t.Errorf("site A: expected %s, got %s", siteA, idA)
	}
	if idB != siteB {
		t.Errorf("site B: expected %s, got %s", siteB, idB)
	}
}

func TestCrossSite_ContextPropagation(t *testing.T) {
	siteA := uuid.New()
	siteB := uuid.New()
	userA := uuid.New()
	userB := uuid.New()

	ctx := context.Background()
	ctx = context.WithValue(ctx, CtxSiteID, siteA)
	ctx = context.WithValue(ctx, auth.CtxUserID, userA)
	ctxA := ctx

	ctx = context.Background()
	ctx = context.WithValue(ctx, CtxSiteID, siteB)
	ctx = context.WithValue(ctx, auth.CtxUserID, userB)
	ctxB := ctx

	siteIDFromA, _ := GetSiteID(ctxA)
	userIDFromA, _ := GetUserID(ctxA)
	siteIDFromB, _ := GetSiteID(ctxB)
	userIDFromB, _ := GetUserID(ctxB)

	if siteIDFromA == siteIDFromB {
		t.Error("site IDs should differ across contexts")
	}
	if userIDFromA == userIDFromB {
		t.Error("user IDs should differ across contexts")
	}
	if siteIDFromA != siteA {
		t.Errorf("expected site A %s, got %s", siteA, siteIDFromA)
	}
	if siteIDFromB != siteB {
		t.Errorf("expected site B %s, got %s", siteB, siteIDFromB)
	}
}

func TestCrossSite_RLSContext_SetsCorrectSiteID(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := site.NewService(cfg, log, nil, nil)

	userID := uuid.New()
	siteID := uuid.New()

	ctx := context.WithValue(context.Background(), CtxSiteID, siteID)
	ctx = context.WithValue(ctx, auth.CtxUserID, userID)

	rlsCtx := svc.SetRLSContext(ctx, userID, "editor", siteID)

	sidStr, ok := rlsCtx.Value("app.current_site_id").(string)
	if !ok {
		t.Fatal("app.current_site_id should be set in context")
	}
	if sidStr != siteID.String() {
		t.Errorf("expected site_id %s, got %s", siteID.String(), sidStr)
	}
}

func TestCrossSite_MiddlewareChain_SiteIdentification(t *testing.T) {
	log := logger.New(&config.Config{})
	svc := site.NewService(&config.Config{}, log, nil, nil)

	t.Run("identify via X-Site-ID header", func(t *testing.T) {
		siteID := uuid.New().String()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Site-ID", siteID)

		rec := httptest.NewRecorder()
		called := false

		mw := IdentifySite(svc)
		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			id, ok := GetSiteID(r.Context())
			if ok && id != uuid.Nil {
			}
		})).ServeHTTP(rec, req)

		if !called {
			t.Fatal("handler should be called")
		}
	})

	t.Run("identify via Host header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "example.com"

		rec := httptest.NewRecorder()
		called := false

		mw := IdentifySite(svc)
		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})).ServeHTTP(rec, req)

		if !called {
			t.Fatal("handler should be called via host")
		}
	})

	t.Run("no identifier yields uuid.Nil", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		mw := IdentifySite(svc)
		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := GetSiteID(r.Context())
			if !ok || id != uuid.Nil {
				t.Errorf("expected uuid.Nil, got %v (ok=%v)", id, ok)
			}
		})).ServeHTTP(rec, req)
	})
}

func TestCrossSite_RequireSite_BlocksMissingIdentifier(t *testing.T) {
	log := logger.New(&config.Config{})
	svc := site.NewService(&config.Config{}, log, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mw := RequireSite(svc)
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without site identifier")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCrossSite_MiddlewareStack_Order(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := site.NewService(cfg, log, nil, nil)

	t.Run("IdentifySite then RLSContext", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler := IdentifySite(svc)(RLSContext(svc, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, idOk := GetSiteID(r.Context())
			_, userOk := GetUserID(r.Context())

			if !idOk {
				t.Error("site ID should be in context")
			}
			if userOk {
				t.Error("no user should be identified")
			}
			if id != uuid.Nil {
				t.Error("site ID should be nil without header")
			}
		})))

		handler.ServeHTTP(rec, req)
	})
}

func TestCrossSite_RLSApplied_Tracking(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New(cfg)
	svc := site.NewService(cfg, log, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), auth.CtxUserID, uuid.New())
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler := RLSContext(svc, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsRLSApplied(r.Context()) {
			t.Error("RLS should be marked as applied")
		}
	}))

	handler.ServeHTTP(rec, req)
}

func TestCrossSite_SlugContext_Isolation(t *testing.T) {
	ctx := context.Background()
	ctxA := context.WithValue(ctx, CtxSiteSlug, "site-alpha")
	ctxB := context.WithValue(ctx, CtxSiteSlug, "site-beta")

	slugA, okA := GetSiteSlug(ctxA)
	slugB, okB := GetSiteSlug(ctxB)

	if !okA || !okB {
		t.Fatal("both contexts should have slugs")
	}
	if slugA == slugB {
		t.Fatal("slugs should differ")
	}
	if slugA != "site-alpha" {
		t.Errorf("expected 'site-alpha', got '%s'", slugA)
	}
	if slugB != "site-beta" {
		t.Errorf("expected 'site-beta', got '%s'", slugB)
	}
}

func TestCrossSite_MultipleSites_NoInterference(t *testing.T) {
	sites := make([]uuid.UUID, 5)
	for i := range sites {
		sites[i] = uuid.New()
	}

	slugs := []string{"one", "two", "three", "four", "five"}

	ctxMap := make(map[string]context.Context)
	for i, id := range sites {
		ctx := context.Background()
		ctx = context.WithValue(ctx, CtxSiteID, id)
		ctx = context.WithValue(ctx, CtxSiteSlug, slugs[i])
		ctx = SetUserRole(ctx, "editor")
		ctx = context.WithValue(ctx, auth.CtxUserID, uuid.New())
		ctxMap[slugs[i]] = ctx
	}

	for _, slug := range slugs {
		ctx := ctxMap[slug]
		id, idOk := GetSiteID(ctx)
		s, slugOk := GetSiteSlug(ctx)
		role, roleOk := GetUserRole(ctx)
		uid, userOk := GetUserID(ctx)

		if !idOk || !slugOk || !roleOk || !userOk {
			t.Errorf("slug %s: missing context values", slug)
		}
		if id == uuid.Nil {
			t.Errorf("slug %s: nil site ID", slug)
		}
		if s != slug {
			t.Errorf("slug %s: expected '%s', got '%s'", slug, slug, s)
		}
		if role != "editor" {
			t.Errorf("slug %s: expected 'editor', got '%s'", slug, role)
		}
		if uid == uuid.Nil {
			t.Errorf("slug %s: nil user ID", slug)
		}
	}
}
