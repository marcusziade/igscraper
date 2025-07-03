package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Message types for the TUI

// DownloadStartMsg is sent when a download starts
type DownloadStartMsg struct {
	ID       string
	Username string
	Filename string
	Size     int64
}

// DownloadProgressMsg is sent to update download progress
type DownloadProgressMsg struct {
	ID         string
	Downloaded int64
	Speed      float64
}

// DownloadCompleteMsg is sent when a download completes
type DownloadCompleteMsg struct {
	ID string
}

// DownloadErrorMsg is sent when a download fails
type DownloadErrorMsg struct {
	ID    string
	Error error
}

// RateLimitUpdateMsg is sent to update rate limit status
type RateLimitUpdateMsg struct {
	Used    int
	Max     int
	ResetAt time.Time
}

// LogMsg is sent to add a log message
type LogMsg struct {
	Level   string
	Message string
}

// WindowSizeMsg is sent when the terminal is resized
type WindowSizeMsg struct {
	Width  int
	Height int
}

// TickMsg is sent periodically to update the UI
type TickMsg time.Time

// Update handles all messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case TickMsg:
		// Regular UI update tick
		return m, tea.Batch(
			tickCmd(),
			m.spinner.Tick,
		)

	case DownloadStartMsg:
		m.AddDownload(msg.ID, msg.Username, msg.Filename, msg.Size)
		m.StartDownload(msg.ID)
		m.AddLogMessage("INFO", "Started download: "+msg.Filename)
		return m, nil

	case DownloadProgressMsg:
		m.UpdateDownloadProgress(msg.ID, msg.Downloaded, msg.Speed)
		return m, nil

	case DownloadCompleteMsg:
		m.CompleteDownload(msg.ID)
		if download, ok := m.downloads[msg.ID]; ok {
			m.AddLogMessage("SUCCESS", "Completed: "+download.Filename)
		}
		return m, nil

	case DownloadErrorMsg:
		m.FailDownload(msg.ID, msg.Error)
		if download, ok := m.downloads[msg.ID]; ok {
			m.AddLogMessage("ERROR", "Failed: "+download.Filename+" - "+msg.Error.Error())
		}
		return m, nil

	case RateLimitUpdateMsg:
		m.UpdateRateLimit(msg.Used, msg.Max, msg.ResetAt)
		return m, nil

	case LogMsg:
		m.AddLogMessage(msg.Level, msg.Message)
		return m, nil

	case WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "Q", "ctrl+c":
		return m, tea.Quit

	case "p", "P":
		m.isPaused = !m.isPaused
		if m.isPaused {
			m.AddLogMessage("WARN", "Downloads paused by user")
		} else {
			m.AddLogMessage("INFO", "Downloads resumed by user")
		}
		return m, nil

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "ctrl+l":
		// Clear logs
		m.mu.Lock()
		m.logMessages = []LogMessage{}
		m.mu.Unlock()
		return m, nil
	}

	return m, nil
}

// Commands

// tickCmd returns a command that sends a tick message
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Helper functions for external use

// SendDownloadStart creates a message to start a download
func SendDownloadStart(id, username, filename string, size int64) tea.Msg {
	return DownloadStartMsg{
		ID:       id,
		Username: username,
		Filename: filename,
		Size:     size,
	}
}

// SendDownloadProgress creates a message to update download progress
func SendDownloadProgress(id string, downloaded int64, speed float64) tea.Msg {
	return DownloadProgressMsg{
		ID:         id,
		Downloaded: downloaded,
		Speed:      speed,
	}
}

// SendDownloadComplete creates a message when download completes
func SendDownloadComplete(id string) tea.Msg {
	return DownloadCompleteMsg{ID: id}
}

// SendDownloadError creates a message when download fails
func SendDownloadError(id string, err error) tea.Msg {
	return DownloadErrorMsg{ID: id, Error: err}
}

// SendRateLimitUpdate creates a message to update rate limit
func SendRateLimitUpdate(used, max int, resetAt time.Time) tea.Msg {
	return RateLimitUpdateMsg{
		Used:    used,
		Max:     max,
		ResetAt: resetAt,
	}
}

// SendLog creates a log message
func SendLog(level, message string) tea.Msg {
	return LogMsg{Level: level, Message: message}
}