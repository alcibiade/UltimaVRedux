package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

// These seed searches use only detached streams. The selected bounded fixture
// is reset before each cast; production RNG is never used to arrange outcomes.
// areaSpellSeed returns the first seed whose RNG stream satisfies accepts.
func areaSpellSeed(t *testing.T, accepts func(*beradrng.Source) bool) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 1_000_000; seed++ {
		if accepts(beradrng.New(seed)) {
			return seed
		}
	}
	t.Fatal("no deterministic fixture seed")

	return 0
}

func addOutcomePython(state *GameState, position references.Position) (*CombatMonster, int) {
	// ENEMY-0001: Python has base DEX 18 and HP 10. A construction draw of
	// four gives zero encounter adjustment, retaining the real species DEX.
	ref := (*references.NewAllEnemyReferences(state.GameReferences.TileReferences))[references.MonsterSnake]
	monster := NewCombatMonster(references.MonsterSnake, &ref, position, newFixedRNG(4))

	return monster, state.CombatScene.AddMonsterToQueue(monster)
}

func TestCombatAreaEarthquakePythonOutcomesAndRNG(t *testing.T) {
	for _, test := range []struct {
		name string
		hits [3]bool
	}{
		{name: "all avoided", hits: [3]bool{false, false, false}},
		{name: "mixed", hits: [3]bool{false, true, false}},
		{name: "all damaged", hits: [3]bool{true, true, true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := newSyntheticCombatConeState(t)
			var pythons [3]*CombatMonster
			var indices [3]int
			for index := range pythons {
				pythons[index], indices[index] = addOutcomePython(state, references.Position{X: references.Coordinate(index + 3), Y: 3})
				require.Equal(t, 18, pythons[index].CurrentDex())
			}
			ally, _ := addOutcomePython(state, references.Position{X: 7, Y: 3})
			ally.SetLoyalty(CombatMonsterLoyaltyParty)
			inactive, _ := addOutcomePython(state, references.Position{X: 8, Y: 3})
			inactive.SetActive(false)
			dead, deadIndex := addOutcomePython(state, references.Position{X: 9, Y: 3})
			dead.SetCurrentHP(0)
			state.CombatScene.TurnQueue.MarkDead(deadIndex, false)
			seed := areaSpellSeed(t, func(rng *beradrng.Source) bool {
				for _, hit := range test.hits {
					if (max(1, rng.Intn(61)/2) >= 18) != hit {
						return false
					}
					if hit && rng.Intn(20)+1 >= 10 {
						return false // Isolate save/damage ordering from death-object draws.
					}
				}
				return true
			})
			state.SetRandomSeed(seed)
			wantRNG := beradrng.New(seed)
			callbacks.Reset()
			result := state.CastCombatEarthquakeResult(0)
			require.Len(t, result.Targets, 3)
			require.Len(t, callbacks.CombatEvents, 3)
			for index, target := range result.Targets {
				roll := max(1, wantRNG.Intn(61)/2)
				wantDamage := 0
				wantOutcome := CombatSpellAvoided
				if test.hits[index] {
					wantDamage = wantRNG.Intn(20) + 1
					wantOutcome = CombatSpellDamaged
				}
				assert.Equal(t, wantOutcome, target.Outcome)
				assert.Equal(t, wantDamage, target.HPLost)
				assert.Equal(t, 10-wantDamage, pythons[index].CurrentHP())
				assert.Equal(t, indices[index], target.Target.Index)
				assert.Equal(t, pythons[index].CombatantID(), target.CombatantID)
				event := callbacks.CombatEvents[index]
				require.NotNil(t, event.SpellTarget)
				assert.Equal(t, target, *event.SpellTarget)
				if test.hits[index] {
					assert.Equal(t, CombatFeedbackDamage, event.Result)
					assert.Equal(t, ClassifyCombatMonsterInjury(10-wantDamage, 10), event.MonsterInjury.Severity)
					assert.Equal(t, target.MonsterInjury, event.MonsterInjury)
				} else {
					assert.Equal(t, CombatFeedbackMiss, event.Result)
					assert.Equal(t, CombatInjuryNone, event.MonsterInjury.Severity)
				}
				assert.Equal(t, roll, event.Diagnostics.Hit.D30Roll)
				assert.Equal(t, 18, event.Diagnostics.Hit.Threshold)
				assert.Equal(t, test.hits[index], event.Diagnostics.Damage.AttackRolled)
			}
			assert.Equal(t, 10, ally.CurrentHP())
			assert.Equal(t, 10, inactive.CurrentHP())
			assert.Equal(t, uint16(100), state.PartyState.GetCharacter(0).GetCurrentHp())
			assert.Equal(t, wantRNG.Calls(), state.GetRNG().Calls())
			assert.Equal(t, wantRNG.Intn(1000000), state.GetRNG().Intn(1000000))
			assert.Empty(t, callbacks.CombatImpactCalls, "the single result event owns local presentation")
			wantCast := CombatAreaSpellAffected
			if test.name == "all avoided" {
				wantCast = CombatAreaSpellUnaffected
			}
			assert.Equal(t, wantCast, result.Outcome())
		})
	}
}

