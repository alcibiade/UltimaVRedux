package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyGolden(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "golden.json")
	require.NoError(t, os.WriteFile(path, []byte("same\n"), 0o600))
	assert.NoError(t, verifyGolden(path, []byte("same\n")))

	err := verifyGolden(path, []byte("different\n"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "verification mismatch")
	assert.ErrorContains(t, err, "sha256")
}

func TestRootCommandRequiresDataDirectory(t *testing.T) {
	t.Parallel()

	command := newRootCommand(nil)
	command.SetArgs(nil)

	err := command.Execute()

	require.Error(t, err)
	assert.ErrorContains(t, err, `required flag(s) "data-dir" not set`)
}
