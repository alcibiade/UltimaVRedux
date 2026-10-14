package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestCombatCrownUseProtectsAgainstDaemonMagicWithoutBlockingMelee(t *testing.T) {
	state, callbacks, daemon, daemonIndex := setupHostileDaemonTest(t)
	require.NotNil(t, state)
	state.PartyState.Inventory.QuestItems.Set(references.Crown, 1)
	item := requireUsableItem(t, state.GetUsableItemsForContext(UsageCombat), references.Crown)
	require.Empty(t, item.UnavailableReason)
	sequence := state.CastUseCommitSequence()
	require.True(t, state.UseItem(item), "exercise authoritative Use, not the toggle helper")
	require.Equal(t, sequence+1, state.CastUseCommitSequence())

	from, to := clearRangeNinePositions(t, state.CombatScene.CombatMapRef)
	daemon.SetPosition(from)
	state.CombatScene.PartyPositions[0] = to
	rngCalls := state.GetRNG().Calls()
	require.False(t, state.executeHostileMonsterSpecialMove(daemon, daemonIndex))
	require.Equal(t, rngCalls, state.GetRNG().Calls(), "Crown suppresses special attempts before their RNG")
	nearest := gDaemonNearestPC(state, daemon, daemonIndex)
	require.False(t, state.monsterAttackWithRNG(daemon, daemonIndex, nearest, newSequenceRNG(0)))
	require.Empty(t, callbacks.CombatEvents)

	state.CombatScene.PartyPositions[0] = references.Position{X: from.X + 1, Y: from.Y}
	nearest = gDaemonNearestPC(state, daemon, daemonIndex)
	state.SetRandomSeed(2)
	require.True(t, state.monsterAttack(daemon, daemonIndex, nearest))
	require.NotEmpty(t, callbacks.CombatEvents)
	require.EqualValues(t, 1, state.PartyState.Inventory.QuestItems.Get(references.Crown))

	require.True(t, state.UseItem(item))
	require.False(t, state.IsCrownActive())
	require.Equal(t, sequence+2, state.CastUseCommitSequence())
}
