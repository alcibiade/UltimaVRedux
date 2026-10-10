package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDungeonStateTestGame(t *testing.T, cells references.DungeonCells) *GameState {
	t.Helper()
	state := &GameState{}
	state.SetRandomSeed(1)
	state.SystemCallbacks = NewMockSystemCallbacks(t).ToSystemCallbacks()
	var roster [party_state.NRosterSlots]*party_state.PlayerCharacter
	for index := range roster {
		roster[index] = &party_state.PlayerCharacter{PartyMap: party_state.PartyMapUnrecruited}
	}
	roster[0].PartyMap = party_state.PartyMapActive
	state.PartyState = *party_state.NewPartyStateFromRoster(roster)
	state.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.Deceit,
		Position: references.Position{X: 1, Y: 1},
		Floor:    0,
	}
	state.MapState.XTilesVisibleOnGameScreen = 19
	state.MapState.YTilesVisibleOnGameScreen = 13
	state.MapState.LayeredMaps = *map_state.NewLayeredMaps(
		nil,
		&references.LargeMapReference{},
		&references.LargeMapReference{},
		19,
		13,
	)
	active, err := newDungeonState(
		references.Deceit,
		cells,
		references.DungeonFacingEast,
		references.DungeonArrivedFromAbove,
	)
	require.NoError(t, err)
	state.activeDungeon = active
	state.MapState.LayeredMaps.ResetAndCreateDungeonMap(cells, nil, 19, 13)
	state.synchronizeDungeonRoomEncounterForStablePosition()

	return state
}

func attachDungeonStateForTest(t *testing.T, state *GameState, cells references.DungeonCells) {
	t.Helper()
	state.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.Deceit,
		Position: references.Position{X: 1, Y: 1},
		Floor:    0,
	}
	active, err := newDungeonState(
		references.Deceit,
		cells,
		references.DungeonFacingEast,
		references.DungeonArrivedFromAbove,
	)
	require.NoError(t, err)
	state.activeDungeon = active
	state.MapState.LayeredMaps.ResetAndCreateDungeonMap(
		cells,
		state.GameReferences.TileReferences,
		state.MapState.XTilesVisibleOnGameScreen,
		state.MapState.YTilesVisibleOnGameScreen,
	)
	state.synchronizeDungeonRoomEncounterForStablePosition()
}

func TestDungeonStateSnapshotIsDetached(t *testing.T) {
	var cells references.DungeonCells
	position := references.Position{X: 2, Y: 3}
	cells.SetCell(4, position, references.DungeonCell(0xc8))
	state := newDungeonStateTestGame(t, cells)

	snapshot := state.activeDungeon.Snapshot()
	snapshot.Cells.SetCell(4, position, references.DungeonCell(0xb0))
	snapshot.Wanderer = &DungeonWanderer{Position: position}

	cell, ok := state.activeDungeon.Cell(4, position)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0xc8), cell)
	assert.Nil(t, state.activeDungeon.wanderer)
}

func TestTransformDungeonCellUpdatesAuthorityAndProjection(t *testing.T) {
	var cells references.DungeonCells
	position := references.Position{X: 2, Y: 3}
	cells.SetCell(0, position, references.DungeonCell(0xc4))
	state := newDungeonStateTestGame(t, cells)

	change, err := state.TransformDungeonCell(0, position, func(cell references.DungeonCell) references.DungeonCell {
		return references.DungeonCell(0xb0 | (cell.Raw() & 0x08))
	})
	require.NoError(t, err)
	assert.Equal(t, references.DungeonCell(0xc4), change.Before)
	assert.Equal(t, references.DungeonCell(0xb0), change.After)
	assert.Equal(t, uint64(1), change.Revision)

	cell, ok := state.activeDungeon.Cell(0, position)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0xb0), cell)
	projection := state.MapState.LayeredMaps.GetLayeredMap(references.DungeonMapType, 0)
	assert.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall),
		projection.SpriteIndexByLayer(map_state.MapLayer, position))
}

func TestInstallDungeonStateRejectsMismatchedLocationIdentity(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	state.MapState.PlayerLocation.Location = references.Doom
	prepared, err := newDungeonState(
		references.Deceit,
		references.DungeonCells{},
		references.DungeonFacingEast,
		references.DungeonArrivedFromAbove,
	)
	require.NoError(t, err)

	assert.PanicsWithValue(t, "dungeon state source does not match player location", func() {
		state.installDungeonState(prepared)
	})
}

func TestTransformDungeonCellRejectsStaleActiveDungeonIdentity(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	state.MapState.PlayerLocation.Location = references.Doom

	assert.PanicsWithValue(t, "active dungeon does not match player location", func() {
		_, _ = state.TransformDungeonCell(0, references.Position{}, func(cell references.DungeonCell) references.DungeonCell {
			return cell
		})
	})
}

func TestTransformDungeonCellNoOpDoesNotAdvanceRevision(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	change, err := state.TransformDungeonCell(0, references.Position{}, func(cell references.DungeonCell) references.DungeonCell {
		return cell
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(0), change.Revision)
}
