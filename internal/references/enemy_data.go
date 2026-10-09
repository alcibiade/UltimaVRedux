package references

// enemy_data.go — Inline original Ultima V creature reference data.
//
// Creature ratings, abilities, and identities follow the catalog in ref:ENEMY-0001.
// Do NOT edit these values without cross-referencing against the originals.
//
// References:
//   - Base attributes, armor, damage, HP, appearance bound, and treasure rating
//   - 16-bit ability words with the meanings documented in ref:ENEMY-0001
//   - Attack range; 1 means melee only → MissileNone
//   - Missile index: 0=Arrow … 7=Rock; maps to MissileType
//   - Singular and group display names
//   - MonsterType index of each creature's companion candidate
//
// Terrain-specific spawn selection is described in ref:ENEMY-0004.

// nTotalEnemies is the total number of creature catalog entries; ref:ENEMY-0001.
const nTotalEnemies = 48

// monStatsData holds [str, dex, int, arm, dmg, hp, num, treas] per monster index.
// Order matches the creature index in ref:ENEMY-0001.
//
//nolint:gochecknoglobals
var monStatsData = [nTotalEnemies][8]int{
	/* 0  Mage       */ {10, 15, 20, 0, 15, 10, 3, 20},
	/* 1  Bard       */ {15, 20, 10, 4, 12, 15, 9, 10},
	/* 2  Fighter    */ {20, 15, 10, 8, 15, 20, 6, 15},
	/* 3  Avatar     */ {25, 25, 25, 7, 30, 20, 1, 25},
	/* 4  Villager   */ {12, 12, 12, 0, 6, 8, 1, 10},
	/* 5  Merchant   */ {12, 12, 18, 0, 6, 8, 1, 10},
	/* 6  Jester     */ {12, 18, 12, 0, 6, 8, 1, 10},
	/* 7  BanjoPlayer*/ {12, 16, 14, 0, 6, 8, 1, 10},
	/* 8  Stocks     */ {12, 12, 12, 0, 0, 5, 1, 0},
	/* 9  Manacles   */ {12, 12, 12, 0, 0, 5, 1, 0},
	/* 10 Child      */ {8, 8, 8, 0, 0, 5, 1, 0},
	/* 11 Beggar     */ {8, 8, 8, 0, 0, 5, 1, 0},
	/* 12 Guard      */ {22, 30, 10, 6, 30, 99, 8, 5},
	/* 13 Wanderer   */ {30, 30, 30, 30, 99, 99, 1, 0},
	/* 14 Blackthorn */ {30, 30, 30, 30, 30, 99, 1, 0},
	/* 15 LordBritish*/ {30, 30, 30, 30, 99, 99, 1, 0},
	/* 16 SeaHorse   */ {17, 20, 20, 2, 10, 30, 3, 0},
	/* 17 Squid      */ {24, 20, 8, 0, 20, 50, 2, 0},
	/* 18 SeaSerpent */ {17, 17, 8, 2, 30, 70, 1, 0},
	/* 19 Shark      */ {20, 17, 5, 0, 8, 22, 10, 0},
	/* 20 Rat        */ {5, 20, 5, 0, 6, 10, 10, 5},
	/* 21 Bat        */ {5, 30, 5, 0, 6, 5, 16, 0},
	/* 22 Spider     */ {10, 10, 5, 0, 8, 10, 4, 5},
	/* 23 Ghost      */ {1, 20, 10, 0, 12, 20, 6, 0},
	/* 24 Slime      */ {6, 6, 2, 0, 4, 10, 16, 0},
	/* 25 Gremlin    */ {10, 21, 10, 2, 4, 10, 13, 12},
	/* 26 Mimic      */ {20, 30, 12, 3, 15, 30, 1, 20},
	/* 27 Reaper     */ {20, 25, 12, 4, 20, 40, 3, 25},
	/* 28 Gazer      */ {8, 10, 25, 0, 10, 20, 4, 0},
	/* 29 Crawler    */ {17, 15, 12, 0, 15, 35, 4, 0},
	/* 30 Gargoyle   */ {20, 10, 5, 15, 20, 40, 1, 0},
	/* 31 Insect     */ {1, 30, 1, 0, 4, 5, 10, 0},
	/* 32 Orc        */ {15, 13, 10, 2, 12, 10, 10, 11},
	/* 33 Skeleton   */ {10, 20, 5, 0, 12, 20, 8, 13},
	/* 34 Snake      */ {5, 18, 8, 1, 8, 10, 4, 0},
	/* 35 Ettin      */ {20, 15, 12, 3, 15, 30, 6, 17},
	/* 36 Headless   */ {19, 12, 8, 2, 12, 20, 8, 12},
	/* 37 Wisp       */ {8, 30, 20, 0, 20, 40, 4, 0},
	/* 38 Devil      */ {25, 25, 25, 5, 20, 75, 4, 0},
	/* 39 Dragon     */ {30, 25, 25, 10, 30, 99, 2, 30},
	/* 40 SandTrap   */ {25, 25, 5, 10, 30, 80, 1, 25},
	/* 41 Troll      */ {18, 17, 9, 4, 15, 15, 4, 15},
	/* 42 ForceField */ {0, 0, 0, 0, 0, 0, 0, 0},
	/* 43 Whirlpool  */ {0, 0, 0, 0, 0, 0, 0, 0},
	/* 44 Mongbat    */ {10, 30, 15, 4, 20, 20, 16, 5},
	/* 45 Corpser    */ {17, 10, 8, 0, 15, 40, 4, 0},
	/* 46 RotWorm    */ {5, 17, 6, 0, 6, 5, 10, 0},
	/* 47 ShadowLord */ {25, 30, 30, 10, 30, 99, 1, 0},
}

