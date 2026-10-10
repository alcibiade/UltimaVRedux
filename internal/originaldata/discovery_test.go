package originaldata

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecommendedDirectoryUsesResolvedDocumentsFolderAcrossPlatforms(t *testing.T) {
	for _, test := range []struct {
		name     string
		profile  HostProfile
		expected string
	}{
		{
			name:     "windows redirected documents",
			profile:  HostProfile{GOOS: "windows", HomeDirectory: `C:\Users\Avatar`, DocumentsDir: `D:\Personal\Documents`},
			expected: filepath.Join(`D:\Personal\Documents`, "UltimaV"),
		},
		{
			name:     "macOS documents",
			profile:  HostProfile{GOOS: "darwin", HomeDirectory: "/Users/avatar", DocumentsDir: "/Users/avatar/Documents"},
			expected: "/Users/avatar/Documents/UltimaV",
		},
		{
			name:     "Linux XDG documents",
			profile:  HostProfile{GOOS: "linux", HomeDirectory: "/home/avatar", DocumentsDir: "/home/avatar/Docs"},
			expected: "/home/avatar/Docs/UltimaV",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, RecommendedDirectory(test.profile))
		})
	}
}

func TestDiscoverUsesHeroicMetadataAndDeduplicatesCanonicalPaths(t *testing.T) {
	home := t.TempDir()
	install := filepath.Join(home, "Games", "Ultima V")
	require.NoError(t, os.MkdirAll(install, 0o700))
	metadataPath := filepath.Join(home, ".config", "heroic", "gog_store", "installed.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(metadataPath), 0o700))
	metadata, err := json.Marshal(map[string]any{
		"installed": []map[string]string{{"appName": GOGGameID, "install_path": install}},
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(metadataPath, metadata, 0o600))
	alias := filepath.Join(home, "alias")
	require.NoError(t, os.Symlink(install, alias))

	report := Discover(context.Background(), DiscoveryOptions{
		Host: HostProfile{
			GOOS: "linux", HomeDirectory: home, DocumentsDir: filepath.Join(home, "Documents"),
			ExecutablePath: filepath.Join(home, "bin", "ultima5redux"),
			Environment:    map[string]string{"XDG_CONFIG_HOME": filepath.Join(home, ".config")},
		},
		ConfiguredDirectory: alias,
		Validate: func(_ context.Context, path string) error {
			if canonicalIdentity(path, "linux") == canonicalIdentity(install, "linux") {
				return nil
			}

			return errors.New("not a fixture installation")
		},
	})

	require.Len(t, report.Candidates, 1)
	assert.Equal(t, canonicalIdentity(install, "linux"), canonicalIdentity(report.Candidates[0].Path, "linux"))
	assert.Equal(t, filepath.Join(home, "Documents", "UltimaV"), report.RecommendedDirectory)
}

func TestDiscoverResolvesCurrentAndLegacyMacBundleLayouts(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, "Applications", "Ultima V™.app", "Contents", "Resources", "game")
	legacy := filepath.Join(
		root, "Legacy.app", "Contents", "Resources", "game", "Ultima 5.app", "Contents", "Resources",
		"Ultima 5.boxer", "C ULTIMA5.harddisk",
	)
	for _, directory := range []string{current, legacy} {
		require.NoError(t, os.MkdirAll(directory, 0o700))
	}

	report := Discover(context.Background(), DiscoveryOptions{
		Host: HostProfile{GOOS: "darwin", HomeDirectory: root, DocumentsDir: filepath.Join(root, "Documents")},
		AdditionalSeeds: []Seed{
			{Path: filepath.Join(root, "Applications", "Ultima V™.app"), Source: SourceGOG},
			{Path: filepath.Join(root, "Legacy.app"), Source: SourceGOG},
		},
		Validate: func(_ context.Context, path string) error {
			if path == current || path == legacy {
				return nil
			}

			return errors.New("not game data")
		},
	})

	assert.ElementsMatch(t, []string{current, legacy}, StableCandidatePaths(report))
}

func TestDiscoverCancellationReturnsNoLateCandidates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report := Discover(ctx, DiscoveryOptions{
		Host:     HostProfile{GOOS: "linux", HomeDirectory: t.TempDir()},
		Validate: func(context.Context, string) error { return nil },
	})

	assert.True(t, report.Cancelled)
	assert.Empty(t, report.Candidates)
}

