package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func drainDungeonInjuryFeedback(t *testing.T, state *GameState) {
	t.Helper()
	for range 100 {
		state.EventQueueManager.UpdateFeedback()
		pending := false
		for _, kind := range []DungeonInjuryKind{DungeonInjuryFire, DungeonInjuryElectrical, DungeonInjuryPoison} {
			name, _ := dungeonInjuryFeedbackIdentity(kind)
			pending = pending || state.EventQueueManager.HasFeedback(name)
		}
		if !pending {
			return
		}
		state.ElapsedMs += 10
	}
	t.Fatal("dungeon injury feedback did not drain within its bounded lifetime")
}

func TestDungeonInjuryResultCountsActualClampedLossAndPreservesDiceOrder(t *testing.T) {
	state, callbacks := newDungeonInteractionTestState(t, 0x82, 43)
	characters := configureDungeonHazardParty(t, state, party_state.Good, party_state.Poisoned, party_state.Sleep, party_state.Dead)
	characters[0].SetHp(1)
	state.SetRandomSeed(43)
	state.RollDice(8) // The fatal first member still consumes its ordinary die.
	secondDamage, thirdDamage := state.RollDice(8), state.RollDice(8)
	expectedNext := state.RollDice(8)
	state.SetRandomSeed(43)
	callbacks.Reset()

	result := state.damageDungeonParty(DungeonInjuryFire)

	assert.Equal(t, dungeonInjuryResult{kind: DungeonInjuryFire, hpLost: 1 + secondDamage + thirdDamage}, result)
	assert.Equal(t, []uint16{0, uint16(100 - secondDamage), uint16(100 - thirdDamage), 100}, dungeonHazardHP(characters))
	assert.Equal(t, uint64(3), state.GetRNG().Calls())
	assert.Equal(t, expectedNext, state.RollDice(8))
	assert.Empty(t, callbacks.SoundEffectsPlayed, "resolution itself does not multiply feedback per member")
	state.presentDungeonInjury(result)
	assert.Equal(t, []SoundEffect{SoundFireBurn}, callbacks.SoundEffectsPlayed, "sound dispatch is immediate")
	beforeHP, beforeDate, beforeRNG := dungeonHazardHP(characters), state.DateTime, state.GetRNG().Calls()
	drainDungeonInjuryFeedback(t, state)
	assert.Equal(t, []DungeonInjuryKind{DungeonInjuryFire, DungeonInjuryNone}, callbacks.DungeonInjuryCalls)
	assert.Equal(t, beforeHP, dungeonHazardHP(characters))
	assert.Equal(t, beforeDate, state.DateTime)
	assert.Equal(t, beforeRNG, state.GetRNG().Calls())
	assert.Empty(t, callbacks.ScreenFlashColors)
	assert.Zero(t, callbacks.BlockInputCalls)
}

