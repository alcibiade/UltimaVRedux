package main

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func newDungeonFieldDispelTestScene(t *testing.T) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()
	scene, callbacks := newDungeonViewTestScene(t)
	refresh := scene.gameState.SystemCallbacks.Screen.UpdateDisplay
	scene.gameState.SystemCallbacks.Screen.UpdateDisplay = func() {
		refresh()
		scene.pruneDungeonFieldDispels()
		scene.updateActorAnimationClocks()
	}
	snapshot, active := scene.gameState.DungeonStateSnapshot()
	require.True(t, active)
	player := scene.gameState.MapState.PlayerLocation
	for depth := range dungeonViewDepth {
		position := dungeonRelativePosition(player.Position, snapshot.Facing, depth, 0)
		_, err := scene.gameState.TransformDungeonCell(player.Floor, position,
			func(references.DungeonCell) references.DungeonCell { return 0 })
		require.NoError(t, err)
	}
	scene.gameState.ElapsedMs = 1377
	t.Cleanup(scene.clearDungeonFieldDispels)

	return scene, callbacks
}

func admitTestDungeonFieldDispel(t *testing.T, scene *GameScene, position references.Position, raw byte) game_state.DungeonCellChange {
	t.Helper()
	state := scene.gameState
	floor := state.MapState.PlayerLocation.Floor
	_, err := state.TransformDungeonCell(floor, position, func(references.DungeonCell) references.DungeonCell { return references.DungeonCell(raw) })
	require.NoError(t, err)
	change, err := state.TransformDungeonCell(floor, position, func(before references.DungeonCell) references.DungeonCell {
		return references.DungeonCell(before.Raw() & 0x08)
	})
	require.NoError(t, err)
	scene.presentDungeonFieldDispel(change)

	return change
}

func dungeonFieldDispelTestPlan(t *testing.T, scene *GameScene, lit, subdued bool) dungeonRenderPlan {
	t.Helper()
	snapshot, active := scene.gameState.DungeonStateSnapshot()
	require.True(t, active)
	snapshot.Wanderer = nil

	return buildDungeonRenderPlanWithPresentation(&snapshot, scene.gameState.MapState.PlayerLocation, lit, subdued,
		dungeonRenderPresentation{
			environmentTimeMs: scene.gameState.ElapsedMs, creatureTimeMs: scene.dungeonActorAnimationTime(),
			state: scene.gameState, fieldDispels: scene.dungeonFieldDispels,
		})
}

func renderFieldDispelTestPlan(t *testing.T, plan dungeonRenderPlan, now int64) *ebiten.Image {
	t.Helper()
	canvas := ebiten.NewImage(dungeonCanvasWidth, dungeonCanvasHeight)
	clip := ebiten.NewImage(dungeonCanvasWidth, dungeonCanvasHeight)
	panel := ebiten.NewImage(80, dungeonCanvasHeight)
	item := ebiten.NewImage(2, 2)
	item.Fill(color.RGBA{R: 90, G: 90, B: 90, A: 255})
	for _, resource := range []*ebiten.Image{canvas, clip, panel, item} {
		t.Cleanup(resource.Deallocate)
	}
	drawDungeonRenderPlan(canvas, clip, syntheticDungeonSpriteSource{panel: panel, item: item}, dungeonTestGraffitiOutput(), plan, now)

	return canvas
}