func TestCombatAreaSleepMixedSavesImmunityAndStatus(t *testing.T) {
	state, callbacks := newSyntheticCombatConeState(t)
	state.PartyState.GetCharacter(0).Intelligence = 30
	positions := []references.Position{{X: 4, Y: 3}, {X: 5, Y: 3}, {X: 6, Y: 3}}
	resisted, _ := addSyntheticConeMonster(state, references.MonsterOrc, positions[0], 255)
	immune, _ := addSyntheticConeMonster(state, references.MonsterBlackthorn, positions[1], 0)
	slept, _ := addSyntheticConeMonster(state, references.MonsterOrc, positions[2], 0)
	slept.SetLoyalty(CombatMonsterLoyaltyParty)
	state.SetRandomSeed(37)
	wantRNG := beradrng.New(37)
	for range 3 {
		wantRNG.Intn(61)
	}
	result := state.resolveCombatConeResults(references.InZu, 0, append(positions, positions[0]))
	require.Len(t, result.Targets, 3, "a duplicate candidate must not reroll or fall through")
	require.Len(t, callbacks.CombatEvents, 3)
	assert.Equal(t, CombatSpellResisted, result.Targets[0].Outcome)
	assert.Equal(t, CombatSpellImmune, result.Targets[1].Outcome)
	assert.Equal(t, CombatSpellSlept, result.Targets[2].Outcome)
	assert.False(t, resisted.IsAsleep())
	assert.False(t, immune.IsAsleep())
	assert.True(t, slept.IsAsleep(), "friendly cone targets retain eligibility")
	for _, monster := range []*CombatMonster{resisted, immune, slept} {
		assert.Equal(t, 100, monster.CurrentHP())
	}
	assert.Equal(t, CombatAreaSpellAffected, result.Outcome(), "sleep is successful without HP loss")
	assert.Equal(t, wantRNG.Calls(), state.GetRNG().Calls())
	assert.Equal(t, wantRNG.Intn(1000000), state.GetRNG().Intn(1000000))
}

func TestCombatAreaEarthquakePythonDeathAwardsXPAndPreservesRNG(t *testing.T) {
	state, callbacks := newSyntheticCombatConeState(t)
	python, index := addOutcomePython(state, references.Position{X: 5, Y: 3})
	seed := areaSpellSeed(t, func(rng *beradrng.Source) bool {
		return max(1, rng.Intn(61)/2) >= 18 && rng.Intn(20)+1 >= 10
	})
	state.SetRandomSeed(seed)
	wantRNG := beradrng.New(seed)
	wantRNG.Intn(61)
	damage := wantRNG.Intn(20) + 1
	wantRNG.Intn(61) // Python's zero-treasure death-object check still draws.
	xp := state.PartyState.GetCharacter(0).Exp
	result := state.CastCombatEarthquakeResult(0)
	require.Len(t, result.Targets, 1)
	assert.Zero(t, python.CurrentHP())
	assert.Equal(t, 10, result.Targets[0].HPLost)
	assert.True(t, result.Targets[0].Fatal)
	assert.Equal(t, CombatInjuryNone, result.Targets[0].MonsterInjury.Severity)
	assert.True(t, result.Targets[0].Disclosed)
	assert.Equal(t, python.CombatantID(), result.Targets[0].CombatantID)
	assert.Equal(t, index, result.Targets[0].Target.Index)
	assert.Equal(t, xp+uint16(combatCalcExperience(10)), state.PartyState.GetCharacter(0).Exp)
	require.Len(t, callbacks.CombatEvents, 1)
	assert.Equal(t, damage, callbacks.CombatEvents[0].Diagnostics.Damage.AttackRoll)
	assert.Equal(t, 1, state.CombatScene.ObjectCount())
	assert.Equal(t, wantRNG.Calls(), state.GetRNG().Calls())
	assert.Equal(t, wantRNG.Intn(1000000), state.GetRNG().Intn(1000000))
}

