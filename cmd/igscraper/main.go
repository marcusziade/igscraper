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

var (
	configFile    = flag.String("config", "", "Path to configuration file")
	sessionID     = flag.String("session-id", "", "Instagram session ID")
	csrfToken     = flag.String("csrf-token", "", "Instagram CSRF token")
	outputDir     = flag.String("output", "", "Output directory for downloads")
	concurrent    = flag.Int("concurrent", 3, "Number of concurrent downloads")
	rateLimit     = flag.Int("rate-limit", 60, "Requests per minute")
	notifications = flag.Bool("notifications", true, "Enable desktop notifications")
)

func main() {
	flag.Parse()

	// Show ASCII logo
	ui.PrintLogo()

	// Get username from args
	args := flag.Args()
	if len(args) != 1 {
		ui.PrintError("Usage: igscraper [flags] <instagram_username>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	username := strings.TrimSpace(args[0])
	ui.PrintInfo("Target Profile", username)

	// Build command line flags map
	flags := make(map[string]interface{})
	if *sessionID != "" {
		flags["session-id"] = *sessionID
	}
	if *csrfToken != "" {
		flags["csrf-token"] = *csrfToken
	}
	if *outputDir != "" {
		flags["output"] = *outputDir
	}
	if *concurrent != 3 {
		flags["concurrent-downloads"] = *concurrent
	}
	if *rateLimit != 60 {
		flags["requests-per-minute"] = *rateLimit
	}
	if !*notifications {
		flags["notifications-enabled"] = false
	}

	// Load configuration
	cfg, err := config.Load(*configFile, flags)
	if err != nil {
		ui.PrintError("Failed to load configuration", err.Error())
		os.Exit(1)
	}

	// Initialize logger
	logger.Initialize(&cfg.Logging)
	logger.WithField("version", "2.0").Info("Instagram Scraper starting")

	// Validate credentials
	if cfg.Instagram.SessionID == "" || cfg.Instagram.SessionID == "YOUR_SESSION_ID" {
		logger.Error("Missing Instagram session ID")
		ui.PrintError("Missing Instagram session ID", "Please provide via --session-id flag or IGSCRAPER_SESSION_ID env var")
		os.Exit(1)
	}

	if cfg.Instagram.CSRFToken == "" || cfg.Instagram.CSRFToken == "YOUR_CSRF_TOKEN" {
		logger.Error("Missing Instagram CSRF token")
		ui.PrintError("Missing Instagram CSRF token", "Please provide via --csrf-token flag or IGSCRAPER_CSRF_TOKEN env var")
		os.Exit(1)
	}

	logger.WithField("username", username).Info("Starting scrape operation")

	// Create and run scraper
	ui.PrintHighlight("[INITIATING EXTRACTION SEQUENCE]")
	
	s, err := scraper.New(cfg)
	if err != nil {
		ui.PrintError("Failed to initialize scraper", err.Error())
		os.Exit(1)
	}

	err = s.DownloadUserPhotos(username)
	if err != nil {
		logger.WithError(err).WithField("username", username).Error("Extraction failed")
		ui.PrintError("EXTRACTION FAILED", err.Error())
		os.Exit(1)
	}

	logger.WithField("username", username).Info("Extraction completed successfully")
	ui.PrintSuccess("[EXTRACTION COMPLETED SUCCESSFULLY]")
}