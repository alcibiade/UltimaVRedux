package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// deedTestSeeds covers seed zero (the PRNG's degenerate state), small and large seeds.
var deedTestSeeds = []uint32{0, 1, 7, 0xdeadbeef, DeedSeed(0, Position{X: 130, Y: 100}), DeedSeed(41, Position{X: 3, Y: 250})}

func generateDeedForTest(t *testing.T, terrain DeedTerrain, seed uint32) DeedLayout {
	t.Helper()
	layout, err := GenerateDeedMap(terrain, seed, DeedGeneratorVersion)
	require.NoError(t, err)

	return layout
}

func walkable(sprite indexes.SpriteIndex) bool {
	return (&Tile{Index: sprite}).IsWalkingPassable()
}

func TestDeedMapGenerationIsDeterministicAndSeedSensitive(t *testing.T) {
	for _, terrain := range DeedTerrains() {
		first := generateDeedForTest(t, terrain, 1234)
		again := generateDeedForTest(t, terrain, 1234)
		other := generateDeedForTest(t, terrain, 1235)
		assert.Equal(t, first, again, "%s: same seed, same map", terrain.Key())
		assert.NotEqual(t, first.Tiles, other.Tiles, "%s: another seed, another map", terrain.Key())
	}
}

// TestDeedMapGeneratorVersionOneIsFrozen pins version 1: a change here breaks every saved deed.
// Bump DeedGeneratorVersion and keep version 1 instead of updating these values.
func TestDeedMapGeneratorVersionOneIsFrozen(t *testing.T) {
	layout, err := GenerateDeedMap(DeedTerrainGrassland, 1, 1)
	require.NoError(t, err)
	checksum := uint32(0)
	for x := range XSmallMapTiles {
		for y := range YSmallMapTiles {
			checksum = checksum*31 + uint32(layout.Tiles[x][y])
		}
	}
	assert.Equal(t, uint32(0x3d446ecf), checksum)
	assert.Len(t, layout.Nodes, 27)
}

func TestDeedMapRejectsUnknownVersionsAndTerrains(t *testing.T) {
	_, err := GenerateDeedMap(DeedTerrainGrassland, 1, 0)
	require.Error(t, err)
	_, err = GenerateDeedMap(DeedTerrainGrassland, 1, DeedGeneratorVersion+1)
	require.Error(t, err)
	_, err = GenerateDeedMap(DeedTerrain(99), 1, DeedGeneratorVersion)
	require.Error(t, err)
}

func TestDeedMapLayoutRules(t *testing.T) {
	for _, terrain := range DeedTerrains() {
		profile := DeedProfile(terrain)
		for _, seed := range deedTestSeeds {
			layout := generateDeedForTest(t, terrain, seed)
			name := terrain.Key()

			// The interior is buildable ground, a resource node or the entry lane.
			for x := range XSmallMapTiles {
				for y := range YSmallMapTiles {
					position := Position{X: x, Y: y}
					if DeedSquareDepth(position) < DeedInteriorDepth {
						continue
					}
					_, isNode := layout.NodeAt(position)
					assert.True(t, isNode || IsDeedBuildableGround(layout.Tile(position)),
						"%s seed %d: interior square %v holds %d", name, seed, position, layout.Tile(position))
				}
			}
			// The yard is open and buildable.
			for x := DeedMapCentre.X - DeedYardRadius; x <= DeedMapCentre.X+DeedYardRadius; x++ {
				for y := DeedMapCentre.Y - DeedYardRadius; y <= DeedMapCentre.Y+DeedYardRadius; y++ {
					assert.Equal(t, profile.Ground, layout.Tile(Position{X: x, Y: y}), "%s seed %d: yard %d,%d", name, seed, x, y)
				}
			}
			// The entry lane runs from the south edge through the entry square to the yard.
			assert.True(t, IsDeedEntryPath(DeedMapEntry))
			for y := DeedMapCentre.Y + DeedYardRadius + 1; y < YSmallMapTiles; y++ {
				assert.True(t, walkable(layout.Tile(Position{X: DeedMapEntry.X, Y: y})), "%s seed %d: lane %d", name, seed, y)
			}
			// Nodes match the quota, wear their sprites, keep clear of the ring and never touch.
			counts := map[DeedResourceKind]int{}
			for index, node := range layout.Nodes {
				counts[node.Kind]++
				assert.Equal(t, node.Sprite, layout.Tile(node.Position))
				assert.GreaterOrEqual(t, DeedSquareDepth(node.Position), DeedBorderDepth+DeedNodeMargin)
				for _, other := range layout.Nodes[index+1:] {
					assert.Greater(t, chebyshev(node.Position, other.Position), 1, "%s seed %d: nodes touch", name, seed)
				}
				if node.Kind == DeedResourceReagent {
					found := false
					for _, quota := range profile.Nodes {
						if quota.Kind == DeedResourceReagent {
							for _, reagent := range quota.Reagents {
								found = found || reagent == node.Reagent
							}
						}
					}
					assert.True(t, found, "%s: reagent %d is not wild here", name, node.Reagent)
				}
			}
			for _, quota := range profile.Nodes {
				assert.Positive(t, counts[quota.Kind])
			}
			wantTotal := 0
			for _, quota := range profile.Nodes {
				wantTotal += quota.Count
			}
			assert.Len(t, layout.Nodes, wantTotal, "%s seed %d: every quota is placed", name, seed)
			assertDeedInteriorConnected(t, layout, name, seed)
		}
	}
}

