package ui

import "time"

// TUI is an interface for terminal user interfaces
type TUI interface {
	StartDownload(id, username, filename string, size int64)
	UpdateDownloadProgress(id string, downloaded int64, speed float64)
	CompleteDownload(id string)
	FailDownload(id string, err error)
	UpdateRateLimit(used, max int, resetAt time.Time)
	LogInfo(format string, args ...interface{})
	LogSuccess(format string, args ...interface{})
	LogWarning(format string, args ...interface{})
	LogError(format string, args ...interface{})
	IsPaused() bool
}