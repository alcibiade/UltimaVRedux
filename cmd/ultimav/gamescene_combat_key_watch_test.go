package main

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestCombatSkullKeyPickerDispatchAndOneAction(t *testing.T) {
	for _, dungeon := range []bool{false, true} {
		for _, tile := range []indexes.SpriteIndex{
			indexes.MagicLockDoor, indexes.MagicLockDoorWithView, indexes.LockedDoor, indexes.BrickFloor,
		} {
			t.Run(fmt.Sprintf("dungeon=%v/tile=%v", dungeon, tile), func(t *testing.T) {
				scene, callbacks := newCombatKeyWatchScene(t, references.SkullKeys, dungeon)
				expected, expectedCallbacks := newCombatKeyWatchScene(t, references.SkullKeys, dungeon)
				target := references.Position{X: 5, Y: 4}
				decoy := references.Position{X: 1, Y: 0}
				for _, candidate := range []*GameScene{scene, expected} {
					setProductionCombatPushTerrain(candidate.gameState, target, tile)
					setProductionCombatPushTerrain(candidate.gameState, decoy, indexes.MagicLockDoor)
				}
				before := snapshotBlockedCombatState(t, scene.gameState)
				commits := scene.gameState.CastUseCommitSequence()
				scene.combatMapInputHandler(ebiten.KeyU)
				dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
				dialog.HandleEnter()
				dialog.HandleEnter()
				waitForUseAuthorityDialog(t, scene)
				before.ElapsedMs = scene.gameState.ElapsedMs
				require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState), "prompt is free")
				scene.keyboard.SimulateKeyPress(ebiten.KeyArrowUp)
				drainCombatUseWorkflow(t, scene)

				key, _ := expected.gameState.BeginSkullKeyUse()
				result, paid := expected.gameState.CommitCombatSkullKeyUse(key, references.Up)
				require.True(t, paid)
				expected.gameState.FinishCombatAction()
				assertCombatKeyWatchState(t, scene, expected)
				require.Equal(t, expectedCallbacks.TimeAdvanced(), callbacks.TimeAdvanced())
				require.Equal(t, commits+1, scene.gameState.CastUseCommitSequence())
				require.EqualValues(t, 1, scene.gameState.PartyState.Inventory.Provisions.SkullKeys.Get())
				require.EqualValues(t, indexes.MagicLockDoor, scene.gameState.CombatScene.CombatMapRef.GetTileNumber(decoy))
				if result.Result == game_state.SkullKeyUnlockSuccess {
					require.Contains(t, callbacks.Messages, "Unlocked!")
					require.Contains(t, callbacks.SoundEffectsPlayed, game_state.SoundUnlock)
					require.Len(t, scene.visualEffects, 1)
					require.Equal(t, target, *scene.visualEffects[0].WorldPos)
				} else {
					require.Contains(t, callbacks.Messages, "No effect!")
					require.NotContains(t, callbacks.SoundEffectsPlayed, game_state.SoundUnlock)
					require.Empty(t, scene.visualEffects)
				}
				require.Nil(t, scene.castUseDirectionDialog)
				require.Zero(t, scene.dialogStack.Count())
				drainCombatUseWorkflow(t, scene)
				assertCombatKeyWatchState(t, scene, expected)
			})
		}
	}
}

func TestCombatKeyAndWatchCancellationAndShutdownAreFreeBeforeCommit(t *testing.T) {
	for _, item := range []references.Item{references.SkullKeys, references.PocketWatch} {
		for _, exit := range []string{"menu cancel", "menu shutdown", "direction escape", "direction pass", "direction shutdown"} {
			if item == references.PocketWatch && exit != "menu cancel" && exit != "menu shutdown" {
				continue
			}
			t.Run(fmt.Sprintf("%v/%s", item, exit), func(t *testing.T) {
				scene, _ := newCombatKeyWatchScene(t, item, false)
				before := snapshotBlockedCombatState(t, scene.gameState)
				commits := scene.gameState.CastUseCommitSequence()
				scene.combatMapInputHandler(ebiten.KeyU)
				dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
				switch exit {
				case "menu cancel":
					dialog.Dismiss()
				case "menu shutdown":
					scene.gameState.EventQueueManager.Shutdown()
				default:
					dialog.HandleEnter()
					waitForUseAuthorityDialog(t, scene)
					switch exit {
					case "direction escape":
						scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
					case "direction pass":
						scene.keyboard.SimulateKeyPress(ebiten.KeySpace)
					case "direction shutdown":
						scene.gameState.EventQueueManager.Shutdown()
					}
				}
				drainCombatUseWorkflow(t, scene)
				before.ElapsedMs = scene.gameState.ElapsedMs
				require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
				require.Equal(t, commits, scene.gameState.CastUseCommitSequence())
				require.Zero(t, scene.dialogStack.Count())
				require.Nil(t, scene.castUseDirectionDialog)
				require.False(t, scene.gameState.EventQueueManager.HasInputBlockingWorkflows())
			})
		}
	}
}

