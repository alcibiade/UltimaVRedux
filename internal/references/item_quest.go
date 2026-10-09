package references

type QuestItem int

func (q QuestItem) ID() int {
	return int(q)
}

func (q QuestItem) Type() ItemType {
	return ItemTypeQuestItem
}

// Note: GetName(), GetPickupMessage(), and GetSpriteIndex() have been removed.
// Use InventoryItemReferences.GetQuestItemDisplayName(), GetQuestItemPickupMessage(),
// and GetQuestItemSpriteIndex() instead. These values come from item_quest_data.go.

const (
	Amulet           QuestItem = 0
	Crown            QuestItem = 1
	Sceptre          QuestItem = 2
	ShardOfFalsehood QuestItem = 3
	ShardOfHatred    QuestItem = 4
	ShardOfCowardice QuestItem = 5
	// Former SpecialItems - now unified under QuestItem
	Carpet      QuestItem = 6
	Grapple     QuestItem = 7
	Spyglass    QuestItem = 8
	HMSCape     QuestItem = 9
	PocketWatch QuestItem = 10
	BlackBadge  QuestItem = 11
	WoodenBox   QuestItem = 12
	Sextant     QuestItem = 13
)

// NumberOfQuestItems is the number of authentic quest inventory item values.
const NumberOfQuestItems = 14

// IsValid reports whether q identifies one of the authentic quest inventory items.
func (q QuestItem) IsValid() bool {
	return q >= Amulet && q < NumberOfQuestItems
}

// QuestItemCatalog returns every authentic quest inventory identity in save-data order.
func QuestItemCatalog() []QuestItem {
	items := make([]QuestItem, 0, NumberOfQuestItems)
	for item := Amulet; item < NumberOfQuestItems; item++ {
		items = append(items, item)
	}

	return items
}
