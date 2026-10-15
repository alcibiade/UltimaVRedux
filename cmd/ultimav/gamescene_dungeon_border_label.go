package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// A dungeon readout owns only its backing. Output retains semantic text and the
// text package owns glyph caching; the frame is an immutable borrowed raster.
type dungeonBorderLabel struct {
	key           dungeonBorderLabelKey
	backing       *ebiten.Image
	backingBounds image.Rectangle
	textBounds    image.Rectangle
	borderStroke  int
	borderBlue    color.RGBA
}

type dungeonBorderLabelKey struct {
	value    string
	output   *text.Output
	font     *text.UltimaFont
	face     *etext.GoTextFace
	fontSize float64
	frame    *ebiten.Image
	band     image.Rectangle
	padding  int
}

func (label *dungeonBorderLabel) draw(
	screen *ebiten.Image, output *text.Output, value string, frame *ebiten.Image, band image.Rectangle,
) {
	key := dungeonBorderLabelKey{
		value: value, output: output, font: output.Font, face: output.Font.TextFace,
		fontSize: output.Font.TextFace.Size, frame: frame, band: band,
		padding: max(1, int(math.Round(text.GetScaledNumberToResolution(display.GetManager(), windTextPadding)))),
	}
	label.ensure(key)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(label.backingBounds.Min.X), float64(label.backingBounds.Min.Y))
	screen.DrawImage(label.backing, op)
	output.DrawPanelContent(screen, label.textBounds.Min, label.textBounds.Dx(), etext.AlignCenter)
}

func (label *dungeonBorderLabel) ensure(key dungeonBorderLabelKey) {
	if label.backing != nil && label.key == key {
		return
	}
	if label.backing == nil || label.key.frame != key.frame || label.key.band != key.band {
		label.borderStroke, label.borderBlue = mapBorderBandStyle(key.frame, key.band)
	}
	label.key = key
	key.output.Reset()
	key.output.AppendLine().White(key.value).Done()
	width, _ := key.font.MeasureString(key.value)
	height := int(math.Ceil(key.output.GetFontHeight()))
	label.backingBounds, label.textBounds = dungeonBorderLabelBounds(key.band, int(math.Ceil(width)), height, key.padding)
	if label.backing != nil {
		label.backing.Deallocate()
	}
	label.backing = ebiten.NewImage(label.backingBounds.Dx(), label.backingBounds.Dy())
	label.backing.Fill(ucolor.EGABlack)
	drawMapBorderBookends(label.backing, label.borderStroke, label.borderBlue)
}

func dungeonBorderLabelBounds(band image.Rectangle, textWidth, textHeight, padding int) (image.Rectangle, image.Rectangle) {
	capWidth := max(1, band.Dy()/2)
	width := textWidth + 2*(capWidth+padding)
	x := mapFrameCenteredX(band, width)
	backing := image.Rect(x, band.Min.Y, x+width, band.Max.Y)
	textX := x + capWidth + padding
	textY := band.Min.Y + (band.Dy()-textHeight)/2

	return backing, image.Rect(textX, textY, textX+textWidth, textY+textHeight)
}

func (label *dungeonBorderLabel) release() {
	if label.backing != nil {
		label.backing.Deallocate()
	}
	*label = dungeonBorderLabel{}
}

func (g *GameScene) drawDungeonLevelLabel(screen *ebiten.Image, status dungeonHUDStatus) {
	outer := g.borders.outsideBorder.Bounds()
	band := image.Rect(outer.Min.X, outer.Min.Y, outer.Max.X, g.borders.surfaceAperture.Min.Y)
	g.dungeonHUD.levelLabel.draw(screen, g.dungeonHUD.level, status.levelText(), g.borders.outsideBorder, band)
}

func (g *GameScene) drawDungeonDirectionLabel(screen *ebiten.Image, status dungeonHUDStatus) {
	outer := g.borders.outsideBorder.Bounds()
	band := image.Rect(outer.Min.X, g.borders.surfaceAperture.Max.Y, outer.Max.X, outer.Max.Y)
	g.dungeonHUD.directionLabel.draw(screen, g.dungeonHUD.direction, status.directionText(), g.borders.outsideBorder, band)
}
