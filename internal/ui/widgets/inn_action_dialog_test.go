package widgets

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestInnActionDialogRestAffordableAndUnaffordableAreConfirmable(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	affordableIntent := testInnRestIntent(t, testInnShopState(), true)
	keyboard := newWidgetTestKeyboard(0)
	dialog := NewInnActionDialog(InnActionDialogConfig{
		Keyboard:     keyboard,
		Intent:       affordableIntent,
		IsKeyPressed: func(ebiten.Key) bool { return false },
	})
	body := compactInnText(dialog.GetBodyText())

	require.Equal(t, "PREVIEW - Approve Rest", compactInnText(dialog.GetTitleText()))
	require.Contains(t, body, "PREVIEW: authority has not run")
	require.Contains(t, body, "Listed charge: 1234 gold")
	require.Contains(t, body, "Gold after listed payment: 4321 -> 3087")
	require.Contains(t, body, "Wake: 6:03AM on 2-4-139")
	require.Contains(t, body, "Successful Rest ends active spells")
	require.Contains(t, body, "Unaffected living party members fully heal")
	require.Contains(t, body, "Dead and unchanged: Avatar")
	require.Contains(t, body, "Poisoned: Shamino, Mariahxxx")
	require.Contains(t, body, "WILL DIE during Rest")
	require.NotContains(t, body, "Guest rooms")
	require.NotContains(t, body, "Active party")
	require.NotContains(t, body, "HP 40 ->")
	require.Equal(t, "[Enter] Confirm   [Esc] Back", dialog.GetFooterGuidance())
	require.True(t, dialog.IsPreview())

	pressInnActionKey(dialog, keyboard, ebiten.KeyEnter)
	request, ok := dialog.ConsumePendingRequest()
	require.True(t, ok)
	require.Equal(t, affordableIntent.Request, request)

	poor := testInnShopState()
	setTestInnGold(&poor, 100)
	poorIntent := testInnRestIntent(t, poor, false)
	keyboard = newWidgetTestKeyboard(0)
	dialog = NewInnActionDialog(InnActionDialogConfig{
		Keyboard: keyboard, Intent: poorIntent, IsKeyPressed: func(ebiten.Key) bool { return false },
	})
	body = compactInnText(dialog.GetBodyText())
	require.Contains(t, body, "shortfall 1134")
	require.Contains(t, body, "Confirmation remains available")
	pressInnActionKey(dialog, keyboard, ebiten.KeyNumpadEnter)
	request, ok = dialog.ConsumePendingRequest()
	require.True(t, ok, "unaffordable Rest approval must still emit its exact request")
	require.Equal(t, poorIntent.Request, request)
}

func TestInnActionDialogRestSummarizesUniversalHealing(t *testing.T) {
	state := testInnShopState()
	for index := range state.Rest.MemberProjections {
		projection := &state.Rest.MemberProjections[index]
		projection.StatusBefore = party_state.Good
		projection.StatusAfter = party_state.Good
		projection.HPBefore = 1
		projection.HPAfter = projection.MaxHP
		projection.Warnings = nil
	}
	state.Rest.Warnings = nil
	dialog := NewInnActionDialog(InnActionDialogConfig{
		Keyboard: newWidgetTestKeyboard(0), Intent: testInnRestIntent(t, state, false),
		IsKeyPressed: func(ebiten.Key) bool { return false },
	})
	body := compactInnText(dialog.GetBodyText())

	require.Contains(t, body, "Everyone fully heals")
	require.NotContains(t, body, "Avatar:")
	require.NotContains(t, body, "Iolo:")
}

