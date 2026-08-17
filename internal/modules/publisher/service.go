package publisher

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
	"nexora/internal/pkg/audit"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
	"nexora/internal/pkg/sitedomain"
	"nexora/internal/pkg/sitelang"
)

type Service struct {
	log               *logger.Logger
	db                *database.Database
	cache             *cache.Cache
	repo              *Repository
	val               *Validator
	eventBus          *kernel.EventBus
	auditLog          *audit.Logger
	siteResolver      sitedomain.Resolver
	publishGate       PublishGate
	contentEnhancer   ContentEnhancer
	editorialGate     EditorialGate
	qualityGate       QualityGate
	minPublishScore   float64
	minEditorialScore float64
	defaultAuthor     string
}

func NewService(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *Service {
	var pool database.Pool
	if db != nil {
		pool = db.Pool
	}
	return &Service{
		log:               log,
		db:                db,
		cache:             ch,
		repo:              NewRepository(pool),
		val:               NewValidator(),
		auditLog:          audit.New(pool, log),
		publishGate:       nil,
		minPublishScore:   cfg.SEO.MinPublishScore,
		minEditorialScore: cfg.Editorial.MinFinalScore,
		defaultAuthor:     cfg.SEO.DefaultAuthor,
	}
}

func (s *Service) SetEventBus(bus *kernel.EventBus) {
	s.eventBus = bus
}

// SetSiteResolver registers the per-site context resolver (domain + primary
// language). Publication URLs always come from the site's own configuration;
// when no resolver is registered or the site has no verified domain, calls
// fail explicitly instead of falling back to a placeholder domain.
func (s *Service) SetSiteResolver(r sitedomain.Resolver) {
	s.siteResolver = r
}

// resolveSiteContext resolves the public domain and primary language for a
// site. It returns an explicit error when no resolver is registered or the
// site has no verified domain, so callers never publish with a placeholder
// domain.
func (s *Service) resolveSiteContext(ctx context.Context, siteID uuid.UUID) (sitedomain.SiteContext, error) {
	if s.siteResolver == nil {
		return sitedomain.SiteContext{}, ErrDomainUnresolved
	}
	sc, err := s.siteResolver.Resolve(ctx, siteID)
	if err != nil {
		if errors.Is(err, sitedomain.ErrNoVerifiedDomain) {
			return sitedomain.SiteContext{}, ErrNoVerifiedDomain
		}
		s.log.Warn("site resolver failed, refusing to publish without a resolved domain",
			"site_id", siteID.String(), "error", err)
		return sitedomain.SiteContext{}, ErrDomainUnresolved
	}
	if sc.Domain == "" {
		return sitedomain.SiteContext{}, ErrNoVerifiedDomain
	}
	return sc, nil
}

// effectiveLanguage returns the publish language for a site: an explicit
// request is honored (subject to the caller's validation), an empty request
// falls back to the site's primary language, and the site-level pin
// (sitelang) keeps its existing precedence as the top layer.
func (s *Service) effectiveLanguage(ctx context.Context, siteID uuid.UUID, requested string) (string, error) {
	lang := strings.ToLower(requested)
	if lang == "" {
		sc, err := s.resolveSiteContext(ctx, siteID)
		if err != nil {
			return "", err
		}
		lang = sc.PrimaryLanguage
	}
	return sitelang.Resolve(siteID, lang), nil
}

// SetPublishGate registers the SEO publish gate. When set and the configured
// minimum score is > 0, generated content below the threshold is blocked.
func (s *Service) SetPublishGate(g PublishGate) {
	s.publishGate = g
}

// SetContentEnhancer registers the pre-publish content enhancer (SEO engine).
// The enhancer may add internal/external links and gap fillers; it always
// fails open.
func (s *Service) SetContentEnhancer(e ContentEnhancer) {
	s.contentEnhancer = e
}

// SetEditorialGate registers the editorial brain publish gate. When set and
// the configured minimum editorial score is > 0, content whose latest review
// is below the threshold is sent back to review instead of published.
func (s *Service) SetEditorialGate(g EditorialGate) {
	s.editorialGate = g
}

// SetQualityGate registers the publish quality gate (depth/structure/substance
// analysis) applied to auto-generated content before the SEO gate. The gate
// never blocks on its own evaluation errors — only on a below-threshold
// verdict.
func (s *Service) SetQualityGate(g QualityGate) {
	s.qualityGate = g
}

// checkEditorialGate enforces the editorial note threshold.
//
// Manual publishes (autoPublish=false) fail open: an unavailable evaluation
// (missing review or infra error) never blocks a human-initiated publish —
// no fabricated score is produced, the gate is simply skipped.
//
// Auto-publishes (autoPublish=true) never publish without a real note: a
// missing review triggers a deterministic ReviewForGate; if the note cannot
// be produced the publish blocks with ErrEditorialReviewUnavailable so the
// article goes to the manual review screen instead of the front page.
func (s *Service) checkEditorialGate(ctx context.Context, siteID uuid.UUID, postID *uuid.UUID, title, content, lang string, autoPublish bool) error {
	if s.editorialGate == nil || s.minEditorialScore <= 0 {
		return nil
	}
	if strings.TrimSpace(title) == "" && strings.TrimSpace(content) == "" {
		return nil
	}
	score, err := s.editorialGate.CheckEditorialScore(ctx, EditorialGateInput{
		SiteID:   siteID,
		PostID:   postID,
		Title:    title,
		Content:  content,
		Language: lang,
	})
	if err != nil {
		if errors.Is(err, ErrNoEditorialReview) {
			if !autoPublish {
				s.log.Info("no editorial review for manual publish, skipping gate transparently")
				return nil
			}
			// Auto-publish: generate the note right here (deterministic,
			// real review of the exact content being published) instead of
			// fabricating one. If the note cannot be produced, block: the
			// article must go through the manual review screen.
			if reviewer, ok := s.editorialGate.(EditorialReviewer); ok {
				score, err = reviewer.ReviewForGate(ctx, EditorialGateInput{
					SiteID:   siteID,
					PostID:   postID,
					Title:    title,
					Content:  content,
					Language: lang,
				})
				if err != nil {
					return fmt.Errorf("%w: %v", ErrEditorialReviewUnavailable, err)
				}
			} else {
				return fmt.Errorf("%w: gate cannot produce an editorial note", ErrEditorialReviewUnavailable)
			}
		} else {
			if autoPublish {
				return fmt.Errorf("%w: %v", ErrEditorialReviewUnavailable, err)
			}
			s.log.Warn("editorial gate evaluation failed, allowing manual publish", "error", err)
			return nil
		}
	}
	if score < s.minEditorialScore {
		return fmt.Errorf("%w: editorial score %.2f below minimum %.2f", ErrEditorialScoreBelowMinimum, score, s.minEditorialScore)
	}
	return nil
}

// checkQualityGate enforces the publish quality gate for auto-generated
// content. It is fail-open on evaluation errors (never on the verdict): a
// real below-threshold result blocks with the full breakdown.
func (s *Service) checkQualityGate(ctx context.Context, in QualityGateInput) error {
	if s.qualityGate == nil {
		return nil
	}
	if strings.TrimSpace(in.Title) == "" && strings.TrimSpace(in.Content) == "" {
		return nil
	}
	res, err := s.qualityGate.CheckQuality(ctx, in)
	if err != nil {
		s.log.Warn("quality gate evaluation failed, allowing publish", "error", err)
		return nil
	}
	if res == nil {
		return nil
	}
	if !res.Passed {
		detail := ""
		if len(res.Issues) > 0 {
			msgs := make([]string, 0, len(res.Issues))
			for _, i := range res.Issues {
				msgs = append(msgs, i.Field+": "+i.Message)
			}
			detail = ": " + strings.Join(msgs, "; ")
		}
		return fmt.Errorf("%w: score %.2f below minimum %.2f (words %d, h2 %d)%s",
			ErrQualityGateBlocked, res.Score, res.MinScore, res.WordCount, res.H2Count, detail)
	}
	return nil
}

// enhanceContent runs the registered content enhancer and returns the possibly
// enhanced content plus the generated meta description ("" when the enhancer
// produced none). It also returns the enhancement object itself so callers can
// persist its artifacts (featured image, internal/external links). Fails open:
// on any error the content is returned unchanged and enh is nil.
func (s *Service) enhanceContent(ctx context.Context, siteID uuid.UUID, postID *uuid.UUID, title, content, keyword, category, lang string, featuredURL, featuredAlt string, featuredCredit *ImageCredit) (string, string, *ContentEnhancement) {
	if s.contentEnhancer == nil || strings.TrimSpace(content) == "" {
		return content, "", nil
	}
	out, err := s.contentEnhancer.EnhanceBeforePublish(ctx, ContentEnhancerInput{
		SiteID:              siteID,
		PostID:              postID,
		Title:               title,
		Content:             content,
		Keyword:             keyword,
		Category:            category,
		Language:            lang,
		FeaturedImageURL:    featuredURL,
		FeaturedImageAlt:    featuredAlt,
		FeaturedImageCredit: featuredCredit,
	})
	if err != nil {
		s.log.Warn("content enhancement failed, publishing original", "error", err)
		return content, "", nil
	}
	if out == nil || strings.TrimSpace(out.Content) == "" {
		return content, "", nil
	}
	return out.Content, out.MetaDescription, out
}

func (s *Service) checkPublishGate(ctx context.Context, siteID uuid.UUID, postID *uuid.UUID, title, content, lang, metaDescription, keyword, authorName string) error {
	if s.publishGate == nil || s.minPublishScore <= 0 {
		return nil
	}
	if strings.TrimSpace(title) == "" && strings.TrimSpace(content) == "" {
		return nil
	}
	in := PublishGateInput{
		SiteID:          siteID,
		PostID:          postID,
		Title:           title,
		Content:         content,
		Language:        lang,
		MetaDescription: metaDescription,
		Keyword:         keyword,
		AuthorName:      authorName,
	}
	score, err := s.publishGate.CheckPublishScore(ctx, in)
	if err != nil {
		s.log.Warn("publish gate evaluation failed, allowing publish", "error", err)
		return nil
	}
	if score < s.minPublishScore {
		// When the gate can explain itself, surface the concrete audit
		// issues in the error so operators see exactly what to fix
		// (e.g. "images: 30/100; internal_links: 0/100").
		if detailer, ok := s.publishGate.(PublishGateDetailer); ok {
			if dScore, issues, derr := detailer.CheckPublishScoreWithIssues(ctx, in); derr == nil && dScore == score {
				msg := fmt.Sprintf("seo score %.2f below minimum %.2f", score, s.minPublishScore)
				if len(issues) > 0 {
					msg += ": " + strings.Join(issues, "; ")
				}
				return fmt.Errorf("%w: %s", ErrSEOPublishBlocked, msg)
			}
		}
		return fmt.Errorf("%w: seo score %.2f below minimum %.2f", ErrSEOPublishBlocked, score, s.minPublishScore)
	}
	return nil
}

func (s *Service) fireEvent(ctx context.Context, eventType kernel.EventType, payload interface{}, siteID uuid.UUID) {
	if s.eventBus != nil {
		s.eventBus.EmitAsync(ctx, eventType, payload, siteID.String())
	}
}

func (s *Service) pool() (database.Pool, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, ErrDatabaseNotAvail
	}
	return s.db.Pool, nil
}

