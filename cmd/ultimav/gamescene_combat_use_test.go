package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestCombatUseKeyOpensItemWorkflowWithoutDirectionOrImmediateCompletion(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	clearCombatUseTestInventory(scene.gameState)
	scene.gameState.PartyState.Inventory.Potions.Set(references.Black, 1)
	scene.gameState.BeginCombatTurns(game_state.CombatTurnsPresented)
	identityBefore := scene.gameState.CombatScene.GetCurrentTurnIdentity()

	scene.combatMapInputHandler(ebiten.KeyU)

	assert.True(t, scene.gameState.EventQueueManager.HasWorkflow("use_item_workflow"))
	assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
	assert.Equal(t, identityBefore, scene.gameState.CombatScene.GetCurrentTurnIdentity())
}

func TestCombatUseCancellationPreservesItemAndActiveAction(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	clearCombatUseTestInventory(scene.gameState)
	scene.gameState.PartyState.Inventory.Potions.Set(references.Black, 1)
	identityBefore := scene.gameState.CombatScene.GetCurrentTurnIdentity()

	scene.StartCombatUseWorkflow()
	dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
	dialog.Dismiss()
	drainDismissingOverlays(t, scene, "use_item_workflow")

	assert.Equal(t, uint16(1), scene.gameState.PartyState.Inventory.Potions.Get(references.Black))
	assert.Equal(t, identityBefore, scene.gameState.CombatScene.GetCurrentTurnIdentity())
	assert.False(t, scene.gameState.CombatScene.IsPartyMemberInvisible(identityBefore.EntityIndex))
}

func TestCombatSceptreItemCancellationCreatesNoVisualEffect(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	clearCombatUseTestInventory(scene.gameState)
	clearCombatUseQuestItems(scene.gameState)
	scene.gameState.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)

	scene.StartCombatUseWorkflow()
	dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
	dialog.Dismiss()
	drainDismissingOverlays(t, scene, "use_item_workflow")

	assert.Empty(t, scene.combatSceptreEffects)
	assert.Empty(t, scene.combatFieldDispels)
}

func TestBlackPotionAppliesInvisibilityAndCompletesOneAction(t *testing.T) {
	scene, mock := newCombatUseTestScene(t)
	clearCombatUseTestInventory(scene.gameState)
	scene.gameState.PartyState.Inventory.Potions.Set(references.Black, 1)
	seed := findPotionRollSeed(t, scene.gameState, func(roll int) bool { return roll >= 2 })
	scene.gameState.SetRandomSeed(seed)
	identityBefore := scene.gameState.CombatScene.GetCurrentTurnIdentity()

	scene.StartCombatUseWorkflow()
	dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
	dialog.HandleEnter()
	drainDismissingOverlays(t, scene, "use_item_workflow")

	assert.Zero(t, scene.gameState.PartyState.Inventory.Potions.Get(references.Black))
	assert.True(t, scene.gameState.CombatScene.IsPartyMemberInvisible(identityBefore.EntityIndex))
	assert.Empty(t, mock.TimeAdvanced(), "combat Use must finish a DEX action rather than advancing field time")
}

func TestPurplePotionTransformsActiveCombatantAndCompletesAction(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	clearCombatUseTestInventory(scene.gameState)
	scene.gameState.PartyState.Inventory.Potions.Set(references.Purple, 1)
	seed := findPotionRollSeed(t, scene.gameState, func(roll int) bool { return roll >= 2 })
	scene.gameState.SetRandomSeed(seed)
	identityBefore := scene.gameState.CombatScene.GetCurrentTurnIdentity()
	monstersBefore := scene.gameState.CombatScene.MonsterManager.Count()

	scene.StartCombatUseWorkflow()
	dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
	dialog.HandleEnter()
	drainDismissingOverlays(t, scene, "use_item_workflow")

	assert.Zero(t, scene.gameState.PartyState.Inventory.Potions.Get(references.Purple))
	assert.True(t, scene.gameState.CombatScene.PartyActive[identityBefore.EntityIndex])
	assert.True(t, scene.gameState.CombatScene.IsCombatPCRat(identityBefore))
	assert.Equal(t, monstersBefore, scene.gameState.CombatScene.MonsterManager.Count())
}

