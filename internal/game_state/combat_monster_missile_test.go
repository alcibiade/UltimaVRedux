package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func setupMonsterMissileTest(
	t *testing.T,
	monsterType references.MonsterType,
	distance int,
) (*GameState, *MockSystemCallbacks, *CombatMonster, int, nearestPCResult) {
	t.Helper()
	gs, mock, _, _ := setupHostileDaemonTest(t)
	setOpenCombatMissileMap(gs)
	clearCombatMonstersForTest(gs.CombatScene)
	ref := (*gs.GameReferences.EnemyReferences)[monsterType]
	abilities := make(map[references.EnemyAbility]bool, len(ref.EnemyAbilities))
	for ability, enabled := range ref.EnemyAbilities {
		abilities[ability] = enabled
	}
	ref.EnemyAbilities = abilities
	ref.AttackRange = max(distance, 2)
	ref.LargeMapMissileRange = ref.AttackRange
	if ref.LargeMapMissile == references.MissileNone {
		ref.LargeMapMissile = references.MissileGreen
	}
	ref.Dexterity = 99
	ref.Damage = 20
	from := references.Position{X: 2, Y: 5}
	to := references.Position{X: from.X + references.Coordinate(distance), Y: from.Y}
	monster := NewCombatMonster(monsterType, &ref, from, newFixedRNG(1))
	monster.SetCurrentHP(100)
	monsterIndex := gs.CombatScene.AddMonsterToQueue(monster)
	gs.CombatScene.PartyPositions[0] = to
	character := gs.PartyState.GetCharacter(0)
	character.Dexterity = 1
	character.SetStatus(party_state.Good)
	character.SetEquippedItem(references.SlotArmour, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotAmulet, references.NoEquipment)
	mock.Reset()

	return gs, mock, monster, monsterIndex, gDaemonNearestPC(gs, monster, monsterIndex)
}

func TestMonsterMissileDecisionBoundaryAndMimicGuarantee(t *testing.T) {
	t.Run("127 fires and defers impact", func(t *testing.T) {
		gs, mock, monster, monsterIndex, nearest := setupMonsterMissileTest(t, references.MonsterOrc, 4)
		hpBefore := gs.PartyState.GetCharacter(0).GetCurrentHp()

		launch, attacked := gs.prepareMonsterAttackWithRNG(monster, monsterIndex, nearest, newSequenceRNG(127, 29))

		require.True(t, attacked)
		require.NotNil(t, launch)
		assert.Equal(t, hpBefore, gs.PartyState.GetCharacter(0).GetCurrentHp())
		assert.Empty(t, mock.CombatEvents)
		require.True(t, gs.ResolvePendingMonsterMissile())
		require.Len(t, mock.CombatEvents, 1)
		assert.Equal(t, CombatEventRangedAttack, mock.CombatEvents[0].Kind)
	})

	t.Run("128 seeks", func(t *testing.T) {
		gs, mock, monster, monsterIndex, nearest := setupMonsterMissileTest(t, references.MonsterOrc, 4)

		launch, attacked := gs.prepareMonsterAttackWithRNG(monster, monsterIndex, nearest, newSequenceRNG(128))

		assert.False(t, attacked)
		assert.Nil(t, launch)
		assert.Nil(t, gs.CombatScene.pendingMonsterMissile)
		assert.Empty(t, mock.CombatEvents)
	})

	t.Run("mimic always fires", func(t *testing.T) {
		gs, _, mimic, mimicIndex, nearest := setupMonsterMissileTest(t, references.MonsterMimic, 2)

		launch, attacked := gs.prepareMonsterAttackWithRNG(mimic, mimicIndex, nearest, newSequenceRNG(128, 0, 1))

		assert.True(t, attacked)
		assert.NotNil(t, launch)
	})
}