// --- Publish ---

func (s *Service) PublishArticle(ctx context.Context, siteID, userID uuid.UUID, req PublishRequest) (*PublishResponse, error) {
	return s.publishArticleInternal(ctx, siteID, userID, req, false)
}

// publishArticleInternal is the shared publish funnel. skipEditorialGate is
// true only for content a human approved on a review screen: the approval is
// the editorial decision, so the gate is skipped transparently (no
// fabricated note). SEO gate and quality gate always apply.
func (s *Service) publishArticleInternal(ctx context.Context, siteID, userID uuid.UUID, req PublishRequest, skipEditorialGate bool) (*PublishResponse, error) {
	if req.Title == "" && req.PostID != nil {
		rec, err := s.repo.GetPostForPublish(ctx, siteID, *req.PostID)
		if err != nil {
			return nil, err
		}
		req.Title = rec.Title
		if req.Content == "" {
			req.Content = rec.Content
		}
		if req.Excerpt == "" {
			req.Excerpt = rec.Excerpt
		}
		if req.Slug == "" && rec.Slug != "" {
			req.Slug = rec.Slug
		}
		if req.Language == "" && rec.Language != "" {
			req.Language = rec.Language
		}
		for _, f := range []string{"meta_title", "meta_description", "tags", "categories", "featured_image_url", "author_id"} {
			if v, ok := rec.PostMeta[f]; ok {
				switch f {
				case "meta_title":
					if req.MetaTitle == "" {
						req.MetaTitle, _ = v.(string)
					}
				case "meta_description":
					if req.MetaDescription == "" {
						req.MetaDescription, _ = v.(string)
					}
				case "featured_image_url":
					if req.FeaturedImageURL == "" {
						req.FeaturedImageURL, _ = v.(string)
					}
				}
			}
		}
	}

	if req.Title == "" {
		return nil, ErrTitleRequired
	}

	sc, err := s.resolveSiteContext(ctx, siteID)
	if err != nil {
		return nil, err
	}
	// The site's primary language is the root URL language; the pin (sitelang
	// override) supersedes the locale-derived primary wherever it applies.
	prim := sitelang.Resolve(siteID, sc.PrimaryLanguage)

	lang := strings.ToLower(req.Language)
	if lang == "" {
		lang = prim
	}
	if lang != "pt" && lang != "en" {
		return nil, ErrInvalidLanguage
	}
	// Site-level pin (e.g. AIWorkSimple = English-only): the override wins,
	// so a publication for that site is always created in "en".
	lang = sitelang.Resolve(siteID, lang)

	vis := req.Visibility
	if vis == "" {
		vis = VisibilityPublic
	}
	if err := s.val.ValidateVisibility(vis); err != nil {
		return nil, err
	}

	slug := req.Slug
	if slug == "" {
		slug = s.val.GenerateSlug(req.Title)
	} else {
		validSlug, err := s.val.ValidateSlug(slug)
		if err != nil {
			return nil, err
		}
		slug = validSlug
	}

	exists, err := s.repo.CheckDuplicateSlug(ctx, siteID, slug, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateSlug
	}

	if req.PostID != nil {
		if err := s.checkPublishGate(ctx, siteID, req.PostID, req.Title, req.Content, lang, req.MetaDescription, "", ""); err != nil {
			return nil, err
		}
		if !skipEditorialGate {
			if err := s.checkEditorialGate(ctx, siteID, req.PostID, req.Title, req.Content, lang, false); err != nil {
				return nil, err
			}
		}
	}

	now := time.Now()
	pubID := uuid.New()

	url := s.val.GenerateURL(slug, lang, prim, sc.Domain)
	canonical := req.CanonicalURL
	if canonical == "" {
		canonical = s.val.GenerateCanonicalURL(slug, lang, prim, sc.Domain)
	}

	wordCount := countWords(req.Content)
	readingTime := int(math.Ceil(float64(wordCount) / 200))

	pub := &Publication{
		ID:                  pubID,
		SiteID:              siteID,
		PostID:              req.PostID,
		Title:               req.Title,
		Content:             req.Content,
		Excerpt:             req.Excerpt,
		Slug:                slug,
		URL:                 url,
		CanonicalURL:        canonical,
		Language:            lang,
		Translations:        req.Translations,
		MultilingualURLs:    buildMultilingualURLs(req.Translations, slug, sc.Domain, prim),
		Status:              PubStatusPublished,
		Visibility:          vis,
		AuthorID:            req.AuthorID,
		PublishedBy:         &userID,
		PublishedAt:         &now,
		IsFeatured:          req.IsFeatured,
		MetaTitle:           req.MetaTitle,
		MetaDescription:     req.MetaDescription,
		OgImage:             req.OgImage,
		FeaturedImageURL:    req.FeaturedImageURL,
		FeaturedImageAlt:    req.FeaturedImageAlt,
		FeaturedImageCredit: req.FeaturedImageCredit,
		Tags:                req.Tags,
		Categories:          req.Categories,
		WordCount:           wordCount,
		ReadingTime:         readingTime,
		Revision:            1,
		Checksum:            s.val.ComputeChecksum(&Publication{Title: req.Title, Content: req.Content, Slug: slug, Tags: req.Tags, Categories: req.Categories, Revision: 1}),
		Source:              coalesceStr(req.Source, "manual"),
		Metadata:            req.Metadata,
		CreatedBy:           &userID,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	// The social card (og_image) and the featured image must agree: when only
	// one of them is provided, mirror it onto the other so the public article
	// page and the social preview always show the same photograph.
	if pub.OgImage == "" {
		pub.OgImage = pub.FeaturedImageURL
	}
	if pub.FeaturedImageURL == "" {
		pub.FeaturedImageURL = pub.OgImage
	}

	if err := s.repo.CreatePublication(ctx, pub); err != nil {
		return nil, fmt.Errorf("failed to publish article: %w", err)
	}

	s.recordHistory(ctx, pubID, siteID, HistoryPublished, "", string(pub.Status),
		pub.Title, pub.Slug, nil, "article published", &userID, now)

	s.fireEvent(ctx, EventPubPublished, map[string]interface{}{
		"publication_id": pubID.String(),
		"site_id":        siteID.String(),
		"slug":           slug,
		"url":            url,
		"language":       lang,
		"title":          req.Title,
	}, siteID)

	s.fireSEOEvents(ctx, siteID)

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("publisher.published"),
		EntityType: "publication",
		EntityID:   &pubID,
		Payload:    map[string]interface{}{"title": req.Title, "slug": slug, "language": lang},
	})

	s.cachePurge(ctx, siteID, pubID, slug)

	return &PublishResponse{Publication: pub}, nil
}

