package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/visual"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const (
	// REDUX DEVIATION: COMBAT-0012: original special-move arrival uses a dissolve and pulse.
	// Staged pacing and bright leading pixels make that arrival readable at scale.
	combatGateArrivalMs        = 125
	combatGateMaterializeMs    = 600
	combatGateSettleMs         = 125
	combatGateDurationMs       = combatGateArrivalMs + combatGateMaterializeMs + combatGateSettleMs
	combatPossessionDurationMs = 600
	combatCorpserDurationMs    = 420
	combatBlinkDurationMs      = combatTeleportDepartureMs + combatTeleportHiddenMs + combatTeleportArrivalMs
)

type combatGatePhase uint8

const (
	combatGateArrival combatGatePhase = iota
	combatGateMaterialize
	combatGateSettle
	combatGateComplete
)

// Results have already committed. These snapshots never resolve an action or
// use gameplay RNG. Encounter + identity guards prevent recycled-slot effects.
type combatSignatureEffect struct {
	Encounter           *game_state.CombatState
	Kind                game_state.CombatEventKind
	Condition           game_state.CombatConditionResult
	Actor               game_state.CombatTurnIdentity
	CombatantID         game_state.CombatantID
	PossessionEpisodeID uint64
	From, To            references.Position
	Sprite              indexes.SpriteIndex
	StartedMs           int64
	DurationMs          int64
	OriginDisclosed     bool
	Frame               *ebiten.Image
	BodyFrame           *ebiten.Image
	HealthFrame         *ebiten.Image
	Dissolve            DissolveEffectState
	Highlight           *ebiten.Image
	GatePhase           combatGatePhase
	PhaseStartedMs      int64
	PhaseDurationsMs    [3]int64
}

func (g *GameScene) showCombatSignatureEffect(event game_state.CombatEvent) {
	timing := g.captureCombatTiming()
	var sound game_state.SoundEffect
	effect := combatSignatureEffect{Encounter: g.gameState.CombatScene, Kind: event.Kind,
		Condition: event.Condition, From: event.From, To: event.To, StartedMs: g.gameState.ElapsedMs}
	mode := visual.DissolveModeReveal
	switch event.Kind { //nolint:exhaustive // Only these published successes have signature feedback.
	case game_state.CombatEventGating:
		effect.Actor = game_state.CombatTurnIdentity{EntityIndex: event.Created.Index}
		effect.CombatantID = event.Created.CombatantID
		effect.From, effect.To = event.Created.Position, event.Created.Position
		effect.PhaseDurationsMs = [3]int64{timing.durationMs(combatGateArrivalMs), timing.durationMs(combatGateMaterializeMs), timing.durationMs(combatGateSettleMs)}
		effect.DurationMs = effect.PhaseDurationsMs[0] + effect.PhaseDurationsMs[1] + effect.PhaseDurationsMs[2]
		effect.PhaseStartedMs = effect.StartedMs
	case game_state.CombatEventPossession:
		effect.PossessionEpisodeID = event.PossessionEpisodeID
		sound = game_state.SoundMonsterPossession
		effect.Actor = combatTurnIdentityForFeedback(event.Target)
		effect.DurationMs = timing.durationMs(combatPossessionDurationMs)
		effect.OriginDisclosed = g.captureCombatReportIdentity(event.Attacker).Disclosed
	case game_state.CombatEventTeleport:
		sound = game_state.SoundMonsterTeleport
		effect.Actor = combatTurnIdentityForFeedback(event.Attacker)
		effect.PhaseDurationsMs = [3]int64{timing.durationMs(combatTeleportDepartureMs), timing.durationMs(combatTeleportHiddenMs), timing.durationMs(combatTeleportArrivalMs)}
		effect.DurationMs = effect.PhaseDurationsMs[0] + effect.PhaseDurationsMs[1] + effect.PhaseDurationsMs[2]
	case game_state.CombatEventCondition:
		switch event.Condition { //nolint:exhaustive // Routine captivity/sleep turns have no onset cue.
		case game_state.CombatConditionDraggedUnder:
			sound, mode = game_state.SoundCorpserCapture, visual.DissolveModeHide
		case game_state.CombatConditionRegurgitated:
			sound = game_state.SoundCorpserRelease
		default:
			return
		}
		effect.Actor = combatTurnIdentityForFeedback(event.Target)
		effect.From, effect.To = event.Target.Position, event.Target.Position
		effect.DurationMs = timing.durationMs(combatCorpserDurationMs)
	default:
		return
	}
	if effect.Kind != game_state.CombatEventGating {
		g.gameState.SystemCallbacks.Audio.PlaySoundEffect(sound)
	}
	if effect.Encounter == nil || g.spriteSheet == nil || !g.prepareCombatSignatureSprite(&effect) {
		return
	}
	if effect.Kind != game_state.CombatEventPossession {
		sprite := g.spriteSheet.GetSprite(effect.Sprite)
		if sprite == nil {
			return
		}
		effect.Frame = ebiten.NewImage(sprite.Bounds().Dx(), sprite.Bounds().Dy())
		effect.BodyFrame = ebiten.NewImage(sprite.Bounds().Dx(), sprite.Bounds().Dy())
		effect.HealthFrame = ebiten.NewImage(sprite.Bounds().Dx(), sprite.Bounds().Dy())
		if effect.Kind == game_state.CombatEventGating {
			effect.Highlight = ebiten.NewImage(sprite.Bounds().Dx(), sprite.Bounds().Dy())
		}
		if effect.Kind == game_state.CombatEventGating || effect.Kind == game_state.CombatEventCondition {
			//nolint:gosec // Preserve signed sprite/coordinate bits only in a cosmetic seed.
			seed := presentationMix64(uint64(effect.CombatantID) ^ uint64(effect.Sprite)<<16 ^
				uint64(uint16(effect.To.X))<<32 ^ uint64(uint16(effect.To.Y))<<48)
			effect.Dissolve.Initialize(sprite.Bounds().Dx(), sprite.Bounds().Dy(), mode, deterministic_random.New(seed))
		}
	}
	g.combatSignatureEffects = append(g.combatSignatureEffects, &effect)
}

