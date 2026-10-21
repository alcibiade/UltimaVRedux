package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/minimap"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestMerchantTesterCatalogReportsCanonicalRowsAndReviewedContext(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.NewMagincia, references.MerchantGuildNewMagincia)
	request := debugging.Request{CommandID: debugging.CommandMerchantTesterCatalog}

	result := fixture.scene.ExecuteDebugCommand(request)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionQuery, result.EffectiveExecutionMode)
	assert.Empty(t, result.Changes)
	var data debugging.MerchantTesterCatalogData
	require.NoError(t, json.Unmarshal(result.Data, &data))
	assert.Equal(t, debugWorldSourceFingerprint(fixture.state), data.SourceFingerprint)
	assert.Equal(t, int(fixture.state.MapState.PlayerLocation.Location), data.CurrentContext.LocationID)
	assert.Equal(t, fixture.state.MapState.PlayerLocation.Location.String(), data.CurrentContext.LocationName)
	assert.Equal(t, int(fixture.state.MapState.PlayerLocation.Floor), data.CurrentContext.Floor)
	assert.Equal(t, int(fixture.state.MapState.PlayerLocation.Position.X), data.CurrentContext.X)
	assert.Equal(t, int(fixture.state.MapState.PlayerLocation.Position.Y), data.CurrentContext.Y)
	require.Len(t, data.Merchants, 46)

	families := make(map[string]bool)
	ids := make(map[string]bool)
	implemented := 0
	armsImplemented := 0
	healersImplemented := 0
	reagentsReady := 0
	horsesReady := 0
	shipsReady := 0
	innsReady := 0
	tavernsReady := 0
	unimplemented := 0
	for _, row := range data.Merchants {
		families[row.FamilyID] = true
		ids[row.MerchantID] = true
		assert.NotEmpty(t, row.TownName)
		assert.NotEmpty(t, row.ShopName)
		assert.NotEmpty(t, row.OwnerName)
		assert.NotEmpty(t, row.FamilyLabel)
		assert.GreaterOrEqual(t, row.DialogCode, int(references.MerchantFamilyArms))
		assert.LessOrEqual(t, row.DialogCode, int(references.MerchantFamilyInn))
		if row.Implemented {
			implemented++
			assert.Contains(t, []string{"arms", "tavern", "reagent", "guild", "healer", "horse", "ship", "inn"}, row.FamilyID)
			assert.Empty(t, row.UnavailableReason)
			assert.True(t, row.PreviewAvailable)
			if row.FamilyID == "arms" {
				armsImplemented++
			}
			if row.FamilyID == "healer" {
				healersImplemented++
			}
			if row.FamilyID == "reagent" && row.PreviewAvailable {
				reagentsReady++
			}
			if row.FamilyID == "horse" && row.PreviewAvailable {
				horsesReady++
			}
			if row.FamilyID == "ship" && row.PreviewAvailable {
				shipsReady++
			}
			if row.FamilyID == "inn" && row.PreviewAvailable {
				innsReady++
			}
			if row.FamilyID == "tavern" && row.PreviewAvailable {
				tavernsReady++
			}
		} else {
			unimplemented++
			assert.NotEmpty(t, row.UnavailableReason)
			assert.False(t, row.PreviewAvailable)
			assert.False(t, row.LiveAvailable)
		}
	}
	assert.Equal(t, map[string]bool{
		"arms": true, "tavern": true, "horse": true, "ship": true,
		"reagent": true, "guild": true, "healer": true, "inn": true,
	}, families)
	assert.Equal(t, 46, implemented)
	assert.Equal(t, 9, armsImplemented)
	assert.Equal(t, 7, healersImplemented)
	assert.Equal(t, 5, reagentsReady)
	assert.Equal(t, 3, horsesReady)
	assert.Equal(t, 4, shipsReady)
	assert.Equal(t, 6, innsReady)
	assert.Equal(t, 9, tavernsReady)
	assert.Zero(t, unimplemented)
	assert.True(t, ids["arms.britain"])
	assert.True(t, ids["guild.paws"])
	assert.True(t, ids["healer.minoc"])
	assert.True(t, ids["horse.trinsic"])
	assert.True(t, ids["horse.north-britanny"])
	assert.True(t, ids["horse.paws"])
	assert.True(t, ids["ship.jhelom"])
	assert.True(t, ids["ship.minoc"])
	assert.True(t, ids["ship.east-britanny"])
	assert.True(t, ids["ship.buccaneers-den"])
	assert.False(t, ids["britain.arms"], "issue examples must not replace repository merchant identities")

	options := fixture.scene.merchantTesterOptionSet()
	assert.Equal(t, debugging.OptionProviderMerchantTesterMerchants, options.Provider)
	require.Len(t, options.Choices, 46)
	assert.Equal(t, "arms.britain", options.Choices[0].Value)
}

func TestMerchantTesterPreviewStopsAtCommitWithoutGameplayMutation(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	fixture.state.PartyState.Inventory.Gold.Set(1000)
	fixture.state.PartyState.Inventory.Equipment.Set(references.Bow, 3)
	session, err := fixture.state.BeginMerchantSession(fixture.npc)
	require.NoError(t, err)
	beforeLocation := fixture.state.MapState.PlayerLocation
	beforeDate := fixture.state.DateTime
	beforeRNGCalls := fixture.state.GetRNG().Calls()
	beforeInventory := fixture.state.PartyState.Inventory.LockerSnapshot()
	beforeSpells := fixture.state.PartyState.Inventory.SpellSnapshot()

	result := fixture.scene.ExecuteDebugCommand(merchantTesterPreviewRequest(references.MerchantArmsBritain))

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.Contains(t, result.Summary, "workflow queued")
	assert.Empty(t, result.Changes)
	dialog := waitForArmsDialog(t, fixture)
	assert.Same(t, dialog, fixture.scene.dialogStack.PeekTopModalDialog())
	assert.Contains(t, dialog.GetShell().GetHeaderView().ShopIdentity, "Iolo's Bows")
	assert.True(t, dialog.IsPreview())
	beforeTradeCallbacks := *fixture.callbacks
	selectArmsEquipment(t, fixture, dialog, widgets.ArmsShopBuy, references.Bow)
	completeArmsTrade(t, fixture, dialog, 2)
	assert.Equal(t, widgets.ArmsShopBrowse, dialog.GetPhase())
	assert.Contains(t, dialog.GetReceiptText(), "gold and equipment are unchanged")
	selectArmsEquipment(t, fixture, dialog, widgets.ArmsShopSell, references.Bow)
	completeArmsTrade(t, fixture, dialog, 2)
	assert.Contains(t, dialog.GetReceiptText(), "gold and equipment are unchanged")
	assert.Equal(t, beforeLocation, fixture.state.MapState.PlayerLocation)
	assert.Equal(t, beforeDate, fixture.state.DateTime)
	assert.Equal(t, beforeRNGCalls, fixture.state.GetRNG().Calls())
	assert.Equal(t, beforeInventory, fixture.state.PartyState.Inventory.LockerSnapshot())
	assert.Equal(t, beforeSpells, fixture.state.PartyState.Inventory.SpellSnapshot())
	assert.Equal(t, beforeTradeCallbacks, *fixture.callbacks)
	_, err = fixture.state.GetArmsShopState(session)
	assert.NoError(t, err, "preview must not issue or supersede a merchant session")

	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Zero(t, fixture.scene.dialogStack.Count())
	assert.Empty(t, fixture.callbacks.Messages)
	assert.Empty(t, fixture.callbacks.SoundEffectsPlayed)
	assert.Empty(t, fixture.callbacks.TimeAdvanced())
	assert.Equal(t, 1, fixture.callbacks.BlockInputCalls)
	assert.Equal(t, 1, fixture.callbacks.UnblockInputCalls)
	fixture.state.EndMerchantSession(session)
}

func TestMerchantTesterGuildPreviewStopsAtCommitWithoutGameplayMutation(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	beforeLocation := fixture.state.MapState.PlayerLocation
	beforeDate := fixture.state.DateTime
	beforeRNGCalls := fixture.state.GetRNG().Calls()
	beforeInventory := fixture.state.PartyState.Inventory.LockerSnapshot()

	result := fixture.scene.ExecuteDebugCommand(merchantTesterPreviewRequest(references.MerchantGuildPaws))
	require.True(t, result.Success)
	dialog := waitForGuildDialog(t, fixture)
	require.True(t, dialog.IsPreview())
	beforeTradeCallbacks := *fixture.callbacks
	completeGuildPurchase(t, fixture, dialog, 1)

	assert.Equal(t, beforeLocation, fixture.state.MapState.PlayerLocation)
	assert.Equal(t, beforeDate, fixture.state.DateTime)
	assert.Equal(t, beforeRNGCalls, fixture.state.GetRNG().Calls())
	assert.Equal(t, beforeInventory, fixture.state.PartyState.Inventory.LockerSnapshot())
	assert.Equal(t, beforeTradeCallbacks, *fixture.callbacks)
	assert.Contains(t, dialog.GetReceiptText(), "unchanged")

	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Empty(t, fixture.callbacks.TimeAdvanced())
}

