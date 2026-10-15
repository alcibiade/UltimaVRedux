package main

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func newDungeonInjuryTestScene(t *testing.T) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()
	scene, callbacks := newDungeonViewTestScene(t)
	state := scene.gameState
	state.SetRandomSeed(43)
	for index := range state.PartyState.GetCharacterCount() {
		state.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapUnrecruited
	}
	character := state.PartyState.GetCharacter(0)
	character.PartyMap = party_state.PartyMapActive
	character.SetMaxHp(1000)
	character.SetHp(1000)
	character.SetStatus(party_state.Good)
	character.Dexterity = 1
	// Keep incidental dungeon actors out of these presentation integration tests.
	state.ActiveDurationEffect = game_state.DurationEffect{EffectType: game_state.DurationEffectTimeStop, TurnsRemaining: 100}
	state.ElapsedMs = 1377
	record := state.SystemCallbacks.Visual.ShowDungeonInjury
	state.SystemCallbacks.Visual.ShowDungeonInjury = func(kind game_state.DungeonInjuryKind) {
		record(kind)
		scene.showDungeonInjury(kind)
	}
	t.Cleanup(func() {
		state.EventQueueManager.Shutdown()
		scene.clearDungeonInjury()
	})

	return scene, callbacks
}

func setDungeonInjuryTestCell(t *testing.T, scene *GameScene, position references.Position, cell references.DungeonCell) {
	t.Helper()
	_, err := scene.gameState.TransformDungeonCell(scene.gameState.MapState.PlayerLocation.Floor,
		position, func(references.DungeonCell) references.DungeonCell { return cell })
	require.NoError(t, err)
}

func TestDungeonInjuryTintHasBriefTranslucentPeakAndCleanFade(t *testing.T) {
	for _, test := range []struct {
		kind game_state.DungeonInjuryKind
		base ucolor.EGAColor
	}{
		{game_state.DungeonInjuryFire, ucolor.EGALightRed},
		{game_state.DungeonInjuryElectrical, ucolor.EGALightBlue},
		{game_state.DungeonInjuryPoison, ucolor.EGALightGreen},
	} {
		t.Run(fmt.Sprint(test.kind), func(t *testing.T) {
			initial := dungeonInjuryTint(test.kind, 0)
			assert.Equal(t, color.NRGBA{R: test.base.R, G: test.base.G, B: test.base.B, A: ucolor.AlphaLight}, initial)
			assert.Less(t, initial.A, uint8(128), "terrain and text remain readable")
			assert.Equal(t, initial, dungeonInjuryTint(test.kind, 40))
			previous := initial.A
			for elapsed := int64(41); elapsed <= 200; elapsed++ {
				current := dungeonInjuryTint(test.kind, elapsed)
				assert.LessOrEqual(t, current.A, previous)
				previous = current.A
			}
			assert.Zero(t, previous)
			assert.Equal(t, color.NRGBA{}, dungeonInjuryTint(test.kind, -1))
			assert.Equal(t, color.NRGBA{}, dungeonInjuryTint(test.kind, 1000))
		})
	}
}

func TestDungeonInjuryViewportVisibleUnlitWhileNormalFlashAndInvertStaySuppressed(t *testing.T) {
	manager := display.GetManager()
	saved := *manager
	t.Cleanup(func() { *manager = saved })
	manager.PublishLogicalSize(320, 200)
	scene, _ := newDungeonInjuryTestScene(t)
	scene.screenFlashR, scene.screenFlashG, scene.screenFlashB = 1, 1, 1
	scene.screenFlashIntensity, scene.screenInvertIntensity = 1, 1
	viewport := scene.mapViewportRectangle()
	for _, kind := range []game_state.DungeonInjuryKind{
		game_state.DungeonInjuryFire, game_state.DungeonInjuryElectrical, game_state.DungeonInjuryPoison,
	} {
		scene.showDungeonInjury(kind)
		canvas := ebiten.NewImage(320, 200)
		t.Cleanup(canvas.Deallocate)
		canvas.Fill(color.Black)
		scene.drawViewportFlashEffects(canvas, false)
		inside := color.RGBAModel.Convert(canvas.At(viewport.Dx()/2, 100)).(color.RGBA)
		assert.NotEqual(t, color.RGBA{A: 255}, inside, "injury is perceptible on complete darkness")
		assert.LessOrEqual(t, max(inside.R, inside.G, inside.B), uint8(ucolor.AlphaLight+1),
			"premultiplied tint is translucent, without white flash or inversion")
		assert.Equal(t, color.RGBA{A: 255}, color.RGBAModel.Convert(canvas.At(viewport.Max.X+1, 100)),
			"HP/messages in the right rail are untouched")
		scene.clearDungeonInjury()
		canvas.Fill(color.Black)
		scene.drawViewportFlashEffects(canvas, false)
		assert.Equal(t, color.RGBA{A: 255}, color.RGBAModel.Convert(canvas.At(10, 10)))
	}
	assert.Equal(t, 1.0, scene.screenFlashIntensity, "injury never borrows or clears the normal flash channel")
	assert.Equal(t, 1.0, scene.screenInvertIntensity)
}

