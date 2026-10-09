package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCastPartyConfirmationCostMatrix(t *testing.T) {
	for _, location := range []references.Location{references.Britain, references.BritanniaUnderworld, references.Deceit} {
		for _, outcome := range []string{"cancel", "stale MP", "stale mixture", "no effect", "success"} {
			t.Run(location.String()+"/"+outcome, func(t *testing.T) {
				scene, callbacks := newCastUseCommitScene(t, location)
				state := scene.gameState
				require.True(t, state.SetActivePlayer(0))
				caster := state.PartyState.GetCharacter(0)
				caster.Class = party_state.Wizard
				caster.Level = 8
				caster.SetMp(8)
				state.PartyState.Inventory.Spells.Set(references.AnNox, 2)
				before := state.DateTime
				calls := state.GetRNG().Calls()
				callbacks.Reset()
				if location == references.Deceit {
					scene.dungeonMapInputHandler(ebiten.KeyC)
				} else {
					scene.startSpellCastingWorkflow()
				}
				dialog := waitForProductionSpellDialog(t, scene)
				enterSpellSyllables(scene, dialog, []ebiten.Key{ebiten.KeyA, ebiten.KeyN})
				scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
				dialog.Update()
				targetDialog := waitForProductionCharacterSelection(t, scene)
				assert.Equal(t, byte(8), caster.GetCurrentMp())
				assert.Equal(t, uint16(2), state.PartyState.Inventory.Spells.Get(references.AnNox))
				assert.Equal(t, before, state.DateTime)
				assert.Equal(t, calls, state.GetRNG().Calls())
				target := state.PartyState.GetCharacter(targetDialog.GetCurrentSelection())
				if outcome == "success" {
					target.SetStatus(party_state.Poisoned)
				}
				if outcome == "stale MP" {
					caster.SetMp(0)
				}
				if outcome == "stale mixture" {
					state.PartyState.Inventory.Spells.Set(references.AnNox, 0)
				}
				if outcome == "cancel" {
					targetDialog.HandleEscape()
				} else {
					targetDialog.HandleEnter()
				}
				drainProductionCastWorkflow(t, scene)
				paid := outcome == "success" || outcome == "no effect"
				if paid {
					minutes := 1
					if location == references.BritanniaUnderworld {
						minutes = game_state.DefaultLargeMapMinutesPerTurn
					}
					before.Advance(minutes)
					assert.Equal(t, []int{minutes}, callbacks.TimeAdvanced())
					assert.Equal(t, byte(7), caster.GetCurrentMp())
					assert.Equal(t, uint16(1), state.PartyState.Inventory.Spells.Get(references.AnNox))
					assert.Equal(t, uint64(1), state.CastUseCommitSequence())
				} else {
					assert.Empty(t, callbacks.TimeAdvanced())
					assert.Equal(t, calls, state.GetRNG().Calls())
					if outcome == "stale mixture" {
						assert.Zero(t, state.PartyState.Inventory.Spells.Get(references.AnNox))
						assert.Equal(t, byte(8), caster.GetCurrentMp())
					} else {
						assert.Equal(t, uint16(2), state.PartyState.Inventory.Spells.Get(references.AnNox))
					}
					assert.Zero(t, state.CastUseCommitSequence())
				}
				assert.Equal(t, before, state.DateTime)
				assert.True(t, target.IsGood())
				assert.Zero(t, scene.dialogStack.Count())
				if location == references.Deceit {
					assert.True(t, state.IsAtDungeonCommandBoundary())
				}
			})
		}
	}
}

