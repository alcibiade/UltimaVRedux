package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatMonsterInjuryStrictTruncatedQuarterBoundaries(t *testing.T) {
	for _, test := range []struct {
		baseline int
		bands    []CombatInjurySeverity // Positive HP, in ascending order.
	}{
		{20, []CombatInjurySeverity{
			CombatInjuryCritical, CombatInjuryCritical, CombatInjuryCritical, CombatInjuryCritical,
			CombatInjuryHeavilyWounded, CombatInjuryHeavilyWounded, CombatInjuryHeavilyWounded, CombatInjuryHeavilyWounded, CombatInjuryHeavilyWounded,
			CombatInjuryLightlyWounded, CombatInjuryLightlyWounded, CombatInjuryLightlyWounded, CombatInjuryLightlyWounded, CombatInjuryLightlyWounded,
			CombatInjuryBarelyWounded, CombatInjuryBarelyWounded, CombatInjuryBarelyWounded, CombatInjuryBarelyWounded, CombatInjuryBarelyWounded, CombatInjuryBarelyWounded,
		}},
		{11, []CombatInjurySeverity{CombatInjuryCritical, CombatInjuryHeavilyWounded, CombatInjuryHeavilyWounded,
			CombatInjuryLightlyWounded, CombatInjuryLightlyWounded, CombatInjuryBarelyWounded, CombatInjuryBarelyWounded,
			CombatInjuryBarelyWounded, CombatInjuryBarelyWounded, CombatInjuryBarelyWounded, CombatInjuryBarelyWounded}},
		{7, []CombatInjurySeverity{CombatInjuryHeavilyWounded, CombatInjuryLightlyWounded, CombatInjuryBarelyWounded,
			CombatInjuryBarelyWounded, CombatInjuryBarelyWounded, CombatInjuryBarelyWounded, CombatInjuryBarelyWounded}},
		{4, []CombatInjurySeverity{CombatInjuryHeavilyWounded, CombatInjuryLightlyWounded, CombatInjuryBarelyWounded, CombatInjuryBarelyWounded}},
		{3, []CombatInjurySeverity{CombatInjuryBarelyWounded, CombatInjuryBarelyWounded, CombatInjuryBarelyWounded}},
		{2, []CombatInjurySeverity{CombatInjuryBarelyWounded, CombatInjuryBarelyWounded}},
		{1, []CombatInjurySeverity{CombatInjuryBarelyWounded}},
	} {
		t.Run(fmt.Sprintf("baseline%d", test.baseline), func(t *testing.T) {
			for index, want := range test.bands {
				assert.Equal(t, want, ClassifyCombatMonsterInjury(index+1, test.baseline), "remaining HP %d", index+1)
			}
			assert.Equal(t, CombatInjuryBarelyWounded, ClassifyCombatMonsterInjury(test.baseline+10, test.baseline))
			assert.Equal(t, CombatInjuryNone, ClassifyCombatMonsterInjury(0, test.baseline))
			assert.Equal(t, CombatInjuryNone, ClassifyCombatMonsterInjury(-1, test.baseline))
		})
	}
	assert.Zero(t, CombatInjuryNone)
	assert.Equal(t, CombatInjuryNone, ClassifyCombatMonsterInjury(1, 0))
	assert.Equal(t, CombatInjuryNone, ClassifyCombatMonsterInjury(1, -1))
}

func TestCombatMonsterInjuryCaptureIsPureAndDetached(t *testing.T) {
	state, _ := newSyntheticCombatConeState(t)
	monster, index := addSyntheticConeMonster(state, references.MonsterOrc, references.Position{X: 5, Y: 3}, 10)
	monster.enemyRef.HitPoints = 11
	monster.SetCurrentHP(3)
	monster.SetLoyalty(CombatMonsterLoyaltyParty)
	monster.SetFleeing(false)
	monster.SetAsleep(true)
	monster.SetPoisoned(true)
	calls := state.GetRNG().Calls()
	identity := CombatTurnIdentity{EntityIndex: index}
	injury := state.captureCombatMonsterInjury(identity, 1)
	require.Equal(t, CombatInjuryHeavilyWounded, injury.Severity)
	assert.True(t, injury.Disclosed)
	assert.Equal(t, CombatMonsterLoyaltyParty, injury.Loyalty)
	assert.Equal(t, monster.CombatantID(), injury.CombatantID)
	assert.Equal(t, calls, state.GetRNG().Calls(), "middle-band feedback must not perform the fleeing roll")
	assert.False(t, monster.IsFleeing())
	assert.True(t, monster.IsAsleep())
	assert.True(t, monster.IsPoisoned())
	assert.Equal(t, 3, monster.CurrentHP())
	assert.Equal(t, CombatInjuryNone, state.captureCombatMonsterInjury(identity, 0).Severity)
	assert.Equal(t, CombatInjuryNone, state.captureCombatMonsterInjury(identity, -1).Severity)
	for _, pc := range []CombatTurnIdentity{{IsPartyMember: true}, {IsPartyMember: true, Slot: 1}} {
		assert.Equal(t, CombatMonsterInjury{}, state.captureCombatMonsterInjury(pc, 1), "PCs, including hostile PCs and clones, have no monster band")
	}
	monster.SetInvisible(true)
	hidden := state.captureCombatMonsterInjury(identity, 1)
	assert.False(t, hidden.Disclosed)
	monster.SetInvisible(false)
	monster.Heal(100)
	assert.Equal(t, CombatInjuryHeavilyWounded, injury.Severity)
	assert.False(t, hidden.Disclosed)
	monster.TakeDamage(100)
	assert.Equal(t, CombatInjuryNone, state.captureCombatMonsterInjury(identity, 11).Severity)
	assert.Equal(t, CombatInjuryHeavilyWounded, injury.Severity)
	monster.monsterType = references.MonsterTroll
	monster.combatantID++
	assert.Equal(t, references.MonsterOrc, injury.MonsterType)
	assert.NotEqual(t, monster.CombatantID(), injury.CombatantID)
}

