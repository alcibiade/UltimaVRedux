package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// REDUX DEVIATION: CRAFT-0007: the original has no crafting. With Gameplay.Crafting.Enabled,
// Alt+C turns carried materials into tools from a data-driven recipe table; see ref:CRAFT-0007.

// Craft rejection reasons.
var (
	ErrCraftNotHere        = errors.New("crafting does not apply here")
	ErrCraftUnknownRecipe  = errors.New("unknown recipe")
	ErrCraftNeedsStation   = errors.New("the recipe's station is not within reach")
	ErrCraftLacksInputs    = errors.New("the party lacks the recipe's inputs")
	ErrCraftNoRoomToOutput = errors.New("the party cannot carry the recipe's output")
)

// CraftRecipeOption is one recipe as the Craft menu shows it.
type CraftRecipeOption struct {
	Recipe references.CraftRecipe
	// Affordable means the party carries every input.
	Affordable bool
	// StationReady means the recipe's station, if any, is within reach.
	StationReady bool
}

// Available reports whether the recipe can be crafted now.
func (o CraftRecipeOption) Available() bool {
	return o.Affordable && o.StationReady
}

// CraftRecipeOptions lists every recipe in menu order with what the party lacks for it.
func (g *GameState) CraftRecipeOptions() []CraftRecipeOption {
	recipes := references.CraftRecipes()
	options := make([]CraftRecipeOption, 0, len(recipes))
	for _, recipe := range recipes {
		options = append(options, CraftRecipeOption{
			Recipe: recipe, Affordable: g.carriesCraftGoods(recipe.Inputs), StationReady: g.craftStationWithinReach(recipe.Station),
		})
	}

	return options
}

// Craft makes one recipe (ref:CRAFT-0007). On success it spends the inputs, adds the outputs,
// prints what was made and spends the recipe's minutes beyond the ordinary turn; the caller then
// completes the ordinary command cycle, so the craft costs exactly the recipe's minutes. A
// refusal prints why, spends nothing and must not be followed by a command cycle. Crafting
// consumes no game RNG.
func (g *GameState) Craft(id references.CraftRecipeID) error {
	messages := g.SystemCallbacks.Message
	if !g.CraftingCommandApplies(CraftingCraft) {
		messages.AddRowStr("Not here!")

		return ErrCraftNotHere
	}
	recipe, ok := references.CraftRecipeByID(id)
	if !ok {
		return fmt.Errorf("%w: %d", ErrCraftUnknownRecipe, id)
	}
	if !g.craftStationWithinReach(recipe.Station) {
		messages.AddRowStr(fmt.Sprintf("Thou dost need a %s nearby!", recipe.Station.Name()))

		return ErrCraftNeedsStation
	}
	if !g.carriesCraftGoods(recipe.Inputs) {
		messages.AddRowStr(fmt.Sprintf("Thou dost need %s.", references.DescribeCraftGoods(recipe.Inputs)))

		return ErrCraftLacksInputs
	}
	for _, output := range recipe.Outputs {
		if g.craftGoodsQuantity(output)+output.Quantity > craftGoodsCap(output) {
			messages.AddRowStr(fmt.Sprintf("Thou canst carry no more %s!", output.Name()))

			return ErrCraftNoRoomToOutput
		}
	}
	for _, input := range recipe.Inputs {
		g.changeCraftGoods(input, false)
	}
	for _, output := range recipe.Outputs {
		g.changeCraftGoods(output, true)
	}
	messages.AddRowStr(fmt.Sprintf("Crafted %s.", references.DescribeCraftGoods(recipe.Outputs)))
	g.spendCraftingLabor(recipe.Minutes)

	return nil
}

// craftStationWithinReach reports whether a recipe's station is usable where the party stands.
// Step 3 (ADR-014) adds the station pieces and their "within one square" check here.
func (g *GameState) craftStationWithinReach(station references.CraftStation) bool {
	switch station {
	case references.CraftStationNone:
		return true
	default:
		panic(fmt.Sprintf("craft station %d has no reach rule", station))
	}
}

func (g *GameState) carriesCraftGoods(goods []references.CraftGoods) bool {
	for _, item := range goods {
		if g.craftGoodsQuantity(item) < item.Quantity {
			return false
		}
	}

	return true
}

// craftGoodsQuantity returns how many of the goods' identity the party carries.
func (g *GameState) craftGoodsQuantity(goods references.CraftGoods) uint16 {
	inventory := &g.PartyState.Inventory
	switch goods.Kind {
	case references.CraftGoodsMaterial:
		return inventory.Materials.Get(references.Material(goods.ID))
	case references.CraftGoodsTool:
		return inventory.Tools.Get(references.Tool(goods.ID))
	case references.CraftGoodsReagent:
		return inventory.Reagent.Get(references.Reagent(goods.ID))
	default:
		panic(fmt.Sprintf("craft goods kind %d has no inventory", goods.Kind))
	}
}

func craftGoodsCap(goods references.CraftGoods) uint16 {
	switch goods.Kind {
	case references.CraftGoodsMaterial:
		return party_state.ItemQuantityLargeMax
	case references.CraftGoodsTool, references.CraftGoodsReagent:
		return party_state.ItemQuantitySmallMax
	default:
		panic(fmt.Sprintf("craft goods kind %d has no inventory", goods.Kind))
	}
}

// changeCraftGoods adds or removes goods the caller has already checked.
func (g *GameState) changeCraftGoods(goods references.CraftGoods, add bool) {
	inventory := &g.PartyState.Inventory
	var changed bool
	switch goods.Kind {
	case references.CraftGoodsMaterial:
		material := references.Material(goods.ID)
		if add {
			inventory.Materials.IncrementBy(material, goods.Quantity)
		}
		changed = add || inventory.Materials.DecrementBy(material, goods.Quantity)
	case references.CraftGoodsTool:
		tool := references.Tool(goods.ID)
		if add {
			inventory.Tools.IncrementBy(tool, goods.Quantity)
		}
		changed = add || inventory.Tools.DecrementBy(tool, goods.Quantity)
	case references.CraftGoodsReagent:
		reagent := references.Reagent(goods.ID)
		if add {
			inventory.Reagent.IncrementBy(reagent, goods.Quantity)
		}
		changed = add || inventory.Reagent.DecrementBy(reagent, goods.Quantity)
	default:
		panic(fmt.Sprintf("craft goods kind %d has no inventory", goods.Kind))
	}
	if !changed {
		panic("checked craft input could not be spent")
	}
}
