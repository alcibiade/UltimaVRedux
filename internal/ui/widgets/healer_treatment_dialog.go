package widgets

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

const (
	healerTreatmentStartPercentX    = 0.30
	healerTreatmentEndPercentX      = 0.70
	healerTreatmentStartPercentY    = 0.22
	healerTreatmentEndPercentY      = 0.78
	healerTreatmentHeaderRatio      = 0.16
	healerTreatmentFooterRatio      = 0.20
	healerTreatmentMinimumInset     = 6
	healerTreatmentLineSpacing      = 1.12
	healerTreatmentMaximumBodyLines = 8
	healerTreatmentMinimumTextChars = 8
	healerTreatmentFooterLines      = 1
)

var _ Widget = (*HealerTreatmentDialog)(nil)

// HealerTreatmentDialogConfig configures one nested treatment approval modal.
type HealerTreatmentDialogConfig struct {
	Keyboard *input.Keyboard
	Intent   HealerTreatmentIntent
}

// HealerTreatmentDialog presents target, effect, and charge before authority.
type HealerTreatmentDialog struct {
	keyboard  *input.Keyboard
	intent    HealerTreatmentIntent
	dismissed bool
	cancelled bool
	pending   *game_state.HealerServiceRequest

	frame *merchantModalFrame

	headerOutput *text.Output
	bodyOutput   *text.Output
	footerOutput *text.Output
}

// NewHealerTreatmentDialog creates a centered keyboard-only approval modal.
func NewHealerTreatmentDialog(config HealerTreatmentDialogConfig) *HealerTreatmentDialog {
	if config.Keyboard == nil {
		panic("HealerTreatmentDialog: keyboard is required")
	}
	validateHealerTreatmentIntent(config.Intent)
	intent := config.Intent
	if config.Intent.Recipient != nil {
		recipient := *config.Intent.Recipient
		intent.Recipient = &recipient
	}
	dialog := &HealerTreatmentDialog{
		keyboard:  config.Keyboard,
		intent:    intent,
		dismissed: false,
		cancelled: false,
		pending:   nil,
		frame: newMerchantModalFrame(merchantModalFrameConfig{
			placement:             healerTreatmentPlacement(),
			headerRatio:           healerTreatmentHeaderRatio,
			footerRatio:           healerTreatmentFooterRatio,
			minimumInset:          healerTreatmentMinimumInset,
			lineSpacingMultiplier: healerTreatmentLineSpacing,
			footerMinimumLines:    healerTreatmentFooterLines,
		}),
		headerOutput: nil,
		bodyOutput:   nil,
		footerOutput: nil,
	}
	dialog.ensureLayout()

	return dialog
}

// Draw renders the nested approval modal.
func (d *HealerTreatmentDialog) Draw(screen *ebiten.Image) {
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

// Update confirms or cancels the treatment approval.
func (d *HealerTreatmentDialog) Update() {
	if d.dismissed {
		return
	}
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

// ConsumePendingRequest returns and clears the approved treatment request.
func (d *HealerTreatmentDialog) ConsumePendingRequest() (game_state.HealerServiceRequest, bool) {
	if d.pending == nil {
		return emptyHealerServiceRequest(), false
	}
	request := *d.pending
	d.pending = nil

	return request, true
}

// GetBounds returns the centered modal bounds.
func (d *HealerTreatmentDialog) GetBounds() image.Rectangle {
	d.ensureLayout()

	return d.frame.bounds()
}

// IsCancelled reports whether Escape closed the modal.
func (d *HealerTreatmentDialog) IsCancelled() bool { return d.cancelled }

// IsDismissed reports whether the modal can be removed.
func (d *HealerTreatmentDialog) IsDismissed() bool { return d.dismissed }

// Dismiss closes the modal without approving treatment.
func (d *HealerTreatmentDialog) Dismiss() { d.dismissed = true }

func (d *HealerTreatmentDialog) ensureLayout() {
	if !d.frame.ensureLayout() {
		return
	}
	d.headerOutput, d.bodyOutput, d.footerOutput = d.frame.newOutputs(merchantModalTextBudget{
		lineSpacing:      healerTreatmentLineSpacing,
		maximumBodyLines: healerTreatmentMaximumBodyLines,
		footerLines:      healerTreatmentFooterLines,
		minimumChars:     healerTreatmentMinimumTextChars,
	})
	d.populateOutputs()
}

func (d *HealerTreatmentDialog) populateOutputs() {
	if d.headerOutput == nil || d.bodyOutput == nil || d.footerOutput == nil {
		return
	}
	d.headerOutput.Reset()
	d.headerOutput.AppendLine().WithAlignment(etext.AlignCenter).
		Text("Approve "+healerServiceDisplayName(d.intent.Request.Service), DialogHeaderColor).Done()
	d.bodyOutput.Reset()
	if d.intent.IsAll() {
		d.populateAllBody()
	} else {
		d.populateRecipientBody()
	}
	d.footerOutput.Reset()
	appendControlGuidance(d.footerOutput, "[Enter] Confirm   [Esc] Back", d.frame.footerText.Dx())
}

func (d *HealerTreatmentDialog) populateRecipientBody() {
	recipient := d.intent.Recipient
	projection := recipient.Treatments[int(d.intent.Request.Service)]
	d.bodyOutput.AppendLine().Yellow("Target: ").White(recipient.Name).Done()
	d.bodyOutput.AppendLine().
		White(healerProjectionText(d.intent.Request.Service, *recipient, projection)).
		WordWrap().Done()
	priceText, priceColor := healerChargeText(d.intent.Service, d.intent.Gold)
	d.bodyOutput.AppendLine().Text(priceText, priceColor).Done()
}

func (d *HealerTreatmentDialog) populateAllBody() {
	count := d.intent.AllService.RecipientCount
	label := "party members"
	if count == 1 {
		label = "party member"
	}
	d.bodyOutput.AppendLine().Yellow("Target: ").Whitef("Roster order (%d)", count).Done()
	d.bodyOutput.AppendLine().Whitef("%s: up to %d %s.", d.intent.Request.Service, count, label).Done()
	priceText, priceColor := healerAllChargeText(d.intent.AllService, d.intent.Gold)
	d.bodyOutput.AppendLine().Text(priceText, priceColor).Done()
}

func healerTreatmentPlacement() sprites.PercentBasedPlacement {
	return sprites.PercentBasedPlacement{
		StartPercentX: healerTreatmentStartPercentX,
		EndPercentX:   healerTreatmentEndPercentX,
		StartPercentY: healerTreatmentStartPercentY,
		EndPercentY:   healerTreatmentEndPercentY,
	}
}

func validateHealerTreatmentIntent(intent HealerTreatmentIntent) {
	if intent.Request.Service != intent.Service.Service ||
		intent.Request.Service != intent.AllService.Service {
		panic("HealerTreatmentDialog: service quotes do not match request")
	}
	if intent.IsAll() {
		if intent.Recipient != nil || intent.AllService.RecipientCount == 0 {
			panic("HealerTreatmentDialog: aggregate treatment requires eligible recipients")
		}

		return
	}
	if intent.Recipient == nil || intent.Recipient.CharacterIndex != intent.Request.CharacterIndex ||
		!intent.Recipient.Treatments[int(intent.Request.Service)].Eligibility.IsEligible() {
		panic("HealerTreatmentDialog: eligible detached recipient is required")
	}
}