func TestProductionPhysicalRangedEnemiesLaunchConfiguredMissiles(t *testing.T) {
	tests := []struct {
		monster references.MonsterType
		missile references.MissileType
	}{
		{monster: references.MonsterBard, missile: references.MissileArrow},
		{monster: references.MonsterGuard, missile: references.MissileAxe},
		{monster: references.MonsterGargoyle, missile: references.MissileRock},
		{monster: references.MonsterEttin, missile: references.MissileRock},
		{monster: references.MonsterTroll, missile: references.MissileAxe},
	}
	for _, test := range tests {
		t.Run(test.monster.Name(), func(t *testing.T) {
			gs, mock, _, _ := setupHostileDaemonTest(t)
			setOpenCombatMissileMap(gs)
			clearCombatMonstersForTest(gs.CombatScene)
			reference := (*gs.GameReferences.EnemyReferences)[test.monster]
			from := references.Position{X: 2, Y: 5}
			to := references.Position{X: 4, Y: 5}
			monster := NewCombatMonster(test.monster, &reference, from, newFixedRNG(1))
			monsterIndex := gs.CombatScene.AddMonsterToQueue(monster)
			gs.CombatScene.PartyPositions[0] = to
			character := gs.PartyState.GetCharacter(0)
			character.SetStatus(party_state.Good)
			character.SetEquippedItem(references.SlotAmulet, references.NoEquipment)
			nearest := gDaemonNearestPC(gs, monster, monsterIndex)
			hpBefore := character.GetCurrentHp()
			mock.Reset()

			launch, attacked := gs.prepareMonsterAttackWithRNG(
				monster,
				monsterIndex,
				nearest,
				newSequenceRNG(127, 29, 0, 0),
			)

			require.True(t, attacked)
			require.NotNil(t, launch)
			assert.Equal(t, test.missile, launch.Missile)
			assert.Equal(t, hpBefore, character.GetCurrentHp(), "damage must wait for projectile impact")
			assert.Empty(t, mock.CombatEvents)
			require.True(t, gs.ResolvePendingMonsterMissile())
			require.Len(t, mock.CombatEvents, 1)
			assert.Equal(t, CombatEventRangedAttack, mock.CombatEvents[0].Kind)
			assert.Equal(t, test.missile, mock.CombatEvents[0].Diagnostics.Missile)
		})
	}
}

func TestOrdinaryMonsterMissileUsesFixedDamageWithoutImpactRNG(t *testing.T) {
	gs, mock, monster, monsterIndex, nearest := setupMonsterMissileTest(t, references.MonsterOrc, 4)
	character := gs.PartyState.GetCharacter(0)
	hpBefore := character.GetCurrentHp()
	launch, attacked := gs.prepareMonsterAttackWithRNG(
		monster, monsterIndex, nearest, newSequenceRNG(127, 29),
	)
	require.True(t, attacked)
	require.NotNil(t, launch)
	gs.SetRandomSeed(17)
	callsBefore := gs.GetRNG().Calls()

	require.True(t, gs.ResolvePendingMonsterMissile())

	assert.Equal(t, uint16(20), hpBefore-character.GetCurrentHp())
	assert.Equal(t, callsBefore, gs.GetRNG().Calls(), "unarmored missile impact consumes no damage RNG")
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, 20, mock.CombatEvents[0].Diagnostics.Damage.AttackRoll)
	assert.False(t, mock.CombatEvents[0].Diagnostics.Damage.AttackRolled)
}

func TestExecuteMonsterTurnUsesMissileBoundaryThenReportsActualAction(t *testing.T) {
	t.Run("127 projectile", func(t *testing.T) {
		gs, mock, monster, monsterIndex, _ := setupMonsterMissileTest(t, references.MonsterOrc, 4)
		entry := &TurnQueueEntry{Combatant: monster, EntityIndex: monsterIndex}
		plan := gs.planMonsterTurn(entry)
		seed := seedForFirstIntnValue(t, gs, monsterSpecialRollRange, 127)
		gs.SetRandomSeed(seed)
		hpBefore := gs.PartyState.GetCharacter(0).GetCurrentHp()

		result := gs.executeMonsterTurn(entry, plan)

		require.NotNil(t, result.ProjectileLaunch)
		assert.False(t, result.Moved)
		assert.Equal(t, hpBefore, gs.PartyState.GetCharacter(0).GetCurrentHp())
		assert.Empty(t, mock.CombatEvents)
		require.True(t, gs.ResolvePendingMonsterMissile())
		require.Len(t, mock.CombatEvents, 1)
	})

	t.Run("128 movement", func(t *testing.T) {
		gs, mock, monster, monsterIndex, _ := setupMonsterMissileTest(t, references.MonsterOrc, 4)
		entry := &TurnQueueEntry{Combatant: monster, EntityIndex: monsterIndex}
		plan := gs.planMonsterTurn(entry)
		seed := seedForFirstIntnValue(t, gs, monsterSpecialRollRange, 128)
		gs.SetRandomSeed(seed)
		from := monster.Position()

		result := gs.executeMonsterTurn(entry, plan)

		assert.Nil(t, result.ProjectileLaunch)
		assert.True(t, result.Moved)
		assert.NotEqual(t, from, monster.Position())
		require.Len(t, mock.CombatEvents, 1)
		assert.Equal(t, CombatEventMove, mock.CombatEvents[0].Kind)
	})
}

