package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestConsumedBritanniaTurnResolvesCodexGuardianSynchronously(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.FloorNumber(references.OVERWORLD)).
		WithPlayerAt(codexGuardianX, codexGuardianY).
		WithSystemCallbacks().
		Build()
	state.DebugOptions.MonsterGen = false
	callbacks.Reset()

	state.completeSurfaceCycleWithActorPhase(references.LargeMapType, 2, false)

	assert.Equal(t, references.Position{X: codexGuardianX, Y: codexGuardianY + 1},
		state.MapState.PlayerLocation.Position)
	assert.Contains(t, callbacks.Messages, "\"Thou art not upon a Sacred Quest!\nPassage denied!\"")
	assert.True(t, callbacks.SoundEffectWasPlayed(SoundGuardianBlock))
	assert.False(t, state.EventQueueManager.HasWorkflow("codex_guardian"))
}

func TestCodexGuardianPassRepeatsOnEveryEligibleConsumedTurn(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.FloorNumber(references.OVERWORLD)).
		WithPlayerAt(codexGuardianX, codexGuardianY).
		WithSystemCallbacks().
		Build()
	state.DebugOptions.MonsterGen = false
	state.SetShrineQuestActive(0, true)
	callbacks.Reset()

	state.completeSurfaceCycleWithActorPhase(references.LargeMapType, 2, false)
	state.completeSurfaceCycleWithActorPhase(references.LargeMapType, 2, false)

	assert.Equal(t, references.Position{X: codexGuardianX, Y: codexGuardianY},
		state.MapState.PlayerLocation.Position)
	assert.Equal(t, 2, countMatchingMessages(callbacks.Messages, "\"Pass, Seeker!\""))
	assert.False(t, callbacks.SoundEffectWasPlayed(SoundGuardianBlock))
}

func TestCodexGuardianRequiresExactBritanniaOverworldCheckpoint(t *testing.T) {
	tests := []struct {
		name     string
		floor    references.FloorNumber
		position references.Position
	}{
		{name: "Underworld floor", floor: references.FloorNumber(references.UNDERWORLD), position: references.Position{X: codexGuardianX, Y: codexGuardianY}},
		{name: "Other coordinate", floor: references.FloorNumber(references.OVERWORLD), position: references.Position{X: codexGuardianX + 1, Y: codexGuardianY}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithFloor(test.floor).
				WithPlayerAt(int(test.position.X), int(test.position.Y)).
				WithSystemCallbacks().
				Build()
			state.DebugOptions.MonsterGen = false
			callbacks.Reset()

			state.completeSurfaceCycleWithActorPhase(references.LargeMapType, 2, false)

			require.Equal(t, test.position, state.MapState.PlayerLocation.Position)
			assert.NotContains(t, callbacks.Messages, "\"Pass, Seeker!\"")
			assert.False(t, callbacks.SoundEffectWasPlayed(SoundGuardianBlock))
		})
	}
}
