package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixtureGraphicsInventory = fixtureInventory + `
{"ImportPath":"graphics","XTestGoFiles":["graphics_test.go"]}`

const fixtureLogicDependencies = fixtureInventory + `
{"ImportPath":"core.test","Name":"main","Deps":["core [core.test]","testing"]}`

const fixtureTestDependencies = fixtureGraphicsInventory + `
{"ImportPath":"core.test","Name":"main","Deps":["core [core.test]","testing"]}
{"ImportPath":"graphics.test","Name":"main","Deps":["graphics_test [graphics.test]","helper","github.com/hajimehoshi/ebiten/v2","testing"]}`

const fixtureGraphicsPass = `{"Action":"start","Package":"graphics"}
{"Action":"run","Package":"graphics","Test":"TestDraw"}
{"Action":"run","Package":"graphics","Test":"TestDraw/child"}
{"Action":"pause","Package":"graphics","Test":"TestDraw/child"}
{"Action":"cont","Package":"graphics","Test":"TestDraw/child"}
{"Action":"pass","Package":"graphics","Test":"TestDraw/child"}
{"Action":"pass","Package":"graphics","Test":"TestDraw"}
{"Action":"pass","Package":"graphics"}`

func setProfileEnvironment(t *testing.T, profile string) {
	t.Helper()
	environment, err := profileEnvironment(profile, nil)
	require.NoError(t, err)
	for _, entry := range environment {
		key, value, _ := strings.Cut(entry, "=")
		t.Setenv(key, value)
	}
}

func TestProfileEnvironment(t *testing.T) {
	for _, test := range []struct {
		profile, cpus, memory string
	}{
		{profileConstrained, "2", "256MiB"},
		{profileThroughput, "4", "2GiB"},
	} {
		t.Run(test.profile, func(t *testing.T) {
			original := []string{"GOFLAGS=-tags='one two' -mod=readonly -p=99", "ULTIMA5_DATA_PATH=/private/original",
				"GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local", "DISPLAY=:17"}
			environment, err := profileEnvironment(test.profile, original)
			require.NoError(t, err)
			require.Equal(t, original, environment[:len(original)])
			require.Equal(t, []string{"GOMAXPROCS=" + test.cpus, "GOMEMLIMIT=" + test.memory}, environment[len(original):])
			_, err = profileEnvironment(test.profile, environment)
			require.NoError(t, err, "matching explicit settings remain valid")
			for _, conflict := range []string{"GOMAXPROCS=0", "GOMAXPROCS=99", "GOMEMLIMIT=512MiB", "GOMEMLIMIT=off", "GOMEMLIMIT="} {
				_, err = profileEnvironment(test.profile, append(slices.Clone(environment), conflict))
				require.ErrorContains(t, err, "requires")
			}
			_, err = profileEnvironment(test.profile, append([]string{"GOMAXPROCS=1"}, environment...))
			require.NoError(t, err, "the last inherited environment entry wins, as with exec.Cmd")
		})
	}
}

func TestProfileEnvironmentUsesPlatformKeySemantics(t *testing.T) {
	for _, test := range []struct {
		name, goos  string
		environment []string
		wantError   string
	}{
		{name: "Windows lowercase memory", goos: "windows", environment: []string{"gomemlimit=512MiB"}, wantError: "requires GOMEMLIMIT=2GiB"},
		{name: "Windows mixed CPU", goos: "windows", environment: []string{"GoMaxProcs=1"}, wantError: "requires GOMAXPROCS=4"},
		{name: "Windows matching case variants", goos: "windows", environment: []string{"goMAXprocs=4", "goMEMlimit=2GiB"}},
		{name: "Windows later lowercase conflict", goos: "windows", environment: []string{"GOMEMLIMIT=2GiB", "gomemlimit=512MiB"}, wantError: "requires GOMEMLIMIT=2GiB"},
		{name: "Windows later uppercase wins", goos: "windows", environment: []string{"gomemlimit=512MiB", "GOMEMLIMIT=2GiB"}},
		{name: "Windows later lowercase wins", goos: "windows", environment: []string{"GOMEMLIMIT=512MiB", "gomemlimit=2GiB"}},
		{name: "Unix lowercase is unrelated", goos: "darwin", environment: []string{"gomaxprocs=1", "gomemlimit=512MiB"}},
		{name: "Unix lowercase cannot shadow uppercase", goos: "linux", environment: []string{"GOMEMLIMIT=2GiB", "gomemlimit=512MiB"}},
		{name: "Unix uppercase conflict remains", goos: "linux", environment: []string{"GOMEMLIMIT=512MiB", "gomemlimit=2GiB"}, wantError: "requires GOMEMLIMIT=2GiB"},
	} {
		t.Run(test.name, func(t *testing.T) {
			environment, err := profileEnvironmentForOS(profileThroughput, test.goos, test.environment)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.environment, environment[:len(test.environment)])
			require.Equal(t, []string{"GOMAXPROCS=4", "GOMEMLIMIT=2GiB"}, environment[len(test.environment):])
		})
	}
}

