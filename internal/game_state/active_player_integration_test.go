package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestActivePlayerSelection_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Verify initial state - no active player
	assert.False(t, gs.HasActivePlayer())
	assert.Equal(t, -1, gs.GetActivePlayerIndex())
	assert.Nil(t, gs.GetActivePlayer())

	// Set up some characters in the party
	gs.PartyState.GetCharacter(0).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)
	copy(gs.PartyState.GetCharacter(0).Name[:], "Alice")

	gs.PartyState.GetCharacter(1).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(1).SetStatus(party_state.Good)
	copy(gs.PartyState.GetCharacter(1).Name[:], "Bob")

	// Test setting active player to first character
	t.Run("SetActivePlayer_FirstCharacter", func(t *testing.T) {
		mockCallbacks.Reset()

		success := gs.SetActivePlayer(0)
		assert.True(t, success)
		assert.True(t, gs.HasActivePlayer())
		assert.Equal(t, 0, gs.GetActivePlayerIndex())

		activePlayer := gs.GetActivePlayer()
		assert.NotNil(t, activePlayer)
		assert.Equal(t, "Alice", string(activePlayer.Name[:5]))

		// Verify UpdateStatsDisplay callback was called
		assert.True(t, mockCallbacks.StatsDisplayUpdated > 0)
	})

	// Test setting active player to second character
	t.Run("SetActivePlayer_SecondCharacter", func(t *testing.T) {
		mockCallbacks.Reset()

		success := gs.SetActivePlayer(1)
		assert.True(t, success)
		assert.True(t, gs.HasActivePlayer())
		assert.Equal(t, 1, gs.GetActivePlayerIndex())

		activePlayer := gs.GetActivePlayer()
		assert.NotNil(t, activePlayer)
		assert.Equal(t, "Bob", string(activePlayer.Name[:3]))

		// Verify UpdateStatsDisplay callback was called
		assert.True(t, mockCallbacks.StatsDisplayUpdated > 0)
	})

	// Test clearing active player
	t.Run("ClearActivePlayer", func(t *testing.T) {
		mockCallbacks.Reset()

		gs.ClearActivePlayer()
		assert.False(t, gs.HasActivePlayer())
		assert.Equal(t, -1, gs.GetActivePlayerIndex())
		assert.Nil(t, gs.GetActivePlayer())

		// Verify UpdateStatsDisplay callback was called
		assert.True(t, mockCallbacks.StatsDisplayUpdated > 0)
	})

	// Test invalid player indices
	t.Run("SetActivePlayer_InvalidIndices", func(t *testing.T) {
		mockCallbacks.Reset()

		// Debug: Check what character 2 actually looks like
		char2 := gs.PartyState.GetCharacter(2)
		t.Logf("Character 2: PartyStatus=%v, Name=%q", char2.PartyMap, string(char2.Name[:]))

		// Out of range indices
		assert.False(t, gs.SetActivePlayer(-2))
		assert.False(t, gs.SetActivePlayer(6))

		// Empty character slot - force it to be empty for this test
		gs.PartyState.GetCharacter(2).PartyMap = party_state.PartyMapUnrecruited
		assert.False(t, gs.SetActivePlayer(2)) // Character 2 not set up

		// No callbacks should be triggered for failed attempts
		assert.Equal(t, 0, mockCallbacks.StatsDisplayUpdated)
	})
}

func TestActivePlayerWithJimmy_Integration(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Set up a character in the party
	gs.PartyState.GetCharacter(0).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)
	copy(gs.PartyState.GetCharacter(0).Name[:], "Alice")

	// Test Jimmy command without active player
	t.Run("Jimmy_NoActivePlayer", func(t *testing.T) {

		// SelectCharacterForJimmy should use fallback logic
		selectedChar := gs.SelectCharacterForJimmy()
		assert.NotNil(t, selectedChar)
		assert.Equal(t, "Alice", string(selectedChar.Name[:5]))
	})

	// Test Jimmy command with active player
	t.Run("Jimmy_WithActivePlayer", func(t *testing.T) {

		// Set active player
		gs.SetActivePlayer(0)

		// SelectCharacterForJimmy should use active player
		selectedChar := gs.SelectCharacterForJimmy()
		assert.NotNil(t, selectedChar)
		assert.Equal(t, "Alice", string(selectedChar.Name[:5]))

		// Should be the same character as active player
		activePlayer := gs.GetActivePlayer()
		assert.Equal(t, activePlayer, selectedChar)
	})
}

func TestSelectCharacterForJimmyPoisonedEligibility(t *testing.T) {
	tests := []struct {
		name   string
		first  party_state.CharacterStatus
		second party_state.CharacterStatus
		active int
		want   int
	}{
		{name: "active poisoned over good first", first: party_state.Good, second: party_state.Poisoned, active: 1, want: 1},
		{name: "poisoned first fallback", first: party_state.Poisoned, second: party_state.Good, active: -1, want: 0},
		{name: "later poisoned fallback without good", first: party_state.Dead, second: party_state.Poisoned, active: -1, want: 1},
		{name: "all ineligible", first: party_state.Sleep, second: party_state.Charmed, active: -1, want: -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(references.Britain).
				WithSystemCallbacks().
				Build()
			state.ClearActivePlayer()
			state.PartyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
				character.SetStatus(party_state.Dead)
			})
			for index, status := range []party_state.CharacterStatus{test.first, test.second} {
				character := state.PartyState.GetCharacter(index)
				character.PartyMap = party_state.PartyMapActive
				character.SetMaxHp(100)
				character.SetHp(100)
				character.SetStatus(status)
				copy(character.Name[:], "Actor")
			}
			if test.active >= 0 {
				require.True(t, state.SetActivePlayer(test.active))
			}

			selected := state.SelectCharacterForJimmy()

			if test.want < 0 {
				assert.Nil(t, selected)
			} else {
				assert.Same(t, state.PartyState.GetCharacter(test.want), selected)
			}
		})
	}
}

