// Package logger provides a structured logging interface for the Instagram scraper.
//
// It wraps the zerolog library to provide a clean, easy-to-use API with support for:
// - Multiple log levels (Debug, Info, Warn, Error, Fatal)
// - Structured logging with fields
// - Pretty console output with colors
// - File output with rotation support
// - Context support for request tracing
// - Global logger instance for easy access
//
// Basic Usage:
//
//	import "igscraper/pkg/logger"
//
//	// Initialize the global logger
//	cfg := &config.LoggingConfig{
//	    Level: "info",
//	    File: "/var/log/igscraper.log",
//	}
//	err := logger.Initialize(cfg)
//
//	// Use the global logger
//	logger.Info("Application started")
//	logger.WithField("username", "john_doe").Info("User logged in")
//	logger.WithError(err).Error("Failed to download image")
//
// Advanced Usage:
//
//	// Create a logger instance with fields
//	log := logger.GetLogger().
//	    WithField("component", "downloader").
//	    WithField("session_id", "12345")
//
//	// Use structured logging
//	log.InfoWithFields("Download completed", map[string]interface{}{
//	    "file": "image.jpg",
//	    "size": 1024000,
//	    "duration": time.Second * 5,
//	})
//
// The logger supports the following configuration options:
// - Level: Log level (debug, info, warn, error, fatal)
// - File: Path to log file (empty for console only)
// - MaxSize: Maximum size in MB before rotation
// - MaxBackups: Number of old log files to keep
// - MaxAge: Maximum age in days for log files
// - Compress: Whether to compress old log files
package logger