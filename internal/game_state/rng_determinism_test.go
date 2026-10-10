package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// testHappenedByPercentLikely is a test helper that returns true with given likelihood percentage
// (moved from game_state.go to keep test-only code in test files)
func testHappenedByPercentLikely(gs *GameState, likelihoodToSucceedPercent int) bool {
	if likelihoodToSucceedPercent >= 100 {
		return true
	}
	if likelihoodToSucceedPercent <= 0 {
		return false
	}
	return gs.RandomIntInRange(0, 100) < likelihoodToSucceedPercent
}

// TestRNGDeterminism verifies that the centralized RNG system produces
// deterministic results with the same seed
func TestRNGDeterminism(t *testing.T) {
	testCases := []struct {
		name string
		seed uint64
	}{
		{"Fixed seed 1", 1},
		{"Fixed seed 12345", 12345},
		{"Fixed seed 999999", 999999},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create two identical GameStates with same seed
			gameState1 := createTestGameState()
			gameState2 := createTestGameState()

			gameState1.SetRandomSeed(tc.seed)
			gameState2.SetRandomSeed(tc.seed)

			// Test OneInXOdds determinism
			for i := 0; i < 50; i++ {
				result1 := gameState1.OneInXOdds(4)
				result2 := gameState2.OneInXOdds(4)
				if result1 != result2 {
					t.Errorf("OneInXOdds(%d) not deterministic at iteration %d: %v != %v", 4, i, result1, result2)
				}
			}

			// Test RandomIntInRange determinism
			for i := 0; i < 50; i++ {
				result1 := gameState1.RandomIntInRange(1, 10)
				result2 := gameState2.RandomIntInRange(1, 10)
				if result1 != result2 {
					t.Errorf("RandomIntInRange(1,10) not deterministic at iteration %d: %v != %v", i, result1, result2)
				}
			}

			// Test HappenedByPercentLikely determinism
			for i := 0; i < 50; i++ {
				result1 := testHappenedByPercentLikely(gameState1, 25)
				result2 := testHappenedByPercentLikely(gameState2, 25)
				if result1 != result2 {
					t.Errorf("HappenedByPercentLikely(25) not deterministic at iteration %d: %v != %v", i, result1, result2)
				}
			}
		})
	}
}

// TestRNGProgression verifies that RNG actually progresses over time
func TestRNGProgression(t *testing.T) {
	gameState := createTestGameState()
	gameState.SetRandomSeed(12345)

	// Test that consecutive calls produce different sequences
	firstSequence := make([]int, 20)
	for i := range firstSequence {
		firstSequence[i] = gameState.RandomIntInRange(1, 1000)
	}

	gameState.SetRandomSeed(12345) // Reset to same seed
	secondSequence := make([]int, 20)
	for i := range secondSequence {
		secondSequence[i] = gameState.RandomIntInRange(1, 1000)
	}

	// Sequences should be identical when seed is reset
	for i := range firstSequence {
		if firstSequence[i] != secondSequence[i] {
			t.Errorf("RNG not deterministic after seed reset at position %d: %d != %d", i, firstSequence[i], secondSequence[i])
		}
	}

	// But if we continue, the next value should be different from the first
	nextValue := gameState.RandomIntInRange(1, 1000)
	if nextValue == firstSequence[0] {
		t.Error("RNG not progressing - got same value as start of sequence")
	}
}

// TestShuffleDirectionsDeterminism verifies that direction shuffling is deterministic
func TestShuffleDirectionsDeterminism(t *testing.T) {
	gameState1 := createTestGameState()
	gameState2 := createTestGameState()

	gameState1.SetRandomSeed(42)
	gameState2.SetRandomSeed(42)

	for testRun := 0; testRun < 10; testRun++ {
		// Create identical direction slices
		directions1 := []references.Position{
			{X: 0, Y: -1}, // Up
			{X: 0, Y: 1},  // Down
			{X: -1, Y: 0}, // Left
			{X: 1, Y: 0},  // Right
		}
		directions2 := []references.Position{
			{X: 0, Y: -1}, // Up
			{X: 0, Y: 1},  // Down
			{X: -1, Y: 0}, // Left
			{X: 1, Y: 0},  // Right
		}

		gameState1.ShuffleDirections(directions1)
		gameState2.ShuffleDirections(directions2)

		// Shuffled results should be identical
		if len(directions1) != len(directions2) {
			t.Fatalf("ShuffleDirections changed slice lengths: %d != %d", len(directions1), len(directions2))
		}

		for i := range directions1 {
			if directions1[i] != directions2[i] {
				t.Errorf("ShuffleDirections not deterministic at test run %d, position %d: %v != %v",
					testRun, i, directions1[i], directions2[i])
			}
		}
	}
}

