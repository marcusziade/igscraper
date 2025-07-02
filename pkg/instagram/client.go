package instagram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"igscraper/pkg/logger"
)

// Error types for Instagram API operations
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

// Error represents an Instagram API error
type Error struct {
	Type    ErrorType
	Message string
	Code    int
}

func (e *Error) Error() string {
	return fmt.Sprintf("instagram %s error (code %d): %s", e.Type, e.Code, e.Message)
}

// Client represents an Instagram API client
type Client struct {
	httpClient *http.Client
	headers    map[string]string
	baseURL    string
	logger     logger.Logger
}

// NewClient creates a new Instagram API client
func NewClient(timeout time.Duration, log logger.Logger) *Client {
	// Use default logger if none provided
	if log == nil {
		log = logger.GetLogger()
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		headers: map[string]string{
			"User-Agent":      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
			"Accept-Language": "en-US,en;q=0.9",
			"Accept-Encoding": "gzip, deflate, br",
			"Cache-Control":   "no-cache",
			"Pragma":          "no-cache",
			"Sec-Fetch-Dest":  "document",
			"Sec-Fetch-Mode":  "navigate",
			"Sec-Fetch-Site":  "none",
			"Sec-Fetch-User":  "?1",
		},
		baseURL: BaseURL,
		logger:  log,
	}
}

// SetHeader sets a custom header for the client
func (c *Client) SetHeader(key, value string) {
	c.headers[key] = value
}

// SetHeaders sets multiple headers at once
func (c *Client) SetHeaders(headers map[string]string) {
	for key, value := range headers {
		c.headers[key] = value
	}
}

// doRequest performs an HTTP request with the configured headers
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	// Set all headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// Log the request
	start := time.Now()
	c.logger.DebugWithFields("sending HTTP request", map[string]interface{}{
		"method": req.Method,
		"url":    req.URL.String(),
	})

	resp, err := c.httpClient.Do(req)
	duration := time.Since(start)

	if err != nil {
		c.logger.ErrorWithFields("HTTP request failed", map[string]interface{}{
			"method":   req.Method,
			"url":      req.URL.String(),
			"error":    err.Error(),
			"duration": duration,
		})
		return nil, &Error{
			Type:    ErrorTypeNetwork,
			Message: fmt.Sprintf("network error: %v", err),
			Code:    0,
		}
	}

	// Log successful response
	c.logger.DebugWithFields("HTTP request completed", map[string]interface{}{
		"method":   req.Method,
		"url":      req.URL.String(),
		"status":   resp.StatusCode,
		"duration": duration,
	})

	return resp, nil
}

// doRequestWithRetry performs an HTTP request with retry logic
func (c *Client) doRequestWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
	var lastErr error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Log retry attempt
			c.logger.WarnWithFields("retrying HTTP request", map[string]interface{}{
				"method":  req.Method,
				"url":     req.URL.String(),
				"attempt": attempt,
				"error":   lastErr.Error(),
			})
			
			// Wait before retry
			time.Sleep(time.Second * time.Duration(attempt))
		}
		
		resp, err := c.doRequest(req)
		if err != nil {
			lastErr = err
			
			// Check if it's a network error that's worth retrying
			if errVal, ok := err.(*Error); ok && errVal.Type == ErrorTypeNetwork {
				continue
			}
			return nil, err
		}
		
		// Check if response indicates we should retry
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			lastErr = &Error{
				Type:    ErrorTypeServerError,
				Message: fmt.Sprintf("server returned status %d", resp.StatusCode),
				Code:    resp.StatusCode,
			}
			resp.Body.Close()
			continue
		}
		
		// Success
		return resp, nil
	}
	
	c.logger.ErrorWithFields("max retries exceeded", map[string]interface{}{
		"method":      req.Method,
		"url":         req.URL.String(),
		"max_retries": maxRetries,
		"last_error":  lastErr.Error(),
	})
	
	return nil, lastErr
}

// Get performs a GET request to the specified URL
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, &Error{
			Type:    ErrorTypeUnknown,
			Message: fmt.Sprintf("failed to create request: %v", err),
			Code:    0,
		}
	}

	return c.doRequest(req)
}

// GetJSON performs a GET request and decodes the JSON response
func (c *Client) GetJSON(url string, target interface{}) error {
	resp, err := c.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check status code
	if err := c.checkResponseStatus(resp); err != nil {
		return err
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &Error{
			Type:    ErrorTypeNetwork,
			Message: fmt.Sprintf("failed to read response body: %v", err),
			Code:    resp.StatusCode,
		}
	}

	// Decode JSON
	if err := json.Unmarshal(body, target); err != nil {
		// Create a preview of the body for debugging
		bodyPreview := string(body)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		
		c.logger.ErrorWithFields("failed to parse JSON response", map[string]interface{}{
			"url":          url,
			"status":       resp.StatusCode,
			"error":        err.Error(),
			"body_preview": bodyPreview,
		})
		return &Error{
			Type:    ErrorTypeParsing,
			Message: fmt.Sprintf("failed to parse JSON: %v", err),
			Code:    resp.StatusCode,
		}
	}

	return nil
}

