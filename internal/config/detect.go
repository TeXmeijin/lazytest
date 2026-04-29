package config

import (
	"os"
	"path/filepath"
	"strings"
)

// skipDirs are directories to skip during framework detection walks.
var skipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	".git":         true,
}

// DetectFrameworks scans root (up to 3 levels deep) for known test frameworks
// and returns a Target for each one found.
func DetectFrameworks(root string) ([]Target, error) {
	var targets []Target

	// Check root level for phpunit
	if t, found := detectPHPUnitTarget(root, ""); found {
		targets = append(targets, t)
	}

	// Check root level for vitest
	if t, found := detectVitestTarget(root, ""); found {
		targets = append(targets, t)
	}

	// Check root level for jest
	if t, found := detectJestTarget(root, ""); found {
		targets = append(targets, t)
	}

	// Walk up to 3 levels deep for nested projects
	err := walkLimited(root, 3, func(dir, rel string) {
		if t, found := detectPHPUnitTarget(dir, rel); found {
			// Avoid duplicates if already found at root
			if !hasTarget(targets, "phpunit", dir) {
				targets = append(targets, t)
			}
		}
		if t, found := detectVitestTarget(dir, rel); found {
			if !hasTarget(targets, "vitest", dir) {
				targets = append(targets, t)
			}
		}
		if t, found := detectJestTarget(dir, rel); found {
			if !hasTarget(targets, "jest", dir) {
				targets = append(targets, t)
			}
		}
	})
	if err != nil {
		return nil, err
	}

	// If nothing was found, return a default phpunit target
	if len(targets) == 0 {
		t := Target{Name: "phpunit"}
		t.applyDefaults()
		targets = append(targets, t)
	}

	return targets, nil
}

// walkLimited walks directories up to maxDepth levels, calling fn for each directory.
func walkLimited(root string, maxDepth int, fn func(dir, rel string)) error {
	return walkDir(root, "", 0, maxDepth, fn)
}

func walkDir(base, rel string, depth, maxDepth int, fn func(dir, rel string)) error {
	if depth > maxDepth {
		return nil
	}

	dir := base
	if rel != "" {
		dir = filepath.Join(base, rel)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // skip unreadable dirs
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if skipDirs[name] || strings.HasPrefix(name, ".") {
			continue
		}

		childRel := name
		if rel != "" {
			childRel = filepath.Join(rel, name)
		}

		fn(filepath.Join(base, childRel), childRel)

		if depth+1 < maxDepth {
			if err := walkDir(base, childRel, depth+1, maxDepth, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

// detectPHPUnitTarget checks if dir contains phpunit.xml or phpunit.xml.dist.
func detectPHPUnitTarget(dir, relFromRoot string) (Target, bool) {
	var data []byte
	for _, name := range []string{"phpunit.xml", "phpunit.xml.dist"} {
		d, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			data = d
			break
		}
	}
	if data == nil {
		return Target{}, false
	}

	dirs := parsePHPUnitDirs(data)

	// Normalize test_dirs relative to root
	if relFromRoot != "" {
		for i, d := range dirs {
			dirs[i] = filepath.ToSlash(filepath.Join(relFromRoot, d))
			if dirs[i][len(dirs[i])-1] != '/' {
				dirs[i] += "/"
			}
		}
	}

	t := Target{
		Name:     "phpunit",
		TestDirs: dirs,
	}
	if relFromRoot != "" {
		t.WorkingDir = relFromRoot + "/"
	}
	t.applyDefaults()
	return t, true
}

// detectVitestTarget checks if dir contains vitest.config.{ts,js,mts}.
func detectVitestTarget(dir, relFromRoot string) (Target, bool) {
	found := false
	for _, name := range []string{"vitest.config.ts", "vitest.config.mts", "vitest.config.js"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			found = true
			break
		}
	}
	if !found {
		return Target{}, false
	}

	t := Target{
		Name: "vitest",
	}
	if relFromRoot != "" {
		t.WorkingDir = relFromRoot + "/"
		t.TestDirs = []string{relFromRoot + "/src/"}
	}
	t.applyDefaults()
	return t, true
}

// detectJestTarget checks if dir contains jest.config.{ts,js,mjs,cjs}.
func detectJestTarget(dir, relFromRoot string) (Target, bool) {
	found := false
	for _, name := range []string{"jest.config.ts", "jest.config.js", "jest.config.mjs", "jest.config.cjs"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			found = true
			break
		}
	}
	if !found {
		return Target{}, false
	}

	t := Target{
		Name: "jest",
	}
	if relFromRoot != "" {
		t.WorkingDir = relFromRoot + "/"
		t.TestDirs = []string{relFromRoot + "/src/"}
	}
	t.applyDefaults()
	return t, true
}

func hasTarget(targets []Target, name, dir string) bool {
	for _, t := range targets {
		if t.Name == name {
			return true
		}
	}
	return false
}
