package main

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestCombatChestFeedbackRetiresOnlyCapturedInstances(t *testing.T) {
	scene, _ := newChestFeedbackScopeTestScene(t)
	first := references.Position{X: 5, Y: 5}
	second := references.Position{X: 6, Y: 5}
	// Deliberately identical text and widget types: neither identifies ownership.
	scene.ShowActionFeedback("same words", false)
	unrelated := scene.transientActionFeedback[0]
	owned := captureChestFeedbackForTest(scene, first, "same words")
	otherChest := captureChestFeedbackForTest(scene, second, "same words")
	scope := scene.combatChestFeedback[owned]

	scene.retireCombatChestFeedback(scope)
	scene.retireCombatChestFeedback(scope)

	assert.False(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog))
	assert.True(t, scene.dialogStack.ContainsWidgetInstance(unrelated.dialog))
	assert.True(t, scene.dialogStack.ContainsWidgetInstance(otherChest.dialog))
	require.Len(t, scene.combatChestFeedback, 1)
	assert.Contains(t, scene.combatChestFeedback, otherChest)
}

func TestCombatChestFeedbackCapturesEverySurvivingHandleAndPreservesHandled(t *testing.T) {
	scene, _ := newChestFeedbackScopeTestScene(t)
	position := references.Position{X: 5, Y: 5}
	identity := admitChestFeedbackIdentityForTest(scene, position)
	require.True(t, scene.captureCombatChestJimmyFeedback(func() (bool, *game_state.CombatChestIdentity) {
		for range transientFeedbackLimit + 2 {
			scene.ShowActionFeedback("action feedback", false)
		}

		return true, identity
	}))
	require.Len(t, scene.combatChestFeedback, transientFeedbackLimit)
	scope := scene.combatChestFeedback[scene.transientActionFeedback[0]]
	scene.retireCombatChestFeedback(scope)
	assert.Empty(t, scene.combatChestFeedback)
	assert.Zero(t, scene.dialogStack.Count())

	for _, handled := range []bool{false, true} {
		assert.Equal(t, handled, scene.captureCombatChestJimmyFeedback(func() (bool, *game_state.CombatChestIdentity) {
			scene.ShowActionFeedback("ordinary Jimmy result", false)

			return handled, nil
		}))
	}
	assert.Empty(t, scene.combatChestFeedback, "doors and handled failures have no chest-disarm owner")
	assert.Len(t, scene.transientActionFeedback, 2)
}

func TestCombatChestFeedbackLifecycleForgetsExpiredEvictedAndExternalHandles(t *testing.T) {
	for _, mode := range []string{"expiry", "capacity", "external", "scene"} {
		t.Run(mode, func(t *testing.T) {
			scene, state := newChestFeedbackScopeTestScene(t)
			owned := captureChestFeedbackForTest(scene, references.Position{X: 5, Y: 5}, "disarmed")
			scope := scene.combatChestFeedback[owned]
			switch mode {
			case "expiry":
				state.ElapsedMs = transientFeedbackLifetimeMs
				scene.updateTransientActionFeedback()
			case "capacity":
				for range transientFeedbackLimit {
					scene.ShowActionFeedback("unrelated", false)
				}
			case "external":
				require.True(t, scene.dialogStack.RemoveWidgetInstance(owned.dialog))
				scene.updateTransientActionFeedback()
			case "scene":
				scene.PrepareForSceneReplacement()
				scene.PrepareForSceneReplacement()
			}
			assert.Empty(t, scene.combatChestFeedback)
			assert.False(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog))
			remaining := scene.dialogStack.Count()
			scene.retireCombatChestFeedback(scope)
			scene.retireCombatChestFeedback(scope)
			assert.Equal(t, remaining, scene.dialogStack.Count())
		})
	}
}