// checkResponseStatus checks the HTTP response status and returns appropriate errors
func (c *Client) checkResponseStatus(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		c.logger.WarnWithFields("authentication error", map[string]interface{}{
			"status": resp.StatusCode,
			"url":    resp.Request.URL.String(),
		})
		return &Error{
			Type:    ErrorTypeAuth,
			Message: "authentication required",
			Code:    resp.StatusCode,
		}
	case http.StatusNotFound:
		c.logger.WarnWithFields("resource not found", map[string]interface{}{
			"status": resp.StatusCode,
			"url":    resp.Request.URL.String(),
		})
		return &Error{
			Type:    ErrorTypeNotFound,
			Message: "resource not found",
			Code:    resp.StatusCode,
		}
	case http.StatusTooManyRequests:
		c.logger.WarnWithFields("rate limit exceeded", map[string]interface{}{
			"status": resp.StatusCode,
			"url":    resp.Request.URL.String(),
		})
		return &Error{
			Type:    ErrorTypeRateLimit,
			Message: "rate limit exceeded",
			Code:    resp.StatusCode,
		}
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		c.logger.ErrorWithFields("server error", map[string]interface{}{
			"status": resp.StatusCode,
			"url":    resp.Request.URL.String(),
		})
		return &Error{
			Type:    ErrorTypeServerError,
			Message: "server error",
			Code:    resp.StatusCode,
		}
	default:
		if resp.StatusCode >= 400 {
			c.logger.ErrorWithFields("unexpected API error", map[string]interface{}{
				"status": resp.StatusCode,
				"url":    resp.Request.URL.String(),
			})
			return &Error{
				Type:    ErrorTypeUnknown,
				Message: fmt.Sprintf("unexpected status code: %d", resp.StatusCode),
				Code:    resp.StatusCode,
			}
		}
		return nil
	}
}

// FetchUserProfile fetches the Instagram user profile data
func (c *Client) FetchUserProfile(username string) (*InstagramResponse, error) {
	url := GetProfileURL(username)
	
	c.logger.DebugWithFields("fetching user profile", map[string]interface{}{
		"username": username,
		"url":      url,
	})
	
	var response InstagramResponse
	if err := c.GetJSON(url, &response); err != nil {
		c.logger.ErrorWithFields("failed to fetch user profile", map[string]interface{}{
			"username": username,
			"error":    err.Error(),
		})
		return nil, err
	}

	// Check if login is required
	if response.RequiresToLogin {
		c.logger.WarnWithFields("authentication required for profile", map[string]interface{}{
			"username": username,
		})
		return nil, &Error{
			Type:    ErrorTypeAuth,
			Message: "Instagram requires authentication to view this profile",
			Code:    http.StatusUnauthorized,
		}
	}

	c.logger.DebugWithFields("successfully fetched user profile", map[string]interface{}{
		"username": username,
	})

	return &response, nil
}

// FetchUserMedia fetches paginated media for a user
func (c *Client) FetchUserMedia(userID string, after string) (*InstagramResponse, error) {
	url := GetMediaURL(userID, after)
	
	c.logger.DebugWithFields("fetching user media", map[string]interface{}{
		"user_id": userID,
		"after":   after,
		"url":     url,
	})
	
	var response InstagramResponse
	if err := c.GetJSON(url, &response); err != nil {
		c.logger.ErrorWithFields("failed to fetch user media", map[string]interface{}{
			"user_id": userID,
			"after":   after,
			"error":   err.Error(),
		})
		return nil, err
	}

	c.logger.DebugWithFields("successfully fetched user media", map[string]interface{}{
		"user_id": userID,
	})

	return &response, nil
}

// DownloadPhoto downloads a photo from the given URL
func (c *Client) DownloadPhoto(photoURL string) ([]byte, error) {
	c.logger.DebugWithFields("downloading photo", map[string]interface{}{
		"url": photoURL,
	})

	resp, err := c.Get(photoURL)
	if err != nil {
		c.logger.ErrorWithFields("failed to download photo", map[string]interface{}{
			"url":   photoURL,
			"error": err.Error(),
		})
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.ErrorWithFields("failed to read photo data", map[string]interface{}{
			"url":   photoURL,
			"error": err.Error(),
		})
		return nil, &Error{
			Type:    ErrorTypeNetwork,
			Message: fmt.Sprintf("failed to download photo: %v", err),
			Code:    0,
		}
	}

	c.logger.DebugWithFields("successfully downloaded photo", map[string]interface{}{
		"url":  photoURL,
		"size": len(data),
	})

	return data, nil
}