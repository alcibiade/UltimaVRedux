package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func setupCombatAutoActionTest(t *testing.T) (*GameState, int) {
	t.Helper()

	gs, _, activeIndex := setupCombatDoorActionTest(t)

	clearCombatMonstersForTest(gs.CombatScene)
	character := gs.PartyState.GetCharacter(activeIndex)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotWeapon, references.ShortSword)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	for index := range party_state.MaxActiveParty {
		gs.CombatScene.PartyActive[index] = index == activeIndex
	}

	theMap := gs.GetCurrentLayeredMap()
	for x := references.Coordinate(0); x < references.XCombatMapTiles; x++ {
		for y := references.Coordinate(0); y < references.YCombatMapTiles; y++ {
			position := references.Position{X: x, Y: y}
			theMap.SetTileByLayer(map_state.MapLayer, &position, indexes.StoneBrickWall)
		}
	}

	return gs, activeIndex
}

func setCombatAutoTiles(gs *GameState, tile indexes.SpriteIndex, positions ...references.Position) {
	for _, position := range positions {
		gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, tile)
	}
}

func addCombatAutoMonster(gs *GameState, position references.Position) *CombatMonster {
	enemyReferences := *gs.GameReferences.EnemyReferences
	monster := NewCombatMonster(
		references.MonsterOrc,
		&enemyReferences[references.MonsterOrc],
		position,
		gs.AsRandomProvider(),
	)
	gs.CombatScene.MonsterManager.AddMonster(monster)

	return monster
}

func TestPlanCombatAutoAction_AttacksAdjacentEnemy(t *testing.T) {
	gs, activeIndex := setupCombatAutoActionTest(t)

	start := references.Position{X: 2, Y: 2}
	target := references.Position{X: 3, Y: 3}
	gs.CombatScene.SetPartyMemberPosition(activeIndex, start)
	setCombatAutoTiles(gs, indexes.BrickFloor, start, target)
	addCombatAutoMonster(gs, target)

	action := gs.PlanCombatAutoAction()

	assert.Equal(t, CombatAutoActionAttack, action.Kind)
	assert.Equal(t, target, action.Target)
}

func TestPlanCombatAutoAction_RangedAttackSelectsClosestEnemy(t *testing.T) {
	gs, activeIndex := setupCombatAutoActionTest(t)

	start := references.Position{X: 1, Y: 1}
	farTarget := references.Position{X: 10, Y: 1}
	closeTarget := references.Position{X: 2, Y: 1}
	gs.CombatScene.SetPartyMemberPosition(activeIndex, start)
	gs.PartyState.GetCharacter(activeIndex).SetEquippedItem(references.SlotWeapon, references.MagicAxe)
	setCombatAutoTiles(gs, indexes.BrickFloor, start, farTarget, closeTarget)
	addCombatAutoMonster(gs, farTarget)
	addCombatAutoMonster(gs, closeTarget)

	action := gs.PlanCombatAutoAction()

	assert.Equal(t, CombatAutoActionAttack, action.Kind)
	assert.Equal(t, closeTarget, action.Target)
}

func TestPlanCombatAutoAction_RangedAttackBreaksDistanceTieByLowestHP(t *testing.T) {
	gs, activeIndex := setupCombatAutoActionTest(t)

	start := references.Position{X: 4, Y: 4}
	healthyTarget := references.Position{X: 1, Y: 4}
	woundedTarget := references.Position{X: 7, Y: 4}
	gs.CombatScene.SetPartyMemberPosition(activeIndex, start)
	gs.PartyState.GetCharacter(activeIndex).SetEquippedItem(references.SlotWeapon, references.MagicAxe)
	setCombatAutoTiles(gs, indexes.BrickFloor, start, healthyTarget, woundedTarget)
	addCombatAutoMonster(gs, healthyTarget)
	wounded := addCombatAutoMonster(gs, woundedTarget)
	wounded.SetCurrentHP(1)

	action := gs.PlanCombatAutoAction()

	assert.Equal(t, CombatAutoActionAttack, action.Kind)
	assert.Equal(t, woundedTarget, action.Target)
}

