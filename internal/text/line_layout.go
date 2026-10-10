package text

import (
	"github.com/bradhannah/UltimaVRedux/internal/profiling"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

type segmentGeometry struct {
	text  string
	kind  FontType
	scale float64
}

type lineLayout struct {
	typography CacheKey
	inputs     []segmentGeometry
	advances   []float64
	total      float64
	valid      bool
}

func (o *Output) clearLineLayouts() {
	for index := range o.lineLayouts {
		layout := &o.lineLayouts[index]
		clear(layout.inputs)
		layout.valid = false
	}
}

func (o *Output) layoutLine(row int, segments []ColorSegment) *lineLayout {
	if len(o.lineLayouts) != len(o.lines) {
		o.lineLayouts = make([]lineLayout, len(o.lines))
	}
	layout := &o.lineLayouts[row]
	key := (TextRenderParams{Content: "", Face: o.Font.TextFace,
		Options: &etext.LayoutOptions{
			LineSpacing: o.lineSpacing, PrimaryAlign: etext.AlignStart, SecondaryAlign: etext.AlignStart,
		}, Color: nil}).CacheKey()
	matching := layout.valid && layout.typography == key && len(layout.inputs) == len(segments)
	if matching {
		for index, segment := range segments {
			input := layout.inputs[index]
			if input.text != segment.Text || input.kind != segment.FontType || input.scale != segment.FontScale {
				matching = false

				break
			}
		}
	}
	if matching {
		return layout // Colours and alignment affect paint/placement, not intrinsic advances.
	}
	profiling.Count(profiling.TextLayoutBuilds)
	if cap(layout.inputs) < len(segments) {
		layout.inputs = make([]segmentGeometry, len(segments))
		layout.advances = make([]float64, len(segments))
	}
	if len(layout.inputs) > len(segments) {
		clear(layout.inputs[len(segments):])
	}
	layout.inputs = layout.inputs[:len(segments)]
	layout.advances = layout.advances[:len(segments)]
	layout.total = 0
	for index, segment := range segments {
		layout.inputs[index] = segmentGeometry{text: segment.Text, kind: segment.FontType, scale: segment.FontScale}
		layout.advances[index] = o.getSegmentWidth(segment)
		layout.total += layout.advances[index]
	}
	layout.typography, layout.valid = key, true

	return layout
}
