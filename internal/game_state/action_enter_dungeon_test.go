package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestDungeonEntrySetupMatchesOriginalSurfaceUnderworldAndDoomRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		location      references.Location
		sourceFloor   references.FloorNumber
		wantFloor     references.FloorNumber
		wantPosition  references.Position
		wantFacing    references.DungeonFacing
		wantDirection references.DungeonLastDirection
	}{
		{
			name: "surface Deceit", location: references.Deceit, sourceFloor: 0,
			wantFloor: 0, wantPosition: references.Position{X: 1, Y: 1},
			wantFacing: references.DungeonFacingEast, wantDirection: references.DungeonArrivedFromAbove,
		},
		{
			name: "underworld Deceit", location: references.Deceit, sourceFloor: -1,
			wantFloor: 7, wantPosition: references.Position{X: 7, Y: 7},
			wantFacing: references.DungeonFacingWest, wantDirection: references.DungeonArrivedFromBelow,
		},
		{
			name: "underworld Doom descends", location: references.Doom, sourceFloor: -1,
			wantFloor: 0, wantPosition: references.Position{X: 1, Y: 1},
			wantFacing: references.DungeonFacingEast, wantDirection: references.DungeonArrivedFromAbove,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			floor, position, facing, lastDirection := DungeonEntrySetup(test.location, test.sourceFloor)

			assert.Equal(t, test.wantFloor, floor)
			assert.Equal(t, test.wantPosition, position)
			assert.Equal(t, test.wantFacing, facing)
			assert.Equal(t, test.wantDirection, lastDirection)
		})
	}
}

func TestActionEnterLargeMapAdmitsOpenedDungeonRoutesWithoutOutdoorTime(t *testing.T) {
	tests := []struct {
		name              string
		dungeon           references.Location
		sourceFloor       references.FloorNumber
		defeatShadowlords bool
		wantFloor         references.FloorNumber
		wantPosition      references.Position
		wantFacing        references.DungeonFacing
		wantDirection     references.DungeonLastDirection
	}{
		{
			name: "surface cave", dungeon: references.Deceit, sourceFloor: 0,
			wantFloor: 0, wantPosition: references.Position{X: 1, Y: 1},
			wantFacing: references.DungeonFacingEast, wantDirection: references.DungeonArrivedFromAbove,
		},
		{
			name: "surface mine", dungeon: references.Despise, sourceFloor: 0,
			wantFloor: 0, wantPosition: references.Position{X: 1, Y: 1},
			wantFacing: references.DungeonFacingEast, wantDirection: references.DungeonArrivedFromAbove,
		},
		{
			name: "Underworld entrance", dungeon: references.Deceit, sourceFloor: -1,
			wantFloor: 7, wantPosition: references.Position{X: 7, Y: 7},
			wantFacing: references.DungeonFacingWest, wantDirection: references.DungeonArrivedFromBelow,
		},
		{
			name: "Doom after Shadowlords", dungeon: references.Doom, sourceFloor: -1,
			defeatShadowlords: true,
			wantFloor:         0, wantPosition: references.Position{X: 1, Y: 1},
			wantFacing: references.DungeonFacingEast, wantDirection: references.DungeonArrivedFromAbove,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithFloor(test.sourceFloor).
				WithSystemCallbacks().
				Build()
			entrance := state.GetDungeonEntranceCoordinates()[test.dungeon]
			state.MapState.PlayerLocation.Position = entrance
			state.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
			state.SetDungeonState(test.dungeon, true)
			if test.defeatShadowlords {
				for lord := range references.ShadowlordCount {
					state.ShadowlordState.Kill(references.Shadowlord(lord))
				}
			}
			state.MapState.Lighting.RestoreTorchMinutes(41)
			state.MagicLightMinutesRemaining = 23
			state.MapState.Lighting.SetMagicLight(true)
			callbacks.Reset()

			require.True(t, state.ActionEnterLargeMap())
			assert.True(t, callbacks.InputBlocked)
			runDungeonWorkflow(t, state, "enter_location")

			assert.Equal(t, references.PlayerLocation{
				Location: test.dungeon,
				Floor:    test.wantFloor,
				Position: test.wantPosition,
			}, state.MapState.PlayerLocation)
			snapshot, active := state.DungeonStateSnapshot()
			require.True(t, active)
			assert.Equal(t, test.dungeon, snapshot.Location)
			assert.Equal(t, test.wantFacing, snapshot.Facing)
			assert.Equal(t, test.wantDirection, snapshot.LastDirection)
			assert.Nil(t, state.CurrentNPCAIController)
			assert.Equal(t, 41, state.MapState.Lighting.TorchMinutesRemaining())
			assert.Equal(t, 23, state.MagicLightMinutesRemaining)
			assert.Empty(t, callbacks.TimeAdvanced())
			if test.dungeon == references.Doom {
				assert.Empty(t, state.journalDungeons(), "Doom entry still supplies no observation")
			}
			assert.False(t, callbacks.InputBlocked)
			assert.Equal(t, 1, callbacks.BlockInputCalls)
			assert.Equal(t, 1, callbacks.UnblockInputCalls)
			require.NotEmpty(t, callbacks.MapFadeLevels)
			assert.Zero(t, callbacks.MapFadeLevels[len(callbacks.MapFadeLevels)-1])
		})
	}
}