func (g *GameScene) prepareCombatSignatureSprite(effect *combatSignatureEffect) bool {
	if effect.Actor.IsPartyMember {
		if !effect.Encounter.IsCombatPCActive(effect.Actor) {
			return false
		}
		character := g.gameState.PartyState.GetCharacter(effect.Actor.EntityIndex)
		effect.Sprite = combatPartySpriteIndex(character, effect.Encounter.IsCombatPCInvisible(effect.Actor),
			effect.Encounter.IsCombatPCRat(effect.Actor))
		if effect.Encounter.IsCombatPCSleeping(effect.Actor) {
			effect.Sprite = indexes.DeadBody
		}

		return true
	}
	monster := effect.Encounter.MonsterManager.GetMonster(effect.Actor.EntityIndex)
	if monster == nil || monster.IsInvisible() || monster.EnemyRef() == nil || monster.EnemyRef().KeyFrameTile == nil ||
		(effect.CombatantID != 0 && effect.CombatantID != monster.CombatantID()) {
		return false
	}
	effect.CombatantID = monster.CombatantID()
	effect.Sprite = monster.EnemyRef().KeyFrameTile.Index

	return true
}

func (g *GameScene) combatSignatureEffectCurrent(effect *combatSignatureEffect) bool {
	if g.gameState.CombatScene != effect.Encounter || !g.gameState.IsInCombat() {
		return false
	}
	if effect.Kind == game_state.CombatEventGating {
		if effect.GatePhase == combatGateComplete {
			return false
		}
	} else if g.gameState.ElapsedMs-effect.StartedMs >= effect.DurationMs {
		return false
	}
	if effect.Actor.IsPartyMember {
		if !effect.Encounter.IsCombatPCActive(effect.Actor) || g.gameState.PartyState.GetCharacter(effect.Actor.EntityIndex).IsDead() {
			return false
		}
		if effect.Encounter.GetCombatantPosition(effect.Actor) != effect.To {
			return false
		}
		if effect.Kind == game_state.CombatEventPossession {
			episode, _, _ := effect.Encounter.CombatPossessionProgress(effect.Actor)

			return effect.Encounter.IsCombatPCPossessed(effect.Actor) && episode == effect.PossessionEpisodeID
		}
		if effect.Kind == game_state.CombatEventCondition {
			underground := effect.Encounter.IsCombatPCUnderground(effect.Actor)

			return underground == (effect.Condition == game_state.CombatConditionDraggedUnder)
		}

		return true
	}
	monster := effect.Encounter.MonsterManager.GetMonster(effect.Actor.EntityIndex)

	return monster != nil && monster.CombatantID() == effect.CombatantID && monster.IsActive() &&
		!monster.IsDead() && !monster.IsInvisible() && monster.Position() == effect.To
}

