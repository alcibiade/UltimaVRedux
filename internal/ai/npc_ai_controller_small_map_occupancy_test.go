package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestSmallMapMovementOccupancyUsesExactFloor(t *testing.T) {
	position := references.Position{X: 10, Y: 11}
	otherFloor := map_units.NewNPCFriendly(references.NPCReference{}, 1)
	otherFloor.SetPos(position)
	otherFloor.SetFloor(0)
	sameFloor := map_units.NewNPCFriendly(references.NPCReference{}, 2)
	sameFloor.SetPos(position)
	sameFloor.SetFloor(1)
	controller := &NPCAIControllerSmallMap{
		mapUnits: map_units.MapUnits{otherFloor},
	}

	assert.False(t, controller.isPositionOccupiedOnFloor(position, 1))
	controller.mapUnits = append(controller.mapUnits, sameFloor)
	assert.True(t, controller.isPositionOccupiedOnFloor(position, 1))
}
