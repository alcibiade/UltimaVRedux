package widgets

import (
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

const (
	tavernActionStartPercentX    = 0.24
	tavernActionEndPercentX      = 0.76
	tavernActionStartPercentY    = 0.17
	tavernActionEndPercentY      = 0.83
	tavernActionHeaderRatio      = 0.15
	tavernActionFooterRatio      = 0.18
	tavernActionMinimumInset     = 5
	tavernActionLineSpacing      = 1.08
	tavernActionMaximumBodyLines = 18
	tavernActionFooterLines      = 2
	tavernActionMinimumTextChars = 8
	tavernActionNormalFooter     = "[Enter] Confirm   [Esc] Back"
	tavernActionWarningFooter    = "[Enter] Another Drink   [Esc] I've Had Enough"
	tavernActionProvisionFooter  = "[Left/Right] Quantity   [M] Maximum\n[Enter] Buy   [Esc] Back"
)

var _ Widget = (*TavernActionDialog)(nil)

// TavernActionDialogConfig configures one compact keyboard-only Tavern approval.
type TavernActionDialogConfig struct {
	Keyboard     *input.Keyboard
	Intent       TavernActionIntent
	IsKeyPressed func(ebiten.Key) bool
}

// TavernActionDialog presents detached reviewed consequences and emits exact requests unchanged.
type TavernActionDialog struct {
	keyboard        *input.Keyboard
	isKeyPressed    func(ebiten.Key) bool
	intent          TavernActionIntent
	visible         bool
	dismissed       bool
	cancelled       bool
	actionKeysReady bool
	pending         *TavernActionRequest
	provisionPacks  int

	frame        *merchantModalFrame
	headerOutput *text.Output
	bodyOutput   *text.Output
	footerOutput *text.Output
}

// NewTavernActionDialog creates a compact approval from one fully detached intent union.
func NewTavernActionDialog(config TavernActionDialogConfig) *TavernActionDialog {
	if config.Keyboard == nil {
		panic("TavernActionDialog: keyboard is required")
	}
	validateTavernActionIntent(config.Intent)
	isKeyPressed := config.IsKeyPressed
	if isKeyPressed == nil {
		isKeyPressed = ebiten.IsKeyPressed
	}
	footerMinimumLines := 1
	if config.Intent.Action == TavernActionProvision {
		footerMinimumLines = tavernActionFooterLines
	}
	dialog := &TavernActionDialog{ //nolint:exhaustruct // Resolution-dependent presentation fields initialize on demand.
		keyboard:        config.Keyboard,
		isKeyPressed:    isKeyPressed,
		intent:          cloneTavernActionIntent(config.Intent),
		provisionPacks:  config.Intent.PackCount,
		visible:         true,
		actionKeysReady: tavernActionControlsReleased(isKeyPressed),
		frame: newMerchantModalFrame(merchantModalFrameConfig{
			placement:             tavernActionPlacement(),
			headerRatio:           tavernActionHeaderRatio,
			footerRatio:           tavernActionFooterRatio,
			minimumInset:          tavernActionMinimumInset,
			lineSpacingMultiplier: tavernActionLineSpacing,
			footerMinimumLines:    footerMinimumLines,
		}),
	}
	config.Keyboard.PrepareForTransition(0)
	dialog.ensureLayout()

	return dialog
}

// Draw renders the opaque semantic Tavern approval modal.
//
//nolint:dupl // Merchant approval dialogs intentionally share the established modal rendering shape.
func (d *TavernActionDialog) Draw(screen *ebiten.Image) {
	if screen == nil || !d.IsVisible() {
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

// Update confirms one normal action, chooses warned Stop/Continue, or cancels.
func (d *TavernActionDialog) Update() {
	if !d.IsVisible() {
		return
	}
	if !d.prepareActionKeys() {
		return
	}
	if d.intent.Drink.Warning {
		switch {
		case d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape):
			d.accept(d.intent.StopRequest)
		case d.confirmPressed():
			d.accept(d.intent.Request)
		}

		return
	}
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		d.cancel()

		return
	}
	if d.intent.Action == TavernActionProvision && d.updateProvisionQuantity() {
		return
	}
	if d.confirmPressed() {
		d.accept(d.intent.Request)
	}
}

func (d *TavernActionDialog) confirmPressed() bool {
	return d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) ||
		d.keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter)
}

