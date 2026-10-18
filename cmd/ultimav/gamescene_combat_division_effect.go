package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const (
	combatDivisionTetherDurationMs      = 280
	combatDivisionDurationMs            = 720
	combatDivisionTetherDurationSeconds = float64(combatDivisionTetherDurationMs) / 1000
	combatDivisionBudSourcePixels       = 3.0
	combatDivisionTetherShadowPixels    = 2.0
	combatDivisionTetherFacePixels      = 1.0
	combatDivisionOutlineIntervalMs     = 110
)

// CombatDivisionEffect is presentation-only feedback linking a dividing monster to its child.
type CombatDivisionEffect struct {
	ParentIndex, ChildIndex       int
	ParentPosition, ChildPosition references.Position
	StartedMs                     int64
	Elapsed                       float64
	timing                        combatPresentationTiming
}

func (g *GameScene) showCombatDivisionEffect(event game_state.CombatEvent) {
	g.combatDivisionEffects = append(g.combatDivisionEffects, CombatDivisionEffect{
		StartedMs:   g.gameState.ElapsedMs,
		timing:      g.captureCombatTiming(),
		ParentIndex: event.Attacker.Index, ChildIndex: event.Target.Index,
		ParentPosition: event.From, ChildPosition: event.To,
	})
}

func (g *GameScene) updateCombatDivisionEffects(_ float64) {
	if !g.gameState.IsInCombat() {
		g.combatDivisionEffects = g.combatDivisionEffects[:0]

		return
	}
	writeIndex := 0
	for index := range g.combatDivisionEffects {
		effect := &g.combatDivisionEffects[index]
		effect.Elapsed = float64(max(g.gameState.ElapsedMs-effect.StartedMs, 0)) / 1000
		if effect.Elapsed >= effect.timing.seconds(combatDivisionDurationMs) {
			continue
		}
		g.combatDivisionEffects[writeIndex] = *effect
		writeIndex++
	}
	g.combatDivisionEffects = g.combatDivisionEffects[:writeIndex]
}

func (g *GameScene) combatDivisionChildOutlinePhase(monsterIndex int) (int, bool) {
	for index := len(g.combatDivisionEffects) - 1; index >= 0; index-- {
		effect := g.combatDivisionEffects[index]
		if effect.ChildIndex != monsterIndex || effect.Elapsed < effect.timing.seconds(combatDivisionTetherDurationMs) {
			continue
		}

		return int(effect.Elapsed*1000/float64(effect.timing.durationMs(combatDivisionOutlineIntervalMs))) % 2, true
	}

	return 0, false
}

func (g *GameScene) drawCombatDivisionEffects(screen *ebiten.Image) {
	for index := range g.combatDivisionEffects {
		effect := g.combatDivisionEffects[index]
		if effect.Elapsed >= effect.timing.seconds(combatDivisionTetherDurationMs) {
			continue
		}
		if !g.gameState.CombatTerrainAllowsObjectDisplay(effect.ParentPosition) ||
			!g.gameState.CombatTerrainAllowsObjectDisplay(effect.ChildPosition) {
			continue
		}
		g.drawCombatDivisionTether(screen, effect)
	}
}

func (g *GameScene) drawCombatDivisionTether(screen *ebiten.Image, effect CombatDivisionEffect) {
	parentOptions := g.combatMonsterDrawOptions(effect.ParentPosition, effect.ParentIndex)
	childOptions := g.combatMonsterDrawOptions(effect.ChildPosition, effect.ChildIndex)
	parentX, parentY := parentOptions.GeoM.Apply(
		combatEntityCenterSourcePixels, combatEntityCenterSourcePixels,
	)
	childX, childY := childOptions.GeoM.Apply(
		combatEntityCenterSourcePixels, combatEntityCenterSourcePixels,
	)
	progress := effect.Elapsed / effect.timing.seconds(combatDivisionTetherDurationMs)
	headX := parentX + (childX-parentX)*progress
	headY := parentY + (childY-parentY)*progress
	scale := combatActivePCSourcePixelScale(parentOptions)
	drawTether := func(width float32, clr ucolor.EGAColor) {
		vector.StrokeLine(
			screen,
			float32(parentX), float32(parentY), float32(headX), float32(headY),
			width*scale,
			clr,
			false,
		)
	}
	drawTether(combatDivisionTetherShadowPixels, ucolor.EGABlack)
	drawTether(combatDivisionTetherFacePixels, ucolor.EGALightGreen)
	budSize := float32(combatDivisionBudSourcePixels) * scale
	vector.FillRect(
		screen,
		float32(headX)-budSize/2,
		float32(headY)-budSize/2,
		budSize,
		budSize,
		ucolor.EGAGreen,
		false,
	)
}
