package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestActionLookLargeMapWrapsAllWorldSeams(t *testing.T) {
	for _, floor := range []references.FloorNumber{references.FloorNumber(references.OVERWORLD), references.FloorNumber(references.UNDERWORLD)} {
		for _, seam := range largeMapSeamCases() {
			t.Run(fmt.Sprintf("floor_%d/%s", floor, seam.name), func(t *testing.T) {
				state, callbacks := NewIntegrationTestBuilder(t).
					WithLocation(references.BritanniaUnderworld).
					WithFloor(floor).
					WithPlayerAt(int(seam.start.X), int(seam.start.Y)).
					WithSystemCallbacks().
					Build()
				state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &seam.wrapped, indexes.BrickFloor)
				callbacks.Reset()
				var popupMessages []string
				state.SystemCallbacks.UI.ShowActionFeedback = func(message string, alsoAddToOutput bool) {
					assert.False(t, alsoAddToOutput)
					popupMessages = append(popupMessages, message)
				}

				require.True(t, state.ActionLookLargeMap(seam.direction))

				description := state.GameReferences.LookReferences.GetTileLookDescription(indexes.BrickFloor)
				assert.Equal(t, []string{"Thou dost see " + description}, popupMessages)
				assert.Equal(t, []string{"Look-Thou dost see " + description}, callbacks.Messages)
			})
		}
	}
}

func TestActionLookLargeMapDoesNotWrapDynamicObjectsAcrossWorldSeam(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.FloorNumber(references.OVERWORLD)).
		WithPlayerAt(255, 31).
		WithSystemCallbacks().
		Build()
	wrapped := references.Position{X: 0, Y: 31}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &wrapped, indexes.BrickFloor)
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapUnitLayer, &wrapped, indexes.BardKeyIndex)
	callbacks.Reset()

	require.True(t, state.ActionLookLargeMap(references.Right))

	description := state.GameReferences.LookReferences.GetTileLookDescription(indexes.BrickFloor)
	assert.Contains(t, callbacks.Messages, "Look-Thou dost see "+description)
	assert.NotContains(t, callbacks.Messages,
		"Look-Thou dost see "+state.GameReferences.LookReferences.GetTileLookDescription(indexes.BardKeyIndex))
}

func TestActionKlimbLargeMapWrapsAllWorldSeams(t *testing.T) {
	for _, floor := range []references.FloorNumber{references.FloorNumber(references.OVERWORLD), references.FloorNumber(references.UNDERWORLD)} {
		for _, seam := range largeMapSeamCases() {
			for _, target := range []struct {
				name    string
				tile    indexes.SpriteIndex
				success bool
			}{
				{name: "climbable", tile: indexes.SmallMountains, success: true},
				{name: "impassable", tile: indexes.TallMountains, success: false},
			} {
				t.Run(fmt.Sprintf("floor_%d/%s/%s", floor, seam.name, target.name), func(t *testing.T) {
					state, _ := NewIntegrationTestBuilder(t).
						WithLocation(references.BritanniaUnderworld).
						WithFloor(floor).
						WithPlayerAt(int(seam.start.X), int(seam.start.Y)).
						WithSystemCallbacks().
						Build()
					state.PartyState.Inventory.QuestItems.Set(references.Grapple, 1)
					state.PartyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
						character.Dexterity = 30
					})
					state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &seam.wrapped, target.tile)

					assert.Equal(t, target.success, state.ActionKlimbLargeMap(seam.direction))
					if target.success {
						assert.Equal(t, seam.wrapped, state.MapState.PlayerLocation.Position)
					} else {
						assert.Equal(t, seam.start, state.MapState.PlayerLocation.Position)
					}
				})
			}
		}
	}
}

func TestResolveLookSignPieceWrapsLargeMapRedirect(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.FloorNumber(references.OVERWORLD)).
		WithPlayerAt(0, 10).
		WithSystemCallbacks().
		Build()
	redirect := references.Position{X: 0, Y: 9}
	terminal := references.Position{X: 255, Y: 9}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &redirect, indexes.LargeSignRight)
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &terminal, indexes.BrickFloor)

	tile, redirected := state.resolveLookSignPiece(&redirect, indexes.LargeSignRight)

	assert.True(t, redirected)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), tile)
}

type largeMapSeamCase struct {
	name      string
	start     references.Position
	direction references.Direction
	wrapped   references.Position
}

func largeMapSeamCases() []largeMapSeamCase {
	return []largeMapSeamCase{
		{name: "north", start: references.Position{X: 31, Y: 0}, direction: references.Up, wrapped: references.Position{X: 31, Y: 255}},
		{name: "south", start: references.Position{X: 31, Y: 255}, direction: references.Down, wrapped: references.Position{X: 31, Y: 0}},
		{name: "west", start: references.Position{X: 0, Y: 31}, direction: references.Left, wrapped: references.Position{X: 255, Y: 31}},
		{name: "east", start: references.Position{X: 255, Y: 31}, direction: references.Right, wrapped: references.Position{X: 0, Y: 31}},
	}
}
