package mainscreen

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func provisionTestSummary(size image.Point) *ProvisionSummary {
	summary := NewProvisionSummaryForWidth(size.X)
	bottom := int(float64(size.Y) * 0.975)
	summary.SetBounds(image.Rect(int(float64(size.X)*0.77), bottom-ProvisionSummaryHeight(size.X),
		int(float64(size.X)*0.99), bottom))

	return summary
}

func maximumProvisionTestState() *game_state.GameState {
	state := &game_state.GameState{DateTime: datetime.UltimaDate{Year: 65535, Month: 13, Day: 28, Hour: 12, Minute: 59}}
	state.PartyState.Inventory.Provisions.Food.Set(9999)
	state.PartyState.Inventory.Provisions.Torches.Set(99)
	state.PartyState.Inventory.Provisions.Keys.Set(99)
	state.PartyState.Inventory.Provisions.Gems.Set(99)
	state.PartyState.Inventory.Gold.Set(9999)
	state.PartyState.Karma.Value = 99
	state.PartyState.Inventory.Equipment.Set(references.Arrows, 20)
	state.PartyState.Inventory.Equipment.Set(references.Quarrels, 0)

	return state
}

func TestProvisionResourcesUseSelectedFourRowOrderAndSemanticColors(t *testing.T) {
	state := maximumProvisionTestState()
	resources := provisionResources(state)
	labels := [4][2]string{{"F", "T"}, {"K", "G"}, {"$", "Ka"}, {"Ar", "Qr"}}
	values := [4][2]string{{"9999", "99"}, {"99", "99"}, {"9999", "99"}, {"20", "0"}}
	colors := [4][2]color.Color{
		{ucolor.EGALightGreen, ucolor.EGALightRed}, {ucolor.EGAYellow, ucolor.EGALightCyan},
		{ucolor.EGAYellow, ucolor.EGALightMagenta}, {ucolor.EGALightCyan, ucolor.EGALightCyan},
	}
	for row := range resources {
		for column, resource := range resources[row] {
			require.Equal(t, labels[row][column], resource.label)
			require.Equal(t, values[row][column], resource.value)
			require.Equal(t, colors[row][column], resource.labelColor)
			if row < 3 {
				require.Empty(t, resource.status)
				require.Equal(t, ucolor.UITextPrimary.RGBA(), resource.valueColor)
			}
		}
	}
	require.Equal(t, "LOW", resources[3][0].status)
	require.Equal(t, ucolor.SupplyLow, resources[3][0].valueColor)
	require.Equal(t, "OUT", resources[3][1].status)
	require.Equal(t, ucolor.SupplyEmpty, resources[3][1].valueColor)
	require.Equal(t, "13-28-65535", state.DateTime.GetDateAsString())
}

