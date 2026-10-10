package widgets

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	pkgcolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestShipShopDialogBoundedNavigationCardsAndManifest(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	keyboard := newWidgetTestKeyboard(0)
	dialog := newTestShipShopDialog(keyboard, testShipShopState())

	require.Equal(t, ShipShopBrowse, dialog.GetPhase())
	require.Equal(t, 0, dialog.GetSelectedIndex())
	require.Equal(t, references.FrigateVehicle, dialog.GetSelectedVehicle())
	pressShipShopKey(dialog, keyboard, ebiten.KeyLeft)
	require.Equal(t, 0, dialog.GetSelectedIndex())
	pressShipShopKey(dialog, keyboard, ebiten.KeyRight)
	require.Equal(t, 1, dialog.GetSelectedIndex())
	pressShipShopKey(dialog, keyboard, ebiten.KeyRight)
	require.Equal(t, 1, dialog.GetSelectedIndex())
	pressShipShopKey(dialog, keyboard, ebiten.KeyLeft)
	require.Equal(t, 0, dialog.GetSelectedIndex())

	frigateCard := compactShipText(dialog.GetOfferCardText(0))
	skiffCard := compactShipText(dialog.GetOfferCardText(1))
	require.Equal(t, "Frigate Includes 2 skiffs 600 gold", frigateCard)
	require.Equal(t, "Skiff Harbour tender 200 gold", skiffCard)
	manifest := compactShipText(dialog.GetManifestText())
	require.Contains(t, manifest, "Harbour: Jhelom Harbour")
	require.Contains(t, manifest, "Berth: Empty")
	require.Contains(t, manifest, "Vessel: None")
	require.Contains(t, manifest, "Action: Dispatch new frigate")
	require.Contains(t, manifest, "Includes 2 skiffs; arrival at Jhelom Harbour.")
	require.Contains(t, manifest, "Escort unavailable - 3 minutes")
	require.NotContains(t, manifest, "Skiffs aboard: 0/12")

	pressShipShopKey(dialog, keyboard, ebiten.KeyRight)
	manifest = compactShipText(dialog.GetManifestText())
	require.Contains(t, manifest, "Action: Dispatch standalone skiff")
	require.Contains(t, manifest, "One standalone skiff will arrive at this harbour.")
	require.Equal(t, FontSizeMainScreen, dialog.GetFontSize())
	require.Equal(t, "[Left/Right] Select   [Enter] Review   [Esc] Leave",
		dialog.GetShell().GetFooterView().Guidance)

	visible := strings.ToLower(strings.Join([]string{
		frigateCard,
		skiffCard,
		manifest,
		dialog.GetLatestExchange(),
		dialog.GetShell().GetFooterView().Guidance,
	}, " "))
	for _, forbidden := range shipDialogForbiddenDetails() {
		require.NotContains(t, visible, forbidden)
	}
}

func TestShipShopDialogDirectionalInputWaitsForRelease(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	heldRight := true
	config := testShipShopDialogConfig(keyboard, testShipShopState())
	config.IsKeyPressed = func(key ebiten.Key) bool {
		return key == ebiten.KeyRight && heldRight
	}
	dialog := NewShipShopDialog(config)

	keyboard.SimulateKeyPress(ebiten.KeyRight)
	dialog.Update()
	require.Equal(t, 0, dialog.GetSelectedIndex(), "held transition direction must not bleed into Browse")
	heldRight = false
	dialog.Update()
	require.Equal(t, 1, dialog.GetSelectedIndex(), "selection begins only after the held direction releases")
}