func TestDungeonInjuryCompositesAfterLitBlackAndSubduedPlansWithoutTerrainDisclosure(t *testing.T) {
	scene, _ := newDungeonFieldDispelTestScene(t)
	state := scene.gameState
	player := state.MapState.PlayerLocation
	admitTestDungeonFieldDispel(t, scene, player.Position, 0x82)
	snapshot, active := state.DungeonStateSnapshot()
	require.True(t, active)
	snapshot.Wanderer = nil
	require.True(t, snapshot.Cells.SetCell(player.Floor, player.Position, 0x82))
	for _, lighting := range []struct {
		lit, subdued bool
	}{{true, false}, {false, false}, {false, true}} {
		plan := dungeonFieldDispelTestPlan(t, scene, lighting.lit, lighting.subdued)
		if !lighting.lit {
			assert.Empty(t, dungeonPlanOps(plan, func(op dungeonRenderOp) bool {
				return op.kind == dungeonRenderMonster
			}), "subdued geometry must not disclose unlit monsters")
		}
		if !lighting.lit && !lighting.subdued {
			require.True(t, plan.dark)
			require.Zero(t, plan.count, "complete darkness admits no terrain, field, or fizzle operations")
		}
		if lighting.subdued {
			require.Equal(t, dungeonSubduedUnlitBrightness, plan.brightness)
			require.False(t, plan.trueLight)
			require.Len(t, dungeonPlanOps(plan, func(op dungeonRenderOp) bool {
				return op.kind == dungeonRenderFieldDispel
			}), 1, "subdued mode retains the same dim field pixels as ordinary field rendering")
		}
		baseline := renderFieldDispelTestPlan(t, plan, state.ElapsedMs)
		ordinary := buildDungeonRenderPlan(&snapshot, player, lighting.lit, lighting.subdued, false, state.ElapsedMs)
		assertDungeonImagesEqual(t, renderFieldDispelTestPlan(t, ordinary, state.ElapsedMs), baseline)
		canvas := ebiten.NewImage(dungeonCanvasWidth+40, dungeonCanvasHeight+10)
		t.Cleanup(canvas.Deallocate)
		canvas.Fill(color.RGBA{R: 21, G: 31, B: 41, A: 255})
		var options ebiten.DrawImageOptions
		options.GeoM.Translate(5, 5)
		canvas.DrawImage(baseline, &options)
		viewport := image.Rect(5, 5, dungeonCanvasWidth+5, dungeonCanvasHeight+5)
		tint := dungeonInjuryTint(game_state.DungeonInjuryFire, 0)
		drawDungeonInjuryTint(canvas, viewport, tint)
		for y := range canvas.Bounds().Dy() {
			for x := range canvas.Bounds().Dx() {
				actual := color.RGBAModel.Convert(canvas.At(x, y)).(color.RGBA)
				if !image.Pt(x, y).In(viewport) {
					require.Equal(t, color.RGBA{R: 21, G: 31, B: 41, A: 255}, actual)
					continue
				}
				background := color.RGBAModel.Convert(baseline.At(x-5, y-5)).(color.RGBA)
				if !lighting.lit && !lighting.subdued {
					require.Equal(t, color.RGBA{A: 255}, background, "injury overlays black, never hidden terrain")
				}
				// Source-over depends only on the already-visible pixel, never on
				// terrain, field contents, lighting radius, or hidden map data.
				for _, channel := range [][3]uint8{{actual.R, background.R, tint.R}, {actual.G, background.G, tint.G}, {actual.B, background.B, tint.B}} {
					expected := (float64(channel[1])*float64(255-tint.A) + float64(channel[2])*float64(tint.A)) / 255
					require.InDelta(t, expected, channel[0], 2)
				}
			}
		}
	}
}