func TestMerchantTesterHealerPreviewStopsAtCommitWithoutGameplayMutation(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	avatar := fixture.state.PartyState.GetCharacter(0)
	avatar.SetMaxHp(60)
	avatar.SetHp(18)
	avatar.SetStatus(party_state.Poisoned)
	beforeLocation := fixture.state.MapState.PlayerLocation
	beforeDate := fixture.state.DateTime
	beforeRNGCalls := fixture.state.GetRNG().Calls()
	beforeGold := fixture.state.PartyState.Inventory.Gold.Get()
	beforeStatus := avatar.GetStatus()
	beforeHP := avatar.GetCurrentHp()

	result := fixture.scene.ExecuteDebugCommand(merchantTesterPreviewRequest(references.MerchantHealerMinoc))
	require.True(t, result.Success)
	dialog := waitForHealerDialog(t, fixture)
	require.True(t, dialog.IsPreview())
	beforeTradeCallbacks := *fixture.callbacks
	completeHealerService(t, fixture, dialog)

	assert.Equal(t, beforeLocation, fixture.state.MapState.PlayerLocation)
	assert.Equal(t, beforeDate, fixture.state.DateTime)
	assert.Equal(t, beforeRNGCalls, fixture.state.GetRNG().Calls())
	assert.Equal(t, beforeGold, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, beforeStatus, avatar.GetStatus())
	assert.Equal(t, beforeHP, avatar.GetCurrentHp())
	assert.Equal(t, beforeTradeCallbacks, *fixture.callbacks)
	assert.Contains(t, dialog.GetReceiptText(), "no treatment")

	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Empty(t, fixture.callbacks.TimeAdvanced())
}

func TestMerchantTesterReagentPreviewStopsAfterConfirmedRequestWithoutMutation(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	fixture.state.PartyState.Inventory.Gold.Set(500)
	fixture.state.PartyState.Inventory.Reagent.Set(references.BloodMoss, 12)
	session, err := fixture.state.BeginMerchantSession(fixture.npc)
	require.NoError(t, err)
	beforeLocation := fixture.state.MapState.PlayerLocation
	beforeDate := fixture.state.DateTime
	beforeRNGCalls := fixture.state.GetRNG().Calls()
	beforeInventory := fixture.state.PartyState.Inventory.LockerSnapshot()
	beforeSpells := fixture.state.PartyState.Inventory.SpellSnapshot()

	result := fixture.scene.ExecuteDebugCommand(merchantTesterPreviewRequest(references.MerchantReagentCove))
	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.Empty(t, result.Changes)
	shop := waitForReagentDialog(t, fixture)
	assert.True(t, shop.IsPreview())
	assert.Contains(t, shop.GetDetailText(), "PREVIEW")
	selectReagent(t, fixture, shop, references.BloodMoss)
	beforeRequestCallbacks := *fixture.callbacks
	purchase := completeReagentPurchase(t, fixture, shop, 2)

	assert.True(t, purchase.IsPreview())
	assert.Equal(t, beforeLocation, fixture.state.MapState.PlayerLocation)
	assert.Equal(t, beforeDate, fixture.state.DateTime)
	assert.Equal(t, beforeRNGCalls, fixture.state.GetRNG().Calls())
	assert.Equal(t, beforeInventory, fixture.state.PartyState.Inventory.LockerSnapshot())
	assert.Equal(t, beforeSpells, fixture.state.PartyState.Inventory.SpellSnapshot())
	assert.Equal(t, beforeRequestCallbacks, *fixture.callbacks)
	assert.Contains(t, shop.GetReceiptText(), "no purchase was committed")
	_, err = fixture.state.GetArmsShopState(session)
	assert.NoError(t, err, "reagent preview must not issue or supersede a merchant session")

	pressMerchantDialogKey(fixture, ebiten.KeyEscape)
	assert.Equal(t, widgets.ReagentShopFarewell, shop.GetPhase())
	assert.True(t, fixture.state.EventQueueManager.HasActiveWorkflows())
	pressMerchantDialogKey(fixture, ebiten.KeyEnter)
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Empty(t, fixture.callbacks.TimeAdvanced())
	assert.Equal(t, 1, fixture.callbacks.BlockInputCalls)
	assert.Equal(t, 1, fixture.callbacks.UnblockInputCalls)
	fixture.state.EndMerchantSession(session)
}

func TestMerchantTesterHorsePreviewConfirmsDeferredOfferWithoutGameplayEffects(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	fixture.state.PartyState.Inventory.Gold.Set(500)
	session, err := fixture.state.BeginMerchantSession(fixture.npc)
	require.NoError(t, err)
	beforeLocation := fixture.state.MapState.PlayerLocation
	beforeDate := fixture.state.DateTime
	beforeRNGCalls := fixture.state.GetRNG().Calls()
	beforeInventory := fixture.state.PartyState.Inventory.LockerSnapshot()
	beforeUnits := append(map_units.MapUnits(nil), (*fixture.state.CurrentNPCAIController.GetNpcs())...)

	result := fixture.scene.ExecuteDebugCommand(merchantTesterPreviewRequest(references.MerchantHorsePaws))

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.Empty(t, result.Changes)
	dialog := waitForHorseDialog(t, fixture)
	assert.True(t, dialog.IsPreview())
	assert.Equal(t, game_state.HorseShopBlockAvailabilityDeferred, dialog.GetQuote().Placement.BlockStatus)
	assert.Contains(t, strings.Join(strings.Fields(dialog.GetStatusText()), " "), "Availability will be checked in Live.")
	beforePurchaseCallbacks := *fixture.callbacks
	purchaseDialog := completeHorsePurchase(t, fixture, dialog)

	assert.True(t, purchaseDialog.IsPreview())
	assert.Equal(t, widgets.HorseShopPreviewStopped, dialog.GetPhase())
	assert.Contains(t, strings.Join(strings.Fields(dialog.GetStatusText()), " "), "no purchase was committed")
	assert.Equal(t, beforeLocation, fixture.state.MapState.PlayerLocation)
	assert.Equal(t, beforeDate, fixture.state.DateTime)
	assert.Equal(t, beforeRNGCalls, fixture.state.GetRNG().Calls())
	assert.Equal(t, beforeInventory, fixture.state.PartyState.Inventory.LockerSnapshot())
	assert.Equal(t, beforeUnits, *fixture.state.CurrentNPCAIController.GetNpcs())
	assert.Equal(t, beforePurchaseCallbacks, *fixture.callbacks)
	_, err = fixture.state.GetArmsShopState(session)
	assert.NoError(t, err, "horse Preview must not issue, supersede, or end a MerchantSession")

	pressMerchantDialogKey(fixture, ebiten.KeyEnter)
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Zero(t, fixture.scene.dialogStack.Count())
	assert.Empty(t, fixture.callbacks.TimeAdvanced())
	assert.Equal(t, 1, fixture.callbacks.BlockInputCalls)
	assert.Equal(t, 1, fixture.callbacks.UnblockInputCalls)
	fixture.state.EndMerchantSession(session)
}

