package map_state

import (
	"fmt"
	"slices"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/require"
)

type sourceCollectionDimensions struct {
	size references.Coordinate
	wrap bool
}

func (d sourceCollectionDimensions) GetTilesVisibleOnScreen() (int, int) { return 19, 13 }
func (d sourceCollectionDimensions) GetTopLeftExtent() references.Position {
	return references.Position{}
}
func (d sourceCollectionDimensions) GetBottomRightExtent() references.Position {
	return references.Position{X: d.size, Y: d.size}
}
func (d sourceCollectionDimensions) GetBottomRightWithoutOverflow() references.Position {
	return d.GetBottomRightExtent()
}
func (d sourceCollectionDimensions) IsWrappedMap() bool { return d.wrap }

func newSourceCollectionFixture(size references.Coordinate, wrap bool) (*LayeredMap, Lighting) {
	layer := newLayeredMap(size, size, references.NewTileReferences(), 19, 13, wrap)
	for x := -coordinateOffset; x < size+overflowTiles; x++ {
		for y := -coordinateOffset; y < size+overflowTiles; y++ {
			position := references.Position{X: x, Y: y}
			var sprite indexes.SpriteIndex = indexes.BrickFloor
			if (int(x)*31+int(y)*17)&15 == 0 {
				sprite = indexes.Brazier
			}
			layer.SetTileByLayer(MapLayer, &position, sprite)
		}
	}
	return layer, NewLighting(sourceCollectionDimensions{size: size, wrap: wrap})
}

func TestVisibleLightCollectionPreservesLegacyOrderDistanceAndTint(t *testing.T) {
	for _, size := range []references.Coordinate{256, 32, 11, 4} {
		for _, center := range []references.Position{
			{X: 0, Y: 0}, {X: size - 1, Y: 0}, {X: 0, Y: size - 1},
			{X: size - 1, Y: size - 1}, {X: size / 2, Y: size / 2},
		} {
			t.Run(fmt.Sprintf("%d/%v", size, center), func(t *testing.T) {
				layer, lighting := newSourceCollectionFixture(size, size == 256)
				gates := []references.Position{center, center}
				quests := []QuestItemLightInfo{{Position: center, MarkerIndex: ShardMarkerIndex, LightRadius: 3}}
				npcs := []NPCLightInfo{{Position: center, MarkerIndex: WispMarkerIndex, LightRadius: 2}}
				for _, override := range []indexes.SpriteIndex{indexes.NoSprites, indexes.StoneBrickWall, indexes.RightSconce} {
					target := center.Right()
					layer.SetTileByLayer(MapOverrideLayer, &target, override)
					layer.RecalculateVisibleTilesWithCustomLights(center, &lighting, gates, quests, npcs, PlayerLightTorch)
					require.LessOrEqual(t, len(layer.visiblePositions), 576)
					positions := make(map[references.Position]bool)
					for _, position := range layer.visiblePositions {
						positions[position] = true
					}
					for x, row := range layer.visibleFlags {
						for y, visible := range row {
							if visible {
								require.True(t, positions[references.Position{X: x, Y: y}])
							}
						}
					}
					legacy := layer.getAllLightSourcesInRange(center)
					filtered := make(LightSources, 0)
					for _, source := range legacy {
						if layer.visibleFlags[source.Pos.X][source.Pos.Y] {
							filtered = append(filtered, source)
						}
					}
					require.Equal(t, filtered, layer.getVisibleStaticLightSources(center))
					layer.addMoongateLightSources(&legacy, gates)
					layer.addQuestItemLightSources(&legacy, quests)
					layer.addNPCLightSources(&legacy, npcs)
					require.Equal(t, lighting.BuildLightSourceDistanceMap(legacy, layer.visibleFlags, PlayerLightTorch, center),
						layer.GetLightSourcesDistanceMap())
					expected, actual := NewLightSourceTintingState(), NewLightSourceTintingState()
					expected.SetLargeMapMode(size == 256)
					actual.SetLargeMapMode(size == 256)
					expected.BuildLitTilesMap(legacy, layer.visibleFlags)
					actual.BuildLitTilesMap(layer.GetLastComputedLightSources(), layer.visibleFlags)
					require.Equal(t, expected.litTilesMap, actual.litTilesMap, "including capped/equal-weight contribution order")
					for position := range expected.litTilesMap {
						for _, darkness := range []float32{0, 0.5, 1} {
							require.Equal(t, expected.GetTintScaleFactors(position, 1234, darkness),
								actual.GetTintScaleFactors(position, 1234, darkness))
						}
					}
				}
			})
		}
	}
}

func TestCustomCollectionPreservesPrimedLegacyCache(t *testing.T) {
	layer, lighting := newSourceCollectionFixture(32, false)
	center := references.Position{X: 16, Y: 16}
	layer.RecalculateVisibleTilesWithMoongates(center, &lighting, nil, PlayerLightNone)
	require.True(t, layer.lightSourceCacheValid)
	legacy := slices.Clone(layer.cachedLightSources)
	layer.RecalculateVisibleTilesWithCustomLights(center, &lighting, nil, nil, nil, PlayerLightNone)
	require.Equal(t, legacy, layer.cachedLightSources)
	layer.RecalculateVisibleTilesWithMoongates(center, &lighting, nil, PlayerLightNone)
	require.Equal(t, legacy, layer.cachedLightSources)
}

func BenchmarkStaticLightCollection(b *testing.B) {
	layer, lighting := newSourceCollectionFixture(256, true)
	center := references.Position{X: 0, Y: 0}
	layer.RecalculateVisibleTilesWithCustomLights(center, &lighting, nil, nil, nil, PlayerLightNone)
	for _, collector := range []struct {
		name    string
		collect func(references.Position) LightSources
	}{{"exhaustive", layer.getAllLightSourcesInRange}, {"visible", layer.getVisibleStaticLightSources}} {
		b.Run(collector.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				if len(collector.collect(center)) == 0 {
					b.Fatal("fixture must include emitters")
				}
			}
		})
	}
}

func BenchmarkLightingRecalculation(b *testing.B) {
	for _, exhaustive := range []bool{true, false} {
		b.Run(fmt.Sprintf("exhaustive=%t", exhaustive), func(b *testing.B) {
			layer, lighting := newSourceCollectionFixture(256, true)
			// The compatibility branch retains the exhaustive collector while
			// performing the same current LOS/position bookkeeping in both cases.
			layer.lightSourceCacheValid = exhaustive
			center := references.Position{X: 0, Y: 0}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				layer.RecalculateVisibleTilesWithCustomLights(center, &lighting, nil, nil, nil, PlayerLightTorch)
			}
		})
	}
}