func TestMonsterMissileBlockedEndpointHasNoImpactOrDamage(t *testing.T) {
	gs, mock, monster, monsterIndex, nearest := setupMonsterMissileTest(t, references.MonsterOrc, 4)
	blocked := references.Position{X: 4, Y: 5}
	setBlockedCombatMissileMap(gs, blocked)
	hpBefore := gs.PartyState.GetCharacter(0).GetCurrentHp()

	launch, attacked := gs.prepareMonsterAttackWithRNG(monster, monsterIndex, nearest, newSequenceRNG(0, 29))

	require.True(t, attacked)
	require.NotNil(t, launch)
	assert.True(t, launch.Blocked)
	assert.Equal(t, blocked, launch.Impact)
	require.True(t, gs.ResolvePendingMonsterMissile())
	assert.Equal(t, hpBefore, gs.PartyState.GetCharacter(0).GetCurrentHp())
	assert.Empty(t, mock.CombatImpactCalls)
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, blocked, mock.CombatEvents[0].To)
	assert.True(t, mock.CombatEvents[0].Diagnostics.ProjectileBlocked)
}

func TestMonsterMissileRedirectHitsEitherLoyaltyCollateral(t *testing.T) {
	gs, mock, monster, monsterIndex, nearest := setupMonsterMissileTest(t, references.MonsterOrc, 4)
	monster.currentDex = 1
	primary := gs.CombatScene.PartyPositions[0]
	gs.PartyState.GetCharacter(0).Dexterity = 99
	collateralPosition := references.Position{X: primary.X, Y: primary.Y + 1}
	allyRef := (*gs.GameReferences.EnemyReferences)[references.MonsterTroll]
	ally := NewCombatMonster(references.MonsterTroll, &allyRef, collateralPosition, newFixedRNG(1))
	ally.SetLoyalty(CombatMonsterLoyaltyMonster)
	ally.SetCurrentHP(100)
	allyIndex := gs.CombatScene.AddMonsterToQueue(ally)
	hpBefore := ally.CurrentHP()

	launch, attacked := gs.prepareMonsterAttackWithRNG(
		monster, monsterIndex, nearest, newSequenceRNG(0, 0, 1, 2),
	)

	require.True(t, attacked)
	require.NotNil(t, launch)
	assert.Equal(t, collateralPosition, launch.Impact)
	require.True(t, gs.ResolvePendingMonsterMissile())
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, allyIndex, mock.CombatEvents[0].Target.Index)
	assert.False(t, mock.CombatEvents[0].Target.IsPartyMember)
	assert.Less(t, ally.CurrentHP(), hpBefore)
}

func TestMonsterRangedPoisonReplacesDamageAtImpact(t *testing.T) {
	for _, test := range []struct {
		name       string
		poisonRoll int
		wantPoison bool
	}{
		{name: "zero continues to damage", poisonRoll: 0, wantPoison: false},
		{name: "nonzero poisons", poisonRoll: 1, wantPoison: true},
		{name: "two poisons", poisonRoll: 2, wantPoison: true},
		{name: "three poisons", poisonRoll: 3, wantPoison: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			gs, mock, snake, snakeIndex, nearest := setupMonsterMissileTest(t, references.MonsterSnake, 3)
			hpBefore := gs.PartyState.GetCharacter(0).GetCurrentHp()
			launch, attacked := gs.prepareMonsterAttackWithRNG(snake, snakeIndex, nearest, newSequenceRNG(0, 29))
			require.True(t, attacked)
			require.NotNil(t, launch)
			seed := seedForFirstIntnValue(t, gs, monsterSpecialEffectRollRange, test.poisonRoll)
			gs.SetRandomSeed(seed)

			require.True(t, gs.ResolvePendingMonsterMissile())

			require.Len(t, mock.CombatEvents, 1)
			event := mock.CombatEvents[0]
			assert.True(t, event.Diagnostics.PoisonRolled)
			assert.Equal(t, test.poisonRoll, event.Diagnostics.PoisonRoll)
			assert.Equal(t, test.wantPoison, gs.PartyState.GetCharacter(0).IsPoisoned())
			if test.wantPoison {
				assert.Equal(t, CombatFeedbackPoison, event.Result)
				assert.Equal(t, hpBefore, gs.PartyState.GetCharacter(0).GetCurrentHp())
			} else {
				assert.NotEqual(t, CombatFeedbackPoison, event.Result)
				assert.LessOrEqual(t, gs.PartyState.GetCharacter(0).GetCurrentHp(), hpBefore)
			}
		})
	}
}

