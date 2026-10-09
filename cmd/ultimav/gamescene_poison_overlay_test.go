package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// TestActivePlayerNeedsPoisonOverlay_NoActivePlayer prevents a draw-time panic
// when death or transitional state clears the active player to index -1.
func TestActivePlayerNeedsPoisonOverlay_NoActivePlayer(t *testing.T) {
	gameState := &game_state.GameState{}
	gameState.ClearActivePlayer()

	assert.NotPanics(t, func() {
		assert.False(t, activePlayerNeedsPoisonOverlay(gameState))
	})
}

func TestPoisonPulseColor_CyclesBetweenDarkAndLightGreen(t *testing.T) {
	dark := poisonPulseColor(3 * poisonPulsePeriodMs / 4)
	light := poisonPulseColor(poisonPulsePeriodMs / 4)

	assert.Equal(t, ucolor.EGAGreen.ToRGBA(), dark)
	assert.Equal(t, ucolor.EGALightGreen.ToRGBA(), light)
}

func TestPoisonPulseColor_RepeatsAfterPeriod(t *testing.T) {
	assert.Equal(t, poisonPulseColor(0), poisonPulseColor(poisonPulsePeriodMs))
}
