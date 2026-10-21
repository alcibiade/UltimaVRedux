package references

import (
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// EnemyReferences is the monster table indexed by MonsterType.
type EnemyReferences []EnemyReference

const (
	nFirstEnemyTileReferenceIndex = 320 // Sea horse
	nFramesPerEnemy               = 4
)

// NewAllEnemyReferences builds the complete creature catalog from inline ratings; ref:ENEMY-0001.
// No DATA.OVL or JSON file is required.
func NewAllEnemyReferences(tiles *Tiles) *EnemyReferences {
	enemyRefs := make(EnemyReferences, 0, nTotalEnemies)

	for i := range nTotalEnemies {
		missile, missileRange := largeMapMissileFromMontype(i)

		spriteIndex := indexes.SpriteIndex(nFirstEnemyTileReferenceIndex + (i * nFramesPerEnemy))

		ref := EnemyReference{
			MonsterType:  MonsterType(i),
			KeyFrameTile: tiles.GetTile(spriteIndex),

			Strength:       monStatsData[i][0],
			Dexterity:      monStatsData[i][1],
			Intelligence:   monStatsData[i][2],
			Armour:         monStatsData[i][3],
			Damage:         monStatsData[i][4],
			HitPoints:      monStatsData[i][5],
			MaxPerMap:      monStatsData[i][6],
			TreasureNumber: monStatsData[i][7],

			EnemyAbilities: abilitiesFromAdmodes(admodes16Data[i]),

			Name:                 monsterNameData[i],
			NamePlural:           monsterIDData[i],
			ActivelyAttacks:      !activelyAttacksExceptions[i],
			CanFlyOverWater:      canFlyOverWaterSet[i],
			CanPassThroughWalls:  canPassThroughWallsSet[i],
			DoNotMove:            doNotMoveSet[i],
			HasWhirlpoolMovement: hasWhirlpoolMovementSet[i],
			IsSandEnemy:          isSandEnemySet[i],
			IsWaterEnemy:         isWaterEnemySet[i],
			LargeMapMissile:      missile,
			LargeMapMissileRange: missileRange,

			AttackRange: int(monRangeData[i]),
			Friend:      nil,
		}

		enemyRefs = append(enemyRefs, ref)
	}

	// Second pass: populate friend pointers
	for i := range nTotalEnemies {
		enemyRefs[i].Friend = &enemyRefs[friendsData[i]]
	}

	return &enemyRefs
}
