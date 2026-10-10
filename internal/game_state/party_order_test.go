package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestSwapPartyMembersKeepsActiveSlotAndRefreshesStats(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	require.True(t, state.SetActivePlayer(1))
	first := state.PartyState.GetCharacter(1).GetNameAsString()
	second := state.PartyState.GetCharacter(2).GetNameAsString()
	callbacks.Reset()

	require.NoError(t, state.SwapPartyMembers(1, 2))

	assert.Equal(t, 1, state.GetActivePlayerIndex())
	assert.Equal(t, second, state.GetActivePlayer().GetNameAsString())
	assert.Equal(t, second, state.PartyState.GetCharacter(1).GetNameAsString())
	assert.Equal(t, first, state.PartyState.GetCharacter(2).GetNameAsString())
	assert.Equal(t, 1, callbacks.StatsDisplayUpdated)
}
