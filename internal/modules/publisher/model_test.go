package publisher

import (
	"testing"

	"nexora/internal/kernel"
)

func TestPubStatus_Valid(t *testing.T) {
	tests := []struct {
		status PubStatus
		valid  bool
	}{
		{PubStatusDraft, true},
		{PubStatusPublished, true},
		{PubStatusScheduled, true},
		{PubStatusUnpublished, true},
		{PubStatusArchived, true},
		{PubStatusDeleted, true},
		{PubStatus("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.status {
		case PubStatusDraft, PubStatusPublished, PubStatusScheduled,
			PubStatusUnpublished, PubStatusArchived, PubStatusDeleted:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("PubStatus(%s) valid = %v, want %v", tt.status, valid, tt.valid)
		}
	}
}

func TestQueueStatus_Valid(t *testing.T) {
	tests := []struct {
		status QueueStatus
		valid  bool
	}{
		{QueuePending, true},
		{QueueRunning, true},
		{QueueCompleted, true},
		{QueueFailed, true},
		{QueueCancelled, true},
		{QueueStatus("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.status {
		case QueuePending, QueueRunning, QueueCompleted, QueueFailed, QueueCancelled:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("QueueStatus(%s) valid = %v, want %v", tt.status, valid, tt.valid)
		}
	}
}

func TestScheduleStatus_Valid(t *testing.T) {
	tests := []struct {
		status ScheduleStatus
		valid  bool
	}{
		{ScheduleScheduled, true},
		{ScheduleRunning, true},
		{ScheduleCompleted, true},
		{ScheduleCancelled, true},
		{ScheduleFailed, true},
		{ScheduleStatus("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.status {
		case ScheduleScheduled, ScheduleRunning, ScheduleCompleted, ScheduleCancelled, ScheduleFailed:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("ScheduleStatus(%s) valid = %v, want %v", tt.status, valid, tt.valid)
		}
	}
}

func TestQueueAction_Valid(t *testing.T) {
	tests := []struct {
		action QueueAction
		valid  bool
	}{
		{QueueActionPublish, true},
		{QueueActionUnpublish, true},
		{QueueActionRepublish, true},
		{QueueActionUpdate, true},
		{QueueAction("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.action {
		case QueueActionPublish, QueueActionUnpublish, QueueActionRepublish, QueueActionUpdate:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("QueueAction(%s) valid = %v, want %v", tt.action, valid, tt.valid)
		}
	}
}

func TestVisibility_Valid(t *testing.T) {
	tests := []struct {
		vis   Visibility
		valid bool
	}{
		{VisibilityPublic, true},
		{VisibilityPrivate, true},
		{VisibilityPassword, true},
		{Visibility("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.vis {
		case VisibilityPublic, VisibilityPrivate, VisibilityPassword:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("Visibility(%s) valid = %v, want %v", tt.vis, valid, tt.valid)
		}
	}
}

func TestHistoryAction_Valid(t *testing.T) {
	tests := []struct {
		action HistoryAction
		valid  bool
	}{
		{HistoryCreated, true},
		{HistoryPublished, true},
		{HistoryUpdated, true},
		{HistoryUnpublished, true},
		{HistoryRepublished, true},
		{HistoryScheduled, true},
		{HistoryCancelled, true},
		{HistoryArchived, true},
		{HistoryDeleted, true},
		{HistoryAction("invalid"), false},
	}

	for _, tt := range tests {
		valid := false
		switch tt.action {
		case HistoryCreated, HistoryPublished, HistoryUpdated, HistoryUnpublished,
			HistoryRepublished, HistoryScheduled, HistoryCancelled, HistoryArchived, HistoryDeleted:
			valid = true
		}
		if valid != tt.valid {
			t.Errorf("HistoryAction(%s) valid = %v, want %v", tt.action, valid, tt.valid)
		}
	}
}

func TestSentinelErrors(t *testing.T) {
	errs := []error{
		ErrPublicationNotFound,
		ErrDuplicateSlug,
		ErrInvalidSlug,
		ErrInvalidLanguage,
		ErrInvalidVisibility,
		ErrInvalidStatus,
		ErrInvalidAction,
		ErrInvalidRecurrence,
		ErrTitleRequired,
		ErrDatabaseNotAvail,
		ErrQueueItemNotFound,
		ErrScheduleNotFound,
		ErrScheduleAlreadyActive,
		ErrMaxRetriesExceeded,
		ErrPublicationAlreadyPublished,
		ErrPublicationNotPublished,
		ErrCannotModifyPublished,
		ErrHistoryNotFound,
		ErrMetricsNotFound,
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
		{EventPubCreated, "EventPubCreated"},
		{EventPubPublished, "EventPubPublished"},
		{EventPubUpdated, "EventPubUpdated"},
		{EventPubUnpublished, "EventPubUnpublished"},
		{EventPubRepublished, "EventPubRepublished"},
		{EventPubScheduled, "EventPubScheduled"},
		{EventPubCancelled, "EventPubCancelled"},
		{EventPubArchived, "EventPubArchived"},
		{EventPubDeleted, "EventPubDeleted"},
		{EventPubQueueAdded, "EventPubQueueAdded"},
		{EventPubQueueStarted, "EventPubQueueStarted"},
		{EventPubQueueCompleted, "EventPubQueueCompleted"},
		{EventPubQueueFailed, "EventPubQueueFailed"},
		{EventPubQueueRetried, "EventPubQueueRetried"},
		{EventPubSitemapUpdate, "EventPubSitemapUpdate"},
		{EventPubRSSUpdate, "EventPubRSSUpdate"},
		{EventPubRobotsRefresh, "EventPubRobotsRefresh"},
		{EventPubCachePurge, "EventPubCachePurge"},
	}

	for _, e := range events {
		if e.event == "" {
			t.Errorf("%s should not be empty", e.name)
		}
	}
}
