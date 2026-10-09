package map_units

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransientSurfaceChestIndependentLifetimeAndBoundedAdmission(t *testing.T) {
	position := references.Position{X: 16, Y: 21}
	var units MapUnits
	for number := range MaximumNpcsPerMap {
		chest := units.InsertTransientSurfaceChest(references.LordBritishsCastle, position, references.Basement, 7)
		require.NotNil(t, chest)
		assert.Equal(t, number, chest.MapUnitDetails().NPCNum)
		assert.True(t, chest.IsTransientSurfaceChest())
		assert.False(t, chest.IsAuthoredSurfaceChest())
		assert.Equal(t, 7, chest.SurfaceChestTreasureValue())
		assert.Equal(t, references.ChestUninspected, chest.SurfaceChestAssessment())
	}
	assert.Nil(t, units.InsertTransientSurfaceChest(references.LordBritishsCastle, position, references.Basement, 2))
	require.True(t, units.RemoveNPCByNumber(23))
	child := units.InsertTransientSurfaceChest(references.LordBritishsCastle, position, references.Basement, 2)
	require.NotNil(t, child)
	assert.Equal(t, 23, child.MapUnitDetails().NPCNum)
	assert.False(t, child.IsAuthoredSurfaceChest(), "reusing source slot 23 cannot confer ownership")
	units.DiscardTransientSurfaceChests()
	assert.Empty(t, units)
	assert.False(t, child.IsVisible())
}
