// Integration tests for View command workflows with real game data
// These tests validate the view command flow including gem consumption,
// SystemCallbacks integration, and GemViewDialog triggering.
package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestViewCommand_SmallMap_WithGems tests V command with gems in inventory
func TestViewCommand_SmallMap_WithGems(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithPlayerAt(12, 12).
		WithSystemCallbacks().
		Build()

	t.Run("ViewConsumesGem", func(t *testing.T) {
		mockCallbacks.Reset()

		// Give party some gems
		gs.PartyState.Inventory.Provisions.Gems.Set(5)
		initialGems := gs.PartyState.Inventory.Provisions.Gems.Get()

		result := gs.ActionViewSmallMap()

		// Should succeed
		assert.True(t, result, "View should succeed with gems")

		// Should have consumed one gem
		assert.Equal(t, initialGems-1, gs.PartyState.Inventory.Provisions.Gems.Get(),
			"Should consume exactly one gem")

		// Should have triggered ShowGemViewInteraction callback
		assert.True(t, mockCallbacks.ShowGemViewInteractionCalled,
			"Should trigger ShowGemViewInteraction callback")
		assert.Equal(t, []int{1}, mockCallbacks.TimeAdvanced())
	})

	t.Run("ViewLastGem", func(t *testing.T) {
		mockCallbacks.Reset()

		// Set to exactly 1 gem
		gs.PartyState.Inventory.Provisions.Gems.Set(1)

		result := gs.ActionViewSmallMap()

		// Should succeed
		assert.True(t, result, "View should succeed with last gem")

		// Should have consumed the last gem
		assert.Equal(t, uint16(0), gs.PartyState.Inventory.Provisions.Gems.Get(),
			"Should have zero gems after using last one")

		// Should have triggered ShowGemViewInteraction callback
		assert.True(t, mockCallbacks.ShowGemViewInteractionCalled,
			"Should trigger ShowGemViewInteraction callback")
		assert.Equal(t, []int{1}, mockCallbacks.TimeAdvanced())
	})
}

func TestViewCommandDungeonConsumesGemWithoutSurfaceCallbacks(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithFloor(2).
		WithPlayerAt(3, 4).
		WithSystemCallbacks().
		Build()
	gs.PartyState.Inventory.Provisions.Gems.Set(1)
	mockCallbacks.Reset()

	token, start := gs.BeginDungeonCommandCycle()
	model, ok := gs.ActionViewDungeonMap(token)

	require.True(t, ok)
	assert.Equal(t, 1, start.ElapsedMinutes)
	assert.Equal(t, references.Deceit, model.Location)
	assert.Equal(t, references.FloorNumber(2), model.Floor)
	assert.Zero(t, gs.PartyState.Inventory.Provisions.Gems.Get())
	assert.False(t, mockCallbacks.ShowGemViewInteractionCalled)
	assert.False(t, gs.IsAtDungeonCommandBoundary())

	gs.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	assert.True(t, gs.IsAtDungeonCommandBoundary())
}

func TestViewCommandDungeonWithoutGemStillClaimsFullCompletion(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithSystemCallbacks().
		Build()
	gs.PartyState.Inventory.Provisions.Gems.Set(0)
	mockCallbacks.Reset()

	token, _ := gs.BeginDungeonCommandCycle()
	_, ok := gs.ActionViewDungeonMap(token)

	assert.False(t, ok)
	assert.Contains(t, mockCallbacks.Messages, "You have none!")
	assert.False(t, gs.IsAtDungeonCommandBoundary())
	gs.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	assert.True(t, gs.IsAtDungeonCommandBoundary())
}

// View without gems reports "You have none!" and opens no gem view on either map.
func TestViewCommand_NoGems(t *testing.T) {
	for _, test := range []struct {
		name     string
		location references.Location
		x, y     int
		view     func(*GameState) bool
	}{
		{name: "small map", location: references.LordBritishsCastle, x: 12, y: 12, view: (*GameState).ActionViewSmallMap},
		{name: "large map", location: references.BritanniaUnderworld, x: 100, y: 100, view: (*GameState).ActionViewLargeMap},
	} {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(test.location).
			WithPlayerAt(test.x, test.y).
			WithSystemCallbacks().
			Build()
		mockCallbacks.Reset()
		gs.PartyState.Inventory.Provisions.Gems.Set(0)

		assert.False(t, test.view(gs), test.name)
		assert.Equal(t, []string{"You have none!"}, mockCallbacks.Messages, test.name)
		assert.False(t, mockCallbacks.ShowGemViewInteractionCalled, test.name)
	}
}

// TestViewCommand_LargeMap_WithGems tests V command on large map with gems
func TestViewCommand_LargeMap_WithGems(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	t.Run("ViewConsumesGemOnLargeMap", func(t *testing.T) {
		mockCallbacks.Reset()

		// Give party some gems
		gs.PartyState.Inventory.Provisions.Gems.Set(3)
		initialGems := gs.PartyState.Inventory.Provisions.Gems.Get()

		result := gs.ActionViewLargeMap()

		// Should succeed
		assert.True(t, result, "View should succeed with gems on large map")

		// Should have consumed one gem
		assert.Equal(t, initialGems-1, gs.PartyState.Inventory.Provisions.Gems.Get(),
			"Should consume exactly one gem on large map")

		// Should have triggered ShowGemViewInteraction callback
		assert.True(t, mockCallbacks.ShowGemViewInteractionCalled,
			"Should trigger ShowGemViewInteraction callback on large map")
		assert.Equal(t, []int{1}, mockCallbacks.TimeAdvanced())
	})
}

// TestViewCommand_MultipleUses tests consuming multiple gems sequentially
func TestViewCommand_MultipleUses(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithPlayerAt(12, 12).
		WithSystemCallbacks().
		Build()

	t.Run("SequentialViewUses", func(t *testing.T) {
		// Give party 3 gems
		gs.PartyState.Inventory.Provisions.Gems.Set(3)

		// Use view 3 times
		for i := 0; i < 3; i++ {
			mockCallbacks.Reset()
			result := gs.ActionViewSmallMap()
			assert.True(t, result, "View %d should succeed", i+1)
		}

		// Should have 0 gems left
		assert.Equal(t, uint16(0), gs.PartyState.Inventory.Provisions.Gems.Get(),
			"Should have zero gems after 3 uses")

		// Fourth use should fail
		mockCallbacks.Reset()
		result := gs.ActionViewSmallMap()
		assert.False(t, result, "View should fail with no gems remaining")
		assert.Equal(t, "You have none!", mockCallbacks.Messages[0],
			"Should show 'You have none!' on fourth attempt")
	})
}
