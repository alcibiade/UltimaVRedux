package references

import (
	"strconv"
	"strings"
)

// REDUX DEVIATION: CRAFT-0008: the original has no stations, refined materials or cooking.
// With Gameplay.Crafting.Enabled, recipes are made at stations built on deed maps, and the
// Fireplace cooks Forage into the party's original food; see ref:CRAFT-0008.

// Crafting recipes (ref:CRAFT-0007, ref:CRAFT-0008).

// CraftStation is the deed-map piece a recipe needs within one square of the party
// (ref:CRAFT-0008). A recipe with CraftStationNone works wherever Craft applies. The value is
// not a save identity: a station is a build piece, and pieces are saved by BuildPieceID.
type CraftStation int

// Crafting stations.
const (
	CraftStationNone CraftStation = iota
	CraftStationWorkbench
	CraftStationSawmill
	CraftStationForge
	CraftStationAlchemyTable
	CraftStationFireplace
)

// CraftStations returns every station in menu order, without CraftStationNone.
func CraftStations() []CraftStation {
	return []CraftStation{
		CraftStationWorkbench, CraftStationSawmill, CraftStationForge, CraftStationAlchemyTable, CraftStationFireplace,
	}
}

// Name returns the player-facing station name.
func (s CraftStation) Name() string {
	switch s {
	case CraftStationNone:
		return "none"
	case CraftStationWorkbench:
		return "Workbench"
	case CraftStationSawmill:
		return "Sawmill"
	case CraftStationForge:
		return "Forge"
	case CraftStationAlchemyTable:
		return "Alchemy table"
	case CraftStationFireplace:
		return "Fireplace"
	default:
		return "Unknown"
	}
}

// CraftCategory groups recipes in the Craft menu and build pieces in the Build menu. Later
// steps append categories (weapons, armour, potions, scrolls). The value is not a save identity.
type CraftCategory int

// Craft categories, in menu order.
const (
	CraftCategoryTools CraftCategory = iota
	CraftCategoryMaterials
	CraftCategoryStations
	CraftCategoryWalls
	CraftCategoryFloors
	CraftCategoryTables
	CraftCategoryFurnishings
	CraftCategoryFood
)

// CraftCategories returns every category in menu order.
func CraftCategories() []CraftCategory {
	return []CraftCategory{
		CraftCategoryTools, CraftCategoryMaterials, CraftCategoryStations, CraftCategoryWalls,
		CraftCategoryFloors, CraftCategoryTables, CraftCategoryFurnishings, CraftCategoryFood,
	}
}

// Name returns the menu label of the category.
func (c CraftCategory) Name() string {
	switch c {
	case CraftCategoryTools:
		return "Tools"
	case CraftCategoryMaterials:
		return "Refined materials"
	case CraftCategoryStations:
		return "Stations"
	case CraftCategoryWalls:
		return "Walls and doors"
	case CraftCategoryFloors:
		return "Floors"
	case CraftCategoryTables:
		return "Tables and chairs"
	case CraftCategoryFurnishings:
		return "Furnishings"
	case CraftCategoryFood:
		return "Food"
	default:
		return "Other"
	}
}

// IsBuildCategory reports whether the category holds build pieces.
func (c CraftCategory) IsBuildCategory() bool {
	switch c {
	case CraftCategoryStations, CraftCategoryWalls, CraftCategoryFloors, CraftCategoryTables, CraftCategoryFurnishings:
		return true
	case CraftCategoryTools, CraftCategoryMaterials, CraftCategoryFood:
		return false
	default:
		return false
	}
}

// CraftGoodsKind says which inventory a recipe input or output belongs to. Step 4 adds the
// original weapons, armour and shields (references.Equipment) and step 5 potions and scrolls.
type CraftGoodsKind int

