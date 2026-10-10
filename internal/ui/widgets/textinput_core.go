package widgets

import (
	_ "embed"
	"fmt"
	"image/color"
	"regexp"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/bradhannah/UltimaVRedux/internal/profiling"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ultimacolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/grammar"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

const (
	keyPressDelay            = 300
	cursorWidth              = 10
	cursorBlinkRateInMs      = 1000
	pauseBlinkKeyPressedInMs = 750
)

//go:embed assets/ultima-v-warriors-of-destiny/ultima-v-warriors-of-destiny.ttf
var rawUltimaTTF []byte

//nolint:gochecknoglobals // Parsed data is immutable; each input retains its own hinting/size/cursor-metric face.
var textInputFontSource = sync.OnceValue(func() *opentype.Font {
	profiling.Count(profiling.FontSourceParses)
	source, err := opentype.Parse(rawUltimaTTF)
	if err != nil {
		panic(fmt.Sprintf("parsing embedded text input font: %v", err))
	}

	return source
})

type TextInputColors struct {
	DefaultColor          color.Color
	NoMatchesColor        color.Color
	OneMatchColor         color.Color
	MoreThanOneMatchColor color.Color
}

type TextInputCallbacks struct {
	AmbiguousAutoComplete func(string)
	OnEnter               func()
	OnCommandExecuted     func(string) // Called after successful command execution with the command string
	ShowCommandList       func()       // Called when Tab is pressed on empty input (for paginated command list)
}

type TextInput struct {
	output     *text.Output
	ultimaFont *text.UltimaFont
	face       font.Face

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
	textInput := &TextInput{}
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

func (t *TextInput) SetFontPoint(fontPointSize float64) {
	t.ultimaFont = text.NewUltimaFont(fontPointSize)
	t.getAndSetTtf(fontPointSize)
	t.output.SetFont(t.ultimaFont, fontPointSize)
}

func (t *TextInput) getAndSetTtf(fontPointSize float64) {
	face, err := opentype.NewFace(textInputFontSource(), &opentype.FaceOptions{
		Size:    fontPointSize, // Set font size here
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(fmt.Sprintf("creating mandatory text input font face: %v", err))
	}
	t.face = face
}

func (t *TextInput) GetText() string {
	return t.output.GetOutputStr(false)
}

func (t *TextInput) ClearText() {
	t.output.Clear()
}

func (t *TextInput) SetText(text string) {
	t.output.Clear()
	if text != "" {
		t.output.AddText(text)
	}
}

// CalculateTextWidth returns the total width of the string in pixels
func (t *TextInput) CalculateTextWidth(s string) int {
	// Use the same text measurement system as the actual rendering (UltimaFont)
	// This ensures the cursor position matches the actual text rendering
	if t.ultimaFont != nil && t.ultimaFont.TextFace != nil {
		// Use Ebitengine's text measurement which matches the rendering
		width, _ := etext.Measure(s, t.ultimaFont.TextFace, 0)

		return int(width)
	}

	// Fallback to the old method if UltimaFont is not available
	var width fixed.Int26_6
	for _, r := range s {
		awidth, _ := t.face.GlyphAdvance(r)
		width += awidth
	}

	return width.Ceil() // Convert from fixed.Int26_6 to int
}

func (t *TextInput) CalculateTextHeight(s string) int {
	if s == "" {
		return 0
	}
	rect, _, _ := t.face.GlyphBounds(rune(s[0]))

	return rect.Min.Y.Ceil()
}

// getFontHeight returns the height of the font in pixels
func (t *TextInput) getFontHeight() int {
	if t.face != nil {
		// Get the font metrics to calculate proper height
		metrics := t.face.Metrics()

		return (metrics.Height + metrics.Descent).Ceil()
	}
	// Fallback to a reasonable default if face is not available
	return 16
}
func (t *TextInput) ForceWaitForNextKeyPress() {
	t.keyboard.SetLastKeyPressedNowPlusMs(500)
	t.keyboard.ForceLastKeyPressed(ebiten.KeyEnter)
}

// GetAllCommandEntries returns all available command entries for display
// Returns nil if no text commands are configured
func (t *TextInput) GetAllCommandEntries() []grammar.CommandEntry {
	if t.textCommands == nil {
		return nil
	}

	return t.textCommands.GetAllCommandEntries()
}
