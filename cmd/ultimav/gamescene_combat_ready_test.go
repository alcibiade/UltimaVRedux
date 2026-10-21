package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestCombatReadyDialogBindsNonAvatarAndDisablesCharacterNavigation(t *testing.T) {
	scene, _ := newProductionReadyScene(t)
	clearProductionReadyEquipment(scene.gameState)
	scene.gameState.PartyState.Inventory.Equipment.Set(references.Dagger, 1)
	actor := productionReadyPC(scene.gameState, 1)
	require.NotZero(t, actor.EntityIndex)
	scene.gameState.CombatScene.CurrentTurnIdentity = actor
	scene.gameState.CombatScene.CurrentTurnIndex = actor.EntityIndex

	scene.combatMapInputHandler(ebiten.KeyR)
	dialog := waitForProductionReadyDialog(t, scene)
	require.Equal(t, actor.EntityIndex, dialog.GetSelectedCharacterIndex())

	pressProductionReadyKey(scene, dialog, ebiten.KeyLeft)
	assert.Equal(t, actor.EntityIndex, dialog.GetSelectedCharacterIndex())
	pressProductionReadyKey(scene, dialog, ebiten.Key1)
	assert.Equal(t, actor.EntityIndex, dialog.GetSelectedCharacterIndex())
}

func TestControlledMonsterReadyRejectsWithoutActionLoss(t *testing.T) {
	state, callbacks, _ := setupControlledMonsterInputTest(t)
	scene := &GameScene{gameState: state}
	before := snapshotBlockedCombatState(t, state)

	scene.combatMapInputHandler(ebiten.KeyR)

	assert.Contains(t, callbacks.Messages, "Can't!")
	assert.False(t, state.EventQueueManager.HasWorkflow("ready_equipment_workflow"))
	assert.Equal(t, before, snapshotBlockedCombatState(t, state))
}

func TestCombatReadyBodyArmourErrorStaysOpenUntilVictory(t *testing.T) {
	scene, _ := newProductionReadyScene(t)
	clearProductionReadyEquipment(scene.gameState)
	actor := scene.gameState.CombatScene.GetCurrentTurnIdentity()
	character := scene.gameState.PartyState.GetCharacter(actor.EntityIndex)
	character.Strength = 30
	scene.gameState.PartyState.Inventory.Equipment.Set(references.ClothArmour, 1)

	scene.combatMapInputHandler(ebiten.KeyR)
	dialog := waitForProductionReadyDialog(t, scene)
	pressProductionReadyKey(scene, dialog, ebiten.KeyEnter)
	assert.False(t, dialog.IsDismissed())
	assert.Equal(t, references.NoEquipment, character.GetEquippedItem(references.SlotArmour))
	assert.Equal(t, actor, scene.gameState.CombatScene.GetCurrentTurnIdentity())

	scene.gameState.CombatScene.Victory = true
	pressProductionReadyKey(scene, dialog, ebiten.KeyEnter)
	assert.False(t, dialog.IsDismissed(), "normal successful changes retain the modal")
	assert.Equal(t, references.ClothArmour, character.GetEquippedItem(references.SlotArmour))
}

func TestCombatReadyAmmoErrorDoesNotMutateAndRemainsActive(t *testing.T) {
	scene, _ := newProductionReadyScene(t)
	clearProductionReadyEquipment(scene.gameState)
	actor := scene.gameState.CombatScene.GetCurrentTurnIdentity()
	character := scene.gameState.PartyState.GetCharacter(actor.EntityIndex)
	character.Strength = 30
	scene.gameState.PartyState.Inventory.Equipment.Set(references.Crossbow, 1)

	scene.combatMapInputHandler(ebiten.KeyR)
	dialog := waitForProductionReadyDialog(t, scene)
	pressProductionReadyKey(scene, dialog, ebiten.KeyEnter)
	assert.False(t, dialog.IsDismissed())
	assert.Equal(t, references.NoEquipment, character.GetEquippedItem(references.SlotWeapon))
	assert.Equal(t, uint16(1), scene.gameState.PartyState.Inventory.Equipment.Get(references.Crossbow))

	scene.gameState.PartyState.Inventory.Equipment.Set(references.Quarrels, 2)
	pressProductionReadyKey(scene, dialog, ebiten.KeyEnter)
	assert.Equal(t, references.Crossbow, character.GetEquippedItem(references.SlotWeapon))
	assert.Equal(t, uint16(2), scene.gameState.PartyState.Inventory.Equipment.Get(references.Quarrels))
}

