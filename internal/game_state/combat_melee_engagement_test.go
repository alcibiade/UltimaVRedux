package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestMonsterMeleeRemembersEngagerBeforeMiss(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	target := findNearestPC(monster.Position(), gs.CombatScene)
	require.True(t, target.found)
	require.Equal(t, monsterAdjacentDistance, target.distance)
	require.False(t, monster.HasAbility(references.PoisonAtRange) || monster.HasAbility(references.InfectWithPlague))

	defender := gs.PartyState.GetCharacter(target.identity.EntityIndex)
	// Original combat damage rules use STR for this bludgeoning Orc, not reference DEX.
	// See ref:EQUIP-0016.
	require.True(t, monster.HasAbility(references.Bludgeons))
	missSeed := uint64(0)
	foundMissSeed := false
	for seed := uint64(0); seed < 1000; seed++ {
		gs.SetRandomSeed(seed)
		if CombatHitCheck(monster.EnemyRef().Strength,
			CombatGetEffectiveDex(int(defender.Dexterity), gs.CombatScene.isCombatPCSleeping(target.identity)), false,
			gs.AsRandomProvider()).Outcome == CombatHitMiss {
			missSeed = seed
			foundMissSeed = true
			break
		}
	}
	require.True(t, foundMissSeed)
	gs.SetRandomSeed(missSeed)
	mock.Reset()

	require.True(t, gs.monsterMeleeAttack(monster, 0, target))
	require.NotEmpty(t, mock.CombatEvents)
	assert.Equal(t, CombatFeedbackMiss, mock.CombatEvents[len(mock.CombatEvents)-1].Result)
	engager, found := gs.rememberedMeleeEngager(target.identity)
	assert.True(t, found)
	assert.Equal(t, CombatTurnIdentity{IsPartyMember: false, EntityIndex: 0}, engager)
}

func TestRememberMeleeEngagerReplacesWithoutFallingBack(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	target := CombatTurnIdentity{IsPartyMember: true, EntityIndex: firstActiveCombatPC(gs)}
	valid := CombatTurnIdentity{IsPartyMember: false, EntityIndex: 0}
	inactiveReplacement := CombatTurnIdentity{IsPartyMember: false, EntityIndex: 99}
	require.Equal(t, monsterAdjacentDistance,
		integerDistance(monster.Position(), gs.CombatScene.GetCombatantPosition(target)))

	gs.rememberMeleeEngager(target, valid)
	requireRememberedMeleeEngager(t, gs, target, valid)
	gs.rememberMeleeEngager(target, inactiveReplacement)

	requireRememberedMeleeEngager(t, gs, target, inactiveReplacement)
	_, found := gs.validRememberedMeleeEngager(target)
	assert.False(t, found, "an invalid latest attacker must not reveal the previously remembered attacker")
}

func TestCombatCastInterferencePreservesExactActorResourcesAndRNG(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	target := findNearestPC(monster.Position(), gs.CombatScene)
	require.True(t, target.found)
	gs.CombatScene.CurrentTurnIdentity = target.identity
	gs.CombatScene.currentTurnIdentitySet = true
	attacker := CombatTurnIdentity{EntityIndex: 0}
	gs.rememberMeleeEngager(target.identity, attacker)
	caster := gs.PartyState.GetCharacter(target.identity.EntityIndex)
	caster.Class = party_state.Wizard
	caster.Level = 8
	caster.SetMp(8)
	gs.PartyState.Inventory.Spells.Set(references.InAn, 1)
	gs.SetRandomSeed(99)
	mock.Reset()

	assert.True(t, gs.CheckCombatCastInterference())
	assert.Equal(t, target.identity, gs.CombatScene.GetCurrentTurnIdentity())
	assert.Equal(t, byte(8), caster.GetCurrentMp())
	assert.Equal(t, uint16(1), gs.PartyState.Inventory.Spells.Get(references.InAn))
	assert.Equal(t, 30, gs.RollDice(30), "interference must not advance gameplay RNG")
	assert.Equal(t, []string{gs.combatIdentityName(attacker) + " interferes!"}, mock.Messages)
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, CombatEventInterruption, mock.CombatEvents[0].Kind)
	assert.Equal(t, CombatInterruptedCast, mock.CombatEvents[0].InterruptedAction)
	assert.Equal(t, combatFeedbackIdentity(target.identity, gs.combatTurnPosition(target.identity)), mock.CombatEvents[0].Target)
}

