package widgets

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestTavernActionDialogNormalActionsEmitExactRequestsUnchanged(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	tests := []struct {
		name   string
		intent TavernActionIntent
		text   string
	}{
		{name: "meal", intent: testTavernMealIntent(t), text: "3 diners x 4 gold each"},
		{name: "drink", intent: testTavernOrdinaryDrinkIntent(t), text: "3 drinks x 1 gold each"},
		{name: "provisions", intent: testTavernProvisionIntent(t), text: "PACKS < 1 >"},
		{name: "information", intent: testTavernInformationIntent(t), text: "Price: 50 gold"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keyboard := newWidgetTestKeyboard(0)
			dialog := NewTavernActionDialog(TavernActionDialogConfig{
				Keyboard: keyboard, Intent: test.intent, IsKeyPressed: func(ebiten.Key) bool { return false },
			})
			require.Contains(t, compactTavernText(dialog.GetBodyText()), test.text)
			wantFooter := tavernActionNormalFooter
			if test.intent.Action == TavernActionProvision {
				wantFooter = tavernActionProvisionFooter
			}
			require.Equal(t, wantFooter, dialog.GetFooterGuidance())
			require.Equal(t, test.intent, dialog.GetIntent())
			if test.intent.Action == TavernActionProvision {
				detached := dialog.GetIntent()
				detached.Provision.Approvals[0].TotalCost++
				require.NotEqual(t, detached, dialog.GetIntent())
			}

			pressTavernActionKey(dialog, keyboard, ebiten.KeyEnter)
			request, ok := dialog.ConsumePendingRequest()
			require.True(t, ok)
			require.Equal(t, test.intent.Request, request)
			_, duplicate := dialog.ConsumePendingRequest()
			require.False(t, duplicate)
			require.True(t, dialog.IsComplete())
			require.False(t, dialog.IsVisible())
		})
	}
}

func TestTavernActionDialogWarningUsesImmersiveEnterOrEscapeChoice(t *testing.T) {
	state := testTavernState(references.TavernMenuMuttonAleRations)
	setTestTavernDrunk(&state, 3)
	intent := testTavernIntentForOrderKey(t, state, ebiten.KeyA)
	require.True(t, intent.Drink.Warning)

	t.Run("Enter accepts another drink", func(t *testing.T) {
		keyboard := newWidgetTestKeyboard(0)
		dialog := NewTavernActionDialog(TavernActionDialogConfig{
			Keyboard: keyboard, Intent: intent, IsKeyPressed: func(ebiten.Key) bool { return false },
		})
		require.Equal(t, tavernActionWarningFooter, dialog.GetFooterGuidance())
		body := compactTavernText(dialog.GetBodyText())
		require.Contains(t, body, "But haven't ye had enough to drink?")
		require.Contains(t, body, "3 drinks x 1 gold each")
		require.Contains(t, body, "Total: 3 gold")
		require.Contains(t, body, "Gold after payment: 1000 -> 997")
		require.NotContains(t, body, "smashed")
		require.NotContains(t, body, "karma")
		require.NotContains(t, body, "customer")
		pressTavernActionKey(dialog, keyboard, ebiten.KeyEnter)
		request, ok := dialog.ConsumePendingRequest()
		require.True(t, ok)
		require.Equal(t, intent.Request, request)
		require.Equal(t, game_state.TavernDrinkApprovalContinue, request.Drink.Approval)
	})

	t.Run("Escape has enough", func(t *testing.T) {
		keyboard := newWidgetTestKeyboard(0)
		dialog := NewTavernActionDialog(TavernActionDialogConfig{
			Keyboard: keyboard, Intent: intent, IsKeyPressed: func(ebiten.Key) bool { return false },
		})
		pressTavernActionKey(dialog, keyboard, ebiten.KeyEscape)
		request, ok := dialog.ConsumePendingRequest()
		require.True(t, ok)
		require.Equal(t, intent.StopRequest, request)
		require.Equal(t, game_state.TavernStopRequest{
			ReviewedMerchantID: state.Merchant.ID, ReviewedDrunk: 3,
		}, request.Stop)
	})

	t.Run("letter shortcuts are inert", func(t *testing.T) {
		keyboard := newWidgetTestKeyboard(0)
		dialog := NewTavernActionDialog(TavernActionDialogConfig{
			Keyboard: keyboard, Intent: intent, IsKeyPressed: func(ebiten.Key) bool { return false },
		})
		pressTavernActionKey(dialog, keyboard, ebiten.KeyS)
		pressTavernActionKey(dialog, keyboard, ebiten.KeyC)
		require.False(t, dialog.HasPendingRequest())
		require.False(t, dialog.IsDismissed())
	})
}

