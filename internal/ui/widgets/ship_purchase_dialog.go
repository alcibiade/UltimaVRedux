package widgets

import (
	"image"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

const (
	shipModalHeaderRatio      = 0.13
	shipModalFooterRatio      = 0.16
	shipModalMinimumInset     = 5
	shipModalLineSpacing      = 1.08
	shipModalMaximumBodyLines = 12
	shipModalFooterLines      = 2
	shipModalMinimumTextChars = 8
	shipPurchaseStartPercentX = 0.25
	shipPurchaseEndPercentX   = 0.75
	shipPurchaseStartPercentY = 0.12
	shipPurchaseEndPercentY   = 0.88
)

var _ Widget = (*ShipPurchaseDialog)(nil)

// ShipPurchaseDialogConfig configures one nested vessel approval modal.
type ShipPurchaseDialogConfig struct {
	Keyboard *input.Keyboard
	Intent   ShipPurchaseIntent
}

// ShipPurchaseDialog displays only detached quote consequences and emits the reviewed request.
type ShipPurchaseDialog struct {
	quantityInput
	keyboard  *input.Keyboard
	intent    ShipPurchaseIntent
	quantity  int
	dismissed bool
	cancelled bool
	pending   *game_state.ShipPurchaseRequest
	frame     *merchantModalFrame

	headerOutput *text.Output
	bodyOutput   *text.Output
	footerOutput *text.Output
}

// NewShipPurchaseDialog creates a centered keyboard-only purchase approval.
func NewShipPurchaseDialog(config ShipPurchaseDialogConfig) *ShipPurchaseDialog {
	if config.Keyboard == nil {
		panic("ShipPurchaseDialog: keyboard is required")
	}
	validateShipPurchaseIntent(config.Intent)
	dialog := &ShipPurchaseDialog{
		quantityInput: newQuantityInput(config.Keyboard, config.Keyboard.IsKeyHeld,
			ebiten.KeyM, ebiten.KeyUp, ebiten.KeyDown),
		keyboard:     config.Keyboard,
		intent:       config.Intent,
		quantity:     1,
		dismissed:    false,
		cancelled:    false,
		pending:      nil,
		frame:        newShipMerchantModalFrame(shipPurchasePlacement()),
		headerOutput: nil,
		bodyOutput:   nil,
		footerOutput: nil,
	}
	config.Keyboard.PrepareForTransition(0)
	dialog.ensureLayout()

	return dialog
}

// Draw renders the opaque semantic merchant modal.
func (d *ShipPurchaseDialog) Draw(screen *ebiten.Image) {
	if screen == nil || d.dismissed {
		return
	}
	d.ensureLayout()
	d.frame.draw(screen)
	d.headerOutput.DrawPanelContent(screen,
		image.Pt(d.frame.headerText.Min.X, verticallyCenteredOutputY(d.frame.headerText, d.headerOutput)),
		d.frame.headerText.Dx(), etext.AlignCenter)
	d.bodyOutput.DrawPanelContent(screen, d.frame.bodyText.Min, d.frame.bodyText.Dx(), etext.AlignStart)
	d.footerOutput.DrawPanelContent(screen,
		image.Pt(d.frame.footerText.Min.X, verticallyCenteredOutputY(d.frame.footerText, d.footerOutput)),
		d.frame.footerText.Dx(), etext.AlignCenter)
}

// Update confirms or cancels the reviewed request after release debounce.
func (d *ShipPurchaseDialog) Update() {
	if d.dismissed {
		return
	}
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		d.cancelled = true
		d.Dismiss()

		return
	}
	if d.handleQuantityInput() {
		return
	}
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) ||
		d.keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter) {
		request := d.intent.Request
		approval := d.GetApprovalView()
		request.Quantity = approval.Quantity
		request.ReviewedTotalPrice = approval.TotalPrice
		d.pending = &request
		d.Dismiss()
	}
}

