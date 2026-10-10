package main

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestLinearTalkDialogLayoutJoinsTranscriptAndInputPanes(t *testing.T) {
	bounds := image.Rect(0, 0, 1280, 720)
	layout := calculateLinearTalkDialogLayout(bounds)
	inputTextRect := percentRect(
		bounds,
		textInputStartPercentX,
		textInputStartPercentY,
		textInputEndPercentX,
		textInputEndPercentY,
	)

	require.True(t, layout.joinedTranscriptRect.In(layout.frameRect))
	require.True(t, layout.continuousTranscriptRect.In(layout.frameRect))
	require.True(t, inputTextRect.In(layout.frameRect))
	require.Less(t, layout.joinedTranscriptRect.Max.Y, layout.dividerY)
	require.Greater(t, layout.continuousTranscriptRect.Max.Y, layout.dividerY)
	expectedVerticalPadding := layout.pixelUnit * linearTalkContentVerticalPaddingUnits
	require.Equal(t, layout.frameRect.Min.Y+expectedVerticalPadding, layout.continuousTranscriptRect.Min.Y)
	require.Equal(t, layout.frameRect.Max.Y-expectedVerticalPadding, layout.continuousTranscriptRect.Max.Y)
	require.Greater(t, inputTextRect.Min.Y, layout.dividerY)
	require.Equal(t, layout.dividerY, layout.inputPaneRect.Min.Y)
	require.Equal(t, layout.frameRect.Max, layout.inputPaneRect.Max)
}

func TestLinearTalkDialogDrawsJoinedWhiteFrame(t *testing.T) {
	// Joined presentation remains an intentional product alternative; retain its frame contract.
	setLinearTalkDialogTestResolution(t, 1280, 720)
	keyboard := newPresentationTestKeyboard()
	dialog := &LinearTalkDialog{ //nolint:exhaustruct // Focused rendering fixture.
		gameScene: &GameScene{ //nolint:exhaustruct // Only display-dependent dialog initialization is needed.
			gameState: &game_state.GameState{},
			keyboard:  keyboard,
		},
	}
	dialog.initializeResizeableVisualElements()
	screen := newPresentationRaster(1280, 720)
	layout := calculateLinearTalkDialogLayout(screen.Bounds())
	dialog.drawJoinedLinearTalkDialog(screen, layout)

	topFramePoint := image.Pt(layout.frameRect.Min.X+layout.frameRect.Dx()/2, layout.frameRect.Min.Y)
	dividerPoint := image.Pt(layout.frameRect.Min.X+layout.frameRect.Dx()/2, layout.dividerY)
	interiorPoint := image.Pt(
		layout.frameRect.Min.X+layout.pixelUnit*4,
		layout.frameRect.Min.Y+layout.pixelUnit*4,
	)
	requireLinearTalkDialogPixel(t, screen, topFramePoint, widgets.DialogRuleColor)
	requireLinearTalkDialogPixel(t, screen, dividerPoint, widgets.DialogRuleColor)
	requireLinearTalkDialogPixel(t, screen, interiorPoint, ucolor.DialogBackground.RGBA())
	requireLinearTalkDialogPixel(t, screen, layout.frameRect.Min, color.Transparent)
}

func TestLinearTalkDialogC3KeepsUndividedTranscriptAboveFooter(t *testing.T) {
	setLinearTalkDialogTestResolution(t, 1280, 720)
	keyboard := newPresentationTestKeyboard()
	dialog := &LinearTalkDialog{ //nolint:exhaustruct // Focused rendering fixture.
		gameScene: &GameScene{ //nolint:exhaustruct // Only display-dependent dialog initialization is needed.
			gameState: &game_state.GameState{},
			keyboard:  keyboard,
		},
	}
	dialog.initializeResizeableVisualElements()
	screen := newPresentationRaster(1280, 720)
	dialog.Draw(screen)
	layout := dialog.layoutWithFooter(screen.Bounds())

	topFramePoint := image.Pt(layout.frameRect.Min.X+layout.frameRect.Dx()/2, layout.frameRect.Min.Y)
	formerDividerPoint := image.Pt(layout.frameRect.Min.X+layout.frameRect.Dx()/2, layout.dividerY)
	requireLinearTalkDialogPixel(t, screen, topFramePoint, widgets.DialogRuleColor)
	requireLinearTalkDialogPixel(t, screen, formerDividerPoint, ucolor.DialogBackground.RGBA())
}

func TestLinearTalkDialogC3BufferCanFillVisibleTranscript(t *testing.T) {
	setTalkHistoryTestResolution(t, 1280, 720)
	keyboard := newPresentationTestKeyboard()
	dialog := &LinearTalkDialog{ //nolint:exhaustruct // Focused rendering fixture.
		gameScene: &GameScene{ //nolint:exhaustruct // Only display-dependent dialog initialization is needed.
			gameState: &game_state.GameState{},
			keyboard:  keyboard,
		},
	}
	dialog.initializeResizeableVisualElements()
	layout := calculateLinearTalkDialogLayout(image.Rect(0, 0, 1280, 720))
	visibleRows := dialog.Output.GetLineCapacity(layout.continuousTranscriptRect.Dy())

	require.GreaterOrEqual(t, linearTalkOutputMaxLines(activeLinearTalkPresentation), visibleRows)
	require.Greater(t, visibleRows, talkMaxLines)
}

