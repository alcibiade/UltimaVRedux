package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckHourlyFoodConsumption_NoHourChange verifies no additional
// food consumption without a newly observed hour boundary. ref:PROV-0002
func TestCheckHourlyFoodConsumption_NoHourChange(t *testing.T) {
	builder := NewIntegrationTestBuilder(t).
		WithSystemCallbacks().
		WithRandomSeed(12345)
	gs, mock := builder.Build()

	// Set up initial state
	gs.DateTime.Hour = 6 // Meal time
	gs.lastProcessedHour = 6
	gs.PartyState.Inventory.Provisions.Food.Set(100)

	initialFood := gs.PartyState.Inventory.Provisions.Food.Get()

	// Call checkHourlyFoodConsumption
	gs.checkHourlyFoodConsumption()

	// Verify no food was consumed (hour didn't change)
	assert.Equal(t, initialFood, gs.PartyState.Inventory.Provisions.Food.Get(),
		"Food should not be consumed when hour hasn't changed")

	// Verify no messages were added
	assert.Empty(t, mock.Messages, "No messages should be displayed when hour hasn't changed")
}

// TestCheckHourlyFoodConsumption_MealTime_WithFood verifies that food is consumed
// at meal times (6, 12, 18) when the party has food.
// Original: original party-effect rules party status update food consumption logic.
// See ref:TIME-0002.
func TestCheckHourlyFoodConsumption_MealTime_WithFood(t *testing.T) {
	mealTimes := []int{6, 12, 18}

	for _, mealHour := range mealTimes {
		t.Run("MealTime"+string(rune('0'+mealHour/6)), func(t *testing.T) {
			builder := NewIntegrationTestBuilder(t).
				WithSystemCallbacks().
				WithRandomSeed(12345)
			gs, mock := builder.Build()

			// Set up initial state - previous hour was before meal time
			gs.DateTime.Hour = byte(mealHour)
			gs.lastProcessedHour = mealHour - 1
			gs.PartyState.Inventory.Provisions.Food.Set(100)

			activeMembers := gs.PartyState.GetActivePartyCount()
			initialFood := gs.PartyState.Inventory.Provisions.Food.Get()

			// Call checkHourlyFoodConsumption
			gs.checkHourlyFoodConsumption()

			// Verify food was consumed (1 per active member)
			expectedFood := initialFood - uint16(activeMembers)
			assert.Equal(t, expectedFood, gs.PartyState.Inventory.Provisions.Food.Get(),
				"Food should be consumed at meal time hour %d", mealHour)

			// Verify lastProcessedHour was updated
			assert.Equal(t, mealHour, gs.lastProcessedHour,
				"lastProcessedHour should be updated")
			assert.Equal(t, []TransientIconKind{TransientIconEating}, mock.TransientIconCalls)

			// No starvation message when food is available
			for _, msg := range mock.Messages {
				assert.NotContains(t, msg, "Starving",
					"Should not show starvation message when food is available")
			}
		})
	}
}

func TestProcessBedRestProvisionHourReportsActualMealAndStarvation(t *testing.T) {
	gs, mock := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(12345).
		Build()
	gs.PartyState.Inventory.Provisions.Food.Set(100)
	foodBefore := gs.PartyState.Inventory.Provisions.Food.Get()

	meal := gs.ProcessBedRestProvisionHour(6)

	assert.False(t, meal.Starving)
	assert.Empty(t, meal.DeadIndexes)
	assert.Less(t, gs.PartyState.Inventory.Provisions.Food.Get(), foodBefore)
	assert.Equal(t, 6, gs.lastProcessedHour)
	assert.Equal(t, []TransientIconKind{TransientIconEating}, mock.TransientIconCalls)

	mock.Reset()
	gs.PartyState.Inventory.Provisions.Food.Set(0)
	starvation := gs.ProcessBedRestProvisionHour(7)

	assert.True(t, starvation.Starving)
	assert.Equal(t, 7, gs.lastProcessedHour)
	assert.Equal(t, []TransientIconKind{TransientIconStarving}, mock.TransientIconCalls)
}