func TestCombatReadyMultipleChangesPersistUntilEscapeAndCompleteOnce(t *testing.T) {
	actual, _ := newProductionReadyScene(t)
	expected, _ := newProductionReadyScene(t)
	for _, scene := range []*GameScene{actual, expected} {
		clearProductionReadyEquipment(scene.gameState)
		actor := scene.gameState.CombatScene.GetCurrentTurnIdentity()
		scene.gameState.PartyState.GetCharacter(actor.EntityIndex).Strength = 30
		scene.gameState.PartyState.Inventory.Equipment.Set(references.LeatherHelm, 1)
		scene.gameState.PartyState.Inventory.Equipment.Set(references.Dagger, 1)
	}

	actual.combatMapInputHandler(ebiten.KeyR)
	dialog := waitForProductionReadyDialog(t, actual)
	pressProductionReadyKey(actual, dialog, ebiten.KeyEnter)
	pressProductionReadyKey(actual, dialog, ebiten.KeyDown)
	pressProductionReadyKey(actual, dialog, ebiten.KeyEnter)
	actor := actual.gameState.CombatScene.GetCurrentTurnIdentity()
	character := actual.gameState.PartyState.GetCharacter(actor.EntityIndex)
	assert.Equal(t, references.LeatherHelm, character.GetEquippedItem(references.SlotHelm))
	assert.Equal(t, references.Dagger, character.GetEquippedItem(references.SlotWeapon))
	assert.False(t, dialog.IsDismissed())

	pressProductionReadyKey(actual, dialog, ebiten.KeyEscape)
	drainProductionReadyWorkflow(t, actual)
	applyExpectedProductionReadyChanges(t, expected, references.LeatherHelm, references.Dagger)
	expected.gameState.FinishCombatAction()
	assertProductionReadyGameplayEqual(t, expected.gameState, actual.gameState)
	assert.Zero(t, actual.dialogStack.Count())
	assert.False(t, actual.gameState.EventQueueManager.HasWorkflow("ready_equipment_workflow"))
}

func TestCombatReadyEscapeWithoutChangeCompletesExactlyOnceAfterCleanup(t *testing.T) {
	actual, _ := newProductionReadyScene(t)
	expected, _ := newProductionReadyScene(t)
	for _, scene := range []*GameScene{actual, expected} {
		clearProductionReadyEquipment(scene.gameState)
		scene.gameState.PartyState.Inventory.Equipment.Set(references.Dagger, 1)
	}

	actual.combatMapInputHandler(ebiten.KeyR)
	dialog := waitForProductionReadyDialog(t, actual)
	pressProductionReadyKey(actual, dialog, ebiten.KeyEscape)
	drainProductionReadyWorkflow(t, actual)
	expected.gameState.FinishCombatAction()

	assertProductionReadyGameplayEqual(t, expected.gameState, actual.gameState)
	assert.Zero(t, actual.dialogStack.Count())
	assert.False(t, actual.keyboard.IsWaitingForRelease())
	beforeUpdate := snapshotBlockedCombatState(t, actual.gameState)
	actual.gameState.ElapsedMs += DialogPollingRateMs
	actual.gameState.EventQueueManager.Update()
	afterUpdate := snapshotBlockedCombatState(t, actual.gameState)
	afterUpdate.ElapsedMs = beforeUpdate.ElapsedMs
	assert.Equal(t, beforeUpdate, afterUpdate, "a drained Ready workflow cannot finalize twice")
}

