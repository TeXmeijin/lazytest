package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const ConfigFileName = ".lazytest.yml"

// Target represents a single test framework target in a monorepo.
type Target struct {
	Name            string   `yaml:"name"`
	Command         string   `yaml:"command"`
	TestDirs        []string `yaml:"test_dirs"`
	FilePattern     string   `yaml:"file_pattern"`
	PathStripPrefix string   `yaml:"path_strip_prefix"`
	WorkingDir      string   `yaml:"working_dir"`
}

// Config represents the lazytest configuration.
type Config struct {
	Targets []Target `yaml:"targets"`
}

// Load reads configuration from the given path or auto-detects.
// If configPath is empty, it looks for .lazytest.yml in the current directory.
// If .lazytest.yml is not found, it falls back to framework auto-detection.
func Load(configPath string) (Config, error) {
	if configPath == "" {
		configPath = ConfigFileName
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Fall back to framework auto-detection
			targets, detectErr := DetectFrameworks(".")
			if detectErr != nil {
				return Config{}, detectErr
			}
			cfg := Config{Targets: targets}
			cfg.applyDefaults()
			return cfg, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	for i := range cfg.Targets {
		cfg.Targets[i].applyDefaults()
	}
	cfg.applyDefaults()
	return cfg, nil
}

// applyDefaults fills in missing Config-level fields.
func (c *Config) applyDefaults() {
}

// applyDefaults fills in missing Target fields based on the target name.
func (t *Target) applyDefaults() {
	switch t.Name {
	case "vitest":
		if t.FilePattern == "" {
			t.FilePattern = "*.test.ts,*.test.tsx"
		}
		if t.Command == "" {
			t.Command = "npx vitest run --reporter={reporter} {files}"
		}
		if len(t.TestDirs) == 0 {
			t.TestDirs = []string{"src/"}
		}
	case "jest":
		if t.FilePattern == "" {
			t.FilePattern = "*.test.ts,*.test.tsx,*.test.js,*.test.jsx"
		}
		if t.Command == "" {
			t.Command = "npx jest --reporters={reporter} {files}"
		}
		if len(t.TestDirs) == 0 {
			t.TestDirs = []string{"src/"}
		}
	default: // phpunit and others
		if t.FilePattern == "" {
			t.FilePattern = "*Test.php"
		}
		if t.Command == "" {
			t.Command = "./vendor/bin/phpunit --teamcity {files}"
		}
		if len(t.TestDirs) == 0 {
			t.TestDirs = []string{"tests/"}
		}
	}
}

// FindProjectRoot walks up from the given directory looking for config files.
func FindProjectRoot(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	for {
		// Check for .lazytest.yml
		if _, err := os.Stat(filepath.Join(dir, ConfigFileName)); err == nil {
			return dir, nil
		}
		// Check for phpunit.xml
		if _, err := os.Stat(filepath.Join(dir, "phpunit.xml")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "phpunit.xml.dist")); err == nil {
			return dir, nil
		}
		// Check for vitest config files
		for _, name := range []string{"vitest.config.ts", "vitest.config.mts", "vitest.config.js"} {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				return dir, nil
			}
		}
		// Check for jest config files
		for _, name := range []string{"jest.config.ts", "jest.config.js", "jest.config.mjs", "jest.config.cjs"} {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", os.ErrNotExist
}
