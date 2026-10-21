package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// IsHelm returns true if this equipment is a helm.
func (e Equipment) IsHelm() bool {
	return e.GetSlotType() == SlotHelm
}

// IsRing returns true if this equipment is a ring.
func (e Equipment) IsRing() bool {
	return e.GetSlotType() == SlotRing
}

// IsAmulet returns true if this equipment is an amulet.
func (e Equipment) IsAmulet() bool {
	return e.GetSlotType() == SlotAmulet
}

// DefinitionForCatalog returns the active merchant assigned to a family row.
// Source rows with no active merchant are deliberately absent.
func (r *MerchantReferences) DefinitionForCatalog(
	family MerchantFamily,
	row int,
) (MerchantDefinition, bool) {
	definition, ok := r.byCatalog[merchantCatalogKey{family: family, row: row}]

	return definition, ok
}

// ToQuestItem returns the corresponding QuestItem shard for this Shadowlord.
func (s Shadowlord) ToQuestItem() QuestItem {
	switch s {
	case Faulinei:
		return ShardOfFalsehood
	case Astaroth:
		return ShardOfHatred
	case Nosfentor:
		return ShardOfCowardice
	default:
		return ShardOfFalsehood
	}
}

// HasSignAt returns true if there is a sign at the specified location and coordinates.
func (sr *SignReferences) HasSignAt(location Location, floor, x, y int) bool {
	return sr.GetSignAt(location, floor, x, y) != nil
}

func (t *Tile) IsWalkableDuringWander() bool {
	// Actual NPC steps allow beach and block chairs outside scheduled travel; ref:NPC-0004.
	if t.Index >= indexes.OutsidePath1 && t.Index <= indexes.OutsidePath4 {
		return true
	}

	return t.IsNPCTerrainPassable() && !t.IsChair()
}

// IsTable returns true if this tile is a table
func (t *Tile) IsTable() bool {
	return t.Is(indexes.TableLeft) || t.Is(indexes.TableMiddle) || t.Is(indexes.TableRight) ||
		t.Is(indexes.TableFoodTop) || t.Is(indexes.TableFoodBottom) || t.Is(indexes.TableFoodBoth) ||
		t.Is(indexes.CandleOnTable)
}
