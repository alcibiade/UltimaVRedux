package text

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// FontType indicates which font to use for rendering a text segment
type FontType int

const (
	// FontTypeNormal uses the standard TrueType font
	FontTypeNormal FontType = iota
	// FontTypeRunic uses the bitmap runic font (8x8 scaled)
	FontTypeRunic
	// FontTypeIBM uses the bitmap IBM font (8x8 scaled).
	FontTypeIBM
	// FontTypeCommandPrompt draws the original one-cell bitmap input marker.
	FontTypeCommandPrompt
)

// FontScale constants for semantic text sizing
const (
	// FontScaleNormal is the default font scale (1.0x)
	FontScaleNormal float64 = 1.0
	// FontScaleHeader is used for section headers (1.25x larger)
	FontScaleHeader float64 = 1.25
)

// ColorSegment represents a piece of text with a specific color and font type
type ColorSegment struct {
	Text      string
	Color     color.Color
	FontType  FontType // Defaults to FontTypeNormal (zero value)
	FontScale float64  // Scale multiplier for font size (0 or 1.0 = normal, 1.25 = header)
}

// LineBuilder provides a fluent API for building multi-colored text lines
type LineBuilder struct {
	segments           []ColorSegment
	output             *Output      // Reference to parent Output for Done()
	plainTextCallback  func(string) // Optional callback for plain text version (dual output)
	alignment          text.Align   // Horizontal alignment (default: AlignStart)
	continuationIndent int          // Columns reserved before wrapped continuation rows.
}

// newLineBuilder creates a new line builder (package-private)
func newLineBuilder(output *Output) *LineBuilder {
	return &LineBuilder{
		segments: make([]ColorSegment, 0, 4), // Pre-allocate for typical case
		output:   output,
	}
}

// Text adds a text segment with specified color
func (lb *LineBuilder) Text(text string, textColor color.Color) *LineBuilder {
	lb.segments = append(lb.segments, ColorSegment{Text: text, Color: textColor})

	return lb
}

// CommandPromptMarker adds the one-column bitmap marker using the supplied outline color.
func (lb *LineBuilder) CommandPromptMarker(markerColor color.Color) *LineBuilder {
	lb.segments = append(lb.segments, ColorSegment{
		Text:      " ",
		Color:     markerColor,
		FontType:  FontTypeCommandPrompt,
		FontScale: FontScaleNormal,
	})

	return lb
}

// Textf adds a formatted text segment with specified color
func (lb *LineBuilder) Textf(format string, textColor color.Color, args ...interface{}) *LineBuilder {
	return lb.Text(fmt.Sprintf(format, args...), textColor)
}

// White adds white text segment
func (lb *LineBuilder) Spoken(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGAWhite)
}

// White adds white text segment
func (lb *LineBuilder) Spokenf(text string, args ...interface{}) *LineBuilder {
	return lb.Textf(text, ucolor.EGAWhite, args...)
}

// AvatarSpoken adds light green text segment for player/avatar dialogue
func (lb *LineBuilder) AvatarWhatHeDid(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGALightGreen)
}

// AvatarSpokenf adds formatted light green text segment for player/avatar dialogue
func (lb *LineBuilder) AvatarWhatHeDidf(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGALightGreen, args...)
}

// AvatarSpoken adds light green text segment for player/avatar dialogue
func (lb *LineBuilder) AvatarSpoken(text string) *LineBuilder {
	// return lb.Text(text, ucolor.EGALightGreen)
	return lb.Text(text, ucolor.White)
}

// AvatarSpokenf adds formatted light green text segment for player/avatar dialogue
func (lb *LineBuilder) AvatarSpokenf(format string, args ...interface{}) *LineBuilder {
	// return lb.Textf(format, ucolor.EGALightGreen, args...)
	return lb.Textf(format, ucolor.EGAWhite, args...)
}

// White adds white text segment
func (lb *LineBuilder) White(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGAWhite)
}

// Whitef adds formatted white text segment
func (lb *LineBuilder) Whitef(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGAWhite, args...)
}