func (s *Service) PublishGeneratedArticle(ctx context.Context, req PublishGeneratedRequest) (*Publication, error) {
	pubReq := PublishRequest{
		Title:               req.Title,
		Content:             req.Content,
		Excerpt:             req.Excerpt,
		Slug:                req.Slug,
		Language:            req.Language,
		MetaTitle:           req.MetaTitle,
		MetaDescription:     req.MetaDescription,
		FeaturedImageURL:    req.FeaturedImageURL,
		FeaturedImageAlt:    req.FeaturedImageAlt,
		FeaturedImageCredit: req.FeaturedImageCredit,
		Tags:                req.Tags,
		Categories:          req.Categories,
		Source:              coalesceStr(req.Source, "generated"),
	}
	// Site-level pin (e.g. AIWorkSimple = English-only): generated content
	// is always published in the site's pinned language when no explicit
	// language was requested; otherwise the site's primary language applies.
	lang, err := s.effectiveLanguage(ctx, req.SiteID, req.Language)
	if err != nil {
		return nil, err
	}
	pubReq.Language = lang
	keyword := req.Keyword // "" → enhancer/gate derive deterministically from title+content
	var enhMeta string
	var enh *ContentEnhancement
	pubReq.Content, enhMeta, enh = s.enhanceContent(ctx, req.SiteID, nil, req.Title, req.Content, keyword, firstCategory(req.Categories), pubReq.Language, req.FeaturedImageURL, req.FeaturedImageAlt, req.FeaturedImageCredit)
	if enh != nil {
		if enh.FeaturedImageURL != "" {
			pubReq.FeaturedImageURL = enh.FeaturedImageURL
		}
		if enh.FeaturedImageAlt != "" {
			pubReq.FeaturedImageAlt = enh.FeaturedImageAlt
		}
		if enh.FeaturedImageCredit != nil {
			pubReq.FeaturedImageCredit = enh.FeaturedImageCredit
		}
		// The enhancer may derive a sharper focus keyword than the caller's;
		// the gate must evaluate the same keyword the content was enriched
		// with, so title/keyword/meta/keyword checks stay consistent.
		if enh.Keyword != "" {
			keyword = enh.Keyword
		}
	}
	if pubReq.MetaDescription == "" {
		pubReq.MetaDescription = enhMeta
	}
	// EEAT needs a byline: the explicit author wins, otherwise the site's
	// configured default author (SEO_DEFAULT_AUTHOR). Never fabricated.
	author := req.AuthorName
	if author == "" {
		author = s.defaultAuthor
	}
	if err := s.checkQualityGate(ctx, QualityGateInput{
		SiteID:        req.SiteID,
		Title:         req.Title,
		Content:       pubReq.Content,
		Language:      pubReq.Language,
		ContentType:   req.ContentType,
		ResearchFacts: req.ResearchFacts,
	}); err != nil {
		return nil, err
	}
	if err := s.checkPublishGate(ctx, req.SiteID, nil, req.Title, pubReq.Content, pubReq.Language, pubReq.MetaDescription, keyword, author); err != nil {
		return nil, err
	}
	if !req.EditorialApproved {
		// Human-approved content skips the editorial gate: the approval is
		// the editorial decision. Everything else is evaluated against a
		// real review note — never a fabricated 100.
		if err := s.checkEditorialGate(ctx, req.SiteID, nil, req.Title, pubReq.Content, pubReq.Language, true); err != nil {
			return nil, err
		}
	}
	resp, err := s.publishArticleInternal(ctx, req.SiteID, uuid.Nil, pubReq, req.EditorialApproved)
	if err != nil {
		return nil, err
	}
	return resp.Publication, nil
}