// admodes16Data holds each creature's 16-bit ability word; ref:ENEMY-0001.
// Bit interpretation: see abilitiesFromAdmodes below for full mapping.
//
//nolint:gochecknoglobals
var admodes16Data = [nTotalEnemies]uint16{
	/* 0  Mage       */ 0x8000,
	/* 1  Bard       */ 0x4000,
	/* 2  Fighter    */ 0x0000,
	/* 3  Avatar     */ 0x0000,
	/* 4  Villager   */ 0x0000,
	/* 5  Merchant   */ 0x0000,
	/* 6  Jester     */ 0x0000,
	/* 7  BanjoPlayer*/ 0x0000,
	/* 8  Stocks     */ 0x0000,
	/* 9  Manacles   */ 0x0000,
	/* 10 Child      */ 0x0000,
	/* 11 Beggar     */ 0x0000,
	/* 12 Guard      */ 0x0000,
	/* 13 Wanderer   */ 0xb008,
	/* 14 Blackthorn */ 0xbc48,
	/* 15 LordBritish*/ 0xb008,
	/* 16 SeaHorse   */ 0xc001,
	/* 17 Squid      */ 0x4005,
	/* 18 SeaSerpent */ 0x4001,
	/* 19 Shark      */ 0x0001,
	/* 20 Rat        */ 0x0300,
	/* 21 Bat        */ 0x0001,
	/* 22 Spider     */ 0x0204,
	/* 23 Ghost      */ 0x0821,
	/* 24 Slime      */ 0x0011,
	/* 25 Gremlin    */ 0x0002,
	/* 26 Mimic      */ 0x0280,
	/* 27 Reaper     */ 0x8080,
	/* 28 Gazer      */ 0x8040,
	/* 29 Crawler    */ 0x0000,
	/* 30 Gargoyle   */ 0x0090,
	/* 31 Insect     */ 0x0001,
	/* 32 Orc        */ 0x0080,
	/* 33 Skeleton   */ 0x0020,
	/* 34 Snake      */ 0x0004,
	/* 35 Ettin      */ 0x4080,
	/* 36 Headless   */ 0x0080,
	/* 37 Wisp       */ 0x2041,
	/* 38 Devil      */ 0x8461,
	/* 39 Dragon     */ 0x0400,
	/* 40 SandTrap   */ 0x0000,
	/* 41 Troll      */ 0x4000,
	/* 42 ForceField */ 0x0000,
	/* 43 Whirlpool  */ 0x0000,
	/* 44 Mongbat    */ 0x0000,
	/* 45 Corpser    */ 0x0000,
	/* 46 RotWorm    */ 0x0300,
	/* 47 ShadowLord */ 0xbb61,
}

