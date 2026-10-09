package widgets

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestConnectClassicPaneGridNormalizesSharedSeams(t *testing.T) {
	t.Parallel()

	frames := connectClassicPaneGrid([][]image.Rectangle{
		{
			image.Rect(10, 10, 30, 40),
			image.Rect(34, 10, 60, 40),
		},
		{
			image.Rect(10, 44, 60, 70),
		},
	}, 2)

	require.Equal(t, []image.Rectangle{
		image.Rect(10, 10, 34, 44),
		image.Rect(32, 10, 60, 44),
		image.Rect(10, 42, 60, 70),
	}, frames)
}

func TestApplyClassicPixelRoundedMaskClipsOuterCorners(t *testing.T) {
	t.Parallel()

	buffer := ebiten.NewImage(40, 30)
	buffer.Fill(color.RGBA{R: 255, A: 255})
	applyClassicPixelRoundedMask(buffer, 2)

	_, _, _, cornerAlpha := buffer.At(0, 0).RGBA()
	red, _, _, centerAlpha := buffer.At(20, 15).RGBA()
	require.Zero(t, cornerAlpha)
	require.Equal(t, uint32(0xffff), red)
	require.Equal(t, uint32(0xffff), centerAlpha)
}
