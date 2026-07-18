package workflow

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
		{QueueStatusPending, true},
		{QueueStatusRunning, true},
		{QueueStatusPaused, true},
		{QueueStatusCompleted, true},
		{QueueStatusFailed, true},
		{QueueStatusCancelled, true},
		{QueueStatus("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.status {
		case QueueStatusPending, QueueStatusRunning, QueueStatusPaused,
			QueueStatusCompleted, QueueStatusFailed, QueueStatusCancelled:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("QueueStatus(%s) valid = %v, want %v", tt.status, valid, tt.valid)
		}
	}
}

func TestAllWorkflowSteps_Length(t *testing.T) {
	if len(AllWorkflowSteps) != 8 {
		t.Errorf("expected 8 workflow steps, got %d", len(AllWorkflowSteps))
	}

	expected := []WorkflowStep{
		StepResearch, StepWriter, StepHumanWriter, StepEditorialEngine,
		StepSEOEngine, StepQualityCheck, StepPublisher, StepFinished,
	}
	for i, s := range AllWorkflowSteps {
		if s != expected[i] {
			t.Errorf("AllWorkflowSteps[%d] = %s, want %s", i, s, expected[i])
		}
	}
}

func TestStepDependencies(t *testing.T) {
	if len(StepDependencies) != 8 {
		t.Errorf("expected 8 step dependency entries, got %d", len(StepDependencies))
	}

	if len(StepDependencies[StepResearch]) != 0 {
		t.Error("research should have no dependencies")
	}
	if len(StepDependencies[StepWriter]) != 1 || StepDependencies[StepWriter][0] != StepResearch {
		t.Error("writer should depend on research")
	}
	if len(StepDependencies[StepFinished]) != 1 || StepDependencies[StepFinished][0] != StepPublisher {
		t.Error("finished should depend on publisher")
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
		ErrJobPaused,
		ErrStepNotFound,
		ErrStepAlreadyCompleted,
		ErrDependencyFailed,
		ErrDependencyPending,
		ErrInvalidTitle,
		ErrInvalidLanguage,
		ErrInvalidPriority,
		ErrDatabaseNotAvail,
		ErrMaxRetriesExceeded,
		ErrQueueItemNotFound,
		ErrQueueItemPaused,
		ErrQueueItemRunning,
		ErrNotificationNotFound,
		ErrInvalidAction,
		ErrInvalidStep,
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
		{EventWorkflowCreated, "EventWorkflowCreated"},
		{EventWorkflowStarted, "EventWorkflowStarted"},
		{EventWorkflowProgress, "EventWorkflowProgress"},
		{EventWorkflowPaused, "EventWorkflowPaused"},
		{EventWorkflowResumed, "EventWorkflowResumed"},
		{EventWorkflowCompleted, "EventWorkflowCompleted"},
		{EventWorkflowFailed, "EventWorkflowFailed"},
		{EventWorkflowCancelled, "EventWorkflowCancelled"},
		{EventWorkflowRetry, "EventWorkflowRetry"},
		{EventWorkflowStepStarted, "EventWorkflowStepStarted"},
		{EventWorkflowStepCompleted, "EventWorkflowStepCompleted"},
		{EventWorkflowStepFailed, "EventWorkflowStepFailed"},
		{EventWorkflowQueued, "EventWorkflowQueued"},
		{EventWorkflowQueueProcessed, "EventWorkflowQueueProcessed"},
		{EventWorkflowQueueStalled, "EventWorkflowQueueStalled"},
		{EventWorkflowPublicationReady, "EventWorkflowPublicationReady"},
		{EventWorkflowQualityFailed, "EventWorkflowQualityFailed"},
		{EventWorkflowSEOFailed, "EventWorkflowSEOFailed"},
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
		{StepResearch, "research"},
		{StepWriter, "writer"},
		{StepHumanWriter, "human_writer"},
		{StepEditorialEngine, "editorial_engine"},
		{StepSEOEngine, "seo_engine"},
		{StepQualityCheck, "quality_check"},
		{StepPublisher, "publisher"},
		{StepFinished, "finished"},
	}

	for _, tt := range tests {
		if string(tt.step) != tt.name {
			t.Errorf("WorkflowStep(%s) = %s, want %s", tt.name, string(tt.step), tt.name)
		}
	}
}

func TestNotificationSeverity(t *testing.T) {
	tests := []struct {
		severity NotificationSeverity
		valid    bool
	}{
		{SeverityInfo, true},
		{SeverityWarning, true},
		{SeverityError, true},
		{SeverityCritical, true},
		{SeveritySuccess, true},
		{NotificationSeverity("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.severity {
		case SeverityInfo, SeverityWarning, SeverityError, SeverityCritical, SeveritySuccess:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("NotificationSeverity(%s) valid = %v, want %v", tt.severity, valid, tt.valid)
		}
	}
}

func TestCoalesceStr(t *testing.T) {
	if coalesceStr("hello", "fallback") != "hello" {
		t.Error("expected original string")
	}
	if coalesceStr("", "fallback") != "fallback" {
		t.Error("expected fallback string")
	}
}
