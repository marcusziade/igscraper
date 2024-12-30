package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type InstagramResponse struct {
	RequiresToLogin bool   `json:"requires_to_login"`
	Data            Data   `json:"data"`
	Status          string `json:"status"`
}

type Data struct {
	User User `json:"user"`
}

type User struct {
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

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <instagram_username>")
		return
	}

	username := os.Args[1]
	fmt.Printf("Starting download for user: %s\n", username)

	outputDir := fmt.Sprintf("%s_photos", username)
	fmt.Printf("Creating output directory: %s\n", outputDir)

	err := os.MkdirAll(outputDir, 0o755)
	if err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// You'll need to get these values from your browser after logging in
	headers := http.Header{
		"User-Agent":       []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"},
		"Accept":           []string{"*/*"},
		"Accept-Language":  []string{"en-US,en;q=0.5"},
		"X-IG-App-ID":      []string{"936619743392459"}, // Instagram web app ID
		"X-Requested-With": []string{"XMLHttpRequest"},
		"Connection":       []string{"keep-alive"},
		"Referer":          []string{"https://www.instagram.com/"},
		"Cookie": []string{
			"sessionid=YOUR_SESSION_ID;", // Add your session ID here
			"csrftoken=YOUR_CSRF_TOKEN;", // Add your CSRF token here
		},
	}

	fmt.Println("Starting photo download process...")
	err = downloadPhotos(client, headers, username, outputDir)
	if err != nil {
		fmt.Printf("Error downloading photos: %v\n", err)
		return
	}

	fmt.Println("Download completed successfully!")
}

func downloadPhotos(client *http.Client, headers http.Header, username, outputDir string) error {
	maxRetries := 3
	retryDelay := time.Second * 2

	// Using the Instagram GraphQL API endpoint
	endpoint := fmt.Sprintf("https://www.instagram.com/api/v1/users/web_profile_info/?username=%s", username)
	fmt.Printf("Fetching profile from: %s\n", endpoint)

	for i := 0; i < maxRetries; i++ {
		fmt.Printf("Attempt %d/%d\n", i+1, maxRetries)

		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			return fmt.Errorf("error creating request: %v", err)
		}

		req.Header = headers
		fmt.Println("Sending request with headers:", headers)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Request error on attempt %d: %v\n", i+1, err)
			if i == maxRetries-1 {
				return fmt.Errorf("error making request: %v", err)
			}
			time.Sleep(retryDelay)
			continue
		}
		defer resp.Body.Close()

		fmt.Printf("Response status code: %d\n", resp.StatusCode)
		fmt.Println("Response headers:", resp.Header)

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			fmt.Printf("Response body: %s\n", string(bodyBytes))

			if resp.StatusCode == 401 || resp.StatusCode == 403 {
				return fmt.Errorf("authentication required or invalid credentials")
			}

			if i == maxRetries-1 {
				return fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
			}
			time.Sleep(retryDelay)
			continue
		}

		var result InstagramResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("error decoding JSON: %v", err)
		}

		if result.RequiresToLogin {
			return fmt.Errorf("this profile requires authentication")
		}

		fmt.Printf("Found %d media items\n", len(result.Data.User.EdgeOwnerToTimelineMedia.Edges))

		for i, edge := range result.Data.User.EdgeOwnerToTimelineMedia.Edges {
			if !edge.Node.IsVideo {
				fmt.Printf("Downloading photo %d/%d (ID: %s)\n", i+1, len(result.Data.User.EdgeOwnerToTimelineMedia.Edges), edge.Node.ID)
				err := downloadPhoto(client, headers, edge.Node.DisplayURL, outputDir, edge.Node.Shortcode)
				if err != nil {
					fmt.Printf("Error downloading photo %s: %v\n", edge.Node.ID, err)
					continue
				}
				// Add delay between downloads to avoid rate limiting
				time.Sleep(time.Second)
			} else {
				fmt.Printf("Skipping video %d/%d (ID: %s)\n", i+1, len(result.Data.User.EdgeOwnerToTimelineMedia.Edges), edge.Node.ID)
			}
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded")
}

func downloadPhoto(client *http.Client, headers http.Header, url, outputDir, shortcode string) error {
	fmt.Printf("Downloading photo from URL: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Use a different set of headers for the image download
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

	fmt.Printf("Photo download status code: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("%s.jpg", shortcode))
	fmt.Printf("Saving photo to: %s\n", filename)

	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error saving photo: %v", err)
	}

	fmt.Printf("Successfully downloaded photo: %s (%d bytes)\n", filename, written)
	return nil
}

