package game_state

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatPoisonFallbackAppliesMonsterProtectionOnce(t *testing.T) {
	tests := []struct {
		name           string
		targetType     references.MonsterType
		expectedDamage int
		undead         bool
		immortal       bool
	}{
		{name: "undead", targetType: references.MonsterSkeleton, expectedDamage: 6, undead: true},
		{name: "immortal", targetType: references.MonsterLordBritish, immortal: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := newSyntheticCombatDamageState(t)
			attacker, attackerIndex := addSyntheticProductionMonster(
				state, references.MonsterSnake, references.Position{X: 5, Y: 5},
			)
			attacker.currentDex = 99
			target, targetIndex := addSyntheticProductionMonster(
				state, test.targetType, references.Position{X: 6, Y: 5},
			)
			hpBefore := target.CurrentHP()
			experienceBefore := state.PartyState.GetAvatar().Exp
			seed := seedForExactCombatDraws(t, []int{61, 4, 21}, []int{0, 1, 12})
			state.SetRandomSeed(seed)
			callsBefore := state.GetRNG().Calls()
			callbacks.Reset()

			state.resolveMonsterMeleeAgainstMonster(attacker, attackerIndex, target, targetIndex)

			assert.Equal(t, hpBefore-test.expectedDamage, target.CurrentHP())
			assert.Equal(t, experienceBefore, state.PartyState.GetAvatar().Exp)
			assert.Equal(t, callsBefore+3, state.GetRNG().Calls(), "hit, poison selector, and fallback each draw once")
			require.Len(t, callbacks.CombatEvents, 1)
			event := callbacks.CombatEvents[0]
			assert.Equal(t, 12, event.Diagnostics.PoisonFallbackDamage)
			assert.Equal(t, test.expectedDamage, event.Diagnostics.FinalDamage)
			assert.Equal(t, test.undead, event.Diagnostics.TargetUndead)
			assert.Equal(t, test.immortal, event.Diagnostics.TargetImmortal)
			assert.False(t, event.Diagnostics.MagicDamage)
			assert.Equal(t, test.expectedDamage, event.HPLost)
		})
	}
}

