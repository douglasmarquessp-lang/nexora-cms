package writer

import (
	"testing"
)

func TestModuleName(t *testing.T) {
	if ModuleName != "writer" {
		t.Errorf("expected 'writer', got %q", ModuleName)
	}
}

func TestJobStatusConstants(t *testing.T) {
	tests := []struct {
		status   JobStatus
		expected string
	}{
		{JobStatusDraft, "draft"},
		{JobStatusWriting, "writing"},
		{JobStatusReview, "review"},
		{JobStatusApproved, "approved"},
		{JobStatusPublished, "published"},
		{JobStatusFailed, "failed"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
		}
	}
}

func TestSectionStatusConstants(t *testing.T) {
	tests := []struct {
		status   SectionStatus
		expected string
	}{
		{SectionStatusPending, "pending"},
		{SectionStatusWriting, "writing"},
		{SectionStatusCompleted, "completed"},
		{SectionStatusReview, "review"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
		}
	}
}

func TestStyleSlugConstants(t *testing.T) {
	tests := []struct {
		slug     StyleSlug
		expected string
	}{
		{StyleJournalistic, "journalistic"},
		{StyleTechnical, "technical"},
		{StyleTutorial, "tutorial"},
		{StyleReview, "review"},
		{StyleComparative, "comparative"},
		{StyleList, "list"},
		{StyleOpinion, "opinion"},
		{StyleCompleteGuide, "complete_guide"},
	}
	for _, tt := range tests {
		if string(tt.slug) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.slug))
		}
	}
}

func TestOutlineSectionTypeConstants(t *testing.T) {
	tests := []struct {
		t        OutlineSectionType
		expected string
	}{
		{OutlineH1, "h1"},
		{OutlineH2, "h2"},
		{OutlineH3, "h3"},
	}
	for _, tt := range tests {
		if string(tt.t) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.t))
		}
	}
}

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrWritingJobNotFound, "writing job not found"},
		{ErrOutlineNotFound, "outline not found"},
		{ErrSectionNotFound, "section not found"},
		{ErrVersionNotFound, "version not found"},
		{ErrStyleNotFound, "writing style not found"},
		{ErrDatabaseNotAvail, "database not available"},
		{ErrInvalidLanguage, "language must be 'pt' or 'en'"},
		{ErrHeadlineRequired, "headline is required"},
		{ErrJobNotEditable, "job is not editable in current status"},
	}
	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, tt.err.Error())
		}
	}
}

func TestEventConstants(t *testing.T) {
	events := []string{
		string(EventWriterJobCreated),
		string(EventWriterJobUpdated),
		string(EventWriterJobCompleted),
		string(EventWriterVersionCreated),
		string(EventWriterVersionRestored),
	}
	for _, e := range events {
		if e == "" {
			t.Error("expected non-empty event constant")
		}
	}
}
