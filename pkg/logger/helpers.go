package logger

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/rs/zerolog"
)

// LoggerWithCaller adds caller information to the logger
func LoggerWithCaller(skip int) Logger {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return GetLogger()
	}
	
	// Extract just the filename without the full path
	parts := strings.Split(file, "/")
	filename := parts[len(parts)-1]
	
	return GetLogger().WithField("caller", fmt.Sprintf("%s:%d", filename, line))
}

// LogRequest logs HTTP request information
func LogRequest(method, url string, statusCode int, duration float64) {
	fields := map[string]interface{}{
		"method":      method,
		"url":         url,
		"status_code": statusCode,
		"duration_ms": duration,
	}
	
	if statusCode >= 200 && statusCode < 300 {
		GetLogger().InfoWithFields("HTTP request completed", fields)
	} else if statusCode >= 400 && statusCode < 500 {
		GetLogger().WarnWithFields("HTTP request client error", fields)
	} else if statusCode >= 500 {
		GetLogger().ErrorWithFields("HTTP request server error", fields)
	}
}

// LogDownload logs download operations
func LogDownload(username, mediaID, mediaType string, success bool, err error) {
	fields := map[string]interface{}{
		"username":   username,
		"media_id":   mediaID,
		"media_type": mediaType,
		"success":    success,
	}
	
	logger := GetLogger().WithFields(fields)
	
	if err != nil {
		logger.WithError(err).Error("Download failed")
	} else if success {
		logger.Info("Download completed")
	} else {
		logger.Warn("Download skipped")
	}
}

// LogRateLimit logs rate limiting events
func LogRateLimit(endpoint string, retryAfter int) {
	GetLogger().WithFields(map[string]interface{}{
		"endpoint":     endpoint,
		"retry_after":  retryAfter,
		"action":       "rate_limited",
	}).Warn("Rate limit reached, backing off")
}

// LogScrapeProgress logs scraping progress
func LogScrapeProgress(username string, scraped, total int) {
	percentage := 0.0
	if total > 0 {
		percentage = float64(scraped) / float64(total) * 100
	}
	
	GetLogger().WithFields(map[string]interface{}{
		"username":   username,
		"scraped":    scraped,
		"total":      total,
		"percentage": fmt.Sprintf("%.1f%%", percentage),
	}).Info("Scraping progress")
}

// LogComponentStart logs when a component starts
func LogComponentStart(component string, config map[string]interface{}) {
	logger := GetLogger().WithField("component", component)
	
	if len(config) > 0 {
		logger = logger.WithFields(config)
	}
	
	logger.Info("Component started")
}

// LogComponentStop logs when a component stops
func LogComponentStop(component string, reason string) {
	GetLogger().WithFields(map[string]interface{}{
		"component": component,
		"reason":    reason,
	}).Info("Component stopped")
}

// LogMetrics logs performance metrics
func LogMetrics(operation string, metrics map[string]interface{}) {
	fields := map[string]interface{}{
		"operation": operation,
		"type":      "metrics",
	}
	
	// Merge metrics into fields
	for k, v := range metrics {
		fields[k] = v
	}
	
	GetLogger().InfoWithFields("Performance metrics", fields)
}

// MustGetLogger gets the logger or panics if it fails
func MustGetLogger() Logger {
	logger := GetLogger()
	if logger == nil {
		panic("logger not initialized")
	}
	return logger
}

// NewNopLogger creates a no-operation logger for testing
func NewNopLogger() Logger {
	return &nopLogger{}
}

// nopLogger is a logger that does nothing (useful for testing)
type nopLogger struct{}

func (n *nopLogger) Debug(msg string)                                             {}
func (n *nopLogger) Info(msg string)                                              {}
func (n *nopLogger) Warn(msg string)                                              {}
func (n *nopLogger) Error(msg string)                                             {}
func (n *nopLogger) Fatal(msg string)                                             {}
func (n *nopLogger) WithField(key string, value interface{}) Logger               { return n }
func (n *nopLogger) WithFields(fields map[string]interface{}) Logger              { return n }
func (n *nopLogger) WithError(err error) Logger                                   { return n }
func (n *nopLogger) WithContext(ctx context.Context) Logger                       { return n }
func (n *nopLogger) DebugWithFields(msg string, fields map[string]interface{})    {}
func (n *nopLogger) InfoWithFields(msg string, fields map[string]interface{})     {}
func (n *nopLogger) WarnWithFields(msg string, fields map[string]interface{})     {}
func (n *nopLogger) ErrorWithFields(msg string, fields map[string]interface{})    {}
func (n *nopLogger) FatalWithFields(msg string, fields map[string]interface{})    {}
func (n *nopLogger) GetZerolog() *zerolog.Logger                                  { return nil }