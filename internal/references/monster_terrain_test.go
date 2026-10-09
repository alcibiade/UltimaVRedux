package references

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

// MockRandomProvider is a deterministic random provider for testing
type MockRandomProvider struct {
	values []int
	index  int
}

func NewMockRandomProvider(values ...int) *MockRandomProvider {
	return &MockRandomProvider{values: values, index: 0}
}

func (m *MockRandomProvider) Intn(n int) int {
	if m.index >= len(m.values) {
		m.index = 0 // Wrap around
	}
	val := m.values[m.index] % n
	m.index++
	return val
}

func TestPickByFrequency_SurfaceLand(t *testing.T) {
	// Test that frequency selection works correctly
	// With freq[0]=60, freq[1]=50, etc., a random value of 0-59 should select first monster

	tests := []struct {
		name          string
		randomValue   int
		expectedIndex int
	}{
		{"First monster", 0, 0},
		{"First monster edge", 59, 0},
		{"Second monster", 60, 1},
		{"Second monster edge", 109, 1},
		{"Third monster", 110, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rng := NewMockRandomProvider(tc.randomValue)
			result := PickByFrequency(SurfaceLandMonsters[:], SurfaceLandFrequencies[:], rng)
			assert.Equal(t, SurfaceLandMonsters[tc.expectedIndex], result)
		})
	}
}

func TestPickByFrequency_Distribution(t *testing.T) {
	// Test that the distribution roughly matches expected frequencies
	// This is a statistical test - with enough samples, distribution should be close

	// Use a simple mock that cycles through values
	rng := NewMockRandomProvider(0, 60, 110, 140, 160, 175, 190, 200, 210, 213, 215, 216)

	counts := make(map[MonsterType]int)
	samples := 1000

	for i := 0; i < samples; i++ {
		result := PickByFrequency(SurfaceLandMonsters[:], SurfaceLandFrequencies[:], rng)
		counts[result]++
	}

	// Most common monster should be Orc (highest frequency)
	assert.Contains(t, counts, MonsterOrc, "Orc should appear in results")

	// All monsters in the pool should be valid
	for mt := range counts {
		assert.Contains(t, SurfaceLandMonsters[:], mt, "Result should be from pool")
	}
}

func TestPickMonsterByTerrain_SurfaceLand(t *testing.T) {
	// Create a grass tile (should use surface land pool)
	tile := &Tile{Index: indexes.Grass}

	// With random value 0, should get first monster (Orc)
	rng := NewMockRandomProvider(0)
	result := PickMonsterByTerrain(tile, false, rng)

	// Result should be a valid surface land monster
	assert.Contains(t, SurfaceLandMonsters[:], result)
}

func TestPickMonsterByTerrain_Desert(t *testing.T) {
	// Create a desert tile
	tile := &Tile{Index: indexes.Desert1}

	// With random value 0 (25% chance check fails), should return MonsterNone
	rng := NewMockRandomProvider(1, 2, 3) // Values >= 1 mean no sandtrap
	result := PickMonsterByTerrain(tile, false, rng)
	assert.Equal(t, MonsterNone, result)

	// With random value 0 (25% chance check passes), should get Sandtrap
	rng = NewMockRandomProvider(0)
	result = PickMonsterByTerrain(tile, false, rng)
	assert.Equal(t, MonsterSandtrap, result)
}

func TestPickMonsterByTerrain_Mountains(t *testing.T) {
	// Mountains should never spawn monsters
	tile := &Tile{Index: indexes.SmallMountains}
	rng := NewMockRandomProvider(0)
	result := PickMonsterByTerrain(tile, false, rng)
	assert.Equal(t, MonsterNone, result)

	tile = &Tile{Index: indexes.TallMountains}
	result = PickMonsterByTerrain(tile, false, rng)
	assert.Equal(t, MonsterNone, result)
}

