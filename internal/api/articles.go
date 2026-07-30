package api

import (
	"strconv"
	"time"

	"github.com/google/uuid"

	"nexora/internal/api/middleware"
	"nexora/internal/api/rest"
	categoriesModule "nexora/internal/modules/categories"
	publisherModule "nexora/internal/modules/publisher"
	researchModule "nexora/internal/modules/research"
	"nexora/internal/pkg/logger"
)

type publicArticleHandler struct {
	publisherSvc *publisherModule.Service
	researchSvc  *researchModule.Service
	log          *logger.Logger
}

func newPublicArticleHandler(deps *Dependencies) *publicArticleHandler {
	return &publicArticleHandler{
		publisherSvc: deps.PublisherSvc,
		researchSvc:  deps.ResearchSvc,
		log:          deps.Log,
	}
}

type PublicArticleResponse struct {
	ID               uuid.UUID              `json:"id"`
	SiteID           uuid.UUID              `json:"site_id"`
	Title            string                 `json:"title"`
	Slug             string                 `json:"slug"`
	Excerpt          string                 `json:"excerpt,omitempty"`
	Content          string                 `json:"content,omitempty"`
	FeaturedImageURL string                 `json:"featured_image_url,omitempty"`
	AuthorID         *uuid.UUID             `json:"author_id,omitempty"`
	PublishedAt      *time.Time             `json:"published_at"`
	MetaTitle        string                 `json:"meta_title,omitempty"`
	MetaDescription  string                 `json:"meta_description,omitempty"`
	OgImage          string                 `json:"og_image,omitempty"`
	CanonicalURL     string                 `json:"canonical_url,omitempty"`
	Language         string                 `json:"language"`
	Tags             []string               `json:"tags,omitempty"`
	Categories       []string               `json:"categories,omitempty"`
	WordCount        int                    `json:"word_count"`
	ReadingTime      int                    `json:"reading_time"`
	FreshnessScore   float64                `json:"freshness_score,omitempty"`
	Sources          []PublicArticleSource  `json:"sources,omitempty"`
}

