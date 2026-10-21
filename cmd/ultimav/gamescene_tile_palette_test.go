package main

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/camping"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestDungeonPaletteShaderMatchesOriginalEGAPlaneSwap(t *testing.T) {
	sourceColors := []ucolor.EGAColor{
		ucolor.EGABlack, ucolor.EGABlue, ucolor.EGAGreen, ucolor.EGACyan,
		ucolor.EGARed, ucolor.EGAMagenta, ucolor.EGABrown, ucolor.EGALightGray,
		ucolor.EGADarkGray, ucolor.EGALightBlue, ucolor.EGALightGreen, ucolor.EGALightCyan,
		ucolor.EGALightRed, ucolor.EGALightMagenta, ucolor.EGAYellow, ucolor.EGAWhite,
	}
	want := []ucolor.EGAColor{
		ucolor.EGABlack, ucolor.EGABlue, ucolor.EGARed, ucolor.EGAMagenta,
		ucolor.EGAGreen, ucolor.EGACyan, ucolor.EGABrown, ucolor.EGALightGray,
		ucolor.EGADarkGray, ucolor.EGALightBlue, ucolor.EGALightRed, ucolor.EGALightMagenta,
		ucolor.EGALightGreen, ucolor.EGALightCyan, ucolor.EGAYellow, ucolor.EGAWhite,
	}
	source := ebiten.NewImage(len(sourceColors)+1, 1)
	for index, egaColor := range sourceColors {
		source.Set(index, 0, egaColor)
	}
	source.Set(len(sourceColors), 0, color.RGBA{R: 0, G: 170, B: 0, A: 128})
	target := ebiten.NewImage(source.Bounds().Dx(), 1)
	target.DrawRectShader(target.Bounds().Dx(), 1, getDungeonPaletteShader(), &ebiten.DrawRectShaderOptions{
		Images: [4]*ebiten.Image{source, nil, nil, nil},
	})

	for index, egaColor := range want {
		assert.Equal(t, egaColor.ToRGBA(), target.At(index, 0), "EGA index %d", index)
	}
	assert.Equal(t, color.RGBA{R: 170, G: 0, B: 0, A: 128}, target.At(len(sourceColors), 0))
}

func TestDungeonPaletteRemapsOnlyOriginalDriverTileSet(t *testing.T) {
	affected := []indexes.SpriteIndex{
		indexes.Grass,
		indexes.LeftDesert2, indexes.RightDesert2,
		indexes.Path1, indexes.Path2, indexes.Path3, indexes.Path4,
		indexes.Path5, indexes.Path6, indexes.Path7,
		indexes.OutsidePath1, indexes.OutsidePath2, indexes.OutsidePath3, indexes.OutsidePath4,
		indexes.OutsideWaterCoast1, indexes.OutsideWaterCoast2,
		indexes.OutsideWaterCoast3, indexes.OutsideWaterCoast4,
		indexes.SmallRockWall,
		indexes.WaterStream1, indexes.WaterStream2, indexes.WaterStream3, indexes.WaterStream4,
		indexes.WaterStream5, indexes.WaterStream6, indexes.WaterStream7, indexes.WaterStream8,
		indexes.WaterStream9, indexes.WaterStream10,
		indexes.TrollBridgeHoriz, indexes.TrollBridgeVert,
		indexes.WaterStream11, indexes.WaterStream12, indexes.WaterStream13, indexes.WaterStream14,
		indexes.FenceHoriz,
	}
	require.Len(t, affected, 36)
	for _, sprite := range affected {
		assert.True(t, dungeonPaletteRemapsSprite(sprite), "sprite %d", sprite)
	}
	for _, sprite := range []indexes.SpriteIndex{
		indexes.Swamp, indexes.Forest1, indexes.Bridge, indexes.WoodFloor1,
		indexes.LargeRockWall, indexes.ShadowlordBoundary1, indexes.FenceVert,
	} {
		assert.False(t, dungeonPaletteRemapsSprite(sprite), "sprite %d", sprite)
	}
}

