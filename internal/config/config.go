package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const ConfigFileName = ".lazytest.yml"

// Config represents the lazytest configuration.
type Config struct {
	Command         string   `yaml:"command"`
	TestDirs        []string `yaml:"test_dirs"`
	FilePattern     string   `yaml:"file_pattern"`
	PathStripPrefix string   `yaml:"path_strip_prefix"`
	Editor          string   `yaml:"editor"`
}

// Load reads configuration from the given path or auto-detects.
// If configPath is empty, it looks for .lazytest.yml in the current directory.
// If .lazytest.yml is not found, it falls back to phpunit.xml detection.
func Load(configPath string) (Config, error) {
	if configPath == "" {
		configPath = ConfigFileName
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Fall back to phpunit.xml detection
			return DetectPHPUnit(".")
		}
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.FilePattern == "" {
		c.FilePattern = "*Test.php"
	}
	if c.Command == "" {
		c.Command = "./vendor/bin/phpunit --teamcity {files}"
	}
	if len(c.TestDirs) == 0 {
		c.TestDirs = []string{"tests/"}
	}
	if c.Editor == "" {
		if editor := os.Getenv("LAZYTEST_EDITOR"); editor != "" {
			c.Editor = editor
		} else {
			c.Editor = "zed"
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

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", os.ErrNotExist
}
