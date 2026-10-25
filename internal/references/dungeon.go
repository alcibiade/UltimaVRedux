package references

// DungeonCellCount is the complete packed working-map size for one dungeon.
const DungeonCellCount = DungeonFloorCount * int(XDungeonMapTiles) * int(YDungeonMapTiles)

// DungeonCellKind is the high-nibble semantic class of one packed dungeon cell.
type DungeonCellKind byte

// Dungeon cell kinds map directly to original high-nibble values 0x0 through 0xF.
const (
	DungeonPassage DungeonCellKind = iota
	DungeonLadderUp
	DungeonLadderDown
	DungeonLadderBoth
	DungeonClosedChest
	DungeonFountain
	DungeonPitOrTrap
	DungeonOpenChest
	DungeonEnergyField
	DungeonReserved
	DungeonClearedRoom
	DungeonWall
	DungeonSpecialWall
	DungeonHiddenDoor
	DungeonRevealedDoor
	DungeonLiveRoom
)

// DungeonCell preserves the original packed byte. Payload interpretation is
// specific to Kind; callers must not treat the low nibble as generic flags.
type DungeonCell byte

// Raw returns the complete original packed byte.
func (c DungeonCell) Raw() byte {
	return byte(c)
}

// Kind returns the high-nibble semantic class.
func (c DungeonCell) Kind() DungeonCellKind {
	return DungeonCellKind(byte(c) >> 4)
}

// Payload returns the class-specific low nibble.
func (c DungeonCell) Payload() byte {
	return byte(c) & 0x0f
}

// HasBit08 reports the original bit without assigning a class-independent meaning.
func (c DungeonCell) HasBit08() bool {
	return byte(c)&0x08 != 0
}

// DungeonCells is the complete mutable working map for one dungeon, indexed
// as [floor][y][x] to match the file's floor/row/column order; ref:DNG-0001.
type DungeonCells [DungeonFloorCount][YDungeonMapTiles][XDungeonMapTiles]DungeonCell

// IsDungeonCoordinate reports whether floor and X/Y address the 8x8x8 map.
func IsDungeonCoordinate(floor FloorNumber, position Position) bool {
	return floor >= 0 && floor < DungeonFloorCount &&
		position.X >= 0 && position.X < XDungeonMapTiles &&
		position.Y >= 0 && position.Y < YDungeonMapTiles
}

// Cell returns one packed cell when the coordinate is valid.
func (c *DungeonCells) Cell(floor FloorNumber, position Position) (DungeonCell, bool) {
	if c == nil || !IsDungeonCoordinate(floor, position) {
		return 0, false
	}

	return c[int(floor)][int(position.Y)][int(position.X)], true
}

// SetCell replaces one packed cell when the coordinate is valid.
func (c *DungeonCells) SetCell(floor FloorNumber, position Position, cell DungeonCell) bool {
	if c == nil || !IsDungeonCoordinate(floor, position) {
		return false
	}

	c[int(floor)][int(position.Y)][int(position.X)] = cell

	return true
}

// DungeonFacing preserves the original 0..3 clockwise facing values.
type DungeonFacing byte

// Dungeon facing values preserve original clockwise numeric identities.
const (
	DungeonFacingNorth DungeonFacing = iota
	DungeonFacingEast
	DungeonFacingSouth
	DungeonFacingWest
)

// IsValid reports whether facing is one of the original four values.
func (f DungeonFacing) IsValid() bool {
	return f <= DungeonFacingWest
}

// TurnRight returns the next clockwise facing.
func (f DungeonFacing) TurnRight() DungeonFacing {
	if !f.IsValid() {
		panic("invalid dungeon facing")
	}

	return (f + 1) % 4
}

// TurnLeft returns the next counter-clockwise facing.
func (f DungeonFacing) TurnLeft() DungeonFacing {
	if !f.IsValid() {
		panic("invalid dungeon facing")
	}

	return (f + 3) % 4
}

// TurnAround returns the opposite facing.
func (f DungeonFacing) TurnAround() DungeonFacing {
	if !f.IsValid() {
		panic("invalid dungeon facing")
	}

	return (f + 2) % 4
}

// Direction converts facing to the existing absolute input direction.
func (f DungeonFacing) Direction() Direction {
	switch f {
	case DungeonFacingNorth:
		return Up
	case DungeonFacingEast:
		return Right
	case DungeonFacingSouth:
		return Down
	case DungeonFacingWest:
		return Left
	default:
		panic("invalid dungeon facing")
	}
}

// DungeonLastDirection preserves lastdir, including vertical arrival values.
type DungeonLastDirection byte

// Dungeon last-direction values include the two original vertical arrivals.
const (
	DungeonLastNorth DungeonLastDirection = iota
	DungeonLastEast
	DungeonLastSouth
	DungeonLastWest
	DungeonArrivedFromBelow
	DungeonArrivedFromAbove
)

// IsValid reports whether lastdir is an original cardinal or vertical value.
func (d DungeonLastDirection) IsValid() bool {
	return d <= DungeonArrivedFromAbove
}

// WrapDungeonCoordinate wraps one axis onto the 8-tile dungeon floor grid.
func WrapDungeonCoordinate(value int) Coordinate {
	size := int(XDungeonMapTiles)
	value %= size
	if value < 0 {
		value += size
	}

	return Coordinate(value)
}