func TestKalXenCorpScrollConsumesAndSummonsLoyalDaemon(t *testing.T) {
	for _, origin := range []references.Location{references.Stonegate, references.Doom, references.PalaceOfBlackthorn} {
		t.Run(origin.String(), func(t *testing.T) {
			scene, mock := newCombatUseTestScene(t)
			clearCombatUseTestInventory(scene.gameState)
			scene.gameState.CombatScene.PreCombatPlayerLocation.Location = origin
			scene.gameState.PartyState.Inventory.Scrolls.Set(references.ScrollKalXenCorp, 1)
			scene.gameState.SetRandomSeed(1)
			mock.Reset()

			scene.StartCombatUseWorkflow()
			dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
			dialog.HandleEnter()
			drainDismissingOverlays(t, scene, "use_item_workflow")

			assert.Zero(t, scene.gameState.PartyState.Inventory.Scrolls.Get(references.ScrollKalXenCorp))
			require.NotEmpty(t, mock.CombatEvents)
			summon := mock.CombatEvents[0].Summon
			require.NotNil(t, summon)
			assert.Equal(t, game_state.CombatSummonBoundToParty, summon.Binding)
			created := scene.gameState.CombatScene.MonsterManager.GetMonster(summon.Created[0].Index)
			assert.Equal(t, game_state.CombatMonsterLoyaltyParty, created.Loyalty())
		})
	}
}

func TestCombatDurationScrollsActivateAndCompleteAction(t *testing.T) {
	tests := []struct {
		name   string
		scroll references.Scroll
		effect game_state.DurationEffectType
	}{
		{"Vas Lor", references.ScrollVasLor, game_state.DurationEffectNone},
		{"In Sanct", references.ScrollInSanct, game_state.DurationEffectProtection},
		{"In An", references.ScrollInAn, game_state.DurationEffectNegateMagic},
		{"An Tym", references.ScrollAnTym, game_state.DurationEffectTimeStop},
	}

	for _, test := range tests {
		for _, origin := range []references.Location{references.Stonegate, references.Doom, references.PalaceOfBlackthorn} {
			t.Run(origin.String()+"/"+test.name, func(t *testing.T) {
				scene, _ := newCombatUseTestScene(t)
				clearCombatUseTestInventory(scene.gameState)
				scene.gameState.CombatScene.PreCombatPlayerLocation.Location = origin
				scene.gameState.PartyState.Inventory.Scrolls.Set(test.scroll, 1)
				scene.StartCombatUseWorkflow()
				dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
				dialog.HandleEnter()
				drainDismissingOverlays(t, scene, "use_item_workflow")

				assert.Zero(t, scene.gameState.PartyState.Inventory.Scrolls.Get(test.scroll))
				if test.scroll == references.ScrollVasLor {
					assert.Equal(t, 240, scene.gameState.MagicLightMinutesRemaining)
					assert.True(t, scene.gameState.MapState.Lighting.HasMagicLight())
				} else {
					assert.True(t, scene.gameState.HasActiveDurationEffect(test.effect))
				}
			})
		}
	}
}

func TestKalXenCorpScrollNoRoomStillConsumesAndCompletesAction(t *testing.T) {
	scene, mock := newCombatUseTestScene(t)
	clearCombatUseTestInventory(scene.gameState)
	scene.gameState.PartyState.Inventory.Scrolls.Set(references.ScrollKalXenCorp, 1)
	enemyReferences := *scene.gameState.GameReferences.EnemyReferences
	// ref:COMBAT-0001: ordinary monster allocation uses combat slots 6..31.
	for scene.gameState.CombatScene.MonsterManager.Count() < 26 {
		monster := game_state.NewCombatMonster(
			references.MonsterOrc,
			&enemyReferences[references.MonsterOrc],
			references.Position{},
			scene.gameState.AsRandomProvider(),
		)
		scene.gameState.CombatScene.AddMonsterToQueue(monster)
	}
	mock.Reset()

	scene.StartCombatUseWorkflow()
	dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
	dialog.HandleEnter()
	drainDismissingOverlays(t, scene, "use_item_workflow")

	assert.Zero(t, scene.gameState.PartyState.Inventory.Scrolls.Get(references.ScrollKalXenCorp),
		"original Use consumes the scroll before summon placement (ref:CONSUME-0001)")
	assert.Empty(t, mock.CombatEvents)
}

