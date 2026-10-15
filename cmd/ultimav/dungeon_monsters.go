//nolint:mnd // Animation instructions and normalized coordinates are verified original presentation facts.
package main

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	dungeonMonsterFrameMs        = 100
	dungeonMonsterRandomRange    = 101
	dungeonMonsterVisibleSamples = 50
	dungeonMonsterSampleStride   = 37
)

//nolint:gochecknoglobals // ref:DNG-0009: immutable first-person creature presentation table.
var dungeonMonsterSpecialFrames = [...]byte{1, 3, 2, 3}

func appendDungeonWanderer(
	plan *dungeonRenderPlan,
	wanderer *game_state.DungeonWanderer,
	position references.Position,
	depth int,
	presentationTimeMs int64,
) {
	if !plan.trueLight || wanderer == nil || depth < 1 || depth > 3 ||
		wanderer.Floor != plan.floor || wanderer.Position != position {
		return
	}
	definition, monster, ok := references.DungeonWanderingMonsterDefinitionFor(wanderer.Monster)
	if !ok {
		panic(fmt.Sprintf("dungeon wanderer monster %d has no original MON archive", wanderer.Monster))
	}
	leftFrame, rightFrame := dungeonMonsterHalfFrames(
		*plan,
		*wanderer,
		monster,
		definition.Animation,
		presentationTimeMs,
	)
	appendDungeonMonsterHalf(plan, *wanderer, monster, leftFrame, depth, false)
	appendDungeonMonsterHalf(plan, *wanderer, monster, rightFrame, depth, true)
}

func appendDungeonMonsterHalf(
	plan *dungeonRenderPlan,
	wanderer game_state.DungeonWanderer,
	monster, frame, depth int,
	right bool,
) {
	xPosition, yPosition := dungeonMonsterPosition(depth, wanderer.OnCeiling, right)
	halfFrame := frame*3 + depth - 1
	plan.append(dungeonRenderOp{
		kind:        dungeonRenderMonster,
		role:        dungeonRenderContents,
		position:    wanderer.Position,
		depth:       depth,
		spriteIndex: monster,
		x:           xPosition,
		y:           yPosition,
		mirror:      right,
		top:         wanderer.OnCeiling,
		fieldColor:  0,
		phase:       byte(halfFrame), //nolint:gosec // Frame 0..1 and depth 1..3 bound the source index to 0..5.
		dissolve:    nil,
		fieldTimeMs: 0,
	})
}

func dungeonMonsterPosition(depth int, onCeiling, right bool) (int, int) {
	if depth < 1 || depth > 3 {
		panic("invalid dungeon monster depth")
	}
	leftX := [...]int{0, 56, 64, 72}
	groundY := [...]int{0, 72, 82, 84}
	ceilingY := [...]int{0, 26, 56, 71}
	xPosition := leftX[depth]
	if right {
		xPosition = 80
	}
	yPosition := groundY[depth]
	if onCeiling {
		yPosition = ceilingY[depth]
	}

	return xPosition, yPosition
}

func dungeonMonsterHalfFrames(
	plan dungeonRenderPlan,
	wanderer game_state.DungeonWanderer,
	monster int,
	instruction byte,
	presentationTimeMs int64,
) (int, int) {
	sync := instruction & 0x90
	animationType := instruction & 0x60
	var left, right int
	frame := uint64(max(presentationTimeMs, 0) / dungeonMonsterFrameMs)

	// REDUX DEVIATION: DNG-0009: original creature redraw consumes two shared RNG
	// rolls on every redraw and persists animation in object[1].anim. Redux derives
	// cosmetic half-frames from the actor clock and snapshot identity. Time Stop
	// slows that clock instead of replacing the creature's pattern with a fixed pose.
	switch animationType {
	case 0x20:
		left = int((frame + 1) & 1)
		right = left
	case 0x60:
		specialFrame := (3 - int(frame%4)) & 3
		left = int(dungeonMonsterSpecialFrames[specialFrame] >> 1)
		right = int(dungeonMonsterSpecialFrames[specialFrame] & 1)
	default:
		left = dungeonMonsterRandomHalf(plan, wanderer, monster, frame, 0)
		right = dungeonMonsterRandomHalf(plan, wanderer, monster, frame, 1)
	}
	if sync != 0 {
		right = left
		if sync == 0x90 {
			right ^= 1
		}
	}

	return left, right
}

func dungeonMonsterRandomHalf(
	plan dungeonRenderPlan,
	wanderer game_state.DungeonWanderer,
	monster int,
	frame uint64,
	half uint64,
) int {
	floor := uint64(uint8(wanderer.Floor))           //nolint:gosec // Same-floor render validation bounds this to 0..7.
	positionX := uint64(uint16(wanderer.Position.X)) //nolint:gosec // Dungeon positions are wrapped to 0..7.
	positionY := uint64(uint16(wanderer.Position.Y)) //nolint:gosec // Dungeon positions are wrapped to 0..7.
	seed := uint64(plan.location) |
		floor<<8 |
		positionX<<16 |
		positionY<<24 |
		uint64(monster)<<32 //nolint:gosec // Monster archive indexes are bounded to 0..7.
	block := frame / dungeonMonsterRandomRange
	frameInBlock := frame % dungeonMonsterRandomRange
	seed ^= block * 0x9e3779b97f4a7c15
	seed ^= half * 0xbf58476d1ce4e5b9
	offset := presentationMix64(seed) % dungeonMonsterRandomRange
	sample := (frameInBlock*dungeonMonsterSampleStride + offset) % dungeonMonsterRandomRange
	if sample < dungeonMonsterVisibleSamples {
		return 1
	}

	return 0
}