func (g *GameScene) updateCombatSignatureEffects() {
	kept := g.combatSignatureEffects[:0]
	for _, effect := range g.combatSignatureEffects {
		if effect.Kind == game_state.CombatEventGating && g.combatSignatureEffectCurrent(effect) {
			g.updateCombatGatePhase(effect)
		}
		if !g.combatSignatureEffectCurrent(effect) {
			effect.cleanup()

			continue
		}
		if effect.Dissolve.IsActive() {
			progress := effect.progress(g.gameState.ElapsedMs)
			if effect.Kind == game_state.CombatEventGating {
				progress = effect.gateRevealProgress(g.gameState.ElapsedMs)
			}
			effect.Dissolve.UpdateProgress(progress)
		}
		kept = append(kept, effect)
	}
	clear(g.combatSignatureEffects[len(kept):])
	g.combatSignatureEffects = kept
}

// Advance at most one phase per update, anchored to actual phase admission.
// A hitch cannot spend the next phase before its first presentation. The live
// predicate deliberately does not expire gates by elapsed total: workflows poll
// before visual effects update and must not cut off a just-admitted phase.
func (g *GameScene) updateCombatGatePhase(effect *combatSignatureEffect) {
	var duration int64
	switch effect.GatePhase {
	case combatGateArrival:
		duration = effect.phaseDuration(0, combatGateArrivalMs)
	case combatGateMaterialize:
		duration = effect.phaseDuration(1, combatGateMaterializeMs)
	case combatGateSettle:
		duration = effect.phaseDuration(2, combatGateSettleMs)
	case combatGateComplete:
		return
	}
	if g.gameState.ElapsedMs-effect.PhaseStartedMs < duration {
		return
	}
	effect.GatePhase++
	effect.PhaseStartedMs = g.gameState.ElapsedMs
	if effect.GatePhase == combatGateMaterialize && g.gameState.CombatTerrainAllowsObjectDisplay(effect.To) {
		g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundMonsterGate)
	}
}

func (effect *combatSignatureEffect) gateRevealProgress(nowMs int64) float64 {
	switch effect.GatePhase {
	case combatGateArrival:
		return 0
	case combatGateMaterialize:
		return min(float64(max(nowMs-effect.PhaseStartedMs, 0))/float64(effect.phaseDuration(1, combatGateMaterializeMs)), 1)
	case combatGateSettle, combatGateComplete:
		return 1
	default:
		panic("unknown gate phase")
	}
}

func (effect *combatSignatureEffect) progress(nowMs int64) float64 {
	return min(float64(max(nowMs-effect.StartedMs, 0))/float64(effect.DurationMs), 1)
}

func (effect *combatSignatureEffect) cleanup() {
	effect.Dissolve.Cleanup()
	if effect.Frame != nil {
		effect.Frame.Deallocate()
		effect.Frame = nil
	}
	if effect.BodyFrame != nil {
		effect.BodyFrame.Deallocate()
		effect.BodyFrame = nil
	}
	if effect.HealthFrame != nil {
		effect.HealthFrame.Deallocate()
		effect.HealthFrame = nil
	}
	if effect.Highlight != nil {
		effect.Highlight.Deallocate()
		effect.Highlight = nil
	}
}

func (g *GameScene) clearCombatSignatureEffects() {
	for _, effect := range g.combatSignatureEffects {
		effect.cleanup()
	}
	clear(g.combatSignatureEffects)
	g.combatSignatureEffects = g.combatSignatureEffects[:0]
}

func (g *GameScene) hasCombatSignatureEffects() bool {
	for _, effect := range g.combatSignatureEffects {
		if g.combatSignatureEffectCurrent(effect) {
			return true
		}
	}

	return false
}

func (g *GameScene) combatSignatureHidesActor(identity game_state.CombatTurnIdentity) bool {
	for _, effect := range g.combatSignatureEffects {
		if effect.Actor == identity && effect.Kind != game_state.CombatEventPossession && g.combatSignatureEffectCurrent(effect) {
			return true
		}
	}

	return false
}

func (effect *combatSignatureEffect) phaseDuration(index int, authored int64) int64 {
	if effect.PhaseDurationsMs[index] > 0 {
		return effect.PhaseDurationsMs[index]
	}

	return authored // Zero-valued presentation fixtures retain the 1x baseline.
}

func (effect *combatSignatureEffect) blink(milliseconds int64) (combatTeleportPhase, float32) {
	departure := effect.phaseDuration(0, combatTeleportDepartureMs)
	hidden := effect.phaseDuration(1, combatTeleportHiddenMs)
	arrival := effect.phaseDuration(2, combatTeleportArrivalMs)
	if milliseconds < departure {
		return combatTeleportDeparture, float32(milliseconds) / float32(departure)
	}
	milliseconds -= departure
	if milliseconds < hidden {
		return combatTeleportHidden, 0
	}

	return combatTeleportArrival, min(float32(milliseconds-hidden)/float32(arrival), 1)
}

