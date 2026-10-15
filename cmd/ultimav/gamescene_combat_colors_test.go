package main

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestGetCombatAimColor_UsesSemanticPalette(t *testing.T) {
	tests := []struct {
		name        string
		disposition game_state.CombatAimDisposition
		intensity   float32
		expected    ucolor.EGAColor
	}{
		{"empty bright", game_state.CombatAimEmpty, 1, ucolor.CombatAimEmptyBright},
		{"empty dim", game_state.CombatAimEmpty, 0, ucolor.CombatAimEmptyDim},
		{"friendly bright", game_state.CombatAimFriendly, 1, ucolor.CombatAimFriendlyBright},
		{"friendly dim", game_state.CombatAimFriendly, 0, ucolor.CombatAimFriendlyDim},
		{"enemy bright", game_state.CombatAimEnemy, 1, ucolor.CombatAimEnemyBright},
		{"enemy dim", game_state.CombatAimEnemy, 0, ucolor.CombatAimEnemyDim},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected.ToRGBA(), getCombatAimColor(test.disposition, test.intensity))
		})
	}
}

func TestGetCombatAimColor_InterpolatesVGAIntermediate(t *testing.T) {
	actual := getCombatAimColor(game_state.CombatAimEmpty, 0.5)
	assert.Equal(t, color.RGBA{R: 212, G: 212, B: 0, A: 255}, actual)
}

func TestCombatOutlinePulses_AreQuantizedAndBounded(t *testing.T) {
	tests := []struct {
		name   string
		pulse  func(int64) float32
		period int64
		low    float32
		high   float32
		steps  int
	}{
		{
			"active PC", activePCOutlinePulse,
			sprites.ActivePCOutlinePulsePeriodMs,
			sprites.ActivePCOutlinePulseLow, sprites.ActivePCOutlinePulseHigh,
			sprites.ActivePCOutlinePulseSteps,
		},
		{
			"active monster", activeMonsterOutlinePulse,
			sprites.ActiveMonsterOutlinePulsePeriodMs,
			sprites.ActiveMonsterOutlinePulseLow, sprites.ActiveMonsterOutlinePulseHigh,
			sprites.ActiveMonsterOutlinePulseSteps,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.InDelta(t, test.low, test.pulse(0), 0.001)
			assert.InDelta(t, test.high, test.pulse(test.period/2), 0.001)

			values := make(map[float32]struct{})
			for elapsed := int64(0); elapsed < test.period; elapsed += 10 {
				value := test.pulse(elapsed)
				assert.GreaterOrEqual(t, value, test.low)
				assert.LessOrEqual(t, value, test.high)
				values[value] = struct{}{}
			}
			assert.LessOrEqual(t, len(values), test.steps)
		})
	}
}

func TestCombatReticleFrameHasIndependentOpaqueSemanticRoles(t *testing.T) {
	assert.Equal(t, color.RGBA{R: 255, G: 255, B: 255, A: 255}, ucolor.CombatAimFrameFace.ToRGBA())
	assert.Equal(t, color.RGBA{A: 255}, ucolor.CombatAimFrameBorder.ToRGBA())
	assert.NotEqual(t, ucolor.CombatAimFrameFace, ucolor.CombatWakeMarkerFace)
}

func TestCombatReticleProductionDispositionAndPulseOnlyChangeCentre(t *testing.T) {
	scene, actor := newCombatMarkerTestScene(t)
	state := scene.gameState
	_, monster := signatureMonster(t, scene, references.MonsterOrc, references.Position{X: 7, Y: 5})
	partyPosition := state.CombatScene.GetCombatantPosition(actor)
	emptyPosition := references.Position{X: 9, Y: 9}
	setCombatConcealmentTerrain(state, emptyPosition, indexes.BrickFloor)
	for _, test := range []struct {
		name        string
		position    references.Position
		disposition game_state.CombatAimDisposition
		hidden      bool
	}{
		{"enemy", monster.Position(), game_state.CombatAimEnemy, false},
		{"friendly", partyPosition, game_state.CombatAimFriendly, false},
		{"empty", emptyPosition, game_state.CombatAimEmpty, false},
		{"invisible enemy", monster.Position(), game_state.CombatAimEmpty, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			monster.SetInvisible(test.hidden)
			require.Equal(t, test.disposition, state.GetCombatAimDisposition(test.position))
			state.CombatScene.AttackTargeting = &game_state.CombatAttackTargeting{Cursor: test.position}
			options := scene.combatSquareMarkerDrawOptions(test.position)
			screen := ebiten.NewImageWithOptions(combatFixtureRasterBounds(options), &ebiten.NewImageOptions{Unmanaged: true})
			defer screen.Dispose()
			for _, elapsed := range []int64{0, 225, 450, 675, 900} {
				state.ElapsedMs = elapsed
				screen.Clear()
				scene.drawCombatAttackCrosshair(screen)
				assertCombatMarkerPixels(t, screen, options, selectedReticleRaster, ucolor.EGAWhite,
					getCombatAimColor(test.disposition, combatAimPulse(elapsed)), make([]byte, screen.Bounds().Dx()*screen.Bounds().Dy()*4))
			}
			state.CombatScene.AttackTargeting = nil
			screen.Clear()
			scene.drawCombatAttackCrosshair(screen)
			assert.Zero(t, opaquePixelCount(screen), "ending targeting removes both frame and centre")
		})
	}
}

