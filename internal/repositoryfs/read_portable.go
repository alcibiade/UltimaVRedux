//go:build !aix && !android && !darwin && !dragonfly && !freebsd && !illumos && !ios && !linux && !netbsd && !openbsd && !solaris

package repositoryfs

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// ReadRegularFile verifies the opened descriptor against the current path on
// platforms without O_NOFOLLOW. Reading remains descriptor-based.
//
//nolint:nonamedreturns // Deferred close preserves read/stat errors and rejects a failed close.
func ReadRegularFile(path string) (data []byte, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			data = nil
			err = errors.Join(err, fmt.Errorf("close %s: %w", path, closeErr))
		}
	}()
	opened, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	current, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("lstat %s: %w", path, err)
	}
	if !opened.Mode().IsRegular() || !current.Mode().IsRegular() || !os.SameFile(opened, current) {
		return nil, ErrNotRegular
	}
	data, err = io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return data, nil
}
