package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Cyberpunk color palette
	neonCyan    = lipgloss.Color("#00FFFF")
	neonMagenta = lipgloss.Color("#FF00FF")
	neonPink    = lipgloss.Color("#FF10F0")
	neonGreen   = lipgloss.Color("#39FF14")
	neonYellow  = lipgloss.Color("#FFFF00")
	neonOrange  = lipgloss.Color("#FF6700")
	darkBg      = lipgloss.Color("#0A0E27")
	darkBg2     = lipgloss.Color("#1A1E37")
	dimWhite    = lipgloss.Color("#B0B0B0")
	brightWhite = lipgloss.Color("#FFFFFF")

	// Base styles
	baseStyle = lipgloss.NewStyle().
			Background(darkBg).
			Foreground(dimWhite)

	// Logo style with gradient effect
	logoStyle = lipgloss.NewStyle().
			Foreground(neonCyan).
			Bold(true).
			Padding(1, 0).
			Align(lipgloss.Center)

	// Panel styles
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(neonMagenta).
			Background(darkBg2).
			Padding(1, 2)

	// Progress bar styles
	progressBarStyle = lipgloss.NewStyle().
				Foreground(neonGreen).
				Background(darkBg)

	progressEmptyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#333333"))

	// Stats styles
	statsLabelStyle = lipgloss.NewStyle().
			Foreground(neonCyan).
			Bold(true)

	statsValueStyle = lipgloss.NewStyle().
			Foreground(neonYellow)

	// Status styles
	successStyle = lipgloss.NewStyle().
			Foreground(neonGreen).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(neonOrange).
			Bold(true)

	// Queue item styles
	queueItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	queueItemActiveStyle = lipgloss.NewStyle().
				Foreground(neonGreen).
				Bold(true).
				PaddingLeft(2)

	queueItemCompletedStyle = lipgloss.NewStyle().
				Foreground(dimWhite).
				Faint(true).
				PaddingLeft(2)

	// Log styles
	logTimestampStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666"))

	logMessageStyle = lipgloss.NewStyle().
			Foreground(dimWhite)

	// Help style
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Padding(1, 0, 0, 2)

	// Title styles for panels
	titleStyle = lipgloss.NewStyle().
			Background(neonMagenta).
			Foreground(darkBg).
			Bold(true).
			Padding(0, 1)

	// Rate limit styles
	rateLimitNormalStyle = lipgloss.NewStyle().
				Foreground(neonGreen)

	rateLimitWarningStyle = lipgloss.NewStyle().
				Foreground(neonOrange)

	rateLimitCriticalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000"))

	// Download speed styles
	speedStyle = lipgloss.NewStyle().
			Foreground(neonCyan)

	// ASCII art style
	asciiArtStyle = lipgloss.NewStyle().
			Foreground(neonCyan).
			Bold(true)
)

// GetProgressBarStyle returns the appropriate style based on progress percentage
func GetProgressBarStyle(percentage float64) lipgloss.Style {
	switch {
	case percentage >= 80:
		return progressBarStyle.Foreground(neonGreen)
	case percentage >= 50:
		return progressBarStyle.Foreground(neonYellow)
	case percentage >= 30:
		return progressBarStyle.Foreground(neonOrange)
	default:
		return progressBarStyle.Foreground(neonMagenta)
	}
}

// GetRateLimitStyle returns the appropriate style based on rate limit usage
func GetRateLimitStyle(usage float64) lipgloss.Style {
	switch {
	case usage >= 90:
		return rateLimitCriticalStyle
	case usage >= 70:
		return rateLimitWarningStyle
	default:
		return rateLimitNormalStyle
	}
}

// GlowText creates a glowing text effect using ANSI escape codes
func GlowText(text string, color lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(color).
		Bold(true).
		Render(text)
}