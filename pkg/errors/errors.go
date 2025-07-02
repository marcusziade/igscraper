package errors

import "fmt"

// ErrorType represents different types of errors that can occur
type ErrorType string

const (
	ErrorTypeNetwork      ErrorType = "network"
	ErrorTypeRateLimit    ErrorType = "rate_limit"
	ErrorTypeAuth         ErrorType = "auth"
	ErrorTypeParsing      ErrorType = "parsing"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeServerError  ErrorType = "server_error"
	ErrorTypeUnknown      ErrorType = "unknown"
)

// Error represents an API error with type information
type Error struct {
	Type    ErrorType
	Message string
	Code    int
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s error (code %d): %s", e.Type, e.Code, e.Message)
}

// IsRetryable checks if an error type should be retried
func IsRetryable(errorType ErrorType) bool {
	switch errorType {
	case ErrorTypeNetwork, ErrorTypeRateLimit, ErrorTypeServerError:
		return true
	case ErrorTypeAuth, ErrorTypeNotFound, ErrorTypeParsing:
		return false
	default:
		return false
	}
}

// IsRetryableStatusCode checks if an HTTP status code indicates a retryable error
func IsRetryableStatusCode(statusCode int) bool {
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