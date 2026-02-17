package ui

import (
	"context"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/meijin/lazytest/internal/config"
	"github.com/meijin/lazytest/internal/domain"
	"github.com/meijin/lazytest/internal/parser"
	"github.com/meijin/lazytest/internal/runner"
)

// Mode represents the current UI mode.
type Mode int

const (
	ModeSearch Mode = iota
	ModeRunning
	ModeResults
)

// Messages

// testEventMsg carries a single streaming event from a test run.
// runID ties it to a specific execution so stale events from cancelled runs are ignored.
type testEventMsg struct {
	runID  uint64
	event  *parser.Event
	events <-chan *parser.Event
	errs   <-chan error
}

type testDoneMsg struct {
	runID uint64
	err   error
}

// App is the root bubbletea model.
type App struct {
	mode      Mode
	search    SearchModel
	running   RunningModel
	results   ResultsModel
	executor  *runner.Executor
	editor    string
	lastRun   *domain.TestRun
	lastFiles []string
	cancel    context.CancelFunc
	runID     uint64 // incremented on each new test execution
	width     int
	height    int
	err       error
}

func NewApp(cfg config.Config, files []domain.TestFile) App {
	return App{
		mode:     ModeSearch,
		search:   NewSearchModel(files),
		running:  NewRunningModel(),
		results:  NewResultsModel(),
		executor: runner.NewExecutor(cfg),
		editor:   cfg.Editor,
	}
}

func (a App) Init() tea.Cmd {
	return a.search.input.Focus()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case testEventMsg:
		// Ignore events from a cancelled/old run
		if msg.runID != a.runID {
			// Drain the channel in background to avoid goroutine leak
			return a, drainEvents(msg.events, msg.errs)
		}
		if a.mode == ModeRunning {
			a.running.HandleEvent(msg.event)
		}
		return a, waitForEvent(a.runID, msg.events, msg.errs)

	case testDoneMsg:
		if msg.runID != a.runID {
			return a, nil
		}
		a.err = msg.err
		run := a.running.BuildTestRun(a.lastFiles)
		a.lastRun = run
		a.results.SetRun(run)
		a.updateFileStatuses(run)
		a.mode = ModeResults
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)
	}

	// Delegate to current mode
	switch a.mode {
	case ModeSearch:
		var cmd tea.Cmd
		a.search, cmd = a.search.Update(msg)
		return a, cmd
	case ModeResults:
		var cmd tea.Cmd
		a.results, cmd = a.results.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a *App) updateFileStatuses(run *domain.TestRun) {
	statusMap := make(map[string]domain.TestStatus)
	for _, f := range a.lastFiles {
		statusMap[f] = domain.StatusPassed
	}
	if run.Failed > 0 {
		for _, f := range a.lastFiles {
			statusMap[f] = domain.StatusFailed
		}
	}
	a.search.UpdatePrevStatus(statusMap)
}

