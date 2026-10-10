package widgets

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

const (
	horsePurchaseStartPercentX    = 0.30
	horsePurchaseEndPercentX      = 0.70
	horsePurchaseStartPercentY    = 0.24
	horsePurchaseEndPercentY      = 0.76
	horsePurchaseHeaderRatio      = 0.18
	horsePurchaseFooterRatio      = 0.22
	horsePurchaseMinimumInset     = 6
	horsePurchaseLineSpacing      = 1.12
	horsePurchaseMaximumBodyLines = 6
	horsePurchaseMinimumTextChars = 8
	horsePurchaseFooterLines      = 2
)

var _ Widget = (*HorsePurchaseDialog)(nil)

// HorsePurchaseDialogConfig configures one nested horse approval modal.
type HorsePurchaseDialogConfig struct {
	Keyboard *input.Keyboard
	Intent   HorsePurchaseIntent
}

// HorsePurchaseDialog presents the horse price and party-gold consequence before authority.
type HorsePurchaseDialog struct {
	keyboard  *input.Keyboard
	intent    HorsePurchaseIntent
	dismissed bool
	cancelled bool
	pending   *game_state.HorsePurchaseRequest

	frame        *merchantModalFrame
	headerOutput *text.Output
	bodyOutput   *text.Output
	footerOutput *text.Output
}

// NewHorsePurchaseDialog creates a centered keyboard-only approval modal.
func NewHorsePurchaseDialog(config HorsePurchaseDialogConfig) *HorsePurchaseDialog {
	if config.Keyboard == nil {
		panic("HorsePurchaseDialog: keyboard is required")
	}
	validateHorsePurchaseIntent(config.Intent)
	dialog := &HorsePurchaseDialog{
		keyboard:  config.Keyboard,
		intent:    config.Intent,
		dismissed: false,
		cancelled: false,
		pending:   nil,
		frame: newMerchantModalFrame(merchantModalFrameConfig{
			placement:             horsePurchasePlacement(),
			headerRatio:           horsePurchaseHeaderRatio,
			footerRatio:           horsePurchaseFooterRatio,
			minimumInset:          horsePurchaseMinimumInset,
			lineSpacingMultiplier: horsePurchaseLineSpacing,
			footerMinimumLines:    1,
		}),
		headerOutput: nil,
		bodyOutput:   nil,
		footerOutput: nil,
	}
	config.Keyboard.PrepareForTransition(0)
	dialog.ensureLayout()

	return dialog
}

// Draw renders the centered opaque horse approval modal.
func (d *HorsePurchaseDialog) Draw(screen *ebiten.Image) {
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
	d.bodyOutput.DrawPanelContent(
		screen,
		image.Pt(d.frame.bodyText.Min.X, verticallyCenteredOutputY(d.frame.bodyText, d.bodyOutput)),
		d.frame.bodyText.Dx(),
		etext.AlignStart,
	)
	d.footerOutput.DrawPanelContent(
		screen,
		image.Pt(d.frame.footerText.Min.X, verticallyCenteredOutputY(d.frame.footerText, d.footerOutput)),
		d.frame.footerText.Dx(),
		etext.AlignCenter,
	)
}

// Update confirms or cancels the horse approval after transition-key release.
func (d *HorsePurchaseDialog) Update() {
	if d.dismissed {
		return
	}
	d.ensureLayout()
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) {
		d.cancelled = true
		d.dismissed = true

		return
	}
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) ||
		d.keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter) {
		request := d.intent.Request
		d.pending = &request
		d.dismissed = true
	}
}

// HasPendingPurchaseRequest reports whether confirmation emitted an unconsumed request.
func (d *HorsePurchaseDialog) HasPendingPurchaseRequest() bool { return d.pending != nil }

// ConsumePendingPurchaseRequest returns and clears the one confirmed authority request.
func (d *HorsePurchaseDialog) ConsumePendingPurchaseRequest() (game_state.HorsePurchaseRequest, bool) {
	if d.pending == nil {
		return emptyHorsePurchaseRequest(), false
	}
	request := *d.pending
	d.pending = nil

	return request, true
}

// GetIntent returns the detached approval data displayed by the modal.
func (d *HorsePurchaseDialog) GetIntent() HorsePurchaseIntent { return d.intent }

// GetTitleText returns the exact purchase title.
func (d *HorsePurchaseDialog) GetTitleText() string {
	if d.headerOutput == nil {
		return ""
	}

	return d.headerOutput.GetOutputStr(false)
}

// GetBodyText returns the horse, price, and party-gold consequence presentation.
func (d *HorsePurchaseDialog) GetBodyText() string {
	if d.bodyOutput == nil {
		return ""
	}

	return d.bodyOutput.GetOutputStr(false)
}

// GetFooterGuidance returns the exact keyboard-only approval controls.
func (d *HorsePurchaseDialog) GetFooterGuidance() string {
	return "[Enter] Confirm   [Esc] Back"
}

// GetBounds returns the centered modal bounds.
func (d *HorsePurchaseDialog) GetBounds() image.Rectangle {
	d.ensureLayout()

	return d.frame.bounds()
}

// GetBodyBounds returns the framed approval body used for semantic-background checks.
func (d *HorsePurchaseDialog) GetBodyBounds() image.Rectangle {
	d.ensureLayout()

	return d.frame.bodyBounds
}

// GetFontSize returns the standard merchant font size.
func (d *HorsePurchaseDialog) GetFontSize() StandardFontSize { return FontSizeMainScreen }