func TestCombatSceptreDispelConsumesOneActionWithoutConsumption(t *testing.T) {
	scene, mock := newCombatUseTestScene(t)
	clearCombatUseTestInventory(scene.gameState)
	clearCombatUseQuestItems(scene.gameState)
	scene.gameState.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
	identityBefore := scene.gameState.CombatScene.GetCurrentTurnIdentity()
	origin := scene.gameState.CombatScene.GetPartyMemberPosition(identityBefore.EntityIndex)
	target := references.Position{X: origin.X, Y: origin.Y + 1}
	require.True(t, scene.gameState.CombatScene.AddField(game_state.NewCombatField(game_state.CombatFieldFire, target)))
	require.True(t, scene.gameState.CombatScene.AddField(game_state.NewCombatField(game_state.CombatFieldSleep, target)))
	mock.Reset()

	selectCombatSceptre(t, scene)
	assertCombatSceptreActorEffect(t, scene, identityBefore, origin)
	resolveCombatSceptreDirection(t, scene, ebiten.KeyArrowDown)
	drainDismissingOverlays(t, scene, "use_item_workflow")

	assert.Equal(t, uint16(1), scene.gameState.PartyState.Inventory.QuestItems.Get(references.Sceptre))
	require.Len(t, scene.gameState.CombatScene.Fields(), 1)
	assert.Equal(t, game_state.CombatFieldSleep, scene.gameState.CombatScene.Fields()[0].Kind())
	assert.Contains(t, mock.Messages, "Field dissolved!")
	assert.Empty(t, mock.TimeAdvanced())
}

func TestCombatSceptreBarrierClearSkipsDirectionalFieldFallback(t *testing.T) {
	scene, mock := newCombatUseTestScene(t)
	clearCombatUseTestInventory(scene.gameState)
	clearCombatUseQuestItems(scene.gameState)
	scene.gameState.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
	identityBefore := scene.gameState.CombatScene.GetCurrentTurnIdentity()
	scene.gameState.CombatScene.SetPartyMemberPosition(identityBefore.EntityIndex, references.Position{X: 5, Y: 5})
	target := references.Position{X: 5, Y: 6}
	scene.gameState.CombatScene.CombatMapRef.SetTileNumber(target, indexes.ShadowlordBoundary1)
	scene.gameState.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &target, indexes.ShadowlordBoundary1)
	require.True(t, scene.gameState.CombatScene.AddField(game_state.NewCombatField(game_state.CombatFieldFire, target)))
	mock.Reset()

	scene.StartCombatUseWorkflow()
	dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
	dialog.HandleEnter()
	drainDismissingOverlays(t, scene, "use_item_workflow")

	assert.Equal(t, indexes.SpriteIndex(indexes.Grass), scene.gameState.CombatScene.CombatMapRef.GetTileNumber(target))
	assert.Equal(t, indexes.SpriteIndex(indexes.Grass),
		scene.gameState.GetCurrentLayeredMap().SpriteIndexByLayer(map_state.MapLayer, target))
	require.Len(t, scene.gameState.CombatScene.Fields(), 1, "terrain clearing must suppress the object-field fallback")
	assert.Equal(t, uint16(1), scene.gameState.PartyState.Inventory.QuestItems.Get(references.Sceptre))
	assert.NotContains(t, mock.Messages, "Field dissolved!")
	assert.NotContains(t, mock.Messages, "No effect!")
	assert.Empty(t, mock.TimeAdvanced())
	assert.Zero(t, scene.dialogStack.Count())
	assertCombatSceptreActorEffect(t, scene, identityBefore, references.Position{X: 5, Y: 5})
}

