package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/meijin/lazytest/internal/domain"
)

// SearchModel is the file search and selection mode.
type SearchModel struct {
	input    textinput.Model
	allFiles []domain.TestFile
	filtered []domain.TestFile
	cursor   int
	width    int
	height   int
}

func NewSearchModel(files []domain.TestFile) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter test files..."
	ti.Focus()
	ti.Prompt = "> "
	ti.PromptStyle = searchPromptStyle
	ti.CharLimit = 256

	return SearchModel{
		input:    ti,
		allFiles: files,
		filtered: files,
	}
}

func (m *SearchModel) SetFiles(files []domain.TestFile) {
	m.allFiles = files
	m.applyFilter()
}

func (m *SearchModel) UpdatePrevStatus(results map[string]domain.TestStatus) {
	for i := range m.allFiles {
		if status, ok := results[m.allFiles[i].Path]; ok {
			m.allFiles[i].PrevStatus = status
		}
	}
	m.applyFilter()
}

func (m *SearchModel) ClearInput() {
	m.input.SetValue("")
	m.applyFilter()
}

func (m *SearchModel) FilteredFiles() []string {
	paths := make([]string, len(m.filtered))
	for i, f := range m.filtered {
		paths[i] = f.Path
	}
	return paths
}

func (m *SearchModel) AllFiles() []string {
	paths := make([]string, len(m.allFiles))
	for i, f := range m.allFiles {
		paths[i] = f.Path
	}
	return paths
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, searchKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case key.Matches(msg, searchKeys.Down):
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	prevValue := m.input.Value()
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != prevValue {
		m.applyFilter()
	}
	return m, cmd
}

func (m *SearchModel) applyFilter() {
	query := strings.ToLower(m.input.Value())
	if query == "" {
		m.filtered = m.allFiles
	} else {
		m.filtered = nil
		for _, f := range m.allFiles {
			if strings.Contains(strings.ToLower(f.Path), query) {
				m.filtered = append(m.filtered, f)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m SearchModel) View(width, height int) string {
	// Header: search input + count
	countLabel := fmt.Sprintf("%d/%d files", len(m.filtered), len(m.allFiles))
	if m.input.Value() != "" && len(m.filtered) > 0 {
		countLabel = fmt.Sprintf("%d/%d files (Enter: run %d)", len(m.filtered), len(m.allFiles), len(m.filtered))
	}
	countStr := searchCountStyle.Render(countLabel)
	inputView := m.input.View()
	headerLeft := inputView
	headerRight := countStr

	headerPad := width - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight) - 2
	if headerPad < 1 {
		headerPad = 1
	}
	header := headerLeft + strings.Repeat(" ", headerPad) + headerRight

	// File list
	listHeight := height - 2 // header + separator
	if listHeight < 0 {
		listHeight = 0
	}

	// Scroll window
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}
	end := start + listHeight
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	var lines []string
	for i := start; i < end; i++ {
		f := m.filtered[i]
		style := normalItemStyle
		prefix := "  "
		if i == m.cursor {
			style = selectedItemStyle
			prefix = "▸ "
		}

		// Previous status icon
		var statusIcon string
		switch f.PrevStatus {
		case domain.StatusPassed:
			statusIcon = passedStyle.Render("✓")
		case domain.StatusFailed:
			statusIcon = failedStyle.Render("✗")
		default:
			statusIcon = pendingStyle.Render("○")
		}

		line := fmt.Sprintf("%s%s", prefix, style.Render(f.Path))
		pad := width - lipgloss.Width(line) - lipgloss.Width(statusIcon) - 2
		if pad < 1 {
			pad = 1
		}
		line = line + strings.Repeat(" ", pad) + statusIcon
		lines = append(lines, line)
	}

	// Pad remaining height
	for len(lines) < listHeight {
		lines = append(lines, "")
	}

	return header + "\n" + strings.Join(lines, "\n")
}
