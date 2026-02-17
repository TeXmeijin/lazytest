package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/meijin/lazytest/internal/config"
)

func TestScanFindsMatchingFiles(t *testing.T) {
	dir := t.TempDir()

	// Create test structure
	dirs := []string{
		filepath.Join(dir, "tests", "Feature"),
		filepath.Join(dir, "tests", "Unit"),
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}

	testFiles := []string{
		filepath.Join(dir, "tests", "Feature", "LoginTest.php"),
		filepath.Join(dir, "tests", "Feature", "Auth", "RegisterTest.php"),
		filepath.Join(dir, "tests", "Unit", "HelperTest.php"),
		filepath.Join(dir, "tests", "Feature", "Helper.php"), // should not match
	}
	os.MkdirAll(filepath.Join(dir, "tests", "Feature", "Auth"), 0755)
	for _, f := range testFiles {
		os.WriteFile(f, []byte("<?php\n"), 0644)
	}

	scanDir := filepath.Join(dir, "tests")
	files, err := ScanFiles([]string{scanDir}, "*Test.php")
	if err != nil {
		t.Fatalf("ScanFiles error: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("got %d files, want 3: %v", len(files), files)
	}
}

func TestScanMultipleDirs(t *testing.T) {
	dir := t.TempDir()

	dir1 := filepath.Join(dir, "tests1")
	dir2 := filepath.Join(dir, "tests2")
	os.MkdirAll(dir1, 0755)
	os.MkdirAll(dir2, 0755)

	os.WriteFile(filepath.Join(dir1, "FooTest.php"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir2, "BarTest.php"), []byte(""), 0644)

	files, err := ScanFiles([]string{dir1, dir2}, "*Test.php")
	if err != nil {
		t.Fatalf("ScanFiles error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
}

func TestScanNonExistentDirSkipped(t *testing.T) {
	files, err := ScanFiles([]string{"/tmp/nonexistent-dir-lazytest"}, "*Test.php")
	if err != nil {
		t.Fatalf("ScanFiles error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("got %d files, want 0", len(files))
	}
}

func TestScanEmptyInput(t *testing.T) {
	files, err := ScanFiles(nil, "*Test.php")
	if err != nil {
		t.Fatalf("ScanFiles error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("got %d files, want 0", len(files))
	}
}

func TestScanResultsSorted(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ZTest.php"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "ATest.php"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "MTest.php"), []byte(""), 0644)

	files, err := ScanFiles([]string{dir}, "*Test.php")
	if err != nil {
		t.Fatalf("ScanFiles error: %v", err)
	}

	for i := 1; i < len(files); i++ {
		if files[i] < files[i-1] {
			t.Errorf("not sorted: %v", files)
			break
		}
	}
}

func TestScanCommaPattern(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "app.test.ts"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "page.test.tsx"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "util.ts"), []byte(""), 0644)

	files, err := ScanFiles([]string{dir}, "*.test.ts,*.test.tsx")
	if err != nil {
		t.Fatalf("ScanFiles error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2: %v", len(files), files)
	}
}

func TestScanAllTargets(t *testing.T) {
	dir := t.TempDir()

	// PHP tests
	phpDir := filepath.Join(dir, "server", "tests")
	os.MkdirAll(phpDir, 0755)
	os.WriteFile(filepath.Join(phpDir, "UserTest.php"), []byte(""), 0644)

	// Vitest tests
	vtDir := filepath.Join(dir, "client", "src")
	os.MkdirAll(vtDir, 0755)
	os.WriteFile(filepath.Join(vtDir, "App.test.ts"), []byte(""), 0644)
	os.WriteFile(filepath.Join(vtDir, "Page.test.tsx"), []byte(""), 0644)

	targets := []config.Target{
		{
			Name:        "phpunit",
			TestDirs:    []string{phpDir},
			FilePattern: "*Test.php",
		},
		{
			Name:        "vitest",
			TestDirs:    []string{vtDir},
			FilePattern: "*.test.ts,*.test.tsx",
		},
	}

	files, err := ScanAllTargets(targets)
	if err != nil {
		t.Fatalf("ScanAllTargets error: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("got %d files, want 3: %v", len(files), files)
	}

	// Should be sorted: phpunit first (alphabetically), then vitest
	if files[0].TargetName != "phpunit" {
		t.Errorf("files[0].TargetName = %q, want phpunit", files[0].TargetName)
	}
	if files[1].TargetName != "vitest" {
		t.Errorf("files[1].TargetName = %q, want vitest", files[1].TargetName)
	}
	if files[2].TargetName != "vitest" {
		t.Errorf("files[2].TargetName = %q, want vitest", files[2].TargetName)
	}
}

func TestScanAllTargetsEmpty(t *testing.T) {
	files, err := ScanAllTargets(nil)
	if err != nil {
		t.Fatalf("ScanAllTargets error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("got %d files, want 0", len(files))
	}
}
