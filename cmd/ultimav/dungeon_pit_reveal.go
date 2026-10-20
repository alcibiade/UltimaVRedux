package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/visual"
	rng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

const (
	dungeonPitRevealDurationMs = 3000
	dungeonPitRevealMaxActive  = 4
	dungeonPitItemBase         = 8
)

type dungeonPitRevealEffect struct {
	location  references.Location
	floor     references.FloorNumber
	position  references.Position
	startedAt int64
	dissolves [4]DissolveEffectState
}

func (g *GameScene) startDungeonPitReveal(position references.Position) {
	if g.gameState == nil || g.dungeonSprites == nil {
		panic("pit reveal requires game state and dungeon sprites")
	}
	player := g.gameState.MapState.PlayerLocation
	effect := dungeonPitRevealEffect{
		location: player.Location, floor: player.Floor, position: position, startedAt: g.gameState.ElapsedMs,
		dissolves: [4]DissolveEffectState{},
	}
	effect.initialize(g.dungeonSprites)
	g.appendDungeonPitReveal(effect)
}

func (effect *dungeonPitRevealEffect) initialize(source dungeonSpriteSource) {
	for depth := range effect.dissolves {
		item, ok := source.Item(dungeonPitItemBase + depth)
		if !ok {
			panic("pit reveal is missing a dungeon pit sprite")
		}
		effect.dissolves[depth].Initialize(item.Bounds().Dx(), item.Bounds().Dy(),
			visual.DissolveModeReveal, rng.New(dungeonPitRevealSeed(*effect, depth)))
	}
}

func (g *GameScene) appendDungeonPitReveal(effect dungeonPitRevealEffect) {
	if len(g.dungeonPitReveals) == dungeonPitRevealMaxActive {
		g.dungeonPitReveals[0].cleanup()
		copy(g.dungeonPitReveals, g.dungeonPitReveals[1:])
		g.dungeonPitReveals = g.dungeonPitReveals[:dungeonPitRevealMaxActive-1]
	}
	g.dungeonPitReveals = append(g.dungeonPitReveals, effect)
}

func (g *GameScene) updateDungeonPitReveals() {
	if len(g.dungeonPitReveals) == 0 {
		return
	}
	player := g.gameState.MapState.PlayerLocation
	writeIndex := 0
	for index := range g.dungeonPitReveals {
		effect := g.dungeonPitReveals[index]
		progress, active := dungeonPitRevealProgress(effect.startedAt, g.gameState.ElapsedMs)
		if !active || player.Location != effect.location || player.Floor != effect.floor {
			effect.cleanup()

			continue
		}
		for depth := range effect.dissolves {
			effect.dissolves[depth].UpdateProgress(progress)
		}
		g.dungeonPitReveals[writeIndex] = effect
		writeIndex++
	}
	clear(g.dungeonPitReveals[writeIndex:])
	g.dungeonPitReveals = g.dungeonPitReveals[:writeIndex]
}

func dungeonPitRevealProgress(startedAt, presentationTimeMs int64) (float64, bool) {
	if presentationTimeMs < startedAt {
		panic("pit reveal elapsed time moved backwards")
	}
	elapsed := presentationTimeMs - startedAt
	if elapsed >= dungeonPitRevealDurationMs {
		return 1, false
	}

	return float64(elapsed) / dungeonPitRevealDurationMs, true
}

func applyDungeonPitReveal(plan *dungeonRenderPlan, effect *dungeonPitRevealEffect, presentationTimeMs int64) {
	if plan.dark || plan.location != effect.location || plan.floor != effect.floor {
		return
	}
	if _, active := dungeonPitRevealProgress(effect.startedAt, presentationTimeMs); !active {
		return
	}
	// Mask the existing floor halves in place so doorway clipping and occlusion stay intact.
	for index := range plan.count {
		operation := &plan.ops[index]
		if operation.kind == dungeonRenderItem && !operation.top && operation.position == effect.position &&
			operation.spriteIndex == dungeonPitItemBase+operation.depth {
			operation.dissolve = &effect.dissolves[operation.depth]
		}
	}
}

func drawDungeonPitReveal(canvas, item *ebiten.Image, command dungeonRenderOp, brightness float32) {
	state := command.dissolve
	if state == nil || !state.IsActive() {
		panic("pit reveal is missing an initialized dissolve state")
	}
	state.buffer.Clear()
	var copyOptions ebiten.DrawImageOptions
	state.buffer.DrawImage(item, &copyOptions)
	var maskOptions ebiten.DrawImageOptions
	maskOptions.Blend = ebiten.BlendDestinationIn
	state.buffer.DrawImage(state.effect.GetMask(), &maskOptions)
	drawDungeonSprite(canvas, state.buffer, command, brightness)
}

func (effect *dungeonPitRevealEffect) cleanup() {
	for depth := range effect.dissolves {
		effect.dissolves[depth].Cleanup()
	}
}

//nolint:gosec // Effect construction restricts these values to dungeon IDs, coordinates, and depths.
func dungeonPitRevealSeed(effect dungeonPitRevealEffect, depth int) uint64 {
	return presentationMix64(
		uint64(effect.location) | uint64(uint8(effect.floor))<<8 |
			uint64(uint16(effect.position.X))<<16 | uint64(uint16(effect.position.Y))<<24 | uint64(depth)<<32,
	)
}
