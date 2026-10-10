package main

import (
	"image"
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	ultimacolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	linearTalkPaneCenterDivisor           = 2
	linearTalkContentVerticalPaddingUnits = 5
)

type linearTalkPresentationStyle uint8

const (
	linearTalkPresentationJoined linearTalkPresentationStyle = iota
	linearTalkPresentationContinuousColon
	linearTalkPresentationContinuousMarker

	// REDUX DEVIATION: TALK-0002: original conversation text stays in
	// the permanent lower-right message window. Redux uses a modal card for
	// readability while preserving the original continuous, bottom-fed flow.
	// Keep the alternatives compiled while the conversation presentation is being evaluated in-game.
	activeLinearTalkPresentation = linearTalkPresentationContinuousMarker
)

type linearTalkDialogLayout struct {
	frameRect                image.Rectangle
	joinedTranscriptRect     image.Rectangle
	continuousTranscriptRect image.Rectangle
	inputPaneRect            image.Rectangle
	dividerY                 int
	pixelUnit                int
	footerRect               image.Rectangle
	footerDividerY           int
}

// Draw renders the selected experimental conversation presentation.
func (d *LinearTalkDialog) Draw(screen *ebiten.Image) {
	d.ensureTalkLayout()
	layout := d.layoutWithFooter(screen.Bounds())
	if d.historyPanel != nil {
		d.drawLinearTalkBacking(screen, layout)
		rect := layout.continuousTranscriptRect
		d.historyPanel.DrawPanelContentRange(screen, rect.Min, rect.Dx(), etext.AlignStart,
			d.historyRow, max(1, d.historyPanel.GetLineCapacity(rect.Dy())))
		d.drawConversationFooter(screen, layout)
		d.drawLinearTalkOuterFrame(screen, layout)

		return
	}
	switch activeLinearTalkPresentation {
	case linearTalkPresentationJoined:
		d.drawJoinedLinearTalkDialog(screen, layout)
	case linearTalkPresentationContinuousColon, linearTalkPresentationContinuousMarker:
		d.drawContinuousLinearTalkDialog(screen, layout, activeLinearTalkPresentation)
	}
	d.drawConversationFooter(screen, layout)
}

func (d *LinearTalkDialog) drawJoinedLinearTalkDialog(
	screen *ebiten.Image,
	layout linearTalkDialogLayout,
) {
	d.drawLinearTalkBacking(screen, layout)
	d.drawInputBackground(screen, layout)
	d.drawLinearTalkOutput(screen, layout.joinedTranscriptRect)
	d.drawLinearTalkOuterFrame(screen, layout)
	vector.FillRect(
		screen,
		float32(layout.frameRect.Min.X),
		float32(layout.dividerY),
		float32(layout.frameRect.Dx()),
		float32(layout.pixelUnit),
		widgets.DialogRuleColor,
		false,
	)
	d.TextInput.Draw(screen)
}

func (d *LinearTalkDialog) drawContinuousLinearTalkDialog(
	screen *ebiten.Image,
	layout linearTalkDialogLayout,
	style linearTalkPresentationStyle,
) {
	d.drawLinearTalkBacking(screen, layout)
	outputRect := layout.continuousTranscriptRect
	d.drawLinearTalkOutput(screen, outputRect)
	d.drawLinearTalkOuterFrame(screen, layout)
	d.updateContinuousInputPlacement(screen.Bounds(), layout, style)
	d.TextInput.Draw(screen)
}

func (d *LinearTalkDialog) drawLinearTalkBacking(
	screen *ebiten.Image,
	layout linearTalkDialogLayout,
) {
	widgets.FillClassicPixelRoundedRect(
		screen,
		layout.frameRect,
		layout.pixelUnit,
		ultimacolor.DialogBackground.RGBA(),
	)
}

func (d *LinearTalkDialog) drawLinearTalkOutput(screen *ebiten.Image, rect image.Rectangle) {
	d.Output.DrawScrollingBufferWithHeight(screen, image.Point{
		X: rect.Min.X,
		Y: rect.Min.Y,
	}, rect.Dx(), rect.Dy(), false, etext.AlignStart, etext.AlignEnd)
}

func (d *LinearTalkDialog) drawLinearTalkOuterFrame(
	screen *ebiten.Image,
	layout linearTalkDialogLayout,
) {
	widgets.DrawClassicPixelRoundedFrame(
		screen,
		layout.frameRect,
		layout.pixelUnit,
		widgets.DialogRuleColor,
	)
}

