package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidYAML(t *testing.T) {
	dir := t.TempDir()

	yamlContent := `command: "docker compose exec php-fpm php artisan test --teamcity {files}"
test_dirs:
  - src/tests/Feature
  - src/tests/Unit
file_pattern: "*Test.php"
path_strip_prefix: "src/"
`
	configPath := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Command != `docker compose exec php-fpm php artisan test --teamcity {files}` {
		t.Errorf("Command = %q", cfg.Command)
	}
	if len(cfg.TestDirs) != 2 {
		t.Fatalf("TestDirs length = %d, want 2", len(cfg.TestDirs))
	}
	if cfg.TestDirs[0] != "src/tests/Feature" {
		t.Errorf("TestDirs[0] = %q", cfg.TestDirs[0])
	}
	if cfg.FilePattern != "*Test.php" {
		t.Errorf("FilePattern = %q", cfg.FilePattern)
	}
	if cfg.PathStripPrefix != "src/" {
		t.Errorf("PathStripPrefix = %q", cfg.PathStripPrefix)
	}
}

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()

	yamlContent := `command: "phpunit --teamcity {files}"
`
	configPath := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.FilePattern != "*Test.php" {
		t.Errorf("FilePattern default = %q, want *Test.php", cfg.FilePattern)
	}
	if len(cfg.TestDirs) != 1 || cfg.TestDirs[0] != "tests/" {
		t.Errorf("TestDirs default = %v, want [tests/]", cfg.TestDirs)
	}
}

func TestLoadMissingFileFallsBackToPHPUnit(t *testing.T) {
	dir := t.TempDir()

	// Create a phpunit.xml
	phpunitContent := `<?xml version="1.0" encoding="UTF-8"?>
<phpunit>
  <testsuites>
    <testsuite name="Feature">
      <directory>tests/Feature</directory>
    </testsuite>
    <testsuite name="Unit">
      <directory>tests/Unit</directory>
    </testsuite>
  </testsuites>
</phpunit>
`
	if err := os.WriteFile(filepath.Join(dir, "phpunit.xml"), []byte(phpunitContent), 0644); err != nil {
		t.Fatalf("writing phpunit.xml: %v", err)
	}

	// chdir so Load finds phpunit.xml
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.TestDirs) != 2 {
		t.Fatalf("TestDirs length = %d, want 2", len(cfg.TestDirs))
	}
	if cfg.TestDirs[0] != "tests/Feature/" {
		t.Errorf("TestDirs[0] = %q, want tests/Feature/", cfg.TestDirs[0])
	}
	if cfg.Command != "./vendor/bin/phpunit --teamcity {files}" {
		t.Errorf("Command = %q", cfg.Command)
	}
}

func TestDetectPHPUnit(t *testing.T) {
	dir := t.TempDir()

	phpunitContent := `<?xml version="1.0" encoding="UTF-8"?>
<phpunit>
  <testsuites>
    <testsuite name="Tests">
      <directory>tests</directory>
    </testsuite>
  </testsuites>
</phpunit>
`
	if err := os.WriteFile(filepath.Join(dir, "phpunit.xml"), []byte(phpunitContent), 0644); err != nil {
		t.Fatalf("writing phpunit.xml: %v", err)
	}

	cfg, err := DetectPHPUnit(dir)
	if err != nil {
		t.Fatalf("DetectPHPUnit returned error: %v", err)
	}

	if len(cfg.TestDirs) != 1 || cfg.TestDirs[0] != "tests/" {
		t.Errorf("TestDirs = %v, want [tests/]", cfg.TestDirs)
	}
}

func TestDetectPHPUnitDist(t *testing.T) {
	dir := t.TempDir()

	phpunitContent := `<?xml version="1.0" encoding="UTF-8"?>
<phpunit>
  <testsuites>
    <testsuite name="Feature">
      <directory>tests/Feature</directory>
    </testsuite>
  </testsuites>
</phpunit>
`
	if err := os.WriteFile(filepath.Join(dir, "phpunit.xml.dist"), []byte(phpunitContent), 0644); err != nil {
		t.Fatalf("writing phpunit.xml.dist: %v", err)
	}

	cfg, err := DetectPHPUnit(dir)
	if err != nil {
		t.Fatalf("DetectPHPUnit returned error: %v", err)
	}

	if len(cfg.TestDirs) != 1 {
		t.Fatalf("TestDirs length = %d, want 1", len(cfg.TestDirs))
	}
}

func TestDetectPHPUnitNotFound(t *testing.T) {
	dir := t.TempDir()

	cfg, err := DetectPHPUnit(dir)
	if err != nil {
		t.Fatalf("DetectPHPUnit returned error: %v", err)
	}

	// Should return defaults
	if cfg.FilePattern != "*Test.php" {
		t.Errorf("FilePattern = %q", cfg.FilePattern)
	}
}
