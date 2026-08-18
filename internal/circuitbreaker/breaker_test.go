package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerTransitions(t *testing.T) {
	cfg := Config{
		FailureThreshold:    3,
		RecoveryTime:        100 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}
	cb := NewBreaker(cfg)

	if !cb.IsClosed() {
		t.Errorf("expected initial state closed, got %s", cb.GetStateString())
	}

	testErr := errors.New("simulated backend failure")

	for i := 0; i < 3; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return testErr
		})
	}

	if !cb.IsOpen() {
		t.Errorf("expected state open after 3 failures, got %s", cb.GetStateString())
	}

	time.Sleep(150 * time.Millisecond)

	var executed bool
	err := cb.Execute(context.Background(), func() error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("expected success in half-open trial, got %v", err)
	}
	if !executed {
		t.Errorf("expected trial function to execute")
	}
	if !cb.IsClosed() {
		t.Errorf("expected state closed after successful half-open recovery, got %s", cb.GetStateString())
	}
}

func TestCircuitBreakerSuccess(t *testing.T) {
	cfg := Config{
		FailureThreshold:    3,
		RecoveryTime:        1 * time.Second,
		HalfOpenMaxRequests: 1,
	}
	cb := NewBreaker(cfg)

	var count int
	err := cb.Execute(context.Background(), func() error {
		count++
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if count != 1 {
		t.Errorf("expected execution count 1, got %d", count)
	}
	if !cb.IsClosed() {
		t.Errorf("expected breaker to remain closed")
	}
}