func TestTavernActionDialogEscapeCancelsWithoutRequestAndParentReturnsToWine(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	parent := newTestTavernShopDialog(keyboard, testTavernState(references.TavernMenuWineCheese), nil)
	pressTavernShopKey(parent, keyboard, ebiten.KeyW)
	pressTavernShopKey(parent, keyboard, ebiten.KeyD)
	intent, ok := parent.ConsumePendingActionIntent()
	require.True(t, ok)

	action := NewTavernActionDialog(TavernActionDialogConfig{
		Keyboard: keyboard, Intent: intent, IsKeyPressed: func(ebiten.Key) bool { return false },
	})
	pressTavernActionKey(action, keyboard, ebiten.KeyEscape)
	require.True(t, action.IsCancelled())
	require.False(t, action.HasPendingRequest())
	parent.CancelActionApproval()
	require.Equal(t, TavernOrderWinePicker, parent.GetOrderSubflow())
	require.Equal(t, 3, parent.GetSelectedWineIndex())
	require.False(t, parent.IsDismissed())
}

func TestTavernActionDialogInformationAppearsOnlyAfterAuthorityQuote(t *testing.T) {
	queue := newTavernRuneQueue()
	state := testTavernState(references.TavernMenuMuttonAleRations)
	state.CustomerEligible = true
	keyboard := newWidgetTestKeyboard(0)
	parent := newTestTavernShopDialog(keyboard, state, queue.Append)
	pressTavernShopKey(parent, keyboard, ebiten.KeyC)
	queue.Push("honesty")
	parent.Update()
	queue.Push("")
	require.Contains(t, strings.ToLower(parent.GetWorkspaceText()), "honesty")
	pressTavernShopKey(parent, keyboard, ebiten.KeyEnter)
	topic, ok := parent.ConsumePendingTopicIntent()
	require.True(t, ok)
	parent.ApplyInformationQuote(topic, testTavernInformationQuote(parent.GetState(), "hone", 50), "I may know someone.")
	intent, ok := parent.ConsumePendingActionIntent()
	require.True(t, ok)

	action := NewTavernActionDialog(TavernActionDialogConfig{
		Keyboard: keyboard, Intent: intent, IsKeyPressed: func(ebiten.Key) bool { return false },
	})
	body := compactTavernText(action.GetBodyText())
	require.Contains(t, body, "Price: 50 gold")
	require.NotContains(t, body, "Recognized subject")
	require.NotContains(t, body, "hone")
	require.NotContains(t, body, "honesty", "approval exposes authority's recognized subject, not unmatched raw input")
}

func TestTavernActionDialogHeldControlsWaitForReleaseWithoutAccidentalChoice(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyEnter: true, ebiten.KeyEscape: true, ebiten.KeyS: true, ebiten.KeyC: true}
	keyboard := newWidgetTestKeyboard(0)
	dialog := NewTavernActionDialog(TavernActionDialogConfig{
		Keyboard: keyboard, Intent: testTavernMealIntent(t),
		IsKeyPressed: func(key ebiten.Key) bool { return held[key] },
	})
	dialog.Update()
	require.False(t, dialog.IsDismissed())
	require.False(t, dialog.HasPendingRequest())

	for key := range held {
		held[key] = false
	}
	dialog.Update()
	require.False(t, dialog.IsDismissed(), "release frame must not confirm or cancel")
	pressTavernActionKey(dialog, keyboard, ebiten.KeyEnter)
	require.True(t, dialog.HasPendingRequest())
}

func TestTavernActionDialogPreviewResponsiveSemanticFooterAndLifecycle(t *testing.T) {
	setMerchantShellTestResolution(t, 320, 180)
	intent := testTavernMealIntent(t)
	intent.Preview = true
	dialog := NewTavernActionDialog(TavernActionDialogConfig{
		Keyboard: newWidgetTestKeyboard(0), Intent: intent, IsKeyPressed: func(ebiten.Key) bool { return false },
	})
	require.True(t, dialog.IsPreview())
	require.Contains(t, compactTavernText(dialog.GetTitleText()), "PREVIEW - Approve Meal")
	require.Contains(t, compactTavernText(dialog.GetBodyText()), "PREVIEW: authority has not run")
	require.False(t, dialog.GetBounds().Empty())
	require.Equal(t, FontSizeMainScreen, dialog.GetFontSize())

	for _, line := range parseMerchantFooterGuidance(dialog.GetFooterGuidance()) {
		for _, segment := range line {
			if strings.HasPrefix(segment.Text, "[") {
				require.Equal(t, DialogControlKeyColor, segment.Color)
			} else {
				require.Equal(t, DialogControlHintColor, segment.Color)
			}
		}
	}
	require.True(t, dialog.IsVisible())
	dialog.Dismiss()
	require.True(t, dialog.IsComplete())
	require.False(t, dialog.IsVisible())
	dialog.ShowWithoutAnimation()
	require.False(t, dialog.IsVisible())
}

