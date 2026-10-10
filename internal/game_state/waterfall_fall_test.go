// Test for waterfall fall behavior
package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaterfallHazard_FallDownTwoTiles_Integration(t *testing.T) {
	// Setup: Avatar at x=100, y=96 on overworld floor 0
	// Place waterfall at player position
	// Expect: Player falls to y+2 (y=98) - matches original Ultima 5 behavior

	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld). // Overworld-type map
		WithPlayerAt(100, 96).                        // Initial position
		WithSystemCallbacks().
		Build()

	initialY := int(gs.MapState.PlayerLocation.Position.Y)
	assert.Equal(t, 96, initialY, "Initial Y position should be 96")

	// Place waterfall tile at player's current position (100, 96)
	waterfallPos := gs.MapState.PlayerLocation.Position
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &waterfallPos, indexes.Waterfall1)

	// Set low DEX to ensure damage check fails
	character := gs.PartyState.GetCharacter(0)
	character.Dexterity = 5
	initialHP := character.GetCurrentHp()

	// Trigger waterfall hazard
	gs.ProcessWaterfallHazard(&waterfallPos)
	require.True(t, gs.EventQueueManager.HasWorkflow("waterfall_hazard"))
	assert.True(t, callbacks.InputBlocked)
	assert.Equal(t, 1, callbacks.BlockInputCalls)

	// Process event queue: fade out → message → position change → glide → fade in → damage
	gs.EventQueueManager.Update() // Execute First()
	gs.ElapsedMs += 400           // Advance past fade effects (300ms + 100ms)
	for i := 0; i < 20; i++ {
		gs.EventQueueManager.Update()
		gs.ElapsedMs++
	}

	// Advance past glide duration (2500ms)
	gs.ElapsedMs += 2500
	for i := 0; i < 20; i++ {
		gs.EventQueueManager.Update()
		gs.ElapsedMs++
	}

	// Advance past fade in (300ms)
	gs.ElapsedMs += 300
	for i := 0; i < 20; i++ {
		gs.EventQueueManager.Update()
		gs.ElapsedMs++
	}

	// Verify: Player should have moved DOWN two tiles (y=96 → y=98) - matches original
	finalY := int(gs.MapState.PlayerLocation.Position.Y)
	assert.Equal(t, 98, finalY, "Player should have fallen to y=98 (y+2 of waterfall)")

	// Verify: X position should remain unchanged (x=100)
	finalX := int(gs.MapState.PlayerLocation.Position.X)
	assert.Equal(t, 100, finalX, "Player X position should remain 100")

	// Verify: Player took damage (DEX check failed)
	assert.Less(t, int(character.GetCurrentHp()), int(initialHP),
		"Player should take 1 HP damage when DEX check fails")
}

func TestWaterfallHazardCanonicalEntranceTransitionsToUnderworld(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(0).
		WithPlayerAt(54, 136).
		WithSystemCallbacks().
		Build()
	trigger := state.MapState.PlayerLocation.Position
	tile := state.GetTileInDirectionWrapped(&trigger, references.Down)
	require.NotNil(t, tile)
	require.True(t, tile.IsWaterfall(), "original BRIT.DAT waterfall must be immediately south of the trigger")

	state.ProcessWaterfallHazard(&trigger)
	for range 100 {
		state.EventQueueManager.Update()
		state.ElapsedMs += 3000
		if !state.EventQueueManager.HasActiveWorkflows() {
			break
		}
	}

	require.False(t, state.EventQueueManager.HasActiveWorkflows())
	assert.Equal(t, references.BritanniaUnderworld, state.MapState.PlayerLocation.Location)
	assert.Equal(t, references.Basement, state.MapState.PlayerLocation.Floor)
	assert.Equal(t, references.Position{X: 54, Y: 138}, state.MapState.PlayerLocation.Position)
	assert.Equal(t, state.GetCurrentLargeMapNPCAIController(), state.CurrentNPCAIController)
}
