package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestActionOpenLargeMap_Door_Success(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()
	target := references.Up.GetNewPositionInDirection(&gs.MapState.PlayerLocation.Position)
	largeMap := gs.GetCurrentLayeredMap()
	largeMap.SetTileByLayer(map_state.MapOverrideLayer, target, indexes.RegularDoor)
	smallMap := loadSmallMapControl(gs)
	smallTarget := references.Position{X: 10, Y: 10}
	smallMap.SetTileByLayer(map_state.MapOverrideLayer, &smallTarget, indexes.RegularDoor)
	callbacks.Reset()

	assert.True(t, gs.ActionOpenLargeMap(references.Up))
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), largeMap.GetTileTopMapOnlyTile(target).Index)
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), smallMap.GetTileTopMapOnlyTile(&smallTarget).Index)
	assert.Equal(t, []string{"Opened!"}, callbacks.Messages)
}

func TestActionOpenLargeMap_InvalidPosition(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()
	target := references.Up.GetNewPositionInDirection(&gs.MapState.PlayerLocation.Position)
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, target, indexes.Grass)
	callbacks.Reset()

	assert.False(t, gs.ActionOpenLargeMap(references.Up))
	assert.Equal(t, []string{"Nothing to open!"}, callbacks.Messages)
}