func ComputeFreshnessScore(publishedAt *time.Time) float64 {
	if publishedAt == nil {
		return 0.0
	}
	days := time.Since(*publishedAt).Hours() / 24
	if days < 0 {
		return 1.0
	}
	score := 1.0 - (days / 365.0)
	if score < 0 {
		return 0.0
	}
	return score
}

// --- Schedule ---

func (s *Service) SchedulePublication(ctx context.Context, siteID, userID uuid.UUID, req ScheduleRequest) (*PublishResponse, error) {
	if _, err := s.repo.GetPublicationByID(ctx, siteID, req.PublicationID); err != nil {
		return nil, err
	}

	action := req.Action
	if action == "" {
		action = "publish"
	}

	schedID := uuid.New()
	now := time.Now()

	schedule := &Schedule{
		ID:              schedID,
		SiteID:          siteID,
		PublicationID:   req.PublicationID,
		ScheduledAt:     req.ScheduledAt,
		Action:          action,
		Status:          ScheduleScheduled,
		Recurrence:      req.Recurrence,
		RecurrenceEnd:   req.RecurrenceEnd,
		NotifyOnPublish: req.NotifyOnPublish,
		CreatedBy:       &userID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreateSchedule(ctx, schedule); err != nil {
		return nil, fmt.Errorf("failed to schedule publication: %w", err)
	}

	s.recordHistory(ctx, req.PublicationID, siteID, HistoryScheduled, "", "scheduled",
		"", "", map[string]interface{}{
			"scheduled_at": req.ScheduledAt,
			"action":       action,
		}, "publication scheduled", &userID, now)

	s.fireEvent(ctx, EventPubScheduled, map[string]interface{}{
		"publication_id": req.PublicationID.String(),
		"site_id":        siteID.String(),
		"scheduled_at":   req.ScheduledAt,
		"action":         action,
	}, siteID)

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("publisher.scheduled"),
		EntityType: "publication_schedule",
		EntityID:   &schedID,
		Payload:    map[string]interface{}{"publication_id": req.PublicationID.String(), "scheduled_at": req.ScheduledAt},
	})

	pub, _ := s.repo.GetPublicationByID(ctx, siteID, req.PublicationID)
	return &PublishResponse{Publication: pub, Schedule: schedule}, nil
}

