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
	filtered []matchedFile
	selected map[string]bool // fileKey → true
	cursor   int
	width    int
	height   int
}

// fileKey returns a unique key for a TestFile.
func fileKey(f domain.TestFile) string {
	return f.TargetName + "\x00" + f.Path
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
		filtered: filterFuzzy(files, ""),
		selected: make(map[string]bool),
	}
}

func (m *SearchModel) SetFiles(files []domain.TestFile) {
	m.allFiles = files
	m.selected = make(map[string]bool)
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
	m.selected = make(map[string]bool)
	m.applyFilter()
}

// SelectedFiles returns the files to run.
// If any files are toggled, returns all toggled files (in original order).
// Otherwise returns just the cursor file.
func (m *SearchModel) SelectedFiles() []domain.TestFile {
	if len(m.selected) > 0 {
		var files []domain.TestFile
		for _, f := range m.allFiles {
			if m.selected[fileKey(f)] {
				files = append(files, f)
			}
		}
		return files
	}
	if len(m.filtered) > 0 {
		return []domain.TestFile{m.filtered[m.cursor].file}
	}
	return nil
}

// SelectedCount returns the number of toggled files.
func (m *SearchModel) SelectedCount() int {
	return len(m.selected)
}

// FilteredFiles returns the currently filtered files as TestFile slice.
func (m *SearchModel) FilteredFiles() []domain.TestFile {
	result := make([]domain.TestFile, len(m.filtered))
	for i, mf := range m.filtered {
		result[i] = mf.file
	}
	return result
}

// AllFiles returns all files as TestFile slice.
func (m *SearchModel) AllFiles() []domain.TestFile {
	result := make([]domain.TestFile, len(m.allFiles))
	copy(result, m.allFiles)
	return result
}

// ToggleSelection toggles the cursor file and moves cursor down.
func (m *SearchModel) ToggleSelection() {
	if len(m.filtered) == 0 {
		return
	}
	f := m.filtered[m.cursor].file
	k := fileKey(f)
	if m.selected[k] {
		delete(m.selected, k)
	} else {
		m.selected[k] = true
	}
	if m.cursor < len(m.filtered)-1 {
		m.cursor++
	}
}

// ToggleAll selects all filtered files, or deselects all if already all selected.
func (m *SearchModel) ToggleAll() {
	allSelected := true
	for _, mf := range m.filtered {
		if !m.selected[fileKey(mf.file)] {
			allSelected = false
			break
		}
	}
	if allSelected {
		for _, mf := range m.filtered {
			delete(m.selected, fileKey(mf.file))
		}
	} else {
		for _, mf := range m.filtered {
			m.selected[fileKey(mf.file)] = true
		}
	}
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
		case key.Matches(msg, searchKeys.Toggle):
			m.ToggleSelection()
			return m, nil
		case key.Matches(msg, searchKeys.SelectAll):
			m.ToggleAll()
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
	m.filtered = filterFuzzy(m.allFiles, m.input.Value())
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m SearchModel) View(width, height int) string {
	// Header: search input + count
	selCount := len(m.selected)
	countLabel := fmt.Sprintf("%d/%d files", len(m.filtered), len(m.allFiles))
	if selCount > 0 {
		countLabel = fmt.Sprintf("%d/%d files (%d selected)", len(m.filtered), len(m.allFiles), selCount)
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
		mf := m.filtered[i]
		f := mf.file
		isSelected := m.selected[fileKey(f)]
		style := normalItemStyle
		hlStyle := matchHighlightStyle
		prefix := "  "
		if i == m.cursor {
			style = selectedItemStyle
			hlStyle = selectedMatchHighlightStyle
			prefix = "▸ "
		}

		// Selection marker
		marker := " "
		if isSelected {
			marker = selectedMarkerStyle.Render("◆")
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

		// Target badge
		badge := targetBadge(f.TargetName)

		renderedPath := renderWithHighlight(f.Path, mf.indices, style, hlStyle)
		line := fmt.Sprintf("%s%s%s %s", prefix, marker, badge, renderedPath)
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
