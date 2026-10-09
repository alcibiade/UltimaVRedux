package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// This fixture loads only a synthetic CBT, not licensed original game data.
func newCombatAttackPreviewTest(t *testing.T, weapon references.Equipment) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	gs, mock := newSyntheticDungeonTriggerCombatState(t)
	character := gs.PartyState.GetCharacter(0)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotWeapon, weapon)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	gs.PartyState.Inventory.Equipment.Set(references.Arrows, 10)
	gs.CombatScene.CurrentTurnIdentity = CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
	gs.CombatScene.currentTurnIdentitySet = true
	setOpenCombatMissileMap(gs)

	return gs, mock
}

func TestCombatAttackPreviewSwarmSelectsClosestClearShotWithoutWalkingRoute(t *testing.T) {
	gs, _ := newCombatAttackPreviewTest(t, references.Bow)
	blocked, _ := addSyntheticTriggerMonster(gs, references.Position{X: 5, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	clear, _ := addSyntheticTriggerMonster(gs, references.Position{X: 2, Y: 7}, CombatMonsterLoyaltyMonster, references.MissileNone)
	closest, _ := addSyntheticTriggerMonster(gs, references.Position{X: 2, Y: 5}, CombatMonsterLoyaltyMonster, references.MissileNone)
	setBlockedCombatMissileMap(gs, references.Position{X: 3, Y: 2})
	for x := range references.XCombatMapTiles {
		for y := range references.YCombatMapTiles {
			position := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
			gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.StoneBrickWall)
		}
	}

	preview := gs.PreviewCombatAttack()
	require.Len(t, preview.Targets, 2)
	assert.Equal(t, clear.Position(), preview.Targets[0].Position, "stable monster order, not nearest-first")
	assert.NotEqual(t, blocked.Position(), preview.Targets[0].Position)
	action := gs.PlanCombatAutoAction()
	assert.Equal(t, CombatAutoActionAttack, action.Kind)
	assert.Equal(t, closest.Position(), action.Target)
	assert.True(t, gs.BeginCombatAutoAttack(action.Target))
	result := gs.ContinueCombatAutoAttack()
	require.NotNil(t, result.Launch)
	assert.False(t, result.Launch.Blocked, "intermediate actors do not add interception rules")
}

func TestCombatAttackPreviewRangeGeometryAndBounds(t *testing.T) {
	tests := []struct {
		name     string
		weapon   references.Equipment
		position references.Position
		status   CombatShotStatus
	}{
		{"melee diagonal", references.ShortSword, references.Position{X: 3, Y: 3}, CombatShotClear},
		{"melee too far", references.ShortSword, references.Position{X: 4, Y: 2}, CombatShotOutOfRange},
		{"bow diagonal integer edge", references.Bow, references.Position{X: 9, Y: 5}, CombatShotClear},
		{"bow beyond edge", references.Bow, references.Position{X: 10, Y: 2}, CombatShotOutOfRange},
		{"negative coordinate", references.Bow, references.Position{X: -1, Y: 2}, CombatShotOutOfRange},
		{"past map edge", references.Bow, references.Position{X: 11, Y: 2}, CombatShotOutOfRange},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, _ := newCombatAttackPreviewTest(t, test.weapon)
			require.True(t, gs.BeginCombatAttack())
			gs.CombatScene.AttackTargeting.Cursor = test.position
			preview := gs.PreviewCombatAttack()
			require.NotNil(t, preview.Aim)
			assert.Equal(t, test.status, preview.Aim.Status)
		})
	}
}