// monRangeData holds each creature's attack range; ref:ENEMY-0001.
// Value 1 means melee-only (no missile).
//
//nolint:gochecknoglobals
var monRangeData = [nTotalEnemies]byte{
	/* 0  Mage       */ 7,
	/* 1  Bard       */ 3,
	/* 2  Fighter    */ 1,
	/* 3  Avatar     */ 1,
	/* 4  Villager   */ 1,
	/* 5  Merchant   */ 1,
	/* 6  Jester     */ 1,
	/* 7  BanjoPlayer*/ 1,
	/* 8  Stocks     */ 1,
	/* 9  Manacles   */ 1,
	/* 10 Child      */ 1,
	/* 11 Beggar     */ 1,
	/* 12 Guard      */ 15,
	/* 13 Wanderer   */ 9,
	/* 14 Blackthorn */ 9,
	/* 15 LordBritish*/ 9,
	/* 16 SeaHorse   */ 5,
	/* 17 Squid      */ 7,
	/* 18 SeaSerpent */ 9,
	/* 19 Shark      */ 1,
	/* 20 Rat        */ 1,
	/* 21 Bat        */ 1,
	/* 22 Spider     */ 1,
	/* 23 Ghost      */ 1,
	/* 24 Slime      */ 1,
	/* 25 Gremlin    */ 1,
	/* 26 Mimic      */ 2,
	/* 27 Reaper     */ 9,
	/* 28 Gazer      */ 5,
	/* 29 Crawler    */ 1,
	/* 30 Gargoyle   */ 9,
	/* 31 Insect     */ 1,
	/* 32 Orc        */ 1,
	/* 33 Skeleton   */ 1,
	/* 34 Snake      */ 3,
	/* 35 Ettin      */ 5,
	/* 36 Headless   */ 1,
	/* 37 Wisp       */ 1,
	/* 38 Devil      */ 9,
	/* 39 Dragon     */ 9,
	/* 40 SandTrap   */ 1,
	/* 41 Troll      */ 5,
	/* 42 ForceField */ 1,
	/* 43 Whirlpool  */ 1,
	/* 44 Mongbat    */ 1,
	/* 45 Corpser    */ 1,
	/* 46 RotWorm    */ 1,
	/* 47 ShadowLord */ 9,
}