func TestCombatReadyVanishingRingDismissesImmediatelyWithoutRefund(t *testing.T) {
	scene, callbacks := newProductionReadyScene(t)
	clearProductionReadyEquipment(scene.gameState)
	scene.gameState.SetRandomSeed(productionReadyRingSeed(t, true))
	actor := scene.gameState.CombatScene.GetCurrentTurnIdentity()
	character := scene.gameState.PartyState.GetCharacter(actor.EntityIndex)
	scene.gameState.PartyState.Inventory.Equipment.Set(references.RingRegeneration, 1)
	callbacks.Reset()

	scene.combatMapInputHandler(ebiten.KeyR)
	dialog := waitForProductionReadyDialog(t, scene)
	pressProductionReadyKey(scene, dialog, ebiten.KeyEnter)

	assert.True(t, dialog.IsDismissed())
	assert.Equal(t, references.NoEquipment, character.GetEquippedItem(references.SlotRing))
	assert.Zero(t, scene.gameState.PartyState.Inventory.Equipment.Get(references.RingRegeneration))
	assert.Contains(t, strings.Join(callbacks.Messages, ""), "Ring vanishes!")
	drainProductionReadyWorkflow(t, scene)
	assert.Zero(t, scene.dialogStack.Count())
}

func TestCombatReadyUnreadyInvisibilityRingClearsEffect(t *testing.T) {
	scene, _ := newProductionReadyScene(t)
	clearProductionReadyEquipment(scene.gameState)
	actor := scene.gameState.CombatScene.GetCurrentTurnIdentity()
	character := scene.gameState.PartyState.GetCharacter(actor.EntityIndex)
	character.SetEquippedItem(references.SlotRing, references.RingInvisibility)
	require.True(t, scene.gameState.CastCombatSanctLor(actor.EntityIndex))
	callsBefore := scene.gameState.GetRNG().Calls()
	scene.combatMapInputHandler(ebiten.KeyR)
	dialog := waitForProductionReadyDialog(t, scene)
	pressProductionReadyKey(scene, dialog, ebiten.KeyEnter)
	assert.False(t, scene.gameState.CombatScene.IsCombatPCInvisible(actor))
	assert.Equal(t, uint16(1), scene.gameState.PartyState.Inventory.Equipment.Get(references.RingInvisibility))
	assert.Equal(t, callsBefore, scene.gameState.GetRNG().Calls())
	pressProductionReadyKey(scene, dialog, ebiten.KeyEscape)
	drainProductionReadyWorkflow(t, scene)
	assert.False(t, scene.gameState.CombatScene.IsCombatPCInvisible(actor))
}

func TestCombatReadyEmptyInventorySkipsModalAndConsumesOneAction(t *testing.T) {
	actual, callbacks := newProductionReadyScene(t)
	expected, _ := newProductionReadyScene(t)
	clearProductionReadyEquipment(actual.gameState)
	clearProductionReadyEquipment(expected.gameState)

	actual.combatMapInputHandler(ebiten.KeyR)
	expected.gameState.FinishCombatAction()

	assert.Contains(t, callbacks.Messages, "Thou art empty-\nhanded!")
	assert.False(t, actual.gameState.EventQueueManager.HasWorkflow("ready_equipment_workflow"))
	assert.Zero(t, actual.dialogStack.Count())
	assert.Equal(t, snapshotBlockedCombatState(t, expected.gameState), snapshotBlockedCombatState(t, actual.gameState))
}