func TestResolveDirectoryDeduplicatesSymlinkEquivalentLayouts(t *testing.T) {
	root := t.TempDir()
	installation := filepath.Join(root, "UltimaV")
	require.NoError(t, os.Mkdir(installation, 0o700))
	require.NoError(t, os.Symlink(installation, filepath.Join(root, "Ultima5")))

	matches, err := ResolveDirectory(context.Background(), root, "linux", func(_ context.Context, path string) error {
		if canonicalIdentity(path, "linux") == canonicalIdentity(installation, "linux") {
			return nil
		}

		return errors.New("not game data")
	})

	require.NoError(t, err)
	assert.Equal(t, []string{installation}, matches)
}

func TestDiscoverEnforcesSeedAndHeroicMetadataBounds(t *testing.T) {
	home := t.TempDir()
	metadataPath := filepath.Join(home, ".config", "heroic", "gog_store", "installed.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(metadataPath), 0o700))
	require.NoError(t, os.WriteFile(metadataPath, make([]byte, heroicMetadataMaximumBytes+1), 0o600))
	additional := make([]Seed, discoverySeedLimit+1)
	for index := range additional {
		additional[index] = Seed{Path: filepath.Join(home, "seed", strconv.Itoa(index)), Source: SourceCommon}
	}

	report := Discover(context.Background(), DiscoveryOptions{
		Host: HostProfile{
			GOOS: "linux", HomeDirectory: home, DocumentsDir: filepath.Join(home, "Documents"),
			Environment: map[string]string{"XDG_CONFIG_HOME": filepath.Join(home, ".config")},
		},
		AdditionalSeeds: additional,
		Validate:        func(context.Context, string) error { return nil },
	})

	assert.Len(t, report.CheckedRoots, discoverySeedLimit)
	assert.Condition(t, func() bool {
		return slices.ContainsFunc(report.Details, func(detail Detail) bool {
			return strings.Contains(detail.Reason, "metadata exceeds")
		})
	})
	assert.Condition(t, func() bool {
		return slices.ContainsFunc(report.Details, func(detail Detail) bool {
			return strings.Contains(detail.Reason, "bounded search stopped")
		})
	})
}

func TestWindowsGOGSeedsAcceptCustomRecordsAndRejectOtherPublishers(t *testing.T) {
	seeds := WindowsGOGSeeds([]WindowsGOGRegistryRecord{
		{GamePath: `D:\Games\Ultima V`, UninstallPath: "", UninstallPublisher: ""},
		{GamePath: "", UninstallPath: `E:\GOG\Ultima 5`, UninstallPublisher: "GOG.com"},
		{GamePath: "", UninstallPath: `C:\Unrelated`, UninstallPublisher: "Other"},
	})

	require.Len(t, seeds, 2)
	assert.Equal(t, `D:\Games\Ultima V`, seeds[0].Path)
	assert.Equal(t, "GOG installation record", seeds[0].Label)
	assert.Equal(t, `E:\GOG\Ultima 5`, seeds[1].Path)
	assert.Equal(t, "GOG uninstall record", seeds[1].Label)
}

func TestCommonSeedsUseGOGRootsAndProgramFilesFallback(t *testing.T) {
	programFiles := filepath.Join(string(filepath.Separator), "Program Files")
	seeds := commonSeeds(HostProfile{
		GOOS: "windows", HomeDirectory: filepath.Join(string(filepath.Separator), "Users", "Avatar"),
		DocumentsDir: filepath.Join(string(filepath.Separator), "Users", "Avatar", "Documents"),
		Environment:  map[string]string{"ProgramFiles": programFiles, "SystemDrive": "C:"},
	})

	paths := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		paths = append(paths, seed.Path)
	}
	gogRoot := filepath.Join(programFiles, "GOG Galaxy", "Games")
	assert.Contains(t, paths, gogRoot)
	assert.Contains(t, boundedLayouts(gogRoot, "windows"),
		filepath.Join(gogRoot, "Ultima 4-5-6", "Ultima 5"))
}
