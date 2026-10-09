package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMoongateTravelRouteShrineOfSpirituality(t *testing.T) {
	gs := &GameState{
		DateTime: datetime.UltimaDate{
			Year:   1,
			Month:  1,
			Day:    1,
			Hour:   0, // Midnight
			Minute: 5, // Within the 10-minute window
		},
	}
	gs.moonstoneSystem = NewMoonstoneSystem()

	route, err := gs.GetMoongateTravelRoute()

	assert.NoError(t, err)
	assert.Equal(t, MoongateTravelToSpirituality, route.Kind)
	assert.Equal(t, SpiritualityShrineIndex, route.ShrineIndex)
	assert.Equal(t, MoongateDestination{}, route.Destination, "Spirituality must not carry world coordinates")
}

// TestGetMoongateDestination_ShrineTimeWindow tests the shrine time window boundaries
func TestGetMoongateDestination_ShrineTimeWindow(t *testing.T) {
	testCases := []struct {
		name         string
		hour         uint8
		minute       uint8
		expectShrine bool
	}{
		{"Exactly midnight", 0, 0, true},
		{"5 minutes after midnight", 0, 5, true},
		{"9 minutes after midnight", 0, 9, true},
		{"10 minutes after midnight", 0, 10, false},
		{"15 minutes after midnight", 0, 15, false},
		{"11:59 PM", 23, 59, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs := &GameState{
				DateTime: datetime.UltimaDate{
					Year:   1,
					Month:  1,
					Day:    1,
					Hour:   tc.hour,
					Minute: tc.minute,
				},
			}
			gs.moonstoneSystem = NewMoonstoneSystem()

			route, err := gs.GetMoongateTravelRoute()

			if tc.expectShrine {
				assert.NoError(t, err, "Should not error for shrine time")
				assert.Equal(t, MoongateTravelToSpirituality, route.Kind)
			} else {
				// Should get normal moonstone destination, not shrine
				if err == nil {
					assert.Equal(t, MoongateTravelToMoonstone, route.Kind)
				}
			}
		})
	}
}

// TestGetMoongateDestination_NormalMoongate tests regular moongate travel
func TestGetMoongateDestination_NormalMoongate(t *testing.T) {
	// Set up game state with known moon phase
	gs := &GameState{
		DateTime: datetime.UltimaDate{
			Year:   1,
			Month:  1,
			Day:    2, // Day 2 should give us phase 1 (Crescent Waxing)
			Hour:   11,
			Minute: 0,
		},
	}
	gs.moonstoneSystem = NewMoonstoneSystem()

	route, err := gs.GetMoongateTravelRoute()

	assert.NoError(t, err, "Should not error for normal moongate")
	assert.Equal(t, MoongateTravelToMoonstone, route.Kind)
	destination := route.Destination

	// Should target moonstone 1 (phase 1 maps to moonstone 1)
	expectedMoonstone := gs.moonstoneSystem.Moonstones[1]
	assert.Equal(t, expectedMoonstone.X, destination.X, "Should match moonstone 1 X coordinate")
	assert.Equal(t, expectedMoonstone.Y, destination.Y, "Should match moonstone 1 Y coordinate")
	assert.Equal(t, expectedMoonstone.Map, destination.Map, "Should match moonstone 1 map")
	assert.Equal(t, expectedMoonstone.Level, destination.Level, "Should match moonstone 1 level")
}

// TestGetMoongateDestination_InvalidMoonstone tests error handling
func TestGetMoongateDestination_InvalidMoonstone(t *testing.T) {
	gs := &GameState{
		DateTime: datetime.UltimaDate{
			Year:   1,
			Month:  1,
			Day:    2,
			Hour:   11,
			Minute: 0,
		},
	}
	gs.moonstoneSystem = NewMoonstoneSystem()

	// Mark moonstone 1 as found and in inventory
	gs.moonstoneSystem.Moonstones[1].State = MoonstoneStateInInventory
	gs.moonstoneSystem.Moonstones[1].Map = 0xFF
	gs.moonstoneSystem.Moonstones[1].IsFound = true

	_, err := gs.GetMoongateTravelRoute()

	assert.Error(t, err, "Should error when moonstone is in inventory")
	assert.Contains(t, err.Error(), "moonstone 1 is in inventory", "Should indicate specific problem")
}