func TestMerchantTesterShipPreviewConfirmsWithoutGameplayEffects(t *testing.T) {
	fixture := newShipMerchantTalkFixture(t, references.Jhelom, references.MerchantShipJhelom)
	fixture.state.PartyState.Inventory.Gold.Set(2000)
	frigate, err := fixture.state.PlaceVehicleAtBritanniaDock(
		references.FrigateVehicle,
		references.Jhelom,
		true,
	)
	require.NoError(t, err)
	session, err := fixture.state.BeginMerchantSession(fixture.npc)
	require.NoError(t, err)
	beforeSessionState, err := fixture.state.GetShipShopState(session)
	require.NoError(t, err)
	controller := fixture.state.LargeMapNPCAIController[references.OVERWORLD]
	require.NotNil(t, controller)
	units := controller.GetNpcs()
	require.NotNil(t, units)
	beforeUnits := append(map_units.MapUnits(nil), (*units)...)
	beforeLocation := fixture.state.MapState.PlayerLocation
	beforeDate := fixture.state.DateTime
	beforeGold := fixture.state.PartyState.Inventory.Gold.Get()
	beforeRNGCalls := fixture.state.GetRNG().Calls()
	beforeSkiffs := frigate.GetVehicleDetails().GetSkiffQuantity()
	require.True(t, frigate.GetVehicleDetails().IsPendingDelivery())

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterPreviewRequest(references.MerchantShipJhelom),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.Empty(t, result.Changes)
	shop := waitForShipDialog(t, fixture)
	assert.True(t, shop.IsPreview())
	assert.Contains(t, shop.GetManifestText(), "PREVIEW")
	pressMerchantDialogKey(fixture, ebiten.KeyRight)
	require.Equal(t, game_state.ShipPurchaseLoadSkiff, shop.GetSelectedQuote().Action)
	beforePurchaseCallbacks := *fixture.callbacks
	purchase := completeShipPurchase(t, fixture, shop)

	assert.True(t, purchase.IsPreview())
	assert.Equal(t, widgets.ShipShopPreviewStopped, shop.GetPhase())
	assert.Contains(t, shop.GetReceiptText(), "no vessel was purchased")
	assert.Same(t, controller, fixture.state.LargeMapNPCAIController[references.OVERWORLD])
	assert.Equal(t, beforeUnits, *units)
	assert.Same(t, frigate, units.GetVehicleAtPositionAndFloorOrNil(
		beforeSessionState.Berth.Position,
		references.MainFloor,
	))
	assert.True(t, frigate.GetVehicleDetails().IsPendingDelivery())
	assert.Equal(t, beforeSkiffs, frigate.GetVehicleDetails().GetSkiffQuantity())
	assert.Equal(t, beforeLocation, fixture.state.MapState.PlayerLocation)
	assert.Equal(t, beforeDate, fixture.state.DateTime)
	assert.Equal(t, beforeGold, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, beforeRNGCalls, fixture.state.GetRNG().Calls())
	assert.Equal(t, beforePurchaseCallbacks, *fixture.callbacks)
	afterSessionState, err := fixture.state.GetShipShopState(session)
	require.NoError(t, err, "Ship Preview must not issue or supersede a MerchantSession")
	assert.Equal(t, beforeSessionState, afterSessionState)

	pressMerchantDialogKey(fixture, ebiten.KeyEnter)
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Empty(t, fixture.callbacks.TimeAdvanced())
	assert.Zero(t, fixture.callbacks.ScreenCallCount)
	assert.Equal(t, 1, fixture.callbacks.BlockInputCalls)
	assert.Equal(t, 1, fixture.callbacks.UnblockInputCalls)
	_, err = fixture.state.GetShipShopState(session)
	require.NoError(t, err, "Ship Preview cleanup must preserve the active MerchantSession")
	fixture.state.EndMerchantSession(session)
}

func TestMerchantTesterShipPreviewTravelStopsBeforeTransition(t *testing.T) {
	fixture := newShipMerchantTalkFixture(t, references.Minoc, references.MerchantShipMinoc)
	frigate, err := fixture.state.PlaceVehicleAtBritanniaDock(
		references.FrigateVehicle,
		references.Minoc,
		true,
	)
	require.NoError(t, err)
	beforeLocation := fixture.state.MapState.PlayerLocation
	beforeDate := fixture.state.DateTime
	beforeRNGCalls := fixture.state.GetRNG().Calls()

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterPreviewRequest(references.MerchantShipMinoc),
	)
	require.True(t, result.Success)
	shop := waitForShipDialog(t, fixture)
	pressMerchantDialogKey(fixture, ebiten.KeyD)
	travel := waitForShipTravelDialog(t, fixture)
	require.True(t, travel.IsPreview())
	pressMerchantDialogKey(fixture, ebiten.KeyEnter)
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return shop.GetPhase() == widgets.ShipShopPreviewStopped
	})

	assert.Equal(t, beforeLocation, fixture.state.MapState.PlayerLocation)
	assert.Equal(t, beforeDate, fixture.state.DateTime)
	assert.Equal(t, beforeRNGCalls, fixture.state.GetRNG().Calls())
	assert.True(t, frigate.GetVehicleDetails().IsPendingDelivery())
	assert.Empty(t, fixture.callbacks.TimeAdvanced())
	assert.Contains(t, shop.GetReceiptText(), "remains in town")
	pressMerchantDialogKey(fixture, ebiten.KeyEnter)
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
}

func TestMerchantTesterInnPreviewStopsAllActionsWithoutGameplayMutation(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantInnBritain)
	prepareInnTalkRoster(fixture, 2)
	guest := fixture.state.PartyState.GetCharacter(8)
	guest.PartyMap = party_state.PartyMapAtInn(references.Britain)
	guest.MonthsAtInn = 1
	fixture.state.PartyState.Inventory.Gold.Set(1000)
	fixture.state.ActiveDurationEffect = game_state.DurationEffect{
		EffectType: game_state.DurationEffectQuickness, TurnsRemaining: 18,
	}
	session, err := fixture.state.BeginMerchantSession(fixture.npc)
	require.NoError(t, err)
	beforeLocation := fixture.state.MapState.PlayerLocation
	beforeDate := fixture.state.DateTime
	beforeRNG := fixture.state.GetRNG().Calls()
	beforeRoster := fixture.state.CurrentPartyRosterSnapshot()
	beforeGold := fixture.state.PartyState.Inventory.Gold.Get()
	beforeSpell := fixture.state.ActiveDurationEffect

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterPreviewRequest(references.MerchantInnBritain),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.Empty(t, result.Changes)
	dialog := waitForInnDialog(t, fixture)
	require.True(t, dialog.IsPreview())
	require.Contains(t, dialog.GetModeHeaderText(), "PREVIEW")
	beforeActionsCallbacks := *fixture.callbacks

	completeInnAction(t, fixture, dialog)
	require.Equal(t, widgets.InnShopRest, dialog.GetMode())
	require.Contains(t, dialog.GetReceiptText(), "gold, party, rooms, spells, and roster are unchanged")
	pressMerchantDialogKey(fixture, ebiten.KeyL)
	pressMerchantDialogKey(fixture, ebiten.KeyDown)
	completeInnAction(t, fixture, dialog)
	require.Equal(t, widgets.InnShopLeave, dialog.GetMode())
	pressMerchantDialogKey(fixture, ebiten.KeyP)
	completeInnAction(t, fixture, dialog)
	require.Equal(t, widgets.InnShopPickup, dialog.GetMode())

	assert.Equal(t, beforeLocation, fixture.state.MapState.PlayerLocation)
	assert.Equal(t, beforeDate, fixture.state.DateTime)
	assert.Equal(t, beforeRNG, fixture.state.GetRNG().Calls())
	assert.Equal(t, beforeRoster, fixture.state.CurrentPartyRosterSnapshot())
	assert.Equal(t, beforeGold, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, beforeSpell, fixture.state.ActiveDurationEffect)
	assert.Equal(t, beforeActionsCallbacks, *fixture.callbacks)
	_, err = fixture.state.GetInnShopState(session)
	require.NoError(t, err, "Inn Preview must not issue, supersede, or end a MerchantSession")
	assert.False(t, fixture.state.EventQueueManager.HasFeedback("inn_rest_feedback"))
	assert.False(t, fixture.state.EventQueueManager.HasFeedback("inn_pickup_feedback"))

	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Empty(t, fixture.callbacks.TimeAdvanced())
	assert.Equal(t, 1, fixture.callbacks.BlockInputCalls)
	assert.Equal(t, 1, fixture.callbacks.UnblockInputCalls)
	fixture.state.EndMerchantSession(session)
}