// Yellow adds yellow text segment
func (lb *LineBuilder) Yellow(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGAYellow)
}

// Yellowf adds formatted yellow text segment
func (lb *LineBuilder) Yellowf(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGAYellow, args...)
}

// Red adds red text segment
func (lb *LineBuilder) Red(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGARed)
}

// Redf adds formatted red text segment
func (lb *LineBuilder) Redf(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGARed, args...)
}

// LightRed adds light red text segment
func (lb *LineBuilder) LightRed(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGALightRed)
}

// LightRedf adds formatted light red text segment
func (lb *LineBuilder) LightRedf(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGALightRed, args...)
}

// Green adds green text segment
func (lb *LineBuilder) Green(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGAGreen)
}

// Greenf adds formatted green text segment
func (lb *LineBuilder) Greenf(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGAGreen, args...)
}

// Blue adds blue text segment
func (lb *LineBuilder) Blue(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGABlue)
}

// Bluef adds formatted blue text segment
func (lb *LineBuilder) Bluef(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGABlue, args...)
}

// Magenta adds magenta text segment
func (lb *LineBuilder) Magenta(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGAMagenta)
}

// Magentaf adds formatted magenta text segment
func (lb *LineBuilder) Magentaf(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGAMagenta, args...)
}

// LightBlue adds light blue text segment
func (lb *LineBuilder) LightBlue(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGALightBlue)
}

// LightBluef adds formatted light blue text segment
func (lb *LineBuilder) LightBluef(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGALightBlue, args...)
}

// Brown adds brown text segment
func (lb *LineBuilder) Brown(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGABrown)
}

// Brownf adds formatted brown text segment
func (lb *LineBuilder) Brownf(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGABrown, args...)
}

// LightGreen adds light green text segment
func (lb *LineBuilder) LightGreen(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGALightGreen)
}

// LightGreenf adds formatted light green text segment
func (lb *LineBuilder) LightGreenf(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGALightGreen, args...)
}

// DarkGray adds dark gray text segment
func (lb *LineBuilder) DarkGray(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGADarkGray)
}

// DarkGrayf adds formatted dark gray text segment
func (lb *LineBuilder) DarkGrayf(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGADarkGray, args...)
}

// LightGray adds light gray text segment
func (lb *LineBuilder) LightGray(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGALightGray)
}

// LightGrayf adds formatted light gray text segment
func (lb *LineBuilder) LightGrayf(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGALightGray, args...)
}

// Cyan adds cyan text segment
func (lb *LineBuilder) Cyan(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGACyan)
}

// Cyanf adds formatted cyan text segment
func (lb *LineBuilder) Cyanf(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGACyan, args...)
}

// LightCyan adds light cyan text segment
func (lb *LineBuilder) LightCyan(text string) *LineBuilder {
	return lb.Text(text, ucolor.EGALightCyan)
}

// LightCyanf adds formatted light cyan text segment
func (lb *LineBuilder) LightCyanf(format string, args ...interface{}) *LineBuilder {
	return lb.Textf(format, ucolor.EGALightCyan, args...)
}

// Runic adds a runic text segment using the bitmap runic font.
// The text will be rendered using the runic font at a scale matching the TTF line height.
// Color is inherited from the previous segment, defaulting to EGABrown if no previous segment.
func (lb *LineBuilder) Runic(text string) *LineBuilder {
	// Determine color: use previous segment's color if available, otherwise default to brown
	var segmentColor color.Color = ucolor.EGABrown
	if len(lb.segments) > 0 {
		segmentColor = lb.segments[len(lb.segments)-1].Color
	}

	return lb.RunicWithColor(text, segmentColor)
}

// RunicWithColor adds a runic text segment with a specific color.
// The text will be rendered using the bitmap runic font.
func (lb *LineBuilder) RunicWithColor(text string, textColor color.Color) *LineBuilder {
	lb.segments = append(lb.segments, ColorSegment{
		Text:     text,
		Color:    textColor,
		FontType: FontTypeRunic,
	})

	return lb
}

