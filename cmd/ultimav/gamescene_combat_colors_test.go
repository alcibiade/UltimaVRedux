package main

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestGetCombatAimColor_UsesSemanticPalette(t *testing.T) {
	tests := []struct {
		name        string
		disposition game_state.CombatAimDisposition
		intensity   float32
		expected    ucolor.EGAColor
	}{
		{"empty bright", game_state.CombatAimEmpty, 1, ucolor.CombatAimEmptyBright},
		{"empty dim", game_state.CombatAimEmpty, 0, ucolor.CombatAimEmptyDim},
		{"friendly bright", game_state.CombatAimFriendly, 1, ucolor.CombatAimFriendlyBright},
		{"friendly dim", game_state.CombatAimFriendly, 0, ucolor.CombatAimFriendlyDim},
		{"enemy bright", game_state.CombatAimEnemy, 1, ucolor.CombatAimEnemyBright},
		{"enemy dim", game_state.CombatAimEnemy, 0, ucolor.CombatAimEnemyDim},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected.ToRGBA(), getCombatAimColor(test.disposition, test.intensity))
		})
	}
}

func TestGetCombatAimColor_InterpolatesVGAIntermediate(t *testing.T) {
	actual := getCombatAimColor(game_state.CombatAimEmpty, 0.5)
	assert.Equal(t, color.RGBA{R: 212, G: 212, B: 0, A: 255}, actual)
}

func TestCombatOutlinePulses_AreQuantizedAndBounded(t *testing.T) {
	tests := []struct {
		name   string
		pulse  func(int64) float32
		period int64
		low    float32
		high   float32
		steps  int
	}{
		{
			"active PC", activePCOutlinePulse,
			sprites.ActivePCOutlinePulsePeriodMs,
			sprites.ActivePCOutlinePulseLow, sprites.ActivePCOutlinePulseHigh,
			sprites.ActivePCOutlinePulseSteps,
		},
		{
			"active monster", activeMonsterOutlinePulse,
			sprites.ActiveMonsterOutlinePulsePeriodMs,
			sprites.ActiveMonsterOutlinePulseLow, sprites.ActiveMonsterOutlinePulseHigh,
			sprites.ActiveMonsterOutlinePulseSteps,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.InDelta(t, test.low, test.pulse(0), 0.001)
			assert.InDelta(t, test.high, test.pulse(test.period/2), 0.001)

			values := make(map[float32]struct{})
			for elapsed := int64(0); elapsed < test.period; elapsed += 10 {
				value := test.pulse(elapsed)
				assert.GreaterOrEqual(t, value, test.low)
				assert.LessOrEqual(t, value, test.high)
				values[value] = struct{}{}
			}
			assert.LessOrEqual(t, len(values), test.steps)
		})
	}
}

func TestCombatReticleBoundsAlignWithInsetTileCell(t *testing.T) {
	drawOptions := &ebiten.DrawImageOptions{} //nolint:exhaustruct // Identity transform isolates source geometry.

	left, top, right, bottom := combatReticleBounds(drawOptions)

	assert.Equal(t, float64(combatSourceTileInset), left)
	assert.Equal(t, float64(combatSourceTileInset), top)
	assert.Equal(t, float64(combatSourceTileInset+combatSourceTilePixels), right)
	assert.Equal(t, float64(combatSourceTileInset+combatSourceTilePixels), bottom)
}

// TestCombatPartyTurnIndex_HidesPlayerSelectionDuringMonsterTurn prevents the
// previous PC retaining a green outline while an enemy is moving or attacking.
func TestCombatPartyTurnIndex_HidesPlayerSelectionDuringMonsterTurn(t *testing.T) {
	assert.Equal(t, game_state.NoTargetPC, combatPartyTurnIndex(2, 0, false))
	assert.Equal(t, game_state.NoTargetPC, combatPartyTurnIndex(game_state.NoActiveMonster, 0, true))
	assert.Equal(t, 0, combatPartyTurnIndex(game_state.NoActiveMonster, 0, false))
}
