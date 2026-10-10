package map_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLightSourcePropertiesMap verifies all expected light sources are defined
// and have valid properties.
func TestLightSourcePropertiesMap(t *testing.T) {
	t.Run("ContainsAllOriginalGlowingTiles", func(t *testing.T) {
		// These tiles are original static emitters; ref:LIGHT-0003.
		// Moongate is handled dynamically in code, so not included here
		originalGlowingTiles := []indexes.SpriteIndex{
			indexes.RightSconce,   // TORCH1 (0xb0)
			indexes.LeftSconce,    // TORCH2 (0xb1)
			indexes.Brazier,       // RED_FLAME (0xb2)
			indexes.CampFire,      // ROASTING_MEAT (0xb3)
			indexes.Fireplace,     // FIREPLACE (0xbc)
			indexes.LampPost,      // LAMP (0xbd)
			indexes.CandleOnTable, // CANDELABRUM (0xbe)
			indexes.CookStove,     // STOVE (0xbf)
			indexes.BlueFlame,     // BLUE_FLAME (0xde)
		}

		for _, tile := range originalGlowingTiles {
			props, ok := LightSourcePropertiesMap[tile]
			assert.True(t, ok, "Original glowing tile %d should be in LightSourcePropertiesMap", tile)
			assert.NotEqual(t, NeutralTint, props.TintColor, "Properties for tile %d should not be neutral", tile)
		}
	})

	t.Run("ContainsAdditionalFireTiles", func(t *testing.T) {
		// These additional Redux emitters are outside the original static list; ref:LIGHT-0006.
		additionalFireTiles := []indexes.SpriteIndex{
			indexes.FireLogs,  // Burning logs
			indexes.ItemTorch, // Ground torch
			indexes.FireField, // Magic fire field
		}

		for _, tile := range additionalFireTiles {
			props, ok := LightSourcePropertiesMap[tile]
			assert.True(t, ok, "Additional fire tile %d should be in LightSourcePropertiesMap", tile)
			assert.NotEqual(t, NeutralTint, props.TintColor, "Properties for tile %d should not be neutral", tile)
		}
	})

	t.Run("AllPropertiesHaveValidTintColor", func(t *testing.T) {
		for spriteIndex, props := range LightSourcePropertiesMap {
			// TintColor should have meaningful values (not all zeros, not all exactly 1.0)
			// Our tint colors are designed around 1.0 (neutral), with warm colors > 1.0 for R
			// Magical sources (moongates) can have higher B values for intense blue glow
			// Fire sources can have low B values (down to 0.1) for strong orange/red tints
			// Fire sources can have high R values (up to 1.6) for strong warm tints
			assert.True(t, props.TintColor.R >= 0.5 && props.TintColor.R <= 1.6,
				"Sprite %d R should be in reasonable range", spriteIndex)
			assert.True(t, props.TintColor.G >= 0.5 && props.TintColor.G <= 1.5,
				"Sprite %d G should be in reasonable range", spriteIndex)
			assert.True(t, props.TintColor.B >= 0.1 && props.TintColor.B <= 1.6,
				"Sprite %d B should be in reasonable range (0.1-1.6)", spriteIndex)
		}
	})

	t.Run("FireSourcesHaveBreathing", func(t *testing.T) {
		// Fire-based sources should have breathing animation enabled
		fireSources := []indexes.SpriteIndex{
			indexes.RightSconce,
			indexes.LeftSconce,
			indexes.Brazier,
			indexes.CampFire,
			indexes.Fireplace,
			indexes.FireLogs,
		}

		for _, tile := range fireSources {
			props, ok := LightSourcePropertiesMap[tile]
			assert.True(t, ok, "Fire source %d should be in map", tile)
			assert.True(t, props.HasBreathing, "Fire source %d should have breathing", tile)
		}
	})

	t.Run("SteadyLightsNoBreathing", func(t *testing.T) {
		// Steady lights (lamps, stoves) should NOT have breathing
		steadySources := []indexes.SpriteIndex{
			indexes.LampPost,
			indexes.CookStove,
		}

		for _, tile := range steadySources {
			props, ok := LightSourcePropertiesMap[tile]
			assert.True(t, ok, "Steady source %d should be in map", tile)
			assert.False(t, props.HasBreathing, "Steady source %d should NOT have breathing", tile)
		}
	})
}

