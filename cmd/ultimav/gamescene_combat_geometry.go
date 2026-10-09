package main

import (
	"image"
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

const (
	combatViewportStartColumn = 0
	combatViewportStartRow    = 1
	combatMapCenterCoordinate = 5
	// The outer screen border is wider than the combat HUD border at the map edges.
	combatLeftGutterSourcePixels       = 8
	combatRightGutterSourcePixels      = 3
	combatBattlefieldWidthSourcePixels = int(references.XCombatMapTiles) * sprites.TileSize
	combatHUDStartSourcePixel          = combatLeftGutterSourcePixels +
		combatBattlefieldWidthSourcePixels + combatRightGutterSourcePixels
)

func combatPositionToViewportTile(position references.Position) (int, int) {
	return combatViewportStartColumn + int(position.X),
		combatViewportStartRow + int(position.Y)
}

func combatPositionToSourcePixels(position references.Position) (float64, float64) {
	tileX, tileY := combatPositionToViewportTile(position)
	sourceX, sourceY := sprites.ConvertPosition16to18(tileX, tileY)

	return sourceX + combatLeftGutterSourcePixels, sourceY
}

func combatViewportSourceX(screenBounds image.Rectangle, sourcePixel int) int {
	mapSourceWidth := xTilesVisibleOnGameScreen * sprites.TileSize
	mapFraction := gameScreenPercents.StartPercentX +
		(gameScreenPercents.EndPercentX-gameScreenPercents.StartPercentX)*
			float64(sourcePixel)/float64(mapSourceWidth)

	return screenBounds.Min.X + int(math.Round(float64(screenBounds.Dx())*mapFraction))
}

func combatViewportRowY(screenBounds image.Rectangle, row int) int {
	mapFraction := gameScreenPercents.StartPercentY +
		(gameScreenPercents.EndPercentY-gameScreenPercents.StartPercentY)*
			float64(row)/float64(yTilesVisibleOnGameScreen)

	return screenBounds.Min.Y + int(math.Round(float64(screenBounds.Dy())*mapFraction))
}

func combatBattlefieldRectangle(screenBounds image.Rectangle) image.Rectangle {
	return image.Rect(
		combatViewportSourceX(screenBounds, combatLeftGutterSourcePixels),
		combatViewportRowY(screenBounds, combatViewportStartRow),
		combatViewportSourceX(
			screenBounds,
			combatLeftGutterSourcePixels+combatBattlefieldWidthSourcePixels,
		),
		combatViewportRowY(screenBounds, combatViewportStartRow+int(references.YCombatMapTiles)),
	)
}

func combatHUDRectangle(screenBounds image.Rectangle) image.Rectangle {
	return image.Rect(
		combatViewportSourceX(screenBounds, combatHUDStartSourcePixel),
		combatViewportRowY(screenBounds, 0),
		combatViewportSourceX(screenBounds, xTilesVisibleOnGameScreen*sprites.TileSize),
		combatViewportRowY(screenBounds, yTilesVisibleOnGameScreen),
	)
}