func TestActivePlayerStateManagement_Integration(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Set up characters with different statuses
	gs.PartyState.GetCharacter(0).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)
	copy(gs.PartyState.GetCharacter(0).Name[:], "Alice")

	gs.PartyState.GetCharacter(1).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(1).SetStatus(party_state.Dead)
	copy(gs.PartyState.GetCharacter(1).Name[:], "Bob")

	gs.PartyState.GetCharacter(2).PartyMap = party_state.PartyMapUnrecruited
	gs.PartyState.GetCharacter(2).SetStatus(party_state.Good)
	copy(gs.PartyState.GetCharacter(2).Name[:], "Charlie")

	// Test validation of character status
	t.Run("ValidateCharacterStatus", func(t *testing.T) {
		// Good character in party should work
		assert.True(t, gs.SetActivePlayer(0))

		// Dead and sleeping characters cannot own active input.
		assert.False(t, gs.SetActivePlayer(1))
		gs.PartyState.GetCharacter(1).SetStatus(party_state.Sleep)
		assert.False(t, gs.SetActivePlayer(1))

		gs.PartyState.GetCharacter(0).SetHp(0)
		assert.False(t, gs.SetActivePlayer(0))
		gs.PartyState.GetCharacter(0).SetHp(gs.PartyState.GetCharacter(0).GetMaxHp())

		// Character not in party should fail
		assert.False(t, gs.SetActivePlayer(2))
	})

	// Test character name validation
	t.Run("ValidateCharacterName", func(t *testing.T) {
		// Character with empty name should fail
		gs.PartyState.GetCharacter(3).PartyMap = party_state.PartyMapActive
		gs.PartyState.GetCharacter(3).SetStatus(party_state.Good)
		// Explicitly clear the name (all zeros)
		for i := range gs.PartyState.GetCharacter(3).Name {
			gs.PartyState.GetCharacter(3).Name[i] = 0
		}

		assert.False(t, gs.SetActivePlayer(3))
	})
}

func TestActivePlayerSelectionClearsWhenCharacterBecomesUnavailable(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*party_state.PlayerCharacter)
	}{
		{name: "dead", mutate: func(character *party_state.PlayerCharacter) { character.SetStatus(party_state.Dead) }},
		{name: "asleep", mutate: func(character *party_state.PlayerCharacter) { character.SetStatus(party_state.Sleep) }},
		{name: "zero hp", mutate: func(character *party_state.PlayerCharacter) { character.SetHp(0) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.Britain).
				WithSystemCallbacks().
				Build()
			character := state.PartyState.GetCharacter(0)
			character.PartyMap = party_state.PartyMapActive
			character.SetStatus(party_state.Good)
			character.SetHp(character.GetMaxHp())
			copy(character.Name[:], "Alice")
			require.True(t, state.SetActivePlayer(0))
			callbacks.Reset()

			test.mutate(character)

			assert.False(t, state.HasActivePlayer())
			assert.Equal(t, -1, state.GetActivePlayerIndex())
			assert.Nil(t, state.GetActivePlayer())
			assert.Equal(t, 1, callbacks.StatsDisplayUpdated)
		})
	}
}

func TestActivePlayerCallbackIntegration_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Set up a character
	gs.PartyState.GetCharacter(0).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)
	copy(gs.PartyState.GetCharacter(0).Name[:], "Alice")

	// Test that callbacks are triggered correctly
	t.Run("CallbacksTriggered", func(t *testing.T) {
		mockCallbacks.Reset()

		// Setting active player should trigger UpdateStatsDisplay
		gs.SetActivePlayer(0)
		assert.Equal(t, 1, mockCallbacks.StatsDisplayUpdated)

		// Clearing active player should also trigger UpdateStatsDisplay
		gs.ClearActivePlayer()
		assert.Equal(t, 2, mockCallbacks.StatsDisplayUpdated)

		// Setting to same player should still trigger callback
		gs.SetActivePlayer(0)
		gs.SetActivePlayer(0)
		assert.Equal(t, 4, mockCallbacks.StatsDisplayUpdated)
	})

	// Test with nil SystemCallbacks (should not crash)
	t.Run("NilSystemCallbacks", func(t *testing.T) {
		originalCallbacks := gs.SystemCallbacks
		gs.SystemCallbacks = nil

		// Should not crash
		assert.True(t, gs.SetActivePlayer(0))
		gs.ClearActivePlayer()

		// Restore callbacks
		gs.SystemCallbacks = originalCallbacks
	})
}