func TestCombatDirectFieldDamageAppliesMagicalMonsterProtectionOnce(t *testing.T) {
	tests := []struct {
		name           string
		spell          references.Spell
		targetType     references.MonsterType
		ranges         []int
		values         []int
		rawDamage      int
		expectedDamage int
		expectedCalls  uint64
		undead         bool
		immortal       bool
	}{
		{
			name: "poison against undead", spell: references.InNoxGrav, targetType: references.MonsterSkeleton,
			ranges: []int{21}, values: []int{12}, rawDamage: 12, expectedDamage: 12, expectedCalls: 1, undead: true,
		},
		{
			name: "poison against immortal", spell: references.InNoxGrav, targetType: references.MonsterLordBritish,
			ranges: []int{21}, values: []int{12}, rawDamage: 12, expectedCalls: 1, immortal: true,
		},
		{
			name: "fire against undead", spell: references.InFlamGrav, targetType: references.MonsterSkeleton,
			ranges: []int{21}, values: []int{11}, rawDamage: 12, expectedDamage: 12, expectedCalls: 1, undead: true,
		},
		{
			name: "fire against immortal", spell: references.InFlamGrav, targetType: references.MonsterLordBritish,
			ranges: []int{21, 30}, values: []int{19, 0}, rawDamage: 19, expectedCalls: 2, immortal: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := newSyntheticCombatDamageState(t)
			target, targetIndex := addSyntheticProductionMonster(
				state, test.targetType, references.Position{X: 6, Y: 5},
			)
			hpBefore := target.CurrentHP()
			experienceBefore := state.PartyState.GetAvatar().Exp
			seed := seedForExactCombatDraws(t, test.ranges, test.values)
			state.SetRandomSeed(seed)
			oracle := deterministic_random.New(seed)
			for index, maximum := range test.ranges {
				require.Equal(t, test.values[index], oracle.Intn(maximum))
			}
			callsBefore := state.GetRNG().Calls()
			callbacks.Reset()
			installSyntheticPendingFieldSpell(state, test.spell, target.Position())

			require.True(t, state.ResolvePendingCombatSpell())

			assert.Equal(t, hpBefore-test.expectedDamage, target.CurrentHP())
			assert.Equal(t, experienceBefore, state.PartyState.GetAvatar().Exp)
			assert.Equal(t, callsBefore+test.expectedCalls, state.GetRNG().Calls())
			assert.Equal(t, oracle.Intn(1000000), state.GetRNG().Intn(1000000), "field reporting preserves the next gameplay draw")
			assert.Len(t, state.CombatScene.Fields(), 1)
			require.Len(t, callbacks.CombatEvents, 2, "one creation and one resolved field result, without duplicate attack narration")
			assert.Equal(t, CombatEventFieldCreated, callbacks.CombatEvents[0].Kind)
			assert.Zero(t, countCombatEvents(callbacks.CombatEvents, CombatEventAttack))
			if test.spell == references.InFlamGrav {
				assert.Empty(t, callbacks.CombatImpactCalls, "direct fire has only its field-event presentation owner")
			} else {
				require.Len(t, callbacks.CombatImpactCalls, 1, "direct poison retains one floating damage callback")
				impact := callbacks.CombatImpactCalls[0].Feedback
				assert.Equal(t, targetIndex, impact.Target.Index)
				assert.Equal(t, test.expectedDamage, impact.HPLost)
				assert.False(t, impact.Fatal)
				assert.Equal(t, CombatAttackDeliveryEnvironmental, impact.Presentation.Delivery)
			}
			resultEvent := callbacks.CombatEvents[1]
			require.Equal(t, CombatEventFieldTriggered, resultEvent.Kind)
			assert.Equal(t, 1, countCombatEvents(callbacks.CombatEvents, CombatEventFieldTriggered))
			assert.Equal(t, combatFieldKindForSpell(test.spell), resultEvent.FieldKind)
			assert.Equal(t, targetIndex, resultEvent.Target.Index)
			assert.Equal(t, test.spell, resultEvent.Diagnostics.Spell)
			assert.False(t, resultEvent.Fatal)
			if test.spell == references.InFlamGrav {
				assert.Equal(t, test.rawDamage, resultEvent.Diagnostics.Damage.Damage)
				assert.Equal(t, test.values[0]+1, resultEvent.Diagnostics.Damage.AttackRoll)
				assert.True(t, resultEvent.Diagnostics.Damage.AttackRolled)
				assert.Equal(t, len(test.values) > 1, resultEvent.Diagnostics.Damage.ArmorRolled)
				if len(test.values) > 1 {
					assert.Equal(t, test.values[1]+1, resultEvent.Diagnostics.Damage.ArmorRoll)
				}
				require.NotNil(t, resultEvent.FireInjury)
				assert.Equal(t, test.expectedDamage, resultEvent.FireInjury.HPLost)
			} else {
				assert.Nil(t, resultEvent.FireInjury, "poison's floating callback must not gain a second fire-impact owner")
			}
			assert.Equal(t, test.expectedDamage, resultEvent.Diagnostics.FinalDamage)
			assert.Equal(t, test.undead, resultEvent.Diagnostics.TargetUndead)
			assert.Equal(t, test.immortal, resultEvent.Diagnostics.TargetImmortal)
			assert.True(t, resultEvent.Diagnostics.MagicDamage)
			assert.Equal(t, test.expectedDamage, resultEvent.HPLost)
			if test.immortal {
				assert.Equal(t, CombatInjuryNone, resultEvent.MonsterInjury.Severity, "zero-loss immunity must not fabricate injury text")
			} else {
				assert.Equal(t, ClassifyCombatMonsterInjury(hpBefore-test.expectedDamage, target.MaxHP()), resultEvent.MonsterInjury.Severity)
				assert.Equal(t, target.CombatantID(), resultEvent.MonsterInjury.CombatantID)
			}
		})
	}
}

func TestCombatDirectFireFieldPreservesPlayerKillExperience(t *testing.T) {
	state, _ := newSyntheticCombatDamageState(t)
	target, _ := addSyntheticProductionMonster(
		state, references.MonsterOrc, references.Position{X: 6, Y: 5},
	)
	target.SetCurrentHP(1)
	caster := state.PartyState.GetAvatar()
	experienceBefore := caster.Exp
	state.SetRandomSeed(seedForExactCombatDraws(t, []int{21}, []int{20}))
	installSyntheticPendingFieldSpell(state, references.InFlamGrav, target.Position())

	require.True(t, state.ResolvePendingCombatSpell())

	assert.True(t, target.IsDead())
	assert.Equal(t, experienceBefore+uint16(CombatCalcExperience(target.MaxHP())), caster.Exp)
}

