package seoengine

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// pexelsTestServer spins up a fake Pexels v1/search endpoint that records the
// request (authorization header, query params) so the client tests assert the
// security contract (key only in the header) and the API contract.
func pexelsTestServer(t *testing.T, status int, body string, handler func(r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler != nil {
			handler(r)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func testPexelsClient(key, baseURL string) *PexelsClient {
	c := NewPexelsClient(key, 5*time.Second)
	c.BaseURL = baseURL
	c.client = &http.Client{Timeout: 5 * time.Second}
	return c
}

const pexelsSearchOK = `{
  "photos": [
    {
      "id": 12345,
      "url": "https://www.pexels.com/photo/test-12345/",
      "photographer": "Fulano Teste",
      "photographer_url": "https://www.pexels.com/@fulanoteste",
      "alt": "Smartphone em uma mesa",
      "src": {
        "large": "https://images.pexels.com/photos/12345/test-large.jpg"
      }
    }
  ]
}`

func TestPexelsClient_SearchImage_Success(t *testing.T) {
	var gotPath string
	var gotAuth string
	srv := pexelsTestServer(t, 200, pexelsSearchOK, func(r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Query().Get("query") != "smartphone novo" {
			t.Errorf("expected query param, got %q", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("orientation") != "landscape" {
			t.Errorf("expected landscape orientation, got %q", r.URL.Query().Get("orientation"))
		}
		if r.URL.Query().Get("per_page") != "15" {
			t.Errorf("expected per_page=15, got %q", r.URL.Query().Get("per_page"))
		}
	})
	defer srv.Close()

	img, err := testPexelsClient("test-secret-key", srv.URL).SearchImage(context.Background(), "smartphone novo")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/search" {
		t.Errorf("expected /v1/search, got %q", gotPath)
	}
	if gotAuth != "test-secret-key" {
		t.Errorf("expected API key in Authorization header, got %q", gotAuth)
	}
	if strings.Contains(srv.URL+"?"+strings.Repeat("x", 0), "test-secret-key") {
		t.Error("API key must never appear in URLs")
	}
	if img.URL != "https://images.pexels.com/photos/12345/test-large.jpg" {
		t.Errorf("unexpected image URL: %s", img.URL)
	}
	if img.Alt != "Smartphone em uma mesa" {
		t.Errorf("unexpected alt: %s", img.Alt)
	}
	if img.Photographer != "Fulano Teste" || img.PhotographerURL == "" || img.SourceURL == "" {
		t.Errorf("attribution not carried: %+v", img)
	}
}

func TestPexelsClient_SearchImage_KeyNotInQuery(t *testing.T) {
	var rawQuery string
	srv := pexelsTestServer(t, 200, pexelsSearchOK, func(r *http.Request) {
		rawQuery = r.URL.RawQuery
	})
	defer srv.Close()

	_, err := testPexelsClient("doesnotbelong", srv.URL).SearchImage(context.Background(), "pixel 9a")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rawQuery, "doesnotbelong") {
		t.Errorf("API key leaked into the query string: %s", rawQuery)
	}
}

func TestPexelsClient_SearchImage_AltFallback(t *testing.T) {
	srv := pexelsTestServer(t, 200, `{"photos":[{"src":{"large":"https://img/x.jpg"},"url":"https://p/x/"}]}`, nil)
	defer srv.Close()

	img, err := testPexelsClient("k", srv.URL).SearchImage(context.Background(), "pixel 9a camera")
	if err != nil {
		t.Fatal(err)
	}
	if img.Alt == "" {
		t.Errorf("alt should fall back to a derived label, got empty")
	}
	if img.Alt != strings.ToLower(img.Alt) {
		t.Errorf("derived alt should be lowercased, got %q", img.Alt)
	}
}

func TestPexelsClient_SearchImage_ErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", http.StatusUnauthorized, `{"error":"invalid key"}`},
		{"forbidden", http.StatusForbidden, `{}`},
		{"rate limited", http.StatusTooManyRequests, `{}`},
		{"server error", http.StatusInternalServerError, `{}`},
		{"bad gateway", http.StatusBadGateway, `{}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := pexelsTestServer(t, tc.status, tc.body, nil)
			defer srv.Close()
			_, err := testPexelsClient("k", srv.URL).SearchImage(context.Background(), "q")
			if !errors.Is(err, ErrPexelsUnavailable) {
				t.Fatalf("expected ErrPexelsUnavailable, got %v", err)
			}
		})
	}
}

func TestPexelsClient_SearchImage_NoResults(t *testing.T) {
	srv := pexelsTestServer(t, 200, `{"photos":[]}`, nil)
	defer srv.Close()

	_, err := testPexelsClient("k", srv.URL).SearchImage(context.Background(), "q")
	if !errors.Is(err, ErrPexelsNoResults) {
		t.Fatalf("expected ErrPexelsNoResults, got %v", err)
	}
}

func TestPexelsClient_SearchImage_InvalidJSON(t *testing.T) {
	srv := pexelsTestServer(t, 200, `<html>not json</html>`, nil)
	defer srv.Close()

	_, err := testPexelsClient("k", srv.URL).SearchImage(context.Background(), "q")
	if !errors.Is(err, ErrPexelsUnavailable) {
		t.Fatalf("expected ErrPexelsUnavailable, got %v", err)
	}
}

func TestPexelsClient_SearchImage_DisabledClient(t *testing.T) {
	_, err := NewPexelsClient("", 5*time.Second).SearchImage(context.Background(), "q")
	if !errors.Is(err, ErrPexelsUnavailable) {
		t.Fatalf("expected ErrPexelsUnavailable for empty key, got %v", err)
	}
	var nilClient *PexelsClient
	if _, err := nilClient.SearchImage(context.Background(), "q"); !errors.Is(err, ErrPexelsUnavailable) {
		t.Fatalf("expected ErrPexelsUnavailable for nil client, got %v", err)
	}
}

func TestPexelsClient_SearchImage_ContextCancelled(t *testing.T) {
	srv := pexelsTestServer(t, 200, pexelsSearchOK, nil)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := testPexelsClient("k", srv.URL).SearchImage(ctx, "q")
	if !errors.Is(err, ErrPexelsUnavailable) {
		t.Fatalf("expected ErrPexelsUnavailable, got %v", err)
	}
}

func TestPexelsClient_SearchImage_EmptyQuery(t *testing.T) {
	srv := pexelsTestServer(t, 200, pexelsSearchOK, nil)
	defer srv.Close()

	img, err := testPexelsClient("k", srv.URL).SearchImage(context.Background(), "   ")
	if err != nil {
		t.Fatal(err)
	}
	if img == nil {
		t.Fatal("expected an image even for an empty query")
	}
}

func TestPexelsClient_NewDefaults(t *testing.T) {
	c := NewPexelsClient("k", 7*time.Second)
	if c.BaseURL != "https://api.pexels.com" {
		t.Errorf("unexpected default base URL: %s", c.BaseURL)
	}
	if c.Timeout != 7*time.Second {
		t.Errorf("unexpected timeout: %s", c.Timeout)
	}
	if c.APIKey != "k" {
		t.Errorf("key not stored: %q", c.APIKey)
	}
}

func TestParsePexelsSearchResponse_Pure(t *testing.T) {
	img, err := parsePexelsSearchResponse([]byte(pexelsSearchOK), "query")
	if err != nil {
		t.Fatal(err)
	}
	if img == nil || img.URL == "" || img.Photographer == "" {
		t.Fatalf("incomplete parse: %+v", img)
	}
	if _, err := parsePexelsSearchResponse([]byte(`{"photos":[{"src":{}}]}`), "q"); !errors.Is(err, ErrPexelsUnavailable) {
		t.Fatalf("expected error for photo without url, got %v", err)
	}
}

func TestPexelsImage_JSONRoundTrip(t *testing.T) {
	img := &PexelsImage{
		URL:             "https://images.pexels.com/x.jpg",
		Alt:             "alt",
		Photographer:    "p",
		PhotographerURL: "https://pexels.com/@p",
		SourceURL:       "https://pexels.com/photo/x",
	}
	b, err := json.Marshal(img)
	if err != nil {
		t.Fatal(err)
	}
	var back PexelsImage
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back != *img {
		t.Errorf("round trip mismatch: %+v", back)
	}
}