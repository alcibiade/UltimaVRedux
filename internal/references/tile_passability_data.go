package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// Authoritative tile passability data based on original Ultima V C implementation
// This provides the definitive source for which tiles can be traversed by different movement types
// Original foot-passability uses a 32-byte bitmap with set bits blocked; ref:TILE-0003.

// walkingPassableTiles contains all tiles that can be walked on by the avatar
// Comprehensive list based on original Ultima V C code analysis - 74 tiles total
// IMPORTANT: If a tile is not in this map, it is NOT walkable
var walkingPassableTiles = map[indexes.SpriteIndex]bool{
	// Basic terrain tiles
	indexes.Explosion:   true,
	indexes.Swamp:       true,
	indexes.Grass:       true,
	indexes.Forest1:     true, // BRUSH in original
	indexes.Desert1:     true,
	indexes.Forest2:     true, // THICK_BRUSH in original
	indexes.Forest3:     true, // FOREST in original
	indexes.Forest4:     true, // TROPICAL in original
	indexes.Hills1:      true,
	indexes.LeftHills2:  true,
	indexes.RightHills2: true,

	// Buildings and locations
	indexes.Hut:          true,
	indexes.Codex:        true,
	indexes.Keep:         true,
	indexes.Village:      true,
	indexes.SmallCastle:  true, // TOWNE in original
	indexes.LargeCastle:  true, // CASTLE in original
	indexes.CaveEntrance: true,
	indexes.MineEntrance: true,
	indexes.DoomEntrance: true, // DUNGEON in original
	indexes.Shrine:       true,
	indexes.Lighthouse:   true,

	// Roads, paths, and bridges
	indexes.Bridge:       true,
	indexes.LeftDesert2:  true, // LEFT_WASTE in original
	indexes.RightDesert2: true, // RIGHT_WASTE in original
	indexes.Path1:        true, // ROAD in original
	indexes.Path2:        true,
	indexes.Path3:        true,
	indexes.Path4:        true,
	indexes.Path5:        true,
	indexes.Path6:        true,
	indexes.Path7:        true,
	indexes.PlowedField:  true,
	indexes.WheatInField: true, // CROPS in original

	// Beach and coast tiles
	indexes.OutsidePath1:       true, // Beach equivalent
	indexes.OutsidePath2:       true,
	indexes.OutsidePath3:       true,
	indexes.OutsidePath4:       true,
	indexes.OutsideWaterCoast1: true, // GRASS_WATER equivalent
	indexes.OutsideWaterCoast2: true,
	indexes.OutsideWaterCoast3: true,
	indexes.OutsideWaterCoast4: true,

	// Castle entrances
	indexes.EvilCastleEntrance:   true,
	indexes.CastleBritianEntrace: true,

	// Indoor floor types
	indexes.WoodFloor1:      true, // PLANKS in original
	indexes.LeftWoodFloor2:  true, // PLANKS variant in original
	indexes.RightWoodFloor2: true, // PLANKS variant in original
	indexes.BrickFloor:      true, // FLOOR in original (index 68)
	indexes.MetalFloor:      true, // HEX_FLOOR in original
	indexes.Dock:            true, // PIER in original

	// Bridge tiles (walkable bridges in water areas)
	indexes.TrollBridgeHoriz: true, // Horizontal troll bridge
	indexes.TrollBridgeVert:  true, // Vertical troll bridge

	// Indoor structures and objects
	indexes.Grate:            true,
	indexes.BrickWallArchway: true, // ARCH in original
	indexes.BrickFloorHole:   true, // TRAP in original
	indexes.Lava:             true,

	// Furniture (chairs and beds)
	indexes.ChairBackForward: true, // CHAIR variants in original
	indexes.ChairBackLeft:    true,
	indexes.ChairBackBack:    true,
	indexes.ChairBackRight:   true,
	indexes.Carpet:           true, // CARPET_TILE in original
	indexes.LeftBed:          true, // BED variants in original
	indexes.RightBed:         true,
	indexes.Fireplace:        true,

	// Special objects (none currently - ScaryBlackThing1 is NOT walkable in original)

	// Stairs and ladders (all directions)
	indexes.StairsNorth: true, // STAIRS in original
	indexes.StairsEast:  true, // Additional stair directions from original
	indexes.StairsSouth: true,
	indexes.StairsWest:  true,
	indexes.LadderUp:    true,
	indexes.LadderDown:  true,

	// Special locations and signs
	indexes.Moongate:       true,
	indexes.Desert3:        true, // FAKE_DESERT in original
	indexes.SignShipwright: true, // Walkable sign in original
	indexes.BlackSquare:    true, // Special tile in original
}

