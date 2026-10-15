package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestDungeonFieldDispelOwnerMatchesVisitFloorAndCellNotGlobalRevision(t *testing.T) {
	var cells references.DungeonCells
	position := references.Position{X: 1, Y: 1}
	require.True(t, cells.SetCell(0, position, references.DungeonCell(0x8b)))
	state := newDungeonStateTestGame(t, cells)
	result := state.DispelCurrentOrFacingDungeonField()
	require.True(t, result.Cleared)
	owner, valid := state.CaptureDungeonFieldDispelOwner(result.Change)
	require.True(t, valid)
	rng := state.GetRNG().Calls()
	state.activeDungeon.setFacingAndLastDirection(references.DungeonFacingWest)
	_, err := state.TransformDungeonCell(0, references.Position{X: 2, Y: 2}, func(references.DungeonCell) references.DungeonCell { return 0x40 })
	require.NoError(t, err)
	assert.True(t, state.MatchesDungeonFieldDispelOwner(owner))
	state.MapState.PlayerLocation.Position = references.Position{X: 3, Y: 1}
	assert.True(t, state.MatchesDungeonFieldDispelOwner(owner))
	state.MapState.PlayerLocation.Floor = 1
	assert.False(t, state.MatchesDungeonFieldDispelOwner(owner))
	state.MapState.PlayerLocation.Floor = 0
	state.CombatScene = &CombatState{}
	assert.False(t, state.MatchesDungeonFieldDispelOwner(owner))
	state.CombatScene = nil
	copyOfVisit := *state.activeDungeon
	state.activeDungeon = &copyOfVisit
	assert.False(t, state.MatchesDungeonFieldDispelOwner(owner), "same coordinates and contents are not the same visit")
	assert.Equal(t, rng, state.GetRNG().Calls())
}

func TestDungeonFieldDispelOwnerSynchronouslyRetiredBeforeSleepFieldConsumption(t *testing.T) {
	var cells references.DungeonCells
	position := references.Position{X: 1, Y: 1}
	require.True(t, cells.SetCell(0, position, references.DungeonCell(0x88)))
	state := newDungeonStateTestGame(t, cells)
	result := state.DispelCurrentOrFacingDungeonField()
	owner, valid := state.CaptureDungeonFieldDispelOwner(result.Change)
	require.True(t, valid)
	live := true
	refresh := state.SystemCallbacks.Screen.UpdateDisplay
	state.SystemCallbacks.Screen.UpdateDisplay = func() {
		refresh()
		live = live && state.MatchesDungeonFieldDispelOwner(owner)
	}
	_, err := state.TransformDungeonCell(0, position, func(references.DungeonCell) references.DungeonCell { return 0x88 })
	require.NoError(t, err)
	assert.False(t, live, "replacement must retire before another command/update")
	state.processDungeonSleepField()
	assert.True(t, state.MatchesDungeonFieldDispelOwner(owner), "raw equality alone would miss replacement")
	assert.False(t, live, "retired presentation must not revive after the sleep field consumes itself")
}

func TestDungeonFieldDispelOwnerRejectsInvalidAdmissionAndNoOpRefresh(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	refreshes := 0
	refresh := state.SystemCallbacks.Screen.UpdateDisplay
	state.SystemCallbacks.Screen.UpdateDisplay = func() { refresh(); refreshes++ }
	change, err := state.TransformDungeonCell(0, references.Position{X: 1, Y: 1}, func(cell references.DungeonCell) references.DungeonCell { return cell })
	require.NoError(t, err)
	_, valid := state.CaptureDungeonFieldDispelOwner(change)
	assert.False(t, valid)
	assert.Zero(t, refreshes)
	_, err = state.TransformDungeonCell(0, references.Position{X: 8, Y: 1}, func(cell references.DungeonCell) references.DungeonCell { return cell })
	require.Error(t, err)
	assert.Zero(t, refreshes)
	_, err = state.TransformDungeonCell(0, references.Position{X: 1, Y: 1}, func(references.DungeonCell) references.DungeonCell { return 0x83 })
	require.NoError(t, err)
	assert.Equal(t, 1, refreshes)
	result := state.DispelCurrentOrFacingDungeonField()
	_, valid = state.CaptureDungeonFieldDispelOwner(result.Change)
	assert.True(t, valid)
	result.Change.After = 0x08
	_, valid = state.CaptureDungeonFieldDispelOwner(result.Change)
	assert.False(t, valid)
}
