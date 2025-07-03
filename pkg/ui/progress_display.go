package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ProgressDisplay provides a clean, minimal progress display
type ProgressDisplay struct {
	mu              sync.Mutex
	username        string
	totalPhotos     int
	downloadedCount int
	currentPhoto    string
	startTime       time.Time
	lastUpdate      time.Time
	bytesDownloaded int64
	errors          int
	isDebug         bool
}

// NewProgressDisplay creates a new progress display
func NewProgressDisplay(username string, totalPhotos int, debug bool) *ProgressDisplay {
	return &ProgressDisplay{
		username:    username,
		totalPhotos: totalPhotos,
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
		isDebug:     debug,
	}
}

// StartDownload marks the start of a new download
func (p *ProgressDisplay) StartDownload(shortcode string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.currentPhoto = shortcode
	p.lastUpdate = time.Now()
	
	if !p.isDebug {
		p.printProgress()
	}
}

// CompleteDownload marks a download as complete
func (p *ProgressDisplay) CompleteDownload(shortcode string, size int64, metadata map[string]interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.downloadedCount++
	p.bytesDownloaded += size
	p.lastUpdate = time.Now()
	
	if !p.isDebug {
		p.printProgress()
	} else {
		// In debug mode, show more details
		p.printDebugComplete(shortcode, size, metadata)
	}
}

// FailDownload marks a download as failed
func (p *ProgressDisplay) FailDownload(shortcode string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.errors++
	p.lastUpdate = time.Now()
	
	if !p.isDebug {
		p.printProgress()
	} else {
		fmt.Printf("\n%s Failed: %s - %v\n", Red("✗"), shortcode, err)
	}
}

// printProgress prints the minimal progress line
func (p *ProgressDisplay) printProgress() {
	// Calculate stats
	elapsed := time.Since(p.startTime)
	rate := float64(p.downloadedCount) / elapsed.Minutes()
	eta := p.calculateETA()
	
	// Build progress bar
	progress := float64(p.downloadedCount) / float64(p.totalPhotos)
	barWidth := 20
	filled := int(progress * float64(barWidth))
	bar := strings.Repeat("━", filled) + strings.Repeat("─", barWidth-filled)
	
	// Format line
	line := fmt.Sprintf("\r%s [%s] %d/%d • %.1f/min • %s • %s",
		Cyan(p.username),
		bar,
		p.downloadedCount,
		p.totalPhotos,
		rate,
		p.formatBytes(p.bytesDownloaded),
		eta,
	)
	
	// Add current photo if downloading
	if p.currentPhoto != "" {
		line += fmt.Sprintf(" • %s", p.currentPhoto)
	}
	
	// Add errors if any
	if p.errors > 0 {
		line += fmt.Sprintf(" • %s", Red(fmt.Sprintf("%d errors", p.errors)))
	}
	
	// Clear line and print
	fmt.Printf("\r%s\r%s", strings.Repeat(" ", 120), line)
}

// printDebugComplete prints detailed info in debug mode
func (p *ProgressDisplay) printDebugComplete(shortcode string, size int64, metadata map[string]interface{}) {
	fmt.Printf("\n%s %s • %s", 
		Green("✓"),
		shortcode,
		p.formatBytes(size),
	)
	
	// Add metadata if available
	if caption, ok := metadata["caption"].(string); ok && caption != "" {
		// Truncate caption
		if len(caption) > 50 {
			caption = caption[:47] + "..."
		}
		fmt.Printf(" • %s", Dim(caption))
	}
	
	if likes, ok := metadata["likes"].(int); ok {
		fmt.Printf(" • %s", Dim(fmt.Sprintf("♥ %d", likes)))
	}
	
	fmt.Println()
}

// Complete marks the entire operation as complete
func (p *ProgressDisplay) Complete() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	elapsed := time.Since(p.startTime)
	
	fmt.Printf("\n\n%s Downloaded %d photos from @%s\n",
		Green("✓"),
		p.downloadedCount,
		p.username,
	)
	
	// Summary stats
	fmt.Printf("  %s %s in %s (%.1f photos/min)\n",
		Dim("•"),
		p.formatBytes(p.bytesDownloaded),
		p.formatDuration(elapsed),
		float64(p.downloadedCount)/elapsed.Minutes(),
	)
	
	if p.errors > 0 {
		fmt.Printf("  %s %d downloads failed\n", 
			Dim("•"),
			p.errors,
		)
	}
}

// calculateETA estimates time remaining
func (p *ProgressDisplay) calculateETA() string {
	if p.downloadedCount == 0 {
		return "calculating..."
	}
	
	remaining := p.totalPhotos - p.downloadedCount
	elapsed := time.Since(p.startTime)
	rate := float64(p.downloadedCount) / elapsed.Seconds()
	
	if rate == 0 {
		return "calculating..."
	}
	
	etaSeconds := float64(remaining) / rate
	eta := time.Duration(etaSeconds) * time.Second
	
	return p.formatDuration(eta)
}

// formatDuration formats a duration in a human-readable way
func (p *ProgressDisplay) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

// formatBytes formats bytes in a human-readable way
func (p *ProgressDisplay) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// RateLimitWarning shows a rate limit warning
func (p *ProgressDisplay) RateLimitWarning(waitTime time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	fmt.Printf("\n%s Rate limit reached. Waiting %s...\n", 
		Yellow("⚠"),
		p.formatDuration(waitTime),
	)
}

// ScanningBatch indicates scanning a new batch
func (p *ProgressDisplay) ScanningBatch(page int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.isDebug {
		fmt.Printf("\n%s Scanning page %d...\n", Magenta("→"), page)
	}
}

// UpdateTotal updates the total photo count
func (p *ProgressDisplay) UpdateTotal(total int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.totalPhotos = total
}

// SetDownloadedCount sets the initial downloaded count (for resume)
func (p *ProgressDisplay) SetDownloadedCount(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.downloadedCount = count
}