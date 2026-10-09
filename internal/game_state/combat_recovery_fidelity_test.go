package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatDefeatRequestsRecoveryOnceAfterMapRestoration(t *testing.T) {
	state, source, _ := newSyntheticExactIdentityCastUseState(t)
	returnLocation := references.PlayerLocation{
		Location: references.Britain,
		Floor:    1,
		Position: references.Position{X: 12, Y: 13},
	}
	state.CombatScene.PreCombatPlayerLocation = returnLocation
	character := state.PartyState.GetCharacter(source.EntityIndex)
	character.SetHp(0)
	character.SetStatus(party_state.Dead)
	deathRequests := 0
	state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() {
		deathRequests++
		assert.Nil(t, state.CombatScene, "combat must be cleared before recovery review")
		assert.Equal(t, returnLocation, state.MapState.PlayerLocation)
	}

	state.EndCombat(CombatDefeat)

	require.Equal(t, 1, deathRequests)
	assert.False(t, state.combatRecoveryReviewDeferred)
}

func TestPresentedLethalMonsterStepRequestsRecoveryAfterCombatRestoration(t *testing.T) {
	state, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if state == nil {
		return
	}
	returnLocation := state.CombatScene.PreCombatPlayerLocation
	activeIndex := state.CombatScene.GetActiveCombatantIndex()
	for index := range state.CombatScene.PartyActive {
		character := state.PartyState.GetCharacter(index)
		if index == activeIndex {
			character.SetStatus(party_state.Good)
			character.SetHp(1)

			continue
		}
		character.SetHp(0)
		character.SetStatus(party_state.Dead)
		if state.CombatScene.PartyActive[index] {
			state.CombatScene.PartyActive[index] = false
			state.CombatScene.TurnQueue.MarkDead(index, true)
		}
	}
	monster.EnemyRef().Dexterity = 100
	monster.EnemyRef().Damage = combatInstantKillDamage
	forceNextMonsterTurnForTest(t, state, 0)
	deathRequests := 0
	state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() {
		deathRequests++
		assert.Nil(t, state.CombatScene)
		assert.Equal(t, returnLocation, state.MapState.PlayerLocation)
		assert.False(t, state.combatRecoveryReviewDeferred)
	}

	state.BeginCombatTurns(CombatTurnsPresented)
	step, scheduled := state.ReviewScheduledCombatStep()
	require.True(t, scheduled)
	require.Equal(t, CombatPresentationMonsterTurn, step.Kind)
	_, err := state.ResolveScheduledCombatStep(step.Token)
	require.NoError(t, err)
	assert.True(t, state.PartyState.GetCharacter(activeIndex).IsDead())
	assert.NotNil(t, state.CombatScene, "presentation completion owns the combat transition")
	assert.Zero(t, deathRequests, "recovery cannot start while combat remains active")

	require.NoError(t, state.CompleteScheduledCombatStep(step.Token))
	assert.Nil(t, state.CombatScene)
	assert.Equal(t, 1, deathRequests)
}
