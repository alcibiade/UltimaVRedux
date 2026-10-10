package references

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestOriginalCreatureTerrainMatchesSourceForEveryMonsterAndTerrain(t *testing.T) {
	enemies := NewAllEnemyReferences(NewTileReferences())
	for monster := range 48 {
		shape := 0x40 + 4*monster
		assert.Equal(t, shape, OriginalMonsterShape(MonsterType(monster)))
		for terrain := range 256 {
			tile := &Tile{Index: indexes.SpriteIndex(terrain)}
			want := sourceCreatureLegalMove(shape, terrain)
			for frame := range 4 {
				assert.Equal(t, want, OriginalCreatureTerrainAllowsShape(shape+frame, tile),
					"monster=%d frame=%d terrain=%02x", monster, frame, terrain)
			}
			assert.Equal(t, want, (*enemies)[monster].CanMoveToTile(tile),
				"production reference monster=%d terrain=%02x", monster, terrain)
		}
	}
}

// Independent terrain oracle: original shape classes and foot-passability bytes,
// using raw tile IDs (ref:ENEMY-0002 and ref:TILE-0003). Do not call production movement helpers or
// infer the expected class from Redux flags or sprite names.
func sourceCreatureLegalMove(shape, terrain int) bool {
	legtab := [64]int{
		0, 0, 0, 0, 3, 2, 0, 0, 6, 6, 5, 6, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 2,
		1, 1, 1, 1, 0, 2, 0, 4, 0, 0, 0, 0, 2, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 2, 2, 10, 0, -1, 9, 2, 8, 7, 4,
	}
	legmov := [32]byte{
		0x70, 0x0c, 0x00, 0x28, 0x01, 0xf3, 0x00, 0xbd,
		0x72, 0x3f, 0xff, 0xff, 0xff, 0xcf, 0xff, 0xff,
		0xfc, 0xf6, 0x0f, 0xff, 0xff, 0xc7, 0xff, 0xf7,
		0xf0, 0x3f, 0xff, 0xf3, 0xff, 0xff, 0xff, 0xbe,
	}
	onFoot := legmov[terrain/8]&(1<<uint(7-terrain%8)) == 0
	if shape&0xfe != 0x1c && shape&0xf0 != 0x40 && terrain&0xfc == 0x90 {
		onFoot = false
	}
	water := terrain < 4 || terrain >= 0x60 && terrain <= 0x6f
	switch legtab[shape/4] {
	case 0:
		return onFoot
	case 1:
		return water
	case 2:
		return water || onFoot
	case 4:
		return !water
	case 7:
		return terrain == 4
	case 8:
		return terrain == 5
	case 9:
		return terrain == 1
	case 10:
		return terrain == 7
	default:
		return false
	}
}

func TestOriginalCreatureTerrainPlayerAndInvalidInputs(t *testing.T) {
	for _, shape := range []int{0x1c, 0x1d} {
		for terrain := range 256 {
			assert.Equal(t, sourceCreatureLegalMove(shape, terrain),
				OriginalCreatureTerrainAllowsShape(shape, &Tile{Index: indexes.SpriteIndex(terrain)}))
		}
	}
	for _, terrain := range []*Tile{nil, {Index: -1}, {Index: 256}, {Index: indexes.Orc1}} {
		assert.False(t, OriginalCreatureTerrainAllowsShape(0x9c, terrain))
	}
	for _, shape := range []int{-1, 0x10, 0x3c, 0x100} {
		assert.False(t, OriginalCreatureTerrainAllowsShape(shape, &Tile{Index: indexes.Grass}))
	}
}

func TestOriginalCreatureTerrainMetadata(t *testing.T) {
	enemies := NewAllEnemyReferences(NewTileReferences())
	for monster, enemy := range *enemies {
		wantFlying := monster == 13 || monster == 15 || monster == 21 || monster == 28 ||
			monster == 38 || monster == 39 || monster == 44
		assert.Equal(t, wantFlying, enemy.CanFlyOverWater, "monster=%d", monster)
		assert.Equal(t, monster == 23 || monster == 47, enemy.CanPassThroughWalls, "monster=%d", monster)
	}
	assert.False(t, (*enemies)[MonsterCrawler].DoNotMove,
		"tactical pursuit excludes Reaper and Mimic, not Crawler (ref:ENEMY-0002)")
	assert.True(t, (*enemies)[MonsterReaper].DoNotMove)
	assert.True(t, (*enemies)[MonsterMimic].DoNotMove)
}

func TestOriginalCreatureTerrainKeepsPirateVehicleVariant(t *testing.T) {
	tiles := NewTileReferences()
	stocks := (*NewAllEnemyReferences(tiles))[MonsterStocks]
	for _, facing := range []indexes.SpriteIndex{
		indexes.PirateShipUp, indexes.PirateShipRight, indexes.PirateShipDown, indexes.PirateShipLeft,
	} {
		pirate := NewPirateShipEnemyReference(tiles.GetTile(facing))
		for _, test := range []struct {
			tile indexes.SpriteIndex
			want bool
		}{
			{indexes.Water1, true}, {indexes.Water2, true}, {indexes.WaterCoast, false},
			{indexes.WaterStream1, false}, {indexes.Grass, false},
		} {
			assert.Equal(t, test.want, pirate.CanMoveToTile(tiles.GetTile(test.tile)),
				"facing=%d terrain=%d", facing, test.tile)
		}
		assert.False(t, pirate.CanMoveToTile(nil))
	}
	assert.True(t, stocks.CanMoveToTile(tiles.GetTile(indexes.Grass)))
	assert.False(t, stocks.CanMoveToTile(tiles.GetTile(indexes.Water1)))
}