func TestValidRememberedMeleeEngagerValidation(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*GameState, *CombatMonster, CombatTurnIdentity)
	}{
		{name: "adjacent"},
		{name: "nonadjacent", setup: func(_ *GameState, monster *CombatMonster, _ CombatTurnIdentity) {
			monster.SetPosition(references.Position{X: 0, Y: 0})
		}},
		{name: "asleep", setup: func(_ *GameState, monster *CombatMonster, _ CombatTurnIdentity) {
			monster.SetAsleep(true)
		}},
		{name: "dead", setup: func(_ *GameState, monster *CombatMonster, _ CombatTurnIdentity) {
			monster.SetCurrentHP(0)
		}},
		{name: "inactive", setup: func(_ *GameState, monster *CombatMonster, _ CombatTurnIdentity) {
			monster.SetActive(false)
		}},
		{name: "fled", setup: func(gs *GameState, _ *CombatMonster, attacker CombatTurnIdentity) {
			gs.CombatScene.TurnQueue.MarkFled(attacker.EntityIndex, false)
		}},
		{name: "allegiance changed", setup: func(_ *GameState, monster *CombatMonster, _ CombatTurnIdentity) {
			monster.SetLoyalty(CombatMonsterLoyaltyParty)
		}},
		{name: "time stopped", setup: func(gs *GameState, _ *CombatMonster, _ CombatTurnIdentity) {
			require.True(t, gs.ActivateTimeStopEffect(10))
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
			if gs == nil {
				return
			}
			target := CombatTurnIdentity{IsPartyMember: true, EntityIndex: firstActiveCombatPC(gs)}
			attacker := CombatTurnIdentity{IsPartyMember: false, EntityIndex: 0}
			gs.rememberMeleeEngager(target, attacker)
			if test.setup != nil {
				test.setup(gs, monster, attacker)
			}

			actual, found := gs.validRememberedMeleeEngager(target)
			if test.name == "adjacent" {
				assert.True(t, found)
				assert.Equal(t, attacker, actual)
			} else {
				assert.False(t, found)
			}
		})
	}
}

func TestRememberedMeleeEngagerUsesExactCloneIdentity(t *testing.T) {
	gs, callbacks, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	partyIndex := firstActiveCombatPC(gs)
	clone := gs.CombatScene.addPCCloneFromSource(
		CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex},
		partyIndex,
		references.Position{X: monster.Position().X + 1, Y: monster.Position().Y},
		NewCombatPCProvider(gs.PartyState.GetCharacter(partyIndex)),
	)
	attacker := CombatTurnIdentity{IsPartyMember: false, EntityIndex: 0}
	gs.rememberMeleeEngager(clone, attacker)

	_, normalFound := gs.rememberedMeleeEngager(CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex})
	assert.False(t, normalFound)
	requireRememberedMeleeEngager(t, gs, clone, attacker)
	gs.CombatScene.CurrentTurnIdentity = clone
	gs.CombatScene.currentTurnIdentitySet = true
	calls := gs.GetRNG().Calls()
	callbacks.Reset()
	require.True(t, gs.CheckCombatCastInterference())
	require.Len(t, callbacks.CombatEvents, 1)
	assert.Equal(t, combatFeedbackIdentity(clone, gs.combatTurnPosition(clone)), callbacks.CombatEvents[0].Target)
	assert.NotZero(t, callbacks.CombatEvents[0].Target.Slot, "feedback must identify the interrupted clone, not its roster source")
	assert.Equal(t, calls, gs.GetRNG().Calls())
}

func TestRememberedMeleeEngagerClearsAtActionAndSkipCompletion(t *testing.T) {
	t.Run("completed action", func(t *testing.T) {
		gs, _ := setupCombatStatusTest(t)
		if gs == nil {
			return
		}
		target := CombatTurnIdentity{IsPartyMember: true, EntityIndex: firstActiveCombatPC(gs)}
		gs.CombatScene.CurrentTurnIdentity = target
		gs.CombatScene.currentTurnIdentitySet = true
		gs.rememberMeleeEngager(target, CombatTurnIdentity{IsPartyMember: false, EntityIndex: 99})

		gs.FinishCombatAction()

		_, found := gs.rememberedMeleeEngager(target)
		assert.False(t, found)
	})

	t.Run("skipped action", func(t *testing.T) {
		gs, _ := setupCombatStatusTest(t)
		if gs == nil {
			return
		}
		target := CombatTurnIdentity{IsPartyMember: true, EntityIndex: firstActiveCombatPC(gs)}
		gs.setCombatPCSleeping(target, true)
		forceNextCombatActor(gs.CombatScene.TurnQueue, target.EntityIndex, true)
		gs.CombatScene.CurrentTurnIdentity = CombatTurnIdentity{IsPartyMember: true, EntityIndex: (target.EntityIndex + 1) % party_state.MaxActiveParty}
		gs.CombatScene.currentTurnIdentitySet = true
		gs.rememberMeleeEngager(target, CombatTurnIdentity{IsPartyMember: false, EntityIndex: 99})
		gs.SetRandomSeed(findCombatWakeSeed(t, gs, combatPCWakeRollMaximum, func(roll int) bool {
			return roll >= combatPCWakeThreshold
		}))

		gs.AdvanceCombatTurn()

		_, found := gs.rememberedMeleeEngager(target)
		assert.False(t, found)
	})
}

func requireRememberedMeleeEngager(
	t *testing.T,
	gs *GameState,
	target CombatTurnIdentity,
	expected CombatTurnIdentity,
) {
	t.Helper()
	actual, found := gs.rememberedMeleeEngager(target)
	require.True(t, found)
	assert.Equal(t, expected, actual)
}
