package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func setupPlayerRangedAttackTest(
	t *testing.T,
	weapon references.Equipment,
	targetType references.MonsterType,
	targetDistance int,
) (*GameState, *MockSystemCallbacks, *CombatMonster, int, references.Position) {
	t.Helper()
	gs, mock := setupCombatAttackTargeting(t)
	setOpenCombatMissileMap(gs)
	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	origin := references.Position{X: 2, Y: 5}
	gs.CombatScene.SetPartyMemberPosition(attackerIndex, origin)
	for index := range gs.CombatScene.PartyActive {
		if index != attackerIndex && gs.CombatScene.PartyActive[index] {
			gs.CombatScene.SetPartyMemberPosition(index, references.Position{
				X: references.Coordinate(index + 1), Y: 10,
			})
		}
	}
	targetPosition := references.Position{X: origin.X + references.Coordinate(targetDistance), Y: origin.Y}
	enemyReferences := *gs.GameReferences.EnemyReferences
	targetReference := enemyReferences[targetType]
	targetReference.Dexterity = 1
	targetReference.HitPoints = 100
	target := NewCombatMonster(targetType, &targetReference, targetPosition, newFixedRNG(1))
	target.SetCurrentHP(100)
	targetIndex := gs.CombatScene.AddMonsterToQueue(target)
	character := gs.PartyState.GetCharacter(attackerIndex)
	character.Dexterity = 99
	character.Strength = 99
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotWeapon, weapon)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	require.True(t, gs.BeginCombatAttack())
	gs.CombatScene.AttackTargeting.Cursor = targetPosition
	gs.SetRandomSeed(findPlayerRangedHitSeed(t, gs, weapon, target))
	mock.Reset()

	return gs, mock, target, targetIndex, targetPosition
}

func setOpenCombatMissileMap(gs *GameState) {
	rawData := &[references.XCombatMapTiles][references.YCombatMapTiles]byte{}
	for x := range references.XCombatMapTiles {
		for y := range references.YCombatMapTiles {
			rawData[x][y] = byte(indexes.BrickFloor)
		}
	}
	gs.CombatScene.CombatMapRef.SetRawData(rawData)
}

func setBlockedCombatMissileMap(gs *GameState, blocked references.Position) {
	rawData := &[references.XCombatMapTiles][references.YCombatMapTiles]byte{}
	for x := range references.XCombatMapTiles {
		for y := range references.YCombatMapTiles {
			rawData[x][y] = byte(indexes.BrickFloor)
		}
	}
	rawData[blocked.X][blocked.Y] = byte(indexes.StoneBrickWall)
	gs.CombatScene.CombatMapRef.SetRawData(rawData)
}

func findPlayerRangedHitSeed(
	t *testing.T,
	gs *GameState,
	weapon references.Equipment,
	target *CombatMonster,
) uint64 {
	t.Helper()
	attacker := gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex())
	attackerStat := int(attacker.Dexterity)
	if weapon.IsBludgeoning() {
		attackerStat = int(attacker.Strength)
	}
	for seed := uint64(0); seed < 1000; seed++ {
		gs.SetRandomSeed(seed)
		// Original combat damage rules read combat queue DEX, not the reference stat.
		// See ref:COMBAT-0004.
		if CombatHitCheck(attackerStat, target.CurrentDex(), false, gs.AsRandomProvider()).Outcome != CombatHitMiss {
			return seed
		}
	}
	t.Fatal("expected a deterministic ranged hit seed")

	return 0
}

