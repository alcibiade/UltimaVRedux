// Package originaldata discovers and validates locally installed original Ultima V data.
package originaldata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
)

const (
	// GOGGameID is the verified GOG catalogue identifier for Ultima V.
	GOGGameID                  = "1207662443"
	discoverySeedCapacity      = 24
	discoverySeedLimit         = 64
	heroicInstalledEntryLimit  = 256
	heroicMetadataMaximumBytes = 1 << 20
	goosWindows                = "windows"
	goosDarwin                 = "darwin"
	goosLinux                  = "linux"
)

var (
	errMetadataTooLarge = errors.New("provider metadata exceeds the discovery size limit")
	errHomeNotAbsolute  = errors.New("set an absolute home folder before locating Ultima V game files")
	errHomeNotDirectory = errors.New("the configured home path must be a folder before locating game files")
)

// Source identifies how a candidate path was discovered.
type Source string

// Discovery sources identify the finite catalogue category that produced a candidate.
const (
	SourceConfigured  Source = "Configured location"
	SourcePrevious    Source = "Previously used location"
	SourceGOG         Source = "GOG installation"
	SourceHeroic      Source = "Heroic installation"
	SourceCommon      Source = "Common installation location"
	SourceRecommended Source = "Recommended personal folder"
)

// HostProfile contains the host paths used to produce a finite discovery catalogue.
type HostProfile struct {
	GOOS          string
	HomeDirectory string
	Environment   map[string]string
	ProviderSeeds []Seed
}

// Seed is one bounded root or installation path to inspect.
type Seed struct {
	Path   string
	Source Source
	Label  string
}

// WindowsGOGRegistryRecord is the platform-neutral result of one registry-view query.
type WindowsGOGRegistryRecord struct {
	GamePath           string
	UninstallPath      string
	UninstallPublisher string
}

// Candidate is a verified playable original-data directory.
type Candidate struct {
	Path   string
	Source Source
	Label  string
}

// Detail records an existing candidate that could not be used.
type Detail struct {
	Path   string
	Source Source
	Reason string
}

// Report is the complete stable result of one bounded search.
type Report struct {
	Candidates           []Candidate
	Details              []Detail
	CheckedRoots         []Seed
	RecommendedDirectory string
	Cancelled            bool
}

// ValidateFunc validates one exact original-data directory.
type ValidateFunc func(context.Context, string) error

// DiscoveryOptions configures a deterministic bounded search.
type DiscoveryOptions struct {
	Host                  HostProfile
	ConfiguredDirectory   string
	PreviousDataDirectory string
	AdditionalSeeds       []Seed
	Validate              ValidateFunc
}

// CurrentHostProfile resolves host directories once for production discovery.
func CurrentHostProfile() (HostProfile, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return HostProfile{}, fmt.Errorf("locating home directory: %w", err)
	}
	if strings.TrimSpace(home) == "" || !filepath.IsAbs(home) {
		return HostProfile{}, errHomeNotAbsolute
	}
	info, err := os.Stat(home)
	if err != nil {
		return HostProfile{}, fmt.Errorf("make your home folder available before locating game files: %w", err)
	}
	if !info.IsDir() {
		return HostProfile{}, errHomeNotDirectory
	}
	host := HostProfile{
		GOOS:          runtime.GOOS,
		HomeDirectory: home,
		Environment:   environmentMap(),
		ProviderSeeds: platformProviderSeeds(),
	}

	return host, nil
}

// RecommendedDirectory returns the simple personal folder shown when managed
// installation discovery does not find a usable copy.
func RecommendedDirectory(host HostProfile) string {
	if strings.TrimSpace(host.HomeDirectory) == "" || !filepath.IsAbs(host.HomeDirectory) {
		return ""
	}

	return filepath.Join(host.HomeDirectory, "UltimaV")
}

// Discover completes the finite catalogue before returning stable verified results.
//
//nolint:gocognit,funlen // One bounded loop keeps path identity, diagnostics, and stable ordering auditable.
func Discover(ctx context.Context, options DiscoveryOptions) Report {
	report := Report{
		Candidates: nil, Details: nil, CheckedRoots: nil,
		RecommendedDirectory: RecommendedDirectory(options.Host), Cancelled: false,
	}
	if report.RecommendedDirectory == "" {
		report.Details = []Detail{{Path: options.Host.HomeDirectory, Source: SourceRecommended,
			Reason: "An absolute home folder is required; set it before checking installations again."}}

		return report
	}
	validator := options.Validate
	if validator == nil {
		validator = ValidateInstallation
	}
	seeds, seedDetails := discoverySeeds(options)
	report.CheckedRoots = slices.Clone(seeds)
	report.Details = append(report.Details, seedDetails...)
	seenPaths := make(map[string]struct{})
	seenCandidates := make(map[string]struct{})
	for _, seed := range seeds {
		for _, candidatePath := range boundedLayouts(seed.Path, options.Host.GOOS) {
			if ctx.Err() != nil {
				report.Cancelled = true

				return report
			}
			absolute, err := filepath.Abs(candidatePath)
			if err != nil {
				continue
			}
			identity := pathIdentity(absolute, options.Host.GOOS)
			if _, exists := seenPaths[identity]; exists {
				continue
			}
			seenPaths[identity] = struct{}{}
			info, statErr := os.Stat(absolute)
			if statErr != nil {
				if !errors.Is(statErr, os.ErrNotExist) {
					report.Details = append(report.Details, Detail{
						Path: absolute, Source: seed.Source, Reason: statErr.Error(),
					})
				}

				continue
			}
			if !info.IsDir() {
				continue
			}
			if err := validator(ctx, absolute); err != nil {
				report.Details = append(report.Details, Detail{
					Path: absolute, Source: seed.Source, Reason: err.Error(),
				})

				continue
			}
			canonical := canonicalIdentity(absolute, options.Host.GOOS)
			if _, exists := seenCandidates[canonical]; exists {
				continue
			}
			seenCandidates[canonical] = struct{}{}
			label := strings.TrimSpace(seed.Label)
			if label == "" {
				label = string(seed.Source)
			}
			report.Candidates = append(report.Candidates, Candidate{
				Path: absolute, Source: seed.Source, Label: label,
			})
		}
	}

	return report
}

