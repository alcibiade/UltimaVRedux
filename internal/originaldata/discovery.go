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
	errEmptyDirectory     = errors.New("enter an Ultima V folder")
	errDataFileSelected   = errors.New("choose the folder containing DATA.OVL, not a file")
	errRequiredDataAbsent = errors.New("required game files are missing")
	errMetadataTooLarge   = errors.New("provider metadata exceeds the discovery size limit")
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
	SourcePortable    Source = "Portable copy"
	SourceRecommended Source = "Recommended personal folder"
)

// HostProfile contains the host paths used to produce a finite discovery catalogue.
type HostProfile struct {
	GOOS           string
	HomeDirectory  string
	DocumentsDir   string
	ExecutablePath string
	Environment    map[string]string
	ProviderSeeds  []Seed
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
	executable, err := os.Executable()
	if err != nil {
		return HostProfile{}, fmt.Errorf("locating executable: %w", err)
	}
	documents, err := platformDocumentsDirectory(home)
	if err != nil {
		documents = filepath.Join(home, "Documents")
	}
	host := HostProfile{
		GOOS:           runtime.GOOS,
		HomeDirectory:  home,
		DocumentsDir:   documents,
		ExecutablePath: executable,
		Environment:    environmentMap(),
		ProviderSeeds:  platformProviderSeeds(),
	}

	return host, nil
}

// RecommendedDirectory returns the simple personal folder shown when managed
// installation discovery does not find a usable copy.
func RecommendedDirectory(host HostProfile) string {
	documents := strings.TrimSpace(host.DocumentsDir)
	if documents == "" {
		documents = filepath.Join(host.HomeDirectory, "Documents")
	}

	return filepath.Join(documents, "UltimaV")
}

// ExpandUserPath expands only the portable home shorthand. It deliberately
// does not evaluate shell variables, commands, or escape sequences.
func ExpandUserPath(value, homeDirectory string) string {
	value = strings.TrimSpace(value)
	if value == "~" {
		return homeDirectory
	}
	if strings.HasPrefix(value, "~"+string(filepath.Separator)) {
		return filepath.Join(homeDirectory, strings.TrimPrefix(value, "~"+string(filepath.Separator)))
	}

	return value
}

// Discover completes the finite catalogue before returning stable verified results.
//
//nolint:gocognit,funlen // One bounded loop keeps path identity, diagnostics, and stable ordering auditable.
func Discover(ctx context.Context, options DiscoveryOptions) Report {
	report := Report{
		Candidates: nil, Details: nil, CheckedRoots: nil,
		RecommendedDirectory: RecommendedDirectory(options.Host), Cancelled: false,
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

// ResolveDirectory validates a manually entered data directory or one of the
// finite known layouts directly beneath it. It never performs a recursive scan.
func ResolveDirectory(ctx context.Context, enteredPath, goos string, validator ValidateFunc) ([]string, error) {
	enteredPath = strings.TrimSpace(enteredPath)
	if enteredPath == "" {
		return nil, errEmptyDirectory
	}
	absolute, err := filepath.Abs(enteredPath)
	if err != nil {
		return nil, fmt.Errorf("resolving selected folder: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return nil, fmt.Errorf("checking selected folder: %w", err)
	}
	if !info.IsDir() {
		return nil, errDataFileSelected
	}
	if validator == nil {
		validator = ValidateInstallation
	}
	var matches []string
	var validationErrors []error
	seenMatches := make(map[string]struct{})
	for _, candidate := range boundedLayouts(absolute, goos) {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("cancelling folder resolution: %w", err)
		}
		candidateInfo, statErr := os.Stat(candidate)
		if statErr != nil || !candidateInfo.IsDir() {
			continue
		}
		if validationErr := validator(ctx, candidate); validationErr != nil {
			validationErrors = append(validationErrors, validationErr)

			continue
		}
		identity := canonicalIdentity(candidate, goos)
		if _, exists := seenMatches[identity]; exists {
			continue
		}
		seenMatches[identity] = struct{}{}
		matches = append(matches, candidate)
	}
	if len(matches) == 0 {
		if len(validationErrors) > 0 {
			return nil, fmt.Errorf("these game files are not usable: %w", validationErrors[0])
		}

		return nil, errRequiredDataAbsent
	}
	sort.Strings(matches)

	return slices.Compact(matches), nil
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
	seeds = append(seeds, options.Host.ProviderSeeds...)
	heroic, details := heroicSeeds(options.Host)
	seeds = append(seeds, heroic...)
	seeds = append(seeds, commonSeeds(options.Host)...)
	if options.Host.ExecutablePath != "" {
		executableDirectory := filepath.Dir(options.Host.ExecutablePath)
		for _, name := range []string{"OLD", "UltimaV", "Ultima 5", "Ultima V"} {
			appendDirectory(filepath.Join(executableDirectory, name), SourcePortable, "Local copy beside Redux")
		}
	}
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
	add := func(path string, source Source, label string) {
		if strings.TrimSpace(path) != "" {
			seeds = append(seeds, Seed{Path: path, Source: source, Label: label})
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
		add(filepath.Join(programFilesX86, "GOG Galaxy", "Games"), SourceCommon, "GOG Galaxy default")
		add(
			filepath.Join(systemDrive+string(filepath.Separator), "GOG Games"),
			SourceCommon,
			"GOG offline installer",
		)
		add(filepath.Join(host.HomeDirectory, "Games", "Heroic", "Ultima 5"), SourceCommon, "Heroic default")
	case goosDarwin:
		for _, root := range []string{"/Applications", filepath.Join(host.HomeDirectory, "Applications")} {
			add(filepath.Join(root, "Ultima V™.app"), SourceCommon, "GOG macOS application")
			add(filepath.Join(root, "Ultima V.app"), SourceCommon, "GOG macOS application")
		}
	case goosLinux:
		add(filepath.Join(host.HomeDirectory, "Games", "Heroic", "Ultima 5"), SourceCommon, "Heroic default")
		add(filepath.Join(host.HomeDirectory, "GOG Games"), SourceCommon, "Manual GOG copy")
	}
	for _, name := range []string{"UltimaV", "Ultima 5", "Ultima V"} {
		add(filepath.Join(host.DocumentsDir, name), SourceRecommended, "Recommended personal folder")
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
			if entry.AppName == GOGGameID && strings.TrimSpace(entry.InstallPath) != "" {
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
		metadataPaths = append(metadataPaths,
			filepath.Join(host.Environment["APPDATA"], "heroic", "gog_store", "installed.json"),
		)
	case goosDarwin:
		metadataPaths = append(metadataPaths,
			filepath.Join(host.HomeDirectory, "Library", "Application Support", "heroic", "gog_store", "installed.json"),
		)
	case goosLinux:
		xdgConfig := host.Environment["XDG_CONFIG_HOME"]
		if xdgConfig == "" {
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
