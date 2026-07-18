package publisher

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

var (
	slugRegex       = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	alphanumRegex   = regexp.MustCompile(`[^a-z0-9]+`)
	multiDashRegex  = regexp.MustCompile(`-{2,}`)
	edgeDashRegex   = regexp.MustCompile(`^-|-$`)
	schemeRegex     = regexp.MustCompile(`^(https?:\/\/)`)
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) ValidateSlug(slug string) (string, error) {
	s := strings.TrimSpace(slug)
	if s == "" {
		return "", ErrInvalidSlug
	}

	s = strings.ToLower(s)
	s = alphanumRegex.ReplaceAllString(s, "-")
	s = multiDashRegex.ReplaceAllString(s, "-")
	s = edgeDashRegex.ReplaceAllString(s, "")

	if len(s) < 2 || len(s) > 200 {
		return "", ErrInvalidSlug
	}

	if !slugRegex.MatchString(s) {
		return "", ErrInvalidSlug
	}

	return s, nil
}

func (v *Validator) GenerateSlug(title string) string {
	s := strings.TrimSpace(title)
	s = strings.ToLower(s)
	s = alphanumRegex.ReplaceAllString(s, "-")
	s = multiDashRegex.ReplaceAllString(s, "-")
	s = edgeDashRegex.ReplaceAllString(s, "")

	if len(s) < 2 {
		return "untitled"
	}
	if len(s) > 200 {
		s = s[:200]
		s = edgeDashRegex.ReplaceAllString(s, "")
	}

	return s
}

func (v *Validator) GenerateURL(slug, language string, siteDomain string) string {
	base := strings.TrimRight(siteDomain, "/")
	lang := strings.ToLower(language)
	if lang == "" || lang == "pt" {
		return fmt.Sprintf("%s/%s", base, slug)
	}
	return fmt.Sprintf("%s/%s/%s", base, lang, slug)
}

func (v *Validator) GenerateCanonicalURL(slug, language, primaryLanguage, siteDomain string) string {
	base := strings.TrimRight(siteDomain, "/")
	lang := strings.ToLower(language)
	prim := strings.ToLower(primaryLanguage)

	if lang == prim || lang == "" {
		return fmt.Sprintf("%s/%s", base, slug)
	}
	return fmt.Sprintf("%s/%s/%s", base, lang, slug)
}

func (v *Validator) ComputeChecksum(pub *Publication) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%d",
		pub.Title, pub.Content, pub.Slug, pSliceToString(pub.Tags), pSliceToString(pub.Categories), pub.Revision)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

func (v *Validator) ValidateLanguage(lang string) error {
	if lang == "" {
		return nil
	}
	l := strings.ToLower(lang)
	if l != "pt" && l != "en" {
		return ErrInvalidLanguage
	}
	return nil
}

func (v *Validator) ValidateVisibility(vis Visibility) error {
	switch vis {
	case VisibilityPublic, VisibilityPrivate, VisibilityPassword:
		return nil
	case "":
		return nil
	default:
		return ErrInvalidVisibility
	}
}

func (v *Validator) ValidateRecurrence(r string) error {
	if r == "" {
		return nil
	}
	valid := map[string]bool{
		"daily": true, "weekly": true, "monthly": true, "yearly": true,
		"weekdays": true, "custom": true,
	}
	if !valid[r] {
		return ErrInvalidRecurrence
	}
	return nil
}

func (v *Validator) SanitizeURL(url string) string {
	s := strings.TrimSpace(url)
	if s == "" {
		return ""
	}
	if !schemeRegex.MatchString(s) {
		s = "https://" + s
	}
	return strings.TrimRight(s, "/")
}

func pSliceToString(s []string) string {
	return strings.Join(s, ",")
}
