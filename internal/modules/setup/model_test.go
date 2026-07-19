package setup

import (
	"errors"
	"testing"
)

func TestModuleName(t *testing.T) {
	if ModuleName != "setup" {
		t.Errorf("expected ModuleName 'setup', got %q", ModuleName)
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrAlreadyInstalled", ErrAlreadyInstalled, "system is already installed"},
		{"ErrNotInstalled", ErrNotInstalled, "system is not installed yet"},
		{"ErrInvalidEmail", ErrInvalidEmail, "invalid email address"},
		{"ErrWeakPassword", ErrWeakPassword, "password does not meet strength requirements"},
		{"ErrRequiredField", ErrRequiredField, "required field is missing"},
		{"ErrDatabaseNotAvail", ErrDatabaseNotAvail, "database not available"},
		{"ErrInvalidInput", ErrInvalidInput, "invalid input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("expected %q, got %q", tt.msg, tt.err.Error())
			}
			if !errors.Is(tt.err, tt.err) {
				t.Errorf("errors.Is should match itself")
			}
		})
	}
}

func TestEventTypes(t *testing.T) {
	if string(EventSetupStarted) != "setup.started" {
		t.Errorf("expected 'setup.started', got %q", string(EventSetupStarted))
	}
	if string(EventSetupFinished) != "setup.finished" {
		t.Errorf("expected 'setup.finished', got %q", string(EventSetupFinished))
	}
}
