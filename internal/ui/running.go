package ui

import (
	"fmt"
	"strings"

	"github.com/meijin/lazytest/internal/domain"
	"github.com/meijin/lazytest/internal/parser"
	"github.com/meijin/lazytest/internal/runner"
)

// targetRunState tracks the running state for a single target.
type targetRunState struct {
	suites  []*domain.TestSuite
	current *domain.TestSuite
	done    bool
	errMsg  string
}

// RunningModel displays real-time test execution progress.
type RunningModel struct {
	targetRuns  map[string]*targetRunState
	targetOrder []string // preserve insertion order
	width       int
	height      int
}

func NewRunningModel() RunningModel {
	return RunningModel{}
}

func (m *RunningModel) Reset() {
	m.targetRuns = make(map[string]*targetRunState)
	m.targetOrder = nil
}

func (m *RunningModel) HandleEvent(te *runner.TargetEvent) {
	if te == nil {
		return
	}

	// Ensure target state exists
	state, ok := m.targetRuns[te.TargetName]
	if !ok {
		state = &targetRunState{}
		m.targetRuns[te.TargetName] = state
		m.targetOrder = append(m.targetOrder, te.TargetName)
	}

	if te.Done {
		state.done = true
		if te.Error != "" {
			state.errMsg = te.Error
		}
		return
	}

	ev := te.Event
	if ev == nil {
		return
	}

	switch ev.Type {
	case parser.EventSuiteStarted:
		suite := &domain.TestSuite{
			Name:   ev.Name,
			Status: domain.StatusRunning,
		}
		state.suites = append(state.suites, suite)
		state.current = suite

	case parser.EventSuiteFinished:
		if state.current != nil && state.current.Name == ev.Name {
			state.current.Status = state.current.ComputeStatus()
			state.current = nil
		}

	case parser.EventTestStarted:
		if state.current != nil {
			tc := &domain.TestCase{
				Name:   ev.Name,
				Suite:  state.current.Name,
				Status: domain.StatusRunning,
			}
			state.current.Tests = append(state.current.Tests, tc)
		}

	case parser.EventTestFinished:
		if state.current != nil {
			for _, tc := range state.current.Tests {
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
		if state.current != nil {
			for _, tc := range state.current.Tests {
				if tc.Name == ev.Name {
					tc.Status = domain.StatusFailed
					tc.Message = ev.Message
					tc.Details = ev.Details
					break
				}
			}
		}

	case parser.EventTestIgnored:
		if state.current != nil {
			for _, tc := range state.current.Tests {
				if tc.Name == ev.Name {
					tc.Status = domain.StatusSkipped
					tc.Message = ev.Message
					break
				}
			}
		}
	}
}

// AllDone returns true when all targets have finished.
func (m *RunningModel) AllDone() bool {
	if len(m.targetRuns) == 0 {
		return false
	}
	for _, state := range m.targetRuns {
		if !state.done {
			return false
		}
	}
	return true
}

// BuildAggregatedRun creates an AggregatedRun from all target states.
func (m *RunningModel) BuildAggregatedRun(files []domain.TestFile) *domain.AggregatedRun {
	agg := &domain.AggregatedRun{}

	for _, targetName := range m.targetOrder {
		state := m.targetRuns[targetName]

		// Collect files for this target
		var targetFilePaths []string
		for _, f := range files {
			if f.TargetName == targetName {
				targetFilePaths = append(targetFilePaths, f.Path)
			}
		}

		run := &domain.TestRun{
			TargetName: targetName,
			Suites:     state.suites,
			Files:      targetFilePaths,
		}
		for _, s := range state.suites {
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
		agg.AddRun(run)
	}

	return agg
}

func (m RunningModel) View(width, height int) string {
	var lines []string

	lines = append(lines, runningStyle.Render("◉ Running tests..."))
	lines = append(lines, "")

	for _, targetName := range m.targetOrder {
		state := m.targetRuns[targetName]

		// Target header with badge
		badge := targetBadge(targetName)
		status := runningStyle.Render("running")
		if state.done {
			if state.errMsg != "" {
				status = failedStyle.Render("error")
			} else {
				status = passedStyle.Render("done")
			}
		}
		lines = append(lines, fmt.Sprintf("%s %s", badge, status))

		if state.errMsg != "" {
			// Show first few lines of the error
			errLines := strings.Split(state.errMsg, "\n")
			maxErr := 5
			if len(errLines) > maxErr {
				errLines = errLines[:maxErr]
			}
			for _, el := range errLines {
				if el != "" {
					lines = append(lines, failedStyle.Render("  "+el))
				}
			}
		}

		for _, suite := range state.suites {
			icon := statusStyle(suite.ComputeStatus().Icon()).Render(suite.ComputeStatus().Icon())
			lines = append(lines, fmt.Sprintf("  %s %s", icon, suite.Name))

			for _, tc := range suite.Tests {
				tcIcon := statusStyle(tc.Status.Icon()).Render(tc.Status.Icon())
				dur := ""
				if tc.Duration > 0 {
					dur = durationStyle.Render(fmt.Sprintf(" %dms", tc.Duration.Milliseconds()))
				}
				lines = append(lines, fmt.Sprintf("    %s %s%s", tcIcon, tc.Name, dur))
			}
		}

		lines = append(lines, "")
	}

	// Only show visible lines within height
	maxLines := height
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	for len(lines) < maxLines {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
