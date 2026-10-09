package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

// Note: Placeholder test TestSearchAndGetWorkflow_Integration was removed.
// The comprehensive test TestGlassSwordSearch_GetCommand_Integration in glass_sword_search_test.go
// covers the complete Search → Get workflow with real game data.

func TestSearchSystemManager_formatFoundItems(t *testing.T) {
	gameState := &GameState{}
	searchManager := NewSearchSystemManager(gameState)

	testCases := []struct {
		name     string
		items    []references.ItemAndQuantity
		expected string
	}{
		{
			name:     "No items",
			items:    []references.ItemAndQuantity{},
			expected: "nothing of note.",
		},
		{
			name: "Single gold item",
			items: []references.ItemAndQuantity{
				{Quantity: 15, Item: references.Gold},
			},
			expected: "15 gold coins!",
		},
		{
			name: "Single torch",
			items: []references.ItemAndQuantity{
				{Quantity: 1, Item: references.Torches},
			},
			expected: "a torch!",
		},
		{
			name: "Multiple torches",
			items: []references.ItemAndQuantity{
				{Quantity: 3, Item: references.Torches},
			},
			expected: "3 torches!",
		},
		{
			name: "Multiple items",
			items: []references.ItemAndQuantity{
				{Quantity: 10, Item: references.Gold},
				{Quantity: 2, Item: references.Torches},
			},
			expected: "2 items!",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := searchManager.FormatFoundItems(tc.items)
			assert.Equal(t, tc.expected, result)
		})
	}
}
