package widgets

import (
	"image"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ultimacolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// Draw renders the text input widget with prompt, placeholder, and cursor
func (t *TextInput) Draw(screen *ebiten.Image) {
	textRect := sprites.GetRectangleFromPercents(t.mainTextPlacement)
	currentX := float64(textRect.Min.X)
	currentY := float64(textRect.Min.Y)

	var promptWidth float64
	if t.promptPrefix != "" {
		promptColor := t.promptColor
		if promptColor == nil {
			promptColor = color.White
		}

		promptImg := text.GetCachedText(t.promptPrefix, t.ultimaFont.TextFace,
			&etext.LayoutOptions{LineSpacing: 0, PrimaryAlign: etext.AlignStart},
			promptColor)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(currentX, currentY)
		screen.DrawImage(promptImg, op)

		promptWidth, _ = etext.Measure(t.promptPrefix, t.ultimaFont.TextFace, 0)
		currentX += promptWidth
	}

	userInput := t.output.GetOutputStr(false)
	if userInput == "" && t.placeholderText != "" {
		placeholderColor := t.placeholderColor
		if placeholderColor == nil {
			placeholderColor = ultimacolor.EGADarkGray
		}

		placeholderImg := text.GetCachedText(t.placeholderText, t.ultimaFont.TextFace,
			&etext.LayoutOptions{LineSpacing: 0, PrimaryAlign: etext.AlignStart},
			placeholderColor)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(currentX, currentY)
		screen.DrawImage(placeholderImg, op)
	} else {
		inputColor := t.getTextColor()
		t.output.SetColor(inputColor)
		t.output.RecolorCurrentLine(inputColor)
		textRectWidth := textRect.Dx()
		t.output.DrawScrollingBuffer(screen, image.Point{
			X: int(currentX),
			Y: textRect.Min.Y,
		}, textRectWidth, false, etext.AlignStart, etext.AlignStart)
	}

	t.drawCursorWithOffset(screen, promptWidth)
}

func (t *TextInput) getTextColor() color.Color {
	if t.textCommands == nil {
		return t.inputColors.DefaultColor
	}

	textStr := t.GetText()
	matches := t.textCommands.OneOrMoreCommandsMatch(textStr)
	if matches {
		return t.inputColors.OneMatchColor
	} else {
		return t.inputColors.NoMatchesColor
	}
}

func (t *TextInput) drawCursorWithOffset(screen *ebiten.Image, promptWidth float64) {
	if !t.cursorVisible {
		return
	}

	if t.keyboard.GetMsSinceLastKeyPress() > pauseBlinkKeyPressedInMs {
		blink := time.Now().UnixMilli() % cursorBlinkRateInMs
		if blink <= cursorBlinkRateInMs/2 {
			return
		}
	}

	cursorPlacement := t.createCursorPlacement()
	textRect := sprites.GetRectangleFromPercents(cursorPlacement)

	inputWidth := t.CalculateTextWidth(t.output.GetOutputStr(false))

	fontHeight := t.getFontHeight()

	vector.FillRect(screen,
		float32(textRect.Min.X)+float32(promptWidth)+float32(inputWidth),
		float32(textRect.Min.Y),
		cursorWidth,
		float32(fontHeight),
		t.getTextColor(),
		false)
}

func (t *TextInput) createCursorPlacement() sprites.PercentBasedPlacement {
	cursorPlacement := sprites.PercentBasedPlacement{
		StartPercentX: t.mainTextPlacement.StartPercentX,
		EndPercentX:   t.mainTextPlacement.EndPercentX,
		StartPercentY: t.mainTextPlacement.StartPercentY - 0.003,
		EndPercentY:   t.mainTextPlacement.EndPercentY - 0.032,
	}

	return cursorPlacement
}
