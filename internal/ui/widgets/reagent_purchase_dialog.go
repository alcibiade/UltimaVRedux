package widgets

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

const (
	reagentPurchaseStartPercentX    = 0.27
	reagentPurchaseEndPercentX      = 0.73
	reagentPurchaseStartPercentY    = 0.12
	reagentPurchaseEndPercentY      = 0.88
	reagentPurchaseHeaderRatio      = 0.12
	reagentPurchaseFooterRatio      = 0.17
	reagentPurchaseMinimumInset     = 6
	reagentPurchaseLineSpacing      = 1.12
	reagentPurchaseMaximumBodyLines = 13
	reagentPurchaseMinimumTextChars = 8
	reagentPurchaseFooterLines      = 2
)

var _ Widget = (*ReagentPurchaseDialog)(nil)

// ReagentPurchaseDialogConfig configures one nested reagent quantity modal.
type ReagentPurchaseDialogConfig struct {
	Keyboard *input.Keyboard
	Quote    game_state.ReagentOfferQuote
	Preview  bool
}

// ReagentPurchaseDialog selects supplied approval views and emits one stable request.
type ReagentPurchaseDialog struct {
	keyboard    *input.Keyboard
	quote       game_state.ReagentOfferQuote
	preview     bool
	bundleCount uint16
	dismissed   bool
	cancelled   bool
	pending     *game_state.ReagentPurchaseRequest

	frame        *merchantModalFrame
	headerOutput *text.Output
	bodyOutput   *text.Output
	footerOutput *text.Output
}

// NewReagentPurchaseDialog creates a centered keyboard-only quantity modal.
func NewReagentPurchaseDialog(config ReagentPurchaseDialogConfig) *ReagentPurchaseDialog {
	if config.Keyboard == nil {
		panic("ReagentPurchaseDialog: keyboard is required")
	}
	validateReagentPurchaseQuote(config.Quote)
	dialog := &ReagentPurchaseDialog{
		keyboard:    config.Keyboard,
		quote:       cloneReagentOfferQuote(config.Quote),
		preview:     config.Preview,
		bundleCount: 1,
		dismissed:   false,
		cancelled:   false,
		pending:     nil,
		frame: newMerchantModalFrame(merchantModalFrameConfig{
			placement:             reagentPurchasePlacement(),
			headerRatio:           reagentPurchaseHeaderRatio,
			footerRatio:           reagentPurchaseFooterRatio,
			minimumInset:          reagentPurchaseMinimumInset,
			lineSpacingMultiplier: reagentPurchaseLineSpacing,
			footerMinimumLines:    reagentPurchaseFooterLines,
		}),
		headerOutput: nil,
		bodyOutput:   nil,
		footerOutput: nil,
	}
	dialog.ensureLayout()

	return dialog
}

// Draw renders the centered opaque purchase modal.
//

func (d *ReagentPurchaseDialog) Draw(screen *ebiten.Image) {
	if screen == nil || d.dismissed {
		return
	}
	d.ensureLayout()
	d.frame.draw(screen)
	d.headerOutput.DrawPanelContent(
		screen,
		image.Pt(d.frame.headerText.Min.X, verticallyCenteredOutputY(d.frame.headerText, d.headerOutput)),
		d.frame.headerText.Dx(),
		etext.AlignCenter,
	)
	d.bodyOutput.DrawPanelContent(screen, d.frame.bodyText.Min, d.frame.bodyText.Dx(), etext.AlignStart)
	d.footerOutput.DrawPanelContent(
		screen,
		image.Pt(d.frame.footerText.Min.X, verticallyCenteredOutputY(d.frame.footerText, d.footerOutput)),
		d.frame.footerText.Dx(),
		etext.AlignCenter,
	)
}

// Update processes quantity selection, confirmation, and cancellation.
func (d *ReagentPurchaseDialog) Update() {
	if d.dismissed {
		return
	}
	d.ensureLayout()
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		d.cancelled = true
		d.dismissed = true

		return
	}
	if d.handleQuantityInput() {
		return
	}
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) ||
		d.keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter) {
		request := game_state.ReagentPurchaseRequest{
			Reagent:     d.quote.Reagent,
			BundleCount: d.bundleCount,
		}
		d.pending = &request
		d.dismissed = true
	}
}

