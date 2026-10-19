package widgets

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/text"
)

// inputTextGeometry uses the complete shaped line for selection and caret
// positions. Byte offsets refer to the host's displayed/preedit string; native
// sessions, source projection, validation and interaction stay in each widget.
func inputTextGeometry(font *text.UltimaFont, value string, start, end, caret int) (float64, float64, float64) {
	layout := font.Layout(value)
	coverage, err := layout.RangeRects(start, end)
	if err != nil {
		panic(fmt.Sprintf("invalid host selection geometry: %v", err))
	}
	site, err := layout.CaretAt(caret, true)
	if err != nil {
		panic(fmt.Sprintf("invalid host caret geometry: %v", err))
	}
	left, right := site.Position.X, site.Position.X
	for index, rect := range coverage.Rects {
		if index == 0 {
			left, right = rect.Min.X, rect.Max.X
		} else {
			left, right = min(left, rect.Min.X), max(right, rect.Max.X)
		}
	}

	return left, right - left, site.Position.X
}
