package references

// Location identifies a map. Values 1-40 are the original maps; 0xFC-0xFF are virtual locations such as camping.
type Location byte

// GeneralMapType is the kind of map a location opens.
type GeneralMapType int

// General map kinds; LargeMapType covers the overworld and underworld.
const (
	LargeMapType GeneralMapType = iota
	SmallMapType
	CombatMapType
	DungeonMapType
	CampingMapType
)

// Locations; EmptyLocation means no location.
//
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

// GetMapType returns the kind of map this location opens.
func (l Location) GetMapType() GeneralMapType {
	switch {
	case l == BritanniaUnderworld:
		return LargeMapType
	case l >= Deceit && l <= Doom:
		return DungeonMapType
	case l == CombatRestingShrine || l == CampingLocation || l == ShrineLocation || l == CodexLocation ||
		l == CombatLocation || l == BlackthornPrison || l == FinaleLocation:
		return CombatMapType
	}

	return SmallMapType
}