func TestUseConsumableConfirmationCostMatrix(t *testing.T) {
	for _, location := range []references.Location{references.Britain, references.BritanniaUnderworld, references.Deceit} {
		for _, potion := range []bool{false, true} {
			for _, outcome := range []string{"cancel", "stale inventory", "confirmed"} {
				itemName := "resurrection scroll"
				if potion {
					itemName = "potion"
				}
				t.Run(location.String()+"/"+itemName+"/"+outcome, func(t *testing.T) {
					scene, callbacks := newCastUseCommitScene(t, location)
					state := scene.gameState
					clearUsableItemsForDungeonView(state)
					quantity := func() uint16 { return state.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp) }
					setQuantity := func(n uint16) { state.PartyState.Inventory.Scrolls.Set(references.ScrollInManiCorp, n) }
					workflow := "resurrection_scroll"
					if potion {
						quantity = func() uint16 { return state.PartyState.Inventory.Potions.Get(references.Red) }
						setQuantity = func(n uint16) { state.PartyState.Inventory.Potions.Set(references.Red, n) }
						workflow = "potion_workflow"
					}
					setQuantity(2)
					before := state.DateTime
					calls := state.GetRNG().Calls()
					callbacks.Reset()
					if location == references.Deceit {
						selectOnlyDungeonUseItem(t, scene)
					} else {
						selectSurfaceUseItem(t, scene)
						drainUseSelectionWorkflow(t, scene)
					}
					dialog := waitForProductionCharacterSelection(t, scene)
					assert.Equal(t, uint16(2), quantity())
					assert.Equal(t, before, state.DateTime)
					assert.Equal(t, calls, state.GetRNG().Calls())
					if outcome == "stale inventory" {
						setQuantity(0)
					}
					if outcome == "cancel" {
						dialog.HandleEscape()
					} else {
						dialog.HandleEnter()
					}
					drainDungeonEffectWorkflow(t, scene, workflow)
					if outcome == "confirmed" {
						minutes := 1
						if location == references.BritanniaUnderworld {
							minutes = game_state.DefaultLargeMapMinutesPerTurn
						}
						before.Advance(minutes)
						assert.Equal(t, uint16(1), quantity())
						assert.Equal(t, []int{minutes}, callbacks.TimeAdvanced())
						assert.Equal(t, uint64(1), state.CastUseCommitSequence())
					} else {
						assert.Empty(t, callbacks.TimeAdvanced())
						assert.Equal(t, calls, state.GetRNG().Calls())
						assert.Zero(t, state.CastUseCommitSequence())
					}
					assert.Equal(t, before, state.DateTime)
					assert.Zero(t, scene.dialogStack.Count())
					if location == references.Deceit {
						assert.True(t, state.IsAtDungeonCommandBoundary())
					}
				})
			}
		}
	}
}

func TestWindScrollDirectionEscapeAndSpaceAreFree(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyEscape, ebiten.KeySpace, ebiten.KeyRight} {
		t.Run(key.String(), func(t *testing.T) {
			scene, callbacks := newCastUseCommitScene(t, references.BritanniaUnderworld)
			state := scene.gameState
			clearUsableItemsForDungeonView(state)
			state.PartyState.Inventory.Scrolls.Set(references.ScrollRelHur, 2)
			before := state.DateTime
			calls := state.GetRNG().Calls()
			callbacks.Reset()
			selectSurfaceUseItem(t, scene)
			drainUseSelectionWorkflow(t, scene)
			waitForUseAuthorityDialog(t, scene)
			scene.keyboard.SimulateKeyPress(key)
			drainNamedSurfaceUseWorkflow(t, scene, "wind_change_scroll")
			if key == ebiten.KeyRight {
				assert.Equal(t, uint16(1), state.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur))
				assert.Equal(t, []int{game_state.DefaultLargeMapMinutesPerTurn}, callbacks.TimeAdvanced())
			} else {
				assert.Equal(t, uint16(2), state.PartyState.Inventory.Scrolls.Get(references.ScrollRelHur))
				assert.Equal(t, before, state.DateTime)
				assert.Equal(t, calls, state.GetRNG().Calls())
			}
			assert.Zero(t, scene.dialogStack.Count())
		})
	}
}

func newCastUseCommitScene(t *testing.T, location references.Location) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()
	if location == references.Deceit {
		return newDungeonViewTestScene(t)
	}
	return newSurfaceUseOwnershipSceneAt(t, location, 0, references.Position{X: 15, Y: 15})
}

func TestDungeonUseRejectsMovedSelectorWithoutCommitment(t *testing.T) {
	scene, callbacks := newDungeonViewTestScene(t)
	state := scene.gameState
	clearUsableItemsForDungeonView(state)
	state.PartyState.Inventory.Scrolls.Set(references.ScrollVasLor, 2)
	before := state.DateTime
	calls := state.GetRNG().Calls()
	callbacks.Reset()
	scene.dungeonMapInputHandler(ebiten.KeyU)
	dialog := waitForDungeonItemDialog(t, scene)
	state.MapState.PlayerLocation.Position.X++
	dialog.HandleEnter()
	drainDungeonWorkflow(t, scene, "dungeon_use_item_workflow")

	assert.Equal(t, uint16(2), state.PartyState.Inventory.Scrolls.Get(references.ScrollVasLor))
	assert.Zero(t, state.CastUseCommitSequence())
	assert.Equal(t, before, state.DateTime)
	assert.Equal(t, calls, state.GetRNG().Calls())
	assert.Empty(t, callbacks.TimeAdvanced())
	assert.Contains(t, callbacks.Messages, "Use context changed!")
	assert.True(t, state.IsAtDungeonCommandBoundary())
	assert.False(t, state.EventQueueManager.HasActiveWorkflows())
	assert.Zero(t, scene.dialogStack.Count())
}

