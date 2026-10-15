package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/semantic_input"
)

// Input handling and utility methods for GameScene
// Helper methods for input blocking and display updates

// blockAllInput blocks or unblocks all user input
func (g *GameScene) blockAllInput(blocked bool) {
	g.inputBlocked = blocked
}

// IsAnyKeyPressed checks if any key is currently pressed
func (g *GameScene) IsAnyKeyPressed() bool {
	if g.captureRunicTranslationChord() {
		return false
	}
	// Check if any key is currently pressed
	return g.keyboard.IsAnyKeyPressed()
}

// GetPressedKey returns the currently pressed key for event queue key-aware loops
// Returns nil if no key is pressed
// Used by WhileAwaitingInput() for automatic sailing direction changes and stop detection
func (g *GameScene) GetPressedKey() *semantic_input.Key {
	return g.keyboard.GetPressedKey()
}

// updateDisplay triggers immediate screen refresh to show new map items
// This is the equivalent of the original game's losflag |= 2 mechanism
func (g *GameScene) updateDisplay() {
	// Reconcile presentation synchronously: an intermediate cell replacement or
	// visit change must retire old remnants even if the final cell matches again.
	g.pruneDungeonFieldDispels()
	g.updateActorAnimationClocks()
}

// processNPCAutoTalk consumes an NPC command handed off by the preceding NPC turn.
// It preserves the original distinction between Talk, Astaroth influence,
// hostile Guard contact, and tactical combat.
func (g *GameScene) processNPCAutoTalk() bool {
	// Only process on small maps (towns, buildings) where NPCs have conversations
	if g.gameState.MapState.PlayerLocation.Location.GetMapType() != references.SmallMapType {
		return false
	}

	// Get current AI controller
	if g.gameState.CurrentNPCAIController == nil {
		return false
	}

	// Only process small map AI controller
	controller, ok := g.gameState.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	if !ok {
		return false
	}

	// Check each NPC for a pending interaction.
	npcs := controller.GetNpcs()
	if npcs == nil {
		return false
	}

	for _, mapUnit := range *npcs {
		npc, ok := mapUnit.(*map_units.NPCFriendly)
		if !ok || npc.MapUnitDetails().PendingInteraction == map_units.PendingNPCInteractionNone {
			continue
		}
		pendingInteraction := npc.MapUnitDetails().PendingInteraction
		// The original npccmd was ephemeral and consumed immediately after the
		// NPC turn. Clear Redux's pending command before validating it so stale save,
		// schedule, visibility, or movement state cannot retry indefinitely.
		npc.MapUnitDetails().PendingInteraction = map_units.PendingNPCInteractionNone
		if npc.IsEmptyMapUnit() || !npc.IsVisible() {
			continue
		}

		// Skip NPCs not on the same floor as the player - can't interact across floors
		if npc.Floor() != g.gameState.MapState.PlayerLocation.Floor {
			continue
		}

		if !npc.PosPtr().IsNextTo(
			g.gameState.MapState.PlayerLocation.Position,
			npc.Floor(),
			g.gameState.MapState.PlayerLocation.Floor,
		) {
			continue
		}

		logging.Debug(logging.CategoryConversation,
			"Pending interaction %d triggered for NPC %d (DialogNumber: %d) at position (%d,%d)",
			pendingInteraction,
			npc.MapUnitDetails().NPCNum, npc.NPCReference.DialogNumber,
			npc.PosPtr().X, npc.PosPtr().Y)

		switch pendingInteraction {
		case map_units.PendingNPCInteractionNone:
			panic("consumed empty pending NPC interaction")
		case map_units.PendingNPCInteractionTalk:
			// Astaroth contact also uses this workflow: response, damage, then flee.
			// The pending command does not own those gameplay effects or another turn.
			g.startNPCTalkWorkflow(npc)
		case map_units.PendingNPCInteractionGuardArrest:
			g.startNPCGuardArrestWorkflow(npc)
		case map_units.PendingNPCInteractionHostileCombat:
			if err := g.gameState.QueueSmallMapCombatEncounter(
				npc,
				game_state.SmallMapEncounterHostileNPCAdjacency,
			); err != nil {
				logging.Error(logging.CategoryGameState, "Unable to queue hostile small-map NPC combat: %v", err)
				g.gameState.SystemCallbacks.Message.AddRowStr("Combat could not begin.")
			}
		default:
			panic("unsupported pending NPC interaction")
		}

		return true
	}

	return false // No auto-talk triggered
}
