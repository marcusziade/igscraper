package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	assert.NotNil(t, cfg)
	
	// Test Instagram defaults
	assert.NotEmpty(t, cfg.Instagram.UserAgent)
	assert.Equal(t, "v1", cfg.Instagram.APIVersion)
	
	// Test RateLimit defaults
	assert.Equal(t, 60, cfg.RateLimit.RequestsPerMinute)
	assert.Equal(t, 10, cfg.RateLimit.BurstSize)
	assert.Equal(t, 2.0, cfg.RateLimit.BackoffMultiplier)
	assert.Equal(t, 3, cfg.RateLimit.MaxRetries)
	assert.Equal(t, 5*time.Second, cfg.RateLimit.RetryDelay)
	
	// Test Retry defaults
	assert.True(t, cfg.Retry.Enabled)
	assert.Equal(t, 3, cfg.Retry.MaxAttempts)
	assert.Equal(t, 1*time.Second, cfg.Retry.BaseDelay)
	assert.Equal(t, 60*time.Second, cfg.Retry.MaxDelay)
	assert.Equal(t, 2.0, cfg.Retry.Multiplier)
	assert.Equal(t, 0.1, cfg.Retry.JitterFactor)
	
	// Test Output defaults
	assert.Equal(t, "./downloads", cfg.Output.BaseDirectory)
	assert.True(t, cfg.Output.CreateUserFolders)
	assert.Equal(t, "{shortcode}.{ext}", cfg.Output.FileNamePattern)
	assert.False(t, cfg.Output.OverwriteExisting)
	
	// Test Download defaults
	assert.Equal(t, 3, cfg.Download.ConcurrentDownloads)
	assert.Equal(t, 30*time.Second, cfg.Download.DownloadTimeout)
	assert.Equal(t, 3, cfg.Download.RetryAttempts)
	assert.False(t, cfg.Download.SkipVideos)
	assert.False(t, cfg.Download.SkipImages)
	assert.Equal(t, int64(0), cfg.Download.MinFileSize)
	assert.Equal(t, int64(0), cfg.Download.MaxFileSize)
	
	// Test Notifications defaults
	assert.True(t, cfg.Notifications.Enabled)
	assert.True(t, cfg.Notifications.OnComplete)
	assert.True(t, cfg.Notifications.OnError)
	assert.True(t, cfg.Notifications.OnRateLimit)
	assert.Equal(t, 10, cfg.Notifications.ProgressInterval)
	assert.Equal(t, "terminal", cfg.Notifications.NotificationType)
	
	// Test Logging defaults
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Empty(t, cfg.Logging.File)
	assert.Equal(t, 100, cfg.Logging.MaxSize)
	assert.Equal(t, 3, cfg.Logging.MaxBackups)
	assert.Equal(t, 7, cfg.Logging.MaxAge)
	assert.False(t, cfg.Logging.Compress)
}