func TestShipShopDialogPendingFrigateManifestAndSemanticBacking(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	state := testPendingFrigateShipShopState(true)
	dialog := newTestShipShopDialog(newWidgetTestKeyboard(0), state)
	workspace := dialog.GetShell().GetWorkspaceBounds()
	merchantPane := dialog.GetShell().GetMerchantPaneBounds()
	usableBodyWidth := merchantPane.Dx() + workspace.Dx()

	require.InDelta(t, merchantShellMerchantPaneRatio,
		float64(merchantPane.Dx())/float64(usableBodyWidth), 0.01)
	require.True(t, dialog.GetShowroomBounds().In(workspace))
	require.True(t, dialog.GetManifestBounds().In(workspace))
	require.True(t, dialog.GetOfferCardBounds(0).In(dialog.GetShowroomBounds()))
	require.True(t, dialog.GetOfferCardBounds(1).In(dialog.GetShowroomBounds()))
	manifest := compactShipText(dialog.GetManifestText())
	require.Contains(t, manifest, "Berth: Pending arrival")
	require.Contains(t, manifest, "Vessel: Frigate")
	require.Contains(t, manifest, "Skiffs aboard: 2/12")
	require.Contains(t, manifest, "[D] Escort to vessel - 3 minutes")
	require.Contains(t, dialog.GetShell().GetFooterView().Guidance, "[D] Escort")
	require.Equal(t, "Frigate AT BERTH [D] Escort", compactShipText(dialog.GetOfferCardText(0)))
	require.NotContains(t, dialog.GetOfferCardText(0), "gold")
	require.Contains(t, compactShipText(dialog.GetOfferCardText(1)), "200 gold each")

	screen := ebiten.NewImage(1280, 720)
	dialog.Draw(screen)
	requireImageRegionContainsColor(t, screen, dialog.GetManifestBounds(), MerchantDialogBackgroundColor)
	requireImageRegionContainsColor(t, screen, dialog.GetOfferCardBounds(0), pkgcolor.EGABlue)
	requireImageRegionContainsColor(t, screen, dialog.cardSpriteBounds[0], pkgcolor.EGALightMagenta)
	requireImageRegionContainsColor(t, screen, dialog.cardSpriteBounds[1], pkgcolor.EGALightCyan)
}

func TestShipShopDialogManifestNamesEveryBerthStateWithoutInternals(t *testing.T) {
	tests := []struct {
		name       string
		state      game_state.ShipShopState
		berth      string
		vessel     string
		skiffCount string
	}{
		{name: "empty", state: testShipShopState(), berth: "Empty", vessel: "None", skiffCount: ""},
		{name: "pending", state: testPendingFrigateShipShopState(true), berth: "Pending arrival", vessel: "Frigate", skiffCount: "2/12"},
		{name: "moored", state: testMooredFrigateShipShopState(), berth: "Moored", vessel: "Frigate", skiffCount: "2/12"},
		{name: "obstructed", state: testObstructedShipShopState(), berth: "Obstructed", vessel: "Unrelated obstruction", skiffCount: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dialog := newTestShipShopDialog(newWidgetTestKeyboard(0), test.state)
			manifest := compactShipText(dialog.GetManifestText())
			require.Contains(t, manifest, "Berth: "+test.berth)
			require.Contains(t, manifest, "Vessel: "+test.vessel)
			if test.skiffCount != "" {
				require.Contains(t, manifest, "Skiffs aboard: "+test.skiffCount)
			}
			for _, forbidden := range shipDialogForbiddenDetails() {
				require.NotContains(t, strings.ToLower(manifest), forbidden)
			}
		})
	}
}

func TestShipShopDialogBlockedOffersAcknowledgeWithoutPurchaseIntent(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	dialog := newTestShipShopDialog(keyboard, testPendingSkiffShipShopState())

	pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
	require.Equal(t, ShipShopNonterminalRefusal, dialog.GetPhase())
	require.Equal(t, "\"That berth cannot take this order just now.\"", dialog.GetLatestExchange())
	require.Contains(t, dialog.GetReceiptText(), "berth is obstructed")
	require.False(t, dialog.HasPendingPurchaseIntent())
	_, emitted := dialog.ConsumePendingPurchaseIntent()
	require.False(t, emitted)
	require.False(t, dialog.IsDismissed())

	pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
	require.Equal(t, ShipShopBrowse, dialog.GetPhase())
	require.Empty(t, dialog.GetReceiptText())
	require.False(t, dialog.IsDismissed())

	pressShipShopKey(dialog, keyboard, ebiten.KeyRight)
	pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
	require.Equal(t, ShipShopNonterminalRefusal, dialog.GetPhase())
	require.Contains(t, dialog.GetReceiptText(), "standalone skiff already holds this berth")
	require.False(t, dialog.HasPendingPurchaseIntent())
	pressShipShopKey(dialog, keyboard, ebiten.KeyEscape)
	require.Equal(t, ShipShopBrowse, dialog.GetPhase())
}

