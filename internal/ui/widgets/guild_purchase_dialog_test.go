package widgets

import (
	"image"
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	pkgcolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestGuildPurchaseDialogConstructorContracts(t *testing.T) {
	quote := testGuildShopState().Offers[0]
	valid := GuildPurchaseDialogConfig{
		Keyboard: newWidgetTestKeyboard(0),
		Quote:    quote,
		Pitch:    "Three keys for one hundred ninety gold.",
	}

	tests := []struct {
		name   string
		mutate func(*GuildPurchaseDialogConfig)
	}{
		{name: "keyboard", mutate: func(config *GuildPurchaseDialogConfig) { config.Keyboard = nil }},
		{name: "pitch", mutate: func(config *GuildPurchaseDialogConfig) { config.Pitch = " " }},
		{name: "provision", mutate: func(config *GuildPurchaseDialogConfig) {
			config.Quote.Provision = references.NoProvision
		}},
		{name: "bundle", mutate: func(config *GuildPurchaseDialogConfig) { config.Quote.BundleQuantity = 0 }},
		{name: "price", mutate: func(config *GuildPurchaseDialogConfig) { config.Quote.PricePerBundle = 0 }},
		{name: "maximum", mutate: func(config *GuildPurchaseDialogConfig) { config.Quote.MaxBundles++ }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.mutate(&config)
			require.Panics(t, func() { NewGuildPurchaseDialog(config) })
		})
	}

	require.NotPanics(t, func() { NewGuildPurchaseDialog(valid) })
}

func TestGuildPurchaseDialogQuantityBoundsProjectionAndSingleRequest(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	keyboard := newWidgetTestKeyboard(0)
	quote := testGuildShopState().Offers[2]
	dialog := NewGuildPurchaseDialog(GuildPurchaseDialogConfig{
		Keyboard: keyboard,
		Quote:    quote,
		Pitch:    "Five torches for twelve gold.",
	})

	require.Equal(t, uint16(1), dialog.GetBundleCount())
	require.Equal(t, FontSizeMainScreen, dialog.GetFontSize())
	require.Equal(t, "Five torches for twelve gold.", dialog.GetPitch())
	require.Contains(t, dialog.GetBodyText(), `"Five torches for twelve`)
	require.Contains(t, dialog.GetBodyText(), `gold."`)
	require.Contains(t, dialog.GetBodyText(), "BUNDLES")
	require.Contains(t, dialog.GetBodyText(), "<   1   >")
	require.Contains(t, dialog.GetBodyText(), "of 14")
	require.Contains(t, dialog.GetBodyText(), "5 TORCHES per bundle")
	require.Contains(t, dialog.GetBodyText(), "TORCHES  30  ->  35")
	require.Contains(t, dialog.GetBodyText(), "GOLD  500  ->  488")
	require.NotContains(t, strings.ToUpper(dialog.GetBodyText()), "TOTAL PRICE")
	require.Contains(t, dialog.totalOutput.GetOutputStr(false), "TOTAL PRICE")
	require.Contains(t, dialog.totalOutput.GetOutputStr(false), "12 GOLD")
	require.Equal(t, "Left/Right: Quantity  M: Max\nEnter: Confirm  Esc: Back", dialog.GetFooterGuidance())
	require.Equal(t, guildPurchaseFooterLines, dialog.footerOutput.GetPanelContentLineCount())
	require.Contains(t, dialog.footerOutput.GetOutputStr(false), "Left/Right: Quantity  M: Max")
	require.Contains(t, dialog.footerOutput.GetOutputStr(false), "Enter: Confirm  Esc: Back")
	require.NotContains(t, dialog.GetFooterGuidance(), "Up/Down")

	pressGuildPurchaseKey(dialog, keyboard, ebiten.KeyLeft)
	require.Equal(t, uint16(1), dialog.GetBundleCount())
	pressGuildPurchaseKey(dialog, keyboard, ebiten.KeyDown)
	require.Equal(t, uint16(2), dialog.GetBundleCount())
	pressGuildPurchaseKey(dialog, keyboard, ebiten.KeyUp)
	require.Equal(t, uint16(1), dialog.GetBundleCount())
	pressGuildPurchaseKey(dialog, keyboard, ebiten.KeyRight)
	require.Equal(t, uint16(2), dialog.GetBundleCount())
	pressGuildPurchaseKey(dialog, keyboard, ebiten.KeyLeft)
	require.Equal(t, uint16(1), dialog.GetBundleCount())
	pressGuildPurchaseKey(dialog, keyboard, ebiten.KeyM)
	require.Equal(t, quote.MaxBundles, dialog.GetBundleCount())
	require.Contains(t, dialog.GetBodyText(), "<   14   >")
	require.Contains(t, dialog.GetBodyText(), "TORCHES  30  ->  99")
	require.Contains(t, dialog.totalOutput.GetOutputStr(false), "168 GOLD")

	pressGuildPurchaseKey(dialog, keyboard, ebiten.KeyEnter)
	require.True(t, dialog.IsDismissed())
	require.False(t, dialog.IsCancelled())
	require.True(t, dialog.HasPendingPurchaseRequest())
	request, ok := dialog.ConsumePendingPurchaseRequest()
	require.True(t, ok)
	require.Equal(t, game_state.GuildPurchaseRequest{
		Provision:   references.Torches,
		BundleCount: quote.MaxBundles,
	}, request)

	pressGuildPurchaseKey(dialog, keyboard, ebiten.KeyEnter)
	_, duplicate := dialog.ConsumePendingPurchaseRequest()
	require.False(t, duplicate, "dismissed modal must not emit duplicate requests")
}