func TestInnActionDialogLeaveShowsExactDetailsAndSevereWarning(t *testing.T) {
	intent := testInnLeaveIntent(t, testInnShopState(), 2, false)
	dialog := NewInnActionDialog(InnActionDialogConfig{
		Keyboard: newWidgetTestKeyboard(0), Intent: intent, IsKeyPressed: func(ebiten.Key) bool { return false },
	})
	body := compactInnText(dialog.GetBodyText())

	require.Equal(t, "Approve Leave", compactInnText(dialog.GetTitleText()))
	require.Contains(t, body, "Companion: Shamino")
	require.Contains(t, body, "Inn: The Wayfarer Inn")
	require.Contains(t, body, "Charge today: 0 gold")
	require.Contains(t, body, "Current monthly rate: 1234 gold")
	require.Contains(t, body, "Minimum stay: one billed month")
	require.Contains(t, body, "Future rate: recalculated when picked up")
	require.Contains(t, body, "Status retained: Poisoned")
	require.Contains(t, body, "Party: 4/6 -> 3/6")
	require.Contains(t, body, "Guest rooms: 3/4 -> 4/4")
	require.Contains(t, body, "SEVERE: this removes the last living member")
}

func TestInnActionDialogPickupShowsPoisonAndInsufficientGuardWarnings(t *testing.T) {
	poison := testInnPickupIntent(t, testInnShopState(), 0, true)
	dialog := NewInnActionDialog(InnActionDialogConfig{
		Keyboard: newWidgetTestKeyboard(0), Intent: poison, IsKeyPressed: func(ebiten.Key) bool { return false },
	})
	body := compactInnText(dialog.GetBodyText())
	require.Contains(t, body, "PREVIEW: authority has not run")
	require.Contains(t, body, "Guest: Geoffreyy")
	require.Contains(t, body, "Recorded / billable months: 0 / 1")
	require.Contains(t, body, "Minimum charge is one month")
	require.Contains(t, body, "Monthly rate: 1234 gold")
	require.Contains(t, body, "Listed charge: 1234 gold")
	require.Contains(t, body, "Geoffreyy is Poisoned and WILL DIE")

	guard := testInnPickupIntent(t, testInnShopState(), 2, false)
	keyboard := newWidgetTestKeyboard(0)
	dialog = NewInnActionDialog(InnActionDialogConfig{
		Keyboard: keyboard, Intent: guard, IsKeyPressed: func(ebiten.Key) bool { return false },
	})
	body = compactInnText(dialog.GetBodyText())
	require.Contains(t, body, "Listed charge: 4936 gold")
	require.Contains(t, body, "Gold 4321; shortfall 615")
	require.Contains(t, body, "GUARD WARNING")
	require.Contains(t, body, "confirming without enough gold activates guards")
	pressInnActionKey(dialog, keyboard, ebiten.KeyEnter)
	request, ok := dialog.ConsumePendingRequest()
	require.True(t, ok, "guard-risk Pick Up remains confirmable")
	require.Equal(t, guard.Request, request)
}

func TestInnActionDialogDeadBlockedActionIsRejected(t *testing.T) {
	state := testInnShopState()
	intent := InnActionIntent{
		Action:         InnActionPickup,
		Request:        innPickupRequest(state, state.GuestRows[1]),
		Rest:           state.Rest,
		Pickup:         state.GuestRows[1],
		InnName:        state.Merchant.ShopName,
		RoomOccupancy:  state.RoomOccupancy,
		RoomCapacity:   state.RoomCapacity,
		PartySize:      state.PartySize,
		PartyCapacity:  state.PartyCapacity,
		MonthlyRate:    state.GuestRows[1].MonthlyRate,
		HasMonthlyRate: true,
		Preview:        false,
	}

	require.Panics(t, func() {
		NewInnActionDialog(InnActionDialogConfig{Keyboard: newWidgetTestKeyboard(0), Intent: intent})
	}, "dead stored guests must never reach the approval widget")
}

