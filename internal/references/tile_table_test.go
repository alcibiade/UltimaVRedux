package references

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestTablePassability tests that tables properly block walking
func TestTablePassability(t *testing.T) {
	t.Run("TableVariants", func(t *testing.T) {
		tableVariants := []struct {
			name  string
			index indexes.SpriteIndex
		}{
			{"TableLeft", indexes.TableLeft},
			{"TableMiddle", indexes.TableMiddle},
			{"TableRight", indexes.TableRight},
			{"TableFoodTop", indexes.TableFoodTop},
			{"TableFoodBottom", indexes.TableFoodBottom},
			{"TableFoodBoth", indexes.TableFoodBoth},
			{"CandleOnTable", indexes.CandleOnTable},
		}

		for _, table := range tableVariants {
			t.Run(table.name, func(t *testing.T) {
				tile := &Tile{Index: table.index}

				// Tables should NOT be walkable
				assert.False(t, tile.IsWalkingPassable(),
					"%s should not be walkable", table.name)

				// Tables should be identified as tables
				assert.True(t, tile.IsTable(),
					"%s should be identified as table", table.name)

				// Tables should be talk-overable (for shopkeepers)
				assert.True(t, tile.IsTalkOverable(),
					"%s should be talk-overable", table.name)

				// Tables should not be passable for any vehicle type
				assert.False(t, tile.IsPassable(NoPartyVehicle),
					"%s should not be passable on foot", table.name)
				assert.False(t, tile.IsPassable(HorseVehicle),
					"%s should not be passable on horse", table.name)
			})
		}
	})

	t.Run("CompareWithChairs", func(t *testing.T) {
		// Chairs should still be walkable (can sit on them)
		chairTypes := []indexes.SpriteIndex{
			indexes.ChairBackForward,
			indexes.ChairBackBack,
			indexes.ChairBackLeft,
			indexes.ChairBackRight,
		}

		for _, chairIndex := range chairTypes {
			chair := &Tile{Index: chairIndex}
			assert.True(t, chair.IsWalkingPassable(),
				"Chair %v should be walkable (can sit on them)", chairIndex)
			assert.True(t, chair.IsChair(),
				"Chair %v should be identified as chair", chairIndex)
			assert.False(t, chair.IsTable(),
				"Chair %v should not be identified as table", chairIndex)
		}
	})

	t.Run("NonTableTiles", func(t *testing.T) {
		// Verify non-table tiles are not identified as tables
		nonTables := []indexes.SpriteIndex{
			indexes.Grass,
			indexes.BrickFloor,
			indexes.ChairBackForward,
			indexes.Barrel,
		}

		for _, index := range nonTables {
			tile := &Tile{Index: index}
			assert.False(t, tile.IsTable(),
				"Tile %v should not be identified as table", index)
		}
	})
}

// TestTableBlocksAvatar tests that the avatar cannot walk through tables
func TestTableBlocksAvatar(t *testing.T) {
	t.Run("AllTablesBlockWalking", func(t *testing.T) {
		// Test all table types
		tables := []indexes.SpriteIndex{
			indexes.TableLeft,
			indexes.TableMiddle,
			indexes.TableRight,
			indexes.TableFoodTop,
			indexes.TableFoodBottom,
			indexes.TableFoodBoth,
			indexes.CandleOnTable,
		}

		for _, tableIndex := range tables {
			table := &Tile{Index: tableIndex}

			// Avatar walking check (NoPartyVehicle)
			assert.False(t, table.IsPassable(NoPartyVehicle),
				"Avatar should not be able to walk through table %v", tableIndex)

			// Direct walking passability check
			assert.False(t, table.IsWalkingPassable(),
				"Table %v should not be walkable", tableIndex)
		}
	})
}