func TestPlayerRangedAmmoWeaponsCommitThenResolve(t *testing.T) {
	tests := []struct {
		name   string
		weapon references.Equipment
		ammo   references.Equipment
	}{
		{name: "bow arrow", weapon: references.Bow, ammo: references.Arrows},
		{name: "crossbow quarrel", weapon: references.Crossbow, ammo: references.Quarrels},
		{name: "magic bow arrow", weapon: references.MagicBow, ammo: references.Arrows},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, mock, target, targetIndex, _ := setupPlayerRangedAttackTest(t, test.weapon, references.MonsterOrc, 3)
			gs.PartyState.Inventory.Equipment.Set(test.ammo, 2)
			hpBefore := target.CurrentHP()

			confirmation := gs.ConfirmCombatAttackTarget()

			require.Equal(t, CombatAttackSourcePending, confirmation.Outcome)
			require.NotNil(t, confirmation.Launch)
			assert.Equal(t, uint16(1), gs.PartyState.Inventory.Equipment.Get(test.ammo))
			assert.Equal(t, hpBefore, target.CurrentHP(), "damage must wait for impact")
			require.NotNil(t, gs.CombatScene.AttackTargeting)
			assert.Equal(t, 0, gs.CombatScene.AttackTargeting.SourceIndex)
			assert.Empty(t, mock.CombatEvents)

			resolution := gs.ResolvePendingCombatAttack()

			assert.Equal(t, CombatAttackSequenceComplete, resolution.Outcome)
			assert.True(t, resolution.ShowImpact)
			assert.Nil(t, gs.CombatScene.AttackTargeting)
			require.NotEmpty(t, mock.CombatEvents)
			event := mock.CombatEvents[len(mock.CombatEvents)-1]
			assert.Equal(t, CombatEventRangedAttack, event.Kind)
			assert.Equal(t, test.weapon, event.Diagnostics.Source)
			assert.Equal(t, test.weapon.GetMissileType(), event.Diagnostics.Missile)
			assert.Equal(t, targetIndex, event.Target.Index)
		})
	}
}

func TestLastAmmoUnreadiesEveryExactFiringWeapon(t *testing.T) {
	gs, _, _, _, _ := setupPlayerRangedAttackTest(t, references.Crossbow, references.MonsterOrc, 3)
	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	otherIndex := (attackerIndex + 1) % party_state.MaxActiveParty
	require.True(t, gs.PartyState.GetCharacter(otherIndex).IsInParty())
	gs.PartyState.GetCharacter(otherIndex).SetEquippedItem(references.SlotWeapon, references.Crossbow)
	gs.PartyState.Inventory.Equipment.Set(references.Crossbow, 0)
	gs.PartyState.Inventory.Equipment.Set(references.Quarrels, 1)

	confirmation := gs.ConfirmCombatAttackTarget()

	require.Equal(t, CombatAttackSourcePending, confirmation.Outcome)
	assert.Equal(t, references.NoEquipment,
		gs.PartyState.GetCharacter(attackerIndex).GetEquippedItem(references.SlotWeapon))
	assert.Equal(t, references.NoEquipment,
		gs.PartyState.GetCharacter(otherIndex).GetEquippedItem(references.SlotWeapon))
	assert.Equal(t, uint16(2), gs.PartyState.Inventory.Equipment.Get(references.Crossbow))
}

func TestUnavailableCombatAttackSourcesRejectZeroAmmoBeforeAim(t *testing.T) {
	tests := []struct {
		name   string
		weapon references.Equipment
		ammo   references.Equipment
	}{
		{name: "bow arrows", weapon: references.Bow, ammo: references.Arrows},
		{name: "crossbow quarrels", weapon: references.Crossbow, ammo: references.Quarrels},
		{name: "magic bow arrows", weapon: references.MagicBow, ammo: references.Arrows},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, mock, target, _, _ := setupPlayerRangedAttackTest(t, test.weapon, references.MonsterOrc, 3)
			gs.PartyState.Inventory.Equipment.Set(test.ammo, 0)
			gs.SetRandomSeed(77)
			expectedNextRandom := gs.GetRNG().Intn(1000)
			gs.SetRandomSeed(77)
			hpBefore := target.CurrentHP()

			result := gs.SkipUnavailableCombatAttackSources()

			assert.Equal(t, CombatAttackSequenceComplete, result.Outcome)
			assert.Nil(t, result.Launch)
			assert.Nil(t, gs.CombatScene.AttackTargeting)
			assert.Equal(t, expectedNextRandom, gs.GetRNG().Intn(1000))
			assert.Equal(t, hpBefore, target.CurrentHP())
			assert.Contains(t, mock.Messages, "No "+test.ammo.String()+"!")
			assert.Empty(t, mock.CombatEvents)
		})
	}
}

