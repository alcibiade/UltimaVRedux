package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestLightEmittingTiles verifies that the original static emitter tiles
// have correct LightEmission values in the runtime tile definitions.
//
// The original list includes gates, lamps, candelabra, red/blue flames, stoves,
// both wall torches, roasting meat, and fireplaces; ref:LIGHT-0003.
func TestLightEmittingTiles(t *testing.T) {
	tiles := NewTileReferences()
	require.NotNil(t, tiles, "TileReferences should load successfully")

	t.Run("OriginalGlowingArrayTiles", func(t *testing.T) {
		// These tiles belong to the original static-emitter list
		// and MUST have LightEmission > 0
		originalGlowingTiles := []struct {
			name             string
			index            indexes.SpriteIndex
			expectedEmission int
			originalConstant string
			originalHexValue string
		}{
			{"RightSconce", indexes.RightSconce, 3, "TORCH1", "0xb0"},
			{"LeftSconce", indexes.LeftSconce, 3, "TORCH2", "0xb1"},
			{"Brazier", indexes.Brazier, 3, "RED_FLAME", "0xb2"},
			{"CampFire", indexes.CampFire, 3, "ROASTING_MEAT", "0xb3"},
			{"Fireplace", indexes.Fireplace, 3, "FIREPLACE", "0xbc"},
			{"LampPost", indexes.LampPost, 3, "LAMP", "0xbd"},
			{"CandleOnTable", indexes.CandleOnTable, 3, "CANDELABRUM", "0xbe"},
			{"CookStove", indexes.CookStove, 3, "STOVE", "0xbf"},
			{"BlueFlame", indexes.BlueFlame, 4, "BLUE_FLAME", "0xde"},
			// Note: Moongate (0xdc) emission is handled dynamically, not by static tile metadata.
		}

		for _, tc := range originalGlowingTiles {
			t.Run(tc.name, func(t *testing.T) {
				tile := tiles.GetTile(tc.index)
				require.NotNil(t, tile, "%s tile should exist in the runtime definitions", tc.name)

				assert.Equal(t, tc.expectedEmission, tile.GetLightEmission(),
					"%s (original: %s=%s) should have LightEmission=%d",
					tc.name, tc.originalConstant, tc.originalHexValue, tc.expectedEmission)

				assert.Greater(t, tile.GetLightEmission(), 0,
					"%s must emit light (LightEmission > 0) - original static emitter (ref:LIGHT-0003)",
					tc.name)
			})
		}
	})

	t.Run("AdditionalFireTiles", func(t *testing.T) {
		// These additional emitters are outside the original static list; ref:LIGHT-0006.
		// We add them for enhanced gameplay
		additionalFireTiles := []struct {
			name             string
			index            indexes.SpriteIndex
			expectedEmission int
			reason           string
		}{
			{"FireLogs", indexes.FireLogs, 3, "Burning logs should emit light like fireplace"},
			{"ItemTorch", indexes.ItemTorch, 3, "Torch item on ground should emit light"},
			{"FireField", indexes.FireField, 3, "Magic fire field should emit light"},
		}

		for _, tc := range additionalFireTiles {
			t.Run(tc.name, func(t *testing.T) {
				tile := tiles.GetTile(tc.index)
				require.NotNil(t, tile, "%s tile should exist in the runtime definitions", tc.name)

				assert.Equal(t, tc.expectedEmission, tile.GetLightEmission(),
					"%s should have LightEmission=%d: %s",
					tc.name, tc.expectedEmission, tc.reason)
			})
		}
	})

	t.Run("MagicDoorGlow", func(t *testing.T) {
		// Magic locked doors have a faint magical glow
		magicDoors := []struct {
			name             string
			index            indexes.SpriteIndex
			expectedEmission int
		}{
			{"MagicLockDoor", indexes.MagicLockDoor, 2},
			{"MagicLockDoorWithView", indexes.MagicLockDoorWithView, 2},
		}

		for _, tc := range magicDoors {
			t.Run(tc.name, func(t *testing.T) {
				tile := tiles.GetTile(tc.index)
				require.NotNil(t, tile, "%s tile should exist", tc.name)

				assert.Equal(t, tc.expectedEmission, tile.GetLightEmission(),
					"%s should have faint magical glow (LightEmission=%d)",
					tc.name, tc.expectedEmission)
			})
		}
	})

	t.Run("NonLightEmittingTiles", func(t *testing.T) {
		// These common tiles should NOT emit light
		nonGlowingTiles := []struct {
			name  string
			index indexes.SpriteIndex
		}{
			{"Grass", indexes.Grass},
			{"BrickFloor", indexes.BrickFloor},
			{"Water1", indexes.Water1},
			{"Barrel", indexes.Barrel},
			{"Chest", indexes.Chest},
			{"ChairBackForward", indexes.ChairBackForward},
			{"TableMiddle", indexes.TableMiddle},
		}

		for _, tc := range nonGlowingTiles {
			t.Run(tc.name, func(t *testing.T) {
				tile := tiles.GetTile(tc.index)
				require.NotNil(t, tile, "%s tile should exist", tc.name)

				assert.Equal(t, 0, tile.GetLightEmission(),
					"%s should not emit light (LightEmission should be 0)",
					tc.name)
			})
		}
	})
}

// TestGetLightEmissionGetter verifies the getter returns the correct value
func TestGetLightEmissionGetter(t *testing.T) {
	tiles := NewTileReferences()

	t.Run("GetterReturnsCorrectValue", func(t *testing.T) {
		brazier := tiles.GetTile(indexes.Brazier)
		require.NotNil(t, brazier)

		// Verify getter returns same as direct field access would
		assert.Equal(t, 3, brazier.GetLightEmission())
	})

	t.Run("GetterReturnsZeroForNonEmitting", func(t *testing.T) {
		grass := tiles.GetTile(indexes.Grass)
		require.NotNil(t, grass)

		assert.Equal(t, 0, grass.GetLightEmission())
	})
}

// TestLightEmissionConsistency verifies that all fire-related tiles have consistent values
func TestLightEmissionConsistency(t *testing.T) {
	tiles := NewTileReferences()

	t.Run("AllFireSourcesHaveSameRadius", func(t *testing.T) {
		// Standard fire sources should all have the same radius (3)
		standardFireTiles := []indexes.SpriteIndex{
			indexes.RightSconce,
			indexes.LeftSconce,
			indexes.Brazier,
			indexes.CampFire,
			indexes.Fireplace,
			indexes.LampPost,
			indexes.CandleOnTable,
			indexes.CookStove,
			indexes.FireLogs,
			indexes.ItemTorch,
			indexes.FireField,
		}

		for _, idx := range standardFireTiles {
			tile := tiles.GetTile(idx)
			require.NotNil(t, tile, "Tile %d should exist", idx)

			assert.Equal(t, 3, tile.GetLightEmission(),
				"Tile %s (%d) should have standard fire LightEmission of 3",
				tile.Name, idx)
		}
	})

	t.Run("BlueFlameHasStrongerEmission", func(t *testing.T) {
		blueFlame := tiles.GetTile(indexes.BlueFlame)
		require.NotNil(t, blueFlame)

		// BlueFlame is magical and should be brighter than standard fire
		assert.Equal(t, 4, blueFlame.GetLightEmission(),
			"BlueFlame should have stronger LightEmission (4) than standard fire (3)")

		brazier := tiles.GetTile(indexes.Brazier)
		require.NotNil(t, brazier)

		assert.Greater(t, blueFlame.GetLightEmission(), brazier.GetLightEmission(),
			"BlueFlame should emit more light than standard Brazier")
	})
}
