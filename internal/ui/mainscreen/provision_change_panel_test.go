package mainscreen

import (
	"fmt"
	"image"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestProvisionChangePanelFitsSupportedLayoutsWithFeaturedFood(t *testing.T) {
	for _, size := range []image.Point{
		{X: 1280, Y: 720}, {X: 1366, Y: 768}, {X: 1920, Y: 1080}, {X: 2560, Y: 1440}, {X: 3840, Y: 2160},
	} {
		t.Run(size.String(), func(t *testing.T) {
			var panel ProvisionChangePanel
			var rows [provisionChangeCount]ProvisionChange
			for index, name := range []string{"Food", "Gold", "Keys", "Gems", "Torches", "Skull keys"} {
				rows[index] = ProvisionChange{Name: name, Gained: 9999, Lost: 9999, Total: 9999}
			}
			viewport := image.Rectangle{Max: size}
			panel.Update(rows, viewport, size.X)
			require.True(t, panel.bounds.In(viewport))
			require.False(t, panel.foodBounds.Empty())
			require.True(t, panel.foodBounds.In(panel.bounds))
			for index, cell := range panel.cells {
				if index == 0 {
					require.Contains(t, panel.foodHeading.GetOutputStr(false), "FOOD -9999")
					require.Contains(t, panel.foodDetails.GetOutputStr(false), "+9999")
					continue
				}
				require.True(t, cell.In(panel.bounds))
				require.False(t, cell.Overlaps(panel.foodBounds))
				assert.False(t, cell.Overlaps(panel.headerBounds))
				for other := index + 1; other < len(panel.cells); other++ {
					assert.False(t, cell.Overlaps(panel.cells[other]))
				}
				output := panel.outputs[index]
				assert.Equal(t, fmt.Sprintf("%s +9999 -9999  Now 9999", rows[index].Name), output.GetOutputStr(false))
				width, _ := output.Font.MeasureString(output.GetOutputStr(false))
				assert.LessOrEqual(t, int(math.Ceil(width)), cell.Dx())
				assert.Equal(t, 1, output.GetPanelContentLineCount())
				assert.GreaterOrEqual(t, output.GetLineCapacity(cell.Dy()), 1)
			}
			assert.NotEqual(t, widgets.DialogLabelColor, widgets.DialogValueColor)
			outputs := panel.outputs
			panel.Update(rows, viewport, size.X)
			assert.Equal(t, outputs, panel.outputs, "stable content reuses all buffers")
			rows[0].Lost = 0
			panel.Update(rows, viewport, size.X)
			require.True(t, panel.foodBounds.Empty())
			assert.True(t, panel.bounds.In(viewport), "large summaries reflow instead of shrinking standard text")
		})
	}
}

func TestProvisionTypographyIsIndependentOfMapViewport(t *testing.T) {
	var panel ProvisionChangePanel
	var rows [provisionChangeCount]ProvisionChange
	rows[0] = ProvisionChange{Name: "Food", Lost: 6, Total: 100}
	rows[1] = ProvisionChange{Name: "Gold", Gained: 25, Total: 125}
	const screenWidth = 1280
	for _, width := range []int{1280, 960, 563} {
		panel.Update(rows, image.Rect(0, 0, width, 720), screenWidth)
		assert.InDelta(t, float64(widgets.FontSizeMainScreen)*screenWidth/1920, panel.header.Font.TextFace.Size, 0.001)
		assert.Equal(t, panel.header.Font.TextFace.Size, panel.foodDetails.Font.TextFace.Size)
		assert.InDelta(t, float64(widgets.FontSizeDialogTitle)*screenWidth/1920, panel.foodHeading.Font.TextFace.Size, 0.001)
		assert.True(t, panel.bounds.In(image.Rect(0, 0, width, 720)))
	}
	panel.Update(rows, image.Rect(0, 0, 960, 720), 1920)
	assert.Equal(t, float64(widgets.FontSizeMainScreen), panel.header.Font.TextFace.Size)
}

func TestProvisionTypographyRebuildsOnlyForLogicalWidthChange(t *testing.T) {
	var panel ProvisionChangePanel
	var rows [provisionChangeCount]ProvisionChange
	viewport := image.Rect(0, 0, 960, 720)
	panel.Update(rows, viewport, 1280)
	header, heading, details, outputs := panel.header, panel.foodHeading, panel.foodDetails, panel.outputs
	rows[0] = ProvisionChange{Name: "Food", Lost: 4, Total: 96}
	panel.Update(rows, viewport, 1280)
	panel.Update(rows, viewport.Add(image.Pt(10, 5)), 1280)
	panel.Place(image.Pt(20, 10))
	panel.Update([provisionChangeCount]ProvisionChange{}, viewport, 1280)
	assert.Same(t, header, panel.header)
	assert.Same(t, heading, panel.foodHeading)
	assert.Same(t, details, panel.foodDetails)
	assert.True(t, outputs == panel.outputs, "content/placement/expiry must retain output identities")
	panel.Update(rows, viewport, 1788)
	assert.NotSame(t, header, panel.header)
	assert.NotSame(t, heading, panel.foodHeading)
	assert.NotSame(t, details, panel.foodDetails)
	assert.InDelta(t, float64(widgets.FontSizeMainScreen)*1788/1920, panel.header.Font.TextFace.Size, 0.001)
}

func TestActualFoodLossUsesProminentHeadingAndPreservesGrossGains(t *testing.T) {
	var panel ProvisionChangePanel
	var rows [provisionChangeCount]ProvisionChange
	rows[0] = ProvisionChange{Name: "Food", Lost: 4, Total: 96}
	panel.Update(rows, image.Rect(0, 0, 1280, 720), 1280)
	require.Equal(t, "FOOD -4", panel.foodHeading.GetOutputStr(false))
	require.Equal(t, "Remaining: 96", panel.foodDetails.GetOutputStr(false))
	require.Greater(t, panel.foodHeading.Font.TextFace.Size, panel.header.Font.TextFace.Size)
	require.Less(t, panel.bounds.Max.Y, 72, "ordinary food notice fits above merchant content")
	rows[0].Gained, rows[0].Total = 10, 106
	panel.Update(rows, image.Rect(0, 0, 1280, 720), 1280)
	require.Contains(t, panel.foodHeading.GetOutputStr(false), "-4")
	require.Contains(t, panel.foodDetails.GetOutputStr(false), "+10")
	require.Contains(t, panel.foodDetails.GetOutputStr(false), "106")
	panel.Update([provisionChangeCount]ProvisionChange{}, image.Rect(0, 0, 1280, 720), 1280)
	require.True(t, panel.bounds.Empty())
}

func TestProvisionChangePanelPreservesHugeAmountsAndReflowsOnResize(t *testing.T) {
	var panel ProvisionChangePanel
	var rows [provisionChangeCount]ProvisionChange
	for index := range rows {
		rows[index] = ProvisionChange{Name: "Skull keys", Gained: math.MaxUint64, Lost: math.MaxUint64, Total: 99}
	}
	for _, size := range []image.Point{{X: 1920, Y: 1080}, {X: 1280, Y: 720}} {
		viewport := image.Rectangle{Max: size}
		panel.Update(rows, viewport, size.X)
		require.True(t, panel.bounds.In(viewport))
		for index, output := range panel.outputs {
			assert.Equal(t, "Skull keys +18446744073709551615 -18446744073709551615  Now 99", output.GetOutputStr(false))
			width, _ := output.Font.MeasureString(output.GetOutputStr(false))
			assert.LessOrEqual(t, int(math.Ceil(width)), panel.cells[index].Dx())
		}
	}
	panel.Update([provisionChangeCount]ProvisionChange{}, image.Rect(0, 0, 1280, 720), 1280)
	assert.True(t, panel.bounds.Empty(), "expiry removes backing as well as text")
}
