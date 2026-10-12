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
	labels := [2][3]string{{"F", "K", "G"}, {"$", "T", "Ka"}}
	values := [2][3]string{{"9999", "99", "99"}, {"9999", "99", "99"}}
	colors := [2][3]color.Color{
		{ucolor.EGALightGreen, ucolor.EGAYellow, ucolor.EGALightCyan},
		{ucolor.EGAYellow, ucolor.EGALightRed, ucolor.EGALightMagenta},
	}
	for row := range resources {
		for column, resource := range resources[row] {
			require.Equal(t, labels[row][column], resource.label)
			require.Equal(t, values[row][column], resource.value)
			require.Equal(t, colors[row][column], resource.labelColor)
			require.Empty(t, resource.status)
			require.Equal(t, ucolor.UITextPrimary.RGBA(), resource.valueColor)
		}
	}
	ammunition := provisionAmmunitionResources(state)
	require.Equal(t, "Ar", ammunition[0].label)
	require.Equal(t, "Qr", ammunition[1].label)
	require.Equal(t, "20", ammunition[0].value)
	require.Equal(t, "0", ammunition[1].value)
	require.Equal(t, "LOW", ammunition[0].status)
	require.Equal(t, ucolor.SupplyLow, ammunition[0].statusColor)
	require.Equal(t, "OUT", ammunition[1].status)
	require.Equal(t, ucolor.SupplyEmpty, ammunition[1].statusColor)
	for _, resource := range ammunition {
		require.Equal(t, ucolor.EGALightCyan, resource.labelColor)
		require.Equal(t, ucolor.UITextPrimary.RGBA(), resource.valueColor, "quantities stay white even for LOW/OUT")
	}
	require.Equal(t, "13-28-65535", state.DateTime.GetDateAsString())
}

func TestProvisionFourRowGeometryFitsMaximumValuesAtEveryResolution(t *testing.T) {
	for _, size := range []image.Point{{1280, 720}, {1365, 768}, {1920, 1080}, {2560, 1440}, {3840, 2160}} {
		t.Run(size.String(), func(t *testing.T) {
			summary := provisionTestSummary(size)
			spaceWidth, _ := summary.ultimaFont.MeasureString(" ")
			require.Positive(t, spaceWidth, "the font must provide a measurable space advance")
			minimumSeparator := int(math.Ceil(spaceWidth / 2))
			rows := summary.rows()
			require.Len(t, rows, 4)
			require.Equal(t, summary.bounds.Min.Y, rows[0].Min.Y)
			require.Equal(t, summary.bounds.Max.Y, rows[3].Max.Y)
			for index, row := range rows {
				require.True(t, row.In(summary.bounds), "supplies must never shift left into the rasterized frame")
				require.Equal(t, summary.bounds.Min.X, row.Min.X, "retain the scene's safe left content edge")
				require.Equal(t, summary.rowHeight, row.Dy())
				require.GreaterOrEqual(t, row.Dy(), int(math.Ceil(summary.output.GetFontHeight())))
				if index > 0 {
					require.Equal(t, rows[index-1].Max.Y, row.Min.Y, "four disjoint, evenly spaced rows")
				}
			}
			for rowIndex, resources := range provisionResources(maximumProvisionTestState()) {
				cells := summary.resourceCells(rows[rowIndex])
				for column, cell := range cells {
					assertProvisionCellFits(t, summary, rows[rowIndex], cell, resources[column])
					require.True(t, cell.status.Empty())
					if column > 0 {
						require.GreaterOrEqual(t, cell.label.Min.X-cells[column-1].value.Max.X, minimumSeparator)
					}
					third := image.Rect(rows[rowIndex].Min.X+column*rows[rowIndex].Dx()/3, rows[rowIndex].Min.Y,
						rows[rowIndex].Min.X+(column+1)*rows[rowIndex].Dx()/3, rows[rowIndex].Max.Y)
					require.InDelta(t, cell.label.Min.X-third.Min.X, third.Max.X-cell.value.Max.X, 1,
						"the fixed label/quantity block is centered in its third")
					first := summary.resourceCells(rows[0])[column]
					require.Equal(t, first.label.Max.X, cell.label.Max.X, "labels share a right edge across rows")
					require.Equal(t, first.value.Min.X, cell.value.Min.X, "quantities share a left edge across rows")
				}
			}
			ammo := summary.ammunitionCells(rows[2])
			left, right := summary.calendarCells(rows[2])
			halves := [2]image.Rectangle{left, right}
			for column, resource := range provisionAmmunitionResources(maximumProvisionTestState()) {
				assertProvisionCellFits(t, summary, rows[2], ammo[column], resource)
				require.InDelta(t, ammo[column].label.Min.X-halves[column].Min.X,
					halves[column].Max.X-ammo[column].status.Max.X, 1, "ammunition blocks are centered in calendar-aligned halves")
				require.True(t, ammo[column].status.In(rows[2]))
				require.Equal(t, summary.statusGap, ammo[column].status.Min.X-ammo[column].value.Max.X)
				require.GreaterOrEqual(t, summary.statusGap, int(math.Ceil(spaceWidth)), "warnings have a full-space gutter")
				for _, value := range []string{"LOW", "OUT"} {
					require.LessOrEqual(t, measuredProvisionWidth(summary.ultimaFont, value), ammo[column].status.Dx())
				}
			}
			require.GreaterOrEqual(t, ammo[1].label.Min.X-ammo[0].status.Max.X, summary.labelGap)
			date, clock := summary.calendarCells(rows[3])
			require.True(t, date.In(rows[3]) && clock.In(rows[3]))
			require.Equal(t, rows[3].Min.X, date.Min.X)
			require.Equal(t, rows[3].Max.X, clock.Max.X)
			require.Equal(t, date.Max.X, clock.Min.X)
			require.InDelta(t, date.Dx(), clock.Dx(), 1, "date and time occupy equal halves")
			for _, value := range []string{"13-28-65535", "13-28-139"} {
				require.LessOrEqual(t, measuredProvisionWidth(summary.ultimaFont, value), date.Dx())
			}
			for _, value := range []string{"12:59PM", "12:00AM"} {
				require.LessOrEqual(t, measuredProvisionWidth(summary.ultimaFont, value), clock.Dx())
			}
			summary.labelGap *= 2
			require.Equal(t, rows, summary.rows(), "text spacing must not enlarge frame clearance or steal more content width")
		})
	}
}

