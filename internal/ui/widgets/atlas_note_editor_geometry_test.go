package widgets

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"
)

func TestAtlasNoteEditorScaledFramesAndGeometryAtPresets(t *testing.T) {
	for _, size := range config.SupportedWindowResolutions() {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			setQuestJournalTestResolution(t, size.X, size.Y)
			editor := NewAtlasNoteEditor(AtlasNoteEditorConfig{Keyboard: newWidgetTestKeyboard(0),
				Submit: func(string, string) error { return nil }, Cancel: func() {}})
			screen := ebiten.NewImageWithOptions(editor.pane, nil)
			defer screen.Deallocate()
			unit := ClassicUIPixelUnit(size.X, size.Y)
			for focus := range editor.fields {
				editor.focus = focus
				screen.Fill(ucolor.DialogBackground.RGBA())
				editor.Draw(screen)
				for field, rect := range editor.fieldRects {
					frameColour := DialogRuleLightColor
					if field == focus {
						frameColour = DialogLabelColor
					}
					assertAtlasNoteEditorFramePixels(t, screen, rect, unit, frameColour)
					layout := editor.fieldLayout(field)
					interior := rect.Inset(unit)
					require.Equal(t, unit, layout.pixelUnit)
					require.True(t, layout.content.In(interior))
					require.Equal(t, 2*unit, layout.content.Min.X-interior.Min.X)
					require.Equal(t, 2*unit, interior.Max.X-layout.content.Max.X)
					require.Equal(t, unit, layout.content.Min.Y-interior.Min.Y)
					require.Equal(t, unit, interior.Max.Y-layout.content.Max.Y)
					require.Equal(t, layout.content.Dx(), layout.textWidth+unit, "end caret has its own scaled width")
					require.Positive(t, layout.capacity)
					if field == 0 {
						require.Equal(t, 1, layout.capacity, "Map Label remains a single visible row")
					} else {
						require.LessOrEqual(t, layout.lineOrigin(layout.capacity-1).Y+layout.textHeight, layout.content.Max.Y)
						require.Greater(t, layout.lineOrigin(layout.capacity).Y+layout.textHeight, layout.content.Max.Y,
							"capacity includes exactly the complete measured rows")
					}
					assertAtlasNoteEditorContentPixels(t, editor, field, screen)
				}
			}
		})
	}
}

func TestAtlasNoteEditorCenteredLabelBaselineAndTopAlignedDescriptionAtPresets(t *testing.T) {
	for _, size := range config.SupportedWindowResolutions() {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			setQuestJournalTestResolution(t, size.X, size.Y)
			editor := NewAtlasNoteEditor(AtlasNoteEditorConfig{
				Record:   journal.AtlasNoteRecord{Label: "Agjpqy_|[]!?", Description: "Agjpqy_|[]!?\nSecond row"},
				Keyboard: newWidgetTestKeyboard(0), Submit: func(string, string) error { return nil }, Cancel: func() {},
			})
			var screen *ebiten.Image
			if size.X == 1280 || size.X == 1366 || size.X == 3840 {
				screen = ebiten.NewImageWithOptions(editor.pane, nil)
				t.Cleanup(screen.Dispose)
			}
			for focus := range editor.fields {
				editor.focus = focus
				editor.cursors[1] = len([]rune("Agjpqy_|[]!?"))
				editor.ensureLayout()
				for field := range editor.fields {
					editor.prepareFieldScroll(field, editor.fieldLayout(field))
				}
				if screen != nil {
					editor.Draw(screen)
				}
				unit := ClassicUIPixelUnit(size.X, size.Y)
				fontHeight := int(math.Ceil(editor.output.GetTotalHeightForLines(1)))
				labelInterior := editor.fieldRects[0].Inset(unit)
				expectedLabelY := labelInterior.Min.Y + (labelInterior.Dy()-fontHeight)/2
				labelLayout := editor.fieldLayout(0)
				require.Equal(t, expectedLabelY, labelLayout.origin.Y)
				ascent := editor.font.TextFace.Metrics().HAscent
				require.Equal(t, float64(expectedLabelY)+ascent, float64(labelLayout.origin.Y)+ascent,
					"baseline uses the centered measured font cell")
				topGap := labelLayout.origin.Y - labelInterior.Min.Y
				bottomGap := labelInterior.Max.Y - labelLayout.origin.Y - fontHeight
				require.InDelta(t, topGap, bottomGap, 1, "text and caret have balanced top/bottom space")
				descriptionLayout := editor.fieldLayout(1)
				require.Equal(t, editor.fieldRects[1].Min.Y+2*unit, descriptionLayout.origin.Y)
				require.Less(t, descriptionLayout.origin.Y,
					descriptionLayout.content.Min.Y+(descriptionLayout.content.Dy()-fontHeight)/2,
					"Description starts at its scaled top inset, not the vertical center")
				for field := range editor.fields {
					assertAtlasNoteEditorContentPixels(t, editor, field, screen)
				}
			}
		})
	}
}

