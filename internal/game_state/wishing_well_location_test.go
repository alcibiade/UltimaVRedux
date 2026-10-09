package game_state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestWishingWellHorseAndMustangRespectOriginalLocationGate(t *testing.T) {
	for _, location := range []references.Location{references.Paws, references.EmpathAbbey, references.IolosHut} {
		for _, wish := range []string{"horse", "Mustang"} {
			t.Run(location.String()+"/"+wish, func(t *testing.T) {
				state, _ := NewIntegrationTestBuilder(t).
					WithLocation(location).
					WithPlayerAt(10, 10).
					WithSystemCallbacks().Build()
				state.SetRandomSeed(1)
				state.PartyState.Inventory.Gold.Set(1)
				well := findWishingWellForTest(t, state)
				review, err := state.ReviewWishingWell(well)
				require.NoError(t, err)
				dateBefore := state.DateTime
				unitCount := len(*state.CurrentNPCAIController.GetNpcs())
				outcome, err := state.CommitWishingWellWish(WishingWellCommitRequest{
					WellPosition: well, Wish: wish, ExpectedFingerprint: review.Fingerprint,
				})
				require.NoError(t, err)
				require.Equal(t, uint16(1), outcome.GoldBefore)
				require.Zero(t, outcome.GoldAfter)
				require.Zero(t, state.PartyState.Inventory.Gold.Get())
				require.Equal(t, dateBefore, state.DateTime)
				require.Zero(t, outcome.ElapsedMinutes)
				require.Len(t, *state.CurrentNPCAIController.GetNpcs(), unitCount)
				if location == references.IolosHut {
					require.Equal(t, WishingWellNoEffect, outcome.Effect)
					require.Equal(t, references.NoPartyVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
					require.Empty(t, outcome.AppearanceMessage)
				} else {
					require.Equal(t, WishingWellHorseEffect, outcome.Effect)
					require.Equal(t, references.HorseVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
				}
			})
		}
	}
}
