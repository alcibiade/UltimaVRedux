package environment

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestWindAdvanceRejectsOverflowBeforeRandomDraw(t *testing.T) {
	rng := beradrng.New(12345)
	wind := NewWindSystem(rng, &MockWindSystemCallbacks{})
	wind.Restore(WindSnapshot{Direction: East, NextTickMs: math.MaxInt64 - 99})
	before := wind.Snapshot()
	assert.Panics(t, func() { wind.AdvanceThrough(math.MaxInt64 - 40) })
	assert.Equal(t, before, wind.Snapshot())
	assert.Zero(t, rng.Calls())
}

func TestWindAdvanceNearDeadlineLimitDoesNotWrap(t *testing.T) {
	rng := beradrng.New(12345)
	wind := NewWindSystem(rng, &MockWindSystemCallbacks{})
	wind.Restore(WindSnapshot{Direction: East, NextTickMs: math.MaxInt64 - WindTickIntervalMs})
	wind.AdvanceThrough(math.MaxInt64 - WindTickIntervalMs)
	assert.Equal(t, int64(math.MaxInt64), wind.Snapshot().NextTickMs)
}

// MockWindSystemCallbacks for testing
type MockWindSystemCallbacks struct {
	UIUpdates []WindDirection
	Messages  []string
}

func TestWindTickPhaseRestoresDeterministically(t *testing.T) {
	firstRNG := beradrng.New(12345)
	first := NewWindSystem(firstRNG, &MockWindSystemCallbacks{})
	first.Restore(WindSnapshot{Direction: East, UpdateCounter: 2, NextTickMs: 100})
	first.AdvanceThrough(209)

	saved := first.Snapshot()
	restoredRNG := beradrng.Restore(firstRNG.Seed(), firstRNG.Calls())
	restored := NewWindSystem(restoredRNG, &MockWindSystemCallbacks{})
	restored.Restore(saved)

	first.AdvanceThrough(320)
	restored.AdvanceThrough(320)
	assert.Equal(t, first.Snapshot(), restored.Snapshot())
	assert.Equal(t, firstRNG.Calls(), restoredRNG.Calls())
}

func TestWindAdvanceThroughIsIndependentOfElapsedPartitions(t *testing.T) {
	partitionedRNG := beradrng.New(12345)
	partitioned := NewWindSystem(partitionedRNG, &MockWindSystemCallbacks{})
	for elapsedMs := WindTickIntervalMs; elapsedMs <= WindTickIntervalMs*100; elapsedMs += WindTickIntervalMs {
		partitioned.AdvanceThrough(elapsedMs)
	}

	singleRNG := beradrng.New(12345)
	single := NewWindSystem(singleRNG, &MockWindSystemCallbacks{})
	single.AdvanceThrough(WindTickIntervalMs * 100)

	assert.Equal(t, partitioned.Snapshot(), single.Snapshot())
	assert.Equal(t, partitionedRNG.Calls(), singleRNG.Calls())
}

func (m *MockWindSystemCallbacks) ShowWindUI(direction WindDirection) {
	m.UIUpdates = append(m.UIUpdates, direction)
}

func (m *MockWindSystemCallbacks) AddRowStr(message string) {
	m.Messages = append(m.Messages, message)
}

func (m *MockWindSystemCallbacks) Reset() {
	m.UIUpdates = m.UIUpdates[:0]
	m.Messages = m.Messages[:0]
}

func TestWindDirectionString(t *testing.T) {
	testCases := []struct {
		direction WindDirection
		expected  string
		uiString  string
	}{
		{Calm, "Calm", "Calm Winds"},
		{North, "North", "North Winds"},
		{South, "South", "South Winds"},
		{East, "East", "East Winds"},
		{West, "West", "West Winds"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.direction.String())
			assert.Equal(t, tc.uiString, tc.direction.GetUIString())
		})
	}
}

func TestNewWindSystem(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	mockCallbacks := &MockWindSystemCallbacks{}

	windSystem := NewWindSystem(rng, mockCallbacks)

	assert.NotNil(t, windSystem)
	assert.Equal(t, Calm, windSystem.GetCurrentWind())
	assert.True(t, windSystem.IsWindCalm())
}

