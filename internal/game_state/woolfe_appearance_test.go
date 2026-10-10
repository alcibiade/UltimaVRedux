package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/require"
)

func TestWoolfeFighterPresentationWithAndWithoutVisitingShadowlord(t *testing.T) {
	for _, lord := range []references.Shadowlord{
		references.NoShadowlord, references.Faulinei, references.Astaroth, references.Nosfentor,
	} {
		t.Run(lord.String(), func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(references.Trinsic).
				WithFloor(0).
				WithPlayerAt(25, 8).
				WithRandomSeed(602).
				WithSystemCallbacks().
				Build()
			state.DateTime.Hour = 14
			state.DateTime.Minute = 49
			state.CurrentNPCAIController.HardResetNPCs(false)
			state.ShadowlordState.Reset()
			if lord != references.NoShadowlord {
				state.ShadowlordState.TownAssignments[lord] = references.Trinsic
				present := state.collectShadowlordPresenceData(references.Trinsic, state.MapState.PlayerLocation.Position)
				require.Equal(t, []references.Shadowlord{lord}, present)
			}

			position := references.Position{X: 25, Y: 7}
			unit := state.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(position, 0)
			require.NotNil(t, unit)
			woolfe, ok := (*unit).(*map_units.NPCFriendly)
			require.True(t, ok)
			require.Equal(t, 3, woolfe.MapUnitDetails().NPCNum)
			require.Equal(t, byte(31), woolfe.NPCReference.DialogNumber)
			require.Equal(t, indexes.FighterKeyIndex, woolfe.GetKeySpriteIndex())
			require.Equal(t, references.NoPartyVehicle, woolfe.GetVehicleDetails().VehicleType)
			terrain := state.GetCurrentLayeredMap().GetTileTopMapOnlyTile(&position)
			require.NotNil(t, terrain)
			presentation := state.GetNPCSpriteReplacement(terrain.Index, woolfe.GetKeySpriteIndex(), position)
			require.Equal(t, indexes.FighterKeyIndex, presentation.Sprite)
			require.Equal(t, indexes.FighterKeyIndex, state.GetDisplaySpriteAtPosition(position))
		})
	}
}