func TestUnavailableCombatAttackSourcesAdvancesPastZeroAmmo(t *testing.T) {
	gs, mock, _, _, _ := setupPlayerRangedAttackTest(t, references.Bow, references.MonsterOrc, 3)
	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	gs.PartyState.GetCharacter(attackerIndex).SetEquippedItem(references.SlotShield, references.SpikedShield)
	gs.CombatScene.AttackTargeting = nil
	require.True(t, gs.BeginCombatAttack())
	require.Equal(t, references.Bow, gs.CombatScene.AttackTargeting.CurrentSource().Equipment)
	gs.PartyState.Inventory.Equipment.Set(references.Arrows, 0)
	mock.Reset()

	result := gs.SkipUnavailableCombatAttackSources()

	assert.Equal(t, CombatAttackSourceResolved, result.Outcome)
	require.NotNil(t, gs.CombatScene.AttackTargeting)
	assert.Equal(t, references.SpikedShield, gs.CombatScene.AttackTargeting.CurrentSource().Equipment)
	assert.Contains(t, mock.Messages, "No Arrows!")
}

func TestPlayerRangedZeroAmmoConfirmationSkipsWithoutRNGOrLaunch(t *testing.T) {
	gs, mock, target, _, _ := setupPlayerRangedAttackTest(t, references.Bow, references.MonsterOrc, 3)
	gs.PartyState.Inventory.Equipment.Set(references.Arrows, 0)
	gs.SetRandomSeed(77)
	expectedNextRandom := gs.GetRNG().Intn(1000)
	gs.SetRandomSeed(77)
	hpBefore := target.CurrentHP()

	confirmation := gs.ConfirmCombatAttackTarget()

	assert.Equal(t, CombatAttackSequenceComplete, confirmation.Outcome)
	assert.Nil(t, confirmation.Launch)
	assert.Equal(t, expectedNextRandom, gs.GetRNG().Intn(1000))
	assert.Equal(t, hpBefore, target.CurrentHP())
	assert.Contains(t, mock.Messages, "No Arrows!")
	assert.Empty(t, mock.CombatEvents)
}

func TestEmptyRangedAimConsumesOnlyFlamingOil(t *testing.T) {
	tests := []struct {
		name          string
		weapon        references.Equipment
		inventoryItem references.Equipment
		wantQuantity  uint16
		wantField     bool
	}{
		{name: "bow", weapon: references.Bow, inventoryItem: references.Arrows, wantQuantity: 2},
		{name: "flaming oil", weapon: references.FlamingOil, inventoryItem: references.FlamingOil, wantQuantity: 1, wantField: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, mock, _, _, _ := setupPlayerRangedAttackTest(t, test.weapon, references.MonsterOrc, 3)
			emptyAim := references.Position{X: 2, Y: 2}
			gs.CombatScene.AttackTargeting.Cursor = emptyAim
			gs.PartyState.Inventory.Equipment.Set(test.inventoryItem, 2)

			confirmation := gs.ConfirmCombatAttackTarget()

			require.Equal(t, CombatAttackSourcePending, confirmation.Outcome)
			assert.Equal(t, test.wantQuantity, gs.PartyState.Inventory.Equipment.Get(test.inventoryItem))
			resolution := gs.ResolvePendingCombatAttack()
			assert.Equal(t, CombatAttackSequenceComplete, resolution.Outcome)
			assert.False(t, resolution.ShowImpact)
			require.NotEmpty(t, mock.CombatEvents)
			event := mock.CombatEvents[len(mock.CombatEvents)-1]
			assert.Equal(t, CombatEventRangedAttack, event.Kind)
			assert.Equal(t, confirmation.Launch.Impact, event.To)
			if test.wantField {
				require.Len(t, gs.CombatScene.Fields(), 1)
				assert.Equal(t, CombatFieldFire, gs.CombatScene.Fields()[0].Kind())
				assert.Equal(t, confirmation.Launch.Impact, gs.CombatScene.Fields()[0].Position())
			} else {
				assert.Empty(t, gs.CombatScene.Fields())
			}
		})
	}
}

