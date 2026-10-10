package texteditor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandUsesArgumentArraysForSupportedPlatforms(t *testing.T) {
	path := `/home/Avatar's Saves/config.yaml`
	for _, test := range []struct {
		goos string
		name string
		args []string
	}{
		{goos: "windows", name: "notepad.exe", args: []string{path}},
		{goos: "darwin", name: "open", args: []string{"-t", path}},
		{goos: "linux", name: "xdg-open", args: []string{path}},
	} {
		t.Run(test.goos, func(t *testing.T) {
			name, args, err := Command(test.goos, path)
			require.NoError(t, err)
			assert.Equal(t, test.name, name)
			assert.Equal(t, test.args, args)
		})
	}
}

func TestCommandRejectsUnsupportedPlatform(t *testing.T) {
	_, _, err := Command("plan9", "/tmp/config.yaml")
	require.ErrorContains(t, err, "unsupported")
}
