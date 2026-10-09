package main

import (
	"image/color"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

// MessageDialog helper methods for Event Queue use

func (g *GameScene) appendToCurrentRowStr(str string) {
	g.output.AppendText(str)
}

func (g *GameScene) addRowStr(str string) {
	// PRIMARY: Always write to output pane (authentic Ultima V record)
	g.output.AddText(str)

	// SECONDARY: Mirror to MessageDialog if visible AND accepts mirrored text
	// Skip mirroring for static content dialogs (like action feedback popups)
	if g.dialogStack.HasOpenDialog() {
		topDialog := g.dialogStack.PeekTopModalDialog()
		if messageDialog, ok := topDialog.(*widgets.MessageDialog); ok && messageDialog.IsVisible() {
			if !messageDialog.IsStaticContent() {
				messageDialog.AppendLine(str)
			}
		}
	}
}

// showCommandPrompt starts an input line with the original game's arrow marker.
func (g *GameScene) showCommandPrompt(command string) {
	g.output.AddCommandPrompt(command)
}

func (g *GameScene) appendStyledMessage(message game_state.StyledMessage) {
	builder := g.appendLineBuilderDualOutput()
	if message.LeadingBlank {
		builder.NewLine()
	}
	for _, segment := range message.Segments {
		switch segment.Style {
		case game_state.MessageSpoken:
			builder.Spoken(segment.Text)
		case game_state.MessageQuotation:
			builder.LightGray(segment.Text)
		case game_state.MessageHighlight:
			builder.LightGreen(segment.Text)
		default:
			panic("unsupported message segment style")
		}
	}
	builder.Done()
}

// appendLineBuilderDualOutput creates a LineBuilder that writes to BOTH MessageDialog and output pane
// - Colored text appears in MessageDialog (modern readability)
// - Plain text appears in output pane (authentic Ultima V record)
func (g *GameScene) appendLineBuilderDualOutput() *text.LineBuilder {
	// Get top MessageDialog from stack
	if !g.dialogStack.HasOpenDialog() {
		// No dialog - return a builder that only writes to output
		return g.output.AppendLine()
	}

	topDialog := g.dialogStack.PeekTopModalDialog()
	messageDialog, ok := topDialog.(*widgets.MessageDialog)
	if !ok {
		// Top dialog is not MessageDialog - return output-only builder
		return g.output.AppendLine()
	}

	builder := messageDialog.AppendLineBuilder()

	// Set callback to also write plain text to output pane
	builder.SetPlainTextCallback(func(plainText string) {
		g.output.AddText(plainText)
	})

	return builder
}

// createAndPushMessageDialog creates a new MessageDialog and pushes it onto the dialog stack.
func (g *GameScene) createAndPushMessageDialog() {
	// Create new MessageDialog with standard camping configuration
	messageDialog := widgets.NewCampingMessageDialog()

	// Push onto dialog stack
	g.dialogStack.PushModalDialog(messageDialog)

	// Trigger fade-in animation with dissolve effect (duration set by SetMessageDialogFadeDurations)
	messageDialog.Show()
}

func (g *GameScene) createAndPushCampingResultDialog(message string) {
	dialog := widgets.NewCampingResultDialog(message)
	g.dialogStack.PushModalDialog(dialog)
	dialog.Show()
}

// popMessageDialog removes MessageDialog from stack when done
// Uses PeekTopModalDialog to get explicit reference before removal
func (g *GameScene) popMessageDialog() {
	if g.dialogStack.HasOpenDialog() {
		topDialog := g.dialogStack.PeekTopModalDialog()
		g.dialogStack.RemoveWidgetInstance(topDialog)
	}
}

// hideMessageDialog triggers MessageDialog fade-out animation without removing from stack
// Pattern: Call Hide() to start animation → Wait(GetMessageDialogFadeOutDuration()) → PopMessageDialog()
func (g *GameScene) hideMessageDialog() {
	if !g.dialogStack.HasOpenDialog() {
		return
	}
	topDialog := g.dialogStack.PeekTopModalDialog()
	if messageDialog, ok := topDialog.(*widgets.MessageDialog); ok {
		messageDialog.Hide() // Starts fade-out animation (dialog remains in stack for Update() calls)
	}
}

// CONTENT MANIPULATION METHODS - Modify dialog content and query properties (work on top dialog in stack)

func (g *GameScene) clearMessageDialog() {
	if !g.dialogStack.HasOpenDialog() {
		return
	}
	topDialog := g.dialogStack.PeekTopModalDialog()
	if messageDialog, ok := topDialog.(*widgets.MessageDialog); ok {
		messageDialog.Clear()
	}
}

func (g *GameScene) setMessagePrompt(prompt string) {
	if !g.dialogStack.HasOpenDialog() {
		return
	}
	topDialog := g.dialogStack.PeekTopModalDialog()
	if messageDialog, ok := topDialog.(*widgets.MessageDialog); ok {
		messageDialog.SetPrompt(prompt)
	}
}

func (g *GameScene) setMessagePromptWithColor(prompt string, promptColor color.Color) {
	if !g.dialogStack.HasOpenDialog() {
		return
	}
	topDialog := g.dialogStack.PeekTopModalDialog()
	if messageDialog, ok := topDialog.(*widgets.MessageDialog); ok {
		messageDialog.SetPromptWithColor(prompt, promptColor)
	}
}

func (g *GameScene) clearMessagePrompt() {
	if !g.dialogStack.HasOpenDialog() {
		return
	}
	topDialog := g.dialogStack.PeekTopModalDialog()
	if messageDialog, ok := topDialog.(*widgets.MessageDialog); ok {
		messageDialog.ClearPrompt()
	}
}

func (g *GameScene) showPressAnyKeyPrompt() {
	if !g.dialogStack.HasOpenDialog() {
		return
	}
	topDialog := g.dialogStack.PeekTopModalDialog()
	if messageDialog, ok := topDialog.(*widgets.MessageDialog); ok {
		messageDialog.SetAcknowledgementPrompt(widgets.DialogAnyKeyHint)
	}
}

func (g *GameScene) setMessageDialogFadeDurations(fadeInMs, fadeOutMs int) {
	if !g.dialogStack.HasOpenDialog() {
		return
	}
	topDialog := g.dialogStack.PeekTopModalDialog()
	if messageDialog, ok := topDialog.(*widgets.MessageDialog); ok {
		messageDialog.SetFadeDurations(fadeInMs, fadeOutMs)
	}
}

func (g *GameScene) getMessageDialogFadeInDuration() int {
	if !g.dialogStack.HasOpenDialog() {
		return 300 // Default duration
	}
	topDialog := g.dialogStack.PeekTopModalDialog()
	if messageDialog, ok := topDialog.(*widgets.MessageDialog); ok {
		return messageDialog.GetFadeInDuration()
	}

	return 300
}

func (g *GameScene) getMessageDialogFadeOutDuration() int {
	if !g.dialogStack.HasOpenDialog() {
		return 200 // Default duration
	}
	topDialog := g.dialogStack.PeekTopModalDialog()
	if messageDialog, ok := topDialog.(*widgets.MessageDialog); ok {
		return messageDialog.GetFadeOutDuration()
	}

	return 200
}

// Screen callback implementations for SystemCallbacks integration

func (g *GameScene) updateStatsDisplay() {
	if g.characterSummary != nil {
		g.characterSummary.UpdateStatsDisplay()
	}
}

func (g *GameScene) refreshInventoryDisplay() {
	if g.provisionSummary != nil {
		g.provisionSummary.MarkInventoryChanged()
	}
}