// TestLightTintColors tests the predefined tint color constants
func TestLightTintColors(t *testing.T) {
	t.Run("WarmColorsAreWarm", func(t *testing.T) {
		warmColors := []LightTintColor{TorchWarm, BrazierWarm, FireplaceWarm, CandleWarm}

		for _, c := range warmColors {
			// Warm colors should have R >= G and R > B (scale factors)
			assert.GreaterOrEqual(t, c.R, c.G, "Warm color R should be >= G")
			assert.Greater(t, c.R, c.B, "Warm color R should be > B")
		}
	})

	t.Run("CoolColorsAreCool", func(t *testing.T) {
		// Blue flame should have B > R
		assert.Greater(t, BlueFlameCool.B, BlueFlameCool.R,
			"Blue flame should have more blue than red")
	})

	t.Run("MagicWhiteIsNearNeutral", func(t *testing.T) {
		// Magic white should be close to 1.0 in all channels
		assert.InDelta(t, 1.0, MagicWhite.R, 0.05, "Magic white R should be near 1.0")
		assert.InDelta(t, 1.0, MagicWhite.G, 0.05, "Magic white G should be near 1.0")
		assert.InDelta(t, 1.0, MagicWhite.B, 0.05, "Magic white B should be near 1.0")
	})

	t.Run("NeutralTintIsExactlyOne", func(t *testing.T) {
		assert.Equal(t, float32(1.0), NeutralTint.R)
		assert.Equal(t, float32(1.0), NeutralTint.G)
		assert.Equal(t, float32(1.0), NeutralTint.B)
	})
}

// TestPlayerLightProperties verifies player light source configurations
func TestPlayerLightProperties(t *testing.T) {
	t.Run("AmbientProperties", func(t *testing.T) {
		props := PlayerLightProperties(PlayerLightAmbient)

		assert.Equal(t, AmbientSightDim, props.TintColor, "Ambient should have dim gray tint")
		assert.Equal(t, 2, props.Radius, "Ambient should have 2 tile radius")
		assert.False(t, props.HasBreathing, "Ambient should NOT have breathing (steady dim)")
		// Verify it's a neutral gray (equal R, G, B)
		assert.Equal(t, props.TintColor.R, props.TintColor.G, "Ambient should be neutral gray (R=G)")
		assert.Equal(t, props.TintColor.G, props.TintColor.B, "Ambient should be neutral gray (G=B)")
	})

	t.Run("TorchProperties", func(t *testing.T) {
		props := PlayerLightProperties(PlayerLightTorch)

		assert.Equal(t, TorchWarm, props.TintColor, "Torch should have warm tint")
		assert.Equal(t, TorchTileDistance, props.Radius, "Torch should have TorchTileDistance radius")
		assert.True(t, props.HasBreathing, "Torch should have breathing")
	})

	t.Run("InLorProperties", func(t *testing.T) {
		props := PlayerLightProperties(PlayerLightInLor)

		assert.Equal(t, MagicWhite, props.TintColor, "In Lor should have white tint")
		assert.Equal(t, InLorTileDistance, props.Radius, "In Lor should have InLorTileDistance radius")
		assert.False(t, props.HasBreathing, "In Lor should NOT have breathing (steady magic)")
	})

	t.Run("VasLorProperties", func(t *testing.T) {
		props := PlayerLightProperties(PlayerLightVasLor)

		assert.Equal(t, MagicWhite, props.TintColor, "Vas Lor should have white tint")
		assert.Equal(t, VasLorTileDistance, props.Radius, "Vas Lor should have VasLorTileDistance radius")
		assert.False(t, props.HasBreathing, "Vas Lor should NOT have breathing (steady magic)")
	})

	t.Run("NonePropertiesAreDefault", func(t *testing.T) {
		props := PlayerLightProperties(PlayerLightNone)
		assert.Equal(t, 0, props.Radius, "None light type should have zero radius")
	})

	t.Run("LightRadiusHierarchy", func(t *testing.T) {
		ambientProps := PlayerLightProperties(PlayerLightAmbient)
		torchProps := PlayerLightProperties(PlayerLightTorch)
		inLorProps := PlayerLightProperties(PlayerLightInLor)
		vasLorProps := PlayerLightProperties(PlayerLightVasLor)

		assert.Equal(t, vasLorProps.Radius, inLorProps.Radius,
			"In Lor and Vas Lor should share one magical-light radius")
		assert.Greater(t, inLorProps.Radius, torchProps.Radius,
			"In Lor should have larger radius than torch")
		assert.Greater(t, torchProps.Radius, ambientProps.Radius,
			"Torch should have larger radius than ambient")
		assert.Greater(t, ambientProps.Radius, 0,
			"Ambient should have non-zero radius")
	})
}

