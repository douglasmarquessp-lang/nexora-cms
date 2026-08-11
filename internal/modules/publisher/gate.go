package publisher

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// PublishGateInput carries the information a publish gate needs to evaluate
// whether content may be published.
type PublishGateInput struct {
	SiteID          uuid.UUID
	PostID          *uuid.UUID
	Title           string
	Content         string
	Language        string
	MetaDescription string
	// Keyword (optional) is the focus keyword from the generation pipeline.
	// Empty → the gate derives it deterministically from title+content.
	Keyword string
	// AuthorName (optional) feeds the EEAT analysis. Empty → the gate uses
	// the site's configured default author when set.
	AuthorName string
}

// PublishGate is implemented by the SEO engine. It returns the content's SEO
// score (0-100). The publisher compares it against the configured minimum.
// Implementations must fail open: return an error only when evaluation is
// impossible, never block on infra hiccups.
type PublishGate interface {
	CheckPublishScore(ctx context.Context, in PublishGateInput) (float64, error)
}

// PublishGateDetailer is the optional richer variant of PublishGate. When the
// gate implements it, blocked publications carry the concrete audit issues
// (e.g. "images: 30/100", "keyword: 40/100") so the caller can show exactly
// what kept the article below the minimum.
type PublishGateDetailer interface {
	CheckPublishScoreWithIssues(ctx context.Context, in PublishGateInput) (float64, []string, error)
}

// ErrSEOPublishBlocked is returned when generated content does not reach the
// configured minimum SEO score.
var ErrSEOPublishBlocked = errors.New("seo score below minimum for publishing")
