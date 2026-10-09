package text

import (
	"image/color"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	ultimacolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// StyledLine holds either a simple single-color line or multi-color segments
type StyledLine struct {
	Segments  []ColorSegment // Non-empty for multi-color lines
	Text      string         // Used for simple single-color lines
	Color     color.Color    // Default color for simple lines
	Alignment text.Align     // Horizontal alignment (default: AlignStart for left-aligned)
}

type Output struct {
	Font               *UltimaFont
	lines              []StyledLine // Changed from []string to support multi-color
	nextLineToIndex    int
	lineSpacing        float64
	maxLines           int
	color              color.Color
	maxCharsPerLine    int
	hyphenateLongWords bool
	debugOptions       *references.DebugOptions // Debug options for rendering placeholders and overlays
	history            *OutputHistory
	journalHistory     *journalHistory
	commandHintAdded   bool
	lineLayouts        []lineLayout // Current physical-row geometry; never owns cached text images.
}

func NewOutput(font *UltimaFont, lineSpacing float64, maxLines, maxCharsPerLine int) *Output {
	output := &Output{}
	output.Font = font
	output.lineSpacing = lineSpacing
	output.maxLines = maxLines
	output.maxCharsPerLine = maxCharsPerLine
	output.hyphenateLongWords = true
	output.lines = make([]StyledLine, maxLines)
	output.SetColor(ultimacolor.White)

	return output
}

func (o *Output) SetFont(font *UltimaFont, lineSpacing float64) {
	o.Font = font
	o.lineSpacing = lineSpacing
	o.clearLineLayouts()
}

// SetMaxCharsPerLine updates the word-wrapping width for subsequently appended text.
func (o *Output) SetMaxCharsPerLine(maxCharsPerLine int) {
	if maxCharsPerLine <= 0 {
		panic("Output maxCharsPerLine must be positive")
	}
	o.maxCharsPerLine = maxCharsPerLine
}

// SetHyphenateLongWords controls whether wrapping adds a visual hyphen when an
// uninterrupted token crosses the configured character width.
func (o *Output) SetHyphenateLongWords(enabled bool) {
	o.hyphenateLongWords = enabled
}

func (o *Output) SetColor(color color.Color) {
	o.color = color
	// Guard: Skip if no lines allocated (maxLines = 0)
	if o.maxLines == 0 {
		return
	}
	// Update current line's color if it's a simple line (not multi-color)
	currentRow := o.getCurrentRow()
	if len(o.lines[currentRow].Segments) == 0 {
		line := o.lines[currentRow]
		line.Color = color
		o.lines[currentRow] = line
	}
}

// RecolorCurrentLine applies one color to all content on the current line.
// TextInput uses this for syntax coloring after its editable text has been
// represented as styled segments.
func (o *Output) RecolorCurrentLine(lineColor color.Color) {
	if o.maxLines == 0 {
		return
	}

	currentRow := o.getCurrentRow()
	line := o.lines[currentRow]
	line.Color = lineColor
	for index := range line.Segments {
		line.Segments[index].Color = lineColor
	}
	o.lines[currentRow] = line
}

// SetDebugOptions sets the debug options for rendering placeholders
// Pass nil to disable debug features
func (o *Output) SetDebugOptions(debugOptions *references.DebugOptions) {
	o.debugOptions = debugOptions
}

func (o *Output) Clear() {
	o.clearLineLayouts()
	if o.journalHistory != nil {
		o.journalHistory.startTarget(false)
	}
	for i := range o.lines {
		o.lines[i] = StyledLine{} // Clear to empty StyledLine
	}
}

// Reset clears all content AND resets the line index to 0
// Use this when starting fresh content that should render from the top
func (o *Output) Reset() {
	o.Clear()
	o.nextLineToIndex = 0
}

// ============================================================================
// Buffer Management - Internal Helpers
// ============================================================================

func (o *Output) getCurrentRow() int {
	// Guard against maxLines = 0 to prevent divide by zero
	// This can occur when MessageDialog has zero content capacity
	if o.maxLines == 0 {
		return 0
	}

	lastLineAdded := (o.nextLineToIndex - 1) % o.maxLines
	if lastLineAdded < 0 {
		lastLineAdded = o.maxLines - 1
	}

	return lastLineAdded
}

func (o *Output) GetLengthOfCurrentRowStr() int {
	// Guard: Return 0 if no lines allocated
	if o.maxLines == 0 {
		return 0
	}
	currentRow := o.getCurrentRow()
	line := o.lines[currentRow]
	if len(line.Segments) > 0 {
		// Styled line - count all segment lengths
		total := 0
		for _, seg := range line.Segments {
			total += utf8.RuneCountInString(seg.Text)
		}

		return total
	}

	return utf8.RuneCountInString(line.Text)
}

func (o *Output) RemoveRightSideCharacter() {
	// Guard: Skip if no lines allocated
	if o.maxLines == 0 {
		return
	}
	if o.journalHistory != nil {
		o.journalHistory.removeCharacter()
	}
	nCurrentRow := o.getCurrentRow()
	line := o.lines[nCurrentRow]
	if len(line.Segments) > 0 {
		// AppendText may create one segment per typed character. Discard empty
		// trailing segments before deleting so repeated Backspace presses continue
		// through the logical text.
		for len(line.Segments) > 0 && line.Segments[len(line.Segments)-1].Text == "" {
			line.Segments = line.Segments[:len(line.Segments)-1]
		}
		if len(line.Segments) > 0 {
			lastIdx := len(line.Segments) - 1
			_, size := utf8.DecodeLastRuneInString(line.Segments[lastIdx].Text)
			line.Segments[lastIdx].Text = line.Segments[lastIdx].Text[:len(line.Segments[lastIdx].Text)-size]
			if line.Segments[lastIdx].Text == "" {
				line.Segments = line.Segments[:lastIdx]
			}
		}
	} else if len(line.Text) > 0 {
		// Simple line
		_, size := utf8.DecodeLastRuneInString(line.Text)
		line.Text = line.Text[:len(line.Text)-size]
	}
	o.lines[nCurrentRow] = line
}

// ============================================================================
// Typography Measurements - Use these for consistent text height calculations
// ============================================================================

// GetLineSpacing returns the line spacing (baseline-to-baseline distance) used for this output.
// This is the value passed to NewOutput or SetFont.
func (o *Output) GetLineSpacing() float64 {
	return o.lineSpacing
}

// GetFontHeight returns the actual font height in pixels (ascent + descent).
// This represents the height of rendered glyphs, not including line gap.
func (o *Output) GetFontHeight() float64 {
	metrics := o.Font.TextFace.Metrics()

	return metrics.HAscent + metrics.HDescent
}

// GetTotalHeightForLines calculates the total height for N lines of text.
// Formula: (N-1) × lineSpacing + fontHeight
// This accounts for N-1 gaps between baselines plus the actual text height.
func (o *Output) GetTotalHeightForLines(n int) float64 {
	if n <= 0 {
		return 0
	}
	if n == 1 {
		return o.GetFontHeight()
	}

	return float64(n-1)*o.lineSpacing + o.GetFontHeight()
}

// GetLineCapacity returns the greatest number of complete lines that fit in height.
func (o *Output) GetLineCapacity(height int) int {
	fontHeight := o.GetFontHeight()
	if height <= 0 || float64(height) < fontHeight || o.lineSpacing <= 0 {
		return 0
	}

	return 1 + int(math.Floor((float64(height)-fontHeight)/o.lineSpacing))
}

// GetPanelContentLineCount returns the physical row count used by DrawPanelContent.
func (o *Output) GetPanelContentLineCount() int {
	for index := o.maxLines - 1; index >= 0; index-- {
		line := o.lines[index]
		if len(line.Segments) > 0 || strings.TrimSpace(line.Text) != "" {
			return index + 1
		}
	}

	return 0
}

// FirstPanelLineWithColor locates highlighted text after physical wrapping. This
// lets a reader reveal the actual rendered match instead of estimating its page.
func (o *Output) FirstPanelLineWithColor(start, end int, target color.Color) (int, bool) {
	red, green, blue, alpha := target.RGBA()
	matches := func(value string, c color.Color) bool {
		if value == "" || c == nil {
			return false
		}
		cr, cg, cb, ca := c.RGBA()

		return cr == red && cg == green && cb == blue && ca == alpha
	}
	for row := max(0, start); row < min(end, o.GetPanelContentLineCount()); row++ {
		line := o.lines[row]
		if matches(line.Text, line.Color) {
			return row, true
		}
		for _, segment := range line.Segments {
			if matches(segment.Text, segment.Color) {
				return row, true
			}
		}
	}

	return 0, false
}

func (o *Output) GetOutputStr(bShowEmptyNewLines bool) string {
	var outputStr string
	nTotalLines := len(o.lines)
	for i := range nTotalLines {
		line := o.lines[(i+o.nextLineToIndex)%o.maxLines]

		var lineText string
		if len(line.Segments) > 0 {
			// Styled line - concatenate all segments
			for _, seg := range line.Segments {
				lineText += seg.Text
			}
		} else {
			lineText = line.Text
		}

		outputStr += lineText
		if (bShowEmptyNewLines || lineText != "") && i < nTotalLines-1 {
			outputStr += " \n"
		}
	}

	return outputStr
}
