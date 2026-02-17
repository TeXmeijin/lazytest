package runner

import (
	"context"
	"os/exec"
	"strings"

	"github.com/meijin/lazytest/internal/config"
	"github.com/meijin/lazytest/internal/parser"
)

// Executor manages test command execution.
type Executor struct {
	Config config.Config
}

// NewExecutor creates a new Executor with the given config.
func NewExecutor(cfg config.Config) *Executor {
	return &Executor{Config: cfg}
}

// BuildCommand constructs the full command string from the template and file list.
func (e *Executor) BuildCommand(files []string) string {
	transformed := make([]string, len(files))
	for i, f := range files {
		if e.Config.PathStripPrefix != "" {
			f = strings.TrimPrefix(f, e.Config.PathStripPrefix)
		}
		transformed[i] = f
	}

	cmd := e.Config.Command
	joined := strings.Join(transformed, " ")
	cmd = strings.ReplaceAll(cmd, "{files}", joined)

	if len(transformed) > 0 {
		cmd = strings.ReplaceAll(cmd, "{file}", transformed[0])
	}

	return cmd
}

// Run executes the test command and streams events through channels.
func (e *Executor) Run(ctx context.Context, files []string) (<-chan *parser.Event, <-chan error) {
	events := make(chan *parser.Event, 100)
	errs := make(chan error, 1)

	cmdStr := e.BuildCommand(files)
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		close(events)
		errs <- err
		close(errs)
		return events, errs
	}

	// Combine stderr into stdout so we capture all output
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		close(events)
		errs <- err
		close(errs)
		return events, errs
	}

	go func() {
		defer close(errs)
		// ParseStream closes the events channel when done
		parser.ParseStream(stdout, events)
		// Wait for command to finish - ignore exit code since test failures
		// produce non-zero exit codes which is expected
		cmd.Wait()
	}()

	return events, errs
}
