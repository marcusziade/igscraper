package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"igscraper/pkg/auth"
	"igscraper/pkg/config"
	"igscraper/pkg/logger"
	"igscraper/pkg/scraper"
	"igscraper/pkg/ui"
	"igscraper/pkg/ui/tui"
)

var (
	// Scrape command flags
	outputDir   string
	concurrent  int
	rateLimit   int
	accountName string
	maxRetries  int
	downloadTimeout int
	resumeDownload bool
	forceRestart bool
	useTUI bool
)

// scrapeCmd represents the scrape command
var scrapeCmd = &cobra.Command{
	Use:   "scrape <username>",
	Short: "Download photos from an Instagram user profile",
	Long: `Download all photos from an Instagram user's profile.

This command requires valid Instagram credentials to be configured either through:
  - Stored credentials (use 'igscraper auth login' to store)
  - Environment variables (IGSCRAPER_SESSION_ID and IGSCRAPER_CSRF_TOKEN)
  - Configuration file

The scraper will create a directory named after the username and download all
photos with their original quality.`,
	Example: `  # Download photos using default settings
  igscraper scrape johndoe

  # Download to specific directory with custom concurrent downloads
  igscraper scrape johndoe --output ./photos --concurrent 5

  # Use a specific stored account
  igscraper scrape johndoe --account myaccount

  # Disable notifications and set custom rate limit
  igscraper scrape johndoe --notifications=false --rate-limit 30

  # Resume an interrupted download
  igscraper scrape johndoe --resume

  # Force restart, ignoring existing checkpoint
  igscraper scrape johndoe --force-restart`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runScrape(cmd, args)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(scrapeCmd)

	// Local flags for scrape command
	scrapeCmd.Flags().StringVarP(&outputDir, "output", "o", "", "output directory for downloads (default: current directory)")
	scrapeCmd.Flags().IntVar(&concurrent, "concurrent", 3, "number of concurrent downloads")
	scrapeCmd.Flags().IntVar(&rateLimit, "rate-limit", 60, "requests per minute")
	scrapeCmd.Flags().StringVarP(&accountName, "account", "a", "", "use specific stored account")
	scrapeCmd.Flags().IntVar(&maxRetries, "max-retries", 3, "maximum number of retry attempts")
	scrapeCmd.Flags().IntVar(&downloadTimeout, "download-timeout", 30, "download timeout in seconds")
	scrapeCmd.Flags().BoolVar(&resumeDownload, "resume", false, "resume from last checkpoint")
	scrapeCmd.Flags().BoolVar(&forceRestart, "force-restart", false, "force restart, ignoring existing checkpoint")
	scrapeCmd.Flags().BoolVar(&useTUI, "tui", false, "use interactive terminal UI with real-time progress")
	
	// Also add these flags to root command for backward compatibility
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "", "output directory for downloads (default: current directory)")
	rootCmd.Flags().IntVar(&concurrent, "concurrent", 3, "number of concurrent downloads")
	rootCmd.Flags().IntVar(&rateLimit, "rate-limit", 60, "requests per minute")
	rootCmd.Flags().StringVarP(&accountName, "account", "a", "", "use specific stored account")
	rootCmd.Flags().BoolVar(&resumeDownload, "resume", false, "resume from last checkpoint")
	rootCmd.Flags().BoolVar(&forceRestart, "force-restart", false, "force restart, ignoring existing checkpoint")
	rootCmd.Flags().BoolVar(&useTUI, "tui", false, "use interactive terminal UI with real-time progress")
}

