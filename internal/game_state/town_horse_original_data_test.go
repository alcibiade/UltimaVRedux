package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestOriginalTownHorseHitchingAndTalkClassification(t *testing.T) {
	tests := []struct {
		name       string
		location   references.Location
		position   references.Position
		owned      bool
		hitchTiles map[references.Position]indexes.SpriteIndex
	}{
		{
			name:     "Lord British Castle talk-zero horse is hitched and boardable",
			location: references.LordBritishsCastle,
			position: references.Position{X: 21, Y: 27},
			owned:    false,
			hitchTiles: map[references.Position]indexes.SpriteIndex{
				{X: 20, Y: 27}: indexes.WoodFloorShipEdge,
				{X: 22, Y: 27}: indexes.WoodFloorShipEdge,
			},
		},
		{
			name:     "Iolo Hut nonzero-talk horse refuses boarding",
			location: references.IolosHut,
			position: references.Position{X: 27, Y: 3},
			owned:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(test.location).
				WithFloor(references.MainFloor).
				WithSystemCallbacks().
				Build()
			horse := state.CurrentNPCAIController.GetNpcs().GetVehicleAtPositionAndFloorOrNil(
				test.position,
				references.MainFloor,
			)
			require.NotNil(t, horse)
			assert.Equal(t, references.HorseVehicle, horse.GetVehicleDetails().VehicleType)
			assert.Equal(t, test.owned, horse.GetVehicleDetails().IsOwnedByNPC)
			for position, wantTile := range test.hitchTiles {
				tile := state.GetCurrentLayeredMap().GetTileByLayer(map_state.MapLayer, &position)
				require.NotNil(t, tile)
				assert.Equal(t, wantTile, tile.Index)
			}
		})
	}
}
