package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatTargetDefenseAppliesIdentityAndDurationOverridesWithoutRNG(t *testing.T) {
	gs, _ := setupControlledMonsterTest(t)
	mimic, mimicIndex := addControlledTestMonster(
		gs, references.MonsterMimic, references.Position{X: 4, Y: 4}, CombatMonsterLoyaltyParty,
	)
	orc, orcIndex := addControlledTestMonster(
		gs, references.MonsterOrc, references.Position{X: 5, Y: 4}, CombatMonsterLoyaltyParty,
	)
	mimicCurrentDex := mimic.GetDexterity()
	orcCurrentDex := orc.GetDexterity()
	callsBefore := gs.GetRNG().Calls()

	mimicDefense := gs.combatTargetDefense(CombatTurnIdentity{EntityIndex: mimicIndex})
	assert.Equal(t, mimic.BaseDex(), mimicDefense.BaseDex)
	assert.Equal(t, 1, mimicDefense.EffectiveDex)
	assert.Equal(t, mimicCurrentDex, mimic.GetDexterity())

	orcIdentity := CombatTurnIdentity{EntityIndex: orcIndex}
	ordinaryDefense := gs.combatTargetDefense(orcIdentity)
	assert.Equal(t, orc.BaseDex(), ordinaryDefense.BaseDex)
	assert.Equal(t, orc.CurrentDex(), ordinaryDefense.EffectiveDex)
	require.True(t, gs.ActivateTimeStopEffect(1))
	assert.Equal(t, 1, gs.combatTargetDefense(orcIdentity).EffectiveDex,
		"An Tym target defense applies to party-loyal monsters")
	assert.Equal(t, orcCurrentDex, orc.GetDexterity())
	gs.ClearDurationEffect()
	assert.Equal(t, orc.CurrentDex(), gs.combatTargetDefense(orcIdentity).EffectiveDex)
	assert.Equal(t, callsBefore, gs.GetRNG().Calls())
}

func TestPhysicalHitDiagnosticsSeparateMimicBaseAndEffectiveDex(t *testing.T) {
	gs, mock, mimic := setupCombatWithMonster(t, references.MonsterMimic, references.Down, 1)
	pcIndex := gs.CombatScene.GetActiveCombatantIndex()
	character := gs.PartyState.GetCharacter(pcIndex)
	character.Dexterity = 30
	character.SetEquippedItem(references.SlotWeapon, references.LongSword)

	gs.performSingleAttack(
		character, references.LongSword, references.SlotWeapon, mimic, 0, pcIndex,
		newSequenceRNG(29, 14, 0),
	)

	require.NotEmpty(t, mock.CombatEvents)
	diagnostics := mock.CombatEvents[0].Diagnostics
	assert.Equal(t, mimic.BaseDex(), diagnostics.TargetBaseDex)
	assert.Equal(t, 1, diagnostics.Hit.DefenderEffectiveDex)
}

