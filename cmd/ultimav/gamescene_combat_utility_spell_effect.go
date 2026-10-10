package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const (
	combatUtilityApproachDurationMs = 180
	combatUtilityResultDurationMs   = 220
	combatTeleportDepartureMs       = 180
	combatTeleportHiddenMs          = 80
	combatTeleportArrivalMs         = 200
	combatTeleportFlashIntensity    = 0.18
)

type combatUtilityEffectPhase int

const (
	combatUtilityApproach combatUtilityEffectPhase = iota
	combatUtilityResult
)

type combatUtilitySpellEffect struct {
	Spell    references.Spell
	Position references.Position
	Phase    combatUtilityEffectPhase
	Progress float32
}

type combatTeleportPhase int

const (
	combatTeleportDeparture combatTeleportPhase = iota
	combatTeleportHidden
	combatTeleportArrival
)

type combatTeleportSpellEffect struct {
	Caster   game_state.CombatTurnIdentity
	From     references.Position
	To       references.Position
	Phase    combatTeleportPhase
	Progress float32
}

func (g *GameScene) drawCombatUtilitySpellEffects(screen *ebiten.Image) {
	if g.combatUtilityEffect != nil {
		effect := g.combatUtilityEffect
		if g.gameState.CombatTerrainAllowsObjectDisplay(effect.Position) {
			inset := combatUtilityEffectInset(effect.Spell, effect.Phase, effect.Progress)
			drawCombatCornerCue(
				screen,
				g.combatEntityDrawOptions(effect.Position, false, int(effect.Position.X)*16+int(effect.Position.Y), 0),
				inset,
				combatUtilityEffectColor(effect.Spell, effect.Phase),
			)
		}
	}
	if g.combatTeleportEffect == nil || g.combatTeleportEffect.Phase == combatTeleportHidden {
		return
	}
	effect := g.combatTeleportEffect
	position := effect.From
	inset := effect.Progress * 5
	color := ucolor.EGALightMagenta
	if effect.Phase == combatTeleportArrival {
		position = effect.To
		inset = (1 - effect.Progress) * 5
		color = ucolor.EGALightCyan
	}
	if !g.gameState.CombatTerrainAllowsObjectDisplay(position) {
		return
	}
	drawCombatCornerCue(
		screen,
		g.combatEntityDrawOptions(position, true, effect.Caster.EntityIndex, effect.Caster.Slot),
		inset,
		color,
	)
}

func combatUtilityEffectInset(spell references.Spell, phase combatUtilityEffectPhase, progress float32) float32 {
	if phase == combatUtilityResult {
		return float32(math.Sin(float64(progress)*math.Pi)) * 2
	}
	if spell == references.AnExPor {
		return progress * 5
	}

	return (1 - progress) * 5
}

func combatUtilityEffectColor(spell references.Spell, phase combatUtilityEffectPhase) ucolor.EGAColor {
	if phase == combatUtilityResult {
		if spell == references.AnExPor {
			return ucolor.EGALightMagenta
		}

		return ucolor.EGAWhite
	}
	//nolint:exhaustive // Only utility spells create this effect.
	switch spell {
	case references.AnSanct:
		return ucolor.EGAYellow
	case references.AnExPor:
		return ucolor.EGAMagenta
	case references.InExPor:
		return ucolor.EGALightCyan
	default:
		panic("unsupported combat utility spell effect")
	}
}

func drawCombatCornerCue(
	screen *ebiten.Image,
	drawOptions *ebiten.DrawImageOptions,
	inset float32,
	color ucolor.EGAColor,
) {
	drawCombatCornerCueWithin(screen, drawOptions, 0, sprites.TileSize, inset, color)
}

func drawCombatCornerCueWithin(
	screen *ebiten.Image,
	drawOptions *ebiten.DrawImageOptions,
	origin, size, inset float32,
	color ucolor.EGAColor,
) {
	const (
		cornerLength = 4
		shadowWidth  = 2
		faceWidth    = 1
	)
	leftSource, topSource, rightSource, bottomSource := combatCornerCueSourceBounds(origin, size, inset)
	left, top := drawOptions.GeoM.Apply(leftSource, topSource)
	right, bottom := drawOptions.GeoM.Apply(rightSource, bottomSource)
	scale := combatActivePCSourcePixelScale(drawOptions)
	length := float32(right-left) * cornerLength / size
	drawSegments := func(width float32, clr ucolor.EGAColor) {
		segments := [][4]float32{
			{float32(left), float32(top), float32(left) + length, float32(top)},
			{float32(left), float32(top), float32(left), float32(top) + length},
			{float32(right), float32(top), float32(right) - length, float32(top)},
			{float32(right), float32(top), float32(right), float32(top) + length},
			{float32(left), float32(bottom), float32(left) + length, float32(bottom)},
			{float32(left), float32(bottom), float32(left), float32(bottom) - length},
			{float32(right), float32(bottom), float32(right) - length, float32(bottom)},
			{float32(right), float32(bottom), float32(right), float32(bottom) - length},
		}
		for _, segment := range segments {
			vector.StrokeLine(screen, segment[0], segment[1], segment[2], segment[3], width*scale, clr, false)
		}
	}
	drawSegments(shadowWidth, ucolor.EGABlack)
	drawSegments(faceWidth, color)
}

