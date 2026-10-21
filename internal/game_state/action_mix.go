package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// MixRequest represents a request from the dialog to execute a mix operation.
// The dialog creates this when the user presses Enter with valid selection.
type MixRequest struct {
	Spell       references.Spell // The spell to mix
	Quantity    int              // Number of spells to mix (1-99)
	ReagentMask uint16           // Bitmask of selected reagents
}

// MixOutcome represents the result of a mix operation for UI display.
type MixOutcome struct {
	Success     bool        // True if reagents matched and spells created
	Message     string      // "Done!" or failure message
	IsNewRecipe bool        // True if this was a new recipe discovery
	SpellName   string      // Full spell name for confirmation display
	Quantity    int         // Requested quantity, retained for dialog reagent deductions
	Added       int         // Actual spells added after the inventory cap
	Stock       uint16      // Spell stock after successful mixing
	ReagentMask uint16      // Echo back for local inventory update in dialog
	TrapResult  *TrapResult // Boom trap result on failure (nil on success)
}

// ActionMixSmallMap reports whether the party can open the mix dialog on a town or keep map (it needs at least one reagent).
func (g *GameState) ActionMixSmallMap() bool {
	if !g.HasAnyReagents() {
		g.SystemCallbacks.UI.ShowActionFeedback("No reagents owned!", true)

		return false
	}
	// If we have reagents, the caller (GameScene) will open the dialog
	return true
}

// ActionMixLargeMap reports whether the party can open the mix dialog on the overworld or underworld (it needs at least one reagent).
func (g *GameState) ActionMixLargeMap() bool {
	if !g.HasAnyReagents() {
		g.SystemCallbacks.UI.ShowActionFeedback("No reagents owned!", true)

		return false
	}
	// If we have reagents, the caller (GameScene) will open the dialog
	return true
}

// ActionMixDungeonMap reports whether the party can open the mix dialog on a dungeon (it needs at least one reagent).
func (g *GameState) ActionMixDungeonMap() bool {
	if !g.HasAnyReagents() {
		g.SystemCallbacks.UI.ShowActionFeedback("No reagents owned!", true)

		return false
	}
	// If we have reagents, the caller (GameScene) will open the dialog
	return true
}

// HasAnyReagents returns true if the party has at least one reagent.
func (g *GameState) HasAnyReagents() bool {
	reagents := []references.Reagent{
		references.SulfurAsh,
		references.Ginseng,
		references.Garlic,
		references.SpiderSilk,
		references.BloodMoss,
		references.BlackPearl,
		references.NightShade,
		references.MandrakeRoot,
	}
	for _, r := range reagents {
		if g.PartyState.Inventory.Reagent.Get(r) > 0 {
			return true
		}
	}

	return false
}

// ExecuteMixing processes a mix request and returns the outcome.
// This consumes reagents BEFORE checking correctness (authentic Ultima V behavior).
// On success: creates spells, potentially records new recipe, returns success outcome.
// On failure: reagents already consumed, returns failure outcome.
func (g *GameState) ExecuteMixing(request MixRequest) MixOutcome {
	if request.Quantity < 1 || request.Quantity > g.MaximumMixQuantity(request.ReagentMask) {
		return MixOutcome{Message: "Insufficient reagents!"}
	}
	// Consume reagents first (Ultima V behavior - reagents lost even on failure)
	g.consumeReagentsForMix(request.ReagentMask, request.Quantity)

	// Get required reagents for this spell
	requiredMask := references.GetSpellReagents(request.Spell)

	// Check if selected reagents match required reagents
	if references.GetSpellInfo(request.Spell) == nil || request.ReagentMask != requiredMask {
		// Wrong recipe - reagents wasted, trigger boom trap
		trapResult := g.ExecuteBoom()
		message := "Reagents wasted!"
		if trapResult != nil {
			message = trapResult.Message
		}

		return MixOutcome{
			Success:     false,
			Message:     message,
			IsNewRecipe: false,
			SpellName:   references.GetSpellName(request.Spell),
			Quantity:    request.Quantity,
			ReagentMask: request.ReagentMask,
			TrapResult:  trapResult,
		}
	}

	// Correct recipe - create spells
	before := g.PartyState.Inventory.Spells.Get(request.Spell)
	g.PartyState.Inventory.Spells.IncrementBy(request.Spell, uint16(request.Quantity))
	stock := g.PartyState.Inventory.Spells.Get(request.Spell)

	// Check if this is a new recipe discovery
	isNewRecipe := !g.MagicState.IsRecipeKnown(request.Spell)
	if isNewRecipe {
		g.MagicState.RecordRecipe(request.Spell, request.ReagentMask)
	}
	g.ConfirmSpellKnowledge(request.Spell, ExperimentationKnowledgeSource())
	// REDUX DEVIATION: reagent mixing has no direct success sound; a failed recipe
	// instead reaches failed-experiment trap selection. Redux reserves a single success cue after creation.
	// See ref:MIX-0002.
	g.SystemCallbacks.Audio.PlaySoundEffect(SoundMixComplete)

	return MixOutcome{
		Success:     true,
		Message:     "Done!",
		IsNewRecipe: isNewRecipe,
		SpellName:   references.GetSpellName(request.Spell),
		Quantity:    request.Quantity,
		Added:       int(stock) - int(before),
		Stock:       stock,
		ReagentMask: request.ReagentMask,
	}
}

// MaximumMixQuantity validates the selected inventory, independently of whether
// the recipe is correct. Experimentation still consumes reagents on failure.
func (g *GameState) MaximumMixQuantity(mask uint16) int {
	if mask == 0 || mask&^uint16(0xff) != 0 {
		return 0
	}
	quantity := 99
	for reagent := references.SulfurAsh; reagent <= references.MandrakeRoot; reagent++ {
		if mask&(1<<uint(reagent)) != 0 {
			quantity = min(quantity, int(g.PartyState.Inventory.Reagent.Get(reagent)))
		}
	}

	return quantity
}

// consumeReagentsForMix deducts reagents based on the mask and quantity.
func (g *GameState) consumeReagentsForMix(reagentMask uint16, quantity int) {
	reagents := []references.Reagent{
		references.SulfurAsh,
		references.Ginseng,
		references.Garlic,
		references.SpiderSilk,
		references.BloodMoss,
		references.BlackPearl,
		references.NightShade,
		references.MandrakeRoot,
	}
	for _, r := range reagents {
		bit := uint16(1) << uint(r)
		if (reagentMask & bit) != 0 {
			// Consume 'quantity' of this reagent
			g.PartyState.Inventory.Reagent.DecrementBy(r, uint16(quantity))
		}
	}
}