func TestSuccessfulMonsterHitUsesTheftPoisonGazerDamageOrder(t *testing.T) {
	gs, _ := setupControlledMonsterTest(t)
	target := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
	character := gs.PartyState.GetCharacter(0)
	ref := (*gs.GameReferences.EnemyReferences)[references.MonsterGremlin]
	ref.EnemyAbilities = map[references.EnemyAbility]bool{
		references.StealsFood:    true,
		references.PoisonAtRange: true,
	}
	attacker := NewCombatMonster(references.MonsterGremlin, &ref, references.Position{}, newFixedRNG(1))

	t.Run("theft replaces every later effect", func(t *testing.T) {
		gs.PartyState.Inventory.Provisions.Food.Set(3)
		character.SetStatus(party_state.Good)
		rng := newSequenceRNG(1, 3)
		diagnostics := CombatDiagnostics{}

		result := gs.resolveSuccessfulMonsterHit(attacker, target, true, rng, &diagnostics)

		assert.True(t, result.replacedDamage)
		assert.False(t, result.damageSelected)
		assert.Equal(t, CombatFeedbackTheft, result.feedback)
		assert.Zero(t, gs.PartyState.Inventory.Provisions.Food.Get())
		assert.Equal(t, 3, diagnostics.FoodStolen)
		assert.False(t, diagnostics.PoisonRolled)
		assert.Equal(t, 1, rng.index)
	})

	t.Run("failed theft proceeds to poison", func(t *testing.T) {
		gs.PartyState.Inventory.Provisions.Food.Set(10)
		character.SetStatus(party_state.Good)
		rng := newSequenceRNG(0, 2, 19)
		diagnostics := CombatDiagnostics{}

		result := gs.resolveSuccessfulMonsterHit(attacker, target, true, rng, &diagnostics)

		assert.Equal(t, CombatFeedbackPoison, result.feedback)
		assert.True(t, character.IsPoisoned())
		assert.Equal(t, 10, int(gs.PartyState.Inventory.Provisions.Food.Get()))
		assert.Equal(t, 0, diagnostics.TheftRoll)
		assert.Equal(t, 2, diagnostics.PoisonRoll)
		assert.False(t, diagnostics.PoisonFallbackRolled)
		assert.Equal(t, 2, rng.index)
	})

	t.Run("empty food skips theft roll", func(t *testing.T) {
		gs.PartyState.Inventory.Provisions.Food.Set(0)
		character.SetStatus(party_state.Good)
		rng := newSequenceRNG(3)
		diagnostics := CombatDiagnostics{}

		result := gs.resolveSuccessfulMonsterHit(attacker, target, true, rng, &diagnostics)

		assert.Equal(t, CombatFeedbackPoison, result.feedback)
		assert.False(t, diagnostics.TheftRolled)
		assert.Equal(t, 3, diagnostics.PoisonRoll)
		assert.Equal(t, 1, rng.index)
	})

	t.Run("ordinary attacker selects no replacement", func(t *testing.T) {
		ordinaryRef := (*gs.GameReferences.EnemyReferences)[references.MonsterOrc]
		ordinary := NewCombatMonster(references.MonsterOrc, &ordinaryRef, references.Position{}, newFixedRNG(1))
		rng := newSequenceRNG(3)
		diagnostics := CombatDiagnostics{}

		result := gs.resolveSuccessfulMonsterHit(ordinary, target, true, rng, &diagnostics)

		assert.False(t, result.replacedDamage)
		assert.Equal(t, 0, rng.index)
	})

	t.Run("failed Gremlin theft selects ordinary damage", func(t *testing.T) {
		gremlinRef := (*gs.GameReferences.EnemyReferences)[references.MonsterGremlin]
		gremlin := NewCombatMonster(references.MonsterGremlin, &gremlinRef, references.Position{}, newFixedRNG(1))
		gs.PartyState.Inventory.Provisions.Food.Set(10)
		rng := newSequenceRNG(0)
		diagnostics := CombatDiagnostics{}

		result := gs.resolveSuccessfulMonsterHit(gremlin, target, true, rng, &diagnostics)

		assert.False(t, result.replacedDamage)
		assert.True(t, diagnostics.TheftRolled)
		assert.Equal(t, 1, rng.index)
	})

	t.Run("failed poison precedes Gazer sleep and ordinary damage", func(t *testing.T) {
		gazerRef := (*gs.GameReferences.EnemyReferences)[references.MonsterGazer]
		gazerRef.EnemyAbilities = map[references.EnemyAbility]bool{references.PoisonAtRange: true}
		gazer := NewCombatMonster(references.MonsterGazer, &gazerRef, references.Position{}, newFixedRNG(1))
		character.SetStatus(party_state.Good)
		gs.CombatScene.setCombatPCSleeping(target, false)
		rng := newSequenceRNG(0, 19)
		diagnostics := CombatDiagnostics{}

		result := gs.resolveSuccessfulMonsterHit(gazer, target, false, rng, &diagnostics)

		assert.Equal(t, CombatFeedbackSleep, result.feedback)
		assert.True(t, gs.CombatScene.isCombatPCSleeping(target))
		assert.Equal(t, 1, rng.index)

		rng = newSequenceRNG(0, 19)
		diagnostics = CombatDiagnostics{}
		result = gs.resolveSuccessfulMonsterHit(gazer, target, false, rng, &diagnostics)
		assert.False(t, result.replacedDamage, "already sleeping reaches ordinary damage")
		assert.Equal(t, 1, rng.index)
	})
}