func (d *TavernActionDialog) prepareActionKeys() bool {
	if d.actionKeysReady {
		return true
	}
	if !tavernActionControlsReleased(d.isKeyPressed) {
		return false
	}
	d.actionKeysReady = true

	return true
}

func (d *TavernActionDialog) updateProvisionQuantity() bool {
	switch {
	case d.keyboard.IsKeyPressedAndRegister(ebiten.KeyLeft):
		d.setProvisionPackCount(d.provisionPacks - 1)
	case d.keyboard.IsKeyPressedAndRegister(ebiten.KeyRight):
		d.setProvisionPackCount(d.provisionPacks + 1)
	case d.keyboard.IsKeyPressedAndRegister(ebiten.KeyM):
		d.setProvisionPackCount(d.intent.Provision.MaximumUsefulPacks)
	default:
		return false
	}

	return true
}

func (d *TavernActionDialog) setProvisionPackCount(packCount int) {
	packCount = max(tavernMinimumProvisionPacks, min(d.intent.Provision.MaximumUsefulPacks, packCount))
	if packCount == d.provisionPacks {
		return
	}
	d.provisionPacks = packCount
	d.intent.PackCount = packCount
	d.intent.Request.Provision.PackCount = packCount
	d.populateOutputs()
}

// ConsumePendingRequest returns and clears the exact approved comparison request.
func (d *TavernActionDialog) ConsumePendingRequest() (TavernActionRequest, bool) {
	if d.pending == nil {
		return emptyTavernActionRequest(), false
	}
	request := *d.pending
	d.pending = nil

	return request, true
}

// HasPendingRequest reports whether an approval emitted an unconsumed request.
func (d *TavernActionDialog) HasPendingRequest() bool { return d.pending != nil }

// GetIntent returns a detached copy of the immutable approval union.
func (d *TavernActionDialog) GetIntent() TavernActionIntent { return cloneTavernActionIntent(d.intent) }

// GetProvisionPackCount returns the selected provision quantity.
func (d *TavernActionDialog) GetProvisionPackCount() int { return d.provisionPacks }

// GetTitleText returns the visible action title, including PREVIEW when applicable.
func (d *TavernActionDialog) GetTitleText() string {
	if d.headerOutput == nil {
		return ""
	}

	return d.headerOutput.GetOutputStr(false)
}

// GetBodyText returns exact mode-specific approval presentation.
func (d *TavernActionDialog) GetBodyText() string {
	if d.bodyOutput == nil {
		return ""
	}

	return d.bodyOutput.GetOutputStr(false)
}

// GetFooterGuidance returns normal confirmation or explicit warning controls.
func (d *TavernActionDialog) GetFooterGuidance() string {
	if d.intent.Drink.Warning {
		return tavernActionWarningFooter
	}
	if d.intent.Action == TavernActionProvision {
		return tavernActionProvisionFooter
	}

	return tavernActionNormalFooter
}

// GetBounds returns the centered responsive modal bounds.
func (d *TavernActionDialog) GetBounds() image.Rectangle {
	d.ensureLayout()

	return d.frame.bounds()
}

// GetBodyBounds returns the semantically backed modal body.
func (d *TavernActionDialog) GetBodyBounds() image.Rectangle {
	d.ensureLayout()

	return d.frame.bodyBounds
}

// GetFontSize returns the standard main-screen merchant font size.
func (d *TavernActionDialog) GetFontSize() StandardFontSize { return FontSizeMainScreen }

// IsPreview reports whether workflow will stop before Tavern authority.
func (d *TavernActionDialog) IsPreview() bool { return d.intent.Preview }

// ShowWithoutAnimation makes a non-complete approval immediately visible.
func (d *TavernActionDialog) ShowWithoutAnimation() {
	if !d.dismissed {
		d.visible = true
	}
}

// IsComplete reports whether approval confirmed, cancelled, or was dismissed.
func (d *TavernActionDialog) IsComplete() bool { return d.dismissed }

// IsCancelled reports whether Escape returned to the parent without a request.
func (d *TavernActionDialog) IsCancelled() bool { return d.cancelled }

