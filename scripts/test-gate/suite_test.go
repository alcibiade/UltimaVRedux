package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuiteSelectionInventoriesOtherRequiredScopeWithoutClaimingExecution(t *testing.T) {
	setProfileEnvironment(t, profileConstrained)
	t.Setenv("GOFLAGS", "-run=Nothing -skip=. -count=9 -list=. -fuzz=FuzzHidden")
	for _, suite := range []string{"logic", "render", "all"} {
		t.Run(suite, func(t *testing.T) {
			var calls [][]string
			fakeRunner := func(_ context.Context, arguments, environment []string, stdout, _ io.Writer) error {
				calls = append(calls, slices.Clone(arguments))
				assert.Contains(t, environment, "GOFLAGS=-run=Nothing -skip=. -count=9 -list=. -fuzz=FuzzHidden")
				output := fixtureGraphicsInventory
				if slices.Contains(arguments, "-test") {
					output = fixtureTestDependencies
				} else if arguments[0] == "test" {
					assert.Contains(t, arguments, "-fuzz=")
					assert.Contains(t, arguments, "-count=1")
					assert.Contains(t, arguments, "-parallel=1")
					output = fixturePass
					if arguments[len(arguments)-1] == "graphics" {
						output = fixtureGraphicsPass
					}
				}
				_, err := io.WriteString(stdout, output)
				return err
			}
			var stdout, stderr bytes.Buffer
			require.NoError(t, run(t.Context(), []string{"-profile=constrained", "-suite=" + suite}, &stdout, &stderr, fakeRunner))
			require.Contains(t, stderr.String(), "inventory packages=3")
			require.NotContains(t, stderr.String(), "MISSING")
			switch suite {
			case "logic":
				require.Len(t, calls, 3)
				require.Equal(t, []string{"assets", "core"}, calls[2][len(calls[2])-2:])
				require.Contains(t, stderr.String(), "graphics: ASSIGNED render (other required scope; not executed)")
				require.Contains(t, stderr.String(), "packages=2/2")
				require.NotContains(t, stdout.String(), `"Package":"graphics"`)
			case "render":
				require.Len(t, calls, 3)
				require.Equal(t, "graphics", calls[2][len(calls[2])-1])
				require.Contains(t, stderr.String(), "core: ASSIGNED logic (other required scope; not executed)")
				require.Contains(t, stderr.String(), "packages=1/1")
				require.NotContains(t, stdout.String(), `"Package":"core"`)
			case "all":
				require.Len(t, calls, 4)
				require.NotContains(t, stderr.String(), "ASSIGNED")
				require.Contains(t, stderr.String(), "packages=3/3")
			}
		})
	}
}

func TestRenderBackendFailureCannotBecomeOptionalScopeSkip(t *testing.T) {
	setProfileEnvironment(t, profileConstrained)
	backendError := errors.New("native backend unavailable")
	fakeRunner := func(_ context.Context, arguments, _ []string, stdout, stderr io.Writer) error {
		if arguments[0] == "test" {
			_, err := io.WriteString(stderr, backendError.Error())
			return errors.Join(backendError, err)
		}
		output := fixtureGraphicsInventory
		if slices.Contains(arguments, "-test") {
			output = fixtureTestDependencies
		}
		_, err := io.WriteString(stdout, output)
		return err
	}
	var stderr bytes.Buffer
	err := run(t.Context(), []string{"-suite=render"}, io.Discard, &stderr, fakeRunner)
	require.ErrorIs(t, err, backendError)
	require.ErrorContains(t, err, "missing package terminal for graphics")
	require.NotContains(t, err.Error(), "missing package terminal for core")
	require.Contains(t, stderr.String(), "graphics: MISSING")
	require.Contains(t, stderr.String(), "core: ASSIGNED logic")
}

func TestStrictRenderRejectsSkipWithoutRequiringUnselectedLogicResults(t *testing.T) {
	setProfileEnvironment(t, profileConstrained)
	t.Setenv("ULTIMA5_DATA_PATH", t.TempDir())
	fakeRunner := func(_ context.Context, arguments, _ []string, stdout, _ io.Writer) error {
		output := fixtureGraphicsInventory
		if slices.Contains(arguments, "-test") {
			output = fixtureTestDependencies
		} else if arguments[0] == "test" {
			output = strings.Replace(fixtureGraphicsPass, `"pass","Package":"graphics","Test":"TestDraw/child"`,
				`"skip","Package":"graphics","Test":"TestDraw/child"`, 1)
		}
		_, err := io.WriteString(stdout, output)
		return err
	}
	err := run(t.Context(), []string{"-suite=render", "-strict"}, io.Discard, io.Discard, fakeRunner)
	require.ErrorContains(t, err, `graphics/TestDraw/child terminal="skip"`)
	require.NotContains(t, err.Error(), "missing package terminal for core")
}

func TestEmptySelectedScopeStillReportsOtherRequiredWork(t *testing.T) {
	setProfileEnvironment(t, profileConstrained)
	calls := 0
	fakeRunner := func(_ context.Context, arguments, _ []string, stdout, _ io.Writer) error {
		calls++
		assert.Equal(t, "list", arguments[0], "an empty scope must not accidentally execute ./...")
		output := fixtureInventory
		if slices.Contains(arguments, "-test") {
			output = fixtureLogicDependencies
		}
		_, err := io.WriteString(stdout, output)
		return err
	}
	var stderr bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-suite=render"}, io.Discard, &stderr, fakeRunner))
	require.Equal(t, 2, calls)
	require.Contains(t, stderr.String(), "selected packages=0 inventory packages=2 assigned other scope=2")
	require.Contains(t, stderr.String(), "core: ASSIGNED logic")
	require.NotContains(t, stderr.String(), "core: pass")
}