// Kinds of crafting goods.
const (
	CraftGoodsMaterial CraftGoodsKind = iota
	CraftGoodsTool
	CraftGoodsReagent
	// CraftGoodsPiece is a build piece, carried until Build places it (ref:CRAFT-0009).
	CraftGoodsPiece
	// CraftGoodsProvision is one of the party's original provisions: food, torches, gems or
	// keys (references.Provision). Gold is never a craft good.
	CraftGoodsProvision
)

// CraftGoods is a quantity of one material, tool, reagent, piece or provision. ID is the
// Material, Tool, Reagent, BuildPieceID or Provision value, according to Kind.
type CraftGoods struct {
	Kind     CraftGoodsKind
	ID       int
	Quantity uint16
}

// MaterialGoods is a quantity of a material.
func MaterialGoods(material Material, quantity uint16) CraftGoods {
	return CraftGoods{Kind: CraftGoodsMaterial, ID: int(material), Quantity: quantity}
}

// ToolGoods is a quantity of a tool.
func ToolGoods(tool Tool, quantity uint16) CraftGoods {
	return CraftGoods{Kind: CraftGoodsTool, ID: int(tool), Quantity: quantity}
}

// ReagentGoods is a quantity of a reagent.
func ReagentGoods(reagent Reagent, quantity uint16) CraftGoods {
	return CraftGoods{Kind: CraftGoodsReagent, ID: int(reagent), Quantity: quantity}
}

// PieceGoods is a quantity of a build piece.
func PieceGoods(piece BuildPieceID, quantity uint16) CraftGoods {
	return CraftGoods{Kind: CraftGoodsPiece, ID: int(piece), Quantity: quantity}
}

// ProvisionGoods is a quantity of an original provision.
func ProvisionGoods(provision Provision, quantity uint16) CraftGoods {
	return CraftGoods{Kind: CraftGoodsProvision, ID: int(provision), Quantity: quantity}
}

// ProvisionName returns the player-facing name of a provision.
func ProvisionName(provision Provision) string {
	switch provision {
	case Torches:
		return "Torches"
	case Gem:
		return "Gems"
	case Key:
		return "Keys"
	case SkullKeys:
		return "Skull Keys"
	case Food:
		return "Food"
	case Gold:
		return "Gold"
	case NoProvision:
		return "Unknown"
	default:
		return "Unknown"
	}
}

// Name returns the goods' player-facing name, without the quantity.
func (c CraftGoods) Name() string {
	switch c.Kind {
	case CraftGoodsMaterial:
		return Material(c.ID).Name()
	case CraftGoodsTool:
		return Tool(c.ID).Name()
	case CraftGoodsReagent:
		return Reagent(c.ID).Name()
	case CraftGoodsPiece:
		return BuildPieceID(c.ID).Piece().Name
	case CraftGoodsProvision:
		return ProvisionName(Provision(c.ID))
	default:
		return "Unknown"
	}
}

// Describe renders goods such as "2 Wood".
func (c CraftGoods) Describe() string {
	return strconv.Itoa(int(c.Quantity)) + " " + c.Name()
}

// DescribeCraftGoods renders a list such as "2 Wood, 1 Stone".
func DescribeCraftGoods(goods []CraftGoods) string {
	parts := make([]string, 0, len(goods))
	for _, item := range goods {
		parts = append(parts, item.Describe())
	}
	if len(parts) == 0 {
		return "nothing"
	}

	return strings.Join(parts, ", ")
}

// CraftRecipeID identifies a recipe. Recipes are not saved; the value only needs to be stable
// within one build.
type CraftRecipeID int

// Recipes.
const (
	RecipeAxe CraftRecipeID = iota
	RecipePick
	RecipePlanks
	RecipeCutStone
	RecipeIngots
	RecipeWorkbench
	RecipeSawmill
	RecipeForge
	RecipeAlchemyTable
	RecipeFireplace
	RecipeStoneWall
	RecipeWindow
	RecipeDoor
	RecipeFenceEastWest
	RecipeFenceNorthSouth
	RecipeWoodFloor
	RecipeStoneFloor
	RecipeTableWestEnd
	RecipeTableMiddle
	RecipeTableEastEnd
	RecipeChairForward
	RecipeChairLeft
	RecipeChairBack
	RecipeChairRight
	RecipeBed
	RecipeDresser
	RecipeBookcase
	RecipeEndTable
	RecipeBarrel
	RecipeStorageChest
	RecipeCookFood
)

