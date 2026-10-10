package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestCombatHealthBarFillPixels(t *testing.T) {
	tests := []struct {
		name              string
		currentHP, maxHP  int
		expectedFillPixel int
	}{
		{name: "full", currentHP: 30, maxHP: 30, expectedFillPixel: 11},
		{name: "half", currentHP: 15, maxHP: 30, expectedFillPixel: 6},
		{name: "living minimum", currentHP: 1, maxHP: 100, expectedFillPixel: 1},
		{name: "zero", currentHP: 0, maxHP: 30, expectedFillPixel: 0},
		{name: "invalid maximum", currentHP: 10, maxHP: 0, expectedFillPixel: 0},
		{name: "clamped overheal", currentHP: 40, maxHP: 30, expectedFillPixel: 11},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedFillPixel, combatHealthBarFillPixels(test.currentHP, test.maxHP))
		})
	}
}

func TestCombatHealthBarColorThresholds(t *testing.T) {
	assert.Equal(t, ucolor.From(ucolor.EGALightGreen).Alpha(combatHealthBarAlpha).RGBA(), combatHealthBarColor(51, 100))
	assert.Equal(t, ucolor.From(ucolor.EGAYellow).Alpha(combatHealthBarAlpha).RGBA(), combatHealthBarColor(50, 100))
	assert.Equal(t, ucolor.From(ucolor.EGAYellow).Alpha(combatHealthBarAlpha).RGBA(), combatHealthBarColor(25, 100))
	assert.Equal(t, ucolor.From(ucolor.EGALightRed).Alpha(combatHealthBarAlpha).RGBA(), combatHealthBarColor(24, 100))
}

func TestCombatHealthBarUsesFortyPercentOpacity(t *testing.T) {
	assert.Equal(t, uint8(102), uint8(combatHealthBarAlpha))
}

func TestCombatHealthBarUsesOpaqueBlackBackground(t *testing.T) {
	assert.Equal(t, uint8(ucolor.AlphaOpaque), uint8(combatHealthBarBackgroundAlpha))
}
