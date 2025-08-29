package instagram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"igscraper/pkg/config"
	"igscraper/pkg/errors"
	"igscraper/pkg/logger"
	"igscraper/pkg/retry"
)

// Re-export error types for backward compatibility
type Error = errors.Error
type ErrorType = errors.ErrorType

const (
	ErrorTypeNetwork     = errors.ErrorTypeNetwork
	ErrorTypeRateLimit   = errors.ErrorTypeRateLimit
	ErrorTypeAuth        = errors.ErrorTypeAuth
	ErrorTypeParsing     = errors.ErrorTypeParsing
	ErrorTypeNotFound    = errors.ErrorTypeNotFound
	ErrorTypeServerError = errors.ErrorTypeServerError
	ErrorTypeUnknown     = errors.ErrorTypeUnknown
)

// Client represents an Instagram API client
type Client struct {
	httpClient  *http.Client
	headers     map[string]string
	baseURL     string
	logger      logger.Logger
	retrier     *retry.HTTPRetrier
	retryConfig *config.RetryConfig
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
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 1 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		headers: map[string]string{
			"User-Agent":       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Accept":           "*/*",
			"Accept-Language":  "en-US,en;q=0.5",
			"X-IG-App-ID":      "936619743392459",
			"X-Requested-With": "XMLHttpRequest",
			"Referer":          "https://www.instagram.com/",
		},
		baseURL:     BaseURL,
		logger:      log,
		retrier:     retry.NewHTTPRetrier(3, log),
		retryConfig: nil,
	}
}

// NewClientWithConfig creates a new Instagram API client with retry configuration
func NewClientWithConfig(timeout time.Duration, retryConfig *config.RetryConfig, log logger.Logger) *Client {
	// Use default logger if none provided
	if log == nil {
		log = logger.GetLogger()
	}

	var retrier *retry.HTTPRetrier
	if retryConfig != nil && retryConfig.Enabled {
		retrier = retry.NewHTTPRetrier(retryConfig.MaxAttempts, log)
	} else {
		retrier = retry.NewHTTPRetrier(0, log)
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 1 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		headers: map[string]string{
			"User-Agent":       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Accept":           "*/*",
			"Accept-Language":  "en-US,en;q=0.5",
			"X-IG-App-ID":      "936619743392459",
			"X-Requested-With": "XMLHttpRequest",
			"Referer":          "https://www.instagram.com/",
		},
		baseURL:     BaseURL,
		logger:      log,
		retrier:     retrier,
		retryConfig: retryConfig,
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

func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	start := time.Now()
	headerLog := make(map[string]string)
	for k, v := range req.Header {
		if k == "Cookie" && len(v) > 0 {
			if len(v[0]) > 50 {
				headerLog[k] = v[0][:50] + "..."
			} else {
				headerLog[k] = v[0]
			}
		} else if len(v) > 0 {
			headerLog[k] = v[0]
		}
	}
	c.logger.DebugWithFields("sending HTTP request", map[string]interface{}{
		"method":  req.Method,
		"url":     req.URL.String(),
		"headers": headerLog,
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
		return nil, &errors.Error{
			Type:    errors.ErrorTypeNetwork,
			Message: fmt.Sprintf("network error: %v", err),
			Code:    0,
		}
	}

	c.logger.DebugWithFields("HTTP request completed", map[string]interface{}{
		"method":   req.Method,
		"url":      req.URL.String(),
		"status":   resp.StatusCode,
		"duration": duration,
	})

	return resp, nil
}

func (c *Client) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	if c.retrier == nil || (c.retryConfig != nil && !c.retryConfig.Enabled) {
		return c.doRequest(req)
	}

	var resp *http.Response
	var lastErr error

	err := c.retrier.DoWithErrorType(func() error {
		var err error
		resp, err = c.doRequest(req)
		if err != nil {
			lastErr = err
			return err
		}

		// Check if response indicates we should retry
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			lastErr = &errors.Error{
				Type:    errors.ErrorTypeServerError,
				Message: fmt.Sprintf("server returned status %d", resp.StatusCode),
				Code:    resp.StatusCode,
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				lastErr.(*errors.Error).Type = errors.ErrorTypeRateLimit
			}
			resp.Body.Close()
			return lastErr
		}

		// Check for other errors that shouldn't be retried
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			lastErr = &errors.Error{
				Type:    errors.ErrorTypeAuth,
				Message: fmt.Sprintf("authentication error: %d", resp.StatusCode),
				Code:    resp.StatusCode,
			}
			return lastErr
		}

		if resp.StatusCode == 404 {
			lastErr = &errors.Error{
				Type:    errors.ErrorTypeNotFound,
				Message: "resource not found",
				Code:    resp.StatusCode,
			}
			return lastErr
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// Get performs a GET request to the specified URL
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, &errors.Error{
			Type:    errors.ErrorTypeUnknown,
			Message: fmt.Sprintf("failed to create request: %v", err),
			Code:    0,
		}
	}

	return c.doRequestWithRetry(req)
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
		return &errors.Error{
			Type:    errors.ErrorTypeNetwork,
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
		return &errors.Error{
			Type:    errors.ErrorTypeParsing,
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
		return &errors.Error{
			Type:    errors.ErrorTypeAuth,
			Message: "authentication required",
			Code:    resp.StatusCode,
		}
	case http.StatusNotFound:
		c.logger.WarnWithFields("resource not found", map[string]interface{}{
			"status": resp.StatusCode,
			"url":    resp.Request.URL.String(),
		})
		return &errors.Error{
			Type:    errors.ErrorTypeNotFound,
			Message: "resource not found",
			Code:    resp.StatusCode,
		}
	case http.StatusTooManyRequests:
		c.logger.WarnWithFields("rate limit exceeded", map[string]interface{}{
			"status": resp.StatusCode,
			"url":    resp.Request.URL.String(),
		})
		return &errors.Error{
			Type:    errors.ErrorTypeRateLimit,
			Message: "rate limit exceeded",
			Code:    resp.StatusCode,
		}
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		c.logger.ErrorWithFields("server error", map[string]interface{}{
			"status": resp.StatusCode,
			"url":    resp.Request.URL.String(),
		})
		return &errors.Error{
			Type:    errors.ErrorTypeServerError,
			Message: "server error",
			Code:    resp.StatusCode,
		}
	default:
		if resp.StatusCode >= 400 {
			c.logger.ErrorWithFields("unexpected API error", map[string]interface{}{
				"status": resp.StatusCode,
				"url":    resp.Request.URL.String(),
			})
			return &errors.Error{
				Type:    errors.ErrorTypeUnknown,
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
		return nil, &errors.Error{
			Type:    errors.ErrorTypeAuth,
			Message: "Instagram requires authentication to view this profile",
			Code:    http.StatusUnauthorized,
		}
	}

	c.logger.DebugWithFields("successfully fetched user profile", map[string]interface{}{
		"username": username,
	})

	return &response, nil
}

func (c *Client) FetchUserMedia(userID string, after string) (*InstagramResponse, error) {
	url := fmt.Sprintf("%s/api/v1/feed/user/%s/", BaseURL, userID)
	if after != "" {
		url = fmt.Sprintf("%s?max_id=%s", url, after)
	}

	c.logger.DebugWithFields("fetching user media via feed API", map[string]interface{}{
		"user_id": userID,
		"after":   after,
		"url":     url,
	})

	var feedResponse map[string]interface{}
	if err := c.GetJSON(url, &feedResponse); err == nil {
		response := &InstagramResponse{
			Status: "ok",
			Data: Data{
				User: User{
					ID:                       userID,
					EdgeOwnerToTimelineMedia: EdgeOwnerToTimelineMedia{},
				},
			},
		}

		if items, ok := feedResponse["items"].([]interface{}); ok {
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					edge := c.parseFeedItem(itemMap)
					if edge != nil {
						response.Data.User.EdgeOwnerToTimelineMedia.Edges = append(
							response.Data.User.EdgeOwnerToTimelineMedia.Edges,
							*edge,
						)
					}
				}
			}
		}

		if moreAvailable, ok := feedResponse["more_available"].(bool); ok {
			response.Data.User.EdgeOwnerToTimelineMedia.PageInfo.HasNextPage = moreAvailable
		}
		if nextMaxID, ok := feedResponse["next_max_id"].(string); ok {
			response.Data.User.EdgeOwnerToTimelineMedia.PageInfo.EndCursor = nextMaxID
		}

		c.logger.DebugWithFields("successfully fetched user media via feed API", map[string]interface{}{
			"user_id":     userID,
			"after":       after,
			"media_count": len(response.Data.User.EdgeOwnerToTimelineMedia.Edges),
		})

		return response, nil
	}

	url = GetMediaURL(userID, after)

	c.logger.DebugWithFields("falling back to GraphQL API", map[string]interface{}{
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
		"user_id":     userID,
		"after":       after,
		"media_count": len(response.Data.User.EdgeOwnerToTimelineMedia.Edges),
	})

	return &response, nil
}

// DownloadPhoto downloads a photo from the given URL
func (c *Client) DownloadPhoto(photoURL string) ([]byte, error) {
	resp, err := c.Get(photoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, &errors.Error{
			Type:    errors.ErrorTypeNetwork,
			Message: fmt.Sprintf("failed to download photo: HTTP %d", resp.StatusCode),
			Code:    resp.StatusCode,
		}
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &errors.Error{
			Type:    errors.ErrorTypeNetwork,
			Message: fmt.Sprintf("failed to read photo data: %v", err),
			Code:    resp.StatusCode,
		}
	}

	return data, nil
}

func (c *Client) parseFeedItem(item map[string]interface{}) *Edge {
	edge := &Edge{
		Node: Node{},
	}

	if code, ok := item["code"].(string); ok {
		edge.Node.Shortcode = code
	}

	if id, ok := item["id"].(string); ok {
		edge.Node.ID = id
	} else if pk, ok := item["pk"].(string); ok {
		edge.Node.ID = pk
	}

	if imageVersions, ok := item["image_versions2"].(map[string]interface{}); ok {
		if candidates, ok := imageVersions["candidates"].([]interface{}); ok && len(candidates) > 0 {
			if candidate, ok := candidates[0].(map[string]interface{}); ok {
				if url, ok := candidate["url"].(string); ok {
					edge.Node.DisplayURL = url
				}
			}
		}
	}

	if carouselMedia, ok := item["carousel_media"].([]interface{}); ok && len(carouselMedia) > 0 {
		if firstItem, ok := carouselMedia[0].(map[string]interface{}); ok {
			if imageVersions, ok := firstItem["image_versions2"].(map[string]interface{}); ok {
				if candidates, ok := imageVersions["candidates"].([]interface{}); ok && len(candidates) > 0 {
					if candidate, ok := candidates[0].(map[string]interface{}); ok {
						if url, ok := candidate["url"].(string); ok {
							edge.Node.DisplayURL = url
						}
					}
				}
			}
		}
	}

	if caption, ok := item["caption"].(map[string]interface{}); ok {
		if text, ok := caption["text"].(string); ok {
			edge.Node.EdgeMediaToCaption = EdgeMediaToCaption{
				Edges: []CaptionEdge{
					{Node: CaptionNode{Text: text}},
				},
			}
		}
	}

	if takenAt, ok := item["taken_at"].(float64); ok {
		edge.Node.TakenAtTimestamp = int64(takenAt)
	}

	if mediaType, ok := item["media_type"].(float64); ok {
		edge.Node.IsVideo = mediaType == 2
	}

	return edge
}
