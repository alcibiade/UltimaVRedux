package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

const (
	originalPlayerShape = 0x1c
	originalNPCShape    = 0x40
)

type originalCreatureMovementClass uint8

const (
	originalCreatureMovementOnFoot originalCreatureMovementClass = iota
	originalCreatureMovementWater
	originalCreatureMovementFlying
	originalCreatureMovementEther
	originalCreatureMovementSwamp
	originalCreatureMovementGrass
	originalCreatureMovementDeepWater
	originalCreatureMovementDesert
	originalCreatureMovementRejected
)

// OriginalMonsterShape converts a monster-table index to its original base
// shape. These are not Redux's expanded sprite indices.
func OriginalMonsterShape(monsterType MonsterType) int {
	return originalNPCShape + 4*int(monsterType)
}

// OriginalCreatureTerrainAllowsShape applies shared terrain admission for creature
// and on-foot player shapes. Outdoor movement uses the same rule; ref:ENEMY-0002.
// Callers supply underlying terrain and check occupancy separately. Vehicle
// shapes (including pirate ships) use their own movement predicates.
func OriginalCreatureTerrainAllowsShape(shape int, terrain *Tile) bool {
	if terrain == nil || terrain.Index < 0 || terrain.Index > 0xff {
		return false
	}
	if shape&0xfe != originalPlayerShape && (shape < originalNPCShape || shape > 0xff) {
		return false
	}

	switch originalCreatureMovementClassForShape(shape) {
	case originalCreatureMovementOnFoot:
		return originalCreatureTerrainOnFoot(shape, terrain.Index)
	case originalCreatureMovementWater:
		return originalCreatureTerrainIsWater(terrain.Index)
	case originalCreatureMovementFlying:
		return originalCreatureTerrainIsWater(terrain.Index) ||
			originalCreatureTerrainOnFoot(shape, terrain.Index)
	case originalCreatureMovementEther:
		return !originalCreatureTerrainIsWater(terrain.Index)
	case originalCreatureMovementSwamp:
		return terrain.Index == indexes.Swamp
	case originalCreatureMovementGrass:
		return terrain.Index == indexes.Grass
	case originalCreatureMovementDeepWater:
		return terrain.Index == indexes.Water1
	case originalCreatureMovementDesert:
		return terrain.Index == indexes.Desert1
	case originalCreatureMovementRejected:
		return false
	default:
		panic("unknown original creature movement class")
	}
}

// originalCreatureMovementClassForShape selects the terrain class by four-frame family; ref:ENEMY-0002.
func originalCreatureMovementClassForShape(shape int) originalCreatureMovementClass {
	switch shape >> 2 {
	case 32, 33, 34, 35: // Sea Horse, Squid, Sea Serpent, Shark.
		return originalCreatureMovementWater
	case 29, 31, 37, 44, 54, 55, 60: // Wanderer, Lord British, Bat, Gazer, Daemon, Dragon, Mongbat.
		return originalCreatureMovementFlying
	case 39, 63: // Ghost, Shadowlord.
		return originalCreatureMovementEther
	case 62: // Rot Worm.
		return originalCreatureMovementSwamp
	case 61: // Corpser.
		return originalCreatureMovementGrass
	case 59: // Whirlpool.
		return originalCreatureMovementDeepWater
	case 56: // Sand Trap.
		return originalCreatureMovementDesert
	case 58: // Force Field.
		return originalCreatureMovementRejected
	default:
		return originalCreatureMovementOnFoot
	}
}

func originalCreatureTerrainIsWater(terrain indexes.SpriteIndex) bool {
	return terrain < indexes.Swamp || (terrain&0xf0) == indexes.WaterStream1
}

func originalCreatureTerrainOnFoot(shape int, terrain indexes.SpriteIndex) bool {
	// Foot-passability bitmap; set bits are blocked; ref:TILE-0003.
	legmov := [32]byte{
		0x70, 0x0c, 0x00, 0x28, 0x01, 0xf3, 0x00, 0xbd,
		0x72, 0x3f, 0xff, 0xff, 0xff, 0xcf, 0xff, 0xff,
		0xfc, 0xf6, 0x0f, 0xff, 0xff, 0xc7, 0xff, 0xf7,
		0xf0, 0x3f, 0xff, 0xf3, 0xff, 0xff, 0xff, 0xbe,
	}
	index := int(terrain)
	if legmov[index>>3]&(0x80>>uint(index&7)) != 0 {
		return false
	}

	isPlayer := shape&0xfe == originalPlayerShape
	isFirstHumanClass := shape&0xf0 == originalNPCShape
	isChair := terrain&0xfc == indexes.ChairBackForward

	return !isChair || isPlayer || isFirstHumanClass
}