func TestDungeonInjuryProductionCombinedSourcesSerializeAndExpireWithoutBlocking(t *testing.T) {
	scene, callbacks := newDungeonInjuryTestScene(t)
	state := scene.gameState
	character := state.PartyState.GetCharacter(0)
	character.SetStatus(party_state.Poisoned)
	player := state.MapState.PlayerLocation
	snapshot, active := state.DungeonStateSnapshot()
	require.True(t, active)
	front := dungeonRelativePosition(player.Position, snapshot.Facing, 1, 0)
	setDungeonInjuryTestCell(t, scene, player.Position, 0x82)
	setDungeonInjuryTestCell(t, scene, front, 0x83)
	callbacks.Reset()
	for range 10 {
		scene.dungeonMapInputHandler(ebiten.KeyUp)
		require.True(t, state.IsAtDungeonCommandBoundary())
		require.False(t, callbacks.InputBlocked)
	}
	assert.Equal(t, player.Position, state.MapState.PlayerLocation.Position)
	require.Len(t, callbacks.SoundEffectsPlayed, 30, "audio is requested at every resolved source admission")
	for index := 0; index < len(callbacks.SoundEffectsPlayed); index += 3 {
		assert.Equal(t, []game_state.SoundEffect{game_state.SoundElectricalDamage, game_state.SoundFireBurn, game_state.SoundPoison},
			callbacks.SoundEffectsPlayed[index:index+3])
	}
	beforeHP, beforeRNG, beforeDate := character.GetCurrentHp(), state.GetRNG().Calls(), state.DateTime
	visibleFrames := make(map[game_state.DungeonInjuryKind]int)
	for range 80 {
		state.EventQueueManager.UpdateFeedback()
		scene.updateDungeonInjury()
		if scene.dungeonInjuryVisible() {
			visibleFrames[scene.dungeonInjury.kind]++
		}
		state.ElapsedMs += 10
	}
	assert.Equal(t, []game_state.DungeonInjuryKind{
		game_state.DungeonInjuryElectrical, game_state.DungeonInjuryNone,
		game_state.DungeonInjuryFire, game_state.DungeonInjuryNone,
		game_state.DungeonInjuryPoison, game_state.DungeonInjuryNone,
	}, callbacks.DungeonInjuryCalls)
	for _, kind := range []game_state.DungeonInjuryKind{
		game_state.DungeonInjuryElectrical, game_state.DungeonInjuryFire, game_state.DungeonInjuryPoison,
	} {
		assert.Equal(t, 20, visibleFrames[kind], "each source stays visible for its complete 200 ms pulse")
	}
	assert.Equal(t, dungeonInjuryPulse{}, scene.dungeonInjury)
	assert.Equal(t, beforeHP, character.GetCurrentHp())
	assert.Equal(t, beforeRNG, state.GetRNG().Calls())
	assert.Equal(t, beforeDate, state.DateTime)
	assert.Zero(t, callbacks.BlockInputCalls)
	assert.Zero(t, scene.screenFlashIntensity)
}

func TestDungeonInjuryFatalProductionPassStillPresentsAndCleansUp(t *testing.T) {
	scene, callbacks := newDungeonInjuryTestScene(t)
	state := scene.gameState
	character := state.PartyState.GetCharacter(0)
	character.SetHp(1)
	setDungeonInjuryTestCell(t, scene, state.MapState.PlayerLocation.Position, 0x82)
	deaths := 0
	state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() { deaths++ }
	callbacks.Reset()
	scene.dungeonMapInputHandler(ebiten.KeySpace)
	require.Equal(t, 1, deaths)
	require.True(t, character.IsDead())
	require.Equal(t, []game_state.SoundEffect{game_state.SoundFireBurn}, callbacks.SoundEffectsPlayed)
	state.EventQueueManager.UpdateFeedback()
	require.True(t, scene.dungeonInjuryVisible(), "fatal injury is not filtered by the now-dead party")
	require.Equal(t, []game_state.DungeonInjuryKind{game_state.DungeonInjuryFire}, callbacks.DungeonInjuryCalls)
	state.ElapsedMs += 100
	state.EventQueueManager.UpdateFeedback()
	scene.updateDungeonInjury()
	require.True(t, scene.dungeonInjuryVisible(), "fatal injury remains visible during its fade")
	tint := dungeonInjuryTint(scene.dungeonInjury.kind, state.ElapsedMs-scene.dungeonInjury.startedAt)
	assert.Positive(t, tint.A)
	assert.Less(t, tint.A, uint8(ucolor.AlphaLight))
	state.EventQueueManager.Shutdown()
	assert.Equal(t, dungeonInjuryPulse{}, scene.dungeonInjury)
	assert.Equal(t, []game_state.DungeonInjuryKind{game_state.DungeonInjuryFire, game_state.DungeonInjuryNone}, callbacks.DungeonInjuryCalls)
	assert.Equal(t, 1, deaths)
	assert.False(t, callbacks.InputBlocked)
}

