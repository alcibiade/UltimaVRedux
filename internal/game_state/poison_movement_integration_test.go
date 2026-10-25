package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestPoisonDamageDuringConsumedSurfaceUpdate(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Set up party with poisoned character
	character := gs.PartyState.GetCharacter(0)
	character.SetHp(10) // Set HP so we can track damage
	character.SetStatus(party_state.Poisoned)

	// Reset mock callbacks to clear initialization messages
	mockCallbacks.Reset()

	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)

	// Verify poison damage was applied (1 HP per poisoned character)
	assert.Equal(t, uint16(9), character.GetCurrentHp(),
		"Poisoned character should take 1 HP damage during movement")

	// Verify character is still poisoned (didn't die)
	assert.True(t, character.IsPoisoned(),
		"Character should still be poisoned after taking damage")

	// Verify visual feedback was triggered (green flash event queue)
	// The ShowPoisonFeedback() creates an event queue named "poison_feedback"
	assert.True(t, gs.EventQueueManager.HasFeedback(poisonFeedbackQueueName),
		"Poison visual feedback should trigger event queue")
	assert.Empty(t, mockCallbacks.TransientIconCalls, "recurring poison damage is not poison onset")
}

func TestRepeatedConsumedUpdatesCoalescePoisonFeedback(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	character := gs.PartyState.GetCharacter(0)
	character.SetHp(10)
	character.SetStatus(party_state.Poisoned)
	mockCallbacks.Reset()

	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)
	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)

	assert.Equal(t, uint16(8), character.GetCurrentHp())
	require.True(t, gs.EventQueueManager.HasFeedback(poisonFeedbackQueueName))

	gs.EventQueueManager.UpdateFeedback()
	gs.ElapsedMs += poisonFlashDurationMs
	for range 3 {
		gs.EventQueueManager.UpdateFeedback()
	}
	assert.False(t, gs.EventQueueManager.HasFeedback(poisonFeedbackQueueName))
	assert.Equal(t, 1, countEqual(mockCallbacks.Messages, "Oof!"))
	assert.Equal(t, 1, countEqual(mockCallbacks.SoundEffectsPlayed, SoundWhiteNoiseDamage))

	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)
	assert.Equal(t, uint16(7), character.GetCurrentHp())
	assert.True(t, gs.EventQueueManager.HasFeedback(poisonFeedbackQueueName))
}

func TestPoisonDeathDuringConsumedSurfaceUpdate(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Set up party with poisoned character at 1 HP (will die from poison)
	character := gs.PartyState.GetCharacter(0)
	character.SetHp(1)
	character.SetStatus(party_state.Poisoned)

	// Reset mock callbacks to clear initialization messages
	mockCallbacks.Reset()

	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)

	// Verify character died from poison
	assert.Equal(t, uint16(0), character.GetCurrentHp(),
		"Character should have 0 HP after poison damage")
	assert.True(t, character.IsDead(),
		"Character should be marked as dead")

	// Verify death message was shown
	assert.True(t, len(mockCallbacks.Messages) > 0,
		"Death message should be displayed")
	assert.Contains(t, mockCallbacks.Messages[0], "Alas!",
		"Death message should contain 'Alas!'")
	assert.Contains(t, mockCallbacks.Messages[0], "died",
		"Death message should mention death")

	// Verify death feedback was triggered (not poison feedback)
	assert.True(t, gs.EventQueueManager.HasFeedback(deathFeedbackQueueName),
		"Death visual feedback should trigger event queue")
}

func TestMultiplePoisonedCharactersDuringConsumedSurfaceUpdate(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Set up multiple poisoned characters
	char1 := gs.PartyState.GetCharacter(0)
	char1.SetHp(10)
	char1.SetStatus(party_state.Poisoned)

	char2 := gs.PartyState.GetCharacter(1)
	char2.SetHp(15)
	char2.SetStatus(party_state.Poisoned)

	char3 := gs.PartyState.GetCharacter(2)
	char3.SetHp(20)
	char3.SetStatus(party_state.Good) // Not poisoned

	// Reset mock callbacks
	mockCallbacks.Reset()

	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)

	// Verify all poisoned characters took damage
	assert.Equal(t, uint16(9), char1.GetCurrentHp(),
		"First poisoned character should take 1 HP damage")
	assert.Equal(t, uint16(14), char2.GetCurrentHp(),
		"Second poisoned character should take 1 HP damage")
	assert.Equal(t, uint16(20), char3.GetCurrentHp(),
		"Non-poisoned character should take no damage")

	// Verify poison feedback was triggered
	assert.True(t, gs.EventQueueManager.HasFeedback(poisonFeedbackQueueName),
		"Poison visual feedback should trigger event queue")
}

func TestConsumedSurfacePoisonSkipsInactiveRoster(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	inactive := gs.PartyState.GetCharacter(party_state.MaxActiveParty)
	inactive.PartyMap = party_state.PartyMapUnrecruited
	inactive.SetHp(10)
	inactive.SetStatus(party_state.Poisoned)

	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)

	assert.Equal(t, uint16(10), inactive.GetCurrentHp())
	assert.Equal(t, party_state.Poisoned, inactive.GetStatus())
}

func TestNoPoisonNoDamageDuringConsumedSurfaceUpdate(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Ensure no characters are poisoned
	for i := 0; i < gs.PartyState.GetCharacterCount(); i++ {
		char := gs.PartyState.GetCharacter(i)
		char.SetHp(10)
		char.SetStatus(party_state.Good)
	}

	// Reset mock callbacks
	mockCallbacks.Reset()

	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)

	// Verify no damage was dealt
	for i := 0; i < gs.PartyState.GetCharacterCount(); i++ {
		char := gs.PartyState.GetCharacter(i)
		assert.Equal(t, uint16(10), char.GetCurrentHp(),
			"No character should take damage when no one is poisoned")
	}

	// Verify no messages were shown
	assert.Equal(t, 0, len(mockCallbacks.Messages),
		"No messages should be shown when no poison effects occur")

	// Verify no event queues were triggered
	assert.False(t, gs.EventQueueManager.HasFeedback(poisonFeedbackQueueName),
		"No poison feedback queue should exist without poison")
	assert.False(t, gs.EventQueueManager.HasFeedback(deathFeedbackQueueName),
		"No death feedback queue should exist without deaths")
}