func TestTilePaletteModeFollowsDisplayedDungeonContext(t *testing.T) {
	state := new(game_state.GameState)
	scene := &GameScene{gameState: state}
	assert.Equal(t, tilePaletteNormal, scene.tilePaletteMode())

	state.MapState.PlayerLocation.Location = references.CombatLocation
	state.CombatScene = &game_state.CombatState{ //nolint:exhaustruct // Palette context needs only map territory.
		CombatMapRef: references.NewCombatMapReference(0, references.DungeonCombatTerritory),
	}
	assert.Equal(t, tilePaletteDungeon, scene.tilePaletteMode())

	state.CombatScene.CombatMapRef = references.NewCombatMapReference(0, references.BritanniaCombatTerritory)
	assert.Equal(t, tilePaletteNormal, scene.tilePaletteMode())

	state.CombatScene = nil
	state.MapState.PlayerLocation.Location = references.CampingLocation
	state.CampingScene = camping.NewDungeonCampingScene(1, -1, references.PlayerLocation{}, &references.CombatMapReference{})
	assert.Equal(t, tilePaletteDungeon, scene.tilePaletteMode())

	state.CampingScene = camping.NewCampingSceneWithCombatRef(1, -1, references.PlayerLocation{}, &references.CombatMapReference{})
	assert.Equal(t, tilePaletteNormal, scene.tilePaletteMode())

	state.CampingScene = camping.NewDungeonCampingScene(1, -1, references.PlayerLocation{}, &references.CombatMapReference{})
	state.MapState.PlayerLocation.Location = references.Shame
	assert.Equal(t, tilePaletteNormal, scene.tilePaletteMode(), "stale dungeon camping state must not recolor exploration")

	scene.finalePresentation = new(lordBritishFinalePresentation)
	assert.Equal(t, tilePaletteFinale, scene.tilePaletteMode())
}

func TestTilePaletteModeCoversEveryDungeonTacticalOrigin(t *testing.T) {
	for name, configureOrigin := range map[string]func(*game_state.CombatState){
		"authored Shame room": func(combat *game_state.CombatState) {
			combat.DungeonRoomOrigin = &game_state.DungeonRoomEncounterOrigin{Location: references.Shame}
		},
		"generated wanderer encounter": func(combat *game_state.CombatState) {
			combat.DungeonWandererOrigin = &game_state.DungeonWandererEncounterOrigin{Location: references.Shame}
		},
		"dungeon camping ambush": func(combat *game_state.CombatState) {
			combat.CampingOrigin = &game_state.CampingCombatOrigin{}
		},
	} {
		t.Run(name, func(t *testing.T) {
			state := new(game_state.GameState)
			state.MapState.PlayerLocation.Location = references.CombatLocation
			state.CombatScene = &game_state.CombatState{ //nolint:exhaustruct // Palette context needs only map territory and origin.
				CombatMapRef: references.NewCombatMapReference(0, references.DungeonCombatTerritory),
			}
			configureOrigin(state.CombatScene)

			scene := &GameScene{gameState: state}
			assert.Equal(t, tilePaletteDungeon, scene.tilePaletteMode())
		})
	}
}

func TestDrawTilePaletteSpriteAppliesDungeonModeBeforeColorScale(t *testing.T) {
	state := new(game_state.GameState)
	state.MapState.PlayerLocation.Location = references.CombatLocation
	state.CombatScene = &game_state.CombatState{ //nolint:exhaustruct // Palette context needs only map territory.
		CombatMapRef: references.NewCombatMapReference(0, references.DungeonCombatTerritory),
	}
	scene := &GameScene{gameState: state}
	source := ebiten.NewImage(1, 1)
	source.Set(0, 0, ucolor.EGAGreen)

	affectedTarget := ebiten.NewImage(1, 1)
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.Scale(0.5, 0.5, 0.5, 1)
	scene.drawTilePaletteSprite(affectedTarget, source, op, indexes.Grass)
	assert.Equal(t, color.RGBA{R: 85, G: 0, B: 0, A: 255}, affectedTarget.At(0, 0))

	unaffectedTarget := ebiten.NewImage(1, 1)
	scene.drawTilePaletteSprite(unaffectedTarget, source, op, indexes.Forest1)
	assert.Equal(t, color.RGBA{R: 0, G: 85, B: 0, A: 255}, unaffectedTarget.At(0, 0))
}
