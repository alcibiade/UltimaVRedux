package introarchive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndValidateRejectsNonemptyCorruptArchive(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "TITLE.BIT"), []byte{1}, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadAndValidate(directory)
	if err == nil {
		t.Fatal("LoadAndValidate() accepted corrupt TITLE.BIT")
	}
}