func TestCombatCloneRangedAttackUsesExactIdentityAndPosition(t *testing.T) {
	gs, mock, target, _, _ := setupPlayerRangedAttackTest(t, references.Sling, references.MonsterOrc, 3)
	partyIndex := gs.CombatScene.GetActiveCombatantIndex()
	clonePosition := references.Position{X: 1, Y: 5}
	clone := gs.CombatScene.addPCCloneFromSource(
		CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex},
		partyIndex, clonePosition, NewCombatPCProvider(gs.PartyState.GetCharacter(partyIndex)),
	)
	gs.CombatScene.CurrentTurnIdentity = clone
	gs.CombatScene.currentTurnIdentitySet = true
	gs.CombatScene.AttackTargeting = nil
	require.True(t, gs.BeginCombatAttack())
	assert.Equal(t, clone, gs.CombatScene.AttackTargeting.Attacker)
	assert.Equal(t, clonePosition, gs.CombatScene.AttackTargeting.Origin)
	gs.CombatScene.AttackTargeting.Cursor = target.Position()
	mock.Reset()

	confirmation := gs.ConfirmCombatAttackTarget()
	require.NotNil(t, confirmation.Launch)
	assert.Equal(t, clonePosition, confirmation.Launch.From)
	gs.ResolvePendingCombatAttack()

	require.NotEmpty(t, mock.CombatEvents)
	event := mock.CombatEvents[len(mock.CombatEvents)-1]
	assert.Equal(t, clonePosition, event.From)
	assert.Equal(t, clonePosition, event.Attacker.Position)
}

func TestPhysicalRangedAttackCanHitManuallySelectedInvisibleMonster(t *testing.T) {
	gs, _, target, _, _ := setupPlayerRangedAttackTest(t, references.Sling, references.MonsterShadowlord, 3)
	target.SetInvisible(true)

	confirmation := gs.ConfirmCombatAttackTarget()

	require.Equal(t, CombatAttackSourcePending, confirmation.Outcome)
	require.NotNil(t, gs.CombatScene.pendingAttack)
	assert.True(t, gs.CombatScene.pendingAttack.primarySet)
	assert.True(t, gs.CombatScene.pendingAttack.diagnostics.Hit.Rolled)
}

func TestCloneThrownWeaponRecoveryUsesCloneRange(t *testing.T) {
	gs, _, target, _, targetPosition := setupPlayerRangedAttackTest(
		t, references.Dagger, references.MonsterOrc, 3,
	)
	partyIndex := gs.CombatScene.GetActiveCombatantIndex()
	source := CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex}
	clonePosition := references.Position{X: targetPosition.X - 1, Y: targetPosition.Y}
	clone := gs.CombatScene.addPCCloneFromSource(
		source, partyIndex, clonePosition, NewCombatPCProvider(gs.PartyState.GetCharacter(partyIndex)),
	)
	gs.PartyState.Inventory.Equipment.Set(references.Dagger, 0)
	sources := collectCombatAttackSources(gs.PartyState.GetCharacter(partyIndex))
	require.NotEmpty(t, sources)
	require.Equal(t, references.Dagger, sources[0].Equipment)

	assert.True(t, gs.commitPlayerRangedResource(clone, sources[0], true, target.Position()))
	assert.Equal(t, references.Dagger,
		gs.PartyState.GetCharacter(partyIndex).GetEquippedItem(references.SlotWeapon))
}

func TestCombatCloneDirectRangedTargetUsesLoyaltyAndHitRoll(t *testing.T) {
	for _, test := range []struct {
		name       string
		possessed  bool
		wantLaunch bool
	}{
		{name: "friendly clone rejected"},
		{name: "possessed clone targeted", possessed: true, wantLaunch: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			gs, _, _, _, targetPosition := setupPlayerRangedAttackTest(t, references.Sling, references.MonsterOrc, 3)
			clearCombatMonstersForTest(gs.CombatScene)
			attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
			targetIndex := (attackerIndex + 1) % party_state.MaxActiveParty
			clone := gs.CombatScene.addPCCloneFromSource(
				CombatTurnIdentity{IsPartyMember: true, EntityIndex: targetIndex},
				targetIndex, targetPosition, NewCombatPCProvider(gs.PartyState.GetCharacter(targetIndex)),
			)
			gs.CombatScene.setCombatPCPossessed(clone, test.possessed)
			gs.CombatScene.AttackTargeting.Cursor = targetPosition

			confirmation := gs.ConfirmCombatAttackTarget()

			assert.Equal(t, test.wantLaunch, confirmation.Launch != nil)
			if test.wantLaunch {
				require.NotNil(t, gs.CombatScene.pendingAttack)
				assert.Equal(t, clone, gs.CombatScene.pendingAttack.primaryTarget)
				assert.True(t, gs.CombatScene.pendingAttack.diagnostics.Hit.Rolled)
			}
		})
	}
}

