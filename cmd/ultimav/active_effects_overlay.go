package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

type transientIcon struct {
	kind        game_state.TransientIconKind
	expiresAtMs int64
}

type transientIconPresentation struct {
	spriteIndex indexes.SpriteIndex
	colorTint   ucolor.EGAColor
	durationMs  int64
}

// ActiveEffectsOverlay retains brief map-centered event feedback. Durable effects
// are derived from GameState and rendered by ActiveEffectsPanel.
type ActiveEffectsOverlay struct {
	effects     []transientIcon
	spriteSheet *sprites.SpriteSheet
}

const (
	effectSpriteScale           = 0.75
	effectBorderWidth           = 1.0
	effectAlpha                 = float32(0.9)
	effectMarginTiles           = 1
	effectSpacingPixels         = 1.0
	transientCenterDivisor      = 2
	colorChannelMaximum         = 255
	transientIconCapacity       = 5
	transientEatingDurationMs   = int64(1000)
	transientStarvingDurationMs = int64(500)
	transientStatusDurationMs   = int64(1500)
	transientLevelUpDurationMs  = int64(3000)
)

func NewActiveEffectsOverlay(spriteSheet *sprites.SpriteSheet) *ActiveEffectsOverlay {
	return &ActiveEffectsOverlay{
		effects:     make([]transientIcon, 0, transientIconCapacity),
		spriteSheet: spriteSheet,
	}
}

func transientIconPresentationFor(kind game_state.TransientIconKind) transientIconPresentation {
	switch kind {
	case game_state.TransientIconEating:
		return transientIconPresentation{indexes.ItemFood, ucolor.EGAWhite, transientEatingDurationMs}
	case game_state.TransientIconStarving:
		return transientIconPresentation{indexes.ItemFood, ucolor.EGALightRed, transientStarvingDurationMs}
	case game_state.TransientIconHealing:
		return transientIconPresentation{indexes.SignHealer, ucolor.EGALightGreen, transientStatusDurationMs}
	case game_state.TransientIconPoisonOnset:
		return transientIconPresentation{indexes.PoisonField, ucolor.EGAGreen, transientStatusDurationMs}
	case game_state.TransientIconPoisonCure:
		return transientIconPresentation{indexes.PoisonField, ucolor.EGALightCyan, transientStatusDurationMs}
	case game_state.TransientIconLevelUp:
		return transientIconPresentation{indexes.StarPattern, ucolor.EGAYellow, transientLevelUpDurationMs}
	case game_state.TransientIconKarmaIncrease:
		return transientIconPresentation{indexes.Shrine, ucolor.EGALightGreen, transientStatusDurationMs}
	case game_state.TransientIconKarmaDecrease:
		return transientIconPresentation{indexes.Shrine, ucolor.EGALightRed, transientStatusDurationMs}
	default:
		panic("unsupported transient icon kind")
	}
}

// RegisterTransient refreshes a live kind in place or appends a distinct event.
func (o *ActiveEffectsOverlay) RegisterTransient(kind game_state.TransientIconKind, currentElapsedMs int64) {
	presentation := transientIconPresentationFor(kind)
	o.Update(currentElapsedMs)
	for index := range o.effects {
		if o.effects[index].kind == kind {
			o.effects[index].expiresAtMs = currentElapsedMs + presentation.durationMs

			return
		}
	}
	if len(o.effects) == transientIconCapacity {
		copy(o.effects, o.effects[1:])
		o.effects = o.effects[:len(o.effects)-1]
	}
	o.effects = append(o.effects, transientIcon{
		kind:        kind,
		expiresAtMs: currentElapsedMs + presentation.durationMs,
	})
}

func (o *ActiveEffectsOverlay) Update(currentElapsedMs int64) {
	writeIndex := 0
	for index := range o.effects {
		effect := o.effects[index]
		if currentElapsedMs >= effect.expiresAtMs {
			continue
		}
		o.effects[writeIndex] = effect
		writeIndex++
	}
	clear(o.effects[writeIndex:])
	o.effects = o.effects[:writeIndex]
}

func (o *ActiveEffectsOverlay) Clear() {
	clear(o.effects)
	o.effects = o.effects[:0]
}

func (o *ActiveEffectsOverlay) DrawWithScaling(
	screen *ebiten.Image,
	mapImage *ebiten.Image,
	placement sprites.PercentBasedPlacement,
	visibleTilesX, visibleTilesY int,
) {
	for effectIndex, effect := range o.effects {
		tileX := visibleTilesX / transientCenterDivisor
		tileY := visibleTilesY - effectMarginTiles - 1 - effectIndex
		if tileY < 0 {
			continue
		}

		presentation := transientIconPresentationFor(effect.kind)
		screenX, screenY := sprites.ConvertPosition16to18(tileX, tileY)
		screenY -= float64(effectIndex) * effectSpacingPixels
		sprite := o.spriteSheet.GetSprite(presentation.spriteIndex)
		if sprite == nil {
			continue
		}

		drawOptions := new(ebiten.DrawImageOptions)
		drawOptions.GeoM.Scale(effectSpriteScale, effectSpriteScale)
		drawOptions.GeoM.Translate(screenX+effectBorderWidth, screenY+effectBorderWidth)
		drawOptions.ColorScale.ScaleAlpha(effectAlpha)
		drawOptions.ColorScale.Scale(
			float32(presentation.colorTint.R)/colorChannelMaximum,
			float32(presentation.colorTint.G)/colorChannelMaximum,
			float32(presentation.colorTint.B)/colorChannelMaximum,
			1,
		)

		screenTransform := new(ebiten.DrawImageOptions)
		sprites.ApplyDrawOptionsFromPercentsForWholeScreen(screenTransform, mapImage, placement)
		drawOptions.GeoM.Concat(screenTransform.GeoM)
		screen.DrawImage(sprite, drawOptions)
	}
}