func TestDungeonInjuryAdvanceAndBackUpDispatchOncePerResolvedSource(t *testing.T) {
	for _, test := range []struct {
		cell  references.DungeonCell
		kind  DungeonInjuryKind
		sound SoundEffect
		rolls uint64
	}{
		{0x82, DungeonInjuryFire, SoundFireBurn, 2},
		{0x8a, DungeonInjuryFire, SoundFireBurn, 2},
		{0x83, DungeonInjuryElectrical, SoundElectricalDamage, 2},
		{0x81, DungeonInjuryPoison, SoundPoison, 3},
		{0x89, DungeonInjuryPoison, SoundPoison, 3},
	} {
		for _, movement := range []DungeonMovement{DungeonAdvance, DungeonBackUp} {
			t.Run(fmt.Sprintf("%02x/%d", test.cell, movement), func(t *testing.T) {
				state, callbacks := newDungeonInteractionTestState(t, 0, 43)
				characters := configureDungeonHazardParty(t, state, party_state.Good, party_state.Good, party_state.Dead)
				state.DateTime.Hour, state.DateTime.Minute = 12, 10
				state.lastProcessedHour = 12
				state.activeDungeon.facing = references.DungeonFacingEast
				if movement == DungeonBackUp {
					state.activeDungeon.facing = references.DungeonFacingWest
				}
				target := references.Position{X: 2, Y: 1}
				setDungeonInteractionCell(t, state, target, test.cell)
				beforeRNG, expectedDate := state.GetRNG().Calls(), state.DateTime
				expectedDate.Advance(1)
				callbacks.Reset()

				result := performDungeonMove(t, state, movement)

				if test.kind == DungeonInjuryElectrical {
					assert.Equal(t, DungeonMoveElectricField, result.Outcome)
					assert.Equal(t, references.Position{X: 1, Y: 1}, state.MapState.PlayerLocation.Position)
				} else {
					assert.Equal(t, DungeonMoveMoved, result.Outcome)
					assert.Equal(t, target, state.MapState.PlayerLocation.Position)
				}
				assert.Equal(t, test.cell, dungeonInteractionCellAt(t, state, target))
				assert.Less(t, characters[0].GetCurrentHp(), uint16(100))
				assert.Less(t, characters[1].GetCurrentHp(), uint16(100))
				assert.EqualValues(t, 100, characters[2].GetCurrentHp(), "dead members are unaffected")
				assert.Equal(t, test.rolls, state.GetRNG().Calls()-beforeRNG)
				assert.Equal(t, expectedDate, state.DateTime)
				assert.True(t, state.IsAtDungeonCommandBoundary())
				assert.Equal(t, []SoundEffect{test.sound}, callbacks.SoundEffectsPlayed)
				assert.False(t, state.EventQueueManager.HasActiveWorkflows())
				drainDungeonInjuryFeedback(t, state)
				assert.Equal(t, []DungeonInjuryKind{test.kind, DungeonInjuryNone}, callbacks.DungeonInjuryCalls)
				assert.Zero(t, callbacks.BlockInputCalls)
			})
		}
	}
}

func TestDungeonInjuryRapidElectricalBounceRetainsOriginalCellSources(t *testing.T) {
	for _, origin := range []references.DungeonCell{0x82, 0x81} {
		t.Run(fmt.Sprintf("%02x", origin), func(t *testing.T) {
			state, callbacks := newDungeonInteractionTestState(t, origin, 43)
			characters := configureDungeonHazardParty(t, state, party_state.Poisoned, party_state.Poisoned)
			for _, character := range characters {
				character.SetMaxHp(1000)
				character.SetHp(1000)
			}
			state.activeDungeon.facing = references.DungeonFacingEast
			state.DateTime.Hour, state.DateTime.Minute = 12, 10
			state.lastProcessedHour = 12
			setDungeonInteractionCell(t, state, references.Position{X: 2, Y: 1}, 0x83)
			callbacks.Reset()
			beforeRNG := state.GetRNG().Calls()
			var expectedSounds []SoundEffect
			for range 12 {
				result := performDungeonMove(t, state, DungeonAdvance)
				require.Equal(t, DungeonMoveElectricField, result.Outcome)
				expectedSounds = append(expectedSounds, SoundElectricalDamage)
				if origin == 0x82 {
					expectedSounds = append(expectedSounds, SoundFireBurn)
				}
				expectedSounds = append(expectedSounds, SoundPoison)
			}
			assert.Equal(t, expectedSounds, callbacks.SoundEffectsPlayed)
			assert.Equal(t, uint64(48), state.GetRNG().Calls()-beforeRNG)
			assert.Equal(t, references.Position{X: 1, Y: 1}, state.MapState.PlayerLocation.Position)
			assert.Equal(t, origin, currentDungeonInteractionCell(t, state))
			expectedVisuals := []DungeonInjuryKind{DungeonInjuryElectrical, DungeonInjuryNone}
			if origin == 0x82 {
				expectedVisuals = append(expectedVisuals, DungeonInjuryFire, DungeonInjuryNone)
			}
			expectedVisuals = append(expectedVisuals, DungeonInjuryPoison, DungeonInjuryNone)
			beforeHP, beforeDate, afterRNG := dungeonHazardHP(characters), state.DateTime, state.GetRNG().Calls()
			drainDungeonInjuryFeedback(t, state)
			assert.Equal(t, expectedVisuals, callbacks.DungeonInjuryCalls, "twelve rapid passes coalesce to at most three pulses")
			assert.Equal(t, beforeHP, dungeonHazardHP(characters))
			assert.Equal(t, beforeDate, state.DateTime)
			assert.Equal(t, afterRNG, state.GetRNG().Calls())
			assert.Zero(t, callbacks.BlockInputCalls)
			callbacks.Reset()
			completeDungeonHazardCommand(state)
			drainDungeonInjuryFeedback(t, state)
			assert.Equal(t, expectedVisuals[2:], callbacks.DungeonInjuryCalls,
				"standing in the persistent origin field presents again after completion")
		})
	}
}

