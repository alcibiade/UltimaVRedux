package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestFidelityF14WindDamageEndpoints(t *testing.T) {
	for _, poison := range []bool{false, true} {
		maximum := 30
		if poison {
			maximum = 20
		}
		for _, pcTarget := range []bool{false, true} {
			for _, damage := range []int{0, maximum} {
				t.Run(fmt.Sprintf("poison%t/pc%t/damage%d", poison, pcTarget, damage), func(t *testing.T) {
					gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
					caster := gs.CombatScene.GetActiveCombatantIndex()
					pc := gs.PartyState.GetCharacter(caster)
					pc.SetStatus(party_state.Poisoned)
					pc.SetMaxHp(100)
					pc.SetHp(100)
					pc.Dexterity = 1
					ref := *monster.EnemyRef()
					ref.HitPoints = 100
					monster.enemyRef = &ref
					monster.SetCurrentHP(100)
					monster.currentDex = 1
					position := monster.Position()
					if pcTarget {
						position = gs.CombatScene.GetPartyMemberPosition(caster)
					}
					var seed uint64
					for candidate := uint64(1); candidate < 10000; candidate++ {
						rng := beradrng.New(candidate)
						if poison {
							rng.Intn(61)
						}
						if rng.Intn(maximum+1) == damage {
							seed = candidate
							break
						}
					}
					require.NotZero(t, seed)
					wantRNG := beradrng.New(seed)
					if poison {
						wantRNG.Intn(61)
					}
					wantRNG.Intn(maximum + 1)
					gs.SetRandomSeed(seed)
					mock.Reset()
					if poison {
						applied, onset := gs.applyCombatInNoxHurAtPosition(caster, position)
						assert.True(t, applied)
						assert.False(t, onset)
					} else {
						assert.True(t, gs.applyCombatInFlamHurAtPosition(caster, position))
					}
					require.Len(t, mock.CombatEvents, 1)
					event := mock.CombatEvents[0]
					assert.Equal(t, damage, event.HPLost)
					assert.Equal(t, maximum, event.Diagnostics.Damage.AttackMax)
					assert.Equal(t, damage, event.Diagnostics.Damage.AttackRoll)
					assert.Equal(t, damage, event.Diagnostics.FinalDamage)
					assert.True(t, event.Diagnostics.Damage.AttackRolled)
					assert.Equal(t, poison, event.Diagnostics.Hit.Rolled)
					if pcTarget {
						assert.Equal(t, uint16(100-damage), pc.GetCurrentHp())
					} else {
						assert.Equal(t, 100-damage, monster.CurrentHP())
					}
					assert.Equal(t, wantRNG.Calls(), gs.GetRNG().Calls())
					assert.Equal(t, wantRNG.Intn(1000000), gs.GetRNG().Intn(1000000))
				})
			}
		}
	}
}

func TestFidelityF14MonsterToxifyEndpointsAndGoodPCDraws(t *testing.T) {
	for _, pcTarget := range []bool{false, true} {
		for _, damage := range []int{0, 20} {
			t.Run(fmt.Sprintf("pc%t/damage%d", pcTarget, damage), func(t *testing.T) {
				gs, _, attacker := setupCombatWithMonster(t, references.MonsterSnake, references.Down, 42)
				target := CombatTurnIdentity{EntityIndex: 0}
				if pcTarget {
					target = gs.CombatScene.GetCurrentTurnIdentity()
					gs.PartyState.GetCharacter(target.EntityIndex).SetStatus(party_state.Poisoned)
				}
				rng := newStrictSummonRNG(t, 1, damage)
				diagnostics := CombatDiagnostics{}
				result := gs.resolveSuccessfulMonsterHit(attacker, target, false, rng, &diagnostics)
				assert.Equal(t, []int{4, 21}, rng.requests)
				assert.True(t, result.damageSelected)
				assert.Equal(t, damage, result.damage)
				assert.True(t, diagnostics.PoisonFallbackRolled)
				assert.Equal(t, damage, diagnostics.PoisonFallbackDamage)
				assert.Equal(t, damage, diagnostics.FinalDamage)
				if pcTarget {
					gs.PartyState.GetCharacter(target.EntityIndex).SetStatus(party_state.Good)
					rng = newStrictSummonRNG(t, 1)
					diagnostics = CombatDiagnostics{}
					result = gs.resolveSuccessfulMonsterHit(attacker, target, false, rng, &diagnostics)
					assert.Equal(t, []int{4}, rng.requests)
					assert.Equal(t, CombatFeedbackPoison, result.feedback)
					assert.False(t, diagnostics.PoisonFallbackRolled)
				}
			})
		}
	}
}