func TestProvisionFourRowGeometryFitsMaximumValuesAtEveryResolution(t *testing.T) {
	for _, size := range []image.Point{{1280, 720}, {1365, 768}, {1920, 1080}, {2560, 1440}, {3840, 2160}} {
		t.Run(size.String(), func(t *testing.T) {
			summary := provisionTestSummary(size)
			spaceWidth, _ := summary.ultimaFont.MeasureString(" ")
			require.Positive(t, spaceWidth, "the font must provide a measurable space advance")
			minimumSeparator := int(math.Ceil(spaceWidth / 2))
			require.GreaterOrEqual(t, summary.gap, minimumSeparator)
			rows := summary.rows()
			require.Len(t, rows, 4)
			require.Equal(t, summary.bounds.Min.Y, rows[0].Min.Y)
			require.Equal(t, summary.bounds.Max.Y, rows[3].Max.Y)
			firstCells, firstCalendar := summary.rowCells(rows[0])
			for index, row := range rows {
				require.True(t, row.In(summary.bounds), "supplies must never shift left into the rasterized frame")
				require.Equal(t, summary.bounds.Min.X, row.Min.X, "retain the scene's safe left content edge")
				require.Equal(t, summary.rowHeight, row.Dy())
				require.GreaterOrEqual(t, row.Dy(), int(math.Ceil(summary.output.GetFontHeight())))
				if index > 0 {
					require.Equal(t, rows[index-1].Max.Y, row.Min.Y, "four disjoint, evenly spaced rows")
				}
				cells, calendar := summary.rowCells(row)
				require.True(t, calendar.In(row))
				require.Equal(t, firstCalendar.Min.X, calendar.Min.X, "calendar stays in its own right column")
				require.GreaterOrEqual(t, calendar.Min.X-cells[1].value.Max.X, minimumSeparator,
					"date/time need a visible half-space gutter after the resource quantities")
				require.GreaterOrEqual(t, cells[1].label.Min.X-cells[0].status.Max.X, minimumSeparator,
					"Ar LOW/OUT needs a visible half-space gutter before the Qr label")
				if index == 3 {
					require.True(t, cells[1].status.In(calendar), "Qr status uses the otherwise empty calendar column")
				}
				for column, cell := range cells {
					require.True(t, cell.label.In(row) && cell.value.In(row) && cell.status.In(row))
					require.Equal(t, firstCells[column].label.Min.X, cell.label.Min.X)
					require.Equal(t, firstCells[column].value.Min.X, cell.value.Min.X)
					require.Equal(t, firstCells[column].value.Max.X, cell.value.Max.X, "all quantities share a right edge")
					require.Equal(t, firstCells[column].status.Min.X, cell.status.Min.X)
					require.GreaterOrEqual(t, cell.value.Min.X-cell.label.Max.X, summary.labelGap)
					require.GreaterOrEqual(t, cell.status.Min.X-cell.value.Max.X, minimumSeparator,
						"LOW/OUT needs a visible half-space gutter after the right-aligned count")
					require.False(t, cell.label.Overlaps(calendar), "calendar may not obscure a resource label")
					require.False(t, cell.value.Overlaps(calendar), "calendar may not obscure a quantity")
					for _, label := range []string{"F", "T", "K", "G", "$", "Ka", "Ar", "Qr"} {
						require.LessOrEqual(t, measuredProvisionWidth(summary.ultimaFont, label), cell.label.Dx())
					}
					for _, value := range []string{"0", "1", "20", "99", "100", "218", "9999"} {
						font, output := summary.valueTextLayout(value, cell.value.Dx())
						require.LessOrEqual(t, measuredProvisionWidth(font, value), cell.value.Dx(), "%s must be complete", value)
						require.LessOrEqual(t, int(math.Ceil(output.GetFontHeight())), row.Dy())
					}
				}
				for _, value := range []string{"13-28-65535", "13-28-139", "12:59PM", "12:00AM"} {
					font, _ := summary.valueTextLayout(value, calendar.Dx())
					require.LessOrEqual(t, measuredProvisionWidth(font, value), calendar.Dx(), "%s must be complete", value)
				}
			}
			summary.gap *= 2
			require.Equal(t, rows, summary.rows(), "text spacing must not enlarge frame clearance or steal more content width")
		})
	}
}

