package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type persistentEffectPresentation struct {
	sprite   indexes.SpriteIndex
	animated bool
	tintR    float32
	tintG    float32
	tintB    float32
}

const (
	fullTint                 = float32(1)
	magicLightRedTint        = float32(0.75)
	magicLightGreenTint      = float32(0.9)
	shadowlordGreenTint      = float32(0.55)
	shadowlordBlueTint       = float32(0.8)
	intoxicationGreenTint    = float32(0.35)
	dangerTint               = float32(0.3)
	maximumPersistentEffects = 7
	maximumWornArtifacts     = 3
	maximumStatusCells       = maximumPersistentEffects + maximumWornArtifacts
	statusArtworkPixels      = 16
	statusCellPixels         = statusArtworkPixels + 2
	statusGapPixels          = 2
	layoutCenterDivisor      = 2
	statusRightInsetDivisor  = 4
	effectHashMultiplier     = 31
)

func persistentEffectPresentationFor(id game_state.PersistentEffectID) (persistentEffectPresentation, bool) {
	if presentation, ok := lightAndDurationEffectPresentation(id); ok {
		return presentation, true
	}

	return contextualEffectPresentation(id)
}

func lightAndDurationEffectPresentation(id game_state.PersistentEffectID) (persistentEffectPresentation, bool) {
	switch id {
	case game_state.PersistentEffectTorch:
		return persistentEffectPresentation{indexes.ItemTorch, true, fullTint, fullTint, fullTint}, true
	case game_state.PersistentEffectMagicLight:
		return persistentEffectPresentation{
			indexes.BlueFlame, true, magicLightRedTint, magicLightGreenTint, fullTint,
		}, true
	case game_state.PersistentEffectProtection:
		return persistentEffectPresentation{indexes.ItemShield, false, fullTint, fullTint, fullTint}, true
	case game_state.PersistentEffectNegateMagic:
		return persistentEffectPresentation{indexes.ItemAnkh, false, fullTint, fullTint, fullTint}, true
	case game_state.PersistentEffectTimeStop:
		return persistentEffectPresentation{indexes.Hourglass1, false, fullTint, fullTint, fullTint}, true
	case game_state.PersistentEffectQuickness:
		return persistentEffectPresentation{indexes.Hourglass2, false, fullTint, fullTint, fullTint}, true
	case game_state.PersistentEffectConfusion,
		game_state.PersistentEffectCrownSuppression,
		game_state.PersistentEffectIntoxication,
		game_state.PersistentEffectShadowlord,
		game_state.PersistentEffectStarvation,
		game_state.PersistentEffectHMSCape:
		return persistentEffectPresentation{0, false, 0, 0, 0}, false
	default:
		return persistentEffectPresentation{0, false, 0, 0, 0}, false
	}
}

func contextualEffectPresentation(id game_state.PersistentEffectID) (persistentEffectPresentation, bool) {
	switch id {
	case game_state.PersistentEffectConfusion:
		return persistentEffectPresentation{indexes.Whirpool1, true, fullTint, fullTint, fullTint}, true
	case game_state.PersistentEffectIntoxication:
		return persistentEffectPresentation{
			indexes.ItemPotion, false, fullTint, intoxicationGreenTint, fullTint,
		}, true
	case game_state.PersistentEffectShadowlord:
		return persistentEffectPresentation{
			indexes.ShadowLord1, true, fullTint, shadowlordGreenTint, shadowlordBlueTint,
		}, true
	case game_state.PersistentEffectStarvation:
		return persistentEffectPresentation{indexes.ItemFood, false, fullTint, dangerTint, dangerTint}, true
	case game_state.PersistentEffectHMSCape:
		return persistentEffectPresentation{indexes.ShipSailsUp, false, fullTint, fullTint, fullTint}, true
	case game_state.PersistentEffectTorch,
		game_state.PersistentEffectMagicLight,
		game_state.PersistentEffectProtection,
		game_state.PersistentEffectNegateMagic,
		game_state.PersistentEffectCrownSuppression,
		game_state.PersistentEffectTimeStop,
		game_state.PersistentEffectQuickness:
		return persistentEffectPresentation{0, false, 0, 0, 0}, false
	default:
		return persistentEffectPresentation{0, false, 0, 0, 0}, false
	}
}

type ActiveEffectsPanel struct {
	gameState       *game_state.GameState
	spriteSheet     *sprites.SpriteSheet
	bounds          image.Rectangle
	visibleEffects  []game_state.PersistentEffectStatus
	iconBounds      []image.Rectangle
	worn            []game_state.WornArtifactStatus
	wornBounds      []image.Rectangle
	separatorBounds image.Rectangle
	layout          statusBandLayout
}

func NewActiveEffectsPanel(
	state *game_state.GameState,
	spriteSheet *sprites.SpriteSheet,
	bounds image.Rectangle,
) *ActiveEffectsPanel {
	panel := &ActiveEffectsPanel{
		gameState:      state,
		spriteSheet:    spriteSheet,
		bounds:         bounds,
		visibleEffects: nil,
		iconBounds:     nil,
		worn:           nil,
		wornBounds:     nil,
		layout:         calculateStatusBandLayout(bounds),
	}
	panel.SetBounds(bounds)

	return panel
}

