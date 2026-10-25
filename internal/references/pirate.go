package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// PirateShipMaxHP is the original overworld pirate ship hull strength.
const PirateShipMaxHP = 100

// IsPirateShipSprite reports whether a sprite is one of the four pirate facings.
func IsPirateShipSprite(sprite indexes.SpriteIndex) bool {
	return sprite == indexes.PirateShipUp ||
		sprite == indexes.PirateShipRight ||
		sprite == indexes.PirateShipDown ||
		sprite == indexes.PirateShipLeft
}

// PirateShipSprite returns the pirate facing for direction; no direction faces down.
func PirateShipSprite(direction Direction) indexes.SpriteIndex {
	switch direction {
	case Up:
		return indexes.PirateShipUp
	case Right:
		return indexes.PirateShipRight
	case Left:
		return indexes.PirateShipLeft
	case Down, NoneDirection:
		return indexes.PirateShipDown
	default:
		return indexes.PirateShipDown
	}
}

// NewPirateShipEnemyReference creates the special overworld pirate representation.
// MonsterStocks is the original monster-table slot reused for pirate ships.
func NewPirateShipEnemyReference(tile *Tile) EnemyReference {
	if tile == nil || !IsPirateShipSprite(tile.Index) {
		panic("pirate ship requires a pirate ship tile")
	}

	return EnemyReference{
		MonsterType:          MonsterStocks,
		KeyFrameTile:         tile,
		Armour:               0,
		Damage:               8,
		Dexterity:            15,
		HitPoints:            PirateShipMaxHP,
		Intelligence:         10,
		MaxPerMap:            0,
		Strength:             15,
		TreasureNumber:       0,
		EnemyAbilities:       map[EnemyAbility]bool{NoCorpse: true},
		Name:                 "Pirate Ship",
		NamePlural:           "Pirate Ships",
		ActivelyAttacks:      true,
		CanFlyOverWater:      false,
		CanPassThroughWalls:  false,
		DoNotMove:            false,
		HasWhirlpoolMovement: false,
		IsSandEnemy:          false,
		IsWaterEnemy:         true,
		LargeMapMissile:      MissileCannonBall,
		LargeMapMissileRange: 4,
		AttackRange:          4,
		Friend:               nil,
	}
}
