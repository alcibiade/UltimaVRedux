package text

import (
	"fmt"
	"strings"
	"testing"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"
)

func TestOutputLogicalHistoryReflowsStylesAndAppendBoundaries(t *testing.T) {
	output := NewOutput(nil, 1, 14, 80)
	output.RetainLogicalHistory(256, 256*1024)
	output.AddStyledText([]ColorSegment{{Text: "Runic clue ", Color: ucolor.EGAWhite, FontType: FontTypeRunic, FontScale: FontScaleNormal}}, etext.AlignStart)
	output.AppendText("continued")
	output.AddTextWithColor("Received: Skull Keys", ucolor.EGAYellow)
	for index := range 30 {
		output.AddTextRaw(fmt.Sprintf("Later %d", index))
	}
	beforeCount := len(output.history.entries)
	for _, chars := range []int{10, 80, 10, 80} {
		output.ReflowRetained(nil, 1, chars)
		require.Equal(t, beforeCount, len(output.history.entries), "reflow must not record itself")
		panel := output.RetainedHistoryPanel()
		require.Contains(t, panel.GetOutputStr(true), "Skull")
		foundRunic, foundReward := false, false
		for _, line := range panel.lines {
			for _, segment := range line.Segments {
				if segment.FontType == FontTypeRunic {
					foundRunic = true
				}
			}
			if strings.Contains(line.Text, "Received") {
				require.Equal(t, ucolor.EGAYellow, line.Color)
				foundReward = true
			}
		}
		require.True(t, foundRunic)
		require.True(t, foundReward)
		if chars == 80 {
			require.Contains(t, panel.GetOutputStr(true), "Runic clue continued")
		}
	}
}

func TestOutputLogicalHistoryBoundsAndIsolation(t *testing.T) {
	output := NewOutput(nil, 1, 14, 80)
	output.RetainLogicalHistory(256, 256*1024)
	for index := range 300 {
		output.AddTextRaw(fmt.Sprintf("Line %03d", index))
	}
	require.Len(t, output.history.entries, 256)
	panel := output.RetainedHistoryPanel()
	require.Contains(t, panel.GetOutputStr(true), "Earlier conversation text truncated")
	require.NotContains(t, panel.GetOutputStr(true), "Line 000")
	require.Contains(t, panel.GetOutputStr(true), "Line 299")
	output.AddTextRaw(strings.Repeat("x", 256*1024+1))
	require.LessOrEqual(t, output.history.bytes, 256*1024)
	require.Len(t, output.history.entries, 1)
	require.Len(t, output.history.entries[0].raw, 256*1024)
	other := NewOutput(nil, 1, 14, 80)
	other.RetainLogicalHistory(256, 256*1024)
	require.Empty(t, other.RetainedHistoryPanel().GetOutputStr(true))
}

func TestOutputLogicalHistoryTruncationPreservesLiveFontOnlyResize(t *testing.T) {
	output := NewOutput(nil, 1, 14, 45)
	output.RetainLogicalHistory(256, 256*1024)
	output.AddTextRaw("Visible prefix: ")
	for range 300 {
		output.AppendText("x")
	}
	before := output.GetOutputStr(true)
	require.True(t, output.history.truncated)
	require.Contains(t, before, "Visible prefix")
	for _, spacing := range []float64{2, 1, 3} {
		output.ReflowRetained(nil, spacing, 45)
		require.Equal(t, before, output.GetOutputStr(true))
	}
	retained := len(output.history.entries)
	for range 300 {
		output.AddTextRaw("")
	}
	require.Len(t, output.history.entries, retained, "non-emissions cannot evict visible history")
}

func TestOutputLogicalHistoryOversizedStyledEmissionRetainsUTF8Tail(t *testing.T) {
	output := NewOutput(nil, 1, 14, 45)
	output.RetainLogicalHistory(256, 7)
	output.AddStyledText([]ColorSegment{
		{Text: "old prefix", Color: ucolor.EGAWhite, FontType: FontTypeNormal, FontScale: FontScaleNormal},
		{Text: "\u00e9\u00e9TAIL", Color: ucolor.EGAYellow, FontType: FontTypeRunic, FontScale: FontScaleNormal},
	}, etext.AlignStart)
	require.LessOrEqual(t, output.history.bytes, 7)
	require.Len(t, output.history.entries, 1)
	require.Equal(t, "\u00e9TAIL", output.history.entries[0].segments[0].Text)
	require.Equal(t, FontTypeRunic, output.history.entries[0].segments[0].FontType)
	require.Equal(t, ucolor.EGAYellow, output.history.entries[0].segments[0].Color)
	panel := output.RetainedHistoryPanel().GetOutputStr(true)
	require.Contains(t, panel, historyTruncationNotice)
	require.Contains(t, panel, "TAIL")
}
