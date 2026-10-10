package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestExtractCommandRequiresFlagsAndRejectsArguments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		args    []string
		message string
	}{
		{name: "flags", args: []string{"extract"}, message: `required flag(s) "data-dir", "output" not set`},
		{name: "arguments", args: []string{"extract", "extra", "--data-dir", ".", "--output", filepath.Join(t.TempDir(), "out")}, message: "unknown command"},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			command := newRootCommand()
			command.SetArgs(testCase.args)
			command.SetOut(&bytes.Buffer{})
			command.SetErr(&bytes.Buffer{})
			err := command.Execute()
			require.ErrorContains(t, err, testCase.message)
		})
	}
}

func TestRootAndWorkspaceWithoutSubcommandPrintHelp(t *testing.T) {
	t.Parallel()

	for _, arguments := range [][]string{nil, {"workspace"}} {
		command := newRootCommandWithWorkspace("/private/conversations")
		command.SetArgs(arguments)
		output := &bytes.Buffer{}
		command.SetOut(output)
		command.SetErr(&bytes.Buffer{})

		require.NoError(t, command.Execute())
		require.Contains(t, output.String(), "Usage:")
	}
}

func TestWorkspaceCommandDefaultsAndOverrides(t *testing.T) {
	t.Parallel()

	command := newRootCommandWithWorkspace("/private/conversations")
	workspaceCommand, _, err := command.Find([]string{"workspace"})
	require.NoError(t, err)
	assertFlagDefault(t, workspaceCommand, "data-dir", defaultDataDirectory)
	assertFlagDefault(t, workspaceCommand, "workspace", "/private/conversations")
	assertFlagDefault(t, workspaceCommand, "patch-dir", defaultPatchDirectory)

	overrideRoot := t.TempDir()
	command = newRootCommandWithWorkspace("/private/conversations")
	command.SetArgs([]string{
		"workspace", "check",
		"--data-dir", filepath.Join(overrideRoot, "data"),
		"--workspace", filepath.Join(overrideRoot, "workspace"),
		"--patch-dir", filepath.Join(overrideRoot, "patches"),
	})
	command.SetOut(&bytes.Buffer{})
	command.SetErr(&bytes.Buffer{})
	err = command.Execute()
	require.ErrorContains(t, err, filepath.Join(overrideRoot, "workspace"))
}

func TestWorkspaceSubcommandsRejectArguments(t *testing.T) {
	t.Parallel()

	for _, subcommand := range []string{"init", "diff", "check", "patches", "verify"} {
		subcommand := subcommand
		t.Run(subcommand, func(t *testing.T) {
			t.Parallel()

			command := newRootCommandWithWorkspace("/private/conversations")
			command.SetArgs([]string{"workspace", subcommand, "extra"})
			command.SetOut(&bytes.Buffer{})
			command.SetErr(&bytes.Buffer{})
			err := command.Execute()
			require.ErrorContains(t, err, "unknown command")
		})
	}
}

func assertFlagDefault(t *testing.T, command *cobra.Command, name, want string) {
	t.Helper()
	flag := command.Flag(name)
	require.NotNil(t, flag)
	require.Equal(t, want, flag.DefValue)
}
