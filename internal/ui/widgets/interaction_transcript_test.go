package widgets

import (
	"fmt"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestInteractionTranscriptRetainsContentAcrossPromptStages(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	dialog := newTestInteractionTranscript(keyboard)
	dialog.AppendLine("Thou dost see a well.")
	dialog.BeginYesNo("Drop a coin?")
	require.Equal(t, yesNoControlHint, dialog.footerOutput.GetOutputStr(false))
	yesNoInputX := dialog.textInput.mainTextPlacement.StartPercentX
	require.Greater(t, dialog.transcriptOutput.GetLineSpacing(), GetScaledLineHeight(FontSizeLarge))
	yesNoInputRect := sprites.GetRectangleFromPercents(dialog.textInput.mainTextPlacement)
	expectedYesNoRowY := dialog.layout.transcriptRect.Min.Y + int(math.Round(dialog.transcriptOutput.GetLineSpacing()))
	require.Equal(t, expectedYesNoRowY, yesNoInputRect.Min.Y)
	require.Equal(t, interactionTranscriptPromptMarker, dialog.textInput.promptPrefix)
	require.Equal(t, 2, dialog.transcriptOutput.GetPanelContentLineCount())

	keyboard.SimulateKeyPress(ebiten.KeyY)
	dialog.Update()
	require.False(t, dialog.InputComplete(), "yes/no choice must still require Enter")
	keyboard.SimulateKeyPress(ebiten.KeyEnter)
	dialog.Update()

	require.True(t, dialog.InputComplete())
	require.True(t, dialog.Submitted())
	require.Equal(t, "Yes", dialog.Response())
	require.Empty(t, dialog.footerOutput.GetOutputStr(false))
	require.Contains(t, dialog.TranscriptText(), "Thou dost see a well.")
	require.Contains(t, dialog.TranscriptText(), "Drop a coin?  > Yes")

	dialog.BeginText("Thy wish?", "^[a-zA-Z]{0,12}$")
	require.Less(t, dialog.textInput.mainTextPlacement.StartPercentX, yesNoInputX)
	wishInputRect := sprites.GetRectangleFromPercents(dialog.textInput.mainTextPlacement)
	expectedWishRowY := dialog.layout.transcriptRect.Min.Y +
		int(math.Round(dialog.transcriptOutput.GetLineSpacing()*2))
	require.Equal(t, expectedWishRowY, wishInputRect.Min.Y)
	require.Less(t, wishInputRect.Min.Y, dialog.layout.transcriptRect.Max.Y)
	for _, key := range []ebiten.Key{ebiten.KeyH, ebiten.KeyO, ebiten.KeyR, ebiten.KeyS, ebiten.KeyE} {
		keyboard.SimulateKeyPress(key)
		dialog.Update()
	}
	keyboard.SimulateKeyPress(ebiten.KeyEnter)
	dialog.Update()

	require.True(t, dialog.InputComplete())
	require.Equal(t, "HORSE", dialog.Response())
	require.Contains(t, dialog.TranscriptText(), "Thy wish?  > HORSE")

	dialog.AppendLine("No effect!")
	dialog.BeginAcknowledgement()
	require.Contains(t, dialog.footerOutput.GetOutputStr(false), "Enter/Esc: Close")
	keyboard.SimulateKeyPress(ebiten.KeyEscape)
	dialog.Update()
	require.True(t, dialog.InputComplete())
	require.Empty(t, dialog.footerOutput.GetOutputStr(false))
}

func TestInteractionTranscriptDefaultEscapeStillCancels(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	dialog := newTestInteractionTranscript(keyboard)
	dialog.BeginText("Mantra:", "^[A-Z]*$")
	require.Contains(t, dialog.ControlHint(), "Esc: Cancel")
	keyboard.SimulateKeyPress(ebiten.KeyA)
	dialog.Update()
	keyboard.SimulateKeyPress(ebiten.KeyEscape)
	dialog.Update()
	require.True(t, dialog.InputComplete())
	require.False(t, dialog.Submitted())
	require.Empty(t, dialog.Response())
}

func TestInteractionTranscriptPlacesPromptAfterMultilineEntry(t *testing.T) {
	dialog := NewInteractionTranscript(newWidgetTestKeyboard(0), InteractionTranscriptConfig{
		MaxLines:       8,
		MaxInputLength: 1,
		WidthSamples: []string{
			"Didst thou bring my box?",
			"You reply:  > Yes",
		},
	})
	dialog.AppendLine("Didst thou bring my box?\n\nThe box contains the jewel.")
	dialog.BeginUppercaseYesNo("You reply:")

	physicalRows := dialog.transcriptOutput.GetPanelContentLineCount()
	require.Greater(t, physicalRows, len(dialog.entries)+1)
	inputRect := sprites.GetRectangleFromPercents(dialog.textInput.mainTextPlacement)
	expectedRowY := dialog.layout.transcriptRect.Min.Y +
		int(math.Round(float64(physicalRows-1)*dialog.transcriptOutput.GetLineSpacing()))
	require.Equal(t, expectedRowY, inputRect.Min.Y)
	require.Less(t, inputRect.Min.Y, dialog.layout.transcriptRect.Max.Y)
}

func TestInteractionTranscriptPlacesPromptAfterVisibleScrollingRows(t *testing.T) {
	dialog := NewInteractionTranscript(newWidgetTestKeyboard(0), InteractionTranscriptConfig{
		MaxLines:       6,
		MaxInputLength: 1,
		WidthSamples: []string{
			"Visible history",
			"You reply:  > Yes",
		},
	})
	dialog.AppendLine("\nVisible history")
	dialog.BeginUppercaseYesNo("You reply:")

	require.Equal(t, 3, dialog.transcriptOutput.GetPanelContentLineCount())
	require.Equal(t, 2, dialog.transcriptOutput.GetVisibleScrollingLineCount(dialog.layout.transcriptRect.Dy()))
	inputRect := sprites.GetRectangleFromPercents(dialog.textInput.mainTextPlacement)
	expectedRowY := dialog.layout.transcriptRect.Min.Y +
		int(math.Round(dialog.transcriptOutput.GetLineSpacing()))
	require.Equal(t, expectedRowY, inputRect.Min.Y)
}

func TestInteractionTranscriptPassiveAnimationHasNoKeyboardHint(t *testing.T) {
	dialog := newTestInteractionTranscript(newWidgetTestKeyboard(0))
	dialog.AppendLine("Avatar sneaks across")

	require.Empty(t, dialog.footerOutput.GetOutputStr(false))
	dialog.AppendToLastLine("...")
	require.Contains(t, dialog.TranscriptText(), "Avatar sneaks across...")
	require.Empty(t, dialog.footerOutput.GetOutputStr(false))
}

func TestInteractionTranscriptRequiresFreshKeyWhenHintAppears(t *testing.T) {
	pressed := map[ebiten.Key]bool{ebiten.KeyEnter: true}
	keyboard := input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool {
		return pressed[key]
	})
	dialog := newTestInteractionTranscript(keyboard)
	dialog.BeginAcknowledgement()

	dialog.Update()
	require.False(t, dialog.InputComplete(), "a key held before the hint appeared must not be accepted")

	pressed[ebiten.KeyEnter] = false
	keyboard.ClearFrameConsumption()
	dialog.Update()
	pressed[ebiten.KeyEnter] = true
	dialog.Update()

	require.True(t, dialog.InputComplete(), "a fresh press after release should be accepted")
	require.Empty(t, dialog.footerOutput.GetOutputStr(false))
}

