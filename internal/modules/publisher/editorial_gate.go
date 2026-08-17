package publisher

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// EditorialGateInput carries the information the editorial gate needs to
// evaluate whether content may be published.
type EditorialGateInput struct {
	SiteID   uuid.UUID
	PostID   *uuid.UUID
	Title    string
	Content  string
	Language string
}

// EditorialGate is implemented by the editorial brain. It returns the final
// editorial note (0-100) of the most recent review of the same content. The
// publisher compares it against the configured minimum. Implementations must
// fail open: never block publication on missing reviews or infra hiccups.
type EditorialGate interface {
	CheckEditorialScore(ctx context.Context, in EditorialGateInput) (float64, error)
}

// EditorialReviewer is the optional extension of EditorialGate that can
// produce the editorial note itself (deterministically) when no review
// exists yet. The auto-publish path uses it so generated content is always
// evaluated against a real note — never a fabricated 100. When the review
// cannot be produced, auto-publish blocks with ErrEditorialReviewUnavailable.
type EditorialReviewer interface {
	EditorialGate
	// ReviewForGate runs a full editorial review of the exact content being
	// published and returns its final note. It must be deterministic,
	// DB-backed and never return a score without running the real analysis.
	ReviewForGate(ctx context.Context, in EditorialGateInput) (float64, error)
}

// ErrEditorialScoreBelowMinimum is returned when content's final editorial
// note is below the configured minimum: the article returns to review
// instead of being published.
var ErrEditorialScoreBelowMinimum = errors.New("editorial score below minimum for publishing")

// ErrNoEditorialReview is returned by CheckEditorialScore when no review
// exists for the content being evaluated. Callers decide the disposition:
// manual publishes fail open, auto-publishes generate one via
// EditorialReviewer or block.
var ErrNoEditorialReview = errors.New("no editorial review exists for content")

// ErrEditorialReviewUnavailable is returned by the auto-publish path when the
// editorial evaluation cannot be produced (no review exists and the
// evaluation itself fails). The generated content must go to the manual
// review screen instead of the front page — never published without a real
// note.
var ErrEditorialReviewUnavailable = errors.New("editorial review unavailable for auto-publish")