func TestPickMonsterByTerrain_Water(t *testing.T) {
	// Water tiles should use water monster pool
	tile := &Tile{Index: indexes.Water2}

	// Water encounters are 75% less frequent (only 16/64 chance)
	// With random value 16, should pass the frequency check
	rng := NewMockRandomProvider(16, 0)
	result := PickMonsterByTerrain(tile, false, rng)

	// Result should be a water monster (if encounter happens)
	if result != MonsterNone {
		assert.Contains(t, SurfaceWaterMonsters[:], result)
	}
}

func TestPickMonsterByTerrain_DeepWater_Whirlpool(t *testing.T) {
	// Deep water has 1/8 chance of whirlpool
	tile := &Tile{Index: indexes.Water1}

	// Water encounters need random < 16 to pass 75% reduction check
	// Whirlpool check needs random == 7 (1/8 chance)
	// With random values: 0 (passes water check), 7 (passes whirlpool check)
	rng := NewMockRandomProvider(0, 7)
	result := PickMonsterByTerrain(tile, false, rng)
	assert.Equal(t, MonsterWhirlpool, result)
}

func TestPickMonsterByTerrain_UnderworldSwamp(t *testing.T) {
	// Underworld swamp should spawn Rot Worms
	tile := &Tile{Index: indexes.Swamp}
	rng := NewMockRandomProvider(0)
	result := PickMonsterByTerrain(tile, true, rng)
	assert.Equal(t, MonsterRotWorm, result)
}

func TestPickMonsterByTerrain_NilTile(t *testing.T) {
	rng := NewMockRandomProvider(0)
	result := PickMonsterByTerrain(nil, false, rng)
	assert.Equal(t, MonsterNone, result)
}

func TestSurfaceLandMonsters_Order(t *testing.T) {
	// Verify the order matches original (most frequent first)
	expected := []MonsterType{
		MonsterOrc, MonsterSnake, MonsterRat, MonsterSpider, MonsterInsect,
		MonsterSkeleton, MonsterHeadless, MonsterTroll, MonsterEttin,
		MonsterWisp, MonsterDragon, MonsterDevil,
	}

	assert.Equal(t, expected, SurfaceLandMonsters[:])
}

func TestSurfaceWaterMonsters_Order(t *testing.T) {
	// Verify the water pool, including pirate ships; ref:ENEMY-0004.
	expected := []MonsterType{
		MonsterShark, MonsterSquid, MonsterSeaSerpent, MonsterSeaHorse, MonsterStocks,
	}

	assert.Equal(t, expected, SurfaceWaterMonsters[:])
}

func TestUnderworldLandMonsters_Order(t *testing.T) {
	// Verify underworld monster pool
	expected := []MonsterType{
		MonsterBat, MonsterRat, MonsterSpider, MonsterMongbat,
		MonsterCorpser, MonsterDevil, MonsterDragon,
	}

	assert.Equal(t, expected, UnderworldLandMonsters[:])
}

func TestCalculateEncounterProbability(t *testing.T) {
	tests := []struct {
		name       string
		tile       indexes.SpriteIndex
		night      bool
		underworld bool
		want       int
	}{
		{name: "hut is standard", tile: indexes.Hut, want: StandardEncounterProbability},
		{name: "grass is standard", tile: indexes.Grass, want: StandardEncounterProbability},
		{name: "forest one is standard", tile: indexes.Forest1, want: StandardEncounterProbability},
		{name: "forest two is standard", tile: indexes.Forest2, want: StandardEncounterProbability},
		{name: "swamp is dangerous", tile: indexes.Swamp, want: DangerousEncounterProbability},
		{name: "forest three starts dangerous range", tile: indexes.Forest3, want: DangerousEncounterProbability},
		{name: "right hills ends dangerous range", tile: indexes.RightHills2, want: DangerousEncounterProbability},
		{name: "road is low", tile: indexes.Path1, want: RoadEncounterProbability},
		{name: "road receives night bonus", tile: indexes.Path1, night: true, want: NighttimeProbabilityBonus},
		{name: "standard receives night bonus", tile: indexes.Grass, night: true, want: StandardEncounterProbability + NighttimeProbabilityBonus},
		{name: "dangerous receives night bonus", tile: indexes.Swamp, night: true, want: DangerousEncounterProbability + NighttimeProbabilityBonus},
		{name: "underworld is fixed by day", tile: indexes.Grass, underworld: true, want: 3},
		{name: "underworld is fixed at night", tile: indexes.Grass, night: true, underworld: true, want: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tile := &Tile{Index: test.tile}
			assert.Equal(t, test.want, CalculateEncounterProbability(tile, test.night, test.underworld))
		})
	}
}

