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
	ID                    string               `json:"id"`
	Shortcode             string               `json:"shortcode"`
	DisplayURL            string               `json:"display_url"`
	IsVideo               bool                 `json:"is_video"`
	TakenAtTimestamp      int64                `json:"taken_at_timestamp"`
	Dimensions            MediaDimensions      `json:"dimensions"`
	EdgeMediaToCaption    EdgeMediaToCaption   `json:"edge_media_to_caption"`
	EdgeLikedBy           EdgeLikedBy          `json:"edge_liked_by"`
	EdgeMediaToComment    EdgeMediaToComment   `json:"edge_media_to_comment"`
	Location              *Location            `json:"location,omitempty"`
	Owner                 Owner                `json:"owner"`
	AccessibilityCaption  string               `json:"accessibility_caption,omitempty"`
	VideoViewCount        *int                 `json:"video_view_count,omitempty"`
	VideoDuration         *float64             `json:"video_duration,omitempty"`
	EdgeMediaToTaggedUser EdgeMediaToTaggedUser `json:"edge_media_to_tagged_user"`
	CommentsDisabled      bool                 `json:"comments_disabled"`
}

// MediaDimensions represents the dimensions of the media
type MediaDimensions struct {
	Height int `json:"height"`
	Width  int `json:"width"`
}

// EdgeMediaToCaption contains caption information
type EdgeMediaToCaption struct {
	Edges []CaptionEdge `json:"edges"`
}

// CaptionEdge wraps a caption node
type CaptionEdge struct {
	Node CaptionNode `json:"node"`
}

// CaptionNode contains the caption text
type CaptionNode struct {
	Text string `json:"text"`
}

// EdgeLikedBy contains like count information
type EdgeLikedBy struct {
	Count int `json:"count"`
}

// EdgeMediaToComment contains comment count information
type EdgeMediaToComment struct {
	Count int `json:"count"`
}

// Location represents geographic location
type Location struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	HasPublicPage bool   `json:"has_public_page"`
}

// Owner represents the media owner
type Owner struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// EdgeMediaToTaggedUser contains tagged users information
type EdgeMediaToTaggedUser struct {
	Edges []TaggedUserEdge `json:"edges"`
}

// TaggedUserEdge wraps a tagged user node
type TaggedUserEdge struct {
	Node TaggedUserNode `json:"node"`
}

// TaggedUserNode contains tagged user information
type TaggedUserNode struct {
	User TaggedUser `json:"user"`
	X    float64    `json:"x"`
	Y    float64    `json:"y"`
}

// TaggedUser represents a tagged user
type TaggedUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
}