func TestMerchantTesterTavernPreviewStopsBeforeAuthorityWithoutGameplayEffects(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	fixture.state.PartyState.Inventory.Gold.Set(500)
	input := newMerchantRuneQueue()
	fixture.scene.appendInputChars = input.Append
	session, err := fixture.state.BeginMerchantSession(fixture.npc)
	require.NoError(t, err)
	beforeLocation := fixture.state.MapState.PlayerLocation
	beforeDate := fixture.state.DateTime
	beforeRNG := fixture.state.GetRNG().Calls()
	beforeGold := fixture.state.PartyState.Inventory.Gold.Get()
	beforeFood := fixture.state.PartyState.Inventory.Provisions.Food.Get()

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterPreviewRequest(references.MerchantTavernBritain),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.Empty(t, result.Changes)
	dialog := waitForTavernDialog(t, fixture)
	require.True(t, dialog.IsPreview())
	require.Contains(t, dialog.GetModeHeaderText(), "PREVIEW")
	require.True(t, dialog.GetState().CustomerEligible, "Preview must expose detached information review")
	beforeCallbacks := *fixture.callbacks
	completeTavernAction(t, fixture, dialog, ebiten.KeyEnter)
	require.Contains(t, dialog.GetReceiptText(), "unchanged")
	pressMerchantDialogKey(fixture, ebiten.KeyC)
	input.Push("honesty")
	fixture.scene.updateDialogs()
	pressMerchantDialogKey(fixture, ebiten.KeyEnter)
	informationApproval := waitForTavernActionDialog(t, fixture)
	require.Equal(t, widgets.TavernActionInformation, informationApproval.GetIntent().Action)
	require.NotEmpty(t, informationApproval.GetIntent().InformationPitch)
	pressMerchantDialogKey(fixture, ebiten.KeyEnter)
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return dialog.GetPhase() == widgets.TavernShopBrowse
	})
	require.Contains(t, dialog.GetReceiptText(), "unchanged")
	assert.Equal(t, beforeLocation, fixture.state.MapState.PlayerLocation)
	assert.Equal(t, beforeDate, fixture.state.DateTime)
	assert.Equal(t, beforeRNG, fixture.state.GetRNG().Calls())
	assert.Equal(t, beforeGold, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, beforeFood, fixture.state.PartyState.Inventory.Provisions.Food.Get())
	assert.Equal(t, beforeCallbacks, *fixture.callbacks)
	assert.Nil(t, fixture.scene.tavernMealTableOverlay)
	_, err = fixture.state.GetArmsShopState(session)
	require.NoError(t, err, "Tavern Preview must not issue or supersede a merchant session")

	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Zero(t, fixture.scene.dialogStack.Count())
	assert.Empty(t, fixture.callbacks.TimeAdvanced())
	assert.Equal(t, 1, fixture.callbacks.BlockInputCalls)
	assert.Equal(t, 1, fixture.callbacks.UnblockInputCalls)
	fixture.state.EndMerchantSession(session)
}

func TestMerchantTesterConcreteFamilyPreviewValidation(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	armsMerchant, found := fixture.state.GameReferences.MerchantReferences.Definition(references.MerchantArmsBritain)
	require.True(t, found)
	guildMerchant, found := fixture.state.GameReferences.MerchantReferences.Definition(references.MerchantGuildPaws)
	require.True(t, found)
	healerMerchant, found := fixture.state.GameReferences.MerchantReferences.Definition(references.MerchantHealerMinoc)
	require.True(t, found)
	reagentMerchant, found := fixture.state.GameReferences.MerchantReferences.Definition(references.MerchantReagentCove)
	require.True(t, found)
	horseMerchant, found := fixture.state.GameReferences.MerchantReferences.Definition(references.MerchantHorsePaws)
	require.True(t, found)
	shipMerchant, found := fixture.state.GameReferences.MerchantReferences.Definition(references.MerchantShipJhelom)
	require.True(t, found)
	innMerchant, found := fixture.state.GameReferences.MerchantReferences.Definition(references.MerchantInnBritain)
	require.True(t, found)
	tavernMerchant, found := fixture.state.GameReferences.MerchantReferences.Definition(references.MerchantTavernBritain)
	require.True(t, found)

	armsReview, err := fixture.scene.prepareArmsMerchantPreview(armsMerchant.ID)
	require.NoError(t, err)
	assert.Equal(t, references.MerchantFamilyArms, armsReview.State.Merchant.Family)
	guildReview, err := fixture.scene.prepareGuildMerchantPreview(guildMerchant.ID)
	require.NoError(t, err)
	assert.Equal(t, references.MerchantFamilyGuild, guildReview.State.Merchant.Family)
	healerReview, err := fixture.scene.prepareHealerMerchantPreview(healerMerchant.ID)
	require.NoError(t, err)
	assert.Equal(t, references.MerchantFamilyHealer, healerReview.State.Merchant.Family)
	reagentReview, err := fixture.scene.prepareReagentMerchantPreview(reagentMerchant.ID)
	require.NoError(t, err)
	assert.Equal(t, references.MerchantFamilyReagent, reagentReview.State.Merchant.Family)
	horseReview, err := fixture.scene.prepareHorseMerchantPreview(horseMerchant.ID)
	require.NoError(t, err)
	assert.Equal(t, references.MerchantFamilyHorse, horseReview.State.Merchant.Family)
	assert.Equal(t, game_state.HorseShopBlockAvailabilityDeferred, horseReview.State.Quote.Placement.BlockStatus)
	beforeShipRNGCalls := fixture.state.GetRNG().Calls()
	shipReview, err := fixture.scene.prepareShipMerchantPreview(shipMerchant.ID)
	require.NoError(t, err)
	assert.Equal(t, references.MerchantFamilyShip, shipReview.State.Merchant.Family)
	assert.Equal(t, beforeShipRNGCalls, fixture.state.GetRNG().Calls())
	innReview, err := fixture.scene.prepareInnMerchantPreview(innMerchant.ID)
	require.NoError(t, err)
	assert.Equal(t, references.MerchantFamilyInn, innReview.State.Merchant.Family)
	tavernReview, err := fixture.scene.prepareTavernMerchantPreview(tavernMerchant.ID)
	require.NoError(t, err)
	assert.Equal(t, references.MerchantFamilyTavern, tavernReview.State.Merchant.Family)
	assert.Equal(t, beforeShipRNGCalls, fixture.state.GetRNG().Calls())
	assert.NoError(t, fixture.scene.validateMerchantTesterPreview(armsMerchant))
	assert.NoError(t, fixture.scene.validateMerchantTesterPreview(guildMerchant))
	assert.NoError(t, fixture.scene.validateMerchantTesterPreview(healerMerchant))
	assert.NoError(t, fixture.scene.validateMerchantTesterPreview(reagentMerchant))
	assert.NoError(t, fixture.scene.validateMerchantTesterPreview(horseMerchant))
	assert.NoError(t, fixture.scene.validateMerchantTesterPreview(shipMerchant))
	assert.NoError(t, fixture.scene.validateMerchantTesterPreview(innMerchant))
	assert.NoError(t, fixture.scene.validateMerchantTesterPreview(tavernMerchant))

	_, err = fixture.scene.prepareArmsMerchantPreview(guildMerchant.ID)
	assert.Error(t, err)
	_, err = fixture.scene.prepareGuildMerchantPreview(armsMerchant.ID)
	assert.Error(t, err)
	_, err = fixture.scene.prepareHealerMerchantPreview(armsMerchant.ID)
	assert.Error(t, err)
	_, err = fixture.scene.prepareReagentMerchantPreview(armsMerchant.ID)
	assert.Error(t, err)
	_, err = fixture.scene.prepareHorseMerchantPreview(armsMerchant.ID)
	assert.Error(t, err)
	_, err = fixture.scene.prepareShipMerchantPreview(armsMerchant.ID)
	assert.Error(t, err)
	_, err = fixture.scene.prepareInnMerchantPreview(armsMerchant.ID)
	assert.Error(t, err)
	_, err = fixture.scene.prepareTavernMerchantPreview(armsMerchant.ID)
	assert.Error(t, err)
	fixture.scene.spriteSheet = nil
	_, err = fixture.scene.prepareArmsMerchantPreview(armsMerchant.ID)
	assert.ErrorIs(t, err, errMerchantTesterSpriteUnavailable)
	_, err = fixture.scene.prepareShipMerchantPreview(shipMerchant.ID)
	assert.ErrorIs(t, err, errMerchantTesterSpriteUnavailable)
	_, err = fixture.scene.prepareInnMerchantPreview(innMerchant.ID)
	assert.ErrorIs(t, err, errMerchantTesterSpriteUnavailable)
	_, err = fixture.scene.prepareTavernMerchantPreview(tavernMerchant.ID)
	assert.ErrorIs(t, err, errMerchantTesterSpriteUnavailable)
}