func TestSurfacePlayerUpdateUsesCachedFatalPoisonDinerEligibility(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	for index := range party_state.MaxActiveParty {
		gs.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapUnrecruited
	}
	poisoned := gs.PartyState.GetCharacter(0)
	poisoned.PartyMap = party_state.PartyMapActive
	poisoned.SetStatus(party_state.Poisoned)
	poisoned.SetHp(1)
	survivor := gs.PartyState.GetCharacter(1)
	survivor.PartyMap = party_state.PartyMapActive
	survivor.SetStatus(party_state.Good)
	gs.PartyState.Inventory.Provisions.Food.Set(10)
	gs.DateTime.Hour, gs.DateTime.Minute = 5, 59
	gs.lastProcessedHour = 5

	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)

	assert.True(t, poisoned.IsDead())
	assert.Equal(t, uint16(8), gs.PartyState.Inventory.Provisions.Food.Get(), "fatal poison still counts this update")

	gs.DateTime.Hour, gs.DateTime.Minute = 11, 59
	gs.lastProcessedHour = 11
	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)
	assert.Equal(t, uint16(7), gs.PartyState.Inventory.Provisions.Food.Get(), "the dead member is excluded later")
}

func TestBedRestAttemptsHonorDurationTimeAndHourBucket(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		hour       byte
		minute     byte
		effect     DurationEffect
		wantCycles int
		wantHour   byte
		wantMinute byte
	}{
		{
			name: "time stop ticks while clock stalls", hour: 5, minute: 55,
			effect:     DurationEffect{EffectType: DurationEffectTimeStop, TurnsRemaining: 2},
			wantCycles: 3, wantHour: 6, wantMinute: 5,
		},
		{
			name: "quickness expires before normal advancement", hour: 5, minute: 40,
			effect:     DurationEffect{EffectType: DurationEffectQuickness, TurnsRemaining: 2},
			wantCycles: 3, wantHour: 6, wantMinute: 0,
		},
		{
			name: "midnight ignores starting minute", hour: 23, minute: 55,
			wantCycles: 1, wantHour: 0, wantMinute: 5,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(references.Britain).
				WithPlayerAt(15, 15).
				WithSystemCallbacks().
				Build()
			state.DateTime.Hour, state.DateTime.Minute = testCase.hour, testCase.minute
			state.ActiveDurationEffect = testCase.effect
			rest := state.NewBedRestState(1)
			state.BeginBedRest(rest)
			for int(state.DateTime.Hour) != rest.TargetHour {
				state.ProcessBedRestCycle(rest)
				require.LessOrEqual(t, rest.CurrentCycle, 20)
			}
			state.FinishBedRest(rest)

			assert.Equal(t, testCase.wantCycles, rest.CurrentCycle)
			assert.Equal(t, testCase.wantHour, state.DateTime.Hour)
			assert.Equal(t, testCase.wantMinute, state.DateTime.Minute)
			assert.Equal(t, DurationEffectNone, state.ActiveDurationEffect.EffectType)
		})
	}
}

func TestBedRestPlayerUpdateUsesMixedCachedStatusesAndContinuesAfterDeath(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	for index := range party_state.MaxActiveParty {
		state.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapUnrecruited
	}
	good := state.PartyState.GetCharacter(0)
	good.PartyMap = party_state.PartyMapActive
	good.SetStatus(party_state.Good)
	poisoned := state.PartyState.GetCharacter(1)
	poisoned.PartyMap = party_state.PartyMapActive
	poisoned.SetStatus(party_state.Poisoned)
	poisoned.SetHp(1)
	charmed := state.PartyState.GetCharacter(2)
	charmed.PartyMap = party_state.PartyMapActive
	charmed.SetStatus(party_state.Charmed)
	preexistingSleep := state.PartyState.GetCharacter(3)
	preexistingSleep.PartyMap = party_state.PartyMapActive
	preexistingSleep.SetStatus(party_state.Sleep)
	state.PartyState.Inventory.Provisions.Food.Set(10)
	state.DateTime.Hour, state.DateTime.Minute = 5, 55
	rest := state.NewBedRestState(1)
	state.BeginBedRest(rest)

	first := state.ProcessBedRestCycle(rest)

	assert.Equal(t, []int{1}, first.PoisonDeaths)
	assert.Equal(t, uint16(8), state.PartyState.Inventory.Provisions.Food.Get(), "poisoned and charmed dine; sleepers do not")
	assert.False(t, rest.Interrupted)
	state.ProcessBedRestCycle(rest)
	assert.Equal(t, 2, rest.CurrentCycle, "an individual death does not stop later attempts")
	state.FinishBedRest(rest)
	assert.True(t, good.IsGood())
	assert.True(t, preexistingSleep.IsAsleep(), "cleanup preserves sleep that predates this rest")
}

