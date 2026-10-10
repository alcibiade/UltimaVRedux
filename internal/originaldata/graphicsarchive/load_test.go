package graphicsarchive

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRejectsOversizedArchiveBeforeReading(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TITLE.BIT")
	file, err := os.Create(path) //nolint:gosec // Test fixture path is isolated.
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maximumArchiveBytes + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = Load(context.Background(), filepath.Dir(path), filepath.Base(path))
	if !errors.Is(err, errArchiveTooLarge) {
		t.Fatalf("Load() error = %v, want errArchiveTooLarge", err)
	}
}

func TestLoadHonorsPreCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Load(ctx, t.TempDir(), "TITLE.BIT")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Load() error = %v, want context.Canceled", err)
	}
}
