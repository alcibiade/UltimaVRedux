package widgets

import (
	"image"
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/atlasterrain"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
)

// AtlasLabelMode controls the selected one-or-all place callouts, never region names.
type AtlasLabelMode uint8

// The label key preserves one show/hide preference across real and overview rows.
const (
	AtlasLabelsSelected AtlasLabelMode = iota
	AtlasLabelsNone
)

// AtlasViewState is scene-local reader preference, never native gameplay state.
type AtlasViewState struct {
	Initialized      bool
	CenterX, CenterY float64
	Zoom             int
	Filters          journal.AtlasFilters
	Labels           AtlasLabelMode
	Grid             bool
	SelectedKey      string
	Overview         bool
}

// NewAtlasViewState starts at the whole world, without consulting an Avatar site.
func NewAtlasViewState() AtlasViewState {
	const worldCenter = atlasterrain.WorldSize / 2

	return AtlasViewState{Initialized: true, CenterX: worldCenter, CenterY: worldCenter, Zoom: 1,
		Filters: journal.DefaultAtlasFilters(), Labels: AtlasLabelsSelected,
		Grid: false, SelectedKey: "", Overview: false}
}

func atlasModulo(value float64) float64 {
	return value - atlasterrain.WorldSize*math.Floor(value/atlasterrain.WorldSize)
}
func (v AtlasViewState) span() float64 { return atlasterrain.WorldSize / float64(v.Zoom) }
func (v AtlasViewState) origin() (float64, float64) {
	if v.Zoom == 1 {
		return 0, 0
	}

	return atlasModulo(v.CenterX - v.span()/2), atlasModulo(v.CenterY - v.span()/2)
}

// project is shared by points, grid and labels. Coordinates refer to tile centres.
func (v AtlasViewState) project(x, y float64, rect image.Rectangle) (float64, float64, bool) {
	ox, oy := v.origin()
	dx, dy := atlasModulo(x-ox), atlasModulo(y-oy)

	return float64(rect.Min.X) + dx*float64(rect.Dx())/v.span(),
		float64(rect.Min.Y) + dy*float64(rect.Dy())/v.span(), dx < v.span() && dy < v.span()
}

type atlasTextureRect struct {
	source      image.Rectangle
	x, y, scale float64
}

// textureRects returns at most four non-overlapping slices of one retained image.
//
//nolint:varnamelen // Source/destination x/y denote the four wrapped rectangles.
func (v AtlasViewState) textureRects(rect image.Rectangle) []atlasTextureRect {
	ox, oy := v.origin()
	x, y, span := int(ox*atlasterrain.TileSize), int(oy*atlasterrain.TileSize), int(v.span()*atlasterrain.TileSize)
	xFirst, yFirst := min(span, atlasterrain.ImageSize-x), min(span, atlasterrain.ImageSize-y)
	var result []atlasTextureRect
	for yi, height := range []int{yFirst, span - yFirst} {
		if height == 0 {
			continue
		}
		for xi, width := range []int{xFirst, span - xFirst} {
			if width == 0 {
				continue
			}
			sx, sy, dx, dy := x, y, 0, 0
			if xi == 1 {
				sx = 0
				dx = xFirst
			}
			if yi == 1 {
				sy = 0
				dy = yFirst
			}
			scale := float64(rect.Dx()) / float64(span)
			result = append(result, atlasTextureRect{image.Rect(sx, sy, sx+width, sy+height),
				float64(rect.Min.X) + float64(dx)*scale, float64(rect.Min.Y) + float64(dy)*scale, scale})
		}
	}

	return result
}