// cancelRun cancels the current test execution and bumps the runID
// so any in-flight events from the old run will be ignored.
func (a *App) cancelRun() {
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.runID++ // stale events will have the old ID and be discarded
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.mode {
	case ModeSearch:
		switch {
		case key.Matches(msg, searchKeys.Quit):
			return a, tea.Quit
		case key.Matches(msg, searchKeys.Run):
			files := a.search.FilteredFiles()
			if len(files) > 0 {
				return a, a.startTests(files)
			}
			return a, nil
		case key.Matches(msg, searchKeys.RunAll):
			files := a.search.AllFiles()
			if len(files) > 0 {
				return a, a.startTests(files)
			}
			return a, nil
		case key.Matches(msg, searchKeys.Tab):
			if a.lastRun != nil {
				a.mode = ModeResults
			}
			return a, nil
		}
		// Pass to search model for text input + navigation
		var cmd tea.Cmd
		a.search, cmd = a.search.Update(msg)
		return a, cmd

	case ModeRunning:
		switch {
		case key.Matches(msg, runningKeys.Cancel):
			// Cancel the running test and go back to search
			a.cancelRun()
			a.mode = ModeSearch
			a.search.input.Focus()
			return a, nil
		case key.Matches(msg, searchKeys.Quit):
			a.cancelRun()
			return a, tea.Quit
		}
		return a, nil

	case ModeResults:
		switch {
		case key.Matches(msg, resultsKeys.Quit):
			return a, tea.Quit
		case key.Matches(msg, resultsKeys.Enter):
			a.mode = ModeSearch
			a.search.ClearInput()
			a.search.input.Focus()
			return a, nil
		case key.Matches(msg, resultsKeys.Back):
			a.mode = ModeSearch
			a.search.input.Focus()
			return a, nil
		case key.Matches(msg, resultsKeys.Rerun):
			if len(a.lastFiles) > 0 {
				return a, a.startTests(a.lastFiles)
			}
			return a, nil
		case key.Matches(msg, resultsKeys.RerunAll):
			files := a.search.AllFiles()
			if len(files) > 0 {
				return a, a.startTests(files)
			}
			return a, nil
		case key.Matches(msg, resultsKeys.Open):
			if filePath := a.resolveSelectedFile(); filePath != "" {
				return a, openFileCmd(a.editor, filePath)
			}
			return a, nil
		}
		var cmd tea.Cmd
		a.results, cmd = a.results.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a *App) startTests(files []string) tea.Cmd {
	// Cancel any in-progress run first
	a.cancelRun()

	a.lastFiles = files
	a.running.Reset()
	a.mode = ModeRunning

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	events, errs := a.executor.Run(ctx, files)

	return waitForEvent(a.runID, events, errs)
}

// waitForEvent returns a Cmd that blocks until the next event or completion.
func waitForEvent(runID uint64, events <-chan *parser.Event, errs <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case ev, ok := <-events:
			if !ok {
				err := <-errs
				return testDoneMsg{runID: runID, err: err}
			}
			return testEventMsg{runID: runID, event: ev, events: events, errs: errs}
		case err := <-errs:
			return testDoneMsg{runID: runID, err: err}
		}
	}
}

// drainEvents consumes remaining events from a cancelled run to prevent goroutine leaks.
func drainEvents(events <-chan *parser.Event, errs <-chan error) tea.Cmd {
	return func() tea.Msg {
		for range events {
		}
		<-errs
		return nil
	}
}

// resolveSelectedFile maps the currently selected suite/test in results
// back to a file path from lastFiles.
func (a *App) resolveSelectedFile() string {
	suite := a.results.SelectedSuite()
	if suite == nil {
		return ""
	}

	// Convert PHP namespace to path: Tests\Feature\Auth\LoginTest → tests/feature/auth/logintest
	suitePath := strings.ReplaceAll(suite.Name, `\`, "/")
	suitePath = strings.ToLower(suitePath)

	for _, f := range a.lastFiles {
		normalized := strings.ToLower(f)
		withoutExt := strings.TrimSuffix(normalized, ".php")
		if strings.HasSuffix(withoutExt, suitePath) {
			return f
		}
	}

	// Try partial match on just the class name (last segment)
	parts := strings.Split(suitePath, "/")
	className := parts[len(parts)-1]
	for _, f := range a.lastFiles {
		normalized := strings.ToLower(f)
		withoutExt := strings.TrimSuffix(normalized, ".php")
		segments := strings.Split(withoutExt, "/")
		if segments[len(segments)-1] == className {
			return f
		}
	}

	// If only one file was tested, just return it
	if len(a.lastFiles) == 1 {
		return a.lastFiles[0]
	}

	return ""
}

func openFileCmd(editor, filePath string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command(editor, filePath)
		cmd.Start()
		return nil
	}
}

func (a App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	titleBar := titleStyle.Render("lazytest")
	if a.mode == ModeResults {
		titleBar = titleStyle.Render("Test Results")
	} else if a.mode == ModeRunning {
		titleBar = titleStyle.Render("Running Tests")
	}

	statusBar := renderStatusBar(a.lastRun, a.width-2)
	helpBar := renderHelpBar(a.mode, a.width-2)

	chrome := lipgloss.Height(titleBar) + lipgloss.Height(statusBar) + lipgloss.Height(helpBar) + 2
	contentHeight := a.height - chrome
	if contentHeight < 1 {
		contentHeight = 1
	}
	contentWidth := a.width - 2

	var content string
	switch a.mode {
	case ModeSearch:
		content = a.search.View(contentWidth, contentHeight)
	case ModeRunning:
		content = a.running.View(contentWidth, contentHeight)
	case ModeResults:
		content = a.results.View(contentWidth, contentHeight)
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		content,
		statusBar,
		helpBar,
	)

	return body
}