func TestFidelityF15SleepConeSavesBeforeImmunity(t *testing.T) {
	gs, mock, immune := setupCombatWithMonster(t, references.MonsterLordBritish, references.Down, 42)
	gs.CombatScene.CombatMapRef = newOpenCombatConeMap()
	caster := gs.CombatScene.GetActiveCombatantIndex()
	gs.PartyState.ForEachCharacter(func(index int, _ *party_state.PlayerCharacter) {
		gs.CombatScene.SetPartyMemberPosition(index, references.Position{X: 0, Y: 10})
	})
	origin := references.Position{X: 5, Y: 5}
	gs.CombatScene.SetPartyMemberPosition(caster, origin)
	gs.PartyState.GetCharacter(caster).Intelligence = 20
	spec, ok := CombatConeSpecForSpell(references.InZu)
	require.True(t, ok)
	positions := TraceCombatCone(gs.CombatScene.CombatMapRef, origin, references.Up, spec)
	require.GreaterOrEqual(t, len(positions), 2)
	immune.SetPosition(positions[0])
	ordinary := NewCombatMonster(references.MonsterOrc, mustCombatEnemyReference(t, gs, references.MonsterOrc), positions[1], newFixedRNG(1))
	gs.CombatScene.AddMonsterToQueue(ordinary)
	threshold := (30 + ordinary.EnemyRef().Intelligence - 20) / 2
	var seed uint64
	for candidate := uint64(1); candidate < 10000; candidate++ {
		rng := beradrng.New(candidate)
		first := originalD30FromSourceValue(rng.Intn(61))
		second := originalD30FromSourceValue(rng.Intn(61))
		if first < threshold && second >= threshold {
			seed = candidate
			break
		}
	}
	require.NotZero(t, seed)
	wantRNG := beradrng.New(seed)
	wantRNG.Intn(61)
	wantRNG.Intn(61)
	gs.SetRandomSeed(seed)
	mock.Reset()
	assert.Equal(t, []references.Position{ordinary.Position()}, gs.resolveCombatInZu(caster, positions[:2]))
	assert.False(t, immune.IsAsleep())
	assert.True(t, ordinary.IsAsleep())
	require.Len(t, mock.CombatEvents, 2)
	assert.Equal(t, immune.Position(), mock.CombatEvents[0].To)
	assert.Equal(t, CombatSpellResisted, mock.CombatEvents[0].SpellTarget.Outcome,
		"a successful save terminates Sleep before the immunity gate")
	assert.Equal(t, ordinary.Position(), mock.CombatEvents[1].To)
	assert.Equal(t, CombatSpellSlept, mock.CombatEvents[1].SpellTarget.Outcome)
	assert.Equal(t, uint64(2), gs.GetRNG().Calls())
	assert.Equal(t, wantRNG.Intn(1000000), gs.GetRNG().Intn(1000000))
}

func TestFidelityF16CharmedDivisionCreatesFreshHostileChild(t *testing.T) {
	for _, repeated := range []bool{false, true} {
		t.Run(fmt.Sprintf("repeated%t", repeated), func(t *testing.T) {
			gs, _, parent := setupCombatWithMonster(t, references.MonsterSlime, references.Down, 42)
			setRepeatedMonsterDivisionForTest(t, gs, repeated)
			fillCombatMapForDivisionTest(gs)
			parent.SetCurrentHP(10)
			parent.SetLoyalty(CombatMonsterLoyaltyParty)
			parent.charmed, parent.summoned = true, true
			parent.SetPoisoned(true)
			parent.SetAsleep(true)
			parent.SetFleeing(true)
			parent.SetInvisible(true)
			parent.SetHasActed(true)
			caster := gs.CombatScene.GetActiveCombatantIndex()
			count := gs.CombatScene.MonsterManager.Count()
			gs.SetRandomSeed(42)
			position := gs.randomAdjacentCombatPosition(parent.Position())
			require.True(t, isCombatPositionValidForMonster(gs, position, parent, 0))
			fresh := NewCombatMonster(parent.MonsterType(), parent.EnemyRef(), position, gs.AsRandomProvider())
			wantCalls := gs.GetRNG().Calls()
			wantNext := gs.GetRNG().Intn(1000000)
			gs.SetRandomSeed(42)
			gs.applyCombatDamageToMonster(parent, 0, 1, false, gs.PartyState.GetCharacter(caster), caster, CombatDiagnostics{})
			require.Equal(t, count+1, gs.CombatScene.MonsterManager.Count())
			child := gs.CombatScene.MonsterManager.GetMonster(count)
			assert.True(t, parent.IsPartyLoyal())
			assert.True(t, child.IsHostile())
			assert.False(t, child.charmed)
			assert.False(t, child.summoned)
			assert.Equal(t, 9, parent.CurrentHP())
			assert.Equal(t, parent.CurrentHP(), child.CurrentHP())
			assert.True(t, child.IsActive())
			assert.False(t, child.IsPoisoned())
			assert.False(t, child.IsAsleep())
			assert.False(t, child.IsFleeing())
			assert.False(t, child.IsInvisible())
			assert.False(t, child.hasActed)
			assert.Equal(t, position, child.Position())
			assert.Equal(t, fresh.currentDex, child.currentDex)
			assert.Equal(t, wantCalls, gs.GetRNG().Calls(), "placement then fresh creation DEX")
			assert.Equal(t, wantNext, gs.GetRNG().Intn(1000000))
			assert.Equal(t, !repeated, parent.divisionConsumed)
			assert.Equal(t, !repeated, child.divisionConsumed)
		})
	}
}
