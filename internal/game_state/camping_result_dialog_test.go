package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampingChoosesShortResultOrWandererPresentation(t *testing.T) {
	for _, wanderer := range []bool{false, true} {
		state, _ := newCampingAmbushTestState(t)
		installAutomaticAnyKey(state)
		var results []string
		conversations := 0
		state.SystemCallbacks.Screen.CreateAndPushCampingResultDialog = func(message string) { results = append(results, message) }
		state.SystemCallbacks.Screen.CreateAndPushMessageDialog = func() { conversations++ }
		require.True(t, state.ActionHoleUpLargeMapWithDebug(1, NoGuardAssigned, wanderer, false))
		runCampingWorkflowToCompletion(t, state)
		if wanderer {
			assert.Empty(t, results)
			assert.Equal(t, 1, conversations)
		} else {
			assert.Equal(t, []string{"No effect..."}, results)
			assert.Zero(t, conversations)
		}
	}
}