// Runicf adds a formatted runic text segment.
// Color is inherited from the previous segment, defaulting to EGABrown if no previous segment.
func (lb *LineBuilder) Runicf(format string, args ...interface{}) *LineBuilder {
	return lb.Runic(fmt.Sprintf(format, args...))
}

// IBM adds an IBM text segment using the fixed 8x8 bitmap font.
// Color is inherited from the previous segment, defaulting to white.
func (lb *LineBuilder) IBM(text string) *LineBuilder {
	segmentColor := color.Color(ucolor.EGAWhite)
	if len(lb.segments) > 0 {
		segmentColor = lb.segments[len(lb.segments)-1].Color
	}

	return lb.IBMWithColor(text, segmentColor)
}

// IBMWithColor adds an IBM bitmap text segment with a specific color.
func (lb *LineBuilder) IBMWithColor(text string, textColor color.Color) *LineBuilder {
	lb.segments = append(lb.segments, ColorSegment{
		Text:      text,
		Color:     textColor,
		FontType:  FontTypeIBM,
		FontScale: FontScaleNormal,
	})

	return lb
}

// SlotIcon adds a single runic character representing an equipment slot type.
// The icon is rendered using the runic bitmap font with the specified color.
// If no color is specified, uses the previous segment's color or defaults to EGABrown.
func (lb *LineBuilder) SlotIcon(slot references.EquipmentSlot) *LineBuilder {
	iconChar := string([]byte{slot.RunicIconChar()})

	return lb.Runic(iconChar)
}

// SlotIconWithColor adds a single runic character representing an equipment slot type with a specific color.
func (lb *LineBuilder) SlotIconWithColor(slot references.EquipmentSlot, textColor color.Color) *LineBuilder {
	iconChar := string([]byte{slot.RunicIconChar()})

	return lb.RunicWithColor(iconChar, textColor)
}

// EquipmentIcon adds a single runic character representing a specific equipment item.
// Uses the equipment-specific Runic display code; ref:EQUIP-0003.
// For example, a Bow shows a bow icon, a Halberd shows a halberd icon, etc.
func (lb *LineBuilder) EquipmentIcon(equipment references.Equipment) *LineBuilder {
	iconChar := string([]byte{equipment.GetIconChar()})

	return lb.Runic(iconChar)
}

// EquipmentIconWithColor adds a single runic character representing a specific equipment item with a specific color.
// Uses the equipment-specific Runic display code; ref:EQUIP-0003.
func (lb *LineBuilder) EquipmentIconWithColor(equipment references.Equipment, textColor color.Color) *LineBuilder {
	iconChar := string([]byte{equipment.GetIconChar()})

	return lb.RunicWithColor(iconChar, textColor)
}

// Header adds text with a larger font (1.25x scale) for section headers.
// Default color is white.
func (lb *LineBuilder) Header(text string) *LineBuilder {
	return lb.HeaderWithColor(text, ucolor.EGAWhite)
}

// Headerf adds formatted header text with a larger font (1.25x scale).
// Default color is white.
func (lb *LineBuilder) Headerf(format string, args ...interface{}) *LineBuilder {
	return lb.Header(fmt.Sprintf(format, args...))
}

// HeaderWithColor adds header text with a larger font (1.25x scale) and custom color.
func (lb *LineBuilder) HeaderWithColor(text string, textColor color.Color) *LineBuilder {
	lb.segments = append(lb.segments, ColorSegment{
		Text:      text,
		Color:     textColor,
		FontScale: FontScaleHeader,
	})

	return lb
}

// NewLine commits the current line and starts a new blank line for spacing
func (lb *LineBuilder) NewLine() *LineBuilder {
	// Commit current line if it has content
	if len(lb.segments) > 0 {
		lb.output.AddStyledText(lb.segments, lb.alignment)
		lb.segments = make([]ColorSegment, 0, 4) // Reset for next line
	}
	// Add blank line using styled text with empty segment
	// Using AddStyledText ensures the line is actually added (legacy methods had a bug
	// where empty strings got filtered out by the trailing-newline artifact removal)
	lb.output.AddStyledText([]ColorSegment{{Text: "", Color: ucolor.EGAWhite}}, lb.alignment)

	return lb
}