func (d *ReagentPurchaseDialog) handleQuantityInput() bool {
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyM) {
		d.setBundleCount(uint16(len(d.quote.ApprovalViews))) //nolint:gosec // Valid reagent quotes contain at most 99 views.

		return true
	}
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyLeft) ||
		d.keyboard.IsKeyPressedAndRegister(ebiten.KeyUp) {
		d.setBundleCount(max(uint16(1), d.bundleCount-1))

		return true
	}
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyRight) ||
		d.keyboard.IsKeyPressedAndRegister(ebiten.KeyDown) {
		maximum := uint16(len(d.quote.ApprovalViews)) //nolint:gosec // Valid quotes contain at most 99 views.
		d.setBundleCount(min(maximum, d.bundleCount+1))

		return true
	}

	return false
}

// HasPendingPurchaseRequest reports whether confirmation emitted an unconsumed request.
func (d *ReagentPurchaseDialog) HasPendingPurchaseRequest() bool { return d.pending != nil }

// ConsumePendingPurchaseRequest returns and clears the one confirmed request.
func (d *ReagentPurchaseDialog) ConsumePendingPurchaseRequest() (game_state.ReagentPurchaseRequest, bool) {
	if d.pending == nil {
		return emptyReagentPurchaseRequest(), false
	}
	request := *d.pending
	d.pending = nil

	return request, true
}

// GetBundleCount returns the currently selected number of bundles.
func (d *ReagentPurchaseDialog) GetBundleCount() uint16 { return d.bundleCount }

// GetApprovalView returns the supplied authority view selected by bundle count.
func (d *ReagentPurchaseDialog) GetApprovalView() game_state.ReagentPurchaseApprovalView {
	return d.quote.ApprovalViews[d.bundleCount-1]
}

// GetQuote returns a detached copy of the trusted quote displayed by the modal.
func (d *ReagentPurchaseDialog) GetQuote() game_state.ReagentOfferQuote {
	return cloneReagentOfferQuote(d.quote)
}

// GetBodyText returns the selected supplied approval presentation.
func (d *ReagentPurchaseDialog) GetBodyText() string {
	if d.bodyOutput == nil {
		return ""
	}

	return d.bodyOutput.GetOutputStr(false)
}

// GetFooterGuidance returns the exact keyboard guidance displayed by the modal.
func (d *ReagentPurchaseDialog) GetFooterGuidance() string {
	return "[Left/Right] Quantity   [M] Max\n[Enter] Confirm   [Esc] Back"
}

// GetBounds returns the centered opaque modal bounds.
func (d *ReagentPurchaseDialog) GetBounds() image.Rectangle {
	d.ensureLayout()

	return d.frame.bounds()
}

// GetFontSize returns the standard merchant font size.
func (d *ReagentPurchaseDialog) GetFontSize() StandardFontSize { return FontSizeMainScreen }

// IsPreview reports whether the modal stops before purchase authority.
func (d *ReagentPurchaseDialog) IsPreview() bool { return d.preview }

// IsCancelled reports whether Escape closed the modal without confirmation.
func (d *ReagentPurchaseDialog) IsCancelled() bool { return d.cancelled }

// IsDismissed reports whether the modal may be removed.
func (d *ReagentPurchaseDialog) IsDismissed() bool { return d.dismissed }

// IsVisible reports whether the modal still owns presentation and input.
func (d *ReagentPurchaseDialog) IsVisible() bool { return !d.dismissed }

// Dismiss closes the modal without emitting a request.
func (d *ReagentPurchaseDialog) Dismiss() { d.dismissed = true }

func (d *ReagentPurchaseDialog) setBundleCount(count uint16) {
	if count == d.bundleCount {
		return
	}
	d.bundleCount = count
	d.populateOutputs()
}

func (d *ReagentPurchaseDialog) ensureLayout() {
	if !d.frame.ensureLayout() {
		return
	}
	_, d.headerOutput = GetScaledFontWithOutput(FontConfig{
		Size: FontSizeMainScreen, LineSpacing: 0, LineSpacingMultiplier: reagentPurchaseLineSpacing,
		MaxLines: 1, MaxChars: reagentPurchaseMinimumTextChars,
	})
	bodyFont, bodyOutput := GetScaledFontWithOutput(FontConfig{
		Size: FontSizeMainScreen, LineSpacing: 0, LineSpacingMultiplier: reagentPurchaseLineSpacing,
		MaxLines: reagentPurchaseMaximumBodyLines, MaxChars: reagentPurchaseMinimumTextChars,
	})
	_, d.footerOutput = GetScaledFontWithOutput(FontConfig{
		Size: FontSizeMainScreen, LineSpacing: 0, LineSpacingMultiplier: reagentPurchaseLineSpacing,
		MaxLines: reagentPurchaseFooterLines, MaxChars: reagentPurchaseMinimumTextChars,
	})
	d.headerOutput.SetMaxCharsPerLine(max(
		reagentPurchaseMinimumTextChars,
		d.headerOutput.Font.CharsForWidth(max(1, d.frame.headerText.Dx())),
	))
	d.bodyOutput = text.NewOutput(
		bodyFont,
		bodyOutput.GetLineSpacing(),
		max(1, bodyOutput.GetLineCapacity(max(1, d.frame.bodyText.Dy()))),
		max(reagentPurchaseMinimumTextChars, bodyFont.CharsForWidth(max(1, d.frame.bodyText.Dx()))),
	)
	d.footerOutput.SetMaxCharsPerLine(max(
		reagentPurchaseMinimumTextChars,
		d.footerOutput.Font.CharsForWidth(max(1, d.frame.footerText.Dx())),
	))
	d.populateOutputs()
}

