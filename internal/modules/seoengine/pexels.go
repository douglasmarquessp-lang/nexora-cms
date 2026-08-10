package seoengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrPexelsUnavailable is returned when the Pexels API cannot be reached or
// rejects the request. It never carries the API key.
var ErrPexelsUnavailable = errors.New("pexels provider unavailable")

// ErrPexelsNoResults is returned when the Pexels search returned no photos.
var ErrPexelsNoResults = errors.New("pexels search returned no results")

// PexelsImage is the minimal photo payload the enhancer needs to enrich an
// article with a real photograph (src, honest alt text, attribution).
type PexelsImage struct {
	URL             string `json:"url"`
	Alt             string `json:"alt"`
	Photographer    string `json:"photographer"`
	PhotographerURL string `json:"photographer_url"`
	SourceURL       string `json:"source_url"`
}

// ImageProvider supplies a single landscape photograph for a search query.
// Implementations must be context-aware and fail-closed errors must be
// reported via ErrPexelsUnavailable so callers can fail open (publish without
// an image rather than block).
type ImageProvider interface {
	SearchImage(ctx context.Context, query string) (*PexelsImage, error)
}

// PexelsClient implements ImageProvider against the Pexels REST API
// (https://api.pexels.com/v1/search). The API key travels only in the
// Authorization header and is never part of URLs, logs, or errors.
type PexelsClient struct {
	APIKey  string
	BaseURL string
	Timeout time.Duration
	client  *http.Client
}

// NewPexelsClient builds a client for the given API key. An empty key yields
// a disabled client whose SearchImage always returns ErrPexelsUnavailable.
func NewPexelsClient(apiKey string, timeout time.Duration) *PexelsClient {
	base := "https://api.pexels.com"
	return &PexelsClient{
		APIKey:  apiKey,
		BaseURL: base,
		Timeout: timeout,
		client:  &http.Client{Timeout: timeout},
	}
}

// SearchImage performs a v1/search request for the given English query with
// orientation=landscape and returns the first photo. It never blocks long:
// the request is bounded by the client timeout and the caller's context.
func (c *PexelsClient) SearchImage(ctx context.Context, query string) (*PexelsImage, error) {
	if c == nil || c.client == nil || strings.TrimSpace(c.APIKey) == "" {
		return nil, ErrPexelsUnavailable
	}
	base := c.BaseURL
	if base == "" {
		base = "https://api.pexels.com"
	}
	u, err := url.Parse(base + "/v1/search")
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base url", ErrPexelsUnavailable)
	}
	q := u.Query()
	q.Set("query", strings.TrimSpace(query))
	q.Set("orientation", "landscape")
	q.Set("per_page", "1")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPexelsUnavailable, err)
	}
	// Key goes in the header only — never in the query string (URL logging
	// would leak it through proxies and server logs).
	req.Header.Set("Authorization", c.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: request failed: %v", ErrPexelsUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: invalid api key (status %d)", ErrPexelsUnavailable, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: rate limited (status %d)", ErrPexelsUnavailable, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("%w: unexpected status %d", ErrPexelsUnavailable, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read failed: %v", ErrPexelsUnavailable, err)
	}
	img, err := parsePexelsSearchResponse(body, query)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// parsePexelsSearchResponse extracts the first photo of a v1/search response.
// Pure function (unit-testable without HTTP).
func parsePexelsSearchResponse(body []byte, query string) (*PexelsImage, error) {
	var parsed struct {
		Photos []struct {
			Src struct {
				Large string `json:"large"`
			} `json:"src"`
			Alt             string `json:"alt"`
			Photographer    string `json:"photographer"`
			PhotographerURL string `json:"photographer_url"`
			URL             string `json:"url"`
		} `json:"photos"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%w: invalid search response", ErrPexelsUnavailable)
	}
	if len(parsed.Photos) == 0 {
		return nil, ErrPexelsNoResults
	}
	p := parsed.Photos[0]
	if p.Src.Large == "" {
		return nil, fmt.Errorf("%w: photo without image url", ErrPexelsUnavailable)
	}
	alt := strings.TrimSpace(p.Alt)
	if alt == "" {
		alt = deriveImageAlt(query)
	}
	return &PexelsImage{
		URL:             p.Src.Large,
		Alt:             alt,
		Photographer:    strings.TrimSpace(p.Photographer),
		PhotographerURL: strings.TrimSpace(p.PhotographerURL),
		SourceURL:       strings.TrimSpace(p.URL),
	}, nil
}