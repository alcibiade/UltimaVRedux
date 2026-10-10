package widgets

import (
	"image"
	"image/color"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestBorderLayoutDefersTexturesAndRetainsCapturedSize(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	border := NewOpaqueBorder(sprites.PercentBasedPlacement{
		StartPercentX: 0.25, EndPercentX: 0.5, StartPercentY: 0.25, EndPercentY: 0.5,
	}, BorderWidthStandard, color.RGBA{R: 29, G: 47, B: 61, A: 255})
	t.Cleanup(border.Dispose)
	require.Equal(t, image.Rect(320, 180, 640, 360), border.GetBounds())
	require.Nil(t, border.border, "measuring a pane must not allocate its frame")
	require.Nil(t, border.background, "measuring a pane must not allocate its background")
	display.GetManager().PublishLogicalSize(1920, 1080)
	target := ebiten.NewImageWithOptions(border.GetBounds().Inset(-2), &ebiten.NewImageOptions{Unmanaged: true})
	t.Cleanup(target.Deallocate)
	border.Draw(target)
	require.Equal(t, image.Rect(320, 180, 640, 360), border.GetBounds(), "lazy drawing must preserve construction-time placement")
	require.Equal(t, image.Pt(320, 180), border.border.Bounds().Size())
	pixels := make([]byte, 4)
	target.SubImage(image.Rect(480, 270, 481, 271)).(*ebiten.Image).ReadPixels(pixels)
	require.Equal(t, []byte{29, 47, 61, 255}, pixels)
	shared := border.border
	border.Dispose()
	border.Dispose()
	require.Nil(t, border.background)
	require.Same(t, shared, border.border, "disposing owned background must not dispose shared frame")
	border.Draw(target)
	require.NotNil(t, border.background, "drawing after background release must recreate it")
}