// --- Update ---

func (s *Service) UpdatePublication(ctx context.Context, siteID, userID uuid.UUID, pubID uuid.UUID, req UpdatePublicationRequest) (*Publication, error) {
	existing, err := s.repo.GetPublicationByID(ctx, siteID, pubID)
	if err != nil {
		return nil, err
	}

	changes := make(map[string]interface{})
	updates := make(map[string]interface{})

	if req.Title != nil {
		title := *req.Title
		if title == "" {
			return nil, ErrTitleRequired
		}
		updates["title"] = title
		changes["title"] = map[string]interface{}{"old": existing.Title, "new": title}
	}
	if req.Content != nil {
		updates["content"] = *req.Content
		changes["content"] = map[string]interface{}{"changed": true}
		wc := countWords(*req.Content)
		updates["word_count"] = wc
		updates["reading_time"] = int(math.Ceil(float64(wc) / 200))
	}
	if req.Excerpt != nil {
		updates["excerpt"] = *req.Excerpt
	}
	if req.Slug != nil {
		newSlug, err := s.val.ValidateSlug(*req.Slug)
		if err != nil {
			return nil, err
		}
		exists, err := s.repo.CheckDuplicateSlug(ctx, siteID, newSlug, &pubID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrDuplicateSlug
		}
		sc, err := s.resolveSiteContext(ctx, siteID)
		if err != nil {
			return nil, err
		}
		updates["slug"] = newSlug
		updates["url"] = s.val.GenerateURL(newSlug, existing.Language, sitelang.Resolve(siteID, sc.PrimaryLanguage), sc.Domain)
		if existing.CanonicalURL == "" || strings.Contains(existing.CanonicalURL, existing.Slug) {
			updates["canonical_url"] = s.val.GenerateCanonicalURL(newSlug, existing.Language, sitelang.Resolve(siteID, sc.PrimaryLanguage), sc.Domain)
		}
		changes["slug"] = map[string]interface{}{"old": existing.Slug, "new": newSlug}
	}
	if req.Language != nil {
		lang := strings.ToLower(*req.Language)
		if err := s.val.ValidateLanguage(lang); err != nil {
			return nil, err
		}
		updates["language"] = lang
		changes["language"] = map[string]interface{}{"old": existing.Language, "new": lang}
	}
	if req.Visibility != nil {
		if err := s.val.ValidateVisibility(*req.Visibility); err != nil {
			return nil, err
		}
		updates["visibility"] = string(*req.Visibility)
	}
	if req.IsFeatured != nil {
		updates["is_featured"] = *req.IsFeatured
	}
	if req.MetaTitle != nil {
		updates["meta_title"] = *req.MetaTitle
	}
	if req.MetaDescription != nil {
		updates["meta_description"] = *req.MetaDescription
	}
	if req.OgImage != nil {
		updates["og_image"] = *req.OgImage
	}
	if req.FeaturedImageURL != nil {
		updates["featured_image_url"] = *req.FeaturedImageURL
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
		changes["tags"] = map[string]interface{}{"changed": true}
	}
	if req.Categories != nil {
		updates["categories"] = *req.Categories
		changes["categories"] = map[string]interface{}{"changed": true}
	}
	if req.CanonicalURL != nil {
		updates["canonical_url"] = s.val.SanitizeURL(*req.CanonicalURL)
	}
	if req.Translations != nil {
		sc, err := s.resolveSiteContext(ctx, siteID)
		if err != nil {
			return nil, err
		}
		updates["translations"] = *req.Translations
		updates["multilingual_urls"] = buildMultilingualURLs(*req.Translations, existing.Slug, sc.Domain, sitelang.Resolve(siteID, sc.PrimaryLanguage))
		changes["translations"] = map[string]interface{}{"changed": true}
	}
	if req.Metadata != nil {
		updates["metadata"] = *req.Metadata
	}

	if len(updates) == 0 {
		return existing, nil
	}

	newRevision := existing.Revision + 1
	updates["revision"] = newRevision

	if err := s.repo.UpdatePublication(ctx, siteID, pubID, updates); err != nil {
		return nil, err
	}

	s.recordHistory(ctx, pubID, siteID, HistoryUpdated, string(existing.Status), string(existing.Status),
		existing.Title, existing.Slug, changes, "publication updated", &userID, time.Now())

	s.fireEvent(ctx, EventPubUpdated, map[string]interface{}{
		"publication_id": pubID.String(),
		"site_id":        siteID.String(),
		"changes":        changes,
		"revision":       newRevision,
	}, siteID)

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("publisher.updated"),
		EntityType: "publication",
		EntityID:   &pubID,
		Payload:    map[string]interface{}{"revision": newRevision},
	})

	s.cachePurge(ctx, siteID, pubID, existing.Slug)

	return s.repo.GetPublicationByID(ctx, siteID, pubID)
}

