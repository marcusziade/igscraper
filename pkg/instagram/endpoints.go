package instagram

import (
	"fmt"
	"net/url"
)

const (
	// BaseURL is the base URL for Instagram
	BaseURL = "https://www.instagram.com"

	// ProfileEndpoint is the endpoint pattern for user profiles
	ProfileEndpoint = "/api/v1/users/web_profile_info/"

	// MediaEndpoint is the endpoint pattern for user media
	MediaEndpoint = "/graphql/query/"

	// MediaQueryHash is the query hash for fetching user media
	MediaQueryHash = "e769aa130647d2354c40ea6a439bfc08"

	// DefaultMediaLimit is the default number of media items to fetch per request
	DefaultMediaLimit = 12

	// MaxMediaLimit is the maximum number of media items that can be fetched per request
	MaxMediaLimit = 50
)

// GetProfileURL constructs the URL for fetching a user's profile
func GetProfileURL(username string) string {
	params := url.Values{}
	params.Set("username", username)
	
	return fmt.Sprintf("%s%s?%s", BaseURL, ProfileEndpoint, params.Encode())
}

// GetMediaURL constructs the URL for fetching a user's media with pagination
func GetMediaURL(userID string, after string) string {
	return GetMediaURLWithLimit(userID, after, DefaultMediaLimit)
}

// GetMediaURLWithLimit constructs the URL for fetching a user's media with custom limit
func GetMediaURLWithLimit(userID string, after string, limit int) string {
	// Ensure limit is within bounds
	if limit <= 0 {
		limit = DefaultMediaLimit
	} else if limit > MaxMediaLimit {
		limit = MaxMediaLimit
	}

	variables := map[string]interface{}{
		"id":    userID,
		"first": limit,
	}

	if after != "" {
		variables["after"] = after
	}

	params := url.Values{}
	params.Set("query_hash", MediaQueryHash)
	params.Set("variables", fmt.Sprintf(`{"id":"%s","first":%d,"after":"%s"}`, userID, limit, after))

	return fmt.Sprintf("%s%s?%s", BaseURL, MediaEndpoint, params.Encode())
}

// GetPhotoURL returns the direct URL for a photo
// This is typically the display_url from the Node struct
func GetPhotoURL(node *Node) string {
	if node == nil {
		return ""
	}
	return node.DisplayURL
}

// GetPostURL constructs the URL for a specific post
func GetPostURL(shortcode string) string {
	if shortcode == "" {
		return ""
	}
	return fmt.Sprintf("%s/p/%s/", BaseURL, shortcode)
}

// GetUserProfileURL constructs the public profile URL for a user
func GetUserProfileURL(username string) string {
	if username == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/", BaseURL, username)
}

// IsValidUsername checks if a username is valid according to Instagram rules
func IsValidUsername(username string) bool {
	if username == "" || len(username) > 30 {
		return false
	}

	// Instagram usernames can only contain letters, numbers, periods, and underscores
	for _, char := range username {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '.' || char == '_') {
			return false
		}
	}

	return true
}

// SanitizeUsername removes any invalid characters from a username
func SanitizeUsername(username string) string {
	if username == "" {
		return ""
	}

	// Remove @ symbol if present at the beginning
	if username[0] == '@' {
		username = username[1:]
	}

	// Remove any trailing slashes or spaces
	for len(username) > 0 && (username[len(username)-1] == '/' || username[len(username)-1] == ' ') {
		username = username[:len(username)-1]
	}

	return username
}