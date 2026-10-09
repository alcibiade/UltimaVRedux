package widgets

import (
	"strconv"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/grammar"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

// Shared dialog sizing constants are defined in dialog_constants.go.

// NewYellDialog creates InputBox for yell command (letters only, 10 char max).
// This is a convenience constructor for the Yell command dialog.
//
// Features:
//   - Letter-only validation (a-z, A-Z)
//   - 10 character maximum
//   - Manual replace mode (no auto-submit)
//   - Map-centered positioning
//
// Parameters:
//   - keyboard: Input handler
//   - callback: Function called with (text, submitted) when dialog closes
//
// Example usage:
//
//	yellDialog := widgets.NewYellDialog(keyboard, func(text string, submitted bool) {
//	    if submitted {
//	        gameState.ActionYell(text)
//	    }
//	    dialogStack.PopModalDialog()
//	})
//	dialogStack.PushModalDialog(yellDialog)
func NewYellDialog(keyboard *input.Keyboard, callback func(text string, submitted bool)) *InputBox {
	return NewInputBox("What dost thou yell?", nil, keyboard).
		WithRegexValidation("^[a-zA-Z]{0,10}$").
		WithResultCallback(callback).
		WithAutoReplace(false).
		WithTextLines(DialogConfirmationLines).
		WithCharsPerLine(dialogDefaultCharsPerLine).
		WithHorizontalPosition(DialogHorizontalMapOnly).
		WithVerticalPosition(DialogDefaultVerticalPos).
		Build()
}

// NewSleepDurationDialog creates InputBox for hour entry (1-9 only).
// This is a convenience constructor for sleep duration dialogs (inn, bed, etc.).
//
// Features:
//   - Numeric validation (1-9 only)
//   - Single digit entry with auto-replace
//   - Callback receives parsed integer hours
//   - Map-centered positioning
//
// Parameters:
//   - keyboard: Input handler
//   - callback: Function called with parsed hours (0 if cancelled or invalid)
//
// Example usage:
//
//	sleepDialog := widgets.NewSleepDurationDialog(keyboard, func(hours int) {
//	    if hours > 0 {
//	        performSleep(hours)
//	    }
//	    dialogStack.PopModalDialog()
//	})
//	dialogStack.PushModalDialog(sleepDialog)
func NewSleepDurationDialog(keyboard *input.Keyboard, callback func(hours int)) *InputBox {
	const sleepDurationCharsPerLine = 20 // Narrow dialog for single digit entry

	return NewInputBox("How many hours?", nil, keyboard).
		WithRegexValidation("^[1-9]$").
		WithResultCallback(func(text string, submitted bool) {
			if !submitted {
				callback(0)

				return
			}

			if hours, err := strconv.Atoi(strings.TrimSpace(text)); err == nil {
				callback(hours)
			} else {
				callback(0)
			}
		}).
		WithAutoReplace(true).
		WithTextLines(DialogConfirmationLines).
		WithCharsPerLine(sleepDurationCharsPerLine).
		WithHorizontalPosition(DialogHorizontalMapOnly).
		WithVerticalPosition(DialogDefaultVerticalPos).
		Build()
}

// NewYesNoInputBox creates InputBox for yes/no confirmation prompts.
// This is a convenience constructor for yes/no dialogs with auto-replace.
//
// Features:
//   - Auto-replace enabled (single key press submits)
//   - Auto-close and pop from dialog stack after result callback
//   - Natural wrapping with capacity expanded to fit the complete question
//   - Map-centered positioning
//   - Grammar command for yes/no validation
//
// Parameters:
//   - question: The confirmation question to display
//   - yesNoCommand: Grammar command handling yes/no responses
//   - keyboard: Input handler
//   - dialogStack: Dialog stack to pop from after result callback
//
// Example usage:
//
//	yesNoCommand := widgets.NewYesNoCommand(
//	    func() { performAction() },
//	    func() { /* cancel */ },
//	    &dialogStack,
//	)
//	dialog := widgets.NewYesNoInputBox("Dost thou wish to leave?", yesNoCommand, keyboard, &dialogStack)
//	dialogStack.PushModalDialog(dialog)
func NewYesNoInputBox(question string, yesNoCommand *grammar.TextCommand, keyboard *input.Keyboard, dialogStack *DialogStack) *InputBox {
	questionLines := text.MeasureText(
		question,
		GetScaledFont(inputBoxFontSize),
		GetScaledLineHeight(inputBoxFontSize),
		dialogConfirmationCharsPerLine,
	).Lines

	return NewInputBox(question, yesNoCommand, keyboard).
		WithAutoReplace(true).
		AutoCloseAndPop(dialogStack).
		WithTextLines(max(DialogConfirmationLines, questionLines)).
		WithCharsPerLine(dialogConfirmationCharsPerLine).
		WithHorizontalPosition(DialogHorizontalMapOnly).
		WithVerticalPosition(DialogDefaultVerticalPos).
		Build()
}
