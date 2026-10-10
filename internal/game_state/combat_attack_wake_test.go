package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func newAttackWakeFixture(t *testing.T) (*GameState, *MockSystemCallbacks, CombatTurnIdentity, CombatTurnIdentity, CombatTurnIdentity, *CombatMonster, int) {
	t.Helper()
	state, source, target := newSyntheticExactIdentityCastUseState(t)
	character := state.PartyState.GetCharacter(source.EntityIndex)
	sibling := state.CombatScene.addPCCloneFromSource(source, source.EntityIndex,
		references.Position{X: 8, Y: 8}, NewCombatPCProvider(character))
	state.CombatScene.setCombatantPosition(source, references.Position{X: 2, Y: 5})
	state.CombatScene.setCombatantPosition(target, references.Position{X: 5, Y: 5})
	state.setCombatPCSleeping(target, true)
	state.setCombatPCSleeping(sibling, true)
	ref := &references.EnemyReference{ //nolint:exhaustruct // Bounded synthetic attacker.
		MonsterType: references.MonsterOrc, Name: "Orc", Dexterity: 1,
		Strength: 1, Damage: 10, HitPoints: 100, AttackRange: 4,
		LargeMapMissileRange: 4, LargeMapMissile: references.MissileArrow,
	}
	monster := NewCombatMonster(references.MonsterOrc, ref, references.Position{X: 4, Y: 5}, newFixedRNG(1))
	monsterIndex := state.CombatScene.AddMonsterToQueue(monster)
	mock := NewMockSystemCallbacks(t)
	state.SystemCallbacks = mock.ToSystemCallbacks()
	state.SetRandomSeed(1)

	return state, mock, source, target, sibling, monster, monsterIndex
}

func assertExactAttackWake(t *testing.T, state *GameState, mock *MockSystemCallbacks, target, sibling, current CombatTurnIdentity) {
	t.Helper()
	assert.False(t, state.CombatScene.isCombatPCSleeping(target))
	assert.True(t, state.CombatScene.isCombatPCSleeping(sibling), "shared HP must not wake another tactical copy")
	assert.False(t, state.PartyState.GetCharacter(target.EntityIndex).IsAsleep())
	assert.Equal(t, current, state.CombatScene.GetCurrentTurnIdentity(), "waking grants no immediate action")
	wakes := 0
	for _, event := range mock.CombatEvents {
		if event.Kind == CombatEventCondition && event.Condition == CombatConditionWoke {
			wakes++
			assert.Equal(t, target.Slot, event.Target.Slot)
			assert.Equal(t, target.EntityIndex, event.Target.Index)
		}
	}
	assert.Equal(t, 1, wakes)
	calls := state.GetRNG().Calls()
	assert.Equal(t, combatTurnReady, state.processCombatPCTurnStart(target), "ready at the next scheduled turn")
	assert.Equal(t, calls, state.GetRNG().Calls(), "no spontaneous wake roll after an attack wake")
}

func TestDirectedMeleeWakesExactSleepingPCOnMissDamageAndGraze(t *testing.T) {
	for _, outcome := range []string{"miss", "damage", "graze"} {
		t.Run(outcome, func(t *testing.T) {
			state, mock, _, target, sibling, monster, monsterIndex := newAttackWakeFixture(t)
			character := state.PartyState.GetCharacter(target.EntityIndex)
			roll := 60
			if outcome == "miss" {
				roll = 0
			}
			if outcome == "graze" {
				character.SetEquippedItem(references.SlotArmour, references.PlateMail)
				monster.enemyRef.Damage = 0
			}
			state.SetRandomSeed(seedForFirstIntnValue(t, state, originalD30SourceValueCount, roll))
			current := state.CombatScene.GetCurrentTurnIdentity()
			queueBefore := make([]TurnQueueEntry, len(state.CombatScene.TurnQueue.entries))
			for i, entry := range state.CombatScene.TurnQueue.entries {
				queueBefore[i] = *entry
			}
			nearest := nearestPCResult{found: true, identity: target, pcIndex: target.EntityIndex, distance: 1}
			require.True(t, state.monsterMeleeAttack(monster, monsterIndex, nearest))
			assertExactAttackWake(t, state, mock, target, sibling, current)
			for i, entry := range state.CombatScene.TurnQueue.entries {
				assert.Equal(t, queueBefore[i], *entry, "attack awakening must not reorder or reset queue timers")
			}
			require.NotEmpty(t, mock.CombatEvents)
			want := map[string]CombatFeedbackResult{"miss": CombatFeedbackMiss, "damage": CombatFeedbackDamage, "graze": CombatFeedbackGraze}[outcome]
			assert.Equal(t, want, mock.CombatEvents[0].Result)
			if outcome == "miss" {
				assert.Equal(t, uint64(1), state.GetRNG().Calls(), "miss waking must not consume RNG")
			}
		})
	}
}

