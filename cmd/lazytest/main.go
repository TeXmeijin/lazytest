package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/meijin/lazytest/internal/config"
	"github.com/meijin/lazytest/internal/discovery"
	"github.com/meijin/lazytest/internal/domain"
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

	paths, err := discovery.ScanFiles(cfg.TestDirs, cfg.FilePattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning test files: %v\n", err)
		os.Exit(1)
	}

	if len(paths) == 0 {
		fmt.Fprintf(os.Stderr, "No test files found in %v matching %q\n", cfg.TestDirs, cfg.FilePattern)
		os.Exit(1)
	}

	files := make([]domain.TestFile, len(paths))
	for i, p := range paths {
		files[i] = domain.TestFile{Path: p}
	}

	app := ui.NewApp(cfg, files)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