func TestShipShopDialogUnaffordableApprovalFundsTerminalAndExactDismissal(t *testing.T) {
	state := testShipShopState()
	setShipShopGold(&state, 100)
	keyboard := newWidgetTestKeyboard(0)
	dialog := newTestShipShopDialog(keyboard, state)

	pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
	require.Equal(t, ShipShopAwaitingPurchaseApproval, dialog.GetPhase())
	intent, ok := dialog.ConsumePendingPurchaseIntent()
	require.True(t, ok)
	require.False(t, intent.Quote.Affordable)
	require.Equal(t, shipPurchaseRequestFromState(state, state.FrigateQuote), intent.Request)
	require.False(t, intent.Preview)
	pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
	_, duplicate := dialog.ConsumePendingPurchaseIntent()
	require.False(t, duplicate)

	dialog.CancelPurchaseApproval()
	require.Equal(t, ShipShopBrowse, dialog.GetPhase())
	pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
	intent, ok = dialog.ConsumePendingPurchaseIntent()
	require.True(t, ok)
	dialog.AcceptPurchaseApproval(intent.Request)
	require.Equal(t, ShipShopAwaitingPurchaseResult, dialog.GetPhase())
	require.True(t, dialog.IsAwaitingResult())
	dialog.ApplyFundsRefusal(ShipShopResultDisplay{
		MerchantExchange: "\"Thou hast not the gold for that vessel.\"",
		Receipt:          "Need 600 gold; party has 100.",
	})
	require.Equal(t, ShipShopFundsRefusal, dialog.GetPhase())
	require.False(t, dialog.IsDismissed())
	pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
	require.True(t, dialog.IsDismissed())
	pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
	require.True(t, dialog.IsDismissed())
}

func TestShipShopDialogRefreshAndPurchaseResultPreserveVehicleSelection(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	state := testShipShopState()
	dialog := newTestShipShopDialog(keyboard, state)
	pressShipShopKey(dialog, keyboard, ebiten.KeyRight)

	refreshed := testPendingFrigateShipShopState(true)
	dialog.RefreshBrowse(refreshed)
	require.Equal(t, references.SkiffVehicle, dialog.GetSelectedVehicle())
	require.Equal(t, game_state.ShipPurchaseLoadSkiff, dialog.GetSelectedQuote().Action)
	require.Contains(t, compactShipText(dialog.GetManifestText()), "Load one or more skiffs aboard the current 2/12 frigate")

	dialog = newTestShipShopDialog(keyboard, state)
	pressShipShopKey(dialog, keyboard, ebiten.KeyRight)
	pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
	intent, ok := dialog.ConsumePendingPurchaseIntent()
	require.True(t, ok)
	dialog.AcceptPurchaseApproval(intent.Request)
	resultState := testPendingSkiffShipShopState()
	setShipShopGold(&resultState, 800)
	dialog.ApplyPurchaseResult(successfulShipPurchaseResult(intent, resultState.Gold), resultState,
		ShipShopResultDisplay{
			MerchantExchange: "\"Thy skiff is dispatched.\"",
			Receipt:          "Purchased one skiff for 200 gold.",
		})

	require.Equal(t, ShipShopSuccessReceipt, dialog.GetPhase())
	require.Equal(t, references.SkiffVehicle, dialog.GetSelectedVehicle())
	require.Equal(t, "800", dialog.GetShell().GetHeaderView().PartyGold)
	require.Contains(t, dialog.GetManifestText(), "Purchased one skiff")
	dialog.ContinueShoppingAfterSuccess()
	require.Equal(t, ShipShopBrowse, dialog.GetPhase())
	require.Equal(t, references.SkiffVehicle, dialog.GetSelectedVehicle())
}

