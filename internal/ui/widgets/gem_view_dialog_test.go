package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGemViewOriginMatchesOriginalSurfaceWindow(t *testing.T) {
	tests := []struct {
		player int
		want   int
	}{
		{player: 0, want: 240},
		{player: 7, want: 240},
		{player: 8, want: 0},
		{player: 23, want: 0},
		{player: 24, want: 16},
		{player: 255, want: 240},
	}

	for _, test := range tests {
		assert.Equal(t, test.want, gemViewOrigin(test.player, true))
	}
	assert.Zero(t, gemViewOrigin(24, false), "town view starts at the map origin")
}

func TestGemViewWorldAndMarkerPositionsShareWrappedOrigin(t *testing.T) {
	dialog := &GemViewDialog{ //nolint:exhaustruct // Pure coordinate fixture; rendering resources are unnecessary.
		playerWorldX: 2,
		playerWorldY: 250,
		viewOriginX:  240,
		viewOriginY:  240,
		wrapMap:      true,
	}

	x, y := dialog.worldPosition(31, 31)
	assert.Equal(t, 15, x)
	assert.Equal(t, 15, y)
	markerX, markerY, visible := dialog.markerGridPosition()
	assert.True(t, visible)
	assert.Equal(t, 18, markerX)
	assert.Equal(t, 10, markerY)
}

func TestGemViewMarkerRejectsPositionsOutsideSurfaceWindow(t *testing.T) {
	dialog := &GemViewDialog{ //nolint:exhaustruct // Pure coordinate fixture; rendering resources are unnecessary.
		playerWorldX: 40,
		playerWorldY: 10,
	}

	_, _, visible := dialog.markerGridPosition()
	assert.False(t, visible)
}

func TestGemViewRiverMasksMatchOriginalQuadrants(t *testing.T) {
	assert.Equal(t, uint8(0b0101), gemViewRiverWaterMask(0x60))
	assert.Equal(t, uint8(0b1110), gemViewRiverWaterMask(0x69))
	assert.Equal(t, uint8(0b1111), gemViewRiverWaterMask(0x03), "non-river plotv10 tiles are all water")
}

func TestGemViewRoadMasksAndCutoutsMatchOriginalTiles(t *testing.T) {
	mask, ok := gemViewRoadConnectionMask(0x22)
	assert.True(t, ok)
	assert.Equal(t, uint8(0b1100), mask)
	x, y, hasCutout := gemViewRoadCutout(0x22)
	assert.True(t, hasCutout)
	assert.Equal(t, 1, x)
	assert.Equal(t, 2, y)

	_, ok = gemViewRoadConnectionMask(0x27)
	assert.False(t, ok)
	_, _, hasCutout = gemViewRoadCutout(0x20)
	assert.False(t, hasCutout)
}