func (d *ShipPurchaseDialog) handleQuantityInput() bool {
	key, owned := d.pollQuantityInput()
	if key == nil {
		return owned
	}
	if d.intent.Quote.Action != game_state.ShipPurchaseLoadSkiff || len(d.intent.Quote.ApprovalViews) <= 1 {
		return owned
	}
	switch *key { //nolint:exhaustive // The quantity helper returns only this dialog's configured keys.
	case ebiten.KeyM:
		d.setQuantity(len(d.intent.Quote.ApprovalViews))
	case ebiten.KeyLeft, ebiten.KeyUp:
		d.setQuantity(max(1, d.quantity-1))
	case ebiten.KeyRight, ebiten.KeyDown:
		d.setQuantity(min(len(d.intent.Quote.ApprovalViews), d.quantity+1))
	}

	return owned
}

func (d *ShipPurchaseDialog) setQuantity(quantity int) {
	if quantity == d.quantity {
		return
	}
	d.quantity = quantity
	d.populateOutputs()
}

// ConsumePendingPurchaseRequest returns and clears the one comparison-only request.
func (d *ShipPurchaseDialog) ConsumePendingPurchaseRequest() (game_state.ShipPurchaseRequest, bool) {
	if d.pending == nil {
		return emptyShipPurchaseRequest(), false
	}
	request := *d.pending
	d.pending = nil

	return request, true
}

// GetApprovalView returns the authority-supplied projection for the selected quantity.
func (d *ShipPurchaseDialog) GetApprovalView() game_state.ShipPurchaseApprovalView {
	return d.intent.Quote.ApprovalViews[d.quantity-1]
}

// GetFooterGuidance returns the keyboard-only approval controls.
func (d *ShipPurchaseDialog) GetFooterGuidance() string {
	if d.intent.Quote.Action == game_state.ShipPurchaseLoadSkiff && len(d.intent.Quote.ApprovalViews) > 1 {
		return merchantQuantityFooter
	}

	return "[Enter] Confirm   [Esc] Back"
}

// GetBounds returns the centered modal bounds.
func (d *ShipPurchaseDialog) GetBounds() image.Rectangle {
	d.ensureLayout()

	return d.frame.bounds()
}

// IsCancelled reports whether Escape closed the approval.
func (d *ShipPurchaseDialog) IsCancelled() bool { return d.cancelled }

// IsDismissed reports whether the modal can be removed.
func (d *ShipPurchaseDialog) IsDismissed() bool { return d.dismissed }

// Dismiss closes the modal without emitting a request.
func (d *ShipPurchaseDialog) Dismiss() {
	d.onDialogFocusChanged(false)
	d.dismissed = true
}

func (d *ShipPurchaseDialog) ensureLayout() {
	if !d.frame.ensureLayout() {
		return
	}
	d.headerOutput, d.bodyOutput, d.footerOutput = newShipMerchantModalOutputs(d.frame)
	d.populateOutputs()
}

