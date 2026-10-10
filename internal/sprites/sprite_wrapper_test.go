package sprites

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapSpriteWith18x18Border_NilSafety(t *testing.T) {
	result := WrapSpriteWith18x18Border(nil)
	assert.Nil(t, result, "Should handle nil sprite gracefully")
}

func TestConvertPosition16to18(t *testing.T) {
	testCases := []struct {
		name      string
		tileX     int
		tileY     int
		expectedX float64
		expectedY float64
	}{
		{
			name:      "Origin tile",
			tileX:     0,
			tileY:     0,
			expectedX: -1,
			expectedY: -1,
		},
		{
			name:      "Tile at position 1,1",
			tileX:     1,
			tileY:     1,
			expectedX: 15, // 1*16 - 1
			expectedY: 15, // 1*16 - 1
		},
		{
			name:      "Tile at position 5,5 (center)",
			tileX:     5,
			tileY:     5,
			expectedX: 79, // 5*16 - 1
			expectedY: 79, // 5*16 - 1
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			x, y := ConvertPosition16to18(tc.tileX, tc.tileY)
			assert.Equal(t, tc.expectedX, x, "X position should match")
			assert.Equal(t, tc.expectedY, y, "Y position should match")
		})
	}
}