type PublicArticleSource struct {
	URL            string     `json:"url"`
	Title          string     `json:"title,omitempty"`
	Snippet        string     `json:"snippet,omitempty"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	RetrievedAt    time.Time  `json:"retrieved_at"`
	FreshnessScore float64    `json:"freshness_score,omitempty"`
	IsVerified     bool       `json:"is_verified"`
	DomainRank     int        `json:"domain_rank,omitempty"`
	RelevanceScore int        `json:"relevance_score,omitempty"`
}

type PublicArticleListResponse struct {
	Articles []PublicArticleResponse `json:"articles"`
	Total    int                     `json:"total"`
}

func (h *publicArticleHandler) List(c *rest.Context) {
	siteID, ok := middleware.GetSiteID(c.Request.Context())
	if !ok || siteID == uuid.Nil {
		c.Error(400, "SITE_REQUIRED", "site identifier is required")
		return
	}

	if h.publisherSvc == nil {
		c.Error(503, "SERVICE_UNAVAILABLE", "publisher service not available")
		return
	}

	limit, _ := strconv.Atoi(c.Request.URL.Query().Get("limit"))
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	offset, _ := strconv.Atoi(c.Request.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	language := c.Request.URL.Query().Get("language")

	pubs, total, err := h.publisherSvc.ListPublications(
		c.Request.Context(), siteID, string(publisherModule.PubStatusPublished), language, limit, offset,
	)
	if err != nil {
		h.log.Error("failed to list public articles", "error", err)
		c.Error(500, "INTERNAL", "failed to list articles")
		return
	}

	articles := make([]PublicArticleResponse, 0, len(pubs))
	for _, pub := range pubs {
		articles = append(articles, toPublicArticleResponse(&pub, h.researchSvc, siteID, c))
	}

	c.JSON(200, PublicArticleListResponse{Articles: articles, Total: total})
}

func toPublicArticleResponse(pub *publisherModule.Publication, researchSvc *researchModule.Service, siteID uuid.UUID, c *rest.Context) PublicArticleResponse {
	resp := PublicArticleResponse{
		ID:               pub.ID,
		SiteID:           pub.SiteID,
		Title:            pub.Title,
		Slug:             pub.Slug,
		Excerpt:          pub.Excerpt,
		Content:          pub.Content,
		FeaturedImageURL: pub.FeaturedImageURL,
		AuthorID:         pub.AuthorID,
		PublishedAt:      pub.PublishedAt,
		MetaTitle:        pub.MetaTitle,
		MetaDescription:  pub.MetaDescription,
		OgImage:          pub.OgImage,
		CanonicalURL:     pub.CanonicalURL,
		Language:         pub.Language,
		Tags:             pub.Tags,
		Categories:       pub.Categories,
		WordCount:        pub.WordCount,
		ReadingTime:      pub.ReadingTime,
		FreshnessScore:   publisherModule.ComputeFreshnessScore(pub.PublishedAt),
	}

	if researchSvc != nil && pub.PublishedAt != nil {
		sources, err := researchSvc.GetArticleSources(c.Request.Context(), siteID, map[string]interface{}{
			"article_id": pub.ID,
		})
		if err == nil && len(sources) > 0 {
			for _, src := range sources {
				fs := src.FreshnessScore
				if fs == 0 && src.PublishedAt != nil {
					fs = publisherModule.ComputeFreshnessScore(src.PublishedAt)
				}
				resp.Sources = append(resp.Sources, PublicArticleSource{
					URL:            src.SourceURL,
					Title:          src.Title,
					Snippet:        src.Snippet,
					PublishedAt:    src.PublishedAt,
					RetrievedAt:    src.RetrievedAt,
					FreshnessScore: fs,
					IsVerified:     src.IsVerified,
					DomainRank:     src.DomainRank,
					RelevanceScore: src.RelevanceScore,
				})
			}
		}
	}

	return resp
}

func (h *publicArticleHandler) GetBySlug(c *rest.Context) {
	siteID, ok := middleware.GetSiteID(c.Request.Context())
	if !ok || siteID == uuid.Nil {
		c.Error(400, "SITE_REQUIRED", "site identifier is required")
		return
	}

	slug := c.PathValue("slug")
	if slug == "" {
		c.Error(400, "INVALID_SLUG", "slug is required")
		return
	}

	if h.publisherSvc == nil {
		c.Error(503, "SERVICE_UNAVAILABLE", "publisher service not available")
		return
	}

	pub, err := h.publisherSvc.GetPublicationBySlug(c.Request.Context(), siteID, slug)
	if err != nil {
		c.Error(404, "NOT_FOUND", "article not found")
		return
	}

	if pub.Status != publisherModule.PubStatusPublished {
		c.Error(404, "NOT_FOUND", "article not found")
		return
	}

	c.JSON(200, toPublicArticleResponse(pub, h.researchSvc, siteID, c))
}

// --- Public Categories Handler ---

type publicCategoriesHandler struct {
	categoriesSvc *categoriesModule.Service
	log           *logger.Logger
}

func newPublicCategoriesHandler(deps *Dependencies) *publicCategoriesHandler {
	return &publicCategoriesHandler{
		categoriesSvc: deps.CategoriesSvc,
		log:           deps.Log,
	}
}

func (h *publicCategoriesHandler) List(c *rest.Context) {
	siteID, ok := middleware.GetSiteID(c.Request.Context())
	if !ok || siteID == uuid.Nil {
		c.Error(400, "SITE_REQUIRED", "site identifier is required")
		return
	}

	if h.categoriesSvc == nil {
		c.Error(503, "SERVICE_UNAVAILABLE", "categories service not available")
		return
	}

	resp, err := h.categoriesSvc.List(c.Request.Context(), siteID)
	if err != nil {
		h.log.Error("failed to list public categories", "error", err)
		c.Error(500, "INTERNAL", "failed to list categories")
		return
	}

	c.JSON(200, resp)
}
