package files

import (
	"os"
	"path/filepath"
	"strings"
)

// Resolve returns the path of an original-data file, matching letter case
// loosely because some copy tools and archives lowercase every filename. An
// exact match wins; otherwise one case-insensitive match is used. With none,
// or more than one, it returns the exact path so callers report the usual
// missing-file error.
func Resolve(directory, name string) string {
	exact := filepath.Join(directory, name)
	if _, err := os.Lstat(exact); err == nil {
		return exact
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return exact
	}
	match := ""
	for _, entry := range entries {
		if !strings.EqualFold(entry.Name(), name) {
			continue
		}
		if match != "" {
			return exact
		}
		match = entry.Name()
	}
	if match == "" {
		return exact
	}

	return filepath.Join(directory, match)
}
