package widgets

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/text"
)

func TestTextInputsShareParsedFontAndRetainIndependentCursorMetrics(t *testing.T) {
	first, second := new(TextInput), new(TextInput)
	first.output = text.NewOutput(nil, 0, 1, 30)
	second.output = text.NewOutput(nil, 0, 1, 30)
	first.SetFontPoint(12)
	second.SetFontPoint(24)
	require.NotSame(t, first.ultimaFont.TextFace, second.ultimaFont.TextFace)
	require.Same(t, first.ultimaFont.TextFace.Source, second.ultimaFont.TextFace.Source)
	oldHeight := first.getFontHeight()
	second.SetFontPoint(31.5)
	require.Equal(t, oldHeight, first.getFontHeight())
	require.Greater(t, second.getFontHeight(), oldHeight)
	width, _ := first.ultimaFont.MeasureString("café")
	require.Equal(t, int(width), first.CalculateTextWidth("café"))
}
