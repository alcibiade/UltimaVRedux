package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/visual"
)

const combatVanishDurationMs = 750

// combatVanishEffect is presentation-only FIZZ feedback for an already removed monster.
type combatVanishEffect struct {
	CombatantID game_state.CombatantID
	Index       int
	Position    references.Position
	SpriteIndex indexes.SpriteIndex
	StartedMs   int64
	Elapsed     float64
	Dissolve    DissolveEffectState
	timing      combatPresentationTiming
}

func (g *GameScene) showCombatVanishEffect(event game_state.CombatEvent) {
	vanished := event.Vanished
	if g.spriteSheet == nil || g.gameState == nil || g.gameState.GameReferences == nil ||
		g.gameState.GameReferences.EnemyReferences == nil || !vanished.MonsterType.IsValid() {
		return
	}
	enemyReferences := *g.gameState.GameReferences.EnemyReferences
	if vanished.MonsterType.Index() >= len(enemyReferences) {
		return
	}
	enemyReference := enemyReferences[vanished.MonsterType.Index()]
	if enemyReference.KeyFrameTile == nil {
		return
	}
	spriteIndex := enemyReference.KeyFrameTile.Index
	sprite := g.spriteSheet.GetSprite(spriteIndex)
	if sprite == nil {
		return
	}
	effect := combatVanishEffect{
		StartedMs:   g.gameState.ElapsedMs,
		timing:      g.captureCombatTiming(),
		CombatantID: vanished.CombatantID,
		Index:       vanished.Index,
		Position:    vanished.Position,
		SpriteIndex: spriteIndex,
	}
	effect.Dissolve.Initialize(
		sprite.Bounds().Dx(), sprite.Bounds().Dy(), visual.DissolveModeHide,
		deterministic_random.New(combatVanishSeed(vanished)),
	)
	g.combatVanishEffects = append(g.combatVanishEffects, effect)
}

func combatVanishSeed(vanished game_state.CombatVanishedMonster) uint64 {
	positionX := uint64(uint16(vanished.Position.X)) //nolint:gosec // Coordinates retain their signed bit pattern.
	positionY := uint64(uint16(vanished.Position.Y)) //nolint:gosec // Coordinates retain their signed bit pattern.
	value := uint64(vanished.CombatantID) ^
		uint64(uint16(vanished.MonsterType))<<16 ^ //nolint:gosec // Valid monster types fit in 16 bits.
		positionX<<32 ^ positionY<<48

	return presentationMix64(value)
}

func (g *GameScene) updateCombatVanishEffects(_ float64) {
	if g.gameState == nil || !g.gameState.IsInCombat() {
		g.clearCombatVanishEffects()

		return
	}
	writeIndex := 0
	for index := range g.combatVanishEffects {
		effect := &g.combatVanishEffects[index]
		effect.Elapsed = float64(max(g.gameState.ElapsedMs-effect.StartedMs, 0)) / 1000
		progress := min(effect.Elapsed/effect.timing.seconds(combatVanishDurationMs), 1)
		effect.Dissolve.UpdateProgress(progress)
		if effect.Elapsed >= effect.timing.seconds(combatVanishDurationMs) {
			effect.Dissolve.Cleanup()

			continue
		}
		g.combatVanishEffects[writeIndex] = *effect
		writeIndex++
	}
	clear(g.combatVanishEffects[writeIndex:])
	g.combatVanishEffects = g.combatVanishEffects[:writeIndex]
}

func (g *GameScene) clearCombatVanishEffects() {
	for index := range g.combatVanishEffects {
		g.combatVanishEffects[index].Dissolve.Cleanup()
	}
	g.combatVanishEffects = g.combatVanishEffects[:0]
}

func (g *GameScene) drawCombatVanishEffects(screen *ebiten.Image) {
	for index := range g.combatVanishEffects {
		effect := &g.combatVanishEffects[index]
		if !g.gameState.CombatTerrainAllowsObjectDisplay(effect.Position) {
			continue
		}
		sprite := g.spriteSheet.GetSprite(effect.SpriteIndex)
		if sprite == nil || !effect.Dissolve.IsActive() {
			continue
		}
		sprite = g.spriteSheet.GetTileEffectProcessor().ApplyDynamicSimple(sprite, sprites.BlackOutline)
		g.drawSpriteWithDissolveEffect(
			sprite,
			&effect.Dissolve,
			g.combatMonsterDrawOptions(effect.Position, effect.Index),
			screen,
		)
	}
}