// IsDismissed reports whether workflow may remove the approval.
func (d *TavernActionDialog) IsDismissed() bool { return d.dismissed }

// IsVisible reports whether approval currently owns presentation and input.
func (d *TavernActionDialog) IsVisible() bool { return d.visible && !d.dismissed }

// Dismiss closes approval without emitting a request.
func (d *TavernActionDialog) Dismiss() {
	if d.dismissed {
		return
	}
	d.keyboard.PrepareForTransition(0)
	d.visible = false
	d.dismissed = true
}

func (d *TavernActionDialog) accept(request TavernActionRequest) {
	accepted := request
	d.pending = &accepted
	d.keyboard.PrepareForTransition(0)
	d.visible = false
	d.dismissed = true
}

func (d *TavernActionDialog) cancel() {
	d.cancelled = true
	d.Dismiss()
}

func (d *TavernActionDialog) ensureLayout() {
	if !d.frame.ensureLayout() {
		return
	}
	_, d.headerOutput = GetScaledFontWithOutput(FontConfig{
		Size: FontSizeMainScreen, LineSpacing: 0, LineSpacingMultiplier: tavernActionLineSpacing,
		MaxLines: 1, MaxChars: tavernActionMinimumTextChars,
	})
	bodyFont, bodyOutput := GetScaledFontWithOutput(FontConfig{
		Size: FontSizeMainScreen, LineSpacing: 0, LineSpacingMultiplier: tavernActionLineSpacing,
		MaxLines: tavernActionMaximumBodyLines, MaxChars: tavernActionMinimumTextChars,
	})
	_, d.footerOutput = GetScaledFontWithOutput(FontConfig{
		Size: FontSizeMainScreen, LineSpacing: 0, LineSpacingMultiplier: tavernActionLineSpacing,
		MaxLines: tavernActionFooterLines, MaxChars: tavernActionMinimumTextChars,
	})
	d.headerOutput.SetMaxCharsPerLine(max(
		tavernActionMinimumTextChars,
		d.headerOutput.Font.CharsForWidth(max(1, d.frame.headerText.Dx())),
	))
	d.bodyOutput = text.NewOutput(
		bodyFont,
		bodyOutput.GetLineSpacing(),
		max(1, bodyOutput.GetLineCapacity(max(1, d.frame.bodyText.Dy()))),
		max(tavernActionMinimumTextChars, bodyFont.CharsForWidth(max(1, d.frame.bodyText.Dx()))),
	)
	d.footerOutput.SetMaxCharsPerLine(max(
		tavernActionMinimumTextChars,
		d.footerOutput.Font.CharsForWidth(max(1, d.frame.footerText.Dx())),
	))
	d.populateOutputs()
}

func (d *TavernActionDialog) populateOutputs() {
	if d.headerOutput == nil || d.bodyOutput == nil || d.footerOutput == nil {
		return
	}
	d.populateHeaderOutput()
	d.populateBodyOutput()
	d.populateFooterOutput()
}

func (d *TavernActionDialog) populateHeaderOutput() {
	d.headerOutput.Reset()
	title := d.headerOutput.AppendLine().WithAlignment(etext.AlignCenter)
	if d.intent.Preview {
		title.Text("PREVIEW - ", DialogStatusWarningColor)
	}
	switch {
	case d.intent.Drink.Warning:
		title.Text("One More Drink?", DialogHeaderColor).Done()
	case d.intent.Action == TavernActionProvision:
		title.Text("Purchase "+d.intent.Provision.Provision.Name, DialogHeaderColor).Done()
	default:
		title.Text("Approve "+d.intent.Action.String(), DialogHeaderColor).Done()
	}
}

func (d *TavernActionDialog) populateBodyOutput() {
	d.bodyOutput.Reset()
	if d.intent.Preview {
		d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).
			Text("PREVIEW: authority has not run.", DialogStatusWarningColor).Done()
	}
	switch d.intent.Action {
	case TavernActionMeal:
		d.populateMealBody()
	case TavernActionDrink:
		d.populateDrinkBody()
	case TavernActionProvision:
		d.populateProvisionBody()
	case TavernActionInformation:
		d.populateInformationBody()
	case TavernActionStop, TavernActionUnknown:
		panic("TavernActionDialog.populateOutputs: invalid standalone action")
	default:
		panic("TavernActionDialog.populateOutputs: unknown action")
	}
}

