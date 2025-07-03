package scraper

import "igscraper/pkg/instagram"

// InstagramClient defines the interface for Instagram API operations
type InstagramClient interface {
	GetJSON(url string, target interface{}) error
	DownloadPhoto(photoURL string) ([]byte, error)
	FetchUserProfile(username string) (*instagram.InstagramResponse, error)
	FetchUserMedia(userID string, after string) (*instagram.InstagramResponse, error)
}