package widgets

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/grammar"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

var _ Widget = &InputBox{}

// NewYesNoCommand creates a yes/no grammar command with callbacks.
// This helper extracts the common pattern for yes/no dialogs.
//
// Parameters:
//   - onYes: Callback executed when user enters "yes" (can be nil)
//   - onNo: Callback executed when user enters "no" (can be nil)
//   - dialogStack: DialogStack for accessing and popping the input box
//
// Returns a configured TextCommand ready for use with InputBox.
//
// Example usage:
//
//	yesNoCmd := widgets.NewYesNoCommand(
//	    func() { executeYesAction() },
//	    func() { executeNoAction() },
//	    &g.dialogStack,
//	)
//	dialog := widgets.NewInputBox("Are you sure?", yesNoCmd, keyboard).
//	    WithAutoReplace(true).
//	    WithHorizontalPosition(widgets.DialogHorizontalMapOnly).
//	    Build()
//	g.dialogStack.PushModalDialog(dialog)
func NewYesNoCommand(onYes func(), onNo func(), dialogStack *DialogStack) *grammar.TextCommand {
	return grammar.NewTextCommand(
		[]grammar.Match{
			grammar.MatchStringList{
				Strings:              []string{"yes", "no"},
				Description:          "Yes or No",
				CaseSensitive:        false,
				SingleCharacterInput: true,
			},
		},
		func(_ string, _ *grammar.TextCommand) {
			ib := dialogStack.GetOrAssertTopInputBox()
			if ib == nil {
				return
			}
			response := strings.ToLower(ib.GetText())

			// Invoke onDismiss callback if available (handles auto-pop via AutoCloseAndPop)
			// This must happen BEFORE user callbacks to allow them to safely push new dialogs
			ib.InvokeOnDismiss()

			switch response {
			case "yes":
				if onYes != nil {
					onYes()
				}
			case "no":
				if onNo != nil {
					onNo()
				}
			}
		})
}

// InputBoxConfig defines the size and position of the input box dialog
type InputBoxConfig struct {
	StartPercentX float64 // 0.0-1.0 (left edge position)
	EndPercentX   float64 // 0.0-1.0 (right edge position)
	StartPercentY float64 // 0.0-1.0 (top edge position)
	EndPercentY   float64 // 0.0-1.0 (bottom edge position)
	TextLines     int     // Maximum question lines
	CharsPerLine  int     // Maximum characters per line
	AutoReplace   bool    // Enable single-character auto-replace mode

	// Regex mode support (mutually exclusive with grammar mode)
	RegexPattern     string                            // Regex validation pattern (empty = grammar mode)
	ResultCallback   func(text string, submitted bool) // Callback for regex mode (Enter/Escape)
	EscapeClearsText bool                              // Escape clears input instead of cancelling regex mode
	SpaceCancels     bool                              // Space cancels regex mode without submitting text
}

// InputBoxBuilder provides a fluent API for constructing InputBox instances.
// This replaces the multiple New* constructors with a single, extensible builder pattern.
//
// Example usage:
//
//	// Simple question dialog (backward compatible)
//	dialog := NewInputBox("Are you sure?", textCommand, keyboard).Build()
//
//	// Custom positioned dialog
//	:= NewInputBox("Choose wisely:", textCommand, keyboard).
//	    WithTextLines(2).
//	    WithCharsPerLine(40).
//	    Build()
type InputBoxBuilder struct {
	// Required fields
	question    string
	textCommand *grammar.TextCommand
	keyboard    *input.Keyboard

	// Optional fields with defaults
	// Position configuration
	centerX *float64 // Center point X percentage (nil = use preset)
	centerY *float64 // Center point Y percentage (nil = use preset)

	// Auto-sizing configuration
	textLines    *int // Number of question lines (nil = auto-calculate)
	charsPerLine *int // Max characters per line (nil = use preset default)

	// Single-character mode
	autoReplace bool // Enable auto-replace for single-character inputs

	// Regex mode support (mutually exclusive with textCommand grammar mode)
	regexPattern     *string                           // Regex validation pattern (nil = grammar mode)
	resultCallback   func(text string, submitted bool) // Callback for regex mode results
	escapeClearsText bool
	spaceCancels     bool

	// Dismiss callback
	onDismiss      func() // Optional: callback after resultCallback
	autoCloseStack *DialogStack

	// Custom override
	customConfig *InputBoxConfig // Full custom configuration (highest priority)
}