func TestDungeonFieldDispelFrozenPixelsMatchLiveFieldAcrossDepthsAndColours(t *testing.T) {
	for depth := range dungeonViewDepth {
		t.Run(fmt.Sprintf("depth_%d_colour_%d", depth, depth), func(t *testing.T) {
			scene, _ := newDungeonFieldDispelTestScene(t)
			state := scene.gameState
			snapshot, _ := state.DungeonStateSnapshot()
			player := state.MapState.PlayerLocation
			target := dungeonRelativePosition(player.Position, snapshot.Facing, depth, 0)
			change := admitTestDungeonFieldDispel(t, scene, target, 0x88|byte(depth))
			require.Len(t, scene.dungeonFieldDispels, 1)
			effect := &scene.dungeonFieldDispels[0]
			for projectionDepth := range effect.dissolves {
				assert.Equal(t, dungeonFieldBounds(projectionDepth).Size(), effect.dissolves[projectionDepth].buffer.Bounds().Size())
			}
			snapshot, _ = state.DungeonStateSnapshot()
			snapshot.Wanderer = nil
			require.True(t, snapshot.Cells.SetCell(player.Floor, target, change.Before))
			ordinary := buildDungeonRenderPlan(&snapshot, player, true, false, false, state.ElapsedMs)
			initialPlan := dungeonFieldDispelTestPlan(t, scene, true, false)
			initial := renderFieldDispelTestPlan(t, initialPlan, state.ElapsedMs)
			assertDungeonImagesEqual(t, renderFieldDispelTestPlan(t, ordinary, state.ElapsedMs), initial)
			openings := dungeonPlanOps(initialPlan, func(op dungeonRenderOp) bool { return op.kind == dungeonRenderItem && op.top && op.position == target })
			require.Len(t, openings, 2, "the live overhead opening is drawn once, as two halves")
			buffer, mask := effect.dissolves[depth].buffer, effect.dissolves[depth].effect.GetMask()
			state.ElapsedMs += 300
			scene.updateDungeonFieldDispels()
			middle := renderFieldDispelTestPlan(t, dungeonFieldDispelTestPlan(t, scene, true, false), state.ElapsedMs)
			assert.Same(t, buffer, effect.dissolves[depth].buffer)
			assert.Same(t, mask, effect.dissolves[depth].effect.GetMask())
			for projectionDepth := range effect.dissolves {
				assert.Equal(t, 0.5, effect.dissolves[projectionDepth].GetLevel(), "all reprojection masks share absolute progress")
			}
			// Rendering at a different ordinary field-frame boundary must not
			// reshuffle the admitted field or advance the update-owned mask.
			repeated := renderFieldDispelTestPlan(t, dungeonFieldDispelTestPlan(t, scene, true, false), state.ElapsedMs+100)
			assertDungeonImagesEqual(t, middle, repeated)
			state.ElapsedMs += 300
			// Draw's defensive expiry works even before the next updater.
			expired := renderFieldDispelTestPlan(t, dungeonFieldDispelTestPlan(t, scene, true, false), state.ElapsedMs)
			scene.updateDungeonFieldDispels()
			require.Empty(t, scene.dungeonFieldDispels)
			baseline := renderFieldDispelTestPlan(t, dungeonFieldDispelTestPlan(t, scene, true, false), state.ElapsedMs)
			assertDungeonImagesEqual(t, baseline, expired)
			visible, removed := 0, 0
			for y := range dungeonCanvasHeight {
				for x := range dungeonCanvasWidth {
					start := initial.At(x, y)
					background := baseline.At(x, y)
					current := middle.At(x, y)
					if start == background {
						assert.Equal(t, background, current, "background pixel (%d,%d)", x, y)
						continue
					}
					if current == start {
						visible++
					} else {
						removed++
						assert.Equal(t, background, current, "dissolve only removes frozen field pixels")
					}
				}
			}
			assert.Positive(t, visible)
			assert.Positive(t, removed)
		})
	}
}