func TestWhitePotionCommitsOnlyAfterFinalTargetConfirmation(t *testing.T) {
	for _, location := range []references.Location{references.Britain, references.BritanniaUnderworld, references.Deceit} {
		for _, confirm := range []bool{false, true} {
			name := "cancel"
			if confirm {
				name = "confirmed"
			}
			t.Run(location.String()+"/"+name, func(t *testing.T) {
				scene, callbacks := newCastUseCommitScene(t, location)
				state := scene.gameState
				clearUsableItemsForDungeonView(state)
				state.PartyState.Inventory.Potions.Set(references.White, 2)
				state.SetRandomSeed(findPotionRollSeed(t, state, func(roll int) bool { return roll >= 2 }))
				before := state.DateTime
				calls := state.GetRNG().Calls()
				callbacks.Reset()
				if location == references.Deceit {
					selectOnlyDungeonUseItem(t, scene)
				} else {
					selectSurfaceUseItem(t, scene)
					drainUseSelectionWorkflow(t, scene)
				}
				dialog := waitForProductionCharacterSelection(t, scene)
				assert.Equal(t, uint16(2), state.PartyState.Inventory.Potions.Get(references.White))
				assert.Equal(t, before, state.DateTime)
				assert.Equal(t, calls, state.GetRNG().Calls())
				if confirm {
					dialog.HandleEnter()
					for range 30 {
						state.ElapsedMs += 100
						state.EventQueueManager.Update()
						if scene.potionEffectDialog != nil {
							break
						}
					}
					require.Equal(t, uint64(1), state.CastUseCommitSequence())
					require.NotNil(t, scene.potionEffectDialog)
					assert.Equal(t, location != references.Deceit, scene.xRayVisionActive)
					assert.Equal(t, calls+1, state.GetRNG().Calls(), "mislabelling rolls only after confirmation")
					minutes := 1
					if location == references.BritanniaUnderworld {
						minutes = game_state.DefaultLargeMapMinutesPerTurn
					}
					before.Advance(minutes)
				} else {
					dialog.HandleEscape()
					drainDungeonEffectWorkflow(t, scene, "potion_workflow")
					assert.Equal(t, calls, state.GetRNG().Calls())
					assert.Equal(t, uint16(2), state.PartyState.Inventory.Potions.Get(references.White))
					assert.Empty(t, callbacks.TimeAdvanced())
				}
				state.EventQueueManager.Shutdown()
				if confirm {
					assert.Equal(t, uint16(1), state.PartyState.Inventory.Potions.Get(references.White))
					assert.Len(t, callbacks.TimeAdvanced(), 1, "cleanup charges the committed attempt exactly once")
				}
				assert.Equal(t, before, state.DateTime)
				assert.False(t, scene.xRayVisionActive)
				assert.Nil(t, scene.potionEffectDialog)
				assert.Zero(t, scene.dialogStack.Count())
				if location == references.Deceit {
					assert.True(t, state.IsAtDungeonCommandBoundary())
				}
			})
		}
	}
}

func TestCastDirectionCancellationAndShutdownPreserveResourcesTimeAndRNG(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyEscape, ebiten.KeySpace, ebiten.KeyF12} {
		name := key.String()
		if key == ebiten.KeyF12 {
			name = "shutdown"
		}
		t.Run(name, func(t *testing.T) {
			scene, callbacks := newCastUseCommitScene(t, references.Britain)
			state := scene.gameState
			require.True(t, state.SetActivePlayer(0))
			caster := state.PartyState.GetCharacter(0)
			caster.Class = party_state.Wizard
			caster.Level = 8
			caster.SetMp(8)
			state.PartyState.Inventory.Spells.Set(references.AnYlem, 2)
			state.ConfirmSpellKnowledge(references.AnYlem, game_state.ExperimentationKnowledgeSource())
			before := state.DateTime
			calls := state.GetRNG().Calls()
			callbacks.Reset()
			scene.smallMapInputHandler(ebiten.KeyC)
			dialog := waitForProductionSpellDialog(t, scene)
			enterSpellSyllables(scene, dialog, []ebiten.Key{ebiten.KeyA, ebiten.KeyY})
			scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
			dialog.Update()
			waitForUseAuthorityDialog(t, scene)
			if key == ebiten.KeyF12 {
				state.EventQueueManager.Shutdown()
			} else {
				scene.keyboard.SimulateKeyPress(key)
				drainProductionCastWorkflow(t, scene)
			}
			assert.Equal(t, uint16(2), state.PartyState.Inventory.Spells.Get(references.AnYlem))
			assert.Equal(t, byte(8), caster.GetCurrentMp())
			assert.Equal(t, before, state.DateTime)
			assert.Equal(t, calls, state.GetRNG().Calls())
			assert.Empty(t, callbacks.TimeAdvanced())
			assert.Zero(t, scene.dialogStack.Count())
			assert.Nil(t, scene.castUseDirectionDialog)
		})
	}
}
