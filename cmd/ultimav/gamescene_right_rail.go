package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	gametext "github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/mainscreen"
)

const (
	rightRailStartX       = 0.75
	rightRailPanelStartX  = 0.751
	rightRailContentStart = 0.77
	rightRailContentEnd   = 0.99

	characterPanelEndY = 0.50
	activePanelStartY  = 0.485
	activePanelEndY    = 0.575
	outputPanelStartY  = 0.56
	outputPanelEndY    = 0.91
	provisionPanelY    = 0.895

	activeContentStartY = 0.505
	activeContentEndY   = 0.555
	outputContentStartY = 0.580
	outputContentEndY   = 0.886
	provisionContentY   = 0.915
	// The rasterized bottom frame starts above 0.985. Keep the content inside
	// its inner edge; otherwise the bottom row paints into the white/blue chrome.
	provisionContentEnd = 0.975
	referenceGameWidth  = 1920
)

type rightRailLayout struct {
	railBorder       image.Rectangle
	characterBorder  image.Rectangle
	characterContent image.Rectangle
	activeBorder     image.Rectangle
	outputBorder     image.Rectangle
	outputContent    image.Rectangle
	provisionBorder  image.Rectangle
	provisionContent image.Rectangle
}

type statusBandLayout struct {
	content image.Rectangle
}

func calculateStatusBandLayout(band image.Rectangle) statusBandLayout {
	// Preserve the old content insets, including its asymmetric horizontal padding.
	return statusBandLayout{
		content: percentRect(band,
			(rightRailContentStart-rightRailPanelStartX)/(1-rightRailPanelStartX),
			(activeContentStartY-activePanelStartY)/(activePanelEndY-activePanelStartY),
			(rightRailContentEnd-rightRailPanelStartX)/(1-rightRailPanelStartX),
			(activeContentEndY-activePanelStartY)/(activePanelEndY-activePanelStartY)),
	}
}

func calculateRightRailLayout(bounds image.Rectangle) rightRailLayout {
	layout := rightRailLayout{
		railBorder:       percentRect(bounds, rightRailStartX, 0, 1, 1),
		characterBorder:  percentRect(bounds, rightRailPanelStartX, 0, 1, characterPanelEndY),
		characterContent: percentRect(bounds, rightRailContentStart, 0.025, rightRailContentEnd, 0.48),
		activeBorder:     percentRect(bounds, rightRailPanelStartX, activePanelStartY, 1, activePanelEndY),
		outputBorder:     percentRect(bounds, rightRailPanelStartX, outputPanelStartY, 1, outputPanelEndY),
		outputContent:    percentRect(bounds, rightRailContentStart, outputContentStartY, rightRailContentEnd, outputContentEndY),
		provisionBorder:  percentRect(bounds, rightRailPanelStartX, provisionPanelY, 1, 1),
		provisionContent: percentRect(bounds, rightRailContentStart, provisionContentY, rightRailContentEnd, provisionContentEnd),
	}
	// Grow upward only by the space needed for four normal-font rows, keeping
	// the bottom inset and all map, party and status geometry fixed. The message
	// renderer derives its actual capacity from the shortened clipping surface.
	growth := max(0, mainscreen.ProvisionSummaryHeight(bounds.Dx())-layout.provisionContent.Dy())
	layout.outputBorder.Max.Y -= growth
	layout.outputContent.Max.Y -= growth
	layout.provisionBorder.Min.Y -= growth
	layout.provisionContent.Min.Y -= growth

	return layout
}

func newMainOutput(screenWidth int) (*gametext.UltimaFont, *gametext.Output) {
	scale := float64(screenWidth) / referenceGameWidth
	font := gametext.NewUltimaFont(float64(defaultOutputFontSize) * scale)
	lineSpacing := float64(defaultOutputFontSize) * defaultOutputLineSpacingMult * scale
	output := gametext.NewOutput(font, lineSpacing, maxLinesForOutput, maxCharsPerLine)
	output.EnableJournalHistory()

	return font, output
}

func percentRect(bounds image.Rectangle, startX, startY, endX, endY float64) image.Rectangle {
	return image.Rect(
		bounds.Min.X+int(float64(bounds.Dx())*startX),
		bounds.Min.Y+int(float64(bounds.Dy())*startY),
		bounds.Min.X+int(float64(bounds.Dx())*endX),
		bounds.Min.Y+int(float64(bounds.Dy())*endY),
	)
}

func rectanglePlacement(bounds, rect image.Rectangle) sprites.PercentBasedPlacement {
	return sprites.PercentBasedPlacement{
		StartPercentX: float64(rect.Min.X-bounds.Min.X) / float64(bounds.Dx()),
		EndPercentX:   float64(rect.Max.X-bounds.Min.X) / float64(bounds.Dx()),
		StartPercentY: float64(rect.Min.Y-bounds.Min.Y) / float64(bounds.Dy()),
		EndPercentY:   float64(rect.Max.Y-bounds.Min.Y) / float64(bounds.Dy()),
	}
}

func (g *GameScene) syncRightRailLayout(bounds image.Rectangle) {
	layout := calculateRightRailLayout(bounds)
	if g.rightRail != layout {
		g.rightRail = layout
		if g.provisionSummary != nil {
			g.provisionSummary.SetBounds(layout.provisionContent)
		}
		if g.characterSummary != nil {
			g.characterSummary.SetBounds(layout.characterContent)
		}
		if g.activeEffectsPanel != nil {
			g.activeEffectsPanel.SetBounds(layout.activeBorder)
		}
	}
	if g.outputClipImage == nil || g.outputClipImage.Bounds().Size() != layout.outputContent.Size() {
		g.outputClipImage = ebiten.NewImage(layout.outputContent.Dx(), layout.outputContent.Dy())
	}
}
