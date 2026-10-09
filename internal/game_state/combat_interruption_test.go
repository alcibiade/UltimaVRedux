package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatInterruptionPublishesOnceAtEitherPhysicalRejectionBoundary(t *testing.T) {
	for _, beforeAim := range []bool{false, true} {
		t.Run(map[bool]string{false: "confirmation recheck", true: "before aiming"}[beforeAim], func(t *testing.T) {
			state, callbacks, target, _, _ := setupPlayerRangedAttackTest(t, references.Bow, references.MonsterOrc, 3)
			targeting := state.CombatScene.AttackTargeting
			engagerPosition := references.Position{X: targeting.Origin.X + 1, Y: targeting.Origin.Y}
			_, engagerIndex := addControlledTestMonster(state, references.MonsterTroll, engagerPosition, CombatMonsterLoyaltyMonster)
			engager := CombatTurnIdentity{EntityIndex: engagerIndex}
			state.rememberMeleeEngager(targeting.Attacker, engager)
			state.PartyState.Inventory.Equipment.Set(references.Arrows, 2)
			inventory, hp, date := state.PartyState.Inventory.LockerSnapshot(), target.CurrentHP(), state.DateTime
			state.SetRandomSeed(666)
			callbacks.Reset()
			// Forecast/preview may classify interference but must not record an
			// attempted action, play its sound, advance its source, or draw RNG.
			for range 3 {
				preview := state.PreviewCombatAttack()
				require.NotNil(t, preview.Aim)
				assert.Equal(t, CombatShotInterference, preview.Aim.Status)
			}
			assert.Empty(t, callbacks.CombatEvents)
			assert.Empty(t, callbacks.Messages)
			var result CombatAttackTargetResult
			if beforeAim {
				result = state.SkipUnavailableCombatAttackSources()
			} else {
				result = state.ConfirmCombatAttackTarget()
			}
			assert.Equal(t, CombatAttackSequenceComplete, result.Outcome)
			assert.Nil(t, result.Launch)
			assert.Nil(t, state.CombatScene.pendingAttack)
			require.Len(t, callbacks.CombatEvents, 1)
			event := callbacks.CombatEvents[0]
			assert.Equal(t, CombatEventInterruption, event.Kind)
			assert.Equal(t, CombatInterruptedAttack, event.InterruptedAction)
			assert.Equal(t, references.Bow, event.Diagnostics.Source)
			assert.Equal(t, combatFeedbackIdentity(engager, engagerPosition), event.Attacker)
			assert.Equal(t, combatFeedbackIdentity(targeting.Attacker, targeting.Origin), event.Target)
			assert.Equal(t, []string{"Troll interferes!"}, callbacks.Messages)
			assert.Equal(t, inventory, state.PartyState.Inventory.LockerSnapshot())
			assert.Equal(t, hp, target.CurrentHP())
			assert.Equal(t, date, state.DateTime)
			assert.Zero(t, state.GetRNG().Calls())
			assert.Empty(t, callbacks.SoundEffectsPlayed, "the admitted event owns audio in GameScene")
			state.SkipUnavailableCombatAttackSources()
			state.ConfirmCombatAttackTarget()
			require.Len(t, callbacks.CombatEvents, 1, "a completed source cannot publish again")
		})
	}
}

func TestCombatNonInterferingAttackNeverPublishesInterruption(t *testing.T) {
	state, callbacks, _, _, _ := setupPlayerRangedAttackTest(t, references.Bow, references.MonsterOrc, 3)
	state.PartyState.Inventory.Equipment.Set(references.Arrows, 2)
	assert.Equal(t, CombatAttackTargetIgnored, state.SkipUnavailableCombatAttackSources().Outcome)
	assert.Empty(t, callbacks.CombatEvents)
	result := state.ConfirmCombatAttackTarget()
	require.NotNil(t, result.Launch)
	state.ResolvePendingCombatAttack()
	for _, event := range callbacks.CombatEvents {
		assert.NotEqual(t, CombatEventInterruption, event.Kind)
	}
}