// TestAreMoongatesVisible tests moongate visibility by time
func TestAreMoongatesVisible(t *testing.T) {
	testCases := []struct {
		name    string
		hour    uint8
		visible bool
	}{
		{"Midnight", 0, true},
		{"3 AM", 3, true},
		{"5 AM", 5, false}, // Day time - not visible
		{"8 AM", 8, false}, // Day time - not visible
		{"Noon", 12, false},
		{"6 PM", 18, false},
		{"8 PM", 20, true}, // Night time - visible
		{"10 PM", 22, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs := &GameState{
				DateTime: datetime.UltimaDate{
					Hour: tc.hour,
				},
			}
			gs.initializeMoongateStage()

			visible := gs.AreMoongatesVisible()
			assert.Equal(t, tc.visible, visible, "Visibility should match expected")
		})
	}
}

// TestGetMoongateVisibilityLevel tests visibility level calculation
func TestGetMoongateVisibilityLevel(t *testing.T) {
	testCases := []struct {
		name          string
		hour          uint8
		minute        uint8
		expectedLevel float32
		tolerance     float32
	}{
		{"Midnight full visibility", 0, 0, 1.0, 0.0},
		{"3 AM full visibility", 3, 0, 1.0, 0.0},
		{"5 AM invisible", 5, 0, 0.0, 0.0},
		{"5:30 AM invisible", 5, 30, 0.0, 0.0},
		{"5:59 AM invisible", 5, 59, 0.0, 0.0},
		{"Noon invisible", 12, 0, 0.0, 0.0},
		{"6 PM invisible", 18, 0, 0.0, 0.0},
		{"7 PM invisible", 19, 0, 0.0, 0.0},
		{"7:30 PM invisible", 19, 30, 0.0, 0.0},
		{"7:59 PM invisible", 19, 59, 0.0, 0.0},
		{"8 PM full brightness", 20, 0, 1.0, 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs := &GameState{
				DateTime: datetime.UltimaDate{
					Hour:   tc.hour,
					Minute: tc.minute,
				},
			}
			gs.initializeMoongateStage()

			level := gs.GetMoongateVisibilityLevel()
			if tc.tolerance > 0 {
				assert.InDelta(t, tc.expectedLevel, level, float64(tc.tolerance), "Visibility level should be within tolerance")
			} else {
				assert.Equal(t, tc.expectedLevel, level, "Visibility level should match exactly")
			}
		})
	}
}

// TestMoongateDestination_AllPhases tests that all moon phases produce valid destinations
func TestMoongateDestination_AllPhases(t *testing.T) {
	for phase := 0; phase < 8; phase++ {
		t.Run(fmt.Sprintf("Phase_%d", phase), func(t *testing.T) {
			// Calculate day that produces this phase
			// Using the pattern [0,1,1,2,2,3,3,4,5,5,6,6,7,7,0,1,1,2,2,3,3,4,5,5,6,6,7,7]
			var targetDay uint8
			for day := uint8(1); day <= 28; day++ {
				totalDays := int(day - 1)
				moonPhasePattern := []int{0, 1, 1, 2, 2, 3, 3, 4, 5, 5, 6, 6, 7, 7, 0, 1, 1, 2, 2, 3, 3, 4, 5, 5, 6, 6, 7, 7}
				cycleDay := totalDays % 28
				if moonPhasePattern[cycleDay] == phase {
					targetDay = day
					break
				}
			}

			gs := &GameState{
				DateTime: datetime.UltimaDate{
					Year:   1,
					Month:  1,
					Day:    targetDay,
					Hour:   11,
					Minute: 0,
				},
			}
			gs.moonstoneSystem = NewMoonstoneSystem()

			route, err := gs.GetMoongateTravelRoute()

			assert.NoError(t, err, "Should not error for phase %d", phase)
			assert.Equal(t, MoongateTravelToMoonstone, route.Kind)
			destination := route.Destination

			// Verify destination matches the expected moonstone
			expectedMoonstone := gs.moonstoneSystem.Moonstones[phase]
			assert.Equal(t, expectedMoonstone.X, destination.X, "Phase %d should target correct moonstone", phase)
			assert.Equal(t, expectedMoonstone.Y, destination.Y, "Phase %d should target correct moonstone", phase)
		})
	}
}