func TestDungeonFieldDispelTraversalClippingOcclusionAndWrappedDepthZero(t *testing.T) {
	scene, _ := newDungeonFieldDispelTestScene(t)
	state := scene.gameState
	player := state.MapState.PlayerLocation
	snapshot, _ := state.DungeonStateSnapshot()
	near := dungeonRelativePosition(player.Position, snapshot.Facing, 1, 0)
	far := dungeonRelativePosition(player.Position, snapshot.Facing, 2, 0)
	admitTestDungeonFieldDispel(t, scene, near, 0x82)
	admitTestDungeonFieldDispel(t, scene, far, 0x81)
	_, err := state.TransformDungeonCell(player.Floor, player.Position, func(references.DungeonCell) references.DungeonCell { return 0xe0 })
	require.NoError(t, err)
	snapshot, _ = state.DungeonStateSnapshot()
	snapshot.Wanderer = &game_state.DungeonWanderer{Monster: references.MonsterRat, Floor: player.Floor, Position: near}
	plan := buildDungeonRenderPlanWithPresentation(&snapshot, player, true, false, dungeonRenderPresentation{
		state: state, fieldDispels: scene.dungeonFieldDispels, environmentTimeMs: state.ElapsedMs,
	})
	var contents []dungeonRenderOpKind
	for index := range plan.count {
		op := plan.ops[index]
		if op.kind == dungeonRenderFieldDispel || op.kind == dungeonRenderMonster {
			require.True(t, plan.clipsOperation(index), "effect must belong to doorway contents clip, not a late overlay")
			contents = append(contents, op.kind)
		}
	}
	assert.Equal(t, []dungeonRenderOpKind{dungeonRenderFieldDispel, dungeonRenderFieldDispel, dungeonRenderMonster, dungeonRenderMonster}, contents)
	_, err = state.TransformDungeonCell(player.Floor, near, func(references.DungeonCell) references.DungeonCell { return 0xb0 })
	require.NoError(t, err)
	blocked := dungeonFieldDispelTestPlan(t, scene, true, false)
	assert.Empty(t, dungeonPlanOps(blocked, func(op dungeonRenderOp) bool { return op.kind == dungeonRenderFieldDispel }))
	// Reproject the still-live far-cell remnant around the party itself.
	state.TransitionToLocation(player.Location, player.Floor, far, game_state.TransitionPositionOnly, false)
	current := dungeonFieldDispelTestPlan(t, scene, true, false)
	effects := dungeonPlanOps(current, func(op dungeonRenderOp) bool { return op.kind == dungeonRenderFieldDispel })
	require.Len(t, effects, 1)
	assert.Zero(t, effects[0].depth)
	assert.Equal(t, far, effects[0].position)
}

func TestDungeonFieldDispelLightingAndSmallStepDeterminism(t *testing.T) {
	scene, callbacks := newDungeonFieldDispelTestScene(t)
	state := scene.gameState
	player := state.MapState.PlayerLocation
	admitTestDungeonFieldDispel(t, scene, player.Position, 0x8a)
	snapshot, _ := state.DungeonStateSnapshot()
	snapshot.Wanderer = nil
	require.True(t, snapshot.Cells.SetCell(player.Floor, player.Position, 0x8a))
	for _, lighting := range []struct{ lit, subdued bool }{{true, false}, {false, false}, {false, true}} {
		ordinary := buildDungeonRenderPlan(&snapshot, player, lighting.lit, lighting.subdued, false, state.ElapsedMs)
		fizzle := dungeonFieldDispelTestPlan(t, scene, lighting.lit, lighting.subdued)
		assertDungeonImagesEqual(t, renderFieldDispelTestPlan(t, ordinary, state.ElapsedMs), renderFieldDispelTestPlan(t, fizzle, state.ElapsedMs))
	}
	duplicate := scene.dungeonFieldDispels[0]
	duplicate.dissolves = [dungeonViewDepth]DissolveEffectState{}
	duplicate.initialize()
	t.Cleanup(duplicate.cleanup)
	rng, date, effect := state.GetRNG().Calls(), state.DateTime, state.ActiveDurationEffect
	sounds := append([]game_state.SoundEffect(nil), callbacks.SoundEffectsPlayed...)
	for range 30 {
		state.ElapsedMs += 10
		scene.updateDungeonFieldDispels()
	}
	for depth := range duplicate.dissolves {
		duplicate.dissolves[depth].UpdateProgress(0.5)
		assertDungeonImagesEqual(t, duplicate.dissolves[depth].effect.GetMask(), scene.dungeonFieldDispels[0].dissolves[depth].effect.GetMask())
	}
	before := renderFieldDispelTestPlan(t, dungeonFieldDispelTestPlan(t, scene, true, false), state.ElapsedMs)
	scene.updateDungeonFieldDispels()
	after := renderFieldDispelTestPlan(t, dungeonFieldDispelTestPlan(t, scene, true, false), state.ElapsedMs)
	assertDungeonImagesEqual(t, before, after)
	assert.Equal(t, rng, state.GetRNG().Calls())
	assert.Equal(t, date, state.DateTime)
	assert.Equal(t, effect, state.ActiveDurationEffect)
	assert.Equal(t, sounds, callbacks.SoundEffectsPlayed)
}

