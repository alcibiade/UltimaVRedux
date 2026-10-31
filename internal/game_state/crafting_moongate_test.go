package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestMoongateAnnouncementsAreSilentOnDeedMaps(t *testing.T) {
	announce := func(state *GameState, callbacks *MockSystemCallbacks, hour byte) {
		callbacks.Reset()
		state.DateTime.Hour = hour
		state.UpdateMoongateTransition()
		runNamedWorkflow(t, state, moongateTransitionWorkflow)
	}

	state, callbacks, _ := newBuildFixture(t)
	announce(state, callbacks, 20)
	assert.False(t, callbacks.MessagesContain("shimmering blue door"))
	assert.Equal(t, MoongateStageFull, state.MoongateStage(), "the shared stage still grows")
	announce(state, callbacks, 5)
	assert.False(t, callbacks.MessagesContain("fades from view"))
	assert.Equal(t, MoongateStageHidden, state.MoongateStage())

	overworld, overworldCallbacks := newDeedFixture(t, true)
	overworld.DateTime.Hour = 10
	overworld.initializeMoongateStage()
	announce(overworld, overworldCallbacks, 20)
	assert.True(t, overworldCallbacks.MessagesContain("Instantly, a shimmering blue door springs up!"))
}

func TestAMoonstoneCannotBeBuriedOnADeedMap(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	state.PartyState.Inventory.Moonstones.IncrementBy(references.Moonstone(0), 1)

	assert.False(t, state.CommitMoonstoneBurial(references.Moonstone(0)))
	assert.False(t, state.useMoonstone(references.Moonstone(0), func() {}))
	assert.True(t, state.PartyState.Inventory.Moonstones.HasSome(references.Moonstone(0)))
}
