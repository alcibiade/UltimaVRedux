package references

// Item is an item that can be carried by a character.
type Item interface {
	ID() int        // Unique ID for the item
	Type() ItemType // Type of the item (e.g., Spell, Scroll, etc.)
}

// ItemType is the type of an item. An Item is a carriable object.
type ItemType int

const (
	ItemTypeReagent   ItemType = 0
	ItemTypeEquipment          = 1
	ItemTypeSpell              = 2
	// ItemTypeSpecialItem = 3 - REMOVED: Special items merged into QuestItem
	ItemTypeScroll    = 4
	ItemTypePotion    = 5
	ItemTypeShard     = 6
	ItemTypeQuestItem = 7
	ItemTypeMoonstone = 8
	ItemTypeProvision = 9
)

// ItemFromIdentity reconstructs a runtime item from its stable type and ID.
//
//nolint:cyclop,ireturn // The original item catalog is partitioned by fixed item type and uses Item polymorphism.
func ItemFromIdentity(itemType ItemType, id int) (Item, bool) {
	switch itemType {
	case ItemTypeReagent:
		return Reagent(id), id >= int(SulfurAsh) && id <= int(MandrakeRoot)
	case ItemTypeEquipment:
		return Equipment(id), id >= int(LeatherHelm) && id <= int(Ankh)
	case ItemTypeSpell:
		return Spell(id), id >= int(InLor) && id <= int(AnTym)
	case ItemTypeScroll:
		return Scroll(id), id >= int(ScrollVasLor) && id <= int(ScrollAnTym)
	case ItemTypePotion:
		return Potion(id), id >= int(Blue) && id <= int(White)
	case ItemTypeQuestItem:
		return QuestItem(id), QuestItem(id).IsValid()
	case ItemTypeMoonstone:
		return Moonstone(id), id >= int(NewMoon) && id <= int(CrescentWaning)
	case ItemTypeProvision:
		return Provision(id), id >= int(Torches) && id <= int(Gold)
	default:
		return nil, false
	}
}