func TestCombatMonsterInjuryDamagePublishersCaptureBeforePresentation(t *testing.T) {
	for _, kind := range []CombatEventKind{CombatEventAttack, CombatEventRangedAttack, CombatEventSpell} {
		t.Run(fmt.Sprint(kind), func(t *testing.T) {
			state, mock := newSyntheticCombatConeState(t)
			monster, index := addSyntheticConeMonster(state, references.MonsterOrc, references.Position{X: 5, Y: 3}, 10)
			monster.enemyRef.HitPoints = 20
			monster.SetCurrentHP(12)
			// A callback can observe or mutate later state; the event must already
			// own the 5-HP band before presentation starts.
			state.SystemCallbacks.Visual.ShowCombatImpact = func(feedback CombatAttackFeedback) {
				assert.Equal(t, 7, feedback.HPLost)
				monster.Heal(100)
			}
			if kind == CombatEventSpell {
				state.applyCombatProjectileDamage(pendingCombatSpell{
					spell: references.VasFlam, caster: CombatTurnIdentity{IsPartyMember: true},
					target: CombatTurnIdentity{EntityIndex: index}, targetPosition: monster.Position(),
					hit: CombatHitResult{Outcome: CombatHitNormal},
				}, monster, CombatDamageResult{Damage: 7}, 7)
			} else {
				state.applyCombatDamageToMonsterWithEventKind(monster, index, 7, false, state.PartyState.GetCharacter(0), 0,
					CombatDiagnostics{Source: references.LongSword}, kind)
			}
			require.Len(t, mock.CombatEvents, 1)
			event := mock.CombatEvents[0]
			assert.Equal(t, kind, event.Kind)
			assert.Equal(t, 7, event.HPLost)
			assert.Equal(t, CombatInjuryHeavilyWounded, event.MonsterInjury.Severity)
			assert.Equal(t, 20, monster.CurrentHP())
		})
	}
}

func TestCombatMonsterInjuryFieldDamageHasOneReportAndUnchangedRNG(t *testing.T) {
	for _, kind := range []CombatFieldKind{CombatFieldFire, CombatFieldPoison} {
		for _, direct := range []bool{false, true} {
			t.Run(fmt.Sprintf("kind%d/direct%t", kind, direct), func(t *testing.T) {
				state, mock := newSyntheticCombatConeState(t)
				position := references.Position{X: 5, Y: 3}
				monster, index := addSyntheticConeMonster(state, references.MonsterGuard, position, 10)
				monster.SetCurrentHP(55)
				monster.SetFleeing(true)
				maximum := 21
				if kind == CombatFieldFire && !direct {
					maximum = 11
				}
				seed := areaSpellSeed(t, func(rng *deterministic_random.Source) bool { return rng.Intn(maximum) == 7 })
				state.SetRandomSeed(seed)
				oracle := deterministic_random.New(seed)
				oracle.Intn(maximum)
				if direct && kind == CombatFieldFire {
					oracle.Intn(1) // Direct magic still performs its armor draw.
				}
				state.applyCombatFieldToMonster(NewCombatField(kind, position), monster, index, 0, direct)
				require.Len(t, mock.CombatEvents, 1)
				event := mock.CombatEvents[0]
				assert.Equal(t, CombatEventFieldTriggered, event.Kind)
				assert.Equal(t, 7, event.HPLost)
				assert.Equal(t, 48, monster.CurrentHP())
				assert.Equal(t, CombatInjuryHeavilyWounded, event.MonsterInjury.Severity)
				assert.True(t, monster.IsFleeing(), "field damage must not reevaluate fleeing")
				assert.Equal(t, oracle.Calls(), state.GetRNG().Calls())
				assert.Equal(t, oracle.Intn(10000), state.GetRNG().Intn(10000))
				if direct && kind == CombatFieldPoison {
					require.Len(t, mock.CombatImpactCalls, 1)
					assert.Equal(t, 7, mock.CombatImpactCalls[0].Feedback.HPLost)
				} else {
					assert.Empty(t, mock.CombatImpactCalls)
				}
			})
		}
	}
}

