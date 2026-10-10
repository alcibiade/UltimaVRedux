package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunUsesAllPackagesUncachedAndControlsLaneEnvironment(t *testing.T) {
	setProfileEnvironment(t, profileConstrained)
	for _, strict := range []bool{false, true} {
		t.Run(map[bool]string{false: "public", true: "strict"}[strict], func(t *testing.T) {
			t.Setenv("ULTIMA5_REQUIRE_DATA", "1")
			t.Setenv("ULTIMA5_DATA_PATH", t.TempDir())
			t.Setenv("GOFLAGS", "-run=OnlyOneTest -skip=Core -count=100 -list=.")
			var calls [][]string
			fakeGo := func(_ context.Context, arguments, environment []string, stdout, _ io.Writer) error {
				calls = append(calls, arguments)
				wantMode := "ULTIMA5_REQUIRE_DATA=0"
				if strict {
					wantMode = "ULTIMA5_REQUIRE_DATA=1"
				}
				assert.Equal(t, wantMode, environment[len(environment)-1])
				output := fixturePass
				if arguments[0] == "list" {
					output = fixtureInventory
					if len(calls) == 2 {
						output = fixtureLogicDependencies
					}
				}
				_, err := io.WriteString(stdout, output)
				return err
			}
			arguments := []string{"-race", "-short"}
			if strict {
				arguments = append(arguments, "-strict")
			}
			var stdout, stderr bytes.Buffer
			require.NoError(t, run(t.Context(), arguments, &stdout, &stderr, fakeGo))
			require.Equal(t, [][]string{
				{"list", "-json", "-p=1", "-race=true", "./..."},
				{"list", "-json", "-test", "-p=1", "-race=true", "./..."},
				{"test", "-json", "-count=1", "-run=.", "-skip=^$", "-list=", "-bench=", "-fuzz=", "-race=true", "-short=true", "-p=1", "-parallel=1", "assets", "core"},
			}, calls)
			require.Equal(t, fixturePass, stdout.String(), "the original JSON stream must remain intact")
			require.Contains(t, stderr.String(), "packages=2/2")
			require.Contains(t, stderr.String(), "skipped tests=0")
		})
	}
}

func TestRunPreservesCommandFailuresAndDiagnostics(t *testing.T) {
	setProfileEnvironment(t, profileConstrained)
	processError := errors.New("go test exited with status 37")
	for _, test := range []struct {
		name, output string
	}{
		{name: "complete passing events cannot hide nonzero exit", output: fixturePass},
		{name: "no output build failure"},
		{name: "parse error still drains later diagnostics", output: "not JSON\n" + fixturePass},
	} {
		t.Run(test.name, func(t *testing.T) {
			fakeGo := func(_ context.Context, arguments, _ []string, stdout, stderr io.Writer) error {
				if arguments[0] == "list" {
					inventory := fixtureInventory
					if len(arguments) == 6 {
						inventory = fixtureLogicDependencies
					}
					_, err := io.WriteString(stdout, inventory)
					return err
				}
				_, outErr := io.WriteString(stdout, test.output)
				_, errErr := io.WriteString(stderr, "compiler/test diagnostic\n")
				return errors.Join(processError, outErr, errErr)
			}
			var stdout, stderr bytes.Buffer
			err := run(t.Context(), nil, &stdout, &stderr, fakeGo)
			require.ErrorIs(t, err, processError)
			require.Equal(t, test.output, stdout.String())
			require.Contains(t, stderr.String(), "compiler/test diagnostic")
			require.Contains(t, stderr.String(), "Test gate totals:")
		})
	}
}

func TestStrictCannotSilentlyLoseCoreAssertion(t *testing.T) {
	setProfileEnvironment(t, profileConstrained)
	t.Setenv("ULTIMA5_DATA_PATH", t.TempDir())
	fakeGo := func(_ context.Context, arguments, _ []string, stdout, _ io.Writer) error {
		output := strings.Replace(fixturePass, `"pass","Package":"core","Test"`, `"skip","Package":"core","Test"`, 1)
		if arguments[0] == "list" {
			output = fixtureInventory
			if len(arguments) == 6 {
				output = fixtureLogicDependencies
			}
		}
		_, err := io.WriteString(stdout, output)
		return err // Deliberately successful process and package exit.
	}
	var stdout, stderr bytes.Buffer
	err := run(t.Context(), []string{"-strict"}, &stdout, &stderr, fakeGo)
	require.ErrorContains(t, err, `core/TestMandatoryAssertion terminal="skip"`)
	require.Contains(t, stderr.String(), "SKIP core/TestMandatoryAssertion")
}

