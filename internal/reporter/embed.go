package reporter

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed vitest-reporter.mjs
var vitestReporterJS []byte

//go:embed jest-reporter.js
var jestReporterJS []byte

// EnsureVitestReporter writes the embedded Vitest reporter to a temp file
// and returns its path. The file is overwritten on each call to ensure
// it matches the current binary version.
func EnsureVitestReporter() (string, error) {
	path := filepath.Join(os.TempDir(), "lazytest-vitest-reporter.mjs")
	return path, os.WriteFile(path, vitestReporterJS, 0644)
}

// EnsureJestReporter writes the embedded Jest reporter to a temp file
// and returns its path. The file is overwritten on each call to ensure
// it matches the current binary version.
func EnsureJestReporter() (string, error) {
	path := filepath.Join(os.TempDir(), "lazytest-jest-reporter.js")
	return path, os.WriteFile(path, jestReporterJS, 0644)
}
