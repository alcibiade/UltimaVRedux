package widgets

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestMerchantDialogShellUsesCompactTwoColumnLayout(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	shell := newTestMerchantDialogShell()

	outer := shell.GetBounds()
	header := shell.GetHeaderBounds()
	body := shell.GetBodyBounds()
	merchantPane := shell.GetMerchantPaneBounds()
	workspace := shell.GetWorkspaceBounds()
	portrait := shell.GetPortraitBounds()
	exchange := shell.GetExchangeBounds()
	footer := shell.GetFooterBounds()

	require.InDelta(t, 1280*0.80, outer.Dx(), 1)
	require.InDelta(t, 720*0.80, outer.Dy(), 1)
	require.Equal(t, header.Min.X, footer.Min.X)
	require.Equal(t, header.Max.X, footer.Max.X)
	require.Equal(t, body.Min.X, merchantPane.Min.X)
	require.Equal(t, body.Max.X, workspace.Max.X)
	require.Less(t, merchantPane.Max.X, workspace.Min.X)
	usableWidth := merchantPane.Dx() + workspace.Dx()
	require.InDelta(t, merchantShellMerchantPaneRatio, float64(merchantPane.Dx())/float64(usableWidth), 0.01)
	require.InDelta(t, 1-merchantShellMerchantPaneRatio, float64(workspace.Dx())/float64(usableWidth), 0.01)
	require.True(t, portrait.In(merchantPane))
	require.True(t, exchange.In(merchantPane))
	require.Less(t, portrait.Max.Y, exchange.Min.Y)
	require.Less(t, header.Max.Y, body.Min.Y)
	require.Less(t, body.Max.Y, footer.Min.Y)
	require.Equal(t, FontSizeBody, shell.GetFontSize())

	for _, bounds := range []image.Rectangle{header, body, merchantPane, workspace, portrait, exchange, footer} {
		require.True(t, bounds.In(outer), "%v must be inside %v", bounds, outer)
		require.Positive(t, bounds.Dx())
		require.Positive(t, bounds.Dy())
	}
}

func TestMerchantDialogShellUsesConfiguredBackground(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	shell := newTestMerchantDialogShell()
	screen := ebiten.NewImage(1280, 720)

	shell.DrawBackground(screen)
	point := shell.GetWorkspaceBounds().Min.Add(image.Pt(4, 4))

	requireColorRGBA(t, MerchantDialogBackgroundColor, screen.At(point.X, point.Y))
	r, g, b, a := MerchantDialogBackgroundColor.RGBA()
	require.Zero(t, r)
	require.Zero(t, g)
	require.Zero(t, b)
	require.Equal(t, uint32(0xffff), a)
}

func TestMerchantDialogShellUsesConnectedPanePerimeter(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	shell := newTestMerchantDialogShell()
	screen := ebiten.NewImage(1280, 720)

	shell.DrawBackground(screen)
	shell.DrawCommonRegions(screen)
	frames := shell.connectedFrameBounds()
	header := frames[0]
	outerFramePoint := image.Pt(header.Min.X+header.Dx()/2, header.Min.Y)

	requireColorRGBA(t, DialogRuleColor, screen.At(outerFramePoint.X, outerFramePoint.Y))
	paneCorner := header.Min
	requireColorNotRGBA(t, MerchantDialogBackgroundColor, screen.At(paneCorner.X, paneCorner.Y))
	require.Equal(t, header.Max.Y-shell.pixelUnit(), frames[1].Min.Y)
	require.Equal(t, frames[1].Max.X-shell.pixelUnit(), frames[3].Min.X)
	require.Equal(t, frames[1].Max.Y-shell.pixelUnit(), frames[2].Min.Y)
	require.Equal(t, frames[2].Max.Y-shell.pixelUnit(), frames[4].Min.Y)
	legacyOuter := shell.GetBounds()
	legacyOuterPoint := image.Pt(legacyOuter.Min.X+legacyOuter.Dx()/2, legacyOuter.Min.Y)
	requireColorNotRGBA(t, DialogRuleColor, screen.At(legacyOuterPoint.X, legacyOuterPoint.Y))
}

func TestMerchantDialogShellBoundsLatestExchangeAndGuidance(t *testing.T) {
	setMerchantShellTestResolution(t, 640, 360)
	longText := strings.Repeat("A finalized merchant sentence stays in its assigned region. ", 50)
	shell := NewMerchantDialogShell(MerchantDialogShellConfig{
		Header: MerchantDialogHeaderView{
			ShopIdentity: longText,
			Owner:        longText,
			PartyGold:    longText,
		},
		Exchange: MerchantDialogExchangeView{Text: longText},
		Footer:   MerchantDialogFooterView{Guidance: longText},
	})

	require.LessOrEqual(t, shell.headerOutput.GetPanelContentLineCount(), merchantShellHeaderReservedLines)
	require.Greater(t, shell.exchangePageCount(), 1)
	require.Equal(t, strings.Join(strings.Fields(longText), " "),
		strings.Join(strings.Fields(shell.exchangeOutput.GetOutputStr(true)), " "))
	require.LessOrEqual(
		t,
		shell.footerOutput.GetPanelContentLineCount(),
		shell.footerOutput.GetLineCapacity(shell.footerText.Dy()),
	)
	require.Contains(t, shell.headerOutput.GetOutputStr(false), "...")
	require.NotContains(t, shell.exchangeOutput.GetOutputStr(false), "...")
	require.Contains(t, shell.footerOutput.GetOutputStr(false), "...")
	shell.ensureLayout()
}

