package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration options for the Instagram scraper
type Config struct {
	// Instagram credentials
	Instagram InstagramConfig `yaml:"instagram" json:"instagram"`
	
	// Rate limiting configuration
	RateLimit RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`
	
	// Output settings
	Output OutputConfig `yaml:"output" json:"output"`
	
	// Download settings
	Download DownloadConfig `yaml:"download" json:"download"`
	
	// Notification preferences
	Notifications NotificationConfig `yaml:"notifications" json:"notifications"`
	
	// Logging configuration
	Logging LoggingConfig `yaml:"logging" json:"logging"`
}

// InstagramConfig holds Instagram-specific configuration
type InstagramConfig struct {
	SessionID  string `yaml:"session_id" json:"session_id"`
	CSRFToken  string `yaml:"csrf_token" json:"csrf_token"`
	UserAgent  string `yaml:"user_agent" json:"user_agent"`
	APIVersion string `yaml:"api_version" json:"api_version"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int           `yaml:"requests_per_minute" json:"requests_per_minute"`
	BurstSize         int           `yaml:"burst_size" json:"burst_size"`
	BackoffMultiplier float64       `yaml:"backoff_multiplier" json:"backoff_multiplier"`
	MaxRetries        int           `yaml:"max_retries" json:"max_retries"`
	RetryDelay        time.Duration `yaml:"retry_delay" json:"retry_delay"`
}

// OutputConfig holds output directory configuration
type OutputConfig struct {
	BaseDirectory     string `yaml:"base_directory" json:"base_directory"`
	CreateUserFolders bool   `yaml:"create_user_folders" json:"create_user_folders"`
	FileNamePattern   string `yaml:"file_name_pattern" json:"file_name_pattern"`
	OverwriteExisting bool   `yaml:"overwrite_existing" json:"overwrite_existing"`
}

// DownloadConfig holds download-specific configuration
type DownloadConfig struct {
	ConcurrentDownloads int           `yaml:"concurrent_downloads" json:"concurrent_downloads"`
	DownloadTimeout     time.Duration `yaml:"download_timeout" json:"download_timeout"`
	RetryAttempts       int           `yaml:"retry_attempts" json:"retry_attempts"`
	SkipVideos          bool          `yaml:"skip_videos" json:"skip_videos"`
	SkipImages          bool          `yaml:"skip_images" json:"skip_images"`
	MinFileSize         int64         `yaml:"min_file_size" json:"min_file_size"`
	MaxFileSize         int64         `yaml:"max_file_size" json:"max_file_size"`
}

// NotificationConfig holds notification preferences
type NotificationConfig struct {
	Enabled           bool   `yaml:"enabled" json:"enabled"`
	OnComplete        bool   `yaml:"on_complete" json:"on_complete"`
	OnError           bool   `yaml:"on_error" json:"on_error"`
	OnRateLimit       bool   `yaml:"on_rate_limit" json:"on_rate_limit"`
	ProgressInterval  int    `yaml:"progress_interval" json:"progress_interval"`
	NotificationType  string `yaml:"notification_type" json:"notification_type"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `yaml:"level" json:"level"`
	File       string `yaml:"file" json:"file"`
	MaxSize    int    `yaml:"max_size" json:"max_size"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	MaxAge     int    `yaml:"max_age" json:"max_age"`
	Compress   bool   `yaml:"compress" json:"compress"`
}

// DefaultConfig returns a Config instance with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Instagram: InstagramConfig{
			UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			APIVersion: "v1",
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: 60,
			BurstSize:         10,
			BackoffMultiplier: 2.0,
			MaxRetries:        3,
			RetryDelay:        5 * time.Second,
		},
		Output: OutputConfig{
			BaseDirectory:     "./downloads",
			CreateUserFolders: true,
			FileNamePattern:   "{shortcode}.{ext}",
			OverwriteExisting: false,
		},
		Download: DownloadConfig{
			ConcurrentDownloads: 3,
			DownloadTimeout:     30 * time.Second,
			RetryAttempts:       3,
			SkipVideos:          false,
			SkipImages:          false,
			MinFileSize:         0,
			MaxFileSize:         0, // 0 means no limit
		},
		Notifications: NotificationConfig{
			Enabled:          true,
			OnComplete:       true,
			OnError:          true,
			OnRateLimit:      true,
			ProgressInterval: 10,
			NotificationType: "terminal",
		},
		Logging: LoggingConfig{
			Level:      "info",
			File:       "",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   false,
		},
	}
}

// LoadFromEnv loads configuration from environment variables
func (c *Config) LoadFromEnv() error {
	// Instagram credentials
	if sessionID := os.Getenv("IGSCRAPER_SESSION_ID"); sessionID != "" {
		c.Instagram.SessionID = sessionID
	}
	if csrfToken := os.Getenv("IGSCRAPER_CSRF_TOKEN"); csrfToken != "" {
		c.Instagram.CSRFToken = csrfToken
	}
	if userAgent := os.Getenv("IGSCRAPER_USER_AGENT"); userAgent != "" {
		c.Instagram.UserAgent = userAgent
	}
	
	// Rate limiting
	if rpm := os.Getenv("IGSCRAPER_REQUESTS_PER_MINUTE"); rpm != "" {
		var val int
		fmt.Sscanf(rpm, "%d", &val)
		if val > 0 {
			c.RateLimit.RequestsPerMinute = val
		}
	}
	
	// Output directory
	if outputDir := os.Getenv("IGSCRAPER_OUTPUT_DIR"); outputDir != "" {
		c.Output.BaseDirectory = outputDir
	}
	
	// Concurrent downloads
	if concurrent := os.Getenv("IGSCRAPER_CONCURRENT_DOWNLOADS"); concurrent != "" {
		var val int
		fmt.Sscanf(concurrent, "%d", &val)
		if val > 0 {
			c.Download.ConcurrentDownloads = val
		}
	}
	
	// Notifications
	if notifEnabled := os.Getenv("IGSCRAPER_NOTIFICATIONS_ENABLED"); notifEnabled != "" {
		c.Notifications.Enabled = strings.ToLower(notifEnabled) == "true"
	}
	
	// Logging level
	if logLevel := os.Getenv("IGSCRAPER_LOG_LEVEL"); logLevel != "" {
		c.Logging.Level = logLevel
	}
	
	return nil
}