// --- Unpublish ---

func (s *Service) Unpublish(ctx context.Context, siteID, userID uuid.UUID, pubID uuid.UUID, reason string) (*Publication, error) {
	pub, err := s.repo.GetPublicationByID(ctx, siteID, pubID)
	if err != nil {
		return nil, err
	}
	if pub.Status != PubStatusPublished {
		return nil, ErrPublicationNotPublished
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":         string(PubStatusUnpublished),
		"unpublished_at": now,
	}
	if err := s.repo.UpdatePublication(ctx, siteID, pubID, updates); err != nil {
		return nil, err
	}

	s.recordHistory(ctx, pubID, siteID, HistoryUnpublished, string(PubStatusPublished), string(PubStatusUnpublished),
		pub.Title, pub.Slug, map[string]interface{}{"reason": reason}, coalesceStr(reason, "unpublished"), &userID, now)

	s.fireEvent(ctx, EventPubUnpublished, map[string]interface{}{
		"publication_id": pubID.String(),
		"site_id":        siteID.String(),
		"reason":         reason,
	}, siteID)

	s.fireSEOEvents(ctx, siteID)
	s.cachePurge(ctx, siteID, pubID, pub.Slug)

	return s.repo.GetPublicationByID(ctx, siteID, pubID)
}

// --- Republish ---

func (s *Service) Republish(ctx context.Context, siteID, userID uuid.UUID, pubID uuid.UUID) (*Publication, error) {
	pub, err := s.repo.GetPublicationByID(ctx, siteID, pubID)
	if err != nil {
		return nil, err
	}
	if pub.Status != PubStatusUnpublished {
		return nil, ErrPublicationAlreadyPublished
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":       string(PubStatusPublished),
		"published_at": now,
	}
	if err := s.repo.UpdatePublication(ctx, siteID, pubID, updates); err != nil {
		return nil, err
	}

	s.recordHistory(ctx, pubID, siteID, HistoryRepublished, string(PubStatusUnpublished), string(PubStatusPublished),
		pub.Title, pub.Slug, nil, "article republished", &userID, now)

	s.fireEvent(ctx, EventPubRepublished, map[string]interface{}{
		"publication_id": pubID.String(),
		"site_id":        siteID.String(),
		"slug":           pub.Slug,
	}, siteID)

	s.fireSEOEvents(ctx, siteID)
	s.cachePurge(ctx, siteID, pubID, pub.Slug)

	return s.repo.GetPublicationByID(ctx, siteID, pubID)
}

// --- Cancel Schedule ---

func (s *Service) CancelSchedule(ctx context.Context, siteID, userID uuid.UUID, scheduleID uuid.UUID, reason string) (*Schedule, error) {
	sched, err := s.repo.GetSchedule(ctx, siteID, scheduleID)
	if err != nil {
		return nil, err
	}

	if sched.Status != ScheduleScheduled {
		return nil, fmt.Errorf("schedule is not in scheduled state")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":        string(ScheduleCancelled),
		"cancelled_at":  now,
		"cancel_reason": reason,
	}
	if err := s.repo.UpdateSchedule(ctx, siteID, scheduleID, updates); err != nil {
		return nil, err
	}

	s.recordHistory(ctx, sched.PublicationID, siteID, HistoryCancelled, "scheduled", "cancelled",
		"", "", map[string]interface{}{"reason": reason}, coalesceStr(reason, "schedule cancelled"), &userID, now)

	s.fireEvent(ctx, EventPubCancelled, map[string]interface{}{
		"schedule_id":    scheduleID.String(),
		"publication_id": sched.PublicationID.String(),
		"site_id":        siteID.String(),
		"reason":         reason,
	}, siteID)

	return s.repo.GetSchedule(ctx, siteID, scheduleID)
}

// --- Queue ---