func TestResolveEnterLargeMapDistinguishesAcceptedDoomTransition(t *testing.T) {
	state, _, _, _ := newDoomEntranceEncounterFixture(t, 0)

	outcome := state.ResolveEnterLargeMap()

	assert.Equal(t, EnterLargeMapDoomTransition, outcome)
	assert.True(t, outcome.Accepted())
}

func TestDungeonEntryGeneratesExactlyOneFreshWanderer(t *testing.T) {
	const seed = 265
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.FloorNumber(references.OVERWORLD)).
		WithSystemCallbacks().
		Build()
	entrance := state.GetDungeonEntranceCoordinates()[references.Deceit]
	state.MapState.PlayerLocation.Position = entrance
	state.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
	state.SetDungeonState(references.Deceit, true)
	state.SetRandomSeed(seed)

	expected, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithFloor(0).
		WithPlayerAt(1, 1).
		WithRandomSeed(seed).
		WithSystemCallbacks().
		Build()
	expected.generateDungeonWanderer()
	require.NotNil(t, expected.activeDungeon.wanderer)

	require.True(t, state.ActionEnterLargeMap())
	runDungeonWorkflow(t, state, "enter_location")

	actual, active := state.DungeonStateSnapshot()
	require.True(t, active)
	want, active := expected.DungeonStateSnapshot()
	require.True(t, active)
	assert.Equal(t, want.Wanderer, actual.Wanderer)
	assert.Equal(t, expected.GetRNG().Calls(), state.GetRNG().Calls())
}

func TestActionEnterLargeMapRejectsInvalidDungeonAdmissionWithoutMutation(t *testing.T) {
	tests := []struct {
		name        string
		dungeon     references.Location
		floor       references.FloorNumber
		open        bool
		vehicle     references.VehicleType
		wantMessage string
		wantCurrent string
	}{
		{
			name: "closed entrance", dungeon: references.Deceit, floor: 0,
			wantCurrent: "What?",
		},
		{
			name: "closed Underworld entrance", dungeon: references.Deceit, floor: -1,
			wantCurrent: "What?",
		},
		{
			name: "not on foot", dungeon: references.Deceit, floor: 0, open: true,
			vehicle: references.HorseVehicle, wantMessage: "On foot!",
		},
		{
			name: "Doom on wrong plane", dungeon: references.Doom, floor: 0, open: true,
			wantCurrent: "What?",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithFloor(test.floor).
				WithSystemCallbacks().
				Build()
			entrance := state.GetDungeonEntranceCoordinates()[test.dungeon]
			state.MapState.PlayerLocation.Position = entrance
			state.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
			if test.vehicle != references.NoPartyVehicle {
				npcReference := references.NPCReference{}
				npcReference.NpcType = references.Vehicle
				state.PartyVehicle = *map_units.NewNPCFriendlyVehicle(test.vehicle, npcReference)
			}
			state.SetDungeonState(test.dungeon, test.open)
			before := state.MapState.PlayerLocation
			callbacks.Reset()

			assert.False(t, state.ActionEnterLargeMap())

			assert.Equal(t, before, state.MapState.PlayerLocation)
			_, active := state.DungeonStateSnapshot()
			assert.False(t, active)
			assert.False(t, state.EventQueueManager.HasWorkflow("enter_location"))
			assert.Empty(t, state.journalDungeons(), "rejected entry grants no knowledge")
			assert.Empty(t, callbacks.TimeAdvanced())
			assert.False(t, callbacks.InputBlocked)
			if test.wantMessage != "" {
				assert.Contains(t, callbacks.Messages, test.wantMessage)
			}
			if test.wantCurrent != "" {
				assert.Equal(t, test.wantCurrent, callbacks.CurrentRowContent)
			}
		})
	}
}

