package widgets

import (
	"image"
	stdcolor "image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

const classicPaneCenterDivisor = 2

// connectClassicPaneGrid normalizes an ordered row grid so adjacent frames
// overlap by exactly one classic UI pixel while preserving the outer bounds.
func connectClassicPaneGrid(rows [][]image.Rectangle, pixelUnit int) []image.Rectangle {
	pixelUnit = max(1, pixelUnit)
	connectedRows := make([][]image.Rectangle, len(rows))
	frameCount := 0
	for rowIndex, row := range rows {
		connectedRows[rowIndex] = append([]image.Rectangle(nil), row...)
		frameCount += len(row)
		for column := 0; column+1 < len(row); column++ {
			boundary := (row[column].Max.X + row[column+1].Min.X) / classicPaneCenterDivisor
			connectedRows[rowIndex][column].Max.X = boundary + pixelUnit
			connectedRows[rowIndex][column+1].Min.X = boundary
		}
	}

	for rowIndex := 0; rowIndex+1 < len(connectedRows); rowIndex++ {
		if len(rows[rowIndex]) == 0 || len(rows[rowIndex+1]) == 0 {
			continue
		}
		boundary := (rows[rowIndex][0].Max.Y + rows[rowIndex+1][0].Min.Y) / classicPaneCenterDivisor
		for column := range connectedRows[rowIndex] {
			connectedRows[rowIndex][column].Max.Y = boundary + pixelUnit
		}
		for column := range connectedRows[rowIndex+1] {
			connectedRows[rowIndex+1][column].Min.Y = boundary
		}
	}

	frames := make([]image.Rectangle, 0, frameCount)
	for _, row := range connectedRows {
		frames = append(frames, row...)
	}

	return frames
}

func fillClassicPaneBackgrounds(
	screen *ebiten.Image,
	frames []image.Rectangle,
	pixelUnit int,
	fillColor stdcolor.Color,
) {
	for _, frame := range frames {
		FillClassicPixelRoundedRect(screen, frame, pixelUnit, fillColor)
	}
}

func drawClassicPaneFrames(
	screen *ebiten.Image,
	frames []image.Rectangle,
	pixelUnit int,
	frameColor stdcolor.Color,
) {
	for _, frame := range frames {
		DrawClassicPixelRoundedFrame(screen, frame, pixelUnit, frameColor)
	}
}

// applyClassicPixelRoundedMask removes pixels outside the stepped silhouette
// from a fully rendered local buffer.
func applyClassicPixelRoundedMask(buffer *ebiten.Image, pixelUnit int) {
	if buffer == nil {
		return
	}
	bounds := buffer.Bounds()
	mask := ebiten.NewImageWithOptions(
		image.Rect(0, 0, bounds.Dx(), bounds.Dy()), &ebiten.NewImageOptions{Unmanaged: true})
	defer mask.Deallocate()
	FillClassicPixelRoundedRect(mask, mask.Bounds(), pixelUnit, stdcolor.White)
	op := new(ebiten.DrawImageOptions)
	op.Blend = ebiten.BlendDestinationIn
	buffer.DrawImage(mask, op)
}