func TestMerchantTesterSameTownLiveUsesRepeatedAuthorityAndOneTalkMinute(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.NewMagincia, references.MerchantGuildNewMagincia)
	fixture.npc.PlaceAtScheduleSlot(0)
	setMerchantTalkDesiredSlot(t, fixture.state, fixture.npc, 0)
	fixture.npc.NPCReference.Schedule.Ai[0] = byte(references.Talk3)
	fixture.state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
		references.HorseVehicle,
		fixture.state.MapState.PlayerLocation.Position,
		fixture.state.MapState.PlayerLocation.Floor,
	)
	beforeGold := fixture.state.PartyState.Inventory.Gold.Get()
	beforeKeys := fixture.state.PartyState.Inventory.Provisions.Keys.Get()
	request := merchantTesterLiveRequest(fixture.state, references.MerchantGuildNewMagincia)

	result := fixture.scene.ExecuteDebugCommand(request)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.Empty(t, result.Changes)
	dialog := waitForGuildDialog(t, fixture)
	assert.Equal(t, references.AiType(references.Wander3), references.AiType(fixture.npc.NPCReference.Schedule.Ai[0]))
	consumedSlot, consumed := fixture.npc.MapUnitDetails().ConsumedTalk3ScheduleSlot()
	assert.True(t, consumed)
	assert.Equal(t, references.NPCScheduleSlot(0), consumedSlot)
	price := dialog.GetSelectedQuote().PricePerBundle
	completeGuildPurchase(t, fixture, dialog, 1)
	assert.Equal(t, strconv.Itoa(int(beforeGold)-price), dialog.GetShell().GetHeaderView().PartyGold)
	assert.Equal(t, beforeKeys+3, dialog.GetSelectedQuote().QuantityBefore)
	secondPurchase := completeGuildPurchase(t, fixture, dialog, 1)
	assert.Equal(t, beforeKeys+3, secondPurchase.GetQuote().QuantityBefore)
	assert.Equal(t, beforeGold-uint16(price), secondPurchase.GetQuote().Gold)
	assert.Equal(t, beforeGold-uint16(price*2), fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, beforeKeys+6, fixture.state.PartyState.Inventory.Provisions.Keys.Get())

	beforeTalkTurn := fixture.state.DateTime.Turn
	pressMerchantDialogKey(fixture, ebiten.KeyEscape)
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Zero(t, fixture.scene.dialogStack.Count())
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
}

func TestMerchantTesterSameTownArmsLiveUsesProductionDialog(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	fixture.state.PartyState.Inventory.Gold.Set(1000)
	fixture.state.PartyState.Inventory.Equipment.Set(references.Bow, 2)
	request := merchantTesterLiveRequest(fixture.state, references.MerchantArmsBritain)

	result := fixture.scene.ExecuteDebugCommand(request)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.Empty(t, result.Changes)
	dialog := waitForArmsDialog(t, fixture)
	assert.Contains(t, dialog.GetShell().GetHeaderView().ShopIdentity, "Iolo's Bows")
	assert.False(t, dialog.IsPreview())
	selectArmsEquipment(t, fixture, dialog, widgets.ArmsShopBuy, references.Bow)
	buyQuote := armsBuyRow(t, dialog, references.Bow)
	completeArmsTrade(t, fixture, dialog, 2)
	expectedGold := uint16(1000 - buyQuote.Value*2)
	assert.Equal(t, uint16(4), fixture.state.PartyState.Inventory.Equipment.Get(references.Bow))
	assert.Equal(t, expectedGold, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, strconv.Itoa(int(expectedGold)), dialog.GetShell().GetHeaderView().PartyGold)
	assert.Equal(t, uint16(4), armsBuyRow(t, dialog, references.Bow).Quantity)
	selectArmsEquipment(t, fixture, dialog, widgets.ArmsShopSell, references.Bow)
	sellQuote := armsSellRow(t, dialog, references.Bow)
	completeArmsTrade(t, fixture, dialog, 1)
	expectedGold += uint16(sellQuote.Value)
	assert.Equal(t, uint16(3), fixture.state.PartyState.Inventory.Equipment.Get(references.Bow))
	assert.Equal(t, expectedGold, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, strconv.Itoa(int(expectedGold)), dialog.GetShell().GetHeaderView().PartyGold)
	beforeTalkTurn := fixture.state.DateTime.Turn
	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Zero(t, fixture.scene.dialogStack.Count())
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
}

func TestMerchantTesterSameTownHealerLiveUsesProductionAuthority(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Minoc, references.MerchantHealerMinoc)
	avatar := fixture.state.PartyState.GetCharacter(0)
	avatar.SetMaxHp(60)
	avatar.SetHp(18)
	avatar.SetStatus(party_state.Poisoned)
	beforeGold := fixture.state.PartyState.Inventory.Gold.Get()

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterLiveRequest(fixture.state, references.MerchantHealerMinoc),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	dialog := waitForHealerDialog(t, fixture)
	assert.False(t, dialog.IsPreview())
	completeHealerService(t, fixture, dialog)
	assert.Equal(t, party_state.Good, avatar.GetStatus())
	assert.Equal(t, uint16(18), avatar.GetCurrentHp())
	assert.Equal(t, beforeGold, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Contains(t, dialog.GetReceiptText(), "Free care")

	beforeTalkTurn := fixture.state.DateTime.Turn
	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
}

func TestMerchantTesterSameTownReagentLiveCommitsAndOwnsOneMinute(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Yew, references.MerchantReagentYew)
	fixture.state.PartyState.Inventory.Gold.Set(100)
	fixture.state.PartyState.Inventory.Reagent.Set(references.SpiderSilk, 0)

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterLiveRequest(fixture.state, references.MerchantReagentYew),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.Empty(t, result.Changes)
	shop := waitForReagentDialog(t, fixture)
	assert.False(t, shop.IsPreview())
	selectReagent(t, fixture, shop, references.SpiderSilk)
	quote := shop.GetSelectedQuote()
	completeReagentPurchase(t, fixture, shop, 2)
	expectedQuantity := quote.BundleQuantity * 2
	expectedGold := uint16(100 - quote.PricePerBundle*2)
	assert.Equal(t, expectedQuantity, fixture.state.PartyState.Inventory.Reagent.Get(references.SpiderSilk))
	assert.Equal(t, expectedGold, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, expectedQuantity, shop.GetSelectedQuote().QuantityBefore)
	assert.Equal(t, strconv.Itoa(int(expectedGold)), shop.GetShell().GetHeaderView().PartyGold)
	requireReagentDialogMatchesAuthority(t, fixture, shop, references.MerchantReagentYew)

	beforeTalkTurn := fixture.state.DateTime.Turn
	shop.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Zero(t, fixture.scene.dialogStack.Count())
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
}

func TestMerchantTesterSameTownHorseLiveUsesProductionAuthority(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Trinsic, references.MerchantHorseTrinsic)
	setHorseTalkCandidateTiles(t, fixture, [4]indexes.SpriteIndex{indexes.Grass})
	fixture.state.PartyState.Inventory.Gold.Set(500)
	horsesBefore := countHorseTalkVehicles(fixture.state.CurrentNPCAIController.GetNpcs())

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterLiveRequest(fixture.state, references.MerchantHorseTrinsic),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.Empty(t, result.Changes)
	dialog := waitForHorseDialog(t, fixture)
	assert.False(t, dialog.IsPreview())
	price := dialog.GetQuote().Price
	completeHorsePurchase(t, fixture, dialog)
	assert.Equal(t, widgets.HorseShopSuccess, dialog.GetPhase())
	assert.Equal(t, uint16(500-price), fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, horsesBefore+1, countHorseTalkVehicles(fixture.state.CurrentNPCAIController.GetNpcs()))

	beforeTalkTurn := fixture.state.DateTime.Turn
	pressMerchantDialogKey(fixture, ebiten.KeyEnter)
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Zero(t, fixture.scene.dialogStack.Count())
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
}

func TestMerchantTesterSameTownShipLiveUsesProductionAuthority(t *testing.T) {
	fixture := newShipMerchantTalkFixture(t, references.Jhelom, references.MerchantShipJhelom)
	fixture.state.PartyState.Inventory.Gold.Set(2000)
	units := shipTalkOverworldUnits(t, fixture)

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterLiveRequest(fixture.state, references.MerchantShipJhelom),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.Empty(t, result.Changes)
	shop := waitForShipDialog(t, fixture)
	assert.False(t, shop.IsPreview())
	quote := shop.GetSelectedQuote()
	completeShipPurchase(t, fixture, shop)
	dispatch := waitForShipDispatchDialog(t, fixture)
	expectedGold := uint16(2000 - quote.Price)
	assert.Equal(t, expectedGold, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, expectedGold, shop.GetState().Gold)
	assert.Equal(t, strconv.Itoa(int(expectedGold)), shop.GetShell().GetHeaderView().PartyGold)
	assert.Equal(t, game_state.ShipBerthPendingFrigate, shop.GetState().Berth.State)
	vehicle := units.GetVehicleAtPositionAndFloorOrNil(shop.GetState().Berth.Position, references.MainFloor)
	require.NotNil(t, vehicle)
	assert.True(t, vehicle.GetVehicleDetails().IsPendingDelivery())

	beforeTalkTurn := fixture.state.DateTime.Turn
	pressMerchantDialogKey(fixture, ebiten.KeyEnter)
	assert.True(t, dispatch.ShouldContinueShopping())
	shop.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Equal(t, references.Jhelom, fixture.state.MapState.PlayerLocation.Location)
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
	assert.Zero(t, fixture.scene.dialogStack.Count())
}