func TestCombatKeyAndWatchCancellationDoesNotCompleteUnrelatedCommit(t *testing.T) {
	for _, item := range []references.Item{references.SkullKeys, references.PocketWatch} {
		scene, _ := newCombatKeyWatchScene(t, item, false)
		scene.combatMapInputHandler(ebiten.KeyU)
		dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
		scene.gameState.CommitCastUseAttempt()
		commits := scene.gameState.CastUseCommitSequence()
		before := snapshotBlockedCombatState(t, scene.gameState)
		dialog.Dismiss()
		drainCombatUseWorkflow(t, scene)
		before.ElapsedMs = scene.gameState.ElapsedMs
		require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
		require.Equal(t, commits, scene.gameState.CastUseCommitSequence())
	}
}

func TestCombatSkullKeyInvalidatedPromptCleansUpWithoutMoreInput(t *testing.T) {
	scene, _ := newCombatKeyWatchScene(t, references.SkullKeys, false)
	scene.combatMapInputHandler(ebiten.KeyU)
	waitForTopModal[*widgets.ItemSelectionDialog](t, scene).HandleEnter()
	waitForUseAuthorityDialog(t, scene)
	changeCombatKeyWatchScene(t, scene, references.SkullKeys, "encounter")
	before := snapshotBlockedCombatState(t, scene.gameState)
	commits := scene.gameState.CastUseCommitSequence()
	drainCombatUseWorkflow(t, scene)
	before.ElapsedMs = scene.gameState.ElapsedMs
	require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
	require.Equal(t, commits, scene.gameState.CastUseCommitSequence())
	require.Nil(t, scene.castUseDirectionDialog)
	require.Zero(t, scene.dialogStack.Count())
}

func TestCombatSkullKeyDirectionRejectsStaleOwnerOrStock(t *testing.T) {
	for _, change := range []string{"quantity", "removed", "actor", "clone", "slot provider", "later dispatch", "encounter", "moved", "location", "terrain map", "layered map"} {
		t.Run(change, func(t *testing.T) {
			scene, _ := newCombatKeyWatchScene(t, references.SkullKeys, false)
			setProductionCombatPushTerrain(scene.gameState, references.Position{X: 5, Y: 4}, indexes.MagicLockDoor)
			scene.combatMapInputHandler(ebiten.KeyU)
			waitForTopModal[*widgets.ItemSelectionDialog](t, scene).HandleEnter()
			waitForUseAuthorityDialog(t, scene)
			changeCombatKeyWatchScene(t, scene, references.SkullKeys, change)
			before := snapshotBlockedCombatState(t, scene.gameState)
			commits := scene.gameState.CastUseCommitSequence()
			scene.keyboard.SimulateKeyPress(ebiten.KeyArrowUp)
			drainCombatUseWorkflow(t, scene)
			before.ElapsedMs = scene.gameState.ElapsedMs
			require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
			require.Equal(t, commits, scene.gameState.CastUseCommitSequence())
			require.Zero(t, scene.dialogStack.Count())
			require.Nil(t, scene.castUseDirectionDialog)
		})
	}
}

