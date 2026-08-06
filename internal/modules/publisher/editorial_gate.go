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

// ErrEditorialScoreBelowMinimum is returned when content's final editorial
// note is below the configured minimum: the article returns to review
// instead of being published.
var ErrEditorialScoreBelowMinimum = errors.New("editorial score below minimum for publishing")
