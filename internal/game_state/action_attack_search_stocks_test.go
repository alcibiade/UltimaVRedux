package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestSearchForDefenselessNPCs_Yew searches the entire Yew map for defenseless NPCs
// to understand what the user is actually trying to attack
func TestSearchForDefenselessNPCs_Yew(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Yew).
		WithPlayerAt(20, 17).
		WithSystemCallbacks().
		Build()

	currentMap := gs.GetLayeredMapByCurrentLocation()

	t.Logf("=== Searching Yew map for defenseless NPCs ===")

	foundStocks := []struct {
		x, y  int
		index indexes.SpriteIndex
	}{}

	// Search entire map (typical town map is 32x32)
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			pos := &references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
			tile := currentMap.GetTopTile(pos)

			// Check base tile for defenseless tiles (stocks/manacles/bed)
			if tile != nil && gs.isDefenselessBaseTile(tile.Index) {
				foundStocks = append(foundStocks, struct {
					x, y  int
					index indexes.SpriteIndex
				}{x, y, tile.Index})
			}
		}
	}

	t.Logf("Found %d defenseless NPCs in Yew:", len(foundStocks))
	for _, npc := range foundStocks {
		t.Logf("  Position (%d, %d): tile index %d (0x%x)", npc.x, npc.y, npc.index, npc.index)
	}

	if len(foundStocks) == 0 {
		t.Logf("⚠️  NO DEFENSELESS NPCs FOUND IN YEW")
		t.Logf("User may be confusing Child NPCs with NPCs in stocks")
		t.Logf("Child1-4 (360-363) are NOT defenseless - they are regular NPCs")
	}

	// Also check what's at the user's reported position
	t.Logf("\n=== User's reported position (20, 16) ===")
	userPos := &references.Position{X: 20, Y: 16}
	userTile := currentMap.GetTopTile(userPos)
	if userTile != nil {
		t.Logf("Tile at (20, 16): index %d (0x%x)", userTile.Index, userTile.Index)
		t.Logf("Is defenseless base tile: %v", gs.isDefenselessBaseTile(userTile.Index))

		// Check what range it falls into
		if userTile.Index >= 360 && userTile.Index <= 363 {
			t.Logf("This is Child%d - a regular NPC, NOT in stocks", userTile.Index-359)
		}
	}
}