func TestInteractionTranscriptUppercaseYesNoCannotBeCancelled(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	dialog := newTestInteractionTranscript(keyboard)
	dialog.BeginUppercaseYesNo("Yes or No?")

	keyboard.SimulateKeyPress(ebiten.KeyEscape)
	dialog.Update()
	require.False(t, dialog.InputComplete())
	keyboard.SimulateKeyPress(ebiten.KeyY)
	dialog.Update()
	keyboard.SimulateKeyPress(ebiten.KeyEnter)
	dialog.Update()

	require.True(t, dialog.InputComplete())
	require.Equal(t, "Yes", dialog.Response())
}

func TestInteractionTranscriptUsesStableBoundedLayoutOnResize(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	dialog := newTestInteractionTranscript(newWidgetTestKeyboard(0))
	dialog.BeginYesNo("Drop a coin?")
	originalPanel := dialog.layout.panelRect
	originalTextInput := dialog.textInput
	capacity := dialog.transcriptOutput.GetLineCapacity(dialog.layout.transcriptRect.Dy())
	require.Equal(t, 4, capacity)
	padding := dialog.layout.transcriptRect.Min.X - dialog.layout.panelRect.Min.X
	expectedPanelHeight := padding*2 +
		int(math.Ceil(dialog.transcriptOutput.GetTotalHeightForLines(4))) +
		dialog.layout.lineGap + int(math.Ceil(dialog.footerOutput.GetFontHeight()))
	require.Equal(t, expectedPanelHeight, dialog.layout.panelRect.Dy())
	require.Equal(t, dialog.layout.transcriptRect.Max.Y+dialog.layout.lineGap, dialog.layout.footerRect.Min.Y)
	for index := range capacity + 2 {
		dialog.AppendLine(fmt.Sprintf("Line %d", index))
	}

	require.NotContains(t, dialog.TranscriptText(), "Line 0")
	require.Contains(t, dialog.TranscriptText(), fmt.Sprintf("Line %d", capacity+1))
	dialog.BeginText("Thy wish?", "^[a-zA-Z]{0,12}$")
	require.Equal(t, originalPanel, dialog.layout.panelRect)

	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Dispose)
	dialog.Draw(screen)
	requireClassicFrameSilhouette(t, screen, dialog.layout.panelRect)

	setResponsiveDialogTestResolution(t, 1920, 1080)
	dialog.ensureLayout(1920, 1080)
	require.Same(t, originalTextInput, dialog.textInput)
	require.Contains(t, dialog.TranscriptText(), fmt.Sprintf("Line %d", capacity+1))
	require.Equal(t, 1920, dialog.lastWidth)
	require.Equal(t, 1080, dialog.lastHeight)
	resizedInputRect := sprites.GetRectangleFromPercents(dialog.textInput.mainTextPlacement)
	resizedRows := dialog.transcriptOutput.GetVisibleScrollingLineCount(dialog.layout.transcriptRect.Dy())
	expectedResizedRowY := dialog.layout.transcriptRect.Min.Y +
		int(math.Round(float64(resizedRows-1)*dialog.transcriptOutput.GetLineSpacing()))
	require.InDelta(t, expectedResizedRowY, resizedInputRect.Min.Y, 1)
}

