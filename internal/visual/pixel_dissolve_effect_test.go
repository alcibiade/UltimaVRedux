package visual

import (
	"image"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/stretchr/testify/assert"
)

func TestPixelDissolveEffectUsesStableSeededReveal(t *testing.T) {
	const (
		size = 8
		seed = 0x554C54494D41
	)
	first := NewPixelDissolveEffect(
		size, size, size, size, DissolveModeReveal,
		deterministic_random.New(seed),
	)
	second := NewPixelDissolveEffect(
		size, size, size, size, DissolveModeReveal,
		deterministic_random.New(seed),
	)
	t.Cleanup(first.Dispose)
	t.Cleanup(second.Dispose)

	first.UpdateProgress(0.5)
	second.UpdateProgress(0.5)

	assert.Equal(t, first.pixelData, second.pixelData)
	revealed := 0
	for index := 3; index < len(first.pixelData); index += 4 {
		if first.pixelData[index] != 0 {
			revealed++
		}
	}
	assert.Equal(t, size*size/2, revealed)
}

func TestPixelDissolveLeadingPixelsStayWithinRevealedFrontAndReset(t *testing.T) {
	effect := NewPixelDissolveEffect(16, 16, 8, 8, DissolveModeReveal, deterministic_random.New(42))
	defer effect.Dispose()
	leading := func() []image.Rectangle {
		var pixels []image.Rectangle
		effect.VisitLeadingPixels(0.125, func(rect image.Rectangle) { pixels = append(pixels, rect) })
		return pixels
	}
	assert.Empty(t, leading())
	effect.UpdateProgress(0.5)
	before := append([]byte(nil), effect.pixelData...)
	pixels := leading()
	assert.Len(t, pixels, 8)
	for _, rect := range pixels {
		assert.Equal(t, image.Pt(2, 2), rect.Size())
		point := image.Pt(rect.Min.X/2, rect.Min.Y/2)
		assert.Equal(t, uint8(255), effect.pixelData[(point.Y*8+point.X)*4+3])
	}
	assert.Equal(t, pixels, leading(), "repeated draws do not change pixel order")
	assert.Equal(t, before, effect.pixelData, "highlight query does not change the ordinary mask")
	effect.UpdateProgress(1)
	assert.Empty(t, leading(), "complete silhouettes have settled to normal colours")
	effect.Reset()
	assert.Empty(t, leading())
	effect.UpdateProgress(0.5)
	assert.Equal(t, pixels, leading())
	hide := NewPixelDissolveEffect(8, 8, 8, 8, DissolveModeHide, deterministic_random.New(42))
	defer hide.Dispose()
	hide.UpdateProgress(0.5)
	hide.VisitLeadingPixels(1, func(image.Rectangle) { t.Error("hide mode must not opt into reveal highlights") })
}