func TestTavernActionDialogProvisionResizePreservesQuantityAndRequest(t *testing.T) {
	setMerchantShellTestResolution(t, 320, 180)
	keyboard := newWidgetTestKeyboard(0)
	dialog := NewTavernActionDialog(TavernActionDialogConfig{
		Keyboard: keyboard,
		Intent:   testTavernProvisionIntent(t),
		IsKeyPressed: func(ebiten.Key) bool {
			return false
		},
	})
	pressTavernActionKey(dialog, keyboard, ebiten.KeyRight)
	lowBounds := dialog.GetBounds()
	wantBody := dialog.GetBodyText()
	require.Equal(t, 2, dialog.provisionPacks)
	require.Equal(t, 2, dialog.frame.config.footerMinimumLines)

	setResponsiveDialogTestResolution(t, 1280, 720)
	dialog.ensureLayout()
	require.Greater(t, dialog.GetBounds().Dx(), lowBounds.Dx())
	require.Equal(t, 2, dialog.provisionPacks)
	require.Equal(t, wantBody, dialog.GetBodyText())

	pressTavernActionKey(dialog, keyboard, ebiten.KeyEnter)
	request, ok := dialog.ConsumePendingRequest()
	require.True(t, ok)
	require.Equal(t, 2, request.Provision.PackCount)
}

func TestTavernActionDialogRejectsTamperedTypedIntent(t *testing.T) {
	valid := testTavernMealIntent(t)
	tests := []struct {
		name   string
		mutate func(*TavernActionDialogConfig)
	}{
		{name: "missing keyboard", mutate: func(config *TavernActionDialogConfig) { config.Keyboard = nil }},
		{name: "wrong action", mutate: func(config *TavernActionDialogConfig) {
			config.Intent.Request.Action = TavernActionDrink
		}},
		{name: "changed charge", mutate: func(config *TavernActionDialogConfig) {
			config.Intent.Request.Meal.ReviewedCharge++
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := TavernActionDialogConfig{
				Keyboard: newWidgetTestKeyboard(0), Intent: valid, IsKeyPressed: func(ebiten.Key) bool { return false },
			}
			test.mutate(&config)
			require.Panics(t, func() { NewTavernActionDialog(config) })
		})
	}
}

func TestTavernActionDialogRejectsTamperedProvisionProjection(t *testing.T) {
	intent := testTavernProvisionIntent(t)
	intent.Provision.Approvals[0].TotalCost++

	require.Panics(t, func() {
		NewTavernActionDialog(TavernActionDialogConfig{
			Keyboard: newWidgetTestKeyboard(0), Intent: intent, IsKeyPressed: func(ebiten.Key) bool { return false },
		})
	})
}

func testTavernMealIntent(t *testing.T) TavernActionIntent {
	t.Helper()

	return testTavernIntentForOrderKey(t, testTavernState(references.TavernMenuMuttonAleRations), ebiten.KeyM)
}

func testTavernOrdinaryDrinkIntent(t *testing.T) TavernActionIntent {
	t.Helper()

	return testTavernIntentForOrderKey(t, testTavernState(references.TavernMenuMuttonAleRations), ebiten.KeyA)
}

func testTavernProvisionIntent(t *testing.T) TavernActionIntent {
	t.Helper()
	keyboard := newWidgetTestKeyboard(0)
	parent := newTestTavernShopDialog(keyboard, testTavernState(references.TavernMenuStoutFruitProvisions), nil)
	pressTavernShopKey(parent, keyboard, ebiten.KeyP)
	intent, ok := parent.ConsumePendingActionIntent()
	require.True(t, ok)
	intent.ProvisionPitch = `"Fresh provisions for the road."`

	return intent
}

func testTavernInformationIntent(t *testing.T) TavernActionIntent {
	t.Helper()
	queue := newTavernRuneQueue()
	state := testTavernState(references.TavernMenuMuttonAleRations)
	state.CustomerEligible = true
	keyboard := newWidgetTestKeyboard(0)
	parent := newTestTavernShopDialog(keyboard, state, queue.Append)
	pressTavernShopKey(parent, keyboard, ebiten.KeyC)
	queue.Push("honesty")
	parent.Update()
	queue.Push("")
	pressTavernShopKey(parent, keyboard, ebiten.KeyEnter)
	topic, ok := parent.ConsumePendingTopicIntent()
	require.True(t, ok)
	parent.ApplyInformationQuote(topic, testTavernInformationQuote(parent.GetState(), "hone", 50), "I may know someone.")
	intent, ok := parent.ConsumePendingActionIntent()
	require.True(t, ok)

	return intent
}

func testTavernIntentForOrderKey(
	t *testing.T,
	state game_state.TavernShopState,
	key ebiten.Key,
) TavernActionIntent {
	t.Helper()
	keyboard := newWidgetTestKeyboard(0)
	parent := newTestTavernShopDialog(keyboard, state, nil)
	pressTavernShopKey(parent, keyboard, key)
	intent, ok := parent.ConsumePendingActionIntent()
	require.True(t, ok)
	if intent.Drink.Warning {
		intent.DrinkWarning = `"I beg thy pardon, sir," says Tika. "But haven't ye had enough to drink?"`
	}

	return intent
}

func pressTavernActionKey(dialog *TavernActionDialog, keyboard *input.Keyboard, key ebiten.Key) {
	keyboard.SimulateKeyPress(key)
	dialog.Update()
}
