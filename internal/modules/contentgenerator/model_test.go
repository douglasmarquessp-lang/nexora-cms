package contentgenerator

import (
	"testing"

	"nexora/internal/kernel"
)

func TestGenStatus_Valid(t *testing.T) {
	tests := []struct {
		status GenStatus
		valid  bool
	}{
		{GenStatusPending, true},
		{GenStatusRunning, true},
		{GenStatusPaused, true},
		{GenStatusCompleted, true},
		{GenStatusFailed, true},
		{GenStatusCancelled, true},
		{GenStatusRetrying, true},
		{GenStatus("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.status {
		case GenStatusPending, GenStatusRunning, GenStatusPaused,
			GenStatusCompleted, GenStatusFailed, GenStatusCancelled, GenStatusRetrying:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("GenStatus(%s) valid = %v, want %v", tt.status, valid, tt.valid)
		}
	}
}

func TestStageStatus_Valid(t *testing.T) {
	tests := []struct {
		status StageStatus
		valid  bool
	}{
		{StageStatusPending, true},
		{StageStatusRunning, true},
		{StageStatusCompleted, true},
		{StageStatusFailed, true},
		{StageStatusSkipped, true},
		{StageStatus("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.status {
		case StageStatusPending, StageStatusRunning, StageStatusCompleted,
			StageStatusFailed, StageStatusSkipped:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("StageStatus(%s) valid = %v, want %v", tt.status, valid, tt.valid)
		}
	}
}

func TestValidStages(t *testing.T) {
	if len(ValidStages) != 9 {
		t.Errorf("expected 9 valid stages, got %d", len(ValidStages))
	}

	expected := []GenStage{
		GenStageResearch, GenStageBriefing, GenStageOutline, GenStageSectionGen,
		GenStageSEOOptimization, GenStageQualityReview, GenStageTranslation,
		GenStageFinalReview, GenStagePublishReady,
	}
	for i, s := range ValidStages {
		if s != expected[i] {
			t.Errorf("ValidStages[%d] = %s, want %s", i, s, expected[i])
		}
	}
}

func TestSentinelErrors(t *testing.T) {
	errs := []error{
		ErrJobNotFound,
		ErrStageNotFound,
		ErrJobAlreadyRunning,
		ErrJobAlreadyCompleted,
		ErrJobAlreadyCancelled,
		ErrJobNotRunning,
		ErrStageNotPending,
		ErrInvalidPriority,
		ErrInvalidLanguage,
		ErrDatabaseNotAvail,
		ErrMaxRetriesExceeded,
		ErrDependencyFailed,
		ErrQualityGateFailed,
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
		{EventGenStarted, "EventGenStarted"},
		{EventGenProgress, "EventGenProgress"},
		{EventGenCompleted, "EventGenCompleted"},
		{EventGenFailed, "EventGenFailed"},
		{EventGenRetry, "EventGenRetry"},
		{EventGenCancelled, "EventGenCancelled"},
		{EventGenReviewed, "EventGenReviewed"},
		{EventGenReady, "EventGenReady"},
	}

	for _, e := range events {
		if e.event == "" {
			t.Errorf("%s should not be empty", e.name)
		}
	}
}

func TestGenStageEnum(t *testing.T) {
	tests := []struct {
		stage GenStage
		name  string
	}{
		{GenStageResearch, "research"},
		{GenStageBriefing, "briefing"},
		{GenStageOutline, "outline"},
		{GenStageSectionGen, "section_generation"},
		{GenStageSEOOptimization, "seo_optimization"},
		{GenStageQualityReview, "quality_review"},
		{GenStageTranslation, "translation"},
		{GenStageFinalReview, "final_review"},
		{GenStagePublishReady, "publish_ready"},
	}

	for _, tt := range tests {
		if string(tt.stage) != tt.name {
			t.Errorf("GenStage(%s) = %s, want %s", tt.name, string(tt.stage), tt.name)
		}
	}
}