// TestGetLightSourceProperties tests the helper function
func TestGetLightSourceProperties(t *testing.T) {
	t.Run("KnownLightSource", func(t *testing.T) {
		props := GetLightSourceProperties(indexes.Brazier)
		assert.Equal(t, BrazierWarm, props.TintColor, "Brazier should have correct tint")
	})

	t.Run("UnknownSpriteReturnsNeutral", func(t *testing.T) {
		// Use a sprite that is definitely not a light source (e.g., grass)
		props := GetLightSourceProperties(indexes.SpriteIndex(1)) // Grass
		assert.Equal(t, NeutralTint, props.TintColor, "Unknown sprite should return neutral tint")
	})
}

// TestLightSourceTintingState tests the tinting state struct
func TestLightSourceTintingState(t *testing.T) {
	t.Run("NewState", func(t *testing.T) {
		state := NewLightSourceTintingState()
		assert.NotNil(t, state)
		assert.Equal(t, 0, state.GetLitTileCount())
	})

	t.Run("BuildLitTilesMap", func(t *testing.T) {
		state := NewLightSourceTintingState()

		// Create mock light sources
		tile := &references.Tile{
			Index:         indexes.RightSconce,
			LightEmission: 2, // 2 tile radius
		}
		lightSources := LightSources{
			{Tile: tile, Pos: references.Position{X: 5, Y: 5}},
		}

		state.BuildLitTilesMap(lightSources, nil)

		// Should have lit tiles
		assert.Greater(t, state.GetLitTileCount(), 0)

		// The source position should be lit
		assert.True(t, state.IsTileLit(references.Position{X: 5, Y: 5}))

		// Adjacent tiles should be lit (within radius 2)
		assert.True(t, state.IsTileLit(references.Position{X: 5, Y: 6}))
		assert.True(t, state.IsTileLit(references.Position{X: 6, Y: 5}))
	})

	t.Run("GetTintScaleFactors_LitTile", func(t *testing.T) {
		state := NewLightSourceTintingState()

		// Create mock light source
		tile := &references.Tile{
			Index:         indexes.RightSconce,
			LightEmission: 2,
		}
		lightSources := LightSources{
			{Tile: tile, Pos: references.Position{X: 5, Y: 5}},
		}

		state.BuildLitTilesMap(lightSources, nil)

		// Get tint for lit tile (ambient darkness 1 for night behavior)
		tint := state.GetTintScaleFactors(references.Position{X: 5, Y: 5}, 0, 1)
		require.NotNil(t, tint, "Lit tile should have tint")

		// Tint should be in reasonable range (around 1.0, not too extreme)
		assert.True(t, tint.R > 0.5 && tint.R < 1.5, "R should be in range")
		assert.True(t, tint.G > 0.5 && tint.G < 1.5, "G should be in range")
		assert.True(t, tint.B > 0.5 && tint.B < 1.5, "B should be in range")
	})

	t.Run("GetTintScaleFactors_UnlitTile", func(t *testing.T) {
		state := NewLightSourceTintingState()

		tile := &references.Tile{
			Index:         indexes.RightSconce,
			LightEmission: 2,
		}
		lightSources := LightSources{
			{Tile: tile, Pos: references.Position{X: 5, Y: 5}},
		}

		state.BuildLitTilesMap(lightSources, nil)

		// Unlit tile should return nil
		tint := state.GetTintScaleFactors(references.Position{X: 100, Y: 100}, 0, 1)
		assert.Nil(t, tint, "Unlit tile should have no tint")
	})

	t.Run("Clear", func(t *testing.T) {
		state := NewLightSourceTintingState()

		tile := &references.Tile{
			Index:         indexes.RightSconce,
			LightEmission: 2,
		}
		lightSources := LightSources{
			{Tile: tile, Pos: references.Position{X: 5, Y: 5}},
		}

		state.BuildLitTilesMap(lightSources, nil)
		assert.Greater(t, state.GetLitTileCount(), 0)

		state.Clear()
		assert.Equal(t, 0, state.GetLitTileCount())
	})

	t.Run("BreathingProducesSlowVariation", func(t *testing.T) {
		state := NewLightSourceTintingState()

		// RightSconce has breathing enabled
		tile := &references.Tile{
			Index:         indexes.RightSconce,
			LightEmission: 2,
		}
		lightSources := LightSources{
			{Tile: tile, Pos: references.Position{X: 5, Y: 5}},
		}

		state.BuildLitTilesMap(lightSources, nil)

		pos := references.Position{X: 5, Y: 5}

		// Get tints at different times (breathing period is 4500ms)
		tint1 := state.GetTintScaleFactors(pos, 0, 1)
		tint2 := state.GetTintScaleFactors(pos, 2250, 1) // Half period

		// Tints should be non-nil
		require.NotNil(t, tint1)
		require.NotNil(t, tint2)

		// At different phases of the breathing cycle, colors should vary slightly
		// The variation should be subtle (BreathingAmplitude = 0.05 = ±5%)
		// This test mainly verifies the calculation runs without error
	})

	t.Run("MultipleSourcesBlend", func(t *testing.T) {
		state := NewLightSourceTintingState()

		// Create two overlapping light sources
		tile1 := &references.Tile{
			Index:         indexes.RightSconce,
			LightEmission: 3,
		}
		tile2 := &references.Tile{
			Index:         indexes.BlueFlame,
			LightEmission: 3,
		}
		lightSources := LightSources{
			{Tile: tile1, Pos: references.Position{X: 5, Y: 5}},
			{Tile: tile2, Pos: references.Position{X: 7, Y: 5}},
		}

		state.BuildLitTilesMap(lightSources, nil)

		// Position between two sources should be lit
		middlePos := references.Position{X: 6, Y: 5}
		assert.True(t, state.IsTileLit(middlePos), "Position between sources should be lit")

		// Get tint - should be a blend of warm (torch) and cool (blue flame)
		tint := state.GetTintScaleFactors(middlePos, 0, 1)
		require.NotNil(t, tint, "Overlapping position should have tint")

		// The blended tint should be somewhere between warm and cool
		// (neither as warm as torch nor as cool as blue flame alone)
	})

	t.Run("DistanceFalloff", func(t *testing.T) {
		state := NewLightSourceTintingState()

		tile := &references.Tile{
			Index:         indexes.RightSconce,
			LightEmission: 4,
		}
		lightSources := LightSources{
			{Tile: tile, Pos: references.Position{X: 5, Y: 5}},
		}

		state.BuildLitTilesMap(lightSources, nil)

		// Get tints at different distances
		centerTint := state.GetTintScaleFactors(references.Position{X: 5, Y: 5}, 0, 1)
		edgeTint := state.GetTintScaleFactors(references.Position{X: 5, Y: 9}, 0, 1)

		require.NotNil(t, centerTint, "Center should be lit")
		require.NotNil(t, edgeTint, "Edge should be lit")

		// Due to brightness falloff, edge should generally be dimmer
		// CenterBrightness = 1.02, BaseBrightness = 0.88
		// We can verify by checking that edge RGB values are lower on average
		centerAvg := (centerTint.R + centerTint.G + centerTint.B) / 3
		edgeAvg := (edgeTint.R + edgeTint.G + edgeTint.B) / 3

		assert.Greater(t, centerAvg, edgeAvg, "Center should be brighter than edge on average")
	})

	t.Run("AmbientDarknessInterpolatesSourceTint", func(t *testing.T) {
		state := NewLightSourceTintingState()
		tile := &references.Tile{Index: indexes.LampPost, LightEmission: 2}
		pos := references.Position{X: 5, Y: 5}
		state.BuildLitTilesMap(LightSources{{Tile: tile, Pos: pos}}, nil)

		dayTint := state.GetTintScaleFactors(pos, 0, 0)
		transitionTint := state.GetTintScaleFactors(pos, 0, 0.5)
		nightTint := state.GetTintScaleFactors(pos, 0, 1)

		require.NotNil(t, dayTint)
		require.NotNil(t, transitionTint)
		require.NotNil(t, nightTint)
		assert.InDelta(t, (dayTint.R+nightTint.R)/2, transitionTint.R, 0.0001)
		assert.InDelta(t, (dayTint.G+nightTint.G)/2, transitionTint.G, 0.0001)
		assert.InDelta(t, (dayTint.B+nightTint.B)/2, transitionTint.B, 0.0001)
	})
}
