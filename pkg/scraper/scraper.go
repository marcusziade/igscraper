package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"igscraper/pkg/config"
	"igscraper/pkg/instagram"
	"igscraper/pkg/ratelimit"
	"igscraper/pkg/storage"
	"igscraper/pkg/ui"
)

const (
	retryDelay = time.Second * 2
)

// Scraper orchestrates the Instagram photo download process
type Scraper struct {
	client         *instagram.Client
	storageManager *storage.Manager
	rateLimiter    ratelimit.Limiter
	tracker        *ui.StatusTracker
	notifier       *ui.Notifier
	config         *config.Config
}

// New creates a new Scraper instance
func New(cfg *config.Config) (*Scraper, error) {
	// Create Instagram client
	client := instagram.NewClient(cfg.Download.DownloadTimeout)
	if cfg.Instagram.SessionID != "" {
		client.SetHeader("sessionid", cfg.Instagram.SessionID)
	}
	if cfg.Instagram.CSRFToken != "" {
		client.SetHeader("x-csrftoken", cfg.Instagram.CSRFToken)
	}
	if cfg.Instagram.UserAgent != "" {
		client.SetHeader("User-Agent", cfg.Instagram.UserAgent)
	}

	// Rate limiter based on config
	var rateLimiter ratelimit.Limiter
	if cfg.RateLimit.RequestsPerMinute > 0 {
		rateLimiter = ratelimit.NewTokenBucket(
			cfg.RateLimit.RequestsPerMinute,
			time.Minute,
		)
	} else {
		rateLimiter = ratelimit.NewTokenBucket(60, time.Minute) // Default 60/min
	}

	return &Scraper{
		client:      client,
		rateLimiter: rateLimiter,
		tracker:     ui.NewStatusTracker(),
		notifier:    ui.NewNotifier(),
		config:      cfg,
	}, nil
}

// getOutputDir determines the output directory for a username
func (s *Scraper) getOutputDir(username string) string {
	if s.config.Output.CreateUserFolders {
		return filepath.Join(s.config.Output.BaseDirectory, username+"_photos")
	}
	return s.config.Output.BaseDirectory
}

// DownloadUserPhotos downloads all photos from a user's profile
func (s *Scraper) DownloadUserPhotos(username string) error {
	ui.PrintHighlight("\n[INITIATING EXTRACTION SEQUENCE]\n")
	
	// Setup output directory
	outputDir := s.getOutputDir(username)
	storageManager, err := storage.NewManager(outputDir)
	if err != nil {
		return fmt.Errorf("failed to create storage manager: %w", err)
	}
	s.storageManager = storageManager
	
	// Get initial user data
	userID, err := s.getUserID(username)
	if err != nil {
		return fmt.Errorf("failed to get user ID: %w", err)
	}

	hasMore := true
	endCursor := ""

	for hasMore {
		s.tracker.PrintBatchStatus()

		// Rate limit check
		if !s.rateLimiter.Allow() {
			s.notifier.SendNotification("RATE LIMIT", "Cooling down for 1 hour...")
			ui.PrintWarning("\n[COOLING DOWN FOR 1 HOUR]\n")
			s.rateLimiter.Wait()
			s.notifier.SendNotification("RESUMING", "Continuing extraction process")
		}

		// Fetch media batch
		media, pageInfo, err := s.fetchMediaBatch(username, userID, endCursor)
		if err != nil {
			ui.PrintError("\nError fetching media: %v. Retrying...\n", err)
			time.Sleep(retryDelay)
			continue
		}

		// Process media items
		for _, edge := range media {
			if edge.Node.IsVideo {
				continue
			}

			if s.storageManager.IsDownloaded(edge.Node.Shortcode) {
				continue
			}

			err := s.downloadPhoto(edge.Node.DisplayURL, edge.Node.Shortcode)
			if err != nil {
				ui.PrintError("\nError downloading %s: %v\n", edge.Node.Shortcode, err)
				continue
			}

			s.tracker.IncrementDownloaded()
			s.tracker.PrintProgress()
		}

		// Handle pagination
		if pageInfo.HasNextPage {
			endCursor = pageInfo.EndCursor
		} else {
			hasMore = false
		}
	}

	ui.PrintSuccess("\n[EXTRACTION COMPLETED SUCCESSFULLY]\n")
	return nil
}