func TestReadyWorkflowCleansModalAndKeyboardBeforeCompletionCallback(t *testing.T) {
	scene, _ := newProductionReadyScene(t)
	clearProductionReadyEquipment(scene.gameState)
	scene.gameState.PartyState.Inventory.Equipment.Set(references.Dagger, 1)
	request, outcome := scene.gameState.BeginReadyEquipment(true)
	require.Equal(t, game_state.ReadyBeginDialog, outcome)
	completed := 0
	scene.startReadyEquipmentWorkflowWithCompletion(request, func() {
		completed++
		assert.Zero(t, scene.dialogStack.Count())
		assert.False(t, scene.keyboard.IsWaitingForRelease())
	})
	dialog := waitForProductionReadyDialog(t, scene)
	pressProductionReadyKey(scene, dialog, ebiten.KeyEscape)
	drainProductionReadyWorkflow(t, scene)

	assert.Equal(t, 1, completed)
}

func TestCombatReadyConfirmedSwapAndCancelRetainOneCompletionOwner(t *testing.T) {
	actual, _ := newProductionReadyScene(t)
	expected, _ := newProductionReadyScene(t)
	for _, scene := range []*GameScene{actual, expected} {
		clearProductionReadyEquipment(scene.gameState)
		actor := scene.gameState.CombatScene.GetCurrentTurnIdentity()
		character := scene.gameState.PartyState.GetCharacter(actor.EntityIndex)
		character.Strength = 20
		character.SetEquippedItem(references.SlotArmour, references.LeatherArmour)
		character.SetEquippedItem(references.SlotWeapon, references.SilverSword)
		character.SetEquippedItem(references.SlotShield, references.SmallShield)
		scene.gameState.PartyState.Inventory.Equipment.Set(references.Halberd, 1)
	}
	actor := actual.gameState.CombatScene.GetCurrentTurnIdentity()
	character := actual.gameState.PartyState.GetCharacter(actor.EntityIndex)
	actual.combatMapInputHandler(ebiten.KeyR)
	dialog := waitForProductionReadyDialog(t, actual)
	// Sorted available list: Small Shield, Leather Armour, Halberd, Silver Sword.
	pressProductionReadyKey(actual, dialog, ebiten.KeyDown)
	pressProductionReadyKey(actual, dialog, ebiten.KeyDown)
	pressProductionReadyKey(actual, dialog, ebiten.KeyEnter)
	pressProductionReadyKey(actual, dialog, ebiten.KeyEscape)
	require.False(t, dialog.IsDismissed())
	require.Equal(t, references.SilverSword, character.GetEquippedItem(references.SlotWeapon))
	require.Equal(t, actor, actual.gameState.CombatScene.GetCurrentTurnIdentity())
	pressProductionReadyKey(actual, dialog, ebiten.KeyEnter)
	pressProductionReadyKey(actual, dialog, ebiten.KeyEnter)
	require.Equal(t, references.Halberd, character.GetEquippedItem(references.SlotWeapon))
	require.Equal(t, references.NoEquipment, character.GetEquippedItem(references.SlotShield))
	require.Equal(t, actor, actual.gameState.CombatScene.GetCurrentTurnIdentity())
	pressProductionReadyKey(actual, dialog, ebiten.KeyEscape)
	drainProductionReadyWorkflow(t, actual)
	request, _ := expected.gameState.BeginReadyEquipment(true)
	transaction := game_state.ReadyEquipmentTransaction{Request: request, CharacterIndex: actor.EntityIndex,
		ExpectedOccupant: character.GetNameAsString(), Equipment: references.Halberd}
	preview, err := expected.gameState.PreviewReadyEquipment(transaction)
	require.NoError(t, err)
	transaction.Confirmed = &preview
	_, err = expected.gameState.ApplyReadyEquipment(transaction)
	require.NoError(t, err)
	expected.gameState.FinishCombatAction()
	assertProductionReadyGameplayEqual(t, expected.gameState, actual.gameState)
	require.Zero(t, actual.dialogStack.Count())
}

