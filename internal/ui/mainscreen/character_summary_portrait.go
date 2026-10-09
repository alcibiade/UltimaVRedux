package mainscreen

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
)

// One owned outline per party slot. Source artwork is borrowed from SpriteSheet
// and must never be released with the generated image.
type characterPortrait struct {
	source *ebiten.Image
	rgba   [4]uint32
	image  *ebiten.Image
}

func (portrait *characterPortrait) dispose() {
	if portrait.image != nil {
		portrait.image.Deallocate()
	}
	*portrait = characterPortrait{source: nil, rgba: [4]uint32{}, image: nil}
}

func (c *CharacterSummary) prepareCharacterPortrait(
	index int, source *ebiten.Image, status party_state.CharacterStatus,
) *ebiten.Image {
	portrait := &c.portraits[index]
	clr := c.getStatusColor(status)
	if !c.shouldApplyStatusOutline(status) || clr == nil {
		portrait.dispose()

		return source
	}
	r, g, b, a := clr.RGBA()
	rgba := [4]uint32{r, g, b, a}
	if portrait.source == source && portrait.rgba == rgba && portrait.image != nil {
		return portrait.image
	}
	portrait.dispose()
	outlined := c.spriteSheet.GetTileEffectProcessor().ApplyDynamicOutlineRGBA(source, clr, statusOutlineIntensity)
	if outlined != source {
		*portrait = characterPortrait{source: source, rgba: rgba, image: outlined}
	}

	return outlined
}

// Dispose releases summary-owned images, including retained status outlines.
// SpriteSheet artwork remains borrowed. Repeated calls are harmless.
func (c *CharacterSummary) Dispose() {
	for index := range c.portraits {
		c.portraits[index].dispose()
		if c.characterSummaryImage[index] != nil {
			c.characterSummaryImage[index].Deallocate()
			c.characterSummaryImage[index] = nil
		}
	}
	c.textRows = [party_state.MaxCharactersInParty]characterTextRow{}
}