const (
	// InputBox uses FontSizeLarge (20pt) - maps to typography.FontSizeLarge
	inputBoxFontSize = FontSizeLarge // 20pt, using typography system
	// Line spacing uses 1:1 ratio (same as font size) - default in GetScaledFontWithOutput
)

type InputBox struct {
	textInput    *TextInput
	textQuestion *text.Output
	footerOutput *text.Output
	Question     string

	font *text.UltimaFont

	config InputBoxConfig
	layout inputBoxLayout

	// Regex mode support
	isRegexMode      bool
	regexPattern     string
	resultCallback   func(text string, submitted bool)
	escapeClearsText bool
	spaceCancels     bool
	keyboard         *input.Keyboard
	onDismiss        func() // Optional: callback after resultCallback
	dismissInvoked   bool
	lastWidth        int
	lastHeight       int
}

// NewInputBox creates a new InputBoxBuilder with sensible defaults.
// This is the preferred way to create InputBox instances going forward.
//
// Parameters:
//   - question: The question text to display
//   - textCommand: Grammar command for validation (can be nil if using WithYesNoCallbacks())
//   - keyboard: Input handler
//
// Default configuration:
//   - Position: Map-centered horizontally (37% X), bottom area (70% Y)
//   - Auto-replace: Disabled
//   - Question lines: 2 (default)
//   - Chars per line: 32 (standard InputBox width)
//   - Size: Auto-calculated from content requirements
//
// Example usage:
//
//	// Simple dialog with text command
//	dialog := NewInputBox("Are you sure?", textCommand, keyboard).Build()
//
//	// Yes/no dialog with callbacks (textCommand = nil)
//	dialog := NewInputBox("Are you sure?", nil, keyboard).
//	    WithYesNoCallbacks(onYes, onNo, &dialogStack).
//	    Build()
//
//	// Custom positioned dialog
//	:= NewInputBox("Choose wisely:", textCommand, keyboard).
//	    WithTextLines(2).
//	    Build()
func NewInputBox(question string, textCommand *grammar.TextCommand, keyboard *input.Keyboard) *InputBoxBuilder {
	return &InputBoxBuilder{
		question:    question,
		textCommand: textCommand,
		keyboard:    keyboard,
		autoReplace: false,
	}
}

// WithHorizontalPosition sets the horizontal center position.
// When not specified, defaults to DialogHorizontalMapOnly (37% - map-centered).
//
// Parameters:
//   - xPos: Horizontal positioning (DialogHorizontalMapOnly or DialogHorizontalWholeScreen)
//
// Example: WithHorizontalPosition(DialogHorizontalWholeScreen) centers at 50% X
func (b *InputBoxBuilder) WithHorizontalPosition(xPos DialogHorizontalPosition) *InputBoxBuilder {
	x := xPos.ToPercentage()
	b.centerX = &x

	return b
}

// WithVerticalPosition sets the vertical center position.
// When not specified, defaults to 0.70 (70% from top - standard InputBox bottom position).
//
// Parameters:
//   - y: Vertical center position percentage (0.0-1.0)
//
// Example: WithVerticalPosition(0.82) centers at 82% Y
func (b *InputBoxBuilder) WithVerticalPosition(y float64) *InputBoxBuilder {
	b.centerY = &y

	return b
}

// WithTextLines explicitly sets the maximum number of question text lines.
// This overrides auto-calculation from question text.
func (b *InputBoxBuilder) WithTextLines(lines int) *InputBoxBuilder {
	b.textLines = &lines

	return b
}

// WithCharsPerLine explicitly sets the maximum characters per line.
// This overrides the default value of 32 characters.
func (b *InputBoxBuilder) WithCharsPerLine(chars int) *InputBoxBuilder {
	b.charsPerLine = &chars

	return b
}

// WithAutoReplace enables auto-replace mode for single-character inputs.
// When enabled, typing a new character automatically replaces existing text.
// Ideal for yes/no prompts and direction selections.
func (b *InputBoxBuilder) WithAutoReplace(enabled bool) *InputBoxBuilder {
	b.autoReplace = enabled

	return b
}