// monTypeData holds each creature's missile type index; ref:ENEMY-0001.
// Index maps directly to MissileType: 0=Arrow, 1=CannonBall, 2=Axe,
// 3=Red, 4=Blue, 5=Green, 6=Violet, 7=Rock.
// Only meaningful when monRangeData[i] > 1 (i.e., has a ranged attack).
//
//nolint:gochecknoglobals
var monTypeData = [nTotalEnemies]byte{
	/* 0  Mage       */ 4, // Blue
	/* 1  Bard       */ 0, // Arrow
	/* 2  Fighter    */ 0,
	/* 3  Avatar     */ 0,
	/* 4  Villager   */ 0,
	/* 5  Merchant   */ 0,
	/* 6  Jester     */ 0,
	/* 7  BanjoPlayer*/ 0,
	/* 8  Stocks     */ 0,
	/* 9  Manacles   */ 0,
	/* 10 Child      */ 0,
	/* 11 Beggar     */ 0,
	/* 12 Guard      */ 2, // Axe
	/* 13 Wanderer   */ 4, // Blue
	/* 14 Blackthorn */ 3, // Red
	/* 15 LordBritish*/ 4, // Blue
	/* 16 SeaHorse   */ 4, // Blue
	/* 17 Squid      */ 4, // Blue
	/* 18 SeaSerpent */ 3, // Red
	/* 19 Shark      */ 0,
	/* 20 Rat        */ 0,
	/* 21 Bat        */ 0,
	/* 22 Spider     */ 0,
	/* 23 Ghost      */ 0,
	/* 24 Slime      */ 0,
	/* 25 Gremlin    */ 0,
	/* 26 Mimic      */ 5, // Green
	/* 27 Reaper     */ 4, // Blue
	/* 28 Gazer      */ 6, // Violet
	/* 29 Crawler    */ 0,
	/* 30 Gargoyle   */ 7, // Rock
	/* 31 Insect     */ 0,
	/* 32 Orc        */ 0,
	/* 33 Skeleton   */ 0,
	/* 34 Snake      */ 5, // Green
	/* 35 Ettin      */ 7, // Rock
	/* 36 Headless   */ 0,
	/* 37 Wisp       */ 0,
	/* 38 Devil      */ 3, // Red
	/* 39 Dragon     */ 3, // Red
	/* 40 SandTrap   */ 0,
	/* 41 Troll      */ 2, // Axe
	/* 42 ForceField */ 0,
	/* 43 Whirlpool  */ 0,
	/* 44 Mongbat    */ 0,
	/* 45 Corpser    */ 0,
	/* 46 RotWorm    */ 0,
	/* 47 ShadowLord */ 3, // Red
}

// monsterNameData holds each creature's singular display name; ref:ENEMY-0001.
// Entries 8, 9, 42, 43 are unnamed in the original.
//
//nolint:gochecknoglobals
var monsterNameData = [nTotalEnemies]string{
	/* 0  */ "Mage",
	/* 1  */ "Bard",
	/* 2  */ "Fighter",
	/* 3  */ "Avatar",
	/* 4  */ "Villager",
	/* 5  */ "Merchant",
	/* 6  */ "Jester",
	/* 7  */ "Bard",
	/* 8  */ "",
	/* 9  */ "",
	/* 10 */ "Child",
	/* 11 */ "Beggar",
	/* 12 */ "Guard",
	/* 13 */ "Wanderer",
	/* 14 */ "Blackthorn",
	/* 15 */ "Lord British",
	/* 16 */ "Sea Horse",
	/* 17 */ "Squid",
	/* 18 */ "Sea Serpent",
	/* 19 */ "Shark",
	/* 20 */ "Giant Rat",
	/* 21 */ "Bat",
	/* 22 */ "Giant Spider",
	/* 23 */ "Ghost",
	/* 24 */ "Slime",
	/* 25 */ "Gremlin",
	/* 26 */ "Mimic",
	/* 27 */ "Reaper",
	/* 28 */ "Gazer",
	/* 29 */ "Crawler",
	/* 30 */ "Gargoyle",
	/* 31 */ "Insect Swarm",
	/* 32 */ "Orc",
	/* 33 */ "Skeleton",
	/* 34 */ "Python",
	/* 35 */ "Ettin",
	/* 36 */ "Headless",
	/* 37 */ "Wisp",
	/* 38 */ "Daemon",
	/* 39 */ "Dragon",
	/* 40 */ "Sand Trap",
	/* 41 */ "Troll",
	/* 42 */ "",
	/* 43 */ "",
	/* 44 */ "Mongbat",
	/* 45 */ "Corpser",
	/* 46 */ "Rot Worm",
	/* 47 */ "Shadow Lord",
}