func TestCombatChestFeedbackEncounterCleanupWhileModalPausesLife(t *testing.T) {
	for _, mode := range []string{"ended", "replaced", "new identity"} {
		t.Run(mode, func(t *testing.T) {
			scene, state := newChestFeedbackScopeTestScene(t)
			owned := captureChestFeedbackForTest(scene, references.Position{X: 5, Y: 5}, "disarmed")
			scene.ShowActionFeedback("unrelated", false)
			unrelated := scene.transientActionFeedback[1]
			modal := &countingDialog{}
			scene.dialogStack.PushModalDialog(modal)
			switch mode {
			case "ended":
				state.CombatScene = nil
			case "replaced":
				replacement := *state.CombatScene
				state.CombatScene = &replacement
			case "new identity":
				state.CombatScene.EncounterID++
			}
			state.ElapsedMs += transientFeedbackLifetimeMs * 2
			scene.updateTransientActionFeedback()
			scene.updateTransientActionFeedback()
			assert.Empty(t, scene.combatChestFeedback)
			assert.False(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog))
			assert.True(t, scene.dialogStack.ContainsWidgetInstance(unrelated.dialog))
			assert.Same(t, modal, scene.dialogStack.PeekTopModalDialog())
		})
	}
}

func TestCombatChestFeedbackOldScopeCannotRetireReplacementEncounter(t *testing.T) {
	scene, state := newChestFeedbackScopeTestScene(t)
	position := references.Position{X: 5, Y: 5}
	old := captureChestFeedbackForTest(scene, position, "disarmed")
	oldScope := scene.combatChestFeedback[old]
	state.CombatScene.EncounterID++
	current := captureChestFeedbackForTest(scene, position, "disarmed")
	scene.retireCombatChestFeedback(oldScope)
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(old.dialog))
	assert.True(t, scene.dialogStack.ContainsWidgetInstance(current.dialog))
	require.Len(t, scene.combatChestFeedback, 1)
}

func TestCombatChestFeedbackProductionDoorAndFailureRemainUnassociated(t *testing.T) {
	for _, mode := range []string{"door", "no keys", "broken key", "already safe"} {
		t.Run(mode, func(t *testing.T) {
			scene := newCombatReportTestScene(t)
			state := scene.gameState
			state.SystemCallbacks.UI.ShowActionFeedback = scene.ShowActionFeedback
			actor := state.CombatScene.GetActiveCombatantIndex()
			state.CombatScene.SetPartyMemberPosition(actor, references.Position{X: 5, Y: 9})
			state.PartyState.GetCharacter(actor).Dexterity = 99
			state.PartyState.Inventory.Provisions.Keys.Set(1)
			position := references.Position{X: 5, Y: 10}
			setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
			chest := game_state.NewCombatChest(position, 127, true)
			switch mode {
			case "door":
				state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.LockedDoor)
			case "no keys":
				state.PartyState.Inventory.Provisions.Keys.Set(0)
			case "broken key":
				state.PartyState.GetCharacter(actor).Dexterity = 1
			case "already safe":
				chest.UnlockChest()
			}
			if mode != "door" {
				require.True(t, state.CombatScene.AddObject(chest))
			}
			scene.combatMapJimmySecondary(references.Down)
			require.Len(t, scene.transientActionFeedback, 1)
			feedback := scene.transientActionFeedback[0]
			assert.Empty(t, scene.combatChestFeedback)
			scene.retireCombatChestFeedback(combatChestFeedbackScope{
				combat: state.CombatScene, encounterID: state.CombatScene.EncounterID,
				chest: game_state.CombatChestIdentity{Position: position},
			})
			assert.True(t, scene.dialogStack.ContainsWidgetInstance(feedback.dialog))
		})
	}
}

