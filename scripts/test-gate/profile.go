package main

import (
	"fmt"
	"io"
	"maps"
	"runtime"
	"slices"
	"strings"
)

const (
	profileConstrained = "constrained"
	profileThroughput  = "throughput"
	osWindows          = "windows"
)

func profileEnvironment(profile string, environment []string) ([]string, error) {
	return profileEnvironmentForOS(profile, runtime.GOOS, environment)
}

func profileEnvironmentForOS(profile, goos string, environment []string) ([]string, error) {
	cpus, memory := "2", "256MiB"
	switch profile {
	case profileConstrained:
	case profileThroughput:
		cpus, memory = "4", "2GiB"
	default:
		return nil, fmt.Errorf("%w: unknown profile %q; use constrained or throughput", errGate, profile)
	}
	for _, setting := range []string{"GOMAXPROCS=" + cpus, "GOMEMLIMIT=" + memory} {
		key, wanted, _ := strings.Cut(setting, "=")
		for _, inherited := range slices.Backward(environment) {
			name, value, _ := strings.Cut(inherited, "=")
			matches := name == key || (goos == osWindows && strings.EqualFold(name, key))
			if !matches {
				continue
			}
			if value != wanted {
				return nil, fmt.Errorf("%w: profile %s requires %s, inherited %s=%q; unset it or select matching settings",
					errGate, profile, setting, key, value)
			}

			break
		}
	}

	return append(slices.Clone(environment), "GOMAXPROCS="+cpus, "GOMEMLIMIT="+memory), nil
}

func partitionPackages(reader io.Reader, expected map[string]*packageResult) ([]map[string]*packageResult, error) {
	metadata, err := readPackageMetadata(reader)
	if err != nil {
		return nil, err
	}
	lanes := []map[string]*packageResult{{}, {}}
	for _, name := range slices.Sorted(maps.Keys(expected)) {
		result := expected[name]
		base, found := metadata[name]
		if !found || (len(base.TestGoFiles)+len(base.XTestGoFiles) > 0) != result.hasTests {
			return nil, fmt.Errorf("%w: missing or changed test-aware metadata for %s", errGate, name)
		}
		closures := []packageMetadata{base}
		if result.hasTests {
			binary, exists := metadata[name+".test"]
			if !exists || binary.Name != "main" || len(binary.Deps) == 0 {
				return nil, fmt.Errorf("%w: missing test dependency closure for %s", errGate, name)
			}
			closures = append(closures, binary)
		}
		lane := 0
		for _, closure := range closures {
			if len(closure.Imports) > 0 && len(closure.Deps) == 0 {
				return nil, fmt.Errorf("%w: missing dependency closure for %s", errGate, closure.ImportPath)
			}
			// Go's Deps already resolves transitive internal/external-test imports
			// and import remapping. Synthetic test binaries never become test targets.
			// Include audio and every Ebitengine subpackage conservatively: this
			// classifies linked dependencies, not each assertion's native needs.
			for _, dependency := range append(slices.Clone(closure.Deps), closure.ImportPath) {
				path, _, _ := strings.Cut(dependency, " [")
				const ebiten = "github.com/hajimehoshi/ebiten/v2"
				if path == ebiten || strings.HasPrefix(path, ebiten+"/") {
					lane = 1
				}
			}
		}
		lanes[lane][name] = result
	}

	return lanes, nil
}
