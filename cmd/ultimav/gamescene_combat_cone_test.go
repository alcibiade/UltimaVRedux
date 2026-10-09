package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestCombatConeArrowGeometryPulsesAwayFromCaster(t *testing.T) {
	still := combatConeArrowSourceGeometry(references.Up, 0)
	pulsed := combatConeArrowSourceGeometry(references.Up, 1)

	assert.Less(t, pulsed.tipY, still.tipY)
	assert.Equal(t, still.tipX, pulsed.tipX)
}

func TestCombatConeArrowColorHighlightsSelection(t *testing.T) {
	assert.Equal(t, ucolor.EGABlue, combatConeArrowColor(references.Up, references.NoneDirection, 0))
	assert.Equal(t, ucolor.EGALightBlue, combatConeArrowColor(references.Up, references.NoneDirection, 1))
	assert.Equal(t, ucolor.EGAMagenta, combatConeArrowColor(references.Up, references.Up, 0))
	assert.Equal(t, ucolor.EGALightMagenta, combatConeArrowColor(references.Up, references.Up, 1))
	assert.Equal(t, ucolor.EGALightBlue, combatConeArrowColor(references.Left, references.Up, 1))
}

func TestCombatDirectionalFootprintOpacityFadesWithDistance(t *testing.T) {
	visual := combatDirectionalFootprintVisualForSpell(references.InZu)

	assert.Greater(t,
		combatDirectionalFootprintOpacity(visual, 1, 10, false),
		combatDirectionalFootprintOpacity(visual, 10, 10, false),
	)
	assert.Greater(t,
		combatDirectionalFootprintOpacity(visual, 1, 10, true),
		combatDirectionalFootprintOpacity(visual, 10, 10, true),
	)
	assert.GreaterOrEqual(t,
		combatDirectionalFootprintOpacity(visual, 1, 10, true)-
			combatDirectionalFootprintOpacity(visual, 10, 10, true),
		float32(0.65),
		"cast spray must retain a clearly visible near-to-far opacity drop",
	)
}

func TestCombatDirectionalFootprintTileUsesDeterministicVariants(t *testing.T) {
	visual := combatDirectionalFootprintVisual{
		Tiles: [2]indexes.SpriteIndex{indexes.MagicField, indexes.FireField}, TileCount: 2,
	}
	position := references.Position{X: 3, Y: 4}

	assert.Equal(t,
		combatDirectionalFootprintTile(visual, position),
		combatDirectionalFootprintTile(visual, position),
	)
}

func TestCombatDirectionalFootprintDistanceUsesCastDirection(t *testing.T) {
	origin := references.Position{X: 5, Y: 5}

	assert.Equal(t, 3, combatDirectionalFootprintDistance(origin, references.Position{X: 5, Y: 8}, references.Down))
	assert.Equal(t, 3, combatDirectionalFootprintDistance(origin, references.Position{X: 2, Y: 5}, references.Left))
}
