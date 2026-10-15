package main

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestExplorationPassRepeatsAt300Then120WithoutCatchup(t *testing.T) {
	for _, context := range []struct {
		name     string
		location references.Location
		floor    references.FloorNumber
	}{
		{"town", references.Britain, 0}, {"overworld", references.BritanniaUnderworld, 0},
		{"underworld", references.BritanniaUnderworld, -1}, {"dungeon", references.Deceit, 0},
	} {
		t.Run(context.name, func(t *testing.T) {
			scene, held, _, _ := newRunicReferenceTestScene(t, context.location)
			state := scene.gameState
			state.MapState.PlayerLocation.Floor = context.floor
			if context.location == references.Deceit {
				setDungeonInputTestCell(t, state, state.MapState.PlayerLocation.Position, 0x00)
			}
			state.CurrentNPCAIController.GetNpcs().ClearAllMapUnits()
			scene.keyboard.SetElapsedMillisecondsSource(scene.getElapsedMs)
			start := state.ElapsedMs
			updateAt := func(offset int64) {
				state.ElapsedMs = start + offset
				require.NoError(t, scene.Update(nil))
			}
			before := state.DateTime.Turn
			held[ebiten.KeySpace] = true
			updateAt(0)
			afterOne := state.DateTime.Turn
			require.Greater(t, afterOne, before, "first Pass is immediate")
			updateAt(299)
			require.Equal(t, afterOne, state.DateTime.Turn)
			updateAt(300)
			afterTwo := state.DateTime.Turn
			require.Equal(t, afterOne-before, afterTwo-afterOne)
			updateAt(419)
			require.Equal(t, afterTwo, state.DateTime.Turn)
			updateAt(420)
			require.Equal(t, afterTwo+(afterOne-before), state.DateTime.Turn)
			beforeHitch := state.DateTime.Turn
			updateAt(5000)
			require.Equal(t, beforeHitch+(afterOne-before), state.DateTime.Turn, "no queued catch-up passes")
			delete(held, ebiten.KeySpace)
			updateAt(6000)
			require.Equal(t, beforeHitch+(afterOne-before), state.DateTime.Turn)
		})
	}
}

func TestPassRepeatFocusLossAndModalBoundaryRequireFreshPress(t *testing.T) {
	scene, held, _, _ := newRunicReferenceTestScene(t, references.BritanniaUnderworld)
	focused := true
	scene.transientCommandInput.focused = func() bool { return focused }
	held[ebiten.KeySpace] = true
	require.NoError(t, scene.Update(nil))
	before := scene.gameState.DateTime
	focused = false
	require.NoError(t, scene.Update(nil))
	focused = true
	scene.gameState.ElapsedMs += 1000
	require.NoError(t, scene.Update(nil))
	require.Equal(t, before, scene.gameState.DateTime)
	require.False(t, scene.transientCommandInput.pass.active)
	delete(held, ebiten.KeySpace)
	require.NoError(t, scene.Update(nil))
	held[ebiten.KeySpace] = true
	require.NoError(t, scene.Update(nil))
	require.NotEqual(t, before, scene.gameState.DateTime)
	scene.showCommandHelp()
	require.NoError(t, scene.Update(nil))
	require.True(t, scene.keyboard.IsSuppressedUntilRelease(ebiten.KeySpace))
	require.False(t, scene.transientCommandInput.pass.active)
}

func TestCombatSpaceNeverStartsPassRepeat(t *testing.T) {
	scene, held, _ := newCombatDirectionUpdateScene(t, "last_remaining")
	held[ebiten.KeySpace] = true
	require.NoError(t, scene.Update(nil))
	require.False(t, scene.transientCommandInput.pass.active)
	before := snapshotBlockedCombatState(t, scene.gameState)
	for range 3 {
		require.NoError(t, scene.Update(nil))
		require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
	}
}