// This uses the production Jimmy/Open input helpers and actual core callbacks.
// The input owner must wire captureCombatChestJimmyFeedback around Jimmy.
func TestCombatChestJimmyThenOpenRetiresDisarmBeforeSummary(t *testing.T) {
	scene := newCombatReportTestScene(t)
	state := scene.gameState
	state.SystemCallbacks.UI.ShowActionFeedback = scene.ShowActionFeedback
	_, scene.output = newMainOutput(1280)
	actor := state.CombatScene.GetActiveCombatantIndex()
	state.CombatScene.SetPartyMemberPosition(actor, references.Position{X: 5, Y: 9})
	state.PartyState.GetCharacter(actor).Dexterity = 99
	state.PartyState.Inventory.Provisions.Keys.Set(2)
	position := references.Position{X: 5, Y: 10}
	setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
	require.True(t, state.CombatScene.AddObject(game_state.NewCombatChest(position, 30, true)))
	scene.ShowActionFeedback("unrelated", true)
	unrelated := scene.transientActionFeedback[0]

	scene.combatMapJimmySecondary(references.Down)
	require.Len(t, scene.combatChestFeedback, 1, "production Jimmy must capture its exact disarm toast")
	owned := scene.transientActionFeedback[1]
	history := scene.output.GetOutputStr(false)
	assert.Equal(t, uint16(2), state.PartyState.Inventory.Provisions.Keys.Get())
	assert.False(t, scene.dialogStack.HasModalDialog())
	state.SetRandomSeed(624)
	require.True(t, scene.combatMapOpenSecondary(references.Down))
	require.False(t, state.CombatScene.HasChestIdentity(scene.combatChestFeedback[owned].chest))
	scene.updateTransientActionFeedback()
	assert.True(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog), "the pending reveal snapshot bridges object removal to modal admission")
	state.EventQueueManager.Update()
	dialog, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.CombatLootDialog)
	require.True(t, ok)
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog))
	assert.Empty(t, scene.combatChestFeedback)
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(unrelated.dialog), "contents admission retires all action feedback")
	assert.Contains(t, scene.output.GetOutputStr(false), history, "retiring presentation must retain history")
	inventory, calls := state.PartyState.Inventory.LockerSnapshot(), state.GetRNG().Calls()
	for range 30 {
		state.ElapsedMs += combatChestRevealDelayMs
		state.EventQueueManager.Update()
		scene.updateTransientActionFeedback()
	}
	assert.False(t, dialog.IsDismissed(), "summary still requires ordinary acknowledgement")
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(unrelated.dialog), "old cards cannot return during contents")
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	dialog.Update()
	require.True(t, dialog.IsDismissed())
	state.ElapsedMs += DialogPollingRateMs
	state.EventQueueManager.Update()
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(dialog))
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog), "disarm cannot resurface after dismissal")
	assert.Nil(t, scene.combatLootPreview)
	assert.Equal(t, inventory, state.PartyState.Inventory.LockerSnapshot())
	assert.Equal(t, calls, state.GetRNG().Calls())
	assert.Equal(t, 1, strings.Count(scene.output.GetOutputStr(false), history))
}

func TestCombatChestJimmyThenEmptyOpenRetiresRemovedOwner(t *testing.T) {
	scene := newCombatReportTestScene(t)
	state := scene.gameState
	state.SystemCallbacks.UI.ShowActionFeedback = scene.ShowActionFeedback
	actor := state.CombatScene.GetActiveCombatantIndex()
	state.CombatScene.SetPartyMemberPosition(actor, references.Position{X: 5, Y: 9})
	state.PartyState.GetCharacter(actor).Dexterity = 99
	state.PartyState.Inventory.Provisions.Keys.Set(1)
	position := references.Position{X: 5, Y: 10}
	setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
	require.True(t, state.CombatScene.AddObject(game_state.NewCombatChest(position, 1, true)))
	scene.combatMapJimmySecondary(references.Down)
	require.Len(t, scene.combatChestFeedback, 1)
	owned := scene.transientActionFeedback[0]
	state.SetRandomSeed(169)
	assert.False(t, scene.combatMapOpenSecondary(references.Down))
	assert.False(t, state.EventQueueManager.HasWorkflow("combat_chest_reveal_workflow"))
	assert.False(t, scene.dialogStack.HasModalDialog())
	scene.updateTransientActionFeedback()
	assert.Empty(t, scene.combatChestFeedback)
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog))
}

