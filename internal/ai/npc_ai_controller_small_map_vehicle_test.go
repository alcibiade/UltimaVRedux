package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestDroppedCarpetDoesNotRunScheduledNPCAI(t *testing.T) {
	position := references.Position{X: 25, Y: 3}
	carpet := map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle, position, 7)
	carpet.SetPos(position)
	carpet.SetFloor(0)
	state := &map_state.MapState{
		PlayerLocation: references.PlayerLocation{Floor: 0},
	}
	controller := &NPCAIControllerSmallMap{
		dateTime: &datetime.UltimaDate{},
		mapState: state,
		mapUnits: map_units.MapUnits{carpet},
	}

	controller.calculateNextNPCPosition(carpet)

	assert.Equal(t, position, carpet.Pos())
	assert.Equal(t, references.FloorNumber(0), carpet.Floor())
}
