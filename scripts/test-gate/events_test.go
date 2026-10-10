package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixtureInventory = `{"ImportPath":"core","TestGoFiles":["core_test.go"]}
{"ImportPath":"assets"}`

const fixturePass = `{"Action":"start","Package":"core"}
{"Action":"run","Package":"core","Test":"TestMandatoryAssertion"}
{"Action":"output","Package":"core","Test":"TestMandatoryAssertion","Output":"assertion executed\n"}
{"Action":"pass","Package":"core","Test":"TestMandatoryAssertion"}
{"Action":"pass","Package":"core"}
{"Action":"output","Package":"assets","Output":"? assets [no test files]\n"}
{"Action":"skip","Package":"assets"}`

func TestEventInventory(t *testing.T) {
	for _, test := range []struct {
		name, stream, wantError string
		strict                  bool
	}{
		{name: "pass and no-test package", stream: fixturePass, strict: true},
		{name: "public skipped assertion", stream: strings.Replace(fixturePass,
			`"pass","Package":"core","Test"`, `"skip","Package":"core","Test"`, 1)},
		{name: "strict cannot skip mandatory core assertion", strict: true, wantError: `terminal="skip"`,
			stream: strings.Replace(fixturePass, `"pass","Package":"core","Test"`, `"skip","Package":"core","Test"`, 1)},
		{name: "failed test despite package pass", wantError: `terminal="fail"`,
			stream: strings.Replace(fixturePass, `"pass","Package":"core","Test"`, `"fail","Package":"core","Test"`, 1)},
		{name: "failed package", wantError: "package core ended with fail",
			stream: strings.Replace(fixturePass, `"pass","Package":"core"}`, `"fail","Package":"core"}`, 1)},
		{name: "missing package", wantError: "missing package terminal for assets",
			stream: strings.Replace(fixturePass, `{"Action":"skip","Package":"assets"}`, "", 1)},
		{name: "missing test terminal", wantError: `TestMandatoryAssertion terminal=""`,
			stream: strings.Replace(fixturePass, `{"Action":"pass","Package":"core","Test":"TestMandatoryAssertion"}`, "", 1)},
		{name: "no core test events", wantError: "core has test files but emitted no test events",
			stream: `{"Action":"pass","Package":"core"}{"Action":"skip","Package":"assets"}`},
		{name: "package skipped before tests", wantError: "package core ended with skip",
			stream: `{"Action":"skip","Package":"core"}{"Action":"skip","Package":"assets"}`},
		{name: "duplicate package terminal", wantError: "duplicate package terminal",
			stream: fixturePass + `{"Action":"pass","Package":"core"}`},
		{name: "duplicate test terminal", wantError: "duplicate test terminal",
			stream: strings.Replace(fixturePass, `{"Action":"pass","Package":"core","Test":"TestMandatoryAssertion"}`,
				strings.Repeat(`{"Action":"pass","Package":"core","Test":"TestMandatoryAssertion"}`, 2), 1)},
		{name: "duplicate test run", wantError: "duplicate test run",
			stream: strings.Replace(fixturePass, `{"Action":"run","Package":"core","Test":"TestMandatoryAssertion"}`,
				strings.Repeat(`{"Action":"run","Package":"core","Test":"TestMandatoryAssertion"}`, 2), 1)},
		{name: "test terminal without run", wantError: "test event without run",
			stream: `{"Action":"pass","Package":"core","Test":"TestMandatoryAssertion"}`},
		{name: "unexpected package", wantError: `unexpected package "wrong"`,
			stream: fixturePass + `{"Action":"pass","Package":"wrong"}`},
		{name: "empty stream", wantError: "empty go test event stream"},
		{name: "malformed JSON", stream: fixturePass + "{", wantError: "decoding go test JSON"},
		{name: "invalid action", stream: `{"Action":"wat","Package":"core"}`, wantError: "invalid package action"},
		{name: "empty event", stream: `{}`, wantError: "unexpected package"},
		{name: "build failure", stream: `{"Action":"build-output","ImportPath":"dependency","Output":"compiler error"}
{"Action":"build-fail","ImportPath":"dependency"}`, wantError: "build failed for dependency"},
	} {
		t.Run(test.name, func(t *testing.T) {
			expected, err := readPackages(strings.NewReader(fixtureInventory))
			require.NoError(t, err)
			err = errors.Join(readEvents(strings.NewReader(test.stream), expected), validateResults(expected, test.strict))
			if test.wantError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.wantError)
			}
		})
	}
}

func TestPackageInventoryInput(t *testing.T) {
	for _, input := range []string{"", "{", `{}`, `{"ImportPath":"core"}{"ImportPath":"core"}`} {
		_, err := readPackages(strings.NewReader(input))
		require.Error(t, err, "input %q", input)
	}
	expected, err := readPackages(strings.NewReader(`{"ImportPath":"external","XTestGoFiles":["external_test.go"]}`))
	require.NoError(t, err)
	require.True(t, expected["external"].hasTests)
}

func TestSkippedTestSummaryIncludesAllNamesAndReasons(t *testing.T) {
	expected, err := readPackages(strings.NewReader(fixtureInventory))
	require.NoError(t, err)
	stream := `{"Action":"run","Package":"core","Test":"TestFirst"}
{"Action":"output","Package":"core","Test":"TestFirst","Output":"    first_test.go:5: optional original game data unavailable\n"}
{"Action":"skip","Package":"core","Test":"TestFirst"}
{"Action":"run","Package":"core","Test":"TestSecond"}
{"Action":"output","Package":"core","Test":"TestSecond","Output":"    second_test.go:7: unavailable platform capability\n"}
{"Action":"skip","Package":"core","Test":"TestSecond"}
{"Action":"pass","Package":"core"}
{"Action":"skip","Package":"assets"}`
	require.NoError(t, readEvents(strings.NewReader(stream), expected))
	require.NoError(t, validateResults(expected, false))
	require.Error(t, validateResults(expected, true))
	var summary bytes.Buffer
	require.NoError(t, writeSummary(&summary, expected))
	for _, text := range []string{"SKIP core/TestFirst", "SKIP core/TestSecond", "optional original game data unavailable",
		"unavailable platform capability", "packages=2/2", "tests/subtests=2", "skipped tests=2"} {
		assert.Contains(t, summary.String(), text)
	}
}

func TestLargeOutputAndParallelSubtests(t *testing.T) {
	expected, err := readPackages(strings.NewReader(fixtureInventory))
	require.NoError(t, err)
	stream := strings.Replace(fixturePass, `assertion executed\n`, strings.Repeat("x", 1024*1024), 1)
	parallel := `{"Action":"run","Package":"core","Test":"TestMandatoryAssertion/child"}
{"Action":"pause","Package":"core","Test":"TestMandatoryAssertion/child"}
{"Action":"cont","Package":"core","Test":"TestMandatoryAssertion/child"}
{"Action":"pass","Package":"core","Test":"TestMandatoryAssertion/child"}`
	stream = strings.Replace(stream, `{"Action":"pass","Package":"core","Test":"TestMandatoryAssertion"}`,
		parallel+`{"Action":"pass","Package":"core","Test":"TestMandatoryAssertion"}`, 1)
	require.NoError(t, readEvents(strings.NewReader(stream), expected))
	require.NoError(t, validateResults(expected, true))
}