func TestDungeonInjuryUnaffectedOutcomesDoNotAdmitFeedback(t *testing.T) {
	for _, test := range []struct {
		name   string
		cell   references.DungeonCell
		status party_state.CharacterStatus
		hp     uint16
		dex    byte
	}{
		{"sleep", 0x80, party_state.Good, 100, 1},
		{"resisted poison", 0x81, party_state.Good, 100, 31},
		{"harmless electrical variant", 0x8b, party_state.Good, 100, 1},
		{"dead fire member", 0x82, party_state.Dead, 100, 1},
		{"dead poison member", 0x81, party_state.Dead, 100, 1},
		{"zero HP poison tick", 0, party_state.Poisoned, 0, 1},
		{"zero HP fire", 0x82, party_state.Good, 0, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := newDungeonInteractionTestState(t, test.cell, 43)
			characters := configureDungeonHazardParty(t, state, test.status)
			characters[0].SetHp(test.hp)
			characters[0].SetStatus(test.status)
			characters[0].Dexterity = test.dex
			callbacks.Reset()
			state.processDungeonCurrentCellField()
			state.processDungeonPoisonDamage(state.cachePlayerUpdateStatuses())
			drainDungeonInjuryFeedback(t, state)
			assert.Equal(t, test.hp, characters[0].GetCurrentHp())
			assert.Empty(t, callbacks.DungeonInjuryCalls)
			for _, cue := range []SoundEffect{SoundFireBurn, SoundElectricalDamage, SoundPoison} {
				assert.NotContains(t, callbacks.SoundEffectsPlayed, cue)
			}
		})
	}
}

func TestDungeonInjuryFatalDamageDispatchesBeforeUnchangedWipeHandling(t *testing.T) {
	for _, source := range []DungeonInjuryKind{DungeonInjuryFire, DungeonInjuryElectrical, DungeonInjuryPoison} {
		t.Run(fmt.Sprint(source), func(t *testing.T) {
			cell := references.DungeonCell(0x82)
			if source == DungeonInjuryPoison {
				cell = 0x81
			}
			state, callbacks := newDungeonInteractionTestState(t, cell, 43)
			characters := configureDungeonHazardParty(t, state, party_state.Good)
			characters[0].SetHp(1)
			state.activeDungeon.facing = references.DungeonFacingEast
			setDungeonInteractionCell(t, state, references.Position{X: 2, Y: 1}, 0x83)
			_, cue := dungeonInjuryFeedbackIdentity(source)
			deaths := 0
			state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() {
				deaths++
				assert.Equal(t, []SoundEffect{cue}, callbacks.SoundEffectsPlayed)
			}
			callbacks.Reset()
			if source == DungeonInjuryElectrical {
				performDungeonMove(t, state, DungeonAdvance)
			} else {
				completeDungeonHazardCommand(state)
			}
			assert.Equal(t, 1, deaths)
			assert.True(t, characters[0].IsDead())
			assert.Zero(t, characters[0].GetCurrentHp())
			assert.True(t, state.IsAtDungeonCommandBoundary())
			drainDungeonInjuryFeedback(t, state)
			assert.Equal(t, []DungeonInjuryKind{source, DungeonInjuryNone}, callbacks.DungeonInjuryCalls)
			assert.Equal(t, []SoundEffect{cue}, callbacks.SoundEffectsPlayed)
			assert.Equal(t, 1, deaths, "feedback must not repeat death or damage")
		})
	}
}

