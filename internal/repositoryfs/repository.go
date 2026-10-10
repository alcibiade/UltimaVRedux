// Package repositoryfs provides shared persistence safety primitives for
// filesystem-backed repositories.
package repositoryfs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/gofrs/flock"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

const (
	// ReferenceLockName is the shared lock file used by scenario and snapshot repositories.
	ReferenceLockName = ".scenario-snapshot-references.lock"
	lockTimeout       = 5 * time.Second
	lockRetry         = 10 * time.Millisecond
)

// ErrNotRegular indicates that a repository path is not a regular file.
var ErrNotRegular = errors.New("repository path is not a regular file")

// Root describes one repository root and its required privacy policy.
type Root struct {
	Path     string
	DirMode  os.FileMode
	FileMode os.FileMode
}

// EnsureRoot creates a root and enforces its configured directory mode.
func EnsureRoot(root Root) error {
	if err := os.MkdirAll(root.Path, root.DirMode); err != nil {
		return fmt.Errorf("create repository root: %w", err)
	}
	info, err := os.Stat(root.Path)
	if err != nil {
		return fmt.Errorf("stat repository root: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("repository root is not a directory: %w", ErrNotRegular)
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	if err := os.Chmod(root.Path, root.DirMode); err != nil {
		return fmt.Errorf("set repository root permissions: %w", err)
	}

	return nil
}

// AcquireReferenceLocks locks every unique root in canonical path order.
// All operations that establish or remove scenario snapshot references use
// this same lock namespace, including across cooperating processes.
func AcquireReferenceLocks(roots []Root) (func(), error) {
	unique := make(map[string]Root, len(roots))
	for _, root := range roots {
		root.Path = filepath.Clean(root.Path)
		if existing, ok := unique[root.Path]; ok {
			if root.DirMode.Perm() < existing.DirMode.Perm() {
				unique[root.Path] = root
			}

			continue
		}
		unique[root.Path] = root
	}
	paths := make([]string, 0, len(unique))
	for path := range unique {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	locks := make([]*flock.Flock, 0, len(paths))
	ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()
	for _, path := range paths {
		root := unique[path]
		if err := EnsureRoot(root); err != nil {
			release(locks)

			return nil, fmt.Errorf("create repository root: %w", err)
		}
		lock := flock.New(filepath.Join(path, ReferenceLockName))
		locked, err := lock.TryLockContext(ctx, lockRetry)
		if err != nil {
			release(locks)

			return nil, fmt.Errorf("acquire scenario snapshot reference lock: %w", err)
		}
		if !locked {
			release(locks)

			return nil, errors.New("timed out waiting for scenario snapshot reference lock")
		}
		locks = append(locks, lock)
	}

	return func() { release(locks) }, nil
}

func release(locks []*flock.Flock) {
	for index := len(locks) - 1; index >= 0; index-- {
		if err := locks[index].Unlock(); err != nil {
			// Release runs after publication as well as failed acquisition. Do not
			// turn an already committed transaction into a reported write failure.
			logging.Error(logging.CategorySystem, "Releasing repository reference lock: %v", err)
		}
	}
}
