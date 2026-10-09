package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

func TestDungeonDiscoveriesRecordOnlyImmutableSourcesAndCaptureDeterministically(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithFloor(0).
		WithPlayerAt(1, 1).
		Build()
	state.GameConfig.GameplayConfiguration.PersistHiddenDungeonDiscoveries = true
	doorFloor, doorPosition, _ := findDungeonDiscoverySource(t, state, references.Deceit, func(cell references.DungeonCell) bool {
		return cell.Kind() == references.DungeonHiddenDoor
	})
	pitFloor, pitPosition, _ := findDungeonDiscoverySource(t, state, references.Deceit, isDungeonHiddenPit)
	invalidFloor, invalidPosition, _ := findDungeonDiscoverySource(t, state, references.Deceit, func(cell references.DungeonCell) bool {
		return cell.Kind() != references.DungeonHiddenDoor
	})

	state.recordDungeonDiscovery(dungeonDiscoveryHiddenPit, pitFloor, pitPosition)
	state.recordDungeonDiscovery(dungeonDiscoveryHiddenDoor, doorFloor, doorPosition)
	state.recordDungeonDiscovery(dungeonDiscoveryHiddenDoor, invalidFloor, invalidPosition)

	require.Len(t, state.dungeonDiscoveries, 2)
	first, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	second, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	assert.Equal(t, first.Redux.DungeonDiscoveries, second.Redux.DungeonDiscoveries)
	require.Len(t, first.Redux.DungeonDiscoveries, 2)
	for index := 1; index < len(first.Redux.DungeonDiscoveries); index++ {
		assert.True(t, saveDungeonDiscoveryLess(
			first.Redux.DungeonDiscoveries[index-1],
			first.Redux.DungeonDiscoveries[index],
		))
	}
}

func TestPersistentDungeonDiscoveriesReplayOnlyOnFreshEnabledVisits(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithFloor(0).
		WithPlayerAt(1, 1).
		Build()
	state.GameConfig.GameplayConfiguration.PersistHiddenDungeonDiscoveries = true
	doorFloor, doorPosition, doorSource := findDungeonDiscoverySource(t, state, references.Deceit, func(cell references.DungeonCell) bool {
		return cell.Kind() == references.DungeonHiddenDoor
	})
	pitFloor, pitPosition, pitSource := findDungeonDiscoverySource(t, state, references.Deceit, func(cell references.DungeonCell) bool {
		return cell.Raw() == 0x61
	})
	state.recordDungeonDiscovery(dungeonDiscoveryHiddenDoor, doorFloor, doorPosition)
	state.recordDungeonDiscovery(dungeonDiscoveryHiddenPit, pitFloor, pitPosition)

	enabled, err := state.PrepareDungeonState(
		references.Deceit,
		references.DungeonFacingEast,
		references.DungeonLastEast,
	)
	require.NoError(t, err)
	enabledDoor, ok := enabled.Cell(doorFloor, doorPosition)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0xe0|(doorSource.Raw()&0x08)), enabledDoor)
	enabledPit, ok := enabled.Cell(pitFloor, pitPosition)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(pitSource.Raw()&0xf8), enabledPit)

	state.GameConfig.GameplayConfiguration.PersistHiddenDungeonDiscoveries = false
	disabled, err := state.PrepareDungeonState(
		references.Deceit,
		references.DungeonFacingEast,
		references.DungeonLastEast,
	)
	require.NoError(t, err)
	disabledDoor, ok := disabled.Cell(doorFloor, doorPosition)
	require.True(t, ok)
	assert.Equal(t, doorSource, disabledDoor)
	disabledPit, ok := disabled.Cell(pitFloor, pitPosition)
	require.True(t, ok)
	assert.Equal(t, pitSource, disabledPit)
	assert.Len(t, state.dungeonDiscoveries, 2, "disabling replay must retain the journal")

	state.GameConfig.GameplayConfiguration.PersistHiddenDungeonDiscoveries = true
	reenabled, err := state.PrepareDungeonState(
		references.Deceit,
		references.DungeonFacingEast,
		references.DungeonLastEast,
	)
	require.NoError(t, err)
	reenabledDoor, ok := reenabled.Cell(doorFloor, doorPosition)
	require.True(t, ok)
	assert.Equal(t, enabledDoor, reenabledDoor)
	reenabledPit, ok := reenabled.Cell(pitFloor, pitPosition)
	require.True(t, ok)
	assert.Equal(t, enabledPit, reenabledPit)
}

func TestSaveSnapshotHydrationKeepsActiveDungeonExactBeforeFreshReplay(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithFloor(0).
		WithPlayerAt(1, 1).
		Build()
	state.GameConfig.GameplayConfiguration.PersistHiddenDungeonDiscoveries = true
	floor, position, source := findDungeonDiscoverySource(t, state, references.Deceit, func(cell references.DungeonCell) bool {
		return cell.Kind() == references.DungeonHiddenDoor
	})
	state.recordDungeonDiscovery(dungeonDiscoveryHiddenDoor, floor, position)

	saved, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	hydrated, err := NewGameStateFromSaveSnapshot(
		saved,
		state.GameConfig,
		state.GameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)
	active, ok := hydrated.DungeonStateSnapshot()
	require.True(t, ok)
	activeCell, ok := active.Cells.Cell(floor, position)
	require.True(t, ok)
	assert.Equal(t, source, activeCell, "active snapshot hydration must not replay discoveries")

	fresh, err := hydrated.PrepareDungeonState(
		references.Deceit,
		references.DungeonFacingEast,
		references.DungeonLastEast,
	)
	require.NoError(t, err)
	freshCell, ok := fresh.Cell(floor, position)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0xe0|(source.Raw()&0x08)), freshCell)
}