func TestPartitionPackagesUsesProductionAndResolvedTestClosures(t *testing.T) {
	const inventory = `{"ImportPath":"direct","Imports":["github.com/hajimehoshi/ebiten/v2"],"Deps":["github.com/hajimehoshi/ebiten/v2"]}
{"ImportPath":"transitive","Imports":["helper"],"Deps":["helper","github.com/hajimehoshi/ebiten/v2/vector"]}
{"ImportPath":"internal","TestGoFiles":["internal_test.go"]}
{"ImportPath":"external","XTestGoFiles":["external_test.go"]}
{"ImportPath":"audio","Deps":["github.com/hajimehoshi/ebiten/v2/audio"]}
{"ImportPath":"indexes","Deps":["github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"]}
{"ImportPath":"similar","Deps":["github.com/hajimehoshi/ebiten/v20"]}
{"ImportPath":"assets"}`
	expected, err := readPackages(strings.NewReader(inventory))
	require.NoError(t, err)
	metadata := inventory + `
{"ImportPath":"internal.test","Name":"main","Deps":["internal [internal.test]","testing","github.com/hajimehoshi/ebiten/v2 [internal.test]"]}
{"ImportPath":"external.test","Name":"main","Deps":["external_test [external.test]","testing","github.com/hajimehoshi/ebiten/v2"]}
{"ImportPath":"external_test [external.test]","Imports":["helper"],"Deps":["helper","github.com/hajimehoshi/ebiten/v2"]}`
	lanes, err := partitionPackages(strings.NewReader(metadata), expected)
	require.NoError(t, err)
	require.Equal(t, []string{"assets", "indexes", "similar"}, slices.Sorted(maps.Keys(lanes[0])))
	require.Equal(t, []string{"audio", "direct", "external", "internal", "transitive"}, slices.Sorted(maps.Keys(lanes[1])))
	require.Equal(t, len(expected), len(lanes[0])+len(lanes[1]))
	for _, lane := range lanes {
		for name, result := range lane {
			require.Same(t, expected[name], result, "lane records belong to the original inventory")
		}
	}
}

func TestPartitionPackagesFailsClosed(t *testing.T) {
	expected, err := readPackages(strings.NewReader(fixtureInventory))
	require.NoError(t, err)
	for _, metadata := range []string{
		"", "{", fixtureInventory,
		fixtureInventory + `{"ImportPath":"core.test","Name":"main"}`,
		fixtureInventory + `{"ImportPath":"core.test","Name":"notmain","Deps":["testing"]}`,
		strings.Replace(fixtureTestDependencies, `"assets"`, `"missing"`, 1),
		strings.Replace(fixtureTestDependencies, `"core_test.go"`, ``, 1),
		strings.Replace(fixtureTestDependencies, `"Name":"main"`, `"Name":"main","Incomplete":true`, 1),
		fixtureTestDependencies + `{"ImportPath":"broken","Error":{"Err":"syntax failure"}}`,
		fixtureTestDependencies + `{"ImportPath":"broken","DepsErrors":[{"Err":"missing import"}]}`,
		fixtureTestDependencies + `{"ImportPath":"core"}`,
		strings.Replace(fixtureTestDependencies, `{"ImportPath":"assets"}`, `{"ImportPath":"assets","Imports":["helper"]}`, 1),
	} {
		_, err := partitionPackages(strings.NewReader(metadata), expected)
		require.Error(t, err, "metadata: %s", metadata)
	}
}

