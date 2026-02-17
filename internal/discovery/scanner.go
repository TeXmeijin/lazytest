package discovery

import (
	"os"
	"path/filepath"
	"sort"
)

// ScanFiles scans the given directories for files matching the pattern.
// Returns relative paths sorted alphabetically.
func ScanFiles(dirs []string, pattern string) ([]string, error) {
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

			matched, matchErr := filepath.Match(pattern, d.Name())
			if matchErr != nil {
				return matchErr
			}

			if matched {
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
