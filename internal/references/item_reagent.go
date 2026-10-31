package references

// Reagent is one of the eight magic reagents.
type Reagent int

// ID returns the reagent number within its item type.
func (r Reagent) ID() int {
	return int(r)
}

// Type returns ItemTypeReagent.
func (r Reagent) Type() ItemType {
	return ItemTypeReagent
}

// Reagents in original order, starting at SulfurAsh.
//
//goland:noinspection GoUnusedConst
const (
	SulfurAsh    Reagent = 0
	Ginseng      Reagent = 1
	Garlic       Reagent = 2
	SpiderSilk   Reagent = 3
	BloodMoss    Reagent = 4
	BlackPearl   Reagent = 5
	NightShade   Reagent = 6
	MandrakeRoot Reagent = 7
)

// Name returns the player-facing reagent name used by inventory and shops.
func (r Reagent) Name() string {
	item, ok := newReagentItemReferences()[r]
	if !ok {
		return "Unknown"
	}

	return item.ItemName
}
