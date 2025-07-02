// Package retry provides exponential backoff and retry logic for handling
// transient failures in network operations, particularly for Instagram API calls.
//
// Features:
//   - Multiple backoff strategies (exponential, linear, constant)
//   - Jitter to avoid thundering herd problems
//   - Context support for cancellation
//   - Error-type specific backoff strategies
//   - Configurable retry predicates
//   - Integration with Instagram client error types
//
// Basic usage:
//
//	// Simple retry with defaults
//	err := retry.Do(func() error {
//		return client.FetchUserProfile(username)
//	}, nil)
//
//	// Custom configuration
//	cfg := &retry.Config{
//		MaxAttempts: 5,
//		Backoff: &retry.ExponentialBackoff{
//			BaseDelay:    2 * time.Second,
//			MaxDelay:     30 * time.Second,
//			Multiplier:   2.0,
//			JitterFactor: 0.1,
//		},
//		RetryIf: retry.DefaultRetryIf,
//		Logger:  logger.GetLogger(),
//	}
//	err := retry.Do(operation, cfg)
//
//	// HTTP-specific retrier with error-type backoff
//	retrier := retry.NewHTTPRetrier(3, logger.GetLogger())
//	err := retrier.DoWithErrorType(func() error {
//		return client.DownloadPhoto(url)
//	})
//
// Error Type Handling:
//
// The package provides different backoff strategies for different error types:
//   - Network errors: Quick retries with exponential backoff
//   - Rate limit errors: Longer delays with less aggressive backoff
//   - Server errors: Moderate delays with exponential backoff
//   - Auth/NotFound errors: No retry (non-retryable)
package retry