func TestSetWindManually(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	mockCallbacks := &MockWindSystemCallbacks{}
	windSystem := NewWindSystem(rng, mockCallbacks)

	// Test setting each wind direction
	windSystem.SetWind(North, true)
	assert.Equal(t, North, windSystem.GetCurrentWind())
	assert.False(t, windSystem.IsWindCalm())
	assert.Contains(t, mockCallbacks.UIUpdates, North)

	mockCallbacks.Reset()
	windSystem.SetWind(South, false) // No immediate UI update
	assert.Equal(t, South, windSystem.GetCurrentWind())
	assert.Empty(t, mockCallbacks.UIUpdates) // No UI update called

	windSystem.SetWind(East, true)
	assert.Equal(t, East, windSystem.GetCurrentWind())
	assert.Contains(t, mockCallbacks.UIUpdates, East)

	windSystem.SetWind(West, true)
	assert.Equal(t, West, windSystem.GetCurrentWind())

	// Test back to calm
	windSystem.SetWind(Calm, true)
	assert.Equal(t, Calm, windSystem.GetCurrentWind())
	assert.True(t, windSystem.IsWindCalm())
}

func TestSetWindByDirection(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	mockCallbacks := &MockWindSystemCallbacks{}
	windSystem := NewWindSystem(rng, mockCallbacks)

	testCases := []struct {
		direction references.Direction
		expected  WindDirection
		success   bool
	}{
		{references.Up, North, true},
		{references.Down, South, true},
		{references.Right, East, true},
		{references.Left, West, true},
		{references.NoneDirection, Calm, false}, // Invalid direction
	}

	for _, tc := range testCases {
		testName := "Invalid"
		if tc.direction != references.NoneDirection {
			testName = tc.direction.GetDirectionCompassName()
		}
		t.Run(testName, func(t *testing.T) {
			mockCallbacks.Reset()
			result := windSystem.SetWindByDirection(tc.direction, true)

			assert.Equal(t, tc.success, result)
			if tc.success {
				assert.Equal(t, tc.expected, windSystem.GetCurrentWind())
				assert.Contains(t, mockCallbacks.UIUpdates, tc.expected)
			}
		})
	}
}

func TestAutomaticWindChangeDeterminism(t *testing.T) {
	// Test that wind changes are deterministic with fixed seed
	rng1 := rand.New(rand.NewSource(12345))
	mockCallbacks1 := &MockWindSystemCallbacks{}
	windSystem1 := NewWindSystem(rng1, mockCallbacks1)

	rng2 := rand.New(rand.NewSource(12345))
	mockCallbacks2 := &MockWindSystemCallbacks{}
	windSystem2 := NewWindSystem(rng2, mockCallbacks2)

	// Run 1000 ticks on both systems
	for i := 0; i < 1000; i++ {
		windSystem1.UpdateWindTick()
		windSystem2.UpdateWindTick()
	}

	// Both should have identical wind states
	assert.Equal(t, windSystem1.GetCurrentWind(), windSystem2.GetCurrentWind())
	assert.Equal(t, len(mockCallbacks1.UIUpdates), len(mockCallbacks2.UIUpdates))
}

func TestAutomaticWindChangeFrequency(t *testing.T) {
	// Test that wind changes occur approximately 1/64 of the time
	rng := rand.New(rand.NewSource(99999))
	mockCallbacks := &MockWindSystemCallbacks{}
	windSystem := NewWindSystem(rng, mockCallbacks)

	const iterations = 6400 // 100 * 64 iterations
	for i := 0; i < iterations; i++ {
		windSystem.UpdateWindTick()
	}

	changeCount := len(mockCallbacks.UIUpdates)

	// Should have approximately 100 changes (6400/64), allow for variance
	assert.Greater(t, changeCount, 50, "Should have some wind changes")
	assert.Less(t, changeCount, 150, "Should not have too many changes")
}

