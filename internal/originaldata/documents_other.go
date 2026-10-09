//go:build !windows && !darwin && !linux

package originaldata

import "path/filepath"

func platformDocumentsDirectory(home string) (string, error) {
	return filepath.Join(home, "Documents"), nil
}
