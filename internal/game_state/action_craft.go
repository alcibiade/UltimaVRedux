package game_state

import (
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// CraftRecipeOption is one recipe as presented to the Craft menu.
type CraftRecipeOption struct {
	Recipe     references.CraftRecipe
	Affordable bool
	Owned      uint16
}

// CraftRecipeOptions lists every recipe with whether the party can afford it now.
func (g *GameState) CraftRecipeOptions() []CraftRecipeOption {
	inventory := &g.PartyState.Inventory
	recipes := references.CraftRecipes()
	options := make([]CraftRecipeOption, 0, len(recipes))
	for _, recipe := range recipes {
		options = append(options, CraftRecipeOption{
			Recipe:     recipe,
			Affordable: inventory.CanAfford(recipe.Cost),
			Owned:      inventory.Tools.Get(recipe.Output),
		})
	}

	return options
}

// ApplyCraft makes one tool from materials.
// REDUX DEVIATION: CRAFT-0003: Redux-only crafting; see ref:CRAFT-0003.
func (g *GameState) ApplyCraft(recipeID int) error {
	if !g.IsCraftingEnabled() {
		return ErrCraftingDisabled
	}
	var recipe references.CraftRecipe
	found := false
	for _, candidate := range references.CraftRecipes() {
		if candidate.ID == recipeID {
			recipe, found = candidate, true
		}
	}
	if !found {
		return fmt.Errorf("%w: %d", ErrUnknownRecipe, recipeID)
	}
	inventory := &g.PartyState.Inventory
	if !inventory.SpendMaterials(recipe.Cost) {
		g.SystemCallbacks.Message.AddRowStr(fmt.Sprintf("You need %s.", recipe.Cost.Describe()))

		return ErrNotEnoughMaterials
	}
	inventory.Tools.IncrementBy(recipe.Output, 1)
	article := "a"
	if strings.ContainsRune("AEIOU", rune(recipe.Name[0])) {
		article = "an"
	}
	g.SystemCallbacks.Message.AddRowStr(fmt.Sprintf("You craft %s %s.", article, recipe.Name))
	g.AdvanceGameplayMinutesAndProcessProvisions(recipe.Minutes)

	return nil
}
