package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/meijin/lazytest/internal/domain"
)

// ResultsModel displays test results in a split view.
type ResultsModel struct {
	run         *domain.AggregatedRun
	flatList    []*resultItem
	cursor      int
	focusDetail bool
	filterFails bool
	scrollY     int // detail scroll offset
	width       int
	height      int
}

type resultItem struct {
	suite      *domain.TestSuite
	test       *domain.TestCase
	targetName string
	depth      int // 0 = target header, 1 = suite, 2 = test
}

func NewResultsModel() ResultsModel {
	return ResultsModel{}
}

func (m *ResultsModel) SetRun(run *domain.AggregatedRun) {
	m.run = run
	m.cursor = 0
	m.focusDetail = false
	m.filterFails = false
	m.scrollY = 0
	m.buildFlatList()
}

func (m *ResultsModel) buildFlatList() {
	m.flatList = nil
	if m.run == nil {
		return
	}
	for _, run := range m.run.Runs {
		// Add target header
		m.flatList = append(m.flatList, &resultItem{targetName: run.TargetName, depth: 0})

		for _, suite := range run.Suites {
			if m.filterFails && suite.ComputeStatus() != domain.StatusFailed {
				continue
			}
			m.flatList = append(m.flatList, &resultItem{suite: suite, targetName: run.TargetName, depth: 1})
			for _, tc := range suite.Tests {
				if m.filterFails && tc.Status != domain.StatusFailed {
					continue
				}
				m.flatList = append(m.flatList, &resultItem{test: tc, targetName: run.TargetName, depth: 2})
			}
		}
	}
}

func (m *ResultsModel) SelectedTest() *domain.TestCase {
	if m.cursor >= 0 && m.cursor < len(m.flatList) {
		return m.flatList[m.cursor].test
	}
	return nil
}

func (m *ResultsModel) SelectedItem() *resultItem {
	if m.cursor >= 0 && m.cursor < len(m.flatList) {
		return m.flatList[m.cursor]
	}
	return nil
}

func (m *ResultsModel) SelectedSuite() *domain.TestSuite {
	if m.cursor >= 0 && m.cursor < len(m.flatList) {
		item := m.flatList[m.cursor]
		if item.suite != nil {
			return item.suite
		}
		for i := m.cursor - 1; i >= 0; i-- {
			if m.flatList[i].suite != nil {
				return m.flatList[i].suite
			}
		}
	}
	return nil
}

func (m ResultsModel) Update(msg tea.Msg) (ResultsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, resultsKeys.Up):
			if m.focusDetail {
				if m.scrollY > 0 {
					m.scrollY--
				}
			} else if m.cursor > 0 {
				m.cursor--
				m.scrollY = 0
			}
		case key.Matches(msg, resultsKeys.Down):
			if m.focusDetail {
				m.scrollY++
			} else if m.cursor < len(m.flatList)-1 {
				m.cursor++
				m.scrollY = 0
			}
		case key.Matches(msg, resultsKeys.Right):
			m.focusDetail = true
			m.scrollY = 0
		case key.Matches(msg, resultsKeys.Left):
			if m.focusDetail {
				m.focusDetail = false
			}
		case key.Matches(msg, resultsKeys.Filter):
			m.filterFails = !m.filterFails
			m.cursor = 0
			m.scrollY = 0
			m.focusDetail = false
			m.buildFlatList()
		}
	}
	return m, nil
}