func TestMonsterMissileAmuletOrderingNegationAndExactOnce(t *testing.T) {
	gs, mock, daemon, daemonIndex, nearest := setupMonsterMissileTest(t, references.MonsterDevil, 4)
	gs.PartyState.GetCharacter(0).SetEquippedItem(references.SlotAmulet, references.AmuletOfTurning)
	rng := newSequenceRNG(127, 0, 1, 2)
	hpBefore := gs.PartyState.GetCharacter(0).GetCurrentHp()

	launch, attacked := gs.prepareMonsterAttackWithRNG(daemon, daemonIndex, nearest, rng)

	require.True(t, attacked)
	require.NotNil(t, launch)
	assert.Equal(t, 4, rng.index, "amulet, missile decision, and redirect rolls must retain original order")
	assert.NotEqual(t, nearest.identity, CombatTurnIdentity{})
	assert.Equal(t, hpBefore, gs.PartyState.GetCharacter(0).GetCurrentHp())
	require.True(t, gs.ResolvePendingMonsterMissile())
	assert.False(t, gs.ResolvePendingMonsterMissile())
	require.Len(t, mock.CombatEvents, 1)
	assert.True(t, mock.CombatEvents[0].Diagnostics.ForcedMiss)
	assert.Equal(t, launch.Impact, mock.CombatEvents[0].To)

	gs2, mock2, daemon2, daemonIndex2, nearest2 := setupMonsterMissileTest(t, references.MonsterDevil, 4)
	gs2.ActivateNegateMagicEffect(3)
	launch, attacked = gs2.prepareMonsterAttackWithRNG(daemon2, daemonIndex2, nearest2, newSequenceRNG(0))
	assert.False(t, attacked)
	assert.Nil(t, launch)
	assert.Empty(t, mock2.CombatEvents)
}

func TestMonsterMissilePoisonDoesNotPassBlockOrForcedMiss(t *testing.T) {
	t.Run("blocked", func(t *testing.T) {
		gs, mock, snake, snakeIndex, nearest := setupMonsterMissileTest(t, references.MonsterSnake, 3)
		setBlockedCombatMissileMap(gs, references.Position{X: 4, Y: 5})
		launch, attacked := gs.prepareMonsterAttackWithRNG(snake, snakeIndex, nearest, newSequenceRNG(0, 29))
		require.True(t, attacked)
		require.True(t, launch.Blocked)

		require.True(t, gs.ResolvePendingMonsterMissile())

		require.Len(t, mock.CombatEvents, 1)
		assert.False(t, mock.CombatEvents[0].Diagnostics.PoisonRolled)
		assert.False(t, gs.PartyState.GetCharacter(0).IsPoisoned())
	})

	t.Run("forced miss to empty square", func(t *testing.T) {
		gs, mock, snake, snakeIndex, nearest := setupMonsterMissileTest(t, references.MonsterSnake, 3)
		snake.enemyRef.EnemyAbilities[references.RangedMagic] = true
		gs.PartyState.GetCharacter(0).SetEquippedItem(references.SlotAmulet, references.AmuletOfTurning)
		launch, attacked := gs.prepareMonsterAttackWithRNG(
			snake, snakeIndex, nearest, newSequenceRNG(127, 0, 1, 2),
		)
		require.True(t, attacked)
		require.NotNil(t, launch)
		assert.NotEqual(t, gs.CombatScene.PartyPositions[0], launch.Impact)

		require.True(t, gs.ResolvePendingMonsterMissile())

		require.Len(t, mock.CombatEvents, 1)
		assert.True(t, mock.CombatEvents[0].Diagnostics.ForcedMiss)
		assert.False(t, mock.CombatEvents[0].Diagnostics.PoisonRolled)
		assert.False(t, gs.PartyState.GetCharacter(0).IsPoisoned())
	})
}

