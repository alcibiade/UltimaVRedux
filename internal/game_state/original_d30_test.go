package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/stretchr/testify/assert"
)

func TestOriginalD30ExactSourceMapping(t *testing.T) {
	counts := make(map[int]int, 30)
	for sourceValue := range originalD30SourceValueCount {
		result := originalD30FromSourceValue(sourceValue)
		assert.GreaterOrEqual(t, result, 1)
		assert.LessOrEqual(t, result, 30)
		counts[result]++
	}

	assert.Equal(t, 4, counts[1])
	for result := 2; result <= 29; result++ {
		assert.Equal(t, 2, counts[result], "result %d", result)
	}
	assert.Equal(t, 1, counts[30])
	assert.Equal(t, 1, originalD30FromSourceValue(0))
	assert.Equal(t, 1, originalD30FromSourceValue(3))
	assert.Equal(t, 2, originalD30FromSourceValue(4))
	assert.Equal(t, 29, originalD30FromSourceValue(59))
	assert.Equal(t, 30, originalD30FromSourceValue(60))
}

func TestRollOriginalD30UsesOneDeterministicSourceDraw(t *testing.T) {
	const seed = uint64(448)
	state := createTestGameState()
	state.SetRandomSeed(seed)
	wantSource := deterministic_random.New(seed)

	for range 20 {
		callsBefore := state.GetRNG().Calls()
		want := originalD30FromSourceValue(wantSource.Intn(originalD30SourceValueCount))

		assert.Equal(t, want, state.RollOriginalD30())
		assert.Equal(t, callsBefore+1, state.GetRNG().Calls())
	}
}

func TestUniformD30RemainsDistinctFromOriginalD30(t *testing.T) {
	const seed = uint64(445448)
	uniform := createTestGameState()
	original := createTestGameState()
	uniform.SetRandomSeed(seed)
	original.SetRandomSeed(seed)

	wantUniform := deterministic_random.New(seed).Intn(30) + 1
	assert.Equal(t, wantUniform, uniform.RandomIntInRange(1, 30))
	assert.Equal(t, originalD30FromSourceValue(deterministic_random.New(seed).Intn(61)), original.RollOriginalD30())
	assert.Equal(t, uint64(1), uniform.GetRNG().Calls())
	assert.Equal(t, uint64(1), original.GetRNG().Calls())
}
