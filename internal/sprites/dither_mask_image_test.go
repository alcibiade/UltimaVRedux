package sprites

import (
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

type spritesTestGame struct {
	tests *testing.M
	code  int
	done  chan int
}

func (g *spritesTestGame) Update() error {
	if g.done == nil {
		g.done = make(chan int, 1)
		// Let frame ends release native upload buffers between test operations.
		go func() { g.done <- g.tests.Run() }()
	}
	select {
	case g.code = <-g.done:
		return ebiten.Termination
	default:
		return nil
	}
}

func (*spritesTestGame) Draw(*ebiten.Image) {}

func (*spritesTestGame) Layout(_, _ int) (int, int) {
	return 1, 1
}

func TestMain(tests *testing.M) {
	// RunGame performs lazy native initialization on the main thread first.
	runner := &spritesTestGame{tests: tests, code: 1, done: nil}
	if err := ebiten.RunGame(runner); err != nil {
		panic(err)
	}
	if runner.code != 0 {
		os.Exit(runner.code)
	}
}

func TestDitherMaskImageResizeReuseAndDispose(t *testing.T) {
	var mask DitherMaskImage
	t.Cleanup(mask.Dispose)
	mask.Dispose()
	mask.Dispose()
	require.Nil(t, mask.surface)
	pixels := mask.Resize(2, 3)
	require.Equal(t, make([]byte, 24), pixels)
	owner := mask.surface
	mask.Dispose()
	mask.Dispose()
	require.Nil(t, mask.surface)
	require.Equal(t, make([]byte, 4), mask.Resize(1, 1))
	require.NotSame(t, owner, mask.surface, "game Dispose recreates a fresh SDK surface on next Resize")
	mask.Upload()
}

func TestDitherMaskImageResizeInvalid(t *testing.T) {
	var mask DitherMaskImage
	t.Cleanup(mask.Dispose)
	pixels := mask.Resize(1, 1)
	pixels[3] = 255
	originalImage := mask.surface.Image()

	for _, dimensions := range []image.Point{
		{X: 0, Y: 1}, {X: 1, Y: 0}, {X: 0, Y: 0},
		{X: -1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: -1},
	} {
		t.Run(dimensions.String(), func(t *testing.T) {
			require.Panics(t, func() { mask.Resize(dimensions.X, dimensions.Y) })
			require.Same(t, originalImage, mask.surface.Image())
			require.Equal(t, []byte{0, 0, 0, 255}, pixels)
		})
	}
}

func TestDitherMaskImageDrawSourceOver(t *testing.T) {
	var mask DitherMaskImage
	t.Cleanup(mask.Dispose)
	pixels := mask.Resize(4, 1)
	copy(pixels, []byte{
		0, 0, 0, 0,
		0, 0, 0, 255,
		0, 0, 0, 128,
		32, 48, 64, 255,
	})
	mask.Upload()

	worldColor := color.RGBA{R: 64, G: 128, B: 192, A: 255}
	target := ebiten.NewImage(4, 1)
	t.Cleanup(target.Deallocate)
	target.Fill(worldColor)
	mask.Draw(target, ebiten.GeoM{})
	actual := make([]byte, 16)
	target.ReadPixels(actual)
	expected := []byte{
		64, 128, 192, 255,
		0, 0, 0, 255,
		32, 64, 96, 255,
		32, 48, 64, 255,
	}
	require.Equal(t, expected, actual, "transparent keeps world; black rejects; color is unmodulated")

	mask.Resize(4, 1)
	target.Fill(worldColor)
	mask.Draw(target, ebiten.GeoM{})
	target.ReadPixels(actual)
	require.Equal(t, expected, actual, "clearing the CPU buffer does not upload implicitly")

	mask.Upload()
	target.Fill(worldColor)
	mask.Draw(target, ebiten.GeoM{})
	target.ReadPixels(actual)
	require.Equal(t, []byte{
		64, 128, 192, 255, 64, 128, 192, 255,
		64, 128, 192, 255, 64, 128, 192, 255,
	}, actual, "upload replaces previous rejection pixels with transparency")
}

func TestDitherMaskImageDrawNearestTranslatedSubImage(t *testing.T) {
	var mask DitherMaskImage
	t.Cleanup(mask.Dispose)
	pixels := mask.Resize(3, 2)
	for y := range 2 {
		for x := range 3 {
			if (x+y)%2 == 0 {
				pixels[(y*3+x)*4+3] = 255
			}
		}
	}
	mask.Upload()

	worldColor := color.RGBA{R: 80, G: 120, B: 200, A: 255}
	target := ebiten.NewImage(12, 10)
	t.Cleanup(target.Deallocate)
	target.Fill(worldColor)
	clipBounds := image.Rect(4, 3, 8, 5)
	clipped, ok := target.SubImage(clipBounds).(*ebiten.Image)
	require.True(t, ok)
	var transform ebiten.GeoM
	transform.Scale(2, 2)
	transform.Translate(3, 2)
	originalTransform := transform
	mask.Draw(clipped, transform)
	require.Equal(t, originalTransform, transform)

	actual := make([]byte, 12*10*4)
	target.ReadPixels(actual)
	for y := range 10 {
		for x := range 12 {
			expected := worldColor
			if image.Pt(x, y).In(clipBounds) && ((x-3)/2+(y-2)/2)%2 == 0 {
				expected = color.RGBA{A: 255}
			}
			offset := (y*12 + x) * 4
			require.Equal(t, []byte{expected.R, expected.G, expected.B, expected.A}, actual[offset:offset+4],
				"pixel (%d, %d): nearest scaling must keep crisp pixels inside the caller's clip only", x, y)
		}
	}
}