func TestCombatAreaFlameDamageKeepsDivisionAndRNGOrder(t *testing.T) {
	state, callbacks := newSyntheticCombatConeState(t)
	expected, _ := newSyntheticCombatConeState(t)
	position := references.Position{X: 5, Y: 3}
	monster, _ := addSyntheticConeMonster(state, references.MonsterSlime, position, 1)
	baseline, baselineIndex := addSyntheticConeMonster(expected, references.MonsterSlime, position, 1)
	for _, victim := range []*CombatMonster{monster, baseline} {
		victim.enemyRef.EnemyAbilities = map[references.EnemyAbility]bool{references.DivideOnHit: true}
	}
	seed := areaSpellSeed(t, func(rng *beradrng.Source) bool { return rng.Intn(31) > 0 })
	state.SetRandomSeed(seed)
	expected.SetRandomSeed(seed)
	damage := expected.GetRNG().Intn(31)
	baseline.TakeDamage(damage)
	expected.tryDivideCombatMonster(baseline, baselineIndex)
	result := state.resolveCombatConeResults(references.InFlamHur, 0, []references.Position{position})
	require.Len(t, result.Targets, 1)
	assert.Equal(t, damage, result.Targets[0].HPLost)
	assert.Equal(t, 100-damage, monster.CurrentHP())
	require.Equal(t, 2, state.CombatScene.MonsterManager.Count())
	require.Len(t, callbacks.CombatEvents, 2)
	assert.Equal(t, CombatEventSpell, callbacks.CombatEvents[0].Kind)
	assert.Equal(t, ClassifyCombatMonsterInjury(100-damage, 100), callbacks.CombatEvents[0].MonsterInjury.Severity)
	assert.Equal(t, result.Targets[0].MonsterInjury, callbacks.CombatEvents[0].MonsterInjury)
	assert.Equal(t, CombatEventDivision, callbacks.CombatEvents[1].Kind)
	child := state.CombatScene.MonsterManager.GetMonster(1)
	assert.Equal(t, monster.CurrentHP(), child.CurrentHP())
	assert.Equal(t, expected.CombatScene.MonsterManager.GetMonster(1).Position(), child.Position())
	assert.Equal(t, expected.GetRNG().Calls(), state.GetRNG().Calls())
	assert.Equal(t, expected.GetRNG().Intn(1000000), state.GetRNG().Intn(1000000))
}

func TestCombatAreaVortexResistedImmuneAndDeath(t *testing.T) {
	state, callbacks := newSyntheticCombatConeState(t)
	state.PartyState.GetCharacter(0).Intelligence = 30
	positions := []references.Position{{X: 4, Y: 3}, {X: 5, Y: 3}, {X: 6, Y: 3}}
	resisted, _ := addSyntheticConeMonster(state, references.MonsterOrc, positions[0], 255)
	immune, _ := addSyntheticConeMonster(state, references.MonsterBlackthorn, positions[1], 0)
	killed, _ := addSyntheticConeMonster(state, references.MonsterOrc, positions[2], 0)
	xp := state.PartyState.GetCharacter(0).Exp
	state.SetRandomSeed(37)
	wantRNG := beradrng.New(37)
	for range 4 { // Three saves, followed by the successful target's death-object draw.
		wantRNG.Intn(61)
	}
	result := state.resolveCombatConeResults(references.InVasGravCorp, 0, positions)
	require.Len(t, result.Targets, 3)
	require.Len(t, callbacks.CombatEvents, 3)
	assert.Equal(t, CombatSpellResisted, result.Targets[0].Outcome)
	assert.Equal(t, CombatSpellImmune, result.Targets[1].Outcome)
	assert.Equal(t, CombatSpellDamaged, result.Targets[2].Outcome)
	assert.Equal(t, 100, resisted.CurrentHP())
	assert.Equal(t, 100, immune.CurrentHP())
	assert.Zero(t, killed.CurrentHP())
	assert.Equal(t, 100, result.Targets[2].HPLost)
	assert.True(t, result.Targets[2].Fatal)
	assert.True(t, result.Targets[2].Disclosed, "capture disclosure before the target dies")
	assert.Equal(t, xp+uint16(combatCalcExperience(100)), state.PartyState.GetCharacter(0).Exp)
	assert.Equal(t, 1, state.CombatScene.ObjectCount())
	assert.Equal(t, wantRNG.Calls(), state.GetRNG().Calls())
	assert.Equal(t, wantRNG.Intn(1000000), state.GetRNG().Intn(1000000))
}

