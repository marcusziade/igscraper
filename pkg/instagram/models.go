package instagram

// InstagramResponse represents the top-level response from Instagram API
type InstagramResponse struct {
	RequiresToLogin bool   `json:"requires_to_login"`
	Data            Data   `json:"data"`
	Status          string `json:"status"`
}

// Data wraps the user information in the response
type Data struct {
	User User `json:"user"`
}

// User represents an Instagram user profile
type User struct {
	ID                       string                   `json:"id"`
	EdgeOwnerToTimelineMedia EdgeOwnerToTimelineMedia `json:"edge_owner_to_timeline_media"`
}

// EdgeOwnerToTimelineMedia contains the user's media information
type EdgeOwnerToTimelineMedia struct {
	Count    int      `json:"count"`
	PageInfo PageInfo `json:"page_info"`
	Edges    []Edge   `json:"edges"`
}

// PageInfo contains pagination information
type PageInfo struct {
	HasNextPage bool   `json:"has_next_page"`
	EndCursor   string `json:"end_cursor"`
}

// Edge wraps a single media node
type Edge struct {
	Node Node `json:"node"`
}

// Node represents a single media item (photo or video)
type Node struct {
	ID         string `json:"id"`
	Shortcode  string `json:"shortcode"`
	DisplayURL string `json:"display_url"`
	IsVideo    bool   `json:"is_video"`
}