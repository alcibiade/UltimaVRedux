//go:build linux

package originaldata

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func platformDocumentsDirectory(home string) (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	file, err := os.Open(filepath.Join(configHome, "user-dirs.dirs"))
	if err != nil {
		return filepath.Join(home, "Documents"), nil
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, found := strings.Cut(line, "=")
		if !found || key != "XDG_DOCUMENTS_DIR" {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		value = strings.ReplaceAll(value, "${HOME}", home)
		value = strings.ReplaceAll(value, "$HOME", home)
		if value != "" {
			return filepath.Clean(value), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return filepath.Join(home, "Documents"), nil
}
