package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

var (
	errAddDebugSearchProvision = errors.New("cannot add debug search provision")
	errAlterQuestItem          = errors.New("cannot alter quest item")
)

// AddDebugSearchProvision places a supported provision on the current map without replacing a stack.
func (g *GameState) AddDebugSearchProvision(
	position references.Position,
	provision references.Provision,
	quantity uint16,
) error {
	if g.IsOutOfBounds(position) {
		return fmt.Errorf(
			"%w: position (%d,%d) is outside the current map",
			errAddDebugSearchProvision,
			position.X,
			position.Y,
		)
	}
	if provision != references.Gold && provision != references.Torches &&
		provision != references.Key && provision != references.Food {
		return fmt.Errorf("%w: unsupported provision %d", errAddDebugSearchProvision, provision)
	}

	err := g.ItemStacksMap.AddOrMerge(&position, references.ItemAndQuantity{
		Item:     provision,
		Quantity: quantity,
	})
	if err != nil {
		return fmt.Errorf("%w: adding provision to item stack: %w", errAddDebugSearchProvision, err)
	}

	return nil
}

// ResetAuthenticSearchFoundFlags clears only one-time authentic search-location flags.
func (g *GameState) ResetAuthenticSearchFoundFlags() (int, int) {
	return g.SearchManager.authenticLocationManager.resetOneTimeFoundStatus()
}

// SetQuestItemPresent ensures a quest item is present or removes its entire quantity.
func (g *GameState) SetQuestItemPresent(item references.QuestItem, present bool) (uint16, uint16, error) {
	if !item.IsValid() {
		return 0, 0, fmt.Errorf("%w: unsupported quest item %d", errAlterQuestItem, item)
	}

	before := g.PartyState.Inventory.QuestItems.Get(item)
	after := before
	if present && before == 0 {
		after = 1
	} else if !present && before > 0 {
		after = 0
	}
	if after == before {
		return before, after, nil
	}

	g.PartyState.Inventory.QuestItems.Set(item, after)
	g.SystemCallbacks.Screen.RefreshInventoryDisplay()

	return before, after, nil
}