func (d *TavernActionDialog) populateFooterOutput() {
	d.footerOutput.Reset()
	for _, segments := range parseMerchantFooterGuidance(d.GetFooterGuidance()) {
		line := d.footerOutput.AppendLine().WithAlignment(etext.AlignCenter)
		for _, segment := range segments {
			line.Text(segment.Text, segment.Color)
		}
		line.Done()
	}
}

func (d *TavernActionDialog) populateMealBody() {
	quote := d.intent.Meal
	d.bodyOutput.AppendLine().Yellow("Meal: ").White(quote.Meal.Name).Done()
	d.bodyOutput.AppendLine().Whitef("%d diners", quote.DinerCount).
		LightGray(" x ").Yellowf("%d gold each", quote.Meal.BasePrice).Done()
	d.bodyOutput.AppendLine().LightGray("Total: ").Yellowf("%d gold", quote.Charge).Done()
	d.appendGoldProjection(quote.GoldBefore, quote.GoldAfterPayment, quote.Affordable, quote.Charge)
}

func (d *TavernActionDialog) populateDrinkBody() {
	quote := d.intent.Drink
	if quote.Warning {
		d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).
			Text(d.intent.DrinkWarning, DialogValueColor).WordWrap().Done()
		d.bodyOutput.AppendLine().White("").Done()
	}
	name := quote.Drink.Name
	if quote.WineIndex != game_state.TavernOrdinaryDrinkWineIndex {
		name = quote.Wine.Name
	}
	d.bodyOutput.AppendLine().Yellow("Drink: ").White(name).Done()
	if quote.WineIndex == game_state.TavernOrdinaryDrinkWineIndex {
		d.bodyOutput.AppendLine().Whitef("%d drinks", quote.DinerCount).
			LightGray(" x ").Yellowf("%d gold each", quote.Drink.UnitPrice).Done()
	} else {
		d.bodyOutput.AppendLine().LightGray("Bottle price: ").Yellowf("%d gold", quote.Charge).Done()
	}
	d.bodyOutput.AppendLine().LightGray("Total: ").Yellowf("%d gold", quote.Charge).Done()
	d.appendGoldProjection(quote.GoldBefore, quote.GoldAfterPayment, quote.Affordable, quote.Charge)
}

func (d *TavernActionDialog) populateProvisionBody() {
	quote := d.intent.Provision
	approval := quote.Approvals[d.provisionPacks-1]
	d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).
		Text(d.intent.ProvisionPitch, DialogValueColor).WordWrap().Done()
	d.bodyOutput.AppendLine().White("").Done()
	d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).LightGray("PACKS").Done()
	d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).
		White("<   ").Yellowf("%d", d.provisionPacks).White("   >").Done()
	d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).
		LightGrayf("of %d useful", quote.MaximumUsefulPacks).Done()
	d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).
		Whitef("%d food", quote.Provision.PackQuantity).LightGray(" per pack").Done()
	d.bodyOutput.AppendLine().White("").Done()
	d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).
		LightGray("FOOD  ").Cyanf("%d", quote.FoodBefore).White("  ->  ").
		LightGreenf("%d", approval.FoodAfter).Done()
	d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).
		LightGray("PRICE EACH / TOTAL  ").Yellowf("%d / %d GOLD", quote.PricePerPack, approval.TotalCost).Done()
	d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).
		LightGray("GOLD  ").Cyanf("%d", quote.GoldBefore).White("  ->  ").
		LightGreenf("%d", approval.GoldAfterPurchases).Done()
	d.populateProvisionProjectionNote(approval)
}