func TestDirectedMonsterMissileWakesAtResolutionEvenWhenScattered(t *testing.T) {
	for _, outcome := range []string{"hit", "miss", "blocked"} {
		state, mock, _, target, sibling, monster, monsterIndex := newAttackWakeFixture(t)
		monster.SetPosition(references.Position{X: 2, Y: 5})
		nearest := nearestPCResult{found: true, identity: target, pcIndex: target.EntityIndex, distance: 3}
		roll := 60
		if outcome == "miss" {
			roll = 0
		}
		if outcome == "blocked" {
			setBlockedCombatMissileMap(state, references.Position{X: 4, Y: 5})
		}
		launch, attacked := state.prepareMonsterAttackWithRNG(monster, monsterIndex, nearest, newSequenceRNG(0, roll, 2, 1))
		require.True(t, attacked)
		require.NotNil(t, launch)
		assert.True(t, state.CombatScene.isCombatPCSleeping(target), "launch cannot wake")
		assert.Empty(t, mock.CombatEvents)
		if outcome == "miss" {
			assert.Equal(t, CombatHitMiss, state.CombatScene.pendingMonsterMissile.diagnostics.Hit.Outcome)
			assert.NotEqual(t, state.CombatScene.GetCombatantPosition(target), launch.Impact)
		}
		if outcome == "blocked" {
			assert.True(t, launch.Blocked)
		}
		current := state.CombatScene.GetCurrentTurnIdentity()
		require.True(t, state.ResolvePendingMonsterMissile())
		assertExactAttackWake(t, state, mock, target, sibling, current)
		calls, events := state.GetRNG().Calls(), len(mock.CombatEvents)
		require.False(t, state.ResolvePendingMonsterMissile())
		assert.Equal(t, calls, state.GetRNG().Calls())
		assert.Len(t, mock.CombatEvents, events)
	}
}

func TestDirectedFriendlyAndControlledAttacksWakeExactPC(t *testing.T) {
	for _, path := range []string{"friendly melee", "possessed melee", "controlled melee", "friendly ranged", "controlled ranged"} {
		t.Run(path, func(t *testing.T) {
			state, mock, source, target, sibling, monster, monsterIndex := newAttackWakeFixture(t)
			current := state.CombatScene.GetCurrentTurnIdentity()
			if path == "friendly melee" || path == "possessed melee" {
				state.CombatScene.setCombatantPosition(source, references.Position{X: 5, Y: 4})
			}
			sourceWeapon := CombatAttackSource{Equipment: references.Dagger, Slot: references.SlotWeapon, MaxRange: 4, Missile: references.MissileArrow}
			switch path {
			case "friendly melee":
				state.resolveCombatPCAttackSourceAgainstPC(source, sourceWeapon, target)
			case "possessed melee":
				state.CombatScene.setCombatPCPossessed(source, true)
				state.possessedPCMeleeAttack(source, target)
			case "controlled melee":
				state.CombatScene.setCombatPCPossessed(target, true)
				state.resolveControlledMonsterAttackAgainstPC(monsterIndex, target, state.CombatScene.GetCombatantPosition(target))
			default:
				attacker := source
				if path == "controlled ranged" {
					attacker = CombatTurnIdentity{EntityIndex: monsterIndex}
					monster.SetPosition(references.Position{X: 2, Y: 5})
				}
				state.PartyState.Inventory.Equipment.Set(references.Dagger, 2)
				state.CombatScene.AttackTargeting = &CombatAttackTargeting{
					Attacker: attacker, AttackerIndex: attacker.EntityIndex,
					Origin: state.CombatScene.GetCombatantPosition(attacker), Cursor: state.CombatScene.GetCombatantPosition(target),
					Sources: []CombatAttackSource{sourceWeapon},
				}
				result := state.preparePendingCombatAttack(state.CombatScene.AttackTargeting, sourceWeapon, target, true)
				require.Equal(t, CombatAttackSourcePending, result.Outcome)
				assert.True(t, state.CombatScene.isCombatPCSleeping(target))
				state.ResolvePendingCombatAttack()
				assert.Equal(t, CombatAttackTargetIgnored, state.ResolvePendingCombatAttack().Outcome)
			}
			assertExactAttackWake(t, state, mock, target, sibling, current)
		})
	}
}

