package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	errs "igscraper/pkg/errors"
)

func TestExponentialBackoff(t *testing.T) {
	backoff := &ExponentialBackoff{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.0, // No jitter for predictable testing
	}

	tests := []struct {
		attempt      int
		expectedMin  time.Duration
		expectedMax  time.Duration
		description  string
	}{
		{1, 100 * time.Millisecond, 100 * time.Millisecond, "First attempt"},
		{2, 200 * time.Millisecond, 200 * time.Millisecond, "Second attempt"},
		{3, 400 * time.Millisecond, 400 * time.Millisecond, "Third attempt"},
		{4, 800 * time.Millisecond, 800 * time.Millisecond, "Fourth attempt"},
		{5, 1 * time.Second, 1 * time.Second, "Fifth attempt (capped at max)"},
		{6, 1 * time.Second, 1 * time.Second, "Sixth attempt (still capped)"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			delay := backoff.NextDelay(test.attempt)
			if delay < test.expectedMin || delay > test.expectedMax {
				t.Errorf("Expected delay between %v and %v, got %v",
					test.expectedMin, test.expectedMax, delay)
			}
		})
	}
}

func TestExponentialBackoffWithJitter(t *testing.T) {
	backoff := &ExponentialBackoff{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.3,
	}

	// Test that jitter adds randomness
	delays := make(map[time.Duration]bool)
	for i := 0; i < 10; i++ {
		delay := backoff.NextDelay(2)
		delays[delay] = true
	}

	// With jitter, we should get different delays
	if len(delays) < 2 {
		t.Error("Expected multiple different delays with jitter, but got consistent delays")
	}
}

func TestRetryWithSuccess(t *testing.T) {
	attempts := 0
	op := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	cfg := &Config{
		MaxAttempts: 5,
		Backoff:     &ConstantBackoff{Delay: 10 * time.Millisecond},
		RetryIf:     func(err error) bool { return true },
		Context:     context.Background(),
	}

	err := Do(op, cfg)
	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithMaxAttemptsExceeded(t *testing.T) {
	attempts := 0
	op := func() error {
		attempts++
		return errors.New("persistent error")
	}

	cfg := &Config{
		MaxAttempts: 3,
		Backoff:     &ConstantBackoff{Delay: 10 * time.Millisecond},
		RetryIf:     func(err error) bool { return true },
		Context:     context.Background(),
	}

	err := Do(op, cfg)
	if err == nil {
		t.Error("Expected error when max attempts exceeded")
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithNonRetryableError(t *testing.T) {
	attempts := 0
	authError := &errs.Error{
		Type:    errs.ErrorTypeAuth,
		Message: "authentication required",
		Code:    401,
	}

	op := func() error {
		attempts++
		return authError
	}

	cfg := &Config{
		MaxAttempts: 5,
		Backoff:     &ConstantBackoff{Delay: 10 * time.Millisecond},
		RetryIf:     DefaultRetryIf,
		Context:     context.Background(),
	}

	err := Do(op, cfg)
	if err != authError {
		t.Errorf("Expected auth error, got: %v", err)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retry for auth error), got %d", attempts)
	}
}

func TestRetryWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	op := func() error {
		attempts++
		if attempts == 2 {
			cancel() // Cancel after second attempt
		}
		return errors.New("error")
	}

	cfg := &Config{
		MaxAttempts: 5,
		Backoff:     &ConstantBackoff{Delay: 100 * time.Millisecond},
		RetryIf:     func(err error) bool { return true },
		Context:     ctx,
	}

	err := Do(op, cfg)
	if err == nil {
		t.Error("Expected error when context cancelled")
	}
	if attempts > 3 {
		t.Errorf("Expected at most 3 attempts before cancellation, got %d", attempts)
	}
}

func TestErrorTypeBackoff(t *testing.T) {
	etb := NewErrorTypeBackoff()

	// Test network error backoff
	networkBackoff := etb.GetBackoffForError("network")
	if eb, ok := networkBackoff.(*ExponentialBackoff); ok {
		if eb.BaseDelay != 1*time.Second {
			t.Errorf("Expected network base delay of 1s, got %v", eb.BaseDelay)
		}
	} else {
		t.Error("Expected ExponentialBackoff for network errors")
	}

	// Test rate limit backoff
	rateLimitBackoff := etb.GetBackoffForError("rate_limit")
	if eb, ok := rateLimitBackoff.(*ExponentialBackoff); ok {
		if eb.BaseDelay != 30*time.Second {
			t.Errorf("Expected rate limit base delay of 30s, got %v", eb.BaseDelay)
		}
	} else {
		t.Error("Expected ExponentialBackoff for rate limit errors")
	}
}

func TestLinearBackoff(t *testing.T) {
	backoff := &LinearBackoff{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		Increment:    100 * time.Millisecond,
		JitterFactor: 0.0,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 300 * time.Millisecond},
		{4, 400 * time.Millisecond},
		{5, 500 * time.Millisecond},
		{6, 500 * time.Millisecond}, // Capped at max
	}

	for _, test := range tests {
		delay := backoff.NextDelay(test.attempt)
		if delay != test.expected {
			t.Errorf("Attempt %d: expected %v, got %v", test.attempt, test.expected, delay)
		}
	}
}

func TestDoWithResult(t *testing.T) {
	attempts := 0
	op := func() (string, error) {
		attempts++
		if attempts < 2 {
			return "", errors.New("temporary error")
		}
		return "success", nil
	}

	cfg := &Config{
		MaxAttempts: 3,
		Backoff:     &ConstantBackoff{Delay: 10 * time.Millisecond},
		RetryIf:     func(err error) bool { return true },
		Context:     context.Background(),
	}

	result, err := DoWithResult(op, cfg)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if result != "success" {
		t.Errorf("Expected 'success', got '%s'", result)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}