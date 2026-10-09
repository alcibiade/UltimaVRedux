package text

import (
	"image/color"
	"testing"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/stretchr/testify/require"
)

func TestPanelHighlightLocationUsesWrappedRowsAndRequestedRange(t *testing.T) {
	output := NewOutput(nil, 1, 20, 8)
	output.AppendLine().White("First line before the match").Done()
	before := output.GetPanelContentLineCount()
	output.AppendLine().Text("needle", ucolor.EGALightCyan).White(" after").Done()
	row, found := output.FirstPanelLineWithColor(0, output.GetPanelContentLineCount(), color.RGBA{R: 85, G: 255, B: 255, A: 255})
	require.True(t, found)
	require.Equal(t, before, row)
	_, found = output.FirstPanelLineWithColor(0, before, ucolor.EGALightCyan)
	require.False(t, found)
	output.Reset()
	_, found = output.FirstPanelLineWithColor(0, 20, ucolor.EGALightCyan)
	require.False(t, found)
}