func TestInnActionDialogRequestAndIntentAreDefensiveCopies(t *testing.T) {
	intent := testInnRestIntent(t, testInnShopState(), false)
	wantRequest := intent.Request
	keyboard := newWidgetTestKeyboard(0)
	dialog := NewInnActionDialog(InnActionDialogConfig{
		Keyboard: keyboard, Intent: intent, IsKeyPressed: func(ebiten.Key) bool { return false },
	})

	intent.Rest.MemberProjections[0].Identity.Name = "MUTATED"
	intent.Rest.Warnings[0] = game_state.InnWarningNone
	copyIntent := dialog.GetIntent()
	require.Equal(t, "Avatar", copyIntent.Rest.MemberProjections[0].Identity.Name)
	require.Equal(t, game_state.InnWarningPoisonDeath, copyIntent.Rest.Warnings[0])
	copyIntent.Rest.MemberProjections[0].Identity.Name = "CHANGED AGAIN"
	require.Equal(t, "Avatar", dialog.GetIntent().Rest.MemberProjections[0].Identity.Name)

	pressInnActionKey(dialog, keyboard, ebiten.KeyEnter)
	request, ok := dialog.ConsumePendingRequest()
	require.True(t, ok)
	require.Equal(t, wantRequest, request)
	pressInnActionKey(dialog, keyboard, ebiten.KeyEnter)
	_, duplicate := dialog.ConsumePendingRequest()
	require.False(t, duplicate)
}

func TestInnActionDialogEscapeReturnsToParentAndReleaseDoesNotDismissParent(t *testing.T) {
	held := map[ebiten.Key]bool{}
	isPressed := func(key ebiten.Key) bool { return held[key] }
	keyboard := newWidgetTestKeyboard(0)
	parentConfig := testInnShopDialogConfig(keyboard, testInnShopState())
	parentConfig.IsKeyPressed = isPressed
	parent := NewInnShopDialog(parentConfig)
	pressInnShopKey(parent, keyboard, ebiten.KeyEnter)
	intent, ok := parent.ConsumePendingActionIntent()
	require.True(t, ok)

	action := NewInnActionDialog(InnActionDialogConfig{
		Keyboard: keyboard, Intent: intent, IsKeyPressed: isPressed,
	})
	pressInnActionKey(action, keyboard, ebiten.KeyEscape)
	require.True(t, action.IsCancelled())
	require.True(t, action.IsComplete())
	require.False(t, action.HasPendingRequest())

	held[ebiten.KeyEscape] = true
	parent.CancelActionApproval()
	parent.Update()
	require.False(t, parent.IsDismissed(), "the Escape that closed approval cannot bleed into the parent")
	held[ebiten.KeyEscape] = false
	parent.Update()
	require.False(t, parent.IsDismissed())
}

func TestInnActionDialogHeldEnterWaitsForRelease(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyEnter: true}
	keyboard := newWidgetTestKeyboard(0)
	dialog := NewInnActionDialog(InnActionDialogConfig{
		Keyboard: keyboard,
		Intent:   testInnRestIntent(t, testInnShopState(), false),
		IsKeyPressed: func(key ebiten.Key) bool {
			return held[key]
		},
	})

	dialog.Update()
	require.False(t, dialog.IsDismissed(), "held Enter cannot instantly confirm a newly opened approval")
	require.False(t, dialog.HasPendingRequest())
	held[ebiten.KeyEnter] = false
	dialog.Update()
	require.False(t, dialog.IsDismissed())
	pressInnActionKey(dialog, keyboard, ebiten.KeyEnter)
	require.True(t, dialog.IsDismissed())
	require.True(t, dialog.HasPendingRequest())
}