func TestPlanCombatAutoActionUsesActiveClonePosition(t *testing.T) {
	gs, activeIndex := setupCombatAutoActionTest(t)
	source := CombatTurnIdentity{IsPartyMember: true, EntityIndex: activeIndex}
	clonePosition := references.Position{X: 2, Y: 2}
	target := references.Position{X: 3, Y: 3}
	clone := gs.CombatScene.addPCCloneFromSource(
		source,
		activeIndex,
		clonePosition,
		NewCombatPCProvider(gs.PartyState.GetCharacter(activeIndex)),
	)
	gs.CombatScene.PartyActive[activeIndex] = false
	gs.CombatScene.TurnQueue.markFledIdentity(source)
	gs.CombatScene.CurrentTurnIdentity = clone
	gs.CombatScene.currentTurnIdentitySet = true
	setCombatAutoTiles(gs, indexes.BrickFloor, clonePosition, target)
	addCombatAutoMonster(gs, target)

	action := gs.PlanCombatAutoAction()

	assert.Equal(t, CombatAutoActionAttack, action.Kind)
	assert.Equal(t, target, action.Target)
}

func TestPlanCombatAutoAction_RoutesAroundLockedDoor(t *testing.T) {
	gs, activeIndex := setupCombatAutoActionTest(t)

	start := references.Position{X: 1, Y: 1}
	target := references.Position{X: 5, Y: 1}
	gs.CombatScene.SetPartyMemberPosition(activeIndex, start)
	setCombatAutoTiles(gs, indexes.BrickFloor,
		start,
		references.Position{X: 1, Y: 2},
		references.Position{X: 2, Y: 2},
		references.Position{X: 3, Y: 2},
		references.Position{X: 4, Y: 2},
		target,
	)
	setCombatAutoTiles(gs, indexes.LockedDoor, references.Position{X: 2, Y: 1})
	addCombatAutoMonster(gs, target)

	action := gs.PlanCombatAutoAction()

	assert.Equal(t, CombatAutoActionMove, action.Kind)
	assert.Equal(t, references.Down, action.Direction,
		"planner must take the complete route rather than approaching the locked door")
}

func TestPlanCombatAutoAction_MovementBreaksDistanceTieByLowestHP(t *testing.T) {
	gs, activeIndex := setupCombatAutoActionTest(t)

	start := references.Position{X: 4, Y: 4}
	healthyTarget := references.Position{X: 1, Y: 4}
	woundedTarget := references.Position{X: 7, Y: 4}
	gs.CombatScene.SetPartyMemberPosition(activeIndex, start)
	setCombatAutoTiles(gs, indexes.BrickFloor,
		start,
		references.Position{X: 3, Y: 4},
		references.Position{X: 2, Y: 4},
		healthyTarget,
		references.Position{X: 5, Y: 4},
		references.Position{X: 6, Y: 4},
		woundedTarget,
	)
	addCombatAutoMonster(gs, healthyTarget)
	wounded := addCombatAutoMonster(gs, woundedTarget)
	wounded.SetCurrentHP(1)

	action := gs.PlanCombatAutoAction()

	assert.Equal(t, CombatAutoActionMove, action.Kind)
	assert.Equal(t, references.Right, action.Direction)
	assert.Equal(t, woundedTarget, action.Target)
}

func TestPlanCombatAutoAction_RequiresClosedDoorToBeOpenedManually(t *testing.T) {
	gs, activeIndex := setupCombatAutoActionTest(t)

	start := references.Position{X: 1, Y: 1}
	door := references.Position{X: 2, Y: 1}
	targetApproach := references.Position{X: 3, Y: 1}
	target := references.Position{X: 4, Y: 1}
	gs.CombatScene.SetPartyMemberPosition(activeIndex, start)
	setCombatAutoTiles(gs, indexes.BrickFloor, start, targetApproach, target)
	setCombatAutoTiles(gs, indexes.RegularDoor, door)
	addCombatAutoMonster(gs, target)

	assert.Equal(t, CombatAutoActionUnavailable, gs.PlanCombatAutoAction().Kind)
	require.True(t, gs.ActionOpenCombatMap(references.Right).Handled)

	action := gs.PlanCombatAutoAction()
	assert.Equal(t, CombatAutoActionMove, action.Kind)
	assert.Equal(t, references.Right, action.Direction)
}

func TestPlanCombatAutoAction_SelectsReachableEnemyOverCloserBlockedEnemy(t *testing.T) {
	gs, activeIndex := setupCombatAutoActionTest(t)

	start := references.Position{X: 1, Y: 1}
	blockedTarget := references.Position{X: 3, Y: 1}
	reachableTarget := references.Position{X: 1, Y: 6}
	gs.CombatScene.SetPartyMemberPosition(activeIndex, start)
	setCombatAutoTiles(gs, indexes.BrickFloor,
		start,
		blockedTarget,
		references.Position{X: 1, Y: 2},
		references.Position{X: 1, Y: 3},
		references.Position{X: 1, Y: 4},
		references.Position{X: 1, Y: 5},
		reachableTarget,
	)
	addCombatAutoMonster(gs, blockedTarget)
	addCombatAutoMonster(gs, reachableTarget)

	action := gs.PlanCombatAutoAction()

	assert.Equal(t, CombatAutoActionMove, action.Kind)
	assert.Equal(t, references.Down, action.Direction)
	assert.Equal(t, reachableTarget, action.Target)
}