func TestDungeonInjuryCancellationCleansOnlyStartedPulseOnce(t *testing.T) {
	state, callbacks := newDungeonInteractionTestState(t, 0x82, 43)
	configureDungeonHazardParty(t, state, party_state.Poisoned)
	state.processDungeonElectricField()
	state.processDungeonCurrentCellField()
	state.processDungeonPoisonDamage(state.cachePlayerUpdateStatuses())
	state.EventQueueManager.UpdateFeedback()
	require.Equal(t, []DungeonInjuryKind{DungeonInjuryElectrical}, callbacks.DungeonInjuryCalls)
	state.EventQueueManager.Shutdown()
	state.EventQueueManager.Shutdown()
	assert.Equal(t, []DungeonInjuryKind{DungeonInjuryElectrical, DungeonInjuryNone}, callbacks.DungeonInjuryCalls)
	assert.False(t, callbacks.InputBlocked)
}

func TestDungeonInjuryEachQueuedSourceRetainsPulseUntilItsOwnExpiry(t *testing.T) {
	state, callbacks := newDungeonInteractionTestState(t, 0x82, 43)
	characters := configureDungeonHazardParty(t, state, party_state.Poisoned)
	state.ElapsedMs = 1377
	state.processDungeonElectricField()
	state.processDungeonCurrentCellField()
	state.processDungeonPoisonDamage(state.cachePlayerUpdateStatuses())
	require.Equal(t, []SoundEffect{SoundElectricalDamage, SoundFireBurn, SoundPoison}, callbacks.SoundEffectsPlayed)
	require.Empty(t, callbacks.DungeonInjuryCalls, "admission queues visuals but dispatches sound immediately")
	beforeHP, beforeRNG, beforeDate := dungeonHazardHP(characters), state.GetRNG().Calls(), state.DateTime
	// A late first update must still present a full pulse. Each pending source
	// starts its own lifetime when promoted rather than expiring in the backlog.
	state.ElapsedMs += 5000
	var expected []DungeonInjuryKind
	for _, kind := range []DungeonInjuryKind{DungeonInjuryElectrical, DungeonInjuryFire, DungeonInjuryPoison} {
		name, _ := dungeonInjuryFeedbackIdentity(kind)
		require.True(t, state.EventQueueManager.HasFeedback(name))
		state.EventQueueManager.UpdateFeedback()
		expected = append(expected, kind)
		require.Equal(t, expected, callbacks.DungeonInjuryCalls, "start must not immediately run expiry cleanup")
		require.True(t, state.EventQueueManager.HasFeedback(name))
		state.EventQueueManager.UpdateFeedback()
		require.Equal(t, expected, callbacks.DungeonInjuryCalls, "same-clock updates cannot shorten a pulse")
		state.ElapsedMs += DungeonInjuryDurationMs - 1
		state.EventQueueManager.UpdateFeedback()
		require.Equal(t, expected, callbacks.DungeonInjuryCalls)
		require.True(t, state.EventQueueManager.HasFeedback(name), "source remains live at 199 ms")
		state.ElapsedMs++
		state.EventQueueManager.UpdateFeedback()
		expected = append(expected, DungeonInjuryNone)
		require.Equal(t, expected, callbacks.DungeonInjuryCalls, "one cleanup at 200 ms before the next source")
		require.False(t, state.EventQueueManager.HasFeedback(name))
	}
	state.EventQueueManager.Shutdown()
	assert.Equal(t, expected, callbacks.DungeonInjuryCalls, "normal expiry and cancellation do not double-clear")
	assert.Equal(t, beforeHP, dungeonHazardHP(characters))
	assert.Equal(t, beforeRNG, state.GetRNG().Calls())
	assert.Equal(t, beforeDate, state.DateTime)
	assert.Zero(t, callbacks.BlockInputCalls)
}

