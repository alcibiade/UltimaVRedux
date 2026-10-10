package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/minimap"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// PlayWalkingSoundAt implements minimap.WalkingSoundPlayer.
// It plays walking sounds based on the tile at the given position.
func (g *GameScene) PlayWalkingSoundAt(pos references.Position) {
	if g.audioManager == nil {
		return
	}

	var tileIndex indexes.SpriteIndex
	if finaleTile, ok := g.gameState.LordBritishFinaleTile(pos); ok {
		tileIndex = finaleTile
	} else if g.gameState.ShrineScene != nil && g.gameState.ShrineScene.ShrineMap != nil {
		tileIndex = indexes.SpriteIndex(g.gameState.ShrineScene.ShrineMap.GetTile(int(pos.X), int(pos.Y)))
	} else {
		tile := g.gameState.GetLayeredMapByCurrentLocation().GetTileTopMapOnlyTile(&pos)
		if tile == nil {
			return
		}
		tileIndex = tile.Index
	}

	// Create minimal tile for sound lookup
	tile := &references.Tile{Index: tileIndex}

	// Play walking sound based on tile surface
	if err := g.audioManager.PlayWalkingSound(tile); err != nil {
		logging.Warn(logging.CategoryAudio, "Playing mini-map walking sound: %v", err)
	}
}

// thenRunMiniMapScript runs a MiniMapScript on the current miniMapScene.
// The helper encapsulates executor creation and the animation loop.
// Walking sounds are played automatically if script steps have PlayWalkingSounds=true.
//
// The executor is created at runtime (inside Then()) so that g.miniMapScene
// has already been initialized by a prior First() or Then() call.
//
// Uses real delta time tracking to ensure consistent animation speed regardless
// of frame rate variations.
//
// Usage:
//
//	ThenSequence(g.thenRunMiniMapScript(walkInScript)).
func (g *GameScene) thenRunMiniMapScript(script *minimap.MiniMapScript) func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	const frameIntervalMs = 16

	var executor *minimap.ScriptExecutor
	var lastUpdateMs int64

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			// Create executor at runtime when scene exists
			// Pass g as WalkingSoundPlayer for automatic surface detection
			executor = minimap.NewScriptExecutorWithSounds(g.miniMapScene, script, g)
			// Initialize timing from game state's elapsed time
			lastUpdateMs = g.gameState.ElapsedMs
		}).While(
			func() bool { return !executor.IsComplete() },
			func() {
				// Use real delta time instead of fixed 16ms to prevent timing drift
				now := g.gameState.ElapsedMs
				deltaMs := int(now - lastUpdateMs)
				lastUpdateMs = now

				if deltaMs > 0 {
					executor.Update(deltaMs)
				}
			},
			frameIntervalMs,
		)
	}
}