func TestCombatDirectPCFireUsesMagicDamageWhileStandingFireUsesZeroToTen(t *testing.T) {
	t.Run("direct endpoints without defense", func(t *testing.T) {
		for _, expectedDamage := range []int{1, 21} {
			t.Run(strconv.Itoa(expectedDamage), func(t *testing.T) {
				state, callbacks := newSyntheticCombatDamageState(t)
				identity := state.CombatScene.GetCurrentTurnIdentity()
				character := state.PartyState.GetCharacter(identity.EntityIndex)
				character.SetEquippedItem(references.SlotArmour, references.NoEquipment)
				character.SetEquippedItem(references.SlotShield, references.NoEquipment)
				character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
				character.SetEquippedItem(references.SlotWeapon, references.NoEquipment)
				character.SetEquippedItem(references.SlotRing, references.NoEquipment)
				character.SetEquippedItem(references.SlotAmulet, references.NoEquipment)
				require.Zero(t, state.combatPartyDefense(character))
				hpBefore := character.GetCurrentHp()
				state.SetRandomSeed(seedForExactCombatDraws(t, []int{21}, []int{expectedDamage - 1}))
				callsBefore := state.GetRNG().Calls()
				callbacks.Reset()
				installSyntheticPendingFieldSpell(
					state, references.InFlamGrav, state.CombatScene.GetCombatantPosition(identity),
				)

				require.True(t, state.ResolvePendingCombatSpell())

				assert.Equal(t, hpBefore-uint16(expectedDamage), character.GetCurrentHp())
				assert.Equal(t, callsBefore+1, state.GetRNG().Calls())
				require.Len(t, callbacks.CombatEvents, 2)
				assert.Equal(t, CombatEventFieldCreated, callbacks.CombatEvents[0].Kind)
				assert.Equal(t, CombatEventFieldTriggered, callbacks.CombatEvents[1].Kind)
				assert.Equal(t, expectedDamage, callbacks.CombatEvents[1].HPLost)
			})
		}
	})

	t.Run("direct fire subtracts approved defense", func(t *testing.T) {
		state, callbacks := newSyntheticCombatDamageState(t)
		identity := state.CombatScene.GetCurrentTurnIdentity()
		character := state.PartyState.GetCharacter(identity.EntityIndex)
		character.SetEquippedItem(references.SlotArmour, references.PlateMail)
		defense := state.combatPartyDefense(character)
		require.Positive(t, defense)
		require.Less(t, defense, 21)
		hpBefore := character.GetCurrentHp()
		state.SetRandomSeed(seedForExactCombatDraws(t, []int{21, defense}, []int{20, defense - 1}))
		callsBefore := state.GetRNG().Calls()
		callbacks.Reset()
		installSyntheticPendingFieldSpell(
			state, references.InFlamGrav, state.CombatScene.GetCombatantPosition(identity),
		)

		require.True(t, state.ResolvePendingCombatSpell())

		expectedDamage := 21 - defense
		assert.Equal(t, hpBefore-uint16(expectedDamage), character.GetCurrentHp())
		assert.Equal(t, callsBefore+2, state.GetRNG().Calls(), "direct fire draws attack then armor")
		require.Len(t, callbacks.CombatEvents, 2)
		assert.Equal(t, expectedDamage, callbacks.CombatEvents[1].HPLost)
	})

	t.Run("standing fire remains zero through ten without defense draw", func(t *testing.T) {
		state, callbacks := newSyntheticCombatDamageState(t)
		identity := state.CombatScene.GetCurrentTurnIdentity()
		character := state.PartyState.GetCharacter(identity.EntityIndex)
		character.SetEquippedItem(references.SlotArmour, references.PlateMail)
		position := state.CombatScene.GetCombatantPosition(identity)
		require.True(t, state.CombatScene.AddField(NewCombatField(CombatFieldFire, position)))
		hpBefore := character.GetCurrentHp()
		state.SetRandomSeed(seedForExactCombatDraws(t, []int{11}, []int{10}))
		callsBefore := state.GetRNG().Calls()
		callbacks.Reset()

		state.resolveCombatPCStandingEffects(identity)

		assert.Equal(t, hpBefore-10, character.GetCurrentHp())
		assert.Equal(t, callsBefore+1, state.GetRNG().Calls())
		require.Len(t, callbacks.CombatEvents, 1)
		assert.Equal(t, CombatEventFieldTriggered, callbacks.CombatEvents[0].Kind)
		assert.Equal(t, 10, callbacks.CombatEvents[0].HPLost)
	})
}

