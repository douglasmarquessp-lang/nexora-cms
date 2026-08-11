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

// SEOReviewReport is the full per-dimension breakdown of the publish gate
// evaluation. It powers the editorial review screen: the reviewer sees every
// weighted dimension, the passing threshold, and the blocking issues before
// approving or rejecting the article.
type SEOReviewReport struct {
	Score          float64  `json:"score"`
	MinScore       float64  `json:"min_score"`
	Passes         bool     `json:"passes"`
	Title          float64  `json:"title,omitempty"`
	Meta           float64  `json:"meta,omitempty"`
	Headings       float64  `json:"headings,omitempty"`
	Keyword        float64  `json:"keyword,omitempty"`
	Readability    float64  `json:"readability,omitempty"`
	InternalLinks  float64  `json:"internal_links,omitempty"`
	ExternalLinks  float64  `json:"external_links,omitempty"`
	EEAT           float64  `json:"eeat,omitempty"`
	Images         float64  `json:"images,omitempty"`
	KeywordDensity float64  `json:"keyword_density,omitempty"`
	WordCount      int      `json:"word_count,omitempty"`
	Issues         []string `json:"issues,omitempty"`
}

// PublishGateReviewer is the optional review-oriented variant of PublishGate:
// a full per-dimension SEOReviewReport for the given content, evaluated
// exactly like the publish gate would (same keyword derivation, same page
// rendering, same threshold). Implementations fail open.
type PublishGateReviewer interface {
	ReviewSEO(ctx context.Context, in PublishGateInput) (*SEOReviewReport, error)
}

// ErrSEOPublishBlocked is returned when generated content does not reach the
// configured minimum SEO score.
var ErrSEOPublishBlocked = errors.New("seo score below minimum for publishing")
