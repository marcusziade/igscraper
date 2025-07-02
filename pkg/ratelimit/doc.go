// Package ratelimit provides rate limiting functionality for the Instagram scraper.
//
// This package implements multiple rate limiting algorithms to prevent
// overwhelming Instagram's servers and avoid getting blocked.
//
// Available Implementations:
//
// Token Bucket:
//   - Fixed capacity bucket that refills after a specified period
//   - Suitable for burst traffic followed by quiet periods
//   - Default implementation used by the scraper
//
// Sliding Window:
//   - Tracks requests within a moving time window
//   - More accurate rate limiting over time
//   - Better for consistent request patterns
//
// Interface:
//
// All rate limiters implement the Limiter interface:
//   - Allow() bool - Check if a request is allowed
//   - Wait() - Block until a request is allowed
//   - Reset() - Reset the limiter state
//
// Usage:
//
//	// Token bucket: 50 requests per hour
//	limiter := ratelimit.NewTokenBucket(50, time.Hour)
//	
//	if limiter.Allow() {
//	    // Proceed with request
//	} else {
//	    // Wait for rate limit to reset
//	    limiter.Wait()
//	}
//	
//	// Sliding window: 100 requests per 15 minutes
//	limiter := ratelimit.NewSlidingWindow(100, 15*time.Minute)
//	
//	// Block until allowed
//	limiter.Wait()
//	// Proceed with request
package ratelimit