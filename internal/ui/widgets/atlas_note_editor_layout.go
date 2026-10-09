package widgets

import (
	"image"
	"math"
)

// Field chrome, wrapping, scrolling and caret rendering use the same scaled
// interior. The final logical pixel of each row is reserved for its end caret.
type atlasEditorFieldLayout struct {
	content                          image.Rectangle
	origin                           image.Point
	pixelUnit, textWidth, textHeight int
	lineHeight, capacity             int
}

//nolint:mnd // Insets are classic UI pixels: one frame, two horizontal padding, one vertical padding.
func (e *AtlasNoteEditor) fieldLayout(field int) atlasEditorFieldLayout {
	unit := ClassicUIPixelUnit(e.width, e.height)
	content := e.fieldRects[field].Inset(2 * unit)
	content.Min.X += unit
	content.Max.X -= unit
	textHeight := int(math.Ceil(e.output.GetTotalHeightForLines(1)))
	layout := atlasEditorFieldLayout{
		content: content, origin: content.Min, pixelUnit: unit,
		textWidth: content.Dx() - unit, textHeight: textHeight,
		lineHeight: e.lineHeight, capacity: 0,
	}
	if content.Dy() < textHeight {
		return layout
	}
	layout.capacity = 1 + (content.Dy()-textHeight)/e.lineHeight
	if field == 0 {
		layout.capacity = 1
		layout.origin.Y += (content.Dy() - textHeight) / 2
	}

	return layout
}

func (layout atlasEditorFieldLayout) lineOrigin(visibleRow int) image.Point {
	return layout.origin.Add(image.Pt(0, visibleRow*layout.lineHeight))
}

func (layout atlasEditorFieldLayout) caretRect(visibleRow int, advance float64) image.Rectangle {
	point := layout.lineOrigin(visibleRow)
	point.X += int(math.Ceil(advance))

	return image.Rect(point.X, point.Y, point.X+layout.pixelUnit, point.Y+layout.textHeight)
}
