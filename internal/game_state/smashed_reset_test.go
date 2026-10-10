package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestFullSmallMapTransitionResetsSmashed(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithSystemCallbacks().
		Build()
	state.smashed = 7
	assert.Equal(t, PersistentEffectIntoxication, state.GetPersistentEffectStatuses()[0].ID)

	state.TransitionToLocation(
		references.Britain,
		0,
		references.Position{X: 15, Y: 15},
		TransitionFullLocation,
		false,
	)

	assert.Zero(t, state.smashed)
	assert.Empty(t, state.GetPersistentEffectStatuses())
}

func TestSmallMapRefreshAndFloorTransitionPreserveSmashed(t *testing.T) {
	t.Run("UpdateSmallMap", func(t *testing.T) {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		state.smashed = 6

		state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)

		assert.Equal(t, 6, state.smashed)
		assert.Equal(t, PersistentEffectIntoxication, state.GetPersistentEffectStatuses()[0].ID)
	})

	t.Run("floor only", func(t *testing.T) {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle).
			WithFloor(0).
			WithSystemCallbacks().
			Build()
		state.smashed = 5

		state.TransitionToLocation(
			references.LordBritishsCastle,
			1,
			state.MapState.PlayerLocation.Position,
			TransitionFloorOnly,
			false,
		)

		assert.Equal(t, 5, state.smashed)
		assert.Equal(t, PersistentEffectIntoxication, state.GetPersistentEffectStatuses()[0].ID)
	})
}

func TestExecuteJailSentenceResetsSmashed(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	state.smashed = 4
	state.DateTime.Hour = 8

	state.ExecuteJailSentence()

	assert.Zero(t, state.smashed)
	assert.Empty(t, state.GetPersistentEffectStatuses())
	assert.Equal(t, references.Yew, state.MapState.PlayerLocation.Location)
}