func TestDungeonFieldDispelLifecycleSynchronousReplacementFloorAndVisit(t *testing.T) {
	scene, _ := newDungeonFieldDispelTestScene(t)
	state := scene.gameState
	player := state.MapState.PlayerLocation
	admitTestDungeonFieldDispel(t, scene, player.Position, 0x80)
	backing := scene.dungeonFieldDispels
	_, err := state.TransformDungeonCell(player.Floor, player.Position, func(references.DungeonCell) references.DungeonCell { return 0x80 })
	require.NoError(t, err)
	assert.Empty(t, scene.dungeonFieldDispels, "replacement prunes synchronously")
	assert.Equal(t, dungeonFieldDispelEffect{}, backing[0], "retired owner and GPU references released")
	// Removing the replacement within the same frame must not revive the old effect.
	result := state.DispelCurrentOrFacingDungeonField()
	require.True(t, result.Cleared)
	assert.Empty(t, scene.dungeonFieldDispels)
	scene.presentDungeonFieldDispel(result.Change)
	require.Len(t, scene.dungeonFieldDispels, 1)
	_, err = state.TransformDungeonCell(player.Floor, references.Position{}, func(references.DungeonCell) references.DungeonCell { return 0x40 })
	require.NoError(t, err)
	require.Len(t, scene.dungeonFieldDispels, 1, "unrelated changes preserve target")
	_, err = state.TransformDungeonCell(player.Floor+1, player.Position, func(references.DungeonCell) references.DungeonCell { return 0 })
	require.NoError(t, err)
	state.TransitionToLocation(player.Location, player.Floor+1, player.Position, game_state.TransitionFloorOnly, false)
	assert.Empty(t, scene.dungeonFieldDispels)
	state.TransitionToLocation(player.Location, player.Floor, player.Position, game_state.TransitionFloorOnly, false)
	assert.Empty(t, scene.dungeonFieldDispels)
	admitTestDungeonFieldDispel(t, scene, player.Position, 0x81)
	snapshot, _ := state.DungeonStateSnapshot()
	_, err = state.DebugSetupDungeonRoute(player.Location, player.Floor, player.Position, snapshot.Facing, snapshot.LastDirection)
	require.NoError(t, err)
	assert.Empty(t, scene.dungeonFieldDispels, "fresh same-coordinate authority must retire old effects synchronously")
}

func TestDungeonFieldDispelBoundedEvictionExpiryCleanupAndSaveIsolation(t *testing.T) {
	scene, callbacks := newDungeonFieldDispelTestScene(t)
	state := scene.gameState
	for index := range dungeonFieldDispelMaxActive + 1 {
		admitTestDungeonFieldDispel(t, scene, references.Position{X: references.Coordinate(index), Y: 0}, 0x83)
		state.ElapsedMs += 10
		scene.updateDungeonFieldDispels()
	}
	require.Len(t, scene.dungeonFieldDispels, 4)
	assert.EqualValues(t, 1, scene.dungeonFieldDispels[0].change.Position.X)
	last := scene.dungeonFieldDispels[3].change.Position
	admitTestDungeonFieldDispel(t, scene, last, 0x82)
	require.Len(t, scene.dungeonFieldDispels, 4)
	assert.Equal(t, last, scene.dungeonFieldDispels[3].change.Position)
	saved, err := state.CaptureSaveSnapshot()
	require.NoError(t, err, "cosmetic remnants add no save restriction")
	sounds := append([]game_state.SoundEffect(nil), callbacks.SoundEffectsPlayed...)
	backing := scene.dungeonFieldDispels
	scene.clearDungeonFieldDispels()
	scene.clearDungeonFieldDispels()
	for _, value := range backing {
		assert.Equal(t, dungeonFieldDispelEffect{}, value)
	}
	without, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	assert.Equal(t, saved, without)
	assert.Equal(t, sounds, callbacks.SoundEffectsPlayed)
	admitTestDungeonFieldDispel(t, scene, last, 0x82)
	state.ElapsedMs += dungeonFieldDispelDurationMs - 1
	scene.updateDungeonFieldDispels()
	require.Len(t, scene.dungeonFieldDispels, 1)
	state.ElapsedMs++
	scene.updateDungeonFieldDispels()
	assert.Empty(t, scene.dungeonFieldDispels)
	scene.disposed = true
	scene.presentDungeonFieldDispel(game_state.DungeonCellChange{})
	assert.Empty(t, scene.dungeonFieldDispels)
}