func (g *GameScene) drawCombatSignatureEffects(screen *ebiten.Image) {
	for _, effect := range g.combatSignatureEffects {
		if !g.combatSignatureEffectCurrent(effect) {
			continue
		}
		progress := float32(effect.progress(g.gameState.ElapsedMs))
		if effect.Kind == game_state.CombatEventPossession {
			g.drawCombatPossessionOnset(screen, effect, progress)

			continue
		}
		position, opacity := effect.To, float32(1)
		inset, clr := (1-progress)*4, ucolor.CombatMaterialization
		if effect.Kind == game_state.CombatEventTeleport {
			phase, phaseProgress := effect.blink(max(g.gameState.ElapsedMs-effect.StartedMs, 0))
			if phase == combatTeleportHidden {
				continue
			}
			if phase == combatTeleportDeparture {
				position, opacity, inset = effect.From, 1-phaseProgress, phaseProgress*5
			} else {
				opacity, inset, clr = phaseProgress, (1-phaseProgress)*5, ucolor.CombatTeleportArrival
			}
		}
		if !g.gameState.CombatTerrainAllowsObjectDisplay(position) {
			continue
		}
		options := g.combatEntityDrawOptions(position, effect.Actor.IsPartyMember, effect.Actor.EntityIndex, effect.Actor.Slot)
		if effect.Kind == game_state.CombatEventGating {
			if effect.GatePhase == combatGateArrival {
				drawCombatSeal(screen, options, float32(min(float64(max(g.gameState.ElapsedMs-effect.PhaseStartedMs, 0))/float64(effect.phaseDuration(0, combatGateArrivalMs)), 1)), true)

				continue
			}
			inset = float32(1-effect.gateRevealProgress(g.gameState.ElapsedMs)) * 4
		}
		if effect.Kind == game_state.CombatEventCondition {
			clr = ucolor.CombatUnderground
			if effect.Condition == game_state.CombatConditionDraggedUnder {
				inset = progress * 5
			}
		}
		g.prepareCombatSignatureLayers(effect)
		options.ColorScale.ScaleAlpha(opacity)
		screen.DrawImage(effect.BodyFrame, options)
		if effect.Kind != game_state.CombatEventGating || effect.GatePhase != combatGateSettle {
			drawCombatCornerCue(screen, options, inset, clr)
		}
	}
}

func (g *GameScene) drawCombatPossessionOnset(screen *ebiten.Image, effect *combatSignatureEffect, progress float32) {
	// A stepped blue mote links the published origin to the exact victim. Hidden
	// terrain suppresses either endpoint rather than disclosing an unseen actor.
	if !g.gameState.CombatTerrainAllowsObjectDisplay(effect.To) {
		return
	}
	target := g.combatEntityDrawOptions(effect.To, true, effect.Actor.EntityIndex, effect.Actor.Slot)
	drawCombatCornerCue(screen, target, (1-progress)*5, ucolor.CombatPossessed)
	if !effect.OriginDisclosed || !g.gameState.CombatTerrainAllowsObjectDisplay(effect.From) {
		return
	}
	origin := g.combatEntityDrawOptions(effect.From, false, game_state.NoActiveMonster, 0)
	drawCombatCornerCue(screen, origin, progress*4, ucolor.CombatPossessed)
	fromX, fromY := origin.GeoM.Apply(sprites.TileSize/2, sprites.TileSize/2)
	toX, toY := target.GeoM.Apply(sprites.TileSize/2, sprites.TileSize/2)
	step := float64(int(min(progress*2, 1)*8)) / 8
	x, y := float32(fromX+(toX-fromX)*step), float32(fromY+(toY-fromY)*step)
	size := combatActivePCSourcePixelScale(target) * 2
	vector.FillRect(screen, x-size, y-size, size*2, size*2, ucolor.EGABlack, false)
	vector.FillRect(screen, x-size/2, y-size/2, size, size, ucolor.CombatPossessed, false)
}

// The blue cap sits inside the upper sprite edge, separate from yellow/green
// selection, red targeting, and both this actor's and the northern neighbor's HP.
// It survives both selected/unselected turns.
func drawCombatPossessedMarker(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	left, top := options.GeoM.Apply(10, 2)
	right, bottom := options.GeoM.Apply(14, 4)
	scale := combatActivePCSourcePixelScale(options)
	vector.FillRect(screen, float32(left)-scale, float32(top)-scale,
		float32(right-left)+2*scale, float32(bottom-top)+2*scale, ucolor.EGABlack, false)
	vector.FillRect(screen, float32(left), float32(top), float32(right-left), float32(bottom-top), ucolor.CombatPossessed, false)
}
