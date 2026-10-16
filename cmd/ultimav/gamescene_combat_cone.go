package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const (
	combatConeCastPropagationDurationMs uint64 = 450
)

const combatConeCastHoldDurationMs int64 = 150

const (
	combatConeArrowPulsePeriodMs int64   = 450
	combatConeArrowPulsePhases           = 2
	combatConeArrowShadowStroke  float32 = 2
	combatConeArrowFaceStroke    float32 = 1
)

// combatDirectionalFootprintVisual configures reusable presentation for directional combat effects.
type combatDirectionalFootprintVisual struct {
	Tiles               [2]indexes.SpriteIndex
	TileCount           int
	Color               ucolor.EGAColor
	PreviewNearOpacity  float32
	PreviewFarOpacity   float32
	CastNearOpacity     float32
	CastFarOpacity      float32
	PropagationDuration uint64
	HoldDuration        int64
}

type combatConeCastEffect struct {
	spell     references.Spell
	origin    references.Position
	direction references.Direction
	positions []references.Position
	progress  float32
}

func combatDirectionalFootprintVisualForSpell(spell references.Spell) combatDirectionalFootprintVisual {
	//nolint:exhaustive // Only implemented cone spells require presentation configuration.
	switch spell {
	case references.InZu:
		return combatDirectionalFootprintVisual{
			Tiles:              [2]indexes.SpriteIndex{indexes.MagicField},
			TileCount:          1,
			Color:              ucolor.EGALightMagenta,
			PreviewNearOpacity: 0.25,
			PreviewFarOpacity:  0.12,
			// MagicField has a sparse silhouette, so a wide cast-only range is
			// needed for the spray to visibly thin as it expands.
			CastNearOpacity:     0.95,
			CastFarOpacity:      0.18,
			PropagationDuration: combatConeCastPropagationDurationMs,
			HoldDuration:        combatConeCastHoldDurationMs,
		}
	case references.InNoxHur:
		return combatDirectionalFootprintVisual{
			Tiles:               [2]indexes.SpriteIndex{indexes.PoisonField},
			TileCount:           1,
			Color:               ucolor.EGALightGreen,
			PreviewNearOpacity:  0.28,
			PreviewFarOpacity:   0.14,
			CastNearOpacity:     0.95,
			CastFarOpacity:      0.24,
			PropagationDuration: combatConeCastPropagationDurationMs,
			HoldDuration:        combatConeCastHoldDurationMs,
		}
	case references.InFlamHur:
		return combatDirectionalFootprintVisual{
			Tiles:               [2]indexes.SpriteIndex{indexes.FireField},
			TileCount:           1,
			Color:               ucolor.EGALightRed,
			PreviewNearOpacity:  0.30,
			PreviewFarOpacity:   0.15,
			CastNearOpacity:     0.95,
			CastFarOpacity:      0.25,
			PropagationDuration: combatConeCastPropagationDurationMs,
			HoldDuration:        combatConeCastHoldDurationMs,
		}
	case references.InVasGravCorp:
		return combatDirectionalFootprintVisual{
			Tiles:               [2]indexes.SpriteIndex{indexes.MagicField},
			TileCount:           1,
			Color:               ucolor.EGALightBlue,
			PreviewNearOpacity:  0.30,
			PreviewFarOpacity:   0.15,
			CastNearOpacity:     0.95,
			CastFarOpacity:      0.25,
			PropagationDuration: combatConeCastPropagationDurationMs,
			HoldDuration:        combatConeCastHoldDurationMs,
		}
	default:
		panic("combat cone visual configuration missing")
	}
}

func (g *GameScene) drawCombatConeOverlay(screen *ebiten.Image) {
	var spell references.Spell
	var origin references.Position
	var direction references.Direction
	var positions []references.Position
	progress := float32(1)
	casting := false
	if g.combatConeCast != nil {
		spell = g.combatConeCast.spell
		origin = g.combatConeCast.origin
		direction = g.combatConeCast.direction
		positions = g.combatConeCast.positions
		progress = g.combatConeCast.progress
		casting = true
	} else if targeting := g.gameState.CombatScene.ConeTargeting; targeting != nil &&
		targeting.Direction != references.NoneDirection {
		spell = targeting.Request.Spell
		origin = targeting.Origin
		direction = targeting.Direction
		positions = g.gameState.CombatConePreview()
	}
	if len(positions) == 0 {
		return
	}
	visual := combatDirectionalFootprintVisualForSpell(spell)
	maxDistance := combatDirectionalFootprintMaxDistance(positions, origin, direction)
	for _, position := range positions {
		distance := combatDirectionalFootprintDistance(origin, position, direction)
		if casting && float32(distance) > progress*float32(maxDistance) {
			continue
		}
		opacity := combatDirectionalFootprintOpacity(visual, distance, maxDistance, casting)
		drawOptions := g.combatSquareMarkerDrawOptions(position)
		drawOptions.ColorScale.ScaleWithColor(visual.Color.ToRGBA())
		drawOptions.ColorScale.ScaleAlpha(opacity)
		screen.DrawImage(g.spriteSheet.GetSprite(combatDirectionalFootprintTile(visual, position)), drawOptions)
	}
}

