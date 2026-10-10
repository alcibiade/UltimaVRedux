package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/require"
)

func TestStonegateOriginalTrapRing(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Stonegate).
		WithPlayerAt(15, 16).WithSystemCallbacks().Build()
	var traps []references.Position
	for y := range 32 {
		for x := range 32 {
			position := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
			tile := state.GetCurrentLayeredMap().GetTileByLayer(map_state.MapLayer, &position)
			if tile.Is(indexes.BrickFloorHole) {
				traps = append(traps, position)
			}
			require.False(t, tile.Is(indexes.Grate), "Stonegate contains traps, not grates")
		}
	}
	require.Equal(t, []references.Position{
		{X: 14, Y: 14}, {X: 15, Y: 14}, {X: 16, Y: 14},
		{X: 14, Y: 15}, {X: 16, Y: 15},
		{X: 14, Y: 16}, {X: 15, Y: 16}, {X: 16, Y: 16},
	}, traps)
}

func TestStonegateInitialDamageUsesIndependentSlotOrderedRolls(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Stonegate).
		WithPlayerAt(15, 16).WithSystemCallbacks().Build()
	for slot, status := range []party_state.CharacterStatus{
		party_state.Good, party_state.Sleep, party_state.Poisoned,
		party_state.Dead, party_state.Good, party_state.Good,
	} {
		character := state.PartyState.GetCharacter(slot)
		character.PartyMap = party_state.PartyMapActive
		character.SetMaxHp(100)
		character.SetHp(100)
		character.SetStatus(status)
	}
	state.PartyState.GetCharacter(4).PartyMap = party_state.PartyMapUnrecruited
	state.PartyState.GetCharacter(0).SetHp(1)
	inactive := *state.PartyState.GetCharacter(4)
	outsideActiveSlots := *state.PartyState.GetCharacter(6)
	require.True(t, state.SetActivePlayer(0))
	state.SetRandomSeed(701)
	wantRolls := []int{state.RandomIntInRange(1, 8), state.RandomIntInRange(1, 8),
		state.RandomIntInRange(1, 8), state.RandomIntInRange(1, 8)}
	state.SetRandomSeed(701)
	review, err := state.ReviewStonegateTrap()
	require.NoError(t, err)
	outcome, err := state.CommitStonegateTrap(review.Fingerprint)
	require.NoError(t, err)
	require.Len(t, outcome.InitialDamage, 4)
	for index, slot := range []int{0, 1, 2, 5} {
		damage := outcome.InitialDamage[index]
		require.Equal(t, slot, damage.Before.Slot)
		require.Equal(t, uint16(max(0, int(damage.Before.HP)-wantRolls[index])), damage.After.HP)
		if slot == 0 {
			require.Equal(t, party_state.Dead, damage.After.Status)
		} else {
			require.Equal(t, damage.Before.Status, damage.After.Status)
		}
	}
	require.Equal(t, uint64(4), state.GetRNG().Calls())
	require.Equal(t, inactive, *state.PartyState.GetCharacter(4))
	require.Equal(t, outsideActiveSlots, *state.PartyState.GetCharacter(6))
	require.Equal(t, -1, state.GetActivePlayerIndex())
	for _, member := range outcome.KilledMembers {
		require.Zero(t, member.After.HP)
		require.Equal(t, party_state.Dead, member.After.Status)
	}
	_, err = state.CommitStonegateTrap(review.Fingerprint)
	require.ErrorIs(t, err, ErrStonegateTrapInvalid)
	require.Equal(t, uint64(4), state.GetRNG().Calls())
}

func TestStonegateStandingTerrainAndCarpetProtection(t *testing.T) {
	for _, test := range []struct {
		name     string
		riding   bool
		terrain  indexes.SpriteIndex
		wantTrap bool
	}{
		{name: "carrying does not protect", terrain: indexes.BrickFloorHole, wantTrap: true},
		{name: "riding protects", riding: true, terrain: indexes.BrickFloorHole},
		{name: "grate does not trigger", terrain: indexes.Grate},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.Stonegate).
				WithPlayerAt(15, 16).WithRandomSeed(701).WithSystemCallbacks().Build()
			state.PartyState.Inventory.QuestItems.Set(references.Carpet, 1)
			position := state.MapState.PlayerLocation.Position
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, test.terrain)
			// Object presentation must not hide the standing terrain hazard.
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapUnitLayer, &position, indexes.Chest)
			if test.riding {
				state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle, position, 0)
			}
			state.CompleteSurfacePassCycle(references.SmallMapType)
			require.Equal(t, test.wantTrap, callbacks.StonegateTrapTriggered)
			require.Equal(t, test.wantTrap, state.PartyState.IsAllPartyDead())
			require.Equal(t, []int{1}, callbacks.TimeAdvanced())
			require.Equal(t, references.FloorNumber(0), state.MapState.PlayerLocation.Floor)
		})
	}
}

func TestStonegateRejectedReviewDoesNotDamageOrConsumeRandomness(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Stonegate).
		WithPlayerAt(15, 16).WithSystemCallbacks().Build()
	review, err := state.ReviewStonegateTrap()
	require.NoError(t, err)
	unit := (*state.CurrentNPCAIController.GetNpcs())[0]
	unit.SetPos(references.Position{X: 1, Y: 1})
	state.SetRandomSeed(701)
	_, err = state.CommitStonegateTrap(review.Fingerprint)
	require.ErrorIs(t, err, ErrStonegateTrapStale)
	require.Zero(t, state.GetRNG().Calls())
	require.Equal(t, review.Members, state.activePartyWipeMembers())
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle, state.MapState.PlayerLocation.Position, 0)
	_, err = state.CommitStonegateTrap(review.Fingerprint)
	require.ErrorIs(t, err, ErrStonegateTrapInvalid)
	require.Zero(t, state.GetRNG().Calls())
	require.Equal(t, review.Members, state.activePartyWipeMembers())
}

func TestStonegateTownCheckpointWakesBeforeIndependentDamage(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Stonegate).
		WithPlayerAt(15, 16).WithSystemCallbacks().Build()
	for slot := range party_state.MaxActiveParty {
		state.PartyState.GetCharacter(slot).PartyMap = party_state.PartyMapUnrecruited
	}
	sleeper := state.PartyState.GetCharacter(0)
	sleeper.PartyMap = party_state.PartyMapActive
	sleeper.SetMaxHp(100)
	sleeper.SetHp(100)
	sleeper.SetStatus(party_state.Sleep)
	state.SetRandomSeed(701)
	woke := state.OneInXOdds(AutoWakeOdds)
	damage := state.RandomIntInRange(1, 8)
	state.SetRandomSeed(701)
	var committed StonegateTrapOutcome
	state.SystemCallbacks.Flow.Consequences.TriggerStonegateTrap = func(outcome StonegateTrapOutcome) { committed = outcome }
	state.CompleteSurfacePassCycle(references.SmallMapType)
	require.Len(t, committed.InitialDamage, 1)
	wantStatus := party_state.Sleep
	if woke {
		wantStatus = party_state.Good
	}
	require.Equal(t, wantStatus, committed.InitialDamage[0].Before.Status)
	require.Equal(t, uint16(100-damage), committed.InitialDamage[0].After.HP)
	require.Equal(t, uint64(2), state.GetRNG().Calls())
}