func TestCombatDirectFieldSkipsHiddenCorpserButStandingFieldStillApplies(t *testing.T) {
	state, callbacks := newSyntheticCombatDamageState(t)
	corpser, corpserIndex := addSyntheticProductionMonster(
		state, references.MonsterCorpser, references.Position{X: 6, Y: 5},
	)
	require.True(t, corpser.IsCorpserHidden())
	callsBefore := state.GetRNG().Calls()
	callbacks.Reset()
	installSyntheticPendingFieldSpell(state, references.InZuGrav, corpser.Position())

	require.True(t, state.ResolvePendingCombatSpell())

	assert.False(t, corpser.IsAsleep())
	assert.Len(t, state.CombatScene.Fields(), 1)
	assert.Equal(t, callsBefore, state.GetRNG().Calls())
	require.Len(t, callbacks.CombatEvents, 1)
	assert.Equal(t, CombatEventFieldCreated, callbacks.CombatEvents[0].Kind)

	state.resolveCombatMonsterFieldEffects(corpser, corpserIndex)

	assert.True(t, corpser.IsAsleep())
	assert.Equal(t, callsBefore, state.GetRNG().Calls())
	require.Len(t, callbacks.CombatEvents, 2)
	assert.Equal(t, CombatEventFieldTriggered, callbacks.CombatEvents[1].Kind)
}

func TestCombatDirectFieldSkipsHiddenCorpserAndUsesNextSourceOccupant(t *testing.T) {
	state, callbacks := newSyntheticCombatDamageState(t)
	position := references.Position{X: 6, Y: 5}
	corpser, _ := addSyntheticProductionMonster(state, references.MonsterCorpser, position)
	target, _ := addSyntheticProductionMonster(state, references.MonsterOrc, position)
	require.True(t, corpser.IsCorpserHidden())
	callbacks.Reset()
	installSyntheticPendingFieldSpell(state, references.InZuGrav, position)

	require.True(t, state.ResolvePendingCombatSpell())

	assert.False(t, corpser.IsAsleep())
	assert.True(t, target.IsAsleep(), "whosthere continues to the next eligible source slot")
	require.Len(t, callbacks.CombatEvents, 2)
	assert.Equal(t, CombatEventFieldCreated, callbacks.CombatEvents[0].Kind)
	assert.Equal(t, CombatEventFieldTriggered, callbacks.CombatEvents[1].Kind)
}

func newSyntheticCombatDamageState(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := newSyntheticDungeonTriggerCombatState(t)
	state.GameReferences.EnemyReferences = references.NewAllEnemyReferences(state.GameReferences.TileReferences)

	return state, callbacks
}

func addSyntheticProductionMonster(
	state *GameState,
	monsterType references.MonsterType,
	position references.Position,
) (*CombatMonster, int) {
	reference := (*state.GameReferences.EnemyReferences)[monsterType]
	monster := NewCombatMonster(monsterType, &reference, position, newFixedRNG(1))

	return monster, state.CombatScene.AddMonsterToQueue(monster)
}

func installSyntheticPendingFieldSpell(state *GameState, spell references.Spell, impact references.Position) {
	caster := state.CombatScene.GetCurrentTurnIdentity()
	state.CombatScene.pendingSpell = &pendingCombatSpell{
		spell:     spell,
		casterIdx: caster.EntityIndex,
		caster:    caster,
		launch: CombatProjectileLaunch{
			From:   state.CombatScene.GetCombatantPosition(caster),
			Aim:    impact,
			Impact: impact,
		},
	}
}

func seedForExactCombatDraws(t *testing.T, ranges, values []int) uint64 {
	t.Helper()
	require.Len(t, values, len(ranges))
	for seed := uint64(0); seed < 100000; seed++ {
		rng := deterministic_random.New(seed)
		matched := true
		for index, drawRange := range ranges {
			if rng.Intn(drawRange) != values[index] {
				matched = false
				break
			}
		}
		if matched {
			return seed
		}
	}
	t.Fatal("no deterministic seed produced the requested combat draws")

	return 0
}
