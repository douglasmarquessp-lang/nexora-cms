package revalidate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"nexora/internal/pkg/config"
	"nexora/internal/pkg/logger"
)

func testLog(t *testing.T) *logger.Logger {
	t.Helper()
	return logger.New(&config.Config{})
}

func TestDisabledWithoutConfig(t *testing.T) {
	c := New(nil, "", true, 0, testLog(t))
	if c.Enabled() {
		t.Error("client without urls/token should be disabled")
	}
	if err := c.Revalidate(context.Background(), "ola"); err != nil {
		t.Fatalf("expected nil error when disabled, got %v", err)
	}
}

func TestDisabledFlag(t *testing.T) {
	c := New([]string{"https://x.com"}, "tok", false, 0, testLog(t))
	if c.Enabled() {
		t.Error("expected client disabled via flag")
	}
}

func TestEmptySlugNoOp(t *testing.T) {
	c := New([]string{"https://x.com"}, "tok", true, 0, testLog(t))
	if err := c.Revalidate(context.Background(), ""); err != nil {
		t.Fatalf("expected nil for empty slug, got %v", err)
	}
}

func TestRevalidateSuccess(t *testing.T) {
	var mu sync.Mutex
	var gotHeader, gotBody, gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		gotHeader = r.Header.Get(TokenHeader)
		gotURL = r.URL.Path
		var payload map[string]string
		json.NewDecoder(r.Body).Decode(&payload)
		gotBody = payload["slug"]
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New([]string{srv.URL}, "segredo", true, 5*time.Second, testLog(t))
	if err := c.Revalidate(context.Background(), "meu-slug"); err != nil {
		t.Fatalf("revalidate failed: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if gotHeader != "segredo" {
		t.Errorf("expected token header %q, got %q", "segredo", gotHeader)
	}
	if gotBody != "meu-slug" {
		t.Errorf("expected slug %q, got %q", "meu-slug", gotBody)
	}
	if !strings.HasSuffix(gotURL, "/api/revalidate") {
		t.Errorf("unexpected path: %q", gotURL)
	}
}

func TestTrailingSlashNormalized(t *testing.T) {
	var mu sync.Mutex
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		path = r.URL.Path
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New([]string{srv.URL + "/"}, "tok", true, 0, testLog(t))
	if err := c.Revalidate(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if path != "/api/revalidate" {
		t.Errorf("expected normalized path, got %q", path)
	}
}

func TestRevalidatePartialFailIsFailOpen(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer good.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()

	c := New([]string{bad.URL, good.URL}, "tok", true, time.Second, testLog(t))
	if err := c.Revalidate(context.Background(), "x"); err != nil {
		t.Fatalf("expected fail-open with one success, got %v", err)
	}
}

func TestRevalidateAllFailReturnsError(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer bad.Close()

	c := New([]string{bad.URL}, "tok", true, time.Second, testLog(t))
	if err := c.Revalidate(context.Background(), "x"); err == nil {
		t.Fatal("expected error when the only site fails")
	}
}

func TestRevalidateNetworkErrorCollected(t *testing.T) {
	c := New([]string{"http://127.0.0.1:1"}, "tok", true, 200*time.Millisecond, testLog(t))
	err := c.Revalidate(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error when the only site is unreachable")
	}
	if !strings.Contains(err.Error(), "all sites failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRevalidateContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	c := New([]string{srv.URL}, "tok", true, 5*time.Second, testLog(t))
	_ = c.Revalidate(ctx, "x")
	if time.Since(start) > time.Second {
		t.Error("revalidate took too long after context cancelled")
	}
}

var _ = fmt.Sprintf
var _ = bytes.Compare