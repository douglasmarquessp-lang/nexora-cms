package webui

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

//go:embed all:testdata/dist
var testDist embed.FS

func newTestHandler() http.Handler {
	return NewSPAHandler(mustSub(testDist, "testdata/dist"))
}

func do(req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	newTestHandler().ServeHTTP(rec, req)
	return rec
}

func TestSPARootServesIndex(t *testing.T) {
	rec := do(httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<title>Test Admin</title>") {
		t.Fatalf("GET / body does not contain index.html: %q", rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("GET / Content-Type = %q, want text/html", ct)
	}
}

func TestSPADeepPathFallsBackToIndex(t *testing.T) {
	for _, p := range []string{"/admin", "/admin/login", "/admin/dashboard", "/some/unknown/deep/path"} {
		rec := do(httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", p, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "Test Admin") {
			t.Fatalf("GET %s did not fall back to index.html: %q", p, rec.Body.String())
		}
	}
}

func TestSPAAssetServedWithCacheHeaders(t *testing.T) {
	rec := do(httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /assets/app.js = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "test asset") {
		t.Fatalf("GET /assets/app.js body = %q", body)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Fatalf("asset Cache-Control = %q", cc)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/javascript") {
		t.Fatalf("asset Content-Type = %q, want JS", ct)
	}
}

func TestSPAApiPathNeverFallsBackToIndex(t *testing.T) {
	for _, p := range []string{"/api/v1/unknown", "/api/whatever"} {
		rec := do(httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("GET %s = %d, want 404", p, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "Test Admin") {
			t.Fatalf("GET %s leaked index.html into the API 404", p)
		}
		if !strings.Contains(rec.Body.String(), `"code":"NOT_FOUND"`) {
			t.Fatalf("GET %s body = %q, want JSON NOT_FOUND error", p, rec.Body.String())
		}
	}
}

func TestSPAMethodNotAllowed(t *testing.T) {
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rec := do(httptest.NewRequest(m, "/admin", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s /admin = %d, want 405", m, rec.Code)
		}
	}
}

func TestSPAHeadReturnsHeadersOnly(t *testing.T) {
	rec := do(httptest.NewRequest(http.MethodHead, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("HEAD / = %d, want 200", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD / returned a body (%d bytes)", rec.Body.Len())
	}
}

func TestSPAEmptyEmbedReturns404(t *testing.T) {
	// Simulate the local/CI build where the embedded FS has no index.html.
	empty := NewSPAHandler(embed.FS{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	empty.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET / on empty FS = %d, want 404", rec.Code)
	}
}
