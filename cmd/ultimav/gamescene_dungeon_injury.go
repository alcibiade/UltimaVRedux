package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const dungeonInjuryHoldMs int64 = 40

// The feedback lane serializes sources; this single scene-owned pulse never
// borrows screenFlashIntensity or reveals dungeon geometry.
type dungeonInjuryPulse struct {
	kind      game_state.DungeonInjuryKind
	startedAt int64
	owner     game_state.DungeonInjuryOwner
}

func (g *GameScene) showDungeonInjury(kind game_state.DungeonInjuryKind) {
	if kind == game_state.DungeonInjuryNone {
		g.clearDungeonInjury()

		return
	}
	if g.disposed || g.gameState == nil || g.gameState.IsInCombat() ||
		g.gameState.MapState.PlayerLocation.Location.GetMapType() != references.DungeonMapType {
		return
	}
	g.dungeonInjury = dungeonInjuryPulse{
		kind: kind, startedAt: g.gameState.ElapsedMs, owner: g.gameState.CaptureDungeonInjuryOwner(),
	}
}

func (g *GameScene) clearDungeonInjury() {
	g.dungeonInjury = dungeonInjuryPulse{}
}

func (g *GameScene) dungeonInjuryVisible() bool {
	if g.dungeonInjury.kind == game_state.DungeonInjuryNone || g.disposed ||
		g.gameState == nil || !g.gameState.MatchesDungeonInjuryOwner(g.dungeonInjury.owner) {
		return false
	}
	elapsed := g.gameState.ElapsedMs - g.dungeonInjury.startedAt

	return elapsed >= 0 && elapsed < game_state.DungeonInjuryDurationMs
}

func (g *GameScene) updateDungeonInjury() {
	if !g.dungeonInjuryVisible() {
		g.clearDungeonInjury()
	}
}

func dungeonInjuryTint(kind game_state.DungeonInjuryKind, elapsed int64) color.NRGBA {
	if elapsed < 0 || elapsed >= game_state.DungeonInjuryDurationMs {
		return color.NRGBA{}
	}
	var tint ucolor.EGAColor
	switch kind {
	case game_state.DungeonInjuryNone:
		return color.NRGBA{}
	case game_state.DungeonInjuryFire:
		tint = ucolor.EGALightRed
	case game_state.DungeonInjuryElectrical:
		tint = ucolor.EGALightBlue
	case game_state.DungeonInjuryPoison:
		tint = ucolor.EGALightGreen
	default:
		panic("invalid dungeon injury kind")
	}
	intensity := 1.0
	if elapsed > dungeonInjuryHoldMs {
		intensity = float64(game_state.DungeonInjuryDurationMs-elapsed) /
			float64(game_state.DungeonInjuryDurationMs-dungeonInjuryHoldMs)
	}
	// NRGBA supplies correctly premultiplied channels through color.Color; a
	// partially transparent bright red must not become an additive opaque red.
	return color.NRGBA{R: tint.R, G: tint.G, B: tint.B, A: uint8(ucolor.AlphaLight * intensity)}
}

func (g *GameScene) drawDungeonInjury(screen *ebiten.Image) {
	if !g.dungeonInjuryVisible() {
		return
	}
	drawDungeonInjuryTint(screen, g.mapViewportRectangle(),
		dungeonInjuryTint(g.dungeonInjury.kind, g.gameState.ElapsedMs-g.dungeonInjury.startedAt))
}

func drawDungeonInjuryTint(screen *ebiten.Image, viewport image.Rectangle, tint color.NRGBA) {
	if tint.A == 0 {
		return
	}
	vector.FillRect(screen, float32(viewport.Min.X), float32(viewport.Min.Y),
		float32(viewport.Dx()), float32(viewport.Dy()), tint, false)
}