// QuoteStart adds opening quotation mark with specified color
func (lb *LineBuilder) QuoteStart() *LineBuilder {
	lb.segments = append(lb.segments, ColorSegment{Text: "\"", Color: ucolor.EGALightGray})

	return lb
}

func (lb *LineBuilder) QuoteStartWithColor(textColor color.Color) *LineBuilder {
	lb.segments = append(lb.segments, ColorSegment{Text: "\"", Color: textColor})

	return lb
}

// QuoteEnd adds closing quotation mark with specified color
func (lb *LineBuilder) QuoteEnd() *LineBuilder {
	lb.segments = append(lb.segments, ColorSegment{Text: "\"", Color: ucolor.EGALightGray})

	return lb
}

// QuoteEndWithColor adds closing quotation mark with specified color
func (lb *LineBuilder) QuoteEndWithColor(textColor color.Color) *LineBuilder {
	lb.segments = append(lb.segments, ColorSegment{Text: "\"", Color: textColor})

	return lb
}

// WithAlignment sets the horizontal alignment for the line
func (lb *LineBuilder) WithAlignment(alignment text.Align) *LineBuilder {
	lb.alignment = alignment

	return lb
}

// WithContinuationIndent reserves columns before physical rows created by wrapping.
func (lb *LineBuilder) WithContinuationIndent(columns int) *LineBuilder {
	if columns < 0 {
		panic("LineBuilder continuation indent cannot be negative")
	}
	lb.continuationIndent = columns

	return lb
}

// WordWrap is a self-documenting marker indicating the caller expects word wrapping.
// Note: AddStyledText already wraps text automatically at word boundaries using
// Output.maxCharsPerLine, so this method doesn't change runtime behavior.
// Use it to make intent explicit in calling code.
func (lb *LineBuilder) WordWrap() *LineBuilder {
	// No-op: wrapping is automatic in AddStyledText, but this method
	// serves as documentation that wrapping is expected/desired
	return lb
}

// SetPlainTextCallback sets an optional callback to receive plain text version of the line
// This enables dual output: colored text to primary output, plain text to callback destination
func (lb *LineBuilder) SetPlainTextCallback(callback func(string)) {
	lb.plainTextCallback = callback
}

// convertToPlainText extracts plain text from all color segments
func (lb *LineBuilder) convertToPlainText() string {
	var plain string
	var plainSb445 strings.Builder
	for _, seg := range lb.segments {
		plainSb445.WriteString(seg.Text)
	}
	plain += plainSb445.String()

	return plain
}

// GetSegments returns the current segments for debugging purposes
func (lb *LineBuilder) GetSegments() []ColorSegment {
	return lb.segments
}

// Done commits the built line to the parent Output
// Note: AddStyledText automatically handles word wrapping at word boundaries,
// so text longer than Output.maxCharsPerLine will be split across multiple lines.
func (lb *LineBuilder) Done() {
	// PRIMARY: Write colored text to primary output (MessageDialog)
	// AddStyledText handles word wrapping internally via wrapStyledLine()
	lb.output.addStyledText(lb.segments, lb.alignment, lb.continuationIndent)

	// SECONDARY: Write plain text to callback destination if set (output pane)
	if lb.plainTextCallback != nil {
		plainText := lb.convertToPlainText()
		lb.plainTextCallback(plainText)
	}
}

//// DoneWithoutAdvance commits the built line WITHOUT advancing to the next line
//// This is used specifically for prompts like "You respond:" where user input
//// should appear on the same line as the prompt.
//// The line index will stay at the last written line, allowing AppendToCurrentRowStr()
//// to append user input to the same line.
// func (lb *LineBuilder) DoneWithoutAdvance() {
//	// PRIMARY: Write colored text to primary output (MessageDialog)
//	lb.output.AddStyledPrompt(lb.segments, lb.alignment)
//
//	// SECONDARY: Write plain text to callback destination if set (output pane)
//	if lb.plainTextCallback != nil {
//		plainText := lb.convertToPlainText()
//		lb.plainTextCallback(plainText)
//	}
//}
