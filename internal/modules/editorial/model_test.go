package editorial

import (
	"testing"
)

func TestTaskStatusConstants(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusPending, "pending"},
		{TaskStatusInProgress, "in_progress"},
		{TaskStatusCompleted, "completed"},
		{TaskStatusCancelled, "cancelled"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
		}
	}
}

func TestTaskPriorityConstants(t *testing.T) {
	tests := []struct {
		priority TaskPriority
		expected string
	}{
		{TaskPriorityLow, "low"},
		{TaskPriorityMedium, "medium"},
		{TaskPriorityHigh, "high"},
		{TaskPriorityUrgent, "urgent"},
	}
	for _, tt := range tests {
		if string(tt.priority) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.priority))
		}
	}
}

func TestApprovalStatusConstants(t *testing.T) {
	tests := []struct {
		status   ApprovalStatus
		expected string
	}{
		{ApprovalStatusPending, "pending"},
		{ApprovalStatusApproved, "approved"},
		{ApprovalStatusRejected, "rejected"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, string(tt.status))
		}
	}
}

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrTaskNotFound, "task not found"},
		{ErrRevisionNotFound, "revision not found"},
		{ErrApprovalNotFound, "approval request not found"},
		{ErrCalendarEventNotFound, "calendar event not found"},
		{ErrWidgetNotFound, "widget not found"},
		{ErrDatabaseNotAvail, "database not available"},
	}
	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, tt.err.Error())
		}
	}
}

func TestEventConstants(t *testing.T) {
	_ = []string{
		string(EventTaskCreated),
		string(EventTaskUpdated),
		string(EventTaskDeleted),
		string(EventRevisionSaved),
		string(EventRevisionRestored),
		string(EventApprovalRequested),
		string(EventApprovalGranted),
		string(EventApprovalRejected),
		string(EventCalendarEventCreated),
		string(EventCalendarEventUpdated),
	}
}