func TestShipShopDialogPersistentTravelApprovalRefusalAndCompletion(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	dialog := newTestShipShopDialog(keyboard, testPendingFrigateShipShopState(true))

	pressShipShopKey(dialog, keyboard, ebiten.KeyD)
	require.Equal(t, ShipShopAwaitingTravelApproval, dialog.GetPhase())
	intent, ok := dialog.ConsumePendingTravelIntent()
	require.True(t, ok)
	require.Equal(t, dialog.GetState().Travel, intent.Review)
	dialog.CancelTravelApproval()
	require.Equal(t, ShipShopBrowse, dialog.GetPhase())

	pressShipShopKey(dialog, keyboard, ebiten.KeyD)
	intent, ok = dialog.ConsumePendingTravelIntent()
	require.True(t, ok)
	dialog.AcceptTravelApproval(intent.Review)
	require.Equal(t, ShipShopAwaitingTravelResult, dialog.GetPhase())
	dialog.ApplyTravelRefusal(testPendingFrigateShipShopState(true), ShipShopResultDisplay{
		MerchantExchange: "\"The harbour route changed.\"",
		Receipt:          "Escort unavailable; no travel occurred.",
	})
	require.Equal(t, ShipShopNonterminalRefusal, dialog.GetPhase())
	pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
	require.Equal(t, ShipShopBrowse, dialog.GetPhase())

	pressShipShopKey(dialog, keyboard, ebiten.KeyD)
	intent, ok = dialog.ConsumePendingTravelIntent()
	require.True(t, ok)
	dialog.AcceptTravelApproval(intent.Review)
	dialog.ApplyTravelCompleted(intent.Review)
	require.True(t, dialog.IsDismissed())
}