// WithRegexValidation enables regex validation mode (alternative to grammar commands).
// This is mutually exclusive with grammar mode - Build() will panic if both are set.
//
// Parameters:
//   - pattern: Regular expression pattern for input validation
//
// Example: WithRegexValidation("^[a-zA-Z]*$") - letters only
func (b *InputBoxBuilder) WithRegexValidation(pattern string) *InputBoxBuilder {
	b.regexPattern = &pattern

	return b
}

// WithResultCallback sets the callback for regex mode (invoked on Enter/Escape).
// Required when using WithRegexValidation().
//
// Parameters:
//   - callback: Function called with (text, submitted) when user presses Enter or Escape
//   - text: The input text entered by the user
//   - submitted: true if Enter pressed, false if Escape pressed
//
// Example:
//
//	WithResultCallback(func(text string, submitted bool) {
//	    if submitted && text != "" {
//	        processInput(text)
//	    }
//	    dialogStack.PopModalDialog()
//	})
func (b *InputBoxBuilder) WithResultCallback(callback func(text string, submitted bool)) *InputBoxBuilder {
	b.resultCallback = callback

	return b
}

// WithEscapeClearsText keeps regex input active and clears its text when Escape is pressed.
func (b *InputBoxBuilder) WithEscapeClearsText() *InputBoxBuilder {
	b.escapeClearsText = true

	return b
}

// WithSpaceCancellation treats Space as a cancelled regex result.
func (b *InputBoxBuilder) WithSpaceCancellation() *InputBoxBuilder {
	b.spaceCancels = true

	return b
}

// WithDismissCallback sets a callback to execute after result callback.
// Provides maximum flexibility for custom cleanup logic.
// Callback executes AFTER resultCallback completes.
//
// Example usage:
//
//	dialog := widgets.NewInputBox("Enter name:", textCmd, keyboard).
//	    WithDismissCallback(func() {
//	        dialogStack.PopModalDialog()
//	    }).
//	    Build()
func (b *InputBoxBuilder) WithDismissCallback(callback func()) *InputBoxBuilder {
	b.onDismiss = callback

	return b
}

// AutoCloseAndPop is a convenience method that configures automatic dialog stack removal.
// Creates a dismiss callback that pops the dialog from the provided stack.
// Callback executes AFTER resultCallback completes.
//
// Parameters:
//   - dialogStack: Dialog stack reference for auto-pop (captured in closure)
//
// Example usage:
//
//	dialog := widgets.NewYesNoInputBox("Continue?", yesNoCmd, keyboard, dialogStack)
//	dialogStack.PushModalDialog(dialog)
func (b *InputBoxBuilder) AutoCloseAndPop(dialogStack *DialogStack) *InputBoxBuilder {
	if dialogStack == nil {
		panic("InputBox.AutoCloseAndPop: dialog stack is nil")
	}
	b.autoCloseStack = dialogStack

	return b
}

// WithYesNoCallbacks configures the InputBox as a yes/no prompt with callbacks.
// This is a convenience method that creates the yes/no grammar command internally
// and applies optimal defaults for yes/no prompts.
//
// Parameters:
//   - onYes: Callback executed when user enters "yes" (can be nil)
//   - onNo: Callback executed when user enters "no" (can be nil)
//   - dialogStack: DialogStack for accessing and popping the input box
//
// Automatically applies:
//   - AutoReplace mode (single-character 'y'/'n' input)
//   - TextLines = 2 (unless already set)
//
// Example usage:
//
//	dialog := widgets.NewInputBox("Are you sure?", nil, keyboard).
//	    WithYesNoCallbacks(
//	        func() { executeYesAction() },
//	        func() { executeNoAction() },
//	        &g.dialogStack,
//	    ).
//	    Build()
func (b *InputBoxBuilder) WithYesNoCallbacks(onYes func(), onNo func(), dialogStack *DialogStack) *InputBoxBuilder {
	// Create yes/no command internally
	b.textCommand = NewYesNoCommand(onYes, onNo, dialogStack)

	// Apply yes/no optimized defaults
	b.autoReplace = true

	// Set default question lines if not already specified
	if b.textLines == nil {
		lines := 2
		b.textLines = &lines
	}

	return b
}

