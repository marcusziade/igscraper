package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TUI represents the terminal user interface
type TUI struct {
	program *tea.Program
	model   *Model
}

// NewTUI creates a new TUI instance
func NewTUI(maxConcurrent int) *TUI {
	model := NewModel(maxConcurrent)
	program := tea.NewProgram(&model, tea.WithAltScreen())
	
	return &TUI{
		program: program,
		model:   &model,
	}
}

// Start starts the TUI
func (t *TUI) Start() error {
	go func() {
		// Send initial tick to start the spinner
		time.Sleep(100 * time.Millisecond)
		t.program.Send(TickMsg(time.Now()))
	}()
	
	_, err := t.program.Run()
	return err
}

// Stop stops the TUI gracefully
func (t *TUI) Stop() {
	t.program.Quit()
}

// Send sends a message to the TUI
func (t *TUI) Send(msg tea.Msg) {
	if t.program != nil {
		t.program.Send(msg)
	}
}

// StartDownload notifies the TUI that a download has started
func (t *TUI) StartDownload(id, username, filename string, size int64) {
	t.Send(SendDownloadStart(id, username, filename, size))
}

// UpdateDownloadProgress updates the progress of a download
func (t *TUI) UpdateDownloadProgress(id string, downloaded int64, speed float64) {
	t.Send(SendDownloadProgress(id, downloaded, speed))
}

// CompleteDownload notifies the TUI that a download has completed
func (t *TUI) CompleteDownload(id string) {
	t.Send(SendDownloadComplete(id))
}

// FailDownload notifies the TUI that a download has failed
func (t *TUI) FailDownload(id string, err error) {
	t.Send(SendDownloadError(id, err))
}

// UpdateRateLimit updates the rate limit status
func (t *TUI) UpdateRateLimit(used, max int, resetAt time.Time) {
	t.Send(SendRateLimitUpdate(used, max, resetAt))
}

// Log sends a log message to the TUI
func (t *TUI) Log(level, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	t.Send(SendLog(level, message))
}

// LogInfo logs an info message
func (t *TUI) LogInfo(format string, args ...interface{}) {
	t.Log("INFO", format, args...)
}

// LogSuccess logs a success message
func (t *TUI) LogSuccess(format string, args ...interface{}) {
	t.Log("SUCCESS", format, args...)
}

// LogWarning logs a warning message
func (t *TUI) LogWarning(format string, args ...interface{}) {
	t.Log("WARN", format, args...)
}

// LogError logs an error message
func (t *TUI) LogError(format string, args ...interface{}) {
	t.Log("ERROR", format, args...)
}

// IsPaused returns whether downloads are paused
func (t *TUI) IsPaused() bool {
	t.model.mu.RLock()
	defer t.model.mu.RUnlock()
	return t.model.isPaused
}