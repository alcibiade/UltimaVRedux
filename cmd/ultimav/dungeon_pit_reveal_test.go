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
)

func TestDungeonPitRevealProgress(t *testing.T) {
	for _, test := range []struct {
		name     string
		elapsed  int64
		progress float64
		active   bool
	}{
		{"initial", 0, 0, true},
		{"halfway", 1500, 0.5, true},
		{"last active millisecond", 2999, 2999.0 / 3000, true},
		{"complete", 3000, 1, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			progress, active := dungeonPitRevealProgress(100, 100+test.elapsed)
			assert.InDelta(t, test.progress, progress, 1e-9)
			assert.Equal(t, test.active, active)
		})
	}
	assert.PanicsWithValue(t, "pit reveal elapsed time moved backwards", func() {
		dungeonPitRevealProgress(100, 99)
	})
}

func TestDungeonPitRevealMasksOnlyMatchingFloorHalvesWithoutChangingPlan(t *testing.T) {
	target := references.Position{X: 4, Y: 3}
	effect := dungeonPitRevealEffect{location: references.Deceit, floor: 0, position: target, startedAt: 100}
	plan := dungeonRenderPlan{
		location: references.Deceit, floor: 0, clipEnabled: true, clipFrom: 3, brightness: 1,
	}
	for depth := range 4 {
		for _, mirror := range []bool{false, true} {
			floor := dungeonRenderOp{
				kind: dungeonRenderItem, role: dungeonRenderContents, position: target,
				depth: depth, spriteIndex: 8 + depth, mirror: mirror, x: 10 + depth, y: 40,
			}
			plan.append(floor)
			ceiling := floor
			ceiling.top = true
			plan.append(ceiling)
			otherPit := floor
			otherPit.position.X++
			plan.append(otherPit)
			otherItem := floor
			otherItem.spriteIndex++
			plan.append(otherItem)
			panel := floor
			panel.kind = dungeonRenderPanel
			plan.append(panel)
		}
	}
	before := plan
	applyDungeonPitReveal(&plan, &effect, 1600)
	for index := range plan.count {
		assert.Equal(t, before.clipsOperation(index), plan.clipsOperation(index), "operation %d", index)
		if index%5 == 0 {
			assert.Same(t, &effect.dissolves[plan.ops[index].depth], plan.ops[index].dissolve)
		} else {
			assert.Nil(t, plan.ops[index].dissolve, "operation %d", index)
		}
		plan.ops[index].dissolve = nil
	}
	assert.Equal(t, before, plan, "only dissolve pointers may change")
}

func TestDungeonPitRevealSkipsInapplicablePlans(t *testing.T) {
	for _, test := range []struct {
		name     string
		location references.Location
		floor    references.FloorNumber
		dark     bool
		elapsed  int64
	}{
		{"dark", references.Deceit, 0, true, 1500},
		{"other location", references.Despise, 0, false, 1500},
		{"other floor", references.Deceit, 1, false, 1500},
		{"expired", references.Deceit, 0, false, 3000},
	} {
		t.Run(test.name, func(t *testing.T) {
			effect := dungeonPitRevealEffect{location: references.Deceit, floor: 0, startedAt: 100}
			plan := dungeonRenderPlan{
				location: test.location, floor: test.floor, dark: test.dark, count: 1,
			}
			plan.ops[0] = dungeonRenderOp{kind: dungeonRenderItem, spriteIndex: 8}
			before := plan
			applyDungeonPitReveal(&plan, &effect, 100+test.elapsed)
			assert.Equal(t, before, plan)
		})
	}
}

func TestDungeonPitRevealLifecycle(t *testing.T) {
	item := ebiten.NewImage(16, 16)
	t.Cleanup(item.Deallocate)
	state := new(game_state.GameState)
	state.MapState.PlayerLocation = dungeonTestPlayer(references.Position{X: 4, Y: 4})
	scene := &GameScene{gameState: state}
	for index := range dungeonPitRevealMaxActive + 1 {
		state.ElapsedMs = 100 + int64(index)*100
		effect := dungeonPitRevealEffect{
			location: references.Deceit, floor: 0, startedAt: state.ElapsedMs,
			position: references.Position{X: references.Coordinate(index), Y: 3},
		}
		effect.initialize(syntheticDungeonSpriteSource{item: item})
		scene.appendDungeonPitReveal(effect)
	}
	require.Len(t, scene.dungeonPitReveals, 4)
	for index, effect := range scene.dungeonPitReveals {
		assert.Equal(t, references.Coordinate(index+1), effect.position.X, "oldest effect is evicted")
		assert.Equal(t, references.Deceit, effect.location)
		assert.Equal(t, references.FloorNumber(0), effect.floor)
		assert.Equal(t, int64(200+index*100), effect.startedAt)
		for _, dissolve := range effect.dissolves {
			assert.True(t, dissolve.IsActive())
			require.NotNil(t, dissolve.buffer)
			assert.Equal(t, item.Bounds(), dissolve.buffer.Bounds())
		}
	}
	state.ElapsedMs = 1700
	scene.updateDungeonPitReveals()
	for _, dissolve := range scene.dungeonPitReveals[0].dissolves {
		assert.InDelta(t, 0.5, dissolve.GetLevel(), 1e-9)
	}
	state.ElapsedMs = 3199
	scene.updateDungeonPitReveals()
	require.Len(t, scene.dungeonPitReveals, 4)
	backing := scene.dungeonPitReveals
	state.ElapsedMs = 3200
	scene.updateDungeonPitReveals()
	require.Len(t, scene.dungeonPitReveals, 3)
	assert.Equal(t, references.Coordinate(2), scene.dungeonPitReveals[0].position.X)
	assert.Equal(t, dungeonPitRevealEffect{}, backing[3], "compaction must release trailing references")
	state.MapState.PlayerLocation.Floor = 1
	scene.updateDungeonPitReveals()
	assert.Empty(t, scene.dungeonPitReveals)
	for _, effect := range backing {
		assert.Equal(t, dungeonPitRevealEffect{}, effect)
	}
	effect := dungeonPitRevealEffect{location: references.Deceit, floor: 1, startedAt: state.ElapsedMs}
	effect.initialize(syntheticDungeonSpriteSource{item: item})
	scene.appendDungeonPitReveal(effect)
	backing = scene.dungeonPitReveals
	state.MapState.PlayerLocation.Location = references.Despise
	scene.updateDungeonPitReveals()
	assert.Empty(t, scene.dungeonPitReveals)
	assert.Equal(t, dungeonPitRevealEffect{}, backing[0])
	assert.NotPanics(t, scene.updateDungeonPitReveals)
}

