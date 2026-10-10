package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootCommandRejectsArguments(t *testing.T) {
	command := newRootCommand()
	command.SetArgs([]string{"extra", "--data-dir", ".", "--output", t.TempDir()})

	err := command.Execute()

	require.ErrorContains(t, err, "unknown command")
}