func (m ResultsModel) View(width, height int) string {
	if m.run == nil {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center,
			pendingStyle.Render("No results to display"))
	}

	// Split: left 45%, right 55%
	leftWidth := width*45/100 - 2
	rightWidth := width - leftWidth - 3
	if leftWidth < 20 {
		leftWidth = 20
	}
	if rightWidth < 20 {
		rightWidth = 20
	}

	contentHeight := height

	leftView := m.renderTreeView(leftWidth, contentHeight)
	rightView := m.renderDetailView(rightWidth, contentHeight)

	leftBox := activeBoxStyle
	rightBox := boxStyle
	if m.focusDetail {
		leftBox = boxStyle
		rightBox = activeBoxStyle
	}

	left := leftBox.Width(leftWidth).Height(contentHeight).Render(leftView)
	right := rightBox.Width(rightWidth).Height(contentHeight).Render(rightView)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m ResultsModel) renderTreeView(width, height int) string {
	var lines []string

	// Scroll window
	start := 0
	if m.cursor >= height {
		start = m.cursor - height + 1
	}
	end := start + height
	if end > len(m.flatList) {
		end = len(m.flatList)
	}

	for i := start; i < end; i++ {
		item := m.flatList[i]
		selected := i == m.cursor && !m.focusDetail

		var line string
		switch {
		case item.depth == 0 && item.suite == nil && item.test == nil:
			// Target header
			badge := targetBadge(item.targetName)
			if selected {
				line = fmt.Sprintf("%s %s", badge, selectedItemStyle.Render(item.targetName))
			} else {
				line = fmt.Sprintf("%s %s", badge, suiteNameStyle.Render(item.targetName))
			}

		case item.suite != nil:
			icon := statusStyle(item.suite.ComputeStatus().Icon()).Render(item.suite.ComputeStatus().Icon())
			name := shortSuiteName(item.suite.Name)
			if lipgloss.Width(name) > width-6 {
				name = "..." + name[len(name)-width+7:]
			}
			if selected {
				line = fmt.Sprintf("  %s %s", icon, selectedItemStyle.Render(name))
			} else {
				line = fmt.Sprintf("  %s %s", icon, suiteNameStyle.Render(name))
			}

		case item.test != nil:
			icon := statusStyle(item.test.Status.Icon()).Render(item.test.Status.Icon())
			dur := ""
			if item.test.Duration > 0 {
				dur = durationStyle.Render(fmt.Sprintf(" %dms", item.test.Duration.Milliseconds()))
			}
			name := item.test.Name
			maxName := width - 10 - lipgloss.Width(dur)
			if maxName > 0 && len(name) > maxName {
				name = name[:maxName-1] + "..."
			}
			if selected {
				line = fmt.Sprintf("    %s %s%s", icon, selectedItemStyle.Render(name), dur)
			} else {
				line = fmt.Sprintf("    %s %s%s", icon, testNameStyle.Render(name), dur)
			}
		}

		lines = append(lines, ansi.Truncate(line, width, ""))
	}

	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// shortSuiteName extracts the last segment of a namespace for display.
func shortSuiteName(name string) string {
	// Try backslash (PHP namespace)
	parts := strings.Split(name, `\`)
	if len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], `\`)
	}
	// Try forward slash (path)
	parts = strings.Split(name, "/")
	if len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return name
}

func (m ResultsModel) renderDetailView(width, height int) string {
	if m.cursor < 0 || m.cursor >= len(m.flatList) {
		return ""
	}

	item := m.flatList[m.cursor]
	var lines []string

	switch {
	case item.depth == 0 && item.suite == nil && item.test == nil:
		// Target header - show target summary
		badge := targetBadge(item.targetName)
		lines = append(lines, badge+" "+titleStyle.Render(item.targetName))
		lines = append(lines, "")

		// Find the run for this target
		for _, run := range m.run.Runs {
			if run.TargetName == item.targetName {
				total := run.Passed + run.Failed + run.Skipped
				lines = append(lines, normalItemStyle.Render(fmt.Sprintf("  Tests: %d", total)))
				lines = append(lines, fmt.Sprintf("  %s %d passed", passedStyle.Render("✓"), run.Passed))
				if run.Failed > 0 {
					lines = append(lines, fmt.Sprintf("  %s %d failed", failedStyle.Render("✗"), run.Failed))
				}
				if run.Skipped > 0 {
					lines = append(lines, fmt.Sprintf("  %s %d skipped", skippedStyle.Render("⊘"), run.Skipped))
				}
				lines = append(lines, "")
				lines = append(lines, durationStyle.Render(fmt.Sprintf("  Duration: %dms", run.Duration.Milliseconds())))
				break
			}
		}

	case item.suite != nil:
		lines = append(lines, titleStyle.Render(shortSuiteName(item.suite.Name)))
		lines = append(lines, "")

		passed, failed, skipped := 0, 0, 0
		for _, tc := range item.suite.Tests {
			switch tc.Status {
			case domain.StatusPassed:
				passed++
			case domain.StatusFailed:
				failed++
			case domain.StatusSkipped:
				skipped++
			}
		}
		total := passed + failed + skipped
		lines = append(lines, normalItemStyle.Render(fmt.Sprintf("  Tests: %d", total)))
		lines = append(lines, fmt.Sprintf("  %s %d passed", passedStyle.Render("✓"), passed))
		if failed > 0 {
			lines = append(lines, fmt.Sprintf("  %s %d failed", failedStyle.Render("✗"), failed))
		}
		if skipped > 0 {
			lines = append(lines, fmt.Sprintf("  %s %d skipped", skippedStyle.Render("⊘"), skipped))
		}
		if item.suite.Duration > 0 {
			lines = append(lines, "")
			lines = append(lines, durationStyle.Render(fmt.Sprintf("  Duration: %dms", item.suite.Duration.Milliseconds())))
		}

		var failedTests []*domain.TestCase
		for _, tc := range item.suite.Tests {
			if tc.Status == domain.StatusFailed {
				failedTests = append(failedTests, tc)
			}
		}
		if len(failedTests) > 0 {
			lines = append(lines, "")
			lines = append(lines, failedStyle.Render("  Failed:"))
			for _, tc := range failedTests {
				lines = append(lines, failedStyle.Render("    ✗ ")+testNameStyle.Render(tc.Name))
				if tc.Message != "" {
					lines = append(lines, detailBodyStyle.Render("      "+tc.Message))
				}
			}
		}

	case item.test != nil:
		tc := item.test

		var headerStyle lipgloss.Style
		switch tc.Status {
		case domain.StatusPassed:
			headerStyle = passedStyle
		case domain.StatusFailed:
			headerStyle = failedStyle
		case domain.StatusSkipped:
			headerStyle = skippedStyle
		default:
			headerStyle = normalItemStyle
		}
		lines = append(lines, headerStyle.Render(tc.Status.Icon()+" "+tc.Name))

		if tc.Duration > 0 {
			lines = append(lines, durationStyle.Render(fmt.Sprintf("  %dms", tc.Duration.Milliseconds())))
		}
		lines = append(lines, "")

		switch tc.Status {
		case domain.StatusFailed:
			if tc.Message != "" {
				lines = append(lines, failedStyle.Render("  Message:"))
				for _, ml := range strings.Split(tc.Message, "\n") {
					lines = append(lines, detailBodyStyle.Render("  "+ml))
				}
				lines = append(lines, "")
			}
			if tc.Details != "" {
				lines = append(lines, normalItemStyle.Render("  Stack trace:"))
				for _, dl := range strings.Split(tc.Details, "\n") {
					lines = append(lines, detailBodyStyle.Render("  "+dl))
				}
			}
		case domain.StatusPassed:
			lines = append(lines, passedStyle.Render("  Test passed"))
		case domain.StatusSkipped:
			lines = append(lines, skippedStyle.Render("  Test skipped"))
			if tc.Message != "" {
				lines = append(lines, detailBodyStyle.Render("  "+tc.Message))
			}
		}
	}

	// Truncate lines to fit within width to prevent lipgloss word-wrapping
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], width, "")
	}

	// Apply scroll
	if m.scrollY > 0 && m.scrollY < len(lines) {
		lines = lines[m.scrollY:]
	}

	if len(lines) > height {
		lines = lines[:height]
	}

	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