// CraftRecipe turns Inputs into Outputs in Minutes (the ordinary turn included), at Station.
type CraftRecipe struct {
	ID       CraftRecipeID
	Name     string
	Category CraftCategory
	Inputs   []CraftGoods
	Outputs  []CraftGoods
	Minutes  int
	Station  CraftStation
}

// pieceRecipe is a recipe at the Workbench (unless stated) that makes one piece.
func pieceRecipe(
	id CraftRecipeID, piece BuildPieceID, made uint16, minutes int, station CraftStation, inputs ...CraftGoods,
) CraftRecipe {
	definition := piece.Piece()

	return CraftRecipe{
		ID: id, Name: definition.Name, Category: definition.Category, Minutes: minutes, Station: station,
		Inputs: inputs, Outputs: []CraftGoods{PieceGoods(piece, made)},
	}
}

// CraftRecipes returns every recipe in menu order. The Workbench and Sawmill need no station, so
// a party with bare hands can gather, make tools and build its first stations (ref:CRAFT-0008).
func CraftRecipes() []CraftRecipe {
	return []CraftRecipe{
		{ID: RecipeAxe, Name: "Axe", Category: CraftCategoryTools, Minutes: 30, Station: CraftStationNone,
			Inputs:  []CraftGoods{MaterialGoods(Wood, 2), MaterialGoods(Stone, 1)},
			Outputs: []CraftGoods{ToolGoods(Axe, 1)}},
		{ID: RecipePick, Name: "Pick", Category: CraftCategoryTools, Minutes: 30, Station: CraftStationNone,
			Inputs:  []CraftGoods{MaterialGoods(Wood, 2), MaterialGoods(Stone, 2)},
			Outputs: []CraftGoods{ToolGoods(Pick, 1)}},
		{ID: RecipePlanks, Name: "Planks", Category: CraftCategoryMaterials, Minutes: 20, Station: CraftStationSawmill,
			Inputs:  []CraftGoods{MaterialGoods(Wood, 2)},
			Outputs: []CraftGoods{MaterialGoods(Planks, 4)}},
		{ID: RecipeCutStone, Name: "Cut stone", Category: CraftCategoryMaterials, Minutes: 20, Station: CraftStationWorkbench,
			Inputs:  []CraftGoods{MaterialGoods(Stone, 3)},
			Outputs: []CraftGoods{MaterialGoods(CutStone, 2)}},
		{ID: RecipeIngots, Name: "Ingots", Category: CraftCategoryMaterials, Minutes: 30, Station: CraftStationForge,
			Inputs:  []CraftGoods{MaterialGoods(Ore, 2), MaterialGoods(Wood, 1)},
			Outputs: []CraftGoods{MaterialGoods(Ingots, 1)}},
		pieceRecipe(RecipeWorkbench, PieceWorkbench, 1, 60, CraftStationNone, MaterialGoods(Wood, 6), MaterialGoods(Stone, 2)),
		pieceRecipe(RecipeSawmill, PieceSawmill, 1, 60, CraftStationNone, MaterialGoods(Wood, 8), MaterialGoods(Stone, 4)),
		pieceRecipe(RecipeForge, PieceForge, 1, 90, CraftStationWorkbench, MaterialGoods(CutStone, 8), MaterialGoods(Ore, 3)),
		pieceRecipe(RecipeAlchemyTable, PieceAlchemyTable, 1, 60, CraftStationWorkbench,
			MaterialGoods(Planks, 6), MaterialGoods(Ingots, 1)),
		pieceRecipe(RecipeFireplace, PieceFireplace, 1, 60, CraftStationWorkbench, MaterialGoods(CutStone, 6)),
		pieceRecipe(RecipeStoneWall, PieceStoneWall, 1, 10, CraftStationWorkbench, MaterialGoods(CutStone, 1)),
		pieceRecipe(RecipeWindow, PieceWindow, 1, 15, CraftStationWorkbench, MaterialGoods(CutStone, 1), MaterialGoods(Planks, 1)),
		pieceRecipe(RecipeDoor, PieceDoor, 1, 20, CraftStationWorkbench, MaterialGoods(Planks, 3)),
		pieceRecipe(RecipeFenceEastWest, PieceFenceEastWest, 1, 10, CraftStationWorkbench, MaterialGoods(Planks, 1)),
		pieceRecipe(RecipeFenceNorthSouth, PieceFenceNorthSouth, 1, 10, CraftStationWorkbench, MaterialGoods(Planks, 1)),
		pieceRecipe(RecipeWoodFloor, PieceWoodFloor, 2, 10, CraftStationWorkbench, MaterialGoods(Planks, 1)),
		pieceRecipe(RecipeStoneFloor, PieceStoneFloor, 2, 10, CraftStationWorkbench, MaterialGoods(CutStone, 1)),
		pieceRecipe(RecipeTableWestEnd, PieceTableWestEnd, 1, 15, CraftStationWorkbench, MaterialGoods(Planks, 2)),
		pieceRecipe(RecipeTableMiddle, PieceTableMiddle, 1, 15, CraftStationWorkbench, MaterialGoods(Planks, 2)),
		pieceRecipe(RecipeTableEastEnd, PieceTableEastEnd, 1, 15, CraftStationWorkbench, MaterialGoods(Planks, 2)),
		pieceRecipe(RecipeChairForward, PieceChairForward, 1, 10, CraftStationWorkbench, MaterialGoods(Planks, 1)),
		pieceRecipe(RecipeChairLeft, PieceChairLeft, 1, 10, CraftStationWorkbench, MaterialGoods(Planks, 1)),
		pieceRecipe(RecipeChairBack, PieceChairBack, 1, 10, CraftStationWorkbench, MaterialGoods(Planks, 1)),
		pieceRecipe(RecipeChairRight, PieceChairRight, 1, 10, CraftStationWorkbench, MaterialGoods(Planks, 1)),
		{ID: RecipeBed, Name: "Bed", Category: CraftCategoryFurnishings, Minutes: 30, Station: CraftStationWorkbench,
			Inputs:  []CraftGoods{MaterialGoods(Planks, 4)},
			Outputs: []CraftGoods{PieceGoods(PieceBedHead, 1), PieceGoods(PieceBedFoot, 1)}},
		pieceRecipe(RecipeDresser, PieceDresser, 1, 20, CraftStationWorkbench, MaterialGoods(Planks, 3)),
		pieceRecipe(RecipeBookcase, PieceBookcase, 1, 20, CraftStationWorkbench, MaterialGoods(Planks, 3)),
		pieceRecipe(RecipeEndTable, PieceEndTable, 1, 10, CraftStationWorkbench, MaterialGoods(Planks, 1)),
		pieceRecipe(RecipeBarrel, PieceBarrel, 1, 15, CraftStationWorkbench, MaterialGoods(Planks, 2)),
		pieceRecipe(RecipeStorageChest, PieceStorageChest, 1, 20, CraftStationWorkbench, MaterialGoods(Planks, 4)),
		{ID: RecipeCookFood, Name: "Cook food", Category: CraftCategoryFood, Minutes: 20, Station: CraftStationFireplace,
			Inputs:  []CraftGoods{MaterialGoods(Forage, 2), MaterialGoods(Wood, 1)},
			Outputs: []CraftGoods{ProvisionGoods(Food, 3)}},
	}
}

// CraftRecipeByID returns one recipe.
func CraftRecipeByID(id CraftRecipeID) (CraftRecipe, bool) {
	for _, recipe := range CraftRecipes() {
		if recipe.ID == id {
			return recipe, true
		}
	}

	return CraftRecipe{}, false
}
