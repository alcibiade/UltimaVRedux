package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/shrine"
)

func TestPlayWalkingSoundAtWithoutAudioIsNoOp(t *testing.T) {
	scene := &GameScene{
		gameState: &game_state.GameState{
			ShrineScene: shrine.NewShrineScene(0, references.PlayerLocation{}),
		},
	}

	assert.NotPanics(t, func() {
		scene.PlayWalkingSoundAt(references.Position{X: shrine.CenterX, Y: shrine.EntranceY})
	})
}
