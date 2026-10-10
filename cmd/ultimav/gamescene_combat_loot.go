package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

type combatLootProjection struct {
	position references.Position
	sprite   indexes.SpriteIndex
}

func (g *GameScene) startCombatChestRevealWorkflow(result game_state.CombatOpenResult, token game_state.CombatPresentationToken) {
	var dialog *widgets.CombatLootDialog
	g.gameState.EventQueueManager.Workflow("combat_chest_reveal_workflow").
		Priority(event_queue.PriorityMedium).Sequential().BlockInput().
		First(func() {
			dialog = widgets.NewCombatLootDialog(result.RevealItems, g.keyboard, g.spriteSheet)
			dialog.SetTrapNotice(combatChestTrapText(result.Trap))
			g.dialogStack.PushModalDialog(dialog)
			g.combatLootPreview = &combatLootProjection{position: result.Position, sprite: indexes.NoSprites}
			g.gameState.SystemCallbacks.Message.AddRowStr("Found:")
		}).RepeatWithDelay(len(result.RevealItems), combatChestRevealDelayMs, func(index int) bool {
		entry := result.RevealItems[index]
		dialog.Reveal(index + 1)
		g.combatLootPreview.sprite = entry.Sprite
		g.gameState.SystemCallbacks.Message.AddRowStr(entry.Description)
		g.recordCombatEvent(game_state.CombatEvent{Kind: game_state.CombatEventLootRevealed, LootReveal: entry})

		return true
	}).Then(func() { dialog.FinishReveal() }).
		While(func() bool { return !dialog.IsDismissed() }, func() {}, DialogPollingRateMs).
		Finally(func() {
			if dialog != nil {
				g.dialogStack.MustRemoveWidgetInstance(dialog)
			}
			g.combatLootPreview = nil
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
