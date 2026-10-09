package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestTalkOverable(t *testing.T) {
	talkThroughTiles := []struct {
		name  string
		index indexes.SpriteIndex
	}{
		{name: "TableLeft", index: indexes.TableLeft},
		{name: "TableMiddle", index: indexes.TableMiddle},
		{name: "TableRight", index: indexes.TableRight},
		{name: "MagicLockDoor", index: indexes.MagicLockDoor},
		{name: "MagicLockDoorWithView", index: indexes.MagicLockDoorWithView},
		{name: "Portcullis", index: indexes.Portcullis},
		{name: "TableFoodTop", index: indexes.TableFoodTop},
		{name: "TableFoodBottom", index: indexes.TableFoodBottom},
		{name: "TableFoodBoth", index: indexes.TableFoodBoth},
		{name: "Desk", index: indexes.EndTable},
		{name: "EndTable", index: indexes.CoffeeTable},
		{name: "FenceHoriz", index: indexes.FenceHoriz},
		{name: "FenceVert", index: indexes.FenceVert},
		{name: "LockedDoorView", index: indexes.LockedDoorView},
		{name: "Candelabrum", index: indexes.CandleOnTable},
		{name: "RegularDoorView", index: indexes.RegularDoorView},
		{name: "Crystal", index: indexes.CrystalBall},
	}
	for _, test := range talkThroughTiles {
		t.Run(test.name, func(t *testing.T) {
			tile := &references.Tile{Index: test.index}
			assert.True(t, tile.IsTalkOverable())
		})
	}

	t.Run("RegularFloor_NotTalkOverable", func(t *testing.T) {
		tile := &references.Tile{Index: indexes.BrickFloor}
		assert.False(t, tile.IsTalkOverable(), "BrickFloor should not be talk-overable")
	})

	t.Run("Wall_NotTalkOverable", func(t *testing.T) {
		tile := &references.Tile{Index: indexes.StoneBrickWall}
		assert.False(t, tile.IsTalkOverable(), "Walls should not be talk-overable")
	})
}
