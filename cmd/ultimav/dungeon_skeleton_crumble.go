package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/visual"
	rng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

const (
	// REDUX DEVIATION: DNG-0004: original Search redraws then dissolves the dungeon
	// viewport. Redux keeps mutation immediate but fizzles only the removed skeleton pixels.
	dungeonSkeletonCrumbleDurationMs = 3000
	dungeonSkeletonCrumbleMaxActive  = 4
)

type dungeonSkeletonCrumbleEffect struct {
	location  references.Location
	floor     references.FloorNumber
	position  references.Position
	startedAt int64
	dissolves [28]DissolveEffectState
}

func (g *GameScene) startDungeonSkeletonCrumble(position references.Position) {
	if g.gameState == nil {
		panic("skeleton crumble requires game state")
	}
	player := g.gameState.MapState.PlayerLocation
	if g.dungeonSprites == nil {
		panic("skeleton crumble requires dungeon sprites")
	}
	effect := dungeonSkeletonCrumbleEffect{
		location: player.Location, floor: player.Floor, position: position, startedAt: g.gameState.ElapsedMs,
		dissolves: [28]DissolveEffectState{},
	}
	effect.initialize(g.dungeonSprites)
	g.appendDungeonSkeletonCrumble(effect)
}

func (g *GameScene) appendDungeonSkeletonCrumble(effect dungeonSkeletonCrumbleEffect) {
	if len(g.dungeonSkeletonCrumbles) == dungeonSkeletonCrumbleMaxActive {
		g.dungeonSkeletonCrumbles[0].cleanup()
		copy(g.dungeonSkeletonCrumbles, g.dungeonSkeletonCrumbles[1:])
		g.dungeonSkeletonCrumbles = g.dungeonSkeletonCrumbles[:dungeonSkeletonCrumbleMaxActive-1]
	}
	g.dungeonSkeletonCrumbles = append(g.dungeonSkeletonCrumbles, effect)
}

func (g *GameScene) updateDungeonSkeletonCrumble() {
	if len(g.dungeonSkeletonCrumbles) == 0 || g.gameState == nil {
		return
	}
	player := g.gameState.MapState.PlayerLocation
	writeIndex := 0
	for index := range g.dungeonSkeletonCrumbles {
		effect := g.dungeonSkeletonCrumbles[index]
		if player.Location != effect.location || player.Floor != effect.floor ||
			g.gameState.ElapsedMs-effect.startedAt >= dungeonSkeletonCrumbleDurationMs {
			effect.cleanup()

			continue
		}
		effect.update(g.gameState.ElapsedMs)
		g.dungeonSkeletonCrumbles[writeIndex] = effect
		writeIndex++
	}
	clear(g.dungeonSkeletonCrumbles[writeIndex:])
	g.dungeonSkeletonCrumbles = g.dungeonSkeletonCrumbles[:writeIndex]
}

func appendDungeonSkeletonCrumble(
	plan *dungeonRenderPlan,
	effect *dungeonSkeletonCrumbleEffect,
	presentationTimeMs int64,
) {
	if !dungeonSkeletonCrumbleMatchesPlan(plan, effect) {
		return
	}
	_, active := dungeonSkeletonCrumbleProgress(effect.startedAt, presentationTimeMs)
	if !active {
		return
	}
	for index := 0; index < plan.count; index++ {
		overlay, ok := dungeonSkeletonCrumbleOverlay(plan.ops[index], effect)
		if !ok {
			continue
		}
		insertDungeonRenderOperation(plan, index+1, overlay)
		index++
	}
}

func dungeonSkeletonCrumbleMatchesPlan(plan *dungeonRenderPlan, effect *dungeonSkeletonCrumbleEffect) bool {
	return plan != nil && effect != nil && !plan.dark && plan.location == effect.location && plan.floor == effect.floor
}

func dungeonSkeletonCrumbleOverlay(
	operation dungeonRenderOp,
	effect *dungeonSkeletonCrumbleEffect,
) (dungeonRenderOp, bool) {
	if operation.kind != dungeonRenderPanel || operation.position != effect.position {
		var empty dungeonRenderOp

		return empty, false
	}
	panel, normal, ok := dungeonSkeletonCrumblePanel(operation)
	if !ok {
		var empty dungeonRenderOp

		return empty, false
	}
	if operation.spriteIndex != normal {
		panic("skeleton crumble replacement panel does not match normal wall")
	}
	operation.kind = dungeonRenderSkeletonCrumble
	operation.spriteIndex = panel
	operation.dissolve = &effect.dissolves[panel]

	return operation, true
}