func TestThroughputRunsDisjointLanesSequentially(t *testing.T) {
	setProfileEnvironment(t, profileThroughput)
	t.Setenv("ULTIMA5_DATA_PATH", t.TempDir())
	const goflags = "-tags='one two' -mod=readonly -p=99 -parallel=99 -run=Nothing -skip=. -count=50 -list=."
	t.Setenv("GOFLAGS", goflags)
	var calls [][]string
	var active atomic.Bool
	var firstFinished atomic.Bool
	fakeGo := func(_ context.Context, arguments, environment []string, stdout, _ io.Writer) error {
		assert.False(t, active.Swap(true), "Go invocations must not overlap")
		defer active.Store(false)
		calls = append(calls, slices.Clone(arguments))
		assert.Contains(t, environment, "GOFLAGS="+goflags)
		assert.Contains(t, environment, "GOMAXPROCS=4")
		assert.Contains(t, environment, "GOMEMLIMIT=2GiB")
		assert.Equal(t, "ULTIMA5_REQUIRE_DATA=1", environment[len(environment)-1])
		output := fixtureGraphicsInventory
		switch len(calls) {
		case 2:
			output = fixtureTestDependencies
		case 3:
			output = fixturePass + "\n"
			defer firstFinished.Store(true)
		case 4:
			assert.True(t, firstFinished.Load(), "first test command must have returned")
			output = fixtureGraphicsPass
		}
		_, err := io.WriteString(stdout, output)
		return err
	}
	var stdout, stderr bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-profile=throughput", "-strict", "-race", "-short"}, &stdout, &stderr, fakeGo))
	require.Equal(t, [][]string{
		{"list", "-json", "-p=1", "-race=true", "./..."},
		{"list", "-json", "-test", "-p=1", "-race=true", "./..."},
		{"test", "-json", "-count=1", "-run=.", "-skip=^$", "-list=", "-bench=", "-fuzz=", "-race=true", "-short=true", "-p=2", "-parallel=2", "assets", "core"},
		{"test", "-json", "-count=1", "-run=.", "-skip=^$", "-list=", "-bench=", "-fuzz=", "-race=true", "-short=true", "-p=1", "-parallel=1", "graphics"},
	}, calls)
	require.Equal(t, fixturePass+"\n"+fixtureGraphicsPass, stdout.String())
	require.Contains(t, stderr.String(), "packages=3/3 tests/subtests=3 skipped tests=0")
	require.Equal(t, 1, strings.Count(stderr.String(), "Test gate totals:"))
}

func TestThroughputSkipsEmptyLanes(t *testing.T) {
	setProfileEnvironment(t, profileThroughput)
	for _, graphics := range []bool{false, true} {
		t.Run(map[bool]string{false: "no graphics", true: "only graphics"}[graphics], func(t *testing.T) {
			metadata := fixtureInventory + `{"ImportPath":"core.test","Name":"main","Deps":["testing"]}`
			parallelism := "2"
			if graphics {
				metadata = strings.Replace(metadata, `{"ImportPath":"assets"}`, `{"ImportPath":"assets","Deps":["github.com/hajimehoshi/ebiten/v2"]}`, 1)
				metadata = strings.ReplaceAll(metadata, `["testing"]`, `["testing","github.com/hajimehoshi/ebiten/v2"]`)
				parallelism = "1"
			}
			calls := 0
			fakeGo := func(_ context.Context, arguments, _ []string, stdout, _ io.Writer) error {
				calls++
				output := fixtureInventory
				if calls == 2 {
					output = metadata
				}
				if arguments[0] == "test" {
					assert.Contains(t, arguments, "-p="+parallelism)
					assert.Contains(t, arguments, "-parallel="+parallelism)
					assert.Equal(t, []string{"assets", "core"}, arguments[len(arguments)-2:])
					output = fixturePass
				}
				_, err := io.WriteString(stdout, output)
				return err
			}
			require.NoError(t, run(t.Context(), []string{"-profile=throughput"}, io.Discard, io.Discard, fakeGo))
			require.Equal(t, 3, calls)
		})
	}
}

