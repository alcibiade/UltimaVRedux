package sprites

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestDynamicOutlineOnlyHollowFaceBackingAndSourceImmutability(t *testing.T) {
	processor := &TileEffectProcessor{graphics: newGraphicsProcessor(2)}
	// Nonzero atlas origin and partially transparent body pixels must both work.
	atlas := ebiten.NewImage(18, 18)
	t.Cleanup(atlas.Dispose)
	for y := 7; y < 10; y++ {
		for x := 7; x < 10; x++ {
			atlas.Set(x, y, color.RGBA{R: 80, A: 128})
		}
	}
	source := atlas.SubImage(image.Rect(4, 4, 13, 13)).(*ebiten.Image)
	before := outlineTestPixels(source)
	face := ucolor.CombatActivePCOutline.ToRGBA()
	contour := processor.ApplyDynamicOutlineOnly(source, face)
	require.NotNil(t, contour)
	assert.Equal(t, image.Rect(0, 0, 9, 9), contour.Bounds())
	for y := range 9 {
		for x := range 9 {
			want := color.RGBA{}
			point := image.Pt(x, y)
			switch {
			case point.In(image.Rect(3, 3, 6, 6)):
				// Every original body pixel is transparent, not recoloured.
			case point.In(image.Rect(2, 2, 7, 7)):
				want = face
			case point.In(image.Rect(1, 1, 8, 8)):
				want = ucolor.EGABlack.ToRGBA()
			}
			assert.Equal(t, want, color.RGBAModel.Convert(contour.At(x, y)), "pixel (%d,%d)", x, y)
		}
	}
	assert.Equal(t, before, outlineTestPixels(source))
	assert.Same(t, contour, processor.ApplyDynamicOutlineOnly(source, face))
}

func TestDynamicOutlineOnlyCacheSeparatesFramesColoursAndBodyEffects(t *testing.T) {
	processor := &TileEffectProcessor{graphics: newGraphicsProcessor(2)}
	first := ebiten.NewImage(18, 18)
	second := ebiten.NewImage(18, 18)
	t.Cleanup(first.Dispose)
	t.Cleanup(second.Dispose)
	first.Set(8, 8, ucolor.EGARed)
	second.Set(9, 8, ucolor.EGARed)
	face := ucolor.CombatActivePCOutline.ToRGBA()
	a := processor.ApplyDynamicOutlineOnly(first, face)
	b := processor.ApplyDynamicOutlineOnly(second, face)
	require.NotSame(t, a, b)
	assert.NotEqual(t, outlineTestPixels(a), outlineTestPixels(b))
	assert.Same(t, a, processor.ApplyDynamicOutlineOnly(first, face))
	c := processor.ApplyDynamicOutlineOnly(first, ucolor.CombatActiveMonsterOutline.ToRGBA())
	assert.NotSame(t, a, c)
	assert.NotEqual(t, outlineTestPixels(a), outlineTestPixels(c))
	assert.Equal(t, 2, processor.graphics.Stats().Derived)
	assert.Same(t, a, processor.ApplyDynamicOutlineOnly(first, face), "recent entry survives eviction")
	bAgain := processor.ApplyDynamicOutlineOnly(second, face)
	assert.NotSame(t, b, bAgain, "least recently used frame was evicted")
	assert.Equal(t, outlineTestPixels(b), outlineTestPixels(bAgain))
	body := processor.ApplyDynamicSimple(first, CombatActivePCOutline)
	assert.NotZero(t, outlineTestPixels(body)[(8*18+8)*4+3])
	assert.Zero(t, outlineTestPixels(a)[(8*18+8)*4+3])
	assert.Equal(t, 2, processor.graphics.Stats().Derived)
}

func TestDynamicOutlineOnlyNeverFallsBackToBody(t *testing.T) {
	processor := &TileEffectProcessor{graphics: newGraphicsProcessor(2)}
	face := ucolor.CombatActivePCOutline.ToRGBA()
	assert.Nil(t, processor.ApplyDynamicOutlineOnly(nil, face))
	for _, size := range []int{16, 18} {
		source := ebiten.NewImage(size, size)
		source.Fill(ucolor.EGARed)
		assert.Nil(t, processor.ApplyDynamicOutlineOnly(source, color.RGBA{}))
		contour := processor.ApplyDynamicOutlineOnly(source, face)
		require.NotNil(t, contour)
		assert.NotSame(t, source, contour)
		assert.Equal(t, make([]byte, size*size*4), outlineTestPixels(contour), "solid source has no room for an outer contour")
		source.Dispose()
		assert.Nil(t, processor.ApplyDynamicOutlineOnly(source, face), "disposed sources cannot reuse a cached contour")
		processor.graphics.Clear()
		assert.Nil(t, processor.ApplyDynamicOutlineOnly(source, face), "disposed Bounds must fail closed on a cache miss too")
		assert.Zero(t, processor.graphics.Stats().Derived, "failed source reads do not populate the cache")
	}
}

func TestDynamicOutlineOnlyDoesNotRecoverProgrammingErrors(t *testing.T) {
	source := ebiten.NewImage(18, 18)
	t.Cleanup(source.Dispose)
	source.Set(8, 8, ucolor.EGARed)
	processor := &TileEffectProcessor{}
	assert.Panics(t, func() {
		processor.ApplyDynamicOutlineOnly(source, ucolor.CombatActivePCOutline.ToRGBA())
	}, "missing mandatory cache initialization is not an unreadable-source fallback")
}

func outlineTestPixels(img *ebiten.Image) []byte {
	pixels := make([]byte, img.Bounds().Dx()*img.Bounds().Dy()*4)
	img.ReadPixels(pixels)

	return pixels
}