func (d *TavernActionDialog) populateProvisionProjectionNote(approval game_state.TavernProvisionApproval) {
	switch approval.Outcome {
	case game_state.TavernProvisionCompleted:
		return
	case game_state.TavernProvisionPartial:
		d.bodyOutput.AppendLine().Textf(
			"Only %d of %d packs can currently be paid for.",
			DialogLabelColor,
			approval.PurchasedPacks,
			approval.PackCount,
		).WordWrap().Done()
	case game_state.TavernProvisionSaturated:
		d.bodyOutput.AppendLine().Text(
			"The order stops when carried food reaches 9999.", DialogLabelColor,
		).WordWrap().Done()
	case game_state.TavernProvisionCharityEjection:
		d.bodyOutput.AppendLine().Text(
			"No pack can be afforded. The keeper may offer 1-2 food before ending service.",
			DialogStatusWarningColor,
		).WordWrap().Done()
	case game_state.TavernProvisionEjection:
		d.bodyOutput.AppendLine().Text(
			"No pack can be afforded; the keeper will end service.", DialogStatusWarningColor,
		).WordWrap().Done()
	case game_state.TavernProvisionOutcomeUnknown:
		panic("TavernActionDialog: provision projection has no outcome")
	default:
		panic("TavernActionDialog: provision projection has unknown outcome")
	}
}

func (d *TavernActionDialog) populateInformationBody() {
	quote := d.intent.Information
	d.bodyOutput.AppendLine().Text(d.intent.InformationPitch, DialogValueColor).WordWrap().Done()
	d.bodyOutput.AppendLine().LightGray("Price: ").Yellowf("%d gold", quote.Price).Done()
	d.appendGoldProjection(
		quote.GoldBefore,
		quote.GoldAfterPayment,
		quote.Affordable,
		uint16(quote.Price), //nolint:gosec // Canonical topic price is validated and bounded.
	)
}

func (d *TavernActionDialog) appendGoldProjection(before, after uint16, affordable bool, charge uint16) {
	if affordable {
		d.bodyOutput.AppendLine().LightGray("Gold after payment: ").Cyanf("%d -> %d", before, after).Done()

		return
	}
	d.bodyOutput.AppendLine().Textf(
		"The party has %d gold and needs %d more.",
		DialogStatusWarningColor,
		before,
		int(charge)-int(before),
	).WordWrap().Done()
}

func tavernActionPlacement() sprites.PercentBasedPlacement {
	return sprites.PercentBasedPlacement{
		StartPercentX: tavernActionStartPercentX,
		EndPercentX:   tavernActionEndPercentX,
		StartPercentY: tavernActionStartPercentY,
		EndPercentY:   tavernActionEndPercentY,
	}
}

func validateTavernActionIntent(intent TavernActionIntent) {
	if intent.Action == TavernActionUnknown || intent.Request.Action != intent.Action {
		panic("TavernActionDialog: complete detached Tavern action is required")
	}
	switch intent.Action {
	case TavernActionMeal:
		validateTavernMealIntent(intent)
	case TavernActionDrink:
		validateTavernDrinkIntent(intent)
	case TavernActionProvision:
		validateTavernProvisionIntent(intent)
	case TavernActionInformation:
		validateTavernInformationIntent(intent)
	case TavernActionStop:
		panic("TavernActionDialog: Stop is available only within a warned drink")
	case TavernActionUnknown:
		panic("TavernActionDialog: an empty action cannot be approved")
	default:
		panic("TavernActionDialog: known action is required")
	}
}

func validateTavernMealIntent(intent TavernActionIntent) {
	quote := intent.Meal
	request := intent.Request.Meal
	if request.ReviewedMerchantID == "" || request.ReviewedMeal != quote.Meal ||
		request.ReviewedDinerCount != quote.DinerCount || request.ReviewedCharge != quote.Charge ||
		request.ReviewedGoldBefore != quote.GoldBefore || request.ReviewedFoodBefore != quote.FoodBefore ||
		quote.Charge == 0 {
		panic("TavernActionDialog: exact detached meal review is required")
	}
	validateTavernGoldProjection(quote.Charge, quote.GoldBefore, quote.GoldAfterPayment, quote.Affordable)
}