func (s *Service) AddToQueue(ctx context.Context, siteID, userID uuid.UUID, req QueueRequest) (*QueueItem, error) {
	action := QueueAction(req.Action)
	if action == "" {
		action = QueueActionPublish
	}
	switch action {
	case QueueActionPublish, QueueActionUnpublish, QueueActionRepublish, QueueActionUpdate:
	default:
		return nil, ErrInvalidAction
	}

	if _, err := s.repo.GetPublicationByID(ctx, siteID, req.PublicationID); err != nil {
		return nil, err
	}

	priority := req.Priority
	if priority < 1 || priority > 10 {
		priority = 5
	}

	now := time.Now()
	itemID := uuid.New()

	item := &QueueItem{
		ID:            itemID,
		SiteID:        siteID,
		PublicationID: &req.PublicationID,
		Action:        action,
		Status:        QueuePending,
		Priority:      priority,
		ScheduledFor:  req.ScheduledFor,
		CreatedBy:     &userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.CreateQueueItem(ctx, item); err != nil {
		return nil, fmt.Errorf("failed to add to queue: %w", err)
	}

	s.fireEvent(ctx, EventPubQueueAdded, map[string]interface{}{
		"queue_item_id":  itemID.String(),
		"publication_id": req.PublicationID.String(),
		"site_id":        siteID.String(),
		"action":         action,
	}, siteID)

	return item, nil
}

func (s *Service) ListQueue(ctx context.Context, siteID uuid.UUID, status string, limit, offset int) ([]QueueItem, error) {
	return s.repo.ListQueue(ctx, siteID, status, limit, offset)
}

func (s *Service) RetryQueueItem(ctx context.Context, siteID, userID uuid.UUID, itemID uuid.UUID) (*QueueItem, error) {
	item, err := s.repo.GetQueueItem(ctx, siteID, itemID)
	if err != nil {
		return nil, err
	}
	if item.Status != QueueFailed {
		return nil, fmt.Errorf("queue item is not in failed state")
	}
	if item.RetryCount >= item.MaxRetries {
		return nil, ErrMaxRetriesExceeded
	}

	updates := map[string]interface{}{
		"status":        string(QueuePending),
		"retry_count":   item.RetryCount + 1,
		"error_message": "",
		"started_at":    nil,
		"completed_at":  nil,
	}
	if err := s.repo.UpdateQueueItem(ctx, siteID, itemID, updates); err != nil {
		return nil, err
	}

	s.fireEvent(ctx, EventPubQueueRetried, map[string]interface{}{
		"queue_item_id": itemID.String(),
		"site_id":       siteID.String(),
		"retry_count":   item.RetryCount + 1,
	}, siteID)

	return s.repo.GetQueueItem(ctx, siteID, itemID)
}

// --- Publication CRUD ---

func (s *Service) GetPublication(ctx context.Context, siteID, pubID uuid.UUID) (*Publication, error) {
	pub, err := s.repo.GetPublicationByID(ctx, siteID, pubID)
	if err != nil {
		return nil, err
	}
	return pub, nil
}

func (s *Service) GetPublicationBySlug(ctx context.Context, siteID uuid.UUID, slug string) (*Publication, error) {
	return s.repo.GetPublicationBySlug(ctx, siteID, slug)
}

func (s *Service) ListPublications(ctx context.Context, siteID uuid.UUID, status, language string, limit, offset int) ([]Publication, int, error) {
	return s.repo.ListPublications(ctx, siteID, status, language, limit, offset)
}

func (s *Service) DeletePublication(ctx context.Context, siteID, userID uuid.UUID, pubID uuid.UUID) error {
	pub, err := s.repo.GetPublicationByID(ctx, siteID, pubID)
	if err != nil {
		return err
	}

	if err := s.repo.DeletePublication(ctx, siteID, pubID); err != nil {
		return err
	}

	s.recordHistory(ctx, pubID, siteID, HistoryDeleted, string(pub.Status), "deleted",
		pub.Title, pub.Slug, nil, "publication deleted", &userID, time.Now())

	s.fireEvent(ctx, EventPubDeleted, map[string]interface{}{
		"publication_id": pubID.String(),
		"site_id":        siteID.String(),
	}, siteID)

	s.fireSEOEvents(ctx, siteID)
	s.cachePurge(ctx, siteID, pubID, pub.Slug)

	return nil
}

// --- History ---

func (s *Service) GetPublicationHistory(ctx context.Context, siteID, pubID uuid.UUID, limit, offset int) ([]PublicationHistory, error) {
	return s.repo.ListHistory(ctx, siteID, pubID, limit, offset)
}

func (s *Service) recordHistory(ctx context.Context, pubID, siteID uuid.UUID, action HistoryAction, prevStatus, newStatus, title, slug string, changes map[string]interface{}, reason string, performedBy *uuid.UUID, performedAt time.Time) {
	h := &PublicationHistory{
		ID:             uuid.New(),
		PublicationID:  pubID,
		SiteID:         siteID,
		Action:         action,
		PreviousStatus: prevStatus,
		NewStatus:      newStatus,
		Title:          title,
		Slug:           slug,
		Changes:        changes,
		Reason:         reason,
		PerformedBy:    performedBy,
		PerformedAt:    performedAt,
		CreatedAt:      performedAt,
	}
	if err := s.repo.CreateHistory(ctx, h); err != nil {
		s.log.Error("failed to record history", "error", err)
	}
}

// --- Metrics ---

func (s *Service) GetPublicationMetrics(ctx context.Context, siteID, pubID uuid.UUID) (*PublicationMetrics, error) {
	return s.repo.GetMetrics(ctx, siteID, pubID)
}

func (s *Service) GetMetricsSummary(ctx context.Context, siteID uuid.UUID) (*PublicationMetricsSummary, error) {
	return s.repo.GetMetricsSummary(ctx, siteID)
}

// --- Schedules ---

func (s *Service) ListSchedules(ctx context.Context, siteID uuid.UUID, status string, limit, offset int) ([]Schedule, error) {
	return s.repo.ListSchedules(ctx, siteID, status, limit, offset)
}

func (s *Service) GetSchedule(ctx context.Context, siteID, scheduleID uuid.UUID) (*Schedule, error) {
	return s.repo.GetSchedule(ctx, siteID, scheduleID)
}

// --- Validation ---

func (s *Service) ValidateSlug(ctx context.Context, siteID uuid.UUID, slug string) (bool, string, error) {
	validSlug, err := s.val.ValidateSlug(slug)
	if err != nil {
		return false, "", err
	}
	exists, err := s.repo.CheckDuplicateSlug(ctx, siteID, validSlug, nil)
	if err != nil {
		return false, "", err
	}
	return !exists, validSlug, nil
}

func (s *Service) GenerateSlug(ctx context.Context, siteID uuid.UUID, title string) (string, error) {
	slug := s.val.GenerateSlug(title)
	exists, err := s.repo.CheckDuplicateSlug(ctx, siteID, slug, nil)
	if err != nil {
		return "", err
	}
	if exists {
		for i := 2; i < 100; i++ {
			candidate := fmt.Sprintf("%s-%d", slug, i)
			exists, err := s.repo.CheckDuplicateSlug(ctx, siteID, candidate, nil)
			if err != nil {
				return "", err
			}
			if !exists {
				return candidate, nil
			}
		}
	}
	return slug, nil
}

// GenerateSlugURL returns the preview URL for a slug in the site's effective
// (primary + pinned) language and resolved domain. It is used by the
// generate-slug endpoint so the returned URL reflects the real site. It
// returns an explicit error when the site has no verified domain, so the
// endpoint never emits a placeholder domain.
func (s *Service) GenerateSlugURL(ctx context.Context, siteID uuid.UUID, slug string) (string, error) {
	sc, err := s.resolveSiteContext(ctx, siteID)
	if err != nil {
		return "", err
	}
	prim := sitelang.Resolve(siteID, sc.PrimaryLanguage)
	return s.val.GenerateURL(slug, prim, prim, sc.Domain), nil
}

// --- SEO Events ---

func (s *Service) fireSEOEvents(ctx context.Context, siteID uuid.UUID) {
	s.fireEvent(ctx, EventPubSitemapUpdate, map[string]interface{}{
		"site_id": siteID.String(),
	}, siteID)
	s.fireEvent(ctx, EventPubRSSUpdate, map[string]interface{}{
		"site_id": siteID.String(),
	}, siteID)
	s.fireEvent(ctx, EventPubRobotsRefresh, map[string]interface{}{
		"site_id": siteID.String(),
	}, siteID)
}

func (s *Service) cachePurge(ctx context.Context, siteID uuid.UUID, pubID uuid.UUID, slug string) {
	s.fireEvent(ctx, EventPubCachePurge, map[string]interface{}{
		"site_id":        siteID.String(),
		"publication_id": pubID.String(),
		"slug":           slug,
	}, siteID)
	if s.cache != nil {
		cacheKey := fmt.Sprintf("publication:%s:%s", siteID.String(), pubID.String())
		_ = s.cache.Delete(ctx, cacheKey)
		slugKey := fmt.Sprintf("publication:slug:%s:%s", siteID.String(), slug)
		_ = s.cache.Delete(ctx, slugKey)
	}
}

// --- Helpers ---

func countWords(s string) int {
	if s == "" {
		return 0
	}
	words := strings.Fields(s)
	return len(words)
}

// buildMultilingualURLs builds the per-language URL map for a publication.
// The site's primary language maps to the root; every other language gets a
// /{lang}/ prefix (legacy behavior treated "pt" as the universal root).
func buildMultilingualURLs(translations map[string]interface{}, baseSlug, siteDomain, primaryLanguage string) map[string]interface{} {
	if translations == nil {
		return map[string]interface{}{}
	}
	prim := strings.ToLower(primaryLanguage)
	if prim == "" {
		prim = "pt"
	}
	base := strings.TrimRight(siteDomain, "/")
	result := make(map[string]interface{})
	for lang := range translations {
		langStr := strings.ToLower(lang)
		if langStr == prim {
			result[langStr] = fmt.Sprintf("%s/%s", base, baseSlug)
		} else {
			result[langStr] = fmt.Sprintf("%s/%s/%s", base, langStr, baseSlug)
		}
	}
	return result
}

func coalesceStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// firstCategory returns the first category in the list, or "" when empty.
func firstCategory(categories []string) string {
	for _, c := range categories {
		if strings.TrimSpace(c) != "" {
			return strings.TrimSpace(c)
		}
	}
	return ""
}

// deriveKeyword picks the longest non-stopword token from a title. Deterministic
// primary-keyword fallback for the content enhancer.
func deriveKeyword(title string) string {
	best := ""
	seen := map[string]bool{}
	for _, w := range strings.Fields(strings.ToLower(title)) {
		w = strings.Trim(w, ".,;:!?\"'()[]{}")
		if w == "" || stopWord(w) || seen[w] {
			continue
		}
		seen[w] = true
		if len(w) > len(best) {
			best = w
		}
	}
	if best == "" {
		words := strings.Fields(title)
		if len(words) > 0 {
			return strings.Trim(strings.ToLower(words[0]), ".,;:!?\"'()[]{}")
		}
		return "article"
	}
	return best
}

var publisherStopWords = map[string]bool{
	"a": true, "o": true, "e": true, "de": true, "da": true, "do": true,
	"em": true, "para": true, "com": true, "por": true, "um": true, "uma": true,
	"na": true, "no": true, "os": true, "as": true, "que": true, "ao": true,
	"the": true, "and": true, "of": true, "to": true, "in": true, "for": true,
	"on": true, "with": true, "is": true, "at": true, "from": true, "by": true,
	"sobre": true, "entre": true, "como": true, "mais": true, "mas": true,
	"or": true, "an": true, "be": true, "this": true, "that": true,
	"new": true, "best": true, "top": true, "guide": true, "review": true,
	"latest": true, "cheap": true, "free": true, "buy": true, "sale": true,
	"deal": true, "official": true, "ultimate": true, "complete": true,
	"amazing": true, "great": true, "good": true, "better": true,
	"novo": true, "nova": true, "novos": true, "novas": true,
	"melhor": true, "melhores": true, "gratis": true, "gratuito": true,
	"gratuita": true, "barato": true, "baratos": true, "comprar": true,
	"compra": true, "porque": true, "muito": true, "muitos": true,
	"apenas": true, "somente": true, "nosso": true, "nossa": true,
	"seus": true, "sua": true, "suas": true,
}

func stopWord(w string) bool {
	return publisherStopWords[w]
}