func TestCombatChestFeedbackSummaryCancellationPreservesOtherOwners(t *testing.T) {
	scene := newCombatReportTestScene(t)
	state := scene.gameState
	position := references.Position{X: 5, Y: 10}
	owned := captureChestFeedbackForTest(scene, position, "disarmed")
	scene.ShowActionFeedback("unrelated", false)
	unrelated := scene.transientActionFeedback[1]
	token := state.FinishCombatActionDeferred()
	scene.startCombatChestRevealWorkflow(game_state.CombatOpenResult{
		Position: position,
		ChestID:  scene.combatChestFeedback[owned].chest.ObjectID,
		RevealItems: []game_state.CombatLootReveal{
			{Sprite: indexes.ItemGem, Description: "a gem!"},
		},
	}, token)
	state.EventQueueManager.Update()
	require.True(t, scene.dialogStack.HasModalDialog())
	require.False(t, scene.dialogStack.ContainsWidgetInstance(unrelated.dialog))
	scene.ShowActionFeedback("later feedback", false)
	later := scene.transientActionFeedback[0]
	state.EventQueueManager.Shutdown()
	state.EventQueueManager.Shutdown()
	assert.False(t, scene.dialogStack.HasModalDialog())
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog))
	assert.True(t, scene.dialogStack.ContainsWidgetInstance(later.dialog), "finalization preserves feedback created after admission")
	assert.Empty(t, scene.combatChestFeedback)
	assert.Nil(t, scene.combatLootPreview)
	assert.False(t, state.CombatScene.HasScheduledCombatStep())
}

func TestCombatChestFeedbackColocatedOpenRetiresAllCardsAtAdmission(t *testing.T) {
	scene := newCombatReportTestScene(t)
	state := scene.gameState
	state.SystemCallbacks.UI.ShowActionFeedback = scene.ShowActionFeedback
	actor := state.CombatScene.GetActiveCombatantIndex()
	state.CombatScene.SetPartyMemberPosition(actor, references.Position{X: 5, Y: 9})
	state.PartyState.GetCharacter(actor).Dexterity = 99
	state.PartyState.Inventory.Provisions.Keys.Set(2)
	position := references.Position{X: 5, Y: 10}
	setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
	require.True(t, state.CombatScene.AddObject(game_state.NewCombatChest(position, 30, false)))
	first, found := state.CombatScene.ObjectAt(position)
	require.True(t, found)
	require.True(t, state.CombatScene.AddObject(game_state.NewCombatChest(position, 30, true)))
	last, found := state.CombatScene.ObjectAt(position)
	require.True(t, found)
	scene.combatMapJimmySecondary(references.Down)
	require.Len(t, scene.combatChestFeedback, 1)
	owned := scene.transientActionFeedback[0]
	assert.Equal(t, last.LifetimeID(), scene.combatChestFeedback[owned].chest.ObjectID)
	state.SetRandomSeed(624)
	require.True(t, scene.combatMapOpenSecondary(references.Down))
	require.NotNil(t, scene.combatLootPreview)
	assert.Equal(t, first.LifetimeID(), scene.combatLootPreview.chestScope.chest.ObjectID)
	state.EventQueueManager.Update()
	scene.updateTransientActionFeedback()
	require.True(t, scene.dialogStack.HasModalDialog())
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog), "contents retire even another chest's completed-action card")
	finishChestFeedbackSummaryForTest(t, scene)
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog))
	require.Empty(t, scene.combatChestFeedback)
	actor = state.CombatScene.GetActiveCombatantIndex()
	state.CombatScene.SetPartyMemberPosition(actor, references.Position{X: 5, Y: 9})
	state.SetRandomSeed(624)
	require.True(t, scene.combatMapOpenSecondary(references.Down))
	assert.Equal(t, last.LifetimeID(), scene.combatLootPreview.chestScope.chest.ObjectID)
	scene.updateTransientActionFeedback()
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog), "the first contents already retired this card")
	state.EventQueueManager.Update()
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog))
	assert.Empty(t, scene.combatChestFeedback)
	finishChestFeedbackSummaryForTest(t, scene)
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog))
	assert.EqualValues(t, 2, state.PartyState.Inventory.Provisions.Keys.Get())
}

