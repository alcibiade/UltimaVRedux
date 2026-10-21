package widgets

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"unicode/utf8"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// A measured single line must not be rewrapped by an unrelated M-width budget.
func atlasSingleLine(output *text.Output, value string, colour color.Color) {
	output.Reset()
	output.SetMaxCharsPerLine(max(1, utf8.RuneCountInString(value)))
	output.AppendLine().Text(value, colour).Done()
}

func (d *AtlasDialog) drawHeading(screen *ebiten.Image) {
	d.prepareHeading()
	point := image.Pt(d.mapRect.Min.X, d.mapRect.Min.X)
	width := d.width - point.X - point.X
	d.header.DrawPanelContent(screen, point, width, etext.AlignStart)
	d.zoom.DrawPanelContent(screen, point, width, etext.AlignEnd)
}

func (d *AtlasDialog) prepareHeading() {
	const title = "MAP OF BRITANNIA"
	atlasSingleLine(d.header, title, DialogHeaderColor)
	d.zoom.Reset()
	line := d.zoom.AppendLine().Text("Zoom: ", DialogLabelColor).Textf("%dx", DialogValueColor, d.view.Zoom)
	if d.panning {
		line.Text("   PAN MODE", DialogLabelColor)
	}
	line.Done()
}

func (d *AtlasDialog) drawPlacesHeading(screen *ebiten.Image) {
	const legendGap = 4
	d.preparePlacesHeading()
	d.panelHeader.DrawPanelContent(screen, d.placesHeaderRect.Min, d.placesHeaderRect.Dx(), etext.AlignStart)
	cell := d.atlasSymbolCell()
	center := image.Pt(d.placesHeaderRect.Min.X+cell/2, d.placesHeaderRect.Min.Y+d.rowHeight+d.rowHeight/2)
	drawAtlasStamp(screen, atlasCoordinateStamp(), center, ClassicUIPixelUnit(d.width, d.height), DialogValueColor, false)
	atlasSingleLine(d.row, "Coordinates recorded", DialogControlHintColor)
	d.row.DrawPanelContent(screen, image.Pt(center.X+cell/2+legendGap, d.placesHeaderRect.Min.Y+d.rowHeight),
		d.placesHeaderRect.Dx()-cell-legendGap, etext.AlignStart)
}

func (d *AtlasDialog) preparePlacesHeading() {
	prefix := "PLACES  "
	prefixWidth, _ := d.font.MeasureString(prefix)
	value := fitAtlasText(d.font, fmt.Sprintf("%d total, %d mapped", len(d.entries), d.mappedCount()),
		d.placesHeaderRect.Dx()-int(math.Ceil(prefixWidth)))
	d.panelHeader.Reset()
	d.panelHeader.SetMaxCharsPerLine(max(1, utf8.RuneCountInString(prefix+value)))
	d.panelHeader.AppendLine().Text(prefix, DialogHeaderColor).Text(value, DialogValueColor).Done()
}

func (d *AtlasDialog) drawOverviewRow(screen *ebiten.Image, rect image.Rectangle, selected bool) {
	const separatorTextInset = 2

	colour, prefix := DialogValueColor, "  "
	if selected {
		colour, prefix = DialogSelectedText, "> "
	}
	fillAtlasRect(screen, image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+1), DialogRuleLightColor)
	atlasSingleLine(d.row, fmt.Sprintf("%sAll mapped (%d)", prefix, d.mappedCount()), colour)
	d.row.DrawPanelContent(atlasSubImage(screen, rect), rect.Min.Add(image.Pt(0, separatorTextInset)),
		rect.Dx(), etext.AlignStart)
}

// All categories are always visible. Labels are measured before placing cells;
// compact custom layouts can wrap, while supported landscape sizes use one row.
//
//nolint:mnd // Half a standard margin separates measured category cells.
func (d *AtlasDialog) layoutCategoryTabs(margin int) int {
	padding := margin / 2
	left, top := margin, margin+d.rowHeight
	for choice := journal.AtlasAllPlaces; choice < journal.AtlasPlaceFilterCount; choice++ {
		width, _ := d.font.MeasureString(choice.String())
		cellWidth := int(math.Ceil(width)) + margin
		if left > margin && left+cellWidth > d.width-margin {
			left = margin
			top += d.rowHeight + padding
		}
		d.tabs[choice] = image.Rect(left, top, left+cellWidth, top+d.rowHeight)
		left += cellWidth + padding
	}

	return top + d.rowHeight
}