func TestCombatAttackPreviewUsesOriginalFinalSampleAndTerrainBypass(t *testing.T) {
	for _, weapon := range []references.Equipment{references.Bow, references.MorningStar, references.Halberd} {
		t.Run(weapon.String(), func(t *testing.T) {
			gs, _ := newCombatAttackPreviewTest(t, weapon)
			target, _ := addSyntheticTriggerMonster(gs, references.Position{X: 4, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
			setBlockedCombatMissileMap(gs, references.Position{X: 3, Y: 2})
			require.True(t, gs.BeginCombatAttack())
			gs.CombatScene.AttackTargeting.Cursor = target.Position()
			want := CombatShotTerrainBypass
			if weapon == references.Bow {
				want = CombatShotBlocked
			}
			assert.Equal(t, want, gs.PreviewCombatAttack().Aim.Status)
		})
	}
	gs, _ := newCombatAttackPreviewTest(t, references.Bow)
	target, _ := addSyntheticTriggerMonster(gs, references.Position{X: 3, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	setBlockedCombatMissileMap(gs, target.Position())
	assert.Len(t, gs.PreviewCombatAttack().Targets, 1, "final displayed projectile sample is not collision tested")
}

func TestCombatAttackPreviewAvailabilityAndCurrentSource(t *testing.T) {
	gs, _ := newCombatAttackPreviewTest(t, references.Bow)
	character := gs.PartyState.GetCharacter(0)
	character.SetEquippedItem(references.SlotHelm, references.SpikedHelm)
	target, _ := addSyntheticTriggerMonster(gs, references.Position{X: 5, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	assert.Len(t, gs.PreviewCombatAttack().Targets, 1, "normal preview considers every source")
	require.True(t, gs.BeginCombatAttack())
	gs.CombatScene.AttackTargeting.Cursor = target.Position()
	preview := gs.PreviewCombatAttack()
	require.Len(t, preview.Sources, 1)
	assert.Equal(t, references.SpikedHelm, preview.Sources[0].Equipment)
	assert.Empty(t, preview.Targets)
	assert.Equal(t, CombatShotOutOfRange, preview.Aim.Status)
	gs.CancelCombatAttackSource()
	gs.CombatScene.AttackTargeting.Cursor = target.Position()
	assert.Len(t, gs.PreviewCombatAttack().Targets, 1)
	gs.PartyState.Inventory.Equipment.Set(references.Arrows, 0)
	assert.Equal(t, CombatShotNoAmmo, gs.PreviewCombatAttack().Aim.Status)
	assert.Empty(t, gs.PreviewCombatAttack().Targets)
	gs.CancelCombatAttackSource()
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	preview = gs.PreviewCombatAttack()
	require.Len(t, preview.Sources, 1)
	assert.Equal(t, references.Bow, preview.Sources[0].Equipment, "no fists substituted for an unavailable bow")
	assert.NotEqual(t, CombatAutoActionAttack, gs.PlanCombatAutoAction().Kind)
}

func TestCombatAttackPreviewInterferenceUsesRememberedExactActor(t *testing.T) {
	gs, _ := newCombatAttackPreviewTest(t, references.Bow)
	target, index := addSyntheticTriggerMonster(gs, references.Position{X: 3, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	assert.Len(t, gs.PreviewCombatAttack().Targets, 1, "adjacency alone is not interference")
	actor := gs.CombatScene.GetCurrentTurnIdentity()
	gs.rememberMeleeEngager(actor, CombatTurnIdentity{EntityIndex: index})
	assert.Empty(t, gs.PreviewCombatAttack().Targets)
	gs.PartyState.GetCharacter(0).SetEquippedItem(references.SlotShield, references.MainGauche)
	assert.Len(t, gs.PreviewCombatAttack().Targets, 1, "later melee source remains available")
	gs.PartyState.GetCharacter(0).SetEquippedItem(references.SlotShield, references.NoEquipment)
	target.SetPosition(references.Position{X: 4, Y: 2})
	assert.Len(t, gs.PreviewCombatAttack().Targets, 1, "stale engagement is ignored")
	target.SetPosition(references.Position{X: 3, Y: 2})
	gs.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectTimeStop, TurnsRemaining: 2}
	assert.Len(t, gs.PreviewCombatAttack().Targets, 1)
}

func TestCombatAttackPreviewIsDetachedAndPureDuringAiming(t *testing.T) {
	gs, mock := newCombatAttackPreviewTest(t, references.Bow)
	target, _ := addSyntheticTriggerMonster(gs, references.Position{X: 5, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	rngCalls := gs.GetRNG().Calls()
	inventory := gs.PartyState.Inventory.LockerSnapshot()
	lastTargets := gs.CombatScene.LastAttackTargets
	identity := gs.CombatScene.GetCurrentTurnIdentity()
	hp := target.CurrentHP()
	for range 5 {
		assert.Equal(t, CombatAutoActionAttack, gs.PlanCombatAutoAction().Kind)
		preview := gs.PreviewCombatAttack()
		preview.Sources[0].MaxRange = 99
		preview.Targets[0].Name = "changed detached copy"
	}
	require.True(t, gs.BeginCombatAttack())
	gs.CombatScene.AttackTargeting.Cursor = target.Position()
	targeting := *gs.CombatScene.AttackTargeting
	for range 5 {
		preview := gs.PreviewCombatAttack()
		assert.Equal(t, "Orc", preview.Aim.Name)
		preview.Aim.Position = references.Position{}
		preview.Sources[0].Equipment = references.NoEquipment
	}
	assert.Equal(t, targeting, *gs.CombatScene.AttackTargeting)
	assert.Equal(t, rngCalls, gs.GetRNG().Calls())
	assert.Equal(t, inventory, gs.PartyState.Inventory.LockerSnapshot())
	assert.Equal(t, lastTargets, gs.CombatScene.LastAttackTargets)
	assert.Equal(t, identity, gs.CombatScene.GetCurrentTurnIdentity())
	assert.Equal(t, hp, target.CurrentHP())
	assert.Empty(t, mock.Messages)
	assert.Empty(t, mock.CombatEvents)
}

func TestCombatAttackPreviewVisibilityAndEnemyTurnSuppression(t *testing.T) {
	gs, _ := newCombatAttackPreviewTest(t, references.Bow)
	target, index := addSyntheticTriggerMonster(gs, references.Position{X: 5, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	require.True(t, gs.BeginCombatAttack())
	gs.CombatScene.AttackTargeting.Cursor = target.Position()
	target.SetInvisible(true)
	assert.Empty(t, gs.PreviewCombatAttack().Targets)
	assert.Empty(t, gs.PreviewCombatAttack().Aim.Name)
	gs.CombatScene.DungeonRoomOrigin = &DungeonRoomEncounterOrigin{Location: references.Doom}
	assert.Len(t, gs.PreviewCombatAttack().Targets, 1, "original dungeon assistance legality is unchanged")
	assert.Nil(t, gs.PreviewCombatAttack().Targets[0].Actor, "legality does not disclose invisible artwork")
	assert.Empty(t, gs.PreviewCombatAttack().Aim.Name)
	target.SetLoyalty(CombatMonsterLoyaltyParty)
	assert.Empty(t, gs.PreviewCombatAttack().Targets)
	assert.Nil(t, gs.PreviewCombatAttack().Aim.Actor)
	target.SetInvisible(false)
	assert.Equal(t, CombatShotFriendly, gs.PreviewCombatAttack().Aim.Status)
	gs.CancelInitialCombatAttack()
	gs.CombatScene.CurrentTurnIdentity = CombatTurnIdentity{EntityIndex: index}
	assert.Empty(t, gs.PreviewCombatAttack().Sources)
	assert.Equal(t, CombatAutoActionUnavailable, gs.PlanCombatAutoAction().Kind)
	gs.CombatScene = nil
	assert.Empty(t, gs.PreviewCombatAttack())
}

func TestCombatAutoAttackRangedCommitImpactAndLastAmmo(t *testing.T) {
	gs, mock := newCombatAttackPreviewTest(t, references.Bow)
	target, _ := addSyntheticTriggerMonster(gs, references.Position{X: 5, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	gs.PartyState.GetCharacter(0).SetEquippedItem(references.SlotShield, references.Bow)
	gs.PartyState.Inventory.Equipment.Set(references.Arrows, 1)
	require.True(t, gs.BeginCombatAutoAttack(target.Position()))
	hp := target.CurrentHP()
	launch := gs.ContinueCombatAutoAttack()
	require.Equal(t, CombatAttackSourcePending, launch.Outcome)
	require.NotNil(t, launch.Launch)
	assert.Equal(t, uint16(0), gs.PartyState.Inventory.Equipment.Get(references.Arrows))
	assert.Equal(t, hp, target.CurrentHP())
	assert.Empty(t, mock.CombatEvents)
	assert.Equal(t, CombatAttackTargetIgnored, gs.ContinueCombatAutoAttack().Outcome)
	assert.Equal(t, CombatAttackSourceResolved, gs.ResolvePendingCombatAttack().Outcome)
	assert.Equal(t, CombatAttackTargetIgnored, gs.ResolvePendingCombatAttack().Outcome)
	rngCalls := gs.GetRNG().Calls()
	assert.Equal(t, CombatAttackSequenceComplete, gs.ContinueCombatAutoAttack().Outcome)
	assert.Equal(t, rngCalls, gs.GetRNG().Calls(), "later unready source is skipped without RNG")
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, CombatEventRangedAttack, mock.CombatEvents[0].Kind)
	assert.Nil(t, gs.CombatScene.AttackTargeting)
}

func TestCombatAutoAttackMixedSourcesRevalidateAfterImpact(t *testing.T) {
	gs, mock := newCombatAttackPreviewTest(t, references.Bow)
	target, _ := addSyntheticTriggerMonster(gs, references.Position{X: 5, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	character := gs.PartyState.GetCharacter(0)
	character.SetEquippedItem(references.SlotHelm, references.SpikedHelm)
	character.SetEquippedItem(references.SlotShield, references.Bow)
	require.True(t, gs.BeginCombatAutoAttack(target.Position()))
	launch := gs.ContinueCombatAutoAttack()
	require.NotNil(t, launch.Launch)
	assert.Equal(t, 1, gs.CombatScene.AttackTargeting.SourceIndex, "out-of-range helm skipped")
	gs.ResolvePendingCombatAttack()
	setBlockedCombatMissileMap(gs, references.Position{X: 3, Y: 2})
	assert.Equal(t, CombatAttackSequenceComplete, gs.ContinueCombatAutoAttack().Outcome)
	assert.Equal(t, uint16(9), gs.PartyState.Inventory.Equipment.Get(references.Arrows))
	require.Len(t, mock.CombatEvents, 1)
}

func TestCombatAutoAttackStopsAtChosenTargetDeathWithoutRetargeting(t *testing.T) {
	gs, mock := newCombatAttackPreviewTest(t, references.Bow)
	target, _ := addSyntheticTriggerMonster(gs, references.Position{X: 5, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	other, _ := addSyntheticTriggerMonster(gs, references.Position{X: 2, Y: 5}, CombatMonsterLoyaltyMonster, references.MissileNone)
	gs.PartyState.GetCharacter(0).SetEquippedItem(references.SlotShield, references.Bow)
	require.True(t, gs.BeginCombatAutoAttack(target.Position()))
	gs.ContinueCombatAutoAttack()
	gs.ResolvePendingCombatAttack()
	target.SetCurrentHP(0)
	otherHP := other.CurrentHP()
	assert.Equal(t, CombatAttackSequenceComplete, gs.ContinueCombatAutoAttack().Outcome)
	assert.Equal(t, otherHP, other.CurrentHP())
	assert.Equal(t, uint16(9), gs.PartyState.Inventory.Equipment.Get(references.Arrows))
	require.Len(t, mock.CombatEvents, 1)
}

func TestCombatAutoAttackUsesExactCloneOriginAndSourceOrder(t *testing.T) {
	gs, mock := newCombatAttackPreviewTest(t, references.MagicAxe)
	actor := gs.CombatScene.GetCurrentTurnIdentity()
	clonePosition := references.Position{X: 7, Y: 7}
	clone := gs.CombatScene.addPCCloneFromSource(actor, 0, clonePosition, NewCombatPCProvider(gs.PartyState.GetCharacter(0)))
	gs.CombatScene.CurrentTurnIdentity = clone
	target, _ := addSyntheticTriggerMonster(gs, references.Position{X: 7, Y: 9}, CombatMonsterLoyaltyMonster, references.MissileNone)
	gs.PartyState.GetCharacter(0).SetEquippedItem(references.SlotHelm, references.SpikedHelm)
	gs.PartyState.GetCharacter(0).SetEquippedItem(references.SlotShield, references.MainGauche)
	assert.Equal(t, CombatAutoActionAttack, gs.PlanCombatAutoAction().Kind)
	require.True(t, gs.BeginCombatAutoAttack(target.Position()))
	assert.Equal(t, clone, gs.CombatScene.AttackTargeting.Attacker)
	launch := gs.ContinueCombatAutoAttack()
	require.NotNil(t, launch.Launch)
	assert.Equal(t, clonePosition, launch.Launch.From)
	require.NotNil(t, launch.ReturnLaunch)
	assert.Equal(t, clonePosition, launch.ReturnLaunch.Impact)
	assert.Equal(t, CombatAttackSourceResolved, gs.ResolvePendingCombatAttack().Outcome)
	require.NotEmpty(t, mock.CombatEvents)
	assert.Equal(t, clone.Slot, mock.CombatEvents[0].Attacker.Slot)
	assert.Equal(t, CombatAttackSequenceComplete, gs.ContinueCombatAutoAttack().Outcome, "off-hand melee remains out of range")
}

func TestCombatAutoAttackSourceOrderAndReplacementCannotRetarget(t *testing.T) {
	gs, mock := newCombatAttackPreviewTest(t, references.ShortSword)
	character := gs.PartyState.GetCharacter(0)
	character.SetEquippedItem(references.SlotHelm, references.SpikedHelm)
	character.SetEquippedItem(references.SlotShield, references.MainGauche)
	target, _ := addSyntheticTriggerMonster(gs, references.Position{X: 3, Y: 3}, CombatMonsterLoyaltyMonster, references.MissileNone)
	require.True(t, gs.BeginCombatAutoAttack(target.Position()))
	for _, source := range []references.Equipment{references.SpikedHelm, references.ShortSword, references.MainGauche} {
		result := gs.ContinueCombatAutoAttack()
		assert.Nil(t, result.Launch)
		require.NotEmpty(t, mock.CombatEvents)
		assert.Equal(t, source, mock.CombatEvents[len(mock.CombatEvents)-1].Diagnostics.Source)
	}
	assert.Nil(t, gs.CombatScene.AttackTargeting)
	require.True(t, gs.BeginCombatAutoAttack(target.Position()))
	target.SetActive(false)
	addSyntheticTriggerMonster(gs, target.Position(), CombatMonsterLoyaltyMonster, references.MissileNone)
	count := len(mock.CombatEvents)
	assert.Equal(t, CombatAttackSequenceComplete, gs.ContinueCombatAutoAttack().Outcome)
	assert.Len(t, mock.CombatEvents, count, "replacement in the same square is not the chosen victim")
}

func TestCombatAttackPreviewHiddenCorpserAndUnrelatedTargeting(t *testing.T) {
	gs, _ := newCombatAttackPreviewTest(t, references.Bow)
	target, _ := addSyntheticTriggerMonster(gs, references.Position{X: 5, Y: 2}, CombatMonsterLoyaltyMonster, references.MissileNone)
	target.monsterType = references.MonsterCorpser
	require.True(t, gs.BeginCombatAttack())
	gs.CombatScene.AttackTargeting.Cursor = target.Position()
	assert.Empty(t, gs.PreviewCombatAttack().Targets)
	assert.Empty(t, gs.PreviewCombatAttack().Aim.Name)
	target.SetCorpserExposed(true)
	assert.Len(t, gs.PreviewCombatAttack().Targets, 1)
	gs.CancelInitialCombatAttack()
	gs.CombatScene.SpellTargeting = &CombatSpellTargeting{}
	assert.Empty(t, gs.PreviewCombatAttack())
	gs.CombatScene.SpellTargeting = nil
	gs.CombatScene.SoloMoveMode = true
	gs.CombatScene.SoloMoveCharacterIndex = 1
	assert.Empty(t, gs.PreviewCombatAttack())
	assert.Equal(t, CombatAutoActionUnavailable, gs.PlanCombatAutoAction().Kind)
}
