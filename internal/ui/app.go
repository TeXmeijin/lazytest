package ui

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/meijin/lazytest/internal/config"
	"github.com/meijin/lazytest/internal/domain"
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

type testEventMsg struct {
	runID  uint64
	event  *runner.TargetEvent
	events <-chan *runner.TargetEvent
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
	config    config.Config
	lastRun   *domain.AggregatedRun
	lastFiles []domain.TestFile
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
		config:   cfg,
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
			return a, drainEvents(msg.events, msg.errs)
		}
		if msg.event != nil && a.mode == ModeRunning {
			a.running.HandleEvent(msg.event)

			// Check if all targets are done
			if a.running.AllDone() {
				run := a.running.BuildAggregatedRun(a.lastFiles)
				a.lastRun = run
				a.results.SetRun(run)
				a.updateFileStatuses(run)
				a.mode = ModeResults
				return a, drainEvents(msg.events, msg.errs)
			}
		}
		return a, waitForEvent(a.runID, msg.events, msg.errs)

	case testDoneMsg:
		if msg.runID != a.runID {
			return a, nil
		}
		a.err = msg.err
		if a.mode == ModeRunning {
			run := a.running.BuildAggregatedRun(a.lastFiles)
			a.lastRun = run
			a.results.SetRun(run)
			a.updateFileStatuses(run)
			a.mode = ModeResults
		}
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

func (a *App) updateFileStatuses(run *domain.AggregatedRun) {
	statusMap := make(map[string]domain.TestStatus)

	// Default all tested files to passed
	for _, f := range a.lastFiles {
		statusMap[f.Path] = domain.StatusPassed
	}

	// Mark files as failed if their target has failures
	for _, r := range run.Runs {
		if r.Failed > 0 {
			for _, f := range a.lastFiles {
				if f.TargetName == r.TargetName {
					statusMap[f.Path] = domain.StatusFailed
				}
			}
		}
	}

	a.search.UpdatePrevStatus(statusMap)
}

// cancelRun cancels the current test execution and bumps the runID.
func (a *App) cancelRun() {
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.runID++
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.mode {
	case ModeSearch:
		switch {
		case key.Matches(msg, searchKeys.Quit):
			return a, tea.Quit
		case key.Matches(msg, searchKeys.Run):
			files := a.search.SelectedFiles()
			if len(files) > 0 {
				return a, a.startTests(files)
			}
			return a, nil
		}
		var cmd tea.Cmd
		a.search, cmd = a.search.Update(msg)
		return a, cmd

	case ModeRunning:
		switch {
		case key.Matches(msg, runningKeys.Cancel):
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
				return a, openFileCmd(filePath)
			}
			return a, nil
		}
		var cmd tea.Cmd
		a.results, cmd = a.results.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a *App) startTests(files []domain.TestFile) tea.Cmd {
	a.cancelRun()

	a.lastFiles = files
	a.running.Reset(files)
	a.mode = ModeRunning

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	events, errs := a.executor.Run(ctx, files)

	return waitForEvent(a.runID, events, errs)
}

func waitForEvent(runID uint64, events <-chan *runner.TargetEvent, errs <-chan error) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-events
		if !ok {
			err := <-errs
			return testDoneMsg{runID: runID, err: err}
		}
		return testEventMsg{runID: runID, event: ev, events: events, errs: errs}
	}
}

func drainEvents(events <-chan *runner.TargetEvent, errs <-chan error) tea.Cmd {
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
	item := a.results.SelectedItem()
	if item == nil {
		return ""
	}

	targetName := item.targetName

	// Collect files for this target
	var targetFiles []domain.TestFile
	for _, f := range a.lastFiles {
		if f.TargetName == targetName {
			targetFiles = append(targetFiles, f)
		}
	}

	if len(targetFiles) == 0 {
		return ""
	}

	// If only one file was tested for this target, always return it
	if len(targetFiles) == 1 {
		return targetFiles[0].Path
	}

	suite := a.results.SelectedSuite()
	if suite == nil {
		return targetFiles[0].Path
	}

	suiteName := strings.ReplaceAll(suite.Name, `\`, "/")
	suiteNameLower := strings.ToLower(suiteName)

	// Strategy 1: File path contains suite name (handles working_dir prefix)
	for _, f := range targetFiles {
		if strings.Contains(strings.ToLower(f.Path), suiteNameLower) {
			return f.Path
		}
	}

	// Strategy 2: Extension-insensitive suffix match
	suiteNoExt := stripExtensions(suiteNameLower)
	for _, f := range targetFiles {
		fNoExt := stripExtensions(strings.ToLower(f.Path))
		if strings.HasSuffix(fNoExt, suiteNoExt) {
			return f.Path
		}
	}

	// Strategy 3: Partial match on last segment
	parts := strings.Split(suiteNameLower, "/")
	className := stripExtensions(parts[len(parts)-1])
	for _, f := range targetFiles {
		fLower := strings.ToLower(f.Path)
		segments := strings.Split(fLower, "/")
		lastSeg := stripExtensions(segments[len(segments)-1])
		if lastSeg == className {
			return f.Path
		}
	}

	return targetFiles[0].Path
}

// stripExtensions removes file extensions (e.g. ".test.php" → "", "ExampleTest.php" → "ExampleTest").
func stripExtensions(name string) string {
	for {
		ext := filepath.Ext(name)
		if ext == "" {
			return name
		}
		name = strings.TrimSuffix(name, ext)
	}
}

func openFileCmd(filePath string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", filePath)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", "", filePath)
		default:
			cmd = exec.Command("xdg-open", filePath)
		}
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
