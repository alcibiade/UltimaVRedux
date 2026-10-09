package main

import "github.com/bradhannah/UltimaVRedux/internal/references"

func (g *GameScene) appendDirectionToOutput(direction references.Direction) {
	g.appendToCurrentRowStr(direction.GetDirectionCompassName())
}
