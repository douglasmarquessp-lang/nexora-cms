// Package sitelang centralizes the temporary per-site content language
// configuration. This is the single place where a site can be pinned to a
// specific language so that no generation or publishing pipeline ever
// produces content in the wrong language for that site.
//
// Entries here are site-specific on purpose: other sites keep the existing
// global behavior (explicit request honored, otherwise "pt").
package sitelang

import (
	"strings"

	"github.com/google/uuid"
)

// siteOverrides maps a site ID to its pinned content language. Add an entry
// here to force a site to always produce/publish in that language.
//
// AIWorkSimple (a64d7d72-b97f-4f31-96fd-8aeb15f6184c) is an English-only
// site: every publication must use language "en" regardless of what the
// caller requests, so no Portuguese content is ever published to it.
var siteOverrides = map[string]string{
	"a64d7d72-b97f-4f31-96fd-8aeb15f6184c": "en",
}

// Resolve returns the effective content language for a site. When the site
// has an override (pin), the override always wins — even over an explicit
// request — so an English-only site can never receive Portuguese content.
// Otherwise the request is normalized (lowercase) and an empty value falls
// back to the existing global default "pt"; any unknown value is returned
// as-is so callers keep their own ErrInvalidLanguage validation.
func Resolve(siteID uuid.UUID, requested string) string {
	if lang, ok := siteOverrides[siteID.String()]; ok {
		return lang
	}
	lang := strings.ToLower(requested)
	if lang == "" {
		return "pt"
	}
	return lang
}