func TestCombatPartySpellReticleSeparatesRecipientAndWakingCopy(t *testing.T) {
	scene, actor := newCombatMarkerTestScene(t)
	state, combat := scene.gameState, scene.gameState.CombatScene
	require.True(t, state.CastCombatInQuasXen(actor.EntityIndex, actor))
	var clone game_state.CombatTurnIdentity
	var clonePosition references.Position
	combat.ForEachPCClone(func(identity game_state.CombatTurnIdentity, position references.Position) {
		clone, clonePosition = identity, position
	})
	require.Positive(t, clone.Slot)
	originalPosition := combat.GetCombatantPosition(actor)
	require.NotEqual(t, originalPosition, clonePosition)
	setCombatConcealmentTerrain(state, clonePosition, indexes.BrickFloor)
	combat.PartySleeping[actor.EntityIndex] = true
	state.PartyState.GetCharacter(actor.EntityIndex).SetStatus(party_state.Sleep)
	combat.SpellTargeting = &game_state.CombatSpellTargeting{
		Request: game_state.CastRequest{Spell: references.AnZu, CasterIdx: actor.EntityIndex},
		Caster:  actor, Cursor: clonePosition, Recipient: clone, RecipientSet: true,
	}
	preview := state.ReviewCombatPartySpellRecipient()
	require.True(t, preview.Valid)
	require.True(t, preview.ShowWakingPosition)
	require.Equal(t, originalPosition, preview.WakingPosition)
	cursorOptions := scene.combatSquareMarkerDrawOptions(clonePosition)
	wakeOptions := scene.combatSquareMarkerDrawOptions(originalPosition)
	// Clone placement samples the whole battlefield, not adjacent squares. Keep
	// each cue in native screen coordinates but allocate only its local footprint;
	// a union would also allocate the potentially large empty span between them.
	cursorBounds := combatFixtureRasterBounds(cursorOptions)
	wakeBounds := combatFixtureRasterBounds(wakeOptions)
	cursorPixels, wakePixels := cursorBounds.Dx()*cursorBounds.Dy(), wakeBounds.Dx()*wakeBounds.Dy()
	require.LessOrEqual(t, cursorPixels+wakePixels, 256*256, "combined local raster area: cursor=%v wake=%v", cursorBounds, wakeBounds)
	cursorScreen := ebiten.NewImageWithOptions(cursorBounds, &ebiten.NewImageOptions{Unmanaged: true})
	wakeScreen := ebiten.NewImageWithOptions(wakeBounds, &ebiten.NewImageOptions{Unmanaged: true})
	defer cursorScreen.Dispose()
	defer wakeScreen.Dispose()
	// A moving monster in slot zero must not drag the square-based wake cue.
	scene.combatMovement = CombatMovementAnimation{Active: true, EntityIndex: 0,
		From: references.Position{X: 1, Y: 1}, To: references.Position{X: 2, Y: 1}, Progress: 0.5}
	scene.drawCombatAttackCrosshair(cursorScreen)
	scene.drawCombatAttackCrosshair(wakeScreen)
	assertCombatMarkerPixels(t, cursorScreen, cursorOptions, selectedReticleRaster, ucolor.EGAWhite, widgets.DialogControlKeyColor, nil)
	assertCombatMarkerPixels(t, wakeScreen, wakeOptions, wakeReticleRaster, ucolor.CombatWakeMarkerFace, ucolor.CombatWakeMarkerFace, nil)
	// The wake renderer itself has no selection corners, even when the original
	// and clone share roster HP/status. Only the exact cursor has the white frame.
	wakeScreen.Clear()
	scene.drawCombatPartySpellWakeMarker(wakeScreen)
	assertCombatMarkerPixels(t, wakeScreen, wakeOptions, wakeReticleRaster, ucolor.CombatWakeMarkerFace, ucolor.CombatWakeMarkerFace,
		make([]byte, wakePixels*4))
	setCombatConcealmentTerrain(state, originalPosition, indexes.BlackSquare)
	wakeScreen.Clear()
	scene.drawCombatPartySpellWakeMarker(wakeScreen)
	assert.Zero(t, opaquePixelCount(wakeScreen), "a concealed waking copy is never marked")
	setCombatConcealmentTerrain(state, originalPosition, indexes.BrickFloor)
	combat.SpellTargeting.Cursor = originalPosition
	combat.SpellTargeting.Recipient = actor
	wakeScreen.Clear()
	scene.drawCombatPartySpellWakeMarker(wakeScreen)
	assert.Zero(t, opaquePixelCount(wakeScreen), "coincident wake and cursor have only the primary marker")
	combat.SpellTargeting.Cursor = clonePosition
	combat.SpellTargeting.RecipientSet = false
	cursorScreen.Clear()
	scene.drawCombatAttackCrosshair(cursorScreen)
	assertCombatMarkerPixels(t, cursorScreen, cursorOptions, selectedReticleRaster, ucolor.EGAWhite, widgets.DialogStatusWarningColor,
		make([]byte, cursorPixels*4))
	wakeScreen.Clear()
	scene.drawCombatPartySpellWakeMarker(wakeScreen)
	assert.Zero(t, opaquePixelCount(wakeScreen), "an invalid recipient has no secondary wake cue")
}

// TestCombatPartyTurnIndex_HidesPlayerSelectionDuringMonsterTurn prevents the
// previous PC retaining a green outline while an enemy is moving or attacking.
func TestCombatPartyTurnIndex_HidesPlayerSelectionDuringMonsterTurn(t *testing.T) {
	assert.Equal(t, game_state.NoTargetPC, combatPartyTurnIndex(2, 0, false))
	assert.Equal(t, game_state.NoTargetPC, combatPartyTurnIndex(game_state.NoActiveMonster, 0, true))
	assert.Equal(t, 0, combatPartyTurnIndex(game_state.NoActiveMonster, 0, false))
}