func (p *ActiveEffectsPanel) SetBounds(bounds image.Rectangle) {
	p.bounds = bounds
	p.layout = calculateStatusBandLayout(bounds)
}

func (p *ActiveEffectsPanel) Draw(screen *ebiten.Image) {
	p.refreshLayout()
	p.drawStatusBand(screen)
	p.drawEffects(screen)
}

func (p *ActiveEffectsPanel) drawEffects(screen *ebiten.Image) {
	for index, effect := range p.visibleEffects {
		presentation, ok := persistentEffectPresentationFor(effect.ID)
		if !ok {
			panic("persistent status has no artwork mapping")
		}
		bounds := p.iconBounds[index]
		if bounds.Empty() {
			continue
		}
		inset := statusCellInset(bounds)

		var sprite *ebiten.Image
		if presentation.animated {
			sprite = p.spriteSheet.GetAnimatedSprite(
				presentation.sprite,
				persistentEffectAnimationHash(effect.ID),
				p.gameState.ElapsedMs,
			)
		} else {
			sprite = p.spriteSheet.GetSprite(presentation.sprite)
		}
		if sprite == nil {
			continue
		}
		drawStatusArtwork(screen, unwrappedStatusSprite(sprite), bounds.Inset(inset), presentation)
	}
}

func statusCellInset(bounds image.Rectangle) int {
	return max(1, bounds.Dx()/statusCellPixels)
}

func persistentEffectAnimationHash(id game_state.PersistentEffectID) int32 {
	var result int32
	for _, character := range string(id) {
		result = result*effectHashMultiplier + character
	}

	return result
}

func (p *ActiveEffectsPanel) refreshLayout() {
	p.visibleEffects = p.visibleEffects[:0]
	for _, effect := range p.gameState.GetPersistentEffectStatuses() {
		if effect.ID != game_state.PersistentEffectCrownSuppression {
			p.visibleEffects = append(p.visibleEffects, effect)
		}
	}
	p.iconBounds = statusIconRow(p.layout.content, len(p.visibleEffects), maximumPersistentEffects, true)

	p.worn = p.worn[:0]
	for _, artifact := range p.gameState.GetWornArtifactStatuses() {
		if artifact.Worn {
			p.worn = append(p.worn, artifact)
		}
	}
	p.wornBounds = statusIconRow(p.layout.content, len(p.worn), maximumWornArtifacts, false)
	p.separatorBounds = statusSeparatorBounds(p.wornBounds, p.iconBounds)
}

func statusIconRow(bounds image.Rectangle, count, capacity int, alignRight bool) []image.Rectangle {
	if count <= 0 {
		return nil
	}
	if count > capacity {
		panic("status row exceeds its declared category capacity")
	}
	result := make([]image.Rectangle, count)
	if bounds.Empty() {
		return result
	}
	// Reserve breathing room on the effect side before sizing either group, so
	// full rows retain identical cell sizes without moving the worn-artifact edge.
	bounds.Max.X -= min(bounds.Dx(), max(1, bounds.Dy()/statusRightInsetDivisor))
	preferredSize := (bounds.Dy() + layoutCenterDivisor - 1) / layoutCenterDivisor
	gap := max(0, statusGapPixels*preferredSize/statusCellPixels)
	availableSize := max(0, (bounds.Dx()-(maximumStatusCells-1)*gap)/maximumStatusCells)
	iconSize := min(preferredSize, availableSize)
	gap = max(0, statusGapPixels*iconSize/statusCellPixels)
	y := bounds.Min.Y + (bounds.Dy()-iconSize)/layoutCenterDivisor
	x := bounds.Min.X
	if alignRight {
		x = bounds.Max.X - count*iconSize - (count-1)*gap
	}
	for i := range count {
		left := x + i*(iconSize+gap)
		result[i] = image.Rect(left, y, left+iconSize, y+iconSize)
	}

	return result
}

func statusSeparatorBounds(wornBounds, effectBounds []image.Rectangle) image.Rectangle {
	if len(wornBounds) == 0 || len(effectBounds) == 0 {
		return image.Rect(0, 0, 0, 0)
	}
	left := wornBounds[len(wornBounds)-1].Max.X
	right := effectBounds[0].Min.X
	stroke := statusCellInset(wornBounds[0])
	width := stroke * layoutCenterDivisor
	if right-left < width+stroke*layoutCenterDivisor {
		return image.Rect(0, 0, 0, 0)
	}
	height := min(wornBounds[0].Dy(), effectBounds[0].Dy())
	x := left + (right-left-width)/layoutCenterDivisor
	y := wornBounds[0].Min.Y + (wornBounds[0].Dy()-height)/layoutCenterDivisor

	return image.Rect(x, y, x+width, y+height)
}
