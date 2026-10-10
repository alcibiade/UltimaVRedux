package widgets

import (
	"image"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestMessageDialogPopupLayoutBuildAndVisibleChanges(t *testing.T) {
	const message = "Thou dost find nothing of note."
	tests := []struct {
		name  string
		build func() *MessageDialog
	}{
		{"direct", func() *MessageDialog {
			return NewMessageDialog(message).WithEffects(false).
				WithTextLines(2).WithCharsPerLine(45).WithVerticalPosition(0.5).Build()
		}},
		{"action", func() *MessageDialog {
			return NewActionFeedbackDialog(message, PositionMapCenterBottom)
		}},
		{"search", func() *MessageDialog {
			return NewSearchFeedbackDialog(message, PositionMapCenterLower)
		}},
		{"confirmation", func() *MessageDialog {
			return NewOverlayDialog(message).AsConfirmation().WithUserDismissed(newWidgetTestKeyboard(0)).Build().dialog
		}},
		{"brief feedback", func() *MessageDialog {
			return NewOverlayDialog(message).AsBriefFeedback().Build().dialog
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setResponsiveDialogTestResolution(t, 1280, 720)
			viewport := image.Rect(320, 20, 640, 420)
			setMessageDialogTestPopupViewport(t, func(image.Rectangle) image.Rectangle { return viewport })
			dialog := tt.build()

			require.False(t, dialog.IsVisible(), "Build must resolve layout before show or stack push")
			requireMessageDialogPopupBounds(t, dialog, viewport, 1280)
			require.NotNil(t, dialog.renderOutput)
			assert.Equal(t, strings.Fields(message), strings.Fields(dialog.renderOutput.GetOutputStr(false)))
			if dialog.hasPromptOutput() {
				require.NotNil(t, dialog.renderPrompt)
				assert.Equal(t, strings.Fields(dialog.promptOutput.GetOutputStr(false)),
					strings.Fields(dialog.renderPrompt.GetOutputStr(false)), "Build must also resolve the confirmation prompt")
			}
			source := dialog.output.GetOutputStr(false)
			base := dialog.baseConfig
			dialog.ShowWithoutAnimation()
			dialog.resolvePopupLayout(image.Rect(0, 0, 1280, 720))
			oldBounds := dialog.frameBounds

			// A map/view-mode change must invalidate layout even at the same resolution.
			viewport = image.Rect(480, 40, 760, 400)
			dialog.resolvePopupLayout(image.Rect(0, 0, 1280, 720))
			requireMessageDialogPopupBounds(t, dialog, viewport, 1280)
			assert.NotEqual(t, oldBounds, dialog.frameBounds)

			setResponsiveDialogTestResolution(t, 1920, 1080)
			viewport = image.Rect(180, 60, 780, 600)
			dialog.resolvePopupLayout(image.Rect(0, 0, 1920, 1080))
			requireMessageDialogPopupBounds(t, dialog, viewport, 1920)
			assert.Equal(t, 1920, dialog.lastScreenWidth)
			assert.Equal(t, 1080, dialog.lastScreenHeight)
			assert.True(t, dialog.IsVisible(), "resize must retain the visible instance")
			assert.False(t, dialog.IsDismissed())
			assert.Equal(t, base, dialog.baseConfig)
			assert.Equal(t, source, dialog.output.GetOutputStr(false))
			assert.Equal(t, strings.Fields(message), strings.Fields(dialog.renderOutput.GetOutputStr(false)))

			viewport = image.Rectangle{}
			dialog.resolvePopupLayout(image.Rect(0, 0, 1920, 1080))
			assert.Equal(t, base, dialog.config, "empty provider must restore legacy placement")
			assert.Same(t, dialog.output, dialog.renderOutput)
		})
	}
}

func TestActionFeedbackRetainsCompleteSourceThroughReflow(t *testing.T) {
	for _, message := range []string{
		"Thou dost see the collapsed entrance to the dungeon Deceit",
		"Thou dost see\n\nthe collapsed entrance\n\nto the dungeon Deceit",
		"Thou dost see the caf\u00e9 beyond the entr\u00e9e\n\nThe fa\u00e7ade conceals a cr\u00e8me coloured door\nThe derni\u00e8re rune marks the dungeon Deceit",
	} {
		t.Run(message, func(t *testing.T) {
			setResponsiveDialogTestResolution(t, 1280, 720)
			viewport := image.Rect(320, 20, 640, 420)
			setMessageDialogTestPopupViewport(t, func(image.Rectangle) image.Rectangle { return viewport })
			dialog := NewActionFeedbackDialog(message, PositionMapCenterLower)
			require.True(t, utf8.ValidString(dialog.output.GetOutputStr(true)))
			require.True(t, utf8.ValidString(dialog.renderOutput.GetOutputStr(true)))
			require.Equal(t, message, strings.TrimSpace(strings.ReplaceAll(dialog.output.GetOutputStr(true), " \n", "\n")))
			require.Equal(t, strings.Fields(message), strings.Fields(dialog.renderOutput.GetOutputStr(false)))
			blankLines := strings.Count(message, "\n\n")
			require.Equal(t, blankLines, strings.Count(strings.ReplaceAll(dialog.renderOutput.GetOutputStr(true), " \n", "\n"), "\n\n"))
			dialog.ShowWithoutAnimation()
			for _, width := range []int{280, 600, 320} {
				viewport.Max.X = viewport.Min.X + width
				dialog.resolvePopupLayout(image.Rect(0, 0, 1280, 720))
				require.True(t, utf8.ValidString(dialog.output.GetOutputStr(true)))
				require.True(t, utf8.ValidString(dialog.renderOutput.GetOutputStr(true)))
				require.Equal(t, message, strings.TrimSpace(strings.ReplaceAll(dialog.output.GetOutputStr(true), " \n", "\n")))
				require.Equal(t, strings.Fields(message), strings.Fields(dialog.renderOutput.GetOutputStr(false)))
				require.Equal(t, blankLines, strings.Count(strings.ReplaceAll(dialog.renderOutput.GetOutputStr(true), " \n", "\n"), "\n\n"))
			}
		})
	}
}

func TestMessageDialogPopupLayoutScreenAnchorsBypassViewport(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	viewport := image.Rect(40, 20, 240, 400)
	setMessageDialogTestPopupViewport(t, func(image.Rectangle) image.Rectangle { return viewport })
	tests := []struct {
		name   string
		center float64
		build  func() *MessageDialog
	}{
		{"direct screen", 0.5, func() *MessageDialog {
			return NewMessageDialog("Screen message").WithEffects(false).
				WithHorizontalPosition(DialogHorizontalWholeScreen).WithVerticalPosition(0.5).Build()
		}},
		{"overlay screen", 0.5, func() *MessageDialog {
			return NewOverlayDialog("Screen message").AsBriefFeedback().WithPosition(PositionScreenCenter).Build().dialog
		}},
		{"numeric override", 0.62, func() *MessageDialog {
			return NewOverlayDialog("Screen message").AsBriefFeedback().WithHorizontalCenter(0.62).Build().dialog
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialog := tt.build()
			assert.Equal(t, dialog.baseConfig, dialog.config)
			assert.InDelta(t, tt.center, (dialog.config.StartPercentX+dialog.config.EndPercentX)/2, 0.000001)
			dialog.ShowWithoutAnimation()
			viewport = viewport.Add(image.Pt(100, 0))
			dialog.resolvePopupLayout(image.Rect(0, 0, 1280, 720))
			assert.Equal(t, dialog.baseConfig, dialog.config)
			assert.Empty(t, dialog.popupViewport)
			assert.Same(t, dialog.output, dialog.renderOutput)
		})
	}
}

func TestMessageDialogPopupLayoutEmptyProviderKeepsLegacyBuild(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	for _, provider := range []func(image.Rectangle) image.Rectangle{
		nil,
		func(image.Rectangle) image.Rectangle { return image.Rectangle{} },
	} {
		setMessageDialogTestPopupViewport(t, provider)
		dialog := NewSearchFeedbackDialog("Nothing here.", PositionMapCenterLower)
		assert.Equal(t, dialog.baseConfig, dialog.config)
		assert.InDelta(t, PositionMapCenterLower.GetHorizontalPercent(),
			(dialog.config.StartPercentX+dialog.config.EndPercentX)/2, 0.000001)
		assert.InDelta(t, PositionMapCenterLower.GetVerticalPercent(),
			(dialog.config.StartPercentY+dialog.config.EndPercentY)/2, 0.000001)
		assert.Same(t, dialog.output, dialog.renderOutput)
	}
}

func TestMessageDialogPopupLayoutRetainsBorderOnContentOnlyChanges(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	for _, viewport := range []image.Rectangle{{}, image.Rect(20, 0, 640, 720)} {
		setMessageDialogTestPopupViewport(t, func(image.Rectangle) image.Rectangle { return viewport })
		dialog := NewMessageDialog("One").WithEffects(false).WithSeparatePromptArea().
			WithTextLines(5).WithCharsPerLine(20).Build()
		dialog.SetPrompt("Enter: Close")
		dialog.ShowWithoutAnimation()
		bounds := image.Rect(0, 0, 1280, 720)
		dialog.resolvePopupLayout(bounds)
		frameBounds := dialog.frameBounds
		dialog.AppendLine("Two")
		dialog.SetPrompt("Esc: Close")
		dialog.resolvePopupLayout(bounds)
		assert.Equal(t, frameBounds, dialog.frameBounds, "text changes must retain frame geometry")
		assert.Nil(t, dialog.contentBuffer, "preparing text must not allocate a render target")
		assert.Nil(t, dialog.maskedBuffer)
		dialog.HideWithoutAnimation()
		dialog.ShowWithoutAnimation()
		dialog.resolvePopupLayout(bounds)
		assert.Equal(t, frameBounds, dialog.frameBounds, "showing the same popup must retain its frame geometry")
	}
}

func TestMessageDialogPopupLayoutPreservesStyledContentAndPrompt(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	wide := image.Rect(40, 20, 1000, 420)
	viewport := wide
	setMessageDialogTestPopupViewport(t, func(image.Rectangle) image.Rectangle { return viewport })
	dialog := NewMessageDialog("").WithEffects(false).WithSeparatePromptArea().
		WithTextLines(2).WithCharsPerLine(60).WithVerticalPosition(0.5).Build()
	dialog.AppendLineBuilder().Yellow("Words: ").RunicWithColor("AN NOX VAS FLAM", ucolor.EGALightRed).
		White(" reveal the hidden path").Done()
	dialog.SetPromptWithColor("Enter or Escape to close this message", ucolor.EGALightGreen)
	source, prompt := dialog.output, dialog.promptOutput
	sourceText, promptText := source.GetOutputStr(false), prompt.GetOutputStr(false)
	dialog.ShowWithoutAnimation()
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Dispose)
	dialog.Draw(screen)
	wideBounds := dialog.frameBounds
	wideLines := dialog.renderOutput.GetPanelContentLineCount()
	widePixels := dungeonViewImagePixels(dialog.contentBuffer)

	viewport = image.Rect(400, 20, 660, 420)
	dialog.Draw(screen)
	requireMessageDialogPopupBounds(t, dialog, viewport, 1280)
	assert.Greater(t, dialog.renderOutput.GetPanelContentLineCount(), wideLines)
	assert.Greater(t, dialog.renderPrompt.GetPanelContentLineCount(), 1)
	assert.Greater(t, dialog.frameBounds.Dy(), wideBounds.Dy(), "wrapped rows must grow the dialog")
	assert.Equal(t, strings.Fields(sourceText), strings.Fields(dialog.renderOutput.GetOutputStr(false)))
	assert.Equal(t, strings.Fields(promptText), strings.Fields(dialog.renderPrompt.GetOutputStr(false)))
	assert.Equal(t, sourceText, source.GetOutputStr(false))
	assert.Equal(t, promptText, prompt.GetOutputStr(false))
	for _, textColor := range []ucolor.EGAColor{ucolor.EGAYellow, ucolor.EGALightRed, ucolor.EGALightGreen} {
		requireImageRegionContainsColor(t, dialog.contentBuffer, dialog.frameBounds, textColor)
	}

	viewport = wide
	dialog.Draw(screen)
	assert.Same(t, source, dialog.output)
	assert.Same(t, prompt, dialog.promptOutput)
	assert.Equal(t, sourceText, source.GetOutputStr(false))
	assert.Equal(t, promptText, prompt.GetOutputStr(false))
	assert.Equal(t, wideLines, dialog.renderOutput.GetPanelContentLineCount())
	assert.Equal(t, wideBounds, dialog.frameBounds)
	assert.Equal(t, widePixels, dungeonViewImagePixels(dialog.contentBuffer),
		"widening must restore the original styled/runic glyphs and prompt")
}

