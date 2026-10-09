package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Visual effect system implementation

// playKapowAtPosition shows an explosion sprite at the specified world position.
// Scale animates 0.5 → 1.0 → 0.5 over 1.5 seconds, centered on position.
// Used for cannon hits, combat impacts, etc.
func (g *GameScene) playKapowAtPosition(pos references.Position) {
	effect := VisualEffect{
		WorldPos:    &pos,
		SpriteIndex: indexes.Explosion,
		ScaleMin:    0.5,
		ScaleMax:    1.0,
		Type:        "kapow",
		Duration:    0.5,
		Elapsed:     0.0,
	}
	g.visualEffects = append(g.visualEffects, effect)
}

// Note: Old showMissileEffect() removed - replaced by new projectile system in gamescene_projectile.go
// The new system uses Animate() callbacks for timing control and proper sprite rendering.

// Event queue-friendly methods (new pattern - use with Animate())

func (g *GameScene) setScreenFlashColor(r, green, b float64) {
	// Set flash color only - event queue controls intensity via Animate()
	g.screenFlashR = r
	g.screenFlashG = green
	g.screenFlashB = b

	logging.Debug(logging.CategoryGameState, "Screen flash color set: RGB(%.2f,%.2f,%.2f)", r, green, b)
}

func (g *GameScene) setScreenFlashIntensity(intensity float64) {
	// Set flash intensity only - event queue controls fade via Animate()
	g.screenFlashIntensity = intensity

	logging.Debug(logging.CategoryGameState, "Screen flash intensity set: %.2f", intensity)
}

func (g *GameScene) setScreenShakeOffset(x, y float64) {
	// Set camera shake offset directly - event queue controls jolt pattern via Animate()
	g.screenShakeOffsetX = x
	g.screenShakeOffsetY = y

	logging.Debug(logging.CategoryGameState, "Screen shake offset set: (%.2f, %.2f)", x, y)
}

func (g *GameScene) setScreenInvert(intensity float64, durationMs int) {
	// Set invert parameters
	g.screenInvertIntensity = intensity
	g.screenInvertStartTime = g.getElapsedMs()
	g.screenInvertDuration = int64(durationMs)

	logging.Debug(logging.CategoryGameState, "Screen invert triggered: intensity=%.2f duration=%dms", intensity, durationMs)
}

// Utility methods and moongate system management are in gamescene.go and other specialized files
