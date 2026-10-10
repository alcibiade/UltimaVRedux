package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindPointerViolationsHandlesImportsAndSyntax(t *testing.T) {
	root := t.TempDir()
	writeGoSource(t, root, `package sample

import (
	e "github.com/hajimehoshi/ebiten/v2"
	iu "github.com/hajimehoshi/ebiten/v2/inpututil"
)

func update() {
	_, _ = e.CursorPosition()
	_ = iu.IsMouseButtonJustPressed(e.MouseButtonLeft)
	_ = "ebiten.Wheel()"
	// ebiten.TouchIDs()
	_ = helm.Wheel
}

func shadowImport() {
	e := struct{ Wheel int }{}
	_ = e.Wheel
}
`)

	violations, err := findPointerViolations([]string{root})
	require.NoError(t, err)
	require.Len(t, violations, 3)
	assert.Equal(t, "e.CursorPosition", violations[0].api)
	assert.Equal(t, "iu.IsMouseButtonJustPressed", violations[1].api)
	assert.Equal(t, "e.MouseButtonLeft", violations[2].api)
}

func TestFindPointerViolationsRejectsDotImports(t *testing.T) {
	root := t.TempDir()
	writeGoSource(t, root, `package sample

import . "github.com/hajimehoshi/ebiten/v2"
`)

	violations, err := findPointerViolations([]string{root})
	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Contains(t, violations[0].api, "dot import")
}

func TestFindPointerViolationsOnlyScansConfiguredRoots(t *testing.T) {
	workspace := t.TempDir()
	playerRoot := filepath.Join(workspace, "player")
	workbenchRoot := filepath.Join(workspace, "workbench")
	require.NoError(t, os.Mkdir(playerRoot, 0o755))
	require.NoError(t, os.Mkdir(workbenchRoot, 0o755))
	writeGoSource(t, playerRoot, "package player\n")
	writeGoSource(t, workbenchRoot, `package workbench

import "github.com/hajimehoshi/ebiten/v2"

func update() { _, _ = ebiten.CursorPosition() }
`)

	violations, err := findPointerViolations([]string{playerRoot})
	require.NoError(t, err)
	assert.Empty(t, violations)
}

func writeGoSource(t *testing.T, root, source string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(root, "sample.go"), []byte(source), 0o600))
}
