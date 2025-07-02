package ratelimit

import (
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	tb := NewTokenBucket(5, time.Second)

	// Test initial capacity
	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Errorf("Expected token %d to be available", i+1)
		}
	}

	// Test exhaustion
	if tb.Allow() {
		t.Error("Expected no more tokens to be available")
	}

	// Test refill after waiting
	time.Sleep(time.Second + 100*time.Millisecond)
	if !tb.Allow() {
		t.Error("Expected tokens to be refilled after waiting")
	}

	// Test reset
	tb.tokens = 0
	tb.Reset()
	if tb.tokens != tb.capacity {
		t.Error("Expected tokens to be reset to capacity")
	}
}

func TestSlidingWindow(t *testing.T) {
	sw := NewSlidingWindow(3, time.Second)

	// Test initial requests
	for i := 0; i < 3; i++ {
		if !sw.Allow() {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
	}

	// Test limit reached
	if sw.Allow() {
		t.Error("Expected request to be denied when limit is reached")
	}

	// Test window sliding
	time.Sleep(time.Second + 100*time.Millisecond)
	if !sw.Allow() {
		t.Error("Expected request to be allowed after window slides")
	}

	// Test reset
	sw.Reset()
	if len(sw.requests) != 0 {
		t.Error("Expected requests to be cleared after reset")
	}
}