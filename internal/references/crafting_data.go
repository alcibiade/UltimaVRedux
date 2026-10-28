package references

import (
	"strconv"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// REDUX DEVIATION: CRAFT-0001: the original has no gathering, crafting or building; Redux adds an
// optional overworld housing system (Gameplay.Crafting.Enabled). Every tunable lives in this file.

// Material is a raw building resource gathered from the overworld.
type Material int

// Materials, in save-data order.
const (
	Wood  Material = 0
	Stone Material = 1
)

// NumberOfMaterials is the number of material identities.
const NumberOfMaterials = 2

// Name returns the player-facing material name.
func (m Material) Name() string {
	switch m {
	case Wood:
		return "Wood"
	case Stone:
		return "Stone"
	default:
		return "Unknown"
	}
}

// MaterialCatalog returns every material identity in save-data order.
func MaterialCatalog() []Material {
	return []Material{Wood, Stone}
}

// Tool is a crafted implement that unlocks full-yield gathering.
type Tool int

// Tools, in save-data order.
const (
	Axe  Tool = 0
	Pick Tool = 1
)

// NumberOfTools is the number of tool identities.
const NumberOfTools = 2

// Name returns the player-facing tool name.
func (t Tool) Name() string {
	switch t {
	case Axe:
		return "Axe"
	case Pick:
		return "Pick"
	default:
		return "Unknown"
	}
}

// ToolCatalog returns every tool identity in save-data order.
func ToolCatalog() []Tool {
	return []Tool{Axe, Pick}
}

// MaterialCost is an amount of each material, indexed by Material.
type MaterialCost [NumberOfMaterials]uint16

// Describe renders a cost such as "3 Wood, 2 Stone".
func (c MaterialCost) Describe() string {
	text := ""
	for _, material := range MaterialCatalog() {
		if c[material] == 0 {
			continue
		}
		if text != "" {
			text += ", "
		}
		text += strconv.Itoa(int(c[material])) + " " + material.Name()
	}
	if text == "" {
		return "nothing"
	}

	return text
}

// Halved returns the demolition refund for a cost, rounding down.
func (c MaterialCost) Halved() MaterialCost {
	var refund MaterialCost
	for material := range c {
		refund[material] = c[material] / 2
	}

	return refund
}

// CraftRecipe turns materials into one tool.
type CraftRecipe struct {
	ID      int
	Name    string
	Cost    MaterialCost
	Output  Tool
	Minutes int
}

// CraftRecipes returns every recipe in menu order.
func CraftRecipes() []CraftRecipe {
	return []CraftRecipe{
		{ID: 0, Name: "Axe", Cost: MaterialCost{Wood: 3, Stone: 2}, Output: Axe, Minutes: 30},
		{ID: 1, Name: "Pick", Cost: MaterialCost{Wood: 4, Stone: 3}, Output: Pick, Minutes: 30},
	}
}

// BuildCategory groups build pieces in the Build menu.
type BuildCategory int

// Build categories, in menu order. Each fits on one Build menu page.
const (
	BuildCategoryFloor BuildCategory = iota
	BuildCategoryWall
	BuildCategoryTable
	BuildCategoryFurnishing
)

// BuildCategories returns every category in menu order.
func BuildCategories() []BuildCategory {
	return []BuildCategory{BuildCategoryFloor, BuildCategoryWall, BuildCategoryTable, BuildCategoryFurnishing}
}

// Name returns the menu label for the category.
func (c BuildCategory) Name() string {
	switch c {
	case BuildCategoryFloor:
		return "Floors"
	case BuildCategoryWall:
		return "Walls and doors"
	case BuildCategoryTable:
		return "Tables and chairs"
	case BuildCategoryFurnishing:
		return "Bedroom and storage"
	default:
		return "Other"
	}
}

// BuildPiece is one placeable overworld tile.
type BuildPiece struct {
	ID       int
	Name     string
	Sprite   indexes.SpriteIndex
	Cost     MaterialCost
	Category BuildCategory
	// HomeChest marks the piece that acts as claim storage.
	HomeChest bool
}

// BuildMinutes is the game time spent placing or demolishing one piece.
const BuildMinutes = 5

// BuildPieces returns every placeable piece in menu order. IDs are stable save identities.
func BuildPieces() []BuildPiece {
	return []BuildPiece{
		{ID: 0, Name: "Wood floor", Sprite: indexes.WoodFloor1, Cost: MaterialCost{Wood: 1}, Category: BuildCategoryFloor},
		{ID: 1, Name: "Stone floor", Sprite: indexes.BrickFloor, Cost: MaterialCost{Stone: 1}, Category: BuildCategoryFloor},
		{ID: 2, Name: "Stone wall", Sprite: indexes.StoneBrickWall, Cost: MaterialCost{Stone: 3}, Category: BuildCategoryWall},
		{ID: 3, Name: "Rough wall", Sprite: indexes.SmallRockWall, Cost: MaterialCost{Stone: 2}, Category: BuildCategoryWall},
		{ID: 4, Name: "Window", Sprite: indexes.StoneGlassWindow, Cost: MaterialCost{Wood: 1, Stone: 2}, Category: BuildCategoryWall},
		{ID: 5, Name: "Fence (east-west)", Sprite: indexes.FenceHoriz, Cost: MaterialCost{Wood: 1}, Category: BuildCategoryWall},
		{ID: 6, Name: "Fence (north-south)", Sprite: indexes.FenceVert, Cost: MaterialCost{Wood: 1}, Category: BuildCategoryWall},
		{ID: 7, Name: "Door", Sprite: indexes.RegularDoor, Cost: MaterialCost{Wood: 3}, Category: BuildCategoryWall},
		{ID: 8, Name: "Table (left end)", Sprite: indexes.TableLeft, Cost: MaterialCost{Wood: 2}, Category: BuildCategoryTable},
		{ID: 9, Name: "Table (middle)", Sprite: indexes.TableMiddle, Cost: MaterialCost{Wood: 2}, Category: BuildCategoryTable},
		{ID: 10, Name: "Table (right end)", Sprite: indexes.TableRight, Cost: MaterialCost{Wood: 2}, Category: BuildCategoryTable},
		{ID: 11, Name: "Chair (forward)", Sprite: indexes.ChairBackForward, Cost: MaterialCost{Wood: 1}, Category: BuildCategoryTable},
		{ID: 12, Name: "Chair (left)", Sprite: indexes.ChairBackLeft, Cost: MaterialCost{Wood: 1}, Category: BuildCategoryTable},
		{ID: 13, Name: "Chair (back)", Sprite: indexes.ChairBackBack, Cost: MaterialCost{Wood: 1}, Category: BuildCategoryTable},
		{ID: 14, Name: "Chair (right)", Sprite: indexes.ChairBackRight, Cost: MaterialCost{Wood: 1}, Category: BuildCategoryTable},
		{ID: 15, Name: "Bed (head)", Sprite: indexes.LeftBed, Cost: MaterialCost{Wood: 3}, Category: BuildCategoryFurnishing},
		{ID: 16, Name: "Bed (foot)", Sprite: indexes.RightBed, Cost: MaterialCost{Wood: 3}, Category: BuildCategoryFurnishing},
		{ID: 17, Name: "Dresser", Sprite: indexes.Dresser, Cost: MaterialCost{Wood: 3}, Category: BuildCategoryFurnishing},
		{ID: 18, Name: "Bookcase", Sprite: indexes.Bookcase, Cost: MaterialCost{Wood: 3}, Category: BuildCategoryFurnishing},
		{ID: 19, Name: "End table", Sprite: indexes.EndTable, Cost: MaterialCost{Wood: 1}, Category: BuildCategoryFurnishing},
		{ID: 20, Name: "Barrel", Sprite: indexes.Barrel, Cost: MaterialCost{Wood: 2}, Category: BuildCategoryFurnishing},
		{ID: 21, Name: "Fireplace", Sprite: indexes.Fireplace, Cost: MaterialCost{Wood: 1, Stone: 4}, Category: BuildCategoryFurnishing},
		{ID: 22, Name: "Storage chest", Sprite: indexes.Box, Cost: MaterialCost{Wood: 4}, Category: BuildCategoryFurnishing,
			HomeChest: true},
	}
}

// BuildPieceByID returns the piece with the given stable ID.
func BuildPieceByID(id int) (BuildPiece, bool) {
	for _, piece := range BuildPieces() {
		if piece.ID == id {
			return piece, true
		}
	}

	return BuildPiece{}, false
}

// IsBuildableGround reports whether a piece may be placed over the tile: open ground or a placed floor.
func IsBuildableGround(sprite indexes.SpriteIndex) bool {
	switch sprite {
	case indexes.Grass, indexes.Forest1, indexes.Desert1, indexes.LeftDesert2, indexes.RightDesert2,
		indexes.PlowedField, indexes.Path1, indexes.Path2, indexes.Path3, indexes.Path4, indexes.Path5,
		indexes.Path6, indexes.Path7, indexes.WoodFloor1, indexes.BrickFloor:
		return true
	default:
		return false
	}
}

// GatherSource describes what a tile yields when gathered.
type GatherSource struct {
	Material Material
	Tool     Tool
	// MinYield and MaxYield bound the tool yield; bare hands yield one unit with BareHandOdds.
	MinYield, MaxYield int
	// BareHands reports whether the tile can be worked without the tool.
	BareHands bool
	// Depletes reports whether a tool harvest changes the tile to After.
	Depletes bool
	After    indexes.SpriteIndex
}

// GatherMinutes is the game time spent on one gather attempt.
const GatherMinutes = 10

// GatherBareHandOdds is the one-in-X chance that bare hands find one unit.
const GatherBareHandOdds = 3

// GatherSourceForTile returns the yield rule for a terrain tile, if it can be gathered.
func GatherSourceForTile(sprite indexes.SpriteIndex) (GatherSource, bool) {
	wood := func(after indexes.SpriteIndex) GatherSource {
		return GatherSource{Material: Wood, Tool: Axe, MinYield: 2, MaxYield: 4, BareHands: true, Depletes: true, After: after}
	}
	switch sprite {
	case indexes.Tree, indexes.DeadTree:
		return wood(indexes.Grass), true
	case indexes.Forest4:
		return wood(indexes.Forest3), true
	case indexes.Forest3:
		return wood(indexes.Forest2), true
	case indexes.Forest2:
		return wood(indexes.Forest1), true
	case indexes.Forest1:
		return wood(indexes.Grass), true
	case indexes.Hills1, indexes.LeftHills2, indexes.RightHills2:
		return GatherSource{Material: Stone, Tool: Pick, MinYield: 1, MaxYield: 2, BareHands: true}, true
	case indexes.SmallMountains, indexes.TallMountains:
		return GatherSource{Material: Stone, Tool: Pick, MinYield: 2, MaxYield: 3}, true
	default:
		return GatherSource{}, false
	}
}

// Land claim rules.
const (
	// LandClaimSize is the width and height of a claim, centred on the party.
	LandClaimSize = 16
	// LandClaimGoldCost is the fee paid to stake a claim.
	LandClaimGoldCost = 300
	// MaxLandClaims bounds how many claims the party may hold.
	MaxLandClaims = 4
	// MaxLandClaimNameLength bounds a claim's name.
	MaxLandClaimNameLength = 24
	// MaxLandClaimWaterTiles bounds how much of a claim may be water.
	MaxLandClaimWaterTiles = LandClaimSize * LandClaimSize / 2
	// LandClaimStakeMinutes is the game time spent staking a claim.
	LandClaimStakeMinutes = 10
)

// IsValidLandClaimName reports whether a claim name is 1 to 24 printable game-font characters
// (' ' through '~') with no surrounding spaces: the in-game input box, the save validator and
// the Atlas label rule all accept exactly these names.
func IsValidLandClaimName(name string) bool {
	if name == "" || len(name) > MaxLandClaimNameLength || strings.TrimSpace(name) != name {
		return false
	}
	for index := range len(name) {
		if name[index] < ' ' || name[index] > '~' {
			return false
		}
	}

	return true
}

// LandClaimStakeCost is the material paid to stake a claim.

//
//nolint:gochecknoglobals // Read-only tunable.
var LandClaimStakeCost = MaterialCost{Wood: 10, Stone: 5}
