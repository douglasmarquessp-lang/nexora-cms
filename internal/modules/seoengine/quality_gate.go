package seoengine

import (
	"context"

	"nexora/internal/modules/publisher"
)

var _ publisher.QualityGate = (*Service)(nil)

// CheckQuality implements publisher.QualityGate for the publish funnel.
// It runs the pure, deterministic quality analysis (word count floor from
// SEO_MIN_WORD_COUNT, structure, substance, research grounding) against the
// 80-point minimum. It is DB-free and error-free by construction: the funnel
// fails open only if an implementation ever returns an error.
func (s *Service) CheckQuality(ctx context.Context, in publisher.QualityGateInput) (*publisher.QualityGateResult, error) {
	return publisher.CheckContentQuality(in, s.minWordCount, 3, s.minQualityScore), nil
}
