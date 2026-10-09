package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const (
	combatSceptreActorDurationMs = int64(320)
	combatFieldDispelDurationMs  = int64(420)
	combatSceptreActorInset      = 4
	combatFieldDispelInset       = 7
	combatSceptreCueOrigin       = 1
)

type combatSceptreActorEffect struct {
	Identity    game_state.CombatTurnIdentity
	Position    references.Position
	StartedAtMs int64
}

type combatFieldDispelEffect struct {
	Position    references.Position
	StartedAtMs int64
}

func (g *GameScene) showCombatSceptreActorEffect() {
	identity := g.gameState.CombatScene.GetCurrentTurnIdentity()
	if !identity.IsPartyMember {
		panic("combat Sceptre effect requires an active party member")
	}
	g.combatSceptreEffects = append(g.combatSceptreEffects, combatSceptreActorEffect{
		Identity:    identity,
		Position:    g.gameState.CombatScene.GetCombatantPosition(identity),
		StartedAtMs: g.gameState.ElapsedMs,
	})
}

func (g *GameScene) showCombatFieldDispelEffect(event game_state.CombatEvent) {
	g.combatFieldDispels = append(g.combatFieldDispels, combatFieldDispelEffect{
		Position:    event.To,
		StartedAtMs: g.gameState.ElapsedMs,
	})
}

func (g *GameScene) updateCombatSceptreEffects() {
	if !g.gameState.IsInCombat() {
		g.combatSceptreEffects = g.combatSceptreEffects[:0]
		g.combatFieldDispels = g.combatFieldDispels[:0]

		return
	}

	actorWriteIndex := 0
	for index := range g.combatSceptreEffects {
		effect := g.combatSceptreEffects[index]
		if !combatSceptreEffectActive(effect.StartedAtMs, g.gameState.ElapsedMs, combatSceptreActorDurationMs) {
			continue
		}
		g.combatSceptreEffects[actorWriteIndex] = effect
		actorWriteIndex++
	}
	g.combatSceptreEffects = g.combatSceptreEffects[:actorWriteIndex]

	fieldWriteIndex := 0
	for index := range g.combatFieldDispels {
		effect := g.combatFieldDispels[index]
		if !combatSceptreEffectActive(effect.StartedAtMs, g.gameState.ElapsedMs, combatFieldDispelDurationMs) {
			continue
		}
		g.combatFieldDispels[fieldWriteIndex] = effect
		fieldWriteIndex++
	}
	g.combatFieldDispels = g.combatFieldDispels[:fieldWriteIndex]
}

func combatSceptreEffectActive(startedAtMs, elapsedMs, durationMs int64) bool {
	return elapsedMs-startedAtMs < durationMs
}

func combatSceptreEffectProgress(startedAtMs, elapsedMs, durationMs int64) float32 {
	ageMs := max(elapsedMs-startedAtMs, 0)

	return min(float32(ageMs)/float32(durationMs), 1)
}

func combatSceptreActorCueInset(progress float32) float32 {
	return combatSceptreActorInset * float32(math.Sin(float64(progress)*math.Pi))
}

func combatFieldDispelCueInset(progress float32) float32 {
	return combatFieldDispelInset * progress
}

func (g *GameScene) drawCombatFieldDispelEffects(screen *ebiten.Image) {
	for index := range g.combatFieldDispels {
		effect := g.combatFieldDispels[index]
		if !g.gameState.CombatTerrainAllowsObjectDisplay(effect.Position) {
			continue
		}
		progress := combatSceptreEffectProgress(
			effect.StartedAtMs,
			g.gameState.ElapsedMs,
			combatFieldDispelDurationMs,
		)
		drawCombatCornerCueWithin(
			screen,
			g.combatEntityDrawOptions(effect.Position, false, game_state.NoActiveMonster, 0),
			combatSceptreCueOrigin,
			sprites.TileSize,
			combatFieldDispelCueInset(progress),
			ucolor.EGALightMagenta,
		)
	}
}

func (g *GameScene) drawCombatSceptreActorEffects(
	screen *ebiten.Image,
	identity game_state.CombatTurnIdentity,
	drawOptions *ebiten.DrawImageOptions,
) {
	for index := range g.combatSceptreEffects {
		effect := g.combatSceptreEffects[index]
		if effect.Identity != identity {
			continue
		}
		progress := combatSceptreEffectProgress(
			effect.StartedAtMs,
			g.gameState.ElapsedMs,
			combatSceptreActorDurationMs,
		)
		drawCombatCornerCueWithin(
			screen,
			drawOptions,
			combatSceptreCueOrigin,
			sprites.TileSize,
			combatSceptreActorCueInset(progress),
			ucolor.EGALightCyan,
		)
	}
}