func TestRecoverableThrownWeaponConsumptionByRangeAndSpare(t *testing.T) {
	tests := []struct {
		name         string
		distance     int
		spares       uint16
		wantSpares   uint16
		wantEquipped references.Equipment
	}{
		{name: "recovered adjacent", distance: 1, spares: 0, wantSpares: 0, wantEquipped: references.Dagger},
		{name: "distant spare", distance: 2, spares: 1, wantSpares: 0, wantEquipped: references.Dagger},
		{name: "distant readied copy", distance: 2, spares: 0, wantSpares: 0, wantEquipped: references.NoEquipment},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, _, _, _, _ := setupPlayerRangedAttackTest(t, references.Dagger, references.MonsterOrc, test.distance)
			gs.PartyState.Inventory.Equipment.Set(references.Dagger, test.spares)

			confirmation := gs.ConfirmCombatAttackTarget()

			require.Equal(t, CombatAttackSourcePending, confirmation.Outcome)
			assert.Equal(t, test.wantSpares, gs.PartyState.Inventory.Equipment.Get(references.Dagger))
			assert.Equal(t, test.wantEquipped,
				gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex()).GetEquippedItem(references.SlotWeapon))
		})
	}
}

func TestMagicAxeReturnsWithoutConsumptionAndWaitsForImpact(t *testing.T) {
	gs, mock, target, _, _ := setupPlayerRangedAttackTest(t, references.MagicAxe, references.MonsterOrc, 3)
	hpBefore := target.CurrentHP()

	confirmation := gs.ConfirmCombatAttackTarget()

	require.Equal(t, CombatAttackSourcePending, confirmation.Outcome)
	require.NotNil(t, confirmation.Launch)
	require.NotNil(t, confirmation.ReturnLaunch)
	assert.Equal(t, confirmation.Launch.Impact, confirmation.ReturnLaunch.From)
	assert.Equal(t, confirmation.Launch.From, confirmation.ReturnLaunch.Impact)
	assert.Equal(t, references.MagicAxe,
		gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex()).GetEquippedItem(references.SlotWeapon))
	assert.Equal(t, hpBefore, target.CurrentHP())
	assert.Empty(t, mock.CombatEvents)

	assert.Equal(t, CombatAttackSequenceComplete, gs.ResolvePendingCombatAttack().Outcome)
	assert.NotEmpty(t, mock.CombatEvents)
}

