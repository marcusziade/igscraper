package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"igscraper/pkg/models"
)

const (
	retryDelay = time.Second * 2
	maxPerHour = 100 // Conservative limit
	asciiLogo  = `
    ╔══════════════════════════════════════════════════════════════╗
    ║ ██╗███╗   ██╗███████╗████████╗ █████╗  ██████╗ ██████╗  █████╗  ║
    ║ ██║████╗  ██║██╔════╝╚══██╔══╝██╔══██╗██╔════╝ ██╔══██╗██╔══██╗ ║
    ║ ██║██╔██╗ ██║███████╗   ██║   ███████║██║  ███╗██████╔╝███████║ ║
    ║ ██║██║╚██╗██║╚════██║   ██║   ██╔══██║██║   ██║██╔══██╗██╔══██║ ║
    ║ ██║██║ ╚████║███████║   ██║   ██║  ██║╚██████╔╝██║  ██║██║  ██║ ║
    ║ ╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝ ║
    ║        NETRUNNER EDITION - PHOTO EXTRACTION UTILITY v2.0        ║
    ╚══════════════════════════════════════════════════════════════╝
`
	progressBar   = "█"
	progressEmpty = "░"
)

// Colors for terminal output
var (
	cyan    = colorize("\033[36m%s\033[0m")
	yellow  = colorize("\033[33m%s\033[0m")
	red     = colorize("\033[31m%s\033[0m")
	green   = colorize("\033[32m%s\033[0m")
	magenta = colorize("\033[35m%s\033[0m")
)

func colorize(colorString string) func(string) string {
	return func(text string) string {
		return fmt.Sprintf(colorString, text)
	}
}

// StatusTracker keeps track of download progress
type StatusTracker struct {
	totalDownloaded int
	currentBatch    int
	startTime       time.Time
}

func NewStatusTracker() *StatusTracker {
	return &StatusTracker{
		startTime: time.Now(),
	}
}

func (st *StatusTracker) incrementDownloaded() {
	st.totalDownloaded++
	st.currentBatch++
}

func (st *StatusTracker) resetBatch() {
	st.currentBatch = 0
}

func (st *StatusTracker) getBatchProgress() string {
	const width = 20
	progress := float64(st.currentBatch) / float64(maxPerHour)
	filled := int(progress * float64(width))

	bar := strings.Repeat(progressBar, filled) +
		strings.Repeat(progressEmpty, width-filled)

	return fmt.Sprintf("[%s] %d/100", bar, st.currentBatch)
}

func sendNotification(title, message string) {
	// For Linux
	exec.Command("notify-send", title, message).Run()
	// For macOS
	// exec.Command("osascript", "-e", fmt.Sprintf('display notification "%s" with title "%s"', message, title)).Run()

	fmt.Printf("\n%s: %s\n", cyan(title), yellow(message))
}