// getUserID fetches the user ID for the given username
func (s *Scraper) getUserID(username string) (string, error) {
	endpoint := fmt.Sprintf("https://www.instagram.com/api/v1/users/web_profile_info/?username=%s", username)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	// Set headers from client
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	if s.config.Instagram.SessionID != "" {
		req.Header.Set("Cookie", fmt.Sprintf("sessionid=%s", s.config.Instagram.SessionID))
	}
	if s.config.Instagram.CSRFToken != "" {
		req.Header.Set("x-csrftoken", s.config.Instagram.CSRFToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return "", fmt.Errorf("authentication required or invalid credentials")
		}
		return "", fmt.Errorf("received status code: %d", resp.StatusCode)
	}

	var result instagram.InstagramResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("error decoding JSON: %w", err)
	}

	if result.RequiresToLogin {
		return "", fmt.Errorf("this profile requires authentication")
	}

	return result.Data.User.ID, nil
}

// fetchMediaBatch fetches a batch of media items
func (s *Scraper) fetchMediaBatch(username, userID, endCursor string) ([]instagram.Edge, instagram.PageInfo, error) {
	var endpoint string
	if endCursor == "" {
		endpoint = fmt.Sprintf("https://www.instagram.com/api/v1/users/web_profile_info/?username=%s", username)
	} else {
		variables := fmt.Sprintf(`{"id":"%s","first":50,"after":"%s"}`, userID, endCursor)
		endpoint = fmt.Sprintf("https://www.instagram.com/graphql/query/?query_hash=69cba40317214236af40e7efa697781d&variables=%s", variables)
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, instagram.PageInfo{}, fmt.Errorf("error creating request: %w", err)
	}
	// Set headers from client
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	if s.config.Instagram.SessionID != "" {
		req.Header.Set("Cookie", fmt.Sprintf("sessionid=%s", s.config.Instagram.SessionID))
	}
	if s.config.Instagram.CSRFToken != "" {
		req.Header.Set("x-csrftoken", s.config.Instagram.CSRFToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, instagram.PageInfo{}, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, instagram.PageInfo{}, fmt.Errorf("received status code: %d", resp.StatusCode)
	}

	var result instagram.InstagramResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, instagram.PageInfo{}, fmt.Errorf("error decoding JSON: %w", err)
	}

	media := result.Data.User.EdgeOwnerToTimelineMedia
	return media.Edges, media.PageInfo, nil
}

// downloadPhoto downloads a single photo
func (s *Scraper) downloadPhoto(url, shortcode string) error {
	data, err := s.client.DownloadPhoto(url)
	if err != nil {
		return fmt.Errorf("failed to download photo: %w", err)
	}

	// SavePhoto expects shortcode, not filename
	return s.storageManager.SavePhoto(bytes.NewReader(data), shortcode)
}

// generateFilename generates a filename based on the configured pattern
func (s *Scraper) generateFilename(shortcode string) string {
	pattern := s.config.Output.FileNamePattern
	if pattern == "" {
		pattern = "{shortcode}.jpg"
	}
	
	// Replace placeholders
	filename := strings.ReplaceAll(pattern, "{shortcode}", shortcode)
	filename = strings.ReplaceAll(filename, "{timestamp}", fmt.Sprintf("%d", time.Now().Unix()))
	filename = strings.ReplaceAll(filename, "{date}", time.Now().Format("2006-01-02"))
	
	// Ensure proper extension
	if !strings.Contains(filename, ".") {
		filename += ".jpg"
	}
	
	return filename
}
