package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// BackoffStrategy defines the interface for different backoff strategies
type BackoffStrategy interface {
	// NextDelay returns the next delay duration
	NextDelay(attempt int) time.Duration
	// Reset resets the backoff strategy to initial state
	Reset()
}

// ExponentialBackoff implements exponential backoff with jitter
type ExponentialBackoff struct {
	// BaseDelay is the initial delay duration
	BaseDelay time.Duration
	// MaxDelay is the maximum delay duration
	MaxDelay time.Duration
	// Multiplier is the factor by which delay increases
	Multiplier float64
	// JitterFactor adds randomness to avoid thundering herd (0.0 to 1.0)
	JitterFactor float64
	// attempts tracks the number of attempts made
	attempts int
}

// DefaultExponentialBackoff returns a backoff with sensible defaults
func DefaultExponentialBackoff() *ExponentialBackoff {
	return &ExponentialBackoff{
		BaseDelay:    1 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.1,
	}
}

// NextDelay calculates the next delay with exponential backoff and jitter
func (eb *ExponentialBackoff) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate exponential delay
	delay := float64(eb.BaseDelay) * math.Pow(eb.Multiplier, float64(attempt-1))
	
	// Cap at max delay
	if delay > float64(eb.MaxDelay) {
		delay = float64(eb.MaxDelay)
	}
	
	// Add jitter to avoid thundering herd
	if eb.JitterFactor > 0 {
		jitter := delay * eb.JitterFactor
		// Random value between -jitter and +jitter
		randomJitter := (rand.Float64() * 2 * jitter) - jitter
		delay += randomJitter
	}
	
	// Ensure delay is not negative
	if delay < 0 {
		delay = 0
	}
	
	return time.Duration(delay)
}

// Reset resets the backoff to initial state
func (eb *ExponentialBackoff) Reset() {
	eb.attempts = 0
}

// LinearBackoff implements linear backoff strategy
type LinearBackoff struct {
	// BaseDelay is the fixed delay between attempts
	BaseDelay time.Duration
	// MaxDelay is the maximum delay duration
	MaxDelay time.Duration
	// Increment is the amount to increase delay by each attempt
	Increment time.Duration
	// JitterFactor adds randomness (0.0 to 1.0)
	JitterFactor float64
}

// DefaultLinearBackoff returns a linear backoff with sensible defaults
func DefaultLinearBackoff() *LinearBackoff {
	return &LinearBackoff{
		BaseDelay:    1 * time.Second,
		MaxDelay:     30 * time.Second,
		Increment:    1 * time.Second,
		JitterFactor: 0.1,
	}
}

// NextDelay calculates the next delay with linear backoff
func (lb *LinearBackoff) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	
	// Calculate linear delay
	delay := float64(lb.BaseDelay + lb.Increment*time.Duration(attempt-1))
	
	// Cap at max delay
	if delay > float64(lb.MaxDelay) {
		delay = float64(lb.MaxDelay)
	}
	
	// Add jitter
	if lb.JitterFactor > 0 {
		jitter := delay * lb.JitterFactor
		randomJitter := (rand.Float64() * 2 * jitter) - jitter
		delay += randomJitter
	}
	
	// Ensure delay is not negative
	if delay < 0 {
		delay = 0
	}
	
	return time.Duration(delay)
}

// Reset resets the backoff to initial state
func (lb *LinearBackoff) Reset() {
	// Linear backoff doesn't need to track state
}

// ConstantBackoff implements constant delay backoff
type ConstantBackoff struct {
	Delay time.Duration
}

// NextDelay returns a constant delay
func (cb *ConstantBackoff) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	return cb.Delay
}

// Reset resets the backoff (no-op for constant backoff)
func (cb *ConstantBackoff) Reset() {}

// Wait waits for the specified duration or until context is cancelled
func Wait(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	
	timer := time.NewTimer(delay)
	defer timer.Stop()
	
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ErrorTypeBackoff provides different backoff strategies based on error types
type ErrorTypeBackoff struct {
	// NetworkErrorBackoff for network-related errors
	NetworkErrorBackoff BackoffStrategy
	// RateLimitBackoff for rate limit errors (typically longer delays)
	RateLimitBackoff BackoffStrategy
	// ServerErrorBackoff for 5xx errors
	ServerErrorBackoff BackoffStrategy
	// DefaultBackoff for other retryable errors
	DefaultBackoff BackoffStrategy
}

// NewErrorTypeBackoff creates a new error-type based backoff
func NewErrorTypeBackoff() *ErrorTypeBackoff {
	return &ErrorTypeBackoff{
		NetworkErrorBackoff: &ExponentialBackoff{
			BaseDelay:    1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			JitterFactor: 0.2,
		},
		RateLimitBackoff: &ExponentialBackoff{
			BaseDelay:    30 * time.Second,
			MaxDelay:     5 * time.Minute,
			Multiplier:   1.5,
			JitterFactor: 0.3,
		},
		ServerErrorBackoff: &ExponentialBackoff{
			BaseDelay:    5 * time.Second,
			MaxDelay:     60 * time.Second,
			Multiplier:   2.0,
			JitterFactor: 0.1,
		},
		DefaultBackoff: DefaultExponentialBackoff(),
	}
}

// GetBackoffForError returns the appropriate backoff strategy for the error type
func (etb *ErrorTypeBackoff) GetBackoffForError(errorType string) BackoffStrategy {
	switch errorType {
	case "network":
		return etb.NetworkErrorBackoff
	case "rate_limit":
		return etb.RateLimitBackoff
	case "server_error":
		return etb.ServerErrorBackoff
	default:
		return etb.DefaultBackoff
	}
}