// monsterIDData holds each creature's plural/group display name; ref:ENEMY-0001.
//
//nolint:gochecknoglobals
var monsterIDData = [nTotalEnemies]string{
	/* 0  */ "WIZARDS",
	/* 1  */ "BARD",
	/* 2  */ "FIGHTER",
	/* 3  */ "x",
	/* 4  */ "VILLAGER",
	/* 5  */ "MERCHANT",
	/* 6  */ "JESTER",
	/* 7  */ "BARD",
	/* 8  */ "PIRATES",
	/* 9  */ "x",
	/* 10 */ "CHILD",
	/* 11 */ "BEGGAR",
	/* 12 */ "GUARDS",
	/* 13 */ "x",
	/* 14 */ "BLACKTHORN",
	/* 15 */ "LORD BRITISH",
	/* 16 */ "SEA HORSES",
	/* 17 */ "SQUIDS",
	/* 18 */ "SEA SERPENTS",
	/* 19 */ "SHARKS",
	/* 20 */ "GIANT RATS",
	/* 21 */ "BATS",
	/* 22 */ "SPIDERS",
	/* 23 */ "GHOSTS",
	/* 24 */ "SLIME",
	/* 25 */ "GREMLINS",
	/* 26 */ "MIMICS",
	/* 27 */ "REAPERS",
	/* 28 */ "GAZERS",
	/* 29 */ "x",
	/* 30 */ "GARGOYLE",
	/* 31 */ "INSECTS",
	/* 32 */ "ORCS",
	/* 33 */ "SKELETONS",
	/* 34 */ "SNAKES",
	/* 35 */ "ETTINS",
	/* 36 */ "HEADLESSES",
	/* 37 */ "WISPS",
	/* 38 */ "DAEMONS",
	/* 39 */ "DRAGONS",
	/* 40 */ "SAND TRAPS",
	/* 41 */ "TROLLS",
	/* 42 */ "x",
	/* 43 */ "x",
	/* 44 */ "MONGBATS",
	/* 45 */ "CORPSERS",
	/* 46 */ "ROTWORMS",
	/* 47 */ "SHADOW LORD",
}

// friendsData holds each creature's companion candidate index; ref:ENEMY-0001.
//
//nolint:gochecknoglobals
var friendsData = [nTotalEnemies]byte{
	/* 0  Mage       */ 33,
	/* 1  Bard       */ 1,
	/* 2  Fighter    */ 1,
	/* 3  Avatar     */ 3,
	/* 4  Villager   */ 4,
	/* 5  Merchant   */ 4,
	/* 6  Jester     */ 4,
	/* 7  BanjoPlayer*/ 4,
	/* 8  Stocks     */ 4,
	/* 9  Manacles   */ 4,
	/* 10 Child      */ 10,
	/* 11 Beggar     */ 4,
	/* 12 Guard      */ 12,
	/* 13 Wanderer   */ 13,
	/* 14 Blackthorn */ 14,
	/* 15 LordBritish*/ 15,
	/* 16 SeaHorse   */ 17,
	/* 17 Squid      */ 16,
	/* 18 SeaSerpent */ 17,
	/* 19 Shark      */ 19,
	/* 20 Rat        */ 33,
	/* 21 Bat        */ 21,
	/* 22 Spider     */ 20,
	/* 23 Ghost      */ 33,
	/* 24 Slime      */ 24,
	/* 25 Gremlin    */ 26,
	/* 26 Mimic      */ 35,
	/* 27 Reaper     */ 21,
	/* 28 Gazer      */ 21,
	/* 29 Crawler    */ 24,
	/* 30 Gargoyle   */ 30,
	/* 31 Insect     */ 24,
	/* 32 Orc        */ 41,
	/* 33 Skeleton   */ 0,
	/* 34 Snake      */ 22,
	/* 35 Ettin      */ 36,
	/* 36 Headless   */ 35,
	/* 37 Wisp       */ 23,
	/* 38 Devil      */ 39,
	/* 39 Dragon     */ 39,
	/* 40 SandTrap   */ 40,
	/* 41 Troll      */ 20,
	/* 42 ForceField */ 42,
	/* 43 Whirlpool  */ 43,
	/* 44 Mongbat    */ 44,
	/* 45 Corpser    */ 45,
	/* 46 RotWorm    */ 20,
	/* 47 ShadowLord */ 38,
}