func runScrape(cmd *cobra.Command, args []string) {
	username := strings.TrimSpace(args[0])
	
	// Set quiet mode if log level is error
	if logLevel == "error" {
		ui.SetQuietMode(true)
	}
	
	// If TUI is enabled, we'll handle output differently
	if !useTUI {
		ui.PrintInfo("Target Profile", username)
	}

	// Build flags map from command line
	flags := make(map[string]interface{})
	if outputDir != "" {
		flags["base-directory"] = outputDir
	}
	if concurrent != 3 {
		flags["concurrent-downloads"] = concurrent
	}
	if rateLimit != 60 {
		flags["requests-per-minute"] = rateLimit
	}
	if !notifications {
		flags["enabled"] = false
	}
	if maxRetries != 3 {
		flags["max-attempts"] = maxRetries
	}
	if downloadTimeout != 30 {
		flags["download-timeout"] = downloadTimeout
	}
	// Pass log level to config
	if logLevel != "info" {
		flags["log-level"] = logLevel
	}

	// Load configuration
	cfg, err := config.Load(configFile, flags)
	if err != nil {
		ui.PrintError("Failed to load configuration", err.Error())
		os.Exit(1)
	}

	// Initialize logger
	logger.Initialize(&cfg.Logging)
	logger.WithField("version", version).Info("Instagram Scraper starting")

	// Handle credentials
	credManager, err := auth.NewManager()
	if err != nil {
		ui.PrintError("Failed to initialize credential manager", err.Error())
		os.Exit(1)
	}

	var account *auth.Account

	// Try to get credentials from various sources
	if accountName != "" {
		// Use specific account
		account, err = credManager.Retrieve(accountName)
		if err != nil {
			ui.PrintError("Account not found", accountName)
			ui.PrintInfo("Available accounts", "Use 'igscraper auth list' to see stored accounts")
			os.Exit(1)
		}
	} else if cfg.Instagram.SessionID != "" && cfg.Instagram.CSRFToken != "" && 
			  cfg.Instagram.SessionID != "YOUR_SESSION_ID" && cfg.Instagram.CSRFToken != "YOUR_CSRF_TOKEN" {
		// Use credentials from config/env (backward compatibility)
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
	if useTUI {
		// Create TUI
		terminal := tui.NewTUI(cfg.Download.ConcurrentDownloads)
		
		// Run scraper in a goroutine
		scraperDone := make(chan error)
		go func() {
			s, err := scraper.New(cfg)
			if err != nil {
				scraperDone <- err
				return
			}
			
			// Set the TUI on the scraper
			s.SetTUI(terminal)
			
			err = s.DownloadUserPhotosWithResume(username, resumeDownload, forceRestart)
			scraperDone <- err
		}()
		
		// Run TUI in main thread
		tuiDone := make(chan error)
		go func() {
			tuiDone <- terminal.Start()
		}()
		
		// Wait for either to finish
		select {
		case err := <-scraperDone:
			terminal.Stop()
			<-tuiDone // Wait for TUI to finish
			if err != nil {
				logger.WithError(err).WithField("username", username).Error("Extraction failed")
				os.Exit(1)
			}
		case err := <-tuiDone:
			if err != nil {
				logger.WithError(err).Error("TUI failed")
				os.Exit(1)
			}
		}
		
		logger.WithField("username", username).Info("Extraction completed successfully")
	} else {
		// Original non-TUI flow
		ui.PrintHighlight("[INITIATING EXTRACTION SEQUENCE]")
		
		s, err := scraper.New(cfg)
		if err != nil {
			ui.PrintError("Failed to initialize scraper", err.Error())
			os.Exit(1)
		}

		err = s.DownloadUserPhotosWithResume(username, resumeDownload, forceRestart)
		if err != nil {
			logger.WithError(err).WithField("username", username).Error("Extraction failed")
			ui.PrintError("EXTRACTION FAILED", err.Error())
			os.Exit(1)
		}

		logger.WithField("username", username).Info("Extraction completed successfully")
		ui.PrintSuccess("[EXTRACTION COMPLETED SUCCESSFULLY]")
	}
}

// Make scrape the default command when no subcommand is specified
func init() {
	// Add a hidden alias to make scraping work without the "scrape" subcommand
	origRunE := rootCmd.RunE
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if origRunE != nil {
			return origRunE(cmd, args)
		}
		if len(args) > 0 && !isKnownCommand(args[0]) {
			// If the first argument is not a known command, treat it as a username
			// No need to transfer flags since we're using the same variables
			return scrapeCmd.RunE(scrapeCmd, args)
		}
		// Otherwise show help
		return cmd.Help()
	}
	
	// Set Args to allow arbitrary arguments
	rootCmd.Args = cobra.ArbitraryArgs
}

func isKnownCommand(arg string) bool {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == arg || cmd.HasAlias(arg) {
			return true
		}
	}
	return false
}