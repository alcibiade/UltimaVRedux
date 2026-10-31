package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// REDUX DEVIATION: CRAFT-0009: the original has no player-made furniture or buildings. With
// Gameplay.Crafting.Enabled, original map tiles become crafted pieces the party carries and
// places on its own deed maps; see ref:CRAFT-0009.

// Build pieces (ref:CRAFT-0009). Each piece is an original map tile the party crafts into its
// stock and then places on its own deed map with Build.

// BuildPieceID identifies a build piece. It is a save identity (stock, placements and chest
// contents): append new pieces, never reorder or reuse an ID.
type BuildPieceID int

// Build pieces, in save-data order.
const (
	PieceStoneWall BuildPieceID = iota
	PieceWindow
	PieceDoor
	PieceFenceEastWest
	PieceFenceNorthSouth
	PieceWoodFloor
	PieceStoneFloor
	PieceTableWestEnd
	PieceTableMiddle
	PieceTableEastEnd
	PieceChairForward
	PieceChairLeft
	PieceChairBack
	PieceChairRight
	PieceBedHead
	PieceBedFoot
	PieceDresser
	PieceBookcase
	PieceEndTable
	PieceBarrel
	PieceStorageChest
	PieceWorkbench
	PieceSawmill
	PieceForge
	PieceAlchemyTable
	PieceFireplace
)

// NumberOfBuildPieces is the number of piece identities.
const NumberOfBuildPieces = 26

// Build and Demolish timing (ref:CRAFT-0010), the ordinary turn included.
const (
	BuildPlaceMinutes    = 5
	BuildDemolishMinutes = 5
)

// BuildPiece is one placeable deed-map tile.
type BuildPiece struct {
	ID       BuildPieceID
	Name     string
	Sprite   indexes.SpriteIndex
	Category CraftCategory
	// Station is the crafting station the piece serves as once placed, or CraftStationNone.
	Station CraftStation
	// StorageChest marks the piece whose contents the party keeps (ref:CRAFT-0011).
	StorageChest bool
	// BedHead marks the bed half the party sleeps in (ref:CRAFT-0012).
	BedHead bool
	// LookName replaces the original tile's Look text, whose wording would mislead
	// (a workbench is the "fire logs" tile); empty keeps the original text.
	LookName string
}

// IsFloor reports whether the piece is a floor other pieces may be placed on.
func (p BuildPiece) IsFloor() bool {
	return p.Category == CraftCategoryFloors
}

// Piece returns the definition of a piece ID. It panics on an unknown ID, which only a
// programming error or an unvalidated save can produce.
func (id BuildPieceID) Piece() BuildPiece {
	piece, ok := BuildPieceByID(id)
	if !ok {
		panic("unknown build piece")
	}

	return piece
}