func assertProvisionCellFits(t *testing.T, summary *ProvisionSummary, row image.Rectangle, cell provisionCell, resource provisionResource) {
	t.Helper()
	require.True(t, cell.label.In(row) && cell.value.In(row))
	require.Equal(t, summary.labelGap, cell.value.Min.X-cell.label.Max.X, "quantity sits immediately after its label")
	require.LessOrEqual(t, measuredProvisionWidth(summary.ultimaFont, resource.label), cell.label.Dx())
	require.LessOrEqual(t, measuredProvisionWidth(summary.ultimaFont, resource.value), cell.value.Dx(), "%s quantity must fit at normal size", resource.label)
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
			// Unequal digit counts prove values start at their shared column
			// anchors rather than being right-aligned or individually recentered.
			state.PartyState.Inventory.Provisions.Food.Set(218)
			state.PartyState.Inventory.Gold.Set(1250)
			state.PartyState.Inventory.Provisions.Gems.Set(8)
			state.PartyState.Karma.Value = 75
			summary.Draw(state, actual)
			resources := provisionResources(state)
			rows := summary.rows()
			for rowIndex, rowResources := range resources {
				row := rows[rowIndex]
				cells := summary.resourceCells(row)
				for column, resource := range rowResources {
					cell := cells[column]
					if column > 0 {
						assertProvisionGutterPixels(t, actual, image.Rect(cells[column-1].value.Max.X, row.Min.Y, cell.label.Min.X, row.Max.Y))
					}
					assertProvisionFieldPixels(t, actual, reference, summary.output, cell.label, resource.label, resource.labelColor, etext.AlignEnd)
					assertProvisionFieldPixels(t, actual, reference, summary.output, cell.value, resource.value, resource.valueColor, etext.AlignStart)
				}
			}
			date, clock := summary.calendarCells(rows[3])
			assertProvisionFieldPixels(t, actual, reference, summary.output, date, state.DateTime.GetDateAsString(), ucolor.UITextPrimary.RGBA(), etext.AlignCenter)
			assertProvisionFieldPixels(t, actual, reference, summary.output, clock, state.DateTime.GetTimeAsString(), ucolor.UITextPrimary.RGBA(), etext.AlignCenter)
			cells := summary.ammunitionCells(rows[2])
			for _, count := range []uint16{0, 20, 21, 99} {
				state.PartyState.Inventory.Equipment.Set(references.Arrows, count)
				state.PartyState.Inventory.Equipment.Set(references.Quarrels, count)
				actual.Clear()
				summary.Draw(state, actual)
				row := rows[2]
				assertProvisionGutterPixels(t, actual, image.Rect(cells[0].status.Max.X, row.Min.Y, cells[1].label.Min.X, row.Max.Y))
				for column, resource := range provisionAmmunitionResources(state) {
					cell := cells[column]
					assertProvisionGutterPixels(t, actual, image.Rect(cell.value.Max.X, row.Min.Y, cell.status.Min.X, row.Max.Y))
					assertProvisionFieldPixels(t, actual, reference, summary.output, cell.label, resource.label, resource.labelColor, etext.AlignEnd)
					assertProvisionFieldPixels(t, actual, reference, summary.output, cell.value, resource.value, resource.valueColor, etext.AlignStart)
					if resource.status == "" {
						pixels := make([]byte, cell.status.Dx()*cell.status.Dy()*4)
						actual.SubImage(cell.status).(*ebiten.Image).ReadPixels(pixels)
						require.Equal(t, make([]byte, len(pixels)), pixels, "healthy ammunition keeps its status slot empty")
					} else {
						assertProvisionFieldPixels(t, actual, reference, summary.output, cell.status, resource.status, resource.statusColor, etext.AlignStart)
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