func TestDungeonPitRevealCleanupReleasesDissolves(t *testing.T) {
	item := ebiten.NewImage(8, 8)
	t.Cleanup(item.Deallocate)
	effect := dungeonPitRevealEffect{}
	effect.initialize(syntheticDungeonSpriteSource{item: item})
	for depth := range effect.dissolves {
		effect.dissolves[depth].UpdateProgress(0.5)
	}
	effect.cleanup()
	for _, dissolve := range effect.dissolves {
		assert.False(t, dissolve.IsActive())
		assert.Nil(t, dissolve.buffer)
		assert.Zero(t, dissolve.GetLevel())
	}
	assert.NotPanics(t, effect.cleanup)
}

func TestDungeonPitRevealPixelsPreserveBackground(t *testing.T) {
	background := color.RGBA{G: 0xff, A: 0xff}
	pitColor := color.RGBA{R: 0xff, A: 0xff}
	panel := ebiten.NewImage(dungeonCanvasWidth, dungeonCanvasHeight)
	panel.Fill(background)
	item := ebiten.NewImage(16, 16)
	item.Fill(pitColor)
	item.Set(0, 0, color.Transparent)
	canvas := ebiten.NewImage(dungeonCanvasWidth, dungeonCanvasHeight)
	clipLayer := ebiten.NewImage(dungeonCanvasWidth, dungeonCanvasHeight)
	for _, resource := range []*ebiten.Image{panel, item, canvas, clipLayer} {
		t.Cleanup(resource.Deallocate)
	}
	source := syntheticDungeonSpriteSource{panel: panel, item: item}
	state := new(game_state.GameState)
	state.MapState.PlayerLocation = dungeonTestPlayer(references.Position{})
	scene := &GameScene{gameState: state}
	effect := dungeonPitRevealEffect{location: references.Deceit}
	effect.initialize(source)
	scene.appendDungeonPitReveal(effect)
	output := dungeonTestGraffitiOutput()
	for _, elapsed := range []int64{0, 1500, 3000} {
		state.ElapsedMs = elapsed
		scene.updateDungeonPitReveals()
		plan := dungeonRenderPlan{
			location: references.Deceit, brightness: 1, count: 2,
		}
		plan.ops[0] = dungeonRenderOp{kind: dungeonRenderPanel}
		plan.ops[1] = dungeonRenderOp{kind: dungeonRenderItem, spriteIndex: 8, x: 20, y: 40}
		for index := range scene.dungeonPitReveals {
			applyDungeonPitReveal(&plan, &scene.dungeonPitReveals[index], elapsed)
		}
		drawDungeonRenderPlan(canvas, clipLayer, source, output, plan, elapsed)
		visible := 0
		for y := range 16 {
			for x := range 16 {
				point := image.Pt(20+x, 40+y)
				actual := color.RGBAModel.Convert(canvas.At(point.X, point.Y)).(color.RGBA)
				if actual == pitColor {
					visible++
				} else {
					assertDungeonPixel(t, canvas, point, background)
				}
			}
		}
		assertDungeonPixel(t, canvas, image.Pt(20, 40), background)
		assertDungeonPixel(t, canvas, image.Pt(19, 40), background)
		assertDungeonPixel(t, canvas, image.Pt(36, 56), background)
		switch elapsed {
		case 0:
			assert.Zero(t, visible)
		case 1500:
			assert.Greater(t, visible, 0)
			assert.Less(t, visible, 255)
		case 3000:
			assert.Equal(t, 255, visible)
			assert.Nil(t, plan.ops[1].dissolve)
			assert.Empty(t, scene.dungeonPitReveals)
		}
	}
}