func (d *AtlasDialog) drawCategoryTabs(screen *ebiten.Image) {
	for choice := journal.AtlasAllPlaces; choice < journal.AtlasPlaceFilterCount; choice++ {
		rect := d.tabs[choice]
		colour := DialogEmptyColor
		if choice == d.view.Filters.Places {
			colour = DialogLabelColor
		}
		atlasSingleLine(d.tabText, choice.String(), colour)
		d.tabText.DrawPanelContent(screen, rect.Min, rect.Dx(), etext.AlignCenter)
	}
}

//nolint:mnd // Small label/symbol gaps are measured separately from the row's font.
func (d *AtlasDialog) drawPlaceRow(screen *ebiten.Image, rect image.Rectangle, entry journal.AtlasEntry,
	selected bool) {
	target := atlasSubImage(screen, rect)
	mark := "  "
	colour, symbolColour := atlasPlaceColor(entry), atlasPlaceColor(entry)
	if selected {
		mark, colour = "> ", DialogSelectedText
		symbolColour = DialogSelectedLabelColor
		if entry.Discovered {
			symbolColour = DialogSelectedCheckedMarkColor
		}
	}
	markWidth, _ := d.font.MeasureString(mark)
	cell := d.atlasSymbolCell()
	left := rect.Min.X + int(math.Ceil(markWidth)) + 4
	d.drawAtlasSymbol(target, entry, image.Pt(left+cell/2, rect.Min.Y+rect.Dy()/2), symbolColour)
	textTop := rect.Min.Y + max(0, (rect.Dy()-int(math.Ceil(d.row.GetTotalHeightForLines(1))))/2)
	atlasSingleLine(d.row, mark, colour)
	d.row.DrawPanelContent(target, image.Pt(rect.Min.X, textTop), rect.Dx(), etext.AlignStart)
	left += cell + 8
	name := fitAtlasText(d.font, entry.Name, rect.Max.X-left-cell-8)
	atlasSingleLine(d.row, name, colour)
	d.row.DrawPanelContent(target, image.Pt(left, textTop), rect.Max.X-left, etext.AlignStart)
	if _, ok := entry.PlotPosition(); ok {
		drawAtlasStamp(target, atlasCoordinateStamp(), image.Pt(rect.Max.X-cell/2-4, rect.Min.Y+rect.Dy()/2),
			ClassicUIPixelUnit(d.width, d.height), symbolColour, false)
	}
}

func (d *AtlasDialog) drawPositionStatus(screen *ebiten.Image) {
	d.buildPositionStatus()
	d.position.DrawPanelContent(screen, d.positionRect.Min, d.positionRect.Dx(), etext.AlignStart)
	d.row.DrawPanelContent(screen, d.statusRect.Min, d.statusRect.Dx(), etext.AlignStart)
}