func TestCombatSceptreNoEffectIsPaidAndPassIsFree(t *testing.T) {
	for _, test := range []struct {
		name        string
		key         ebiten.Key
		wantMessage string
	}{
		{name: "no field", key: ebiten.KeyArrowDown, wantMessage: "No effect!"},
		{name: "passed direction", key: ebiten.KeySpace},
	} {
		t.Run(test.name, func(t *testing.T) {
			scene, mock := newCombatUseTestScene(t)
			clearCombatUseTestInventory(scene.gameState)
			clearCombatUseQuestItems(scene.gameState)
			scene.gameState.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
			identityBefore := scene.gameState.CombatScene.GetCurrentTurnIdentity()
			mock.Reset()

			selectCombatSceptre(t, scene)
			assertCombatSceptreActorEffect(
				t,
				scene,
				identityBefore,
				scene.gameState.CombatScene.GetCombatantPosition(identityBefore),
			)
			resolveCombatSceptreDirection(t, scene, test.key)
			drainDismissingOverlays(t, scene, "use_item_workflow")

			assert.Equal(t, uint16(1), scene.gameState.PartyState.Inventory.QuestItems.Get(references.Sceptre))
			assert.Empty(t, scene.gameState.CombatScene.Fields())
			assert.Empty(t, scene.combatFieldDispels)
			if test.wantMessage != "" {
				assert.Contains(t, mock.Messages, test.wantMessage)
			} else {
				assert.NotContains(t, mock.Messages, "No effect!")
				assert.NotContains(t, mock.Messages, "Field dissolved!")
			}
			assert.Empty(t, mock.TimeAdvanced())
		})
	}
}

func assertCombatSceptreActorEffect(
	t *testing.T,
	scene *GameScene,
	wantIdentity game_state.CombatTurnIdentity,
	wantPosition references.Position,
) {
	t.Helper()
	require.Len(t, scene.combatSceptreEffects, 1)
	effect := scene.combatSceptreEffects[0]
	assert.Equal(t, wantIdentity, effect.Identity)
	assert.Equal(t, wantPosition, effect.Position)
}

func newCombatUseTestScene(t *testing.T) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()
	state, mock := newBritainCombatTestState(t, 1)
	require.True(t, state.CombatScene.GetCurrentTurnIdentity().IsPartyMember)

	// Workflow release predicates must see fixture-owned held keys, not host input.
	return &GameScene{
		gameState:   state,
		output:      text.NewOutput(nil, 1, 10, 80),
		keyboard:    input.NewKeyboardWithKeyStateSource(0, func(ebiten.Key) bool { return false }),
		spriteSheet: &sprites.SpriteSheet{},
	}, mock
}

func clearCombatUseTestInventory(state *game_state.GameState) {
	for potion := references.Blue; potion <= references.White; potion++ {
		state.PartyState.Inventory.Potions.Set(potion, 0)
	}
	for scroll := references.ScrollVasLor; scroll <= references.ScrollAnTym; scroll++ {
		state.PartyState.Inventory.Scrolls.Set(scroll, 0)
	}
}

func clearCombatUseQuestItems(state *game_state.GameState) {
	for _, item := range references.QuestItemCatalog() {
		state.PartyState.Inventory.QuestItems.Set(item, 0)
	}
}

func selectCombatSceptre(t *testing.T, scene *GameScene) {
	t.Helper()
	scene.StartCombatUseWorkflow()
	dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
	dialog.HandleEnter()
	waitForCombatSceptreDirectionPrompt(t, scene)
}

func waitForCombatSceptreDirectionPrompt(t *testing.T, scene *GameScene) {
	t.Helper()
	for range 10 {
		scene.gameState.ElapsedMs += DialogPollingRateMs
		scene.gameState.EventQueueManager.Update()
		if _, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.MessageDialog); ok {
			return
		}
	}
	t.Fatal("combat Sceptre direction prompt was not shown")
}

func resolveCombatSceptreDirection(t *testing.T, scene *GameScene, key ebiten.Key) {
	t.Helper()
	scene.keyboard.SimulateKeyPress(key)
	scene.gameState.ElapsedMs += DialogPollingRateMs
	scene.gameState.EventQueueManager.Update()
}