func main() {
	fmt.Print(cyan(asciiLogo))

	if len(os.Args) != 2 {
		fmt.Println(red("Usage: go run main.go <instagram_username>"))
		return
	}

	username := os.Args[1]
	fmt.Printf("%s: %s\n", cyan("Target Profile"), yellow(username))

	outputDir := fmt.Sprintf("%s_photos", username)
	err := os.MkdirAll(outputDir, 0o755)
	if err != nil {
		fmt.Printf(red("Error creating directory: %v\n"), err)
		return
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	headers := http.Header{
		"User-Agent":       []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"},
		"Accept":           []string{"*/*"},
		"Accept-Language":  []string{"en-US,en;q=0.5"},
		"X-IG-App-ID":      []string{"936619743392459"},
		"X-Requested-With": []string{"XMLHttpRequest"},
		"Connection":       []string{"keep-alive"},
		"Referer":          []string{"https://www.instagram.com/"},
		"Cookie": []string{
			"sessionid=YOUR_SESSION_ID;",
			"csrftoken=YOUR_CSRF_TOKEN;",
		},
	}

	fmt.Println(magenta("\n[INITIATING EXTRACTION SEQUENCE]"))
	err = downloadPhotos(client, headers, username, outputDir)
	if err != nil {
		fmt.Printf(red("\nEXTRACTION FAILED: %v\n"), err)
		return
	}

	fmt.Println(green("\n[EXTRACTION COMPLETED SUCCESSFULLY]"))
}

func downloadPhotos(client *http.Client, headers http.Header, username, outputDir string) error {
	hasMore := true
	endCursor := ""
	downloadedPhotos := make(map[string]bool)
	tracker := NewStatusTracker()

	// First request to get the user ID
	endpoint := fmt.Sprintf("https://www.instagram.com/api/v1/users/web_profile_info/?username=%s", username)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header = headers

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making initial request: %v", err)
	}
	defer resp.Body.Close()

	var result models.InstagramResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("error decoding initial JSON: %v", err)
	}

	userId := result.Data.User.ID

	for hasMore {
		var currentEndpoint string
		if endCursor == "" {
			currentEndpoint = endpoint
		} else {
			variables := fmt.Sprintf(`{"id":"%s","first":50,"after":"%s"}`, userId, endCursor)
			currentEndpoint = fmt.Sprintf("https://www.instagram.com/graphql/query/?query_hash=69cba40317214236af40e7efa697781d&variables=%s", variables)
		}

		fmt.Printf("\n%s %s\n", magenta("[SCANNING]"), yellow(tracker.getBatchProgress()))

		req, err := http.NewRequest("GET", currentEndpoint, nil)
		if err != nil {
			return fmt.Errorf("error creating request: %v", err)
		}
		req.Header = headers

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf(red("\nConnection error: %v. Retrying...\n"), err)
			time.Sleep(retryDelay)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == 401 || resp.StatusCode == 403 {
				return fmt.Errorf("authentication required or invalid credentials")
			}
			time.Sleep(retryDelay)
			continue
		}

		var result models.InstagramResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("error decoding JSON: %v", err)
		}

		if result.RequiresToLogin {
			return fmt.Errorf("this profile requires authentication")
		}

		media := result.Data.User.EdgeOwnerToTimelineMedia

		// Process media items
		for _, edge := range media.Edges {
			if !edge.Node.IsVideo && !downloadedPhotos[edge.Node.Shortcode] {
				// Check batch limit
				if tracker.currentBatch >= maxPerHour {
					sendNotification("RATE LIMIT", "Cooling down for 1 hour...")
					fmt.Println(yellow("\n[COOLING DOWN FOR 1 HOUR]"))
					time.Sleep(time.Hour)
					tracker.resetBatch()
					sendNotification("RESUMING", "Continuing extraction process")
				}

				if !downloadedPhotos[edge.Node.Shortcode] {
					filename := filepath.Join(outputDir, fmt.Sprintf("%s.jpg", edge.Node.Shortcode))
					if _, err := os.Stat(filename); err == nil {
						downloadedPhotos[edge.Node.Shortcode] = true
						continue
					}

					err := downloadPhoto(client, headers, edge.Node.DisplayURL, outputDir, edge.Node.Shortcode)
					if err != nil {
						fmt.Printf(red("\nError downloading %s: %v\n"), edge.Node.Shortcode, err)
						continue
					}

					downloadedPhotos[edge.Node.Shortcode] = true
					tracker.incrementDownloaded()
					fmt.Printf("\r%s Total: %d | Batch: %s",
						green("[EXTRACTED]"),
						tracker.totalDownloaded,
						tracker.getBatchProgress())
				}
			}
		}

		// Handle pagination
		pageInfo := media.PageInfo
		if pageInfo.HasNextPage {
			endCursor = pageInfo.EndCursor
		} else {
			hasMore = false
		}
	}

	return nil
}

func downloadPhoto(client *http.Client, headers http.Header, url, outputDir, shortcode string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header = http.Header{
		"User-Agent": headers["User-Agent"],
		"Accept":     []string{"image/webp,*/*"},
		"Referer":    []string{"https://www.instagram.com/"},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error downloading photo: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("%s.jpg", shortcode))
	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error saving photo: %v", err)
	}

	return nil
}
