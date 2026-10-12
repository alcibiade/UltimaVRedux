//go:build windows

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Pure Windows path semantics: no mounted drives, UNC servers or test skips.
func TestLexicalWithinDirectoryAcrossWindowsVolumes(t *testing.T) {
	for _, paths := range [][2]string{
		{`C:\original`, `D:\saves\import.json`},
		{`\\server\original\data`, `\\server\saves\import.json`},
		{`C:\original`, `\\server\share\import.json`},
	} {
		inside, err := lexicalWithinDirectory(paths[0], paths[1])
		require.NoError(t, err)
		require.False(t, inside)
	}
	inside, err := lexicalWithinDirectory(`C:\original`, `c:\original\save.json`)
	require.NoError(t, err)
	require.True(t, inside)
	_, err = lexicalWithinDirectory(`C:\original`, `D:relative.json`)
	require.Error(t, err, "drive-relative errors must not be mistaken for absolute cross-volume paths")
}