func TestGuildPurchaseDialogSaturatesProjectedInventoryAt99(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	quote := testGuildQuote(references.Key, 3, 10, 98, 100)
	dialog := NewGuildPurchaseDialog(GuildPurchaseDialogConfig{
		Keyboard: keyboard,
		Quote:    quote,
		Pitch:    "A final bundle of keys.",
	})

	require.Equal(t, uint16(1), quote.MaxBundles)
	require.Contains(t, dialog.GetBodyText(), "KEYS  98  ->  99")
	require.NotContains(t, strings.ToLower(dialog.GetBodyText()), "overflow")
	require.NotContains(t, strings.ToLower(dialog.GetBodyText()), "full charge")
}

func TestGuildPurchaseDialogZeroMaxStatesAllowOnlyEscape(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	tests := []struct {
		name     string
		quote    game_state.GuildOfferQuote
		note     string
		response string
	}{
		{
			name:     "inventory full",
			quote:    testGuildQuote(references.Key, 3, 10, 99, 100),
			note:     "Inventory max: 99.",
			response: `"Thou canst carry no more keys."`,
		},
		{
			name:     "insufficient money",
			quote:    testGuildQuote(references.Key, 3, 10, 10, 1),
			note:     "Insufficient money.",
			response: `"Thou hast not enough gold for even one bundle."`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keyboard := newWidgetTestKeyboard(0)
			dialog := NewGuildPurchaseDialog(GuildPurchaseDialogConfig{
				Keyboard: keyboard,
				Quote:    test.quote,
				Pitch:    "Original merchant pitch.",
			})

			require.Zero(t, dialog.GetBundleCount())
			require.Zero(t, test.quote.MaxBundles)
			compactBody := strings.Join(strings.Fields(dialog.GetBodyText()), " ")
			require.Contains(t, compactBody, test.response)
			require.Contains(t, dialog.GetBodyText(), test.note)
			require.True(t, dialog.totalBounds.Empty())
			require.Zero(t, dialog.totalOutput.GetPanelContentLineCount())
			require.Equal(t, "Esc: Back", dialog.GetFooterGuidance())
			require.Equal(t, 1, dialog.footerOutput.GetPanelContentLineCount())
			screen := ebiten.NewImage(1280, 720)
			t.Cleanup(screen.Dispose)
			dialog.Draw(screen)
			requireImageRegionContainsColor(t, screen, dialog.frame.bodyText, pkgcolor.EGARed)
			requireOutputVerticallyCentered(t, dialog.frame.footerText, dialog.footerOutput)
			pressGuildPurchaseKey(dialog, keyboard, ebiten.KeyEnter)
			require.False(t, dialog.IsDismissed())
			require.False(t, dialog.HasPendingPurchaseRequest())

			pressGuildPurchaseKey(dialog, keyboard, ebiten.KeyEscape)
			require.True(t, dialog.IsDismissed())
			require.True(t, dialog.IsCancelled())
		})
	}
}