func TestDungeonEntrySurveyCommitsCapturedSourceOnlyAfterTransition(t *testing.T) {
	for _, eligibleAtStart := range []bool{false, true} {
		name := "ineligible source"
		if eligibleAtStart {
			name = "eligible source before dawn"
		}
		t.Run(name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
				WithFloor(0).WithRandomSeed(42).WithSystemCallbacks().Build()
			entrance := state.GetDungeonEntranceCoordinates()[references.Deceit]
			state.MapState.PlayerLocation.Position = entrance
			state.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
			state.SetDungeonState(references.Deceit, true)
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
			state.DateTime.Hour, state.DateTime.Minute = 5, 59
			// Establish an unmeasured entrance before the entry revisit.
			state.observeDungeonEntrance(references.Deceit, entrance, state.sampleDungeonObservation(), false)
			if eligibleAtStart {
				state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
			}
			callbacks.Reset()
			require.True(t, state.ActionEnterLargeMap())
			require.False(t, state.journalDungeons()[0].Sites[0].Survey.Measured)
			// Change eligibility during the fade: only the accepted source matters.
			if eligibleAtStart {
				state.DateTime.Hour, state.DateTime.Minute = 6, 0
				state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
			} else {
				state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
			}
			callbacks.Reset() // TimeAdvanced measures the clock, including our explicit fixture change.
			runDungeonWorkflow(t, state, "enter_location")
			require.Equal(t, references.Deceit, state.MapState.PlayerLocation.Location)
			site := state.journalDungeons()[0].Sites[0]
			require.True(t, site.Inspected)
			require.Equal(t, eligibleAtStart, site.Survey.Measured)
			if eligibleAtStart {
				require.Equal(t, journal.SiteLocation{Map: references.BritanniaUnderworld, X: int(entrance.X), Y: int(entrance.Y)}, site.Survey.Location)
				require.Equal(t, 5, site.Survey.RecordedAt.Time.Hour)
				require.Equal(t, 59, site.Survey.RecordedAt.Time.Minute)
			}
			require.Empty(t, callbacks.TimeAdvanced(), "no extra outdoor entry cycle")
		})
	}
}

func TestDungeonJournalDoomInterceptionIsNotAnEntryObservation(t *testing.T) {
	state, _, _, _ := newDoomEntranceEncounterFixture(t, 1)
	require.True(t, state.ActionEnterLargeMap())
	require.NotNil(t, state.CombatScene)
	require.Empty(t, state.journalDungeons())
}

func TestDungeonEntrySurveyCancellationBoundary(t *testing.T) {
	for _, afterTransition := range []bool{false, true} {
		name := "before transition"
		if afterTransition {
			name = "after transition"
		}
		t.Run(name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
				WithFloor(0).WithRandomSeed(42).WithSystemCallbacks().Build()
			state.MapState.PlayerLocation.Position = state.GetDungeonEntranceCoordinates()[references.Deceit]
			state.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
			state.SetDungeonState(references.Deceit, true)
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
			state.DateTime.Hour = 19
			require.True(t, state.ActionEnterLargeMap())
			if afterTransition {
				for range 30 {
					state.EventQueueManager.Update()
					state.ElapsedMs += 100
					if state.MapState.PlayerLocation.Location == references.Deceit {
						break
					}
				}
				require.Equal(t, references.Deceit, state.MapState.PlayerLocation.Location)
				require.True(t, state.EventQueueManager.HasWorkflow("enter_location"), "cancel during fade-in")
			}
			state.EventQueueManager.Shutdown()
			if afterTransition {
				require.True(t, state.journalDungeons()[0].Sites[0].Survey.Measured)
			} else {
				require.Empty(t, state.journalDungeons())
			}
			require.False(t, callbacks.InputBlocked)
		})
	}
}

func runDungeonWorkflow(t *testing.T, state *GameState, name string) {
	t.Helper()
	for range 200 {
		state.EventQueueManager.Update()
		state.ElapsedMs += 1000
		if !state.EventQueueManager.HasWorkflow(name) {
			return
		}
	}
	require.False(t, state.EventQueueManager.HasWorkflow(name))
}
