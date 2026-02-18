package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/meijin/lazytest/internal/domain"
)

func renderStatusBar(run *domain.AggregatedRun, width int) string {
	if run == nil {
		return statusBarStyle.Width(width).Render("No test results yet")
	}

	stats := fmt.Sprintf(
		"%s %d  %s %d  %s %d  ⏱ %s",
		passedStyle.Render("✓"),
		run.Passed,
		failedStyle.Render("✗"),
		run.Failed,
		skippedStyle.Render("⊘"),
		run.Skipped,
		run.Duration.Round(100*1e6), // round to 100ms
	)

	return statusBarStyle.Width(width).Render(stats)
}

func renderHelpBar(mode Mode, width int) string {
	var items []string

	switch mode {
	case ModeSearch:
		items = []string{
			helpKeyStyle.Render("[Tab]") + " " + helpDescStyle.Render("select"),
			helpKeyStyle.Render("[Ctrl+A]") + " " + helpDescStyle.Render("select all"),
			helpKeyStyle.Render("[Enter]") + " " + helpDescStyle.Render("run"),
			helpKeyStyle.Render("[Ctrl+C]") + " " + helpDescStyle.Render("quit"),
		}
	case ModeRunning:
		items = []string{
			helpDescStyle.Render("Running tests..."),
			helpKeyStyle.Render("[Esc]") + " " + helpDescStyle.Render("cancel"),
			helpKeyStyle.Render("[Ctrl+C]") + " " + helpDescStyle.Render("quit"),
		}
	case ModeResults:
		items = []string{
			helpKeyStyle.Render("[Enter]") + " " + helpDescStyle.Render("search"),
			helpKeyStyle.Render("[o]") + " " + helpDescStyle.Render("open"),
			helpKeyStyle.Render("[r]") + " " + helpDescStyle.Render("rerun"),
			helpKeyStyle.Render("[R]") + " " + helpDescStyle.Render("rerun all"),
			helpKeyStyle.Render("[f]") + " " + helpDescStyle.Render("fails"),
			helpKeyStyle.Render("[l]") + " " + helpDescStyle.Render("detail"),
			helpKeyStyle.Render("[q]") + " " + helpDescStyle.Render("quit"),
		}
	}

	line := lipgloss.JoinHorizontal(lipgloss.Left, joinWithSep(items, "  ")...)
	return statusBarStyle.Width(width).Render(line)
}

func joinWithSep(items []string, sep string) []string {
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items)*2-1)
	for i, item := range items {
		if i > 0 {
			result = append(result, sep)
		}
		result = append(result, item)
	}
	return result
}
