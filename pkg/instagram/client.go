package instagram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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
}

// NewClient creates a new Instagram API client
func NewClient(timeout time.Duration) *Client {
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{
			Type:    ErrorTypeNetwork,
			Message: fmt.Sprintf("network error: %v", err),
			Code:    0,
		}
	}

	return resp, nil
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
		return &Error{
			Type:    ErrorTypeAuth,
			Message: "authentication required",
			Code:    resp.StatusCode,
		}
	case http.StatusNotFound:
		return &Error{
			Type:    ErrorTypeNotFound,
			Message: "resource not found",
			Code:    resp.StatusCode,
		}
	case http.StatusTooManyRequests:
		return &Error{
			Type:    ErrorTypeRateLimit,
			Message: "rate limit exceeded",
			Code:    resp.StatusCode,
		}
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return &Error{
			Type:    ErrorTypeServerError,
			Message: "server error",
			Code:    resp.StatusCode,
		}
	default:
		if resp.StatusCode >= 400 {
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
	
	var response InstagramResponse
	if err := c.GetJSON(url, &response); err != nil {
		return nil, err
	}

	// Check if login is required
	if response.RequiresToLogin {
		return nil, &Error{
			Type:    ErrorTypeAuth,
			Message: "Instagram requires authentication to view this profile",
			Code:    http.StatusUnauthorized,
		}
	}

	return &response, nil
}

// FetchUserMedia fetches paginated media for a user
func (c *Client) FetchUserMedia(userID string, after string) (*InstagramResponse, error) {
	url := GetMediaURL(userID, after)
	
	var response InstagramResponse
	if err := c.GetJSON(url, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// DownloadPhoto downloads a photo from the given URL
func (c *Client) DownloadPhoto(photoURL string) ([]byte, error) {
	resp, err := c.Get(photoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkResponseStatus(resp); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &Error{
			Type:    ErrorTypeNetwork,
			Message: fmt.Sprintf("failed to download photo: %v", err),
			Code:    0,
		}
	}

	return data, nil
}