func TestBedRestMixedStatusMealsAtSourceHours(t *testing.T) {
	for _, mealHour := range []byte{6, 12, 18} {
		t.Run(fmt.Sprintf("hour-%d", mealHour), func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(references.Britain).
				WithPlayerAt(15, 15).
				WithSystemCallbacks().
				Build()
			for index := range party_state.MaxActiveParty {
				state.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapUnrecruited
			}
			for index, status := range []party_state.CharacterStatus{
				party_state.Good,
				party_state.Poisoned,
				party_state.Charmed,
				party_state.Sleep,
				party_state.Dead,
			} {
				character := state.PartyState.GetCharacter(index)
				character.PartyMap = party_state.PartyMapActive
				character.SetStatus(status)
			}
			state.PartyState.Inventory.Provisions.Food.Set(10)
			state.DateTime.Hour, state.DateTime.Minute = mealHour-1, 55
			rest := state.NewBedRestState(1)
			state.BeginBedRest(rest)

			state.ProcessBedRestCycle(rest)

			assert.Equal(t, uint16(8), state.PartyState.Inventory.Provisions.Food.Get(),
				"only cached Poisoned and Charmed members dine while healthy sleepers rest")
		})
	}
}

func TestBedRestTotalPartyDeathContinuesAndCallbackRunsAtFinish(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	for index := range party_state.MaxActiveParty {
		state.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapUnrecruited
	}
	last := state.PartyState.GetCharacter(0)
	last.PartyMap = party_state.PartyMapActive
	last.SetStatus(party_state.Poisoned)
	last.SetHp(1)
	state.PartyState.Inventory.Provisions.Food.Set(100)
	state.DateTime.Hour, state.DateTime.Minute = 5, 40
	deathCalls := 0
	state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() { deathCalls++ }
	rest := state.NewBedRestState(1)
	state.BeginBedRest(rest)

	first := state.ProcessBedRestCycle(rest)
	second := state.ProcessBedRestCycle(rest)

	assert.True(t, first.TotalPartyDeath)
	assert.False(t, second.TotalPartyDeath)
	assert.Equal(t, 2, rest.CurrentCycle, "the source rest loop continues with a dead party")
	assert.Equal(t, 0, deathCalls, "death handling waits until the rest loop exits")
	state.FinishBedRest(rest)
	assert.Equal(t, 1, deathCalls)
}

func TestBedRestContinuesAfterIndividualStarvationDeath(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		WithRandomSeed(12345).
		Build()
	for index := range party_state.MaxActiveParty {
		state.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapUnrecruited
	}
	dying := state.PartyState.GetCharacter(0)
	dying.PartyMap = party_state.PartyMapActive
	dying.SetStatus(party_state.Good)
	dying.SetHp(1)
	survivor := state.PartyState.GetCharacter(1)
	survivor.PartyMap = party_state.PartyMapActive
	survivor.SetStatus(party_state.Good)
	survivor.SetHp(survivor.GetMaxHp())
	state.PartyState.Inventory.Provisions.Food.Set(0)
	state.DateTime.Hour, state.DateTime.Minute = 4, 55
	rest := state.NewBedRestState(2)
	state.BeginBedRest(rest)

	first := state.ProcessBedRestCycle(rest)
	for int(state.DateTime.Hour) != rest.TargetHour {
		state.ProcessBedRestCycle(rest)
	}

	assert.Equal(t, []int{0}, first.Provision.DeadIndexes)
	assert.True(t, dying.IsDead())
	assert.False(t, survivor.IsDead())
	assert.Greater(t, rest.CurrentCycle, 1)
	assert.False(t, rest.Interrupted)
}

