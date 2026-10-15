package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/visual"
)

const (
	dungeonFieldDispelDurationMs = 600
	dungeonFieldDispelMaxActive  = 4
)

type dungeonFieldDispelEffect struct {
	owner     game_state.DungeonFieldDispelOwner
	location  references.Location
	change    game_state.DungeonCellChange
	startedAt int64
	dissolves [dungeonViewDepth]DissolveEffectState
}

// presentDungeonFieldDispel consumes the already-resolved successful removal.
// It owns no command token, completion closure, wait, or gameplay mutation.
func (g *GameScene) presentDungeonFieldDispel(change game_state.DungeonCellChange) {
	if g.disposed || g.gameState == nil {
		return
	}
	owner, valid := g.gameState.CaptureDungeonFieldDispelOwner(change)
	if !valid {
		return
	}
	g.pruneDungeonFieldDispels()
	effect := dungeonFieldDispelEffect{
		owner: owner, location: g.gameState.MapState.PlayerLocation.Location,
		change: change, startedAt: g.gameState.ElapsedMs,
		dissolves: [dungeonViewDepth]DissolveEffectState{},
	}
	effect.initialize()
	g.appendDungeonFieldDispel(effect)
	g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundDispel)
}

func (effect *dungeonFieldDispelEffect) initialize() {
	for depth := range effect.dissolves {
		bounds := dungeonFieldBounds(depth)
		effect.dissolves[depth].Initialize(bounds.Dx(), bounds.Dy(), visual.DissolveModeHide,
			deterministic_random.New(dungeonFieldDispelSeed(effect, depth)))
	}
}

func dungeonFieldBounds(depth int) image.Rectangle {
	minX, maxX, minY, maxY, _, _ := dungeonFieldGeometry(depth)

	return image.Rect(minX, minY, maxX+1, maxY+1)
}

//nolint:gosec,mnd // Seed packing uses validated dungeon identity and bounded 0..3 depths.
func dungeonFieldDispelSeed(effect *dungeonFieldDispelEffect, depth int) uint64 {
	identity := uint64(effect.location) | uint64(uint8(effect.change.Floor))<<8 |
		uint64(uint16(effect.change.Position.X))<<16 | uint64(uint16(effect.change.Position.Y))<<24 |
		uint64(depth)<<32

	return presentationMix64(identity ^ presentationMix64(effect.change.Revision))
}

func (g *GameScene) appendDungeonFieldDispel(effect dungeonFieldDispelEffect) {
	for index := range g.dungeonFieldDispels {
		previous := &g.dungeonFieldDispels[index]
		if previous.owner == effect.owner {
			previous.cleanup()
			copy(g.dungeonFieldDispels[index:], g.dungeonFieldDispels[index+1:])
			clear(g.dungeonFieldDispels[len(g.dungeonFieldDispels)-1:])
			g.dungeonFieldDispels = g.dungeonFieldDispels[:len(g.dungeonFieldDispels)-1]

			break
		}
	}
	if len(g.dungeonFieldDispels) == dungeonFieldDispelMaxActive {
		g.dungeonFieldDispels[0].cleanup()
		copy(g.dungeonFieldDispels, g.dungeonFieldDispels[1:])
		clear(g.dungeonFieldDispels[dungeonFieldDispelMaxActive-1:])
		g.dungeonFieldDispels = g.dungeonFieldDispels[:dungeonFieldDispelMaxActive-1]
	}
	g.dungeonFieldDispels = append(g.dungeonFieldDispels, effect)
}

func dungeonFieldDispelProgress(startedAt, now int64) (float64, bool) {
	if now < startedAt || now-startedAt >= dungeonFieldDispelDurationMs {
		return 1, false
	}

	return float64(now-startedAt) / dungeonFieldDispelDurationMs, true
}

// pruneDungeonFieldDispels also runs synchronously from updateDisplay. It must
// never admit effects, request audio, or recursively refresh gameplay display.
func (g *GameScene) pruneDungeonFieldDispels() {
	write := 0
	for index := range g.dungeonFieldDispels {
		effect := g.dungeonFieldDispels[index]
		if g.disposed || g.gameState == nil || !g.gameState.MatchesDungeonFieldDispelOwner(effect.owner) {
			effect.cleanup()

			continue
		}
		if _, active := dungeonFieldDispelProgress(effect.startedAt, g.gameState.ElapsedMs); !active {
			effect.cleanup()

			continue
		}
		g.dungeonFieldDispels[write] = effect
		write++
	}
	clear(g.dungeonFieldDispels[write:])
	g.dungeonFieldDispels = g.dungeonFieldDispels[:write]
}

func (g *GameScene) updateDungeonFieldDispels() {
	g.pruneDungeonFieldDispels()
	for index := range g.dungeonFieldDispels {
		effect := &g.dungeonFieldDispels[index]
		progress, _ := dungeonFieldDispelProgress(effect.startedAt, g.gameState.ElapsedMs)
		for depth := range effect.dissolves {
			effect.dissolves[depth].UpdateProgress(progress)
		}
	}
}

func (g *GameScene) clearDungeonFieldDispels() {
	for index := range g.dungeonFieldDispels {
		g.dungeonFieldDispels[index].cleanup()
	}
	clear(g.dungeonFieldDispels)
	g.dungeonFieldDispels = nil
}

func (effect *dungeonFieldDispelEffect) cleanup() {
	for depth := range effect.dissolves {
		effect.dissolves[depth].Cleanup()
	}
}

// appendDungeonFieldDispels is called inside the shared far-to-near traversal,
// before live cell contents. Only field pixels are retained, never the old bit08
// opening or the background. Applicable effects also pass a live owner check.
func appendDungeonFieldDispels(
	plan *dungeonRenderPlan, state *game_state.GameState, effects []dungeonFieldDispelEffect,
	position references.Position, depth int, now int64,
) {
	for index := range effects {
		effect := &effects[index]
		if state == nil || !state.MatchesDungeonFieldDispelOwner(effect.owner) ||
			plan.location != effect.location || plan.floor != effect.change.Floor || effect.change.Position != position {
			continue
		}
		if _, active := dungeonFieldDispelProgress(effect.startedAt, now); !active {
			continue
		}
		plan.append(dungeonRenderOp{
			kind: dungeonRenderFieldDispel, role: dungeonRenderContents, position: position, depth: depth,
			fieldColor: effect.change.Before.Payload() & dungeonFieldColorMask, fieldTimeMs: effect.startedAt,
			dissolve:    &effect.dissolves[depth],
			spriteIndex: 0, x: 0, y: 0, mirror: false, top: false, phase: 0,
		})
	}
}

func drawDungeonFieldDispel(canvas *ebiten.Image, command dungeonRenderOp, brightness float32) {
	state := command.dissolve
	if state == nil || !state.IsActive() {
		panic("field dispel requires an initialized dissolve")
	}
	bounds := dungeonFieldBounds(command.depth)
	state.buffer.Clear()
	drawDungeonFieldAt(state.buffer, command, command.fieldTimeMs, brightness, -bounds.Min.X, -bounds.Min.Y)
	var maskOptions ebiten.DrawImageOptions
	maskOptions.Blend = ebiten.BlendDestinationIn
	state.buffer.DrawImage(state.effect.GetMask(), &maskOptions)
	var options ebiten.DrawImageOptions
	options.GeoM.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))
	canvas.DrawImage(state.buffer, &options)
}
