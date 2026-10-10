//go:build !windows

package repositoryfs

import (
	"errors"
	"fmt"
	"os"
)

// SyncDirectory flushes directory metadata after publication or removal.
func SyncDirectory(path string) error {
	directory, err := os.Open(path) //nolint:gosec // Repository roots are application-controlled paths.
	if err != nil {
		return fmt.Errorf("open directory for sync: %w", err)
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	if syncErr != nil {
		syncErr = fmt.Errorf("sync directory: %w", syncErr)
	}
	if closeErr != nil {
		closeErr = fmt.Errorf("close directory: %w", closeErr)
	}

	return errors.Join(syncErr, closeErr)
}
