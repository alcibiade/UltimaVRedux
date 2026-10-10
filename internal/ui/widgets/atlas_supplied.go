package widgets

import (
	"image"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/hajimehoshi/ebiten/v2"
)

//nolint:mnd // Approved decoration shares the surface half-cell projection, not entry authority.
func (d *AtlasDialog) suppliedSymbolCenter(symbol journal.AtlasSymbol) (image.Point, bool) {
	if d.view.Filters.Places != journal.AtlasAllPlaces &&
		symbol.Category != journal.AtlasCategory(d.view.Filters.Places-1) {
		var absent image.Point

		return absent, false
	}
	x, y, ok := d.view.project(float64(symbol.Position.X)+0.5, float64(symbol.Position.Y)+0.5, d.mapRect)

	return image.Pt(int(x), int(y)), ok
}

// Symbols never traverse PlotPosition/selection/name rendering. Coincident
// earned destinations coalesce by neutral identity and position, not discovery.
func (d *AtlasDialog) drawSuppliedSymbols(screen *ebiten.Image) {
	for _, symbol := range d.dossier.Symbols {
		center, visible := d.suppliedSymbolCenter(symbol)
		if !visible {
			continue
		}
		coincident := false
		for _, entry := range d.entries {
			if position, ok := entry.PlotPosition(); ok && entry.Key == symbol.Key && position == symbol.Position {
				coincident = true

				break
			}
		}
		if coincident {
			continue
		}
		stamp := atlasMarkerStamp(symbol.Kind)
		if symbol.Kind == atlasMoonstoneKind {
			stamp = atlasMoonStamp(symbol.Phase)
		}
		drawAtlasStamp(atlasSubImage(screen, d.mapRect), stamp, center,
			ClassicUIPixelUnit(d.width, d.height), DialogAtlasUndiscoveredColor, true)
	}
}
