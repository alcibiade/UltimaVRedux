package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCommandOutputConvention_AppendsResultsToOriginalPrompt(t *testing.T) {
	t.Run("Board what", func(t *testing.T) {
		gameState, callbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()
		if gameState == nil {
			return
		}

		assert.False(t, gameState.ActionBoard())
		assert.Equal(t, []string{"Board "}, callbacks.CommandPrompts)
		assert.Equal(t, "What?", callbacks.CurrentRowContent)
	})

	t.Run("X-it what", func(t *testing.T) {
		gameState, callbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		if gameState == nil {
			return
		}

		assert.False(t, gameState.ActionExit())
		assert.Equal(t, []string{"X-it "}, callbacks.CommandPrompts)
		assert.Equal(t, "what?", callbacks.CurrentRowContent)
	})

	t.Run("Ignite without torches", func(t *testing.T) {
		gameState, callbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		if gameState == nil {
			return
		}
		gameState.PartyState.Inventory.Provisions.Torches.Set(0)

		assert.False(t, gameState.ActionIgnite())
		assert.Equal(t, []string{"Ignite torch!"}, callbacks.CommandPrompts)
		assert.Equal(t, []string{"None owned!"}, callbacks.Messages)
	})

	t.Run("Enter what", func(t *testing.T) {
		gameState, callbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(100, 100).
			WithSystemCallbacks().
			Build()
		if gameState == nil {
			return
		}

		assert.False(t, gameState.ActionEnterLargeMap())
		assert.Equal(t, []string{"Enter "}, callbacks.CommandPrompts)
		assert.Equal(t, "What?", callbacks.CurrentRowContent)
	})
}
