package plugins

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSandbox_Execute(t *testing.T) {
	s := NewSandbox(SandboxConfig{Timeout: time.Second})

	called := false
	err := s.Execute(context.Background(), func(ctx context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("function not called")
	}
}

func TestSandbox_Execute_Error(t *testing.T) {
	s := NewSandbox(SandboxConfig{Timeout: time.Second})

	err := s.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("execution error")
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSandbox_Execute_Timeout(t *testing.T) {
	s := NewSandbox(SandboxConfig{Timeout: 10 * time.Millisecond})

	err := s.Execute(context.Background(), func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			return nil
		}
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestSandbox_ValidateManifest(t *testing.T) {
	s := NewSandbox(SandboxConfig{})

	err := s.ValidateManifest(&PluginManifest{
		ID:      "test-p",
		Name:    "Test",
		Version: "1.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSandbox_ValidateManifest_MissingID(t *testing.T) {
	s := NewSandbox(SandboxConfig{})

	err := s.ValidateManifest(&PluginManifest{
		Name:    "Test",
		Version: "1.0.0",
	})
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestSandbox_ValidateManifest_MissingName(t *testing.T) {
	s := NewSandbox(SandboxConfig{})

	err := s.ValidateManifest(&PluginManifest{
		ID:      "test-p",
		Version: "1.0.0",
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestSandbox_ValidateManifest_MissingVersion(t *testing.T) {
	s := NewSandbox(SandboxConfig{})

	err := s.ValidateManifest(&PluginManifest{
		ID:   "test-p",
		Name: "Test",
	})
	if err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestSandbox_DefaultConfig(t *testing.T) {
	s := NewSandbox(SandboxConfig{})
	if s.config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v", s.config.Timeout)
	}
	if s.config.MaxMemory != 50*1024*1024 {
		t.Errorf("MaxMemory = %d", s.config.MaxMemory)
	}
}

func TestVerifySignature(t *testing.T) {
	err := VerifySignature(PluginSignature{
		PluginID:  "test-p",
		Version:   "1.0.0",
		Signature: "sig",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestVerifySignature_NoID(t *testing.T) {
	err := VerifySignature(PluginSignature{
		Version: "1.0.0",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifySignature_NoVersion(t *testing.T) {
	err := VerifySignature(PluginSignature{
		PluginID: "test-p",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