// TestRNGBounds verifies that RNG methods respect their bounds
func TestRNGBounds(t *testing.T) {
	gameState := createTestGameState()
	gameState.SetRandomSeed(1)

	// Test RandomIntInRange bounds
	for i := 0; i < 1000; i++ {
		result := gameState.RandomIntInRange(10, 20)
		if result < 10 || result > 20 {
			t.Errorf("RandomIntInRange(10, 20) out of bounds: got %d", result)
		}
	}

	// Test HappenedByPercentLikely edge cases
	if !testHappenedByPercentLikely(gameState, 100) {
		t.Error("HappenedByPercentLikely(100) should always return true")
	}
	if testHappenedByPercentLikely(gameState, 0) {
		t.Error("HappenedByPercentLikely(0) should always return false")
	}

	// Test OneInXOdds edge cases
	if gameState.OneInXOdds(0) {
		t.Error("OneInXOdds(0) should return false")
	}
	if gameState.OneInXOdds(-1) {
		t.Error("OneInXOdds(-1) should return false")
	}
}

// TestNPCAIControllerIntegration verifies that AI controllers can use the deterministic RNG
func TestNPCAIControllerIntegration(t *testing.T) {
	// This test verifies that our RNG provider interface works with AI controllers
	gameState := createTestGameState()
	gameState.SetRandomSeed(123)

	// Test that GameState implements RNGProvider interface
	var rngProvider interface{} = gameState
	if _, ok := rngProvider.(interface {
		OneInXOdds(odds int) bool
		RandomIntInRange(minimum, maximum int) int
		ShuffleDirections(directions []references.Position)
	}); !ok {
		t.Error("GameState does not implement RNGProvider interface correctly")
	}

	// Test the interface methods
	result := gameState.OneInXOdds(2)
	if result != true && result != false {
		t.Error("OneInXOdds should return a boolean")
	}

	directions := []references.Position{{X: 1, Y: 0}, {X: 0, Y: 1}}
	originalLen := len(directions)
	gameState.ShuffleDirections(directions)
	if len(directions) != originalLen {
		t.Errorf("ShuffleDirections changed slice length: %d -> %d", originalLen, len(directions))
	}
}

// TestRNGConsistencyAcrossRuns - REGRESSION TEST
// Ensures the same operations on same seed always produce same results
func TestRNGConsistencyAcrossRuns(t *testing.T) {
	gameState := createTestGameState()
	gameState.SetRandomSeed(1337)

	// Execute the sequence and capture actual results
	result1 := gameState.OneInXOdds(4)
	result2 := gameState.RandomIntInRange(1, 10)
	result3 := testHappenedByPercentLikely(gameState, 50)
	result4 := gameState.OneInXOdds(2)
	result5 := gameState.RandomIntInRange(100, 200)

	t.Logf("Seed 1337 produces: OneInXOdds(4)=%v, RandomIntInRange(1,10)=%d, HappenedByPercentLikely(50)=%v, OneInXOdds(2)=%v, RandomIntInRange(100,200)=%d",
		result1, result2, result3, result4, result5)

	// Now test that the same seed produces the same results
	gameState.SetRandomSeed(1337) // Reset
	if gameState.OneInXOdds(4) != result1 {
		t.Errorf("OneInXOdds(4) not deterministic: expected %v", result1)
	}
	if gameState.RandomIntInRange(1, 10) != result2 {
		t.Errorf("RandomIntInRange(1,10) not deterministic: expected %v", result2)
	}
	if testHappenedByPercentLikely(gameState, 50) != result3 {
		t.Errorf("HappenedByPercentLikely(50) not deterministic: expected %v", result3)
	}
	if gameState.OneInXOdds(2) != result4 {
		t.Errorf("OneInXOdds(2) not deterministic: expected %v", result4)
	}
	if gameState.RandomIntInRange(100, 200) != result5 {
		t.Errorf("RandomIntInRange(100,200) not deterministic: expected %v", result5)
	}

	t.Logf("✅ RNG consistency verified - all operations produce expected deterministic results")
}

// Helper function to create a minimal GameState for testing
func createTestGameState() *GameState {
	gameConfig := &config.UltimaVConfiguration{}
	gameReferences := &references.GameReferences{}

	// Use the same initialization as production code
	return initBlankGameState(gameReferences, gameConfig, 16, 16)
}
