package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// DetectWishingWellAtPosition checks if a well tile (indexes.Well) exists at the given position.
// Returns true if a well tile is present and detected.
func (g *GameState) DetectWishingWellAtPosition(pos *references.Position) bool {
	if pos == nil {
		return false
	}

	// Get the top tile at the position
	topTile := g.GetLayeredMapByCurrentLocation().GetTopTile(pos)
	if topTile == nil {
		return false
	}

	// Check if this is a well tile (sprite index 161)
	return topTile.Is(indexes.Well)
}

// TriggerWishingWellModern initiates a wishing well interaction using modern UI (InputBox/MessageDialog).
// This is the new implementation that replaces the conversation-based system.
// Flow: YesNo dialog → InputBox for wish → Gold deduction → Direct boarding or "No effect"
func (g *GameState) TriggerWishingWellModern(pos *references.Position) bool {
	// VALIDATION: Nil position check - programming bug if this happens
	if pos == nil {
		panic("TriggerWishingWellModern called with nil position - caller bug")
	}

	if !g.DetectWishingWellAtPosition(pos) {
		logging.Debug(logging.CategoryGameState,
			"Modern well interaction attempted without a well at position (%d,%d) in %v",
			pos.X, pos.Y, g.MapState.PlayerLocation.Location)

		return false
	}

	g.SystemCallbacks.Flow.WorldInteraction.ShowWellInteraction(pos)

	logging.Debug(logging.CategoryGameState,
		"Modern WishingWell interaction triggered at position (%d,%d) in %v",
		pos.X, pos.Y, g.MapState.PlayerLocation.Location)

	return true
}
