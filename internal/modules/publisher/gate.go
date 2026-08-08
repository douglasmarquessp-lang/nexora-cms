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
}

// PublishGate is implemented by the SEO engine. It returns the content's SEO
// score (0-100). The publisher compares it against the configured minimum.
// Implementations must fail open: return an error only when evaluation is
// impossible, never block on infra hiccups.
type PublishGate interface {
	CheckPublishScore(ctx context.Context, in PublishGateInput) (float64, error)
}

// ErrSEOPublishBlocked is returned when generated content does not reach the
// configured minimum SEO score.
var ErrSEOPublishBlocked = errors.New("seo score below minimum for publishing")