func TestShipShopDialogDispatchTravelPreviewFarewellAndSessionError(t *testing.T) {
	t.Run("dispatch travel", func(t *testing.T) {
		keyboard := newWidgetTestKeyboard(0)
		state := testShipShopState()
		dialog := newTestShipShopDialog(keyboard, state)
		pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
		intent, ok := dialog.ConsumePendingPurchaseIntent()
		require.True(t, ok)
		dialog.AcceptPurchaseApproval(intent.Request)
		refreshed := testPendingFrigateShipShopState(true)
		setShipShopGold(&refreshed, 400)
		dialog.ApplyPurchaseResult(successfulShipPurchaseResult(intent, refreshed.Gold), refreshed,
			ShipShopResultDisplay{MerchantExchange: "Dispatched.", Receipt: "Frigate purchased."})
		dialog.AcceptDispatchTravel(refreshed.Travel)
		require.Equal(t, ShipShopAwaitingTravelResult, dialog.GetPhase())
		dialog.ApplyTravelCompleted(refreshed.Travel)
		require.True(t, dialog.IsDismissed())
	})

	t.Run("preview", func(t *testing.T) {
		keyboard := newWidgetTestKeyboard(0)
		config := testShipShopDialogConfig(keyboard, testShipShopState())
		config.Preview = true
		dialog := NewShipShopDialog(config)
		require.True(t, dialog.IsPreview())
		require.Contains(t, dialog.GetManifestText(), "PREVIEW")
		pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
		intent, ok := dialog.ConsumePendingPurchaseIntent()
		require.True(t, ok)
		require.True(t, intent.Preview)
		dialog.AcceptPurchaseApproval(intent.Request)
		dialog.ApplyPreviewStopped(intent.Request)
		require.Equal(t, ShipShopPreviewStopped, dialog.GetPhase())
		require.Contains(t, dialog.GetReceiptText(), "gold is unchanged")
		require.Equal(t, uint16(1000), dialog.GetState().Gold)
		pressShipShopKey(dialog, keyboard, ebiten.KeyEscape)
		require.True(t, dialog.IsDismissed())
	})

	t.Run("travel preview", func(t *testing.T) {
		keyboard := newWidgetTestKeyboard(0)
		config := testShipShopDialogConfig(keyboard, testPendingFrigateShipShopState(true))
		config.Preview = true
		dialog := NewShipShopDialog(config)
		pressShipShopKey(dialog, keyboard, ebiten.KeyD)
		intent, ok := dialog.ConsumePendingTravelIntent()
		require.True(t, ok)
		require.True(t, intent.Preview)
		dialog.AcceptTravelApproval(intent.Review)
		dialog.ApplyTravelPreviewStopped(intent.Review)
		require.Equal(t, ShipShopPreviewStopped, dialog.GetPhase())
		require.Contains(t, dialog.GetReceiptText(), "remains in town")
		pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
		require.True(t, dialog.IsDismissed())
	})

	t.Run("farewell", func(t *testing.T) {
		keyboard := newWidgetTestKeyboard(0)
		dialog := newTestShipShopDialog(keyboard, testShipShopState())
		pressShipShopKey(dialog, keyboard, ebiten.KeyEscape)
		require.Equal(t, ShipShopFarewell, dialog.GetPhase())
		require.Equal(t, "\"Fair winds, traveler.\"", dialog.GetLatestExchange())
		require.False(t, dialog.IsDismissed())
		pressShipShopKey(dialog, keyboard, ebiten.KeyEscape)
		require.True(t, dialog.IsDismissed())
	})

	t.Run("session error", func(t *testing.T) {
		keyboard := newWidgetTestKeyboard(0)
		dialog := newTestShipShopDialog(keyboard, testShipShopState())
		pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
		dialog.ApplySessionError("The shipwright ledger failed.")
		require.Equal(t, ShipShopSessionError, dialog.GetPhase())
		require.False(t, dialog.HasPendingPurchaseIntent())
		pressShipShopKey(dialog, keyboard, ebiten.KeyEnter)
		require.True(t, dialog.IsDismissed())
	})
}

func TestShipShopDialogStrictDetachedContracts(t *testing.T) {
	valid := testShipShopDialogConfig(newWidgetTestKeyboard(0), testShipShopState())
	tests := []struct {
		name   string
		mutate func(*ShipShopDialogConfig)
	}{
		{name: "keyboard", mutate: func(config *ShipShopDialogConfig) { config.Keyboard = nil }},
		{name: "merchant sprite", mutate: func(config *ShipShopDialogConfig) { config.MerchantSprite = nil }},
		{name: "frigate sprite", mutate: func(config *ShipShopDialogConfig) { config.FrigateSprite = nil }},
		{name: "skiff sprite", mutate: func(config *ShipShopDialogConfig) { config.SkiffSprite = nil }},
		{name: "family", mutate: func(config *ShipShopDialogConfig) {
			config.State.Merchant.Family = references.MerchantFamilyHorse
		}},
		{name: "quote berth", mutate: func(config *ShipShopDialogConfig) {
			config.State.SkiffQuote.Berth.State = game_state.ShipBerthObstructed
		}},
		{name: "travel merchant", mutate: func(config *ShipShopDialogConfig) {
			config.State.Travel.MerchantID = references.MerchantShipMinoc
		}},
		{name: "prose", mutate: func(config *ShipShopDialogConfig) { config.Prose = ShipShopDialogProse{} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.mutate(&config)
			require.Panics(t, func() { NewShipShopDialog(config) })
		})
	}
}