func discoverySeeds(options DiscoveryOptions) ([]Seed, []Detail) {
	seeds := make([]Seed, 0, discoverySeedCapacity)
	appendDirectory := func(path string, source Source, label string) {
		if strings.TrimSpace(path) != "" {
			seeds = append(seeds, Seed{Path: path, Source: source, Label: label})
		}
	}
	appendDirectory(options.ConfiguredDirectory, SourceConfigured, "Configured location")
	appendDirectory(options.PreviousDataDirectory, SourcePrevious, "Previously used folder")
	for _, seed := range options.Host.ProviderSeeds {
		if filepath.IsAbs(seed.Path) {
			seeds = append(seeds, seed)
		}
	}
	heroic, details := heroicSeeds(options.Host)
	seeds = append(seeds, heroic...)
	seeds = append(seeds, commonSeeds(options.Host)...)
	appendDirectory(RecommendedDirectory(options.Host), SourceRecommended, "Recommended personal folder")
	seeds = append(seeds, options.AdditionalSeeds...)

	seeds = deduplicateSeeds(seeds, options.Host.GOOS)
	if len(seeds) > discoverySeedLimit {
		details = append(details, Detail{
			Path: "Discovery catalogue", Source: SourceCommon,
			Reason: fmt.Sprintf("bounded search stopped after %d unique roots", discoverySeedLimit),
		})
		seeds = seeds[:discoverySeedLimit]
	}

	return seeds, details
}

func commonSeeds(host HostProfile) []Seed {
	var seeds []Seed
	add := func(path, label string) {
		if strings.TrimSpace(path) != "" {
			seeds = append(seeds, Seed{Path: path, Source: SourceCommon, Label: label})
		}
	}
	switch host.GOOS {
	case goosWindows:
		programFilesX86 := host.Environment["ProgramFiles(x86)"]
		if programFilesX86 == "" {
			programFilesX86 = host.Environment["ProgramFiles"]
		}
		systemDrive := host.Environment["SystemDrive"]
		if systemDrive == "" {
			systemDrive = "C:"
		}
		if filepath.IsAbs(programFilesX86) {
			add(filepath.Join(programFilesX86, "GOG Galaxy", "Games"), "GOG Galaxy default")
		}
		add(
			filepath.Join(systemDrive+string(filepath.Separator), "GOG Games"),
			"GOG offline installer",
		)
		add(filepath.Join(host.HomeDirectory, "Games", "Heroic", "Ultima 5"), "Heroic default")
	case goosDarwin:
		for _, root := range []string{"/Applications", filepath.Join(host.HomeDirectory, "Applications")} {
			add(filepath.Join(root, "Ultima V™.app"), "GOG macOS application")
			add(filepath.Join(root, "Ultima V.app"), "GOG macOS application")
		}
	case goosLinux:
		add(filepath.Join(host.HomeDirectory, "Games", "Heroic", "Ultima 5"), "Heroic default")
		add(filepath.Join(host.HomeDirectory, "GOG Games"), "Manual GOG copy")
	}

	return seeds
}

func boundedLayouts(root, goos string) []string {
	if strings.TrimSpace(root) == "" {
		return nil
	}
	layouts := []string{
		root,
		filepath.Join(root, "UltimaV"),
		filepath.Join(root, "Ultima5"),
		filepath.Join(root, "Ultima 5"),
		filepath.Join(root, "Ultima V"),
		filepath.Join(root, "Ultima 4-5-6", "Ultima 5"),
		filepath.Join(root, "Ultima 4+5+6", "Ultima 5"),
	}
	if goos == goosDarwin {
		layouts = append(layouts,
			filepath.Join(root, "Contents", "Resources", "game"),
			filepath.Join(root, "Ultima V™.app", "Contents", "Resources", "game"),
			filepath.Join(root, "Ultima V.app", "Contents", "Resources", "game"),
			filepath.Join(
				root, "Contents", "Resources", "game", "Ultima 5.app", "Contents", "Resources",
				"Ultima 5.boxer", "C ULTIMA5.harddisk",
			),
		)
	}

	return slices.Compact(layouts)
}