func TestThroughputFailureKeepsRemainingInventoryIncomplete(t *testing.T) {
	setProfileEnvironment(t, profileThroughput)
	t.Setenv("ULTIMA5_DATA_PATH", t.TempDir())
	processError := errors.New("child failed")
	for _, test := range []struct {
		name, output, want string
		err                error
	}{
		{"nonzero exit", fixturePass, "child failed", processError},
		{"malformed output", "not JSON\n" + fixturePass, "decoding go test JSON", nil},
		{"missing assertion", `{"Action":"pass","Package":"core"}{"Action":"skip","Package":"assets"}`, "no test events", nil},
		{"unexpected lane", fixturePass + fixtureGraphicsPass, "unexpected package", nil},
		{"strict skip", strings.Replace(fixturePass, `"pass","Package":"core","Test"`, `"skip","Package":"core","Test"`, 1), `terminal="skip"`, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			fakeGo := func(_ context.Context, arguments, _ []string, stdout, stderr io.Writer) error {
				calls++
				if arguments[0] == "list" {
					output := fixtureGraphicsInventory
					if calls == 2 {
						output = fixtureTestDependencies
					}
					_, err := io.WriteString(stdout, output)
					return err
				}
				_, outErr := io.WriteString(stdout, test.output)
				_, diagnosticErr := io.WriteString(stderr, "child diagnostic\n")
				return errors.Join(test.err, outErr, diagnosticErr)
			}
			var stdout, stderr bytes.Buffer
			err := run(t.Context(), []string{"-profile=throughput", "-strict"}, &stdout, &stderr, fakeGo)
			require.ErrorContains(t, err, test.want)
			if test.err != nil {
				require.ErrorIs(t, err, test.err)
			}
			require.Equal(t, 3, calls, "do not start another lane after failure")
			require.Equal(t, test.output, stdout.String(), "drain all child output even after invalid JSON")
			require.Contains(t, stderr.String(), "child diagnostic")
			require.Contains(t, stderr.String(), "graphics: MISSING")
			require.Contains(t, stderr.String(), "Test gate totals:")
			require.NotContains(t, stderr.String(), "packages=3/3")
		})
	}
}

func TestThroughputMetadataFailurePreventsTests(t *testing.T) {
	setProfileEnvironment(t, profileThroughput)
	metadataError := errors.New("test dependency listing failed")
	for _, test := range []struct {
		output string
		err    error
	}{
		{}, {output: "{"}, {output: fixtureGraphicsInventory}, {output: fixtureTestDependencies, err: metadataError},
	} {
		calls := 0
		fakeGo := func(_ context.Context, arguments, _ []string, stdout, _ io.Writer) error {
			calls++
			assert.Equal(t, "list", arguments[0])
			if calls == 1 {
				_, err := io.WriteString(stdout, fixtureGraphicsInventory)
				return err
			}
			_, err := io.WriteString(stdout, test.output)
			return errors.Join(test.err, err)
		}
		err := run(t.Context(), []string{"-profile=throughput"}, io.Discard, io.Discard, fakeGo)
		require.Error(t, err)
		if test.err != nil {
			require.ErrorIs(t, err, test.err)
		}
		require.Equal(t, 2, calls)
	}
}

func TestThroughputCancellationDoesNotStartNextLane(t *testing.T) {
	setProfileEnvironment(t, profileThroughput)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	calls := 0
	fakeGo := func(_ context.Context, arguments, _ []string, stdout, _ io.Writer) error {
		calls++
		output := fixtureGraphicsInventory
		if calls == 2 {
			output = fixtureTestDependencies
		}
		if arguments[0] == "test" {
			output = fixturePass
			defer cancel()
		}
		_, err := io.WriteString(stdout, output)
		return err
	}
	var stderr bytes.Buffer
	err := run(ctx, []string{"-profile=throughput"}, io.Discard, &stderr, fakeGo)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 3, calls)
	require.Contains(t, stderr.String(), "graphics: MISSING")
}

func TestProfileConflictPreventsGoCommands(t *testing.T) {
	setProfileEnvironment(t, profileConstrained)
	err := run(t.Context(), []string{"-profile=throughput"}, io.Discard, io.Discard, nil)
	require.ErrorContains(t, err, "requires GOMAXPROCS=4")
}
