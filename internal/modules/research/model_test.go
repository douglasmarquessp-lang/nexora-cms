package research

import (
	"testing"
)

func TestModuleName(t *testing.T) {
	if ModuleName != "research" {
		t.Errorf("expected 'research', got %q", ModuleName)
	}
}

func TestJobStatusConstants(t *testing.T) {
	tests := []struct {
		status   JobStatus
		expected string
	}{
		{JobStatusPending, "pending"},
		{JobStatusRunning, "running"},
		{JobStatusCompleted, "completed"},
		{JobStatusFailed, "failed"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
		}
	}
}

func TestEntityTypeConstants(t *testing.T) {
	tests := []struct {
		entityType EntityType
		expected   string
	}{
		{EntityTypeFact, "fact"},
		{EntityTypeStatistic, "statistic"},
		{EntityTypeCompany, "company"},
		{EntityTypePerson, "person"},
		{EntityTypeProduct, "product"},
		{EntityTypeKeyword, "keyword"},
	}
	for _, tt := range tests {
		if string(tt.entityType) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.entityType))
		}
	}
}

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrResearchJobNotFound, "research job not found"},
		{ErrResearchJobNotEditable, "research job is not editable"},
		{ErrBriefingNotFound, "briefing not found"},
		{ErrDatabaseNotAvail, "database not available"},
		{ErrInvalidLanguage, "language must be 'pt' or 'en'"},
		{ErrTopicRequired, "topic is required"},
	}
	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, tt.err.Error())
		}
	}
}

func TestEventConstants(t *testing.T) {
	events := []string{
		string(EventResearchCreated),
		string(EventResearchUpdated),
		string(EventResearchCompleted),
		string(EventResearchDeleted),
	}
	for _, e := range events {
		if e == "" {
			t.Error("expected non-empty event constant")
		}
	}
}