func TestMerchantTesterSameTownInnLiveUsesProductionAuthority(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantInnBritain)
	prepareInnTalkRoster(fixture, 2)
	fixture.state.PartyState.Inventory.Gold.Set(1000)

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterLiveRequest(fixture.state, references.MerchantInnBritain),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	assert.Empty(t, result.Changes)
	dialog := waitForInnDialog(t, fixture)
	assert.False(t, dialog.IsPreview())
	pressMerchantDialogKey(fixture, ebiten.KeyL)
	pressMerchantDialogKey(fixture, ebiten.KeyDown)
	completeInnAction(t, fixture, dialog)
	assert.Equal(t, widgets.InnShopLeave, dialog.GetMode())
	assert.Equal(t, 1, fixture.state.PartyState.GetActivePartyCount())
	assert.Equal(t, 1, dialog.GetState().RoomOccupancy)
	assert.Contains(t, dialog.GetReceiptText(), "No charge today.")

	beforeTalkTurn := fixture.state.DateTime.Turn
	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
	assert.Zero(t, fixture.scene.dialogStack.Count())
}

func TestMerchantTesterSameTownTavernLiveUsesProductionAuthorityAndOneMinute(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantTavernBritain)
	fixture.state.PartyState.Inventory.Gold.Set(500)
	foodBefore := fixture.state.PartyState.Inventory.Provisions.Food.Get()

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterLiveRequest(fixture.state, references.MerchantTavernBritain),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionWorkflow, result.EffectiveExecutionMode)
	dialog := waitForTavernDialog(t, fixture)
	assert.False(t, dialog.IsPreview())
	diners := dialog.GetState().Meal.DinerCount
	completeTavernAction(t, fixture, dialog, ebiten.KeyEnter)
	assert.Equal(t, fixture.state.PartyState.Inventory.Gold.Get(), dialog.GetState().Gold)
	assert.Equal(t, foodBefore+uint16(diners), dialog.GetState().Food)
	assert.Empty(t, dialog.GetReceiptText(), "meal result belongs to the acknowledged modal")

	beforeTalkTurn := fixture.state.DateTime.Turn
	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
	assert.Zero(t, fixture.scene.dialogStack.Count())
}

func TestMerchantTesterRemoteLiveTransitionsThenQueuesProductionWorkflow(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	before := fixture.state.MapState.PlayerLocation
	beforeRNGCalls := fixture.state.GetRNG().Calls()
	request := merchantTesterLiveRequest(fixture.state, references.MerchantGuildPaws)

	result := fixture.scene.ExecuteDebugCommand(request)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionAlteration, result.EffectiveExecutionMode)
	assert.Equal(t, references.Paws, fixture.state.MapState.PlayerLocation.Location)
	assert.Equal(t, beforeRNGCalls, fixture.state.GetRNG().Calls())
	assert.ElementsMatch(t, []string{"world.locationId", "world.position.x", "world.position.y"}, stateChangePaths(result.Changes))
	assert.NotEqual(t, before, fixture.state.MapState.PlayerLocation)
	var data debugging.MerchantTesterOpenData
	require.NoError(t, json.Unmarshal(result.Data, &data))
	assert.True(t, data.Transitioned)
	assert.True(t, data.WorkflowQueued)
	assert.False(t, data.Preview)

	dialog := waitForGuildDialog(t, fixture)
	beforeGold := fixture.state.PartyState.Inventory.Gold.Get()
	beforeKeys := fixture.state.PartyState.Inventory.Provisions.Keys.Get()
	price := dialog.GetSelectedQuote().PricePerBundle
	completeGuildPurchase(t, fixture, dialog, 1)
	assert.Equal(t, beforeKeys+3, fixture.state.PartyState.Inventory.Provisions.Keys.Get())
	assert.Equal(t, beforeGold-uint16(price), fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, strconv.Itoa(int(beforeGold)-price), dialog.GetShell().GetHeaderView().PartyGold)
	assert.Equal(t, beforeKeys+3, dialog.GetSelectedQuote().QuantityBefore)
	beforeTalkTurn := fixture.state.DateTime.Turn
	pressMerchantDialogKey(fixture, ebiten.KeyEscape)
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Zero(t, fixture.scene.dialogStack.Count())
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
}

func TestMerchantTesterRemoteArmsLiveTransitionsThenQueuesProductionWorkflow(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	fixture.state.ShadowlordState.TownAssignments = [references.ShadowlordCount]references.Location{}
	fixture.state.PartyState.Inventory.Gold.Set(1000)
	setMerchantTesterRemoteFloorTime(t, fixture.state, references.MerchantArmsJhelom)
	before := fixture.state.MapState.PlayerLocation
	beforeRNGCalls := fixture.state.GetRNG().Calls()
	request := merchantTesterLiveRequest(fixture.state, references.MerchantArmsJhelom)

	result := fixture.scene.ExecuteDebugCommand(request)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionAlteration, result.EffectiveExecutionMode)
	assert.Equal(t, references.Jhelom, fixture.state.MapState.PlayerLocation.Location)
	assert.Equal(t, beforeRNGCalls, fixture.state.GetRNG().Calls())
	assert.ElementsMatch(t, []string{"world.locationId", "world.position.x", "world.position.y"}, stateChangePaths(result.Changes))
	assert.NotEqual(t, before, fixture.state.MapState.PlayerLocation)
	dialog := waitForArmsDialog(t, fixture)
	assert.Contains(t, dialog.GetShell().GetHeaderView().ShopIdentity, "Naughty Nomaan's")
	selectArmsEquipment(t, fixture, dialog, widgets.ArmsShopBuy, references.Mace)
	quote := armsBuyRow(t, dialog, references.Mace)
	completeArmsTrade(t, fixture, dialog, 1)
	expectedGold := uint16(1000 - quote.Value)
	assert.Equal(t, uint16(1), fixture.state.PartyState.Inventory.Equipment.Get(references.Mace))
	assert.Equal(t, expectedGold, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, strconv.Itoa(int(expectedGold)), dialog.GetShell().GetHeaderView().PartyGold)
	assert.Equal(t, uint16(1), armsBuyRow(t, dialog, references.Mace).Quantity)
	beforeTalkTurn := fixture.state.DateTime.Turn
	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Zero(t, fixture.scene.dialogStack.Count())
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
}

func TestMerchantTesterRemoteHealerLiveTransitionsThenUsesProductionAuthority(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	avatar := fixture.state.PartyState.GetCharacter(0)
	avatar.SetMaxHp(60)
	avatar.SetHp(18)
	avatar.SetStatus(party_state.Poisoned)
	setMerchantTesterRemoteFloorTime(t, fixture.state, references.MerchantHealerCove)
	beforeGold := fixture.state.PartyState.Inventory.Gold.Get()

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterLiveRequest(fixture.state, references.MerchantHealerCove),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionAlteration, result.EffectiveExecutionMode)
	assert.Equal(t, references.Cove, fixture.state.MapState.PlayerLocation.Location)
	dialog := waitForHealerDialog(t, fixture)
	completeHealerService(t, fixture, dialog)
	assert.Equal(t, party_state.Good, avatar.GetStatus())
	assert.Equal(t, beforeGold-40, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, strconv.Itoa(int(beforeGold-40)), dialog.GetShell().GetHeaderView().PartyGold)

	beforeTalkTurn := fixture.state.DateTime.Turn
	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
}

func TestMerchantTesterRemoteReagentLiveTransitionsCommitsAndOwnsOneMinute(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	fixture.state.PartyState.Inventory.Gold.Set(100)
	fixture.state.PartyState.Inventory.Reagent.Set(references.BloodMoss, 0)
	setMerchantTesterRemoteFloorTime(t, fixture.state, references.MerchantReagentCove)
	before := fixture.state.MapState.PlayerLocation
	beforeRNGCalls := fixture.state.GetRNG().Calls()

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterLiveRequest(fixture.state, references.MerchantReagentCove),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionAlteration, result.EffectiveExecutionMode)
	assert.Equal(t, references.Cove, fixture.state.MapState.PlayerLocation.Location)
	assert.Equal(t, beforeRNGCalls, fixture.state.GetRNG().Calls())
	assert.NotEqual(t, before, fixture.state.MapState.PlayerLocation)
	assert.ElementsMatch(t, []string{"world.locationId", "world.position.x", "world.position.y"}, stateChangePaths(result.Changes))
	shop := waitForReagentDialog(t, fixture)
	selectReagent(t, fixture, shop, references.BloodMoss)
	quote := shop.GetSelectedQuote()
	completeReagentPurchase(t, fixture, shop, 1)
	expectedGold := uint16(100 - quote.PricePerBundle)
	assert.Equal(t, quote.BundleQuantity, fixture.state.PartyState.Inventory.Reagent.Get(references.BloodMoss))
	assert.Equal(t, expectedGold, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, quote.BundleQuantity, shop.GetSelectedQuote().QuantityBefore)
	assert.Equal(t, strconv.Itoa(int(expectedGold)), shop.GetShell().GetHeaderView().PartyGold)
	requireReagentDialogMatchesAuthority(t, fixture, shop, references.MerchantReagentCove)

	beforeTalkTurn := fixture.state.DateTime.Turn
	shop.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Zero(t, fixture.scene.dialogStack.Count())
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
}

