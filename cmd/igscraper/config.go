package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"igscraper/pkg/config"
	"igscraper/pkg/ui"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration files",
	Long: `Manage Instagram Scraper configuration files.

Configuration can be loaded from:
  - Command line flags (highest priority)
  - Environment variables
  - Configuration file
  - Default values (lowest priority)`,
}

// initCmd represents the config init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create an example configuration file",
	Long: `Create an example configuration file with all available options.

The file will be created in the current directory as 'igscraper.yaml'
unless a different path is specified with the --config flag.`,
	Run: runConfigInit,
}

// showCmd represents the config show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long: `Show the current configuration including values from all sources:
  - Command line flags
  - Environment variables
  - Configuration file
  - Default values

Sensitive values like credentials will be masked for security.`,
	Run: runConfigShow,
}

// validateCmd represents the config validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long: `Validate a configuration file for syntax errors and invalid values.

This command checks:
  - YAML syntax
  - Required fields
  - Value types and ranges
  - Path accessibility`,
	Run: runConfigValidate,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(initCmd)
	configCmd.AddCommand(showCmd)
	configCmd.AddCommand(validateCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) {
	// Determine config file path
	configPath := configFile
	if configPath == "" {
		configPath = "igscraper.yaml"
	}

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		ui.PrintError("Configuration file already exists", configPath)
		fmt.Println("\nTo overwrite, first remove the existing file:")
		fmt.Printf("  rm %s\n", configPath)
		os.Exit(1)
	}

	// Create example configuration
	exampleConfig := `# Instagram Scraper Configuration File
# 
# This file contains all available configuration options.
# You can also use environment variables prefixed with IGSCRAPER_
# For example: IGSCRAPER_SESSION_ID, IGSCRAPER_CSRF_TOKEN

# Instagram credentials
instagram:
  # Session ID from Instagram cookies (required)
  # Get this from your browser's developer tools
  session_id: "YOUR_SESSION_ID"
  
  # CSRF token from Instagram cookies (required)
  csrf_token: "YOUR_CSRF_TOKEN"
  
  # User agent string (optional)
  # Leave empty to use default
  user_agent: ""

# Download configuration
download:
  # Output directory for downloads
  # Default: current directory
  output: "./downloads"
  
  # Number of concurrent downloads
  # Range: 1-10
  concurrent_downloads: 3
  
  # Download timeout in seconds
  # Range: 10-300
  timeout: 30
  
  # Overwrite existing files
  overwrite_existing: false

# Rate limiting configuration
rate_limit:
  # Requests per minute
  # Range: 1-120
  requests_per_minute: 60
  
  # Enable burst mode for faster initial downloads
  burst_enabled: true
  
  # Burst size (number of requests allowed in burst)
  burst_size: 10

# Retry configuration
retry:
  # Maximum number of retry attempts
  # Range: 0-10
  max_retries: 3
  
  # Initial backoff duration in seconds
  initial_backoff: 1
  
  # Maximum backoff duration in seconds
  max_backoff: 60
  
  # Backoff multiplier
  multiplier: 2.0

# Logging configuration
logging:
  # Log level: debug, info, warn, error
  level: "info"
  
  # Log format: text, json
  format: "text"
  
  # Log file path (optional)
  # Leave empty to log to stdout only
  file: ""
  
  # Maximum log file size in MB
  max_size: 100
  
  # Maximum number of old log files to keep
  max_backups: 3
  
  # Maximum age of log files in days
  max_age: 30

# UI configuration
ui:
  # Enable colored output
  color_enabled: true
  
  # Enable progress bars
  progress_enabled: true
  
  # Enable desktop notifications
  notifications_enabled: true
  
  # Show download speed
  show_speed: true
  
  # Update interval in milliseconds
  update_interval: 100

# Storage configuration
storage:
  # Create user directories
  create_user_dir: true
  
  # Directory permissions (octal)
  dir_permissions: "0755"
  
  # File permissions (octal)
  file_permissions: "0644"
  
  # Save metadata alongside downloads
  save_metadata: true
  
  # Metadata format: json, yaml
  metadata_format: "json"