func TestDungeonInjurySceneLifecycleRetiresPulses(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*GameScene)
	}{
		{"complete", func(scene *GameScene) { scene.gameState.ElapsedMs += 200 }},
		{"clock rewind", func(scene *GameScene) { scene.gameState.ElapsedMs-- }},
		{"floor", func(scene *GameScene) { scene.gameState.MapState.PlayerLocation.Floor++ }},
		{"location", func(scene *GameScene) { scene.gameState.MapState.PlayerLocation.Location = references.Britain }},
		{"combat", func(scene *GameScene) { scene.gameState.CombatScene = &game_state.CombatState{} }},
		{"replaced state", func(scene *GameScene) { scene.gameState = nil }},
		{"disposed", func(scene *GameScene) { scene.disposed = true }},
	} {
		t.Run(test.name, func(t *testing.T) {
			scene, _ := newDungeonInjuryTestScene(t)
			scene.showDungeonInjury(game_state.DungeonInjuryPoison)
			require.True(t, scene.dungeonInjuryVisible())
			test.change(scene)
			assert.False(t, scene.dungeonInjuryVisible(), "draw defensively expires before an update")
			scene.updateDungeonInjury()
			assert.Equal(t, dungeonInjuryPulse{}, scene.dungeonInjury)
			scene.clearDungeonInjury()
			assert.Equal(t, dungeonInjuryPulse{}, scene.dungeonInjury)
		})
	}
}

func TestDungeonInjurySceneReplacementAndDisposeClearPendingSources(t *testing.T) {
	scene, callbacks := newDungeonInjuryTestScene(t)
	state := scene.gameState
	setDungeonInjuryTestCell(t, scene, state.MapState.PlayerLocation.Position, 0x82)
	state.PartyState.GetCharacter(0).SetStatus(party_state.Poisoned)
	scene.dungeonMapInputHandler(ebiten.KeySpace)
	state.EventQueueManager.UpdateFeedback()
	require.True(t, scene.dungeonInjuryVisible())
	require.True(t, state.EventQueueManager.HasFeedback("dungeon_injury.fire"))
	require.True(t, state.EventQueueManager.HasFeedback("dungeon_injury.poison"))
	require.Equal(t, []game_state.DungeonInjuryKind{game_state.DungeonInjuryFire}, callbacks.DungeonInjuryCalls,
		"replacement exercises an active fire pulse and a genuinely pending poison source")
	scene.PrepareForSceneReplacement()
	assert.Equal(t, dungeonInjuryPulse{}, scene.dungeonInjury)
	scene.Dispose()
	scene.Dispose()
	state.ElapsedMs += 1000
	state.EventQueueManager.UpdateFeedback()
	assert.Equal(t, dungeonInjuryPulse{}, scene.dungeonInjury)
	assert.False(t, state.EventQueueManager.HasFeedback("dungeon_injury.fire"))
	assert.False(t, state.EventQueueManager.HasFeedback("dungeon_injury.poison"))
	assert.Equal(t, []game_state.DungeonInjuryKind{game_state.DungeonInjuryFire, game_state.DungeonInjuryNone}, callbacks.DungeonInjuryCalls,
		"pending poison is canceled, not admitted into the replacement scene")
}

func TestDungeonInjurySameCoordinateFreshVisitDoesNotRevivePulse(t *testing.T) {
	scene, _ := newDungeonInjuryTestScene(t)
	state := scene.gameState
	scene.showDungeonInjury(game_state.DungeonInjuryFire)
	player := state.MapState.PlayerLocation
	snapshot, active := state.DungeonStateSnapshot()
	require.True(t, active)
	_, err := state.DebugSetupDungeonRoute(player.Location, player.Floor, player.Position, snapshot.Facing, snapshot.LastDirection)
	require.NoError(t, err)
	assert.False(t, scene.dungeonInjuryVisible())
	scene.updateDungeonInjury()
	assert.Equal(t, dungeonInjuryPulse{}, scene.dungeonInjury)
}
