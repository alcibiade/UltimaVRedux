package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestUseCommitsRetainExactCombatOwnerAndContext(t *testing.T) {
	changes := []struct {
		name   string
		change func(*GameState, CombatTurnIdentity)
	}{
		{"clone of same roster member", func(g *GameState, clone CombatTurnIdentity) {
			g.CombatScene.CurrentTurnIdentity = clone
		}},
		{"replacement encounter with same identity", func(g *GameState, _ CombatTurnIdentity) {
			replacement := *g.CombatScene
			g.CombatScene = &replacement
		}},
		{"later dispatch of same actor", func(g *GameState, _ CombatTurnIdentity) {
			g.CombatScene.nextPossessionTurn++
		}},
		{"location", func(g *GameState, _ CombatTurnIdentity) {
			g.MapState.PlayerLocation.Position.X++
		}},
		{"encounter ended", func(g *GameState, _ CombatTurnIdentity) {
			g.CombatScene = nil
		}},
		{"quantity increased", func(g *GameState, _ CombatTurnIdentity) {
			g.PartyState.Inventory.Potions.Set(references.Yellow, 2)
			g.PartyState.Inventory.QuestItems.Set(references.HMSCape, 2)
		}},
	}
	for _, change := range changes {
		t.Run(change.name, func(t *testing.T) {
			state, _, clone := newSyntheticExactIdentityCastUseState(t)
			state.PartyState.Inventory.Potions.Set(references.Yellow, 1)
			state.PartyState.Inventory.QuestItems.Set(references.HMSCape, 1)
			potion, review := state.BeginPotionUse(references.Yellow, UsageCombat)
			require.True(t, review.Ready)
			cape, _ := state.BeginHMSCapeUse()
			require.NotNil(t, cape)
			change.change(state, clone)
			inventory := state.PartyState.Inventory.LockerSnapshot()
			rngCalls := state.GetRNG().Calls()
			sequence := state.CastUseCommitSequence()

			outcome, committed := state.CommitPotionUse(potion)
			require.False(t, committed)
			require.Nil(t, outcome)
			require.Equal(t, HMSCapeUseStale, state.CommitHMSCapeUse(cape))
			require.False(t, state.IsHMSCapeActivated())
			require.Equal(t, inventory, state.PartyState.Inventory.LockerSnapshot())
			require.Equal(t, rngCalls, state.GetRNG().Calls())
			require.Equal(t, sequence, state.CastUseCommitSequence())
		})
	}
}

func TestPaidPotionEffectDoesNotTransferToReplacementEncounter(t *testing.T) {
	state, actor, _ := newSyntheticExactIdentityCastUseState(t)
	state.PartyState.Inventory.Potions.Set(references.Yellow, 1)
	commit, _ := state.BeginPotionUse(references.Yellow, UsageCombat)
	outcome, committed := state.CommitPotionUse(commit)
	require.True(t, committed)
	replacement := *state.CombatScene
	state.CombatScene = &replacement
	rngCalls := state.GetRNG().Calls()
	sequence := state.CastUseCommitSequence()

	require.Equal(t, PotionEffectNoEffect, state.CommitPotionEffect(outcome, actor.EntityIndex))
	require.Zero(t, state.PartyState.Inventory.Potions.Get(references.Yellow), "paid failure is not refunded")
	require.Equal(t, sequence, state.CastUseCommitSequence())
	require.Equal(t, rngCalls, state.GetRNG().Calls(), "stale effects cannot draw substitution RNG")
}

func TestHMSCapeConfirmationRetainsExplorationLocation(t *testing.T) {
	state, _ := newUseCommitTestState(t)
	state.PartyState.Inventory.QuestItems.Set(references.HMSCape, 1)
	commit, _ := state.BeginHMSCapeUse()
	state.MapState.PlayerLocation.Position.X++
	sequence := state.CastUseCommitSequence()
	require.Equal(t, HMSCapeUseStale, state.CommitHMSCapeUse(commit))
	require.False(t, state.IsHMSCapeActivated())
	require.Equal(t, sequence, state.CastUseCommitSequence())
}
