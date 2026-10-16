package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestCombatSoloResetModifiedAndOtherNumericKeysKeepExistingScope(t *testing.T) {
	for _, key := range []ebiten.Key{
		ebiten.KeyShiftLeft, ebiten.KeyControlLeft, ebiten.KeyAltLeft, ebiten.KeyMetaLeft,
		ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.KeyNumpad0,
	} {
		t.Run(key.String(), func(t *testing.T) {
			scene, held, _, boundary := newCombatSoloResetUpdateScene(t, true)
			before := snapshotBlockedCombatState(t, scene.gameState)
			held[key] = true
			modified := key == ebiten.KeyShiftLeft || key == ebiten.KeyControlLeft || key == ebiten.KeyAltLeft || key == ebiten.KeyMetaLeft
			if modified {
				held[ebiten.Key0] = true
			}
			require.NoError(t, scene.Update(nil))
			require.True(t, scene.gameState.CombatScene.IsSoloMoveActive())
			require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
			after, eligible := scene.gameState.ProjectCombatSoloReset()
			require.True(t, eligible)
			require.Equal(t, boundary, after)
			delete(held, key)
			require.NoError(t, scene.Update(nil))
			require.True(t, scene.gameState.CombatScene.IsSoloMoveActive(), "releasing only the modifier cannot turn its held 0 into recovery")
		})
	}
}

func TestCombatSoloResetInputOwnersRejectAndDoNotRetainZero(t *testing.T) {
	for _, owner := range []string{"modal", "secondary", "journal", "atlas", "finale", "focus"} {
		t.Run(owner, func(t *testing.T) {
			scene, held, _, boundary := newCombatSoloResetUpdateScene(t, true)
			focused := true
			scene.transientCommandInput.focused = func() bool { return focused }
			var restore func()
			switch owner {
			case "modal":
				scene.showCommandHelp()
				dialog := scene.dialogStack.PeekTopModalDialog()
				restore = func() { scene.dialogStack.RemoveWidgetInstance(dialog) }
			case "secondary":
				scene.secondaryKeyState = OpenDirectionInput
				restore = func() { scene.secondaryKeyState = PrimaryInput }
			case "journal":
				scene.spriteSheet = sharedPresentationTestSpriteSheet()
				scene.ShowQuestJournalDossier(scene.gameState.CurrentQuestJournalDossier())
				restore = func() { scene.closeJournalSession(scene.journalSession, false) }
			case "atlas":
				scene.openAtlas(nil)
				restore = func() { scene.closeAtlasSession(true) }
			case "finale":
				scene.finalePresentation = &lordBritishFinalePresentation{}
				restore = func() { scene.finalePresentation = nil }
			case "focus":
				focused = false
				restore = func() { focused = true }
			}
			held[ebiten.Key0] = true
			require.NoError(t, scene.Update(nil))
			require.True(t, scene.gameState.CombatScene.IsSoloMoveActive())
			after, eligible := scene.gameState.ProjectCombatSoloReset()
			require.True(t, eligible)
			require.Equal(t, boundary, after)
			restore()
			for range 2 {
				require.NoError(t, scene.Update(nil))
				require.True(t, scene.gameState.CombatScene.IsSoloMoveActive(), "carried 0 is not a deferred reset")
			}
			clear(held)
			require.NoError(t, scene.Update(nil))
			held[ebiten.Key0] = true
			require.NoError(t, scene.Update(nil))
			require.False(t, scene.gameState.CombatScene.IsSoloMoveActive(), "a genuinely new press works after the owner leaves")
		})
	}
}

