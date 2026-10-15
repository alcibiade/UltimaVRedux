package main

import (
	"fmt"
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestCombatActivePCMarchingPhaseUsesStableCadence(t *testing.T) {
	assert.Equal(t, 0, combatActivePCMarchingPhase(0))
	assert.Equal(t, 0, combatActivePCMarchingPhase(combatActivePCMarchingIntervalMs-1))
	assert.Equal(t, 1, combatActivePCMarchingPhase(combatActivePCMarchingIntervalMs))
	assert.Equal(t, 1, combatActivePCMarchingPhase(2*combatActivePCMarchingIntervalMs-1))
	assert.Equal(t, 0, combatActivePCMarchingPhase(2*combatActivePCMarchingIntervalMs))
}

func TestCombatActivePCArrowPhaseUsesStableCadence(t *testing.T) {
	assert.Equal(t, 0, combatActivePCArrowPhase(0))
	assert.Equal(t, 0, combatActivePCArrowPhase(combatActivePCArrowIntervalMs-1))
	assert.Equal(t, 1, combatActivePCArrowPhase(combatActivePCArrowIntervalMs))
	assert.Equal(t, 0, combatActivePCArrowPhase(2*combatActivePCArrowIntervalMs))
}

func TestCombatActivePCArrowsMoveInwardWithoutReachingHealthBar(t *testing.T) {
	outward := combatActivePCArrowSourceGeometry(0)
	inward := combatActivePCArrowSourceGeometry(1)

	assert.Greater(t, inward.LeftTipX, outward.LeftTipX)
	assert.Less(t, inward.RightTipX, outward.RightTipX)
	assert.GreaterOrEqual(t, outward.LeftBaseX, float64(combatActivePCArrowLeftBaseX))
	assert.LessOrEqual(t, outward.RightBaseX, float64(combatActivePCArrowRightBaseX))
	assert.Less(t, inward.BottomY, float64(combatHealthBarTopSourcePixel))
}

func TestCombatActivePCSelectionPhaseUsesStableCadence(t *testing.T) {
	assert.Equal(t, 0, combatActivePCSelectionPhase(0))
	assert.Equal(t, 0, combatActivePCSelectionPhase(combatActivePCSelectionIntervalMs-1))
	assert.Equal(t, 1, combatActivePCSelectionPhase(combatActivePCSelectionIntervalMs))
	assert.Equal(t, 0, combatActivePCSelectionPhase(2*combatActivePCSelectionIntervalMs))
}

func TestCombatActivePCSelectionTileExpandsOnePixelEachDirection(t *testing.T) {
	contracted := combatActivePCSelectionSourceGeometry(0)
	expanded := combatActivePCSelectionSourceGeometry(1)
	halfShadow := combatActivePCSelectionShadow / 2

	assert.Equal(t, contracted.Left-1, expanded.Left)
	assert.Equal(t, contracted.Top-1, expanded.Top)
	assert.Equal(t, contracted.Width+2, expanded.Width)
	assert.Equal(t, contracted.Height+2, expanded.Height)
	assert.Equal(t, 0.0, contracted.Left-halfShadow)
	assert.Equal(t, 18.0, contracted.Left+contracted.Width+halfShadow)
}

func TestCombatTargetSelectionUsesSteadyExpandedFrame(t *testing.T) {
	phase := combatActivePCSelectionPhase(combatTargetSelectionElapsedMs)
	geometry := combatActivePCSelectionSourceGeometry(phase)

	assert.Equal(t, 1, phase)
	assert.Equal(t, combatActivePCSelectionGeometry{
		Left: 0, Top: 0, Width: 18, Height: 18,
	}, geometry)
}

func TestActiveCombatPCIndicatorDefaultsToSelectionTile(t *testing.T) {
	assert.Equal(t, combatActivePCIndicatorSelectionTile, activeCombatPCIndicatorStyle)
}

func TestResolveCombatSelectionActorsMapsTurnAndTargetRoles(t *testing.T) {
	playerTurn := resolveCombatSelectionActors(game_state.NoActiveMonster, 2, game_state.NoTargetPC)
	assert.Equal(t, 2, playerTurn.ActivePartyIndex)
	assert.Equal(t, game_state.NoActiveMonster, playerTurn.ActiveMonsterIndex)
	assert.Equal(t, game_state.NoTargetPC, playerTurn.TargetPartyIndex)

	monsterMove := resolveCombatSelectionActors(3, 2, game_state.NoTargetPC)
	assert.Equal(t, game_state.NoTargetPC, monsterMove.ActivePartyIndex)
	assert.Equal(t, 3, monsterMove.ActiveMonsterIndex)
	assert.Equal(t, game_state.NoTargetPC, monsterMove.TargetPartyIndex)

	monsterAttack := resolveCombatSelectionActors(3, 2, 1)
	assert.Equal(t, game_state.NoTargetPC, monsterAttack.ActivePartyIndex)
	assert.Equal(t, 3, monsterAttack.ActiveMonsterIndex)
	assert.Equal(t, 1, monsterAttack.TargetPartyIndex)
}

func TestCombatActiveSelectionFaceColorDistinguishesTurnRole(t *testing.T) {
	assert.Equal(t, ucolor.CombatActivePCOutline, combatActiveSelectionFaceColor(game_state.NoActiveMonster))
	assert.Equal(t, ucolor.CombatActiveMonsterOutline, combatActiveSelectionFaceColor(3))
}

// Literal bounded geometry contracts independent of the production rectangles.
// The expanded stencil starts at source (-1,-1); the contracted one at (0,0).
const activeCornerRaster = `BBBBBB......BBBBBB
BWWWWB......BWWWWB
BWWWWB......BWWWWB
BWWBBB......BBBWWB
BWWB..........BWWB
BBBB..........BBBB
..................
..................
..................
..................
..................
..................
..................
..................
..................
..................
..................
..................`

const expandedActiveCornerRaster = `BBBBBB........BBBBBB
BWWWWB........BWWWWB
BWWWWB........BWWWWB
BWWBBB........BBBWWB
BWWB............BWWB
BBBB............BBBB
....................
....................
....................
....................
....................
....................
....................
....................
....................
....................
....................
....................
....................
....................`

func assertActiveCornerPixels(t *testing.T, screen *ebiten.Image, options *ebiten.DrawImageOptions, elapsed int64, face ucolor.EGAColor, before []byte) {
	t.Helper()
	if before == nil {
		screen = screen.SubImage(combatFixtureRasterBounds(options)).(*ebiten.Image)
	}
	stencil := activeCornerRaster
	expectedOptions := &ebiten.DrawImageOptions{}
	// Pin the documented 450 ms boundary independently of the production phase
	// helper, including the one-source-pixel expansion on both axes.
	if elapsed%900 >= 450 {
		stencil = expandedActiveCornerRaster
		expectedOptions.GeoM.Translate(-1, -1)
	}
	expectedOptions.GeoM.Concat(options.GeoM)
	assertCombatMarkerPixels(t, screen, expectedOptions, stencil, face, face, before)
}

func TestCombatActiveSelectionTwoUpperCornersFullBackingAndPulseGeometry(t *testing.T) {
	for _, scale := range []struct{ x, y float64 }{{1, 1}, {2, 3}, {1366 * 0.76 / 304, 768.0 / 208}} {
		for _, elapsed := range []int64{0, 449, 450, 899, 900} {
			t.Run(fmt.Sprintf("%gx%g/%d", scale.x, scale.y, elapsed), func(t *testing.T) {
				options := &ebiten.DrawImageOptions{}
				options.GeoM.Scale(scale.x, scale.y)
				options.GeoM.Translate(5.35, 5.65)
				bounds := image.Rect(0, 0, 80, 80)
				if elapsed == 899 {
					bounds = image.Rect(9, 9, 70, 30)
				}
				screen := ebiten.NewImageWithOptions(bounds, &ebiten.NewImageOptions{Unmanaged: true})
				defer screen.Dispose()
				screen.Fill(ucolor.EGALightCyan)
				before := signaturePixels(screen)
				drawCombatActiveSelectionCorners(screen, options, elapsed, ucolor.CombatActivePCOutline)
				assertActiveCornerPixels(t, screen, options, elapsed, ucolor.CombatActivePCOutline, before)
			})
		}
	}
}

func TestCombatActiveSelectionProductionCompositionAndAimPriority(t *testing.T) {
	scene, party := newCombatMarkerTestScene(t)
	state, combat := scene.gameState, scene.gameState.CombatScene
	position := combat.GetCombatantPosition(party)
	index, monster := signatureMonster(t, scene, references.MonsterOrc, references.Position{X: 6, Y: 5})
	monster.SetLoyalty(game_state.CombatMonsterLoyaltyParty)
	for _, isMonster := range []bool{false, true} {
		face := ucolor.CombatActivePCOutline
		options := scene.combatPartyIdentityDrawOptions(position, party)
		cursor := position
		if isMonster {
			combat.ActiveMonsterIndex = index
			combat.CurrentTurnIdentity = game_state.CombatTurnIdentity{EntityIndex: index}
			face, cursor = ucolor.CombatActiveMonsterOutline, monster.Position()
			options = scene.combatMonsterDrawOptions(cursor, index)
		}
		for _, elapsed := range []int64{0, 450} {
			state.ElapsedMs = elapsed
			sample := combatFixtureRasterBounds(options)
			screen := ebiten.NewImageWithOptions(sample, &ebiten.NewImageOptions{Unmanaged: true})
			scene.drawAvatarOverlay(screen)
			assertActiveCornerPixels(t, screen, options, elapsed, face, nil)
			combat.AttackTargeting = &game_state.CombatAttackTargeting{Cursor: cursor}
			screen.Clear()
			scene.drawAvatarOverlay(screen)
			assertCombatMarkerPixels(t, screen, scene.combatSquareMarkerDrawOptions(cursor), selectedReticleRaster,
				ucolor.EGAWhite, getCombatAimColor(game_state.CombatAimFriendly, combatAimPulse(elapsed)), nil)
			combat.AttackTargeting = nil
			screen.Dispose()
		}
	}
}

func TestCombatActiveSelectionExactCloneTransformAndRoles(t *testing.T) {
	scene, source := newCombatMarkerTestScene(t)
	state, combat := scene.gameState, scene.gameState.CombatScene
	combat.PartyPossessed[source.EntityIndex] = true
	require.True(t, state.CastCombatInQuasXen(source.EntityIndex, source))
	var clone game_state.CombatTurnIdentity
	combat.ForEachPCClone(func(identity game_state.CombatTurnIdentity, _ references.Position) { clone = identity })
	require.Positive(t, clone.Slot)
	combat.PartyPossessed[source.EntityIndex] = false
	position := combat.GetCombatantPosition(clone)
	setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
	combat.SetPartyMemberPosition(source.EntityIndex, position) // Same roster and same square, different exact entries.
	scene.screenShakeOffsetX, scene.screenShakeOffsetY = 0.35, -0.65
	scene.combatMovement = CombatMovementAnimation{Active: true, IsPartyMember: true, EntityIndex: clone.EntityIndex, Slot: clone.Slot,
		From: references.Position{X: position.X - 1, Y: position.Y}, To: position, Progress: 0.5}
	combat.CurrentTurnIdentity = clone
	options, face := scene.combatActiveSelectionOptions()
	require.NotNil(t, options)
	assert.Equal(t, ucolor.CombatActiveMonsterOutline, face, "possessed clone uses the monster turn role")
	base := scene.combatSquareMarkerDrawOptions(position)
	x, y := options.GeoM.Apply(0, 0)
	wantX, wantY := base.GeoM.Apply(-8, 0) // Half of one source-cell move, plus the shared shake/map transform.
	assert.InDelta(t, wantX, x, 1e-6)
	assert.InDelta(t, wantY, y, 1e-6)
	combat.CurrentTurnIdentity = source
	options, face = scene.combatActiveSelectionOptions()
	require.NotNil(t, options)
	assert.Equal(t, ucolor.CombatActivePCOutline, face)
	x, y = options.GeoM.Apply(0, 0)
	wantX, wantY = base.GeoM.Apply(0, 0)
	assert.InDelta(t, wantX, x, 1e-6, "clone motion cannot move a co-located original's marker")
	assert.InDelta(t, wantY, y, 1e-6)
	combat.SetSoloMoveCharacter((source.EntityIndex + 1) % state.PartyState.GetCharacterCount())
	_, face = scene.combatActiveSelectionOptions()
	assert.Equal(t, ucolor.EGALightGray, face)
	combat.ClearSoloMoveMode()
	scene.combatTeleportEffect = &combatTeleportSpellEffect{Caster: clone, Phase: combatTeleportHidden}
	options, _ = scene.combatActiveSelectionOptions()
	require.NotNil(t, options, "another exact entry's hidden phase cannot suppress the original")
	combat.CurrentTurnIdentity = clone
	options, _ = scene.combatActiveSelectionOptions()
	assert.Nil(t, options)
	scene.combatTeleportEffect = nil
	combat.CurrentTurnIdentity = source
	combat.PartyUnderground[source.EntityIndex] = true
	options, _ = scene.combatActiveSelectionOptions()
	assert.Nil(t, options)
	combat.CurrentTurnIdentity = clone
	options, _ = scene.combatActiveSelectionOptions()
	assert.NotNil(t, options, "the clone does not inherit later source captivity")
	setCombatConcealmentTerrain(state, position, indexes.BlackSquare)
	options, _ = scene.combatActiveSelectionOptions()
	assert.Nil(t, options)
	setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
	state.PartyState.GetCharacter(source.EntityIndex).SetHp(0)
	state.PartyState.GetCharacter(source.EntityIndex).MarkAsDead()
	options, _ = scene.combatActiveSelectionOptions()
	assert.Nil(t, options, "shared death retires both clone and original markers")
	state.CombatScene = nil
	options, _ = scene.combatActiveSelectionOptions()
	assert.Nil(t, options, "combat exit has no retained foreground state")
}

func TestCombatActiveSelectionMonsterVisibilityAndStackRepresentative(t *testing.T) {
	scene, _ := newCombatMarkerTestScene(t)
	state, combat := scene.gameState, scene.gameState.CombatScene
	position := references.Position{X: 7, Y: 5}
	index, monster := signatureMonster(t, scene, references.MonsterCorpser, position)
	combat.ActiveMonsterIndex = index
	options, _ := scene.combatActiveSelectionOptions()
	assert.Nil(t, options, "hidden Corpser has no foreground identity cue")
	monster.SetCorpserExposed(true)
	options, _ = scene.combatActiveSelectionOptions()
	require.NotNil(t, options)
	monster.SetInvisible(true)
	options, _ = scene.combatActiveSelectionOptions()
	assert.Nil(t, options)
	monster.SetInvisible(false)
	setCombatConcealmentTerrain(state, position, indexes.BrickWallArchway)
	options, _ = scene.combatActiveSelectionOptions()
	assert.Nil(t, options)
	setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
	scene.combatSummonEffect = &combatSummonSpellEffect{Phase: combatSummonSeal,
		Result: game_state.CombatSummonResult{Created: []game_state.CombatSummonedMonster{{Index: index}}}}
	options, _ = scene.combatActiveSelectionOptions()
	assert.Nil(t, options, "not-yet-arrived actors stay hidden")
	scene.combatSummonEffect = nil
	monster.SetCurrentHP(0)
	options, _ = scene.combatActiveSelectionOptions()
	assert.Nil(t, options)
	combat.MonsterManager.Clear()
	combat.TurnQueue.RemoveAllMonsters()
	first, insect := signatureMonster(t, scene, references.MonsterInsect, position)
	insect.SetLoyalty(game_state.CombatMonsterLoyaltyParty)
	second, insect := signatureMonster(t, scene, references.MonsterInsect, position)
	insect.SetLoyalty(game_state.CombatMonsterLoyaltyParty)
	combat.ActiveMonsterIndex = second
	scene.combatMovement = CombatMovementAnimation{Active: true, EntityIndex: first,
		From: references.Position{X: position.X - 1, Y: position.Y}, To: position, Progress: 0.5}
	options, face := scene.combatActiveSelectionOptions()
	require.NotNil(t, options)
	assert.Equal(t, ucolor.CombatActiveMonsterOutline, face, "allied insects retain the monster active role")
	x, y := options.GeoM.Apply(0, 0)
	wantX, wantY := scene.combatSquareMarkerDrawOptions(position).GeoM.Apply(-8, 0)
	assert.InDelta(t, wantX, x, 1e-6, "a non-representative member marks the actually drawn aggregate")
	assert.InDelta(t, wantY, y, 1e-6)
	insect.SetInvisible(true)
	options, _ = scene.combatActiveSelectionOptions()
	assert.Nil(t, options, "a hidden stack member cannot acquire the representative's disclosure")
}

func TestCombatThreatenedTargetKeepsSteadyUnderlayWithoutRaisedCorners(t *testing.T) {
	scene, actor := newCombatMarkerTestScene(t)
	combat := scene.gameState.CombatScene
	target := game_state.CombatTurnIdentity{IsPartyMember: true, EntityIndex: (actor.EntityIndex + 1) % scene.gameState.PartyState.GetCharacterCount()}
	require.True(t, combat.IsCombatPCActive(target))
	position := references.Position{X: 2, Y: 2}
	combat.SetPartyMemberPosition(target.EntityIndex, position)
	setCombatConcealmentTerrain(scene.gameState, position, indexes.BrickFloor)
	combat.ActiveTargetIdentity, combat.ActiveTargetIdentitySet = target, true
	options := scene.combatPartyIdentityDrawOptions(position, target)
	screen := ebiten.NewImageWithOptions(combatFixtureRasterBounds(options), &ebiten.NewImageOptions{Unmanaged: true})
	defer screen.Dispose()
	scene.drawCombatActiveSelectionAccents(screen)
	assert.Zero(t, opaquePixelCount(screen), "the threatened actor never receives the active actor's raised accents")
	scene.gameState.ElapsedMs = 0
	scene.drawCombatSelectionUnderlays(screen)
	require.Positive(t, countExactColor(screen, ucolor.CombatThreatenedPCOutline))
	before := signaturePixels(screen)
	screen.Clear()
	scene.gameState.ElapsedMs = 450
	scene.drawCombatSelectionUnderlays(screen)
	assert.Equal(t, before, signaturePixels(screen), "only active turn ownership pulses; threatened intent remains steady")
}
