package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestGetEffectiveTileDoesNotTreatSerpentsHoldAsDoom(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(0).
		WithPlayerAt(146, 241).
		WithSystemCallbacks().
		Build()
	position := state.MapState.PlayerLocation.Position
	state.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapLayer, &position, indexes.Keep)
	state.DungeonStates[DoomIndex] = false

	assert.Equal(t, indexes.SpriteIndex(indexes.Keep), state.GetEffectiveTile(&position))
}

func TestGetEffectiveTileTransformsDungeonEntrancesOnTheirOriginalWorldFloor(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithSystemCallbacks().
		Build()

	tests := []struct {
		name    string
		dungeon references.Location
		floor   references.FloorNumber
	}{
		{name: "Deceit on Britannia", dungeon: references.Deceit, floor: 0},
		{name: "Deceit in Underworld", dungeon: references.Deceit, floor: -1},
		{name: "Doom in Underworld", dungeon: references.Doom, floor: -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			position := state.GetDungeonEntranceCoordinates()[test.dungeon]
			state.MapState.PlayerLocation.Floor = test.floor
			layer := state.GetLayeredMapByCurrentLocation()
			layer.SetTileByLayer(map_state.MapLayer, &position, dungeonOpenEntranceSprite(test.dungeon))
			state.SetDungeonState(test.dungeon, false)

			assert.Equal(t, indexes.SpriteIndex(indexes.BlockEntrance), state.GetEffectiveTile(&position))
		})
	}

	t.Run("open non-Doom entrance uses its source projection on both world planes", func(t *testing.T) {
		position := state.GetDungeonEntranceCoordinates()[references.Deceit]
		state.SetDungeonState(references.Deceit, true)

		for _, floor := range []references.FloorNumber{
			references.FloorNumber(references.OVERWORLD),
			references.FloorNumber(references.UNDERWORLD),
		} {
			state.MapState.PlayerLocation.Floor = floor
			state.GetLayeredMapByCurrentLocation().SetTileByLayer(
				map_state.MapLayer,
				&position,
				dungeonOpenEntranceSprite(references.Deceit),
			)
			assert.Equal(t, indexes.SpriteIndex(indexes.DoomEntrance), state.GetEffectiveTile(&position))
		}
	})

	t.Run("Doom coordinate on Britannia is not transformed", func(t *testing.T) {
		position := state.GetDungeonEntranceCoordinates()[references.Doom]
		state.MapState.PlayerLocation.Floor = 0
		layer := state.GetLayeredMapByCurrentLocation()
		layer.SetTileByLayer(map_state.MapLayer, &position, indexes.Forest1)

		assert.Equal(t, indexes.SpriteIndex(indexes.Forest1), state.GetEffectiveTile(&position))
	})
}
