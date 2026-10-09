package text

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestSplitSegmentsAtNewlines_NoNewlines(t *testing.T) {
	segments := []ColorSegment{
		{Text: "Hello World", Color: ucolor.EGAWhite},
	}

	result := splitSegmentsAtNewlines(segments)

	require.Len(t, result, 1, "Should produce 1 logical line")
	require.Len(t, result[0], 1, "Line should have 1 segment")
	assert.Equal(t, "Hello World", result[0][0].Text)
}

func TestSplitSegmentsAtNewlines_SingleNewline(t *testing.T) {
	segments := []ColorSegment{
		{Text: "Hello\nWorld", Color: ucolor.EGAWhite},
	}

	result := splitSegmentsAtNewlines(segments)

	require.Len(t, result, 2, "Should produce 2 logical lines")
	require.Len(t, result[0], 1, "First line should have 1 segment")
	require.Len(t, result[1], 1, "Second line should have 1 segment")
	assert.Equal(t, "Hello", result[0][0].Text)
	assert.Equal(t, "World", result[1][0].Text)
}

func TestSplitSegmentsAtNewlines_MultipleNewlines(t *testing.T) {
	segments := []ColorSegment{
		{Text: "Line1\nLine2\nLine3", Color: ucolor.EGAWhite},
	}

	result := splitSegmentsAtNewlines(segments)

	require.Len(t, result, 3, "Should produce 3 logical lines")
	assert.Equal(t, "Line1", result[0][0].Text)
	assert.Equal(t, "Line2", result[1][0].Text)
	assert.Equal(t, "Line3", result[2][0].Text)
}

func TestSplitSegmentsAtNewlines_ConsecutiveNewlines(t *testing.T) {
	// Consecutive newlines should produce empty lines (blank line spacing)
	segments := []ColorSegment{
		{Text: "Hello\n\nWorld", Color: ucolor.EGAWhite},
	}

	result := splitSegmentsAtNewlines(segments)

	require.Len(t, result, 3, "Should produce 3 logical lines (including blank)")
	assert.Equal(t, "Hello", result[0][0].Text)
	assert.Len(t, result[1], 0, "Middle line should be empty (blank line)")
	assert.Equal(t, "World", result[2][0].Text)
}

func TestOutputSetMaxCharsPerLineUpdatesStyledWrapping(t *testing.T) {
	output := &Output{maxCharsPerLine: 40} //nolint:exhaustruct // Focused wrapping fixture.
	output.SetMaxCharsPerLine(14)

	wrapped := output.wrapStyledLine([]ColorSegment{{Text: "Squid attacks Shamino", Color: ucolor.EGARed}})

	require.Len(t, wrapped, 2)
	assert.Equal(t, "Squid attacks", styledLineText(wrapped[0]))
	assert.Equal(t, "Shamino", styledLineText(wrapped[1]))
}

func TestWrapStyledLineMovesColoredWordIntactToNextLine(t *testing.T) {
	output := &Output{maxCharsPerLine: 16} //nolint:exhaustruct // Focused wrapping fixture.
	segments := []ColorSegment{
		{Text: "Squid attacks ", Color: ucolor.EGARed},
		{Text: "Shamino", Color: ucolor.EGAYellow},
	}

	wrapped := output.wrapStyledLine(segments)

	require.Len(t, wrapped, 2)
	assert.Equal(t, "Squid attacks ", styledLineText(wrapped[0]))
	assert.Equal(t, "Shamino", styledLineText(wrapped[1]))
	assert.Equal(t, ucolor.EGAYellow, wrapped[1].Segments[0].Color)
}

func TestLineBuilderIndentsWrappedRowsWithoutRepeatingMarker(t *testing.T) {
	output := NewOutput(nil, 0, 4, 15)
	output.AppendLine().
		CommandPromptMarker(ucolor.EGALightBlue).
		WithContinuationIndent(1).
		White("Squid attacks Shamino").
		Done()

	firstLine := output.lines[0]
	secondLine := output.lines[1]
	require.NotEmpty(t, firstLine.Segments)
	require.NotEmpty(t, secondLine.Segments)
	assert.Equal(t, FontTypeCommandPrompt, firstLine.Segments[0].FontType)
	assert.Equal(t, ucolor.EGALightBlue, firstLine.Segments[0].Color)
	assert.Equal(t, FontTypeNormal, secondLine.Segments[0].FontType)
	assert.Equal(t, " ", secondLine.Segments[0].Text)
	for _, segment := range secondLine.Segments {
		assert.NotEqual(t, FontTypeCommandPrompt, segment.FontType)
	}
	assert.Equal(t, " Squid attacks", styledLineText(firstLine))
	assert.Equal(t, " Shamino", styledLineText(secondLine))
}

func TestOutputSetMaxCharsPerLineRejectsInvalidWidth(t *testing.T) {
	output := &Output{maxCharsPerLine: 40} //nolint:exhaustruct // Focused wrapping fixture.

	assert.Panics(t, func() { output.SetMaxCharsPerLine(0) })
	assert.Panics(t, func() { output.SetMaxCharsPerLine(-1) })
}

func TestPlainWrappingPreservesUnicodeAndCanAvoidSyntheticHyphens(t *testing.T) {
	const value = "/Users/Ávatar/Documents/UltimaV"
	output := NewOutput(nil, 0, utf8.RuneCountInString(value)+1, 5)
	output.SetHyphenateLongWords(false)

	output.AddText(value)
	wrapped := output.GetOutputStr(false)
	var rendered strings.Builder
	for _, line := range output.lines {
		rendered.WriteString(line.Text)
	}

	assert.True(t, utf8.ValidString(wrapped))
	assert.Equal(t, value, rendered.String())
}