// abilitiesFromAdmodes converts a 16-bit ability word into an EnemyAbility map; ref:ENEMY-0001.
//
// Bit → EnemyAbility mapping, including bludgeoning and possession; ref:ENEMY-0001.
//
//	0  0x0080  Bludgeons
//	1  0x0040  PossessCharm
//	2  0x0020  Undead
//	3  0x0010  DivideOnHit
//	4  0x0008  Immortal
//	5  0x0004  PoisonAtRange
//	6  0x0002  StealsFood
//	7  0x0001  NoCorpse
//	8  0x8000  RangedMagic
//	9  0x2000  Teleport
//	10 0x1000  DisappearsOnDeath
//	11 0x0800  Invisibility
//	12 0x0400  GatesInDaemon
//	13 0x0200  Poison
//	14 0x0100  InfectWithPlague
func abilitiesFromAdmodes(flags uint16) map[EnemyAbility]bool {
	masks := [15]uint16{
		0x0080, 0x0040, 0x0020, 0x0010, 0x0008, 0x0004, 0x0002, 0x0001,
		0x8000, 0x2000, 0x1000, 0x0800, 0x0400, 0x0200, 0x0100,
	}

	m := make(map[EnemyAbility]bool, len(masks))
	for i, mask := range masks {
		if flags&mask != 0 {
			m[EnemyAbility(i)] = true
		}
	}

	return m
}

// largeMapMissileFromMontype returns the MissileType and attack range for a monster.
// montype index maps directly to MissileType (0=Arrow … 7=Rock).
// When monrange == 1 (melee only) the monster has no missile (MissileNone).
func largeMapMissileFromMontype(monIdx int) (MissileType, int) {
	rng := int(monRangeData[monIdx])
	if rng <= 1 {
		return MissileNone, rng
	}

	return MissileType(monTypeData[monIdx]), rng
}

// ────────────────────────────────────────────────────────────────────
// Hardcoded behavior flags (sourced from AdditionalEnemyFlags.json)
// ────────────────────────────────────────────────────────────────────

// doNotMoveSet retains inert props and the tactical Reaper/Mimic exclusions in
// tactical pursuit (ref:ENEMY-0002). Crawler is not excluded. This is not an outdoor
// exclusion list: ordinary world movement defaults to pursuit (ref:ENEMY-0005).
//
//nolint:gochecknoglobals
var doNotMoveSet = map[int]bool{
	8: true, 9: true, 26: true, 27: true, 42: true,
}

// isWaterEnemySet lists monster indices that are water-based entities.
//
//nolint:gochecknoglobals
var isWaterEnemySet = map[int]bool{
	8: true, 16: true, 17: true, 18: true, 19: true, 43: true,
}

// isSandEnemySet lists monster indices that live on desert tiles.
//
//nolint:gochecknoglobals
var isSandEnemySet = map[int]bool{
	40: true,
}

// canFlyOverWaterSet describes the flying terrain class (ref:ENEMY-0002). Movement
// legality itself is owned by OriginalCreatureTerrainAllowsShape.
//
//nolint:gochecknoglobals
var canFlyOverWaterSet = map[int]bool{
	13: true, 15: true, 21: true, 28: true, 38: true, 39: true, 44: true,
}

// canPassThroughWallsSet describes the ether terrain class; ref:ENEMY-0002.
//
//nolint:gochecknoglobals
var canPassThroughWallsSet = map[int]bool{
	23: true, 47: true,
}

// hasWhirlpoolMovementSet lists monster indices with whirlpool AI movement.
//
//nolint:gochecknoglobals
var hasWhirlpoolMovementSet = map[int]bool{
	43: true,
}

// activelyAttacksExceptions lists the monster indices that do NOT actively attack.
// All other monsters actively attack by default.
//
//nolint:gochecknoglobals
var activelyAttacksExceptions = map[int]bool{
	9: true, 42: true, 43: true,
}