func TestRangedInterferenceSkipsOnlyCurrentSource(t *testing.T) {
	gs, mock, _, _, targetPosition := setupPlayerRangedAttackTest(t, references.Bow, references.MonsterOrc, 3)
	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(attackerIndex)
	character.SetEquippedItem(references.SlotHelm, references.SpikedHelm)
	character.SetEquippedItem(references.SlotShield, references.SpikedShield)
	gs.CombatScene.AttackTargeting = nil
	require.True(t, gs.BeginCombatAttack())
	require.Equal(t, CombatAttackSourceCancelled, gs.CancelCombatAttackSource().Outcome)
	require.Equal(t, references.Bow, gs.CombatScene.AttackTargeting.CurrentSource().Equipment)
	gs.CombatScene.AttackTargeting.Cursor = targetPosition
	engager, engagerIndex := addControlledTestMonster(
		gs, references.MonsterTroll,
		references.Position{X: gs.CombatScene.AttackTargeting.Origin.X + 1, Y: gs.CombatScene.AttackTargeting.Origin.Y},
		CombatMonsterLoyaltyMonster,
	)
	gs.rememberMeleeEngager(
		CombatTurnIdentity{IsPartyMember: true, EntityIndex: attackerIndex},
		CombatTurnIdentity{IsPartyMember: false, EntityIndex: engagerIndex},
	)
	gs.PartyState.Inventory.Equipment.Set(references.Arrows, 2)
	gs.SetRandomSeed(91)
	expectedNextRandom := gs.GetRNG().Intn(1000)
	gs.SetRandomSeed(91)
	mock.Reset()

	confirmation := gs.SkipUnavailableCombatAttackSources()

	assert.Equal(t, CombatAttackSourceResolved, confirmation.Outcome)
	assert.Nil(t, confirmation.Launch)
	require.NotNil(t, gs.CombatScene.AttackTargeting)
	assert.Equal(t, references.SpikedShield, gs.CombatScene.AttackTargeting.CurrentSource().Equipment)
	assert.Equal(t, uint16(2), gs.PartyState.Inventory.Equipment.Get(references.Arrows))
	assert.Equal(t, expectedNextRandom, gs.GetRNG().Intn(1000))
	assert.Contains(t, mock.Messages, engager.MonsterType().Name()+" interferes!")
	require.Len(t, mock.CombatEvents, 1, "interference must reach the structured Battle Report path")
	assert.Equal(t, CombatEventInterruption, mock.CombatEvents[0].Kind)
	assert.Equal(t, CombatInterruptedAttack, mock.CombatEvents[0].InterruptedAction)
	assert.Equal(t, references.Bow, mock.CombatEvents[0].Diagnostics.Source)
	assert.Equal(t, engagerIndex, mock.CombatEvents[0].Attacker.Index)
	assert.Equal(t, attackerIndex, mock.CombatEvents[0].Target.Index)
}

func TestBlockedRangedLaunchCommitsResourceAndPublishesEndpointWithoutImpact(t *testing.T) {
	gs, mock, target, _, _ := setupPlayerRangedAttackTest(t, references.Bow, references.MonsterOrc, 4)
	gs.PartyState.Inventory.Equipment.Set(references.Arrows, 2)
	blockedPosition := references.Position{X: 4, Y: 5}
	setBlockedCombatMissileMap(gs, blockedPosition)
	hpBefore := target.CurrentHP()

	confirmation := gs.ConfirmCombatAttackTarget()

	require.NotNil(t, confirmation.Launch)
	assert.True(t, confirmation.Launch.Blocked)
	assert.Equal(t, blockedPosition, confirmation.Launch.Impact)
	assert.Equal(t, uint16(1), gs.PartyState.Inventory.Equipment.Get(references.Arrows))
	assert.Equal(t, CombatAttackSequenceComplete, gs.ResolvePendingCombatAttack().Outcome)
	assert.Equal(t, hpBefore, target.CurrentHP())
	assert.Empty(t, mock.CombatImpactCalls, "blocked shots must not invoke impact presentation")
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, blockedPosition, mock.CombatEvents[0].To)
	assert.True(t, mock.CombatEvents[0].Diagnostics.ProjectileBlocked)
}

