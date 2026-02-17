package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorPrimary   = lipgloss.Color("#7C3AED") // purple
	colorSuccess   = lipgloss.Color("#22C55E") // green
	colorDanger    = lipgloss.Color("#EF4444") // red
	colorWarning   = lipgloss.Color("#F59E0B") // yellow
	colorMuted     = lipgloss.Color("#6B7280") // gray
	colorHighlight = lipgloss.Color("#E0E7FF") // light purple
	colorBg        = lipgloss.Color("#1E1E2E") // dark bg
	colorBorder    = lipgloss.Color("#4B5563") // border gray

	// Target badge colors
	colorPHPUnit = lipgloss.Color("#4F5B93") // blue-ish (PHP)
	colorVitest  = lipgloss.Color("#729B1B") // green-ish (Vitest)

	// Box styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	activeBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary)

	// Title styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	// Test status icons
	passedStyle = lipgloss.NewStyle().Foreground(colorSuccess)
	failedStyle = lipgloss.NewStyle().Foreground(colorDanger)
	pendingStyle = lipgloss.NewStyle().Foreground(colorMuted)
	runningStyle = lipgloss.NewStyle().Foreground(colorWarning)
	skippedStyle = lipgloss.NewStyle().Foreground(colorWarning)

	// List items
	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorHighlight).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C9D1D9")) // light gray, readable on dark bg

	suiteNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB")).
			Bold(true)

	testNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C9D1D9"))

	// Detail pane
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorDanger).
				MarginBottom(1)

	detailBodyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	// Search input
	searchPromptStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	searchCountStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	// Help bar
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Duration
	durationStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Target badge styles
	phpunitBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(colorPHPUnit).
				Bold(true).
				Padding(0, 1)

	vitestBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(colorVitest).
				Bold(true).
				Padding(0, 1)
)

func statusStyle(icon string) lipgloss.Style {
	switch icon {
	case "✓":
		return passedStyle
	case "✗":
		return failedStyle
	case "○":
		return pendingStyle
	case "◉":
		return runningStyle
	case "⊘":
		return skippedStyle
	default:
		return pendingStyle
	}
}

// targetBadge returns a styled badge string for the given target name.
func targetBadge(name string) string {
	switch name {
	case "phpunit":
		return phpunitBadgeStyle.Render("PHP")
	case "vitest":
		return vitestBadgeStyle.Render("VT")
	default:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorMuted).
			Bold(true).
			Padding(0, 1).
			Render(name)
	}
}
