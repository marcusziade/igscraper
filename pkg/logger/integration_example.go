package logger

// This file shows how to integrate the logger into the main application

/*
Example integration in main.go:

package main

import (
	"flag"
	"os"
	"strings"

	"igscraper/pkg/config"
	"igscraper/pkg/logger"
	"igscraper/pkg/scraper"
	"igscraper/pkg/ui"
)

func main() {
	flag.Parse()

	// Show ASCII logo
	ui.PrintLogo()

	// ... get username and flags ...

	// Load configuration
	cfg, err := config.Load(*configFile, flags)
	if err != nil {
		ui.PrintError("Failed to load configuration", err.Error())
		os.Exit(1)
	}

	// Initialize the logger
	if err := logger.Initialize(&cfg.Logging); err != nil {
		ui.PrintError("Failed to initialize logger", err.Error())
		os.Exit(1)
	}

	// Now you can use the logger throughout the application
	logger.Info("Instagram Scraper starting")
	logger.WithField("username", username).Info("Processing user profile")

	// Log configuration (be careful not to log sensitive data)
	logger.WithFields(map[string]interface{}{
		"output_dir":     cfg.Output.BaseDirectory,
		"concurrent":     cfg.Download.ConcurrentDownloads,
		"rate_limit":     cfg.RateLimit.RequestsPerMinute,
		"log_level":      cfg.Logging.Level,
	}).Debug("Configuration loaded")

	// Create and run scraper with logging
	logger.Info("Initializing scraper")
	
	s, err := scraper.New(cfg)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize scraper")
	}

	// Log component start
	logger.LogComponentStart("scraper", map[string]interface{}{
		"username": username,
		"mode":     "photos",
	})

	err = s.DownloadUserPhotos(username)
	if err != nil {
		logger.WithError(err).WithField("username", username).Error("Download failed")
		logger.LogComponentStop("scraper", "error")
		os.Exit(1)
	}

	logger.LogComponentStop("scraper", "completed")
	logger.WithField("username", username).Info("All downloads completed successfully")
}
*/

// Example integration in scraper package:
/*
func (s *Scraper) DownloadUserPhotos(username string) error {
	log := logger.GetLogger().
		WithField("component", "scraper").
		WithField("username", username)

	log.Info("Starting photo download")

	// Fetch user info
	log.Debug("Fetching user information")
	userInfo, err := s.client.GetUserByUsername(username)
	if err != nil {
		log.WithError(err).Error("Failed to fetch user info")
		return err
	}

	log.WithFields(map[string]interface{}{
		"user_id":     userInfo.ID,
		"post_count":  userInfo.MediaCount,
		"is_private":  userInfo.IsPrivate,
	}).Info("User profile fetched")

	// ... rest of the implementation ...
}
*/

// Example integration in downloader:
/*
func (d *Downloader) Download(url, filepath string) error {
	start := time.Now()
	log := logger.GetLogger().
		WithField("component", "downloader").
		WithField("url", url).
		WithField("filepath", filepath)

	log.Debug("Starting download")

	// ... download logic ...

	duration := time.Since(start)
	log.WithField("duration", duration).Info("Download completed")
	
	// Use helper function for standardized logging
	logger.LogDownload(username, mediaID, "image", true, nil)
	
	return nil
}
*/

// Example integration with rate limiter:
/*
func (l *Limiter) Wait() {
	if l.isRateLimited() {
		logger.LogRateLimit("/api/v1/media", l.retryAfter)
		time.Sleep(l.retryAfter)
	}
}
*/