func testShipShopState() game_state.ShipShopState {
	const gold uint16 = 1000
	berth := game_state.ShipBerthReview{
		Location:       references.Jhelom,
		Position:       references.Position{X: 39, Y: 221},
		State:          game_state.ShipBerthEmpty,
		VehicleType:    references.NoPartyVehicle,
		Pending:        false,
		SkiffCount:     0,
		MapUnitCount:   8,
		HasMapCapacity: true,
	}
	state := game_state.ShipShopState{
		Merchant: references.MerchantDefinition{
			ID:         references.MerchantShipJhelom,
			Family:     references.MerchantFamilyShip,
			Location:   references.Jhelom,
			NPCIndex:   5,
			CatalogRow: 0,
			OwnerName:  "Bantral",
			ShopName:   "Island Shipwrights",
		},
		PricingCharacterIndex: 0,
		PricingCharacterName:  "Avatar",
		PricingIntelligence:   25,
		Gold:                  gold,
		Berth:                 berth,
		Travel: game_state.HarbourTravelReview{
			MerchantID:      references.MerchantShipJhelom,
			DockLocation:    references.Jhelom,
			HarbourName:     "Jhelom Harbour",
			BerthPosition:   berth.Position,
			ArrivalPosition: references.Position{},
			VehicleType:     references.NoPartyVehicle,
			PendingDelivery: false,
			TimeCostMinutes: game_state.HarbourTravelTimeCostMinutes,
			BlockReason:     game_state.HarbourTravelBlockNoBerthVehicle,
			Fingerprint:     "travel-empty",
		},
		FrigateQuote: game_state.ShipPurchaseQuote{
			VehicleType:        references.FrigateVehicle,
			Action:             game_state.ShipPurchaseCreateFrigate,
			BasePrice:          600,
			Price:              600,
			GoldBefore:         gold,
			GoldAfterPayment:   400,
			Affordable:         true,
			BlockReason:        game_state.ShipPurchaseBlockNone,
			Berth:              berth,
			RequestFingerprint: "frigate-empty",
		},
		SkiffQuote: game_state.ShipPurchaseQuote{
			VehicleType:        references.SkiffVehicle,
			Action:             game_state.ShipPurchaseCreateStandaloneSkiff,
			BasePrice:          200,
			Price:              200,
			GoldBefore:         gold,
			GoldAfterPayment:   800,
			Affordable:         true,
			BlockReason:        game_state.ShipPurchaseBlockNone,
			Berth:              berth,
			RequestFingerprint: "skiff-empty",
		},
	}
	setTestShipQuoteApprovals(&state.FrigateQuote)
	setTestShipQuoteApprovals(&state.SkiffQuote)

	return state
}

func testPendingFrigateShipShopState(travelAvailable bool) game_state.ShipShopState {
	state := testShipShopState()
	state.Berth.State = game_state.ShipBerthPendingFrigate
	state.Berth.VehicleType = references.FrigateVehicle
	state.Berth.Pending = true
	state.Berth.SkiffCount = shipShopFrigateSkiffs
	state.Travel.VehicleType = references.FrigateVehicle
	state.Travel.PendingDelivery = true
	state.Travel.Fingerprint = "travel-pending-frigate"
	state.Travel.BlockReason = game_state.HarbourTravelBlockNotOnFoot
	if travelAvailable {
		state.Travel.BlockReason = game_state.HarbourTravelBlockNone
		state.Travel.ArrivalPosition = references.Position{X: 39, Y: 220}
	}
	state.FrigateQuote.Berth = state.Berth
	state.FrigateQuote.BlockReason = game_state.ShipPurchaseBlockFrigateAlreadyPresent
	state.FrigateQuote.RequestFingerprint = "frigate-pending-frigate"
	state.SkiffQuote.Berth = state.Berth
	state.SkiffQuote.Action = game_state.ShipPurchaseLoadSkiff
	state.SkiffQuote.RequestFingerprint = "skiff-pending-frigate"
	setTestShipQuoteApprovals(&state.FrigateQuote)
	setTestShipQuoteApprovals(&state.SkiffQuote)

	return state
}