func combatCornerCueSourceBounds(origin, size, inset float32) (float64, float64, float64, float64) {
	minimum := float64(origin + inset)
	maximum := float64(origin + size - inset)

	return minimum, minimum, maximum, maximum
}

func (g *GameScene) startCombatUtilitySpellEffect(result game_state.CombatAdjacentSpellTargetResult) {
	var sound game_state.SoundEffect
	//nolint:exhaustive // Targeting dispatch limits this workflow to the three adjacent utility spells.
	switch result.Request.Spell {
	case references.AnSanct:
		sound = game_state.SoundUnlock
	case references.AnExPor:
		sound = game_state.SoundMagicLock
	case references.InExPor:
		sound = game_state.SoundMagicUnlock
	default:
		panic("unsupported combat utility spell")
	}
	g.gameState.EventQueueManager.Workflow("combat_utility_spell_effect").
		Priority(event_queue.PriorityHigh).
		Sequential().BlockInput().
		First(func() {
			g.combatUtilityEffect = &combatUtilitySpellEffect{
				Spell: result.Request.Spell, Position: result.Target, Phase: combatUtilityApproach,
			}
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(sound)
		}).Animate(combatUtilityApproachDurationMs, func(progress float32) {
		g.combatUtilityEffect.Progress = progress
	}).Then(func() {
		//nolint:exhaustive // Targeting dispatch limits this workflow to the three adjacent utility spells.
		switch result.Request.Spell {
		case references.AnSanct:
			g.gameState.CastCombatAnSanct(result.Request.CasterIdx, result.Target)
		case references.AnExPor:
			g.gameState.CastCombatAnExPor(result.Request.CasterIdx, result.Target)
		case references.InExPor:
			g.gameState.CastCombatInExPor(result.Request.CasterIdx, result.Target)
		default:
			panic("unsupported combat utility spell")
		}
		g.combatUtilityEffect.Phase = combatUtilityResult
		g.combatUtilityEffect.Progress = 0
	}).Animate(combatUtilityResultDurationMs, func(progress float32) {
		g.combatUtilityEffect.Progress = progress
	}).Finally(func() {
		g.combatUtilityEffect = nil
		g.gameState.FinishCombatAction()
	}).Start()
}

func (g *GameScene) thenCastCombatInPor(casterIdx int) event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		var result game_state.CombatInPorResult

		return q.Then(func() {
			result = g.gameState.PlanCombatInPor(casterIdx)
			if result.Outcome == game_state.CombatInPorResolved {
				g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMagic)
				g.combatTeleportEffect = &combatTeleportSpellEffect{
					Caster: result.Caster, From: result.From, To: result.To, Phase: combatTeleportDeparture,
				}
			}
		}).If(func() bool { return result.Outcome == game_state.CombatInPorResolved }, func(effectQ *event_queue.SequentialEventBuilder) {
			effectQ.Animate(combatTeleportDepartureMs, func(progress float32) {
				g.combatTeleportEffect.Progress = progress
			}).Then(func() {
				g.gameState.ApplyCombatInPor(result)
				g.combatTeleportEffect.Phase = combatTeleportHidden
				g.gameState.SystemCallbacks.Visual.SetScreenFlashColor(0.5, 0.0, 0.5)
				g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(combatTeleportFlashIntensity)
			}).Wait(combatTeleportHiddenMs).Then(func() {
				g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
				g.combatTeleportEffect.Phase = combatTeleportArrival
				g.combatTeleportEffect.Progress = 0
			}).Animate(combatTeleportArrivalMs, func(progress float32) {
				g.combatTeleportEffect.Progress = progress
			}).Then(func() {
				g.combatTeleportEffect = nil
			})
		}).If(func() bool { return result.Outcome == game_state.CombatInPorFailed }, func(failureQ *event_queue.SequentialEventBuilder) {
			failureQ.Then(func() {
				g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundSpellFailure)
				g.gameState.SystemCallbacks.UI.ShowActionFeedback("Failed!", true)
			})
		}).Then(g.clearCombatTeleportEffect)
	}
}

func (g *GameScene) clearCombatTeleportEffect() {
	if g.combatTeleportEffect == nil {
		return
	}
	g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
	g.combatTeleportEffect = nil
}