func TestCombatSoloResetNativeSubjectKeepsModalOwnership(t *testing.T) {
	scene, held, _, boundary := newCombatSoloResetUpdateScene(t, true)
	review, err := scene.gameState.ReviewTavernShopByMerchantID(references.MerchantTavernBritain)
	require.NoError(t, err)
	review.State.CustomerEligible = true
	dialog := widgets.NewTavernShopDialog(widgets.TavernShopDialogConfig{
		Keyboard: scene.keyboard, MerchantSprite: ebiten.NewImage(16, 16),
		State: review.State, IsKeyPressed: scene.keyboard.IsKeyHeld,
		IsWindowFocused: scene.commandInputFocused,
		Prose: widgets.TavernShopDialogProse{
			Greeting: "Welcome.", OrderPrompt: "Thy order?", ConversationPrompt: "What subject?",
			ConversationExit: "Anything else?", UnknownTopic: "I know not.",
		},
		// Nil AppendInputChars deliberately exercises the native Subject owner.
	})
	scene.dialogStack.PushModalDialog(dialog)
	t.Cleanup(func() {
		dialog.Dismiss()
		scene.dialogStack.RemoveWidgetInstance(dialog)
	})
	held[ebiten.KeyTab] = true
	require.NoError(t, scene.Update(nil))
	clear(held)
	require.NoError(t, scene.Update(nil))
	require.Equal(t, widgets.TavernShopConversation, dialog.GetMode())
	before := snapshotBlockedCombatState(t, scene.gameState)
	held[ebiten.Key0] = true
	require.NoError(t, scene.Update(nil))
	require.True(t, scene.gameState.CombatScene.IsSoloMoveActive())
	require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
	require.Same(t, dialog, scene.dialogStack.PeekTopModalDialog())
	require.False(t, dialog.HasPendingTopicIntent())
	after, eligible := scene.gameState.ProjectCombatSoloReset()
	require.True(t, eligible)
	require.Equal(t, boundary, after)
	dialog.Dismiss()
	scene.dialogStack.RemoveWidgetInstance(dialog)
	require.NoError(t, scene.Update(nil))
	require.True(t, scene.gameState.CombatScene.IsSoloMoveActive(), "Subject's held 0 cannot escape to combat")
	clear(held)
	require.NoError(t, scene.Update(nil))
	held[ebiten.Key0] = true
	require.NoError(t, scene.Update(nil))
	require.False(t, scene.gameState.CombatScene.IsSoloMoveActive(), "the first fresh 0 after Subject releases ownership resets immediately")
	require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
	after, eligible = scene.gameState.ProjectCombatSoloReset()
	require.True(t, eligible)
	require.Equal(t, boundary, after)
}

func TestCombatSoloResetRejectsUnrelatedWorkflowAtClaimedAndUnclaimedBoundaries(t *testing.T) {
	for _, claimed := range []bool{false, true} {
		t.Run(map[bool]string{false: "unclaimed", true: "claimed"}[claimed], func(t *testing.T) {
			scene, held, _, _ := newCombatSoloResetUpdateScene(t, claimed)
			scene.gameState.EventQueueManager.Workflow("unrelated_reset_owner").Priority(event_queue.PriorityHigh).
				Sequential().BlockInput().First(func() {}).Wait(1000).Then(func() {}).Start()
			held[ebiten.Key0] = true
			require.NoError(t, scene.Update(nil))
			require.True(t, scene.gameState.CombatScene.IsSoloMoveActive())
			require.True(t, scene.keyboard.IsSuppressedUntilRelease(ebiten.Key0))
		})
	}
}

func TestCombatSoloResetNewVerbTieAndTeardownDoNotCarryIntent(t *testing.T) {
	for _, verb := range []ebiten.Key{ebiten.KeyG, ebiten.KeyO, ebiten.KeyJ, ebiten.KeyP, ebiten.KeyS} {
		t.Run(verb.String(), func(t *testing.T) {
			scene, held, _, _ := newCombatSoloResetUpdateScene(t, true)
			held[ebiten.Key0], held[verb] = true, true
			require.NoError(t, scene.Update(nil))
			require.False(t, scene.gameState.CombatScene.IsSoloMoveActive())
			require.False(t, scene.transientCommandInput.pending.active)
			require.True(t, scene.keyboard.IsSuppressedUntilRelease(verb))
			finishCombatSoloResetWait(t, scene)
			require.Equal(t, PrimaryInput, scene.secondaryKeyState)
		})
	}
	for _, replacement := range []bool{false, true} {
		t.Run(map[bool]string{false: "teardown", true: "replacement"}[replacement], func(t *testing.T) {
			scene, held, _, _ := newCombatSoloResetUpdateScene(t, true)
			require.NoError(t, scene.Update(nil)) // Establish the sampled GameState owner.
			held[ebiten.Key0] = true
			if replacement {
				next, _, _, _ := newCombatSoloResetUpdateScene(t, false)
				scene.gameState = next.gameState
			} else {
				scene.ResetTransientCommandInput()
			}
			require.NoError(t, scene.Update(nil))
			require.True(t, scene.gameState.CombatScene.IsSoloMoveActive())
			require.False(t, scene.transientCommandInput.pending.active)
		})
	}
}