func (d *LinearTalkDialog) updateContinuousInputPlacement(
	bounds image.Rectangle,
	layout linearTalkDialogLayout,
	style linearTalkPresentationStyle,
) {
	inputRect := d.continuousInputRect(layout, style)
	d.TextInput.SetPlacement(rectanglePlacement(bounds, inputRect))
}

func (d *LinearTalkDialog) continuousInputRect(
	layout linearTalkDialogLayout,
	style linearTalkPresentationStyle,
) image.Rectangle {
	inputRect := layout.continuousTranscriptRect
	inputRect.Min.Y = inputRect.Max.Y - int(math.Ceil(d.Output.GetFontHeight()))
	if d.waitingForInput {
		inputRect.Min.X += d.linearTalkResponseMarkerWidth(style)
	}

	return inputRect
}

func (d *LinearTalkDialog) linearTalkResponseMarkerWidth(style linearTalkPresentationStyle) int {
	switch style {
	case linearTalkPresentationContinuousColon:
		width, _ := d.font.MeasureString(": ")

		return int(math.Ceil(width))
	case linearTalkPresentationContinuousMarker:
		markerWidth, _ := d.font.MeasureString("M")
		spaceWidth, _ := d.font.MeasureString(" ")

		return int(math.Ceil(markerWidth + spaceWidth))
	case linearTalkPresentationJoined:
		return 0
	}

	panic("LinearTalkDialog.linearTalkResponseMarkerWidth: unsupported presentation style")
}

func calculateLinearTalkDialogLayout(bounds image.Rectangle) linearTalkDialogLayout {
	outputFrameRect := percentRect(
		bounds,
		borderStartPercentX,
		borderStartPercentY,
		borderEndPercentX,
		talkOutputBorderEndPercentY,
	)
	inputFrameRect := percentRect(
		bounds,
		borderStartPercentX,
		talkInputBorderStartPercentY,
		borderEndPercentX,
		talkInputBorderEndPercentY,
	)
	dividerY := (outputFrameRect.Max.Y + inputFrameRect.Min.Y) / linearTalkPaneCenterDivisor
	frameRect := image.Rect(
		outputFrameRect.Min.X,
		outputFrameRect.Min.Y,
		inputFrameRect.Max.X,
		inputFrameRect.Max.Y,
	)
	pixelUnit := widgets.ClassicUIPixelUnit(bounds.Dx(), bounds.Dy())
	continuousTranscriptRect := percentRect(
		bounds,
		textRectStartPercentX,
		borderStartPercentY,
		borderEndPercentX-percentTextIndentFromBorder,
		talkInputBorderEndPercentY,
	)
	verticalPadding := pixelUnit * linearTalkContentVerticalPaddingUnits
	continuousTranscriptRect.Min.Y = frameRect.Min.Y + verticalPadding
	continuousTranscriptRect.Max.Y = frameRect.Max.Y - verticalPadding

	return linearTalkDialogLayout{
		frameRect: frameRect,
		joinedTranscriptRect: percentRect(
			bounds,
			textRectStartPercentX,
			textRectStartPercentY,
			textRectEndPercentX,
			textRectEndPercentY,
		),
		continuousTranscriptRect: continuousTranscriptRect,
		inputPaneRect:            image.Rect(frameRect.Min.X, dividerY, frameRect.Max.X, frameRect.Max.Y),
		dividerY:                 dividerY,
		pixelUnit:                pixelUnit,
		footerRect:               image.Rect(0, 0, 0, 0),
		footerDividerY:           0,
	}
}

// getInputBackgroundRect returns the rectangle for the input background with inset for rounded corners
func (d *LinearTalkDialog) getInputBackgroundRect(layout linearTalkDialogLayout) image.Rectangle {
	inset := max(backgroundInsetPixels, layout.pixelUnit)

	return layout.inputPaneRect.Inset(inset)
}

// drawInputBackground draws the subdued background over the input area while text input is unavailable.
func (d *LinearTalkDialog) drawInputBackground(screen *ebiten.Image, layout linearTalkDialogLayout) {
	if !d.isPausing && !d.waitingForKeypressContinue && !d.waitingForDismissal {
		return
	}

	bgRect := d.getInputBackgroundRect(layout)
	vector.FillRect(screen,
		float32(bgRect.Min.X),
		float32(bgRect.Min.Y),
		float32(bgRect.Dx()),
		float32(bgRect.Dy()),
		ultimacolor.DisabledTextBackground,
		false)
}
