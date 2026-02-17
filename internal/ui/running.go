package ui

import (
	"fmt"
	"strings"

	"github.com/meijin/lazytest/internal/domain"
	"github.com/meijin/lazytest/internal/parser"
)

// RunningModel displays real-time test execution progress.
type RunningModel struct {
	suites  []*domain.TestSuite
	current *domain.TestSuite
	events  []*parser.Event
	width   int
	height  int
}

func NewRunningModel() RunningModel {
	return RunningModel{}
}

func (m *RunningModel) Reset() {
	m.suites = nil
	m.current = nil
	m.events = nil
}

func (m *RunningModel) HandleEvent(ev *parser.Event) {
	m.events = append(m.events, ev)

	switch ev.Type {
	case parser.EventSuiteStarted:
		suite := &domain.TestSuite{
			Name:   ev.Name,
			Status: domain.StatusRunning,
		}
		m.suites = append(m.suites, suite)
		m.current = suite

	case parser.EventSuiteFinished:
		if m.current != nil && m.current.Name == ev.Name {
			m.current.Status = m.current.ComputeStatus()
			m.current = nil
		}

	case parser.EventTestStarted:
		if m.current != nil {
			tc := &domain.TestCase{
				Name:   ev.Name,
				Suite:  m.current.Name,
				Status: domain.StatusRunning,
			}
			m.current.Tests = append(m.current.Tests, tc)
		}

	case parser.EventTestFinished:
		if m.current != nil {
			for _, tc := range m.current.Tests {
				if tc.Name == ev.Name {
					if tc.Status == domain.StatusRunning {
						tc.Status = domain.StatusPassed
					}
					tc.Duration = ev.Duration
					break
				}
			}
		}

	case parser.EventTestFailed:
		if m.current != nil {
			for _, tc := range m.current.Tests {
				if tc.Name == ev.Name {
					tc.Status = domain.StatusFailed
					tc.Message = ev.Message
					tc.Details = ev.Details
					break
				}
			}
		}

	case parser.EventTestIgnored:
		if m.current != nil {
			for _, tc := range m.current.Tests {
				if tc.Name == ev.Name {
					tc.Status = domain.StatusSkipped
					tc.Message = ev.Message
					break
				}
			}
		}
	}
}

func (m *RunningModel) BuildTestRun(files []string) *domain.TestRun {
	run := &domain.TestRun{
		Suites: m.suites,
		Files:  files,
	}
	for _, s := range m.suites {
		for _, tc := range s.Tests {
			run.Duration += tc.Duration
			switch tc.Status {
			case domain.StatusPassed:
				run.Passed++
			case domain.StatusFailed:
				run.Failed++
			case domain.StatusSkipped:
				run.Skipped++
			}
		}
	}
	return run
}

func (m RunningModel) View(width, height int) string {
	var lines []string

	lines = append(lines, runningStyle.Render("◉ Running tests..."))
	lines = append(lines, "")

	for _, suite := range m.suites {
		icon := statusStyle(suite.ComputeStatus().Icon()).Render(suite.ComputeStatus().Icon())
		lines = append(lines, fmt.Sprintf("%s %s", icon, suite.Name))

		for _, tc := range suite.Tests {
			tcIcon := statusStyle(tc.Status.Icon()).Render(tc.Status.Icon())
			dur := ""
			if tc.Duration > 0 {
				dur = durationStyle.Render(fmt.Sprintf(" %dms", tc.Duration.Milliseconds()))
			}
			lines = append(lines, fmt.Sprintf("  %s %s%s", tcIcon, tc.Name, dur))
		}
	}

	// Only show visible lines within height
	maxLines := height
	if len(lines) > maxLines {
		// Show the last maxLines (auto-scroll to bottom)
		lines = lines[len(lines)-maxLines:]
	}

	// Pad remaining
	for len(lines) < maxLines {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
