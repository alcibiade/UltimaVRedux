package references

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/files"
)

// Location floor counts for different building types
const (
	TownFloorCount    = 2 // Most towns and keeps
	CastleFloorCount  = 5 // Major castles
	DungeonFloorCount = 8 // Dungeons
)

// Small map dimensions: 32 by 32 tiles per floor.
const (
	XSmallMapTiles         = Coordinate(32)
	YSmallMapTiles         = Coordinate(32)
	TotalSmallMapLocations = 32
	smallMapSizeInBytes    = int(XSmallMapTiles * YSmallMapTiles)
)

// SmallMapMasterTypes is the original data file family a small map belongs to.
type SmallMapMasterTypes int

// Small map families, each with its own original data file.
const (
	Castle SmallMapMasterTypes = iota
	Towne
	Dwelling
	Keep
	Dungeon
	CutOrIntroScene
	None
)

// GetSmallMapFile returns the original data file name that holds the family of maps.
func GetSmallMapFile(smallMap SmallMapMasterTypes) string {
	switch smallMap {
	case Castle:
		return files.CastleDat
	case Towne:
		return files.TowneDat
	case Dwelling:
		return files.DwellingDat
	case Keep:
		return files.KeepDat
	case Dungeon:
		return files.DungeonDat
	case CutOrIntroScene, None:
	}

	panic("unhandled default case")
}

// GetMapMasterFromLocation returns the SmallMapMasterType for a given Location
func GetMapMasterFromLocation(location Location) SmallMapMasterTypes {
	switch location {
	case LordBritishsCastle, PalaceOfBlackthorn, EastBritanny, WestBritanny, NorthBritanny, Paws, Cove, BuccaneersDen:
		return Castle
	case Moonglow, Britain, Jhelom, Yew, Minoc, Trinsic, SkaraBrae, NewMagincia:
		return Towne
	case Fogsbane, Stormcrow, Waveguide, Greyhaven, IolosHut, SuteksHut, SinVraalsHut, GrendelsHut:
		return Dwelling
	case Ararat, Bordermarch, Farthing, Windemere, Stonegate, Lycaeum, EmpathAbbey, SerpentsHold:
		return Keep
	case Deceit, Despise, Destard, Wrong, Covetous, Shame, Hythloth, Doom:
		return Dungeon
	case CombatRestingShrine:
		return CutOrIntroScene
	case BritanniaUnderworld:
		return None
	case EmptyLocation, CampingLocation, ShrineLocation, CodexLocation, CombatLocation, BlackthornPrison, FinaleLocation,
		DeedLocation:
		// Virtual and generated locations have no original data file.
	}

	return None
}

// NewSmallMapReferences builds the small map location table from the original files and DATA.OVL names.
func NewSmallMapReferences(gameConfig *config.UltimaVConfiguration, dataOvl *DataOvl) (*LocationReferences, error) {
	smr, err := newSingleMapReferences(gameConfig, dataOvl)
	if err != nil {
		return nil, fmt.Errorf("initializing small map references: %w", err)
	}

	type locationSpec struct {
		location    Location
		hasBasement bool
		floors      int
	}
	groups := [][]locationSpec{
		{
			{LordBritishsCastle, true, CastleFloorCount}, {PalaceOfBlackthorn, true, CastleFloorCount},
			{WestBritanny, false, 1}, {NorthBritanny, false, 1}, {EastBritanny, false, 1},
			{Paws, false, 1}, {Cove, false, 1}, {BuccaneersDen, false, 1},
		},
		{
			{Moonglow, false, TownFloorCount}, {Britain, false, TownFloorCount},
			{Jhelom, false, TownFloorCount}, {Yew, true, TownFloorCount},
			{Minoc, false, TownFloorCount}, {Trinsic, false, TownFloorCount},
			{SkaraBrae, false, TownFloorCount}, {NewMagincia, false, TownFloorCount},
		},
		{
			{Fogsbane, false, 3}, {Stormcrow, false, 3}, {Greyhaven, false, 3}, {Waveguide, false, 3},
			{IolosHut, false, 1}, {SuteksHut, false, 1}, {SinVraalsHut, false, 1}, {GrendelsHut, false, 1},
		},
		{
			{Ararat, false, TownFloorCount}, {Bordermarch, false, TownFloorCount},
			{Farthing, false, 1}, {Windemere, false, 1}, {Stonegate, false, 1},
			{Lycaeum, false, 3}, {EmpathAbbey, false, 3}, {SerpentsHold, true, 3},
		},
		{
			{Deceit, false, 8}, {Despise, false, 8}, {Destard, false, 8}, {Wrong, false, 8},
			{Covetous, false, 8}, {Shame, false, 8}, {Hythloth, false, 8}, {Doom, false, 8},
		},
	}
	for _, group := range groups {
		nextOffset := 0
		for _, spec := range group {
			nextOffset, err = smr.addLocation(spec.location, spec.hasBasement, spec.floors, nextOffset)
			if err != nil {
				return nil, err
			}
		}
	}

	// Not yet added; they need the underworld and combat shrine systems:
	// addLocation(BritanniaUnderworld, true, 2, 0)
	// addLocation(CombatRestingShrine, false, 1, 0)

	return smr, nil
}