func heroicSeeds(host HostProfile) ([]Seed, []Detail) {
	metadataPaths := heroicMetadataPaths(host)
	type heroicEntry struct {
		AppName     string `json:"appName"`
		InstallPath string `json:"install_path"` //nolint:tagliatelle // Heroic's persisted wire key uses snake case.
	}
	type heroicStore struct {
		Installed []heroicEntry `json:"installed"`
	}
	var seeds []Seed
	var details []Detail
	for _, metadataPath := range metadataPaths {
		contents, err := readBoundedMetadata(metadataPath, heroicMetadataMaximumBytes)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				details = append(details, Detail{Path: metadataPath, Source: SourceHeroic, Reason: err.Error()})
			}

			continue
		}
		var store heroicStore
		if err := json.Unmarshal(contents, &store); err != nil {
			details = append(details, Detail{Path: metadataPath, Source: SourceHeroic, Reason: err.Error()})

			continue
		}
		entries := store.Installed
		if len(entries) > heroicInstalledEntryLimit {
			details = append(details, Detail{
				Path: metadataPath, Source: SourceHeroic,
				Reason: fmt.Sprintf("metadata entry limit reached after %d installations", heroicInstalledEntryLimit),
			})
			entries = entries[:heroicInstalledEntryLimit]
		}
		for _, entry := range entries {
			if entry.AppName == GOGGameID && filepath.IsAbs(entry.InstallPath) {
				seeds = append(seeds, Seed{Path: entry.InstallPath, Source: SourceHeroic, Label: "Heroic GOG installation"})
			}
		}
	}

	return seeds, details
}

func heroicMetadataPaths(host HostProfile) []string {
	var metadataPaths []string
	switch host.GOOS {
	case goosWindows:
		if appData := host.Environment["APPDATA"]; filepath.IsAbs(appData) {
			metadataPaths = append(metadataPaths, filepath.Join(appData, "heroic", "gog_store", "installed.json"))
		}
	case goosDarwin:
		metadataPaths = append(metadataPaths,
			filepath.Join(host.HomeDirectory, "Library", "Application Support", "heroic", "gog_store", "installed.json"),
		)
	case goosLinux:
		xdgConfig := host.Environment["XDG_CONFIG_HOME"]
		if !filepath.IsAbs(xdgConfig) {
			xdgConfig = filepath.Join(host.HomeDirectory, ".config")
		}
		metadataPaths = append(metadataPaths,
			filepath.Join(xdgConfig, "heroic", "gog_store", "installed.json"),
			filepath.Join(
				host.HomeDirectory, ".var", "app", "com.heroicgameslauncher.hgl", "config", "heroic",
				"gog_store", "installed.json",
			),
		)
	}

	return metadataPaths
}

func readBoundedMetadata(path string, maximumBytes int64) ([]byte, error) {
	//nolint:gosec // Paths are fixed host metadata locations.
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening metadata %s: %w", path, err)
	}
	contents, err := io.ReadAll(io.LimitReader(file, maximumBytes+1))
	closeErr := file.Close()
	if err != nil {
		return nil, fmt.Errorf("reading metadata: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("closing metadata: %w", closeErr)
	}
	if int64(len(contents)) > maximumBytes {
		return nil, fmt.Errorf("%w: %d bytes", errMetadataTooLarge, maximumBytes)
	}

	return contents, nil
}

func deduplicateSeeds(seeds []Seed, goos string) []Seed {
	seen := make(map[string]struct{}, len(seeds))
	result := make([]Seed, 0, len(seeds))
	for _, seed := range seeds {
		identity := pathIdentity(seed.Path, goos)
		if identity == "" {
			continue
		}
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		result = append(result, seed)
	}

	return result
}

// WindowsGOGSeeds converts verified registry values into discovery seeds.
func WindowsGOGSeeds(records []WindowsGOGRegistryRecord) []Seed {
	var seeds []Seed
	for _, record := range records {
		if strings.TrimSpace(record.GamePath) != "" {
			seeds = append(seeds, Seed{
				Path: record.GamePath, Source: SourceGOG, Label: "GOG installation record",
			})
		}
		if record.UninstallPublisher == "GOG.com" && strings.TrimSpace(record.UninstallPath) != "" {
			seeds = append(seeds, Seed{
				Path: record.UninstallPath, Source: SourceGOG, Label: "GOG uninstall record",
			})
		}
	}

	return seeds
}

func pathIdentity(path, goos string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if goos == goosWindows {
		path = strings.ToLower(path)
	}

	return path
}

func canonicalIdentity(path, goos string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}

	return pathIdentity(path, goos)
}

func environmentMap() map[string]string {
	result := make(map[string]string)
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if found {
			result[key] = value
		}
	}

	return result
}

// StableCandidatePaths returns sorted paths for diagnostics and tests.
func StableCandidatePaths(report Report) []string {
	paths := make([]string, len(report.Candidates))
	for index, candidate := range report.Candidates {
		paths[index] = candidate.Path
	}
	sort.Strings(paths)

	return paths
}
