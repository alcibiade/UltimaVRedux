package widgets

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/opentype"
)

func TestTextInputsShareParsedFontAndRetainIndependentCursorMetrics(t *testing.T) {
	first, second := new(TextInput), new(TextInput)
	first.getAndSetTtf(12)
	second.getAndSetTtf(24)
	faceA, ok := first.face.(*opentype.Face)
	require.True(t, ok)
	faceB, ok := second.face.(*opentype.Face)
	require.True(t, ok)
	require.NotSame(t, faceA, faceB)
	require.Same(t, textInputFontSource(), textInputFontSource())
	oldHeight := first.getFontHeight()
	second.getAndSetTtf(31.5)
	require.Equal(t, oldHeight, first.getFontHeight())
	require.Greater(t, second.getFontHeight(), oldHeight)
}
