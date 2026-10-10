package references

import (
	"math"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestNPCTerrainPassabilityAllOriginalTileIDs(t *testing.T) {
	// Independent allowed-ID oracle for NPC terrain, not a bitmap decode; ref:NPC-0004.
	allowedIDs := []int{
		0, 4, 5, 6, 7, 8, 9, 10, 11, 14, 15,
		29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 44, 45,
		64, 68, 69, 71, 72, 73, 106, 107, 135,
		144, 145, 146, 147, 170, 184, 186, 196, 197, 198, 199, 200, 201,
	}
	preferredCosts := map[int]int{
		5:  3,
		32: 2, 33: 2, 34: 2, 35: 2, 36: 2, 37: 2, 38: 2,
		64: 1, 68: 1, 69: 1, 72: 1, 73: 1, 184: 1, 186: 1,
	}
	for id := range 256 {
		tile := Tile{Index: indexes.SpriteIndex(id)}
		allowed := slices.Contains(allowedIDs, id)
		assert.Equal(t, allowed, tile.IsNPCTerrainPassable(), "raw NPC terrain for tile %d", id)
		assert.Equal(t, allowed, tile.IsPassable(NPC), "NPC dispatch for tile %d", id)

		wantWeight := -1
		if allowed {
			wantWeight = 10
			if cost, ok := preferredCosts[id]; ok {
				wantWeight = cost
			}
		}
		assert.Equal(t, wantWeight, tile.GetNPCWalkableWeight(), "NPC weight for tile %d", id)

		wantWander := (allowed || (id >= 48 && id <= 51)) && !(id >= 144 && id <= 147)
		assert.Equal(t, wantWander, tile.IsWalkableDuringWander(), "NPC wandering for tile %d", id)
	}
}

func TestNPCWalkableWeightDoesNotInheritPlayerExclusions(t *testing.T) {
	// Exercise the fallback independently of the current player allowlist.
	previous := walkingPassableTiles[indexes.Grass]
	t.Cleanup(func() {
		walkingPassableTiles[indexes.Grass] = previous
	})
	delete(walkingPassableTiles, indexes.Grass)

	tile := Tile{Index: indexes.Grass}
	assert.Equal(t, -1, tile.GetWalkableWeight())
	assert.True(t, tile.IsNPCTerrainPassable())
	assert.True(t, tile.IsPassable(NPC))
	assert.True(t, tile.IsWalkableDuringWander())
	assert.Equal(t, 10, tile.GetNPCWalkableWeight())
}

func TestNPCTerrainPassabilityRange(t *testing.T) {
	for _, id := range []int{math.MinInt, -256, -1, 256, 257, 282, 512, math.MaxInt} {
		tile := Tile{Index: indexes.SpriteIndex(id)}
		assert.False(t, tile.IsNPCTerrainPassable(), "out-of-range terrain %d", id)
		assert.False(t, tile.IsPassable(NPC), "out-of-range NPC dispatch %d", id)
		assert.False(t, tile.IsWalkableDuringWander(), "out-of-range wandering %d", id)
		assert.Equal(t, -1, tile.GetNPCWalkableWeight(), "out-of-range weight %d", id)
	}

	for _, tc := range []struct {
		id      int
		allowed bool
	}{
		{0, true},
		{127, false},
		{128, false},
		{135, true},
		{201, true},
		{255, false},
	} {
		tile := Tile{Index: indexes.SpriteIndex(tc.id)}
		assert.Equal(t, tc.allowed, tile.IsNPCTerrainPassable(), "unsigned tile %d", tc.id)
	}
}

func TestNPCWanderingTerrainExceptions(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ids     []int
		raw     bool
		wander  bool
		walking bool
	}{
		{"beaches only override wandering", []int{48, 49, 50, 51}, false, true, true},
		{"coasts remain blocked", []int{52, 53, 54, 55}, false, false, true},
		{"chairs block wandering only", []int{144, 145, 146, 147}, true, false, true},
		{"unlocked doors use raw eligibility", []int{184, 186}, true, true, false},
		{"locked doors have no destination override", []int{151, 152, 185, 187}, false, false, false},
		{"beds have no destination override", []int{171, 172}, false, false, true},
		{"stairs and ladders retain raw eligibility", []int{196, 197, 198, 199, 200, 201}, true, true, true},
		{"hazards remain blocked", []int{134, 140, 143, 188}, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, id := range tc.ids {
				tile := Tile{Index: indexes.SpriteIndex(id)}
				assert.Equal(t, tc.raw, tile.IsNPCTerrainPassable(), "raw terrain %d", id)
				assert.Equal(t, tc.raw, tile.IsPassable(NPC), "NPC dispatch %d", id)
				assert.Equal(t, tc.raw, tile.GetNPCWalkableWeight() > 0, "NPC weighted eligibility %d", id)
				assert.Equal(t, tc.wander, tile.IsWalkableDuringWander(), "wandering %d", id)
				assert.Equal(t, tc.walking, tile.IsWalkingPassable(), "player terrain %d", id)
			}
		})
	}
}

func TestNPCTerrainRulesPreservePlayerAndLandEnemyPassability(t *testing.T) {
	// Snapshot of the existing player allowlist, independent of NPC terrain rules.
	playerAllowedIDs := []int{
		0, 4, 5, 6, 7, 8, 9, 10, 11, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 27,
		29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 44, 45, 48, 49, 50, 51, 52, 53, 54, 55,
		57, 62, 64, 68, 69, 71, 72, 73, 106, 107, 134, 135, 140, 143, 144, 145, 146, 147,
		170, 171, 172, 188, 196, 197, 198, 199, 200, 201, 220, 221, 249, 255,
	}
	for id := range 256 {
		tile := Tile{Index: indexes.SpriteIndex(id)}
		allowed := slices.Contains(playerAllowedIDs, id)
		assert.Equal(t, allowed, tile.IsWalkingPassable(), "player terrain %d", id)
		assert.Equal(t, allowed, tile.IsPassable(NoPartyVehicle), "player dispatch %d", id)
		assert.Equal(t, allowed, tile.IsLandEnemyPassable(), "land enemy terrain %d", id)

		wantWeight := -1
		if allowed {
			wantWeight = 10
		}
		switch id {
		case 64, 68, 69, 72, 73, 184, 186:
			wantWeight = 1
		case 32, 33, 34, 35, 36, 37, 38:
			wantWeight = 2
		case 5:
			wantWeight = 3
		}
		assert.Equal(t, wantWeight, tile.GetWalkableWeight(), "player weight %d", id)
	}
}
