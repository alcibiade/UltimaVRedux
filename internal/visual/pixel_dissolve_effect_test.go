package visual

import (
	"image"
	"testing"

	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
	"github.com/stretchr/testify/assert"
)

func TestPixelDissolveEffectUsesStableSeededReveal(t *testing.T) {
	const (
		size = 8
		seed = 0x554C54494D41
	)
	random := beradrng.New(seed)
	first := NewPixelDissolveEffect(
		size, size, size, size, DissolveModeReveal,
		random,
	)
	second := NewPixelDissolveEffect(
		size, size, size, size, DissolveModeReveal,
		beradrng.New(seed),
	)
	t.Cleanup(first.Dispose)
	t.Cleanup(second.Dispose)

	first.UpdateProgress(0.5)
	second.UpdateProgress(0.5)

	assert.Equal(t, first.pixelData, second.pixelData)
	assert.Positive(t, random.Calls(), "the adapter must pass its supplied presentation source to the SDK")
}

type orderedProjectionRandom struct{}

func (orderedProjectionRandom) Intn(n int) int { return n - 1 }

func TestPixelDissolveProjectsLeadingCellsIntoTargetCoordinates(t *testing.T) {
	// Uneven scaling exercises integer boundaries independently on each axis.
	effect := NewPixelDissolveEffect(13, 7, 4, 2, DissolveModeReveal, orderedProjectionRandom{})
	defer effect.Dispose()
	effect.UpdateProgress(0.5)
	var projected []image.Rectangle
	effect.VisitLeadingPixels(.25, func(rect image.Rectangle) { projected = append(projected, rect) })
	assert.Equal(t, []image.Rectangle{image.Rect(6, 0, 9, 3), image.Rect(9, 0, 13, 3)}, projected)
}