func TestCombatPocketWatchPickerAwaitsDismissalBeforeOneAction(t *testing.T) {
	scene, callbacks := newCombatKeyWatchScene(t, references.PocketWatch, false)
	expected, expectedCallbacks := newCombatKeyWatchScene(t, references.PocketWatch, false)
	commits := scene.gameState.CastUseCommitSequence()
	scene.combatMapInputHandler(ebiten.KeyU)
	dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
	// The clock is read at execution, not when the item picker was opened.
	for _, candidate := range []*GameScene{scene, expected} {
		candidate.gameState.DateTime.Hour, candidate.gameState.DateTime.Minute = 23, 47
	}
	before := snapshotBlockedCombatState(t, scene.gameState)
	dialog.HandleEnter()
	watch := waitForCombatPocketWatch(t, scene)
	require.Equal(t, "The time is "+scene.gameState.DateTime.GetTimeAsString(), watch.GetMessage())
	require.Equal(t, commits+1, scene.gameState.CastUseCommitSequence())
	require.False(t, scene.gameState.EventQueueManager.HasWorkflow("pocket_watch"), "the parent owns the display sequence")
	for range 10 {
		scene.gameState.ElapsedMs += 1000
		scene.gameState.EventQueueManager.Update()
		require.True(t, scene.gameState.EventQueueManager.HasWorkflow("use_item_workflow"))
		require.True(t, scene.gameState.EventQueueManager.HasInputBlockingWorkflows())
		require.False(t, watch.IsDismissed())
		before.ElapsedMs = scene.gameState.ElapsedMs
		require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
	}
	watch.Dismiss()
	watch.Dismiss()
	drainCombatUseWorkflow(t, scene)
	commit := expected.gameState.BeginCombatPocketWatchUse(expected.gameState.CaptureCombatUseOwner())
	_, paid := expected.gameState.CommitCombatPocketWatchUse(commit)
	require.True(t, paid)
	expected.gameState.FinishCombatAction()
	assertCombatKeyWatchState(t, scene, expected)
	require.Equal(t, expectedCallbacks.TimeAdvanced(), callbacks.TimeAdvanced())
	require.EqualValues(t, 1, scene.gameState.PartyState.Inventory.QuestItems.Get(references.PocketWatch))
	require.Zero(t, scene.dialogStack.Count())
	require.Nil(t, scene.pocketWatchDialog)
	require.False(t, scene.gameState.EventQueueManager.HasInputBlockingWorkflows())
	drainCombatUseWorkflow(t, scene)
	assertCombatKeyWatchState(t, scene, expected)
}

func TestCombatPocketWatchRejectsChangesBetweenConstructionAndCommit(t *testing.T) {
	for _, change := range []string{"quantity", "removed", "actor", "clone", "slot provider", "later dispatch", "encounter", "moved", "location", "terrain map", "layered map"} {
		t.Run(change, func(t *testing.T) {
			scene, _ := newCombatKeyWatchScene(t, references.PocketWatch, false)
			state := scene.gameState
			items := state.GetUsableItemsForContext(game_state.UsageCombat)
			require.Len(t, items, 1)
			owner, origin := state.CaptureCombatUseOwner(), state.MapState.PlayerLocation
			var confirmation *hmsCapeConfirmationState
			didCommit := false
			sequence := scene.thenUseCombatItem(items[0], &confirmation, func() bool {
				return scene.validateCombatUseSelection(items[0], owner, origin)
			}, &didCommit)
			builder := state.EventQueueManager.Workflow("use_item_workflow").
				Priority(event_queue.PriorityNormal).Sequential().BlockInput().First(func() {})
			sequence(builder)
			builder.Finally(scene.clearPocketWatchDialog)
			changeCombatKeyWatchScene(t, scene, references.PocketWatch, change)
			before := snapshotBlockedCombatState(t, state)
			commits := state.CastUseCommitSequence()
			builder.Start()
			drainCombatUseWorkflow(t, scene)
			before.ElapsedMs = state.ElapsedMs
			require.Equal(t, before, snapshotBlockedCombatState(t, state))
			require.Equal(t, commits, state.CastUseCommitSequence())
			require.False(t, didCommit)
			require.Zero(t, scene.dialogStack.Count())
		})
	}
}

func TestCombatPocketWatchInvalidatedDisplayCleansUpWithoutCompletingReplacement(t *testing.T) {
	for _, change := range []string{"actor", "clone", "slot provider", "later dispatch", "encounter", "moved", "location", "terrain map", "layered map"} {
		for _, shutdown := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/shutdown=%v", change, shutdown), func(t *testing.T) {
				scene, _ := newCombatKeyWatchScene(t, references.PocketWatch, false)
				scene.combatMapInputHandler(ebiten.KeyU)
				waitForTopModal[*widgets.ItemSelectionDialog](t, scene).HandleEnter()
				watch := waitForCombatPocketWatch(t, scene)
				changeCombatKeyWatchScene(t, scene, references.PocketWatch, change)
				before := snapshotBlockedCombatState(t, scene.gameState)
				commits := scene.gameState.CastUseCommitSequence()
				if shutdown {
					scene.gameState.EventQueueManager.Shutdown()
				} else {
					// Owner invalidation ends the wait without requiring another key.
					for range 20 {
						scene.gameState.ElapsedMs += DialogPollingRateMs
						scene.gameState.EventQueueManager.Update()
					}
				}
				require.False(t, scene.gameState.EventQueueManager.HasWorkflow("use_item_workflow"))
				require.True(t, watch.IsDismissed())
				require.Nil(t, scene.pocketWatchDialog)
				require.Zero(t, scene.dialogStack.Count())
				before.ElapsedMs = scene.gameState.ElapsedMs
				require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
				require.Equal(t, commits, scene.gameState.CastUseCommitSequence())
			})
		}
	}
}

