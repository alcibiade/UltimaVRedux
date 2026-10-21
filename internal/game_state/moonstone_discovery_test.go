package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestMoonstoneDiscoveryWorkflow tests the complete moonstone discovery and pickup workflow
func TestMoonstoneDiscoveryWorkflow(t *testing.T) {
	// Initialize game state with moonstone system
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld). // Large map with moonstones
		WithPlayerAt(100, 100).                       // Generic safe position
		WithSystemCallbacks().
		Build()

	// Ensure moonstone system is initialized
	if gs.moonstoneSystem == nil {
		gs.moonstoneSystem = NewMoonstoneSystem()
	}

	// Find any available moonstone for testing
	var testMoonstoneIndex int = -1
	var testMoonstonePos *references.Position

	for i, moonstone := range gs.moonstoneSystem.Moonstones {
		if moonstone.IsBuried() {
			testMoonstoneIndex = i
			testMoonstonePos = &references.Position{X: references.Coordinate(moonstone.X), Y: references.Coordinate(moonstone.Y)}
			break
		}
	}

	if testMoonstoneIndex < 0 {
		t.Skip("No buried moonstones available for testing")
	}

	// Position player next to the moonstone
	gs.MapState.PlayerLocation.Position = references.Position{
		X: testMoonstonePos.X - 1,
		Y: testMoonstonePos.Y,
	}

	// Verify moonstone is initially buried
	moonstone := gs.moonstoneSystem.GetMoonstoneByIndex(testMoonstoneIndex)
	assert.NotNil(t, moonstone)
	assert.True(t, moonstone.IsBuried())
	assert.False(t, moonstone.IsFound)

	// Set nighttime for moongate visibility
	gs.DateTime.Hour = 22 // 10 PM - moongate should be visible

	// Test the moonstone discovery system directly (simulate successful search at moongate)
	mockCallbacks.Reset()

	// Manually trigger moonstone discovery to test the system
	gs.moonstoneSystem.MarkMoonstoneFoundAtPosition(testMoonstoneIndex)

	// Verify moonstone state changed to FoundNotCollected
	moonstone = gs.moonstoneSystem.GetMoonstoneByIndex(testMoonstoneIndex)
	assert.True(t, moonstone.IsFound)
	assert.True(t, moonstone.IsFoundNotCollected())
	assert.True(t, moonstone.CanCreateMoongate())

	// Add moonstone item to ItemStacksMap to simulate the search discovery
	moonstoneItem := references.Moonstone(testMoonstoneIndex)
	itemAndQuantity := references.ItemAndQuantity{
		Item:     moonstoneItem,
		Quantity: 1,
	}
	itemStack := &references.ItemStack{
		Items: []references.ItemAndQuantity{itemAndQuantity},
	}
	err := gs.ItemStacksMap.Push(testMoonstonePos, itemStack)
	assert.NoError(t, err)

	// Verify Moon sprite was added to world (ItemStacksMap)
	assert.True(t, gs.ItemStacksMap.HasItemStackAtPosition(testMoonstonePos))

	// Position player to get the moonstone
	gs.MapState.PlayerLocation.Position = references.Position{
		X: testMoonstonePos.X - 1,
		Y: testMoonstonePos.Y,
	}

	// Test getting the moonstone
	mockCallbacks.Reset()
	getResult := gs.ActionGetLargeMap(references.Right) // Get moonstone to the right
	assert.True(t, getResult)
	assert.True(t, len(mockCallbacks.Messages) > 0)
	assert.Equal(t, "A moonstone!", mockCallbacks.Messages[0])

	// Verify moonstone is now in inventory state
	moonstone = gs.moonstoneSystem.GetMoonstoneByIndex(testMoonstoneIndex)
	assert.True(t, moonstone.IsInInventory())
	assert.False(t, moonstone.IsFoundNotCollected())

	// Verify Moon sprite was removed from world
	assert.False(t, gs.ItemStacksMap.HasItemStackAtPosition(testMoonstonePos))
}

// TestMoonstoneStateTransitions tests the state transition logic
func TestMoonstoneStateTransitions(t *testing.T) {
	ms := NewMoonstoneSystem()

	// Test initial state
	moonstone := ms.GetMoonstoneByIndex(0)
	assert.True(t, moonstone.IsBuried())
	assert.False(t, moonstone.IsFound)
	assert.True(t, moonstone.CanCreateMoongate())

	// Test MarkMoonstoneFoundAtPosition
	ms.MarkMoonstoneFoundAtPosition(0)
	moonstone = ms.GetMoonstoneByIndex(0)
	assert.True(t, moonstone.IsFound)
	assert.True(t, moonstone.IsFoundNotCollected())
	assert.True(t, moonstone.CanCreateMoongate())

	// Test TakeMoonstoneToInventory
	ms.TakeMoonstoneToInventory(0)
	moonstone = ms.GetMoonstoneByIndex(0)
	assert.True(t, moonstone.IsInInventory())
	assert.False(t, moonstone.IsFoundNotCollected())
	assert.False(t, moonstone.CanCreateMoongate()) // Pickup removes gate eligibility
}
