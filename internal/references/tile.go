package references

import (
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const (
	weightImpassable = -1
	weightIdealPath  = 1
	weightPath       = 2
	weightGrass      = 3
	weightDefault    = 10
)

// Tile represents a single tile and its properties.
//
// The Tile struct contains runtime properties (defined in tile_data.go) and methods
// for tile-specific behavior checks. Methods that return boolean values based solely
// on tile type/index (like IsPushable, IsOpenable, IsKlimable) should be placed here
// rather than in external structs like GameState, as these are intrinsic properties
// of the tile itself.
//
//nolint:tagliatelle
type Tile struct {
	Index         indexes.SpriteIndex
	Name          string `json:"Name"`
	SpeedFactor   int    `json:"SpeedFactor"`
	LightEmission int    `json:"LightEmission"`
	BlocksLight   bool   `json:"BlocksLight"`
	IsWindowField bool   `json:"IsWindow"`
}

func (t *Tile) IsPassable(vehicle VehicleType) bool {
	switch vehicle {
	case CarpetVehicle:
		return t.IsCarpetPassable()
	case HorseVehicle:
		return t.IsHorsePassable()
	case SkiffVehicle:
		return t.IsSkiffPassable()
	case FrigateVehicle:
		return t.IsBoatPassable()
	case NoPartyVehicle:
		return t.IsWalkingPassable()
	case NPC:
		return t.IsNPCTerrainPassable()
	}

	return false
}

// Is provides a generic method for checking if a tile matches a specific sprite index.
// Panics if tile is nil - indicates programming error.
func (t *Tile) Is(spriteIndex indexes.SpriteIndex) bool {
	if t == nil {
		panic("Tile.Is() called on nil tile - this indicates a programming error")
	}

	return t.Index == spriteIndex
}

func (t *Tile) IsChair() bool {
	return t.Is(indexes.ChairBackBack) ||
		t.Is(indexes.ChairBackForward) ||
		t.Is(indexes.ChairBackRight) ||
		t.Is(indexes.ChairBackLeft)
}

// IsChairFacingSouth returns true if this tile is a south-facing chair (ChairBackBack).
// Used for BANJO_PLAYER sprite switching: bards only play music when seated on this chair type.
// The south-facing chair is raw tile 0x92 (146); ref:TILE-0007.
func (t *Tile) IsChairFacingSouth() bool {
	return t.Is(indexes.ChairBackBack)
}

func (t *Tile) IsCannon() bool {
	return t.Is(indexes.CannonLeft) ||
		t.Is(indexes.CannonRight) ||
		t.Is(indexes.CannonUp) ||
		t.Is(indexes.CannonDown)
}

// GetCannonDirection returns the firing direction for cannon tiles.
// Returns NoneDirection if tile is not a cannon.
// The low two bits encode direction: 0=N, 1=E, 2=S, 3=W; ref:VEH-0001.
func (t *Tile) GetCannonDirection() Direction {
	switch t.Index {
	case indexes.CannonUp:
		return Up
	case indexes.CannonRight:
		return Right
	case indexes.CannonDown:
		return Down
	case indexes.CannonLeft:
		return Left
	default:
		return NoneDirection
	}
}

func (t *Tile) IsPath() bool {
	return t.Index >= indexes.Path1 && t.Index <= indexes.Path7
}

func (t *Tile) GetStairsFloorDirection() LadderOrStairType {
	switch t.Index {
	case indexes.StairsNorth, indexes.StairsSouth:
		return LadderOrStairUp
	case indexes.StairsEast, indexes.StairsWest:
		return LadderOrStairDown
	default:
		return NotLadderOrStair
	}
}

func (t *Tile) isNPCNoPenaltyWalkable() bool {
	return t.Is(indexes.BrickFloor) || t.Is(indexes.MetalFloor) || t.Is(indexes.WoodFloor1) || t.Is(indexes.LeftWoodFloor2) || t.Is(indexes.RightWoodFloor2)
}

func (t *Tile) GetWalkableWeight() int {
	if t.Index.IsUnlockedDoor() {
		return weightIdealPath
	}

	if !t.IsWalkingPassable() {
		return weightImpassable
	}

	if t.isNPCNoPenaltyWalkable() {
		return weightIdealPath
	}

	if t.IsPath() {
		return weightPath
	}

	if t.Is(indexes.Grass) {
		return weightGrass
	}

	return weightDefault
}

func (t *Tile) IsWalkableDuringWander() bool {
	// Actual NPC steps allow beach and block chairs outside scheduled travel; ref:NPC-0004.
	if t.Index >= indexes.OutsidePath1 && t.Index <= indexes.OutsidePath4 {
		return true
	}

	return t.IsNPCTerrainPassable() && !t.IsChair()
}

func (t *Tile) GetExtraMovementString() string {
	switch t.SpeedFactor {
	case 4:
		return "Slow Progress!"
	case 6:
		return "Very Slow!"
	case 1, 2, -1:
		return ""
	default:
		return "Untrodden Combat Tile"
	}
}

func (t *Tile) IsWall() bool {
	return t.Is(indexes.LargeRockWall) || t.Is(indexes.StoneBrickWall) || t.Is(indexes.StoneBrickWallSecret)
}

func (t *Tile) IsRoad() bool {
	return t.IsPath() // Roads are paths in this context
}

func (t *Tile) IsSwamp() bool {
	return t.Is(indexes.Swamp)
}

func (t *Tile) IsWater() bool {
	return t.Is(indexes.Water1) || t.Is(indexes.Water2) || t.Is(indexes.WaterCoast)
}

func (t *Tile) IsDesert() bool {
	return t.Is(indexes.Desert1) || t.Is(indexes.LeftDesert2) || t.Is(indexes.RightDesert2)
}

func (t *Tile) IsMountain() bool {
	return t.Is(indexes.SmallMountains) || t.Is(indexes.TallMountains)
}

func (t *Tile) IsWaterfall() bool {
	return t.Is(indexes.Waterfall1) || t.Is(indexes.Waterfall2) ||
		t.Is(indexes.Waterfall3) || t.Is(indexes.Waterfall4)
}

func (t *Tile) IsForest() bool {
	if t == nil {
		return false
	}
	// Forest tiles are identified as passable land tiles that are not other terrain types
	// This logic may need refinement based on actual forest tile indexes
	return t.IsLandEnemyPassable() &&
		!t.Is(indexes.Grass) &&
		!t.Is(indexes.Desert1) &&
		!t.Is(indexes.Swamp) &&
		!t.IsPath() &&
		!t.IsMountain()
}

// IsPushable checks if this tile can be pushed/moved by the player.
// This is an intrinsic property of the tile type, not dependent on game state.
func (t *Tile) IsPushable() bool {
	// Chair variants (logical grouping - keep IsChair() for multiple chairs)
	if t.IsChair() {
		return true
	}

	// Cannon variants (logical grouping - keep IsCannon() for multiple cannons)
	if t.IsCannon() {
		return true
	}

	// Single tile checks using generic Is() pattern - based on original game data
	return t.Is(indexes.Barrel) ||
		t.Is(indexes.EndTable) ||
		// Redux calls original U5 END_TABLE (0xAE) CoffeeTable.
		t.Is(indexes.CoffeeTable) ||
		t.Is(indexes.Vanity) ||
		t.Is(indexes.WaterJugTable) ||
		t.Is(indexes.Dresser) ||
		t.Is(indexes.Box) ||
		t.Is(indexes.Plant) ||
		// Additional pushable items (extended beyond original data)
		t.Is(indexes.TableMiddle) ||
		t.Is(indexes.TableFoodTop) ||
		t.Is(indexes.TableFoodBottom) ||
		t.Is(indexes.TableFoodBoth) ||
		t.Is(indexes.Mirror) ||
		t.Is(indexes.Well) ||
		t.Is(indexes.Brazier) ||
		t.Is(indexes.CookStove) ||
		t.Is(indexes.Chest)
}

// IsBoatPassable returns true if boats (frigates) can pass through this tile
func (t *Tile) IsBoatPassable() bool {
	// Traversal is defined by the shared passability table.
	return boatPassableTiles[t.Index]
}

// IsSkiffPassable returns true if skiffs can pass through this tile
func (t *Tile) IsSkiffPassable() bool {
	// Traversal is defined by the shared passability table.
	return skiffPassableTiles[t.Index]
}

// IsSkiffPassableInDirection checks river bridge clearance along the waterway.
// River masks 10 (N/S) and 5 (E/W) apply to tiles 0x6a/0x6b; ref:VEH-0001.
func (t *Tile) IsSkiffPassableInDirection(direction Direction) bool {
	if t.Is(indexes.TrollBridgeHoriz) {
		return direction == Up || direction == Down
	}
	if t.Is(indexes.TrollBridgeVert) {
		return direction == Left || direction == Right
	}

	return t.IsSkiffPassable()
}

// IsCarpetPassable returns true if magic carpets can pass through this tile
func (t *Tile) IsCarpetPassable() bool {
	// Carpets can pass where walking can...
	if walkingPassableTiles[t.Index] {
		return true
	}
	// ...plus additional tiles they can fly over
	return carpetAdditionalPassableTiles[t.Index]
}

// IsHorsePassable returns true if horses can pass through this tile
func (t *Tile) IsHorsePassable() bool {
	// Horse movement uses on-foot terrain with three exclusions; ref:VEH-0001.
	return t.IsWalkingPassable() && !t.Is(indexes.Swamp) && !t.Is(indexes.Lava) && !t.IsChair()
}

// IsKlimable returns true if this tile can be climbed by players
func (t *Tile) IsKlimable() bool {
	// Based on typical Ultima V mechanics: mountains, ladders, and some structures
	return t.IsMountain() ||
		t.Is(indexes.LadderUp) ||
		t.Is(indexes.LadderDown)
}

// IsNPCTerrainPassable checks only the original NPC terrain bitmap, without
// destination, activity, or occupancy overrides. Tile IDs are unsigned bytes.
func (t *Tile) IsNPCTerrainPassable() bool {
	if t.Index < 0 || t.Index > 255 {
		return false
	}

	// NPC terrain bitmap is MSB-first, with set bits marking blocked tiles; ref:NPC-0004.
	npclegmov := [32]byte{
		0x70, 0x0c, 0xff, 0xf8, 0x01, 0xf3, 0xff, 0xff,
		0x72, 0x3f, 0xff, 0xff, 0xff, 0xcf, 0xff, 0xff,
		0xfe, 0xff, 0x0f, 0xff, 0xff, 0xdf, 0xff, 0x5f,
		0xf0, 0x3f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}

	return npclegmov[t.Index>>3]&(0x80>>(t.Index&7)) == 0
}

// IsLandEnemyPassable returns true if land-based enemies can move through this tile
func (t *Tile) IsLandEnemyPassable() bool {
	// Land enemies can move on walkable land tiles
	return t.IsWalkingPassable() && !t.IsWater()
}

// IsWaterEnemyPassable returns true if water-based enemies can move through this tile
func (t *Tile) IsWaterEnemyPassable() bool {
	// Water enemies move in water tiles
	return t.IsWater()
}

// IsOpenable returns true if this tile can be opened (doors, chests, etc.)
func (t *Tile) IsOpenable() bool {
	// Doors and chests can be opened
	return t.Index.IsDoor() || t.Is(indexes.Chest)
}

// IsWalkingPassable returns true if this tile can be walked through by players on foot
func (t *Tile) IsWalkingPassable() bool {
	// Use authoritative tile data from original Ultima V
	return walkingPassableTiles[t.Index]
}

// IsRangeWeaponPassable returns true if ranged weapons (arrows, etc.) can pass through this tile
func (t *Tile) IsRangeWeaponPassable() bool {
	// Range weapons can pass through most open spaces but not walls, mountains, etc.
	return !t.IsWall() && !t.IsMountain() && !t.Index.IsDoor()
}

// GetSpeedFactor returns the speed factor for this tile
func (t *Tile) GetSpeedFactor() int {
	return t.SpeedFactor
}

// IsFence returns true if this tile is a fence
func (t *Tile) IsFence() bool {
	return t.Is(indexes.FenceHoriz) || t.Is(indexes.FenceVert)
}

// GetLightEmission returns the light emission value for this tile
func (t *Tile) GetLightEmission() int {
	return t.LightEmission
}

// GetNPCWalkableWeight returns the walkable weight for NPCs
func (t *Tile) GetNPCWalkableWeight() int {
	if !t.IsNPCTerrainPassable() {
		return weightImpassable
	}

	// REDUX DEVIATION: Original NPC routing uses pass/fail terrain checks (ref:NPC-0002).
	// Keep Redux weighted path preferences without inheriting player exclusions.
	weight := t.GetWalkableWeight()
	if weight < 0 {
		return weightDefault
	}

	return weight
}

// IsTable returns true if this tile is a table
func (t *Tile) IsTable() bool {
	return t.Is(indexes.TableLeft) || t.Is(indexes.TableMiddle) || t.Is(indexes.TableRight) ||
		t.Is(indexes.TableFoodTop) || t.Is(indexes.TableFoodBottom) || t.Is(indexes.TableFoodBoth) ||
		t.Is(indexes.CandleOnTable)
}

// IsTalkOverable identifies terrain that Talk can skip before inspecting exactly
// one additional position; ref:TALK-0005.
func (t *Tile) IsTalkOverable() bool {
	//nolint:exhaustive // Talk-through terrain has a narrow allowlist; every other sprite is false; ref:TALK-0005.
	switch t.Index {
	case indexes.CrystalBall,
		indexes.TableLeft,
		indexes.TableMiddle,
		indexes.TableRight,
		indexes.MagicLockDoor,
		indexes.MagicLockDoorWithView,
		indexes.Portcullis,
		indexes.TableFoodTop,
		indexes.TableFoodBottom,
		indexes.TableFoodBoth,
		indexes.EndTable,    // Original DESK (0xa5).
		indexes.CoffeeTable, // Original END_TABLE (0xae).
		indexes.LockedDoorView,
		indexes.RegularDoorView,
		indexes.CandleOnTable, // Original CANDELABRUM (0xbe).
		indexes.FenceHoriz,
		indexes.FenceVert:
		return true
	default:
		return false
	}
}

// IsWindow returns true if this tile is a window
func (t *Tile) IsWindow() bool {
	return t.IsWindowField
}

// IsBridge returns true if this tile is any type of bridge.
func (t *Tile) IsBridge() bool {
	return t.Is(indexes.Bridge) ||
		t.Is(indexes.TrollBridgeHoriz) ||
		t.Is(indexes.TrollBridgeVert)
}

// IsTrollBridge returns true for the two overworld bridge tiles checked by
// the original bridge-toll gate; ref:ENV-0004.
func (t *Tile) IsTrollBridge() bool {
	return t.Is(indexes.TrollBridgeHoriz) || t.Is(indexes.TrollBridgeVert)
}

// IsStocksOrManacles returns true if this tile is stocks or manacles.
// Used for prisoner freeing via Jimmy command.
// Jimmy can release stocks and manacles; ref:CRIME-0005.
func (t *Tile) IsStocksOrManacles() bool {
	return t.Is(indexes.Stocks) || t.Is(indexes.Manacles)
}

// IsValidForMoonstoneBurial returns true if this tile is valid terrain for burying a moonstone.
// Original moonstone burial terrain restrictions (ref:MOON-0004):
// - PLOWED_FIELD, CROPS are explicitly allowed
// - terrain > SHOALS (3) AND terrain < HILLS (11): Swamp, Grass, Forest types
// This allows burial on soft ground types but not on water, mountains, buildings, etc.
func (t *Tile) IsValidForMoonstoneBurial() bool {
	// Explicit terrain types from original (PLOWED_FIELD, CROPS)
	if t.Is(indexes.PlowedField) || t.Is(indexes.WheatInField) {
		return true
	}

	// Range check: terrain > SHOALS(3) AND terrain < HILLS(11)
	// This includes: Swamp(4), Grass(5), Forest1(6), Forest2(8), Forest3(9), Forest4(10)
	if t.Is(indexes.Swamp) || t.Is(indexes.Grass) {
		return true
	}

	// Forest tiles (multiple variants)
	if t.Is(indexes.Forest1) || t.Is(indexes.Forest2) || t.Is(indexes.Forest3) || t.Is(indexes.Forest4) {
		return true
	}

	return false
}