func TestMissRedirectCanDamageFriendlyCollateralAndNeverSelf(t *testing.T) {
	gs, mock, target, _, targetPosition := setupPlayerRangedAttackTest(t, references.MagicAxe, references.MonsterOrc, 3)
	attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
	attacker := gs.PartyState.GetCharacter(attackerIndex)
	attacker.Dexterity = 1
	// Original combat damage rules use encounter DEX. Changing the reference after
	// creation does not make this target harder to hit.
	// See ref:COMBAT-0004.
	target.currentDex = 30
	collateralPosition := references.Position{X: targetPosition.X + 1, Y: targetPosition.Y}
	collateralIndex := (attackerIndex + 1) % party_state.MaxActiveParty
	collateral := gs.PartyState.GetCharacter(collateralIndex)
	require.True(t, collateral.IsInParty())
	for _, slot := range references.EquipmentSlots() {
		collateral.SetEquippedItem(slot, references.NoEquipment)
	}
	require.Zero(t, gs.combatPartyDefense(collateral))
	gs.CombatScene.SetPartyMemberPosition(collateralIndex, collateralPosition)
	hpBefore := collateral.GetCurrentHp()
	attackerHPBefore := attacker.GetCurrentHp()
	targetHPBefore := target.CurrentHP()
	gs.SetRandomSeed(0)
	mock.Reset()

	confirmation := gs.ConfirmCombatAttackTarget()

	require.NotNil(t, confirmation.Launch)
	assert.Equal(t, uint64(3), gs.GetRNG().Calls(), "one D30 hit roll, then adjacent X and Y")
	assert.Equal(t, collateralPosition, confirmation.Launch.Aim)
	assert.NotEqual(t, confirmation.Launch.From, confirmation.Launch.Impact)
	assert.Equal(t, hpBefore, collateral.GetCurrentHp(), "damage is deferred until impact")
	assert.Equal(t, CombatAttackSequenceComplete, gs.ResolvePendingCombatAttack().Outcome)
	require.NotEmpty(t, mock.CombatEvents)
	event := mock.CombatEvents[len(mock.CombatEvents)-1]
	assert.True(t, event.Target.IsPartyMember)
	assert.Equal(t, collateralIndex, event.Target.Index)
	assert.Equal(t, CombatHitMiss, event.Diagnostics.Hit.Outcome)
	assert.Equal(t, 30, event.Diagnostics.Hit.DefenderEffectiveDex)
	assert.Equal(t, 29, event.Diagnostics.Hit.Threshold)
	assert.Equal(t, CombatFeedbackDamage, event.Result)
	assert.Less(t, collateral.GetCurrentHp(), hpBefore)
	assert.Equal(t, int(hpBefore-collateral.GetCurrentHp()), event.HPLost)
	assert.Equal(t, attackerHPBefore, attacker.GetCurrentHp())
	assert.Equal(t, targetHPBefore, target.CurrentHP())
	assert.Equal(t, uint64(4), gs.GetRNG().Calls(), "collateral takes damage without a second hit check")
}

func TestPlayerRangedDirectSafetyUsesEffectiveLoyalty(t *testing.T) {
	tests := []struct {
		name       string
		possessed  bool
		wantLaunch bool
		wantAmmo   uint16
	}{
		{name: "ordinary ally rejected", wantAmmo: 1},
		{name: "possessed ally is threat", possessed: true, wantLaunch: true, wantAmmo: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, _, _, _, targetPosition := setupPlayerRangedAttackTest(t, references.Bow, references.MonsterOrc, 3)
			clearCombatMonstersForTest(gs.CombatScene)
			attackerIndex := gs.CombatScene.GetActiveCombatantIndex()
			targetIndex := (attackerIndex + 1) % party_state.MaxActiveParty
			gs.CombatScene.SetPartyMemberPosition(targetIndex, targetPosition)
			gs.CombatScene.PartyPossessed[targetIndex] = test.possessed
			if test.possessed {
				gs.PartyState.GetCharacter(targetIndex).SetStatus(party_state.Charmed)
			}
			gs.PartyState.Inventory.Equipment.Set(references.Arrows, 1)

			confirmation := gs.ConfirmCombatAttackTarget()

			assert.Equal(t, test.wantLaunch, confirmation.Launch != nil)
			assert.Equal(t, test.wantAmmo, gs.PartyState.Inventory.Equipment.Get(references.Arrows))
		})
	}
}

func TestPlayerRangedPhysicalPipelineAppliesArmorAndUndeadModifier(t *testing.T) {
	gs, mock, target, _, _ := setupPlayerRangedAttackTest(t, references.Bow, references.MonsterSkeleton, 3)
	gs.PartyState.Inventory.Equipment.Set(references.Arrows, 2)

	confirmation := gs.ConfirmCombatAttackTarget()
	require.NotNil(t, confirmation.Launch)
	gs.ResolvePendingCombatAttack()

	require.NotEmpty(t, mock.CombatEvents)
	event := mock.CombatEvents[len(mock.CombatEvents)-1]
	assert.True(t, event.Diagnostics.TargetUndead)
	assert.False(t, event.Diagnostics.MagicWeapon)
	assert.Equal(t, target.EnemyRef().Armour, event.Diagnostics.Damage.ArmorValue)
	assert.Equal(t, event.Diagnostics.Damage.Damage/2, event.Diagnostics.FinalDamage)
}

