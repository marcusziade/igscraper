package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"igscraper/cmd/igscraper/commands"
	"igscraper/pkg/auth"
	"igscraper/pkg/config"
	"igscraper/pkg/logger"
	"igscraper/pkg/scraper"
	"igscraper/pkg/ui"
)

var (
	configFile    = flag.String("config", "", "Path to configuration file")
	outputDir     = flag.String("output", "", "Output directory for downloads")
	concurrent    = flag.Int("concurrent", 3, "Number of concurrent downloads")
	rateLimit     = flag.Int("rate-limit", 60, "Requests per minute")
	notifications = flag.Bool("notifications", true, "Enable desktop notifications")
	accountName   = flag.String("account", "", "Use specific stored account")
)

func main() {
	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <command> [arguments]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  <username>          Download photos from Instagram user\n")
		fmt.Fprintf(os.Stderr, "  auth <subcommand>   Manage stored credentials\n")
		fmt.Fprintf(os.Stderr, "    login [username]    Store Instagram credentials\n")
		fmt.Fprintf(os.Stderr, "    logout [username]   Remove stored credentials\n")
		fmt.Fprintf(os.Stderr, "    list               List all stored accounts\n")
		fmt.Fprintf(os.Stderr, "    switch [username]  Switch between accounts\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s auth login myusername\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -output ./photos someuser\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -account myaccount targetuser\n", os.Args[0])
	}

	flag.Parse()

	// Show ASCII logo
	ui.PrintLogo()

	// Get command from args
	args := flag.Args()
	if len(args) == 0 {
		ui.PrintError("No command specified", "")
		flag.Usage()
		os.Exit(1)
	}

	// Handle auth commands
	if args[0] == "auth" {
		if len(args) < 2 {
			commands.PrintAuthHelp()
			os.Exit(1)
		}

		authCmd, err := commands.NewAuthCommand()
		if err != nil {
			ui.PrintError("Failed to initialize auth command", err.Error())
			os.Exit(1)
		}

		if err := authCmd.Execute(args[1], args[2:]); err != nil {
			ui.PrintError("Auth command failed", err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle scraping command
	username := strings.TrimSpace(args[0])
	ui.PrintInfo("Target Profile", username)

	// Build command line flags map
	flags := make(map[string]interface{})
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

	// Handle credentials
	credManager, err := auth.NewManager()
	if err != nil {
		ui.PrintError("Failed to initialize credential manager", err.Error())
		os.Exit(1)
	}

	var account *auth.Account

	// Try to get credentials from various sources
	if *accountName != "" {
		// Use specific account
		account, err = credManager.Retrieve(*accountName)
		if err != nil {
			ui.PrintError("Account not found", *accountName)
			ui.PrintInfo("Available accounts", "Use 'igscraper auth list' to see stored accounts")
			os.Exit(1)
		}
	} else if cfg.Instagram.SessionID != "" && cfg.Instagram.CSRFToken != "" && 
			  cfg.Instagram.SessionID != "YOUR_SESSION_ID" && cfg.Instagram.CSRFToken != "YOUR_CSRF_TOKEN" {
		// Use credentials from config/env (backward compatibility)
		// These are already loaded into cfg
		logger.Info("Using credentials from configuration")
	} else {
		// Try to get default account from credential manager
		account, err = credManager.RetrieveDefault()
		if err != nil {
			// No credentials found anywhere
			logger.Error("No credentials found")
			ui.PrintError("No Instagram credentials found", "")
			fmt.Println("\nTo store credentials securely, run:")
			fmt.Println("  igscraper auth login")
			fmt.Println("\nFor backward compatibility, you can also set environment variables:")
			fmt.Println("  export IGSCRAPER_SESSION_ID=your_session_id")
			fmt.Println("  export IGSCRAPER_CSRF_TOKEN=your_csrf_token")
			os.Exit(1)
		}
	}

	// If we got an account from credential manager, update config
	if account != nil {
		cfg.Instagram.SessionID = account.SessionID
		cfg.Instagram.CSRFToken = account.CSRFToken
		if account.UserAgent != "" {
			cfg.Instagram.UserAgent = account.UserAgent
		}
		logger.WithField("account", account.Username).Info("Using stored credentials")
		ui.PrintInfo("Using account", account.Username)
	}

	// Final credential validation
	if cfg.Instagram.SessionID == "" || cfg.Instagram.SessionID == "YOUR_SESSION_ID" {
		logger.Error("Missing Instagram session ID")
		ui.PrintError("Missing Instagram session ID", "Run 'igscraper auth login' to store credentials")
		os.Exit(1)
	}

	if cfg.Instagram.CSRFToken == "" || cfg.Instagram.CSRFToken == "YOUR_CSRF_TOKEN" {
		logger.Error("Missing Instagram CSRF token")
		ui.PrintError("Missing Instagram CSRF token", "Run 'igscraper auth login' to store credentials")
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