func TestMessageDialogPopupLayoutRetainsLongHistoryAndDissolveState(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	viewport := image.Rect(40, 0, 640, 720)
	setMessageDialogTestPopupViewport(t, func(image.Rectangle) image.Rectangle { return viewport })
	dialog := NewMessageDialog("").WithTextLines(50).WithCharsPerLine(45).
		WithVerticalPosition(0.7).Build()
	for range 30 {
		dialog.AppendLine("Thou dost find nothing!")
	}
	source := dialog.output.GetOutputStr(false)
	dialog.Show()
	dialog.dissolveLevel = 0.5
	started := dialog.fadeStartTime
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	dialog.Draw(screen)
	effect := dialog.dissolveEffect
	require.NotNil(t, effect)
	viewport = image.Rect(60, 0, 440, 720)
	dialog.Draw(screen)
	assert.Same(t, effect, dialog.dissolveEffect)
	assert.Equal(t, fadingIn, dialog.fadeState)
	assert.Equal(t, 0.5, dialog.dissolveLevel)
	assert.Equal(t, started, dialog.fadeStartTime)
	assert.Equal(t, source, dialog.output.GetOutputStr(false))
	assert.Equal(t, strings.Fields(source), strings.Fields(dialog.renderOutput.GetOutputStr(false)))
	assert.GreaterOrEqual(t, dialog.frameBounds.Min.Y, 0)
	assert.LessOrEqual(t, dialog.frameBounds.Max.Y, 720)
	assert.InDelta(t, 0.7, (dialog.config.StartPercentY+dialog.config.EndPercentY)/2, 0.000001)
}

func setMessageDialogTestPopupViewport(t *testing.T, provider func(image.Rectangle) image.Rectangle) {
	t.Helper()
	dm := display.GetManager()
	previous := dm.SetMapPopupViewportProvider(provider)
	t.Cleanup(func() { dm.SetMapPopupViewportProvider(previous) })
}

func requireMessageDialogPopupBounds(t *testing.T, dialog *MessageDialog, viewport image.Rectangle, screenWidth int) {
	t.Helper()
	assert.Equal(t, viewport, dialog.popupViewport)
	assert.InDelta(t, float64(viewport.Min.X+viewport.Max.X)/2,
		(dialog.config.StartPercentX+dialog.config.EndPercentX)*float64(screenWidth)/2, 0.000001)
	bounds := dialog.frameBounds
	assert.GreaterOrEqual(t, bounds.Min.X, viewport.Min.X-1)
	assert.LessOrEqual(t, bounds.Max.X, viewport.Max.X+1)
	assert.InDelta(t, (dialog.baseConfig.StartPercentY+dialog.baseConfig.EndPercentY)/2,
		(dialog.config.StartPercentY+dialog.config.EndPercentY)/2, 0.000001,
		"map viewport must not change the vertical anchor")
}
