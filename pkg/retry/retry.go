package retry

import (
	"context"
	"errors"
	"fmt"
	"time"

	errs "igscraper/pkg/errors"
	"igscraper/pkg/logger"
)

// Operation is a function that performs an operation that might need retrying
type Operation func() error

// OperationWithResult is a function that returns a result and might need retrying
type OperationWithResult[T any] func() (T, error)

// Config holds retry configuration
type Config struct {
	// MaxAttempts is the maximum number of attempts (0 means unlimited)
	MaxAttempts int
	// Backoff strategy to use
	Backoff BackoffStrategy
	// RetryIf determines if an error should be retried
	RetryIf func(error) bool
	// OnRetry is called before each retry attempt
	OnRetry func(attempt int, err error, delay time.Duration)
	// Context for cancellation
	Context context.Context
	// Logger for retry attempts
	Logger logger.Logger
}

// DefaultConfig returns a retry configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts: 3,
		Backoff:     DefaultExponentialBackoff(),
		RetryIf:     DefaultRetryIf,
		OnRetry:     nil,
		Context:     context.Background(),
		Logger:      logger.GetLogger(),
	}
}

// DefaultRetryIf is the default retry predicate
func DefaultRetryIf(err error) bool {
	if err == nil {
		return false
	}
	
	// Check if it's an API error
	var apiErr *errs.Error
	if errors.As(err, &apiErr) {
		return errs.IsRetryable(apiErr.Type)
	}
	
	// Check for context errors (don't retry)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	
	// Default to retrying unknown errors
	return true
}

// IsRetryable checks if an error should be retried based on HTTP status codes
func IsRetryable(statusCode int) bool {
	switch statusCode {
	case 0: // Network error
		return true
	case 429: // Too Many Requests
		return true
	case 500, 502, 503, 504: // Server errors
		return true
	case 401, 403, 404: // Client errors that won't change
		return false
	default:
		return statusCode >= 500 // Retry all 5xx errors
	}
}

// Do executes an operation with retry logic
func Do(op Operation, cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	
	var lastErr error
	attempt := 0
	
	for {
		attempt++
		
		// Check if we've exceeded max attempts
		if cfg.MaxAttempts > 0 && attempt > cfg.MaxAttempts {
			if cfg.Logger != nil {
				cfg.Logger.ErrorWithFields("max retry attempts exceeded", map[string]interface{}{
					"attempts":   attempt - 1,
					"last_error": lastErr.Error(),
				})
			}
			return fmt.Errorf("max retry attempts (%d) exceeded: %w", cfg.MaxAttempts, lastErr)
		}
		
		// Execute the operation
		err := op()
		if err == nil {
			// Success
			if attempt > 1 && cfg.Logger != nil {
				cfg.Logger.DebugWithFields("operation succeeded after retry", map[string]interface{}{
					"attempt": attempt,
				})
			}
			return nil
		}
		
		lastErr = err
		
		// Check if we should retry this error
		if !cfg.RetryIf(err) {
			if cfg.Logger != nil {
				cfg.Logger.DebugWithFields("error is not retryable", map[string]interface{}{
					"error": err.Error(),
				})
			}
			return err
		}
		
		// Calculate delay
		delay := cfg.Backoff.NextDelay(attempt)
		
		// Call OnRetry callback if provided
		if cfg.OnRetry != nil {
			cfg.OnRetry(attempt, err, delay)
		}
		
		// Log retry attempt
		if cfg.Logger != nil {
			cfg.Logger.WarnWithFields("retrying operation", map[string]interface{}{
				"attempt":     attempt,
				"error":       err.Error(),
				"delay_ms":    delay.Milliseconds(),
				"max_attempts": cfg.MaxAttempts,
			})
		}
		
		// Wait before retry
		if err := Wait(cfg.Context, delay); err != nil {
			// Context cancelled
			if cfg.Logger != nil {
				cfg.Logger.WarnWithFields("retry cancelled", map[string]interface{}{
					"attempt": attempt,
					"reason":  err.Error(),
				})
			}
			return fmt.Errorf("retry cancelled: %w", err)
		}
	}
}

// DoWithResult executes an operation that returns a result with retry logic
func DoWithResult[T any](op OperationWithResult[T], cfg *Config) (T, error) {
	var result T
	
	err := Do(func() error {
		var opErr error
		result, opErr = op()
		return opErr
	}, cfg)
	
	return result, err
}

// Retrier provides a reusable retry mechanism
type Retrier struct {
	config *Config
}

// NewRetrier creates a new retrier with the given configuration
func NewRetrier(cfg *Config) *Retrier {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Retrier{config: cfg}
}

// Do executes an operation with retry logic
func (r *Retrier) Do(op Operation) error {
	return Do(op, r.config)
}

// DoWithResult executes an operation that returns a result with retry logic
func (r *Retrier) DoWithResult(op OperationWithResult[any]) (any, error) {
	return DoWithResult(op, r.config)
}

// WithMaxAttempts returns a new retrier with updated max attempts
func (r *Retrier) WithMaxAttempts(maxAttempts int) *Retrier {
	newConfig := *r.config
	newConfig.MaxAttempts = maxAttempts
	return &Retrier{config: &newConfig}
}

// WithBackoff returns a new retrier with updated backoff strategy
func (r *Retrier) WithBackoff(backoff BackoffStrategy) *Retrier {
	newConfig := *r.config
	newConfig.Backoff = backoff
	return &Retrier{config: &newConfig}
}

// WithContext returns a new retrier with updated context
func (r *Retrier) WithContext(ctx context.Context) *Retrier {
	newConfig := *r.config
	newConfig.Context = ctx
	return &Retrier{config: &newConfig}
}

// HTTPRetrier provides retry logic specifically for HTTP operations
type HTTPRetrier struct {
	*Retrier
	errorTypeBackoff *ErrorTypeBackoff
}

// NewHTTPRetrier creates a new HTTP-specific retrier
func NewHTTPRetrier(maxAttempts int, logger logger.Logger) *HTTPRetrier {
	errorTypeBackoff := NewErrorTypeBackoff()
	
	cfg := &Config{
		MaxAttempts: maxAttempts,
		Backoff:     errorTypeBackoff.DefaultBackoff,
		RetryIf:     DefaultRetryIf,
		Context:     context.Background(),
		Logger:      logger,
	}
	
	return &HTTPRetrier{
		Retrier:          NewRetrier(cfg),
		errorTypeBackoff: errorTypeBackoff,
	}
}

// DoWithErrorType executes an operation with error-type specific backoff
func (hr *HTTPRetrier) DoWithErrorType(op Operation) error {
	return Do(op, &Config{
		MaxAttempts: hr.config.MaxAttempts,
		Backoff:     hr.config.Backoff,
		RetryIf:     hr.config.RetryIf,
		Context:     hr.config.Context,
		Logger:      hr.config.Logger,
		OnRetry: func(attempt int, err error, delay time.Duration) {
			// Switch backoff strategy based on error type
			var apiErr *errs.Error
			if errors.As(err, &apiErr) {
				switch apiErr.Type {
				case errs.ErrorTypeNetwork:
					hr.config.Backoff = hr.errorTypeBackoff.NetworkErrorBackoff
				case errs.ErrorTypeRateLimit:
					hr.config.Backoff = hr.errorTypeBackoff.RateLimitBackoff
				case errs.ErrorTypeServerError:
					hr.config.Backoff = hr.errorTypeBackoff.ServerErrorBackoff
				default:
					hr.config.Backoff = hr.errorTypeBackoff.DefaultBackoff
				}
			}
		},
	})
}