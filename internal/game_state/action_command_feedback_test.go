package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestActionOpenSurfaceFeedback(t *testing.T) {
	for _, location := range []references.Location{references.Britain, references.BritanniaUnderworld} {
		for _, test := range []struct {
			name    string
			tile    indexes.SpriteIndex
			message string
			handled bool
		}{
			{"footlocker", indexes.Box, "It's open!", true},
			{"portcullis", indexes.Portcullis, "Too heavy!", false},
			{"nothing", indexes.Grass, "Nothing to open!", false},
			{"physical lock", indexes.LockedDoor, "Locked!", false},
			{"magic lock", indexes.MagicLockDoor, "Locked!", false},
		} {
			t.Run(location.String()+"/"+test.name, func(t *testing.T) {
				state, callbacks := NewIntegrationTestBuilder(t).WithLocation(location).
					WithPlayerAt(15, 15).WithSystemCallbacks().Build()
				target := references.Up.GetNewPositionInDirection(&state.MapState.PlayerLocation.Position)
				state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, target, test.tile)
				callbacks.Reset()
				action := state.ActionOpenSmallMap
				if location == references.BritanniaUnderworld {
					action = state.ActionOpenLargeMap
				}

				assert.Equal(t, test.handled, action(references.Up))
				assert.Equal(t, []string{test.message}, callbacks.Messages)
				assert.Equal(t, test.tile, state.GetCurrentLayeredMap().GetTileTopMapOnlyTile(target).Index)
			})
		}
	}
}

func TestActionGetSmallMapFeedback(t *testing.T) {
	for _, test := range []struct {
		name      string
		tile      indexes.SpriteIndex
		direction references.Direction
		messages  []string
		success   bool
	}{
		{"nothing", indexes.Grass, references.Up, []string{"Nothing to get!"}, false},
		{"plate wrong side", indexes.TableFoodTop, references.Up, []string{"Can't reach plate!"}, false},
		{"plate sideways", indexes.TableFoodBoth, references.Right, []string{"Can't reach plate!"}, false},
		{"crops", indexes.WheatInField, references.Up, []string{"Crops picked!", "Those crops are not yours, Avatar."}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.Britain).
				WithPlayerAt(15, 15).WithSystemCallbacks().Build()
			target := test.direction.GetNewPositionInDirection(&state.MapState.PlayerLocation.Position)
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, target, test.tile)
			food := state.PartyState.Inventory.Provisions.Food.Get()
			callbacks.Reset()

			assert.Equal(t, test.success, state.ActionGetSmallMap(test.direction))
			assert.Equal(t, test.messages, callbacks.Messages)
			wantFood := food
			if test.name == "crops" {
				wantFood++
			}
			assert.Equal(t, wantFood, state.PartyState.Inventory.Provisions.Food.Get())
			if !test.success {
				assert.Equal(t, test.tile, state.GetCurrentLayeredMap().GetTileTopMapOnlyTile(target).Index)
			}
		})
	}
}

func TestActionGetSmallMapCropSaturatesFoodAndRefreshesDisplays(t *testing.T) {
	for _, test := range []struct {
		name      string
		food      uint16
		karma     int
		wantFood  uint16
		wantKarma int
	}{
		{name: "below cap and nonzero karma", food: 9998, karma: 10, wantFood: 9999, wantKarma: 9},
		{name: "at cap and zero karma", food: 9999, karma: 0, wantFood: 9999, wantKarma: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.Britain).
				WithPlayerAt(15, 15).WithSystemCallbacks().Build()
			target := references.Up.GetNewPositionInDirection(&state.MapState.PlayerLocation.Position)
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, target, indexes.WheatInField)
			state.PartyState.Inventory.Provisions.Food.Set(test.food)
			state.PartyState.Karma.Value = test.karma
			callbacks.Reset()

			require.True(t, state.ActionGetSmallMap(references.Up))

			assert.Equal(t, test.wantFood, state.PartyState.Inventory.Provisions.Food.Get())
			assert.Equal(t, test.wantKarma, state.PartyState.Karma.Value)
			assert.Equal(t, indexes.SpriteIndex(indexes.PlowedField), state.GetCurrentLayeredMap().GetTileTopMapOnlyTile(target).Index)
			assert.Equal(t, 1, callbacks.StatsDisplayUpdated)
			assert.Equal(t, 2, callbacks.ScreenCallCount)
			assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
		})
	}
}

func TestActionGetSurfaceScrollUsesReadableName(t *testing.T) {
	for _, location := range []references.Location{references.Britain, references.BritanniaUnderworld} {
		t.Run(location.String(), func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).WithLocation(location).
				WithPlayerAt(15, 15).WithSystemCallbacks().Build()
			target := references.Up.GetNewPositionInDirection(&state.MapState.PlayerLocation.Position)
			require.NoError(t, state.ItemStacksMap.Push(target, &references.ItemStack{Items: references.ItemStacks{
				{Item: references.ScrollKalXenCorp, Quantity: 1},
			}}))
			callbacks.Reset()
			action := state.ActionGetSmallMap
			if location == references.BritanniaUnderworld {
				action = state.ActionGetLargeMap
			}

			assert.True(t, action(references.Up))
			assert.Equal(t, []string{"Kal Xen Corp!"}, callbacks.Messages)
			assert.False(t, state.ItemStacksMap.HasItemStackAtPosition(target))
		})
	}
}

func TestDungeonOpenAndJimmyPreserveDistinctFeedback(t *testing.T) {
	for _, test := range []struct {
		name    string
		cell    references.DungeonCell
		action  func(*GameState, int) bool
		message string
	}{
		{"open", 0x70, (*GameState).ActionOpenDungeonMap, "Already Open!"},
		{"jimmy open", 0x70, (*GameState).ActionJimmyDungeonMap, "Already open!"},
		{"jimmy no keys", 0x41, (*GameState).ActionJimmyDungeonMap, "No keys!"},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := newDungeonInteractionTestState(t, test.cell, 1)
			state.PartyState.Inventory.Provisions.Keys.Set(0)
			callbacks.Reset()

			runDungeonInteraction(t, state, func() { assert.False(t, test.action(state, 0)) })

			assert.Equal(t, []string{test.message}, callbacks.Messages)
		})
	}
}
