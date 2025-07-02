package ui

import (
	"fmt"
	"strings"
	"time"
)

const (
	ProgressBar   = "█"
	ProgressEmpty = "░"
	MaxPerHour    = 100 // Conservative rate limit
)

// StatusTracker keeps track of download progress
type StatusTracker struct {
	TotalDownloaded int
	CurrentBatch    int
	StartTime       time.Time
}

// NewStatusTracker creates a new status tracker
func NewStatusTracker() *StatusTracker {
	return &StatusTracker{
		StartTime: time.Now(),
	}
}

// IncrementDownloaded increments both total and current batch counters
func (st *StatusTracker) IncrementDownloaded() {
	st.TotalDownloaded++
	st.CurrentBatch++
}

// ResetBatch resets the current batch counter
func (st *StatusTracker) ResetBatch() {
	st.CurrentBatch = 0
}

// GetBatchProgress returns a formatted progress bar for the current batch
func (st *StatusTracker) GetBatchProgress() string {
	const width = 20
	progress := float64(st.CurrentBatch) / float64(MaxPerHour)
	filled := int(progress * float64(width))

	bar := strings.Repeat(ProgressBar, filled) +
		strings.Repeat(ProgressEmpty, width-filled)

	return fmt.Sprintf("[%s] %d/%d", bar, st.CurrentBatch, MaxPerHour)
}

// GetElapsedTime returns the elapsed time since tracking started
func (st *StatusTracker) GetElapsedTime() time.Duration {
	return time.Since(st.StartTime)
}

// GetDownloadRate returns the average download rate (items per minute)
func (st *StatusTracker) GetDownloadRate() float64 {
	elapsed := st.GetElapsedTime().Minutes()
	if elapsed == 0 {
		return 0
	}
	return float64(st.TotalDownloaded) / elapsed
}

// PrintProgress prints the current progress status
func (st *StatusTracker) PrintProgress() {
	fmt.Printf("\r%s Total: %d | Batch: %s",
		Green("[EXTRACTED]"),
		st.TotalDownloaded,
		st.GetBatchProgress())
}

// PrintBatchStatus prints the current batch scanning status
func (st *StatusTracker) PrintBatchStatus() {
	fmt.Printf("\n%s %s\n", Magenta("[SCANNING]"), Yellow(st.GetBatchProgress()))
}

// IsRateLimitReached checks if the current batch has reached the rate limit
func (st *StatusTracker) IsRateLimitReached() bool {
	return st.CurrentBatch >= MaxPerHour
}

// GetDownloadedCount returns the total number of downloaded items
func (st *StatusTracker) GetDownloadedCount() int {
	return st.TotalDownloaded
}

// SetDownloadedCount sets the total downloaded count (used for resuming)
func (st *StatusTracker) SetDownloadedCount(count int) {
	st.TotalDownloaded = count
}