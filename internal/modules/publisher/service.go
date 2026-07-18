package publisher

import (
	"context"
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
)

type Service struct {
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	repo     *Repository
	val      *Validator
	eventBus *kernel.EventBus
	auditLog *audit.Logger
	siteDomain string
}

func NewService(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *Service {
	var pool database.Pool
	if db != nil {
		pool = db.Pool
	}
	return &Service{
		log:      log,
		db:       db,
		cache:    ch,
		repo:     NewRepository(pool),
		val:      NewValidator(),
		auditLog: audit.New(pool, log),
		siteDomain: "https://example.com",
	}
}

func (s *Service) SetEventBus(bus *kernel.EventBus) {
	s.eventBus = bus
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
	if req.Title == "" {
		return nil, ErrTitleRequired
	}

	lang := strings.ToLower(req.Language)
	if lang == "" {
		lang = "pt"
	}
	if lang != "pt" && lang != "en" {
		return nil, ErrInvalidLanguage
	}

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

	now := time.Now()
	pubID := uuid.New()

	url := s.val.GenerateURL(slug, lang, s.siteDomain)
	canonical := req.CanonicalURL
	if canonical == "" {
		canonical = s.val.GenerateCanonicalURL(slug, lang, "pt", s.siteDomain)
	}

	wordCount := countWords(req.Content)
	readingTime := int(math.Ceil(float64(wordCount) / 200))

	pub := &Publication{
		ID:               pubID,
		SiteID:           siteID,
		PostID:           req.PostID,
		Title:            req.Title,
		Content:          req.Content,
		Excerpt:          req.Excerpt,
		Slug:             slug,
		URL:              url,
		CanonicalURL:     canonical,
		Language:         lang,
		Translations:     req.Translations,
		MultilingualURLs: buildMultilingualURLs(req.Translations, slug, s.siteDomain),
		Status:           PubStatusPublished,
		Visibility:       vis,
		AuthorID:         req.AuthorID,
		PublishedBy:      &userID,
		PublishedAt:      &now,
		IsFeatured:       req.IsFeatured,
		MetaTitle:        req.MetaTitle,
		MetaDescription:  req.MetaDescription,
		OgImage:          req.OgImage,
		FeaturedImageURL: req.FeaturedImageURL,
		Tags:             req.Tags,
		Categories:       req.Categories,
		WordCount:        wordCount,
		ReadingTime:      readingTime,
		Revision:         1,
		Checksum:         s.val.ComputeChecksum(&Publication{Title: req.Title, Content: req.Content, Slug: slug, Tags: req.Tags, Categories: req.Categories, Revision: 1}),
		Source:           coalesceStr(req.Source, "manual"),
		Metadata:         req.Metadata,
		CreatedBy:        &userID,
		CreatedAt:        now,
		UpdatedAt:        now,
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
		updates["slug"] = newSlug
		updates["url"] = s.val.GenerateURL(newSlug, existing.Language, s.siteDomain)
		if existing.CanonicalURL == "" || strings.Contains(existing.CanonicalURL, existing.Slug) {
			updates["canonical_url"] = s.val.GenerateCanonicalURL(newSlug, existing.Language, "pt", s.siteDomain)
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
		updates["translations"] = *req.Translations
		updates["multilingual_urls"] = buildMultilingualURLs(*req.Translations, existing.Slug, s.siteDomain)
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

	if err := s.repo.UpdatePublication(ctx, pubID, updates); err != nil {
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
	if err := s.repo.UpdatePublication(ctx, pubID, updates); err != nil {
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
	if err := s.repo.UpdatePublication(ctx, pubID, updates); err != nil {
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
	if err := s.repo.UpdateSchedule(ctx, scheduleID, updates); err != nil {
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
		"status":     string(QueuePending),
		"retry_count": item.RetryCount + 1,
		"error_message": "",
		"started_at": nil,
		"completed_at": nil,
	}
	if err := s.repo.UpdateQueueItem(ctx, itemID, updates); err != nil {
		return nil, err
	}

	s.fireEvent(ctx, EventPubQueueRetried, map[string]interface{}{
		"queue_item_id":  itemID.String(),
		"site_id":        siteID.String(),
		"retry_count":    item.RetryCount + 1,
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

func buildMultilingualURLs(translations map[string]interface{}, baseSlug, siteDomain string) map[string]interface{} {
	if translations == nil {
		return map[string]interface{}{}
	}
	result := make(map[string]interface{})
	for lang, _ := range translations {
		langStr := strings.ToLower(lang)
		if langStr == "pt" {
			result[langStr] = fmt.Sprintf("%s/%s", strings.TrimRight(siteDomain, "/"), baseSlug)
		} else {
			result[langStr] = fmt.Sprintf("%s/%s/%s", strings.TrimRight(siteDomain, "/"), langStr, baseSlug)
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
