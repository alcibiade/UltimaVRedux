package map_units

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestSurfaceChestAssessmentRestoreAndPhysicalReset(t *testing.T) {
	chest := NewNPCFriendlySurfaceChest(references.BritanniaUnderworld, references.Position{X: 1, Y: 1}, 0, 1)
	assert.Equal(t, references.ChestUninspected, chest.SurfaceChestAssessment())
	chest.RestoreSurfaceChestNumber(0x80 | 30)
	for _, assessment := range []references.ChestAssessment{references.ChestUncertain, references.ChestTrapped} {
		chest.RestoreSurfaceChestAssessment(assessment)
		assert.Equal(t, assessment, chest.SurfaceChestAssessment())
	}
	assert.Panics(t, func() { chest.RestoreSurfaceChestAssessment(references.ChestSafe) })
	assert.Panics(t, func() { chest.RestoreSurfaceChestAssessment(references.ChestAssessment(4)) })
	chest.UnlockSurfaceChest()
	assert.False(t, chest.SurfaceChestLocked())
	assert.Equal(t, references.ChestSafe, chest.SurfaceChestAssessment())
	assert.Panics(t, func() { chest.RestoreSurfaceChestAssessment(references.ChestTrapped) })
	chest.RestoreSurfaceChestNumber(30)
	assert.Equal(t, references.ChestUninspected, chest.SurfaceChestAssessment(), "restoration never infers safety")
	other := NewNPCFriendlyVehiceNoVehicle()
	assert.Panics(t, func() { other.RestoreSurfaceChestAssessment(references.ChestUncertain) })
}