func dungeonSkeletonCrumblePanel(operation dungeonRenderOp) (int, int, bool) {
	switch operation.role {
	case dungeonRenderCenter:
		if operation.depth == 0 {
			return 0, 0, false
		}

		return dungeonSpecialCenterBase + operation.depth, dungeonNormalCenterBase + operation.depth, true
	case dungeonRenderLeft, dungeonRenderRight:
		return dungeonSpecialSideBase + operation.depth, operation.depth, true
	case dungeonRenderContents:
		return 0, 0, false
	default:
		panic("invalid skeleton crumble render role")
	}
}

func insertDungeonRenderOperation(plan *dungeonRenderPlan, index int, operation dungeonRenderOp) {
	if index < 0 || index > plan.count || plan.count >= len(plan.ops) {
		panic("invalid dungeon render operation insertion")
	}
	copy(plan.ops[index+1:], plan.ops[index:plan.count])
	plan.ops[index] = operation
	plan.count++
	if plan.clipEnabled && index <= plan.clipFrom {
		plan.clipFrom++
	}
}

func dungeonSkeletonCrumbleProgress(startedAt, presentationTimeMs int64) (float64, bool) {
	if presentationTimeMs < startedAt {
		panic("skeleton crumble elapsed time moved backwards")
	}
	elapsed := presentationTimeMs - startedAt
	if elapsed >= dungeonSkeletonCrumbleDurationMs {
		return 1, false
	}

	return float64(elapsed) / dungeonSkeletonCrumbleDurationMs, true
}

func drawDungeonSkeletonCrumble(
	canvas *ebiten.Image,
	delta *ebiten.Image,
	command dungeonRenderOp,
	brightness float32,
) {
	if command.dissolve == nil || !command.dissolve.IsActive() {
		panic("skeleton crumble is missing an initialized dissolve state")
	}
	state := command.dissolve
	state.buffer.Clear()
	var copyOptions ebiten.DrawImageOptions
	state.buffer.DrawImage(delta, &copyOptions)
	var maskOptions ebiten.DrawImageOptions
	maskOptions.Blend = ebiten.BlendDestinationIn
	state.buffer.DrawImage(state.effect.GetMask(), &maskOptions)
	var options ebiten.DrawImageOptions
	scaleX := 1.0
	translateX := float64(command.x)
	if command.mirror {
		scaleX = -1
		translateX += float64(delta.Bounds().Dx())
	}
	options.GeoM.Scale(scaleX, 1)
	options.GeoM.Translate(translateX, float64(command.y))
	options.ColorScale.Scale(brightness, brightness, brightness, 1)
	canvas.DrawImage(state.buffer, &options)
}

func (effect *dungeonSkeletonCrumbleEffect) initialize(source *sprites.DungeonSprites) {
	for _, panel := range [...]int{20, 21, 22, 23, 25, 26, 27} {
		delta, ok := source.SpecialWallPanelDelta(dungeonSkeletonWallSet(effect.location), panel)
		if !ok {
			panic("skeleton crumble is missing a special-wall panel delta")
		}
		effect.dissolves[panel].Initialize(
			delta.Bounds().Dx(),
			delta.Bounds().Dy(),
			visual.DissolveModeHide,
			rng.New(dungeonSkeletonCrumbleSeed(*effect, panel)),
		)
	}
}

func (effect *dungeonSkeletonCrumbleEffect) update(presentationTimeMs int64) {
	progress, active := dungeonSkeletonCrumbleProgress(effect.startedAt, presentationTimeMs)
	if !active {
		panic("updating an expired skeleton crumble")
	}
	for _, panel := range [...]int{20, 21, 22, 23, 25, 26, 27} {
		effect.dissolves[panel].UpdateProgress(progress)
	}
}

func (effect *dungeonSkeletonCrumbleEffect) cleanup() {
	for _, panel := range [...]int{20, 21, 22, 23, 25, 26, 27} {
		effect.dissolves[panel].Cleanup()
	}
}

func dungeonSkeletonWallSet(location references.Location) int {
	wallSet, ok := dungeonWallSetForLocation(location)
	if !ok || wallSet != 2 {
		panic("skeleton crumble requires a skeleton-wall dungeon")
	}

	return wallSet
}

//nolint:gosec // Effect construction restricts these values to valid dungeon IDs, coordinates, and panel indices.
func dungeonSkeletonCrumbleSeed(effect dungeonSkeletonCrumbleEffect, panel int) uint64 {
	return presentationMix64(
		uint64(effect.location) |
			uint64(uint8(effect.floor))<<8 |
			uint64(uint16(effect.position.X))<<16 |
			uint64(uint16(effect.position.Y))<<24 |
			uint64(uint8(panel))<<32,
	)
}
