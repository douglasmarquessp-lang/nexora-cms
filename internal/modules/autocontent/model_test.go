package autocontent

import (
	"testing"

	"nexora/internal/kernel"
)

func TestJobStatus_Valid(t *testing.T) {
	tests := []struct {
		status JobStatus
		valid  bool
	}{
		{JobStatusDraft, true},
		{JobStatusPending, true},
		{JobStatusRunning, true},
		{JobStatusPaused, true},
		{JobStatusCompleted, true},
		{JobStatusFailed, true},
		{JobStatusCancelled, true},
		{JobStatus("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.status {
		case JobStatusDraft, JobStatusPending, JobStatusRunning,
			JobStatusPaused, JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("JobStatus(%s) valid = %v, want %v", tt.status, valid, tt.valid)
		}
	}
}

func TestStepStatus_Valid(t *testing.T) {
	tests := []struct {
		status StepStatus
		valid  bool
	}{
		{StepStatusPending, true},
		{StepStatusRunning, true},
		{StepStatusCompleted, true},
		{StepStatusFailed, true},
		{StepStatusSkipped, true},
		{StepStatusCancelled, true},
		{StepStatus("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.status {
		case StepStatusPending, StepStatusRunning, StepStatusCompleted,
			StepStatusFailed, StepStatusSkipped, StepStatusCancelled:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("StepStatus(%s) valid = %v, want %v", tt.status, valid, tt.valid)
		}
	}
}

func TestQueueStatus_Valid(t *testing.T) {
	tests := []struct {
		status QueueStatus
		valid  bool
	}{
		{QueuePending, true},
		{QueueApproved, true},
		{QueuePublished, true},
		{QueueFailed, true},
		{QueueRejected, true},
		{QueueStatus("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.status {
		case QueuePending, QueueApproved, QueuePublished, QueueFailed, QueueRejected:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("QueueStatus(%s) valid = %v, want %v", tt.status, valid, tt.valid)
		}
	}
}

func TestAllWorkflowSteps_Length(t *testing.T) {
	if len(AllWorkflowSteps) != 14 {
		t.Errorf("expected 14 workflow steps, got %d", len(AllWorkflowSteps))
	}

	expected := []WorkflowStep{
		StepTopic, StepResearch, StepBriefing, StepOutline, StepDraft,
		StepHumanRewrite, StepSEOOptimization, StepFactCheck, StepReadability,
		StepInternalLinking, StepMetadata, StepTranslation, StepFeaturedImage,
		StepReadyForPub,
	}
	for i, s := range AllWorkflowSteps {
		if s != expected[i] {
			t.Errorf("AllWorkflowSteps[%d] = %s, want %s", i, s, expected[i])
		}
	}
}

func TestStepDependencies(t *testing.T) {
	if len(StepDependencies) != 14 {
		t.Errorf("expected 14 step dependency entries, got %d", len(StepDependencies))
	}

	if len(StepDependencies[StepTopic]) != 0 {
		t.Error("topic should have no dependencies")
	}
	if len(StepDependencies[StepResearch]) != 1 || StepDependencies[StepResearch][0] != StepTopic {
		t.Error("research should depend on topic")
	}
	if len(StepDependencies[StepReadyForPub]) != 1 || StepDependencies[StepReadyForPub][0] != StepFeaturedImage {
		t.Error("ready_for_publication should depend on featured_image")
	}
}

func TestStepDisplayNames(t *testing.T) {
	for _, step := range AllWorkflowSteps {
		name, ok := StepDisplayNames[step]
		if !ok {
			t.Errorf("missing display name for step %s", step)
		}
		if name == "" {
			t.Errorf("empty display name for step %s", step)
		}
	}
}

func TestSentinelErrors(t *testing.T) {
	errs := []error{
		ErrJobNotFound,
		ErrJobAlreadyRunning,
		ErrJobAlreadyCompleted,
		ErrJobAlreadyCancelled,
		ErrJobNotRunning,
		ErrStepNotFound,
		ErrStepAlreadyCompleted,
		ErrDependencyFailed,
		ErrDependencyPending,
		ErrInvalidTopic,
		ErrDatabaseNotAvail,
		ErrMaxRetriesExceeded,
		ErrQueueItemNotFound,
		ErrTemplateNotFound,
		ErrResultNotFound,
		ErrInvalidLanguage,
		ErrInvalidStep,
		ErrJobPaused,
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
		{EventAutoCreated, "EventAutoCreated"},
		{EventAutoStarted, "EventAutoStarted"},
		{EventAutoProgress, "EventAutoProgress"},
		{EventAutoPaused, "EventAutoPaused"},
		{EventAutoResumed, "EventAutoResumed"},
		{EventAutoCompleted, "EventAutoCompleted"},
		{EventAutoFailed, "EventAutoFailed"},
		{EventAutoCancelled, "EventAutoCancelled"},
		{EventAutoRetry, "EventAutoRetry"},
		{EventAutoStepStarted, "EventAutoStepStarted"},
		{EventAutoStepCompleted, "EventAutoStepCompleted"},
		{EventAutoStepFailed, "EventAutoStepFailed"},
		{EventAutoQueued, "EventAutoQueued"},
	}

	for _, e := range events {
		if e.event == "" {
			t.Errorf("%s should not be empty", e.name)
		}
	}
}

func TestWorkflowStepEnum(t *testing.T) {
	tests := []struct {
		step WorkflowStep
		name string
	}{
		{StepTopic, "topic"},
		{StepResearch, "research"},
		{StepBriefing, "briefing"},
		{StepOutline, "outline"},
		{StepDraft, "draft"},
		{StepHumanRewrite, "human_rewrite"},
		{StepSEOOptimization, "seo_optimization"},
		{StepFactCheck, "fact_check"},
		{StepReadability, "readability"},
		{StepInternalLinking, "internal_linking"},
		{StepMetadata, "metadata"},
		{StepTranslation, "translation"},
		{StepFeaturedImage, "featured_image"},
		{StepReadyForPub, "ready_for_publication"},
	}

	for _, tt := range tests {
		if string(tt.step) != tt.name {
			t.Errorf("WorkflowStep(%s) = %s, want %s", tt.name, string(tt.step), tt.name)
		}
	}
}