func TestAdvanceBedRestMinutesDoesNotQueueProvisionHourReplay(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	gs.DateTime.Hour, gs.DateTime.Minute = 5, 59
	gs.lastProcessedHour = 5

	assert.Equal(t, 1, gs.AdvanceBedRestMinutes(1))

	assert.Empty(t, gs.pendingProvisionHours)
	gs.ProcessBedRestProvisionHour(6)
	foodAfterBedMeal := gs.PartyState.Inventory.Provisions.Food.Get()
	gs.checkHourlyFoodConsumption()
	assert.Equal(t, foodAfterBedMeal, gs.PartyState.Inventory.Provisions.Food.Get())
}

// TestCheckHourlyFoodConsumption_NonMealTime verifies that food is NOT consumed
// at non-meal times (hours other than 6, 12, 18).
func TestCheckHourlyFoodConsumption_NonMealTime(t *testing.T) {
	nonMealTimes := []int{0, 3, 7, 11, 13, 17, 19, 23}

	for _, hour := range nonMealTimes {
		t.Run("NonMealTime"+string(rune('0'+hour%10)), func(t *testing.T) {
			builder := NewIntegrationTestBuilder(t).
				WithSystemCallbacks().
				WithRandomSeed(12345)
			gs, _ := builder.Build()

			// Set up initial state
			gs.DateTime.Hour = byte(hour)
			gs.lastProcessedHour = hour - 1
			if gs.lastProcessedHour < 0 {
				gs.lastProcessedHour = 23
			}
			gs.PartyState.Inventory.Provisions.Food.Set(100)

			initialFood := gs.PartyState.Inventory.Provisions.Food.Get()

			// Call checkHourlyFoodConsumption
			gs.checkHourlyFoodConsumption()

			// Verify no food was consumed at non-meal time
			assert.Equal(t, initialFood, gs.PartyState.Inventory.Provisions.Food.Get(),
				"Food should not be consumed at non-meal time hour %d", hour)
		})
	}
}

// TestCheckHourlyFoodConsumption_Starvation verifies that starvation damage is applied
// when food runs out at meal time.
// Original: original party-effect rules party damage applies random(1,8) damage.
// See ref:TIME-0002.
func TestCheckHourlyFoodConsumption_Starvation(t *testing.T) {
	builder := NewIntegrationTestBuilder(t).
		WithSystemCallbacks().
		WithRandomSeed(12345)
	gs, mock := builder.Build()

	// Set up initial state with NO food
	gs.DateTime.Hour = 6 // Meal time
	gs.lastProcessedHour = 5
	gs.PartyState.Inventory.Provisions.Food.Set(0)

	// Record initial HP for all characters
	initialHPs := make([]uint16, gs.PartyState.GetActivePartyCount())
	for i := range gs.PartyState.GetActivePartyCount() {
		char := gs.PartyState.GetCharacter(i)
		if char != nil {
			initialHPs[i] = char.GetCurrentHp()
		}
	}

	// Call checkHourlyFoodConsumption
	gs.checkHourlyFoodConsumption()

	// Verify "Starving!" message was shown
	foundStarvingMsg := false
	for _, msg := range mock.Messages {
		if msg == "Starving!" {
			foundStarvingMsg = true
			break
		}
	}
	assert.True(t, foundStarvingMsg, "Should display 'Starving!' message when no food")
	assert.Equal(t, []TransientIconKind{TransientIconStarving}, mock.TransientIconCalls)

	// Verify at least one character took damage (random 1-8 per character)
	damageTaken := false
	for i := range gs.PartyState.GetActivePartyCount() {
		char := gs.PartyState.GetCharacter(i)
		if char != nil && char.GetCurrentHp() < initialHPs[i] {
			damageTaken = true
			break
		}
	}
	assert.True(t, damageTaken, "At least one character should take starvation damage")
}