// Build constructs the final InputBox instance.
// This method applies all configured options and returns a ready-to-use input dialog.
func (b *InputBoxBuilder) Build() *InputBox {
	// Mode validation - must be grammar OR regex, not both
	hasGrammarMode := b.textCommand != nil
	hasRegexMode := b.regexPattern != nil

	if hasGrammarMode && hasRegexMode {
		panic("InputBox cannot use both grammar commands and regex validation - choose one mode")
	}

	if !hasGrammarMode && !hasRegexMode {
		panic("InputBox requires either textCommand (grammar mode) or regex pattern (regex mode)")
	}

	// Regex mode requires callback
	if hasRegexMode && b.resultCallback == nil {
		panic("InputBox regex mode requires WithResultCallback() to handle Enter/Escape")
	}

	// Determine config to use
	var config InputBoxConfig
	if b.customConfig != nil {
		// Custom config takes highest priority
		config = *b.customConfig
	} else {
		// Auto-size from center point + content requirements
		// Apply defaults for unspecified coordinates
		centerX := DialogHorizontalMapOnly.ToPercentage() // Default X: map-centered (37%)
		if b.centerX != nil {
			centerX = *b.centerX
		}

		centerY := 0.70 // Default Y: 70% from top (standard InputBox bottom position)
		if b.centerY != nil {
			centerY = *b.centerY
		}

		textLines := 2     // Default
		charsPerLine := 32 // Default InputBox width

		if b.textLines != nil {
			textLines = *b.textLines
		}
		if b.charsPerLine != nil {
			charsPerLine = *b.charsPerLine
		}

		config = b.calculateAutoSizeDimensions(centerX, centerY, textLines, charsPerLine)
		config.AutoReplace = b.autoReplace

		// Populate regex mode fields if present
		if b.regexPattern != nil {
			config.RegexPattern = *b.regexPattern
		}
		if b.resultCallback != nil {
			config.ResultCallback = b.resultCallback
		}
		config.EscapeClearsText = b.escapeClearsText
		config.SpaceCancels = b.spaceCancels
	}

	// Create InputBox from config
	inputBox := createInputBoxFromConfig(b.question, b.textCommand, b.keyboard, config, b.onDismiss)
	if b.autoCloseStack != nil {
		if b.onDismiss != nil {
			panic("InputBox.Build: AutoCloseAndPop cannot be combined with WithDismissCallback")
		}
		inputBox.onDismiss = func() {
			b.autoCloseStack.MustRemoveWidgetInstance(inputBox)
		}
	}

	return inputBox
}

// calculateAutoSizeDimensions computes dialog dimensions from content requirements using shared function.
func (b *InputBoxBuilder) calculateAutoSizeDimensions(centerX, centerY float64, textLines, charsPerLine int) InputBoxConfig {
	// Use shared auto-sizing function (InputBox has no prompt lines)
	// FontSizeLarge (20pt) with 1:1 line spacing
	placement := CalculateDialogAutoSizeDimensions(NewDialogAutoSizingParams(
		centerX,
		centerY,
		textLines,
		charsPerLine,
		0, // InputBox has no separate prompt area
		float64(inputBoxFontSize),
		float64(inputBoxFontSize), // 1:1 line spacing
	))

	return InputBoxConfig{
		StartPercentX:    placement.StartPercentX,
		EndPercentX:      placement.EndPercentX,
		StartPercentY:    placement.StartPercentY,
		EndPercentY:      placement.EndPercentY,
		TextLines:        textLines,
		CharsPerLine:     charsPerLine,
		AutoReplace:      false, // Set by Build() method
		RegexPattern:     "",
		ResultCallback:   nil,
		EscapeClearsText: false,
		SpaceCancels:     false,
	}
}

