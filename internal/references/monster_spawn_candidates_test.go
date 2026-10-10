package references

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type candidateSelectionRNG struct {
	t     *testing.T
	calls []int
}

func (r *candidateSelectionRNG) Intn(n int) int {
	r.t.Helper()
	r.calls = append(r.calls, n)
	return 0
}

func TestMonsterSpawnCandidatesRejectPassableRoadsEntrancesAndParapets(t *testing.T) {
	// Candidate selection is independent of genprob's player-position odds:
	// even a road with a nighttime generation bonus is not a spawn candidate.
	tiles := NewTileReferences()
	for _, tileRange := range [][2]int{{0x10, 0x26}, {0x50, 0x53}, {0x130, 0x133}} {
		for index := tileRange[0]; index <= tileRange[1]; index++ {
			for _, underworld := range []bool{false, true} {
				t.Run(fmt.Sprintf("tile=%02x/underworld=%t", index, underworld), func(t *testing.T) {
					rng := &candidateSelectionRNG{t: t}
					assert.Equal(t, MonsterNone, PickMonsterByTerrain(tiles.GetTile(indexes.SpriteIndex(index)), underworld, rng))
					assert.Empty(t, rng.calls, "rejected candidate does not roll pool selection")
				})
			}
		}
	}
}

func TestMonsterSpawnCandidatesAdmitOriginalBeachRange(t *testing.T) {
	for index := 0x30; index <= 0x33; index++ {
		for _, underworld := range []bool{false, true} {
			rng := &candidateSelectionRNG{t: t}
			want := MonsterOrc
			if underworld {
				want = MonsterBat
			}
			assert.Equal(t, want, PickMonsterByTerrain(&Tile{Index: indexes.SpriteIndex(index)}, underworld, rng),
				"tile=%02x underworld=%t", index, underworld)
			assert.Equal(t, []int{256}, rng.calls)
		}
	}
}

func TestMonsterSpawnWaterCandidatesRemainDistinctFromMovementWater(t *testing.T) {
	for _, tileRange := range [][2]int{{0x60, 0x6f}, {0xd4, 0xd7}, {0xe4, 0xe7}} {
		for index := tileRange[0]; index <= tileRange[1]; index++ {
			rng := &candidateSelectionRNG{t: t}
			assert.Equal(t, MonsterShark, PickMonsterByTerrain(&Tile{Index: indexes.SpriteIndex(index)}, false, rng),
				"spawn water tile=%02x", index)
			assert.Equal(t, []int{65, 256}, rng.calls)
		}
	}
}

func TestMonsterSpawnCandidatesUseOriginalSeaRange(t *testing.T) {
	for index := range 4 {
		rng := &candidateSelectionRNG{t: t}
		assert.Equal(t, MonsterShark, PickMonsterByTerrain(&Tile{Index: indexes.SpriteIndex(index)}, false, rng))
		wantCalls := []int{65, 256}
		if index == 1 {
			wantCalls = []int{65, 8, 256}
		}
		assert.Equal(t, wantCalls, rng.calls)
	}
}