func TestGuildPurchaseDialogIsCenteredOverGuildShopAndRebuilds(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	shop := newTestGuildShopDialog(newWidgetTestKeyboard(0), testGuildShopState())
	keyboard := newWidgetTestKeyboard(0)
	dialog := NewGuildPurchaseDialog(GuildPurchaseDialogConfig{
		Keyboard: keyboard,
		Quote:    testGuildShopState().Offers[0],
		Pitch:    "Original keys pitch.",
	})
	pressGuildPurchaseKey(dialog, keyboard, ebiten.KeyRight)
	initial := dialog.GetBounds()
	wantBody := dialog.GetBodyText()

	require.True(t, initial.In(shop.GetShell().GetBounds()))
	require.InDelta(t, 640, initial.Min.X+initial.Dx()/2, 1)
	require.Contains(t, dialog.GetBodyText(), "Original keys pitch.")

	setResponsiveDialogTestResolution(t, 1920, 1080)
	dialog.ensureLayout()
	require.Greater(t, dialog.GetBounds().Dx(), initial.Dx())
	require.Greater(t, dialog.GetBounds().Dy(), initial.Dy())
	require.Equal(t, uint16(2), dialog.GetBundleCount())
	require.Equal(t, wantBody, dialog.GetBodyText())
}

func TestGuildPurchaseDialogQuotesPitchExactlyOnceWithoutChangingStoredPitch(t *testing.T) {
	for _, test := range []struct {
		name  string
		pitch string
	}{
		{name: "unquoted", pitch: "Three keys for one hundred ninety gold."},
		{name: "already quoted", pitch: `"Three keys for one hundred ninety gold."`},
		{name: "closing quote only", pitch: `Three keys for one hundred ninety gold."`},
	} {
		t.Run(test.name, func(t *testing.T) {
			dialog := NewGuildPurchaseDialog(GuildPurchaseDialogConfig{
				Keyboard: newWidgetTestKeyboard(0),
				Quote:    testGuildShopState().Offers[0],
				Pitch:    test.pitch,
			})
			body := dialog.GetBodyText()
			pitchEnd := strings.Index(body, "BUNDLES")
			require.Positive(t, pitchEnd)
			presentedPitch := body[:pitchEnd]

			require.Equal(t, test.pitch, dialog.GetPitch())
			require.Equal(t, 2, strings.Count(presentedPitch, `"`))
			require.True(t, strings.HasPrefix(presentedPitch, `"`))
			require.Contains(t, presentedPitch, `gold."`)
		})
	}
}

