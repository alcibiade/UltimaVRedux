package widgets

import (
	"fmt"
	"image/color"
	"math"
	"regexp"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ultimacolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/grammar"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

const (
	defaultCursorHeight      = 16
	keyPressDelay            = 300
	cursorWidth              = 10
	cursorBlinkRateInMs      = 1000
	pauseBlinkKeyPressedInMs = 750
)

// TextInputColors supplies semantic command-match colours for this widget.
type TextInputColors struct {
	DefaultColor          color.Color
	NoMatchesColor        color.Color
	OneMatchColor         color.Color
	MoreThanOneMatchColor color.Color
}

// TextInputCallbacks are optional host-owned command and submission actions.
type TextInputCallbacks struct {
	AmbiguousAutoComplete func(string)
	OnEnter               func()
	OnCommandExecuted     func(string) // Called after successful command execution with the command string
	ShowCommandList       func()       // Called when Tab is pressed on empty input (for paginated command list)
}

// TextInput owns the game's command-entry input policy, using shared typography.
type TextInput struct {
	output     *text.Output
	ultimaFont *text.UltimaFont

	maxCharsPerLine int

	inputColors TextInputColors

	textCommands *grammar.TextCommands

	mainTextPlacement sprites.PercentBasedPlacement

	keyboard *input.Keyboard

	TextInputCallbacks TextInputCallbacks

	// Input validation regex - if set, only input matching this pattern is accepted
	inputValidationRegex *regexp.Regexp

	// Prompt prefix displayed before user input (e.g., "Your interest? > ")
	promptPrefix string
	promptColor  color.Color

	// Cursor visibility control - when false, cursor is not drawn
	cursorVisible bool

	// Placeholder text shown when input is empty (e.g., "ex. name, job, bye")
	placeholderText  string
	placeholderColor color.Color

	// Auto-replace mode for single-character select dialogs
	// When enabled, typing a new character automatically replaces existing text
	autoReplaceMode bool
	freeText        bool
}

// SetFreeTextMode preserves spaces and submits OnEnter without grammar completion.
func (t *TextInput) SetFreeTextMode() {
	t.freeText = true
	t.textCommands = nil
}

//nolint:gochecknoglobals // Immutable legacy command-entry key vocabulary.
var nonAlphaNumericBoundKeys = []ebiten.Key{
	ebiten.KeyEnter,
	ebiten.KeyNumpadEnter,
	ebiten.KeySpace,
	ebiten.KeyBackspace,
	ebiten.KeyDelete,
	ebiten.KeyEscape,
	ebiten.KeyTab,
	ebiten.KeyApostrophe,
	ebiten.KeyBackslash,
	ebiten.KeyBracketLeft,
	ebiten.KeyBracketRight,
	ebiten.KeyComma,
	ebiten.KeyEqual,
	ebiten.KeyGraveAccent,
	ebiten.KeyMinus,
	ebiten.KeyPeriod,
	ebiten.KeySemicolon,
	ebiten.KeySlash,
	ebiten.KeyDown,
	ebiten.KeyLeft,
	ebiten.KeyRight,
	ebiten.KeyUp,
}

// Clear removes all entered text while retaining input focus.
func (t *TextInput) Clear() {
	t.output.Clear()
}

// NewTextInput creates a new text input widget for user text entry with command processing.
//
// The text input supports cursor positioning, character insertion/deletion,
// command parsing, and keyboard event handling with customizable callbacks.
//
// Parameters:
//   - mainTextPlacement: Screen area as percentage-based placement
//   - fontPointSize: Font size in points for text rendering
//   - maxCharsPerLine: Maximum characters per line before wrapping
//   - textCommands: Grammar system for command parsing (can be nil)
//   - callbacks: Function callbacks for text change events
//   - keyboard: Input handler for key processing (required)
//
// Returns a configured TextInput that implements the Widget interface.
func NewTextInput(
	mainTextPlacement sprites.PercentBasedPlacement,
	fontPointSize float64,
	maxCharsPerLine int,
	textCommands *grammar.TextCommands,
	callbacks TextInputCallbacks,
	keyboard *input.Keyboard,
) *TextInput {
	textInput := new(TextInput)
	textInput.maxCharsPerLine = maxCharsPerLine
	textInput.keyboard = keyboard
	textInput.textCommands = textCommands
	textInput.TextInputCallbacks = callbacks
	textInput.mainTextPlacement = mainTextPlacement

	// NOTE: single line input only (for now?)
	textInput.output = text.NewOutput(textInput.ultimaFont, 0, 1, maxCharsPerLine)
	textInput.SetFontPoint(fontPointSize)

	textInput.output.AddText("")

	textInput.inputColors = TextInputColors{
		DefaultColor:          color.White,
		NoMatchesColor:        ultimacolor.Red,
		OneMatchColor:         ultimacolor.Green,
		MoreThanOneMatchColor: ultimacolor.Yellow,
	}

	// Initialize prompt color to white by default
	textInput.promptColor = color.White

	// Initialize cursor as visible by default
	textInput.cursorVisible = true

	return textInput
}

// SetInputColors replaces the widget's semantic match colours.
func (t *TextInput) SetInputColors(inputColors TextInputColors) {
	t.inputColors = inputColors
}

// SetPlacement updates the derived drawing area without replacing input state.
func (t *TextInput) SetPlacement(placement sprites.PercentBasedPlacement) {
	t.mainTextPlacement = placement
}

// SetPromptPrefix sets the prompt text displayed before user input
func (t *TextInput) SetPromptPrefix(prompt string) {
	t.promptPrefix = prompt
}

// SetPromptColor sets the color for the prompt text
func (t *TextInput) SetPromptColor(c color.Color) {
	t.promptColor = c
}

// ClearPrompt removes the prompt prefix
func (t *TextInput) ClearPrompt() {
	t.promptPrefix = ""
}

// SetCursorVisible controls whether the cursor is drawn
func (t *TextInput) SetCursorVisible(visible bool) {
	t.cursorVisible = visible
}

// SetPlaceholder sets the placeholder text and color
func (t *TextInput) SetPlaceholder(text string, c color.Color) {
	t.placeholderText = text
	t.placeholderColor = c
}

// ClearPlaceholder removes the placeholder text
func (t *TextInput) ClearPlaceholder() {
	t.placeholderText = ""
	t.placeholderColor = nil
}

// SetAutoReplaceMode enables or disables auto-replace mode for single-character select dialogs
// When enabled, typing a new character automatically replaces the existing text instead of blocking input
func (t *TextInput) SetAutoReplaceMode(enabled bool) {
	t.autoReplaceMode = enabled
}

// SetInputValidationRegex sets a regex pattern to validate input characters
// Only characters that result in a string matching this pattern will be accepted
// Pass nil to disable validation
func (t *TextInput) SetInputValidationRegex(pattern string) error {
	if pattern == "" {
		t.inputValidationRegex = nil

		return nil
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}

	t.inputValidationRegex = regex

	return nil
}

// SetFontPoint updates independently sized typography without replacing input.
func (t *TextInput) SetFontPoint(fontPointSize float64) {
	t.ultimaFont = text.NewUltimaFont(fontPointSize)
	t.output.SetFont(t.ultimaFont, fontPointSize)
}

// GetText returns the current single-row command text.
func (t *TextInput) GetText() string {
	return t.output.GetOutputStr(false)
}

// ClearText clears the current entry without changing focus.
func (t *TextInput) ClearText() {
	t.output.Clear()
}

// SetText replaces the current entry, for example from command history.
func (t *TextInput) SetText(text string) {
	t.output.Clear()
	if text != "" {
		t.output.AddText(text)
	}
}

// CalculateTextWidth returns the total width of the string in pixels
func (t *TextInput) CalculateTextWidth(s string) int {
	width, _ := t.ultimaFont.MeasureString(s)

	return int(width)
}

// getFontHeight returns the height of the font in pixels
func (t *TextInput) getFontHeight() int {
	if t.ultimaFont != nil {
		metrics := t.ultimaFont.Metrics()

		return int(math.Ceil(metrics.Ascent + 2*metrics.Descent + metrics.LineGap))
	}
	// Fallback to a reasonable default if face is not available
	return defaultCursorHeight
}
