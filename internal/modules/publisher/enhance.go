package publisher

import (
	"context"

	"github.com/google/uuid"
)

// ContentEnhancerInput carries everything a pre-publish enhancer needs to
// enrich generated content before it is published.
type ContentEnhancerInput struct {
	SiteID   uuid.UUID
	PostID   *uuid.UUID
	Title    string
	Content  string
	Keyword  string
	Category string
	Language string
}

// ContentEnhancement is what the enhancer returns: the (possibly modified)
// content plus the intelligence artifacts gathered along the way.
type ContentEnhancement struct {
	Content        string        // content with an appended related-links section (when internal links were added)
	MetaDescription string       // deterministic meta description derived from the content ("" when not derivable)
	InternalLinks  []interface{} // []seoengine.InternalLinkCandidate, serialized to keep the interface decoupled
	ExternalLinks  []interface{} // []seoengine.ExternalLinkCandidate
	GapReport      interface{}   // *seoengine.ContentGapReport
	TopicAuthority interface{}   // *seoengine.TopicalAuthorityReport
	Suggestions    []string      // human-readable PT/EN suggestions
}

// ContentEnhancer is implemented by the SEO engine. It may add internal links,
// verify external link reliability, run the content gap and topic authority
// analyses, and return the enhanced content. Implementations must fail open:
// return the content unchanged (with a nil error) when enhancement is
// impossible, never block publishing on infra hiccups.
type ContentEnhancer interface {
	EnhanceBeforePublish(ctx context.Context, in ContentEnhancerInput) (*ContentEnhancement, error)
}
