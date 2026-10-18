package main

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

// Unlike production/shared artwork, the blank sprite atlas belongs exclusively
// to this fixture. Scene disposal retires workflows and presentation, but does
// not release borrowed sprite sheets; the allocating fixture must release it.
func newOwnedSurfaceCompletionTestScene(t *testing.T, state *game_state.GameState) *GameScene {
	t.Helper()
	retirePresentationFixtureCaches(t)
	scene := newSurfaceCompletionTestScene(state)
	t.Cleanup(scene.spriteSheet.SpriteImage.Deallocate)
	t.Cleanup(scene.Dispose)

	return scene
}