func TestDungeonFieldDispelProductionSceptreRendersBeforeExpiry(t *testing.T) {
	scene, callbacks := newDungeonFieldDispelTestScene(t)
	state := scene.gameState
	clearUsableItemsForDungeonView(state)
	state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
	position := state.MapState.PlayerLocation.Position
	_, err := state.TransformDungeonCell(state.MapState.PlayerLocation.Floor, position,
		func(references.DungeonCell) references.DungeonCell { return 0x8b })
	require.NoError(t, err)
	callbacks.Reset()
	scene.dungeonMapInputHandler(ebiten.KeyU)
	dialog := waitForDungeonItemDialog(t, scene)
	dialog.HandleEnter()
	for range 30 {
		state.ElapsedMs += 10
		state.EventQueueManager.Update()
		scene.updateDungeonFieldDispels()
		if state.IsAtDungeonCommandBoundary() {
			break
		}
	}
	require.True(t, state.IsAtDungeonCommandBoundary())
	require.Len(t, scene.dungeonFieldDispels, 1)
	assert.Equal(t, []game_state.SoundEffect{game_state.SoundDispel}, callbacks.SoundEffectsPlayed)
	plan := dungeonFieldDispelTestPlan(t, scene, true, false)
	operations := dungeonPlanOps(plan, func(op dungeonRenderOp) bool { return op.kind == dungeonRenderFieldDispel })
	require.Len(t, operations, 1)
	assert.Equal(t, position, operations[0].position)
	canvas := renderFieldDispelTestPlan(t, plan, state.ElapsedMs)
	assert.Positive(t, countDungeonFieldPixels(canvas, dungeonFieldColor(3)))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
	beforeSoundCount := len(callbacks.SoundEffectsPlayed)
	beforeStart := scene.dungeonFieldDispels[0].startedAt
	result := state.UseDungeonSceptre()
	assert.False(t, result.Cleared)
	assert.Len(t, callbacks.SoundEffectsPlayed, beforeSoundCount)
	assert.Equal(t, beforeStart, scene.dungeonFieldDispels[0].startedAt)
}

func countDungeonFieldPixels(canvas *ebiten.Image, field color.RGBA) int {
	count := 0
	for y := range canvas.Bounds().Dy() {
		for x := range canvas.Bounds().Dx() {
			if color.RGBAModel.Convert(canvas.At(x, y)) == field {
				count++
			}
		}
	}

	return count
}

func TestDungeonFieldDispelReprojectionUsesAllInitializedDepthMasks(t *testing.T) {
	scene, _ := newDungeonFieldDispelTestScene(t)
	player := scene.gameState.MapState.PlayerLocation
	snapshot, _ := scene.gameState.DungeonStateSnapshot()
	target := dungeonRelativePosition(player.Position, snapshot.Facing, 3, 0)
	admitTestDungeonFieldDispel(t, scene, target, 0x82)
	scene.gameState.ElapsedMs += 450
	scene.updateDungeonFieldDispels()
	for depth := range dungeonViewDepth {
		position := dungeonRelativePosition(target, snapshot.Facing, -depth, 0)
		scene.gameState.TransitionToLocation(player.Location, player.Floor, position, game_state.TransitionPositionOnly, false)
		plan := dungeonFieldDispelTestPlan(t, scene, true, false)
		operations := dungeonPlanOps(plan, func(op dungeonRenderOp) bool { return op.kind == dungeonRenderFieldDispel })
		require.Len(t, operations, 1)
		assert.Equal(t, depth, operations[0].depth)
		assert.Equal(t, 0.75, operations[0].dissolve.GetLevel())
		assert.Equal(t, image.Pt(dungeonFieldBounds(depth).Dx(), dungeonFieldBounds(depth).Dy()), operations[0].dissolve.buffer.Bounds().Size())
	}
}

