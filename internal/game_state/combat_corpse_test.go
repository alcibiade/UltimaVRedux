package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestRollCombatCorpseSearchOriginalOutcomes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rolls    []int
		message  string
		item     references.Provision
		quantity uint16
		poisoned bool
		hasItem  bool
	}{
		{name: "food", rolls: []int{0, 0, 3}, message: "food!", item: references.Food, quantity: 3, hasItem: true},
		{name: "gold", rolls: []int{0, 1, 2}, message: "gold!", item: references.Gold, quantity: 2, hasItem: true},
		{name: "plague", rolls: []int{1, 19}, message: "Plague!", item: references.NoProvision, poisoned: true},
		{name: "bloody pulp", rolls: []int{1, 0, 3, 3}, message: "a bloody pulp!", item: references.NoProvision},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			rollIndex := 0
			outcome := rollCombatCorpseSearch(func(_, _ int) int {
				roll := test.rolls[rollIndex]
				rollIndex++

				return roll
			})

			assert.Equal(t, test.message, outcome.message)
			assert.Equal(t, test.item, outcome.item)
			assert.Equal(t, test.quantity, outcome.quantity)
			assert.Equal(t, test.poisoned, outcome.poisoned)
			assert.Equal(t, test.hasItem, outcome.hasItem)
			assert.Equal(t, len(test.rolls), rollIndex)
		})
	}
}

func TestActionSearchCombatMapConsumesCorpse(t *testing.T) {
	gs, mock := newBritainCombatTestState(t, 169)
	activeIndex := gs.CombatScene.GetActiveCombatantIndex()
	activePosition := gs.CombatScene.GetPartyMemberPosition(activeIndex)
	targetPosition := references.Down.GetNewPositionInDirection(&activePosition)
	require.NotNil(t, targetPosition)
	gs.CombatScene.AddObject(NewCombatCorpse(*targetPosition))
	gs.PartyState.GetCharacter(activeIndex).SetStatus(party_state.Good)
	mock.Reset()

	assert.True(t, gs.ActionSearchCombatMap(references.Down))
	_, corpseFound := gs.CombatScene.objectAtMatching(*targetPosition, CombatObject.IsCorpse)
	assert.False(t, corpseFound)
	assert.NotEmpty(t, mock.Messages)
}