func TestLinearTalkDialogC3InputSharesFinalTranscriptRow(t *testing.T) {
	setLinearTalkDialogTestResolution(t, 1280, 720)
	keyboard := newPresentationTestKeyboard()
	dialog := &LinearTalkDialog{ //nolint:exhaustruct // Focused rendering fixture.
		gameScene: &GameScene{ //nolint:exhaustruct // Only display-dependent dialog initialization is needed.
			gameState: &game_state.GameState{},
			keyboard:  keyboard,
		},
		waitingForInput: true,
	}
	dialog.initializeResizeableVisualElements()
	dialog.addInputPromptForStyle("Your interest?", linearTalkPresentationContinuousMarker)
	layout := calculateLinearTalkDialogLayout(image.Rect(0, 0, 1280, 720))
	inputRect := dialog.continuousInputRect(layout, linearTalkPresentationContinuousMarker)
	expectedRowY := layout.continuousTranscriptRect.Max.Y - int(math.Ceil(dialog.Output.GetFontHeight()))
	dialog.TextInput.SetText("JOB")
	screen := newPresentationRaster(1280, 720)
	dialog.drawContinuousLinearTalkDialog(screen, layout, linearTalkPresentationContinuousMarker)
	markerRect := image.Rect(
		layout.continuousTranscriptRect.Min.X,
		expectedRowY,
		inputRect.Min.X,
		layout.continuousTranscriptRect.Max.Y,
	)

	require.Equal(t, expectedRowY, inputRect.Min.Y)
	require.Greater(t, inputRect.Min.X, layout.continuousTranscriptRect.Min.X)
	requireLinearTalkDialogRegionContainsColor(t, screen, markerRect, ucolor.EGALightGreen)
	require.Equal(t, []string{"Your interest?", ">"}, rowsFromFirstContent(dialog.Output))
	dialog.Output.AppendText("JOB")
	require.Equal(t, []string{"Your interest?", "> JOB"}, rowsFromFirstContent(dialog.Output))
}

func TestLinearTalkDialogDisabledInputBackingStaysBelowDivider(t *testing.T) {
	// Joined presentation remains an intentional product alternative; retain disabled-input separation.
	setLinearTalkDialogTestResolution(t, 1280, 720)
	keyboard := newPresentationTestKeyboard()
	dialog := &LinearTalkDialog{ //nolint:exhaustruct // Focused rendering fixture.
		gameScene: &GameScene{ //nolint:exhaustruct // Only display-dependent dialog initialization is needed.
			gameState: &game_state.GameState{},
			keyboard:  keyboard,
		},
		isPausing: true,
	}
	dialog.initializeResizeableVisualElements()
	screen := newPresentationRaster(1280, 720)
	layout := calculateLinearTalkDialogLayout(screen.Bounds())
	dialog.drawJoinedLinearTalkDialog(screen, layout)
	backgroundRect := dialog.getInputBackgroundRect(layout)
	disabledPixel := linearTalkDialogPixel(t, screen, backgroundRect.Min)
	speechPixel := linearTalkDialogPixel(
		t,
		screen,
		image.Pt(backgroundRect.Min.X, layout.dividerY-layout.pixelUnit),
	)

	require.Greater(t, backgroundRect.Min.Y, layout.dividerY+layout.pixelUnit-1)
	require.NotEqual(t, speechPixel, disabledPixel)
	requireLinearTalkDialogPixel(t, screen,
		image.Pt(backgroundRect.Min.X, layout.dividerY-layout.pixelUnit), ucolor.DialogBackground.RGBA())
}

func setLinearTalkDialogTestResolution(t *testing.T, width, height int) {
	t.Helper()
	setTalkHistoryTestResolution(t, width, height)
}

func requireLinearTalkDialogPixel(
	t *testing.T,
	screen *ebiten.Image,
	point image.Point,
	want color.Color,
) {
	t.Helper()
	actualRGBA := linearTalkDialogPixel(t, screen, point)
	wantRGBA, ok := color.RGBAModel.Convert(want).(color.RGBA)
	require.True(t, ok)
	require.Equal(t, wantRGBA, actualRGBA)
}

func linearTalkDialogPixel(t *testing.T, screen *ebiten.Image, point image.Point) color.RGBA {
	t.Helper()
	actualRGBA, ok := color.RGBAModel.Convert(screen.At(point.X, point.Y)).(color.RGBA)
	require.True(t, ok)

	return actualRGBA
}

func requireLinearTalkDialogRegionContainsColor(
	t *testing.T,
	screen *ebiten.Image,
	region image.Rectangle,
	want color.Color,
) {
	t.Helper()
	wantRGBA, ok := color.RGBAModel.Convert(want).(color.RGBA)
	require.True(t, ok)
	for yCoordinate := region.Min.Y; yCoordinate < region.Max.Y; yCoordinate++ {
		for xCoordinate := region.Min.X; xCoordinate < region.Max.X; xCoordinate++ {
			if linearTalkDialogPixel(t, screen, image.Pt(xCoordinate, yCoordinate)) == wantRGBA {
				return
			}
		}
	}
	t.Fatalf("region %v does not contain color %v", region, wantRGBA)
}