func TestDungeonFieldDispelSceptreUsesWrappedCellAndCurrentCellPrecedence(t *testing.T) {
	scene, callbacks := newDungeonFieldDispelTestScene(t)
	state := scene.gameState
	for range 4 {
		snapshot, _ := state.DungeonStateSnapshot()
		if snapshot.Facing == references.DungeonFacingEast {
			break
		}
		state.TurnDungeon(game_state.DungeonTurnRight)
	}
	player := state.MapState.PlayerLocation
	current := references.Position{X: 7, Y: player.Position.Y}
	wrapped := references.Position{X: 0, Y: player.Position.Y}
	state.TransitionToLocation(player.Location, player.Floor, current, game_state.TransitionPositionOnly, false)
	for _, position := range []references.Position{current, wrapped} {
		_, err := state.TransformDungeonCell(player.Floor, position, func(references.DungeonCell) references.DungeonCell { return 0x8b })
		require.NoError(t, err)
	}
	callbacks.Reset()
	result := state.UseDungeonSceptre()
	require.True(t, result.Cleared)
	scene.presentDungeonFieldDispel(result.Change)
	assert.Equal(t, current, result.Change.Position)
	state.ElapsedMs += 10
	scene.updateDungeonFieldDispels()
	result = state.UseDungeonSceptre()
	require.True(t, result.Cleared)
	scene.presentDungeonFieldDispel(result.Change)
	assert.Equal(t, wrapped, result.Change.Position)
	plan := dungeonFieldDispelTestPlan(t, scene, true, false)
	ops := dungeonPlanOps(plan, func(op dungeonRenderOp) bool { return op.kind == dungeonRenderFieldDispel })
	require.Len(t, ops, 2)
	assert.Equal(t, wrapped, ops[0].position)
	assert.Equal(t, 1, ops[0].depth)
	assert.Equal(t, current, ops[1].position)
	assert.Zero(t, ops[1].depth)
	assert.Equal(t, []game_state.SoundEffect{game_state.SoundDispel, game_state.SoundDispel}, callbacks.SoundEffectsPlayed)
}

func TestDungeonFieldDispelCombatEntryRetiresSynchronously(t *testing.T) {
	scene, _ := newDungeonFieldDispelTestScene(t)
	state := scene.gameState
	admitTestDungeonFieldDispel(t, scene, state.MapState.PlayerLocation.Position, 0x81)
	require.Len(t, scene.dungeonFieldDispels, 1)
	err := state.TransitionToDungeonCombatMap(references.DeceitDungeon, 0, game_state.CombatEntryNorth)
	require.NoError(t, err)
	assert.Empty(t, scene.dungeonFieldDispels, "commitCombatTransition must notify before the next scene update")
	state.TransitionFromCombatMap()
	assert.Empty(t, scene.dungeonFieldDispels)
}

