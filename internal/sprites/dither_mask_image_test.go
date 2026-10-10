package sprites

import (
	"bytes"
	"fmt"
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
	require.Nil(t, mask.image)
	require.Nil(t, mask.pixels)
	require.Zero(t, mask.width)
	require.Zero(t, mask.height)

	pixels := mask.Resize(2, 3)
	require.Equal(t, make([]byte, 24), pixels)
	require.Equal(t, image.Rect(0, 0, 2, 3), mask.image.Bounds())
	originalImage := mask.image
	for index := range pixels {
		pixels[index] = 255
	}
	mask.Upload()

	reused := mask.Resize(2, 3)
	require.Same(t, originalImage, mask.image)
	require.Same(t, &pixels[0], &reused[0])
	require.Equal(t, make([]byte, 24), reused)
	require.Zero(t, testing.AllocsPerRun(100, func() {
		mask.Resize(2, 3)
	}))

	resized := mask.Resize(3, 2)
	require.NotSame(t, originalImage, mask.image)
	require.NotSame(t, &pixels[0], &resized[0])
	require.Equal(t, make([]byte, 24), resized)
	require.Equal(t, image.Rect(0, 0, 3, 2), mask.image.Bounds())
	require.Equal(t, 3, mask.width)
	require.Equal(t, 2, mask.height)
	deallocated := make([]byte, 24)
	originalImage.ReadPixels(deallocated)
	require.Equal(t, make([]byte, 24), deallocated, "resize releases the previous image contents")
	originalImage.Deallocate()

	resized[3] = 255
	mask.Upload()
	lastImage := mask.image
	mask.Dispose()
	mask.Dispose()
	require.Nil(t, mask.image)
	require.Nil(t, mask.pixels)
	require.Zero(t, mask.width)
	require.Zero(t, mask.height)
	lastImage.ReadPixels(deallocated)
	require.Equal(t, make([]byte, 24), deallocated, "dispose releases image contents")
	lastImage.Deallocate()

	require.Equal(t, make([]byte, 4), mask.Resize(1, 1))
	require.NotSame(t, lastImage, mask.image)
	mask.Upload()
}

func TestDitherMaskImageResizeInvalid(t *testing.T) {
	var mask DitherMaskImage
	t.Cleanup(mask.Dispose)
	pixels := mask.Resize(1, 1)
	pixels[3] = 255
	originalImage := mask.image

	for _, dimensions := range []image.Point{
		{X: 0, Y: 1}, {X: 1, Y: 0}, {X: 0, Y: 0},
		{X: -1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: -1},
	} {
		t.Run(dimensions.String(), func(t *testing.T) {
			require.Panics(t, func() { mask.Resize(dimensions.X, dimensions.Y) })
			require.Same(t, originalImage, mask.image)
			require.Same(t, &pixels[0], &mask.pixels[0])
			require.Equal(t, []byte{0, 0, 0, 255}, mask.pixels)
			require.Equal(t, 1, mask.width)
			require.Equal(t, 1, mask.height)
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

func TestDitherMaskImageDrawFractionalPlacementIndependentOfAtlas(t *testing.T) {
	world := color.RGBA{R: 80, G: 120, B: 200, A: 255}
	for _, size := range []image.Point{{X: 304, Y: 208}, {X: 368, Y: 272}} {
		pixels := make([]byte, size.X*size.Y*4)
		opaquePixels := make([]byte, len(pixels))
		for y := range size.Y {
			for x := range size.X {
				i := (y*size.X + x) * 4
				if (x+2*y)%5 < 2 {
					pixels[i+3] = 255
				} else {
					copy(opaquePixels[i:i+3], []byte{world.R, world.G, world.B})
				}
				opaquePixels[i+3] = 255
			}
		}
		source := ebiten.NewImageWithOptions(image.Rectangle{Max: size}, &ebiten.NewImageOptions{Unmanaged: true})
		t.Cleanup(source.Deallocate)
		source.WritePixels(opaquePixels)
		// Keep these alive together: atlas-backed masks would acquire different source origins.
		var masks [2]DitherMaskImage
		for allocation := range masks {
			mask := &masks[allocation]
			t.Cleanup(mask.Dispose)
			copy(mask.Resize(size.X, size.Y), pixels)
			mask.Upload()
		}
		for allocation := range masks {
			mask := &masks[allocation]
			for index, placement := range []struct{ sx, sy, tx, ty float64 }{
				{1, 1, 19.25, -13.5},
				{1, 1, -9.5, 8.25},
				{1.25, 1.75, 0, 0},
				{2.5025, 701.0 / 208.0, 7.25, -3.5},
			} {
				t.Run(fmt.Sprintf("%v/%d/%d", size, allocation, index), func(t *testing.T) {
					// Real game screens are standalone; differing target atlas origins also perturb ties.
					target := ebiten.NewImageWithOptions(image.Rect(0, 0, 400, 208), &ebiten.NewImageOptions{Unmanaged: true})
					defer target.Deallocate()
					target.Fill(world)
					expected := ebiten.NewImageWithOptions(image.Rect(0, 0, 400, 208), &ebiten.NewImageOptions{Unmanaged: true})
					defer expected.Deallocate()
					expected.Fill(world)
					clip := image.Rect(4, 6, 396, 202)
					clipped, ok := target.SubImage(clip).(*ebiten.Image)
					require.True(t, ok)
					expectedClip, ok := expected.SubImage(clip).(*ebiten.Image)
					require.True(t, ok)
					var transform ebiten.GeoM
					transform.Scale(placement.sx, placement.sy)
					transform.Translate(placement.tx, placement.ty)
					mask.Draw(clipped, transform)
					expectedClip.DrawImage(source, &ebiten.DrawImageOptions{GeoM: transform, Filter: ebiten.FilterNearest})
					actual, want := make([]byte, 400*208*4), make([]byte, 400*208*4)
					target.ReadPixels(actual)
					expected.ReadPixels(want)
					for i := 0; i < len(want); i += 4 {
						if !bytes.Equal(want[i:i+4], actual[i:i+4]) {
							t.Fatalf("pixel %d,%d: want %v, got %v", i/4%400, i/4/400, want[i:i+4], actual[i:i+4])
						}
					}
				})
			}
		}
	}
}
