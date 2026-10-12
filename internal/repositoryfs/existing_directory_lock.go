package repositoryfs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// AcquireExistingDirectoryLock shares the repository reference lock namespace
// without creating or chmodding the directory. Standalone exports use this when
// their parent is a user-owned directory rather than a private repository root.
func AcquireExistingDirectoryLock(ctx context.Context, directory string) (func(), error) {
	info, err := os.Lstat(directory)
	if err != nil {
		return nil, fmt.Errorf("inspect publication directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("publication directory: %w", ErrNotRegular)
	}
	lockPath := filepath.Join(directory, ReferenceLockName)
	if info, err = os.Lstat(lockPath); err == nil && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("publication lock: %w", ErrNotRegular)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect publication lock: %w", err)
	}
	lock := flock.New(lockPath)
	lockContext, cancel := context.WithTimeout(ctx, lockTimeout)
	defer cancel()
	locked, err := lock.TryLockContext(lockContext, lockRetry)
	if err != nil {
		return nil, fmt.Errorf("acquire publication lock: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("acquire publication lock: %w", context.DeadlineExceeded)
	}

	return func() { release([]*flock.Flock{lock}) }, nil
}
