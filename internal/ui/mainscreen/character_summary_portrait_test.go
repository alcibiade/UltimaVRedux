package mainscreen

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCharacterPortraitReusePreservesPixelsAndBorrowedSource(t *testing.T) {
	sheet := sprites.NewSpriteSheet()
	summary := NewCharacterSummaryForWidth(sheet, 1280)
	t.Cleanup(summary.Dispose)
	party := summaryBaselineParty()
	source := sheet.GetSprite(party.GetCharacter(0).GetKeySpriteIndex())
	before := summaryPixels(source, source.Bounds())
	var previous *ebiten.Image
	for _, status := range []party_state.CharacterStatus{party_state.Poisoned, party_state.Sleep, party_state.Charmed, party_state.Dead} {
		outlined := summary.prepareCharacterPortrait(0, source, status)
		require.NotSame(t, source, outlined)
		if previous != nil {
			require.NotSame(t, previous, outlined, "status colour changes must replace the outline")
		}
		expected := sheet.GetTileEffectProcessor().ApplyDynamicOutlineRGBA(source, summary.getStatusColor(status), statusOutlineIntensity)
		require.Equal(t, summaryPixels(expected, expected.Bounds()), summaryPixels(outlined, outlined.Bounds()))
		expected.Dispose()
		for range 60 {
			require.Same(t, outlined, summary.prepareCharacterPortrait(0, source, status))
		}
		previous = outlined
	}
	assert.Equal(t, 0.0, testing.AllocsPerRun(20, func() {
		summary.prepareCharacterPortrait(0, source, party_state.Dead)
	}), "unchanged outlines must not generate images or pixel buffers")
	otherSource := sheet.GetSprite(indexes.Grass)
	other := summary.prepareCharacterPortrait(0, otherSource, party_state.Dead)
	require.NotSame(t, previous, other, "changed source artwork must invalidate the outline")
	require.Same(t, source, summary.prepareCharacterPortrait(0, source, party_state.Good))
	assert.Empty(t, summary.portraits[0])
	assert.Equal(t, before, summaryPixels(source, source.Bounds()), "generated-outline retirement must not release shared artwork")
	assert.Same(t, source, sheet.GetSprite(party.GetCharacter(0).GetKeySpriteIndex()))
}

func TestCharacterPortraitLifecycleRetiresInactiveSlotsAndDisposesIdempotently(t *testing.T) {
	summary := NewCharacterSummaryForWidth(sprites.NewSpriteSheet(), 1280)
	t.Cleanup(summary.Dispose)
	party := summaryBaselineParty()
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Dispose)
	summary.Draw(party, screen, -1, false, -1)
	require.NotNil(t, summary.portraits[1].image)
	party.GetCharacter(1).PartyMap = party_state.PartyMapUnrecruited
	summary.Draw(party, screen, -1, false, -1)
	assert.Empty(t, summary.portraits[1])
	assert.NotNil(t, summary.portraits[2].image, "other active slots keep their outlines")
	summary.SetBounds(image.Rect(100, 100, 500, 500))
	retained := summary.portraits[2].image
	summary.Draw(party, screen, -1, false, -1)
	assert.Same(t, retained, summary.portraits[2].image, "scaling does not alter source-sized artwork")
	summary.Dispose()
	summary.Dispose()
	for index := range summary.portraits {
		assert.Empty(t, summary.portraits[index])
		assert.Nil(t, summary.characterSummaryImage[index])
	}
}

func BenchmarkCharacterPortraitOutline(b *testing.B) {
	// Only benchmark steady reuse: generating GPU images continuously inside one
	// test-game Update prevents normal frame-end retirement of driver resources.
	sheet := sprites.NewSpriteSheet()
	summary := NewCharacterSummaryForWidth(sheet, 1280)
	b.Cleanup(summary.Dispose)
	source := sheet.GetSprite(summaryBaselineParty().GetCharacter(0).GetKeySpriteIndex())
	summary.prepareCharacterPortrait(0, source, party_state.Poisoned)
	b.ReportAllocs()
	for b.Loop() {
		summary.prepareCharacterPortrait(0, source, party_state.Poisoned)
	}
}
