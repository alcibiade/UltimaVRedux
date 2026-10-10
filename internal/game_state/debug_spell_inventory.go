package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

var errInvalidDebugSpell = errors.New("invalid castable spell")

// GetDebugSpellInventorySnapshot returns every castable prepared spell quantity.
func (g *GameState) GetDebugSpellInventorySnapshot() party_state.SpellInventorySnapshot {
	return g.PartyState.Inventory.SpellSnapshot()
}

// SetDebugSpellQuantity assigns one castable prepared spell quantity, clamped to 99.
func (g *GameState) SetDebugSpellQuantity(spell references.Spell, quantity uint16) ([]DebugInventoryChange, error) {
	if spell < references.InLor || spell > references.AnTym {
		return nil, fmt.Errorf("%w: %d", errInvalidDebugSpell, spell)
	}
	before := g.PartyState.Inventory.Spells.Get(spell)
	after := min(quantity, party_state.ItemQuantitySmallMax)
	if before == after {
		return []DebugInventoryChange{}, nil
	}

	g.PartyState.Inventory.Spells.Set(spell, after)
	g.SystemCallbacks.Screen.RefreshInventoryDisplay()

	return []DebugInventoryChange{{Path: fmt.Sprintf("spells.%d", spell), Before: before, After: after}}, nil
}

// SetAllDebugSpellQuantitiesToMaximum sets all 48 castable spells to 99 atomically.
func (g *GameState) SetAllDebugSpellQuantitiesToMaximum() []DebugInventoryChange {
	snapshot := g.GetDebugSpellInventorySnapshot()
	changes := make([]DebugInventoryChange, 0, references.TotalSpellCount)
	for spell := references.InLor; spell <= references.AnTym; spell++ {
		before := snapshot[spell]
		if before == party_state.ItemQuantitySmallMax {
			continue
		}
		g.PartyState.Inventory.Spells.Set(spell, party_state.ItemQuantitySmallMax)
		changes = append(changes, DebugInventoryChange{
			Path: fmt.Sprintf("spells.%d", spell), Before: before, After: party_state.ItemQuantitySmallMax,
		})
	}
	if len(changes) > 0 {
		g.SystemCallbacks.Screen.RefreshInventoryDisplay()
	}

	return changes
}