func (d *ReagentPurchaseDialog) populateOutputs() {
	if d.headerOutput == nil || d.bodyOutput == nil || d.footerOutput == nil {
		return
	}
	approval := d.GetApprovalView()
	d.headerOutput.Reset()
	header := d.headerOutput.AppendLine().WithAlignment(etext.AlignCenter)
	if d.preview {
		header.Text("PREVIEW - ", DialogStatusWarningColor)
	}
	header.Text("Purchase "+approval.ReagentName, DialogHeaderColor).Done()

	d.bodyOutput.Reset()
	d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).LightGray("Bundles (before theft)").Done()
	d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).
		White("<   ").Yellowf("%d", approval.BundleCount).White("   >").Done()
	d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).
		LightGrayf("of %d max", approval.MaxBundles).Done()
	d.bodyOutput.AppendLine().Whitef("%d %s per bundle", approval.BundleQuantity, approval.ReagentName).Done()
	d.bodyOutput.AppendLine().LightGray("Requested quantity: ").Whitef("%d", approval.RequestedQuantity).Done()
	d.bodyOutput.AppendLine().LightGray("Inventory: ").Cyanf("%d", approval.QuantityBefore).
		White(" -> ").LightGreenf("%d", approval.QuantityAfter).Done()
	d.bodyOutput.AppendLine().LightGray("Maximum receipt: ").LightGreenf("%d", approval.ActualReceipt).Done()
	d.bodyOutput.AppendLine().LightGray("Price per bundle: ").Yellowf("%d gold", approval.PricePerBundle).Done()
	d.bodyOutput.AppendLine().LightGray("Total price: ").Yellowf("%d gold", approval.TotalPrice).Done()
	d.bodyOutput.AppendLine().LightGray("Gold: ").Cyanf("%d", approval.GoldBefore).
		White(" -> ").LightGreenf("%d", approval.GoldAfterPayment).Done()
	if approval.PaidOverflow {
		d.bodyOutput.AppendLine().Textf(
			"%d %s will not fit.",
			DialogStatusWarningColor,
			approval.OverflowQuantity,
			reagentOverflowNoun(approval.OverflowQuantity),
		).WordWrap().Done()
		d.bodyOutput.AppendLine().Text(
			"Full bundle prices apply.", DialogStatusWarningColor,
		).Done()
	}

	d.footerOutput.Reset()
	d.footerOutput.AppendLine().WithAlignment(etext.AlignCenter).
		White("[Left/Right]").LightGray(" Quantity   ").White("[M]").LightGray(" Max").Done()
	d.footerOutput.AppendLine().WithAlignment(etext.AlignCenter).
		White("[Enter]").LightGray(" Confirm   ").White("[Esc]").LightGray(" Back").Done()
}

func reagentOverflowNoun(quantity uint16) string {
	if quantity == 1 {
		return "reagent"
	}

	return "reagents"
}

func reagentPurchasePlacement() sprites.PercentBasedPlacement {
	return sprites.PercentBasedPlacement{
		StartPercentX: reagentPurchaseStartPercentX,
		EndPercentX:   reagentPurchaseEndPercentX,
		StartPercentY: reagentPurchaseStartPercentY,
		EndPercentY:   reagentPurchaseEndPercentY,
	}
}

func emptyReagentPurchaseRequest() game_state.ReagentPurchaseRequest {
	return game_state.ReagentPurchaseRequest{Reagent: references.SulfurAsh, BundleCount: 0}
}

func validateReagentPurchaseQuote(quote game_state.ReagentOfferQuote) {
	validateReagentOfferQuote(quote, quote.Gold)
	if quote.BlockReason.IsBlocked() || quote.Full || quote.MaxBundles == 0 {
		panic("ReagentPurchaseDialog: an available reagent quote is required")
	}
}
