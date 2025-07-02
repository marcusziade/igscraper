package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// View renders the entire TUI
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	// Build the UI layout
	var sections []string

	// Logo
	sections = append(sections, m.renderLogo())

	// Main content area with two columns
	leftColumn := m.renderLeftColumn()
	rightColumn := m.renderRightColumn()
	
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
		"  ", // spacing
		rightColumn,
	)
	sections = append(sections, mainContent)

	// Help
	if m.showHelp {
		sections = append(sections, m.renderHelp())
	} else {
		sections = append(sections, helpStyle.Render("Press ? for help"))
	}

	// Join all sections vertically
	return baseStyle.Width(m.width).Height(m.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, sections...),
	)
}

// renderLogo renders the cyberpunk logo
func (m Model) renderLogo() string {
	logo := `
╔══════════════════════════════════════════════════════════════╗
║ ██╗███╗   ██╗███████╗████████╗ █████╗  ██████╗ ██████╗  ███╗ ║
║ ██║████╗  ██║██╔════╝╚══██╔══╝██╔══██╗██╔════╝ ██╔══██╗██╔██╗║
║ ██║██╔██╗ ██║███████╗   ██║   ███████║██║  ███╗██████╔╝███████║
║ ██║██║╚██╗██║╚════██║   ██║   ██╔══██║██║   ██║██╔══██╗██╔══██║
║ ██║██║ ╚████║███████║   ██║   ██║  ██║╚██████╔╝██║  ██║██║  ██║
║ ╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝
║        NETRUNNER EDITION - PHOTO EXTRACTION UTILITY v2.0       ║
╚══════════════════════════════════════════════════════════════╝`
	
	return logoStyle.Width(m.width).Render(logo)
}

// renderLeftColumn renders the left side of the UI
func (m Model) renderLeftColumn() string {
	width := (m.width - 4) / 2

	var sections []string

	// Stats panel
	sections = append(sections, m.renderStatsPanel(width))

	// Active downloads panel
	sections = append(sections, m.renderActiveDownloadsPanel(width))

	// Queue panel
	sections = append(sections, m.renderQueuePanel(width))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderRightColumn renders the right side of the UI
func (m Model) renderRightColumn() string {
	width := (m.width - 4) / 2

	var sections []string

	// Rate limit panel
	sections = append(sections, m.renderRateLimitPanel(width))

	// Logs panel
	sections = append(sections, m.renderLogsPanel(width))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderStatsPanel renders the statistics panel
func (m Model) renderStatsPanel(width int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	title := titleStyle.Render(" SYSTEM STATS ")
	
	elapsed := time.Since(m.sessionStartTime)
	totalSpeed, avgSpeed, eta := m.GetDownloadStats()

	stats := []string{
		fmt.Sprintf("%s %s", statsLabelStyle.Render("Session Time:"), statsValueStyle.Render(formatDuration(elapsed))),
		fmt.Sprintf("%s %s", statsLabelStyle.Render("Total Downloaded:"), statsValueStyle.Render(fmt.Sprintf("%d files", m.totalDownloaded))),
		fmt.Sprintf("%s %s", statsLabelStyle.Render("Total Size:"), statsValueStyle.Render(FormatBytes(m.totalSize))),
		fmt.Sprintf("%s %s", statsLabelStyle.Render("Current Speed:"), speedStyle.Render(FormatSpeed(totalSpeed))),
		fmt.Sprintf("%s %s", statsLabelStyle.Render("Average Speed:"), speedStyle.Render(FormatSpeed(avgSpeed))),
		fmt.Sprintf("%s %s", statsLabelStyle.Render("ETA:"), statsValueStyle.Render(formatDuration(eta))),
	}

	if m.isPaused {
		stats = append(stats, warningStyle.Render("⏸  PAUSED"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, stats...)
	
	return panelStyle.Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, content),
	)
}

// renderActiveDownloadsPanel renders the active downloads
func (m Model) renderActiveDownloadsPanel(width int) string {
	title := titleStyle.Render(" ACTIVE DOWNLOADS ")
	
	active := m.GetActiveDownloads()
	
	if len(active) == 0 {
		content := lipgloss.NewStyle().Foreground(dimWhite).Render("No active downloads")
		return panelStyle.Width(width).Render(
			lipgloss.JoinVertical(lipgloss.Left, title, content),
		)
	}

	var downloads []string
	for _, download := range active {
		downloads = append(downloads, m.renderDownloadItem(download, width-4))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, downloads...)
	
	return panelStyle.Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, content),
	)
}

// renderDownloadItem renders a single download with progress bar
func (m Model) renderDownloadItem(item *DownloadItem, width int) string {
	m.mu.RLock()
	progressBar, ok := m.progressBars[item.ID]
	m.mu.RUnlock()
	
	if !ok {
		return ""
	}

	progress := float64(item.Downloaded) / float64(item.Size)
	if progress > 1.0 {
		progress = 1.0
	}

	// Update progress bar
	progressBar.Width = width - 20
	
	info := fmt.Sprintf("%s %s @ %s",
		queueItemActiveStyle.Render(item.Filename),
		lipgloss.NewStyle().Foreground(dimWhite).Render(FormatBytes(item.Downloaded)+"/"+FormatBytes(item.Size)),
		speedStyle.Render(FormatSpeed(item.Speed)),
	)

	bar := progressBar.ViewAs(progress)
	
	return lipgloss.JoinVertical(lipgloss.Left, info, bar)
}

