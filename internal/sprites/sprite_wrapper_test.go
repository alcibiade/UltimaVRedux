package sprites

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandardNPCPipelinePreservesPreparedBorderGeometry(t *testing.T) {
	source := ebiten.NewImage(16, 16)
	t.Cleanup(source.Deallocate)
	source.Set(0, 0, color.RGBA{R: 80, A: 128})
	bordered := WrapSpriteWith18x18Border(source)
	t.Cleanup(bordered.Deallocate)
	processor := NewTileEffectProcessor(nil)
	t.Cleanup(processor.graphics.Close)
	outlined := processor.ApplyDynamic(bordered, StandardNPCPipeline)
	require.Equal(t, image.Rect(0, 0, 18, 18), outlined.Bounds(), "outline stage must not add a second border")
	for y := range 18 {
		for x := range 18 {
			want := color.RGBA{}
			if x < 3 && y < 3 {
				want.A = 255
			}
			if x == 1 && y == 1 {
				want = color.RGBA{R: 80, A: 128}
			}
			require.Equal(t, want, color.RGBAModel.Convert(outlined.At(x, y)), "pixel %d,%d", x, y)
		}
	}
	require.Equal(t, color.RGBA{R: 80, A: 128}, color.RGBAModel.Convert(source.At(0, 0)))
	require.Equal(t, image.Rect(0, 0, 16, 16), processor.ApplyDynamic(source, StandardNPCPipeline).Bounds(),
		"padding is explicitly prepared by the caller, not inferred by outlining")
}

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