func styledLineText(line StyledLine) string {
	var result string
	for _, segment := range line.Segments {
		result += segment.Text
	}

	return result
}

func TestSplitSegmentsAtNewlines_MultipleSegments(t *testing.T) {
	// Newline in second segment
	segments := []ColorSegment{
		{Text: "Hello ", Color: ucolor.EGAWhite},
		{Text: "World\nGoodbye", Color: ucolor.EGAYellow},
	}

	result := splitSegmentsAtNewlines(segments)

	require.Len(t, result, 2, "Should produce 2 logical lines")
	require.Len(t, result[0], 2, "First line should have 2 segments")
	assert.Equal(t, "Hello ", result[0][0].Text)
	assert.Equal(t, ucolor.EGAWhite, result[0][0].Color)
	assert.Equal(t, "World", result[0][1].Text)
	assert.Equal(t, ucolor.EGAYellow, result[0][1].Color)

	require.Len(t, result[1], 1, "Second line should have 1 segment")
	assert.Equal(t, "Goodbye", result[1][0].Text)
	assert.Equal(t, ucolor.EGAYellow, result[1][0].Color, "Color should be preserved")
}

func TestSplitSegmentsAtNewlines_PreservesFontType(t *testing.T) {
	segments := []ColorSegment{
		{Text: "Normal\nRunic", Color: ucolor.EGAWhite, FontType: FontTypeNormal},
	}
	// Modify to have runic in same segment
	segments[0].FontType = FontTypeRunic

	result := splitSegmentsAtNewlines(segments)

	require.Len(t, result, 2)
	assert.Equal(t, FontTypeRunic, result[0][0].FontType, "FontType should be preserved on first line")
	assert.Equal(t, FontTypeRunic, result[1][0].FontType, "FontType should be preserved on second line")
}

func TestSplitSegmentsAtNewlines_MixedRunicAndNormal(t *testing.T) {
	segments := []ColorSegment{
		{Text: "Normal text ", Color: ucolor.EGAWhite, FontType: FontTypeNormal},
		{Text: "RUNIC\nMore runic", Color: ucolor.EGABrown, FontType: FontTypeRunic},
		{Text: " back to normal", Color: ucolor.EGAWhite, FontType: FontTypeNormal},
	}

	result := splitSegmentsAtNewlines(segments)

	require.Len(t, result, 2, "Should produce 2 lines")

	// First line: "Normal text " + "RUNIC"
	require.Len(t, result[0], 2)
	assert.Equal(t, "Normal text ", result[0][0].Text)
	assert.Equal(t, FontTypeNormal, result[0][0].FontType)
	assert.Equal(t, "RUNIC", result[0][1].Text)
	assert.Equal(t, FontTypeRunic, result[0][1].FontType)

	// Second line: "More runic" + " back to normal"
	require.Len(t, result[1], 2)
	assert.Equal(t, "More runic", result[1][0].Text)
	assert.Equal(t, FontTypeRunic, result[1][0].FontType)
	assert.Equal(t, " back to normal", result[1][1].Text)
	assert.Equal(t, FontTypeNormal, result[1][1].FontType)
}

func TestWrapStyledLine_WithNewlines(t *testing.T) {
	// Create Output with 45 char max (typical talk dialog)
	output := &Output{maxCharsPerLine: 45, maxLines: 10}

	segments := []ColorSegment{
		{Text: "Hello World\nGoodbye World", Color: ucolor.EGAWhite},
	}

	result := output.wrapStyledLine(segments)

	require.Len(t, result, 2, "Should produce 2 wrapped lines")
	assert.Equal(t, "Hello World", result[0].Segments[0].Text)
	assert.Equal(t, "Goodbye World", result[1].Segments[0].Text)
}

func TestWrapStyledLine_WithNewlinesAndWordWrap(t *testing.T) {
	// Create Output with 20 char max to force word wrapping
	output := &Output{maxCharsPerLine: 20, maxLines: 10}

	segments := []ColorSegment{
		{Text: "This is a very long line\nShort", Color: ucolor.EGAWhite},
	}

	result := output.wrapStyledLine(segments)

	// First logical line "This is a very long line" should word-wrap
	// Second logical line "Short" should be separate
	require.GreaterOrEqual(t, len(result), 2, "Should produce at least 2 lines")

	// Find the line with "Short" - it should be on its own line
	foundShort := false
	for _, line := range result {
		for _, seg := range line.Segments {
			if seg.Text == "Short" {
				foundShort = true
			}
		}
	}
	assert.True(t, foundShort, "Should find 'Short' as a separate line")
}

func TestWrapStyledLine_BlankLines(t *testing.T) {
	output := &Output{maxCharsPerLine: 45, maxLines: 10}

	// Double newline creates a blank line
	segments := []ColorSegment{
		{Text: "Before\n\nAfter", Color: ucolor.EGAWhite},
	}

	result := output.wrapStyledLine(segments)

	require.Len(t, result, 3, "Should produce 3 lines including blank")
	assert.Equal(t, "Before", result[0].Segments[0].Text)
	assert.Len(t, result[1].Segments, 0, "Middle line should be empty (blank)")
	assert.Equal(t, "After", result[2].Segments[0].Text)
}

func TestWrapSingleLogicalLine_Empty(t *testing.T) {
	output := &Output{maxCharsPerLine: 45, maxLines: 10}

	result := output.wrapSingleLogicalLine([]ColorSegment{})

	require.Len(t, result, 1, "Empty input should produce 1 empty line")
	assert.Len(t, result[0].Segments, 0, "Empty line should have no segments")
}
