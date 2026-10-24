package main

import (
	"image"
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	ultimacolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	linearTalkContentVerticalPaddingUnits = 5
)

// REDUX DEVIATION: TALK-0002: original conversation text stays in
// the permanent lower-right message window. Redux uses a modal card for
// readability while preserving the original continuous, bottom-fed flow.
type linearTalkDialogLayout struct {
	frameRect                image.Rectangle
	continuousTranscriptRect image.Rectangle
	pixelUnit                int
	footerRect               image.Rectangle
	footerDividerY           int
}

// Draw renders the conversation card.
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
	d.drawContinuousLinearTalkDialog(screen, layout)
	d.drawConversationFooter(screen, layout)
}

func (d *LinearTalkDialog) drawContinuousLinearTalkDialog(
	screen *ebiten.Image,
	layout linearTalkDialogLayout,
) {
	d.drawLinearTalkBacking(screen, layout)
	outputRect := layout.continuousTranscriptRect
	d.drawLinearTalkOutput(screen, outputRect)
	d.drawLinearTalkOuterFrame(screen, layout)
	d.updateContinuousInputPlacement(screen.Bounds(), layout)
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
) {
	inputRect := d.continuousInputRect(layout)
	d.TextInput.SetPlacement(rectanglePlacement(bounds, inputRect))
}

func (d *LinearTalkDialog) continuousInputRect(layout linearTalkDialogLayout) image.Rectangle {
	inputRect := layout.continuousTranscriptRect
	inputRect.Min.Y = inputRect.Max.Y - int(math.Ceil(d.Output.GetFontHeight()))
	if d.waitingForInput {
		inputRect.Min.X += d.linearTalkResponseMarkerWidth()
	}

	return inputRect
}

func (d *LinearTalkDialog) linearTalkResponseMarkerWidth() int {
	markerWidth, _ := d.font.MeasureString("M")
	spaceWidth, _ := d.font.MeasureString(" ")

	return int(math.Ceil(markerWidth + spaceWidth))
}

func calculateLinearTalkDialogLayout(bounds image.Rectangle) linearTalkDialogLayout {
	frameRect := percentRect(bounds, borderStartPercentX, borderStartPercentY, borderEndPercentX, borderEndPercentY)
	pixelUnit := widgets.ClassicUIPixelUnit(bounds.Dx(), bounds.Dy())
	continuousTranscriptRect := percentRect(
		bounds,
		textRectStartPercentX,
		borderStartPercentY,
		borderEndPercentX-percentTextIndentFromBorder,
		borderEndPercentY,
	)
	verticalPadding := pixelUnit * linearTalkContentVerticalPaddingUnits
	continuousTranscriptRect.Min.Y = frameRect.Min.Y + verticalPadding
	continuousTranscriptRect.Max.Y = frameRect.Max.Y - verticalPadding

	return linearTalkDialogLayout{
		frameRect:                frameRect,
		continuousTranscriptRect: continuousTranscriptRect,
		pixelUnit:                pixelUnit,
		footerRect:               image.Rect(0, 0, 0, 0),
		footerDividerY:           0,
	}
}
