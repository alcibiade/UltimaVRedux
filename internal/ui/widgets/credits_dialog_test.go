package widgets

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func longTestCredits() []text.CreditParagraph {
	document := text.OriginalAndReduxCredits()
	for range 20 {
		document = append(document, text.CreditParagraph{Label: "Supplied attribution", Value: strings.Repeat("Long credit with original wording. ", 10)},
			text.CreditParagraph{Label: "Source", Value: "https://example.org/" + strings.Repeat("long-unbroken-source", 12)})
	}
	return append(document, text.SpecialThanksCredits()...)
}

func TestCreditsOpeningPauseScrollingPauseCompletionAndReplay(t *testing.T) {
	setGuidanceTestResolution(t, 1280, 720)
	var now int64
	keyboard := newWidgetTestKeyboard(0)
	closed := 0
	dialog := NewCreditsDialog(longTestCredits(), keyboard, func() int64 { return now }, func() { closed++ })
	press := func(key ebiten.Key) { keyboard.SimulateKeyPress(key); dialog.Update() }
	require.Zero(t, dialog.offset)
	now = 1999
	dialog.Update()
	require.Zero(t, dialog.offset)
	now = 3000
	dialog.Update()
	require.InDelta(t, 0.5, dialog.offset, 0.00001)
	press(ebiten.KeySpace)
	require.Equal(t, "Paused", dialog.stateLabel())
	now += 100_000
	dialog.Update()
	require.InDelta(t, 0.5, dialog.offset, 0.00001)
	press(ebiten.KeySpace)
	now += 1000
	dialog.Update()
	require.InDelta(t, 1, dialog.offset, 0.00001, "resume must not catch up paused time")
	press(ebiten.KeyDown)
	require.Equal(t, 2.0, dialog.offset)
	require.True(t, dialog.paused)
	press(ebiten.KeyHome)
	press(ebiten.KeyUp)
	require.Zero(t, dialog.offset)
	press(ebiten.KeyPageDown)
	require.Equal(t, float64(dialog.metrics.GetLineCapacity(dialog.contentBounds().Dy())), dialog.offset)
	press(ebiten.KeyPageUp)
	require.Zero(t, dialog.offset)
	press(ebiten.KeySpace)
	now += int64(math.Ceil(dialog.lastOffset*2000)) + 2000
	dialog.Update()
	require.Equal(t, "Complete", dialog.stateLabel())
	require.Equal(t, dialog.lastOffset, dialog.offset)
	require.Zero(t, closed)
	press(ebiten.KeySpace)
	require.Equal(t, dialog.lastOffset, dialog.offset, "Space at the end never silently restarts")
	press(ebiten.KeyHome)
	require.Equal(t, "Paused", dialog.stateLabel())
	press(ebiten.KeySpace)
	now += 1000
	dialog.Update()
	require.Equal(t, 0.5, dialog.offset)
	press(ebiten.KeyEnd)
	require.Equal(t, dialog.lastOffset, dialog.offset)
	press(ebiten.KeyEscape)
	dialog.Update()
	require.Equal(t, 1, closed)
}

func TestCreditsShortAndEmptyDocumentsRestComplete(t *testing.T) {
	setGuidanceTestResolution(t, 1280, 720)
	for _, document := range [][]text.CreditParagraph{nil, {{Value: "Thank you."}}} {
		var now int64
		keyboard := newWidgetTestKeyboard(0)
		dialog := NewCreditsDialog(document, keyboard, func() int64 { return now }, func() {})
		now = 1_000_000
		dialog.Update()
		require.Zero(t, dialog.offset)
		require.Equal(t, "Complete", dialog.stateLabel())
		keyboard.SimulateKeyPress(ebiten.KeyEscape)
		dialog.Update()
	}
}

func TestCreditsReflowRetainsLogicalReadingLocationAndCompletion(t *testing.T) {
	setGuidanceTestResolution(t, 1280, 720)
	keyboard := newWidgetTestKeyboard(0)
	document := longTestCredits()
	dialog := NewCreditsDialog(document, keyboard, func() int64 { return 0 }, func() {})
	document[0].Heading = "external mutation"
	require.NotEqual(t, document[0], dialog.document[0])
	dialog.offset = float64(dialog.paragraphs[20].start) + 0.5
	dialog.paused = true
	anchor, fraction := dialog.readingAnchor()
	setGuidanceTestResolution(t, 1920, 1080)
	dialog.Update()
	newAnchor, newFraction := dialog.readingAnchor()
	require.Equal(t, anchor, newAnchor)
	require.InDelta(t, fraction, newFraction, 0.00001)
	require.True(t, dialog.paused)
	keyboard.SimulateKeyPress(ebiten.KeyEnd)
	dialog.Update()
	setGuidanceTestResolution(t, 1280, 720)
	dialog.Update()
	require.True(t, dialog.complete())
	require.Equal(t, dialog.lastOffset, dialog.offset)
	keyboard.SimulateKeyPress(ebiten.KeyEscape)
	dialog.Update()
}