func newProductionReadyScene(t *testing.T) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()
	scene, callbacks := newCombatUseTestScene(t)
	scene.gameState.SystemCallbacks.Flow.Items.ShowReadyDialog = scene.startReadyEquipmentWorkflow
	scene.gameState.BeginCombatTurns(game_state.CombatTurnsPresented)
	scene.gameState.CombatScene.Victory = false
	callbacks.Reset()

	return scene, callbacks
}

func clearProductionReadyEquipment(state *game_state.GameState) {
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		state.PartyState.Inventory.Equipment.Set(equipment, 0)
	}
	for index := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(index)
		if character == nil {
			continue
		}
		for slot := references.SlotHelm; slot <= references.SlotAmulet; slot++ {
			character.SetEquippedItem(slot, references.NoEquipment)
		}
	}
}

func waitForProductionReadyDialog(t *testing.T, scene *GameScene) *widgets.ReadyEquipmentDialog {
	t.Helper()
	for range 10 {
		if dialog, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.ReadyEquipmentDialog); ok {
			return dialog
		}
		scene.gameState.ElapsedMs += DialogPollingRateMs
		scene.gameState.EventQueueManager.Update()
	}
	t.Fatal("combat Ready dialog was not shown")

	return nil
}

func pressProductionReadyKey(scene *GameScene, dialog *widgets.ReadyEquipmentDialog, key ebiten.Key) {
	scene.keyboard.SetAllowKeyPressImmediately()
	scene.keyboard.SimulateKeyPress(key)
	dialog.Update()
	scene.gameState.ElapsedMs += DialogPollingRateMs
	scene.gameState.EventQueueManager.Update()
}

func drainProductionReadyWorkflow(t *testing.T, scene *GameScene) {
	t.Helper()
	for range 20 {
		scene.gameState.ElapsedMs += DialogPollingRateMs
		scene.gameState.EventQueueManager.Update()
		if !scene.gameState.EventQueueManager.HasWorkflow("ready_equipment_workflow") {
			return
		}
	}
	t.Fatal("combat Ready workflow did not complete")
}

func productionReadyPC(state *game_state.GameState, start int) game_state.CombatTurnIdentity {
	for index := start; index < party_state.MaxActiveParty; index++ {
		identity := game_state.CombatTurnIdentity{IsPartyMember: true, EntityIndex: index}
		if _, ok := state.CombatScene.TurnQueue.GetEntrySnapshot(identity); ok {
			return identity
		}
	}

	return game_state.CombatTurnIdentity{}
}

func applyExpectedProductionReadyChanges(
	t *testing.T,
	scene *GameScene,
	equipment ...references.Equipment,
) {
	t.Helper()
	identity := scene.gameState.CombatScene.GetCurrentTurnIdentity()
	character := scene.gameState.PartyState.GetCharacter(identity.EntityIndex)
	request := game_state.ReadyEquipmentRequest{
		CombatMode: true, CharacterIndex: identity.EntityIndex,
		LockCharacter: true, CombatIdentity: identity,
	}
	for _, item := range equipment {
		_, err := scene.gameState.ApplyReadyEquipment(game_state.ReadyEquipmentTransaction{
			Request: request, CharacterIndex: identity.EntityIndex,
			ExpectedOccupant: character.GetNameAsString(), Equipment: item,
		})
		require.NoError(t, err)
	}
}

func productionReadyRingSeed(t *testing.T, vanishes bool) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 1024; seed++ {
		if (beradrng.New(seed).Intn(16) == 0) == vanishes {
			return seed
		}
	}
	t.Fatal("unable to locate deterministic production Ready ring seed")

	return 0
}

func assertProductionReadyGameplayEqual(t *testing.T, expected, actual *game_state.GameState) {
	t.Helper()
	want := snapshotBlockedCombatState(t, expected)
	got := snapshotBlockedCombatState(t, actual)
	want.ElapsedMs = got.ElapsedMs
	assert.Equal(t, want, got)
}