// Reserve the same measured worst-case block across fix/policy changes.
// Only list/details yield space; geography and the two help rows stay fixed.
//
//nolint:mnd // Two content rows plus the header/context rows and half-margin gap.
func (d *AtlasDialog) layoutPositionStatus(right, margin int) {
	chars := max(1, d.font.CharsForWidth(d.width-margin-right))
	d.position.SetMaxCharsPerLine(chars)
	d.row.SetMaxCharsPerLine(chars)
	positionLines, statusLines := 3, 4
	for _, context := range []string{"Within this settlement", "Within Empath Abbey", "Within Buccaneer's Den",
		"Within Lord British's Castle", "Within Blackthorn's Castle"} {
		d.position.Reset()
		d.position.AppendLine().Text(context, DialogValueColor).Done()
		positionLines = max(positionLines, 2+d.position.GetPanelContentLineCount())
	}
	for _, reason := range []string{"Night only: 19:00-05:59.", "Use on the world map.",
		"Unavailable underground.", "No sextant carried.", "Measure this place to locate it from the list.",
		"Selected place label shown.", "Selected place label hidden.", "Zoom in to pan."} {
		d.row.Reset()
		d.row.AppendLine().Text(reason, DialogControlHintColor).Done()
		statusLines = max(statusLines, d.row.GetPanelContentLineCount())
		positionLines = max(positionLines, 1+d.row.GetPanelContentLineCount())
	}
	statusHeight := int(math.Ceil(d.row.GetTotalHeightForLines(statusLines)))
	positionHeight := int(math.Ceil(d.position.GetTotalHeightForLines(positionLines + 2)))
	d.statusRect = image.Rect(right, d.mapRect.Max.Y-statusHeight, d.width-margin, d.mapRect.Max.Y)
	d.positionRect = image.Rect(right, d.statusRect.Min.Y-positionHeight-margin/2,
		d.width-margin, d.statusRect.Min.Y-margin/2)
}

// Both buffers wrap complete values using measured normal body typography.
func (d *AtlasDialog) buildPositionStatus() {
	d.position.Reset()
	d.position.SetMaxCharsPerLine(max(1, d.font.CharsForWidth(d.positionRect.Dx())))
	if d.dossier.CurrentFix.Measured {
		d.position.AppendLine().Text("YOUR POSITION", DialogHeaderColor).Done()
		d.position.AppendLine().Text(atlasCoordinates(d.dossier.CurrentFix.Location), DialogValueColor).Done()
		if d.dossier.FixContext != "" {
			d.position.AppendLine().Text(d.dossier.FixContext, DialogValueColor).Done()
		}
	} else {
		d.position.AppendLine().Text("POSITION UNAVAILABLE", DialogHeaderColor).Done()
		d.position.AppendLine().Text(d.dossier.FixUnavailable, DialogValueColor).Done()
	}
	if d.onNoteIntent != nil {
		d.position.AppendLine().Text("N", DialogControlKeyColor).Text(" Mark current position", DialogControlHintColor).Done()
		if d.selectedNoteID() != 0 {
			d.position.AppendLine().Text("E", DialogControlKeyColor).Text(" Edit   ", DialogControlHintColor).
				Text("D", DialogControlKeyColor).Text(" Delete", DialogControlHintColor).Done()
		}
	}
	d.row.Reset()
	d.row.SetMaxCharsPerLine(max(1, d.font.CharsForWidth(d.statusRect.Dx())))
	d.row.AppendLine().Text(d.status, DialogControlHintColor).Done()
}

// Help stays in two physical rows: describe only the active pane's actions.
// Alternate close shortcuts and conditional detail paging live outside this footer.
func (d *AtlasDialog) helpRows() [2][]ControlHint {
	category, selection := ControlHint{hintKeyTabCycle, "Category"}, ControlHint{hintKeyUpDown, hintActionSelect}
	zoom, world := ControlHint{"+/-", "Zoom"}, ControlHint{"0", "World"}
	first := []ControlHint{category, selection, {hintKeyEnter, "Show"}, zoom, world}
	if d.view.Overview {
		first = []ControlHint{category, selection, zoom, world}
	}
	pan := "Pan"
	if d.panning {
		first = []ControlHint{category, {"Arrows", "Pan"}, zoom, world}
		pan = "Places"
	}
	second := []ControlHint{{"P", pan}, {"L", "Label"}, {"G", "Grid"}}
	if d.view.Overview {
		second[1].Action = "Labels"
	}
	if d.dossier.CurrentFix.Measured {
		second = append(second, ControlHint{"A", "Avatar"})
	}
	second = append(second, ControlHint{hintKeyEsc, hintActionClose})

	return [2][]ControlHint{first, second}
}

func (d *AtlasDialog) buildFooter() {
	d.footer.Reset()
	for _, hints := range d.helpRows() {
		PackControlHints(d.footer, hints, d.footerRect.Dx(), etext.AlignStart)
	}
}
