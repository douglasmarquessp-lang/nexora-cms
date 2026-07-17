package ai

import (
	"testing"
	"time"
)

func TestCircuitBreaker_Closed(t *testing.T) {
	cb := newCircuitBreaker(CBConfig{
		Enabled:          true,
		FailureThreshold: 3,
		RecoveryTimeout:  100 * time.Millisecond,
		HalfOpenMaxReqs:  2,
	})

	if !cb.allow() {
		t.Error("expected circuit breaker to allow when closed")
	}
}

func TestCircuitBreaker_Opens(t *testing.T) {
	cb := newCircuitBreaker(CBConfig{
		Enabled:          true,
		FailureThreshold: 2,
		RecoveryTimeout:  100 * time.Millisecond,
		HalfOpenMaxReqs:  1,
	})

	// Two failures should open the circuit
	cb.failure()
	cb.failure()

	if cb.allow() {
		t.Error("expected circuit breaker to block when open")
	}
}

func TestCircuitBreaker_HalfOpen(t *testing.T) {
	cb := newCircuitBreaker(CBConfig{
		Enabled:          true,
		FailureThreshold: 1,
		RecoveryTimeout:  10 * time.Millisecond,
		HalfOpenMaxReqs:  2,
	})

	cb.failure()
	if cb.allow() {
		t.Error("expected circuit breaker to block after failure")
	}

	// Wait for recovery
	time.Sleep(20 * time.Millisecond)

	if !cb.allow() {
		t.Error("expected circuit breaker to allow after recovery timeout (half-open)")
	}
}

func TestCircuitBreaker_HalfOpenLimits(t *testing.T) {
	cb := newCircuitBreaker(CBConfig{
		Enabled:          true,
		FailureThreshold: 1,
		RecoveryTimeout:  10 * time.Millisecond,
		HalfOpenMaxReqs:  2,
	})

	cb.failure()
	time.Sleep(20 * time.Millisecond)

	// First call transitions open→halfOpen, returns true, halfOpenReqs=0
	if !cb.allow() {
		t.Error("expected transition request to be allowed")
	}
	// Now halfOpen: 0 < 2 → allowed, halfOpenReqs=1
	if !cb.allow() {
		t.Error("expected 1st half-open request to be allowed")
	}
	// halfOpen: 1 < 2 → allowed, halfOpenReqs=2
	if !cb.allow() {
		t.Error("expected 2nd half-open request to be allowed")
	}
	// halfOpen: 2 < 2 → blocked
	if cb.allow() {
		t.Error("expected 3rd half-open request to be blocked")
	}
}

func TestCircuitBreaker_Recovery(t *testing.T) {
	cb := newCircuitBreaker(CBConfig{
		Enabled:          true,
		FailureThreshold: 1,
		RecoveryTimeout:  10 * time.Millisecond,
		HalfOpenMaxReqs:  2,
	})

	cb.failure()
	time.Sleep(20 * time.Millisecond)

	// Allow in half-open
	cb.allow()
	// Success should close the circuit
	cb.success()

	if !cb.allow() {
		t.Error("expected circuit breaker to allow after recovery")
	}
}

func TestCircuitBreaker_SuccessInHalfOpen(t *testing.T) {
	cb := newCircuitBreaker(CBConfig{
		Enabled:          true,
		FailureThreshold: 2,
		RecoveryTimeout:  10 * time.Millisecond,
		HalfOpenMaxReqs:  2,
	})

	cb.failure()
	cb.failure()
	time.Sleep(20 * time.Millisecond)

	cb.allow()
	cb.success()

	if !cb.allow() {
		t.Error("expected circuit breaker to be closed after half-open success")
	}
}

func TestCircuitBreaker_FailureInHalfOpen(t *testing.T) {
	cb := newCircuitBreaker(CBConfig{
		Enabled:          true,
		FailureThreshold: 3,
		RecoveryTimeout:  10 * time.Millisecond,
		HalfOpenMaxReqs:  2,
	})

	cb.failure()
	cb.failure()
	cb.failure()
	time.Sleep(20 * time.Millisecond)

	cb.allow()
	cb.failure()

	if cb.allow() {
		t.Error("expected circuit breaker to re-open after half-open failure")
	}
}