// BuildPieceCatalog returns every piece in save-data order, which is also menu order within
// each category.
//
//nolint:funlen // One row per piece keeps the catalogue reviewable in one place.
func BuildPieceCatalog() []BuildPiece {
	return []BuildPiece{
		{ID: PieceStoneWall, Name: "Stone wall", Sprite: indexes.StoneBrickWall, Category: CraftCategoryWalls},
		{ID: PieceWindow, Name: "Window", Sprite: indexes.StoneGlassWindow, Category: CraftCategoryWalls},
		{ID: PieceDoor, Name: "Door", Sprite: indexes.RegularDoor, Category: CraftCategoryWalls},
		{ID: PieceFenceEastWest, Name: "Fence (east-west)", Sprite: indexes.FenceHoriz, Category: CraftCategoryWalls},
		{ID: PieceFenceNorthSouth, Name: "Fence (north-south)", Sprite: indexes.FenceVert, Category: CraftCategoryWalls},
		{ID: PieceWoodFloor, Name: "Wood floor", Sprite: indexes.WoodFloor1, Category: CraftCategoryFloors},
		{ID: PieceStoneFloor, Name: "Stone floor", Sprite: indexes.BrickFloor, Category: CraftCategoryFloors},
		{ID: PieceTableWestEnd, Name: "Table (west end)", Sprite: indexes.TableLeft, Category: CraftCategoryTables},
		{ID: PieceTableMiddle, Name: "Table (middle)", Sprite: indexes.TableMiddle, Category: CraftCategoryTables},
		{ID: PieceTableEastEnd, Name: "Table (east end)", Sprite: indexes.TableRight, Category: CraftCategoryTables},
		{ID: PieceChairForward, Name: "Chair (forward)", Sprite: indexes.ChairBackForward, Category: CraftCategoryTables},
		{ID: PieceChairLeft, Name: "Chair (left)", Sprite: indexes.ChairBackLeft, Category: CraftCategoryTables},
		{ID: PieceChairBack, Name: "Chair (back)", Sprite: indexes.ChairBackBack, Category: CraftCategoryTables},
		{ID: PieceChairRight, Name: "Chair (right)", Sprite: indexes.ChairBackRight, Category: CraftCategoryTables},
		{ID: PieceBedHead, Name: "Bed (head)", Sprite: indexes.LeftBed, Category: CraftCategoryFurnishings,
			BedHead: true, LookName: "thy own bed"},
		{ID: PieceBedFoot, Name: "Bed (foot)", Sprite: indexes.RightBed, Category: CraftCategoryFurnishings,
			LookName: "the foot of thy bed"},
		{ID: PieceDresser, Name: "Dresser", Sprite: indexes.Dresser, Category: CraftCategoryFurnishings},
		{ID: PieceBookcase, Name: "Bookcase", Sprite: indexes.Bookcase, Category: CraftCategoryFurnishings},
		{ID: PieceEndTable, Name: "End table", Sprite: indexes.EndTable, Category: CraftCategoryFurnishings},
		{ID: PieceBarrel, Name: "Barrel", Sprite: indexes.Barrel, Category: CraftCategoryFurnishings},
		{ID: PieceStorageChest, Name: "Storage chest", Sprite: indexes.Box, Category: CraftCategoryFurnishings,
			StorageChest: true, LookName: "thy storage chest"},
		{ID: PieceWorkbench, Name: "Workbench", Sprite: indexes.FireLogs, Category: CraftCategoryStations,
			Station: CraftStationWorkbench, LookName: "a sturdy workbench with a vise"},
		{ID: PieceSawmill, Name: "Sawmill", Sprite: indexes.Stocks, Category: CraftCategoryStations,
			Station: CraftStationSawmill, LookName: "a saw frame for cutting planks"},
		{ID: PieceForge, Name: "Forge", Sprite: indexes.Anvil, Category: CraftCategoryStations,
			Station: CraftStationForge, LookName: "a forge and anvil"},
		{ID: PieceAlchemyTable, Name: "Alchemy table", Sprite: indexes.WaterJugTable, Category: CraftCategoryStations,
			Station: CraftStationAlchemyTable, LookName: "an alchemist's table with flask and retort"},
		{ID: PieceFireplace, Name: "Fireplace", Sprite: indexes.Fireplace, Category: CraftCategoryStations,
			Station: CraftStationFireplace, LookName: "a hearth fit for cooking"},
	}
}

// BuildPieceByID returns the piece with the given stable ID.
func BuildPieceByID(id BuildPieceID) (BuildPiece, bool) {
	if id < 0 || id >= NumberOfBuildPieces {
		return BuildPiece{}, false
	}

	return BuildPieceCatalog()[id], true
}

// BuildPieceForStation returns the piece that serves as a station.
func BuildPieceForStation(station CraftStation) (BuildPiece, bool) {
	for _, piece := range BuildPieceCatalog() {
		if piece.Station == station && station != CraftStationNone {
			return piece, true
		}
	}

	return BuildPiece{}, false
}
