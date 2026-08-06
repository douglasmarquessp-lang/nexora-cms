// Package revalidate triggers Next.js ISR revalidation on the public site(s)
// whenever content changes. Fail-open by design: network errors, bad tokens
// and timeouts are logged and swallowed — a broken revalidation must never
// block the publish flow.
package revalidate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"nexora/internal/pkg/logger"
)

const TokenHeader = "x-revalidate-token"

// Client POSTs {"slug": "..."} to {site}/api/revalidate for every configured
// public URL. With no URLs or no token it is a no-op (logged once at build).
type Client struct {
	publicURLs []string
	token      string
	enabled    bool
	httpClient *http.Client
	log        *logger.Logger
}

func New(publicURLs []string, token string, enabled bool, timeout time.Duration, log *logger.Logger) *Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	c := &Client{
		publicURLs: publicURLs,
		token:      token,
		enabled:    enabled,
		httpClient: &http.Client{Timeout: timeout},
		log:        log,
	}
	if log != nil {
		if len(publicURLs) == 0 || token == "" {
			log.Warn("ISR revalidation configured but incomplete (missing SITE_PUBLIC_URLS or SITE_REVALIDATE_TOKEN); publishing will not revalidate the public site")
		}
	}
	return c
}

func (c *Client) Enabled() bool {
	return c.enabled && len(c.publicURLs) > 0 && c.token != ""
}

// Revalidate asks every public site to revalidate the given slug (and the
// homepage). It returns an error only when every site failed; individual
// failures are logged and skipped.
func (c *Client) Revalidate(ctx context.Context, slug string) error {
	if !c.Enabled() {
		return nil
	}
	if slug == "" {
		return nil
	}

	body, err := json.Marshal(map[string]string{"slug": slug})
	if err != nil {
		return fmt.Errorf("revalidate: marshal body: %w", err)
	}

	var errs []string
	for _, base := range c.publicURLs {
		url := strings.TrimRight(base, "/") + "/api/revalidate"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(TokenHeader, c.token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", url, err))
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			c.log.Debug("ISR revalidation triggered", "url", url, "slug", slug)
			continue
		}
		errs = append(errs, fmt.Sprintf("%s: HTTP %d", url, resp.StatusCode))
	}
	if len(errs) == len(c.publicURLs) && len(errs) > 0 {
		return fmt.Errorf("revalidate: all sites failed: %s", strings.Join(errs, "; "))
	}
	return nil
}
