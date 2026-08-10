package sitelang

import (
	"testing"

	"github.com/google/uuid"
)

const aiworkSimpleID = "a64d7d72-b97f-4f31-96fd-8aeb15f6184c"

func TestResolve_PinnedSiteAlwaysEnglish(t *testing.T) {
	site := uuid.MustParse(aiworkSimpleID)
	cases := []struct{ requested string }{
		{""}, {"pt"}, {"PT"}, {"en"}, {"EN"}, {"fr"},
	}
	for _, c := range cases {
		if got := Resolve(site, c.requested); got != "en" {
			t.Errorf("Resolve(pinned, %q) = %q, want en", c.requested, got)
		}
	}
}

func TestResolve_UnpinnedSiteHonorsValidRequest(t *testing.T) {
	site := uuid.New()
	if got := Resolve(site, "pt"); got != "pt" {
		t.Errorf("Resolve(site, pt) = %q, want pt", got)
	}
	if got := Resolve(site, "en"); got != "en" {
		t.Errorf("Resolve(site, en) = %q, want en", got)
	}
	if got := Resolve(site, "EN"); got != "en" {
		// case-insensitive normalization on the generic path
		t.Errorf("Resolve(site, EN) = %q, want en", got)
	}
}

func TestResolve_DefaultsToPortuguese(t *testing.T) {
	site := uuid.New()
	if got := Resolve(site, ""); got != "pt" {
		t.Errorf("Resolve(site, empty) = %q, want pt", got)
	}
}

func TestResolve_UnknownLanguagePassedThroughForCallerValidation(t *testing.T) {
	site := uuid.New()
	// Unknown values are NOT coerced: callers keep ErrInvalidLanguage.
	if got := Resolve(site, "fr"); got != "fr" {
		t.Errorf("Resolve(site, fr) = %q, want fr", got)
	}
	// ...but a pinned site always resolves, even for unknown requests.
	if got := Resolve(uuid.MustParse(aiworkSimpleID), "fr"); got != "en" {
		t.Errorf("Resolve(aiwork, fr) = %q, want en (pin wins)", got)
	}
}

func TestResolve_OtherSitesUnaffected(t *testing.T) {
	// A different site with an "en" request must stay "en" (no accidental
	// cross-site pinning) and the pinned site must never receive "pt".
	other := uuid.New()
	if got := Resolve(other, "en"); got != "en" {
		t.Errorf("Resolve(other, en) = %q, want en", got)
	}
	if got := Resolve(uuid.MustParse(aiworkSimpleID), "pt"); got != "en" {
		t.Errorf("Resolve(aiwork, pt) = %q, want en (pin wins)", got)
	}
}