func TestCombatAreaWindZeroDamageAndPoisonAvoidance(t *testing.T) {
	for _, spell := range []references.Spell{references.InFlamHur, references.InNoxHur} {
		t.Run(references.GetSpellName(spell), func(t *testing.T) {
			state, callbacks := newSyntheticCombatConeState(t)
			position := references.Position{X: 5, Y: 3}
			monster, _ := addSyntheticConeMonster(state, references.MonsterOrc, position, 1)
			monster.currentDex = 1
			seed := areaSpellSeed(t, func(rng *beradrng.Source) bool {
				maximum := 31
				if spell == references.InNoxHur {
					rng.Intn(61)
					maximum = 21
				}
				return rng.Intn(maximum) == 0
			})
			state.SetRandomSeed(seed)
			result := state.resolveCombatConeResults(spell, 0, []references.Position{position})
			require.Len(t, result.Targets, 1)
			require.Len(t, callbacks.CombatEvents, 1)
			assert.Equal(t, CombatSpellZeroDamage, result.Targets[0].Outcome)
			assert.Equal(t, CombatAreaSpellUnaffected, result.Outcome())
			assert.Equal(t, 100, monster.CurrentHP())
			assert.Zero(t, callbacks.CombatEvents[0].HPLost)
			assert.Zero(t, callbacks.CombatEvents[0].Diagnostics.Damage.AttackRoll)
			assert.True(t, callbacks.CombatEvents[0].Diagnostics.Damage.AttackRolled)
			calls := uint64(1)
			if spell == references.InNoxHur {
				calls++
			}
			assert.Equal(t, calls, state.GetRNG().Calls())
			if spell == references.InNoxHur {
				monster.currentDex = 255
				callbacks.Reset()
				state.SetRandomSeed(seed)
				result = state.resolveCombatConeResults(spell, 0, []references.Position{position})
				assert.Equal(t, CombatSpellAvoided, result.Targets[0].Outcome)
				require.Len(t, callbacks.CombatEvents, 1)
				assert.False(t, callbacks.CombatEvents[0].Diagnostics.Damage.AttackRolled)
				assert.Equal(t, uint64(1), state.GetRNG().Calls())
				assert.Equal(t, 100, monster.CurrentHP())
			}
		})
	}
}

func TestCombatAreaConeNoTargetsUsesCommittedCandidates(t *testing.T) {
	for _, spell := range []references.Spell{references.InZu, references.InFlamHur, references.InNoxHur, references.InVasGravCorp} {
		t.Run(references.GetSpellName(spell), func(t *testing.T) {
			state, callbacks := newSyntheticCombatConeState(t)
			caster := state.PartyState.GetCharacter(0)
			caster.Level = 8
			caster.SetMp(30)
			state.PartyState.Inventory.Spells.Set(spell, 2)
			require.True(t, state.BeginCombatConeTargeting(CastRequest{Spell: spell, CasterIdx: 0}).Valid)
			state.SetCombatConeDirection(references.Up)
			committed := state.ConfirmCombatConeTarget()
			require.Equal(t, CombatConeTargetResolved, committed.Outcome)
			calls := state.GetRNG().Calls()
			// A new visible target after confirmation must not cause a rescan of
			// geometry or turn an empty committed candidate list into a new hit.
			monster, _ := addSyntheticConeMonster(state, references.MonsterOrc, references.Position{X: 5, Y: 3}, 0)
			result := state.ResolveCommittedCombatCone(committed)
			assert.Equal(t, CombatAreaSpellNoTargets, result.Outcome())
			assert.Empty(t, result.Targets)
			assert.Empty(t, callbacks.CombatEvents)
			assert.Equal(t, calls, state.GetRNG().Calls())
			assert.Equal(t, uint16(1), state.PartyState.Inventory.Spells.Get(spell))
			assert.Equal(t, byte(30-int(spell)/6-1), caster.GetCurrentMp())
			assert.Equal(t, uint64(1), state.CastUseCommitSequence())
			assert.Equal(t, 100, monster.CurrentHP())
			assert.False(t, monster.IsAsleep())
		})
	}
}