// renderQueuePanel renders the download queue
func (m Model) renderQueuePanel(width int) string {
	title := titleStyle.Render(" DOWNLOAD QUEUE ")
	
	pending := m.GetPendingDownloads()
	completed := m.GetCompletedDownloads()
	
	var items []string
	
	// Show some pending items
	pendingCount := len(pending)
	if pendingCount > 0 {
		items = append(items, warningStyle.Render(fmt.Sprintf("⏳ %d pending", pendingCount)))
		for i := 0; i < 3 && i < pendingCount; i++ {
			items = append(items, queueItemStyle.Render("• "+pending[i].Filename))
		}
		if pendingCount > 3 {
			items = append(items, lipgloss.NewStyle().Foreground(dimWhite).Render(fmt.Sprintf("  ... and %d more", pendingCount-3)))
		}
	}
	
	// Show recent completed
	completedCount := len(completed)
	if completedCount > 0 {
		items = append(items, "", successStyle.Render(fmt.Sprintf("✓ %d completed", completedCount)))
		start := completedCount - 3
		if start < 0 {
			start = 0
		}
		for i := start; i < completedCount; i++ {
			items = append(items, queueItemCompletedStyle.Render("✓ "+completed[i].Filename))
		}
	}
	
	content := lipgloss.JoinVertical(lipgloss.Left, items...)
	
	return panelStyle.Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, content),
	)
}

// renderRateLimitPanel renders the rate limit status
func (m Model) renderRateLimitPanel(width int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	title := titleStyle.Render(" RATE LIMIT STATUS ")
	
	usage := float64(m.rateLimitUsed) / float64(m.rateLimitMax) * 100
	
	// Create progress bar for rate limit
	barWidth := width - 8
	filled := int(usage * float64(barWidth) / 100)
	empty := barWidth - filled
	
	barStyle := GetRateLimitStyle(usage)
	bar := barStyle.Render(strings.Repeat("█", filled)) + 
		progressEmptyStyle.Render(strings.Repeat("░", empty))
	
	resetIn := time.Until(m.rateLimitResetAt)
	if resetIn < 0 {
		resetIn = 0
	}
	
	content := []string{
		fmt.Sprintf("%s %s", statsLabelStyle.Render("Usage:"), 
			barStyle.Render(fmt.Sprintf("%d/%d (%.0f%%)", m.rateLimitUsed, m.rateLimitMax, usage))),
		bar,
		fmt.Sprintf("%s %s", statsLabelStyle.Render("Reset in:"), 
			statsValueStyle.Render(formatDuration(resetIn))),
	}
	
	return panelStyle.Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(content, "\n")),
	)
}

// renderLogsPanel renders the logs panel
func (m Model) renderLogsPanel(width int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	title := titleStyle.Render(" SYSTEM LOGS ")
	
	// Get recent logs
	start := len(m.logMessages) - 10
	if start < 0 {
		start = 0
	}
	
	var logs []string
	for i := start; i < len(m.logMessages); i++ {
		log := m.logMessages[i]
		timestamp := logTimestampStyle.Render(log.Time.Format("15:04:05"))
		level := lipgloss.NewStyle().Foreground(log.Color).Bold(true).Render(fmt.Sprintf("[%-7s]", log.Level))
		message := logMessageStyle.Render(log.Message)
		
		// Truncate message if too long
		maxMsgLen := width - 25
		if len(message) > maxMsgLen {
			message = message[:maxMsgLen-3] + "..."
		}
		
		logs = append(logs, fmt.Sprintf("%s %s %s", timestamp, level, message))
	}
	
	content := strings.Join(logs, "\n")
	if content == "" {
		content = lipgloss.NewStyle().Foreground(dimWhite).Render("No logs yet...")
	}
	
	// Calculate height for logs panel to fill remaining space
	logsHeight := m.height - 35 // Approximate calculation
	if logsHeight < 5 {
		logsHeight = 5
	}
	
	return panelStyle.Width(width).Height(logsHeight).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, content),
	)
}

// renderHelp renders the help panel
func (m Model) renderHelp() string {
	help := `
  Navigation:
    q/Q      - Quit the application
    p/P      - Pause/Resume downloads
    ?        - Toggle this help

  Status Indicators:
    ` + successStyle.Render("Green") + `    - Active/Healthy
    ` + warningStyle.Render("Orange") + `   - Warning/Pending
    ` + errorStyle.Render("Red") + `      - Error/Critical
    
  Icons:
    ⏳       - Pending download
    ✓        - Completed download
    ⏸        - Paused
    █        - Progress indicator
`
	
	return panelStyle.Width(m.width).Render(help)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "00:00:00"
	}
	
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}