func TestMerchantTesterRemoteHorseLiveTransitionsThenUsesProductionAuthority(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	fixture.state.PartyState.Inventory.Gold.Set(500)
	previousSession, err := fixture.state.BeginMerchantSession(fixture.npc)
	require.NoError(t, err)
	setMerchantTesterRemoteFloorTime(t, fixture.state, references.MerchantHorsePaws)
	before := fixture.state.MapState.PlayerLocation
	beforeRNGCalls := fixture.state.GetRNG().Calls()

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterLiveRequest(fixture.state, references.MerchantHorsePaws),
	)
	require.True(t, result.Success)
	setHorseTalkCandidateTiles(t, fixture, [4]indexes.SpriteIndex{indexes.Grass})

	assert.Equal(t, debugging.ExecutionAlteration, result.EffectiveExecutionMode)
	assert.Equal(t, references.Paws, fixture.state.MapState.PlayerLocation.Location)
	assert.Equal(t, beforeRNGCalls, fixture.state.GetRNG().Calls())
	assert.NotEqual(t, before, fixture.state.MapState.PlayerLocation)
	assert.ElementsMatch(t, []string{"world.locationId", "world.position.x", "world.position.y"}, stateChangePaths(result.Changes))
	horsesBefore := countHorseTalkVehicles(fixture.state.CurrentNPCAIController.GetNpcs())
	dialog := waitForHorseDialog(t, fixture)
	assert.True(t, fixture.callbacks.InputBlocked)
	assert.Equal(t, 1, fixture.callbacks.BlockInputCalls)
	assert.Zero(t, fixture.callbacks.UnblockInputCalls)
	price := dialog.GetQuote().Price
	completeHorsePurchase(t, fixture, dialog)
	assert.Equal(t, widgets.HorseShopSuccess, dialog.GetPhase())
	assert.Equal(t, uint16(500-price), fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, horsesBefore+1, countHorseTalkVehicles(fixture.state.CurrentNPCAIController.GetNpcs()))

	beforeTalkTurn := fixture.state.DateTime.Turn
	pressMerchantDialogKey(fixture, ebiten.KeyEnter)
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Zero(t, fixture.scene.dialogStack.Count())
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
	assert.False(t, fixture.callbacks.InputBlocked)
	assert.Equal(t, 1, fixture.callbacks.BlockInputCalls)
	assert.Equal(t, 1, fixture.callbacks.UnblockInputCalls)
	_, err = fixture.state.GetArmsShopState(previousSession)
	assert.ErrorIs(t, err, game_state.ErrInvalidMerchantSession)
}

func TestMerchantTesterRemoteShipLiveTransitionsThenUsesProductionAuthority(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	fixture.state.ShadowlordState.TownAssignments = [references.ShadowlordCount]references.Location{}
	fixture.state.PartyState.Inventory.Gold.Set(2000)
	units := shipTalkOverworldUnits(t, fixture)
	units.ClearAllMapUnits()
	setMerchantTesterRemoteFloorTime(t, fixture.state, references.MerchantShipJhelom)
	before := fixture.state.MapState.PlayerLocation
	beforeRNGCalls := fixture.state.GetRNG().Calls()

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterLiveRequest(fixture.state, references.MerchantShipJhelom),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionAlteration, result.EffectiveExecutionMode)
	assert.Equal(t, references.Jhelom, fixture.state.MapState.PlayerLocation.Location)
	assert.Equal(t, beforeRNGCalls, fixture.state.GetRNG().Calls())
	assert.NotEqual(t, before, fixture.state.MapState.PlayerLocation)
	assert.ElementsMatch(t, []string{"world.locationId", "world.position.x", "world.position.y"}, stateChangePaths(result.Changes))
	shop := waitForShipDialog(t, fixture)
	assert.False(t, shop.IsPreview())
	quote := shop.GetSelectedQuote()
	completeShipPurchase(t, fixture, shop)
	dispatch := waitForShipDispatchDialog(t, fixture)
	expectedGold := uint16(2000 - quote.Price)
	assert.Equal(t, expectedGold, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, expectedGold, shop.GetState().Gold)
	assert.Equal(t, game_state.ShipBerthPendingFrigate, shop.GetState().Berth.State)
	vehicle := units.GetVehicleAtPositionAndFloorOrNil(shop.GetState().Berth.Position, references.MainFloor)
	require.NotNil(t, vehicle)
	assert.True(t, vehicle.GetVehicleDetails().IsPendingDelivery())

	beforeTalkTurn := fixture.state.DateTime.Turn
	pressMerchantDialogKey(fixture, ebiten.KeyEnter)
	assert.True(t, dispatch.ShouldContinueShopping())
	shop.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Equal(t, references.Jhelom, fixture.state.MapState.PlayerLocation.Location)
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
	assert.Zero(t, fixture.scene.dialogStack.Count())
}

func TestMerchantTesterRemoteInnLiveTransitionsThenUsesProductionAuthority(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	prepareInnTalkRoster(fixture, 2)
	guest := fixture.state.PartyState.GetCharacter(8)
	guest.PartyMap = party_state.PartyMapAtInn(references.Paws)
	guest.MonthsAtInn = 1
	fixture.state.PartyState.Inventory.Gold.Set(1000)
	setMerchantTesterRemoteFloorTime(t, fixture.state, references.MerchantInnPaws)
	before := fixture.state.MapState.PlayerLocation
	beforeRNG := fixture.state.GetRNG().Calls()

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterLiveRequest(fixture.state, references.MerchantInnPaws),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionAlteration, result.EffectiveExecutionMode)
	assert.Equal(t, references.Paws, fixture.state.MapState.PlayerLocation.Location)
	assert.Equal(t, beforeRNG, fixture.state.GetRNG().Calls())
	assert.NotEqual(t, before, fixture.state.MapState.PlayerLocation)
	assert.ElementsMatch(t, []string{"world.locationId", "world.position.x", "world.position.y"}, stateChangePaths(result.Changes))
	dialog := waitForInnDialog(t, fixture)
	assert.False(t, dialog.IsPreview())
	pressMerchantDialogKey(fixture, ebiten.KeyP)
	charge := dialog.GetPickupRows()[0].Charge
	completeInnAction(t, fixture, dialog)
	assert.Equal(t, widgets.InnShopPickup, dialog.GetMode())
	assert.Equal(t, 3, fixture.state.PartyState.GetActivePartyCount())
	assert.Equal(t, uint16(1000)-charge, fixture.state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, fixture.state.PartyState.Inventory.Gold.Get(), dialog.GetState().Gold)
	assert.Zero(t, dialog.GetState().RoomOccupancy)

	beforeTalkTurn := fixture.state.DateTime.Turn
	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
	assert.Zero(t, fixture.scene.dialogStack.Count())
}

func TestMerchantTesterRemoteTavernLiveTransitionsThenUsesProductionAuthority(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	fixture.state.PartyState.Inventory.Gold.Set(500)
	setMerchantTesterRemoteFloorTime(t, fixture.state, references.MerchantTavernPaws)
	before := fixture.state.MapState.PlayerLocation
	beforeRNG := fixture.state.GetRNG().Calls()

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterLiveRequest(fixture.state, references.MerchantTavernPaws),
	)

	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionAlteration, result.EffectiveExecutionMode)
	assert.Equal(t, references.Paws, fixture.state.MapState.PlayerLocation.Location)
	assert.Equal(t, beforeRNG, fixture.state.GetRNG().Calls())
	assert.NotEqual(t, before, fixture.state.MapState.PlayerLocation)
	assert.ElementsMatch(t, []string{"world.locationId", "world.position.x", "world.position.y"}, stateChangePaths(result.Changes))
	dialog := waitForTavernDialog(t, fixture)
	assert.False(t, dialog.IsPreview())
	completeTavernAction(t, fixture, dialog, ebiten.KeyEnter)
	assert.Equal(t, fixture.state.PartyState.Inventory.Gold.Get(), dialog.GetState().Gold)
	assert.Equal(t, fixture.state.PartyState.Inventory.Provisions.Food.Get(), dialog.GetState().Food)

	beforeTalkTurn := fixture.state.DateTime.Turn
	dialog.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})
	assert.Equal(t, beforeTalkTurn+1, fixture.state.DateTime.Turn)
	assert.Zero(t, fixture.scene.dialogStack.Count())
}

