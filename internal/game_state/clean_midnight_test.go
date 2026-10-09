package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCleanMidnight_UserScenario(t *testing.T) {
	// Test with clean original code - user's exact midnight scenario
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 31).
		WithSystemCallbacks().
		Build()
	avatarPos := references.Position{X: 15, Y: 31}
	userPositions := []references.Position{
		{X: 15, Y: 26}, // 5 tiles north
		{X: 11, Y: 27}, // User mentioned position
		{X: 14, Y: 31}, // Adjacent
		{X: 16, Y: 31}, // Adjacent
		{X: 15, Y: 30}, // Adjacent
	}

	// Test at midnight with clean original code
	midnight := datetime.UltimaDate{}
	midnight.SetTimeOfDay(datetime.Midnight)

	britanniaMap := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0)
	// Use RecalculateVisibleTilesWithCustomLights with ambient light (no torch at midnight)
	britanniaMap.RecalculateVisibleTilesWithCustomLights(avatarPos, &gs.MapState.Lighting, nil, nil, nil, map_state.PlayerLightAmbient)

	t.Logf("=== CLEAN ORIGINAL CODE MIDNIGHT TEST ===")
	t.Logf("Avatar at (%d,%d) at midnight with clean original code", avatarPos.X, avatarPos.Y)
	t.Logf("")

	for _, pos := range userPositions {
		visible := britanniaMap.IsPositionVisible(&pos, midnight, false)

		t.Logf("📍 Position (%d,%d):", pos.X, pos.Y)
		if visible {
			t.Logf("  ✅ VISIBLE")
		} else {
			t.Logf("  ❌ NOT VISIBLE")
		}
		t.Logf("")
	}
}