// TestCheckHourlyFoodConsumption_LastProcessedHourUpdates verifies that lastProcessedHour
// is always updated when hour changes, regardless of whether it's a meal time.
func TestCheckHourlyFoodConsumption_LastProcessedHourUpdates(t *testing.T) {
	builder := NewIntegrationTestBuilder(t).
		WithSystemCallbacks().
		WithRandomSeed(12345)
	gs, _ := builder.Build()

	testCases := []struct {
		name        string
		currentHour int
		prevHour    int
	}{
		{"MealTime6", 6, 5},
		{"NonMealTime7", 7, 6},
		{"MealTime12", 12, 11},
		{"NonMealTime15", 15, 14},
		{"MealTime18", 18, 17},
		{"Midnight", 0, 23},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs.DateTime.Hour = byte(tc.currentHour)
			gs.lastProcessedHour = tc.prevHour
			gs.PartyState.Inventory.Provisions.Food.Set(100)

			gs.checkHourlyFoodConsumption()

			assert.Equal(t, tc.currentHour, gs.lastProcessedHour,
				"lastProcessedHour should be updated to current hour")
		})
	}
}

func TestConsumedSmallMapCycleProcessesCrossedHour(t *testing.T) {
	builder := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		WithRandomSeed(12345)
	gs, mock := builder.Build()

	// Set up state just before meal time
	gs.DateTime.Hour = 5
	gs.DateTime.Minute = 59 // Will roll over to 6:00 on next advance
	gs.lastProcessedHour = 5
	gs.PartyState.Inventory.Provisions.Food.Set(0) // No food = starvation

	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)

	// Verify hour rolled over
	require.Equal(t, byte(6), gs.DateTime.Hour, "Hour should have advanced to 6")

	// Verify starvation was processed (we should see "Starving!" message)
	foundStarvingMsg := false
	for _, msg := range mock.Messages {
		if msg == "Starving!" {
			foundStarvingMsg = true
			break
		}
	}
	assert.True(t, foundStarvingMsg,
		"advanceTimeSmallMap should trigger starvation at meal time when no food")
}

func TestConsumedLargeMapCycleProcessesCrossedHour(t *testing.T) {
	builder := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		WithRandomSeed(12345)
	gs, mock := builder.Build()

	// Set up state just before meal time
	gs.DateTime.Hour = 11
	gs.DateTime.Minute = 59 // Will roll over to 12:00 on next advance
	gs.lastProcessedHour = 11
	gs.PartyState.Inventory.Provisions.Food.Set(0) // No food = starvation

	gs.completeSurfaceCycleWithActorPhase(references.LargeMapType, DefaultLargeMapMinutesPerTurn, false)

	// Verify hour rolled over to meal time
	require.Equal(t, byte(12), gs.DateTime.Hour, "Hour should have advanced to 12")

	// Verify starvation was processed
	foundStarvingMsg := false
	for _, msg := range mock.Messages {
		if msg == "Starving!" {
			foundStarvingMsg = true
			break
		}
	}
	assert.True(t, foundStarvingMsg,
		"advanceTimeLargeMap should trigger starvation at meal time when no food")
}

// TestLastProcessedHourInitialization verifies that lastProcessedHour is properly
// initialized when loading a save game.
func TestLastProcessedHourInitialization_Integration(t *testing.T) {
	builder := NewIntegrationTestBuilder(t).
		WithSystemCallbacks()
	gs, _ := builder.Build()

	// After loading, lastProcessedHour should match the saved game's hour
	assert.Equal(t, int(gs.DateTime.Hour), gs.lastProcessedHour,
		"lastProcessedHour should be initialized from DateTime.Hour on load")
}

func TestConsumedSurfaceMealExcludesSleepingAndDeadActiveMembers(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	for index := range gs.PartyState.GetCharacterCount() {
		gs.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapUnrecruited
	}
	statuses := []party_state.CharacterStatus{
		party_state.Good,
		party_state.Poisoned,
		party_state.Sleep,
		party_state.Dead,
	}
	for index, status := range statuses {
		character := gs.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapActive
		character.SetHp(20)
		character.SetStatus(status)
	}
	gs.DateTime.Hour, gs.DateTime.Minute = 5, 59
	gs.lastProcessedHour = 5
	gs.PartyState.Inventory.Provisions.Food.Set(10)

	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)

	assert.Equal(t, uint16(8), gs.PartyState.Inventory.Provisions.Food.Get())
}

