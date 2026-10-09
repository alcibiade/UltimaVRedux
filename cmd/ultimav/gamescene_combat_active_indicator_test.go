package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
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

func TestCombatActiveSelectionFaceColorDistinguishesFaction(t *testing.T) {
	assert.Equal(t, ucolor.CombatActivePCOutline, combatActiveSelectionFaceColor(game_state.NoActiveMonster))
	assert.Equal(t, ucolor.CombatActiveMonsterOutline, combatActiveSelectionFaceColor(3))
}