// LoadFromFile loads configuration from a YAML file
func (c *Config) LoadFromFile(path string) error {
	// If path is empty, try default locations
	if path == "" {
		path = c.findConfigFile()
		if path == "" {
			return nil // No config file found, not an error
		}
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	
	return nil
}

// findConfigFile searches for config file in standard locations
func (c *Config) findConfigFile() string {
	// Check in order of precedence
	locations := []string{
		".igscraper.yaml",
		".igscraper.yml",
		filepath.Join(os.Getenv("HOME"), ".config", "igscraper", "config.yaml"),
		filepath.Join(os.Getenv("HOME"), ".config", "igscraper", "config.yml"),
		filepath.Join(os.Getenv("HOME"), ".igscraper.yaml"),
		filepath.Join(os.Getenv("HOME"), ".igscraper.yml"),
	}
	
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}
	
	return ""
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	var errs []error
	
	// Validate Instagram credentials
	if c.Instagram.SessionID == "" {
		errs = append(errs, errors.New("Instagram session ID is required"))
	}
	if c.Instagram.CSRFToken == "" {
		errs = append(errs, errors.New("Instagram CSRF token is required"))
	}
	
	// Validate rate limiting
	if c.RateLimit.RequestsPerMinute <= 0 {
		errs = append(errs, errors.New("requests per minute must be positive"))
	}
	if c.RateLimit.BurstSize <= 0 {
		errs = append(errs, errors.New("burst size must be positive"))
	}
	if c.RateLimit.MaxRetries < 0 {
		errs = append(errs, errors.New("max retries cannot be negative"))
	}
	
	// Validate download settings
	if c.Download.ConcurrentDownloads <= 0 {
		errs = append(errs, errors.New("concurrent downloads must be positive"))
	}
	if c.Download.ConcurrentDownloads > 10 {
		errs = append(errs, errors.New("concurrent downloads should not exceed 10"))
	}
	if c.Download.DownloadTimeout <= 0 {
		errs = append(errs, errors.New("download timeout must be positive"))
	}
	
	// Validate output settings
	if c.Output.BaseDirectory == "" {
		errs = append(errs, errors.New("output directory is required"))
	}
	if c.Output.FileNamePattern == "" {
		errs = append(errs, errors.New("file name pattern is required"))
	}
	
	// Validate logging
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[strings.ToLower(c.Logging.Level)] {
		errs = append(errs, errors.New("invalid log level"))
	}
	
	// Validate notification type
	validNotifTypes := map[string]bool{
		"terminal": true, "desktop": true, "none": true,
	}
	if !validNotifTypes[strings.ToLower(c.Notifications.NotificationType)] {
		errs = append(errs, errors.New("invalid notification type"))
	}
	
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	
	return nil
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// MergeCommandLineFlags merges command line flags into the configuration
// This is a preparation for command line flag support
func (c *Config) MergeCommandLineFlags(flags map[string]interface{}) {
	// This method will be used when integrating with cobra/flag packages
	// For now, it serves as a placeholder for the structure
	
	if sessionID, ok := flags["session-id"].(string); ok && sessionID != "" {
		c.Instagram.SessionID = sessionID
	}
	if csrfToken, ok := flags["csrf-token"].(string); ok && csrfToken != "" {
		c.Instagram.CSRFToken = csrfToken
	}
	if outputDir, ok := flags["output"].(string); ok && outputDir != "" {
		c.Output.BaseDirectory = outputDir
	}
	if concurrent, ok := flags["concurrent"].(int); ok && concurrent > 0 {
		c.Download.ConcurrentDownloads = concurrent
	}
	if logLevel, ok := flags["log-level"].(string); ok && logLevel != "" {
		c.Logging.Level = logLevel
	}
}

// Load loads configuration from all sources with proper precedence
// Precedence order: Command line flags > Environment variables > .env file > Config file > Defaults
func Load(configPath string, flags map[string]interface{}) (*Config, error) {
	// Try to load .env files (don't fail if they don't exist)
	_ = godotenv.Load(".env")
	_ = godotenv.Load(filepath.Join(os.Getenv("HOME"), ".env"))
	_ = godotenv.Load(filepath.Join(os.Getenv("HOME"), ".igscraper.env"))
	
	// Start with defaults
	config := DefaultConfig()
	
	// Load from config file
	if err := config.LoadFromFile(configPath); err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}
	
	// Override with environment variables (includes values from .env)
	if err := config.LoadFromEnv(); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}
	
	// Override with command line flags
	config.MergeCommandLineFlags(flags)
	
	// Validate final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return config, nil
}
