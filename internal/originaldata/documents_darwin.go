//go:build darwin

package originaldata

import "path/filepath"

func platformDocumentsDirectory(home string) (string, error) {
	return filepath.Join(home, "Documents"), nil
}
