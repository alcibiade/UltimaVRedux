package main

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/visual"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestDungeonSkeletonCrumbleLastsThreeSeconds(t *testing.T) {
	assert.Equal(t, 3000, dungeonSkeletonCrumbleDurationMs)

	progress, active := dungeonSkeletonCrumbleProgress(100, 100)
	assert.True(t, active)
	assert.Zero(t, progress)

	progress, active = dungeonSkeletonCrumbleProgress(100, 100+dungeonSkeletonCrumbleDurationMs/2)
	assert.True(t, active)
	assert.InDelta(t, 0.5, progress, 0.001)

	_, active = dungeonSkeletonCrumbleProgress(100, 100+dungeonSkeletonCrumbleDurationMs)
	assert.False(t, active)
}

func TestDungeonSkeletonCrumbleOverlaysOldSpecialWallPanel(t *testing.T) {
	player := dungeonTestPlayer(references.Position{X: 4, Y: 4})
	target := references.Position{X: 4, Y: 3}
	snapshot := dungeonTestSnapshot(references.DungeonFacingNorth)
	require.True(t, snapshot.Cells.SetCell(player.Floor, player.Position, 0x4f))
	require.True(t, snapshot.Cells.SetCell(player.Floor, target, 0xb0))
	plan := buildDungeonRenderPlan(&snapshot, player, true, false, false, 600)
	effect := dungeonSkeletonCrumbleEffect{
		location: player.Location, floor: player.Floor, position: target, startedAt: 0,
	}

	appendDungeonSkeletonCrumble(&plan, &effect, 600)
	overlays := dungeonPlanOps(plan, func(operation dungeonRenderOp) bool {
		return operation.kind == dungeonRenderSkeletonCrumble
	})

	require.Len(t, overlays, 2)
	assert.Equal(t, 25, overlays[0].spriteIndex)
	assert.Same(t, &effect.dissolves[25], overlays[0].dissolve)
	assert.False(t, overlays[0].mirror)
	assert.True(t, overlays[1].mirror)
	lastOverlayIndex, firstForegroundIndex := -1, plan.count
	for index := range plan.count {
		switch plan.ops[index].kind {
		case dungeonRenderSkeletonCrumble:
			lastOverlayIndex = index
		case dungeonRenderItem, dungeonRenderMonster:
			firstForegroundIndex = min(firstForegroundIndex, index)
		}
	}
	assert.Less(t, lastOverlayIndex, firstForegroundIndex)
}

func TestDungeonSkeletonCrumbleKeepsOverlayOnSourceSideOfClipBoundary(t *testing.T) {
	target := references.Position{X: 4, Y: 3}
	plan := dungeonRenderPlan{
		location:    references.Deceit,
		floor:       0,
		clipEnabled: true,
		clipFrom:    2,
		count:       3,
	}
	plan.ops[0] = dungeonRenderOp{kind: dungeonRenderPanel}
	plan.ops[1] = dungeonRenderOp{
		kind: dungeonRenderPanel, role: dungeonRenderLeft, position: target, depth: 0,
	}
	plan.ops[2] = dungeonRenderOp{kind: dungeonRenderItem}
	effect := dungeonSkeletonCrumbleEffect{location: references.Deceit, floor: 0, position: target, startedAt: 0}

	appendDungeonSkeletonCrumble(&plan, &effect, 0)

	assert.Equal(t, 4, plan.count)
	assert.Equal(t, 3, plan.clipFrom)
	assert.Equal(t, dungeonRenderSkeletonCrumble, plan.ops[2].kind)
	assert.False(t, plan.clipsOperation(2))
	assert.True(t, plan.clipsOperation(3))
}

func TestDungeonSkeletonCrumbleRetainsFourAndCompactsExpiredEffects(t *testing.T) {
	state := new(game_state.GameState)
	state.MapState.PlayerLocation.Location = references.Deceit
	state.MapState.PlayerLocation.Floor = 0
	scene := &GameScene{gameState: state}
	for index := range dungeonSkeletonCrumbleMaxActive + 1 {
		state.ElapsedMs = int64(index) * 100
		scene.appendDungeonSkeletonCrumble(dungeonSkeletonCrumbleEffect{
			location:  references.Deceit,
			floor:     0,
			position:  references.Position{X: references.Coordinate(index)},
			startedAt: state.ElapsedMs,
		})
	}

	require.Len(t, scene.dungeonSkeletonCrumbles, dungeonSkeletonCrumbleMaxActive)
	assert.Equal(t, references.Coordinate(1), scene.dungeonSkeletonCrumbles[0].position.X)
	assert.Equal(t, references.Coordinate(4), scene.dungeonSkeletonCrumbles[3].position.X)

	state.ElapsedMs = 100 + dungeonSkeletonCrumbleDurationMs
	scene.updateDungeonSkeletonCrumble()
	require.Len(t, scene.dungeonSkeletonCrumbles, 3)
	assert.Equal(t, references.Coordinate(2), scene.dungeonSkeletonCrumbles[0].position.X)

	state.MapState.PlayerLocation.Floor = 1
	scene.updateDungeonSkeletonCrumble()
	assert.Empty(t, scene.dungeonSkeletonCrumbles)
}

func TestDungeonSkeletonCrumbleFizzLeavesReplacementWallStationary(t *testing.T) {
	wall := ebiten.NewImage(24, dungeonCanvasHeight)
	wall.Fill(color.RGBA{G: 0xff, A: 0xff})
	delta := ebiten.NewImage(24, dungeonCanvasHeight)
	delta.Set(3, 4, color.RGBA{R: 0xff, A: 0xff})
	state := DissolveEffectState{}
	state.Initialize(24, dungeonCanvasHeight, visual.DissolveModeHide, beradrng.New(1))
	canvas := ebiten.NewImage(dungeonCanvasWidth, dungeonCanvasHeight)
	clipLayer := ebiten.NewImage(dungeonCanvasWidth, dungeonCanvasHeight)
	source := syntheticDungeonSpriteSource{panel: wall, delta: delta, item: wall}
	plan := dungeonRenderPlan{
		brightness: 1,
		count:      2,
	}
	plan.ops[0] = dungeonRenderOp{kind: dungeonRenderPanel}
	plan.ops[1] = dungeonRenderOp{
		kind: dungeonRenderSkeletonCrumble, spriteIndex: 20, dissolve: &state,
	}

	drawDungeonRenderPlan(canvas, clipLayer, source, dungeonTestGraffitiOutput(), plan, 0)
	assertDungeonPixel(t, canvas, image.Pt(3, 4), color.RGBA{R: 0xff, A: 0xff})
	assertDungeonPixel(t, canvas, image.Pt(4, 4), color.RGBA{G: 0xff, A: 0xff})

	state.UpdateProgress(1)
	drawDungeonRenderPlan(canvas, clipLayer, source, dungeonTestGraffitiOutput(), plan, 0)
	assertDungeonPixel(t, canvas, image.Pt(3, 4), color.RGBA{G: 0xff, A: 0xff})
	assertDungeonPixel(t, canvas, image.Pt(3, 12), color.RGBA{G: 0xff, A: 0xff})
}