func TestInnActionDialogSemanticBackgroundFooterColorsAndNoMouseAssumptions(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	dialog := NewInnActionDialog(InnActionDialogConfig{
		Keyboard: newWidgetTestKeyboard(0),
		Intent:   testInnRestIntent(t, testInnShopState(), true),
		IsKeyPressed: func(ebiten.Key) bool {
			return false
		},
	})
	guidance := dialog.GetFooterGuidance()
	visible := strings.ToLower(dialog.GetTitleText() + " " + dialog.GetBodyText() + " " + guidance)
	require.NotContains(t, visible, "mouse")
	require.NotContains(t, visible, "click")
	require.NotContains(t, visible, "tab")
	requireKeyedGuidance(t, guidance)

	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Dispose)
	dialog.Draw(screen)
	requireImageRegionContainsColor(t, screen, dialog.GetBodyBounds(), MerchantDialogBackgroundColor)
	requireImageRegionContainsColor(t, screen, dialog.frame.footerText, DialogControlKeyColor)
	requireImageRegionContainsColor(t, screen, dialog.frame.footerText, DialogControlHintColor)
	require.Equal(t, FontSizeBody, dialog.GetFontSize())
	require.True(t, dialog.IsVisible())
	require.False(t, dialog.IsComplete())
	dialog.ShowWithoutAnimation()
	require.True(t, dialog.IsVisible())
}

func TestInnActionDialogStrictTypedContracts(t *testing.T) {
	valid := InnActionDialogConfig{
		Keyboard:     newWidgetTestKeyboard(0),
		Intent:       testInnRestIntent(t, testInnShopState(), false),
		IsKeyPressed: func(ebiten.Key) bool { return false },
	}
	tests := []struct {
		name   string
		mutate func(*InnActionDialogConfig)
	}{
		{name: "keyboard", mutate: func(config *InnActionDialogConfig) { config.Keyboard = nil }},
		{name: "action", mutate: func(config *InnActionDialogConfig) { config.Intent.Request.Action = InnActionLeave }},
		{name: "charge", mutate: func(config *InnActionDialogConfig) { config.Intent.Request.Rest.ReviewedCharge++ }},
		{name: "blocked rest", mutate: func(config *InnActionDialogConfig) {
			config.Intent.Rest.BlockReason = game_state.InnRestBlockNoVacancy
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			config.Intent = cloneInnActionIntent(valid.Intent)
			test.mutate(&config)
			require.Panics(t, func() { NewInnActionDialog(config) })
		})
	}
}

func testInnRestIntent(
	t *testing.T,
	state game_state.InnShopState,
	preview bool,
) InnActionIntent {
	t.Helper()
	keyboard := newWidgetTestKeyboard(0)
	config := testInnShopDialogConfig(keyboard, state)
	config.Preview = preview
	parent := NewInnShopDialog(config)
	pressInnShopKey(parent, keyboard, ebiten.KeyEnter)
	intent, ok := parent.ConsumePendingActionIntent()
	require.True(t, ok)

	return intent
}

func testInnLeaveIntent(
	t *testing.T,
	state game_state.InnShopState,
	row int,
	preview bool,
) InnActionIntent {
	t.Helper()
	keyboard := newWidgetTestKeyboard(0)
	config := testInnShopDialogConfig(keyboard, state)
	config.Preview = preview
	parent := NewInnShopDialog(config)
	pressInnShopKey(parent, keyboard, ebiten.KeyL)
	for range row {
		pressInnShopKey(parent, keyboard, ebiten.KeyDown)
	}
	pressInnShopKey(parent, keyboard, ebiten.KeyEnter)
	intent, ok := parent.ConsumePendingActionIntent()
	require.True(t, ok)

	return intent
}

func testInnPickupIntent(
	t *testing.T,
	state game_state.InnShopState,
	row int,
	preview bool,
) InnActionIntent {
	t.Helper()
	keyboard := newWidgetTestKeyboard(0)
	config := testInnShopDialogConfig(keyboard, state)
	config.Preview = preview
	parent := NewInnShopDialog(config)
	pressInnShopKey(parent, keyboard, ebiten.KeyP)
	for range row {
		pressInnShopKey(parent, keyboard, ebiten.KeyDown)
	}
	pressInnShopKey(parent, keyboard, ebiten.KeyEnter)
	intent, ok := parent.ConsumePendingActionIntent()
	require.True(t, ok)

	return intent
}

func pressInnActionKey(dialog *InnActionDialog, keyboard *input.Keyboard, key ebiten.Key) {
	keyboard.SimulateKeyPress(key)
	dialog.Update()
}