func TestConsumedSurfaceStarvationRunsOnNonMealCrossedHour(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(12345).
		Build()
	gs.DateTime.Hour, gs.DateTime.Minute = 6, 59
	gs.lastProcessedHour = 6
	gs.PartyState.Inventory.Provisions.Food.Set(0)

	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)

	assert.Equal(t, 7, gs.lastProcessedHour)
	assert.Contains(t, callbacks.Messages, "Starving!")
}

func TestConsumedSurfaceObservesOnlyLatestTimeRequest(t *testing.T) {
	gs, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(12345).
		Build()
	gs.DateTime.Hour, gs.DateTime.Minute = 4, 59
	gs.lastProcessedHour = 4
	gs.PartyState.Inventory.Provisions.Food.Set(0)
	gs.AdvanceGameplayMinutes(121)

	gs.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)

	assert.Equal(t, 7, gs.lastProcessedHour)
	assert.Empty(t, gs.pendingProvisionHours)
	assert.Equal(t, 0, countEqual(callbacks.Messages, "Starving!"),
		"the following same-hour command replaces the earlier crossing")
	assert.Empty(t, callbacks.TransientIconCalls)
}

func TestSelfContainedGameplayTimeProcessesCrossedProvisionHours(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	state.DateTime.Hour, state.DateTime.Minute = 5, 59
	state.lastProcessedHour = 5
	state.PartyState.Inventory.Provisions.Food.Set(100)
	foodBefore := state.PartyState.Inventory.Provisions.Food.Get()

	elapsed := state.AdvanceGameplayMinutesAndProcessProvisions(1)

	assert.Equal(t, 1, elapsed)
	assert.Equal(t, 6, state.lastProcessedHour)
	assert.Empty(t, state.pendingProvisionHours)
	assert.Less(t, state.PartyState.Inventory.Provisions.Food.Get(), foodBefore)
	require.NoError(t, state.ValidateSaveSnapshotEligibility())
}

func TestPlayerUpdateCyclesAdvanceBeggarCooldown(t *testing.T) {
	t.Run("surface", func(t *testing.T) {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		state.lastProcessedHour = int(state.DateTime.Hour)
		state.beggarKarmaCounter = beggarKarmaRewardThreshold - 1

		state.processNonCombatPlayerTurnEffects()

		assert.Equal(t, beggarKarmaRewardThreshold, state.beggarKarmaCounter)
	})

	t.Run("dungeon", func(t *testing.T) {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Deceit).
			WithPlayerAt(1, 1).
			WithSystemCallbacks().
			Build()
		state.lastProcessedHour = int(state.DateTime.Hour)
		state.beggarKarmaCounter = beggarKarmaRewardThreshold - 1

		state.processDungeonEndPlayerFinalEffects()

		assert.Equal(t, beggarKarmaRewardThreshold, state.beggarKarmaCounter)
	})

	t.Run("bed rest", func(t *testing.T) {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		state.beggarKarmaCounter = beggarKarmaRewardThreshold - 1
		rest := state.NewBedRestState(1)
		state.BeginBedRest(rest)
		defer state.FinishBedRest(rest)

		state.ProcessBedRestCycle(rest)

		assert.Equal(t, beggarKarmaRewardThreshold, state.beggarKarmaCounter)
	})

	t.Run("partial bed rest", func(t *testing.T) {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		state.beggarKarmaCounter = beggarKarmaRewardThreshold - 2

		state.AdvanceBedRestMinutes(1)

		assert.Equal(t, beggarKarmaRewardThreshold-2, state.beggarKarmaCounter)
	})
}

func TestBeggarCooldownSaturatesAtOriginalByteMaximum(t *testing.T) {
	state := &GameState{beggarKarmaCounter: maximumBeggarKarmaCounter}

	state.advanceBeggarKarmaCounter()

	assert.Equal(t, maximumBeggarKarmaCounter, state.beggarKarmaCounter)
}
