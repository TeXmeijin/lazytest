package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meijin/lazytest/internal/config"
	"github.com/meijin/lazytest/internal/discovery"
	"github.com/meijin/lazytest/internal/ui"
)

func main() {
	configPath := flag.String("config", "", "path to .lazytest.yml config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	files, err := discovery.ScanAllTargets(cfg.Targets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning test files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No test files found for configured targets\n")
		os.Exit(1)
	}

	app := ui.NewApp(cfg, files)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
