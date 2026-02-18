package runner

import (
	"context"
	"os/exec"
	"strings"
	"sync"

	"github.com/meijin/lazytest/internal/config"
	"github.com/meijin/lazytest/internal/domain"
	"github.com/meijin/lazytest/internal/parser"
	"github.com/meijin/lazytest/internal/reporter"
)

// TargetEvent wraps an event with its source target name.
type TargetEvent struct {
	TargetName string
	Event      *parser.Event
	Done       bool   // true when this target has finished
	Error      string // non-empty if the command failed with no test output
}

// Executor manages test command execution across multiple targets.
type Executor struct {
	Targets      map[string]config.Target
	reporterPath string
}

// NewExecutor creates a new Executor with the given config.
func NewExecutor(cfg config.Config) *Executor {
	targets := make(map[string]config.Target)
	for _, t := range cfg.Targets {
		targets[t.Name] = t
	}
	reporterPath, _ := reporter.EnsureVitestReporter()
	return &Executor{Targets: targets, reporterPath: reporterPath}
}

// BuildCommand constructs the full command string for a specific target.
func (e *Executor) BuildCommand(targetName string, files []string) string {
	target, ok := e.Targets[targetName]
	if !ok {
		return ""
	}

	transformed := make([]string, len(files))
	for i, f := range files {
		if target.PathStripPrefix != "" {
			f = strings.TrimPrefix(f, target.PathStripPrefix)
		}
		// When working_dir is set, strip it from file paths so they're relative to working_dir
		if target.WorkingDir != "" {
			f = strings.TrimPrefix(f, target.WorkingDir)
		}
		transformed[i] = f
	}

	cmd := target.Command
	joined := strings.Join(transformed, " ")
	cmd = strings.ReplaceAll(cmd, "{files}", joined)

	if len(transformed) > 0 {
		cmd = strings.ReplaceAll(cmd, "{file}", transformed[0])
	}

	cmd = strings.ReplaceAll(cmd, "{reporter}", e.reporterPath)

	return cmd
}

// Run executes test commands for all relevant targets in parallel.
// Files are grouped by TargetName and each group runs in its own goroutine.
func (e *Executor) Run(ctx context.Context, files []domain.TestFile) (<-chan *TargetEvent, <-chan error) {
	events := make(chan *TargetEvent, 100)
	errs := make(chan error, 1)

	// Group files by target name
	grouped := make(map[string][]string)
	for _, f := range files {
		grouped[f.TargetName] = append(grouped[f.TargetName], f.Path)
	}

	var wg sync.WaitGroup

	for targetName, targetFiles := range grouped {
		target, ok := e.Targets[targetName]
		if !ok {
			continue
		}

		wg.Add(1)
		go func(tName string, tTarget config.Target, tFiles []string) {
			defer wg.Done()
			e.runTarget(ctx, tName, tTarget, tFiles, events)
		}(targetName, target, targetFiles)
	}

	go func() {
		wg.Wait()
		close(events)
		close(errs)
	}()

	return events, errs
}

// runTarget executes a single target's test command and sends events to the shared channel.
func (e *Executor) runTarget(ctx context.Context, targetName string, target config.Target, files []string, out chan<- *TargetEvent) {
	cmdStr := e.BuildCommand(targetName, files)
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)

	if target.WorkingDir != "" {
		cmd.Dir = target.WorkingDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		out <- &TargetEvent{TargetName: targetName, Done: true, Error: err.Error()}
		return
	}

	// Capture stderr separately so we can report errors
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		out <- &TargetEvent{TargetName: targetName, Done: true, Error: err.Error()}
		return
	}

	targetEvents := make(chan *parser.Event, 100)
	hasStructuredOutput := false
	go func() {
		parser.ParseStream(stdout, targetEvents)
	}()

	for ev := range targetEvents {
		if ev.Type != parser.EventOutput {
			hasStructuredOutput = true
		}
		out <- &TargetEvent{
			TargetName: targetName,
			Event:      ev,
		}
	}

	// Wait for command to finish - ignore exit code since test failures
	// produce non-zero exit codes which is expected
	waitErr := cmd.Wait()

	doneEvent := &TargetEvent{TargetName: targetName, Done: true}
	// If no TeamCity output was produced and the command failed, report the error
	if !hasStructuredOutput && waitErr != nil {
		errMsg := stderrBuf.String()
		if errMsg == "" {
			errMsg = waitErr.Error()
		}
		doneEvent.Error = errMsg
	}
	out <- doneEvent
}