func TestDirectedSpellWakePreservesFreshSleepAndHazardBoundary(t *testing.T) {
	for _, spell := range []references.Spell{references.VasFlam, references.GravPor, references.XenCorp, references.InFlamGrav, references.InNoxGrav, references.InZuGrav, references.InSanctGrav} {
		t.Run(fmt.Sprint(spell), func(t *testing.T) {
			state, mock, source, target, sibling, _, _ := newAttackWakeFixture(t)
			state.PartyState.Inventory.Spells.Set(spell, 1)
			if !isCombatFieldSpell(spell) {
				state.CombatScene.setCombatPCPossessed(target, true)
			}
			require.True(t, state.BeginCombatSpellTargeting(CastRequest{Spell: spell, CasterIdx: source.EntityIndex}).Valid)
			state.CombatScene.SpellTargeting.Cursor = state.CombatScene.GetCombatantPosition(target)
			// Xen Corp's resisted save scatters; the directed sleeper still wakes.
			result := state.confirmCombatSpellTargetWithRNG(newSequenceRNG(0, 2, 1))
			require.Equal(t, CombatSpellTargetResolved, result.Outcome)
			assert.True(t, state.CombatScene.isCombatPCSleeping(target))
			current := state.CombatScene.GetCurrentTurnIdentity()
			require.True(t, state.ResolvePendingCombatSpell())
			require.False(t, state.ResolvePendingCombatSpell())
			if spell == references.InZuGrav || spell == references.InSanctGrav {
				assert.True(t, state.CombatScene.isCombatPCSleeping(target))
			} else {
				assertExactAttackWake(t, state, mock, target, sibling, current)
			}
		})
	}
	for _, kind := range []CombatFieldKind{CombatFieldFire, CombatFieldPoison} {
		state, _, _, target, _, _, _ := newAttackWakeFixture(t)
		state.applyCombatFieldToPC(NewCombatField(kind, state.CombatScene.GetCombatantPosition(target)), target, false)
		assert.True(t, state.CombatScene.isCombatPCSleeping(target), "ongoing hazards are not directed attacks")
	}
}

func TestDirectedAttackWakePreservesDeathAndUnrelatedStatus(t *testing.T) {
	for _, status := range []party_state.CharacterStatus{party_state.Poisoned, party_state.Dead} {
		state, mock, source, target, sibling, _, _ := newAttackWakeFixture(t)
		character := state.PartyState.GetCharacter(target.EntityIndex)
		if status == party_state.Dead {
			character.SetHp(1)
			state.resolveCombatPCAttackSourceAgainstPC(source, CombatAttackSource{Equipment: references.GlassSword, Slot: references.SlotWeapon}, target)
			assert.True(t, character.IsDead())
			for _, event := range mock.CombatEvents {
				assert.NotEqual(t, CombatConditionWoke, event.Condition)
			}
		} else {
			character.SetStatus(status)
			state.possessedPCMeleeAttack(source, target)
			assertExactAttackWake(t, state, mock, target, sibling, source)
		}
		assert.Equal(t, status, character.GetStatus())
	}
}

func TestDirectedAttackOnSourceDoesNotWakeSleepingClone(t *testing.T) {
	state, mock, source, clone, _, monster, monsterIndex := newAttackWakeFixture(t)
	monster.SetPosition(references.Position{X: 3, Y: 5})
	state.setCombatPCSleeping(source, true)
	current := state.CombatScene.GetCurrentTurnIdentity()
	require.True(t, state.monsterMeleeAttack(monster, monsterIndex,
		nearestPCResult{found: true, identity: source, pcIndex: source.EntityIndex, distance: 1}))
	assertExactAttackWake(t, state, mock, source, clone, current)
}

func TestAreaMagicDamageDoesNotWakeUndirectedSleepingPC(t *testing.T) {
	state, mock, source, target, _, _, _ := newAttackWakeFixture(t)
	character := state.PartyState.GetCharacter(target.EntityIndex)
	hpBefore := character.GetCurrentHp()
	state.SetRandomSeed(seedForFirstIntnValue(t, state, 31, 15))
	affected := state.resolveCombatInFlamHur(source.EntityIndex,
		[]references.Position{state.CombatScene.GetCombatantPosition(target)})
	require.Len(t, affected, 1)
	assert.Less(t, character.GetCurrentHp(), hpBefore)
	assert.True(t, state.CombatScene.isCombatPCSleeping(target))
	assert.True(t, character.IsAsleep())
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, CombatFeedbackDamage, mock.CombatEvents[0].Result)
}

func TestUnguardedCampingSleeperWakesOnDirectedAttack(t *testing.T) {
	state, mock := newCampingAmbushTestState(t)
	installAutomaticAnyKey(state)
	require.True(t, state.ActionHoleUpLargeMapWithDebug(6, NoGuardAssigned, false, true))
	runCampingWorkflowToCompletion(t, state)
	require.NoError(t, state.ResolvePendingOverworldContact())
	target := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
	other := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1}
	require.True(t, state.CombatScene.isCombatPCSleeping(target))
	require.True(t, state.PartyState.GetCharacter(0).IsAsleep())
	monster := state.CombatScene.MonsterManager.GetMonster(0)
	position := state.CombatScene.GetCombatantPosition(target)
	monster.SetPosition(references.Position{X: position.X + 1, Y: position.Y})
	current := state.CombatScene.GetCurrentTurnIdentity()
	mock.Reset()
	require.True(t, state.monsterMeleeAttack(monster, 0,
		nearestPCResult{found: true, identity: target, pcIndex: 0, distance: 1}))
	assertExactAttackWake(t, state, mock, target, other, current)
}
