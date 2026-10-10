package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	gametext "github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const (
	wornBadgeGlyph = 0x1D // Badge glyph in IBM charset 0; artifact status is ref:QUEST-0003.
)

func (p *ActiveEffectsPanel) drawStatusBand(screen *ebiten.Image) {
	drawStatusSeparator(screen, p.separatorBounds)
	for index, artifact := range p.worn {
		bounds := p.wornBounds[index]
		if bounds.Empty() {
			continue
		}
		artwork := wornArtifactImage(p.spriteSheet, artifact.ID)
		drawStatusArtwork(screen, artwork, bounds.Inset(statusCellInset(bounds)), persistentEffectPresentation{
			sprite: 0, animated: false, tintR: fullTint, tintG: fullTint, tintB: fullTint,
		})
	}
}

func drawStatusSeparator(screen *ebiten.Image, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}
	stroke := max(1, bounds.Dx()/layoutCenterDivisor)
	vector.FillRect(screen, float32(bounds.Min.X), float32(bounds.Min.Y),
		float32(bounds.Dx()), float32(bounds.Dy()), ucolor.EGABlue, false)
	vector.FillRect(screen, float32(bounds.Min.X), float32(bounds.Min.Y),
		float32(stroke), float32(bounds.Dy()), ucolor.EGAWhite, false)
}

func wornArtifactImage(spriteSheet *sprites.SpriteSheet, id game_state.ToggleItemType) *ebiten.Image {
	switch id {
	case game_state.ToggleItemCrown:
		return unwrappedStatusSprite(spriteSheet.GetSprite(indexes.Crown))
	case game_state.ToggleItemAmulet:
		return unwrappedStatusSprite(spriteSheet.GetSprite(indexes.Amulet))
	case game_state.ToggleItemBadge:
		return gametext.GetBitmapFonts().IBM.GetCharImage(wornBadgeGlyph)
	default:
		panic("unsupported worn artifact identity")
	}
}

func unwrappedStatusSprite(sprite *ebiten.Image) *ebiten.Image {
	if sprite == nil {
		return nil
	}
	// The cached gameplay image wraps 16x16 art in a transparent 18x18 tile.
	if sprite.Bounds().Size() != image.Pt(statusCellPixels, statusCellPixels) {
		return sprite
	}
	artwork, ok := sprite.SubImage(sprite.Bounds().Inset(1)).(*ebiten.Image)
	if !ok {
		panic("status sprite crop has unexpected image type")
	}

	return artwork
}

func drawStatusArtwork(screen, artwork *ebiten.Image, content image.Rectangle, tint persistentEffectPresentation) {
	if artwork == nil || content.Empty() {
		return
	}
	scale := min(float64(content.Dx())/float64(artwork.Bounds().Dx()),
		float64(content.Dy())/float64(artwork.Bounds().Dy()))
	options := &ebiten.DrawImageOptions{}
	options.Filter = ebiten.FilterNearest
	options.GeoM.Scale(scale, scale)
	width := float64(artwork.Bounds().Dx()) * scale
	height := float64(artwork.Bounds().Dy()) * scale
	options.GeoM.Translate(float64(content.Min.X)+(float64(content.Dx())-width)/layoutCenterDivisor,
		float64(content.Min.Y)+(float64(content.Dy())-height)/layoutCenterDivisor)
	options.ColorScale.Scale(tint.tintR, tint.tintG, tint.tintB, 1)
	screen.DrawImage(artwork, options)
}