// IsPreview reports whether this approval stops before purchase authority.
func (d *HorsePurchaseDialog) IsPreview() bool { return d.intent.Preview }

// IsCancelled reports whether Escape closed the modal without confirmation.
func (d *HorsePurchaseDialog) IsCancelled() bool { return d.cancelled }

// IsDismissed reports whether the modal may be removed.
func (d *HorsePurchaseDialog) IsDismissed() bool { return d.dismissed }

// IsVisible reports whether the modal still owns presentation and input.
func (d *HorsePurchaseDialog) IsVisible() bool { return !d.dismissed }

// Dismiss closes the modal without emitting a request.
func (d *HorsePurchaseDialog) Dismiss() { d.dismissed = true }

func (d *HorsePurchaseDialog) ensureLayout() {
	if !d.frame.ensureLayout() {
		return
	}
	_, d.headerOutput = GetScaledFontWithOutput(FontConfig{
		Size: FontSizeMainScreen, LineSpacing: 0, LineSpacingMultiplier: horsePurchaseLineSpacing,
		MaxLines: 1, MaxChars: horsePurchaseMinimumTextChars,
	})
	bodyFont, bodyOutput := GetScaledFontWithOutput(FontConfig{
		Size: FontSizeMainScreen, LineSpacing: 0, LineSpacingMultiplier: horsePurchaseLineSpacing,
		MaxLines: horsePurchaseMaximumBodyLines, MaxChars: horsePurchaseMinimumTextChars,
	})
	_, d.footerOutput = GetScaledFontWithOutput(FontConfig{
		Size: FontSizeMainScreen, LineSpacing: 0, LineSpacingMultiplier: horsePurchaseLineSpacing,
		MaxLines: horsePurchaseFooterLines, MaxChars: horsePurchaseMinimumTextChars,
	})
	d.headerOutput.SetMaxCharsPerLine(max(
		horsePurchaseMinimumTextChars,
		d.headerOutput.Font.CharsForWidth(max(1, d.frame.headerText.Dx())),
	))
	d.bodyOutput = text.NewOutput(
		bodyFont,
		bodyOutput.GetLineSpacing(),
		max(1, bodyOutput.GetLineCapacity(max(1, d.frame.bodyText.Dy()))),
		max(horsePurchaseMinimumTextChars, bodyFont.CharsForWidth(max(1, d.frame.bodyText.Dx()))),
	)
	d.footerOutput.SetMaxCharsPerLine(max(
		horsePurchaseMinimumTextChars,
		d.footerOutput.Font.CharsForWidth(max(1, d.frame.footerText.Dx())),
	))
	d.populateOutputs()
}

func (d *HorsePurchaseDialog) populateOutputs() {
	if d.headerOutput == nil || d.bodyOutput == nil || d.footerOutput == nil {
		return
	}
	d.headerOutput.Reset()
	d.headerOutput.AppendLine().WithAlignment(etext.AlignCenter).
		Text("Purchase Horse", DialogHeaderColor).Done()

	d.bodyOutput.Reset()
	if d.intent.Preview {
		d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).
			Text("Preview", DialogStatusWarningColor).Done()
	}
	d.bodyOutput.AppendLine().WithAlignment(etext.AlignCenter).White("Horse").Done()
	d.bodyOutput.AppendLine().Yellow("Price: ").Whitef("%d gold", d.intent.Quote.Price).Done()
	if d.intent.Quote.Affordable {
		d.bodyOutput.AppendLine().Yellow("Party gold: ").Cyanf("%d", d.intent.Quote.GoldBefore).
			White(" -> ").LightGreenf("%d", d.intent.Quote.GoldAfterPayment).Done()
	} else {
		d.bodyOutput.AppendLine().Yellow("Party gold: ").Cyanf("%d", d.intent.Quote.GoldBefore).Done()
		d.bodyOutput.AppendLine().Yellow("Needed: ").Whitef("%d gold", d.intent.Quote.Price).Done()
	}

	d.footerOutput.Reset()
	d.footerOutput.AppendLine().WithAlignment(etext.AlignCenter).
		White("[Enter]").LightGray(" Confirm   ").White("[Esc]").LightGray(" Back").WordWrap().Done()
}

func horsePurchasePlacement() sprites.PercentBasedPlacement {
	return sprites.PercentBasedPlacement{
		StartPercentX: horsePurchaseStartPercentX,
		EndPercentX:   horsePurchaseEndPercentX,
		StartPercentY: horsePurchaseStartPercentY,
		EndPercentY:   horsePurchaseEndPercentY,
	}
}

func validateHorsePurchaseIntent(intent HorsePurchaseIntent) {
	quote := intent.Quote
	request := intent.Request
	if quote.BasePrice <= 0 || quote.Price <= 0 || quote.Price > math.MaxUint16 {
		panic("HorsePurchaseDialog: valid horse quote is required")
	}
	validateHorseQuoteGold("HorsePurchaseDialog", quote)
	if request.ReviewedMerchantID == "" || request.ReviewedPricingCharacterIndex < 0 ||
		request.ReviewedPrice != quote.Price || request.ReviewedGoldBefore != quote.GoldBefore ||
		request.ReviewedPlacement != quote.Placement {
		panic("HorsePurchaseDialog: detached request does not match quote")
	}
	validateHorsePlacementStatus(
		"HorsePurchaseDialog",
		quote.Placement.BlockStatus,
		intent.Preview,
		false,
	)
}