//nolint:cyclop // Exact drink and warning-alternative comparison fields form one validation boundary.
func validateTavernDrinkIntent(intent TavernActionIntent) {
	quote := intent.Drink
	request := intent.Request.Drink
	wantApproval := game_state.TavernDrinkApprovalPurchase
	if quote.Warning {
		wantApproval = game_state.TavernDrinkApprovalContinue
	}
	if request.ReviewedMerchantID == "" || request.ReviewedDrink != quote.Drink ||
		request.ReviewedWineIndex != quote.WineIndex || request.ReviewedWine != quote.Wine ||
		request.ReviewedDinerCount != quote.DinerCount || request.ReviewedCharge != quote.Charge ||
		request.ReviewedGoldBefore != quote.GoldBefore || request.ReviewedDrunk != quote.Drunk ||
		request.ReviewedSmashed != quote.Smashed || request.ReviewedKarma != quote.Karma ||
		request.Approval != wantApproval || quote.Warning != quote.StopAvailable || quote.Charge == 0 {
		panic("TavernActionDialog: exact detached drink review is required")
	}
	validateTavernGoldProjection(quote.Charge, quote.GoldBefore, quote.GoldAfterPayment, quote.Affordable)
	if quote.Warning {
		if strings.TrimSpace(intent.DrinkWarning) == "" {
			panic("TavernActionDialog: warned drink requires barkeeper prose")
		}
		if intent.StopRequest.Action != TavernActionStop ||
			intent.StopRequest.Stop != (game_state.TavernStopRequest{
				ReviewedMerchantID: request.ReviewedMerchantID,
				ReviewedDrunk:      quote.Drunk,
			}) {
			panic("TavernActionDialog: warned drink requires exact Stop request")
		}
	} else if intent.StopRequest != emptyTavernActionRequest() {
		panic("TavernActionDialog: ordinary drink cannot expose Stop")
	}
}

func validateTavernProvisionIntent(intent TavernActionIntent) {
	quote := intent.Provision
	request := intent.Request.Provision
	if !tavernProvisionReviewMatches(request, quote, intent.ProvisionPitch) ||
		!tavernProvisionQuantityMatches(request.PackCount, intent.PackCount, quote) {
		panic("TavernActionDialog: exact detached provision review is required")
	}
}

func tavernProvisionReviewMatches(
	request game_state.TavernProvisionRequest,
	quote game_state.TavernProvisionQuote,
	pitch string,
) bool {
	return request.ReviewedMerchantID != "" && quote.Provision.Available &&
		strings.TrimSpace(pitch) != "" &&
		request.ReviewedProvision == quote.Provision && request.ReviewedPrice == quote.PricePerPack &&
		request.ReviewedGoldBefore == quote.GoldBefore && request.ReviewedFoodBefore == quote.FoodBefore
}

func tavernProvisionQuantityMatches(
	requestPackCount int,
	packCount int,
	quote game_state.TavernProvisionQuote,
) bool {
	if requestPackCount != packCount || packCount < tavernMinimumProvisionPacks ||
		packCount > quote.MaximumUsefulPacks || quote.MaximumUsefulPacks > tavernMaximumProvisionPacks ||
		!quote.HasValidApprovals() {
		return false
	}

	return true
}

func validateTavernInformationIntent(intent TavernActionIntent) {
	validateTavernInformationPitch(intent.InformationPitch)
	quote := intent.Information
	request := intent.Request.Information
	if request.ReviewedMerchantID == "" || strings.TrimSpace(quote.Topic.Key) == "" ||
		request.ReviewedTopic != quote.Topic || request.ReviewedPrice != quote.Price ||
		request.ReviewedGoldBefore != quote.GoldBefore || quote.Price <= 0 {
		panic("TavernActionDialog: exact authority-recognized information review is required")
	}
	wantAffordable := int(quote.GoldBefore) >= quote.Price
	wantAfter := quote.GoldBefore
	if wantAffordable {
		wantAfter -= uint16(quote.Price) //nolint:gosec // Validated canonical topic price is bounded.
	}
	if quote.Affordable != wantAffordable || quote.GoldAfterPayment != wantAfter {
		panic("TavernActionDialog: information gold projection is inconsistent")
	}
}

func validateTavernInformationPitch(pitch string) {
	if strings.TrimSpace(pitch) == "" {
		panic("TavernActionDialog: source-backed information pitch is required")
	}
}

func tavernActionControlsReleased(isKeyPressed func(ebiten.Key) bool) bool {
	for _, key := range []ebiten.Key{
		ebiten.KeyEscape, ebiten.KeyEnter, ebiten.KeyNumpadEnter,
		ebiten.KeyLeft, ebiten.KeyRight, ebiten.KeyM,
	} {
		if isKeyPressed(key) {
			return false
		}
	}

	return true
}
