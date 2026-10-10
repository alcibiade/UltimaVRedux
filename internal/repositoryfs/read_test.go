package repositoryfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadRegularFileRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "link")
	if err := os.WriteFile(target, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := ReadRegularFile(link); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("ReadRegularFile(symlink) error = %v, want ErrNotRegular", err)
	}
}
