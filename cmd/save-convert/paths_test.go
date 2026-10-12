package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDirectoryAncestryUsesFilesystemIdentity(t *testing.T) {
	root := t.TempDir()
	protected := filepath.Join(root, "protected")
	child := filepath.Join(protected, "child")
	sibling := filepath.Join(root, "protected-other")
	require.NoError(t, os.MkdirAll(child, 0o700))
	require.NoError(t, os.Mkdir(sibling, 0o700))
	info, err := os.Stat(protected)
	require.NoError(t, err)
	// Exercise identity comparison directly, independently of the lexical fast path.
	for _, test := range []struct {
		path   string
		inside bool
	}{
		{protected, true},
		{child, true},
		{filepath.Join(child, "missing", "save.json"), true},
		{sibling, false},
		{root, false},
	} {
		inside, err := sameDirectoryAncestor(info, test.path)
		require.NoError(t, err)
		require.Equal(t, test.inside, inside, test.path)
	}
	_, err = sameDirectoryAncestor(info, filepath.Join(sibling, "invalid\x00directory", "save.json"))
	require.Error(t, err, "inspection errors must not be treated as proof of non-overlap")
}

func TestInputPolicyRejectsCaseAliasesWhenTheyHaveTheSameIdentity(t *testing.T) {
	root := t.TempDir()
	protected := filepath.Join(root, "Protected")
	alias := filepath.Join(root, "pROTECTED")
	source := filepath.Join(protected, "source")
	require.NoError(t, os.MkdirAll(source, 0o755))
	fixture := copyOriginalFixture(t)
	for _, name := range []string{"SAVED.GAM", "SAVED.OOL"} {
		require.NoError(t, os.WriteFile(filepath.Join(source, name), readFile(t, filepath.Join(fixture, name)), 0o600))
	}
	protectedInfo, err := os.Stat(protected)
	require.NoError(t, err)
	aliasInfo, err := os.Stat(alias)
	if os.IsNotExist(err) {
		// Case-sensitive hosts still execute the deterministic identity tests above.
		require.NoError(t, checkInputOverlap(filepath.Join(alias, "save.json"), false, protected))
		return
	}
	require.NoError(t, err)
	if !os.SameFile(protectedInfo, aliasInfo) {
		return
	}
	// Do not depend on EvalSymlinks normalizing the case spelling on this host.
	require.ErrorIs(t, checkInputOverlap(filepath.Join(alias, "save.json"), false, protected), errDestination)
	require.ErrorIs(t, checkInputOverlap(filepath.Join(alias, "slot-01.json"), true, source), errDestination)
	data := t.TempDir()
	for _, args := range [][]string{
		// A case alias of the source's ancestor cannot become a private slot root.
		{"--source", source, "--data-dir", data, "--save-dir", alias, "--slot", "slot-01"},
		// The same protection applies to an original-data ancestor.
		{"--source", fixture, "--data-dir", source, "--save-dir", alias, "--slot", "slot-01"},
		// Standalone output cannot enter either protected input through an alias.
		{"--source", source, "--data-dir", data, "--output", filepath.Join(alias, "source", "save.json")},
		{"--source", fixture, "--data-dir", protected, "--output", filepath.Join(alias, "save.json")},
	} {
		require.ErrorIs(t, run(context.Background(), args, io.Discard, io.Discard), errDestination)
	}
	after, err := os.Stat(protected)
	require.NoError(t, err)
	require.Equal(t, protectedInfo.Mode(), after.Mode(), "rejected ancestor must not be chmodded")
	_, err = os.Stat(filepath.Join(protected, "slot-01.json"))
	require.ErrorIs(t, err, os.ErrNotExist)
	entries, err := os.ReadDir(protected)
	require.NoError(t, err)
	require.Len(t, entries, 1, "only the source directory should remain; no output or lock")
	entries, err = os.ReadDir(source)
	require.NoError(t, err)
	require.Len(t, entries, 2, "only the original save pair should remain")
}

func TestOutputFilenamePolicyKeepsLowercaseExtension(t *testing.T) {
	for _, name := range []string{"original-save.json", "My Adventure.json", "save with spaces.json"} {
		require.NoError(t, validateOutputName(name))
	}
	for _, name := range []string{"save.JSON", "save.Json", "slot-01.json", "SLOT-01.json", "config.json"} {
		require.ErrorIs(t, validateOutputName(name), errDestination)
	}
}

func TestCLIRejectsUppercaseExtensionBeforeConversion(t *testing.T) {
	source := copyOriginalFixture(t)
	root := t.TempDir()
	path := filepath.Join(root, "save.JSON")
	err := run(context.Background(), []string{
		"--source", source, "--data-dir", t.TempDir(), "--output", path,
	}, io.Discard, io.Discard)
	require.ErrorIs(t, err, errDestination)
	require.ErrorContains(t, err, "lowercase .json")
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Empty(t, entries, "rejected filename must not create an output or lock")
}

func TestLexicalWithinDirectoryPreservesOtherPathErrors(t *testing.T) {
	_, err := lexicalWithinDirectory(t.TempDir(), "relative.json")
	require.Error(t, err)
	_, err = withinDirectory(t.TempDir(), "relative.json")
	require.Error(t, err)
}