func TestCreditsMeasuredLayoutClipsOnlyBodyAndPreservesFixedControls(t *testing.T) {
	for _, size := range config.SupportedWindowResolutions() {
		setGuidanceTestResolution(t, size.X, size.Y)
		keyboard := newWidgetTestKeyboard(0)
		var now int64
		dialog := NewCreditsDialog(longTestCredits(), keyboard, func() int64 { return now }, func() {})
		body := dialog.contentBounds()
		require.True(t, body.In(dialog.border.GetInteriorBounds()))
		require.Positive(t, body.Dy())
		for _, paragraph := range dialog.paragraphs {
			for _, line := range strings.Split(paragraph.output.GetOutputStr(true), "\n") {
				width, _ := paragraph.output.Font.MeasureString(strings.TrimRight(line, " "))
				require.LessOrEqual(t, width, float64(body.Dx()), line)
			}
		}
		for _, colour := range []color.Color{DialogControlKeyColor, DialogControlHintColor, DialogLabelColor, DialogValueColor} {
			_, found := dialog.footer.FirstPanelLineWithColor(0, creditsFooterRows, colour)
			require.True(t, found)
		}
		for _, control := range []string{"Space", "Up/Down", "PgUp/PgDn", "Shift+Up/Down", "Home/End", "Esc"} {
			require.Contains(t, dialog.footer.GetOutputStr(false), control)
		}
		// Keep the complete measured layout table; clipping/repeated-draw
		// ownership needs only the tight and fractional typography cases.
		if size.X != 1280 && size.X != 1366 {
			continue
		}
		screen := ebiten.NewImage(size.X, size.Y)
		t.Cleanup(screen.Deallocate)
		dialog.Draw(screen)
		before := creditChromePixels(screen, body)
		canvas := dialog.canvas
		for _, elapsed := range []int64{2500, 3000, 100_000} {
			now = elapsed
			dialog.Update()
			offset := dialog.offset
			dialog.Draw(screen)
			dialog.Draw(screen)
			require.Equal(t, offset, dialog.offset, "rendering cannot advance the roll")
			require.Equal(t, before, creditChromePixels(screen, body), "body text cannot bleed into the fixed header/footer")
			require.Same(t, canvas, dialog.canvas)
		}
		keyboard.SimulateKeyPress(ebiten.KeyEscape)
		dialog.Update()
	}
}

func creditChromePixels(screen *ebiten.Image, body image.Rectangle) []byte {
	pixels := make([]byte, screen.Bounds().Dx()*screen.Bounds().Dy()*4)
	screen.ReadPixels(pixels)
	for y := body.Min.Y; y < body.Max.Y; y++ {
		start := (y*screen.Bounds().Dx() + body.Min.X) * 4
		clear(pixels[start : start+body.Dx()*4])
	}
	return pixels
}

func TestCreditsHeldOpeningKeysAndShiftPagingAreReleaseSafe(t *testing.T) {
	setGuidanceTestResolution(t, 1280, 720)
	held := map[ebiten.Key]bool{ebiten.KeyEnter: true}
	keyboard := input.NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return held[key] })
	closed := 0
	dialog := NewCreditsDialog(longTestCredits(), keyboard, func() int64 { return 0 }, func() { closed++ })
	for range 20 {
		dialog.Update()
	}
	require.False(t, dialog.paused)
	require.Zero(t, closed)
	delete(held, ebiten.KeyEnter)
	dialog.Update()
	held[ebiten.KeyShiftLeft], held[ebiten.KeyDown] = true, true
	keyboard.SetAllowKeyPressImmediately()
	dialog.Update()
	require.Equal(t, float64(dialog.metrics.GetLineCapacity(dialog.contentBounds().Dy())), dialog.offset)
	require.True(t, dialog.paused)
	clear(held)
	dialog.Update()
	held[ebiten.KeyEscape] = true
	keyboard.SetAllowKeyPressImmediately()
	dialog.Update()
	dialog.Update()
	require.Equal(t, 1, closed)
}