func TestCombatChestFeedbackRemovedLifetimeCannotAttachToIdenticalReadmission(t *testing.T) {
	scene := newCombatReportTestScene(t)
	state := scene.gameState
	actor := state.CombatScene.GetActiveCombatantIndex()
	state.CombatScene.SetPartyMemberPosition(actor, references.Position{X: 5, Y: 9})
	position := references.Position{X: 5, Y: 10}
	require.True(t, state.CombatScene.AddObject(game_state.NewCombatChest(position, 1, false)))
	original, found := state.CombatScene.ObjectAt(position)
	require.True(t, found)
	require.True(t, scene.captureCombatChestJimmyFeedback(func() (bool, *game_state.CombatChestIdentity) {
		scene.ShowActionFeedback("same words", false)

		return true, &game_state.CombatChestIdentity{Position: position, ObjectID: original.LifetimeID()}
	}))
	old := scene.transientActionFeedback[0]
	oldScope := scene.combatChestFeedback[old]
	scene.ShowActionFeedback("unrelated", false)
	unrelated := scene.transientActionFeedback[1]
	modal := &countingDialog{}
	scene.dialogStack.PushModalDialog(modal)
	opened := state.ActionOpenCombatMap(references.Down)
	require.Equal(t, original.LifetimeID(), opened.ChestID)
	require.Empty(t, opened.RevealItems, "the value-one fixture must not add overlapping loot")
	require.True(t, state.CombatScene.AddObject(original))
	replacement, found := state.CombatScene.ObjectAt(position)
	require.True(t, found)
	require.Greater(t, replacement.LifetimeID(), original.LifetimeID())
	require.False(t, state.CombatScene.HasChestIdentity(oldScope.chest))
	scene.updateTransientActionFeedback()
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(old.dialog), "removed owner retires even under a modal and with identical contents readmitted")
	assert.True(t, scene.dialogStack.ContainsWidgetInstance(unrelated.dialog))
	assert.Empty(t, scene.combatChestFeedback)
	require.True(t, scene.dialogStack.RemoveWidgetInstance(modal))
	require.True(t, scene.captureCombatChestJimmyFeedback(func() (bool, *game_state.CombatChestIdentity) {
		scene.ShowActionFeedback("same words", false)

		return true, &game_state.CombatChestIdentity{Position: position, ObjectID: replacement.LifetimeID()}
	}))
	current := scene.transientActionFeedback[len(scene.transientActionFeedback)-1]
	require.NotEqual(t, oldScope.chest.ObjectID, scene.combatChestFeedback[current].chest.ObjectID)
	scene.retireCombatChestFeedback(oldScope)
	scene.retireCombatChestFeedback(oldScope)
	assert.True(t, scene.dialogStack.ContainsWidgetInstance(current.dialog))
	require.Len(t, scene.combatChestFeedback, 1)
}

func TestCombatChestFeedbackExpiredSameCellOwnersRemainBounded(t *testing.T) {
	scene, state := newChestFeedbackScopeTestScene(t)
	position := references.Position{X: 5, Y: 5}
	for range transientFeedbackLimit * 2 {
		old := captureChestFeedbackForTest(scene, position, "same words")
		oldScope := scene.combatChestFeedback[old]
		state.ElapsedMs += transientFeedbackLifetimeMs
		scene.updateTransientActionFeedback()
		require.Empty(t, scene.combatChestFeedback)
		require.Empty(t, scene.transientActionFeedback)
		current := captureChestFeedbackForTest(scene, position, "same words")
		require.NotEqual(t, oldScope.chest.ObjectID, scene.combatChestFeedback[current].chest.ObjectID)
		scene.retireCombatChestFeedback(oldScope)
		assert.True(t, scene.dialogStack.ContainsWidgetInstance(current.dialog))
		require.Len(t, scene.combatChestFeedback, 1)
		state.ElapsedMs += transientFeedbackLifetimeMs
		scene.updateTransientActionFeedback()
		require.Empty(t, scene.combatChestFeedback)
		require.Zero(t, scene.dialogStack.Count())
	}
}

