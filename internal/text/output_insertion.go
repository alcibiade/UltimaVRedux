package text

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	ultimacolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// ============================================================================
// Text Insertion API - Simplified & Clear
// ============================================================================

// AddText adds text to the output buffer with wrapping, trimming, and line advancement
// This is the primary method for adding single-color text output
// Automatically handles newlines, word wrapping, and advances nextLineToIndex
func (o *Output) AddText(text string) {
	o.addTextInternal(text, true, o.color)
}

// AddTextRaw adds text without trimming whitespace
// Use this for conversation dialogs and other cases where leading/trailing whitespace is intentional
func (o *Output) AddTextRaw(text string) {
	o.addTextInternal(text, false, o.color)
}

// AddTextWithColor adds text with a specific color, overriding the default
// Automatically handles wrapping and line advancement
func (o *Output) AddTextWithColor(text string, textColor color.Color) {
	savedColor := o.color
	o.color = textColor
	o.addTextInternal(text, true, textColor)
	o.color = savedColor
}

// AddStyledText adds multi-color text with alignment and line advancement
// Use this for colored text segments (e.g., green prompts, white responses)
func (o *Output) AddStyledText(segments []ColorSegment, alignment text.Align) {
	o.addStyledText(segments, alignment, 0)
}

func (o *Output) addStyledText(segments []ColorSegment, alignment text.Align, continuationIndent int) {
	o.retainInsertion(outputInsertion{
		segments: segments, alignment: alignment, indent: continuationIndent,
		raw: "", color: nil, append: false, plain: false, trim: false,
	})
	if len(segments) == 0 {
		return
	}
	// Guard: Skip if no lines allocated
	if o.maxLines == 0 {
		return
	}

	wrappedLines := o.wrapStyledLineWithContinuationIndent(segments, continuationIndent)
	for _, line := range wrappedLines {
		line.Alignment = alignment
		o.lines[o.nextLineToIndex] = line
		o.nextLineToIndex = (o.nextLineToIndex + 1) % o.maxLines
	}
}

// AddStyledPrompt adds multi-color text for prompts (advances to next line after)
// Same as AddStyledText but semantically clearer for prompt use cases
// After writing the prompt, nextLineToIndex points to the next empty slot,
// allowing AppendText() to append user input to the prompt line
func (o *Output) AddStyledPrompt(segments []ColorSegment, alignment text.Align) {
	o.retainInsertion(outputInsertion{
		segments: segments, alignment: alignment, indent: 0,
		raw: "", color: nil, append: false, plain: false, trim: false,
	})
	if len(segments) == 0 {
		return
	}
	// Guard: Skip if no lines allocated
	if o.maxLines == 0 {
		return
	}

	wrappedLines := o.wrapStyledLine(segments)
	for i, line := range wrappedLines {
		line.Alignment = alignment
		o.lines[o.nextLineToIndex] = line

		// Advance for all wrapped lines EXCEPT the last one
		if i < len(wrappedLines)-1 {
			o.nextLineToIndex = (o.nextLineToIndex + 1) % o.maxLines
		}
	}

	// Advance one more time so nextLineToIndex points to next empty slot
	// This maintains the invariant for AppendText()
	o.nextLineToIndex = (o.nextLineToIndex + 1) % o.maxLines
}

// AddCommandPrompt starts a line with the original one-cell bitmap input marker.
func (o *Output) AddCommandPrompt(command string) {
	o.commandHintAdded = false
	segments := []ColorSegment{{
		Text:      " ",
		Color:     ultimacolor.EGABlue,
		FontType:  FontTypeCommandPrompt,
		FontScale: FontScaleNormal,
	}}
	if command != "" {
		segments = append(segments, ColorSegment{
			Text:      command,
			Color:     ultimacolor.EGAWhite,
			FontType:  FontTypeNormal,
			FontScale: FontScaleNormal,
		})
	}
	o.AddStyledPrompt(segments, text.AlignStart)
}

// AppendCommandHintOnce keeps guidance in the command's current append stream.
// Each new command prompt, including a retry, permits exactly one hint.
func (o *Output) AppendCommandHintOnce(hint string) {
	if o.commandHintAdded {
		return
	}
	o.commandHintAdded = true
	o.AppendStyledText([]ColorSegment{{
		Text: " [" + hint + "] ", Color: ultimacolor.EGALightGray,
		FontType: FontTypeNormal, FontScale: FontScaleNormal,
	}})
}

// AppendText appends text to the current line WITHOUT advancing nextLineToIndex
// Use this to add user input after a prompt on the same line
// Call AddText("\n") afterward to advance to the next line
func (o *Output) AppendText(text string) {
	o.AppendStyledText([]ColorSegment{{
		Text:      text,
		Color:     ultimacolor.EGAWhite,
		FontType:  FontTypeNormal,
		FontScale: FontScaleNormal,
	}})
}

