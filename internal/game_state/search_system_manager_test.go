package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

// Note: Placeholder test TestSearchSystemManager_ExecuteSearch_BasicFlow was removed.
// The comprehensive tests in glass_sword_search_test.go and authentic_search_test.go
// cover search execution with real game data.

func TestSearchSystemManager_getLocationPrefix(t *testing.T) {
	gameState := &GameState{}
	searchManager := NewSearchSystemManager(gameState)

	testCases := []struct {
		name     string
		location string
		expected string
	}{
		{
			name:     "Well location",
			location: "well",
			expected: "Near the well\nthou dost find\n",
		},
		{
			name:     "Barrel location",
			location: "barrel",
			expected: "In the barrel\nthou dost find\n",
		},
		{
			name:     "Vanity location",
			location: "vanity",
			expected: "In the vanity\nthou dost find\n",
		},
		{
			name:     "Bed location",
			location: "bed",
			expected: "Under the bed\nthou dost find\n",
		},
		{
			name:     "Fireplace location",
			location: "fireplace",
			expected: "In the fireplace\nthou dost find\n",
		},
		{
			name:     "Brazier location",
			location: "brazier",
			expected: "In the brazier\nthou dost find\n",
		},
		{
			name:     "Wall location",
			location: "wall",
			expected: "In the wall\nthou dost find\n",
		},
		{
			name:     "Stump location",
			location: "stump",
			expected: "In the stump\nthou dost find\n",
		},
		{name: "Shelf location", location: "shelf", expected: "On the shelf\nthou dost find\n"},
		{name: "Bookshelf location", location: "bookshelf", expected: "In the bookshelf\nthou dost find\n"},
		{name: "Desk location", location: "desk", expected: "In the desk\nthou dost find\n"},
		{name: "Dresser location", location: "dresser", expected: "In the dresser\nthou dost find\n"},
		{name: "Trunk location", location: "trunk", expected: "In the trunk\nthou dost find\n"},
		{
			name:     "Unknown location",
			location: "unknown",
			expected: "Thou dost find\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := searchManager.GetLocationPrefix(tc.location)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSearchSystemManager_buildNothingFoundMessageUsesNaturalSentence(t *testing.T) {
	searchManager := NewSearchSystemManager(&GameState{})

	assert.Equal(t,
		"In the barrel, thou dost find nothing of note.",
		searchManager.buildNothingFoundMessage("barrel"),
	)
	assert.Equal(t, MessageNothingOfNote, searchManager.buildNothingFoundMessage("unknown"))
}

func TestSearchSystemManagerOriginalTerrainClassification(t *testing.T) {
	searchManager := NewSearchSystemManager(&GameState{})
	tests := []struct {
		tile indexes.SpriteIndex
		want string
	}{
		{indexes.DeadTree, "stump"},
		{indexes.Bookcase, "shelf"},
		{indexes.BookcaseLeft, "bookshelf"},
		{indexes.BookcaseRight, "bookshelf"},
		{indexes.Well, "well"},
		{indexes.EndTable, "desk"},
		{indexes.Barrel, "barrel"},
		{indexes.Vanity, "vanity"},
		{indexes.LeftBed, "bed"},
		{indexes.RightBed, "bed"},
		{indexes.Dresser, "dresser"},
		{indexes.Box, "trunk"},
		{indexes.Fireplace, "fireplace"},
		{indexes.Brazier, "brazier"},
		{indexes.StoneBrickWall, "wall"},
		{indexes.BlueFlame, ""},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, searchManager.getTileLocationName(test.tile), "tile %d", test.tile)
	}
}

func TestSearchSystemManager_addFoundItemsToMap(t *testing.T) {
	gameState := &GameState{
		ItemStacksMap: *references.NewItemStacksMap(),
	}
	searchManager := NewSearchSystemManager(gameState)

	pos := &references.Position{X: 5, Y: 10}
	items := []references.ItemAndQuantity{
		{
			Quantity: 1,
			Item:     references.Gold,
		},
	}

	err := searchManager.addFoundItemsToMap(pos, items)
	assert.NoError(t, err)

	// Verify the items were added to the map
	assert.True(t, gameState.ItemStacksMap.HasItemStackAtPosition(pos))
}