func TestMonsterPoisonFallbackPreservesEveryIneligibleStatus(t *testing.T) {
	gs, _ := setupControlledMonsterTest(t)
	ref := (*gs.GameReferences.EnemyReferences)[references.MonsterSnake]
	ref.EnemyAbilities = map[references.EnemyAbility]bool{references.PoisonAtRange: true}
	attacker := NewCombatMonster(references.MonsterSnake, &ref, references.Position{}, newFixedRNG(1))
	target := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
	character := gs.PartyState.GetCharacter(0)

	for _, status := range []party_state.CharacterStatus{
		party_state.Poisoned,
		party_state.Sleep,
		party_state.Charmed,
		party_state.Dead,
	} {
		t.Run(string(status), func(t *testing.T) {
			gs.CombatScene.setCombatPCSleeping(target, status == party_state.Sleep)
			character.SetStatus(status)
			rng := newSequenceRNG(1, 19)
			diagnostics := CombatDiagnostics{}

			result := gs.resolveSuccessfulMonsterHit(attacker, target, false, rng, &diagnostics)

			assert.True(t, result.damageSelected)
			assert.Equal(t, 19, result.damage)
			assert.Equal(t, status, character.GetStatus())
			assert.Equal(t, 19, diagnostics.FinalDamage)
			assert.Equal(t, 19, diagnostics.PoisonFallbackDamage)
			assert.Equal(t, 2, rng.index)
		})
	}

	monsterRef := (*gs.GameReferences.EnemyReferences)[references.MonsterOrc]
	monster := NewCombatMonster(references.MonsterOrc, &monsterRef, references.Position{X: 4, Y: 4}, newFixedRNG(1))
	monster.SetPoisoned(true)
	monster.SetAsleep(true)
	monsterIndex := gs.CombatScene.AddMonsterToQueue(monster)
	rng := newSequenceRNG(1, 7)
	diagnostics := CombatDiagnostics{}
	result := gs.resolveSuccessfulMonsterHit(
		attacker, CombatTurnIdentity{EntityIndex: monsterIndex}, false, rng, &diagnostics,
	)
	assert.True(t, result.damageSelected)
	assert.Equal(t, 7, result.damage)
	assert.True(t, monster.IsPoisoned())
	assert.True(t, monster.IsAsleep())

	zeroRNG := newSequenceRNG(1, 0)
	zeroDiagnostics := CombatDiagnostics{}
	zeroResult := gs.resolveSuccessfulMonsterHit(
		attacker, CombatTurnIdentity{EntityIndex: monsterIndex}, false, zeroRNG, &zeroDiagnostics,
	)
	assert.True(t, zeroResult.damageSelected)
	assert.Zero(t, zeroResult.damage)
	assert.Equal(t, CombatFeedbackGraze, zeroResult.feedback)
	assert.True(t, zeroDiagnostics.PoisonFallbackRolled)
	assert.Equal(t, 2, zeroRNG.index)
}

func TestControlledGazerMeleeUsesSharedSleepReplacement(t *testing.T) {
	gs, mock := setupControlledMonsterTest(t)
	gazer, gazerIndex := addControlledTestMonster(
		gs, references.MonsterGazer, references.Position{X: 5, Y: 5}, CombatMonsterLoyaltyParty,
	)
	gazer.currentDex = 99
	target, targetIndex := addControlledTestMonster(
		gs, references.MonsterOrc, references.Position{X: 6, Y: 5}, CombatMonsterLoyaltyMonster,
	)
	authorizeControlledMonster(gs, gazerIndex)
	hpBefore := target.CurrentHP()
	gs.SetRandomSeed(seedForFirstIntnValue(t, gs, originalD30SourceValueCount, 60))
	callsBefore := gs.GetRNG().Calls()
	mock.Reset()

	gs.resolveControlledMonsterAttack(gazerIndex, target, targetIndex, target.Position())

	assert.Equal(t, []SoundEffect{SoundCombatSwing}, mock.SoundEffectsPlayed)
	assert.True(t, target.IsAsleep())
	assert.Equal(t, hpBefore, target.CurrentHP())
	assert.Equal(t, callsBefore+1, gs.GetRNG().Calls())
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, CombatFeedbackSleep, mock.CombatEvents[0].Result)
}

