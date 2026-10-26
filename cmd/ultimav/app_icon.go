package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/appicon"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

// The Windows executable icon is a committed resource object linked from this
// package; regenerate it whenever the icon tile or sprite atlas changes.
//go:generate go run ../../scripts/generate-app-icon -atlas ../../internal/sprites/assets/transparent_u5_tiles.png -output .

// applyApplicationIcon presents the original Avatar tile as the window icon and,
// on macOS, the Dock icon. Icon failures are cosmetic and never block startup.
func applyApplicationIcon() {
	atlas := sprites.TileAtlasPNG()
	icons, err := appicon.WindowIcons(atlas)
	if err != nil {
		logging.Warnf(logging.CategorySystem, "Application icon unavailable: %v", err)

		return
	}
	ebiten.SetWindowIcon(icons)
	if err := setDockIcon(atlas); err != nil {
		logging.Warnf(logging.CategorySystem, "Dock icon unavailable: %v", err)
	}
}
