package legacy_save

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/originaldata"
)

func TestDiscoverFindsAndPreviewsValidatedSavePairs(t *testing.T) {
	home := t.TempDir()
	active := filepath.Join(home, "active")
	if err := os.Mkdir(active, 0o700); err != nil {
		t.Fatal(err)
	}
	writeLegacyDiscoveryBundle(t, active)
	report := Discover(context.Background(), DiscoveryOptions{
		Host: originaldata.HostProfile{
			GOOS: "linux", HomeDirectory: home, DocumentsDir: filepath.Join(home, "Documents"),
			ExecutablePath: filepath.Join(home, "bin", "redux"), Environment: map[string]string{},
		},
		ActiveDataDirectory: active,
	})

	if report.Cancelled || len(report.Candidates) != 1 {
		t.Fatalf("Discover() = %#v, want one candidate", report)
	}
	candidate := report.Candidates[0]
	if candidate.SavedGamePath != filepath.Join(active, "SAVED.GAM") ||
		candidate.Directory != active || candidate.Label != "Save in current game folder" ||
		candidate.AvatarName != "C00" || candidate.Location != 13 {
		t.Fatalf("candidate = %#v", candidate)
	}
}

func TestDiscoverSeparatesIncompleteLocationsFromUsableCandidates(t *testing.T) {
	home := t.TempDir()
	active := filepath.Join(home, "active")
	if err := os.Mkdir(active, 0o700); err != nil {
		t.Fatal(err)
	}
	gam, _ := validBundleBytes()
	if err := os.WriteFile(filepath.Join(active, "SAVED.GAM"), gam, 0o600); err != nil {
		t.Fatal(err)
	}
	report := Discover(context.Background(), DiscoveryOptions{
		Host: originaldata.HostProfile{
			GOOS: "linux", HomeDirectory: home, DocumentsDir: filepath.Join(home, "Documents"),
			Environment: map[string]string{},
		},
		ActiveDataDirectory: active,
	})

	if len(report.Candidates) != 0 || len(report.Details) == 0 {
		t.Fatalf("Discover() = %#v, want diagnostics without candidates", report)
	}
}

func TestDiscoverHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	report := Discover(ctx, DiscoveryOptions{Host: originaldata.HostProfile{
		GOOS: "linux", HomeDirectory: t.TempDir(), Environment: map[string]string{},
	}})

	if !report.Cancelled || len(report.Candidates) != 0 {
		t.Fatalf("Discover() = %#v, want cancelled empty report", report)
	}
}

func writeLegacyDiscoveryBundle(t *testing.T, directory string) {
	t.Helper()
	gam, ool := validBundleBytes()
	if err := os.WriteFile(filepath.Join(directory, "SAVED.GAM"), gam, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "SAVED.OOL"), ool, 0o600); err != nil {
		t.Fatal(err)
	}
}
