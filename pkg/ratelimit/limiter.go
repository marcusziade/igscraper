package ratelimit

import (
	"sync"
	"time"
)

// Limiter defines the interface for rate limiting
type Limiter interface {
	// Allow checks if a request is allowed under the current rate limit
	Allow() bool
	// Wait blocks until the rate limit allows another request
	Wait()
	// Reset resets the rate limiter state
	Reset()
}

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	capacity     int           // Maximum number of tokens
	tokens       int           // Current number of tokens
	refillPeriod time.Duration // Period after which bucket is refilled
	lastRefill   time.Time     // Last time the bucket was refilled
	mu           sync.Mutex
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(capacity int, refillPeriod time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity:     capacity,
		tokens:       capacity,
		refillPeriod: refillPeriod,
		lastRefill:   time.Now(),
	}
}

// Allow checks if a request can proceed
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// Wait blocks until a token is available
func (tb *TokenBucket) Wait() {
	for !tb.Allow() {
		tb.mu.Lock()
		timeUntilRefill := tb.refillPeriod - time.Since(tb.lastRefill)
		tb.mu.Unlock()

		if timeUntilRefill > 0 {
			time.Sleep(timeUntilRefill)
		} else {
			// Small sleep to prevent busy waiting
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Reset resets the token bucket to full capacity
func (tb *TokenBucket) Reset() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.tokens = tb.capacity
	tb.lastRefill = time.Now()
}

// refill adds tokens based on elapsed time
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	if elapsed >= tb.refillPeriod {
		tb.tokens = tb.capacity
		tb.lastRefill = now
	}
}

// SlidingWindow implements a sliding window rate limiter
type SlidingWindow struct {
	windowSize   time.Duration
	maxRequests  int
	requests     []time.Time
	mu           sync.Mutex
}

// NewSlidingWindow creates a new sliding window rate limiter
func NewSlidingWindow(maxRequests int, windowSize time.Duration) *SlidingWindow {
	return &SlidingWindow{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		requests:    make([]time.Time, 0, maxRequests),
	}
}

// Allow checks if a request can proceed
func (sw *SlidingWindow) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	sw.cleanOldRequests(now)

	if len(sw.requests) < sw.maxRequests {
		sw.requests = append(sw.requests, now)
		return true
	}

	return false
}

// Wait blocks until a request is allowed
func (sw *SlidingWindow) Wait() {
	for !sw.Allow() {
		sw.mu.Lock()
		if len(sw.requests) > 0 {
			oldestRequest := sw.requests[0]
			timeToWait := sw.windowSize - time.Since(oldestRequest)
			sw.mu.Unlock()

			if timeToWait > 0 {
				time.Sleep(timeToWait)
			}
		} else {
			sw.mu.Unlock()
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Reset clears all recorded requests
func (sw *SlidingWindow) Reset() {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.requests = sw.requests[:0]
}

// cleanOldRequests removes requests outside the sliding window
func (sw *SlidingWindow) cleanOldRequests(now time.Time) {
	cutoff := now.Add(-sw.windowSize)
	
	// Find the first request that's within the window
	i := 0
	for i < len(sw.requests) && sw.requests[i].Before(cutoff) {
		i++
	}
	
	// Keep only requests within the window
	if i > 0 {
		copy(sw.requests, sw.requests[i:])
		sw.requests = sw.requests[:len(sw.requests)-i]
	}
}