func TestGenerateRandomWindChangeCalmBoundary(t *testing.T) {
	for _, test := range []struct {
		name   string
		rolls  []int
		want   WindDirection
		limits []int
	}{
		{name: "191 rejects calm", rolls: []int{0, 191, 1}, want: North, limits: []int{5, 256, 5}},
		{name: "192 accepts calm", rolls: []int{0, 192}, want: Calm, limits: []int{5, 256}},
	} {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedWindRNG{t: t, rolls: test.rolls}
			windSystem := NewWindSystem(rng, &MockWindSystemCallbacks{})

			windSystem.generateRandomWindChange()

			assert.Equal(t, test.want, windSystem.GetCurrentWind())
			assert.Equal(t, test.limits, rng.limits)
			assert.Equal(t, len(test.rolls), rng.index)
		})
	}

	// Calm has 64 accepted byte values; each cardinal direction has 256.
	acceptedCalmWeight := 256 - 192
	cardinalWeight := 256
	totalWeight := acceptedCalmWeight + 4*cardinalWeight
	assert.Equal(t, 17, totalWeight/acceptedCalmWeight)
	assert.Equal(t, 4, cardinalWeight/acceptedCalmWeight)
}

type scriptedWindRNG struct {
	t      *testing.T
	rolls  []int
	limits []int
	index  int
}

func (r *scriptedWindRNG) Intn(limit int) int {
	r.t.Helper()
	require.Less(r.t, r.index, len(r.rolls))
	roll := r.rolls[r.index]
	require.Less(r.t, roll, limit)
	r.index++
	r.limits = append(r.limits, limit)

	return roll
}

func TestFrigateSailingSpeed_AuthenticUltimaV(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	mockCallbacks := &MockWindSystemCallbacks{}
	windSystem := NewWindSystem(rng, mockCallbacks)

	// Test authentic Ultima V frigate sailing algorithm
	// Wind vectors in order are south, north, west, east; ref:SHIP-0001.
	// Wind vectors: North=(0,1), South=(0,-1), East=(-1,0), West=(1,0)
	// Start at one wait; add one for each movement component differing from wind; ref:SHIP-0001.
	testCases := []struct {
		windDir  WindDirection
		sailDir  references.Direction
		expected int
		name     string
	}{
		// North wind (wind vector: 0, 1)
		{North, references.Up, 2, "North wind, sailing North (opposing Y)"},     // dx: 0==0 ✓, dy: 1!=-1 ✗ -> num = 2
		{North, references.Down, 1, "North wind, sailing South (same Y)"},       // dx: 0==0 ✓, dy: 1==1 ✓ -> num = 1
		{North, references.Right, 3, "North wind, sailing East (opposing X,Y)"}, // dx: 0!=1 ✗, dy: 1!=0 ✗ -> num = 3
		{North, references.Left, 3, "North wind, sailing West (opposing X,Y)"},  // dx: 0!=-1 ✗, dy: 1!=0 ✗ -> num = 3

		// South wind (wind vector: 0, -1)
		{South, references.Up, 1, "South wind, sailing North (same Y)"},         // dx: 0==0 ✓, dy: -1==-1 ✓ -> num = 1
		{South, references.Down, 2, "South wind, sailing South (opposing Y)"},   // dx: 0==0 ✓, dy: -1!=1 ✗ -> num = 2
		{South, references.Right, 3, "South wind, sailing East (opposing X,Y)"}, // dx: 0!=1 ✗, dy: -1!=0 ✗ -> num = 3
		{South, references.Left, 3, "South wind, sailing West (opposing X,Y)"},  // dx: 0!=-1 ✗, dy: -1!=0 ✗ -> num = 3

		// East wind (wind vector: -1, 0)
		{East, references.Up, 3, "East wind, sailing North (opposing X,Y)"},   // dx: -1!=0 ✗, dy: 0!=-1 ✗ -> num = 3
		{East, references.Down, 3, "East wind, sailing South (opposing X,Y)"}, // dx: -1!=0 ✗, dy: 0!=1 ✗ -> num = 3
		{East, references.Right, 2, "East wind, sailing East (opposing X)"},   // dx: -1!=1 ✗, dy: 0==0 ✓ -> num = 2
		{East, references.Left, 1, "East wind, sailing West (same X)"},        // dx: -1==-1 ✓, dy: 0==0 ✓ -> num = 1

		// West wind (wind vector: 1, 0)
		{West, references.Up, 3, "West wind, sailing North (opposing X,Y)"},   // dx: 1!=0 ✗, dy: 0!=-1 ✗ -> num = 3
		{West, references.Down, 3, "West wind, sailing South (opposing X,Y)"}, // dx: 1!=0 ✗, dy: 0!=1 ✗ -> num = 3
		{West, references.Right, 1, "West wind, sailing East (same X)"},       // dx: 1==1 ✓, dy: 0==0 ✓ -> num = 1
		{West, references.Left, 2, "West wind, sailing West (opposing X)"},    // dx: 1!=-1 ✗, dy: 0==0 ✓ -> num = 2
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			windSystem.SetWind(tc.windDir, false)
			speed := windSystem.GetFrigateSailingSpeed(tc.sailDir)
			assert.Equal(t, tc.expected, speed, "Wind %v, sailing %v should give speed %d", tc.windDir, tc.sailDir, tc.expected)
		})
	}
}