func TestDungeonFieldDispelScaledCompositionContainsOnlyProjectedField(t *testing.T) {
	scene, _ := newDungeonFieldDispelTestScene(t)
	state := scene.gameState
	snapshot, _ := state.DungeonStateSnapshot()
	player := state.MapState.PlayerLocation
	position := dungeonRelativePosition(player.Position, snapshot.Facing, 1, 0)
	admitTestDungeonFieldDispel(t, scene, position, 0x82)
	state.ElapsedMs += 200
	scene.updateDungeonFieldDispels()
	withEffect := renderFieldDispelTestPlan(t, dungeonFieldDispelTestPlan(t, scene, true, false), state.ElapsedMs)
	scene.clearDungeonFieldDispels()
	baseline := renderFieldDispelTestPlan(t, dungeonFieldDispelTestPlan(t, scene, true, false), state.ElapsedMs)
	for _, scale := range []float64{1, 1.5, 2} {
		first, second := ebiten.NewImage(400, 400), ebiten.NewImage(400, 400)
		t.Cleanup(first.Deallocate)
		t.Cleanup(second.Deallocate)
		var options ebiten.DrawImageOptions
		options.GeoM.Scale(scale, scale)
		// Native projection plus a representative nonzero shake/viewport offset.
		options.GeoM.Translate(13, 17)
		first.DrawImage(withEffect, &options)
		second.DrawImage(baseline, &options)
		bounds := dungeonFieldBounds(1)
		projected := image.Rect(13+int(float64(bounds.Min.X)*scale), 17+int(float64(bounds.Min.Y)*scale),
			13+int(float64(bounds.Max.X)*scale), 17+int(float64(bounds.Max.Y)*scale))
		changed := 0
		for y := range 400 {
			for x := range 400 {
				if first.At(x, y) != second.At(x, y) {
					changed++
					require.True(t, image.Pt(x, y).In(projected), "scale=%g escaped cell at %d,%d", scale, x, y)
				}
			}
		}
		assert.Positive(t, changed)
	}
}

func TestDungeonFieldDispelProductionAnGravCommitsAndFinishesWhileVisible(t *testing.T) {
	base, callbacks := newDungeonFieldDispelTestScene(t)
	state := base.gameState
	scene := newProductionCastTestScene(state)
	t.Cleanup(scene.clearDungeonFieldDispels)
	refresh := state.SystemCallbacks.Screen.UpdateDisplay
	state.SystemCallbacks.Screen.UpdateDisplay = func() { refresh(); scene.pruneDungeonFieldDispels() }
	require.True(t, state.SetActivePlayer(0))
	caster := state.PartyState.GetCharacter(0)
	caster.Class = party_state.Wizard
	caster.Level = 8
	caster.SetMp(8)
	state.PartyState.Inventory.Spells.Set(references.AnGrav, 1)
	state.ActiveDurationEffect = game_state.DurationEffect{EffectType: game_state.DurationEffectProtection, TurnsRemaining: 5}
	player := state.MapState.PlayerLocation
	_, err := state.TransformDungeonCell(player.Floor, player.Position, func(references.DungeonCell) references.DungeonCell { return 0x8a })
	require.NoError(t, err)
	beforeDate := state.DateTime
	callbacks.Reset()
	scene.dungeonMapInputHandler(ebiten.KeyC)
	dialog := waitForProductionSpellDialog(t, scene)
	enterSpellSyllables(scene, dialog, []ebiten.Key{ebiten.KeyA, ebiten.KeyG})
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	dialog.Update()
	require.True(t, dialog.HasPendingCastRequest())
	for range 500 {
		state.ElapsedMs += 10
		state.EventQueueManager.Update()
		scene.updateDungeonFieldDispels()
		if state.IsAtDungeonCommandBoundary() {
			break
		}
	}
	require.True(t, state.IsAtDungeonCommandBoundary())
	require.Len(t, scene.dungeonFieldDispels, 1)
	assert.Equal(t, player.Position, scene.dungeonFieldDispels[0].change.Position)
	assert.EqualValues(t, 0x08, scene.dungeonFieldDispels[0].change.After)
	assert.Equal(t, byte(8-references.GetSpellCircle(references.AnGrav)), caster.GetCurrentMp())
	assert.Zero(t, state.PartyState.Inventory.Spells.Get(references.AnGrav))
	beforeDate.Advance(1)
	assert.Equal(t, beforeDate, state.DateTime)
	assert.Equal(t, 4, state.ActiveDurationEffect.TurnsRemaining)
	dispelRequests := 0
	for _, sound := range callbacks.SoundEffectsPlayed {
		if sound == game_state.SoundDispel {
			dispelRequests++
		}
	}
	assert.Equal(t, 1, dispelRequests)
	plan := dungeonFieldDispelTestPlan(t, scene, true, false)
	canvas := renderFieldDispelTestPlan(t, plan, state.ElapsedMs)
	assert.Positive(t, countDungeonFieldPixels(canvas, dungeonFieldColor(2)))
}
