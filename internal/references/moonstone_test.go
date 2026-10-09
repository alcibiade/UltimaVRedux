package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestInitialMoonstoneData_BasicStructure tests the basic moonstone reference data structure
func TestInitialMoonstoneData_BasicStructure(t *testing.T) {
	location := InitialMoonstoneData{
		X: 224, Y: 133, Map: 0, Level: 0,
	}

	assert.Equal(t, uint8(224), location.X)
	assert.Equal(t, uint8(133), location.Y)
	assert.Equal(t, uint8(0), location.Map)
	assert.Equal(t, uint8(0), location.Level)
}

// TestInitialMoonstoneLocations_InitialPositions validates the 8 original moonstone starting locations
func TestInitialMoonstoneLocations_InitialPositions(t *testing.T) {
	locations := InitialMoonstoneLocations

	// Verify we have exactly 8 moonstones
	assert.Len(t, locations, 8, "Should have exactly 8 initial moonstone locations")

	// Test each initial position from original Ultima V source
	expectedPositions := []struct{ x, y uint8 }{
		{0xe0, 0x85}, // Moonstone 0: (224, 133)
		{0x60, 0x66}, // Moonstone 1: (96, 102)
		{0x26, 0xe0}, // Moonstone 2: (38, 224)
		{0x32, 0x25}, // Moonstone 3: (50, 37)
		{0xa6, 0x13}, // Moonstone 4: (166, 19)
		{0x68, 0xc2}, // Moonstone 5: (104, 194)
		{0x17, 0x7e}, // Moonstone 6: (23, 126)
		{0xbb, 0xa7}, // Moonstone 7: (187, 167)
	}

	for i, expected := range expectedPositions {
		assert.Equal(t, expected.x, locations[i].X, "Moonstone %d X coordinate should match", i)
		assert.Equal(t, expected.y, locations[i].Y, "Moonstone %d Y coordinate should match", i)
		assert.Equal(t, uint8(0), locations[i].Map, "All initial moonstones should be on map 0 (overworld)")
		assert.Equal(t, uint8(0), locations[i].Level, "All initial moonstones should be at level 0")
	}
}
