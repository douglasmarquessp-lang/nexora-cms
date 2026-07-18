package articlepipeline

import (
	"testing"
)

func TestAllStages_Complete(t *testing.T) {
	expected := 11
	if len(AllStages) != expected {
		t.Errorf("expected %d stages, got %d", expected, len(AllStages))
	}

	seen := make(map[StageName]bool)
	for _, s := range AllStages {
		if seen[s] {
			t.Errorf("duplicate stage: %s", s)
		}
		seen[s] = true
		if _, ok := StageDisplayNames[s]; !ok {
			t.Errorf("missing display name for stage: %s", s)
		}
		if _, ok := StageDependencies[s]; !ok {
			t.Errorf("missing dependencies for stage: %s", s)
		}
	}
}

func TestStageDependencies_Chain(t *testing.T) {
	for i, stage := range AllStages {
		if i == 0 {
			if len(StageDependencies[stage]) != 0 {
				t.Errorf("first stage %s should have no dependencies", stage)
			}
		} else {
			if len(StageDependencies[stage]) == 0 {
				t.Errorf("stage %s should have at least one dependency", stage)
			}
			for _, dep := range StageDependencies[stage] {
				found := false
				for _, s := range AllStages {
					if s == dep {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("dependency %s for stage %s is not a valid stage", dep, stage)
				}
			}
		}
	}
}

func TestPipelineStatuses_AllValid(t *testing.T) {
	valid := map[PipelineStatus]bool{
		PipelineDraft: true, PipelinePending: true, PipelineRunning: true,
		PipelinePaused: true, PipelineCompleted: true, PipelineFailed: true,
		PipelineCancelled: true, PipelineRetrying: true,
	}

	if len(AllPipelineStatuses) != len(valid) {
		t.Errorf("expected %d statuses, got %d", len(valid), len(AllPipelineStatuses))
	}

	for _, s := range AllPipelineStatuses {
		if !valid[s] {
			t.Errorf("unexpected pipeline status: %s", s)
		}
	}
}

func TestStepStatuses_AllValid(t *testing.T) {
	valid := map[StepStatus]bool{
		StepStatusPending: true, StepStatusRunning: true, StepStatusCompleted: true,
		StepStatusFailed: true, StepStatusSkipped: true, StepStatusCancelled: true,
	}

	if len(AllStepStatuses) != len(valid) {
		t.Errorf("expected %d step statuses, got %d", len(valid), len(AllStepStatuses))
	}

	for _, s := range AllStepStatuses {
		if !valid[s] {
			t.Errorf("unexpected step status: %s", s)
		}
	}
}

func TestEventTypes_NotEmpty(t *testing.T) {
	events := []string{
		string(EventPipelineCreated), string(EventPipelineStarted),
		string(EventPipelineProgress), string(EventPipelinePaused),
		string(EventPipelineResumed), string(EventPipelineCompleted),
		string(EventPipelineFailed), string(EventPipelineCancelled),
		string(EventPipelineRetry), string(EventPipelineRestarted),
		string(EventStageStarted), string(EventStageCompleted),
		string(EventStageFailed), string(EventQualityPassed),
		string(EventQualityFailed), string(EventCandidateCreated),
	}
	for _, e := range events {
		if e == "" {
			t.Error("empty event type")
		}
	}
}

func TestSentinelErrors_NotEmpty(t *testing.T) {
	errs := []error{
		ErrJobNotFound, ErrStageNotFound, ErrJobAlreadyRunning,
		ErrJobAlreadyCompleted, ErrJobAlreadyCancelled, ErrJobNotRunning,
		ErrJobNotPaused, ErrStageNotPending, ErrStageAlreadyCompleted,
		ErrInvalidTitle, ErrInvalidLanguage, ErrInvalidPriority,
		ErrDatabaseNotAvail, ErrDependencyFailed, ErrDependencyPending,
		ErrMaxRetriesExceeded, ErrCandidateNotFound, ErrMetricNotFound,
		ErrQualityReportNotFound,
	}
	for _, err := range errs {
		if err.Error() == "" {
			t.Error("empty error message")
		}
	}
}

func TestNextStageName(t *testing.T) {
	for i, stage := range AllStages {
		next := nextStageName(string(stage))
		if i+1 < len(AllStages) {
			if next != string(AllStages[i+1]) {
				t.Errorf("next of %s: expected %s, got %s", stage, AllStages[i+1], next)
			}
		} else {
			if next != "" {
				t.Errorf("next of last stage %s: expected empty, got %s", stage, next)
			}
		}
	}
}

func TestNextStageName_Unknown(t *testing.T) {
	if next := nextStageName("unknown"); next != "" {
		t.Errorf("expected empty for unknown stage, got %s", next)
	}
}

func TestQualityStatuses(t *testing.T) {
	if QualityPending != "pending" {
		t.Errorf("expected pending, got %s", QualityPending)
	}
	if QualityPassed != "passed" {
		t.Errorf("expected passed, got %s", QualityPassed)
	}
	if QualityFailed != "failed" {
		t.Errorf("expected failed, got %s", QualityFailed)
	}
	if QualityWarning != "warning" {
		t.Errorf("expected warning, got %s", QualityWarning)
	}
}

func TestCandidateStatuses(t *testing.T) {
	if CandidateDraft != "draft" {
		t.Errorf("expected draft, got %s", CandidateDraft)
	}
	if CandidateApproved != "approved" {
		t.Errorf("expected approved, got %s", CandidateApproved)
	}
	if CandidatePublished != "published" {
		t.Errorf("expected published, got %s", CandidatePublished)
	}
	if CandidateRejected != "rejected" {
		t.Errorf("expected rejected, got %s", CandidateRejected)
	}
}