func newCombatKeyWatchScene(
	t *testing.T, item references.Item, dungeon bool,
) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()
	scene, callbacks := newStartedCombatPushScene(t)
	state := scene.gameState
	if dungeon {
		require.NoError(t, state.TransitionToCombatMap(0, references.DungeonCombatTerritory, game_state.CombatEntryNorth))
		state.CombatScene.ExitContext = game_state.CombatExitDungeonRoom
		state.BeginCombatTurns(game_state.CombatTurnsImmediate)
	}
	clearCombatUseTestInventory(state)
	clearCombatUseQuestItems(state)
	for moonstone := references.NewMoon; moonstone <= references.CrescentWaning; moonstone++ {
		state.PartyState.Inventory.Moonstones.Set(moonstone, 0)
	}
	state.PartyState.Inventory.Provisions.SkullKeys.Set(0)
	if item == references.SkullKeys {
		state.PartyState.Inventory.Provisions.SkullKeys.Set(2)
	} else {
		state.PartyState.Inventory.QuestItems.Set(references.PocketWatch, 1)
	}
	actor := state.CombatScene.GetCurrentTurnIdentity()
	require.True(t, actor.IsPartyMember)
	state.CombatScene.SetPartyMemberPosition(actor.EntityIndex, references.Position{X: 5, Y: 5})
	state.MapState.PlayerLocation.Position = references.Position{X: 1, Y: 1}
	callbacks.Reset()

	return scene, callbacks
}

func waitForCombatPocketWatch(t *testing.T, scene *GameScene) *widgets.OverlayDialog {
	t.Helper()
	for range 30 {
		scene.gameState.ElapsedMs += DialogPollingRateMs
		scene.gameState.EventQueueManager.Update()
		if scene.pocketWatchDialog != nil {
			return scene.pocketWatchDialog
		}
	}
	t.Fatal("combat watch display did not open")

	return nil
}

func assertCombatKeyWatchState(t *testing.T, actual, expected *GameScene) {
	t.Helper()
	expected.gameState.ElapsedMs = actual.gameState.ElapsedMs
	require.Equal(t, snapshotBlockedCombatState(t, expected.gameState), snapshotBlockedCombatState(t, actual.gameState))
	require.Equal(t, expected.gameState.CastUseCommitSequence(), actual.gameState.CastUseCommitSequence())
	require.Equal(t, PrimaryInput, actual.secondaryKeyState)
}

func changeCombatKeyWatchScene(t *testing.T, scene *GameScene, item references.Item, change string) {
	t.Helper()
	state := scene.gameState
	actor := state.CombatScene.GetCurrentTurnIdentity()
	switch change {
	case "quantity", "removed":
		quantity := uint16(3)
		if change == "removed" {
			quantity = 0
		}
		if item == references.SkullKeys {
			state.PartyState.Inventory.Provisions.SkullKeys.Set(quantity)
		} else {
			state.PartyState.Inventory.QuestItems.Set(references.PocketWatch, quantity)
		}
	case "slot provider":
		state.CombatScene.TurnQueue.GetEntryByIdentity(actor).Combatant =
			game_state.NewCombatPCProvider(state.PartyState.GetCharacter(actor.EntityIndex))
	case "later dispatch":
		state.FinishCombatAction()
		for range 30 {
			if state.CombatScene.GetCurrentTurnIdentity() == actor {
				return
			}
			state.FinishCombatAction()
		}
		t.Fatal("same actor was not dispatched again")
	case "moved":
		state.CombatScene.SetPartyMemberPosition(actor.EntityIndex, references.Position{X: 6, Y: 5})
	case "terrain map":
		replacement := *state.CombatScene.CombatMapRef
		state.CombatScene.CombatMapRef = &replacement
	case "layered map":
		state.MapState.LayeredMaps.ResetAndCreateCombatMap(state.CombatScene.CombatMapRef, state.GameReferences.TileReferences, 19, 13)
	default:
		changeStaleCombatUseState(t, scene, item, change)
	}
}
