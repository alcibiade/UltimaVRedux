package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteJailSentenceTimePolicies(t *testing.T) {
	tests := []struct {
		name            string
		hour            uint8
		minute          uint8
		activate        func(*GameState)
		wantElapsed     int
		wantMinute      uint8
		wantActiveSpell DurationEffectType
	}{
		{name: "normal step", hour: 7, minute: 55, wantElapsed: 20, wantMinute: 15},
		{
			name:   "Quickness halves source step",
			hour:   7,
			minute: 55,
			activate: func(state *GameState) {
				require.True(t, state.ActivateQuicknessEffect(20))
			},
			wantElapsed:     10,
			wantMinute:      5,
			wantActiveSpell: DurationEffectQuickness,
		},
		{
			name:   "Time Stop bypasses source lockup",
			hour:   7,
			minute: 55,
			activate: func(state *GameState) {
				require.True(t, state.ActivateTimeStopEffect(20))
			},
			wantElapsed:     20,
			wantMinute:      15,
			wantActiveSpell: DurationEffectTimeStop,
		},
		{name: "already release hour", hour: 8, minute: 37, wantElapsed: 0, wantMinute: 37},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(references.Britain).
				WithSystemCallbacks().
				Build()
			state.DateTime.Hour = test.hour
			state.DateTime.Minute = test.minute
			state.PartyState.Inventory.Provisions.Food.Set(100)
			state.PartyState.Inventory.Provisions.Keys.Set(7)
			state.PartyState.Inventory.Provisions.SkullKeys.Set(3)
			state.SetCriminalStatus(true)
			state.smashed = 4
			if test.activate != nil {
				test.activate(state)
			}

			elapsed := state.ExecuteJailSentence()

			assert.Equal(t, test.wantElapsed, elapsed)
			assert.Equal(t, uint8(references.JailReleaseHour), state.DateTime.Hour)
			assert.Equal(t, test.wantMinute, state.DateTime.Minute)
			assert.Equal(t, references.Yew, state.MapState.PlayerLocation.Location)
			assert.Equal(t, references.Position{X: references.JailPositionX, Y: references.JailPositionY},
				state.MapState.PlayerLocation.Position)
			assert.Equal(t, references.JailFloorNumber, state.MapState.PlayerLocation.Floor)
			assert.Zero(t, state.PartyState.Inventory.Provisions.Keys.Get())
			assert.Equal(t, uint16(3), state.PartyState.Inventory.Provisions.SkullKeys.Get())
			assert.Equal(t, uint16(100), state.PartyState.Inventory.Provisions.Food.Get())
			assert.False(t, state.IsCriminal())
			assert.Zero(t, state.smashed)
			assert.NotNil(t, state.CurrentNPCAIController)
			if test.wantActiveSpell != DurationEffectNone {
				assert.True(t, state.HasActiveDurationEffect(test.wantActiveSpell))
			}

			worldPosition, worldFloor := state.GameReferences.LocationReferences.WorldLocations.
				GetWorldLocation(references.Yew)
			assert.Equal(t, worldPosition, state.LastLargeMapPosition)
			assert.Equal(t, worldFloor, state.LastLargeMapFloor)
		})
	}
}

func TestExecuteJailSentenceClearsSourceItemsAndSurvivesNativeHydration(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	state.DateTime.Hour = 8
	itemPosition := references.Position{X: 4, Y: 5}
	require.NoError(t, state.ItemStacksMap.Push(&itemPosition, &references.ItemStack{
		Items: references.ItemStacks{{Item: references.Gold, Quantity: 9}},
	}))

	state.ExecuteJailSentence()

	assert.False(t, state.ItemStacksMap.HasItemStackAtPosition(&itemPosition))
	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	hydrated, err := NewGameStateFromSaveSnapshot(
		snapshot,
		state.GameConfig,
		state.GameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)
	assert.Equal(t, state.MapState.PlayerLocation, hydrated.MapState.PlayerLocation)
	assert.Equal(t, state.LastLargeMapPosition, hydrated.LastLargeMapPosition)
	assert.Equal(t, state.LastLargeMapFloor, hydrated.LastLargeMapFloor)
	assert.Zero(t, hydrated.PartyState.Inventory.Provisions.Keys.Get())
}

func TestExecuteJailSentenceUsesYewForMidnightShadowlordMovement(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(310).
		Build()
	state.ShadowlordState.Reset()
	state.DateTime.Hour = 23
	state.DateTime.Minute = 55

	elapsed := state.ExecuteJailSentence()

	assert.Equal(t, 500, elapsed)
	assert.NotContains(t, state.ShadowlordState.TownAssignments, references.Yew,
		"arrest() changes onmap to Yew before sentence time, so midnight cannot assign a Shadowlord there")
	assert.Equal(t, uint8(references.JailReleaseHour), state.DateTime.Hour)
	assert.Equal(t, uint8(15), state.DateTime.Minute)
}

func TestFullLocationTransitionClearsLocationScopedCriminalStatus(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	state.SetCriminalStatus(true)

	state.TransitionToLocation(
		references.Britain,
		state.MapState.PlayerLocation.Floor,
		references.Position{X: 1, Y: 1},
		TransitionFullLocation,
		false,
	)
	assert.True(t, state.IsCriminal())

	state.TransitionToLocation(
		references.Yew,
		0,
		references.Position{X: 10, Y: 10},
		TransitionFullLocation,
		false,
	)
	assert.False(t, state.IsCriminal())
}