func TestDungeonInjuryPoisonResultSeparatesLossFromAlreadyEmptyHP(t *testing.T) {
	state, callbacks := newDungeonInteractionTestState(t, 0, 43)
	characters := configureDungeonHazardParty(t, state,
		party_state.Poisoned, party_state.Poisoned, party_state.Poisoned, party_state.Dead, party_state.Good)
	characters[0].SetHp(0)
	characters[0].SetStatus(party_state.Poisoned)
	characters[1].SetHp(1)
	cache := state.cachePlayerUpdateStatuses()
	beforeRNG := state.GetRNG().Calls()
	callbacks.Reset()

	result := state.resolveCachedPlayerUpdatePoison(cache)

	assert.Equal(t, 2, result.hpLost)
	assert.Equal(t, []int{1}, result.deadIndexes, "only a newly fatal HP transition is a death")
	assert.Equal(t, []uint16{0, 0, 99, 100, 100}, dungeonHazardHP(characters))
	assert.Equal(t, beforeRNG, state.GetRNG().Calls())
	assert.Empty(t, callbacks.SoundEffectsPlayed)
	assert.Empty(t, callbacks.DungeonInjuryCalls)
}

func TestDungeonInjuryHarmlessElectricalMovementAndDeadContactHaveNoCues(t *testing.T) {
	for _, movement := range []DungeonMovement{DungeonAdvance, DungeonBackUp} {
		t.Run(fmt.Sprint(movement), func(t *testing.T) {
			state, callbacks := newDungeonInteractionTestState(t, 0, 43)
			characters := configureDungeonHazardParty(t, state, party_state.Good)
			state.activeDungeon.facing = references.DungeonFacingEast
			if movement == DungeonBackUp {
				state.activeDungeon.facing = references.DungeonFacingWest
			}
			target := references.Position{X: 2, Y: 1}
			setDungeonInteractionCell(t, state, target, 0x8b)
			beforeRNG := state.GetRNG().Calls()
			callbacks.Reset()
			result := performDungeonMove(t, state, movement)
			drainDungeonInjuryFeedback(t, state)
			assert.Equal(t, DungeonMoveMoved, result.Outcome)
			assert.Equal(t, target, state.MapState.PlayerLocation.Position)
			assert.EqualValues(t, 100, characters[0].GetCurrentHp())
			assert.Equal(t, beforeRNG, state.GetRNG().Calls())
			assert.Empty(t, callbacks.SoundEffectsPlayed)
			assert.Empty(t, callbacks.DungeonInjuryCalls)
			characters[0].SetStatus(party_state.Dead)
			state.processDungeonElectricField()
			drainDungeonInjuryFeedback(t, state)
			assert.Equal(t, beforeRNG, state.GetRNG().Calls())
			assert.Empty(t, callbacks.SoundEffectsPlayed)
			assert.Empty(t, callbacks.DungeonInjuryCalls)
		})
	}
}

func TestDungeonInjuryQueuedPulseDoesNotCrossDungeonVisitReplacement(t *testing.T) {
	state, callbacks := newDungeonInteractionTestState(t, 0x82, 43)
	configureDungeonHazardParty(t, state, party_state.Good)
	owner := state.CaptureDungeonInjuryOwner()
	state.processDungeonCurrentCellField()
	player := state.MapState.PlayerLocation
	snapshot, active := state.DungeonStateSnapshot()
	require.True(t, active)
	_, err := state.DebugSetupDungeonRoute(player.Location, player.Floor, player.Position, snapshot.Facing, snapshot.LastDirection)
	require.NoError(t, err)
	assert.False(t, state.MatchesDungeonInjuryOwner(owner))
	drainDungeonInjuryFeedback(t, state)
	assert.Empty(t, callbacks.DungeonInjuryCalls)
	assert.Equal(t, []SoundEffect{SoundFireBurn}, callbacks.SoundEffectsPlayed,
		"audio was already dispatched with the original actual injury")
}