// AppendStyledText appends styled text at the end of the current output row.
// The caller owns deciding whether the source stream has a block boundary.
func (o *Output) AppendStyledText(addedSegments []ColorSegment) {
	o.retainInsertion(outputInsertion{
		segments: addedSegments, append: true, alignment: text.AlignStart, indent: 0,
		raw: "", color: nil, plain: false, trim: false,
	})
	// Guard: Skip if no lines allocated
	if o.maxLines == 0 || len(addedSegments) == 0 {
		return
	}
	lastLineAdded := o.getCurrentRow()
	currentLine := o.lines[lastLineAdded]
	segments := currentLine.Segments
	if len(segments) == 0 && currentLine.Text != "" {
		segments = []ColorSegment{{
			Text:      currentLine.Text,
			Color:     currentLine.Color,
			FontType:  FontTypeNormal,
			FontScale: FontScaleNormal,
		}}
	}
	segments = append(segments, addedSegments...)

	wrappedLines := o.wrapStyledLine(segments)
	if styledSegmentsEndInNewline(addedSegments) && len(wrappedLines) > 0 &&
		styledLineIsEmpty(wrappedLines[len(wrappedLines)-1]) {
		wrappedLines = wrappedLines[:len(wrappedLines)-1]
	}
	for index, line := range wrappedLines {
		line.Alignment = currentLine.Alignment
		if index == 0 {
			o.lines[lastLineAdded] = line

			continue
		}
		o.lines[o.nextLineToIndex] = line
		o.nextLineToIndex = (o.nextLineToIndex + 1) % o.maxLines
	}
}

func styledSegmentsEndInNewline(segments []ColorSegment) bool {
	for index := len(segments) - 1; index >= 0; index-- {
		if segments[index].Text != "" {
			return strings.HasSuffix(segments[index].Text, "\n")
		}
	}

	return false
}

func styledLineIsEmpty(line StyledLine) bool {
	if line.Text != "" {
		return false
	}
	for _, segment := range line.Segments {
		if segment.Text != "" {
			return false
		}
	}

	return true
}

// addTextInternal is the internal implementation for text insertion
func (o *Output) addTextInternal(outputStr string, bTrim bool, textColor color.Color) {
	o.retainInsertion(outputInsertion{
		raw: outputStr, trim: bTrim, color: textColor, plain: true,
		segments: nil, alignment: text.AlignStart, indent: 0, append: false,
	})
	// Guard: Skip if no lines allocated
	if o.maxLines == 0 {
		return
	}
	// Process the string line-by-line, splitting by '\n'
	lines := splitByNewline(outputStr)

	// Remove trailing empty string if present (artifact of split on trailing \n)
	// Example: "text\n\n" splits to ["text", "", ""] - the last "" is an artifact
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		// Preserve intentional blank lines from \n\n sequences
		// These represent explicit spacing in multi-line messages
		if len(line) == 0 {
			o.lines[o.nextLineToIndex] = StyledLine{Text: "", Color: textColor, Segments: nil, Alignment: text.AlignStart}
			o.nextLineToIndex = (o.nextLineToIndex + 1) % o.maxLines

			continue
		}

		// Process each line, splitting by the configured width or nearest space.
		lineRunes := []rune(line)
		for len(lineRunes) > 0 {
			splitAt, splitOnSpace := plainLineBreak(lineRunes, o.maxCharsPerLine)

			// Consume the delimiter selected for wrapping. Raw insertion preserves
			// authored whitespace, but the wrap delimiter is not part of either row.
			chunk := string(lineRunes[:splitAt])
			lineRunes = lineRunes[splitAt:]
			if splitOnSpace {
				lineRunes = lineRunes[1:]
			}
			if bTrim {
				for len(lineRunes) > 0 && lineRunes[0] == ' ' {
					lineRunes = lineRunes[1:]
				}
			}

			// Add hyphen if the chunk has the maximum length and there is more to process
			if o.hyphenateLongWords && splitAt == o.maxCharsPerLine && len(lineRunes) > 0 {
				chunk += "-"
			}

			// Add the chunk to the output slice as simple StyledLine
			o.lines[o.nextLineToIndex] = StyledLine{
				Text:      chunk,
				Color:     textColor,
				Segments:  nil,
				Alignment: text.AlignStart,
			}
			o.nextLineToIndex = (o.nextLineToIndex + 1) % o.maxLines
		}
	}
}

// plainLineBreak preserves leading indentation and consumes only a selected
// inter-word delimiter. The caller owns trimming and visual hyphen insertion.
func plainLineBreak(line []rune, columns int) (int, bool) {
	if len(line) <= columns {
		return len(line), false
	}
	searchStart := 0
	for searchStart < columns && line[searchStart] == ' ' {
		searchStart++
	}
	spaceIndex := -1
	for index := searchStart; index < columns; index++ {
		if line[index] == ' ' {
			spaceIndex = index
		}
	}
	if spaceIndex != -1 {
		return spaceIndex, true
	}

	return columns, false
}

// AppendLine returns a LineBuilder for creating multi-colored lines
func (o *Output) AppendLine() *LineBuilder {
	return newLineBuilder(o)
}

// Helper function to split the input string by '\n' while keeping track of empty lines.
func splitByNewline(input string) []string {
	return strings.Split(input, "\n")
}
