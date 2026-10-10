package originaldata

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/files"
	testoriginaldata "github.com/bradhannah/UltimaVRedux/test/originaldata"
)

func TestValidateInstallationRequiresExactInitialTemplates(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, files.DATA_OVL), make([]byte, originalDataOvlSize), 0o600); err != nil {
		t.Fatal(err)
	}

	err := ValidateInstallation(context.Background(), directory)
	if err == nil || !strings.Contains(err.Error(), files.INIT_GAM) {
		t.Fatalf("ValidateInstallation() error = %v, want missing %s", err, files.INIT_GAM)
	}

	if err := os.WriteFile(filepath.Join(directory, files.INIT_GAM), make([]byte, originalInitGamSize), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, files.INIT_OOL), make([]byte, originalInitObjectsSize-1), 0o600); err != nil {
		t.Fatal(err)
	}
	err = ValidateInstallation(context.Background(), directory)
	if err == nil || !strings.Contains(err.Error(), files.INIT_OOL) {
		t.Fatalf("ValidateInstallation() error = %v, want invalid %s", err, files.INIT_OOL)
	}
}

func TestValidateInstallationRequiresLazyGameplayCatalogue(t *testing.T) {
	directory := t.TempDir()
	fixtures := map[string]int{
		files.DATA_OVL: originalDataOvlSize,
		files.INIT_GAM: originalInitGamSize,
		files.INIT_OOL: originalInitObjectsSize,
	}
	for name, size := range fixtures {
		if err := os.WriteFile(filepath.Join(directory, name), make([]byte, size), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	err := ValidateInstallation(context.Background(), directory)
	if err == nil || !strings.Contains(err.Error(), files.BRIT_DAT) {
		t.Fatalf("ValidateInstallation() error = %v, want missing %s", err, files.BRIT_DAT)
	}
}

func TestValidateInstallationWithAvailableLocalOriginalData(t *testing.T) {
	directory := testoriginaldata.Directory(t, files.DATA_OVL)
	if err := ValidateInstallation(context.Background(), directory); err != nil {
		t.Fatalf("ValidateInstallation() error = %v", err)
	}
}
