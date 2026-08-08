package api

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"nexora/internal/api/middleware"
	"nexora/internal/api/rest"
	publisherModule "nexora/internal/modules/publisher"
	seoengineModule "nexora/internal/modules/seoengine"
)

// siteSchemaResponse is the public site-level JSON-LD payload (Organization +
// WebSite schemas used on the homepage).
type siteSchemaResponse struct {
	Organization string `json:"organization,omitempty"`
	WebSite      string `json:"web_site,omitempty"`
}

// publicSiteSchemaHandler serves the site-level JSON-LD schemas.
type publicSiteSchemaHandler struct {
	publisherSvc *publisherModule.Service
}

func newPublicSiteSchemaHandler(deps *Dependencies) *publicSiteSchemaHandler {
	return &publicSiteSchemaHandler{publisherSvc: deps.PublisherSvc}
}

func (h *publicSiteSchemaHandler) Get(c *rest.Context) {
	siteID, ok := middleware.GetSiteID(c.Request.Context())
	if !ok || siteID == uuid.Nil {
		c.Error(400, "SITE_REQUIRED", "site identifier is required")
		return
	}

	domain := "https://example.com"
	if h.publisherSvc != nil {
		domain = h.publisherSvc.SiteDomain()
	}
	siteName := siteNameFromDomain(domain)
	base := strings.TrimRight(domain, "/") + "/"

	resp := siteSchemaResponse{}
	if org, err := seoengineModule.BuildOrganizationSchema(siteName, base, "", "", nil); err == nil {
		resp.Organization = org
	}
	if ws, err := seoengineModule.BuildWebSiteSchema(siteName, base, base+"busca?q={search_term_string}", "pt"); err == nil {
		resp.WebSite = ws
	}

	c.JSON(http.StatusOK, resp)
}

// HreflangLink is one hreflang alternate entry for international SEO.
type HreflangLink struct {
	Lang string `json:"lang"`
	URL  string `json:"url"`
}

// PublicArticleSEO carries the international-SEO + rich-snippet data attached
// to every public article response.
type PublicArticleSEO struct {
	Hreflang         []HreflangLink `json:"hreflang,omitempty"`
	OgType           string         `json:"og_type,omitempty"`
	TwitterCard      string         `json:"twitter_card,omitempty"`
	SchemaJSONLD     []string       `json:"schema_json_ld,omitempty"`
	SiteSchemaJSONLD []string       `json:"site_schema_json_ld,omitempty"`
}

// buildArticleSEO generates hreflang alternates, Open Graph/Twitter card data
// and the JSON-LD rich snippets (Article/NewsArticle, FAQ, HowTo, Breadcrumb)
// for a publication. Pure and deterministic.
func buildArticleSEO(pub *publisherModule.Publication, siteDomain string) PublicArticleSEO {
	seo := PublicArticleSEO{
		OgType:      "article",
		TwitterCard: "summary_large_image",
	}

	// Hreflang alternates from multilingual URLs (self + siblings).
	self := pub.URL
	if self == "" {
		self = pub.CanonicalURL
	}
	if self != "" {
		seo.Hreflang = append(seo.Hreflang, HreflangLink{Lang: pub.Language, URL: self})
	}
	langs := []string{}
	for lang, u := range pub.MultilingualURLs {
		urlStr, _ := u.(string)
		if urlStr == "" {
			continue
		}
		if urlStr == self {
			continue
		}
		seo.Hreflang = append(seo.Hreflang, HreflangLink{Lang: lang, URL: urlStr})
		langs = append(langs, lang)
	}
	sort.SliceStable(seo.Hreflang, func(i, j int) bool {
		return seo.Hreflang[i].Lang < seo.Hreflang[j].Lang
	})

	// JSON-LD rich snippets.
	url := self
	headline := pub.Title
	desc := pub.MetaDescription
	if desc == "" {
		desc = pub.Excerpt
	}
	image := pub.OgImage
	if image == "" {
		image = pub.FeaturedImageURL
	}
	datePublished := ""
	dateModified := ""
	if pub.PublishedAt != nil {
		datePublished = pub.PublishedAt.Format(time.RFC3339)
		dateModified = pub.UpdatedAt.Format(time.RFC3339)
	}

	if article, err := seoengineModule.BuildArticleSchemaJSONLD(url, pub.Title, headline, desc, image, "", "", "", datePublished, dateModified, pub.Language); err == nil && article != "" {
		seo.SchemaJSONLD = append(seo.SchemaJSONLD, article)
	}
	if isNewsArticle(pub.Categories) {
		if news, err := seoengineModule.BuildNewsArticleSchema(url, headline, desc, image, "", "", datePublished, dateModified); err == nil && news != "" {
			seo.SchemaJSONLD = append(seo.SchemaJSONLD, news)
		}
	}

	contentLower := strings.ToLower(pub.Content)
	if strings.Contains(contentLower, "faq") || strings.Contains(contentLower, "perguntas frequentes") || strings.Contains(contentLower, "perguntas e respostas") {
		if faq, err := seoengineModule.BuildFAQSchema(url, extractFAQPairs(pub.Content)); err == nil && faq != "" {
			seo.SchemaJSONLD = append(seo.SchemaJSONLD, faq)
		}
	}
	if strings.Contains(contentLower, "how to") || strings.Contains(contentLower, "como fazer") || strings.Contains(contentLower, "passo a passo") {
		if howto, err := seoengineModule.BuildHowToSchema(pub.Title, desc, extractHowToSteps(pub.Content)); err == nil && howto != "" {
			seo.SchemaJSONLD = append(seo.SchemaJSONLD, howto)
		}
	}

	if url != "" {
		if breadcrumb, err := seoengineModule.BuildBreadcrumbSchema([]seoengineModule.BreadcrumbItem{
			{Name: "Home", URL: strings.TrimRight(siteDomain, "/") + "/"},
			{Name: pub.Title, URL: url},
		}); err == nil && breadcrumb != "" {
			seo.SchemaJSONLD = append(seo.SchemaJSONLD, breadcrumb)
		}
	}

	// Site-level schemas (Organization + WebSite).
	siteName := siteNameFromDomain(siteDomain)
	if org, err := seoengineModule.BuildOrganizationSchema(siteName, strings.TrimRight(siteDomain, "/")+"/", image, "", nil); err == nil && org != "" {
		seo.SiteSchemaJSONLD = append(seo.SiteSchemaJSONLD, org)
	}
	if site, err := seoengineModule.BuildWebSiteSchema(siteName, strings.TrimRight(siteDomain, "/")+"/", strings.TrimRight(siteDomain, "/")+"/busca?q={search_term_string}", pub.Language); err == nil && site != "" {
		seo.SiteSchemaJSONLD = append(seo.SiteSchemaJSONLD, site)
	}

	return seo
}

