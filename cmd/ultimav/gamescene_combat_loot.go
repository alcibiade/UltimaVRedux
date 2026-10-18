package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

type combatLootProjection struct {
	position   references.Position
	sprite     indexes.SpriteIndex
	chestScope combatChestFeedbackScope
}

func (g *GameScene) startCombatChestRevealWorkflow(result game_state.CombatOpenResult, token game_state.CombatPresentationToken) {
	timing := g.captureCombatTiming()
	var dialog *widgets.CombatLootDialog
	combat := g.gameState.CombatScene
	scope := combatChestFeedbackScope{
		combat: combat, encounterID: combat.EncounterID,
		chest: game_state.CombatChestIdentity{Position: result.Position, ObjectID: result.ChestID},
	}
	// Retain the authoritative removed chest before the first queued callback;
	// feedback reconciliation can run while this reveal awaits admission.
	preview := &combatLootProjection{position: result.Position, sprite: indexes.NoSprites, chestScope: scope}
	g.combatLootPreview = preview
	g.gameState.EventQueueManager.Workflow("combat_chest_reveal_workflow").
		Priority(event_queue.PriorityMedium).Sequential().BlockInput().
		First(func() {
			dialog = widgets.NewCombatLootDialog(result.RevealItems, g.keyboard, g.spriteSheet)
			dialog.SetTrapNotice(combatChestTrapText(result.Trap))
			// Contents replace all completed-action cards at actual admission.
			// Finalization remains scope-owned so later feedback survives cleanup.
			g.clearTransientActionFeedback()
			g.dialogStack.PushModalDialog(dialog)
			g.gameState.SystemCallbacks.Message.AddRowStr("Found:")
		}).RepeatWithDelay(len(result.RevealItems), timing.durationMs(combatChestRevealDelayMs), func(index int) bool {
		entry := result.RevealItems[index]
		dialog.Reveal(index + 1)
		preview.sprite = entry.Sprite
		g.gameState.SystemCallbacks.Message.AddRowStr(entry.Description)
		g.recordCombatEvent(game_state.CombatEvent{Kind: game_state.CombatEventLootRevealed, LootReveal: entry})

		return true
	}).Then(func() { dialog.FinishReveal() }).
		While(func() bool { return !dialog.IsDismissed() }, func() {}, DialogPollingRateMs).
		Finally(func() {
			if dialog != nil {
				g.dialogStack.MustRemoveWidgetInstance(dialog)
			}
			g.retireCombatChestFeedback(scope)
			if g.combatLootPreview == preview {
				g.combatLootPreview = nil
			}
			g.keyboard.PrepareForTransition(0)
			g.completeScheduledCombatStep("chest reveal", token)
		}).Start()
}

func (g *GameScene) combatLootTile(position references.Position, fallback indexes.SpriteIndex) indexes.SpriteIndex {
	if !g.gameState.IsInCombat() || !g.gameState.CombatTerrainAllowsObjectDisplay(position) {
		return fallback
	}
	if preview := g.combatLootPreview; preview != nil && preview.position == position {
		if preview.sprite == indexes.NoSprites {
			return g.gameState.GetBonusTerrainTile(&position)
		}

		return preview.sprite
	}
	if sprite, found := g.gameState.CombatPileSprite(position); found {
		return sprite
	}

	return fallback
}