func (d *ShipPurchaseDialog) populateOutputs() {
	quote := d.intent.Quote
	approval := d.GetApprovalView()
	title := "Purchase " + shipVehicleName(quote.VehicleType)
	if quote.Action == game_state.ShipPurchaseLoadSkiff {
		title = "Purchase Skiffs"
	}
	d.headerOutput.Reset()
	titleLine := d.headerOutput.AppendLine().WithAlignment(etext.AlignCenter)
	if d.intent.Preview {
		titleLine.Text("PREVIEW - ", DialogStatusWarningColor)
	}
	titleLine.Text(title, DialogHeaderColor).Done()
	body := d.bodyOutput
	body.Reset()
	body.AppendLine().LightGray("Vessel: ").White(shipVehicleName(quote.VehicleType)).Done()
	body.AppendLine().LightGray("Action: ").Yellow(shipActionLabel(quote.Action)).Done()
	body.AppendLine().LightGray("Dock: ").White(d.intent.HarbourName).Done()
	switch quote.Action {
	case game_state.ShipPurchaseCreateFrigate:
		body.AppendLine().LightGray("Included skiffs: ").Cyanf("%d", shipShopFrigateSkiffs).Done()
	case game_state.ShipPurchaseLoadSkiff:
		body.AppendLine().LightGray("Skiffs aboard: ").Cyanf(
			"%d/%d", quote.Berth.SkiffCount, game_state.ShipShopMaximumSkiffs,
		).Done()
		body.AppendLine().LightGray("Quantity: ").White("<   ").Yellowf("%d", approval.Quantity).
			White("   >").Done()
		body.AppendLine().LightGray("If completed: ").LightGreenf(
			"%d/%d", approval.SkiffCountAfter, game_state.ShipShopMaximumSkiffs,
		).Done()
	case game_state.ShipPurchaseCreateStandaloneSkiff:
		body.AppendLine().LightGray("Included skiffs: ").Cyan("1 standalone").Done()
	default:
		panic("ShipPurchaseDialog.populateOutputs: unknown action")
	}
	if quote.Action == game_state.ShipPurchaseLoadSkiff {
		body.AppendLine().LightGray("Unit price: ").Yellowf("%d gold", approval.UnitPrice).Done()
		body.AppendLine().LightGray("Total: ").Yellowf("%d gold", approval.TotalPrice).Done()
	} else {
		body.AppendLine().LightGray("Price: ").Yellowf("%d gold", approval.TotalPrice).Done()
	}
	if approval.Affordable {
		body.AppendLine().LightGray("Party gold: ").Cyanf("%d", approval.GoldBefore).
			White(" -> ").LightGreenf("%d", approval.GoldAfterPayment).Done()
	} else {
		body.AppendLine().LightGray("Party gold: ").Cyanf("%d", approval.GoldBefore).Done()
		body.AppendLine().Textf("Need %d gold.", DialogStatusWarningColor, approval.TotalPrice).Done()
	}
	d.footerOutput.Reset()
	appendControlGuidance(d.footerOutput, d.GetFooterGuidance(), d.frame.footerText.Dx())
}

func validateShipPurchaseIntent(intent ShipPurchaseIntent) {
	quote := intent.Quote
	request := intent.Request
	if strings.TrimSpace(intent.HarbourName) == "" || quote.BlockReason.IsBlocked() {
		panic("ShipPurchaseDialog: available detached quote is required")
	}
	validateShipPurchaseQuoteIdentity(quote)
	validateShipPurchaseRequestMatch(quote, request)
	validateShipBerth(quote.Berth)
	validateShipTravelReview(intent.Travel, false)
	validateShipPurchaseIntentLinks(intent)
	validateShipPurchaseGold(quote)
}

func validateShipPurchaseQuoteIdentity(quote game_state.ShipPurchaseQuote) {
	if quote.VehicleType != references.FrigateVehicle && quote.VehicleType != references.SkiffVehicle ||
		quote.BasePrice <= 0 || quote.Price <= 0 || quote.Price > math.MaxUint16 || quote.RequestFingerprint == "" {
		panic("ShipPurchaseDialog: valid detached quote is required")
	}
	validateShipQuoteAction(quote, quote.VehicleType)
}

func validateShipPurchaseIntentLinks(intent ShipPurchaseIntent) {
	if intent.HarbourName != intent.Travel.HarbourName ||
		intent.Request.ReviewedMerchantID != intent.Travel.MerchantID ||
		intent.Quote.Berth.Location != intent.Travel.DockLocation ||
		intent.Quote.Berth.Position != intent.Travel.BerthPosition ||
		intent.Quote.Berth.VehicleType != intent.Travel.VehicleType ||
		intent.Quote.Berth.Pending != intent.Travel.PendingDelivery {
		panic("ShipPurchaseDialog: detached quote and harbour review do not match")
	}
}

func validateShipPurchaseRequestMatch(
	quote game_state.ShipPurchaseQuote,
	request game_state.ShipPurchaseRequest,
) {
	if request.ReviewedMerchantID == "" || request.ReviewedPricingCharacterIndex < 0 ||
		request.VehicleType != quote.VehicleType || request.Action != quote.Action ||
		request.ReviewedPrice != quote.Price || request.ReviewedGoldBefore != quote.GoldBefore ||
		request.ReviewedFingerprint != quote.RequestFingerprint ||
		!shipRequestMatchesApproval(request, quote) {
		panic("ShipPurchaseDialog: detached request does not match quote")
	}
}

