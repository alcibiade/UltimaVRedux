package main

import (
	"bytes"
	"fmt"
	"image"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func combatLocalRaster(t *testing.T, options *ebiten.DrawImageOptions) *ebiten.Image {
	t.Helper()
	screen := ebiten.NewImageWithOptions(combatFixtureRasterBounds(options), &ebiten.NewImageOptions{Unmanaged: true})
	t.Cleanup(screen.Dispose)

	return screen
}

func setSquareVisualTestDisplay(t *testing.T, width, height int) {
	t.Helper()
	manager := display.GetManager()
	saved := *manager
	t.Cleanup(func() { *manager = saved })
	manager.PublishLogicalSize(width, height)
}

// Every mode has a nonzero actor control except the settled movement endpoint.
func setSquareVisualTestMotion(scene *GameScene, index int, position references.Position, mode string) {
	scene.combatMovement = CombatMovementAnimation{}
	scene.combatImpactEffects = nil
	if mode == "recoil" || mode == "lunge-miss" {
		feedback := newTestCombatFeedback(game_state.CombatFeedbackMiss)
		feedback.Attacker.Index, feedback.Attacker.Position = index, position
		feedback.Target.Position = references.Position{X: position.X + 1, Y: position.Y}
		if mode == "recoil" {
			feedback.Result = game_state.CombatFeedbackDamage
			feedback.Attacker.IsPartyMember = true
			feedback.Attacker.Position = references.Position{X: position.X - 1, Y: position.Y}
			feedback.Target = game_state.CombatFeedbackEntity{Index: index, Position: position}
		}
		scene.showCombatImpact(feedback)
		elapsed := combatLungeDurationSeconds / 2
		if mode == "recoil" {
			elapsed = combatJoltDurationSeconds / 6
		}
		scene.combatImpactEffects[0].Elapsed = elapsed
		return
	}
	progress := float32(0)
	switch mode {
	case "move-middle":
		progress = 0.5
	case "move-end":
		progress = 1
	}
	scene.combatMovement = CombatMovementAnimation{Active: true, EntityIndex: index,
		From: references.Position{X: position.X - 1, Y: position.Y}, To: position, Progress: progress}
}

func TestCombatFieldsRenderSquareAnchoredThroughActorMotion(t *testing.T) {
	for _, kind := range []game_state.CombatFieldKind{
		game_state.CombatFieldPoison, game_state.CombatFieldSleep, game_state.CombatFieldFire, game_state.CombatFieldProtection,
	} {
		for _, size := range [][2]int{{400, 208}, {1366, 768}} {
			t.Run(fmt.Sprintf("kind-%d/%dx%d", kind, size[0], size[1]), func(t *testing.T) {
				setSquareVisualTestDisplay(t, size[0], size[1])
				scene := newCombatConcealmentTestState(t)
				state, combat := scene.gameState, scene.gameState.CombatScene
				scene.screenShakeOffsetX, scene.screenShakeOffsetY = 0.35, -0.65
				state.ElapsedMs = 225 // Artwork time is held while actor feedback advances.
				combat.MonsterManager.Clear()
				combat.TurnQueue.RemoveAllMonsters()
				positions := []references.Position{{X: 9, Y: 9}, {X: 6, Y: 6}, {X: 3, Y: 9}}
				for _, position := range positions {
					setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
					require.True(t, combat.AddField(game_state.NewCombatField(kind, position)))
				}
				for index := range 2 {
					got, _ := signatureMonster(t, scene, references.MonsterOrc, references.Position{X: 3 + references.Coordinate(index), Y: 5})
					require.Equal(t, index, got)
				}
				fieldOracle := newCombatFieldSquareOracle(t, scene)
				for _, index := range []int{0, 1} {
					for _, coLocated := range []bool{false, true} {
						monster := combat.MonsterManager.GetMonster(index)
						position := references.Position{X: 3 + references.Coordinate(index), Y: 5}
						if coLocated {
							position = positions[index]
						}
						monster.SetPosition(position)
						actorScreen := combatLocalRaster(t, scene.combatSquareMarkerDrawOptions(position))
						scene.combatMovement = CombatMovementAnimation{}
						scene.combatImpactEffects = nil
						scene.drawCombatMonsterOverlay(actorScreen)
						actorBefore := signaturePixels(actorScreen)
						require.Positive(t, opaquePixelCount(actorScreen))
						for _, mode := range []string{"recoil", "lunge-miss", "move-start", "move-middle", "move-end"} {
							setSquareVisualTestMotion(scene, index, position, mode)
							actorScreen.Clear()
							scene.drawCombatMonsterOverlay(actorScreen)
							assert.Equal(t, mode == "move-end", bytes.Equal(actorBefore, signaturePixels(actorScreen)), "actor control %s", mode)
							fieldOracle.assertFields(t, scene)
						}
					}
				}
				// Reorder a retained slice and then remove an earlier field through
				// the real dispel path. Neither ordinal change may change placement.
				slices.Reverse(combat.Fields())
				setSquareVisualTestMotion(scene, 1, references.Position{X: 4, Y: 5}, "recoil")
				fieldOracle.assertFields(t, scene)
				actor := combat.GetCurrentTurnIdentity()
				combat.SetPartyMemberPosition(actor.EntityIndex, references.Position{X: 3, Y: 8})
				require.True(t, state.CastCombatAnGrav(actor.EntityIndex, references.Down))
				require.Len(t, combat.Fields(), 2)
				fieldOracle.assertFields(t, scene)
			})
		}
	}
}

func TestCombatSquareSpellVisualsIgnoreCoordinateIdentityAliases(t *testing.T) {
	scene, _ := newCombatDoorSpellScene(t, references.AnGrav)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.mapImage = ebiten.NewImage(304, 208)
	t.Cleanup(scene.mapImage.Dispose)
	setSquareVisualTestDisplay(t, 1366, 768)
	scene.screenShakeOffsetX, scene.screenShakeOffsetY = 0.35, -0.65
	state, combat := scene.gameState, scene.gameState.CombatScene
	origin, target := references.Position{X: 0, Y: 0}, references.Position{X: 0, Y: 1}
	actor := combat.GetCurrentTurnIdentity()
	combat.SetPartyMemberPosition(actor.EntityIndex, origin)
	_, _ = signatureMonster(t, scene, references.MonsterOrc, references.Position{X: 9, Y: 9})
	index, monster := signatureMonster(t, scene, references.MonsterOrc, references.Position{X: 5, Y: 5})
	require.Equal(t, int(target.X)*int(references.YCombatMapTiles)+int(target.Y), index)
	require.Equal(t, int(target.X)*16+int(target.Y), index)
	screen := combatLocalRaster(t, scene.combatSquareMarkerDrawOptions(target))
	actorScreen := combatLocalRaster(t, scene.combatSquareMarkerDrawOptions(monster.Position()))
	for _, visual := range []string{"cone-preview", "cone-cast", "adjacent-preview", "disarm", "lock", "unlock"} {
		t.Run(visual, func(t *testing.T) {
			combat.ConeTargeting, combat.AdjacentSpellTargeting = nil, nil
			scene.combatConeCast, scene.combatUtilityEffect = nil, nil
			var draw func(*ebiten.Image)
			switch visual {
			case "cone-preview":
				combat.ConeTargeting = &game_state.CombatConeTargeting{Request: game_state.CastRequest{Spell: references.InNoxHur},
					Origin: origin, Direction: references.Down, Spec: game_state.CombatConeSpec{BlockedByTile: true}}
				require.Contains(t, state.CombatConePreview(), target)
				draw = scene.drawCombatConeOverlay
			case "cone-cast":
				scene.combatConeCast = &combatConeCastEffect{spell: references.InNoxHur, origin: origin,
					direction: references.Down, positions: []references.Position{target}, progress: 1}
				draw = scene.drawCombatConeOverlay
			case "adjacent-preview":
				require.True(t, state.BeginCombatAdjacentSpellTargeting(game_state.CastRequest{
					CasterIdx: actor.EntityIndex, Spell: references.AnGrav}).Valid)
				state.SetCombatAdjacentSpellDirection(references.Down)
				got, _, active := state.CombatAdjacentSpellPreview()
				require.True(t, active)
				require.Equal(t, target, got)
				draw = scene.drawCombatAdjacentSpellTargetPreview
			default:
				spell := references.AnSanct
				if visual == "lock" {
					spell = references.AnExPor
				} else if visual == "unlock" {
					spell = references.InExPor
				}
				scene.combatUtilityEffect = &combatUtilitySpellEffect{Spell: spell, Position: target,
					Phase: combatUtilityApproach, Progress: 0.5}
				draw = scene.drawCombatUtilitySpellEffects
			}
			scene.combatMovement = CombatMovementAnimation{}
			scene.combatImpactEffects = nil
			screen.Clear()
			draw(screen)
			require.Positive(t, opaquePixelCount(screen), "production path must actually draw")
			before := signaturePixels(screen)
			actorScreen.Clear()
			scene.drawCombatMonsterOverlay(actorScreen)
			actorBefore := signaturePixels(actorScreen)
			for _, motion := range []string{"recoil", "lunge-miss", "move-middle"} {
				setSquareVisualTestMotion(scene, index, monster.Position(), motion)
				actorScreen.Clear()
				scene.drawCombatMonsterOverlay(actorScreen)
				require.NotEqual(t, actorBefore, signaturePixels(actorScreen), "actor positive control: %s", motion)
				screen.Clear()
				draw(screen)
				assert.Equal(t, before, signaturePixels(screen), "%s inherits no %s", visual, motion)
			}
		})
	}
}

func TestCombatTeleportCueRetainsExactActorMotion(t *testing.T) {
	scene, actor := newCombatMarkerTestScene(t)
	combat := scene.gameState.CombatScene
	require.True(t, scene.gameState.CastCombatInQuasXen(actor.EntityIndex, actor))
	var clone game_state.CombatTurnIdentity
	combat.ForEachPCClone(func(identity game_state.CombatTurnIdentity, _ references.Position) { clone = identity })
	require.Positive(t, clone.Slot)
	position := combat.GetCombatantPosition(clone)
	setCombatConcealmentTerrain(scene.gameState, position, indexes.BrickFloor)
	scene.combatTeleportEffect = &combatTeleportSpellEffect{Caster: clone, From: position, To: position,
		Phase: combatTeleportArrival, Progress: 0.5}
	screen := combatLocalRaster(t, scene.combatSquareMarkerDrawOptions(position))
	scene.drawCombatUtilitySpellEffects(screen)
	require.Positive(t, opaquePixelCount(screen))
	before := signaturePixels(screen)
	scene.combatMovement = CombatMovementAnimation{Active: true, IsPartyMember: true, EntityIndex: clone.EntityIndex,
		Slot: clone.Slot, From: references.Position{X: position.X - 1, Y: position.Y}, To: position, Progress: 0.5}
	screen.Clear()
	scene.drawCombatUtilitySpellEffects(screen)
	assert.NotEqual(t, before, signaturePixels(screen), "the teleport cue remains actor attached")
	scene.combatMovement.Slot = actor.Slot
	screen.Clear()
	scene.drawCombatUtilitySpellEffects(screen)
	assert.Equal(t, before, signaturePixels(screen), "another exact entry cannot move the clone's cue")
}

// The motion matrix holds artwork time, field positions and display transforms
// fixed. Cache only its independent per-square artwork oracle, never production
// output. A union target batches readback while retaining every original guarded
// rectangle in native screen coordinates (including fractional screen shake).
type combatFieldSquareOracle struct {
	screen    *ebiten.Image
	pixels    *image.RGBA
	expected  map[references.Position]*image.RGBA
	elapsedMs int64
}

func newCombatFieldSquareOracle(t *testing.T, scene *GameScene) *combatFieldSquareOracle {
	t.Helper()
	oracle := &combatFieldSquareOracle{
		expected: make(map[references.Position]*image.RGBA), elapsedMs: scene.getElapsedMs(),
	}
	var bounds image.Rectangle
	for _, field := range scene.gameState.CombatScene.Fields() {
		position := field.Position()
		options := scene.combatSquareMarkerDrawOptions(position)
		region := combatFixtureRasterBounds(options)
		expected := ebiten.NewImageWithOptions(region, &ebiten.NewImageOptions{Unmanaged: true})
		hash := int32(position.X)*int32(references.YCombatMapTiles) + int32(position.Y)
		expected.DrawImage(scene.spriteSheet.GetAnimatedSprite(field.Sprite(), hash, oracle.elapsedMs), options)
		pixels := image.NewRGBA(region)
		expected.ReadPixels(pixels.Pix)
		expected.Dispose()
		opaque := 0
		for offset := 3; offset < len(pixels.Pix); offset += 4 {
			if pixels.Pix[offset] != 0 {
				opaque++
			}
		}
		require.Positive(t, opaque, "oracle for field %v must actually draw", position)
		oracle.expected[position] = pixels
		bounds = bounds.Union(region)
	}
	oracle.screen = ebiten.NewImageWithOptions(bounds, &ebiten.NewImageOptions{Unmanaged: true})
	t.Cleanup(oracle.screen.Dispose)
	oracle.pixels = image.NewRGBA(bounds)

	return oracle
}

func (oracle *combatFieldSquareOracle) assertFields(t *testing.T, scene *GameScene) {
	t.Helper()
	require.Equal(t, oracle.elapsedMs, scene.getElapsedMs(), "cached artwork time must remain fixed")
	oracle.screen.Clear()
	scene.drawCombatFieldOverlay(oracle.screen)
	oracle.screen.ReadPixels(oracle.pixels.Pix)
	for _, field := range scene.gameState.CombatScene.Fields() {
		position := field.Position()
		want, found := oracle.expected[position]
		require.True(t, found, "field %v needs its own oracle", position)
		require.Equal(t, want.Rect, combatFixtureRasterBounds(scene.combatSquareMarkerDrawOptions(position)))
		// Compare every RGBA byte, including transparent guard pixels. Reading
		// the union once is equivalent to reading these same rectangles separately.
		equal := true
		for y := want.Rect.Min.Y; y < want.Rect.Max.Y; y++ {
			wantOffset := want.PixOffset(want.Rect.Min.X, y)
			gotOffset := oracle.pixels.PixOffset(want.Rect.Min.X, y)
			equal = equal && bytes.Equal(want.Pix[wantOffset:wantOffset+want.Stride],
				oracle.pixels.Pix[gotOffset:gotOffset+want.Stride])
		}
		assert.True(t, equal, "field %v stays square anchored", position)
	}
}

func assertCombatFieldAtSquare(t *testing.T, scene *GameScene, field game_state.CombatField) {
	t.Helper()
	options := scene.combatSquareMarkerDrawOptions(field.Position())
	screen := ebiten.NewImageWithOptions(combatFixtureRasterBounds(options), &ebiten.NewImageOptions{Unmanaged: true})
	expected := ebiten.NewImageWithOptions(screen.Bounds(), &ebiten.NewImageOptions{Unmanaged: true})
	defer screen.Dispose()
	defer expected.Dispose()
	position := field.Position()
	hash := int32(position.X)*int32(references.YCombatMapTiles) + int32(position.Y)
	expected.DrawImage(scene.spriteSheet.GetAnimatedSprite(field.Sprite(), hash, scene.getElapsedMs()), options)
	scene.drawCombatFieldOverlay(screen)
	require.Positive(t, opaquePixelCount(expected))
	assert.True(t, bytes.Equal(signaturePixels(expected), signaturePixels(screen)), "field %v stays square anchored", position)
}

func TestCombatFieldArtworkStillAnimatesWithPositionPhaseAndConcealment(t *testing.T) {
	scene := newCombatConcealmentTestState(t)
	state, combat := scene.gameState, scene.gameState.CombatScene
	for ordinal, kind := range []game_state.CombatFieldKind{
		game_state.CombatFieldPoison, game_state.CombatFieldSleep, game_state.CombatFieldFire, game_state.CombatFieldProtection,
	} {
		position := references.Position{X: references.Coordinate(2 + ordinal*2), Y: 5}
		setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
		require.True(t, combat.AddField(game_state.NewCombatField(kind, position)))
		field := combat.Fields()[ordinal]
		screen := combatLocalRaster(t, scene.combatSquareMarkerDrawOptions(position))
		state.ElapsedMs = 0
		scene.drawCombatFieldOverlay(screen)
		before := signaturePixels(screen)
		changed := false
		interval := sprites.TileEffectRegistry[field.Sprite()].IntervalMs
		if interval == 0 {
			interval = 100
		}
		for frame := 1; frame <= 4; frame++ {
			state.ElapsedMs = int64(frame * interval)
			assertCombatFieldAtSquare(t, scene, field)
			screen.Clear()
			scene.drawCombatFieldOverlay(screen)
			changed = changed || !bytes.Equal(before, signaturePixels(screen))
		}
		assert.True(t, changed, "field kind %d still animates", kind)
		for _, terrain := range []indexes.SpriteIndex{indexes.BlackSquare, indexes.BrickWallArchway} {
			setCombatConcealmentTerrain(state, position, terrain)
			screen.Clear()
			scene.drawCombatFieldOverlay(screen)
			assert.Zero(t, opaquePixelCount(screen))
		}
		setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
		scene.combatLootPreview = &combatLootProjection{position: position, sprite: indexes.ItemPotion}
		screen.Clear()
		scene.drawCombatFieldOverlay(screen)
		assert.Zero(t, opaquePixelCount(screen))
		scene.combatLootPreview = nil
		require.True(t, combat.AddObject(game_state.NewCombatLooseItem(position, references.Gem, 1, indexes.ItemGem)))
		screen.Clear()
		scene.drawCombatFieldOverlay(screen)
		assert.Zero(t, opaquePixelCount(screen))
	}
}
