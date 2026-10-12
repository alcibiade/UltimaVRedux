package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Teleportation system methods for GameScene
// Handles moongate teleportation, visual effects, and destination management
//
// Note: Teleportation visual sequence is now managed by event queue system
// in internal/game_state/teleportation_sequence.go

// stabilizeTeleportationDestinationRenderer ensures the destination moongate is visible during fade-in.
func (g *GameScene) stabilizeTeleportationDestinationRenderer(destination game_state.MoongateDestination) {
	if g.moongateVisualSystem == nil {
		panic("GameScene.stabilizeTeleportationDestinationRenderer: moongateVisualSystem is nil - critical system component missing")
	}
	if !g.gameState.AreMoongatesVisible() {
		return
	}

	// Check if there's a moonstone at the destination that should have a moongate
	moonstones := g.gameState.GetMoonstones()
	for i, moonstone := range moonstones {
		if moonstone.X == destination.X && moonstone.Y == destination.Y &&
			moonstone.Map == destination.Map && moonstone.Level == destination.Level && moonstone.CanCreateMoongate() {
			// Create renderer if it doesn't exist
			if g.moongateVisualSystem.activeRenderers[i] == nil {
				position := references.Position{X: references.Coordinate(moonstone.X), Y: references.Coordinate(moonstone.Y)}
				g.moongateVisualSystem.activeRenderers[i] = NewMoongateRenderer(position, i)
			}

			logging.Debug(logging.CategoryMoongate, "Ensured destination moongate %d has a renderer for teleportation", i)

			break
		}
	}
}

// resetTeleportationSourceRenderer clears renderer state before GameState changes location.
func (g *GameScene) resetTeleportationSourceRenderer() {
	g.clearProvisionFeedback()
	// CRITICAL FIX: Reset moongate visual system state before teleportation
	// This clears all renderers and state from the previous location, preventing
	// the bug where moongates repeatedly trigger "fades from view" and block input
	if g.moongateVisualSystem != nil {
		g.moongateVisualSystem.ResetForNewLocation()
	}
}
