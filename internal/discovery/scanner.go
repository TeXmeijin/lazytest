package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/meijin/lazytest/internal/config"
	"github.com/meijin/lazytest/internal/domain"
)

// ScanFiles scans the given directories for files matching the pattern.
// Pattern can be comma-separated (e.g. "*.test.ts,*.test.tsx") for OR matching.
// Returns relative paths sorted alphabetically.
func ScanFiles(dirs []string, pattern string) ([]string, error) {
	patterns := splitPatterns(pattern)
	seen := make(map[string]bool)
	var files []string

	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // skip inaccessible files/dirs
			}
			if d.IsDir() {
				return nil
			}

			if matchesAny(d.Name(), patterns) {
				// Normalize to forward slashes for consistency
				rel := filepath.ToSlash(path)
				if !seen[rel] {
					seen[rel] = true
					files = append(files, rel)
				}
			}
			return nil
		})

		if err != nil {
			if os.IsNotExist(err) {
				continue // skip non-existent dirs
			}
			return nil, err
		}
	}

	sort.Strings(files)
	return files, nil
}

// ScanAllTargets scans files for all targets and returns them as TestFile slice.
// Results are sorted by target name then path.
func ScanAllTargets(targets []config.Target) ([]domain.TestFile, error) {
	var allFiles []domain.TestFile

	for _, target := range targets {
		paths, err := ScanFiles(target.TestDirs, target.FilePattern)
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			allFiles = append(allFiles, domain.TestFile{
				Path:       p,
				TargetName: target.Name,
			})
		}
	}

	// Sort by target name, then by path
	sort.Slice(allFiles, func(i, j int) bool {
		if allFiles[i].TargetName != allFiles[j].TargetName {
			return allFiles[i].TargetName < allFiles[j].TargetName
		}
		return allFiles[i].Path < allFiles[j].Path
	})

	return allFiles, nil
}

// splitPatterns splits a comma-separated pattern string into individual patterns.
func splitPatterns(pattern string) []string {
	parts := strings.Split(pattern, ",")
	var patterns []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			patterns = append(patterns, p)
		}
	}
	return patterns
}

// matchesAny returns true if name matches any of the patterns.
func matchesAny(name string, patterns []string) bool {
	for _, p := range patterns {
		matched, err := filepath.Match(p, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}
