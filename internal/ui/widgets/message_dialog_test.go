package widgets

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageDialogInputDismissalReservesSharedFooterForTinyCapacity(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	dialog := NewMessageDialog("OK").WithEffects(false).
		WithTextLines(1).WithCharsPerLine(2).WithInputDismissal(newWidgetTestKeyboard(0)).Build()
	require.True(t, dialog.hasPromptOutput())
	require.Equal(t, DialogCloseHint, dialog.promptOutput.GetOutputStr(false))
	require.Equal(t, DialogControlHintColor, dialog.promptColor)
	require.GreaterOrEqual(t, dialog.config.TextLines, 2)
	require.Equal(t, "OK", dialog.output.GetOutputStr(false))
}

func TestMessageDialogWorkflowAcknowledgementDoesNotOwnInput(t *testing.T) {
	dialog := NewMessageDialog("The shrine rewards thy quest.").WithEffects(false).
		WithAutoSizeDefault().WithSeparatePromptArea().Build()
	require.Empty(t, dialog.promptOutput.GetOutputStr(false), "no guidance while effects are running")
	dialog.SetAcknowledgementPrompt(DialogAnyKeyHint)
	require.Equal(t, DialogAnyKeyHint, dialog.promptOutput.GetOutputStr(false))
	require.Equal(t, DialogControlHintColor, dialog.promptColor)
	require.False(t, dialog.enableInputDismissal, "the workflow still owns the any-key wait")
	dialog.ClearPrompt()
	require.Empty(t, dialog.promptOutput.GetOutputStr(false))
}

// Builder Pattern Tests

// TestMessageDialogBuilder_Defaults verifies default configuration
func TestMessageDialogBuilder_Defaults(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1920, 1080)

	dialog := NewMessageDialog("Test message").Build()

	require.NotNil(t, dialog, "Dialog should be created")
	assert.False(t, dialog.IsVisible(), "Dialog should start invisible")
	assert.Equal(t, MessageDialogFadeInMs, dialog.GetFadeInDuration(), "Default fade-in duration")
	assert.Equal(t, MessageDialogFadeOutMs, dialog.GetFadeOutDuration(), "Default fade-out duration")
	assert.Equal(t, ContentAlignBottomUp, dialog.config.ContentAlignment)
	assert.True(t, dialog.enableDissolveEffects)
	assert.False(t, dialog.enableDualOutput)
	assert.False(t, dialog.enableInputDismissal)
	assert.Zero(t, dialog.config.PromptLines)
	assert.Equal(t, "Test message", dialog.output.GetOutputStr(false))
	assert.False(t, dialog.IsDismissed())
	dialog.ShowWithoutAnimation()
	assert.True(t, dialog.IsVisible())
	dialog.HideWithoutAnimation()
	assert.False(t, dialog.IsVisible())
	dialog.Clear()
	assert.Empty(t, dialog.output.GetOutputStr(false))
	dialog.AppendLine("Line 1")
	assert.Equal(t, "Line 1", dialog.output.GetOutputStr(false))
}

func TestMessageDialogUsesResponsiveClassicWhiteFrame(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	dialog := NewMessageDialog("A message.").
		WithEffects(false).
		WithAutoSizeDefault().
		Build()
	dialog.ShowWithoutAnimation()
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Dispose)
	dialog.Draw(screen)
	requireClassicFrameSilhouette(t, screen, dialog.frameBounds)

	setResponsiveDialogTestResolution(t, 1920, 1080)
	largeScreen := ebiten.NewImage(1920, 1080)
	t.Cleanup(largeScreen.Dispose)
	dialog.Draw(largeScreen)

	require.Equal(t, 1920, dialog.lastScreenWidth)
	require.Equal(t, 1080, dialog.lastScreenHeight)
	requireClassicFrameSilhouette(t, largeScreen, dialog.frameBounds)
}

func TestActionFeedbackDialogUsesRequestedPosition(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1920, 1080)

	dialog := NewActionFeedbackDialog("A pit!", PositionMapCenterBottom)

	assert.InDelta(t, PositionMapCenterBottom.GetHorizontalPercent(),
		(dialog.config.StartPercentX+dialog.config.EndPercentX)/2, 0.0001)
	assert.InDelta(t, PositionMapCenterBottom.GetVerticalPercent(),
		(dialog.config.StartPercentY+dialog.config.EndPercentY)/2, 0.0001)
	assert.LessOrEqual(t, dialog.config.EndPercentY, 0.96)
}