func testPendingSkiffShipShopState() game_state.ShipShopState {
	state := testShipShopState()
	state.Berth.State = game_state.ShipBerthPendingSkiff
	state.Berth.VehicleType = references.SkiffVehicle
	state.Berth.Pending = true
	state.Travel.VehicleType = references.SkiffVehicle
	state.Travel.PendingDelivery = true
	state.Travel.Fingerprint = "travel-pending-skiff"
	state.Travel.BlockReason = game_state.HarbourTravelBlockNotOnFoot
	state.FrigateQuote.Berth = state.Berth
	state.FrigateQuote.BlockReason = game_state.ShipPurchaseBlockBerthOccupied
	state.FrigateQuote.RequestFingerprint = "frigate-pending-skiff"
	state.SkiffQuote.Berth = state.Berth
	state.SkiffQuote.BlockReason = game_state.ShipPurchaseBlockStandaloneSkiff
	state.SkiffQuote.RequestFingerprint = "skiff-pending-skiff"
	setTestShipQuoteApprovals(&state.FrigateQuote)
	setTestShipQuoteApprovals(&state.SkiffQuote)

	return state
}

func testMooredFrigateShipShopState() game_state.ShipShopState {
	state := testPendingFrigateShipShopState(true)
	state.Berth.State = game_state.ShipBerthMooredFrigate
	state.Berth.Pending = false
	state.Travel.PendingDelivery = false
	state.Travel.Fingerprint = "travel-moored-frigate"
	state.FrigateQuote.Berth = state.Berth
	state.FrigateQuote.RequestFingerprint = "frigate-moored-frigate"
	state.SkiffQuote.Berth = state.Berth
	state.SkiffQuote.RequestFingerprint = "skiff-moored-frigate"
	setTestShipQuoteApprovals(&state.FrigateQuote)
	setTestShipQuoteApprovals(&state.SkiffQuote)

	return state
}

func testObstructedShipShopState() game_state.ShipShopState {
	state := testShipShopState()
	state.Berth.State = game_state.ShipBerthObstructed
	state.FrigateQuote.Berth = state.Berth
	state.FrigateQuote.BlockReason = game_state.ShipPurchaseBlockBerthOccupied
	state.FrigateQuote.RequestFingerprint = "frigate-obstructed"
	state.SkiffQuote.Berth = state.Berth
	state.SkiffQuote.BlockReason = game_state.ShipPurchaseBlockBerthOccupied
	state.SkiffQuote.RequestFingerprint = "skiff-obstructed"
	setTestShipQuoteApprovals(&state.FrigateQuote)
	setTestShipQuoteApprovals(&state.SkiffQuote)

	return state
}

func setShipShopGold(state *game_state.ShipShopState, gold uint16) {
	state.Gold = gold
	quotes := []*game_state.ShipPurchaseQuote{&state.FrigateQuote, &state.SkiffQuote}
	for _, quote := range quotes {
		quote.GoldBefore = gold
		quote.GoldAfterPayment = gold
		quote.Affordable = quote.Price <= int(gold)
		if quote.Affordable {
			quote.GoldAfterPayment = gold - uint16(quote.Price) //nolint:gosec // Test quote prices are bounded.
		}
		setTestShipQuoteApprovals(quote)
	}
}

