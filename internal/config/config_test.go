package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMultiTargetYAML(t *testing.T) {
	dir := t.TempDir()

	yamlContent := `targets:
  - name: phpunit
    command: "docker compose exec app php artisan test --teamcity {files}"
    test_dirs:
      - backend/src/tests/
    file_pattern: "*Test.php"
    path_strip_prefix: "backend/src/"
  - name: vitest
    command: "npx vitest run --reporter=teamcity {files}"
    test_dirs:
      - frontend/next/src/
      - frontend/next/app/
    file_pattern: "*.test.ts,*.test.tsx"
    working_dir: "frontend/next/"
`
	configPath := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.Targets) != 2 {
		t.Fatalf("Targets length = %d, want 2", len(cfg.Targets))
	}

	php := cfg.Targets[0]
	if php.Name != "phpunit" {
		t.Errorf("Targets[0].Name = %q", php.Name)
	}
	if php.Command != `docker compose exec app php artisan test --teamcity {files}` {
		t.Errorf("Targets[0].Command = %q", php.Command)
	}
	if len(php.TestDirs) != 1 || php.TestDirs[0] != "backend/src/tests/" {
		t.Errorf("Targets[0].TestDirs = %v", php.TestDirs)
	}
	if php.FilePattern != "*Test.php" {
		t.Errorf("Targets[0].FilePattern = %q", php.FilePattern)
	}
	if php.PathStripPrefix != "backend/src/" {
		t.Errorf("Targets[0].PathStripPrefix = %q", php.PathStripPrefix)
	}

	vt := cfg.Targets[1]
	if vt.Name != "vitest" {
		t.Errorf("Targets[1].Name = %q", vt.Name)
	}
	if vt.FilePattern != "*.test.ts,*.test.tsx" {
		t.Errorf("Targets[1].FilePattern = %q", vt.FilePattern)
	}
	if vt.WorkingDir != "frontend/next/" {
		t.Errorf("Targets[1].WorkingDir = %q", vt.WorkingDir)
	}
}

func TestLoadSingleTargetDefaults(t *testing.T) {
	dir := t.TempDir()

	yamlContent := `targets:
  - name: phpunit
    command: "phpunit --teamcity {files}"
`
	configPath := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.Targets) != 1 {
		t.Fatalf("Targets length = %d, want 1", len(cfg.Targets))
	}

	php := cfg.Targets[0]
	if php.FilePattern != "*Test.php" {
		t.Errorf("FilePattern default = %q, want *Test.php", php.FilePattern)
	}
	if len(php.TestDirs) != 1 || php.TestDirs[0] != "tests/" {
		t.Errorf("TestDirs default = %v, want [tests/]", php.TestDirs)
	}
}

func TestLoadVitestDefaults(t *testing.T) {
	dir := t.TempDir()

	yamlContent := `targets:
  - name: vitest
`
	configPath := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.Targets) != 1 {
		t.Fatalf("Targets length = %d, want 1", len(cfg.Targets))
	}

	vt := cfg.Targets[0]
	if vt.FilePattern != "*.test.ts,*.test.tsx" {
		t.Errorf("FilePattern default = %q, want *.test.ts,*.test.tsx", vt.FilePattern)
	}
	if vt.Command != "npx vitest run --reporter={reporter} {files}" {
		t.Errorf("Command default = %q", vt.Command)
	}
	if len(vt.TestDirs) != 1 || vt.TestDirs[0] != "src/" {
		t.Errorf("TestDirs default = %v, want [src/]", vt.TestDirs)
	}
}

func TestLoadMissingFileFallsBackToDetection(t *testing.T) {
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

	if len(cfg.Targets) < 1 {
		t.Fatal("expected at least 1 target from auto-detection")
	}

	php := cfg.Targets[0]
	if php.Name != "phpunit" {
		t.Errorf("Target name = %q, want phpunit", php.Name)
	}
	if len(php.TestDirs) != 2 {
		t.Fatalf("TestDirs length = %d, want 2", len(php.TestDirs))
	}
	if php.TestDirs[0] != "tests/Feature/" {
		t.Errorf("TestDirs[0] = %q, want tests/Feature/", php.TestDirs[0])
	}
}

func TestTargetApplyDefaultsPHPUnit(t *testing.T) {
	target := Target{Name: "phpunit"}
	target.applyDefaults()

	if target.FilePattern != "*Test.php" {
		t.Errorf("FilePattern = %q", target.FilePattern)
	}
	if target.Command != "./vendor/bin/phpunit --teamcity {files}" {
		t.Errorf("Command = %q", target.Command)
	}
	if len(target.TestDirs) != 1 || target.TestDirs[0] != "tests/" {
		t.Errorf("TestDirs = %v", target.TestDirs)
	}
}

func TestTargetApplyDefaultsVitest(t *testing.T) {
	target := Target{Name: "vitest"}
	target.applyDefaults()

	if target.FilePattern != "*.test.ts,*.test.tsx" {
		t.Errorf("FilePattern = %q", target.FilePattern)
	}
	if target.Command != "npx vitest run --reporter={reporter} {files}" {
		t.Errorf("Command = %q", target.Command)
	}
	if len(target.TestDirs) != 1 || target.TestDirs[0] != "src/" {
		t.Errorf("TestDirs = %v", target.TestDirs)
	}
}

func TestTargetApplyDefaultsJest(t *testing.T) {
	target := Target{Name: "jest"}
	target.applyDefaults()

	if target.FilePattern != "*.test.ts,*.test.tsx,*.test.js,*.test.jsx" {
		t.Errorf("FilePattern = %q", target.FilePattern)
	}
	if target.Command != "npx jest --reporters={reporter} {files}" {
		t.Errorf("Command = %q", target.Command)
	}
	if len(target.TestDirs) != 1 || target.TestDirs[0] != "src/" {
		t.Errorf("TestDirs = %v", target.TestDirs)
	}
}

func TestLoadJestDefaults(t *testing.T) {
	dir := t.TempDir()

	yamlContent := `targets:
  - name: jest
`
	configPath := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(cfg.Targets) != 1 {
		t.Fatalf("Targets length = %d, want 1", len(cfg.Targets))
	}

	jt := cfg.Targets[0]
	if jt.FilePattern != "*.test.ts,*.test.tsx,*.test.js,*.test.jsx" {
		t.Errorf("FilePattern default = %q", jt.FilePattern)
	}
	if jt.Command != "npx jest --reporters={reporter} {files}" {
		t.Errorf("Command default = %q", jt.Command)
	}
	if len(jt.TestDirs) != 1 || jt.TestDirs[0] != "src/" {
		t.Errorf("TestDirs default = %v", jt.TestDirs)
	}
}

func TestFindProjectRootJest(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "packages", "app")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(dir, "jest.config.ts"), []byte(""), 0644)

	root, err := FindProjectRoot(sub)
	if err != nil {
		t.Fatalf("FindProjectRoot error: %v", err)
	}
	if root != dir {
		t.Errorf("root = %q, want %q", root, dir)
	}
}

func TestFindProjectRootVitest(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "packages", "app")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(dir, "vitest.config.ts"), []byte(""), 0644)

	root, err := FindProjectRoot(sub)
	if err != nil {
		t.Fatalf("FindProjectRoot error: %v", err)
	}
	if root != dir {
		t.Errorf("root = %q, want %q", root, dir)
	}
}