func TestAtlasNoteEditorScaledWrappingCaretAndScrollStayUnclippedAtPresets(t *testing.T) {
	var printable strings.Builder
	for char := ' '; char <= '~'; char++ {
		printable.WriteRune(char)
	}
	descriptions := []string{
		strings.Repeat("M", journal.MaxAtlasNoteDescriptionRunes),
		strings.Repeat(printable.String(), 6)[:journal.MaxAtlasNoteDescriptionRunes],
		strings.Repeat("\n", journal.MaxAtlasNoteDescriptionRunes-1) + "Z",
	}
	for _, size := range config.SupportedWindowResolutions() {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			setQuestJournalTestResolution(t, size.X, size.Y)
			editor := NewAtlasNoteEditor(AtlasNoteEditorConfig{
				Record:   journal.AtlasNoteRecord{Label: strings.Repeat("M", journal.MaxAtlasNoteLabelRunes)},
				Keyboard: newWidgetTestKeyboard(0), Submit: func(string, string) error { return nil }, Cancel: func() {},
			})
			editor.focus = 1
			// Reserve the real longest operation message while exercising the fields.
			editor.message = fmt.Errorf("%w: map notes full (%d); edit or delete a note",
				journal.ErrInvalidAtlasPersonal, journal.MaxAtlasNotes).Error()
			editor.ensureLayout()
			var screen *ebiten.Image
			if size.X == 1280 || size.X == 1366 {
				screen = ebiten.NewImageWithOptions(editor.pane, nil)
				t.Cleanup(screen.Dispose)
			}
			for _, description := range descriptions {
				editor.fields[1] = []rune(description)
				lines := editor.lines(1)
				layout := editor.fieldLayout(1)
				require.Greater(t, len(lines), 1)
				for _, line := range lines {
					advance, _ := editor.font.MeasureString(string(editor.fields[1][line.start:line.end]))
					require.LessOrEqual(t, int(math.Ceil(advance))+layout.pixelUnit, layout.content.Dx(),
						"wrapping leaves enough room for the entire scaled end caret")
				}
				for _, position := range []struct {
					cursor   int
					upstream bool
				}{{0, false}, {lines[0].end, true}, {lines[0].end, false}, {len(editor.fields[1]), false}} {
					editor.cursors[1], editor.upstream[1] = position.cursor, position.upstream
					editor.prepareFieldScroll(1, layout)
					if screen != nil {
						editor.Draw(screen)
					}
					row := editor.cursorLine(1, lines)
					require.GreaterOrEqual(t, row, editor.scroll[1])
					require.Less(t, row, editor.scroll[1]+layout.capacity)
					require.Len(t, editor.lines(0), 1, "maximum label still fits one row")
					if position.cursor == len(editor.fields[1]) && len(lines) > layout.capacity {
						require.Positive(t, editor.scroll[1])
					}
					for field := range editor.fields {
						assertAtlasNoteEditorContentPixels(t, editor, field, screen)
					}
					require.Equal(t, 2, editor.output.GetPanelContentLineCount(), "measured footer keeps its two rows")
					require.Less(t, editor.fieldRects[1].Max.Y, editor.validationRect.Min.Y)
				}
			}
		})
	}
}

func assertAtlasNoteEditorFramePixels(t *testing.T, screen *ebiten.Image, rect image.Rectangle, unit int, frameColour color.Color) {
	t.Helper()
	pixels := atlasMarkerPixels(atlasSubImage(screen, rect))
	background := ucolor.DialogBackground.RGBA()
	ink := color.RGBAModel.Convert(frameColour)
	center := rect.Min.Add(rect.Size().Div(2))
	for offset := range unit + 1 {
		expected := ink
		if offset == unit {
			expected = background
		}
		for _, point := range []image.Point{
			{center.X, rect.Min.Y + offset}, {center.X, rect.Max.Y - 1 - offset},
			{rect.Min.X + offset, center.Y}, {rect.Max.X - 1 - offset, center.Y},
		} {
			require.Equal(t, expected, pixels.At(point.X, point.Y), "exact frame thickness at %v", point)
		}
	}
	// Independent three-cell corner stencil checks the actual stepped pixels,
	// including the empty corner and the absence of a thinner one-pixel outline.
	for y := range 3 * unit {
		for x := range 3 * unit {
			expected := color.Color(background)
			if x/unit+y/unit == 2 {
				expected = ink
			}
			for _, point := range []image.Point{
				{rect.Min.X + x, rect.Min.Y + y}, {rect.Max.X - 1 - x, rect.Min.Y + y},
				{rect.Min.X + x, rect.Max.Y - 1 - y}, {rect.Max.X - 1 - x, rect.Max.Y - 1 - y},
			} {
				require.Equal(t, expected, pixels.At(point.X, point.Y), "scaled stepped corner at %v", point)
			}
		}
	}
}

