package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/test/helpers"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

type originalRecoveryRNG struct {
	*scriptedOriginalScheduleRNG
	source *beradrng.Source
}

func (r *originalRecoveryRNG) RandomIntInRange(minimum, maximum int) int {
	return minimum + r.source.Intn(maximum-minimum+1)
}

func TestSerpentsHoldOriginalGuardScheduledTravelRecovery(t *testing.T) {
	_, refs, err := helpers.LoadGameData()
	originaldata.Check(t, err)
	slr := refs.LocationReferences.GetLocationReference(references.SerpentsHold)
	emptyMap := &references.LargeMapReference{}
	layeredMaps := map_state.NewLayeredMaps(refs.TileReferences, emptyMap, emptyMap, 11, 11)
	layeredMaps.ResetAndCreateSmallMap(slr, refs.TileReferences, 11, 11)
	state := &map_state.MapState{ //nolint:exhaustruct // Controller-only integration fixture.
		PlayerLocation: references.PlayerLocation{
			Location: references.SerpentsHold,
			Position: references.Position{X: 15, Y: 25},
			Floor:    0,
		},
		LayeredMaps: *layeredMaps,
	}
	date := &datetime.UltimaDate{}
	rng := &originalRecoveryRNG{
		scriptedOriginalScheduleRNG: &scriptedOriginalScheduleRNG{t: t},
		source:                      beradrng.New(1),
	}
	controller := NewNPCAIControllerSmallMap(slr, refs.TileReferences, state, date, rng, nil)
	controller.PopulateMapFirstLoad()
	guards := make([]*map_units.NPCFriendly, 0, 8)
	for _, unit := range controller.mapUnits {
		npc := unit.(*map_units.NPCFriendly)
		monster, combatCapable := npc.NPCReference.SmallMapCombatMonsterType()
		if combatCapable && monster == references.MonsterGuard && npc.Floor() == 0 {
			guards = append(guards, npc)
		}
	}
	require.Len(t, guards, 8)
	require.Equal(t, references.Position{X: 6, Y: 10}, guards[0].Pos())
	require.Equal(t, references.Position{X: 9, Y: 10}, guards[1].Pos())

	// The 05:00 shift sends four guards out of the barracks and four in.
	// Without blocked-travel recovery, guards 3, 7 and 8 deadlock at
	// (8,10), (8,11) and (9,11), despite the passing space beside the aisle.
	// This budget covers this seeded fixture, not arbitrary guard configurations.
	date.Hour = 5
	for range 200 {
		controller.AdvanceNextTurnCalcAndMoveNPCs()
	}
	for _, guard := range guards {
		goal := guard.NPCReference.Schedule.GetIndividualNPCBehaviourByUltimaDate(*date)
		assert.Equal(t, goal.Floor, guard.Floor())
		assert.Equal(t, goal.Position, guard.Pos(), "guard %d", guard.MapUnitDetails().NPCNum)
		assert.Equal(t, guard.NPCReference.Schedule.DesiredSlot(*date), guard.ArrivedScheduleSlot(),
			"guard %d must record arrival", guard.MapUnitDetails().NPCNum)
	}
}
