package seoengine

import (
	"testing"

	"nexora/internal/kernel"
)

func TestProjectStatus_Valid(t *testing.T) {
	tests := []struct {
		status ProjectStatus
		valid  bool
	}{
		{ProjectStatusDraft, true},
		{ProjectStatusPending, true},
		{ProjectStatusRunning, true},
		{ProjectStatusCompleted, true},
		{ProjectStatusFailed, true},
		{ProjectStatus("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.status {
		case ProjectStatusDraft, ProjectStatusPending, ProjectStatusRunning,
			ProjectStatusCompleted, ProjectStatusFailed:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("ProjectStatus(%s) valid = %v, want %v", tt.status, valid, tt.valid)
		}
	}
}

func TestImprovementStatus_Valid(t *testing.T) {
	tests := []struct {
		status ImprovementStatus
		valid  bool
	}{
		{ImprovementPending, true},
		{ImprovementApplied, true},
		{ImprovementDismissed, true},
		{ImprovementStatus("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.status {
		case ImprovementPending, ImprovementApplied, ImprovementDismissed:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("ImprovementStatus(%s) valid = %v, want %v", tt.status, valid, tt.valid)
		}
	}
}

func TestImprovementPriority_Valid(t *testing.T) {
	tests := []struct {
		priority ImprovementPriority
		valid    bool
	}{
		{PriorityCritical, true},
		{PriorityHigh, true},
		{PriorityMedium, true},
		{PriorityLow, true},
		{ImprovementPriority("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.priority {
		case PriorityCritical, PriorityHigh, PriorityMedium, PriorityLow:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("ImprovementPriority(%s) valid = %v, want %v", tt.priority, valid, tt.valid)
		}
	}
}

func TestImprovementCategory_Valid(t *testing.T) {
	tests := []struct {
		category ImprovementCategory
		valid    bool
	}{
		{CategoryTitle, true},
		{CategoryMeta, true},
		{CategorySlug, true},
		{CategoryHeading, true},
		{CategoryImage, true},
		{CategorySchema, true},
		{CategoryLink, true},
		{CategoryReadability, true},
		{CategoryEEAT, true},
		{CategoryFreshness, true},
		{CategoryDuplicate, true},
		{CategoryCannibalization, true},
		{CategoryGap, true},
		{CategoryOrphan, true},
		{ImprovementCategory("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		for _, c := range AllCategories {
			if tt.category == c {
				valid = true
				break
			}
		}
		if valid != tt.valid {
			t.Errorf("ImprovementCategory(%s) valid = %v, want %v", tt.category, valid, tt.valid)
		}
	}
}

func TestAllCategories_Length(t *testing.T) {
	if len(AllCategories) != 14 {
		t.Errorf("expected 14 categories, got %d", len(AllCategories))
	}

	expected := []ImprovementCategory{
		CategoryTitle, CategoryMeta, CategorySlug, CategoryHeading,
		CategoryImage, CategorySchema, CategoryLink, CategoryReadability,
		CategoryEEAT, CategoryFreshness, CategoryDuplicate,
		CategoryCannibalization, CategoryGap, CategoryOrphan,
	}
	for i, c := range AllCategories {
		if c != expected[i] {
			t.Errorf("AllCategories[%d] = %s, want %s", i, c, expected[i])
		}
	}
}

func TestSentinelErrors(t *testing.T) {
	errs := []error{
		ErrProjectNotFound,
		ErrKeywordNotFound,
		ErrClusterNotFound,
		ErrAuditNotFound,
		ErrScoreNotFound,
		ErrImprovementNotFound,
		ErrDatabaseNotAvail,
		ErrInvalidLanguage,
		ErrInvalidCategory,
		ErrInvalidPriority,
		ErrInvalidStatus,
		ErrInvalidProjectStatus,
		ErrInvalidContentType,
	}

	for _, err := range errs {
		if err == nil {
			t.Error("sentinel error should not be nil")
		}
	}
}

func TestEventTypes(t *testing.T) {
	events := []struct {
		event kernel.EventType
		name  string
	}{
		{EventSEOProjectCreated, "EventSEOProjectCreated"},
		{EventSEOProjectStarted, "EventSEOProjectStarted"},
		{EventSEOProjectCompleted, "EventSEOProjectCompleted"},
		{EventSEOProjectFailed, "EventSEOProjectFailed"},
		{EventSEOAuditCompleted, "EventSEOAuditCompleted"},
		{EventSEOImprovementAdded, "EventSEOImprovementAdded"},
		{EventSEOImprovementApplied, "EventSEOImprovementApplied"},
	}

	for _, e := range events {
		if e.event == "" {
			t.Errorf("%s should not be empty", e.name)
		}
	}
}

func TestImprovementCategoryString(t *testing.T) {
	tests := []struct {
		category ImprovementCategory
		expected string
	}{
		{CategoryTitle, "title"},
		{CategoryMeta, "meta_description"},
		{CategorySlug, "slug"},
		{CategoryHeading, "heading"},
		{CategoryImage, "image_alt"},
		{CategorySchema, "schema"},
		{CategoryLink, "internal_link"},
		{CategoryReadability, "readability"},
		{CategoryEEAT, "eeat"},
		{CategoryFreshness, "freshness"},
		{CategoryDuplicate, "duplicate"},
		{CategoryCannibalization, "cannibalization"},
		{CategoryGap, "content_gap"},
		{CategoryOrphan, "orphan"},
	}

	for _, tt := range tests {
		if string(tt.category) != tt.expected {
			t.Errorf("ImprovementCategory(%s) = %s, want %s", tt.expected, string(tt.category), tt.expected)
		}
	}
}
