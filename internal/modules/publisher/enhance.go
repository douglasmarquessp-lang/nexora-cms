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
	// FeaturedImageURL is the already-known featured image of the article
	// (e.g. from the generation pipeline). The enhancer embeds it in the
	// body when set; otherwise it may fetch one via its image provider.
	FeaturedImageURL string
	// FeaturedImageAlt is the honest alt text of the featured image.
	FeaturedImageAlt string
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
	// FeaturedImageURL/Alt carry the image the enhancer embedded (or the one
	// it was given). The publisher persists them onto the publication
	// (featured_image_url + og_image) so the public page and social cards are
	// consistent with the stored article.
	FeaturedImageURL string
	FeaturedImageAlt string
}

// ContentEnhancer is implemented by the SEO engine. It may add internal links,
// verify external link reliability, run the content gap and topic authority
// analyses, and return the enhanced content. Implementations must fail open:
// return the content unchanged (with a nil error) when enhancement is
// impossible, never block publishing on infra hiccups.
type ContentEnhancer interface {
	EnhanceBeforePublish(ctx context.Context, in ContentEnhancerInput) (*ContentEnhancement, error)
}