func TestFrigateSailingTiming(t *testing.T) {
	// Test the ship movement timing system: updtcnt >= num % 3
	rng := rand.New(rand.NewSource(12345))
	mockCallbacks := &MockWindSystemCallbacks{}
	windSystem := NewWindSystem(rng, mockCallbacks)

	// Set east wind for predictable testing
	windSystem.SetWind(East, false)

	// With East wind: sailing East (speed 2), sailing West (speed 1)
	// Test sailing East (speed 2, should move when updateCounter >= 2 % 3 = 2)
	eastDirection := references.Right

	// Initially should not move (counter = 0, speed % 3 = 2, 0 >= 2 = false)
	assert.False(t, windSystem.ShouldMoveShip(eastDirection))

	// After 1 increment should still not move (counter = 1, 1 >= 2 = false)
	windSystem.IncrementUpdateCount()
	assert.False(t, windSystem.ShouldMoveShip(eastDirection))

	// After 2nd increment should move (counter = 2, 2 >= 2 = true)
	windSystem.IncrementUpdateCount()
	assert.True(t, windSystem.ShouldMoveShip(eastDirection))

	// Test sailing West (speed 1, should move when updateCounter >= 1 % 3 = 1)
	westDirection := references.Left
	windSystem.updateCounter = 0 // Reset counter

	// Should not move initially (counter = 0, speed % 3 = 1, 0 >= 1 = false)
	assert.False(t, windSystem.ShouldMoveShip(westDirection))

	// Should move after 1 increment (counter = 1, speed % 3 = 1, 1 >= 1 = true)
	windSystem.IncrementUpdateCount()
	assert.True(t, windSystem.ShouldMoveShip(westDirection))
}

func TestFrigateSailingCalm(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	mockCallbacks := &MockWindSystemCallbacks{}
	windSystem := NewWindSystem(rng, mockCallbacks)

	windSystem.SetWind(Calm, false)

	// All ship directions should be stalled in calm winds (high tick count = slow movement)
	directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}
	for _, dir := range directions {
		speed := windSystem.GetFrigateSailingSpeed(dir)
		assert.Equal(t, 4, speed, "All ships should be stalled in calm winds")
	}
}

func TestFrigateSailingInvalidDirection(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	mockCallbacks := &MockWindSystemCallbacks{}
	windSystem := NewWindSystem(rng, mockCallbacks)

	windSystem.SetWind(North, false)

	// Invalid direction should return base speed
	speed := windSystem.GetFrigateSailingSpeed(references.NoneDirection)
	assert.Equal(t, 1, speed, "Invalid direction should return base speed")
}

func TestWindSystemInvalidDirections(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	mockCallbacks := &MockWindSystemCallbacks{}
	windSystem := NewWindSystem(rng, mockCallbacks)

	initialWind := windSystem.GetCurrentWind()

	// Try to set invalid wind direction
	windSystem.SetWind(WindDirection(99), true)

	// Should remain unchanged
	assert.Equal(t, initialWind, windSystem.GetCurrentWind())
	assert.Empty(t, mockCallbacks.UIUpdates)
}