func TestCombatChestFeedbackPendingRevealShutdownRetiresRemovedOwner(t *testing.T) {
	scene := newCombatReportTestScene(t)
	state := scene.gameState
	actor := state.CombatScene.GetActiveCombatantIndex()
	state.CombatScene.SetPartyMemberPosition(actor, references.Position{X: 5, Y: 9})
	position := references.Position{X: 5, Y: 10}
	owned := captureChestFeedbackForTest(scene, position, "disarmed")
	scene.ShowActionFeedback("unrelated", false)
	unrelated := scene.transientActionFeedback[1]
	state.SetRandomSeed(624)
	require.True(t, scene.combatMapOpenSecondary(references.Down))
	scene.updateTransientActionFeedback()
	assert.True(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog))
	assert.False(t, scene.dialogStack.HasModalDialog())
	state.EventQueueManager.Shutdown()
	state.EventQueueManager.Shutdown()
	assert.False(t, scene.dialogStack.ContainsWidgetInstance(owned.dialog))
	assert.True(t, scene.dialogStack.ContainsWidgetInstance(unrelated.dialog))
	assert.Empty(t, scene.combatChestFeedback)
	assert.Nil(t, scene.combatLootPreview)
	assert.False(t, state.EventQueueManager.HasInputBlockingWorkflows())
}

func finishChestFeedbackSummaryForTest(t *testing.T, scene *GameScene) {
	t.Helper()
	dialog, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.CombatLootDialog)
	require.True(t, ok)
	for range 30 {
		scene.gameState.ElapsedMs += combatChestRevealDelayMs
		scene.gameState.EventQueueManager.Update()
		scene.updateTransientActionFeedback()
	}
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	dialog.Update()
	require.True(t, dialog.IsDismissed())
	for range 10 {
		scene.gameState.ElapsedMs += DialogPollingRateMs
		scene.gameState.EventQueueManager.Update()
		if !scene.gameState.EventQueueManager.HasWorkflow("combat_chest_reveal_workflow") {
			return
		}
	}
	t.Fatal("chest summary did not release its workflow")
}

func newChestFeedbackScopeTestScene(t *testing.T) (*GameScene, *game_state.GameState) {
	t.Helper()
	scene, state := newTransientFeedbackTestScene(t, references.Britain)
	state.CombatScene = &game_state.CombatState{EncounterID: 1}
	t.Cleanup(scene.clearTransientActionFeedback)

	return scene, state
}

func captureChestFeedbackForTest(scene *GameScene, position references.Position, message string) *transientActionFeedback {
	identity := admitChestFeedbackIdentityForTest(scene, position)
	scene.captureCombatChestJimmyFeedback(func() (bool, *game_state.CombatChestIdentity) {
		scene.ShowActionFeedback(message, false)

		return true, identity
	})

	return scene.transientActionFeedback[len(scene.transientActionFeedback)-1]
}

func admitChestFeedbackIdentityForTest(scene *GameScene, position references.Position) *game_state.CombatChestIdentity {
	combat := scene.gameState.CombatScene
	if !combat.AddObject(game_state.NewCombatChest(position, 30, false)) {
		panic("chest feedback fixture has no free object slot")
	}
	chest, found := combat.ObjectAt(position)
	if !found {
		panic("chest feedback fixture did not admit its object")
	}

	return &game_state.CombatChestIdentity{Position: position, ObjectID: chest.LifetimeID()}
}