func assertAtlasNoteEditorContentPixels(t *testing.T, editor *AtlasNoteEditor, field int, screen *ebiten.Image) {
	t.Helper()
	layout := editor.fieldLayout(field)
	lines := editor.lines(field)
	row := editor.cursorLine(field, lines)
	padding := layout.pixelUnit
	if field == editor.focus {
		advance, _ := editor.font.MeasureString(string(editor.fields[field][lines[row].start:editor.cursors[field]]))
		require.True(t, layout.caretRect(row-editor.scroll[field], advance).In(layout.content), "complete caret fits measured content")
	}
	if screen == nil {
		return
	}
	// Draw without the field clip, with space on every side. Any escaping ink or
	// truncated bottom row is observable before comparing to the production pixels.
	unclipped := ebiten.NewImage(layout.content.Dx()+2*padding, layout.content.Dy()+2*padding)
	defer unclipped.Deallocate()
	reserved := image.Rectangle{Min: image.Pt(padding, padding), Max: layout.content.Size().Add(image.Pt(padding, padding))}
	shift := reserved.Min.Sub(layout.content.Min)
	output := text.NewOutput(editor.font, editor.output.GetLineSpacing(), 1, journal.MaxAtlasNoteDescriptionRunes)
	for index := editor.scroll[field]; index < min(len(lines), editor.scroll[field]+layout.capacity); index++ {
		line := lines[index]
		value := string(editor.fields[field][line.start:line.end])
		point := layout.content.Min.Add(image.Pt(0, (index-editor.scroll[field])*editor.lineHeight))
		if field == 0 {
			point.Y = editor.fieldRects[field].Min.Y +
				(editor.fieldRects[field].Dy()-int(math.Ceil(output.GetTotalHeightForLines(1))))/2
		}
		output.Reset()
		output.AppendLine().Text(value, DialogValueColor).Done()
		output.DrawPanelContent(unclipped, point.Add(shift), layout.textWidth, etext.AlignStart)
		if field == editor.focus && index == row {
			advance, _ := editor.font.MeasureString(string(editor.fields[field][line.start:editor.cursors[field]]))
			caret := image.Rect(point.X+int(math.Ceil(advance)), point.Y,
				point.X+int(math.Ceil(advance))+padding, point.Y+int(math.Ceil(output.GetTotalHeightForLines(1))))
			require.True(t, caret.In(layout.content), "complete caret %v fits content %v", caret, layout.content)
			fillAtlasRect(unclipped, caret.Add(shift), DialogLabelColor)
			actual := atlasMarkerPixels(atlasSubImage(screen, caret))
			for y := caret.Min.Y; y < caret.Max.Y; y++ {
				for x := caret.Min.X; x < caret.Max.X; x++ {
					require.Equal(t, color.RGBAModel.Convert(DialogLabelColor), actual.At(x, y), "complete scaled caret rasterizes")
				}
			}
		}
	}
	pixels := atlasMarkerPixels(unclipped)
	var ink image.Rectangle
	for y := range pixels.Bounds().Dy() {
		for x := range pixels.Bounds().Dx() {
			if pixels.RGBAAt(x, y).A != 0 {
				ink = ink.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if !ink.Empty() {
		require.True(t, ink.In(reserved), "unclipped text/caret %v exceed content %v", ink, reserved)
	}
	expected := ebiten.NewImage(layout.content.Dx(), layout.content.Dy())
	defer expected.Deallocate()
	expected.Fill(ucolor.DialogBackground.RGBA())
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Translate(float64(-padding), float64(-padding))
	expected.DrawImage(unclipped, options)
	require.Equal(t, atlasMarkerPixels(expected).Pix, atlasMarkerPixels(atlasSubImage(screen, layout.content)).Pix,
		"all measured text and the complete caret render at their shared origin")
}