func combatDirectionalFootprintDistance(
	origin, position references.Position,
	direction references.Direction,
) int {
	//nolint:exhaustive // Only cardinal directions are admitted by combat cone targeting.
	switch direction {
	case references.Up:
		return int(origin.Y - position.Y)
	case references.Down:
		return int(position.Y - origin.Y)
	case references.Left:
		return int(origin.X - position.X)
	case references.Right:
		return int(position.X - origin.X)
	default:
		panic("directional footprint requires a cardinal direction")
	}
}

func combatDirectionalFootprintMaxDistance(
	positions []references.Position,
	origin references.Position,
	direction references.Direction,
) int {
	maxDistance := 1
	for _, position := range positions {
		maxDistance = max(maxDistance, combatDirectionalFootprintDistance(origin, position, direction))
	}

	return maxDistance
}

func combatDirectionalFootprintOpacity(
	visual combatDirectionalFootprintVisual,
	distance, maxDistance int,
	casting bool,
) float32 {
	near, far := visual.PreviewNearOpacity, visual.PreviewFarOpacity
	if casting {
		near, far = visual.CastNearOpacity, visual.CastFarOpacity
	}
	progress := float32(distance) / float32(max(maxDistance, 1))

	return near + (far-near)*progress
}

func combatDirectionalFootprintTile(
	visual combatDirectionalFootprintVisual,
	position references.Position,
) indexes.SpriteIndex {
	if visual.TileCount < 1 || visual.TileCount > len(visual.Tiles) {
		panic("invalid directional footprint tile configuration")
	}
	index := (int(position.X)*31 + int(position.Y)) % visual.TileCount

	return visual.Tiles[index]
}

func (g *GameScene) drawCombatDirectionIndicators(screen *ebiten.Image) {
	combat := g.gameState.CombatScene
	if combat == nil {
		return
	}
	var origin references.Position
	var casterIdx int
	var selectedDirection references.Direction
	switch {
	case combat.ConeTargeting != nil:
		origin = combat.ConeTargeting.Origin
		casterIdx = combat.ConeTargeting.Request.CasterIdx
		selectedDirection = combat.ConeTargeting.Direction
	case combat.AdjacentSpellTargeting != nil:
		if game_state.IsCombatDoorSpell(combat.AdjacentSpellTargeting.Request.Spell) {
			return
		}
		origin = combat.AdjacentSpellTargeting.Origin
		casterIdx = combat.AdjacentSpellTargeting.Request.CasterIdx
		selectedDirection = combat.AdjacentSpellTargeting.Direction
	default:
		return
	}
	if !g.gameState.CombatTerrainAllowsObjectDisplay(origin) {
		return
	}
	drawOptions := g.combatPartyDrawOptions(origin, casterIdx)
	phase := combatConeArrowPulsePhase(g.getElapsedMs())
	for _, direction := range []references.Direction{
		references.Up, references.Down, references.Left, references.Right,
	} {
		geometry := combatConeArrowSourceGeometry(direction, phase)
		drawCombatConeDirectionArrow(
			screen,
			drawOptions,
			geometry,
			combatConeArrowColor(direction, selectedDirection, phase),
		)
	}
}

func (g *GameScene) drawCombatAdjacentSpellTargetPreview(screen *ebiten.Image) {
	if combat := g.gameState.CombatScene; combat != nil && combat.AdjacentSpellTargeting != nil &&
		game_state.IsCombatDoorSpell(combat.AdjacentSpellTargeting.Request.Spell) {
		return
	}
	target, valid, active := g.gameState.CombatAdjacentSpellPreview()
	if !active || g.gameState.CombatScene.AdjacentSpellTargeting.Direction == references.NoneDirection {
		return
	}
	color := ucolor.EGALightRed
	if !g.gameState.CombatTerrainAllowsObjectDisplay(target) {
		color = ucolor.EGALightGray
	} else if valid {
		color = ucolor.EGALightCyan
	}
	drawCombatSelectionFrame(
		screen,
		g.combatSquareMarkerDrawOptions(target),
		g.getElapsedMs(),
		color,
	)
}

