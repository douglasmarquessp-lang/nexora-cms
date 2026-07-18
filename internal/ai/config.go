package ai

import "time"

type AIConfig struct {
	DefaultProvider string        `json:"default_provider"`
	Providers       []ProviderCfg `json:"providers"`
	Retry           RetryConfig   `json:"retry"`
	CircuitBreaker  CBConfig      `json:"circuit_breaker"`
	GlobalTimeout   time.Duration `json:"global_timeout"`
	Enabled         bool          `json:"enabled"`
}

type ProviderCfg struct {
	Name       string        `json:"name"`
	Model      string        `json:"model"`
	APIKey     string        `json:"api_key,omitempty"`
	BaseURL    string        `json:"base_url,omitempty"`
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries"`
	Weight     int           `json:"weight"`
	Priority   int           `json:"priority"`
	Enabled    bool          `json:"enabled"`
}

type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"`
	BaseDelay   time.Duration `json:"base_delay"`
	MaxDelay    time.Duration `json:"max_delay"`
}

type CBConfig struct {
	Enabled          bool          `json:"enabled"`
	FailureThreshold int           `json:"failure_threshold"`
	RecoveryTimeout  time.Duration `json:"recovery_timeout"`
	HalfOpenMaxReqs  int           `json:"half_open_max_requests"`
}

func DefaultConfig() AIConfig {
	return AIConfig{
		DefaultProvider: "",
		Providers:       []ProviderCfg{},
		Retry: RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   100 * time.Millisecond,
			MaxDelay:    5 * time.Second,
		},
		CircuitBreaker: CBConfig{
			Enabled:          true,
			FailureThreshold: 5,
			RecoveryTimeout:  30 * time.Second,
			HalfOpenMaxReqs:  3,
		},
		GlobalTimeout: 60 * time.Second,
		Enabled:       false,
	}
}
