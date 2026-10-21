package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// DetectFountainAtPosition checks if a fountain tile exists at the given position.
// Fountains use sprite indexes 216-219 (Fountain1-4) for animation frames.
// Returns true if a fountain tile is present and detected.
func (g *GameState) DetectFountainAtPosition(pos *references.Position) bool {
	if pos == nil {
		return false
	}

	// Get the top tile at the position
	topTile := g.GetLayeredMapByCurrentLocation().GetTopTile(pos)
	if topTile == nil {
		return false
	}

	// Check if this is any of the fountain animation frames (216-219)
	return topTile.Is(indexes.Fountain1) ||
		topTile.Is(indexes.Fountain2) ||
		topTile.Is(indexes.Fountain3) ||
		topTile.Is(indexes.Fountain4)
}

// TriggerFountainInteraction initiates a fountain drinking interaction using modern UI.
// This is a cosmetic feature - fountains provide refreshment but no healing or stat changes.
// Flow: Character selection → Drinking message → "Refreshing" feedback
//
// Unlike wells, fountains work in all locations (towns, castles) and don't require validation.
func (g *GameState) TriggerFountainInteraction() bool {
	// Delegate to UI layer for dialog sequence
	g.SystemCallbacks.Flow.WorldInteraction.ShowFountainInteraction()

	logging.Debugf(logging.CategoryGameState,
		"Fountain interaction triggered at %v",
		g.MapState.PlayerLocation.Location)

	return true
}
