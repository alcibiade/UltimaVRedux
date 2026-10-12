package repositoryfs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestExistingDirectoryLockPreservesPermissionsAndSharesNamespace(t *testing.T) {
	root := t.TempDir()
	if runtime.GOOS != "windows" {
		if err := os.Chmod(root, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	unlock, err := AcquireExistingDirectoryLock(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o755 {
		t.Fatalf("parent permissions changed to %o", info.Mode().Perm())
	}
	unlock()
	unlockRepository, err := AcquireReferenceLocks([]Root{{Path: root, DirMode: 0o755, FileMode: 0o600}})
	if err != nil {
		t.Fatal(err)
	}
	defer unlockRepository()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	unexpectedUnlock, err := AcquireExistingDirectoryLock(ctx, root)
	if err == nil {
		unexpectedUnlock()
		t.Fatal("existing-directory lock did not share repository namespace")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unexpected lock error: %v", err)
	}
}

func TestExistingDirectoryLockDoesNotCreateMissingParent(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	_, err := AcquireExistingDirectoryLock(context.Background(), missing)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error = %v", err)
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("parent created: %v", err)
	}
}