func TestCombatMonsterInjuryPlayerMissileCommitsOnceAndCapturesAtImpact(t *testing.T) {
	state, mock, target, index, _ := setupPlayerRangedAttackTest(t, references.MagicBow, references.MonsterOrc, 3)
	require.NotNil(t, state)
	state.PartyState.Inventory.Equipment.Set(references.Arrows, 2)
	target.enemyRef.Armour = 0
	target.SetCurrentHP(70)
	before := state.GetRNG().Calls()
	confirmation := state.ConfirmCombatAttackTarget()
	require.Equal(t, CombatAttackSourcePending, confirmation.Outcome)
	assert.Equal(t, 70, target.CurrentHP())
	assert.Empty(t, mock.CombatEvents)
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Equipment.Get(references.Arrows))
	resolved := state.ResolvePendingCombatAttack()
	require.Equal(t, CombatAttackSequenceComplete, resolved.Outcome)
	require.Len(t, mock.CombatEvents, 1)
	event := mock.CombatEvents[0]
	require.Positive(t, event.HPLost)
	require.False(t, event.Fatal)
	assert.Equal(t, CombatEventRangedAttack, event.Kind)
	assert.Equal(t, index, event.Target.Index)
	assert.Equal(t, ClassifyCombatMonsterInjury(70-event.HPLost, 100), event.MonsterInjury.Severity)
	assert.Equal(t, target.CombatantID(), event.MonsterInjury.CombatantID)
	require.Len(t, mock.CombatImpactCalls, 1)
	assert.Equal(t, event.HPLost, mock.CombatImpactCalls[0].Feedback.HPLost)
	assert.Greater(t, state.GetRNG().Calls(), before)
	after := state.GetRNG().Calls()
	assert.Equal(t, CombatAttackTargetIgnored, state.ResolvePendingCombatAttack().Outcome)
	assert.Equal(t, after, state.GetRNG().Calls())
	assert.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Equipment.Get(references.Arrows))
}

func TestCombatMonsterInjuryPoisonWindUsesSurvivingPostHitBand(t *testing.T) {
	state, mock := newSyntheticCombatConeState(t)
	monster, index := addSyntheticConeMonster(state, references.MonsterOrc, references.Position{X: 5, Y: 3}, 1)
	monster.SetLoyalty(CombatMonsterLoyaltyParty)
	monster.SetCurrentHP(55)
	monster.SetFleeing(true)
	seed := areaSpellSeed(t, func(rng *deterministic_random.Source) bool { return rng.Intn(21) == 7 })
	state.SetRandomSeed(seed)
	result := state.applyCombatPoisonWindToxify(0, CombatTurnIdentity{EntityIndex: index}, CombatHitResult{Outcome: CombatHitNormal})
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, 7, result.HPLost)
	assert.Equal(t, CombatSpellDamaged, result.Outcome)
	assert.Equal(t, CombatInjuryHeavilyWounded, result.MonsterInjury.Severity)
	assert.Equal(t, result.MonsterInjury, mock.CombatEvents[0].MonsterInjury)
	assert.Equal(t, CombatMonsterLoyaltyParty, result.MonsterInjury.Loyalty)
	assert.True(t, monster.IsFleeing())
	assert.EqualValues(t, 1, state.GetRNG().Calls())
}

func TestCombatMonsterInjuryPossessedPCMeleeKeepsDamageAndFleeRNG(t *testing.T) {
	state, mock := newSyntheticCombatConeState(t)
	monster, index := addSyntheticConeMonster(state, references.MonsterOrc, references.Position{X: 5, Y: 4}, 1)
	monster.enemyRef.Armour = 0
	monster.SetLoyalty(CombatMonsterLoyaltyParty)
	monster.SetCurrentHP(70)
	attacker := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
	state.PartyState.GetCharacter(0).Dexterity = 255
	state.CombatScene.PartyPossessed[0] = true
	seed := areaSpellSeed(t, func(rng *deterministic_random.Source) bool {
		hit := CombatHitCheck(255, monster.CurrentDex(), false, rng)
		damage := CombatCalcDamage(references.TwoHSword, 0, rng)
		return hit.Outcome != CombatHitMiss && damage.Damage > 0 && damage.Damage < 20
	})
	oracle := deterministic_random.New(seed)
	CombatHitCheck(255, monster.CurrentDex(), false, oracle)
	damage := CombatCalcDamage(references.TwoHSword, 0, oracle)
	state.SetRandomSeed(seed)
	state.possessedPCMeleeAttack(attacker, CombatTurnIdentity{EntityIndex: index})
	require.Len(t, mock.CombatEvents, 1)
	event := mock.CombatEvents[0]
	assert.Equal(t, damage.Damage, event.HPLost)
	assert.Equal(t, CombatInjuryLightlyWounded, event.MonsterInjury.Severity)
	assert.False(t, monster.IsFleeing())
	assert.Equal(t, oracle.Calls(), state.GetRNG().Calls())
	assert.Equal(t, oracle.Intn(10000), state.GetRNG().Intn(10000))
}