// combatDoorSpellCursor supplies a square to the final aim layer. It shares the
// full-size white attack reticle; eligibility never changes its geometry/color.
func (g *GameScene) combatDoorSpellCursor() (references.Position, bool) {
	combat := g.gameState.CombatScene
	if combat == nil || combat.AdjacentSpellTargeting == nil ||
		!game_state.IsCombatDoorSpell(combat.AdjacentSpellTargeting.Request.Spell) {
		return references.Position{}, false
	}
	target, _, active := g.gameState.CombatAdjacentSpellPreview()

	return target, active
}

type combatConeArrowGeometry struct {
	baseOneX, baseOneY float64
	tipX, tipY         float64
	baseTwoX, baseTwoY float64
}

func combatConeArrowSourceGeometry(direction references.Direction, phase int) combatConeArrowGeometry {
	offset := float64(phase % combatConeArrowPulsePhases)
	//nolint:exhaustive // Direction indicators are always cardinal.
	switch direction {
	case references.Up:
		return combatConeArrowGeometry{5, -1 - offset, 9, -4 - offset, 13, -1 - offset}
	case references.Down:
		return combatConeArrowGeometry{5, 19 + offset, 9, 22 + offset, 13, 19 + offset}
	case references.Left:
		return combatConeArrowGeometry{-1 - offset, 5, -4 - offset, 9, -1 - offset, 13}
	case references.Right:
		return combatConeArrowGeometry{19 + offset, 5, 22 + offset, 9, 19 + offset, 13}
	default:
		panic("combat cone indicator requires a cardinal direction")
	}
}

func combatConeArrowPulsePhase(elapsedMs int64) int {
	return int(elapsedMs/combatConeArrowPulsePeriodMs) % combatConeArrowPulsePhases
}

func combatConeArrowColor(
	direction, selectedDirection references.Direction,
	phase int,
) ucolor.EGAColor {
	if direction == selectedDirection {
		if phase == 0 {
			return ucolor.EGAMagenta
		}

		return ucolor.EGALightMagenta
	}
	if phase == 0 {
		return ucolor.EGABlue
	}

	return ucolor.EGALightBlue
}

func drawCombatConeDirectionArrow(
	screen *ebiten.Image,
	drawOptions *ebiten.DrawImageOptions,
	geometry combatConeArrowGeometry,
	faceColor ucolor.EGAColor,
) {
	baseOneX, baseOneY := drawOptions.GeoM.Apply(geometry.baseOneX, geometry.baseOneY)
	tipX, tipY := drawOptions.GeoM.Apply(geometry.tipX, geometry.tipY)
	baseTwoX, baseTwoY := drawOptions.GeoM.Apply(geometry.baseTwoX, geometry.baseTwoY)
	scale := combatActivePCSourcePixelScale(drawOptions)
	drawSegment := func(fromX, fromY, toX, toY float64, width float32, clr ucolor.EGAColor) {
		vector.StrokeLine(
			screen,
			float32(fromX), float32(fromY),
			float32(toX), float32(toY),
			width*scale,
			clr,
			false,
		)
	}
	drawSegment(baseOneX, baseOneY, tipX, tipY, combatConeArrowShadowStroke, ucolor.EGABlack)
	drawSegment(baseTwoX, baseTwoY, tipX, tipY, combatConeArrowShadowStroke, ucolor.EGABlack)
	drawSegment(baseOneX, baseOneY, tipX, tipY, combatConeArrowFaceStroke, faceColor)
	drawSegment(baseTwoX, baseTwoY, tipX, tipY, combatConeArrowFaceStroke, faceColor)
}

func (g *GameScene) startCombatConeSpellEffect(result game_state.CombatConeTargetResult) {
	visual := combatDirectionalFootprintVisualForSpell(result.Request.Spell)
	g.gameState.EventQueueManager.Workflow("combat_cone_spell_effect").
		Priority(event_queue.PriorityHigh).
		Sequential().BlockInput().
		First(func() {
			g.combatConeCast = &combatConeCastEffect{
				spell: result.Request.Spell, origin: result.Origin, direction: result.Request.Direction,
				positions: result.Positions, progress: 0,
			}
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundWhiteNoiseDamage)
		}).Animate(visual.PropagationDuration, func(progress float32) {
		g.combatConeCast.progress = progress
	}).Wait(visual.HoldDuration).Then(func() {
		g.presentCombatAreaSpellSummary(g.gameState.ResolveCommittedCombatCone(result))
	}).Finally(func() {
		g.combatConeCast = nil
		g.gameState.FinishCombatAction()
	}).Start()
}