func TestControlledRangedDiagnosticsUseIdentityAwareEffectiveDex(t *testing.T) {
	for _, test := range []struct {
		name       string
		targetType references.MonsterType
		anTym      bool
	}{
		{name: "Mimic", targetType: references.MonsterMimic},
		{name: "An Tym", targetType: references.MonsterOrc, anTym: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			gs, mock := setupControlledMonsterTest(t)
			setOpenCombatMissileMap(gs)
			attacker, attackerIndex := addControlledTestMonster(
				gs, references.MonsterDevil, references.Position{X: 3, Y: 5}, CombatMonsterLoyaltyParty,
			)
			attacker.currentDex = 99
			target, targetIndex := addControlledTestMonster(
				gs, test.targetType, references.Position{X: 6, Y: 5}, CombatMonsterLoyaltyMonster,
			)
			if test.anTym {
				require.True(t, gs.ActivateTimeStopEffect(1))
			}
			authorizeControlledMonster(gs, attackerIndex)
			gs.SetRandomSeed(seedForFirstIntnValue(t, gs, originalD30SourceValueCount, 60))
			mock.Reset()

			require.True(t, gs.BeginCombatAttack())
			gs.CombatScene.AttackTargeting.Cursor = target.Position()
			require.Equal(t, CombatAttackSourcePending, gs.ConfirmCombatAttackTarget().Outcome)
			require.Equal(t, CombatAttackSequenceComplete, gs.ResolvePendingCombatAttack().Outcome)

			require.Len(t, mock.CombatEvents, 1)
			event := mock.CombatEvents[0]
			assert.Equal(t, targetIndex, event.Target.Index)
			assert.Equal(t, target.BaseDex(), event.Diagnostics.TargetBaseDex)
			assert.Equal(t, 1, event.Diagnostics.Hit.DefenderEffectiveDex)
			if test.anTym {
				gs.ClearDurationEffect()
				assert.Equal(t, target.CurrentDex(), gs.combatTargetDefense(CombatTurnIdentity{EntityIndex: targetIndex}).EffectiveDex)
			}
		})
	}
}

func TestMonsterMeleeMissConsumesNoEffectRoll(t *testing.T) {
	gs, mock, snake := setupCombatWithMonster(t, references.MonsterSnake, references.Down, 1)
	character := gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex())
	character.Dexterity = 99
	snake.currentDex = 1
	nearest := findNearestPC(snake.Position(), gs.CombatScene)
	gs.SetRandomSeed(seedForFirstIntnValue(t, gs, originalD30SourceValueCount, 0))
	callsBefore := gs.GetRNG().Calls()
	mock.Reset()

	require.True(t, gs.monsterMeleeAttack(snake, 0, nearest))

	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, CombatFeedbackMiss, mock.CombatEvents[0].Result)
	assert.False(t, mock.CombatEvents[0].Diagnostics.PoisonRolled)
	assert.Equal(t, callsBefore+1, gs.GetRNG().Calls())
	assert.Equal(t, party_state.Good, character.GetStatus())
}

func TestOrdinaryMonsterMeleeUsesFixedDamageWithoutDamageRNG(t *testing.T) {
	gs, mock, orc := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 1)
	character := gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex())
	character.SetEquippedItem(references.SlotArmour, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	orc.enemyRef.Damage = 20
	gs.SetRandomSeed(seedForFirstIntnValue(t, gs, originalD30SourceValueCount, 60))
	callsBefore := gs.GetRNG().Calls()
	hpBefore := character.GetCurrentHp()
	mock.Reset()

	require.True(t, gs.monsterMeleeAttack(orc, 0, findNearestPC(orc.Position(), gs.CombatScene)))

	assert.Equal(t, uint16(20), hpBefore-character.GetCurrentHp())
	assert.Equal(t, callsBefore+1, gs.GetRNG().Calls(), "only the hit check consumes RNG")
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, 20, mock.CombatEvents[0].Diagnostics.Damage.AttackRoll)
	assert.False(t, mock.CombatEvents[0].Diagnostics.Damage.AttackRolled)
}

func TestMonsterRating99ForcesPCDeathAbove99HP(t *testing.T) {
	gs, mock, orc := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 1)
	character := gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex())
	character.SetMaxHp(200)
	character.SetHp(200)
	character.SetStatus(party_state.Good)
	character.SetEquippedItem(references.SlotArmour, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	orc.enemyRef.Damage = combatInstantKillDamage
	gs.SetRandomSeed(seedForFirstIntnValue(t, gs, originalD30SourceValueCount, 60))
	mock.Reset()

	require.True(t, gs.monsterMeleeAttack(orc, 0, findNearestPC(orc.Position(), gs.CombatScene)))

	assert.True(t, character.IsDead())
	assert.Zero(t, character.GetCurrentHp())
	require.Len(t, mock.CombatEvents, 1)
	assert.True(t, mock.CombatEvents[0].Fatal)
	assert.Equal(t, 200, mock.CombatEvents[0].HPLost)
	assert.True(t, mock.CombatEvents[0].Diagnostics.Damage.InstantKill)
}