func TestStrictRequiresExplicitSourceWithoutFallback(t *testing.T) {
	t.Setenv("ULTIMA5_DATA_PATH", "")
	t.Setenv("ULTIMA5_DATA_DIR", "")
	notCalled := func(context.Context, []string, []string, io.Writer, io.Writer) error {
		t.Error("go must not execute before preflight passes")
		return nil
	}
	err := run(t.Context(), []string{"-strict"}, io.Discard, io.Discard, notCalled)
	require.ErrorContains(t, err, "-strict requires ULTIMA5_DATA_PATH or ULTIMA5_DATA_DIR")
	t.Setenv("ULTIMA5_DATA_DIR", t.TempDir())
	t.Setenv("ULTIMA5_DATA_PATH", filepath.Join(t.TempDir(), "missing"))
	err = run(t.Context(), []string{"-strict"}, io.Discard, io.Discard, notCalled)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestListFailurePreventsTestExecution(t *testing.T) {
	setProfileEnvironment(t, profileConstrained)
	failure := errors.New("go list failed")
	fakeGo := func(_ context.Context, arguments, _ []string, _, _ io.Writer) error {
		require.Equal(t, "list", arguments[0])
		return failure
	}
	err := run(t.Context(), nil, io.Discard, io.Discard, fakeGo)
	require.ErrorIs(t, err, failure)
}

func TestRejectsPackageSelectionAndUnknownFlags(t *testing.T) {
	for _, arguments := range [][]string{{"./internal/config"}, {"-run", "OnlyOneTest"}, {"-typo"}, {"-profile=unknown"}, {"-suite=unknown"}, {"-isolate-graphics-tests"}} {
		require.Error(t, run(t.Context(), arguments, io.Discard, io.Discard, nil))
	}
}

func TestRunGoPreservesExitError(t *testing.T) {
	var stderr bytes.Buffer
	err := runGo(t.Context(), []string{"test-gate-invalid-command"}, os.Environ(), io.Discard, &stderr)
	var exitError *exec.ExitError
	require.ErrorAs(t, err, &exitError)
	require.NotZero(t, exitError.ExitCode())
	require.Contains(t, stderr.String(), "unknown command")
}

func TestGateAgainstIsolatedGoModule(t *testing.T) {
	setProfileEnvironment(t, profileConstrained)
	directory := t.TempDir()
	t.Chdir(directory)
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "-run=Nothing -count=5 -list=.")
	t.Setenv("ULTIMA5_REQUIRE_DATA", "1")
	t.Setenv("ULTIMA5_DATA_PATH", directory)
	require.NoError(t, os.WriteFile("go.mod", []byte("module gatefixture\n\ngo 1.24.0\n"), 0o600))
	require.NoError(t, os.WriteFile("core_test.go", []byte(`package gatefixture
import "testing"
func TestCoreAssertion(t *testing.T) { t.Skip("synthetic missing data") }
`), 0o600))
	require.NoError(t, os.Mkdir("assets", 0o700))
	require.NoError(t, os.WriteFile(filepath.Join("assets", "assets.go"), []byte("package assets\n"), 0o600))
	var stdout, stderr bytes.Buffer
	require.NoError(t, run(t.Context(), nil, &stdout, &stderr, runGo))
	require.Contains(t, stderr.String(), "packages=2/2")
	require.Contains(t, stderr.String(), "skipped tests=1")
	require.Contains(t, stderr.String(), "synthetic missing data")

	stdout.Reset()
	stderr.Reset()
	err := run(t.Context(), []string{"-strict"}, &stdout, &stderr, runGo)
	require.ErrorContains(t, err, `gatefixture/TestCoreAssertion terminal="skip"`)
	require.Contains(t, stderr.String(), "SKIP gatefixture/TestCoreAssertion")
	require.Contains(t, stderr.String(), "gatefixture/assets: skip (test events: 0)")
}

func TestGatePreservesPersistedBuildTags(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)
	t.Setenv("GOWORK", "off")
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")
	// Go consults GOENV when GOFLAGS is unset or empty. Never write the user's file.
	t.Setenv("GOFLAGS", "")
	goenv := filepath.Join(directory, "goenv")
	t.Setenv("GOENV", goenv)
	require.NoError(t, os.WriteFile(goenv,
		[]byte("GOFLAGS=-tags=profilefixture -mod=readonly -p=9 -parallel=9 -run=Nothing -skip=. -count=9 -list=. -timeout=30s\n"), 0o600))
	for name, source := range map[string]string{
		"go.mod": "module gatefixture\n\ngo 1.24.0\n",
		"root_test.go": `package gatefixture
import "testing"
func TestAlways(t *testing.T) {}
`,
		"tagged/tagged_test.go": `//go:build profilefixture

package tagged
import "testing"
func TestPersistedBuildTag(t *testing.T) { t.Log("persisted build tag assertion executed") }
`,
		"assets/assets.go": "package assets\n",
	} {
		require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o700))
		require.NoError(t, os.WriteFile(name, []byte(source), 0o600))
	}
	for _, profile := range []string{profileConstrained, profileThroughput} {
		t.Run(profile, func(t *testing.T) {
			setProfileEnvironment(t, profile)
			var stdout, stderr bytes.Buffer
			require.NoError(t, run(t.Context(), []string{"-profile=" + profile}, &stdout, &stderr, runGo))
			require.Contains(t, stderr.String(), "gatefixture/tagged: pass")
			require.Contains(t, stdout.String(), `"Test":"TestPersistedBuildTag"`)
			require.Contains(t, stderr.String(), "packages=3/3 tests/subtests=2 skipped tests=0")
		})
	}
}
