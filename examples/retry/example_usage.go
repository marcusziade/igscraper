package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"igscraper/pkg/config"
	"igscraper/pkg/instagram"
	"igscraper/pkg/logger"
	"igscraper/pkg/retry"
)

func ExampleDo() {
	// Simple retry with default configuration
	err := retry.Do(func() error {
		// Your operation that might fail
		return someNetworkOperation()
	}, nil)

	if err != nil {
		log.Printf("Operation failed after retries: %v", err)
	}
}

func ExampleDo_customConfig() {
	// Custom retry configuration
	cfg := &retry.Config{
		MaxAttempts: 5,
		Backoff: &retry.ExponentialBackoff{
			BaseDelay:    2 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			JitterFactor: 0.1,
		},
		RetryIf: func(err error) bool {
			// Custom logic to determine if error is retryable
			return err != nil && err.Error() != "permanent error"
		},
		OnRetry: func(attempt int, err error, delay time.Duration) {
			log.Printf("Retry attempt %d after error: %v (waiting %v)", attempt, err, delay)
		},
		Context: context.Background(),
		Logger:  logger.GetLogger(),
	}

	err := retry.Do(func() error {
		return someNetworkOperation()
	}, cfg)

	if err != nil {
		log.Printf("Operation failed: %v", err)
	}
}

func ExampleHTTPRetrier() {
	// Create an HTTP-specific retrier
	retrier := retry.NewHTTPRetrier(3, logger.GetLogger())

	// Use with Instagram client operations
	client := instagram.NewClient(30*time.Second, logger.GetLogger())
	
	err := retrier.DoWithErrorType(func() error {
		_, err := client.FetchUserProfile("username")
		return err
	})

	if err != nil {
		log.Printf("Failed to fetch profile: %v", err)
	}
}

func ExampleDoWithResult() {
	// Retry an operation that returns a result
	result, err := retry.DoWithResult(func() (string, error) {
		// Fetch some data that might fail
		data, err := fetchDataFromAPI()
		if err != nil {
			return "", err
		}
		return data, nil
	}, nil)

	if err != nil {
		log.Printf("Failed to fetch data: %v", err)
	} else {
		fmt.Printf("Data: %s\n", result)
	}
}

func ExampleErrorTypeBackoff() {
	// Different backoff strategies for different error types
	errorBackoff := retry.NewErrorTypeBackoff()
	
	// Configure custom backoffs for specific error types
	errorBackoff.NetworkErrorBackoff = &retry.ExponentialBackoff{
		BaseDelay:    500 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.2,
	}
	
	errorBackoff.RateLimitBackoff = &retry.LinearBackoff{
		BaseDelay:    1 * time.Minute,
		MaxDelay:     5 * time.Minute,
		Increment:    30 * time.Second,
		JitterFactor: 0.1,
	}
	
	// Use with retry configuration
	cfg := &retry.Config{
		MaxAttempts: 5,
		Backoff:     errorBackoff.DefaultBackoff,
		RetryIf:     retry.DefaultRetryIf,
		OnRetry: func(attempt int, err error, delay time.Duration) {
			// Switch backoff based on error type
			if igErr, ok := err.(*instagram.Error); ok {
				switch igErr.Type {
				case instagram.ErrorTypeNetwork:
					cfg.Backoff = errorBackoff.NetworkErrorBackoff
				case instagram.ErrorTypeRateLimit:
					cfg.Backoff = errorBackoff.RateLimitBackoff
				}
			}
		},
	}
	
	retry.Do(someOperation, cfg)
}

func ExampleRetrier_withChaining() {
	// Create a retrier with chained configuration
	retrier := retry.NewRetrier(nil).
		WithMaxAttempts(5).
		WithBackoff(&retry.ExponentialBackoff{
			BaseDelay:    1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   1.5,
			JitterFactor: 0.1,
		}).
		WithContext(context.Background())
	
	err := retrier.Do(func() error {
		return someNetworkOperation()
	})
	
	if err != nil {
		log.Printf("Operation failed: %v", err)
	}
}

func ExampleConfig_fromInstagramConfig() {
	// Create retry config from Instagram scraper config
	cfg := config.DefaultConfig()
	
	retryConfig := &retry.Config{
		MaxAttempts: cfg.Retry.MaxAttempts,
		Backoff: &retry.ExponentialBackoff{
			BaseDelay:    cfg.Retry.BaseDelay,
			MaxDelay:     cfg.Retry.MaxDelay,
			Multiplier:   cfg.Retry.Multiplier,
			JitterFactor: cfg.Retry.JitterFactor,
		},
		RetryIf: retry.DefaultRetryIf,
		Context: context.Background(),
		Logger:  logger.GetLogger(),
	}
	
	retry.Do(someOperation, retryConfig)
}

// Helper functions for examples
func someNetworkOperation() error {
	// Placeholder for network operation
	return nil
}

func someOperation() error {
	// Placeholder for generic operation
	return nil
}

func fetchDataFromAPI() (string, error) {
	// Placeholder for API call
	return "data", nil
}