func TestControlledMonsterDamageUsesFixedRatingAndAppliesRating99(t *testing.T) {
	tests := []struct {
		name      string
		damage    int
		target    references.MonsterType
		immortal  bool
		wantHP    int
		wantFatal bool
		wantFinal int
	}{
		{name: "ordinary damage", damage: 20, target: references.MonsterTroll, wantHP: 180, wantFinal: 20},
		{name: "rating 99", damage: 99, target: references.MonsterTroll, wantFatal: true, wantFinal: 99},
		{name: "rating 99 against undead", damage: 99, target: references.MonsterSkeleton, wantHP: 151, wantFinal: 49},
		{name: "rating 99 against immortal", damage: 99, target: references.MonsterTroll, immortal: true, wantHP: 200},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, mock := setupControlledMonsterTest(t)
			attackerRef := &references.EnemyReference{ //nolint:exhaustruct // Source-derived damage fixture.
				MonsterType: references.MonsterOrc,
				Name:        "Controlled attacker",
				Damage:      test.damage,
				HitPoints:   20,
			}
			targetRef := &references.EnemyReference{ //nolint:exhaustruct // HP above 99 proves forced death.
				MonsterType: test.target,
				Name:        "Target",
				HitPoints:   200,
			}
			if test.immortal {
				targetRef.EnemyAbilities = map[references.EnemyAbility]bool{references.Immortal: true}
			}
			attacker := NewCombatMonster(
				references.MonsterOrc, attackerRef, references.Position{X: 5, Y: 5}, newFixedRNG(1),
			)
			attacker.SetLoyalty(CombatMonsterLoyaltyParty)
			attackerIndex := gs.CombatScene.AddMonsterToQueue(attacker)
			target := NewCombatMonster(
				test.target, targetRef, references.Position{X: 6, Y: 5}, newFixedRNG(1),
			)
			targetIndex := gs.CombatScene.AddMonsterToQueue(target)
			callsBefore := gs.GetRNG().Calls()
			mock.Reset()

			gs.resolveControlledMonsterHitAgainstMonster(
				attackerIndex, target, targetIndex, CombatDiagnostics{}, CombatEventAttack,
			)

			assert.Equal(t, test.wantHP, target.CurrentHP())
			assert.Equal(t, test.wantFatal, target.IsDead())
			if !test.wantFatal {
				assert.Equal(t, callsBefore, gs.GetRNG().Calls(), "unarmored controlled damage consumes no RNG")
			}
			require.Len(t, mock.CombatEvents, 1)
			assert.Equal(t, test.wantFinal, mock.CombatEvents[0].Diagnostics.FinalDamage)
			assert.Equal(t, test.wantFatal, mock.CombatEvents[0].Fatal)
		})
	}
}

func TestGremlinMeleeTheftReplacesDamageForPCAndControlledMonster(t *testing.T) {
	t.Run("PC target", func(t *testing.T) {
		gs, mock, gremlin := setupCombatWithMonster(t, references.MonsterGremlin, references.Down, 1)
		gremlin.currentDex = 99
		character := gs.PartyState.GetCharacter(gs.CombatScene.GetActiveCombatantIndex())
		hpBefore := character.GetCurrentHp()
		gs.PartyState.Inventory.Provisions.Food.Set(10)
		seed := seedForMonsterHitEffects(t, gs, true, false, false)
		gs.SetRandomSeed(seed)
		nearest := findNearestPC(gremlin.Position(), gs.CombatScene)
		mock.Reset()

		require.True(t, gs.monsterMeleeAttack(gremlin, 0, nearest))

		assert.Equal(t, hpBefore, character.GetCurrentHp())
		assert.Equal(t, uint16(5), gs.PartyState.Inventory.Provisions.Food.Get())
		require.Len(t, mock.CombatEvents, 1)
		assert.Equal(t, CombatFeedbackTheft, mock.CombatEvents[0].Result)
		assert.Equal(t, 5, mock.CombatEvents[0].Diagnostics.FoodStolen)
	})

	t.Run("controlled monster target", func(t *testing.T) {
		gs, mock := setupControlledMonsterTest(t)
		gremlin, gremlinIndex := addControlledTestMonster(
			gs, references.MonsterGremlin, references.Position{X: 5, Y: 5}, CombatMonsterLoyaltyMonster,
		)
		gremlin.currentDex = 99
		target, targetIndex := addControlledTestMonster(
			gs, references.MonsterOrc, references.Position{X: 6, Y: 5}, CombatMonsterLoyaltyParty,
		)
		hpBefore := target.CurrentHP()
		gs.PartyState.Inventory.Provisions.Food.Set(4)
		gs.SetRandomSeed(seedForMonsterHitEffects(t, gs, true, false, false))
		mock.Reset()

		gs.resolveMonsterMeleeAgainstMonster(gremlin, gremlinIndex, target, targetIndex)

		assert.Equal(t, hpBefore, target.CurrentHP())
		assert.Zero(t, gs.PartyState.Inventory.Provisions.Food.Get())
		require.Len(t, mock.CombatEvents, 1)
		assert.Equal(t, CombatFeedbackTheft, mock.CombatEvents[0].Result)
	})
}