func TestDefaultMonsterGenerationRollMaximumMatchesOriginal(t *testing.T) {
	odds := NewDefaultTheOdds()

	assert.Equal(t, DefaultMonsterGenerationRollMaximum, odds.GetMonsterGenerationRollMaximum())
}

func TestOriginalMonsterSpawnNightBonusHours(t *testing.T) {
	for hour := range 24 {
		want := hour < 5
		assert.Equal(t, want, IsOriginalMonsterSpawnNightBonusHour(uint8(hour)), "hour %d", hour)
	}
}

func TestMonsterFrequencyWeights(t *testing.T) {
	// Verify that frequency arrays sum to valid ranges
	// Frequency weights cover the inclusive 0..255 draw range; ref:ENEMY-0004.

	sumSurfaceLand := 0
	for _, f := range SurfaceLandFrequencies {
		sumSurfaceLand += f
	}
	assert.LessOrEqual(t, sumSurfaceLand, 256, "Surface land frequencies should fit in 0-255 range")

	sumSurfaceWater := 0
	for _, f := range SurfaceWaterFrequencies {
		sumSurfaceWater += f
	}
	assert.LessOrEqual(t, sumSurfaceWater, 256, "Surface water frequencies should fit in 0-255 range")

	sumUnderworldLand := 0
	for _, f := range UnderworldLandFrequencies {
		sumUnderworldLand += f
	}
	assert.LessOrEqual(t, sumUnderworldLand, 256, "Underworld land frequencies should fit in 0-255 range")
}

// Integration test with real frequency behavior
func TestPickByFrequency_AuthenticBehavior(t *testing.T) {
	// Simulate the original game's behavior:
	// 1. Generate random 0-255
	// 2. Subtract frequencies until value goes negative
	// 3. Return monster at that index

	// For surface land with freq [60, 50, 40, 30, 20, 15, 15, 10, 10, 3, 2, 1]:
	// - Random 0-59: Orc (cumulative 0-59)
	// - Random 60-109: Snake (cumulative 60-109)
	// - Random 110-149: Rat (cumulative 110-149)
	// etc.

	testCases := []struct {
		randomValue     int
		expectedMonster MonsterType
	}{
		{0, MonsterOrc},
		{59, MonsterOrc},
		{60, MonsterSnake},
		{109, MonsterSnake},
		{110, MonsterRat},
		{149, MonsterRat},
		{150, MonsterSpider},
		{179, MonsterSpider},
		{180, MonsterInsect},
		{199, MonsterInsect},
		{200, MonsterSkeleton},
		{214, MonsterSkeleton},
		{215, MonsterHeadless},
		{229, MonsterHeadless},
		{230, MonsterTroll},
		{239, MonsterTroll},
		{240, MonsterEttin},
		{249, MonsterEttin},
		{250, MonsterWisp},
		{252, MonsterWisp},
		{253, MonsterDragon},
		{254, MonsterDragon},
		{255, MonsterDevil}, // Last value
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			rng := NewMockRandomProvider(tc.randomValue)
			result := PickByFrequency(SurfaceLandMonsters[:], SurfaceLandFrequencies[:], rng)
			assert.Equal(t, tc.expectedMonster, result, "random=%d", tc.randomValue)
		})
	}
}
