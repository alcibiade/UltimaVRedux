package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

// Toggle item flash effect constants
const (
	toggleItemFlashDurationMs = 250  // Duration of flash effect in milliseconds
	toggleItemFlashIntensity  = 0.5  // Flash intensity (0.0-1.0)
	toggleItemFlashGoldR      = 1.0  // Gold/white flash for royal items (Crown, Amulet)
	toggleItemFlashGoldG      = 0.95 // Slightly warm gold tint
	toggleItemFlashGoldB      = 0.85 // Warm gold undertone
	toggleItemFlashDarkR      = 0.15 // Dark/sinister flash for Badge
	toggleItemFlashDarkG      = 0.1  // Very dark
	toggleItemFlashDarkB      = 0.2  // Slight purple tint for ominous effect
)

// TriggerToggleItemFlash triggers a screen flash when equipping toggle items.
// Crown/Amulet get white/gold flash (royal items), Badge gets dark/blackish flash (sinister).
// Uses PartyStateBuilder with event queue for consistent animation pattern.
func (g *GameScene) TriggerToggleItemFlash(itemType game_state.ToggleItemType) {
	// Select flash color based on item type
	var red, green, blue float64

	switch itemType {
	case game_state.ToggleItemAmulet, game_state.ToggleItemCrown:
		// White/gold flash for royal items
		red, green, blue = toggleItemFlashGoldR, toggleItemFlashGoldG, toggleItemFlashGoldB
	case game_state.ToggleItemBadge:
		// Dark/blackish flash for sinister badge
		red, green, blue = toggleItemFlashDarkR, toggleItemFlashDarkG, toggleItemFlashDarkB
	default:
		// Default to gold flash for unknown item types
		red, green, blue = toggleItemFlashGoldR, toggleItemFlashGoldG, toggleItemFlashGoldB
	}

	// Use event queue with PartyStateBuilder for consistent animation pattern
	// This runs concurrently with ShowActionFeedback which is already triggered
	g.gameState.EventQueueManager.TriggerFeedback(
		"toggle_item_flash",
		event_queue.PriorityLow,
		func(sequence *event_queue.SequentialEventBuilder) {
			sequence.First(func() {}).ThenSequence(
				game_state.NewPartyStateBuilder(g.gameState).
					ToAllParty().
					WithFlash(toggleItemFlashDurationMs, red, green, blue, toggleItemFlashIntensity).
					Build())
		},
	)
}
