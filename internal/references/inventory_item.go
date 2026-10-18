package references

import (
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// InventoryItemReferences holds detached metadata indexed by each typed item family.
type InventoryItemReferences struct {
	Equipment map[Equipment]InventoryItem
	Reagent   map[Reagent]InventoryItem
	Spell     map[Spell]InventoryItem
	Scroll    map[Scroll]InventoryItem
	Potion    map[Potion]InventoryItem
	QuestItem map[QuestItem]InventoryItem
	Moonstone map[Moonstone]InventoryItem
	Provision map[Provision]InventoryItem
}

// NewInventoryItemsReferences assembles detached presentation metadata from the
// owning item families. Identity comes from typed map keys, never array order.
func NewInventoryItemsReferences() *InventoryItemReferences {
	return &InventoryItemReferences{
		Equipment: newEquipmentItemReferences(),
		Reagent:   newReagentItemReferences(),
		Spell:     newSpellItemReferences(),
		Scroll:    newScrollItemReferences(),
		Potion:    newPotionItemReferences(),
		QuestItem: newQuestItemReferences(),
		Moonstone: newMoonstoneItemReferences(),
		Provision: newProvisionItemReferences(),
	}
}

// GetReferenceByItem resolves presentation metadata using the item's stable typed identity.
func (i *InventoryItemReferences) GetReferenceByItem(item Item) InventoryItem {
	switch item.Type() {
	case ItemTypeEquipment:
		return i.Equipment[Equipment(item.ID())]
	case ItemTypeReagent:
		return i.Reagent[Reagent(item.ID())]
	case ItemTypeSpell:
		return i.Spell[Spell(item.ID())]
	case ItemTypePotion:
		return i.Potion[Potion(item.ID())]
	case ItemTypeScroll:
		return i.Scroll[Scroll(item.ID())]
	case ItemTypeMoonstone:
		return i.Moonstone[Moonstone(item.ID())]
	case ItemTypeProvision:
		return i.Provision[Provision(item.ID())]
	case ItemTypeQuestItem:
		return i.QuestItem[QuestItem(item.ID())]
	}
	// Programming bug: all ItemType values should be handled above
	panic("Unexpected: item type invalid")
}

func (i *InventoryItemReferences) getItemsAsStrings(itemStack *ItemStack) []string {
	items := make([]string, 0, len(itemStack.Items))
	for _, item := range itemStack.Items {
		items = append(items, i.GetReferenceByItem(item.Item).FullDisplayName())
	}

	return items
}

// GetListOfItems lists complete identified labels in stack order, one per line.
func (i *InventoryItemReferences) GetListOfItems(itemStack *ItemStack) string {
	itemStrings := i.getItemsAsStrings(itemStack)

	return strings.Join(itemStrings, "\n")
}

// getScrollItem returns the scroll's InventoryItem or panics if not found.
// Missing metadata is a programming error.
func (i *InventoryItemReferences) getScrollItem(scroll Scroll) InventoryItem {
	item, ok := i.Scroll[scroll]
	if !ok {
		panic(fmt.Sprintf("scroll %d not found in InventoryItemReferences", scroll))
	}

	return item
}

// GetScrollEffectDuration returns the duration from the scroll definition.
// Panics if scroll not found (programming/data error).
func (i *InventoryItemReferences) GetScrollEffectDuration(scroll Scroll) int {
	return i.getScrollItem(scroll).EffectDuration
}

// getQuestItem returns the quest item's InventoryItem or panics if not found.
// Missing metadata is a programming error.
func (i *InventoryItemReferences) getQuestItem(q QuestItem) InventoryItem {
	item, ok := i.QuestItem[q]
	if !ok {
		panic(fmt.Sprintf("quest item %d not found in InventoryItemReferences", q))
	}

	return item
}

// GetQuestItemDisplayName returns the full display name from the quest-item definition.
// Panics if quest item not found (programming/data error).
func (i *InventoryItemReferences) GetQuestItemDisplayName(q QuestItem) string {
	return i.getQuestItem(q).FullDisplayName()
}

// GetQuestItemPickupMessage returns the pickup message from the quest-item definition.
// Panics if quest item not found (programming/data error).
func (i *InventoryItemReferences) GetQuestItemPickupMessage(q QuestItem) string {
	return i.getQuestItem(q).ItemPickupMessage
}

// GetQuestItemSpriteIndex returns the sprite index from the quest-item definition.
// This is used for rendering quest items in the world.
// Panics if quest item not found (programming/data error).
func (i *InventoryItemReferences) GetQuestItemSpriteIndex(q QuestItem) indexes.SpriteIndex {
	return i.getQuestItem(q).ItemSprite
}
