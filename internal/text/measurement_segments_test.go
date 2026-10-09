package text

import (
	"testing"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/stretchr/testify/require"
)

func TestMixedLineMeasurementIncludesBitmapAdvance(t *testing.T) {
	font := NewUltimaFont(20)
	output := NewOutput(font, 24, 1, 80)
	line := output.AppendLine().White("[E]").RunicWithColor("A", ucolor.EGAYellow).White(" Bow (99)")
	plainWidth, _ := font.MeasureString("[E] Bow (99)")
	require.InDelta(t, plainWidth+24, output.MeasureLineSegments(line.GetSegments()), 0.001)
	line.Done()
}