func isNewsArticle(categories []string) bool {
	for _, c := range categories {
		switch strings.ToLower(strings.TrimSpace(c)) {
		case "news", "notícias", "noticias", "atualidade", "novidades":
			return true
		}
	}
	return false
}

// extractFAQPairs pulls simple "Q: ... A: ..." pairs from the content for the
// FAQPage schema. Deterministic; returns at most 10 pairs.
func extractFAQPairs(content string) []seoengineModule.FAQQuestion {
	pairs := []seoengineModule.FAQQuestion{}
	lines := strings.Split(content, "\n")
	var currentQ string
	for _, line := range lines {
		t := strings.TrimSpace(line)
		lower := strings.ToLower(t)
		if strings.HasPrefix(lower, "q:") || strings.HasPrefix(lower, "p:") {
			currentQ = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(t, "Q:"), "P:"))
			currentQ = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(currentQ, "q:"), "p:"))
			continue
		}
		if (strings.HasPrefix(lower, "a:") || strings.HasPrefix(lower, "r:")) && currentQ != "" {
			answer := strings.TrimSpace(t)
			answer = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(answer, "A:"), "R:"))
			answer = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(answer, "a:"), "r:"))
			pairs = append(pairs, seoengineModule.FAQQuestion{Question: currentQ, Answer: answer})
			currentQ = ""
			if len(pairs) >= 10 {
				break
			}
		}
	}
	return pairs
}

// extractHowToSteps pulls "1. ... 2. ..." numbered steps for the HowTo schema.
func extractHowToSteps(content string) []seoengineModule.HowToStep {
	steps := []seoengineModule.HowToStep{}
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		i := 0
		for i < len(t) && t[i] >= '0' && t[i] <= '9' {
			i++
		}
		if i == 0 || i >= len(t) || t[i] != '.' && t[i] != ')' {
			continue
		}
		body := strings.TrimSpace(t[i+1:])
		if body == "" {
			continue
		}
		steps = append(steps, seoengineModule.HowToStep{Title: body, Text: body})
		if len(steps) >= 10 {
			break
		}
	}
	return steps
}

// deriveSiteDomain extracts the scheme+host base from a publication URL so the
// SEO layer can build absolute hreflang/breadcrumb/schema URLs.
func deriveSiteDomain(urls ...string) string {
	for _, u := range urls {
		if u == "" {
			continue
		}
		if strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") {
			lower := strings.ToLower(u)
			if i := strings.Index(lower, "//"); i >= 0 {
				rest := lower[i+2:]
				if j := strings.IndexAny(rest, "/"); j >= 0 {
					return lower[:i+2] + rest[:j]
				}
				return lower
			}
		}
	}
	return "https://example.com"
}

func siteNameFromDomain(siteDomain string) string {
	domain := strings.TrimPrefix(strings.TrimPrefix(siteDomain, "https://"), "http://")
	domain = strings.TrimSuffix(domain, "/")
	parts := strings.Split(domain, ".")
	if len(parts) >= 2 {
		return strings.ToUpper(parts[len(parts)-2][:1]) + parts[len(parts)-2][1:]
	}
	if domain == "" {
		return "Nexora"
	}
	return domain
}
