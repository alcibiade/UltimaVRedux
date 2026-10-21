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

func TestUniqueFoldMatchRefusesAmbiguousCaseVariants(t *testing.T) {
	if got, ok := uniqueFoldMatch([]string{"Brit.dat", "brit.DAT", "TOWNE.DAT"}, BritDat); ok {
		t.Fatalf("ambiguous variants matched %q, want no match", got)
	}
	if got, ok := uniqueFoldMatch([]string{"brit.dat", "TOWNE.DAT"}, BritDat); !ok || got != "brit.dat" {
		t.Fatalf("single variant matched %q (%t), want brit.dat", got, ok)
	}
}
