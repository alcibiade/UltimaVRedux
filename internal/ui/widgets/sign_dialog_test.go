package widgets

import (
	"image"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/stretchr/testify/require"
)

func TestSignSameInstanceReflowRetainsSourceBytes(t *testing.T) {
	setGuidanceTestResolution(t, 3840, 2160)
	raw := []byte{'A', 0xC1, '\n', 'b'}
	dialog := NewSignDialog(SignDialogConfig{
		Placement: sprites.PercentBasedPlacement{StartPercentX: .1, EndPercentX: .65, StartPercentY: .2, EndPercentY: .8},
		RawText:   raw,
	})
	original := append([]byte(nil), raw...)
	raw[1] = 'B'
	for _, size := range []image.Point{{1280, 720}, {3840, 2160}} {
		previousBuffer := dialog.buffer
		display.GetManager().PublishLogicalSize(size.X, size.Y)
		dialog.Update()
		require.Equal(t, original, dialog.rawText, "retain the original Runic high bit")
		require.True(t, dialog.bounds.In(image.Rectangle{Max: size}))
		require.NotSame(t, previousBuffer, dialog.buffer)
		current := dialog.buffer
		dialog.Update()
		require.Same(t, current, dialog.buffer, "stable-size update must reuse rendering")
		require.False(t, dialog.IsDismissed())
	}
}

func TestFittingSignScaleUsesLargestWholePixelScale(t *testing.T) {
	tests := []struct {
		name     string
		interior image.Rectangle
		want     int
	}{
		{name: "720p dialog interior", interior: image.Rect(0, 0, 662, 408), want: 3},
		{name: "1080p dialog interior", interior: image.Rect(0, 0, 994, 610), want: 5},
		{name: "1440p dialog interior", interior: image.Rect(0, 0, 1324, 814), want: 6},
		{name: "4k dialog interior", interior: image.Rect(0, 0, 1986, 1220), want: 10},
		{name: "undersized dialog", interior: image.Rect(0, 0, 100, 80), want: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := fittingSignScale(test.interior); got != test.want {
				t.Errorf("fittingSignScale(%v) = %d, want %d", test.interior, got, test.want)
			}
		})
	}
}

func TestFittingSignScaleLeavesOneCharacterCellPadding(t *testing.T) {
	interior := image.Rect(0, 0, 662, 408)
	scale := fittingSignScale(interior)
	canvasWidth := (signCanvasColumns + signCanvasPaddingCells*2) * text.BitmapCharWidth
	canvasHeight := (signCanvasRows + signCanvasPaddingCells*2) * text.BitmapCharHeight

	if canvasWidth*scale > interior.Dx() || canvasHeight*scale > interior.Dy() {
		t.Fatalf("maximum sign canvas plus padding does not fit at scale %d", scale)
	}
	if canvasWidth*(scale+1) <= interior.Dx() && canvasHeight*(scale+1) <= interior.Dy() {
		t.Errorf("scale %d was not the largest fitting whole-number scale", scale)
	}
}

func TestSignDialogCentersArtworkInsideBorderInterior(t *testing.T) {
	placement := sprites.PercentBasedPlacement{
		StartPercentX: 0.10,
		EndPercentX:   0.65,
		StartPercentY: 0.20,
		EndPercentY:   0.80,
	}
	rawText := []byte("WELCOME")
	dialog := NewSignDialog(SignDialogConfig{Placement: placement, RawText: rawText})
	interior := dialog.border.GetInteriorBounds()
	scale := fittingSignScale(interior)
	result := text.NewSignRenderer(float64(scale)).RenderSign(rawText)
	wantX := interior.Min.X - dialog.bounds.Min.X + (interior.Dx()-result.Width)/centerDivisor
	wantY := interior.Min.Y - dialog.bounds.Min.Y + (interior.Dy()-result.Height)/centerDivisor

	if dialog.signOffsetX != wantX || dialog.signOffsetY != wantY {
		t.Errorf(
			"sign offset = (%d,%d), want interior-centered (%d,%d)",
			dialog.signOffsetX,
			dialog.signOffsetY,
			wantX,
			wantY,
		)
	}
}

func TestSignDialogRendersWarmSemanticColours(t *testing.T) {
	placement := sprites.PercentBasedPlacement{
		StartPercentX: 0.10,
		EndPercentX:   0.65,
		StartPercentY: 0.20,
		EndPercentY:   0.80,
	}
	dialog := NewSignDialog(SignDialogConfig{
		Placement: placement,
		RawText:   []byte{'d', 'f', ':', ';', 'n', 'A', 0xC1},
		Scale:     1,
	})
	expectedColors := []ucolor.EGAColor{
		ucolor.EGABrown,
		ucolor.EGABrown,
		ucolor.EGABrown,
		ucolor.EGABrown,
		ucolor.EGABrown,
		ucolor.EGAYellow,
		ucolor.EGAWhite,
	}

	for column, expected := range expectedColors {
		foundGlyphPixel := false
		wantRed, wantGreen, wantBlue, _ := expected.RGBA()
		for y := dialog.signOffsetY; y < dialog.signOffsetY+text.BitmapCharHeight; y++ {
			startX := dialog.signOffsetX + column*text.BitmapCharWidth
			for x := startX; x < startX+text.BitmapCharWidth; x++ {
				red, green, blue, _ := dialog.buffer.At(x, y).RGBA()
				if red == 0 && green == 0 && blue == 0 {
					continue
				}
				foundGlyphPixel = true
				if red != wantRed || green != wantGreen || blue != wantBlue {
					t.Fatalf(
						"role column %d pixel (%d,%d) = (%d,%d,%d), want (%d,%d,%d)",
						column, x, y, red, green, blue, wantRed, wantGreen, wantBlue,
					)
				}
			}
		}
		if !foundGlyphPixel {
			t.Fatalf("font column %d rendered no glyph pixels", column)
		}
	}
}
