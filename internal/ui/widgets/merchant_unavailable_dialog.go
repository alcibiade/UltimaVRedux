package widgets

import (
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

var _ Widget = (*MerchantUnavailableDialog)(nil)

// MerchantUnavailableDialogConfig contains finalized presentation data for an
// exact merchant whose family transactions have not been implemented yet.
type MerchantUnavailableDialogConfig struct {
	Keyboard       *input.Keyboard
	MerchantSprite *ebiten.Image
	ShopName       string
	OwnerName      string
	PartyGold      uint16
	Message        string
}

// MerchantUnavailableDialog presents an exact merchant through the compact shell.
type MerchantUnavailableDialog struct {
	keyboard  *input.Keyboard
	shell     *MerchantDialogShell
	dismissed bool
}

// NewMerchantUnavailableDialog creates a shell-backed, keyboard-only merchant dialog.
func NewMerchantUnavailableDialog(config MerchantUnavailableDialogConfig) *MerchantUnavailableDialog {
	if config.Keyboard == nil {
		panic("MerchantUnavailableDialog: keyboard is required")
	}
	if config.MerchantSprite == nil {
		panic("MerchantUnavailableDialog: merchant sprite is required")
	}
	if strings.TrimSpace(config.ShopName) == "" || strings.TrimSpace(config.OwnerName) == "" {
		panic("MerchantUnavailableDialog: shop and owner are required")
	}
	if strings.TrimSpace(config.Message) == "" {
		panic("MerchantUnavailableDialog: message is required")
	}

	return &MerchantUnavailableDialog{
		keyboard: config.Keyboard,
		shell: NewMerchantDialogShell(MerchantDialogShellConfig{
			Header: MerchantDialogHeaderView{
				ShopIdentity: config.ShopName,
				Owner:        config.OwnerName,
				PartyGold:    strconv.FormatUint(uint64(config.PartyGold), 10),
			},
			Exchange:       MerchantDialogExchangeView{Text: config.Message},
			Footer:         MerchantDialogFooterView{Guidance: "Enter or Esc: Leave"},
			MerchantSprite: config.MerchantSprite,
		}),
		dismissed: false,
	}
}

// Draw renders the compact common merchant shell.
func (d *MerchantUnavailableDialog) Draw(screen *ebiten.Image) {
	if screen == nil || d.dismissed {
		return
	}
	d.shell.DrawBackground(screen)
	d.shell.DrawCommonRegions(screen)
}

// Update accepts keyboard-only Enter or Escape dismissal.
func (d *MerchantUnavailableDialog) Update() {
	if d.dismissed {
		return
	}
	if handleMerchantSpeechPaging(d.shell, d.keyboard) {
		return
	}
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape) ||
		d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) ||
		d.keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter) {
		d.Dismiss()
	}
}

// Dismiss closes this concrete dialog.
func (d *MerchantUnavailableDialog) Dismiss() {
	d.dismissed = true
}

// IsDismissed reports whether the dialog has completed.
func (d *MerchantUnavailableDialog) IsDismissed() bool {
	return d.dismissed
}

// GetShell returns the concrete dialog's shared presentation shell.
func (d *MerchantUnavailableDialog) GetShell() *MerchantDialogShell {
	return d.shell
}
