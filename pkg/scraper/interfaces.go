package scraper

import (
	"net/http"

	"igscraper/pkg/instagram"
)

// InstagramClient defines the interface for Instagram API operations
type InstagramClient interface {
	Get(url string) (*http.Response, error)
	GetJSON(url string, target interface{}) error
	DownloadPhoto(photoURL string) ([]byte, error)
	FetchUserProfile(username string) (*instagram.InstagramResponse, error)
	FetchUserMedia(userID string, after string) (*instagram.InstagramResponse, error)
}
