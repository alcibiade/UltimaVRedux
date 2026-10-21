package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveMatchesOriginalFilenamesInAnyCase(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "data.ovl"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if got := filepath.Base(Resolve(directory, DataOvl)); got != "data.ovl" && got != DataOvl {
		t.Fatalf("Resolve(DATA.OVL) = %q, want the lowercase copy", got)
	}
	if got, want := Resolve(directory, BritDat), filepath.Join(directory, BritDat); got != want {
		t.Fatalf("missing file resolved to %q, want the exact path %q", got, want)
	}
}

func TestResolveRefusesAmbiguousCaseVariants(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"Brit.dat", "brit.DAT"} {
		if err := os.WriteFile(filepath.Join(directory, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if entries, _ := os.ReadDir(directory); len(entries) < 2 {
		t.Skip("case-insensitive filesystem holds only one variant")
	}

	if got, want := Resolve(directory, BritDat), filepath.Join(directory, BritDat); got != want {
		t.Fatalf("ambiguous variants resolved to %q, want the exact (missing) path %q", got, want)
	}
}