func validateShipPurchaseGold(quote game_state.ShipPurchaseQuote) {
	if len(quote.ApprovalViews) == 0 {
		panic("ShipPurchaseDialog: available quote requires approval views")
	}
	for index, approval := range quote.ApprovalViews {
		validateShipApprovalIdentity(quote, approval, index+1)
		validateShipApprovalGold(approval)
	}
	first := quote.ApprovalViews[0]
	if quote.Affordable != first.Affordable || quote.GoldAfterPayment != first.GoldAfterPayment {
		panic("ShipPurchaseDialog: quote and first approval differ")
	}
}

func validateShipApprovalIdentity(
	quote game_state.ShipPurchaseQuote,
	approval game_state.ShipPurchaseApprovalView,
	quantity int,
) {
	if approval.Quantity != quantity || approval.UnitPrice != quote.Price ||
		approval.TotalPrice != quote.Price*quantity || approval.GoldBefore != quote.GoldBefore {
		panic("ShipPurchaseDialog: approval projection is inconsistent")
	}
}

func validateShipApprovalGold(approval game_state.ShipPurchaseApprovalView) {
	expectedAfter := approval.GoldBefore
	if approval.Affordable {
		if approval.TotalPrice > int(approval.GoldBefore) {
			panic("ShipPurchaseDialog: affordable approval exceeds gold")
		}
		expectedAfter -= uint16(approval.TotalPrice) //nolint:gosec // Trusted affordability bounds the total.
	} else if approval.TotalPrice <= int(approval.GoldBefore) {
		panic("ShipPurchaseDialog: unaffordable approval is within gold")
	}
	if approval.GoldAfterPayment != expectedAfter {
		panic("ShipPurchaseDialog: approval gold consequence is inconsistent")
	}
}

func shipPurchasePlacement() sprites.PercentBasedPlacement {
	return sprites.PercentBasedPlacement{
		StartPercentX: shipPurchaseStartPercentX,
		EndPercentX:   shipPurchaseEndPercentX,
		StartPercentY: shipPurchaseStartPercentY,
		EndPercentY:   shipPurchaseEndPercentY,
	}
}

func newShipMerchantModalFrame(placement sprites.PercentBasedPlacement) *merchantModalFrame {
	return newMerchantModalFrame(merchantModalFrameConfig{
		placement:             placement,
		headerRatio:           shipModalHeaderRatio,
		footerRatio:           shipModalFooterRatio,
		minimumInset:          shipModalMinimumInset,
		lineSpacingMultiplier: shipModalLineSpacing,
		footerMinimumLines:    shipModalFooterLines,
	})
}

func newShipMerchantModalOutputs(frame *merchantModalFrame) (*text.Output, *text.Output, *text.Output) {
	_, headerOutput := GetScaledFontWithOutput(FontConfig{
		Size: FontSizeTitle, LineSpacing: 0, LineSpacingMultiplier: shipModalLineSpacing,
		MaxLines: 1, MaxChars: shipModalMinimumTextChars,
	})
	bodyFont, bodyOutput := GetScaledFontWithOutput(FontConfig{
		Size: FontSizeBody, LineSpacing: 0, LineSpacingMultiplier: shipModalLineSpacing,
		MaxLines: shipModalMaximumBodyLines, MaxChars: shipModalMinimumTextChars,
	})
	_, footerOutput := GetScaledFontWithOutput(FontConfig{
		Size: FontSizeBody, LineSpacing: 0, LineSpacingMultiplier: shipModalLineSpacing,
		MaxLines: shipModalFooterLines, MaxChars: shipModalMinimumTextChars,
	})
	headerOutput.SetMaxCharsPerLine(max(shipModalMinimumTextChars,
		headerOutput.Font.CharsForWidth(max(1, frame.headerText.Dx()))))
	bodyOutput = text.NewOutput(bodyFont, bodyOutput.GetLineSpacing(), shipModalMaximumBodyLines,
		max(shipModalMinimumTextChars, bodyFont.CharsForWidth(max(1, frame.bodyText.Dx()))))
	footerOutput.SetMaxCharsPerLine(max(shipModalMinimumTextChars,
		footerOutput.Font.CharsForWidth(max(1, frame.footerText.Dx()))))

	return headerOutput, bodyOutput, footerOutput
}
