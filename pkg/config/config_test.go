package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	// Test default values
	if config.RateLimit.RequestsPerMinute != 60 {
		t.Errorf("Expected default requests per minute to be 60, got %d", config.RateLimit.RequestsPerMinute)
	}
	
	if config.Download.ConcurrentDownloads != 3 {
		t.Errorf("Expected default concurrent downloads to be 3, got %d", config.Download.ConcurrentDownloads)
	}
	
	if config.Output.BaseDirectory != "./downloads" {
		t.Errorf("Expected default output directory to be ./downloads, got %s", config.Output.BaseDirectory)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Set test environment variables
	os.Setenv("IGSCRAPER_SESSION_ID", "test-session-id")
	os.Setenv("IGSCRAPER_CSRF_TOKEN", "test-csrf-token")
	os.Setenv("IGSCRAPER_REQUESTS_PER_MINUTE", "30")
	os.Setenv("IGSCRAPER_OUTPUT_DIR", "/tmp/test-downloads")
	os.Setenv("IGSCRAPER_CONCURRENT_DOWNLOADS", "5")
	os.Setenv("IGSCRAPER_NOTIFICATIONS_ENABLED", "false")
	os.Setenv("IGSCRAPER_LOG_LEVEL", "debug")
	
	defer func() {
		// Clean up environment variables
		os.Unsetenv("IGSCRAPER_SESSION_ID")
		os.Unsetenv("IGSCRAPER_CSRF_TOKEN")
		os.Unsetenv("IGSCRAPER_REQUESTS_PER_MINUTE")
		os.Unsetenv("IGSCRAPER_OUTPUT_DIR")
		os.Unsetenv("IGSCRAPER_CONCURRENT_DOWNLOADS")
		os.Unsetenv("IGSCRAPER_NOTIFICATIONS_ENABLED")
		os.Unsetenv("IGSCRAPER_LOG_LEVEL")
	}()
	
	config := DefaultConfig()
	err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("Failed to load from environment: %v", err)
	}
	
	// Test loaded values
	if config.Instagram.SessionID != "test-session-id" {
		t.Errorf("Expected session ID to be test-session-id, got %s", config.Instagram.SessionID)
	}
	
	if config.Instagram.CSRFToken != "test-csrf-token" {
		t.Errorf("Expected CSRF token to be test-csrf-token, got %s", config.Instagram.CSRFToken)
	}
	
	if config.RateLimit.RequestsPerMinute != 30 {
		t.Errorf("Expected requests per minute to be 30, got %d", config.RateLimit.RequestsPerMinute)
	}
	
	if config.Output.BaseDirectory != "/tmp/test-downloads" {
		t.Errorf("Expected output directory to be /tmp/test-downloads, got %s", config.Output.BaseDirectory)
	}
	
	if config.Download.ConcurrentDownloads != 5 {
		t.Errorf("Expected concurrent downloads to be 5, got %d", config.Download.ConcurrentDownloads)
	}
	
	if config.Notifications.Enabled != false {
		t.Errorf("Expected notifications to be disabled, got %v", config.Notifications.Enabled)
	}
	
	if config.Logging.Level != "debug" {
		t.Errorf("Expected log level to be debug, got %s", config.Logging.Level)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
	}{
		{
			name: "valid config",
			config: &Config{
				Instagram: InstagramConfig{
					SessionID: "test-session",
					CSRFToken: "test-csrf",
				},
				RateLimit: RateLimitConfig{
					RequestsPerMinute: 60,
					BurstSize:         10,
					MaxRetries:        3,
				},
				Output: OutputConfig{
					BaseDirectory:   "./downloads",
					FileNamePattern: "{shortcode}.{ext}",
				},
				Download: DownloadConfig{
					ConcurrentDownloads: 3,
					DownloadTimeout:     30 * time.Second,
				},
				Notifications: NotificationConfig{
					NotificationType: "terminal",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantError: false,
		},
		{
			name: "missing session ID",
			config: &Config{
				Instagram: InstagramConfig{
					CSRFToken: "test-csrf",
				},
				RateLimit: RateLimitConfig{
					RequestsPerMinute: 60,
					BurstSize:         10,
				},
				Output: OutputConfig{
					BaseDirectory:   "./downloads",
					FileNamePattern: "{shortcode}.{ext}",
				},
				Download: DownloadConfig{
					ConcurrentDownloads: 3,
					DownloadTimeout:     30 * time.Second,
				},
				Notifications: NotificationConfig{
					NotificationType: "terminal",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantError: true,
		},
		{
			name: "invalid concurrent downloads",
			config: &Config{
				Instagram: InstagramConfig{
					SessionID: "test-session",
					CSRFToken: "test-csrf",
				},
				RateLimit: RateLimitConfig{
					RequestsPerMinute: 60,
					BurstSize:         10,
				},
				Output: OutputConfig{
					BaseDirectory:   "./downloads",
					FileNamePattern: "{shortcode}.{ext}",
				},
				Download: DownloadConfig{
					ConcurrentDownloads: 15, // Too high
					DownloadTimeout:     30 * time.Second,
				},
				Notifications: NotificationConfig{
					NotificationType: "terminal",
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantError: true,
		},
		{
			name: "invalid log level",
			config: &Config{
				Instagram: InstagramConfig{
					SessionID: "test-session",
					CSRFToken: "test-csrf",
				},
				RateLimit: RateLimitConfig{
					RequestsPerMinute: 60,
					BurstSize:         10,
				},
				Output: OutputConfig{
					BaseDirectory:   "./downloads",
					FileNamePattern: "{shortcode}.{ext}",
				},
				Download: DownloadConfig{
					ConcurrentDownloads: 3,
					DownloadTimeout:     30 * time.Second,
				},
				Notifications: NotificationConfig{
					NotificationType: "terminal",
				},
				Logging: LoggingConfig{
					Level: "invalid", // Invalid log level
				},
			},
			wantError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestMergeCommandLineFlags(t *testing.T) {
	config := DefaultConfig()
	
	flags := map[string]interface{}{
		"session-id": "flag-session-id",
		"csrf-token": "flag-csrf-token",
		"output":     "/flag/output",
		"concurrent": 7,
		"log-level":  "error",
	}
	
	config.MergeCommandLineFlags(flags)
	
	// Test merged values
	if config.Instagram.SessionID != "flag-session-id" {
		t.Errorf("Expected session ID to be flag-session-id, got %s", config.Instagram.SessionID)
	}
	
	if config.Instagram.CSRFToken != "flag-csrf-token" {
		t.Errorf("Expected CSRF token to be flag-csrf-token, got %s", config.Instagram.CSRFToken)
	}
	
	if config.Output.BaseDirectory != "/flag/output" {
		t.Errorf("Expected output directory to be /flag/output, got %s", config.Output.BaseDirectory)
	}
	
	if config.Download.ConcurrentDownloads != 7 {
		t.Errorf("Expected concurrent downloads to be 7, got %d", config.Download.ConcurrentDownloads)
	}
	
	if config.Logging.Level != "error" {
		t.Errorf("Expected log level to be error, got %s", config.Logging.Level)
	}
}

func TestSaveAndLoadFromFile(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	
	// Create a config and save it
	config := DefaultConfig()
	config.Instagram.SessionID = "save-test-session"
	config.Instagram.CSRFToken = "save-test-csrf"
	config.Download.ConcurrentDownloads = 8
	
	err := config.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	// Load the saved config
	loadedConfig := DefaultConfig()
	err = loadedConfig.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// Verify loaded values
	if loadedConfig.Instagram.SessionID != "save-test-session" {
		t.Errorf("Expected loaded session ID to be save-test-session, got %s", loadedConfig.Instagram.SessionID)
	}
	
	if loadedConfig.Instagram.CSRFToken != "save-test-csrf" {
		t.Errorf("Expected loaded CSRF token to be save-test-csrf, got %s", loadedConfig.Instagram.CSRFToken)
	}
	
	if loadedConfig.Download.ConcurrentDownloads != 8 {
		t.Errorf("Expected loaded concurrent downloads to be 8, got %d", loadedConfig.Download.ConcurrentDownloads)
	}
}