func TestInteractionTranscriptSupportsSceneSpecificVerticalCenter(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	defaultDialog := newTestInteractionTranscript(newWidgetTestKeyboard(0))
	topDialog := NewInteractionTranscript(newWidgetTestKeyboard(0), InteractionTranscriptConfig{
		MaxLines:       4,
		MaxInputLength: 12,
		WidthSamples:   []string{"Blackthorn says:", "Your response?  > VERAMOCOR"},
		CenterYPercent: 0.25,
	})

	require.Less(t, topDialog.layout.panelRect.Min.Y, defaultDialog.layout.panelRect.Min.Y)
	require.InDelta(t, 720*0.25, float64(topDialog.layout.panelRect.Min.Y+topDialog.layout.panelRect.Dy()/2), 1)
}

func TestInteractionTranscriptFillsBoundedRailAndStacksNarrowInput(t *testing.T) {
	setResponsiveDialogTestResolution(t, 320, 180)
	placement := sprites.PercentBasedPlacement{
		StartPercentX: 0.01,
		EndPercentX:   0.30,
		StartPercentY: 0.02,
		EndPercentY:   0.98,
	}
	dialog := NewInteractionTranscript(newWidgetTestKeyboard(0), InteractionTranscriptConfig{
		MaxLines:       6,
		MaxInputLength: 14,
		WidthSamples:   []string{"Blackthorn says:", "Your response?  > VERAMOCOR"},
		PanelPlacement: &placement,
	})
	dialog.AppendLine("Blackthorn demands an answer.")
	dialog.BeginText("Your response?", "^[a-zA-Z]{0,14}$")

	require.Equal(t, percentPlacementToPixelRectangle(placement, 320, 180), dialog.layout.panelRect)
	require.True(t, dialog.stackedInput)
	inputRect := sprites.GetRectangleFromPercents(dialog.textInput.mainTextPlacement)
	require.Equal(t, dialog.layout.transcriptRect.Min.X, inputRect.Min.X)
	require.Less(t, inputRect.Max.Y, dialog.layout.transcriptRect.Max.Y+1)
	require.Equal(t, dialog.layout.transcriptRect.Max.Y+dialog.layout.lineGap, dialog.layout.footerRect.Min.Y)
	require.LessOrEqual(t, dialog.layout.footerRect.Max.Y, dialog.layout.panelRect.Max.Y)
	require.Greater(t, dialog.transcriptOutput.GetLineCapacity(dialog.layout.transcriptRect.Dy()), 6)
}

func TestInteractionTranscriptUsesPassiveControlHintInRailFooter(t *testing.T) {
	setResponsiveDialogTestResolution(t, 320, 200)
	placement := sprites.PercentBasedPlacement{
		StartPercentX: 0.01,
		EndPercentX:   0.30,
		StartPercentY: 0.02,
		EndPercentY:   0.98,
	}
	dialog := NewInteractionTranscript(newWidgetTestKeyboard(0), InteractionTranscriptConfig{
		MaxLines: 6, MaxInputLength: 1,
		WidthSamples:   []string{"We shall be here a while."},
		PanelPlacement: &placement,
	})

	dialog.SetControlHint("Any key: Return to menu")

	require.Contains(t, dialog.ControlHint(), "Any key")
	require.LessOrEqual(t, dialog.footerOutput.GetPanelContentLineCount(), 2)
}

func newTestInteractionTranscript(keyboard *input.Keyboard) *InteractionTranscript {
	return NewInteractionTranscript(keyboard, InteractionTranscriptConfig{
		MaxLines:       4,
		MaxInputLength: 12,
		WidthSamples: []string{
			"Thou dost see a well.",
			"Drop a coin?  > Yes",
			"Thy wish?  > MMMMMMMMMMMM",
			"Thy wish can no longer be granted.",
		},
	})
}
