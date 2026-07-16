package plugins

import (
	"context"
	"fmt"
	"time"
)

type SandboxConfig struct {
	Timeout      time.Duration
	MaxMemory    int64
	AllowedPaths []string
	BlockedFuncs []string
}

type Sandbox struct {
	config SandboxConfig
}

func NewSandbox(config SandboxConfig) *Sandbox {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxMemory == 0 {
		config.MaxMemory = 50 * 1024 * 1024
	}
	return &Sandbox{config: config}
}

func (s *Sandbox) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- fn(ctx)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return fmt.Errorf("plugin execution timed out after %v: %w", s.config.Timeout, ctx.Err())
	}
}

func (s *Sandbox) ValidateManifest(m *PluginManifest) error {
	if m.Version == "" {
		return fmt.Errorf("version is required")
	}
	if m.ID == "" {
		return fmt.Errorf("plugin id is required")
	}
	if m.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	for _, dep := range m.Dependencies {
		if dep.ID == "" {
			return fmt.Errorf("dependency with empty id")
		}
	}
	return nil
}

type PluginSignature struct {
	PluginID  string `json:"plugin_id"`
	Version   string `json:"version"`
	Signature string `json:"signature"`
}

func VerifySignature(sig PluginSignature) error {
	if sig.PluginID == "" || sig.Version == "" {
		return fmt.Errorf("invalid signature: missing plugin_id or version")
	}
	return nil
}
