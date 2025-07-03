package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"igscraper/pkg/instagram"
)

// PhotoMetadata represents all metadata for a downloaded photo
type PhotoMetadata struct {
	// Core identifiers
	ID        string `json:"id"`
	Shortcode string `json:"shortcode"`
	URL       string `json:"url"`
	
	// Media properties
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	IsVideo    bool   `json:"is_video"`
	FileSize   int64  `json:"file_size,omitempty"`
	
	// Timestamps
	TakenAt     time.Time `json:"taken_at"`
	DownloadedAt time.Time `json:"downloaded_at"`
	
	// Content
	Caption              string    `json:"caption,omitempty"`
	AccessibilityCaption string    `json:"accessibility_caption,omitempty"`
	Location             *Location `json:"location,omitempty"`
	
	// Engagement
	LikesCount    int `json:"likes_count"`
	CommentsCount int `json:"comments_count"`
	VideoViews    int `json:"video_views,omitempty"`
	
	// People
	Owner       Owner        `json:"owner"`
	TaggedUsers []TaggedUser `json:"tagged_users,omitempty"`
	
	// Settings
	CommentsDisabled bool `json:"comments_disabled"`
}

// Location represents geographic location
type Location struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Owner represents the media owner
type Owner struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// TaggedUser represents a tagged user with position
type TaggedUser struct {
	ID       string  `json:"id"`
	Username string  `json:"username"`
	FullName string  `json:"full_name"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
}

// FromInstagramNode converts Instagram API data to PhotoMetadata
func FromInstagramNode(node *instagram.Node, fileSize int64) *PhotoMetadata {
	meta := &PhotoMetadata{
		ID:           node.ID,
		Shortcode:    node.Shortcode,
		URL:          node.DisplayURL,
		Width:        node.Dimensions.Width,
		Height:       node.Dimensions.Height,
		IsVideo:      node.IsVideo,
		FileSize:     fileSize,
		TakenAt:      time.Unix(node.TakenAtTimestamp, 0),
		DownloadedAt: time.Now(),
		LikesCount:   node.EdgeLikedBy.Count,
		CommentsCount: node.EdgeMediaToComment.Count,
		AccessibilityCaption: node.AccessibilityCaption,
		CommentsDisabled: node.CommentsDisabled,
		Owner: Owner{
			ID:       node.Owner.ID,
			Username: node.Owner.Username,
		},
	}

	// Extract caption
	if len(node.EdgeMediaToCaption.Edges) > 0 {
		meta.Caption = node.EdgeMediaToCaption.Edges[0].Node.Text
	}

	// Extract location
	if node.Location != nil {
		meta.Location = &Location{
			ID:   node.Location.ID,
			Name: node.Location.Name,
			Slug: node.Location.Slug,
		}
	}

	// Extract video views
	if node.VideoViewCount != nil {
		meta.VideoViews = *node.VideoViewCount
	}

	// Extract tagged users
	for _, edge := range node.EdgeMediaToTaggedUser.Edges {
		meta.TaggedUsers = append(meta.TaggedUsers, TaggedUser{
			ID:       edge.Node.User.ID,
			Username: edge.Node.User.Username,
			FullName: edge.Node.User.FullName,
			X:        edge.Node.X,
			Y:        edge.Node.Y,
		})
	}

	return meta
}

// Save writes the metadata to a JSON file
func (m *PhotoMetadata) Save(photoPath string) error {
	metadataPath := photoPath + ".json"
	
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// Load reads metadata from a JSON file
func Load(photoPath string) (*PhotoMetadata, error) {
	metadataPath := photoPath + ".json"
	
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var meta PhotoMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &meta, nil
}

// GetFormattedCaption returns a truncated caption for display
func (m *PhotoMetadata) GetFormattedCaption(maxLength int) string {
	if m.Caption == "" {
		return ""
	}
	
	// Remove newlines for display
	caption := m.Caption
	if len(caption) > maxLength {
		caption = caption[:maxLength-3] + "..."
	}
	
	return caption
}

// GetAspectRatio returns the aspect ratio as a string
func (m *PhotoMetadata) GetAspectRatio() string {
	if m.Height == 0 {
		return "unknown"
	}
	
	ratio := float64(m.Width) / float64(m.Height)
	
	// Common aspect ratios
	switch {
	case ratio > 1.7 && ratio < 1.8:
		return "16:9"
	case ratio > 1.3 && ratio < 1.4:
		return "4:3"
	case ratio > 0.9 && ratio < 1.1:
		return "1:1"
	case ratio > 0.55 && ratio < 0.57:
		return "9:16"
	case ratio > 0.74 && ratio < 0.76:
		return "3:4"
	default:
		return fmt.Sprintf("%.2f:1", ratio)
	}
}

// MetadataExists checks if metadata file exists for a photo
func MetadataExists(photoPath string) bool {
	metadataPath := photoPath + ".json"
	_, err := os.Stat(metadataPath)
	return err == nil
}

// CleanOrphanedMetadata removes metadata files without corresponding photos
func CleanOrphanedMetadata(directory string) error {
	return filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if it's a metadata file
		if filepath.Ext(path) == ".json" && len(path) > 5 {
			photoPath := path[:len(path)-5] // Remove .json extension
			
			// Check if corresponding photo exists
			if _, err := os.Stat(photoPath); os.IsNotExist(err) {
				// Photo doesn't exist, remove metadata
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove orphaned metadata %s: %w", path, err)
				}
			}
		}

		return nil
	})
}