func TestMoongateRoutingUsesTrammelBeforeNoonAndFeluccaAtNoon(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(224, 133).
		WithSystemCallbacks().
		Build()
	gs.DateTime = datetime.UltimaDate{Year: 1, Month: 1, Day: 5, Hour: 11, Minute: 59}
	beforeCalls := gs.rng.Calls()
	beforeDate := gs.DateTime

	morningRoute, err := gs.GetMoongateTravelRoute()
	assert.NoError(t, err)
	morning := morningRoute.Destination
	assert.Equal(t, gs.moonstoneSystem.Moonstones[2].X, morning.X)
	assert.Equal(t, beforeCalls, gs.rng.Calls())
	assert.Equal(t, beforeDate, gs.DateTime)
	assert.Empty(t, callbacks.TimeAdvanced())
	assert.Empty(t, callbacks.Messages)
	assert.Empty(t, callbacks.SoundEffectsPlayed)

	gs.DateTime.Hour = 12
	gs.DateTime.Minute = 0
	noonDate := gs.DateTime
	noonRoute, err := gs.GetMoongateTravelRoute()
	assert.NoError(t, err)
	noon := noonRoute.Destination
	assert.Equal(t, gs.moonstoneSystem.Moonstones[3].X, noon.X)
	assert.NotEqual(t, morning.X, noon.X)
	assert.Equal(t, beforeCalls, gs.rng.Calls())
	assert.Equal(t, noonDate, gs.DateTime)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestSpiritualityRouteDoesNotRequirePhaseDestinationMoonstone(t *testing.T) {
	for _, test := range []struct {
		name      string
		minute    uint8
		inventory bool
	}{
		{"00:09/available", 9, false},
		{"00:09/inventory", 9, true},
		{"00:10/available", 10, false},
		{"00:10/inventory", 10, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			gs := newMoongateStageTestState(t, 0, MoongateStageFull)
			callbacks := NewMockSystemCallbacks(t)
			gs.SystemCallbacks = callbacks.ToSystemCallbacks()
			gs.moonstoneSystem = NewMoonstoneSystem()
			gs.DateTime = datetime.UltimaDate{Year: 1, Month: 1, Day: 1, Hour: 0, Minute: test.minute}
			gs.MapState.PlayerLocation.Location = references.BritanniaUnderworld
			phaseIndex := gs.DateTime.GetMoongateMoonPhase().ToMoonstoneIndex()
			source := gs.moonstoneSystem.GetMoonstoneByIndex((phaseIndex + 1) % 8)
			gs.MapState.PlayerLocation.Position = references.Position{
				X: references.Coordinate(source.X), Y: references.Coordinate(source.Y),
			}
			if test.inventory {
				gs.moonstoneSystem.Moonstones[phaseIndex].State = MoonstoneStateInInventory
				gs.moonstoneSystem.Moonstones[phaseIndex].Map = 0xFF
			}
			require.True(t, gs.IsPlayerOnActiveMoongate())
			gs.SetRandomSeed(501)
			beforeCalls, beforeDate, beforeLocation := gs.rng.Calls(), gs.DateTime, gs.MapState.PlayerLocation
			beforeStage := gs.MoongateStage()
			callbacks.BindGameState(gs)

			route, err := gs.GetMoongateTravelRoute()
			result := gs.ReviewMoongateTravel()

			if test.minute == 10 && test.inventory {
				require.ErrorContains(t, err, fmt.Sprintf("moonstone %d is in inventory", phaseIndex))
				assert.False(t, result.Success)
				assert.True(t, result.FailedButActivated)
			} else {
				require.NoError(t, err)
				assert.True(t, result.Success)
				assert.False(t, result.FailedButActivated)
				assert.Equal(t, route, result.Route)
				if test.minute == 9 {
					assert.Equal(t, MoongateTravelRoute{
						Kind: MoongateTravelToSpirituality, ShrineIndex: SpiritualityShrineIndex,
					}, route)
				} else {
					assert.Equal(t, MoongateTravelToMoonstone, route.Kind)
					destination, destinationErr := gs.GetMoongateDestinationForPhase(gs.DateTime.GetMoongateMoonPhase())
					require.NoError(t, destinationErr)
					assert.Equal(t, destination, route.Destination)
				}
			}
			assert.Equal(t, beforeCalls, gs.rng.Calls())
			assert.Equal(t, beforeDate, gs.DateTime)
			assert.Equal(t, beforeLocation, gs.MapState.PlayerLocation)
			assert.Equal(t, beforeStage, gs.MoongateStage())
			assert.Nil(t, gs.ShrineScene)
			assert.Empty(t, callbacks.TimeAdvanced())
			assert.Zero(t, callbacks.MessageCallCount)
			assert.Zero(t, callbacks.AudioCallCount)
			assert.Zero(t, callbacks.VisualCallCount)
			assert.Zero(t, callbacks.ScreenCallCount)
			assert.Zero(t, callbacks.FlowCallCount)
			assert.Zero(t, callbacks.BlockInputCalls)
			assert.False(t, gs.EventQueueManager.HasActiveWorkflows())
		})
	}
}
