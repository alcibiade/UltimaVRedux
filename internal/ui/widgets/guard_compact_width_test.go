package widgets

import (
	"image"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGuardComparisonWidthAndGuttersStayContentSized(t *testing.T) {
	options := guardResourceTestOptions()
	for index := range options[:6] {
		options[index].Name = "Avatar"
	}
	dialog := newGuardResourceTestDialog(options, func(int) {}, func() {})
	for _, size := range []image.Point{{1280, 720}, {1920, 1080}, {3440, 1440}} {
		setResponsiveDialogTestResolution(t, size.X, size.Y)
		shell := dialog.shell
		shell.ensureGuardResourceLayout(size.X, size.Y, guardResourceTestView(dialog))
		workspace := shell.guardResourceWorkspace(size.X, size.Y)
		panel := shell.layout.panelRect
		require.Less(t, panel.Dx(), workspace.Dx()*9/10, "large windows must not become empty name columns")
		require.LessOrEqual(t, math.Abs(float64((panel.Min.X-workspace.Min.X)-(workspace.Max.X-panel.Max.X))), 1.0)
		columns := shell.guardResourceColumns(shell.layout.rowRects[0], dialog.options)
		longest := 0.0
		for _, option := range dialog.options {
			shell.appendRosterRow(option, true, columns[0].Dx())
			width, _ := shell.bodyFont.MeasureString(shell.rowOutput.GetOutputStr(false))
			longest = max(longest, width)
		}
		require.LessOrEqual(t, columns[1].Min.X-columns[0].Min.X-int(math.Ceil(longest)), shell.layout.pixelUnit*8)
		for range len(dialog.options) {
			dialog.HandleKeyDown()
			shell.ensureGuardResourceLayout(size.X, size.Y, guardResourceTestView(dialog))
			require.Equal(t, panel, shell.layout.panelRect, "inspection cannot move or resize the card")
		}
	}
}
