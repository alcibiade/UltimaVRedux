package main

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

func TestGameSceneElapsedAdditionRejectsOverflow(t *testing.T) {
	scene := &GameScene{elapsedBaseMs: math.MaxInt64 - 100, startTime: time.Now().Add(-time.Second)}
	assert.Panics(t, func() { scene.getElapsedMs() })
	scene.elapsedBaseMs = 12345
	assert.GreaterOrEqual(t, scene.getElapsedMs(), int64(13345))
	scene.elapsedBaseMs = save_snapshot.MaxElapsedMs
	assert.GreaterOrEqual(t, scene.getElapsedMs(), save_snapshot.MaxElapsedMs+1000)
}

func TestGameSceneCaptureSaveSnapshotRejectsPendingUIInputBeforeGameState(t *testing.T) {
	scene := &GameScene{ //nolint:exhaustruct // Only eligibility-owned fields are relevant.
		gameState:         &game_state.GameState{},
		secondaryKeyState: OpenDirectionInput,
	}

	_, err := scene.CaptureSaveSnapshot()
	if !errors.Is(err, ErrGameSceneSnapshotNotRepresentable) {
		t.Fatalf("CaptureSaveSnapshot() error = %v, want ErrGameSceneSnapshotNotRepresentable", err)
	}
	if errors.Is(err, game_state.ErrSaveSnapshotNotRepresentable) {
		t.Fatalf("CaptureSaveSnapshot() delegated before rejecting scene-owned input: %v", err)
	}
}

func TestGameSceneCaptureSaveSnapshotReturnsGameStateEligibilityError(t *testing.T) {
	scene := &GameScene{ //nolint:exhaustruct // Zero-value scene state is eligible for delegation.
		gameState: &game_state.GameState{},
	}

	_, err := scene.CaptureSaveSnapshot()
	if !errors.Is(err, game_state.ErrSaveSnapshotNotRepresentable) {
		t.Fatalf("CaptureSaveSnapshot() error = %v, want ErrSaveSnapshotNotRepresentable", err)
	}
}