func TestCombatAreaConeExactCloneStatusAndConcealedTarget(t *testing.T) {
	state, callbacks := newSyntheticCombatConeState(t)
	source := state.CombatScene.GetCurrentTurnIdentity()
	position := references.Position{X: 5, Y: 3}
	clone := state.CombatScene.addPCCloneFromSource(source, source.EntityIndex, position,
		NewCombatPCProvider(state.PartyState.GetCharacter(source.EntityIndex)))
	require.NotZero(t, clone.Slot)
	state.PartyState.GetCharacter(0).Dexterity = 1
	state.SetRandomSeed(11)
	result := state.resolveCombatConeResults(references.InNoxHur, 0, []references.Position{position})
	require.Len(t, result.Targets, 1)
	assert.Equal(t, clone.Slot, result.Targets[0].Target.Slot)
	assert.Equal(t, CombatSpellPoisoned, result.Targets[0].Outcome)
	assert.Equal(t, party_state.Poisoned, state.PartyState.GetCharacter(0).GetStatus())
	assert.Equal(t, uint64(1), state.GetRNG().Calls(), "status success has no fallback damage draw")
	assert.Equal(t, CombatAreaSpellAffected, result.Outcome())
	require.Len(t, callbacks.TransientIconCalls, 1)
	monster, _ := addSyntheticConeMonster(state, references.MonsterOrc, position, 0)
	monster.SetInvisible(true)
	result = state.resolveCombatConeResults(references.InZu, 0, []references.Position{position})
	require.Len(t, result.Targets, 1)
	assert.Equal(t, monster.CombatantID(), result.Targets[0].CombatantID)
	assert.False(t, result.Targets[0].Disclosed, "legal target collection is not presentation disclosure")
	assert.True(t, monster.IsAsleep())
}

func TestCombatAreaEarthquakeRetainsExactPossessedCloneIdentity(t *testing.T) {
	state, callbacks := newSyntheticCombatConeState(t)
	source := state.CombatScene.GetCurrentTurnIdentity()
	clone := state.CombatScene.addPCCloneFromSource(source, source.EntityIndex, references.Position{X: 5, Y: 3},
		NewCombatPCProvider(state.PartyState.GetCharacter(source.EntityIndex)))
	state.CombatScene.setCombatPCPossessed(clone, true)
	state.PartyState.GetCharacter(0).Dexterity = 255
	state.SetRandomSeed(1)
	result := state.CastCombatEarthquakeResult(0)
	require.Len(t, result.Targets, 1, "the party-loyal source must not be included")
	assert.Equal(t, CombatSpellAvoided, result.Targets[0].Outcome)
	assert.Equal(t, clone.Slot, result.Targets[0].Target.Slot)
	assert.True(t, result.Targets[0].Target.IsPartyMember)
	assert.Equal(t, uint16(100), state.PartyState.GetCharacter(0).GetCurrentHp())
	require.Len(t, callbacks.CombatEvents, 1)
	assert.Equal(t, source.Slot, callbacks.CombatEvents[0].Attacker.Slot)
	assert.Equal(t, clone.Slot, callbacks.CombatEvents[0].Target.Slot)
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
}

func TestCombatAreaImmunityLabelOnlyComesFromReachedImmunityGate(t *testing.T) {
	for _, spell := range []references.Spell{references.InFlamHur, references.InVasGravCorp} {
		t.Run(references.GetSpellName(spell), func(t *testing.T) {
			state, callbacks := newSyntheticCombatConeState(t)
			position := references.Position{X: 5, Y: 3}
			monster, _ := addSyntheticConeMonster(state, references.MonsterOrc, position, 1)
			monster.enemyRef.EnemyAbilities = map[references.EnemyAbility]bool{references.Immortal: true}
			seed := areaSpellSeed(t, func(rng *beradrng.Source) bool { return rng.Intn(31) > 0 })
			state.SetRandomSeed(seed)
			result := state.resolveCombatConeResults(spell, 0, []references.Position{position})
			require.Len(t, result.Targets, 1)
			assert.Equal(t, 100, monster.CurrentHP())
			assert.Equal(t, uint64(1), state.GetRNG().Calls())
			if spell == references.InFlamHur {
				assert.Equal(t, CombatSpellZeroDamage, result.Targets[0].Outcome)
				assert.Positive(t, callbacks.CombatEvents[0].Diagnostics.Damage.AttackRoll)
				assert.Zero(t, callbacks.CombatEvents[0].Diagnostics.FinalDamage)
			} else {
				assert.Equal(t, CombatSpellImmune, result.Targets[0].Outcome)
				assert.True(t, callbacks.CombatEvents[0].Diagnostics.Hit.Rolled)
			}
		})
	}
}
