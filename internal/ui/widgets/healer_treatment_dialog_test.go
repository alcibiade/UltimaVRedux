package widgets

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestHealerTreatmentDialogShowsIndividualTargetEffectAndCharge(t *testing.T) {
	setMerchantShellTestResolution(t, 320, 180)
	keyboard := newWidgetTestKeyboard(0)
	shop := newTestHealerShopDialog(keyboard, testHealerShopState())
	pressHealerShopKey(shop, keyboard, ebiten.KeyEnter)
	pressHealerShopKey(shop, keyboard, ebiten.KeyEnter)
	intent, ok := shop.ConsumePendingTreatmentIntent()
	require.True(t, ok)

	dialog := NewHealerTreatmentDialog(HealerTreatmentDialogConfig{Keyboard: keyboard, Intent: intent})
	body := strings.Join(strings.Fields(dialog.GetBodyText()), " ")
	lowBounds := dialog.GetBounds()

	assert.Contains(t, body, "Target: Shamino")
	assert.Contains(t, body, "Poisoned to Good")
	assert.Contains(t, body, "Free care")
	assert.Equal(t, "Enter: Confirm  Esc: Back", dialog.GetFooterGuidance())
	assert.Equal(t, FontSizeBody, dialog.GetFontSize())
	setResponsiveDialogTestResolution(t, 1280, 720)
	dialog.ensureLayout()
	require.Greater(t, dialog.GetBounds().Dx(), lowBounds.Dx())
	require.Equal(t, body, strings.Join(strings.Fields(dialog.GetBodyText()), " "))
	require.Equal(t, intent, dialog.GetIntent())
	pressHealerTreatmentKey(dialog, keyboard, ebiten.KeyEnter)
	request, ok := dialog.ConsumePendingRequest()
	require.True(t, ok)
	require.Equal(t, intent.Request, request)
}

func TestHealerTreatmentDialogAllIsShortcutTargetNotPartyRow(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	shop := newTestHealerShopDialog(keyboard, testHealerShopState())
	require.Len(t, shop.recipientRows, healerShopMaximumRecipients)
	pressHealerShopKey(shop, keyboard, ebiten.KeyA)
	intent, ok := shop.ConsumePendingTreatmentIntent()
	require.True(t, ok)

	dialog := NewHealerTreatmentDialog(HealerTreatmentDialogConfig{Keyboard: keyboard, Intent: intent})
	body := strings.Join(strings.Fields(dialog.GetBodyText()), " ")

	assert.Contains(t, body, "Target: Roster order (1)")
	assert.Contains(t, body, "Cure: up to 1 party member")
	assert.Contains(t, body, "Free care for all")
}

func TestHealerTreatmentDialogConfirmAndCancelLifecycle(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	shop := newTestHealerShopDialog(keyboard, testHealerShopState())
	pressHealerShopKey(shop, keyboard, ebiten.KeyEnter)
	pressHealerShopKey(shop, keyboard, ebiten.KeyEnter)
	intent, ok := shop.ConsumePendingTreatmentIntent()
	require.True(t, ok)

	confirmed := NewHealerTreatmentDialog(HealerTreatmentDialogConfig{Keyboard: keyboard, Intent: intent})
	pressHealerTreatmentKey(confirmed, keyboard, ebiten.KeyEnter)
	require.True(t, confirmed.IsDismissed())
	request, ok := confirmed.ConsumePendingRequest()
	require.True(t, ok)
	assert.Equal(t, game_state.HealerServiceRequest{
		Service:        references.HealerCure,
		CharacterIndex: 0,
	}, request)

	cancelled := NewHealerTreatmentDialog(HealerTreatmentDialogConfig{Keyboard: keyboard, Intent: intent})
	pressHealerTreatmentKey(cancelled, keyboard, ebiten.KeyEscape)
	assert.True(t, cancelled.IsDismissed())
	assert.True(t, cancelled.IsCancelled())
	_, ok = cancelled.ConsumePendingRequest()
	assert.False(t, ok)
}

func pressHealerTreatmentKey(
	dialog *HealerTreatmentDialog,
	keyboard *input.Keyboard,
	key ebiten.Key,
) {
	keyboard.SimulateKeyPress(key)
	dialog.Update()
}
