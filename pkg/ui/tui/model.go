package tui

import (
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DownloadState represents the state of a download
type DownloadState int

const (
	DownloadPending DownloadState = iota
	DownloadActive
	DownloadCompleted
	DownloadFailed
)

// DownloadItem represents a single download
type DownloadItem struct {
	ID          string
	Username    string
	Filename    string
	Size        int64
	Downloaded  int64
	State       DownloadState
	StartTime   time.Time
	Speed       float64
	Error       error
}

// Model represents the TUI model
type Model struct {
	// UI components
	spinner      spinner.Model
	progressBars map[string]progress.Model
	
	// Download state
	downloads      map[string]*DownloadItem
	downloadOrder  []string
	activeDownloads int
	maxConcurrent  int
	
	// Stats
	totalDownloaded   int
	totalSize         int64
	sessionStartTime  time.Time
	
	// Rate limiting
	rateLimitMax      int
	rateLimitUsed     int
	rateLimitResetAt  time.Time
	
	// UI state
	width         int
	height        int
	showHelp      bool
	isPaused      bool
	logMessages   []LogMessage
	maxLogMessages int
	
	// Mutex for thread safety
	mu sync.RWMutex
}

// LogMessage represents a log entry
type LogMessage struct {
	Time    time.Time
	Level   string
	Message string
	Color   lipgloss.Color
}

// NewModel creates a new TUI model
func NewModel(maxConcurrent int) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(neonCyan)
	
	return Model{
		spinner:          s,
		progressBars:     make(map[string]progress.Model),
		downloads:        make(map[string]*DownloadItem),
		downloadOrder:    []string{},
		maxConcurrent:    maxConcurrent,
		sessionStartTime: time.Now(),
		logMessages:      []LogMessage{},
		maxLogMessages:   50,
		rateLimitMax:     100, // Default rate limit
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

// AddDownload adds a new download to the queue
func (m *Model) AddDownload(id, username, filename string, size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.downloads[id] = &DownloadItem{
		ID:       id,
		Username: username,
		Filename: filename,
		Size:     size,
		State:    DownloadPending,
	}
	m.downloadOrder = append(m.downloadOrder, id)
	
	// Create progress bar for this download
	p := progress.New(progress.WithDefaultGradient())
	p.Width = 40
	m.progressBars[id] = p
}

// StartDownload marks a download as active
func (m *Model) StartDownload(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if download, ok := m.downloads[id]; ok {
		download.State = DownloadActive
		download.StartTime = time.Now()
		m.activeDownloads++
	}
}

// UpdateDownloadProgress updates the progress of a download
func (m *Model) UpdateDownloadProgress(id string, downloaded int64, speed float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if download, ok := m.downloads[id]; ok {
		download.Downloaded = downloaded
		download.Speed = speed
	}
}

// CompleteDownload marks a download as completed
func (m *Model) CompleteDownload(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if download, ok := m.downloads[id]; ok {
		download.State = DownloadCompleted
		m.activeDownloads--
		m.totalDownloaded++
		m.totalSize += download.Size
	}
}

// FailDownload marks a download as failed
func (m *Model) FailDownload(id string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if download, ok := m.downloads[id]; ok {
		download.State = DownloadFailed
		download.Error = err
		m.activeDownloads--
	}
}

// UpdateRateLimit updates the rate limit status
func (m *Model) UpdateRateLimit(used, max int, resetAt time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.rateLimitUsed = used
	m.rateLimitMax = max
	m.rateLimitResetAt = resetAt
}

// AddLogMessage adds a log message
func (m *Model) AddLogMessage(level, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	color := dimWhite
	switch level {
	case "ERROR":
		color = lipgloss.Color("#FF0000")
	case "WARN":
		color = neonOrange
	case "SUCCESS":
		color = neonGreen
	case "INFO":
		color = neonCyan
	}
	
	m.logMessages = append(m.logMessages, LogMessage{
		Time:    time.Now(),
		Level:   level,
		Message: message,
		Color:   color,
	})
	
	// Keep only the last N messages
	if len(m.logMessages) > m.maxLogMessages {
		m.logMessages = m.logMessages[len(m.logMessages)-m.maxLogMessages:]
	}
}

// GetActiveDownloads returns a slice of active downloads
func (m *Model) GetActiveDownloads() []*DownloadItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var active []*DownloadItem
	for _, id := range m.downloadOrder {
		if download := m.downloads[id]; download != nil && download.State == DownloadActive {
			active = append(active, download)
		}
	}
	return active
}

// GetPendingDownloads returns a slice of pending downloads
func (m *Model) GetPendingDownloads() []*DownloadItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var pending []*DownloadItem
	for _, id := range m.downloadOrder {
		if download := m.downloads[id]; download != nil && download.State == DownloadPending {
			pending = append(pending, download)
		}
	}
	return pending
}

// GetCompletedDownloads returns a slice of completed downloads
func (m *Model) GetCompletedDownloads() []*DownloadItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var completed []*DownloadItem
	for _, id := range m.downloadOrder {
		if download := m.downloads[id]; download != nil && download.State == DownloadCompleted {
			completed = append(completed, download)
		}
	}
	return completed
}

// GetDownloadStats returns various statistics
func (m *Model) GetDownloadStats() (totalSpeed float64, avgSpeed float64, eta time.Duration) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, download := range m.downloads {
		if download.State == DownloadActive {
			totalSpeed += download.Speed
		}
	}
	
	if m.totalDownloaded > 0 {
		elapsed := time.Since(m.sessionStartTime)
		avgSpeed = float64(m.totalSize) / elapsed.Seconds()
	}
	
	// Calculate ETA based on pending downloads
	pendingCount := 0
	for _, download := range m.downloads {
		if download.State == DownloadPending {
			pendingCount++
		}
	}
	
	if avgSpeed > 0 && pendingCount > 0 {
		// Rough estimate based on average download time
		avgDownloadTime := time.Since(m.sessionStartTime) / time.Duration(m.totalDownloaded+1)
		eta = avgDownloadTime * time.Duration(pendingCount)
	}
	
	return
}

// FormatBytes formats bytes to human readable format
func FormatBytes(bytes int64) string {
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

// FormatSpeed formats speed in bytes per second
func FormatSpeed(bytesPerSecond float64) string {
	return fmt.Sprintf("%s/s", FormatBytes(int64(bytesPerSecond)))
}