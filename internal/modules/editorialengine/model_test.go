package editorialengine

import (
	"errors"
	"testing"
)

func TestModuleName(t *testing.T) {
	if ModuleName != "editorialengine" {
		t.Errorf("expected 'editorialengine', got %q", ModuleName)
	}
}

func TestValidStages(t *testing.T) {
	expected := []PipelineStage{
		StageResearch, StageBriefing, StageOutline, StageWriting,
		StageReview, StageSEO, StageTranslation, StagePublish,
	}
	if len(ValidStages) != len(expected) {
		t.Errorf("expected %d stages, got %d", len(expected), len(ValidStages))
	}
	for i, s := range expected {
		if ValidStages[i] != s {
			t.Errorf("stage[%d]: expected %q, got %q", i, s, ValidStages[i])
		}
	}
}

func TestStageConstants(t *testing.T) {
	tests := []struct {
		name  string
		value StageStatus
	}{
		{"pending", StageStatusPending},
		{"in_progress", StageStatusInProgress},
		{"completed", StageStatusCompleted},
		{"failed", StageStatusFailed},
		{"blocked", StageStatusBlocked},
	}
	for _, tt := range tests {
		if string(tt.value) != tt.name {
			t.Errorf("expected %q, got %q", tt.name, string(tt.value))
		}
	}
}

func TestTranslationConstants(t *testing.T) {
	tests := []struct {
		name  string
		value TranslationStatus
	}{
		{"pending", TransStatusPending},
		{"in_progress", TransStatusInProgress},
		{"completed", TransStatusCompleted},
		{"failed", TransStatusFailed},
	}
	for _, tt := range tests {
		if string(tt.value) != tt.name {
			t.Errorf("expected %q, got %q", tt.name, string(tt.value))
		}
	}
}

func TestEventConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"editorial.started", string(EventEditorialStarted)},
		{"editorial.reviewed", string(EventEditorialReviewed)},
		{"editorial.scored", string(EventEditorialScored)},
		{"editorial.translated", string(EventEditorialTranslated)},
		{"editorial.completed", string(EventEditorialCompleted)},
		{"style.updated", string(EventStyleUpdated)},
		{"seo.generated", string(EventSEOGenerated)},
		{"quality.checked", string(EventQualityChecked)},
	}
	for _, tt := range tests {
		if tt.value != tt.name {
			t.Errorf("expected %q, got %q", tt.name, tt.value)
		}
	}
}

func TestErrorConstants(t *testing.T) {
	if ErrPipelineNotFound == nil {
		t.Error("ErrPipelineNotFound should not be nil")
	}
	if ErrStageNotFound == nil {
		t.Error("ErrStageNotFound should not be nil")
	}
	if ErrStyleRulesNotFound == nil {
		t.Error("ErrStyleRulesNotFound should not be nil")
	}
	if ErrSEONotFound == nil {
		t.Error("ErrSEONotFound should not be nil")
	}
	if ErrQualityNotFound == nil {
		t.Error("ErrQualityNotFound should not be nil")
	}
	if ErrTranslationNotFound == nil {
		t.Error("ErrTranslationNotFound should not be nil")
	}
	if ErrPromptDataNotFound == nil {
		t.Error("ErrPromptDataNotFound should not be nil")
	}
	if ErrDatabaseNotAvail == nil {
		t.Error("ErrDatabaseNotAvail should not be nil")
	}
	if ErrInvalidStage == nil {
		t.Error("ErrInvalidStage should not be nil")
	}
	if ErrInvalidStageStatus == nil {
		t.Error("ErrInvalidStageStatus should not be nil")
	}
	if ErrInvalidTranslationDir == nil {
		t.Error("ErrInvalidTranslationDir should not be nil")
	}
	if ErrJobAlreadyInPipeline == nil {
		t.Error("ErrJobAlreadyInPipeline should not be nil")
	}
}

func TestErrorUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	errs := []error{
		ErrPipelineNotFound,
		ErrStageNotFound,
		ErrStyleRulesNotFound,
		ErrSEONotFound,
		ErrQualityNotFound,
		ErrTranslationNotFound,
		ErrPromptDataNotFound,
		ErrDatabaseNotAvail,
		ErrInvalidStage,
		ErrInvalidStageStatus,
		ErrInvalidTranslationDir,
		ErrJobAlreadyInPipeline,
	}
	for _, err := range errs {
		if seen[err.Error()] {
			t.Errorf("duplicate error message: %q", err.Error())
		}
		seen[err.Error()] = true
		if !errors.Is(err, err) {
			t.Errorf("error %v should wrap itself", err)
		}
	}
}
