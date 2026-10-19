package text

import (
	"fmt"
	"strconv"

	beradtext "github.com/bradhannah/berad-sdk/modules/berad-text"

	"github.com/bradhannah/UltimaVRedux/internal/profiling"
)

type segmentGeometry struct {
	text  string
	kind  FontType
	scale float64
}

// Physical-row semantics stay local; shaped segment layouts are SDK-owned.
type lineLayout struct {
	typography [32]byte
	spacing    float64
	inputs     []segmentGeometry
	advances   []float64
	layouts    []*beradtext.Layout
	combined   *beradtext.Layout
	palette    beradtext.Palette
	paintNames []string
	total      float64
	valid      bool
}

func (o *Output) clearLineLayouts() {
	for index := range o.lineLayouts {
		layout := &o.lineLayouts[index]
		clear(layout.inputs)
		clear(layout.layouts)
		layout.combined = nil
		clear(layout.palette)
		layout.valid = false
	}
}

func (o *Output) layoutLine(row int, segments []ColorSegment) *lineLayout {
	if len(o.lineLayouts) != len(o.lines) {
		o.lineLayouts = make([]lineLayout, len(o.lines))
	}
	layout := &o.lineLayouts[row]
	key := o.Font.sdkForScale(1).Identity()
	matching := layout.valid && layout.typography == key && layout.spacing == o.lineSpacing &&
		len(layout.inputs) == len(segments)
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
		return layout
	}
	profiling.Count(profiling.TextLayoutBuilds)
	if cap(layout.inputs) < len(segments) {
		layout.inputs = make([]segmentGeometry, len(segments))
		layout.advances = make([]float64, len(segments))
		layout.layouts = make([]*beradtext.Layout, len(segments))
	}
	if len(layout.inputs) > len(segments) {
		clear(layout.inputs[len(segments):])
		clear(layout.layouts[len(segments):])
	}
	layout.inputs = layout.inputs[:len(segments)]
	layout.advances = layout.advances[:len(segments)]
	layout.layouts = layout.layouts[:len(segments)]
	layout.total = 0
	for index, segment := range segments {
		layout.inputs[index] = segmentGeometry{text: segment.Text, kind: segment.FontType, scale: segment.FontScale}
		layout.layouts[index] = o.segmentLayout(segment)
		if layout.layouts[index] != nil {
			layout.advances[index] = layout.layouts[index].Bounds().Width()
		} else {
			layout.advances[index] = o.getSegmentWidth(segment)
		}
		layout.total += layout.advances[index]
	}
	layout.typography, layout.spacing, layout.valid = key, o.lineSpacing, true
	layout.combineSegments()

	return layout
}

func (l *lineLayout) combineSegments() {
	l.combined = nil
	if len(l.layouts) == 0 {
		return
	}
	spans := make([]beradtext.Span, 0, len(l.layouts))
	l.paintNames = make([]string, len(l.layouts))
	for index, layout := range l.layouts {
		if layout == nil {
			return // Original vector command markers use the compatibility path.
		}
		span := layout.Content().Spans()[0]
		span.Paint = strconv.Itoa(index)
		l.paintNames[index] = span.Paint
		spans = append(spans, span)
	}
	content, err := beradtext.NewContent(spans)
	if err != nil {
		panic(fmt.Sprintf("composing styled game row: %v", err))
	}
	var options beradtext.LayoutOptions
	options.TopAlign = true
	layout, err := beradtext.NewLayout(content, options)
	if err != nil {
		panic(fmt.Sprintf("laying out styled game row: %v", err))
	}
	l.combined = layout
	l.total = layout.Bounds().Width()
	l.palette = make(beradtext.Palette, len(spans))
}

func (o *Output) segmentLayout(segment ColorSegment) *beradtext.Layout {
	switch segment.FontType {
	case FontTypeCommandPrompt:
		return nil // The original command marker remains game-owned vector artwork.
	case FontTypeRunic:
		return GetBitmapFonts().Runic.layout(segment.Text, o.getBitmapScale())
	case FontTypeIBM:
		return GetBitmapFonts().IBM.layout(segment.Text, o.getBitmapScale())
	case FontTypeNormal:
		scale := segment.FontScale
		if scale == 0 {
			scale = FontScaleNormal
		}

		return newTextLayout(segment.Text, o.Font.sdkForScale(scale))
	default:
		panic("unknown game font type")
	}
}