// Compare each field with isolated glyphs: a complete-panel translated reference
// alone would also reproduce an overlap between adjacent labels, counts or dates.
func TestProvisionRenderedFieldsRetainCompleteIsolatedGlyphs(t *testing.T) {
	// Native glyph checks cover the tightest, fractional, and largest scales;
	// the geometry test above covers every supported resolution on the CPU.
	for _, size := range []image.Point{{1280, 720}, {1365, 768}, {3840, 2160}} {
		t.Run(size.String(), func(t *testing.T) {
			summary := provisionTestSummary(size)
			// Keep the real panel dimensions and a two-pixel overflow margin,
			// without allocating or reading back a full-screen target.
			bounds := summary.bounds
			summary.SetBounds(image.Rect(2, 2, bounds.Dx()+2, bounds.Dy()+2))
			actual := ebiten.NewImage(bounds.Dx()+4, bounds.Dy()+4)
			reference := ebiten.NewImage(bounds.Dx()+4, bounds.Dy()+4)
			t.Cleanup(actual.Dispose)
			t.Cleanup(reference.Dispose)
			state := maximumProvisionTestState()
			summary.Draw(state, actual)
			resources := provisionResources(state)
			for rowIndex, row := range summary.rows() {
				cells, calendar := summary.rowCells(row)
				assertProvisionGutterPixels(t, actual, image.Rect(cells[0].status.Max.X, row.Min.Y, cells[1].label.Min.X, row.Max.Y))
				assertProvisionGutterPixels(t, actual, image.Rect(cells[1].value.Max.X, row.Min.Y, calendar.Min.X, row.Max.Y))
				for column, resource := range resources[rowIndex] {
					cell := cells[column]
					assertProvisionGutterPixels(t, actual, image.Rect(cell.value.Max.X, row.Min.Y, cell.status.Min.X, row.Max.Y))
					assertProvisionFieldPixels(t, actual, reference, summary.output, cell.label, resource.label, resource.labelColor, etext.AlignStart)
					_, output := summary.valueTextLayout(resource.value, cell.value.Dx())
					assertProvisionFieldPixels(t, actual, reference, output, cell.value, resource.value, resource.valueColor, etext.AlignEnd)
					if resource.status != "" {
						assertProvisionFieldPixels(t, actual, reference, summary.output, cell.status, resource.status, resource.valueColor, etext.AlignStart)
					}
				}
				if rowIndex < 2 {
					value := [2]string{state.DateTime.GetDateAsString(), state.DateTime.GetTimeAsString()}[rowIndex]
					_, output := summary.valueTextLayout(value, calendar.Dx())
					assertProvisionFieldPixels(t, actual, reference, output, calendar, value, ucolor.UITextPrimary.RGBA(), etext.AlignEnd)
				}
			}
			cells, _ := summary.rowCells(summary.rows()[3])
			for _, count := range []uint16{0, 20, 21} {
				state.PartyState.Inventory.Equipment.Set(references.Arrows, count)
				state.PartyState.Inventory.Equipment.Set(references.Quarrels, count)
				actual.Clear()
				summary.Draw(state, actual)
				row := summary.rows()[3]
				assertProvisionGutterPixels(t, actual, image.Rect(cells[0].status.Max.X, row.Min.Y, cells[1].label.Min.X, row.Max.Y))
				for column, resource := range provisionResources(state)[3] {
					cell := cells[column]
					assertProvisionGutterPixels(t, actual, image.Rect(cell.value.Max.X, row.Min.Y, cell.status.Min.X, row.Max.Y))
					assertProvisionFieldPixels(t, actual, reference, summary.output, cell.label, resource.label, resource.labelColor, etext.AlignStart)
					assertProvisionFieldPixels(t, actual, reference, summary.output, cell.value, resource.value, resource.valueColor, etext.AlignEnd)
					if resource.status == "" {
						pixels := make([]byte, cell.status.Dx()*cell.status.Dy()*4)
						actual.SubImage(cell.status).(*ebiten.Image).ReadPixels(pixels)
						require.Equal(t, make([]byte, len(pixels)), pixels, "healthy ammunition keeps its status slot empty")
					} else {
						assertProvisionFieldPixels(t, actual, reference, summary.output, cell.status, resource.status, resource.valueColor, etext.AlignStart)
					}
				}
			}
		})
	}
}

func assertProvisionGutterPixels(t *testing.T, screen *ebiten.Image, bounds image.Rectangle) {
	t.Helper()
	require.False(t, bounds.Empty(), "the visual separator must have positive width")
	pixels := make([]byte, bounds.Dx()*bounds.Dy()*4)
	screen.SubImage(bounds).(*ebiten.Image).ReadPixels(pixels)
	require.Equal(t, make([]byte, len(pixels)), pixels, "the complete separator %v must remain clear", bounds)
}

func assertProvisionFieldPixels(
	t *testing.T, actual, reference *ebiten.Image, output *text.Output, bounds image.Rectangle,
	value string, valueColor color.Color, alignment etext.Align,
) {
	t.Helper()
	reference.Clear()
	drawProvisionText(reference, output, bounds, value, valueColor, alignment)
	require.Equal(t, 1, output.GetPanelContentLineCount(), "%q must never wrap", value)
	require.Contains(t, output.GetOutputStr(false), value)
	_, hasColor := output.FirstPanelLineWithColor(0, 1, valueColor)
	require.True(t, hasColor)
	// Include adjacent space so an overflowing glyph cannot evade the sample.
	sample := bounds.Inset(-2)
	actualPixels, referencePixels := make([]byte, sample.Dx()*sample.Dy()*4), make([]byte, sample.Dx()*sample.Dy()*4)
	actual.SubImage(sample).(*ebiten.Image).ReadPixels(actualPixels)
	reference.SubImage(sample).(*ebiten.Image).ReadPixels(referencePixels)
	inkPixels := 0
	for index := 0; index < len(referencePixels); index += 4 {
		if referencePixels[index+3] == 0 {
			continue
		}
		inkPixels++
		point := image.Pt(sample.Min.X+(index/4)%sample.Dx(), sample.Min.Y+(index/4)/sample.Dx())
		require.True(t, point.In(bounds), "%q pixel %v must stay inside its own field", value, point)
		require.Equal(t, referencePixels[index:index+4], actualPixels[index:index+4], "%q pixel %v", value, point)
	}
	require.Positive(t, inkPixels, "%q must have visible glyphs", value)
}
