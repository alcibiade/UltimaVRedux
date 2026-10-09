package map_units

import "github.com/bradhannah/UltimaVRedux/internal/references"

// SurfaceLootObjectCapacity is the 31 usable non-avatar object slots; ref:CMD-0009.
const SurfaceLootObjectCapacity = 31

// NewTransientSurfaceChest creates a generated chest with an independent object
// lifetime. A reused controller number never supplies authored NPC identity.
// Hydration restores the saved number and assessment after constructing it.
func NewTransientSurfaceChest(location references.Location, position references.Position,
	floor references.FloorNumber, npcNumber int, number byte,
) *NPCFriendly {
	if npcNumber < 0 || npcNumber >= MaximumNpcsPerMap {
		panic("transient chest number outside runtime namespace")
	}
	chest := NewNPCFriendlySurfaceChest(location, position, floor, npcNumber)
	chest.transientSurfaceChest = true
	chest.SetPos(position)
	chest.SetFloor(floor)
	chest.RestoreSurfaceChestNumber(number)

	return chest
}

// IsTransientSurfaceChest distinguishes generated objects from authored NPCs,
// independently of their sprite, coordinate, physical value or assessment.
func (friendly *NPCFriendly) IsTransientSurfaceChest() bool {
	return friendly != nil && friendly.transientSurfaceChest
}

// InsertTransientSurfaceChest reserves a stable runtime identity. The caller
// owns terrain/occupancy and shared loose-item admission before insertion.
func (m *MapUnits) InsertTransientSurfaceChest(location references.Location, position references.Position,
	floor references.FloorNumber, value byte,
) *NPCFriendly {
	number := m.getNextAvailableNPCIndexNumber()
	if number == -1 {
		return nil
	}
	chest := NewTransientSurfaceChest(location, position, floor, number, value)
	*m = append(*m, chest)

	return chest
}

// DiscardTransientSurfaceChests implements loose-object cleanup on departure.
func (m *MapUnits) DiscardTransientSurfaceChests() {
	kept := (*m)[:0]
	for _, unit := range *m {
		if chest, ok := unit.(*NPCFriendly); ok && chest.IsTransientSurfaceChest() {
			chest.SetVisible(false)

			continue
		}
		kept = append(kept, unit)
	}
	*m = kept
}
