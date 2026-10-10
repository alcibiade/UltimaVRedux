package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

func TestDungeonSearchHiddenPitStartsRevealWithoutBlockingInput(t *testing.T) {
	dataDir := originaldata.Directory(t)
	scene, state, callbacks := newDungeonInputTestScene(t)
	if state == nil {
		return
	}
	var err error
	scene.dungeonSprites, err = sprites.LoadDungeonSprites(dataDir)
	require.NoError(t, err)
	t.Cleanup(scene.Dispose)
	initializeDungeonUpdateTestScene(scene, state)
	require.True(t, state.SetActivePlayer(0))
	state.MapState.Lighting.RestoreTorchMinutes(10)
	target := references.Position{X: 2, Y: 1}
	setDungeonInputTestCell(t, state, target, 0x61)
	callbacks.Reset()

	scene.keyboard.SimulateKeyPress(ebiten.KeyS)
	require.NoError(t, scene.Update(nil))
	require.Equal(t, SearchDirectionInput, scene.secondaryKeyState)
	require.False(t, state.IsAtDungeonCommandBoundary())
	require.Empty(t, scene.dungeonPitReveals)

	scene.keyboard.SimulateKeyPress(ebiten.KeyUp)
	require.NoError(t, scene.Update(nil))

	require.Len(t, scene.dungeonPitReveals, 1, "reveal starts in the resolving input update")
	effect := &scene.dungeonPitReveals[0]
	assert.Equal(t, references.Deceit, effect.location)
	assert.Equal(t, references.FloorNumber(0), effect.floor)
	assert.Equal(t, target, effect.position)
	assert.Equal(t, state.ElapsedMs, effect.startedAt)
	for _, dissolve := range effect.dissolves {
		assert.True(t, dissolve.IsActive())
		assert.Zero(t, dissolve.GetLevel())
	}
	assert.Equal(t, references.DungeonCell(0x60), dungeonInputCellAt(t, state, target))
	assert.Empty(t, scene.dungeonSkeletonCrumbles)
	assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
	assert.Equal(t, game_state.DungeonCommandCycle{}, scene.dungeonSearchCommandToken)
	assert.True(t, state.IsAtDungeonCommandBoundary())
	assert.Equal(t, byte(1), state.DateTime.Minute)
	assert.False(t, scene.inputBlocked)
	assert.False(t, callbacks.InputBlocked)
	assert.Zero(t, callbacks.BlockInputCalls)
	assert.False(t, state.EventQueueManager.HasActiveWorkflows())

	scene.keyboard.SimulateKeyPress(ebiten.KeyLeft)
	require.NoError(t, scene.Update(nil))
	assert.Equal(t, references.DungeonFacingNorth, dungeonInputFacing(t, state))
	assert.Equal(t, byte(2), state.DateTime.Minute)
	assert.True(t, state.IsAtDungeonCommandBoundary())
	assert.Len(t, scene.dungeonPitReveals, 1, "next command must not wait for the reveal")
	assert.Empty(t, scene.dungeonSkeletonCrumbles)
}

func TestDungeonPitRevealSceneDisposeReleasesDissolves(t *testing.T) {
	item := ebiten.NewImage(8, 8)
	t.Cleanup(item.Deallocate)
	effect := dungeonPitRevealEffect{}
	effect.initialize(syntheticDungeonSpriteSource{item: item})
	scene := &GameScene{dungeonPitReveals: []dungeonPitRevealEffect{effect}}
	t.Cleanup(scene.Dispose)
	backing := scene.dungeonPitReveals
	for _, dissolve := range backing[0].dissolves {
		require.True(t, dissolve.IsActive())
		require.NotNil(t, dissolve.buffer)
	}

	scene.Dispose()

	assert.Nil(t, scene.dungeonPitReveals)
	for _, dissolve := range backing[0].dissolves {
		assert.False(t, dissolve.IsActive())
		assert.Nil(t, dissolve.buffer)
	}
	assert.NotPanics(t, scene.Dispose)
}
