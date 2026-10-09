package game_state

import (
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapMutationCaptureSkipsRetainedDungeonProjections(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		Build()
	attachDungeonStateForTest(t, state, references.DungeonCells{})
	position := references.Position{X: 2, Y: 2}
	dungeonProjection := state.MapState.LayeredMaps.GetLayeredMap(references.DungeonMapType, 7)
	dungeonProjection.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.Chest)
	state.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.BritanniaUnderworld,
		Position: references.Position{X: 100, Y: 100},
		Floor:    0,
	}

	mutations := state.captureMapMutations()

	for _, mutation := range mutations {
		assert.NotEqual(t, 7, mutation.Position.Floor,
			"derived dungeon floors must never appear as world map mutations")
	}
}

func TestSaveEligibilityRejectsActiveDungeonCommandOutsideDungeonLocation(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		Build()
	attachDungeonStateForTest(t, state, references.DungeonCells{})
	_, _ = state.BeginDungeonCommandCycle()
	state.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.BritanniaUnderworld,
		Position: references.Position{X: 100, Y: 100},
		Floor:    0,
	}

	err := state.ValidateSaveSnapshotEligibility()

	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "active dungeon command cycle"))
}
