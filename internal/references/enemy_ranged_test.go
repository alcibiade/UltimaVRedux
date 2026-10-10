package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestEnemyReferencesPreserveOriginalRangedWeapons(t *testing.T) {
	enemies := NewAllEnemyReferences(NewTileReferences())
	require.NotNil(t, enemies)
	tests := []struct {
		monster MonsterType
		range_  int
		missile MissileType
	}{
		{monster: MonsterBard, range_: 3, missile: MissileArrow},
		{monster: MonsterGuard, range_: 15, missile: MissileAxe},
		{monster: MonsterWanderer, range_: 9, missile: MissileBlue},
		{monster: MonsterBlackthorn, range_: 9, missile: MissileRed},
		{monster: MonsterLordBritish, range_: 9, missile: MissileBlue},
		{monster: MonsterSeaHorse, range_: 5, missile: MissileBlue},
		{monster: MonsterSquid, range_: 7, missile: MissileBlue},
		{monster: MonsterSeaSerpent, range_: 9, missile: MissileRed},
		{monster: MonsterReaper, range_: 9, missile: MissileBlue},
		{monster: MonsterGargoyle, range_: 9, missile: MissileRock},
		{monster: MonsterEttin, range_: 5, missile: MissileRock},
		{monster: MonsterTroll, range_: 5, missile: MissileAxe},
		{monster: MonsterMage, range_: 7, missile: MissileBlue},
		{monster: MonsterMimic, range_: 2, missile: MissileGreen},
		{monster: MonsterGazer, range_: 5, missile: MissileViolet},
		{monster: MonsterSnake, range_: 3, missile: MissileGreen},
		{monster: MonsterDevil, range_: 9, missile: MissileRed},
		{monster: MonsterDragon, range_: 9, missile: MissileRed},
		{monster: MonsterShadowlord, range_: 9, missile: MissileRed},
	}
	rangedCount := 0
	for _, reference := range *enemies {
		if reference.AttackRange > 1 {
			rangedCount++
		}
	}
	require.Len(t, tests, rangedCount, "every ranged enemy reference must have an explicit expectation")

	for _, test := range tests {
		t.Run(test.monster.Name(), func(t *testing.T) {
			reference := (*enemies)[test.monster]
			assert.Equal(t, test.monster, reference.MonsterType)
			assert.Equal(t, test.range_, reference.AttackRange)
			assert.Equal(t, test.range_, reference.LargeMapMissileRange)
			assert.Equal(t, test.missile, reference.LargeMapMissile)
		})
	}
}

func TestPirateShipReferenceUsesCannonBallAtRangeFour(t *testing.T) {
	tiles := NewTileReferences()
	reference := NewPirateShipEnemyReference(tiles.GetTile(indexes.PirateShipUp))

	assert.Equal(t, MonsterStocks, reference.MonsterType)
	assert.Equal(t, MissileCannonBall, reference.LargeMapMissile)
	assert.Equal(t, 4, reference.LargeMapMissileRange)
	assert.Equal(t, reference.LargeMapMissileRange, reference.AttackRange)
}