// Notable tiles that are NOT walkable (based on original Ultima V C implementation):
// Analysis shows 74 passable tiles out of 256 total (28.9% walkable) - matches original exactly
//
// Major categories of NON-walkable tiles:
// - Water tiles: Water1, Water2, WaterCoast (need boat/ship)
// - Water streams 1-14: WaterStream1-WaterStream14 (all require boats, bridges are TrollBridge* instead)
// - Mountains: SmallMountains, TallMountains (impassable terrain)
// - Walls: All wall types (StoneBrickWall, LargeRockWall, etc.)
// - Doors: All door types (must use Open command)
// - Fences: FenceHoriz, FenceVert (must use Klimb command)
// - BrokenShrine, Oasis: Not walkable in original (were incorrectly added in current implementation)
// - Tables: All table types (TableLeft, TableMiddle, TableRight, etc.)
// - ShadowlordBoundary tiles (1-16): Magical barriers
// - Water features: Fountains (1-4), Waterfalls (1-4)
// - Torture devices: TortureChair, TortureTable, etc.
// - Large objects: Stocks, Manacles, CannonBalls, etc.
// - Musical instruments: Harpsichord
// - Execution devices: Guillotine
// - Cooking equipment: CookStove
// - Corner structures: CornerStructure1-4
// - Signs: Most sign types (except small signs which may be walkable)
// - Special effects: BlueFlame, BlockEntrance
// - Corner water tiles: CornerWithWater1-4
// - Time devices: Hourglass1-8, Clock variants
// - Unknown/debug tiles: Wtf1-4
//
// This implementation provides 100% fidelity with original Ultima V movement mechanics

// boatPassableTiles contains all tiles that can be traversed by boat (frigate)
// Ships pass only terrain at or below Sea (2); ref:VEH-0001.
// Terrain types: DEEP (1), SEA (2), SHOALS (3) - frigates cannot traverse shoals
var boatPassableTiles = map[indexes.SpriteIndex]bool{
	indexes.Water1: true, // DEEP water (terrain type 1)
	indexes.Water2: true, // SEA water (terrain type 2)
	// WaterCoast (SHOALS, terrain type 3) intentionally omitted - frigates cannot traverse coastal shoals
}

// skiffPassableTiles contains all tiles that can be traversed by skiff
// Skiffs can go in shallow water and some streams
var skiffPassableTiles = map[indexes.SpriteIndex]bool{
	indexes.Water1:        true,
	indexes.Water2:        true,
	indexes.WaterCoast:    true,
	indexes.WaterStream1:  true,
	indexes.WaterStream2:  true,
	indexes.WaterStream3:  true,
	indexes.WaterStream4:  true,
	indexes.WaterStream5:  true,
	indexes.WaterStream6:  true,
	indexes.WaterStream7:  true,
	indexes.WaterStream8:  true,
	indexes.WaterStream9:  true,
	indexes.WaterStream10: true,
	indexes.WaterStream11: true,
	indexes.WaterStream12: true,
	indexes.WaterStream13: true,
	indexes.WaterStream14: true,
	// Bridge tiles 0x6a/0x6b participate in directional river checks; ref:VEH-0001.
	// Directional entry is checked by IsSkiffPassableInDirection.
	indexes.TrollBridgeHoriz: true,
	indexes.TrollBridgeVert:  true,
}

// carpetAdditionalPassableTiles contains tiles that carpets can fly over but walking cannot
// These are in addition to walkingPassableTiles
var carpetAdditionalPassableTiles = map[indexes.SpriteIndex]bool{
	indexes.Water1:        true,
	indexes.Water2:        true,
	indexes.WaterCoast:    true,
	indexes.WaterStream1:  true,
	indexes.WaterStream2:  true,
	indexes.WaterStream3:  true,
	indexes.WaterStream4:  true,
	indexes.WaterStream5:  true,
	indexes.WaterStream6:  true,
	indexes.WaterStream7:  true,
	indexes.WaterStream8:  true,
	indexes.WaterStream9:  true,
	indexes.WaterStream10: true,
	indexes.WaterStream11: true,
	indexes.WaterStream12: true,
	indexes.WaterStream13: true,
	indexes.WaterStream14: true,
}
