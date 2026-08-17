package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"nexora/internal/api/middleware"
	"nexora/internal/api/rest"
	"nexora/internal/pkg/sitedomain"
)

// fakeSiteResolver is a deterministic sitedomain.Resolver for API tests.
type fakeSiteResolver struct {
	sc  sitedomain.SiteContext
	err error
}

func (f *fakeSiteResolver) Resolve(ctx context.Context, siteID uuid.UUID) (sitedomain.SiteContext, error) {
	return f.sc, f.err
}

func schemaRequest(siteID uuid.UUID) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schema/site", nil)
	ctx := context.WithValue(req.Context(), middleware.CtxSiteID, siteID)
	return req.WithContext(ctx)
}

func TestSiteSchemaHandler_ENSite(t *testing.T) {
	h := newPublicSiteSchemaHandler(&Dependencies{
		SiteResolver: &fakeSiteResolver{sc: sitedomain.SiteContext{
			Domain:          "https://aiworksimple.com",
			PrimaryLanguage: "en",
		}},
	})

	rec := httptest.NewRecorder()
	rest.AdaptHandler(h.Get).ServeHTTP(rec, schemaRequest(uuid.New()))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "aiworksimple.com") {
		t.Errorf("expected resolved domain in schema, got: %s", body)
	}
	if strings.Contains(body, "example.com") {
		t.Errorf("must not use example.com when a domain is resolved: %s", body)
	}
	if !strings.Contains(body, `search?q={search_term_string}`) {
		t.Errorf("expected English search route /search, got: %s", body)
	}
	if !strings.Contains(body, `inLanguage\":\"en`) {
		t.Errorf("expected en inLanguage, got: %s", body)
	}
}

func TestSiteSchemaHandler_PTSite(t *testing.T) {
	h := newPublicSiteSchemaHandler(&Dependencies{
		SiteResolver: &fakeSiteResolver{sc: sitedomain.SiteContext{
			Domain:          "https://dominio-pt.com",
			PrimaryLanguage: "pt",
		}},
	})

	rec := httptest.NewRecorder()
	rest.AdaptHandler(h.Get).ServeHTTP(rec, schemaRequest(uuid.New()))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "dominio-pt.com") {
		t.Errorf("expected resolved PT domain in schema, got: %s", body)
	}
	if !strings.Contains(body, `busca?q={search_term_string}`) {
		t.Errorf("expected Portuguese search route /busca, got: %s", body)
	}
	if !strings.Contains(body, `inLanguage\":\"pt`) {
		t.Errorf("expected pt inLanguage, got: %s", body)
	}
}

func TestSiteSchemaHandler_NoResolver_ExplicitError(t *testing.T) {
	// No resolver → the schema endpoint must fail explicitly instead of
	// serving schemas with a placeholder domain.
	h := newPublicSiteSchemaHandler(&Dependencies{})

	rec := httptest.NewRecorder()
	rest.AdaptHandler(h.Get).ServeHTTP(rec, schemaRequest(uuid.New()))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without resolver, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "example.com") {
		t.Errorf("placeholder domain must never appear: %s", rec.Body.String())
	}
}

func TestSiteSchemaHandler_ResolverError_ExplicitError(t *testing.T) {
	// Resolver failure (degraded DB) → explicit 422, never a fake domain.
	h := newPublicSiteSchemaHandler(&Dependencies{
		SiteResolver: &fakeSiteResolver{err: context.DeadlineExceeded},
	})

	rec := httptest.NewRecorder()
	rest.AdaptHandler(h.Get).ServeHTTP(rec, schemaRequest(uuid.New()))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on resolver error, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "DOMAIN_UNRESOLVED") {
		t.Errorf("expected DOMAIN_UNRESOLVED, got: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "example.com") {
		t.Errorf("placeholder domain must never appear: %s", rec.Body.String())
	}
}

func TestSiteSchemaHandler_NoVerifiedDomain_ExplicitError(t *testing.T) {
	// Resolver succeeds but the site has no verified domain → explicit 422.
	h := newPublicSiteSchemaHandler(&Dependencies{
		SiteResolver: &fakeSiteResolver{sc: sitedomain.SiteContext{
			Domain:          "",
			PrimaryLanguage: "pt",
		}},
	})

	rec := httptest.NewRecorder()
	rest.AdaptHandler(h.Get).ServeHTTP(rec, schemaRequest(uuid.New()))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 without verified domain, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "example.com") {
		t.Errorf("placeholder domain must never appear: %s", rec.Body.String())
	}
}

func TestSiteSchemaHandler_PlaceholderDomain_ExplicitError(t *testing.T) {
	// A resolver returning example.com is treated as unresolvable.
	h := newPublicSiteSchemaHandler(&Dependencies{
		SiteResolver: &fakeSiteResolver{sc: sitedomain.SiteContext{
			Domain:          "https://example.com",
			PrimaryLanguage: "pt",
		}},
	})

	rec := httptest.NewRecorder()
	rest.AdaptHandler(h.Get).ServeHTTP(rec, schemaRequest(uuid.New()))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for placeholder domain, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSiteSchemaHandler_MissingSite(t *testing.T) {
	h := newPublicSiteSchemaHandler(&Dependencies{})

	rec := httptest.NewRecorder()
	rest.AdaptHandler(h.Get).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/schema/site", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 without site context, got %d", rec.Code)
	}
}

func TestSearchURL(t *testing.T) {
	if got := searchURL("en"); got != "search?q={search_term_string}" {
		t.Errorf("searchURL(en) = %q", got)
	}
	if got := searchURL("pt"); got != "busca?q={search_term_string}" {
		t.Errorf("searchURL(pt) = %q", got)
	}
	if got := searchURL("EN"); got != "search?q={search_term_string}" {
		t.Errorf("searchURL(EN) = %q", got)
	}
	if got := searchURL(""); got != "busca?q={search_term_string}" {
		t.Errorf("searchURL(\"\") = %q", got)
	}
}
