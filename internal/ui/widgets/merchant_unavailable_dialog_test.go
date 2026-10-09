package widgets

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestMerchantUnavailableDialogUsesCompactShellAndKeyboardLeave(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	keyboard := newWidgetTestKeyboard(0)
	dialog := newTestMerchantUnavailableDialog(keyboard)

	require.Equal(t, "Iolo's Bows", dialog.GetShell().GetHeaderView().ShopIdentity)
	require.Equal(t, "Gwenneth", dialog.GetShell().GetHeaderView().Owner)
	require.Equal(t, "500", dialog.GetShell().GetHeaderView().PartyGold)
	require.Equal(t, "Arms dealer service is not yet implemented.", dialog.GetShell().GetExchangeView().Text)
	require.Equal(t, FontSizeMainScreen, dialog.GetShell().GetFontSize())
	guidance := strings.ToLower(dialog.GetShell().GetFooterView().Guidance)
	require.NotContains(t, guidance, "mouse")
	require.NotContains(t, guidance, "arrow")
	dialog.GetShell().ensureLayout()

	keyboard.SimulateKeyPress(ebiten.KeyEscape)
	dialog.Update()
	require.True(t, dialog.IsDismissed())
	require.False(t, dialog.IsVisible())
}

func TestMerchantUnavailableDialogEnterDismisses(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	dialog := newTestMerchantUnavailableDialog(keyboard)

	keyboard.SimulateKeyPress(ebiten.KeyEnter)
	dialog.Update()
	require.True(t, dialog.IsDismissed())
}

func TestMerchantUnavailableDialogConstructorContracts(t *testing.T) {
	valid := MerchantUnavailableDialogConfig{
		Keyboard:       newWidgetTestKeyboard(0),
		MerchantSprite: ebiten.NewImage(16, 16),
		ShopName:       "Iolo's Bows",
		OwnerName:      "Gwenneth",
		PartyGold:      500,
		Message:        "Arms dealer service is not yet implemented.",
	}

	tests := []struct {
		name   string
		mutate func(*MerchantUnavailableDialogConfig)
	}{
		{name: "keyboard", mutate: func(config *MerchantUnavailableDialogConfig) { config.Keyboard = nil }},
		{name: "sprite", mutate: func(config *MerchantUnavailableDialogConfig) { config.MerchantSprite = nil }},
		{name: "shop", mutate: func(config *MerchantUnavailableDialogConfig) { config.ShopName = "" }},
		{name: "owner", mutate: func(config *MerchantUnavailableDialogConfig) { config.OwnerName = "" }},
		{name: "message", mutate: func(config *MerchantUnavailableDialogConfig) { config.Message = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.mutate(&config)
			require.Panics(t, func() { NewMerchantUnavailableDialog(config) })
		})
	}
}

func newTestMerchantUnavailableDialog(keyboard *input.Keyboard) *MerchantUnavailableDialog {
	return NewMerchantUnavailableDialog(MerchantUnavailableDialogConfig{
		Keyboard:       keyboard,
		MerchantSprite: ebiten.NewImage(16, 16),
		ShopName:       "Iolo's Bows",
		OwnerName:      "Gwenneth",
		PartyGold:      500,
		Message:        "Arms dealer service is not yet implemented.",
	})
}