`

	// Write configuration file
	if err := os.WriteFile(configPath, []byte(exampleConfig), 0644); err != nil {
		ui.PrintError("Failed to create configuration file", err.Error())
		os.Exit(1)
	}

	ui.PrintSuccess("Configuration file created: " + configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit the configuration file and add your Instagram credentials")
	fmt.Println("2. Run 'igscraper config validate' to check the configuration")
	fmt.Println("3. Start downloading with 'igscraper scrape <username>'")
}

func runConfigShow(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.Load(configFile, nil)
	if err != nil {
		ui.PrintError("Failed to load configuration", err.Error())
		os.Exit(1)
	}

	// Create a sanitized version for display
	displayCfg := *cfg
	
	// Mask sensitive values
	if displayCfg.Instagram.SessionID != "" {
		if len(displayCfg.Instagram.SessionID) > 8 {
			displayCfg.Instagram.SessionID = displayCfg.Instagram.SessionID[:4] + "..." + displayCfg.Instagram.SessionID[len(displayCfg.Instagram.SessionID)-4:]
		} else {
			displayCfg.Instagram.SessionID = "***"
		}
	}
	
	if displayCfg.Instagram.CSRFToken != "" {
		if len(displayCfg.Instagram.CSRFToken) > 8 {
			displayCfg.Instagram.CSRFToken = displayCfg.Instagram.CSRFToken[:4] + "..." + displayCfg.Instagram.CSRFToken[len(displayCfg.Instagram.CSRFToken)-4:]
		} else {
			displayCfg.Instagram.CSRFToken = "***"
		}
	}

	// Convert to YAML for display
	data, err := yaml.Marshal(&displayCfg)
	if err != nil {
		ui.PrintError("Failed to format configuration", err.Error())
		os.Exit(1)
	}

	ui.PrintHighlight("Current Configuration")
	fmt.Println()
	fmt.Print(string(data))
	
	// Show configuration sources
	fmt.Println("\nConfiguration sources (in order of priority):")
	fmt.Println("1. Command line flags")
	fmt.Println("2. Environment variables (IGSCRAPER_*)")
	if configFile != "" {
		fmt.Printf("3. Configuration file: %s\n", configFile)
	} else {
		fmt.Println("3. Configuration file: (not specified)")
	}
	fmt.Println("4. Default values")
}

func runConfigValidate(cmd *cobra.Command, args []string) {
	// Check if config file is specified
	if configFile == "" {
		// Try to find config file in common locations
		possiblePaths := []string{
			"igscraper.yaml",
			"igscraper.yml",
			".igscraper.yaml",
			".igscraper.yml",
			filepath.Join(os.Getenv("HOME"), ".igscraper.yaml"),
			filepath.Join(os.Getenv("HOME"), ".config", "igscraper", "config.yaml"),
		}
		
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				configFile = path
				break
			}
		}
		
		if configFile == "" {
			ui.PrintError("No configuration file found", "Specify a file with --config flag")
			os.Exit(1)
		}
	}

	ui.PrintInfo("Validating configuration", configFile)

	// Try to load and validate configuration
	cfg, err := config.Load(configFile, nil)
	if err != nil {
		ui.PrintError("Configuration validation failed", err.Error())
		os.Exit(1)
	}

	// Perform additional validation checks
	warnings := []string{}
	errors := []string{}

	// Check credentials
	if cfg.Instagram.SessionID == "" || cfg.Instagram.SessionID == "YOUR_SESSION_ID" {
		warnings = append(warnings, "Instagram session ID not configured")
	}
	if cfg.Instagram.CSRFToken == "" || cfg.Instagram.CSRFToken == "YOUR_CSRF_TOKEN" {
		warnings = append(warnings, "Instagram CSRF token not configured")
	}

	// Check paths
	if cfg.Output.BaseDirectory != "" {
		if err := os.MkdirAll(cfg.Output.BaseDirectory, 0755); err != nil {
			errors = append(errors, fmt.Sprintf("Cannot create output directory: %v", err))
		}
	}

	// Check logging file path
	if cfg.Logging.File != "" {
		dir := filepath.Dir(cfg.Logging.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			errors = append(errors, fmt.Sprintf("Cannot create log directory: %v", err))
		}
	}

	// Check value ranges
	if cfg.Download.ConcurrentDownloads < 1 || cfg.Download.ConcurrentDownloads > 10 {
		errors = append(errors, "concurrent_downloads must be between 1 and 10")
	}
	if cfg.RateLimit.RequestsPerMinute < 1 || cfg.RateLimit.RequestsPerMinute > 120 {
		errors = append(errors, "requests_per_minute must be between 1 and 120")
	}
	if cfg.Retry.MaxAttempts < 0 || cfg.Retry.MaxAttempts > 10 {
		errors = append(errors, "max_attempts must be between 0 and 10")
	}

	// Display results
	if len(errors) > 0 {
		ui.PrintError("Configuration has errors:", "")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		os.Exit(1)
	}

	if len(warnings) > 0 {
		ui.PrintWarning("Configuration warnings:", "")
		for _, warn := range warnings {
			fmt.Printf("  - %s\n", warn)
		}
		fmt.Println()
	}

	ui.PrintSuccess("Configuration is valid")
	
	// Show summary
	fmt.Println("\nConfiguration summary:")
	fmt.Printf("  Output directory: %s\n", cfg.Output.BaseDirectory)
	fmt.Printf("  Concurrent downloads: %d\n", cfg.Download.ConcurrentDownloads)
	fmt.Printf("  Rate limit: %d requests/minute\n", cfg.RateLimit.RequestsPerMinute)
	fmt.Printf("  Max retries: %d\n", cfg.Retry.MaxAttempts)
	fmt.Printf("  Log level: %s\n", cfg.Logging.Level)
}