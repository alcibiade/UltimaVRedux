package main

import (
	"image/color"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestCombatCorpseCompositesRealTerrainThroughAtlasAlpha(t *testing.T) {
	for _, terrain := range []indexes.SpriteIndex{indexes.Grass, indexes.BrickFloor, indexes.Lava} {
		for _, territory := range []references.CombatMapTerritory{references.BritanniaCombatTerritory, references.DungeonCombatTerritory} {
			scene := newCombatConcealmentTestState(t)
			state := scene.gameState
			state.CombatScene.CombatMapRef = references.NewCombatMapReference(0, territory)
			position := references.Position{X: 5, Y: 5}
			setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
			// The underlay must follow effective overrides, not the immutable base.
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &position, terrain)
			scene.unscaledMapImage = ebiten.NewImage(18, 18)
			t.Cleanup(scene.unscaledMapImage.Deallocate)
			draw := func() []byte {
				scene.unscaledMapImage.Clear()
				scene.refreshStaticMapTiles(&position, state.GetCurrentLayeredMap(), &ebiten.DrawImageOptions{})
				pixels := make([]byte, 18*18*4)
				scene.unscaledMapImage.ReadPixels(pixels)
				return pixels
			}
			baseline := draw()
			require.True(t, state.CombatScene.AddObject(game_state.NewCombatCorpse(position)))
			date, calls, objects := state.DateTime, state.GetRNG().Calls(), state.CombatScene.ObjectCount()
			actual := draw()
			art := scene.spriteSheet.GetSprite(indexes.Splat)
			clearPixels, bloodPixels := 0, 0
			for y := 1; y <= 16; y++ {
				for x := 1; x <= 16; x++ {
					pixel := color.RGBAModel.Convert(art.At(x, y)).(color.RGBA)
					offset := (y*18 + x) * 4
					if pixel.A == 0 {
						clearPixels++
						require.Equal(t, baseline[offset:offset+4], actual[offset:offset+4], "ground through corpse at %d,%d", x, y)
					} else if pixel.R > pixel.G && pixel.R > pixel.B {
						bloodPixels++
						require.Equal(t, []byte{pixel.R, pixel.G, pixel.B, pixel.A}, actual[offset:offset+4], "blood keeps its own palette")
					}
				}
			}
			require.Positive(t, clearPixels, "Splat artwork must have transparent background within its logical tile")
			require.Positive(t, bloodPixels)
			require.Equal(t, date, state.DateTime)
			require.Equal(t, calls, state.GetRNG().Calls())
			require.Equal(t, objects, state.CombatScene.ObjectCount())
		}
	}
}
