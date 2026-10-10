package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateGuardExtortionCountsActiveLivingPartyWithoutClamp(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	state.PartyState.ForEachCharacter(func(index int, character *party_state.PlayerCharacter) {
		switch index {
		case 0:
			character.SetStatus(party_state.Poisoned)
		case 1:
			character.MarkAsDead()
		default:
			character.PartyMap = party_state.PartyMapAtInn(references.Britain)
		}
	})

	assert.Equal(t, references.TributePerLivingMember, CalculateGuardExtortion(&state.PartyState))
	assert.Zero(t, CalculateGuardExtortion(nil))

	state.PartyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
		character.MarkAsDead()
	})
	assert.Zero(t, CalculateGuardExtortion(&state.PartyState), "the source has no minimum clamp")
}

func TestCommitGenericGuardTribute(t *testing.T) {
	t.Run("paid", func(t *testing.T) {
		state, callbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		state.PartyState.Inventory.Gold.Set(100)
		review := state.ReviewGuardTribute()

		result := state.CommitGuardTribute(review)

		assert.Equal(t, GuardTributePaid, result)
		assert.Equal(t, uint16(100)-review.Amount, state.PartyState.Inventory.Gold.Get())
		assert.Empty(t, callbacks.TimeAdvanced(), "the owning Talk workflow charges time")
	})

	t.Run("insufficient is atomic", func(t *testing.T) {
		state, callbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		state.PartyState.Inventory.Gold.Set(5)
		review := state.ReviewGuardTribute()
		require.Greater(t, review.Amount, state.PartyState.Inventory.Gold.Get())

		result := state.CommitGuardTribute(review)

		assert.Equal(t, GuardTributeInsufficientGold, result)
		assert.Equal(t, uint16(5), state.PartyState.Inventory.Gold.Get())
		assert.Empty(t, callbacks.Messages)
		assert.Empty(t, callbacks.TimeAdvanced())
	})
}

func TestCommitMinocGuardTributeUsesGoldDivision(t *testing.T) {
	for _, gold := range []uint16{0, 5, 10} {
		t.Run(fmt.Sprintf("gold_%d", gold), func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.Minoc).
				WithSystemCallbacks().
				Build()
			state.PartyState.Inventory.Gold.Set(gold)

			review := state.ReviewGuardTribute()
			result := state.CommitGuardTribute(review)

			assert.Equal(t, GuardTributeMinocCharity, review.Kind)
			assert.Equal(t, gold-gold/2, review.Amount)
			assert.Equal(t, GuardTributePaid, result)
			assert.Equal(t, gold/2, state.PartyState.Inventory.Gold.Get())
			assert.Empty(t, callbacks.TimeAdvanced())
		})
	}
}

func TestCommitGuardTributeRejectsStaleReview(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.Gold.Set(100)
	review := state.ReviewGuardTribute()
	state.PartyState.Inventory.Gold.Set(99)

	assert.PanicsWithValue(t, "guard tribute state changed after review", func() {
		state.CommitGuardTribute(review)
	})
	assert.Equal(t, uint16(99), state.PartyState.Inventory.Gold.Get())
}

func TestBlackthornPasswordUsesFirstFourCharactersCaseInsensitively(t *testing.T) {
	assert.True(t, IsBlackthornPasswordAccepted("IMPE"))
	assert.True(t, IsBlackthornPasswordAccepted("imperial"))
	assert.False(t, IsBlackthornPasswordAccepted(" IMPE"))
	assert.True(t, IsBlackthornPasswordAccepted("IMPE "))
	assert.False(t, IsBlackthornPasswordAccepted("imp"))
	assert.False(t, IsBlackthornPasswordAccepted("nope"))
}
