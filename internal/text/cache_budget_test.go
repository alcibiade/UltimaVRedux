package text

import (
	"image/color"
	"testing"

	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"
)

// Generic raster budget/eviction ownership tests now live in Berad Text. These
// assertions retain the game's physical-row and semantic-style adapter coverage.
func TestOutputLineGeometryTracksOnlyContentAndTypography(t *testing.T) {
	output := NewOutput(NewUltimaFont(20), 24, 3, 100)
	segments := []ColorSegment{{Text: "Strength: ", Color: color.White}, {Text: "20", Color: color.Black}}
	first := output.layoutLine(0, segments)
	advances := &first.advances[0]
	retained := first.layouts[0]
	width := first.total
	segments[0].Color = color.RGBA{R: 255, A: 255}
	require.Same(t, advances, &output.layoutLine(0, segments).advances[0])
	require.Same(t, retained, output.lineLayouts[0].layouts[0])
	require.Equal(t, width, output.lineLayouts[0].total)
	segments[1].Text = "2000"
	require.Greater(t, output.layoutLine(0, segments).total, width)
	beforeResize := output.lineLayouts[0].total
	output.Font.TextFace.Size = 31.5
	require.Greater(t, output.layoutLine(0, segments).total, beforeResize)
	wrapped := output.WrappedCopy(20)
	require.Nil(t, wrapped.lineLayouts)
	output.Reset()
	require.False(t, output.lineLayouts[0].valid)
	require.Empty(t, output.lineLayouts[0].inputs[0].text)
	require.Nil(t, output.lineLayouts[0].layouts[0])
}

func TestUnwrappedStyledFastPathKeepsCallerIsolation(t *testing.T) {
	output := NewOutput(NewUltimaFont(20), 24, 2, 8)
	segments := []ColorSegment{{Text: "éé", Color: color.White}, {Text: " x", Color: color.Black, FontType: FontTypeIBM}}
	output.AddStyledText(segments, etext.AlignEnd)
	require.Equal(t, segments, output.lines[0].Segments)
	require.Equal(t, etext.AlignEnd, output.lines[0].Alignment)
	segments[0].Text = "mutated"
	require.Equal(t, "éé", output.lines[0].Segments[0].Text)
}
