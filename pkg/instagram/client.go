package instagram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
		headers:     defaultBrowserHeaders(),
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
		headers:     defaultBrowserHeaders(),
		baseURL:     BaseURL,
		logger:      log,
		retrier:     retrier,
		retryConfig: retryConfig,
	}
}

func defaultBrowserHeaders() map[string]string {
	return map[string]string{
		"User-Agent":         "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36",
		"Accept":             "*/*",
		"Accept-Language":    "en-US,en;q=0.9",
		"Origin":             "https://www.instagram.com",
		"Referer":            "https://www.instagram.com/",
		"X-IG-App-ID":        "936619743392459",
		"X-Requested-With":   "XMLHttpRequest",
		"X-Instagram-AJAX":   "1",
		"Sec-Fetch-Dest":     "empty",
		"Sec-Fetch-Mode":     "cors",
		"Sec-Fetch-Site":     "same-origin",
	}
}

func responseURL(resp *http.Response) string {
	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		return resp.Request.URL.String()
	}
	return ""
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

		// 401/403/404: return response to caller for body inspection (soft rate-limits
		// often arrive as 401 with "Please wait a few minutes").
		if resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 404 {
			return nil
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
func (c *Client) GetJSON(reqURL string, target interface{}) error {
	resp, err := c.Get(reqURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.decodeJSONResponse(resp, reqURL, target)
}

// PostFormJSON performs a POST form request and decodes the JSON response
func (c *Client) PostFormJSON(reqURL string, form url.Values, target interface{}) error {
	req, err := http.NewRequest("POST", reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return &errors.Error{
			Type:    errors.ErrorTypeUnknown,
			Message: fmt.Sprintf("failed to create request: %v", err),
			Code:    0,
		}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.decodeJSONResponse(resp, reqURL, target)
}

func (c *Client) decodeJSONResponse(resp *http.Response, reqURL string, target interface{}) error {
	if resp.StatusCode == 302 || resp.StatusCode == 301 {
		c.logger.WarnWithFields("Instagram requires authentication", map[string]interface{}{
			"url":    reqURL,
			"status": resp.StatusCode,
		})
		return &errors.Error{
			Type:    errors.ErrorTypeAuth,
			Message: "Instagram session expired or invalid. Please login again with fresh credentials.",
			Code:    resp.StatusCode,
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &errors.Error{
			Type:    errors.ErrorTypeNetwork,
			Message: fmt.Sprintf("failed to read response body: %v", err),
			Code:    resp.StatusCode,
		}
	}

	if err := c.checkResponseStatusWithBody(resp, body); err != nil {
		return err
	}

	if err := json.Unmarshal(body, target); err != nil {
		bodyPreview := string(body)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}

		c.logger.ErrorWithFields("failed to parse JSON response", map[string]interface{}{
			"url":          reqURL,
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

func (c *Client) checkResponseStatus(resp *http.Response) error {
	return c.checkResponseStatusWithBody(resp, nil)
}

func (c *Client) checkResponseStatusWithBody(resp *http.Response, body []byte) error {
	reqURL := responseURL(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		msg := string(body)
		if isSoftRateLimitMessage(msg) {
			c.logger.WarnWithFields("rate limit (soft 401)", map[string]interface{}{
				"status": resp.StatusCode,
				"url":    reqURL,
			})
			return &errors.Error{
				Type:    errors.ErrorTypeRateLimit,
				Message: "rate limited by Instagram; wait a few minutes before retrying",
				Code:    resp.StatusCode,
			}
		}
		c.logger.WarnWithFields("authentication error", map[string]interface{}{
			"status": resp.StatusCode,
			"url":    reqURL,
		})
		return &errors.Error{
			Type:    errors.ErrorTypeAuth,
			Message: "authentication required",
			Code:    resp.StatusCode,
		}
	case http.StatusNotFound:
		c.logger.WarnWithFields("resource not found", map[string]interface{}{
			"status": resp.StatusCode,
			"url":    reqURL,
		})
		return &errors.Error{
			Type:    errors.ErrorTypeNotFound,
			Message: "resource not found",
			Code:    resp.StatusCode,
		}
	case http.StatusTooManyRequests:
		c.logger.WarnWithFields("rate limit exceeded", map[string]interface{}{
			"status": resp.StatusCode,
			"url":    reqURL,
		})
		return &errors.Error{
			Type:    errors.ErrorTypeRateLimit,
			Message: "rate limit exceeded",
			Code:    resp.StatusCode,
		}
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		c.logger.ErrorWithFields("server error", map[string]interface{}{
			"status": resp.StatusCode,
			"url":    reqURL,
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
				"url":    reqURL,
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

func isSoftRateLimitMessage(body string) bool {
	lower := strings.ToLower(body)
	return strings.Contains(lower, "please wait") ||
		strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "too many")
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

// FetchUserMedia fetches a page of media for a user.
// username enables the first-page profile seed and logged-in GraphQL path.
// after is either empty (first page), a feed max_id, or a GraphQL end_cursor.
func (c *Client) FetchUserMedia(userID string, after string) (*InstagramResponse, error) {
	return c.FetchUserMediaWithUsername(userID, "", after)
}

// FetchUserMediaWithUsername is like FetchUserMedia but uses username for better endpoints.
func (c *Client) FetchUserMediaWithUsername(userID, username, after string) (*InstagramResponse, error) {
	var lastErr error

	// First page: web_profile_info already returns timeline edges + GraphQL cursor.
	if after == "" && username != "" {
		c.logger.DebugWithFields("seeding media from profile endpoint", map[string]interface{}{
			"username": username,
			"user_id":  userID,
		})
		profile, err := c.FetchUserProfile(username)
		if err == nil && len(profile.Data.User.EdgeOwnerToTimelineMedia.Edges) > 0 {
			if profile.Data.User.ID == "" {
				profile.Data.User.ID = userID
			}
			c.logger.DebugWithFields("seeded media from profile", map[string]interface{}{
				"username":    username,
				"media_count": len(profile.Data.User.EdgeOwnerToTimelineMedia.Edges),
				"has_next":    profile.Data.User.EdgeOwnerToTimelineMedia.PageInfo.HasNextPage,
			})
			return profile, nil
		}
		if err != nil {
			lastErr = err
		}
	}

	// Feed REST API (max_id cursors look like "123_456").
	if after == "" || isFeedCursor(after) {
		resp, err := c.fetchMediaViaFeed(userID, after)
		if err == nil && len(resp.Data.User.EdgeOwnerToTimelineMedia.Edges) > 0 {
			return resp, nil
		}
		if err != nil {
			lastErr = err
			c.logger.DebugWithFields("feed API media fetch failed", map[string]interface{}{
				"user_id": userID,
				"error":   err.Error(),
			})
		}
	}

	// Modern GraphQL doc_id POST (what the web app / Instaloader use).
	resp, err := c.fetchMediaViaDocID(userID, username, after)
	if err == nil && (len(resp.Data.User.EdgeOwnerToTimelineMedia.Edges) > 0 || !resp.Data.User.EdgeOwnerToTimelineMedia.PageInfo.HasNextPage) {
		return resp, nil
	}
	if err != nil {
		lastErr = err
		c.logger.DebugWithFields("doc_id GraphQL media fetch failed", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
	}

	// Legacy query_hash GraphQL GET.
	legacyURL := GetMediaURL(userID, after)
	c.logger.DebugWithFields("falling back to legacy GraphQL query_hash", map[string]interface{}{
		"user_id": userID,
		"after":   after,
		"url":     legacyURL,
	})
	var response InstagramResponse
	if err := c.GetJSON(legacyURL, &response); err != nil {
		if lastErr != nil {
			return nil, lastErr
		}
		c.logger.ErrorWithFields("failed to fetch user media", map[string]interface{}{
			"user_id": userID,
			"after":   after,
			"error":   err.Error(),
		})
		return nil, err
	}

	c.logger.DebugWithFields("successfully fetched user media via legacy GraphQL", map[string]interface{}{
		"user_id":     userID,
		"after":       after,
		"media_count": len(response.Data.User.EdgeOwnerToTimelineMedia.Edges),
	})
	return &response, nil
}

func isFeedCursor(cursor string) bool {
	if cursor == "" {
		return false
	}
	// Feed max_id is typically "<media_id>_<user_id>" (digits + underscore).
	for _, r := range cursor {
		if (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return strings.Contains(cursor, "_")
}

func (c *Client) fetchMediaViaFeed(userID, after string) (*InstagramResponse, error) {
	reqURL := fmt.Sprintf("%s/api/v1/feed/user/%s/", BaseURL, userID)
	if after != "" {
		reqURL = fmt.Sprintf("%s?max_id=%s&count=%d", reqURL, url.QueryEscape(after), DefaultMediaLimit)
	} else {
		reqURL = fmt.Sprintf("%s?count=%d", reqURL, DefaultMediaLimit)
	}

	c.logger.DebugWithFields("fetching user media via feed API", map[string]interface{}{
		"user_id": userID,
		"after":   after,
		"url":     reqURL,
	})

	var feedResponse map[string]interface{}
	if err := c.GetJSON(reqURL, &feedResponse); err != nil {
		return nil, err
	}

	if status, ok := feedResponse["status"].(string); ok && status != "" && status != "ok" {
		msg, _ := feedResponse["message"].(string)
		if isSoftRateLimitMessage(msg) {
			return nil, &errors.Error{
				Type:    errors.ErrorTypeRateLimit,
				Message: "rate limited by Instagram; wait a few minutes before retrying",
				Code:    http.StatusTooManyRequests,
			}
		}
		return nil, &errors.Error{
			Type:    errors.ErrorTypeAuth,
			Message: fmt.Sprintf("feed API status %s: %s", status, msg),
			Code:    http.StatusUnauthorized,
		}
	}

	response := &InstagramResponse{
		Status: "ok",
		Data: Data{
			User: User{
				ID: userID,
				EdgeOwnerToTimelineMedia: EdgeOwnerToTimelineMedia{
					Edges: make([]Edge, 0),
				},
			},
		},
	}

	if items, ok := feedResponse["items"].([]interface{}); ok {
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				edge := c.parseFeedItem(itemMap)
				if edge != nil && edge.Node.DisplayURL != "" {
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
	response.Data.User.EdgeOwnerToTimelineMedia.PageInfo.EndCursor = stringifyID(feedResponse["next_max_id"])

	c.logger.DebugWithFields("successfully fetched user media via feed API", map[string]interface{}{
		"user_id":     userID,
		"after":       after,
		"media_count": len(response.Data.User.EdgeOwnerToTimelineMedia.Edges),
	})
	return response, nil
}

func (c *Client) fetchMediaViaDocID(userID, username, after string) (*InstagramResponse, error) {
	variables := map[string]interface{}{
		"data": map[string]interface{}{
			"count":                      DefaultMediaLimit,
			"include_relationship_info":  true,
			"latest_besties_reel_media":  true,
			"latest_reel_media":          true,
		},
		"__relay_internal__pv__PolarisFeedShareMenurelayprovider": false,
	}

	docID := MediaDocIDLoggedOut
	if username != "" {
		docID = MediaDocIDLoggedIn
		variables["username"] = username
	} else {
		variables["id"] = userID
	}

	if after != "" {
		variables["after"] = after
		variables["before"] = nil
		variables["first"] = DefaultMediaLimit
		variables["last"] = nil
	}

	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, &errors.Error{
			Type:    errors.ErrorTypeUnknown,
			Message: fmt.Sprintf("failed to encode GraphQL variables: %v", err),
			Code:    0,
		}
	}

	form := url.Values{}
	form.Set("variables", string(variablesJSON))
	form.Set("doc_id", docID)
	form.Set("server_timestamps", "true")

	reqURL := BaseURL + GraphQLEndpoint
	c.logger.DebugWithFields("fetching user media via doc_id GraphQL", map[string]interface{}{
		"user_id":  userID,
		"username": username,
		"after":    after,
		"doc_id":   docID,
	})

	var raw map[string]interface{}
	if err := c.PostFormJSON(reqURL, form, &raw); err != nil {
		return nil, err
	}

	if status, ok := raw["status"].(string); ok && status != "" && status != "ok" {
		msg, _ := raw["message"].(string)
		if isSoftRateLimitMessage(msg) {
			return nil, &errors.Error{
				Type:    errors.ErrorTypeRateLimit,
				Message: "rate limited by Instagram; wait a few minutes before retrying",
				Code:    http.StatusTooManyRequests,
			}
		}
		return nil, &errors.Error{
			Type:    errors.ErrorTypeAuth,
			Message: fmt.Sprintf("GraphQL status %s: %s", status, msg),
			Code:    http.StatusUnauthorized,
		}
	}

	connection := extractMediaConnection(raw)
	if connection == nil {
		return nil, &errors.Error{
			Type:    errors.ErrorTypeParsing,
			Message: "GraphQL response missing media connection",
			Code:    0,
		}
	}

	response := &InstagramResponse{
		Status: "ok",
		Data: Data{
			User: User{
				ID:                       userID,
				EdgeOwnerToTimelineMedia: *connection,
			},
		},
	}

	c.logger.DebugWithFields("successfully fetched user media via doc_id GraphQL", map[string]interface{}{
		"user_id":     userID,
		"after":       after,
		"media_count": len(response.Data.User.EdgeOwnerToTimelineMedia.Edges),
	})
	return response, nil
}

func extractMediaConnection(raw map[string]interface{}) *EdgeOwnerToTimelineMedia {
	data, _ := raw["data"].(map[string]interface{})
	if data == nil {
		return nil
	}

	candidates := []interface{}{
		data["xdt_api__v1__feed__user_timeline_graphql_connection"],
		data["user"],
	}
	if user, ok := data["user"].(map[string]interface{}); ok {
		candidates = append(candidates, user["edge_owner_to_timeline_media"])
	}

	for _, candidate := range candidates {
		connMap, ok := candidate.(map[string]interface{})
		if !ok {
			continue
		}
		// Nested under user
		if media, ok := connMap["edge_owner_to_timeline_media"].(map[string]interface{}); ok {
			connMap = media
		}
		if _, hasEdges := connMap["edges"]; !hasEdges {
			if _, hasPage := connMap["page_info"]; !hasPage {
				continue
			}
		}
		return parseConnectionMap(connMap)
	}
	return nil
}

func parseConnectionMap(conn map[string]interface{}) *EdgeOwnerToTimelineMedia {
	media := &EdgeOwnerToTimelineMedia{
		Edges: make([]Edge, 0),
	}

	if count, ok := conn["count"].(float64); ok {
		media.Count = int(count)
	}

	if pageInfo, ok := conn["page_info"].(map[string]interface{}); ok {
		if hasNext, ok := pageInfo["has_next_page"].(bool); ok {
			media.PageInfo.HasNextPage = hasNext
		}
		media.PageInfo.EndCursor = stringifyID(pageInfo["end_cursor"])
	}

	edges, _ := conn["edges"].([]interface{})
	for _, edgeVal := range edges {
		edgeMap, ok := edgeVal.(map[string]interface{})
		if !ok {
			continue
		}
		node, ok := edgeMap["node"].(map[string]interface{})
		if !ok {
			// Some connections return the node at the top level of each edge
			node = edgeMap
		}
		edge := parseMediaNode(node)
		if edge != nil && edge.Node.DisplayURL != "" {
			media.Edges = append(media.Edges, *edge)
		}
	}
	return media
}

func parseMediaNode(node map[string]interface{}) *Edge {
	// Private API / iPhone-style node
	_, hasImageVersions := node["image_versions2"]
	_, hasMediaType := node["media_type"]
	if hasImageVersions || (node["code"] != nil && hasMediaType) {
		return (*Client)(nil).parseFeedItem(node)
	}

	// Classic GraphQL node
	edge := &Edge{Node: Node{}}
	edge.Node.ID = stringifyID(node["id"])
	if sc, ok := node["shortcode"].(string); ok {
		edge.Node.Shortcode = sc
	}
	if urlStr, ok := node["display_url"].(string); ok {
		edge.Node.DisplayURL = urlStr
	} else if urlStr, ok := node["display_src"].(string); ok {
		edge.Node.DisplayURL = urlStr
	}
	if isVideo, ok := node["is_video"].(bool); ok {
		edge.Node.IsVideo = isVideo
	}
	if ts, ok := node["taken_at_timestamp"].(float64); ok {
		edge.Node.TakenAtTimestamp = int64(ts)
	} else if ts, ok := node["date"].(float64); ok {
		edge.Node.TakenAtTimestamp = int64(ts)
	}
	if dims, ok := node["dimensions"].(map[string]interface{}); ok {
		if h, ok := dims["height"].(float64); ok {
			edge.Node.Dimensions.Height = int(h)
		}
		if w, ok := dims["width"].(float64); ok {
			edge.Node.Dimensions.Width = int(w)
		}
	}
	if captionEdge, ok := node["edge_media_to_caption"].(map[string]interface{}); ok {
		if captionEdges, ok := captionEdge["edges"].([]interface{}); ok && len(captionEdges) > 0 {
			if ce, ok := captionEdges[0].(map[string]interface{}); ok {
				if cn, ok := ce["node"].(map[string]interface{}); ok {
					if text, ok := cn["text"].(string); ok {
						edge.Node.EdgeMediaToCaption = EdgeMediaToCaption{
							Edges: []CaptionEdge{{Node: CaptionNode{Text: text}}},
						}
					}
				}
			}
		}
	}
	if edge.Node.DisplayURL == "" && edge.Node.Shortcode == "" {
		return nil
	}
	return edge
}

func stringifyID(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case json.Number:
		return t.String()
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
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
	} else if sc, ok := item["shortcode"].(string); ok {
		edge.Node.Shortcode = sc
	}

	edge.Node.ID = stringifyID(item["id"])
	if edge.Node.ID == "" {
		edge.Node.ID = stringifyID(item["pk"])
	}

	if imageVersions, ok := item["image_versions2"].(map[string]interface{}); ok {
		if candidates, ok := imageVersions["candidates"].([]interface{}); ok && len(candidates) > 0 {
			// Prefer highest resolution candidate
			bestURL := ""
			bestPixels := 0
			for _, cand := range candidates {
				candidate, ok := cand.(map[string]interface{})
				if !ok {
					continue
				}
				urlStr, _ := candidate["url"].(string)
				w, _ := candidate["width"].(float64)
				h, _ := candidate["height"].(float64)
				pixels := int(w * h)
				if urlStr != "" && pixels >= bestPixels {
					bestPixels = pixels
					bestURL = urlStr
				}
			}
			edge.Node.DisplayURL = bestURL
		}
	}

	if edge.Node.DisplayURL == "" {
		if displayURI, ok := item["display_uri"].(string); ok {
			edge.Node.DisplayURL = displayURI
		} else if displayURL, ok := item["display_url"].(string); ok {
			edge.Node.DisplayURL = displayURL
		}
	}

	if carouselMedia, ok := item["carousel_media"].([]interface{}); ok && len(carouselMedia) > 0 {
		if edge.Node.DisplayURL == "" {
			if firstItem, ok := carouselMedia[0].(map[string]interface{}); ok {
				if imageVersions, ok := firstItem["image_versions2"].(map[string]interface{}); ok {
					if candidates, ok := imageVersions["candidates"].([]interface{}); ok && len(candidates) > 0 {
						if candidate, ok := candidates[0].(map[string]interface{}); ok {
							if urlStr, ok := candidate["url"].(string); ok {
								edge.Node.DisplayURL = urlStr
							}
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
	} else if isVideo, ok := item["is_video"].(bool); ok {
		edge.Node.IsVideo = isVideo
	}

	if originalWidth, ok := item["original_width"].(float64); ok {
		edge.Node.Dimensions.Width = int(originalWidth)
	}
	if originalHeight, ok := item["original_height"].(float64); ok {
		edge.Node.Dimensions.Height = int(originalHeight)
	}

	return edge
}
