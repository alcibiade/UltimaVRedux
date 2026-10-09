package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type waterGenerationRNG struct {
	t      *testing.T
	values []int
	bounds []int
}

func (r *waterGenerationRNG) Intn(n int) int {
	r.t.Helper()
	require.NotEmpty(r.t, r.values)
	value := r.values[0]
	r.values = r.values[1:]
	r.bounds = append(r.bounds, n)
	require.Less(r.t, value, n)
	return value
}

func TestWaterGenerationUsesSurfaceOnlyWhirlpoolRoll(t *testing.T) {
	for _, test := range []struct {
		name       string
		underworld bool
		tile       indexes.SpriteIndex
		values     []int
		bounds     []int
		want       MonsterType
	}{
		{name: "surface deep whirlpool", tile: indexes.Water1, values: []int{0, 7}, bounds: []int{65, 8}, want: MonsterWhirlpool},
		{name: "surface deep other", tile: indexes.Water1, values: []int{0, 6, 0}, bounds: []int{65, 8, 256}, want: SurfaceWaterMonsters[0]},
		{name: "surface shallow", tile: indexes.Water2, values: []int{0, 0}, bounds: []int{65, 256}, want: SurfaceWaterMonsters[0]},
		{name: "underworld deep", underworld: true, tile: indexes.Water1, values: []int{0, 0}, bounds: []int{65, 256}, want: UnderworldWaterMonsters[0]},
		{name: "underworld shallow", underworld: true, tile: indexes.Water2, values: []int{0, 0}, bounds: []int{65, 256}, want: UnderworldWaterMonsters[0]},
		{name: "inclusive upper frequency bound", tile: indexes.Water1, values: []int{64}, bounds: []int{65}, want: MonsterNone},
		{name: "frequency rejection boundary", tile: indexes.Water1, values: []int{16}, bounds: []int{65}, want: MonsterNone},
	} {
		t.Run(test.name, func(t *testing.T) {
			rng := &waterGenerationRNG{t: t, values: test.values}
			assert.Equal(t, test.want, PickMonsterByTerrain(&Tile{Index: test.tile}, test.underworld, rng))
			assert.Equal(t, test.bounds, rng.bounds)
			assert.Empty(t, rng.values)
		})
	}
}