func TestPlanCombatAutoAction_DoesNotMoveWithoutCompleteRoute(t *testing.T) {
	gs, activeIndex := setupCombatAutoActionTest(t)

	start := references.Position{X: 1, Y: 1}
	target := references.Position{X: 4, Y: 1}
	gs.CombatScene.SetPartyMemberPosition(activeIndex, start)
	setCombatAutoTiles(gs, indexes.BrickFloor, start, target)
	addCombatAutoMonster(gs, target)

	action := gs.PlanCombatAutoAction()

	assert.Equal(t, CombatAutoActionUnavailable, action.Kind)
	assert.Equal(t, start, gs.CombatScene.GetPartyMemberPosition(activeIndex))
}

func TestPlanCombatAutoAction_TreatsOtherPartyMembersAsBlockers(t *testing.T) {
	gs, activeIndex := setupCombatAutoActionTest(t)

	start := references.Position{X: 1, Y: 1}
	blocker := references.Position{X: 2, Y: 1}
	targetApproach := references.Position{X: 3, Y: 1}
	target := references.Position{X: 4, Y: 1}
	gs.CombatScene.SetPartyMemberPosition(activeIndex, start)
	blockerIndex := (activeIndex + 1) % party_state.MaxActiveParty
	gs.CombatScene.PartyActive[blockerIndex] = true
	gs.CombatScene.SetPartyMemberPosition(blockerIndex, blocker)
	setCombatAutoTiles(gs, indexes.BrickFloor, start, blocker, targetApproach, target)
	addCombatAutoMonster(gs, target)

	assert.Equal(t, CombatAutoActionUnavailable, gs.PlanCombatAutoAction().Kind)
}

func TestCombatAttack_RejectsInactiveCombatant(t *testing.T) {
	gs, activeIndex := setupCombatAutoActionTest(t)

	start := references.Position{X: 1, Y: 1}
	target := references.Position{X: 2, Y: 1}
	gs.CombatScene.SetPartyMemberPosition(activeIndex, start)
	setCombatAutoTiles(gs, indexes.BrickFloor, start, target)
	monster := addCombatAutoMonster(gs, target)
	hpBefore := monster.CurrentHP()
	gs.CombatScene.PartyActive[activeIndex] = false

	assert.False(t, attackCombatSquare(t, gs, target))
	assert.Equal(t, hpBefore, monster.CurrentHP())
}

func TestPlanCombatAutoAction_IgnoresInvisibleEnemy(t *testing.T) {
	gs, activeIndex := setupCombatAutoActionTest(t)

	start := references.Position{X: 1, Y: 1}
	target := references.Position{X: 2, Y: 1}
	gs.CombatScene.SetPartyMemberPosition(activeIndex, start)
	setCombatAutoTiles(gs, indexes.BrickFloor, start, target)
	monster := addCombatAutoMonster(gs, target)
	monster.SetInvisible(true)

	assert.Equal(t, CombatAutoActionUnavailable, gs.PlanCombatAutoAction().Kind)
	gs.CombatScene.PreCombatPlayerLocation.Location = references.Doom
	assert.Equal(t, CombatAutoActionUnavailable, gs.PlanCombatAutoAction().Kind)
	gs.CombatScene.DungeonRoomOrigin = &DungeonRoomEncounterOrigin{Location: references.Doom}
	assert.Equal(t, CombatAutoActionAttack, gs.PlanCombatAutoAction().Kind)
}

func TestPlanCombatAutoAction_IsNonMutating(t *testing.T) {
	gs, activeIndex := setupCombatAutoActionTest(t)

	start := references.Position{X: 1, Y: 1}
	target := references.Position{X: 4, Y: 1}
	gs.CombatScene.SetPartyMemberPosition(activeIndex, start)
	setCombatAutoTiles(gs, indexes.BrickFloor,
		start,
		references.Position{X: 2, Y: 1},
		references.Position{X: 3, Y: 1},
		target,
	)
	monster := addCombatAutoMonster(gs, target)
	hpBefore := monster.CurrentHP()

	require.Equal(t, CombatAutoActionMove, gs.PlanCombatAutoAction().Kind)
	assert.Equal(t, start, gs.CombatScene.GetPartyMemberPosition(activeIndex))
	assert.Equal(t, hpBefore, monster.CurrentHP())
}