func TestSearchFeedbackDialogRetainsCompleteResultWithoutChangingActionDialog(t *testing.T) {
	const message = "In the barrel, thou dost find nothing of note."
	search := NewSearchFeedbackDialog(message, PositionMapCenterLower)
	action := NewActionFeedbackDialog("Unlocked!", PositionMapCenterLower)

	assert.Equal(t, 1, search.config.TextLines)
	assert.Equal(t, len(message), search.config.CharsPerLine)
	assert.Equal(t, message, search.output.GetOutputStr(false))
	assert.Equal(t, actionFeedbackTextLines, action.config.TextLines)
	assert.Equal(t, actionFeedbackCharsPerLine, action.config.CharsPerLine)
}

func TestSearchFeedbackDialogRetainsGenericExactWidthResult(t *testing.T) {
	const message = "Thou dost find nothing of note."
	search := NewSearchFeedbackDialog(message, PositionMapCenterLower)

	assert.Equal(t, message, search.output.GetOutputStr(false))
}

// TestMessageDialogBuilder_FeatureToggles verifies feature toggles work correctly
func TestMessageDialogBuilder_FeatureToggles(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1920, 1080)

	// Test with effects disabled
	dialogNoEffects := NewMessageDialog("Test").
		WithEffects(false).
		Build()
	assert.False(t, dialogNoEffects.enableDissolveEffects)
	assert.Nil(t, dialogNoEffects.dissolveEffect)

	// Test with dual output disabled (default behavior)
	dialogNoDual := NewMessageDialog("Test").
		Build()
	assert.False(t, dialogNoDual.enableDualOutput)
	assert.Zero(t, dialogNoDual.config.PromptLines)
	dialogPrompt := NewMessageDialog("").WithEffects(false).WithSeparatePromptArea().
		WithTextLines(4).WithCharsPerLine(50).Build()
	assert.True(t, dialogPrompt.enableDualOutput)
	assert.Equal(t, 4, dialogPrompt.config.TextLines)
	assert.Equal(t, 50, dialogPrompt.config.CharsPerLine)
	assert.Equal(t, 1, dialogPrompt.config.PromptLines)
	assert.Empty(t, dialogPrompt.output.GetOutputStr(false))
	dialogPrompt.SetPrompt("Continue")
	assert.Equal(t, "Continue", dialogPrompt.promptOutput.GetOutputStr(false))

	// Test with custom fade durations
	customFadeIn := 1500
	customFadeOut := 500
	dialogCustomFade := NewMessageDialog("Test").
		WithFadeDurations(customFadeIn, customFadeOut).
		Build()
	require.NotNil(t, dialogCustomFade, "Dialog with custom fade should be created")
	assert.Equal(t, customFadeIn, dialogCustomFade.GetFadeInDuration(), "Custom fade-in duration")
	assert.Equal(t, customFadeOut, dialogCustomFade.GetFadeOutDuration(), "Custom fade-out duration")
}

// TestMessageDialogBuilder_CenteredAlignment verifies centered alignment mode
func TestMessageDialogBuilder_CenteredAlignment(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1920, 1080)

	for _, alignment := range []ContentAlignment{ContentAlignCenter, ContentAlignBottomUp} {
		dialog := NewMessageDialog("Test").WithContentTextAlignment(alignment).Build()
		assert.Equal(t, alignment, dialog.config.ContentAlignment)
	}
}

// TestMessageDialogBuilder_AutoSize verifies auto-sizing calculates correct TextLines
func TestMessageDialogBuilder_AutoSize(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1920, 1080)

	tests := []struct {
		name          string
		message       string
		expectedLines int
	}{
		{
			name:          "Two-line death message",
			message:       "An unending darkness\nengulfs thee...",
			expectedLines: 2,
		},
		{
			name:          "Three-line synthetic report",
			message:       "Synthetic first line\nsynthetic second line\nsynthetic third line",
			expectedLines: 3,
		},
		{
			name:          "Four-line with empty line",
			message:       "Someone shouts\n\n\"FORTIS FORTUNA\nAVENTARI\"",
			expectedLines: 4,
		},
		{
			name:          "Single line",
			message:       "Vertigo...",
			expectedLines: 1,
		},
		{
			name:          "Short two-line message (strangeWordsMsg)",
			message:       "Strange words\nare intoned.",
			expectedLines: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialog := NewMessageDialog(tt.message).
				WithAutoSizeDefault().
				WithEffects(false).
				Build()

			require.NotNil(t, dialog, "Dialog should be created")

			t.Logf("Message: %q", tt.message)
			t.Logf("  Config TextLines: %d (expected: %d)", dialog.config.TextLines, tt.expectedLines)
			t.Logf("  Config CharsPerLine: %d", dialog.config.CharsPerLine)

			assert.GreaterOrEqual(t, dialog.config.TextLines, tt.expectedLines,
				"Dialog should have at least %d lines for message %q, got %d",
				tt.expectedLines, tt.message, dialog.config.TextLines)
		})
	}
}