func TestMerchantDialogShellFooterOutputPreservesPlainGuidance(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	guidance := "[Up/Down] Select   [Enter] Buy   [Esc] Leave"
	shell := NewMerchantDialogShell(MerchantDialogShellConfig{
		Footer: MerchantDialogFooterView{Guidance: guidance},
	})

	require.Equal(t, guidance, shell.GetFooterView().Guidance)
	require.Equal(t, guidance, shell.footerOutput.GetOutputStr(false))

	twoLineGuidance := "[Enter] Confirm\n[Esc] Leave"
	shell.Refresh(MerchantDialogHeaderView{}, MerchantDialogExchangeView{}, MerchantDialogFooterView{
		Guidance: twoLineGuidance,
	})
	rendered := strings.ReplaceAll(shell.footerOutput.GetOutputStr(false), " \n", "\n")
	require.Equal(t, twoLineGuidance, rendered)
}

func TestMerchantDialogShellRefreshAndResolutionRebuildPreserveData(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	shell := newTestMerchantDialogShell()
	shell.Refresh(
		MerchantDialogHeaderView{ShopIdentity: "The Rusty Sword", Owner: "Ronan", PartyGold: "975"},
		MerchantDialogExchangeView{Text: "A refreshed exchange."},
		MerchantDialogFooterView{Guidance: "[Enter] Confirm   [Esc] Leave"},
	)
	initialWorkspace := shell.GetWorkspaceBounds()

	setResponsiveDialogTestResolution(t, 1920, 1080)
	largeWorkspace := shell.GetWorkspaceBounds()

	require.Greater(t, largeWorkspace.Dx(), initialWorkspace.Dx())
	require.Greater(t, largeWorkspace.Dy(), initialWorkspace.Dy())
	require.Equal(t, "The Rusty Sword", shell.GetHeaderView().ShopIdentity)
	require.Equal(t, "A refreshed exchange.", shell.GetExchangeView().Text)
	require.Equal(t, "[Enter] Confirm   [Esc] Leave", shell.GetFooterView().Guidance)
	require.Contains(t, shell.headerOutput.GetOutputStr(false), "The Rusty Sword")
	require.Equal(t, "A refreshed exchange.", strings.Join(strings.Fields(shell.exchangeOutput.GetOutputStr(false)), " "))
}

func TestMerchantDialogShellDrawsMerchantSpriteAndNameInPortrait(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	sprite := ebiten.NewImage(16, 16)
	sprite.Fill(DialogHeaderColor)
	shell := NewMerchantDialogShell(MerchantDialogShellConfig{
		Header:         MerchantDialogHeaderView{Owner: "Gwenneth"},
		MerchantSprite: sprite,
	})
	screen := ebiten.NewImage(1280, 720)

	shell.DrawBackground(screen)
	shell.DrawCommonRegions(screen)
	center := shell.portraitSprite.Min.Add(shell.portraitSprite.Size().Div(2))
	r, g, b, a := screen.At(center.X, center.Y).RGBA()

	require.Equal(t, uint32(0xffff), r)
	require.Equal(t, uint32(0xffff), g)
	require.Equal(t, uint32(0xffff), b)
	require.Equal(t, uint32(0xffff), a)
	require.Contains(t, shell.ownerOutput.GetOutputStr(false), "Gwenneth")
}

func newTestMerchantDialogShell() *MerchantDialogShell {
	return NewMerchantDialogShell(MerchantDialogShellConfig{
		Header: MerchantDialogHeaderView{
			ShopIdentity: "The Wayfarer's Arms",
			Owner:        "Gwenneth",
			PartyGold:    "500",
		},
		Exchange: MerchantDialogExchangeView{Text: "Welcome, Avatar."},
		Footer:   MerchantDialogFooterView{Guidance: "Enter: Select  Esc: Leave"},
	})
}

func setMerchantShellTestResolution(t *testing.T, width, height int) {
	t.Helper()
	setResponsiveDialogTestResolution(t, width, height)
}

func requireColorEqual(t *testing.T, expected, actual color.Color) {
	t.Helper()
	expectedRed, expectedGreen, expectedBlue, expectedAlpha := expected.RGBA()
	actualRed, actualGreen, actualBlue, actualAlpha := actual.RGBA()
	require.Equal(t, expectedRed, actualRed)
	require.Equal(t, expectedGreen, actualGreen)
	require.Equal(t, expectedBlue, actualBlue)
	require.Equal(t, expectedAlpha, actualAlpha)
}