func TestAppendWorldSetupChangesIncludesSmashedReset(t *testing.T) {
	changes := make([]debugging.StateChange, 0, 1)
	appendWorldSetupChanges(&changes, game_state.WorldSetupResult{SmashedBefore: 3, SmashedAfter: 0})

	require.Equal(t, []string{"effects.smashed"}, stateChangePaths(changes))
}

func TestMerchantTesterRemotePreflightFailureLeavesSceneStateUnchanged(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	merchant, found := fixture.state.GameReferences.MerchantReferences.Definition(references.MerchantGuildPaws)
	require.True(t, found)
	fixture.state.PartyState.SetDeadNpc(merchant.Location, merchant.NPCIndex)
	beforeLocation := fixture.state.MapState.PlayerLocation
	beforeController := fixture.state.CurrentNPCAIController
	beforeMap := fixture.state.GetCurrentLayeredMap()

	result := fixture.scene.ExecuteDebugCommand(
		merchantTesterLiveRequest(fixture.state, references.MerchantGuildPaws),
	)

	assert.Equal(t, "merchant_context_unsupported", result.ErrorCode)
	assert.Equal(t, beforeLocation, fixture.state.MapState.PlayerLocation)
	assert.Same(t, beforeController, fixture.state.CurrentNPCAIController)
	assert.Same(t, beforeMap, fixture.state.GetCurrentLayeredMap())
	assert.False(t, fixture.state.EventQueueManager.HasActiveWorkflows())
	assert.Zero(t, fixture.scene.dialogStack.Count())
}

func TestMerchantTesterRejectsInvalidSelectionsAndStaleReviewBeforeMutation(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	tests := []struct {
		name    string
		request debugging.Request
		code    string
	}{
		{
			name:    "unknown preview",
			request: merchantTesterPreviewRequest(references.MerchantID("guild.unknown")),
			code:    "merchant_unknown",
		},
		{
			name: "implemented Tavern stale review",
			request: func() debugging.Request {
				request := merchantTesterLiveRequest(fixture.state, references.MerchantTavernBritain)
				request.Arguments["expectedSourceFingerprint"] = mustDebugJSON("stale")

				return request
			}(),
			code: "stale_merchant_context",
		},
		{
			name: "stale live review",
			request: func() debugging.Request {
				request := merchantTesterLiveRequest(fixture.state, references.MerchantGuildPaws)
				request.Arguments["expectedX"] = mustDebugJSON(255)

				return request
			}(),
			code: "stale_merchant_context",
		},
		{
			name: "stale source fingerprint",
			request: func() debugging.Request {
				request := merchantTesterLiveRequest(fixture.state, references.MerchantGuildPaws)
				request.Arguments["expectedSourceFingerprint"] = mustDebugJSON("stale")

				return request
			}(),
			code: "stale_merchant_context",
		},
		{
			name: "missing scene generation",
			request: func() debugging.Request {
				request := merchantTesterLiveRequest(fixture.state, references.MerchantGuildPaws)
				request.ExpectedSceneGeneration = 0

				return request
			}(),
			code: "stale_scene",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beforeLocation := fixture.state.MapState.PlayerLocation
			beforeGold := fixture.state.PartyState.Inventory.Gold.Get()
			beforeRNGCalls := fixture.state.GetRNG().Calls()

			result := fixture.scene.ExecuteDebugCommand(test.request)

			assert.Equal(t, test.code, result.ErrorCode)
			assert.Equal(t, beforeLocation, fixture.state.MapState.PlayerLocation)
			assert.Equal(t, beforeGold, fixture.state.PartyState.Inventory.Gold.Get())
			assert.Equal(t, beforeRNGCalls, fixture.state.GetRNG().Calls())
			assert.False(t, fixture.state.EventQueueManager.HasActiveWorkflows())
			assert.Zero(t, fixture.scene.dialogStack.Count())
		})
	}
}

func setMerchantTesterRemoteFloorTime(
	t *testing.T,
	state *game_state.GameState,
	id references.MerchantID,
) {
	t.Helper()
	merchant, found := state.GameReferences.MerchantReferences.Definition(id)
	require.True(t, found)
	npcs := state.GameReferences.NPCReferences.GetNPCReferencesByLocation(merchant.Location)
	require.NotNil(t, npcs)
	require.Greater(t, len(*npcs), merchant.NPCIndex)
	npc := (*npcs)[merchant.NPCIndex]
	for hour := range 24 {
		candidate := state.DateTime
		candidate.Hour = byte(hour)
		slot := npc.Schedule.DesiredSlot(candidate)
		if npc.Schedule.GetIndividualNPCBehaviourBySlot(slot).Floor == 0 {
			state.DateTime = candidate

			return
		}
	}
	t.Fatalf("merchant %q has no floor-zero schedule time", id)
}

func TestMerchantTesterRejectsIncompatibleActiveContexts(t *testing.T) {
	tests := []struct {
		name  string
		code  string
		apply func(*merchantTalkFixture)
	}{
		{
			name: "dialog", code: "dialog_active",
			apply: func(fixture *merchantTalkFixture) {
				fixture.scene.dialogStack.PushModalDialog(&blockingDebugWorkflowDialog{})
			},
		},
		{
			name: "workflow", code: "gameplay_workflow_active",
			apply: func(fixture *merchantTalkFixture) {
				fixture.state.EventQueueManager.Workflow("blocking_merchant_test").
					Priority(event_queue.PriorityNormal).
					Sequential().
					First(func() {}).
					Finally(func() {}).
					Start()
			},
		},
		{
			name: "transient", code: "transient_context_active",
			apply: func(fixture *merchantTalkFixture) {
				fixture.scene.inputBlocked = true
			},
		},
		{
			name: "scripted", code: "scripted_scene_active",
			apply: func(fixture *merchantTalkFixture) {
				fixture.scene.miniMapScene = minimap.NewMiniMapScene()
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
			test.apply(fixture)
			before := fixture.state.MapState.PlayerLocation

			result := fixture.scene.ExecuteDebugCommand(
				merchantTesterPreviewRequest(references.MerchantGuildPaws),
			)

			assert.Equal(t, test.code, result.ErrorCode)
			assert.Equal(t, before, fixture.state.MapState.PlayerLocation)
		})
	}
}

func TestMerchantTesterPreviewFinallyRemovesOnlyOwnedDialogs(t *testing.T) {
	fixture := newMerchantTalkFixture(t, references.Britain, references.MerchantArmsBritain)
	result := fixture.scene.ExecuteDebugCommand(merchantTesterPreviewRequest(references.MerchantGuildPaws))
	require.True(t, result.Success)
	ownedShop := waitForGuildDialog(t, fixture)
	ownedPurchase := openGuildPurchaseDialog(t, fixture, ownedShop)
	decoy := widgets.NewGuildPurchaseDialog(widgets.GuildPurchaseDialogConfig{
		Keyboard: fixture.scene.keyboard,
		Quote:    ownedPurchase.GetQuote(),
		Pitch:    "Decoy preview pitch.",
	})
	fixture.scene.dialogStack.PushModalDialog(decoy)

	ownedShop.Dismiss()
	pumpMerchantWorkflowUntil(t, fixture, func() bool {
		return !fixture.state.EventQueueManager.HasActiveWorkflows()
	})

	assert.True(t, ownedShop.IsDismissed())
	assert.True(t, ownedPurchase.IsDismissed())
	assert.False(t, decoy.IsDismissed())
	assert.Equal(t, 1, fixture.scene.dialogStack.Count())
	assert.Same(t, decoy, fixture.scene.dialogStack.PeekTopModalDialog())
	assert.Empty(t, fixture.callbacks.TimeAdvanced())
}

func merchantTesterPreviewRequest(id references.MerchantID) debugging.Request {
	return debugging.Request{
		CommandID: debugging.CommandMerchantTesterPreviewOpen,
		Arguments: map[string]json.RawMessage{"merchantId": mustDebugJSON(string(id))},
	}
}

func merchantTesterLiveRequest(
	gameState *game_state.GameState,
	id references.MerchantID,
) debugging.Request {
	arguments := reviewedWorldPositionDebugArguments(gameState)
	arguments["merchantId"] = mustDebugJSON(string(id))

	return debugging.Request{
		CommandID:               debugging.CommandMerchantTesterLiveOpen,
		ExpectedSceneGeneration: 1,
		Arguments:               arguments,
	}
}