func TestGazerHitSleepsBeforeDamageInMeleeAndRange(t *testing.T) {
	t.Run("melee sleeps without extra RNG", func(t *testing.T) {
		gs, mock, gazer := setupCombatWithMonster(t, references.MonsterGazer, references.Down, 1)
		gazer.currentDex = 99
		identity := CombatTurnIdentity{IsPartyMember: true, EntityIndex: gs.CombatScene.GetActiveCombatantIndex()}
		character := gs.PartyState.GetCharacter(identity.EntityIndex)
		hpBefore := character.GetCurrentHp()
		gs.SetRandomSeed(seedForFirstIntnValue(t, gs, originalD30SourceValueCount, 60))
		callsBefore := gs.GetRNG().Calls()
		mock.Reset()

		require.True(t, gs.monsterMeleeAttack(gazer, 0, findNearestPC(gazer.Position(), gs.CombatScene)))

		assert.True(t, gs.CombatScene.isCombatPCSleeping(identity))
		assert.Equal(t, hpBefore, character.GetCurrentHp())
		assert.Equal(t, callsBefore+1, gs.GetRNG().Calls())
		require.Len(t, mock.CombatEvents, 1)
		assert.Equal(t, CombatFeedbackSleep, mock.CombatEvents[0].Result)
	})

	t.Run("already sleeping receives normal damage", func(t *testing.T) {
		gs, mock, gazer := setupCombatWithMonster(t, references.MonsterGazer, references.Down, 1)
		gazer.currentDex = 99
		identity := CombatTurnIdentity{IsPartyMember: true, EntityIndex: gs.CombatScene.GetActiveCombatantIndex()}
		character := gs.PartyState.GetCharacter(identity.EntityIndex)
		character.SetEquippedItem(references.SlotArmour, references.NoEquipment)
		character.SetEquippedItem(references.SlotShield, references.NoEquipment)
		character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
		gs.setCombatPCSleeping(identity, true)
		hpBefore := character.GetCurrentHp()
		gs.SetRandomSeed(seedForFirstIntnValue(t, gs, originalD30SourceValueCount, 60))
		mock.Reset()

		require.True(t, gs.monsterMeleeAttack(gazer, 0, findNearestPC(gazer.Position(), gs.CombatScene)))

		assert.Less(t, character.GetCurrentHp(), hpBefore)
		assert.False(t, gs.CombatScene.isCombatPCSleeping(identity))
		require.Len(t, mock.CombatEvents, 2)
		assert.Equal(t, CombatFeedbackDamage, mock.CombatEvents[0].Result)
		assert.Equal(t, CombatConditionWoke, mock.CombatEvents[1].Condition)
	})

	t.Run("ranged sleeps at impact without resolution RNG", func(t *testing.T) {
		gs, mock, gazer, gazerIndex, nearest := setupMonsterMissileTest(t, references.MonsterGazer, 3)
		if gs == nil {
			return
		}
		identity := nearest.identity
		character := gs.PartyState.GetCharacter(identity.EntityIndex)
		hpBefore := character.GetCurrentHp()
		launch, attacked := gs.prepareMonsterAttackWithRNG(
			gazer, gazerIndex, nearest, newSequenceRNG(0, 29),
		)
		require.True(t, attacked)
		require.NotNil(t, launch)
		gs.SetRandomSeed(1)
		callsBefore := gs.GetRNG().Calls()

		require.True(t, gs.ResolvePendingMonsterMissile())

		assert.True(t, gs.CombatScene.isCombatPCSleeping(identity))
		assert.Equal(t, hpBefore, character.GetCurrentHp())
		assert.Equal(t, callsBefore, gs.GetRNG().Calls())
		require.Len(t, mock.CombatEvents, 1)
		assert.Equal(t, CombatEventRangedAttack, mock.CombatEvents[0].Kind)
		assert.Equal(t, CombatFeedbackSleep, mock.CombatEvents[0].Result)
	})
}