func TestPersistentHiddenPitReplayPreservesHighValuedFallDestination(t *testing.T) {
	position := references.Position{X: 2, Y: 3}
	var cells references.DungeonCells
	require.True(t, cells.SetCell(0, position, 0x69))
	require.True(t, cells.SetCell(1, position, 0xf0))
	state := &GameState{ //nolint:exhaustruct // Focused replay fixture.
		GameConfig: &config.UltimaVConfiguration{ //nolint:exhaustruct // Focused replay fixture.
			GameplayConfiguration: &config.GameplayConfiguration{
				PersistHiddenDungeonDiscoveries: true,
			},
		},
		dungeonDiscoveries: map[dungeonDiscoveryKey]dungeonDiscoveryKind{
			{location: references.Deceit, floor: 0, position: position}: dungeonDiscoveryHiddenPit,
		},
	}

	require.NoError(t, state.applyPersistentDungeonDiscoveries(references.Deceit, &cells))
	source, ok := cells.Cell(0, position)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0x68), source)
	below, ok := cells.Cell(1, position)
	require.True(t, ok)
	assert.Equal(t, references.DungeonCell(0xf0), below)
}

func TestDungeonSearchPersistsOnlyOriginalHiddenDoorAndSearchablePit(t *testing.T) {
	for _, test := range []struct {
		name          string
		match         func(references.DungeonCell) bool
		wantFound     bool
		wantKind      dungeonDiscoveryKind
		wantTransform func(references.DungeonCell) references.DungeonCell
	}{
		{
			name: "hidden door",
			match: func(cell references.DungeonCell) bool {
				return cell.Kind() == references.DungeonHiddenDoor
			},
			wantFound: true,
			wantKind:  dungeonDiscoveryHiddenDoor,
			wantTransform: func(cell references.DungeonCell) references.DungeonCell {
				return references.DungeonCell(0xe0 | (cell.Raw() & 0x08))
			},
		},
		{
			name: "searchable hidden pit",
			match: func(cell references.DungeonCell) bool {
				return cell.Raw() == 0x61
			},
			wantFound: true,
			wantKind:  dungeonDiscoveryHiddenPit,
			wantTransform: func(references.DungeonCell) references.DungeonCell {
				return 0x60
			},
		},
		{
			name: "fall-only hidden pit",
			match: func(cell references.DungeonCell) bool {
				return cell.Raw() == 0x69
			},
			wantFound: false,
			wantTransform: func(cell references.DungeonCell) references.DungeonCell {
				return cell
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.Deceit).
				WithSystemCallbacks().
				Build()
			state.GameConfig.GameplayConfiguration.PersistHiddenDungeonDiscoveries = true
			floor, target, source := findDungeonDiscoverySource(t, state, references.Deceit, test.match)
			player := references.Position{
				X: references.Coordinate((int(target.X) + int(references.XDungeonMapTiles) - 1) % int(references.XDungeonMapTiles)),
				Y: target.Y,
			}
			_, err := state.DebugSetupDungeonRoute(
				references.Deceit,
				floor,
				player,
				references.DungeonFacingEast,
				references.DungeonLastEast,
			)
			require.NoError(t, err)
			state.MapState.Lighting.RestoreTorchMinutes(10)
			callbacks.Reset()

			token, _ := state.BeginDungeonCommandCycle()
			found := state.ActionSearchDungeonMap(references.Up)
			state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

			assert.Equal(t, test.wantFound, found)
			active, ok := state.DungeonStateSnapshot()
			require.True(t, ok)
			got, ok := active.Cells.Cell(floor, target)
			require.True(t, ok)
			assert.Equal(t, test.wantTransform(source), got)
			key := dungeonDiscoveryKey{location: references.Deceit, floor: floor, position: target}
			if test.wantFound {
				assert.Equal(t, test.wantKind, state.dungeonDiscoveries[key])
			} else {
				assert.NotContains(t, state.dungeonDiscoveries, key)
			}
			assert.True(t, state.IsAtDungeonCommandBoundary())
		})
	}
}

func findDungeonDiscoverySource(
	t *testing.T,
	state *GameState,
	location references.Location,
	match func(references.DungeonCell) bool,
) (references.FloorNumber, references.Position, references.DungeonCell) {
	t.Helper()
	reference := state.GameReferences.LocationReferences.GetLocationReference(location)
	require.NotNil(t, reference)
	for floor := references.FloorNumber(0); floor < references.DungeonFloorCount; floor++ {
		for y := references.Coordinate(0); y < references.YDungeonMapTiles; y++ {
			for x := references.Coordinate(0); x < references.XDungeonMapTiles; x++ {
				position := references.Position{X: x, Y: y}
				cell, ok := reference.GetDungeonCell(floor, position)
				require.True(t, ok)
				if match(cell) {
					return floor, position, cell
				}
			}
		}
	}
	t.Fatal("matching immutable dungeon source cell not found")

	return 0, references.Position{}, 0
}

func saveDungeonDiscoveryLess(
	left save_snapshot.DungeonDiscoverySnapshot,
	right save_snapshot.DungeonDiscoverySnapshot,
) bool {
	if left.Location != right.Location {
		return left.Location < right.Location
	}
	if left.Floor != right.Floor {
		return left.Floor < right.Floor
	}
	if left.Y != right.Y {
		return left.Y < right.Y
	}
	if left.X != right.X {
		return left.X < right.X
	}

	return left.Kind < right.Kind
}
