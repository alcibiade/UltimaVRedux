// Package sprites provides tile visual effect definitions.
package sprites

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// alternatingColumnWaterPhase is the original water tile's even-column phase offset.
const alternatingColumnWaterPhase = 6

const (
	poisonFieldAnimationIntervalMs     = 175
	sleepFieldAnimationIntervalMs      = 100
	protectionFieldAnimationIntervalMs = 150
)

// TileEffectRegistry defines visual effects for all tiles that need them.
// Effects are applied in order (pipeline), with each effect operating on
// the output of the previous effect.
//
// Use the fluent builder API:
//
//	indexes.RightSconce: Effect().
//	    WithBackground(indexes.BrickFloor).
//	    WithBounds(6, 1, 12, 4).
//	    AddVerticalRise(FilterRedOnly).
//	    Build(),
var TileEffectRegistry = map[indexes.SpriteIndex]TileEffectPipeline{

	// ========================================================================
	// FLAME TILES - Wall-mounted with brick background
	// ========================================================================

	indexes.RightSconce: Effect().
		WithBackground(indexes.BrickFloor).
		WithBounds(inclusiveTileBounds{6, 1, 12, 4}).
		AddVerticalRise(FilterFire).
		Build(),

	indexes.LeftSconce: Effect().
		WithBackground(indexes.BrickFloor).
		WithBounds(inclusiveTileBounds{5, 1, 11, 4}).
		AddVerticalRise(FilterFire).
		Build(),

	indexes.Brazier: Effect().
		// WithBackground(indexes.MetalFloor).
		WithBounds(inclusiveTileBounds{3, 0, 11, 6}).
		AddVerticalRise(FilterFire).
		Build(),

	indexes.Fireplace: Effect().
		WithBackground(indexes.BrickFloor).
		WithBounds(inclusiveTileBounds{4, 6, 13, 14}).
		AddVerticalRise(FilterFire).
		Build(),

	indexes.CandleOnTable: Effect().
		WithBackground(indexes.BrickFloor).
		WithBounds(inclusiveTileBounds{4, 2, 12, 2}).
		AddVerticalRise(FilterFire).
		Build(),

	// ========================================================================
	// FLAME TILES - Outdoor with grass background
	// ========================================================================
	indexes.LampPost: Effect().
		WithBackground(indexes.Grass).
		WithBounds(inclusiveTileBounds{6, 3, 9, 4}).
		AddVerticalRise(FilterFire).
		Build(),

	indexes.CampFire: Effect().
		WithBackground(indexes.Grass).
		WithBounds(inclusiveTileBounds{4, 10, 8, 12}).
		AddVerticalRise(FilterFire).
		Build(),

	indexes.FireLogs: Effect().
		WithBounds(inclusiveTileBounds{4, 6, 12, 14}).
		AddVerticalRise(FilterFire).
		Build(),

	indexes.CookStove: Effect().
		WithBackground(indexes.Grass).
		WithBounds(inclusiveTileBounds{6, 7, 10, 9}).
		AddVerticalRise(FilterFire).
		Build(),

	indexes.BlueFlame: Effect().
		WithBackground(indexes.MetalFloor).
		WithBounds(inclusiveTileBounds{4, 3, 11, 6}).
		AddVerticalRise(FilterBlueFire).
		Build(),

	// ========================================================================
	// FLAME TILES - No background (overlays terrain)
	// ========================================================================

	indexes.FireField: Effect().
		AddVerticalRise(FilterFire).
		Build(),

	// Combat magic fields retain their distinct original sprites while using
	// field-specific motion to make each spell's effect readable in combat.
	indexes.PoisonField: Effect().
		WithInterval(poisonFieldAnimationIntervalMs).
		AddVerticalRise(FilterPoison).
		Build(),

	indexes.MagicField: Effect().
		WithInterval(sleepFieldAnimationIntervalMs).
		AddVerticalRise(FilterSleep).
		AddPulse(FilterSleep).
		Build(),

	indexes.ElectricField: Effect().
		WithInterval(protectionFieldAnimationIntervalMs).
		AddBrightnessPulse(FilterProtection).
		Build(),

	indexes.ItemTorch: Effect().
		AddVerticalRise(FilterFire).
		Build(),

	// ========================================================================
	// LAVA TILES
	// ========================================================================

	indexes.Lava: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddWaveDisplace(FilterAll).
		AddVerticalScroll().
		Build(),

	// ========================================================================
	// WATER TILES - Pure water with vertical scroll animation
	// ========================================================================

	indexes.Water1: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddVerticalScroll().
		Build(),

	indexes.Water2: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddVerticalScroll().
		Build(),

	indexes.WaterCoast: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddVerticalScroll().
		Build(),

	// ========================================================================
	// MIXED LAND/WATER TILES - Transparency overlay with animated water background
	// Blue/water-colored pixels become transparent, revealing WaterCoast animation
	// ========================================================================

	// Outside Water Coast tiles (land with water edge)
	indexes.OutsideWaterCoast1: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.OutsideWaterCoast2: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.OutsideWaterCoast3: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.OutsideWaterCoast4: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	// Corner with water tiles (diagonal land/water corners)
	indexes.CornerWithWater1: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.CornerWithWater2: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.CornerWithWater3: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.CornerWithWater4: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	// Water stream tiles (river/stream paths)
	indexes.WaterStream1: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.WaterStream2: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.WaterStream3: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.WaterStream4: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.WaterStream5: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.WaterStream6: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.WaterStream7: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.WaterStream8: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.WaterStream9: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.WaterStream10: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.WaterStream11: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.WaterStream12: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.WaterStream13: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.WaterStream14: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	// Troll bridge tiles (bridge structures over water)
	indexes.TrollBridgeHoriz: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	indexes.TrollBridgeVert: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),

	// Dock tile (dock structure over water)
	indexes.Dock: Effect().
		WithInterval(MsPerFrameForWater).
		WithEvenXOffset(alternatingColumnWaterPhase).
		AddTransparencyOverlay(indexes.WaterCoast).
		Build(),
}