func setTestShipQuoteApprovals(quote *game_state.ShipPurchaseQuote) {
	quote.ApprovalViews = nil
	if quote.BlockReason.IsBlocked() {
		return
	}
	maximum := 1
	if quote.Action == game_state.ShipPurchaseLoadSkiff {
		remaining := game_state.ShipShopMaximumSkiffs - quote.Berth.SkiffCount
		maximum = min(remaining, max(1, int(quote.GoldBefore)/quote.Price))
	}
	for quantity := 1; quantity <= maximum; quantity++ {
		total := quote.Price * quantity
		affordable := total <= int(quote.GoldBefore)
		afterGold := quote.GoldBefore
		if affordable {
			afterGold -= uint16(total) //nolint:gosec // Test totals are bounded by affordability.
		}
		skiffsAfter := quote.Berth.SkiffCount
		switch quote.Action {
		case game_state.ShipPurchaseCreateFrigate:
			skiffsAfter = shipShopFrigateSkiffs
		case game_state.ShipPurchaseCreateStandaloneSkiff:
			skiffsAfter = 0
		case game_state.ShipPurchaseLoadSkiff:
			skiffsAfter += quantity
		}
		quote.ApprovalViews = append(quote.ApprovalViews, game_state.ShipPurchaseApprovalView{
			Quantity:         quantity,
			UnitPrice:        quote.Price,
			TotalPrice:       total,
			GoldBefore:       quote.GoldBefore,
			GoldAfterPayment: afterGold,
			Affordable:       affordable,
			SkiffCountBefore: quote.Berth.SkiffCount,
			SkiffCountAfter:  skiffsAfter,
		})
	}
	if len(quote.ApprovalViews) > 0 {
		quote.Affordable = quote.ApprovalViews[0].Affordable
		quote.GoldAfterPayment = quote.ApprovalViews[0].GoldAfterPayment
	}
}

func testShipShopDialogConfig(keyboard *input.Keyboard, state game_state.ShipShopState) ShipShopDialogConfig {
	merchantSprite := ebiten.NewImage(16, 16)
	merchantSprite.Fill(pkgcolor.EGAYellow)
	frigateSprite := ebiten.NewImage(16, 16)
	frigateSprite.Fill(pkgcolor.EGALightMagenta)
	skiffSprite := ebiten.NewImage(16, 16)
	skiffSprite.Fill(pkgcolor.EGALightCyan)

	return ShipShopDialogConfig{
		Keyboard:       keyboard,
		MerchantSprite: merchantSprite,
		FrigateSprite:  frigateSprite,
		SkiffSprite:    skiffSprite,
		State:          state,
		Prose: ShipShopDialogProse{
			Greeting:               "\"Choose a vessel, traveler.\"",
			Farewell:               "\"Fair winds, traveler.\"",
			BlockedAcknowledgement: "\"That berth cannot take this order just now.\"",
		},
		Preview:      false,
		IsKeyPressed: func(ebiten.Key) bool { return false },
	}
}

func newTestShipShopDialog(keyboard *input.Keyboard, state game_state.ShipShopState) *ShipShopDialog {
	return NewShipShopDialog(testShipShopDialogConfig(keyboard, state))
}

func pressShipShopKey(dialog *ShipShopDialog, keyboard *input.Keyboard, key ebiten.Key) {
	keyboard.SimulateKeyPress(key)
	dialog.Update()
}

func successfulShipPurchaseResult(intent ShipPurchaseIntent, goldAfter uint16) game_state.ShipPurchaseResult {
	return game_state.ShipPurchaseResult{
		MerchantID:            intent.Request.ReviewedMerchantID,
		PricingCharacterIndex: intent.Request.ReviewedPricingCharacterIndex,
		VehicleType:           intent.Request.VehicleType,
		Action:                intent.Request.Action,
		Quantity:              intent.Request.Quantity,
		BasePrice:             intent.Quote.BasePrice,
		UnitPrice:             intent.Request.ReviewedPrice,
		Price:                 intent.Request.ReviewedTotalPrice,
		GoldBefore:            intent.Request.ReviewedGoldBefore,
		GoldAfterPayment:      goldAfter,
		Theft:                 0,
		GoldAfterTheft:        goldAfter,
		DockLocation:          intent.Quote.Berth.Location,
		DockPosition:          intent.Quote.Berth.Position,
		PendingDelivery:       true,
		SkiffCountBefore:      intent.Quote.Berth.SkiffCount,
		SkiffCountAfter:       shipShopFrigateSkiffs,
		HullHP:                99,
		Direction:             references.Direction(0),
	}
}

func shipDialogForbiddenDetails() []string {
	return []string{
		"coordinate", "position", "hull", "sails", "direction", "ownership", "controller",
		"map unit", "fingerprint",
	}
}

func compactShipText(value string) string { return strings.Join(strings.Fields(value), " ") }