func TestMonsterAmuletRollOccursBeforeRangeAndDoesNotChangeAdjacentMelee(t *testing.T) {
	gs, mock, daemon, daemonIndex, nearest := setupMonsterMissileTest(t, references.MonsterDevil, 1)
	gs.PartyState.GetCharacter(0).SetEquippedItem(references.SlotAmulet, references.AmuletOfTurning)
	rng := newSequenceRNG(127)

	launch, attacked := gs.prepareMonsterAttackWithRNG(daemon, daemonIndex, nearest, rng)

	assert.True(t, attacked)
	assert.Nil(t, launch)
	assert.Equal(t, 1, rng.index)
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, CombatEventAttack, mock.CombatEvents[0].Kind)

	gs2, mock2, daemon2, daemonIndex2, nearest2 := setupMonsterMissileTest(t, references.MonsterDevil, 5)
	daemon2.enemyRef.AttackRange = 2
	gs2.PartyState.GetCharacter(0).SetEquippedItem(references.SlotAmulet, references.AmuletOfTurning)
	rng = newSequenceRNG(127)
	launch, attacked = gs2.prepareMonsterAttackWithRNG(daemon2, daemonIndex2, nearest2, rng)
	assert.False(t, attacked)
	assert.Nil(t, launch)
	assert.Equal(t, 1, rng.index, "turning roll must precede the range rejection")
	assert.Empty(t, mock2.CombatEvents)
}

func TestPlanMonsterTurnDoesNotGuaranteeRangedOpportunity(t *testing.T) {
	gs, mock, monster, monsterIndex, _ := setupMonsterMissileTest(t, references.MonsterOrc, 4)
	entry := &TurnQueueEntry{Combatant: monster, EntityIndex: monsterIndex}
	positionBefore := monster.Position()
	hpBefore := gs.PartyState.GetCharacter(0).GetCurrentHp()

	plan := gs.planMonsterTurn(entry)

	assert.Equal(t, MonsterTurnSeek, plan.Action)
	assert.False(t, plan.HasTarget)
	assert.Equal(t, NoTargetPC, plan.TargetPCIndex)
	assert.Equal(t, positionBefore, monster.Position())
	assert.Equal(t, hpBefore, gs.PartyState.GetCharacter(0).GetCurrentHp())
	assert.Empty(t, mock.CombatEvents)
}

func TestMonsterMissileResolvesBeforeAttackerStandingField(t *testing.T) {
	gs, mock, monster, monsterIndex, _ := setupMonsterMissileTest(t, references.MonsterOrc, 4)
	require.True(t, gs.CombatScene.AddField(NewCombatField(CombatFieldFire, monster.Position())))
	entry := &TurnQueueEntry{Combatant: monster, EntityIndex: monsterIndex}
	plan := gs.planMonsterTurn(entry)
	gs.SetRandomSeed(seedForFirstIntnValue(t, gs, monsterSpecialRollRange, 127))
	mock.Reset()

	result := gs.executeMonsterTurn(entry, plan)

	require.NotNil(t, result.ProjectileLaunch)
	assert.Empty(t, mock.CombatEvents, "standing fields wait until the committed missile resolves")
	require.True(t, gs.ResolvePendingMonsterMissile())
	require.GreaterOrEqual(t, len(mock.CombatEvents), 2)
	assert.Equal(t, CombatEventRangedAttack, mock.CombatEvents[0].Kind)
	assert.Equal(t, CombatEventFieldTriggered, mock.CombatEvents[1].Kind)
}

func seedForFirstIntnValue(t *testing.T, gs *GameState, maximum, wanted int) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 10000; seed++ {
		gs.SetRandomSeed(seed)
		if gs.GetRNG().Intn(maximum) == wanted {
			return seed
		}
	}
	t.Fatalf("no seed produced Intn(%d) == %d", maximum, wanted)

	return 0
}
