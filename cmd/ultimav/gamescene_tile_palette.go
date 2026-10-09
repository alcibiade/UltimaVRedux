package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/camping"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type tilePaletteMode uint8

const (
	tilePaletteNormal tilePaletteMode = iota
	tilePaletteDungeon
	tilePaletteFinale
)

func (g *GameScene) drawTilePaletteSprite(
	target, sprite *ebiten.Image,
	op *ebiten.DrawImageOptions,
	spriteIndex indexes.SpriteIndex,
) {
	var shader *ebiten.Shader
	switch g.tilePaletteMode() {
	case tilePaletteDungeon:
		if dungeonPaletteRemapsSprite(spriteIndex) {
			shader = getDungeonPaletteShader()
		}
	case tilePaletteFinale:
		if finalePaletteRemapsSprite(spriteIndex) {
			shader = getFinalePaletteShader()
		}
	case tilePaletteNormal:
	}
	if shader == nil {
		target.DrawImage(sprite, op)

		return
	}
	target.DrawRectShader(sprite.Bounds().Dx(), sprite.Bounds().Dy(), shader, &ebiten.DrawRectShaderOptions{
		GeoM:       op.GeoM,
		ColorScale: op.ColorScale,
		Blend:      op.Blend,
		Images:     [4]*ebiten.Image{sprite, nil, nil, nil},
	})
}

func (g *GameScene) tilePaletteMode() tilePaletteMode {
	if g.finalePresentation != nil {
		return tilePaletteFinale
	}
	if g.gameState == nil {
		return tilePaletteNormal
	}
	if g.gameState.IsInCombat() && g.gameState.CombatScene != nil &&
		g.gameState.CombatScene.CombatMapRef != nil &&
		g.gameState.CombatScene.CombatMapRef.Territory == references.DungeonCombatTerritory {
		return tilePaletteDungeon
	}
	if g.gameState.IsInCamping() && g.gameState.CampingScene != nil &&
		g.gameState.CampingScene.CampingMap != nil &&
		g.gameState.CampingScene.CampingMap.CampingType == camping.DungeonCamping {
		return tilePaletteDungeon
	}

	return tilePaletteNormal
}

func dungeonPaletteRemapsSprite(sprite indexes.SpriteIndex) bool {
	// Root EGA.DRV _SPARE4_ command 3 swaps bitplanes 1 and 2 for these exact tiles.
	//nolint:exhaustive // Only the original driver's exact affected tile set should return true.
	switch sprite {
	case indexes.Grass,
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
		indexes.FenceHoriz:
		return true
	default:
		return false
	}
}

func finalePaletteRemapsSprite(sprite indexes.SpriteIndex) bool {
	//nolint:exhaustive // Only the EGA driver's exact affected tile set should return true.
	switch sprite {
	case indexes.BrickFloor,
		indexes.BookcaseLeft, indexes.BookcaseRight,
		indexes.ChairBackForward, indexes.ChairBackBack,
		indexes.TableLeft, indexes.TableRight, indexes.TableFoodBottom,
		indexes.LeftBed, indexes.RightBed, indexes.Box,
		indexes.RightSconce, indexes.LeftSconce, indexes.CookStove,
		indexes.Moongate, indexes.ItemGem, indexes.ItemSandalwoodBox,
		indexes.SleepingInBed,
		indexes.SitChairUpEat1, indexes.SitChairUpEat2,
		indexes.SitChairUpEat3, indexes.SitChairUpEat4:
		return true
	default:
		return false
	}
}