func TestGuildPurchaseDialogOptionBGeometryAndColorHierarchy(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	dialog := NewGuildPurchaseDialog(GuildPurchaseDialogConfig{
		Keyboard: newWidgetTestKeyboard(0),
		Quote:    testGuildShopState().Offers[0],
		Pitch:    "Keys pitch.",
	})
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Dispose)
	dialog.Draw(screen)

	require.Contains(t, dialog.headerOutput.GetOutputStr(false), "PURCHASE KEYS")
	require.Equal(t, dialog.frame.bodyBounds.Max.Y, dialog.frame.footerBounds.Min.Y)
	require.False(t, dialog.totalBounds.Empty())
	require.Less(t, dialog.frame.bodyText.Max.Y, dialog.totalBounds.Min.Y)
	require.LessOrEqual(t, dialog.totalBounds.Max.Y, dialog.frame.bodyBounds.Max.Y)

	headerFramePoint := image.Pt(
		dialog.frame.headerBounds.Min.X+dialog.frame.pixelUnit()*4,
		dialog.frame.headerBounds.Min.Y,
	)
	requireColorRGBA(t, DialogRuleColor, screen.At(headerFramePoint.X, headerFramePoint.Y))
	requireOutputVerticallyCentered(t, dialog.frame.headerText, dialog.headerOutput)
	requireOutputVerticallyCentered(t, dialog.frame.footerText, dialog.footerOutput)

	selectorLine := guildPurchaseBodyLineRegion(dialog, 3)
	requireImageRegionContainsColor(t, screen, selectorLine, pkgcolor.EGAYellow)
	requireImageRegionContainsColor(t, screen, selectorLine, pkgcolor.EGAWhite)
	projectionLine := guildPurchaseBodyLineRegion(dialog, 7)
	requireImageRegionContainsColor(t, screen, projectionLine, pkgcolor.EGALightGray)
	requireImageRegionContainsColor(t, screen, projectionLine, pkgcolor.EGACyan)
	requireImageRegionContainsColor(t, screen, projectionLine, pkgcolor.EGAWhite)
	requireImageRegionContainsColor(t, screen, projectionLine, pkgcolor.EGALightGreen)
	requireImageRegionContainsColor(t, screen, dialog.totalText, pkgcolor.EGAYellow)
	requireImageRegionContainsColor(t, screen, dialog.totalText, pkgcolor.EGAWhite)
}

func requireOutputVerticallyCentered(t *testing.T, bounds image.Rectangle, output interface {
	GetPanelContentLineCount() int
	GetTotalHeightForLines(int) float64
}) {
	t.Helper()
	lineCount := output.GetPanelContentLineCount()
	textHeight := int(math.Ceil(output.GetTotalHeightForLines(lineCount)))
	startY := bounds.Min.Y + max(0, (bounds.Dy()-textHeight)/guildPurchaseCenterDivisor)
	topSpace := startY - bounds.Min.Y
	bottomSpace := bounds.Max.Y - (startY + textHeight)
	require.InDelta(t, topSpace, bottomSpace, 1)
}

func guildPurchaseBodyLineRegion(dialog *GuildPurchaseDialog, line int) image.Rectangle {
	top := dialog.frame.bodyText.Min.Y + int(math.Round(float64(line)*dialog.bodyOutput.GetLineSpacing()))
	height := int(math.Ceil(dialog.bodyOutput.GetFontHeight()))

	return image.Rect(
		dialog.frame.bodyText.Min.X,
		top,
		dialog.frame.bodyText.Max.X,
		min(dialog.frame.bodyText.Max.Y, top+height),
	)
}

func pressGuildPurchaseKey(dialog *GuildPurchaseDialog, keyboard *input.Keyboard, key ebiten.Key) {
	keyboard.SimulateKeyPress(key)
	dialog.Update()
}

func TestGuildPurchaseDialogPhysicalZeroMaximumCannotConfirm(t *testing.T) {
	keys := newPhysicalQuantityKeys()
	dialog := NewGuildPurchaseDialog(GuildPurchaseDialogConfig{
		Keyboard: keys.keyboard, Quote: testGuildQuote(references.Key, 3, 10, 99, 100),
		Pitch: "A bundle of keys.",
	})
	for _, key := range []ebiten.Key{ebiten.KeyLeft, ebiten.KeyRight, ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyM, ebiten.KeyEnter} {
		keys.held[key] = true
		for range 60 {
			keys.tick(dialog)
		}
		require.Zero(t, dialog.GetBundleCount())
		require.False(t, dialog.HasPendingPurchaseRequest())
		require.False(t, dialog.IsDismissed())
		keys.release(dialog)
	}
	keys.held[ebiten.KeyEscape] = true
	keys.tick(dialog)
	require.True(t, dialog.IsCancelled())
}
