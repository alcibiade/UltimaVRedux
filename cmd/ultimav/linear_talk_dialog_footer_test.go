package main

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/stretchr/testify/require"
)

func newTalkFooterTestDialog() *LinearTalkDialog {
	dialog := &LinearTalkDialog{gameScene: &GameScene{gameState: &game_state.GameState{}, keyboard: newPresentationTestKeyboard()}, waitingForInput: true}
	dialog.initializeResizeableVisualElements()
	dialog.addInputPromptForStyle("Your interest?", linearTalkPresentationContinuousMarker)
	dialog.TextInput.SetText("FLAM")

	return dialog
}

func TestTalkFooterIsConnectedBelowUnchangedInlineInput(t *testing.T) {
	setLinearTalkDialogTestResolution(t, 1280, 720)
	dialog := newTalkFooterTestDialog()
	screen := newPresentationRaster(1280, 720)
	t.Cleanup(screen.Deallocate)
	base := calculateLinearTalkDialogLayout(screen.Bounds())
	layout := dialog.layoutWithFooter(screen.Bounds())
	before := dialog.Output.GetOutputStr(true)
	dialog.Draw(screen)
	require.Equal(t, base.continuousTranscriptRect.Size(), layout.continuousTranscriptRect.Size(), "footer must not consume transcript space")
	inputRect := dialog.continuousInputRect(layout, activeLinearTalkPresentation)
	require.Equal(t, layout.continuousTranscriptRect.Max.Y, inputRect.Max.Y)
	require.Less(t, inputRect.Max.Y, layout.footerDividerY)
	require.Greater(t, layout.footerRect.Min.Y, layout.footerDividerY)
	require.True(t, layout.footerRect.In(layout.frameRect))
	require.True(t, layout.frameRect.In(screen.Bounds()))
	requireLinearTalkDialogPixel(t, screen,
		image.Pt(layout.frameRect.Min.X+layout.frameRect.Dx()/2, layout.footerDividerY), widgets.DialogRuleColor)
	requireLinearTalkDialogPixel(t, screen,
		image.Pt(layout.frameRect.Min.X+layout.frameRect.Dx()/2, layout.frameRect.Min.Y-1), color.Transparent)
	requireLinearTalkDialogRegionContainsColor(t, screen, layout.footerRect, widgets.DialogControlKeyColor)
	requireLinearTalkDialogRegionContainsColor(t, screen, layout.footerRect, widgets.DialogControlHintColor)
	require.Equal(t, before, dialog.Output.GetOutputStr(true), "hints must not enter speech or its retained history")
	require.Equal(t, "FLAM", dialog.TextInput.GetText())
	require.NotContains(t, interestPlaceholder, "|")
	require.Equal(t, " name, job, bye", interestPlaceholder)
	require.Contains(t, dialog.footerOutput.GetOutputStr(false), "History")
}

func TestTalkFooterStatesFitAndDoNotMoveInlineInput(t *testing.T) {
	for _, resolution := range append(config.SupportedWindowResolutions(), config.ScreenResolution{X: 640, Y: 400}, config.ScreenResolution{X: 320, Y: 200}) {
		setTalkHistoryTestResolution(t, resolution.X, resolution.Y)
		dialog := newTalkFooterTestDialog()
		require.Same(t, dialog.font, dialog.footerOutput.Font, "footer must use the conversation font")
		require.Equal(t, dialog.Output.GetFontHeight(), dialog.footerOutput.GetFontHeight())
		require.Equal(t, dialog.Output.GetLineSpacing(), dialog.footerOutput.GetLineSpacing())
		bounds := image.Rect(0, 0, resolution.X, resolution.Y)
		layout := dialog.layoutWithFooter(bounds)
		require.True(t, layout.frameRect.In(bounds))
		require.Equal(t, calculateLinearTalkDialogLayout(bounds).continuousTranscriptRect.Size(), layout.continuousTranscriptRect.Size())
		transcript := dialog.Output.GetOutputStr(true)
		for _, state := range []string{"input", "continue", "close", "kept", "history", "failure"} {
			dialog.waitingForKeypressContinue = state == "continue"
			dialog.waitingForDismissal = state == "close"
			dialog.historyPanel = nil
			dialog.setExcerptNotice("", false)
			switch state {
			case "kept":
				dialog.setExcerptNotice("Excerpt kept", true)
			case "history":
				dialog.historyPanel = dialog.Output.RetainedHistoryPanel()
			case "failure":
				dialog.setExcerptNotice("Unable to keep excerpt: "+strings.Repeat("long-path/", 30), false)
			}
			require.Equal(t, layout, dialog.layoutWithFooter(bounds))
			widgets.PrepareKeyboardFooter(layout.footerRect, dialog.conversationFooterGuidance(layout.footerRect.Dx()),
				talkFooterLines, dialog.footerOutput)
			require.Equal(t, transcript, dialog.Output.GetOutputStr(true), "preparing hints must not change speech or retained history")
			require.Equal(t, "FLAM", dialog.TextInput.GetText())
			for _, line := range strings.Split(dialog.footerOutput.GetOutputStr(false), "\n") {
				measured, _ := dialog.footerOutput.Font.MeasureString(strings.TrimSpace(line))
				require.LessOrEqual(t, measured, float64(layout.footerRect.Dx()), "%dx%d %s: %s", resolution.X, resolution.Y, state, line)
			}
			require.LessOrEqual(t, dialog.footerOutput.GetPanelContentLineCount(), talkFooterLines)
			guidance := dialog.conversationFooterGuidance(layout.footerRect.Dx())
			switch state {
			case "continue":
				require.Contains(t, guidance, "[Space] Continue")
			case "close":
				require.Contains(t, guidance, "[Space] Close")
			case "kept":
				require.Contains(t, guidance, "[Tab] Excerpt kept")
			case "failure":
				require.Contains(t, guidance, "...")
				require.Contains(t, guidance, "[Tab] Retry")
			}
		}
	}
}

func TestTalkCaptureConfirmationExpiresWithoutChangingConversation(t *testing.T) {
	setTalkHistoryTestResolution(t, 1280, 720)
	dialog := newTalkFooterTestDialog()
	before := dialog.Output.GetOutputStr(true)
	dialog.setExcerptNotice("Excerpt kept", true)
	dialog.gameScene.gameState.ElapsedMs = excerptNoticeDurationMs - 1
	dialog.expireExcerptNotice()
	require.Equal(t, "Excerpt kept", dialog.excerptNotice)
	dialog.gameScene.gameState.ElapsedMs++
	dialog.expireExcerptNotice()
	require.Empty(t, dialog.excerptNotice)
	require.Equal(t, before, dialog.Output.GetOutputStr(true))
	require.Equal(t, "FLAM", dialog.TextInput.GetText())
	dialog.setExcerptNotice("Unable to keep excerpt", false)
	dialog.gameScene.gameState.ElapsedMs += excerptNoticeDurationMs
	dialog.expireExcerptNotice()
	require.Equal(t, "Unable to keep excerpt", dialog.excerptNotice, "a failure should remain readable")
}