func TestLoadFromEnv(t *testing.T) {
	// Save current env vars
	oldEnv := make(map[string]string)
	envVars := []string{
		"IGSCRAPER_SESSION_ID",
		"IGSCRAPER_CSRF_TOKEN",
		"IGSCRAPER_USER_AGENT",
		"IGSCRAPER_REQUESTS_PER_MINUTE",
		"IGSCRAPER_OUTPUT_DIR",
		"IGSCRAPER_CONCURRENT_DOWNLOADS",
		"IGSCRAPER_NOTIFICATIONS_ENABLED",
		"IGSCRAPER_LOG_LEVEL",
	}
	
	for _, key := range envVars {
		oldEnv[key] = os.Getenv(key)
	}
	
	// Restore env vars after test
	defer func() {
		for key, value := range oldEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()
	
	// Set test env vars
	os.Setenv("IGSCRAPER_SESSION_ID", "env_session")
	os.Setenv("IGSCRAPER_CSRF_TOKEN", "env_csrf")
	os.Setenv("IGSCRAPER_USER_AGENT", "env_agent")
	os.Setenv("IGSCRAPER_REQUESTS_PER_MINUTE", "120")
	os.Setenv("IGSCRAPER_OUTPUT_DIR", "/env/output")
	os.Setenv("IGSCRAPER_CONCURRENT_DOWNLOADS", "5")
	os.Setenv("IGSCRAPER_NOTIFICATIONS_ENABLED", "false")
	os.Setenv("IGSCRAPER_LOG_LEVEL", "debug")
	
	cfg := DefaultConfig()
	err := cfg.LoadFromEnv()
	require.NoError(t, err)
	
	assert.Equal(t, "env_session", cfg.Instagram.SessionID)
	assert.Equal(t, "env_csrf", cfg.Instagram.CSRFToken)
	assert.Equal(t, "env_agent", cfg.Instagram.UserAgent)
	assert.Equal(t, 120, cfg.RateLimit.RequestsPerMinute)
	assert.Equal(t, "/env/output", cfg.Output.BaseDirectory)
	assert.Equal(t, 5, cfg.Download.ConcurrentDownloads)
	assert.False(t, cfg.Notifications.Enabled)
	assert.Equal(t, "debug", cfg.Logging.Level)
}

func TestLoadFromFile(t *testing.T) {
	t.Run("valid yaml file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "test_config.yaml")
		
		// Create test config
		testConfig := `
instagram:
  session_id: file_session
  csrf_token: file_csrf
  user_agent: file_agent
  api_version: v2

rate_limit:
  requests_per_minute: 30
  burst_size: 5
  backoff_multiplier: 1.5
  max_retries: 5
  retry_delay: 10s

retry:
  enabled: false
  max_attempts: 2
  base_delay: 2s
  max_delay: 30s
  multiplier: 1.5
  jitter_factor: 0.2

output:
  base_directory: /file/output
  create_user_folders: false
  file_name_pattern: "{date}_{shortcode}.jpg"
  overwrite_existing: true

download:
  concurrent_downloads: 2
  download_timeout: 60s
  retry_attempts: 5
  skip_videos: true
  skip_images: false
  min_file_size: 1024
  max_file_size: 10485760

notifications:
  enabled: false
  on_complete: false
  on_error: true
  on_rate_limit: false
  progress_interval: 20
  notification_type: desktop

logging:
  level: warn
  file: /var/log/igscraper.log
  max_size: 50
  max_backups: 5
  max_age: 14
  compress: true
`
		
		err := os.WriteFile(configPath, []byte(testConfig), 0644)
		require.NoError(t, err)
		
		cfg := DefaultConfig()
		err = cfg.LoadFromFile(configPath)
		require.NoError(t, err)
		
		// Verify all values were loaded
		assert.Equal(t, "file_session", cfg.Instagram.SessionID)
		assert.Equal(t, "file_csrf", cfg.Instagram.CSRFToken)
		assert.Equal(t, "file_agent", cfg.Instagram.UserAgent)
		assert.Equal(t, "v2", cfg.Instagram.APIVersion)
		
		assert.Equal(t, 30, cfg.RateLimit.RequestsPerMinute)
		assert.Equal(t, 5, cfg.RateLimit.BurstSize)
		assert.Equal(t, 1.5, cfg.RateLimit.BackoffMultiplier)
		assert.Equal(t, 5, cfg.RateLimit.MaxRetries)
		assert.Equal(t, 10*time.Second, cfg.RateLimit.RetryDelay)
		
		assert.False(t, cfg.Retry.Enabled)
		assert.Equal(t, 2, cfg.Retry.MaxAttempts)
		assert.Equal(t, 2*time.Second, cfg.Retry.BaseDelay)
		assert.Equal(t, 30*time.Second, cfg.Retry.MaxDelay)
		assert.Equal(t, 1.5, cfg.Retry.Multiplier)
		assert.Equal(t, 0.2, cfg.Retry.JitterFactor)
		
		assert.Equal(t, "/file/output", cfg.Output.BaseDirectory)
		assert.False(t, cfg.Output.CreateUserFolders)
		assert.Equal(t, "{date}_{shortcode}.jpg", cfg.Output.FileNamePattern)
		assert.True(t, cfg.Output.OverwriteExisting)
		
		assert.Equal(t, 2, cfg.Download.ConcurrentDownloads)
		assert.Equal(t, 60*time.Second, cfg.Download.DownloadTimeout)
		assert.Equal(t, 5, cfg.Download.RetryAttempts)
		assert.True(t, cfg.Download.SkipVideos)
		assert.False(t, cfg.Download.SkipImages)
		assert.Equal(t, int64(1024), cfg.Download.MinFileSize)
		assert.Equal(t, int64(10485760), cfg.Download.MaxFileSize)
		
		assert.False(t, cfg.Notifications.Enabled)
		assert.False(t, cfg.Notifications.OnComplete)
		assert.True(t, cfg.Notifications.OnError)
		assert.False(t, cfg.Notifications.OnRateLimit)
		assert.Equal(t, 20, cfg.Notifications.ProgressInterval)
		assert.Equal(t, "desktop", cfg.Notifications.NotificationType)
		
		assert.Equal(t, "warn", cfg.Logging.Level)
		assert.Equal(t, "/var/log/igscraper.log", cfg.Logging.File)
		assert.Equal(t, 50, cfg.Logging.MaxSize)
		assert.Equal(t, 5, cfg.Logging.MaxBackups)
		assert.Equal(t, 14, cfg.Logging.MaxAge)
		assert.True(t, cfg.Logging.Compress)
	})
	
	t.Run("invalid yaml", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "invalid.yaml")
		
		invalidYAML := `
instagram:
  session_id: [this is invalid
`
		err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
		require.NoError(t, err)
		
		cfg := DefaultConfig()
		err = cfg.LoadFromFile(configPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})
	
	t.Run("non-existent file", func(t *testing.T) {
		cfg := DefaultConfig()
		err := cfg.LoadFromFile("/non/existent/path/config.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})
	
	t.Run("empty path searches default locations", func(t *testing.T) {
		cfg := DefaultConfig()
		err := cfg.LoadFromFile("")
		// Should not error, just returns nil if no config found
		assert.NoError(t, err)
	})
}

func TestFindConfigFile(t *testing.T) {
	t.Run("finds config in current directory", func(t *testing.T) {
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		
		err := os.Chdir(tempDir)
		require.NoError(t, err)
		
		// Create config file
		configPath := filepath.Join(tempDir, ".igscraper.yaml")
		err = os.WriteFile(configPath, []byte("test: true"), 0644)
		require.NoError(t, err)
		
		cfg := DefaultConfig()
		found := cfg.findConfigFile()
		assert.Equal(t, ".igscraper.yaml", found)
	})
	
	t.Run("no config file found", func(t *testing.T) {
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		
		err := os.Chdir(tempDir)
		require.NoError(t, err)
		
		cfg := DefaultConfig()
		found := cfg.findConfigFile()
		assert.Empty(t, found)
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func(*Config)
		expectError bool
		errorContains []string
	}{
		{
			name: "valid config",
			setupConfig: func(cfg *Config) {
				cfg.Instagram.SessionID = "valid_session"
				cfg.Instagram.CSRFToken = "valid_csrf"
			},
			expectError: false,
		},
		{
			name: "missing credentials",
			setupConfig: func(cfg *Config) {
				cfg.Instagram.SessionID = ""
				cfg.Instagram.CSRFToken = ""
			},
			expectError: true,
			errorContains: []string{"session ID is required", "CSRF token is required"},
		},
		{
			name: "invalid rate limit",
			setupConfig: func(cfg *Config) {
				cfg.Instagram.SessionID = "valid"
				cfg.Instagram.CSRFToken = "valid"
				cfg.RateLimit.RequestsPerMinute = -1
				cfg.RateLimit.BurstSize = 0
				cfg.RateLimit.MaxRetries = -1
			},
			expectError: true,
			errorContains: []string{
				"requests per minute must be positive",
				"burst size must be positive",
				"max retries cannot be negative",
			},
		},
		{
			name: "invalid download settings",
			setupConfig: func(cfg *Config) {
				cfg.Instagram.SessionID = "valid"
				cfg.Instagram.CSRFToken = "valid"
				cfg.Download.ConcurrentDownloads = 0
				cfg.Download.DownloadTimeout = 0
			},
			expectError: true,
			errorContains: []string{
				"concurrent downloads must be positive",
				"download timeout must be positive",
			},
		},
		{
			name: "too many concurrent downloads",
			setupConfig: func(cfg *Config) {
				cfg.Instagram.SessionID = "valid"
				cfg.Instagram.CSRFToken = "valid"
				cfg.Download.ConcurrentDownloads = 15
			},
			expectError: true,
			errorContains: []string{"concurrent downloads should not exceed 10"},
		},
		{
			name: "invalid output settings",
			setupConfig: func(cfg *Config) {
				cfg.Instagram.SessionID = "valid"
				cfg.Instagram.CSRFToken = "valid"
				cfg.Output.BaseDirectory = ""
				cfg.Output.FileNamePattern = ""
			},
			expectError: true,
			errorContains: []string{
				"output directory is required",
				"file name pattern is required",
			},
		},
		{
			name: "invalid log level",
			setupConfig: func(cfg *Config) {
				cfg.Instagram.SessionID = "valid"
				cfg.Instagram.CSRFToken = "valid"
				cfg.Logging.Level = "invalid"
			},
			expectError: true,
			errorContains: []string{"invalid log level"},
		},
		{
			name: "invalid notification type",
			setupConfig: func(cfg *Config) {
				cfg.Instagram.SessionID = "valid"
				cfg.Instagram.CSRFToken = "valid"
				cfg.Notifications.NotificationType = "invalid"
			},
			expectError: true,
			errorContains: []string{"invalid notification type"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.setupConfig(cfg)
			
			err := cfg.Validate()
			
			if tt.expectError {
				assert.Error(t, err)
				for _, contains := range tt.errorContains {
					assert.Contains(t, err.Error(), contains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSave(t *testing.T) {
	t.Run("save to new file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "saved_config.yaml")
		
		cfg := DefaultConfig()
		cfg.Instagram.SessionID = "save_test"
		cfg.Instagram.CSRFToken = "save_csrf"
		
		err := cfg.Save(configPath)
		require.NoError(t, err)
		
		// Verify file exists
		_, err = os.Stat(configPath)
		assert.NoError(t, err)
		
		// Load and verify
		loadedCfg := DefaultConfig()
		err = loadedCfg.LoadFromFile(configPath)
		require.NoError(t, err)
		
		assert.Equal(t, cfg.Instagram.SessionID, loadedCfg.Instagram.SessionID)
		assert.Equal(t, cfg.Instagram.CSRFToken, loadedCfg.Instagram.CSRFToken)
	})
	
	t.Run("creates directory if needed", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "subdir", "config.yaml")
		
		cfg := DefaultConfig()
		err := cfg.Save(configPath)
		require.NoError(t, err)
		
		// Verify directory was created
		_, err = os.Stat(filepath.Dir(configPath))
		assert.NoError(t, err)
	})
	
	t.Run("overwrites existing file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		
		// Create initial file
		cfg1 := DefaultConfig()
		cfg1.Instagram.SessionID = "first"
		err := cfg1.Save(configPath)
		require.NoError(t, err)
		
		// Overwrite with new config
		cfg2 := DefaultConfig()
		cfg2.Instagram.SessionID = "second"
		err = cfg2.Save(configPath)
		require.NoError(t, err)
		
		// Load and verify
		loadedCfg := DefaultConfig()
		err = loadedCfg.LoadFromFile(configPath)
		require.NoError(t, err)
		
		assert.Equal(t, "second", loadedCfg.Instagram.SessionID)
	})
}

func TestMergeCommandLineFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    map[string]interface{}
		expected func(*Config)
	}{
		{
			name: "merge all flags",
			flags: map[string]interface{}{
				"session-id":           "flag_session",
				"csrf-token":           "flag_csrf",
				"output":               "/flag/output",
				"concurrent-downloads": 7,
				"requests-per-minute":  90,
				"notifications-enabled": false,
				"log-level":            "error",
			},
			expected: func(cfg *Config) {
				cfg.Instagram.SessionID = "flag_session"
				cfg.Instagram.CSRFToken = "flag_csrf"
				cfg.Output.BaseDirectory = "/flag/output"
				cfg.Download.ConcurrentDownloads = 7
				cfg.RateLimit.RequestsPerMinute = 90
				cfg.Notifications.Enabled = false
				cfg.Logging.Level = "error"
			},
		},
		{
			name: "partial flags",
			flags: map[string]interface{}{
				"session-id": "partial_session",
				"output":     "/partial/output",
			},
			expected: func(cfg *Config) {
				cfg.Instagram.SessionID = "partial_session"
				cfg.Output.BaseDirectory = "/partial/output"
			},
		},
		{
			name: "empty flags",
			flags: map[string]interface{}{},
			expected: func(cfg *Config) {
				// No changes
			},
		},
		{
			name: "invalid flag types ignored",
			flags: map[string]interface{}{
				"concurrent-downloads": "not a number",
				"requests-per-minute":  -1,
			},
			expected: func(cfg *Config) {
				// Invalid values should be ignored
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			originalCfg := *cfg // Copy for comparison
			
			cfg.MergeCommandLineFlags(tt.flags)
			
			expectedCfg := originalCfg
			tt.expected(&expectedCfg)
			
			// Compare specific fields that should have changed
			if sessionID, ok := tt.flags["session-id"].(string); ok && sessionID != "" {
				assert.Equal(t, expectedCfg.Instagram.SessionID, cfg.Instagram.SessionID)
			}
			if csrfToken, ok := tt.flags["csrf-token"].(string); ok && csrfToken != "" {
				assert.Equal(t, expectedCfg.Instagram.CSRFToken, cfg.Instagram.CSRFToken)
			}
			if output, ok := tt.flags["output"].(string); ok && output != "" {
				assert.Equal(t, expectedCfg.Output.BaseDirectory, cfg.Output.BaseDirectory)
			}
			if concurrent, ok := tt.flags["concurrent-downloads"].(int); ok && concurrent > 0 {
				assert.Equal(t, expectedCfg.Download.ConcurrentDownloads, cfg.Download.ConcurrentDownloads)
			}
			if rpm, ok := tt.flags["requests-per-minute"].(int); ok && rpm > 0 {
				assert.Equal(t, expectedCfg.RateLimit.RequestsPerMinute, cfg.RateLimit.RequestsPerMinute)
			}
			if _, ok := tt.flags["notifications-enabled"].(bool); ok {
				assert.Equal(t, expectedCfg.Notifications.Enabled, cfg.Notifications.Enabled)
			}
			if logLevel, ok := tt.flags["log-level"].(string); ok && logLevel != "" {
				assert.Equal(t, expectedCfg.Logging.Level, cfg.Logging.Level)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("precedence order", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create config file
		configPath := filepath.Join(tempDir, "config.yaml")
		configContent := `
instagram:
  session_id: file_session
  csrf_token: file_csrf
output:
  base_directory: /file/output
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)
		
		// Set environment variables
		os.Setenv("IGSCRAPER_SESSION_ID", "env_session")
		os.Setenv("IGSCRAPER_OUTPUT_DIR", "/env/output")
		defer os.Unsetenv("IGSCRAPER_SESSION_ID")
		defer os.Unsetenv("IGSCRAPER_OUTPUT_DIR")
		
		// Command line flags
		flags := map[string]interface{}{
			"session-id": "flag_session",
		}
		
		cfg, err := Load(configPath, flags)
		require.NoError(t, err)
		
		// Verify precedence: flags > env > file > defaults
		assert.Equal(t, "flag_session", cfg.Instagram.SessionID) // From flags
		assert.Equal(t, "file_csrf", cfg.Instagram.CSRFToken)    // From file (no env or flag)
		assert.Equal(t, "/env/output", cfg.Output.BaseDirectory) // From env (no flag)
	})
	
	t.Run("validation failure", func(t *testing.T) {
		flags := map[string]interface{}{
			"session-id": "", // Invalid empty session
		}
		
		cfg, err := Load("", flags)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration validation failed")
		assert.Nil(t, cfg)
	})
	
	t.Run("loads .env file", func(t *testing.T) {
		tempDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		
		err := os.Chdir(tempDir)
		require.NoError(t, err)
		
		// Create .env file
		envContent := `IGSCRAPER_SESSION_ID=dotenv_session
IGSCRAPER_CSRF_TOKEN=dotenv_csrf`
		err = os.WriteFile(".env", []byte(envContent), 0644)
		require.NoError(t, err)
		
		// Clear any existing env vars
		os.Unsetenv("IGSCRAPER_SESSION_ID")
		os.Unsetenv("IGSCRAPER_CSRF_TOKEN")
		
		cfg, err := Load("", nil)
		require.NoError(t, err)
		
		assert.Equal(t, "dotenv_session", cfg.Instagram.SessionID)
		assert.Equal(t, "dotenv_csrf", cfg.Instagram.CSRFToken)
	})
}

func TestConfigSerialization(t *testing.T) {
	t.Run("yaml round trip", func(t *testing.T) {
		original := DefaultConfig()
		original.Instagram.SessionID = "test_session"
		original.Instagram.CSRFToken = "test_csrf"
		original.RateLimit.RequestsPerMinute = 45
		original.Download.ConcurrentDownloads = 8
		
		// Marshal to YAML
		data, err := yaml.Marshal(original)
		require.NoError(t, err)
		
		// Unmarshal back
		var loaded Config
		err = yaml.Unmarshal(data, &loaded)
		require.NoError(t, err)
		
		// Compare key fields
		assert.Equal(t, original.Instagram.SessionID, loaded.Instagram.SessionID)
		assert.Equal(t, original.Instagram.CSRFToken, loaded.Instagram.CSRFToken)
		assert.Equal(t, original.RateLimit.RequestsPerMinute, loaded.RateLimit.RequestsPerMinute)
		assert.Equal(t, original.Download.ConcurrentDownloads, loaded.Download.ConcurrentDownloads)
	})
}

func TestDurationParsing(t *testing.T) {
	t.Run("parse duration from yaml", func(t *testing.T) {
		yamlContent := `
rate_limit:
  retry_delay: 10s
retry:
  base_delay: 500ms
  max_delay: 1m30s
download:
  download_timeout: 45s
`
		var cfg Config
		err := yaml.Unmarshal([]byte(yamlContent), &cfg)
		require.NoError(t, err)
		
		assert.Equal(t, 10*time.Second, cfg.RateLimit.RetryDelay)
		assert.Equal(t, 500*time.Millisecond, cfg.Retry.BaseDelay)
		assert.Equal(t, 90*time.Second, cfg.Retry.MaxDelay)
		assert.Equal(t, 45*time.Second, cfg.Download.DownloadTimeout)
	})
}

// Benchmark tests
func BenchmarkDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}

func BenchmarkValidate(b *testing.B) {
	cfg := DefaultConfig()
	cfg.Instagram.SessionID = "bench_session"
	cfg.Instagram.CSRFToken = "bench_csrf"
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = cfg.Validate()
	}
}

func BenchmarkLoadFromEnv(b *testing.B) {
	os.Setenv("IGSCRAPER_SESSION_ID", "bench_session")
	os.Setenv("IGSCRAPER_CSRF_TOKEN", "bench_csrf")
	defer os.Unsetenv("IGSCRAPER_SESSION_ID")
	defer os.Unsetenv("IGSCRAPER_CSRF_TOKEN")
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		cfg := DefaultConfig()
		_ = cfg.LoadFromEnv()
	}
}

func BenchmarkSaveAndLoad(b *testing.B) {
	tempDir := b.TempDir()
	configPath := filepath.Join(tempDir, "bench_config.yaml")
	
	cfg := DefaultConfig()
	cfg.Instagram.SessionID = "bench_session"
	cfg.Instagram.CSRFToken = "bench_csrf"
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = cfg.Save(configPath)
		loadedCfg := DefaultConfig()
		_ = loadedCfg.LoadFromFile(configPath)
	}
}