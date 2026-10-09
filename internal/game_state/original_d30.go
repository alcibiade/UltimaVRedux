package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

const originalD30SourceValueCount = 61

// originalD30FromSourceValue maps a bounded 0..60 draw through integer
// halving and the original zero-result replacement. The original generator
// distribution is not independently established. ref:RNG-0001
func originalD30FromSourceValue(sourceValue int) int {
	if sourceValue < 0 || sourceValue >= originalD30SourceValueCount {
		panic("original d30 source value is outside 0..60")
	}
	result := sourceValue / 2
	if result == 0 {
		return 1
	}

	return result
}

func rollOriginalD30(rng references.RandomProvider) int {
	return originalD30FromSourceValue(rng.Intn(originalD30SourceValueCount))
}

// RollOriginalD30 applies the transformed-d30 mapping only at reviewed
// call sites; the current provider controls reproducibility. ref:RNG-0001
func (g *GameState) RollOriginalD30() int {
	return originalD30FromSourceValue(g.rng.Intn(originalD30SourceValueCount))
}