// assertDeedInteriorConnected walks from the south edge of the entry lane and requires every
// walkable interior square to be reached, and every node to have a reachable neighbour.
func assertDeedInteriorConnected(t *testing.T, layout DeedLayout, name string, seed uint32) {
	t.Helper()
	start := Position{X: DeedMapEntry.X, Y: YSmallMapTiles - 1}
	reached := map[Position]bool{start: true}
	queue := []Position{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, step := range []Position{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			next := Position{X: current.X + step.X, Y: current.Y + step.Y}
			if next.X < 0 || next.Y < 0 || next.X >= XSmallMapTiles || next.Y >= YSmallMapTiles ||
				reached[next] || !walkable(layout.Tile(next)) {
				continue
			}
			reached[next] = true
			queue = append(queue, next)
		}
	}
	require.True(t, reached[DeedMapEntry], "%s seed %d: entry unreachable from the south edge", name, seed)
	for x := range XSmallMapTiles {
		for y := range YSmallMapTiles {
			position := Position{X: x, Y: y}
			if DeedSquareDepth(position) >= DeedInteriorDepth && walkable(layout.Tile(position)) {
				assert.True(t, reached[position], "%s seed %d: %v is cut off", name, seed, position)
			}
		}
	}
	for _, node := range layout.Nodes {
		reachable := false
		for _, step := range []Position{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			reachable = reachable || reached[Position{X: node.Position.X + step.X, Y: node.Position.Y + step.Y}]
		}
		assert.True(t, reachable, "%s seed %d: node %v cannot be reached", name, seed, node.Position)
	}
}

func TestDeedTerrainClassificationAndKeys(t *testing.T) {
	for _, terrain := range DeedTerrains() {
		parsed, ok := DeedTerrainFromKey(terrain.Key())
		require.True(t, ok)
		assert.Equal(t, terrain, parsed)
	}
	_, ok := DeedTerrainFromKey("swamp")
	assert.False(t, ok)
	for _, sprite := range []indexes.SpriteIndex{indexes.Water1, indexes.Swamp, indexes.SmallMountains, indexes.TallMountains,
		indexes.Hut, indexes.Village, indexes.Path1, indexes.Bridge, indexes.Lava, indexes.Moongate, indexes.Shrine} {
		_, ok := DeedTerrainForSprite(sprite)
		assert.False(t, ok, "sprite %d", sprite)
	}
}

func TestDeedSeedIsStableAndDistinct(t *testing.T) {
	assert.Equal(t, DeedSeed(3, Position{X: 10, Y: 20}), DeedSeed(3, Position{X: 10, Y: 20}))
	assert.NotEqual(t, DeedSeed(3, Position{X: 10, Y: 20}), DeedSeed(4, Position{X: 10, Y: 20}))
	assert.NotEqual(t, DeedSeed(3, Position{X: 10, Y: 20}), DeedSeed(3, Position{X: 20, Y: 10}))
}

func TestGeneratedSmallLocationReferenceHasNoOriginalData(t *testing.T) {
	layout := generateDeedForTest(t, DeedTerrainDesert, 5)
	reference := NewGeneratedSmallLocationReference(DeedLocation, "Dunes", &layout.Tiles, indexes.Desert1)

	assert.Equal(t, []FloorNumber{0}, reference.ListOfFloors)
	assert.False(t, reference.HasBasement())
	assert.Empty(t, *reference.GetNPCReferences())
	assert.Equal(t, "Enter your land\n\nDunes", reference.EnteringText)
	assert.Equal(t, "Leaving your land\n\nDunes", reference.GetExitingText())
	assert.Equal(t, indexes.SpriteIndex(indexes.Desert1), reference.GetOuterTile())
	assert.Equal(t, layout.Tile(DeedMapEntry), reference.GetTileNumber(DeedMapEntry, 0))
	assert.Equal(t, None, GetMapMasterFromLocation(DeedLocation))
	assert.Equal(t, SmallMapType, DeedLocation.GetMapType())

	reference.SetTileNumber(DeedMapEntry, 0, indexes.BrickFloor)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), reference.GetTileNumber(DeedMapEntry, 0))
}

func TestDeedLocationIsClearOfOriginalAndVirtualLocations(t *testing.T) {
	assert.Greater(t, DeedLocation, CombatRestingShrine)
	assert.Less(t, DeedLocation, FinaleLocation)
	assert.True(t, DeedLocation.IsDeedLocation())
	assert.False(t, Britain.IsDeedLocation())
}
