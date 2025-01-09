package models

type InstagramResponse struct {
	RequiresToLogin bool   `json:"requires_to_login"`
	Data            Data   `json:"data"`
	Status          string `json:"status"`
}

type Data struct {
	User User `json:"user"`
}

type User struct {
	ID                       string                   `json:"id"`
	EdgeOwnerToTimelineMedia EdgeOwnerToTimelineMedia `json:"edge_owner_to_timeline_media"`
}

type EdgeOwnerToTimelineMedia struct {
	Count    int      `json:"count"`
	PageInfo PageInfo `json:"page_info"`
	Edges    []Edge   `json:"edges"`
}

type PageInfo struct {
	HasNextPage bool   `json:"has_next_page"`
	EndCursor   string `json:"end_cursor"`
}

type Edge struct {
	Node Node `json:"node"`
}

type Node struct {
	ID         string `json:"id"`
	Shortcode  string `json:"shortcode"`
	DisplayURL string `json:"display_url"`
	IsVideo    bool   `json:"is_video"`
}
