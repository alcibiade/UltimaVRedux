package main

import "image"

func (g *GameScene) mapPopupViewport(screenBounds image.Rectangle) image.Rectangle {
	if g.gameState != nil && g.gameState.IsInCombat() {
		return combatBattlefieldRectangle(screenBounds)
	}

	return image.Rectangle{}
}
