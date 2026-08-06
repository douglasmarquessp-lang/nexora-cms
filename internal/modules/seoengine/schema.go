package seoengine

import (
	"encoding/json"
)

// Schema builders produce valid JSON-LD rich-snippet markup for the 7 standard
// schema types. Deterministic given the inputs.

// BuildArticleSchemaJSONLD returns an Article JSON-LD object as raw JSON.
func BuildArticleSchemaJSONLD(url, title, headline, description, image, author, publisherName, publisherLogo, datePublished, dateModified, language string) (string, error) {
	article := map[string]interface{}{
		"@context":      "https://schema.org",
		"@type":         "Article",
		"mainEntityOfPage": map[string]interface{}{"@type": "WebPage", "@id": url},
		"headline":      headline,
		"description":   description,
		"image":         image,
		"datePublished": datePublished,
		"inLanguage":    language,
	}
	if author != "" {
		article["author"] = map[string]interface{}{"@type": "Person", "name": author}
	}
	if publisherName != "" {
		publisher := map[string]interface{}{"@type": "Organization", "name": publisherName}
		if publisherLogo != "" {
			publisher["logo"] = map[string]interface{}{"@type": "ImageObject", "url": publisherLogo}
		}
		article["publisher"] = publisher
	}
	if dateModified != "" {
		article["dateModified"] = dateModified
	}
	return renderSchema(article)
}

// BuildNewsArticleSchema renders a NewsArticle JSON-LD.
func BuildNewsArticleSchema(url, headline, description, image, author, publisherName string, dates ...string) (string, error) {
	news := map[string]interface{}{
		"@context":      "https://schema.org",
		"@type":         "NewsArticle",
		"mainEntityOfPage": map[string]interface{}{"@type": "WebPage", "@id": url},
		"headline":      headline,
		"description":   description,
		"image":         image,
	}
	if author != "" {
		news["author"] = map[string]interface{}{"@type": "Person", "name": author}
	}
	if len(dates) > 0 && dates[0] != "" {
		news["datePublished"] = dates[0]
	}
	if len(dates) > 1 {
		news["dateModified"] = dates[1]
	}
	return renderSchema(news)
}

// FAQQuestion is a single Q/A pair for FAQPage schema.
type FAQQuestion struct {
	Question string `json:"q"`
	Answer   string `json:"a"`
}

// BuildFAQSchema renders an FAQPage JSON-LD from Q/A pairs found in content.
func BuildFAQSchema(url string, faqs []FAQQuestion) (string, error) {
	mainEntity := make([]map[string]interface{}, 0, len(faqs))
	for _, f := range faqs {
		if f.Question == "" || f.Answer == "" {
			continue
		}
		mainEntity = append(mainEntity, map[string]interface{}{
			"@type":          "Question",
			"name":           f.Question,
			"acceptedAnswer": map[string]interface{}{"@type": "Answer", "text": f.Answer},
		})
	}
	if len(mainEntity) == 0 {
		return "", nil
	}
	return renderSchema(map[string]interface{}{
		"@context":   "https://schema.org",
		"@type":      "FAQPage",
		"mainEntity": mainEntity,
	})
}

// HowToStep is one step in a HowTo schema.
type HowToStep struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	Image      string `json:"image,omitempty"`
	Duration   string `json:"duration,omitempty"`
}

// BuildHowToSchema renders a HowTo JSON-LD.
func BuildHowToSchema(name, description string, steps []HowToStep) (string, error) {
	stepList := make([]map[string]interface{}, 0, len(steps))
	for i, st := range steps {
		position := i + 1
		step := map[string]interface{}{
			"@type":    "HowToStep",
			"position": position,
			"name":     st.Title,
			"text":     st.Text,
		}
		if st.Image != "" {
			step["image"] = st.Image
		}
		if st.Duration != "" {
			step["duration"] = st.Duration
		}
		stepList = append(stepList, step)
	}
	return renderSchema(map[string]interface{}{
		"@context":        "https://schema.org",
		"@type":           "HowTo",
		"name":            name,
		"description":     description,
		"step":            stepList,
	})
}

// BreadcrumbItem is one hierarchy entry.
type BreadcrumbItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// BuildBreadcrumbSchema renders a BreadcrumbList JSON-LD.
func BuildBreadcrumbSchema(items []BreadcrumbItem) (string, error) {
	itemList := make([]map[string]interface{}, 0, len(items))
	for i, it := range items {
		itemList = append(itemList, map[string]interface{}{
			"@type":    "ListItem",
			"position": i + 1,
			"name":     it.Name,
			"item":     it.URL,
		})
	}
	return renderSchema(map[string]interface{}{
		"@context":        "https://schema.org",
		"@type":           "BreadcrumbList",
		"itemListElement": itemList,
	})
}

// OrganizationData builds the site-level Organization schema (typically on the
// homepage).
func BuildOrganizationSchema(name, url, logo, description string, sameAs []string) (string, error) {
	org := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "Organization",
		"name":        name,
		"url":         url,
		"description": description,
		"logo":        logo,
	}
	if len(sameAs) > 0 {
		org["sameAs"] = sameAs
	}
	return renderSchema(org)
}

// BuildWebSiteSchema renders a WebSite JSON-LD (site-level, usually homepage).
func BuildWebSiteSchema(name, url, searchURL, language string) (string, error) {
	site := map[string]interface{}{
		"@context":   "https://schema.org",
		"@type":      "WebSite",
		"name":       name,
		"url":        url,
		"publisher":  map[string]interface{}{"@type": "Organization", "name": name},
		"inLanguage": language,
	}
	if searchURL != "" {
		site["potentialAction"] = map[string]interface{}{
			"@type":       "SearchAction",
			"target":      map[string]interface{}{"@type": "EntryPoint", "urlTemplate": searchURL},
			"query-input": "required name=search_term_string",
		}
	}
	return renderSchema(site)
}

func renderSchema(obj map[string]interface{}) (string, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}