func TestControlledRangedPoisonFallbackIsSelectedFatalDamage(t *testing.T) {
	gs, mock := setupControlledMonsterTest(t)
	setOpenCombatMissileMap(gs)
	attackerRef := &references.EnemyReference{ //nolint:exhaustruct // Focused deterministic combat fixture.
		MonsterType:          references.MonsterSnake,
		Name:                 "Controlled Snake",
		Dexterity:            99,
		Damage:               99,
		HitPoints:            20,
		AttackRange:          3,
		LargeMapMissileRange: 3,
		LargeMapMissile:      references.MissileGreen,
		EnemyAbilities:       map[references.EnemyAbility]bool{references.PoisonAtRange: true},
	}
	targetRef := &references.EnemyReference{ //nolint:exhaustruct // Focused deterministic combat fixture.
		MonsterType: references.MonsterOrc,
		Name:        "Target Orc",
		Dexterity:   1,
		HitPoints:   5,
	}
	attacker := NewCombatMonster(
		references.MonsterSnake, attackerRef, references.Position{X: 3, Y: 5}, newFixedRNG(1),
	)
	attacker.SetLoyalty(CombatMonsterLoyaltyParty)
	attackerIndex := gs.CombatScene.AddMonsterToQueue(attacker)
	target := NewCombatMonster(
		references.MonsterOrc, targetRef, references.Position{X: 6, Y: 5}, newFixedRNG(1),
	)
	targetIndex := gs.CombatScene.AddMonsterToQueue(target)
	authorizeControlledMonster(gs, attackerIndex)
	experienceBefore := gs.PartyState.GetAvatar().Exp
	gs.SetRandomSeed(seedForMonsterHitEffects(t, gs, false, true, true))
	mock.Reset()

	require.True(t, gs.BeginCombatAttack())
	gs.CombatScene.AttackTargeting.Cursor = target.Position()
	require.Equal(t, CombatAttackSourcePending, gs.ConfirmCombatAttackTarget().Outcome)
	require.Equal(t, CombatAttackSequenceComplete, gs.ResolvePendingCombatAttack().Outcome)

	require.True(t, target.IsDead())
	require.Len(t, mock.CombatEvents, 1)
	event := mock.CombatEvents[0]
	assert.Equal(t, CombatEventRangedAttack, event.Kind)
	assert.Equal(t, targetIndex, event.Target.Index)
	assert.True(t, event.Fatal)
	assert.Equal(t, targetRef.HitPoints, event.HPLost)
	assert.GreaterOrEqual(t, event.Diagnostics.PoisonFallbackDamage, event.HPLost)
	assert.LessOrEqual(t, event.Diagnostics.PoisonFallbackDamage, monsterPoisonDamageRange-1)
	assert.Equal(t, experienceBefore, gs.PartyState.GetAvatar().Exp,
		"party-loyal monsters preserve the established no-XP attribution policy")
}

func seedForMonsterHitEffects(
	t *testing.T,
	gs *GameState,
	wantTheft bool,
	wantPoison bool,
	wantFatalFallback bool,
) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 10000; seed++ {
		gs.SetRandomSeed(seed)
		gs.GetRNG().Intn(originalD30SourceValueCount)
		if wantTheft {
			if gs.GetRNG().Intn(monsterSpecialEffectRollRange) != 0 {
				return seed
			}

			continue
		}
		if wantPoison && gs.GetRNG().Intn(monsterSpecialEffectRollRange) == 0 {
			continue
		}
		if wantFatalFallback && gs.GetRNG().Intn(monsterPoisonDamageRange) < 5 {
			continue
		}

		return seed
	}
	t.Fatal("no deterministic monster hit seed found")

	return 0
}