// createInputBoxFromConfig constructs an InputBox from a configuration struct.
// This is the low-level constructor used by the builder's Build() method.
func createInputBoxFromConfig(question string, textCommand *grammar.TextCommand, keyboard *input.Keyboard, config InputBoxConfig, onDismiss func()) *InputBox {
	inputBox := &InputBox{}
	inputBox.font = GetScaledFont(inputBoxFontSize)

	inputBox.Question = question
	inputBox.keyboard = keyboard
	inputBox.onDismiss = onDismiss
	inputBox.config = config

	// Determine operating mode
	inputBox.isRegexMode = config.RegexPattern != ""
	inputBox.regexPattern = config.RegexPattern
	inputBox.resultCallback = config.ResultCallback
	inputBox.escapeClearsText = config.EscapeClearsText
	inputBox.spaceCancels = config.SpaceCancels

	// Create text commands for grammar mode (nil for regex mode)
	var textCommands *grammar.TextCommands
	if !inputBox.isRegexMode {
		textCommands = &grammar.TextCommands{
			*textCommand,
		}
	}

	inputBox.textInput = NewTextInput(
		sprites.PercentBasedPlacement{
			StartPercentX: 0,
			EndPercentX:   0,
			StartPercentY: 0,
			EndPercentY:   0,
		},
		GetScaledLineHeight(inputBoxFontSize), config.CharsPerLine,
		textCommands,
		TextInputCallbacks{
			AmbiguousAutoComplete: func(message string) {
			},
		},
		keyboard)
	// Set all input colors to light green for dialog input
	inputBox.textInput.SetInputColors(TextInputColors{
		DefaultColor:          color.DialogInputTextColor,
		NoMatchesColor:        color.DialogInputTextColor,
		OneMatchColor:         color.DialogInputTextColor,
		MoreThanOneMatchColor: color.DialogInputTextColor,
	})

	// Configure TextInput based on mode
	if inputBox.isRegexMode {
		// Regex mode: Use SetInputValidationRegex
		if err := inputBox.textInput.SetInputValidationRegex(config.RegexPattern); err != nil {
			panic("Invalid regex pattern: " + err.Error())
		}

		// Note: MaxCharacters is enforced by maxCharsPerLine parameter in NewTextInput constructor
		// Regex validation provides the primary input filtering

		inputBox.textInput.SetAutoReplaceMode(config.AutoReplace)
	} else {
		// Grammar mode: Existing logic
		if config.AutoReplace {
			inputBox.textInput.SetAutoReplaceMode(true)
		}
	}

	screenWidth, screenHeight := display.GetManager().GetCurrentSize()
	inputBox.ensureLayout(screenWidth, screenHeight)

	return inputBox
}

func (i *InputBox) Draw(screen *ebiten.Image) {
	if screen == nil {
		return
	}
	i.ensureLayout(screen.Bounds().Dx(), screen.Bounds().Dy())
	i.drawPromptCard(screen)
}

func (i *InputBox) Update() {
	screenWidth, screenHeight := display.GetManager().GetCurrentSize()
	i.ensureLayout(screenWidth, screenHeight)
	if i.isRegexMode {
		// Regex mode: Handle Enter/Escape at InputBox level (like TextInputDialog)
		var text string
		var submitted bool

		if i.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) ||
			i.keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter) {
			text = i.textInput.GetText()
			submitted = true
		} else if i.spaceCancels && i.keyboard.IsKeyPressedAndRegister(ebiten.KeySpace) {
			text = ""
			submitted = false
		} else if i.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
			if i.escapeClearsText {
				i.textInput.Clear()

				return
			}
			text = ""
			submitted = false
		} else {
			// Let TextInput handle character input
			i.textInput.Update()

			return
		}

		// Key was pressed - invoke callbacks
		i.resultCallback(text, submitted)

		// Invoke dismiss callback AFTER result callback
		i.InvokeOnDismiss()

		return
	} else {
		// Grammar mode: Existing logic - TextInput handles everything
		i.textInput.Update()
	}
}

func (i *InputBox) GetText() string {
	return i.textInput.GetText()
}

// InvokeOnDismiss calls the onDismiss callback if it's set.
// This is used by grammar commands to trigger auto-pop behavior via AutoCloseAndPop.
func (i *InputBox) InvokeOnDismiss() {
	if i.dismissInvoked {
		return
	}
	i.dismissInvoked = true
	if i.onDismiss != nil {
		i.onDismiss()
	}
}