func TestPendingRangedAttackResolvesExactlyOnce(t *testing.T) {
	gs, mock, target, _, _ := setupPlayerRangedAttackTest(t, references.Sling, references.MonsterOrc, 3)
	confirmation := gs.ConfirmCombatAttackTarget()
	require.Equal(t, CombatAttackSourcePending, confirmation.Outcome)
	hpBefore := target.CurrentHP()

	first := gs.ResolvePendingCombatAttack()
	hpAfterFirst := target.CurrentHP()
	eventsAfterFirst := len(mock.CombatEvents)
	second := gs.ResolvePendingCombatAttack()

	assert.Equal(t, CombatAttackSequenceComplete, first.Outcome)
	assert.Equal(t, CombatAttackTargetIgnored, second.Outcome)
	assert.LessOrEqual(t, hpAfterFirst, hpBefore)
	assert.Equal(t, hpAfterFirst, target.CurrentHP())
	assert.Equal(t, eventsAfterFirst, len(mock.CombatEvents))
}

func TestControlledRangedAttackUsesPendingBlockedAndImpactLifecycle(t *testing.T) {
	t.Run("impact", func(t *testing.T) {
		gs, mock := setupControlledMonsterTest(t)
		setOpenCombatMissileMap(gs)
		attacker, attackerIndex := addControlledTestMonster(
			gs, references.MonsterDevil, references.Position{X: 2, Y: 5}, CombatMonsterLoyaltyParty,
		)
		target, targetIndex := addControlledTestMonster(
			gs, references.MonsterOrc, references.Position{X: 6, Y: 5}, CombatMonsterLoyaltyMonster,
		)
		target.currentDex = 1
		authorizeControlledMonster(gs, attackerIndex)
		require.True(t, gs.BeginCombatAttack())
		gs.CombatScene.AttackTargeting.Cursor = target.Position()
		mock.Reset()

		confirmation := gs.ConfirmCombatAttackTarget()
		require.Equal(t, CombatAttackSourcePending, confirmation.Outcome)
		assert.Empty(t, mock.CombatEvents)
		assert.Equal(t, CombatAttackSequenceComplete, gs.ResolvePendingCombatAttack().Outcome)
		require.NotEmpty(t, mock.CombatEvents)
		assert.Equal(t, CombatEventRangedAttack, mock.CombatEvents[len(mock.CombatEvents)-1].Kind)
		assert.Equal(t, attacker.Position(), mock.CombatEvents[len(mock.CombatEvents)-1].From)
		assert.Equal(t, targetIndex, mock.CombatEvents[len(mock.CombatEvents)-1].Target.Index)
	})

	t.Run("blocked", func(t *testing.T) {
		gs, mock := setupControlledMonsterTest(t)
		setOpenCombatMissileMap(gs)
		_, attackerIndex := addControlledTestMonster(
			gs, references.MonsterDevil, references.Position{X: 2, Y: 5}, CombatMonsterLoyaltyParty,
		)
		target, _ := addControlledTestMonster(
			gs, references.MonsterOrc, references.Position{X: 6, Y: 5}, CombatMonsterLoyaltyMonster,
		)
		authorizeControlledMonster(gs, attackerIndex)
		require.True(t, gs.BeginCombatAttack())
		gs.CombatScene.AttackTargeting.Cursor = target.Position()
		blockedPosition := references.Position{X: 4, Y: 5}
		setBlockedCombatMissileMap(gs, blockedPosition)
		hpBefore := target.CurrentHP()
		mock.Reset()

		confirmation := gs.ConfirmCombatAttackTarget()
		require.NotNil(t, confirmation.Launch)
		assert.True(t, confirmation.Launch.Blocked)
		assert.Equal(t, blockedPosition, confirmation.Launch.Impact)
		gs.ResolvePendingCombatAttack()
		assert.Equal(t, hpBefore, target.CurrentHP())
		assert.Empty(t, mock.CombatImpactCalls)
		require.Len(t, mock.CombatEvents, 1)
		assert.True(t, mock.CombatEvents[0].Diagnostics.ProjectileBlocked)
	})
}
