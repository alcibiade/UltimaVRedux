package references

type Location byte

type GeneralMapType int

const (
	LargeMapType GeneralMapType = iota
	SmallMapType
	CombatMapType
	DungeonMapType
	CampingMapType
)

//goland:noinspection GoSnakeCaseUsage
const (
	EmptyLocation       Location = 0xFF
	CampingLocation     Location = 0xFE // Virtual location for camping mode
	ShrineLocation      Location = 0xFD // Virtual location for shrine meditation
	CodexLocation       Location = 0xFC // Virtual location for Codex reading
	CombatLocation      Location = 0xFB // Virtual location for combat mode
	BlackthornPrison    Location = 0xFA // Virtual location for Blackthorn's interrogation
	FinaleLocation      Location = 0xF9 // Virtual location for Lord British's terminal rescue
	BritanniaUnderworld Location = 0x00
	Moonglow            Location = 1
	Britain             Location = 2
	Jhelom              Location = 3
	Yew                 Location = 4
	Minoc               Location = 5
	Trinsic             Location = 6
	SkaraBrae           Location = 7
	NewMagincia         Location = 8 // Town
	Fogsbane            Location = 9
	Stormcrow           Location = 10
	Greyhaven           Location = 11
	Waveguide           Location = 12
	IolosHut            Location = 13
	SuteksHut           Location = 14
	SinVraalsHut        Location = 15
	GrendelsHut         Location = 16 // Dwelling
	LordBritishsCastle  Location = 17
	PalaceOfBlackthorn  Location = 18
	WestBritanny        Location = 19
	NorthBritanny       Location = 20
	EastBritanny        Location = 21
	Paws                Location = 22
	Cove                Location = 23
	BuccaneersDen       Location = 24 // Castle
	Ararat              Location = 25
	Bordermarch         Location = 26
	Farthing            Location = 27
	Windemere           Location = 28
	Stonegate           Location = 29
	Lycaeum             Location = 30 // Keep
	EmpathAbbey         Location = 31
	SerpentsHold        Location = 32
	Deceit              Location = 33 // Dungeons
	Despise             Location = 34
	Destard             Location = 35
	Wrong               Location = 36
	Covetous            Location = 37
	Shame               Location = 38
	Hythloth            Location = 39
	Doom                Location = 40
	CombatRestingShrine Location = 41
)

// Guard encounter map IDs; ref:CRIME-0001.
const (
	JailLocationID             = Britain            // Map for jail (Britain)
	CharityGuardLocationID     = Minoc              // Map 5: Charity guards (50% gold demand)
	BlackthornCastleLocationID = PalaceOfBlackthorn // Map 0x12: Password guards
)

// Map coordinate system constants for large map wraparound calculations
// The large map is 256x256 tiles with wraparound at edges
const (
	// MapCoordinateWrapMask is used to wrap coordinates within 0-255 range (bitwise AND)
	MapCoordinateWrapMask = 0xff
	// MapWrapThreshold is the distance threshold for detecting map edge wraparound
	MapWrapThreshold = 128
	// MapCoordinateSpace is the full size of the large map coordinate space (256x256)
	MapCoordinateSpace = 256
	// DynamicSpawnIndex indicates an entity was spawned dynamically (not from save data)
	DynamicSpawnIndex = -1
)

func (l Location) GetMapType() GeneralMapType {
	switch l {
	case BritanniaUnderworld:
		return LargeMapType
	case Deceit, Despise, Destard, Wrong, Covetous, Shame, Hythloth, Doom:
		return DungeonMapType
	case CombatRestingShrine, CampingLocation, ShrineLocation, CodexLocation, CombatLocation, BlackthornPrison, FinaleLocation:
		return CombatMapType
	}

	return SmallMapType
}

func GetListOfAllSmallMaps() []string {
	names := make([]string, 0)

	for loc := Moonglow; loc <= SerpentsHold; loc++ {
		names = append(names, loc.String())
	}

	return names
}
