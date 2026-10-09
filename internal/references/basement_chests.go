package references

import "github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"

// BasementChestSource identifies one stationary CASTLE.NPC authored chest.
type BasementChestSource struct {
	NPCNumber int
	Position  Position
}

// LordBritishBasementChests returns CASTLE.NPC records 23..25 in ledger order.
// The NPC schedule layout puts record 25's three X bytes (ref:NPC-0001) at
// 0x193..0x195: each is 0x0d (13), not 0x0a (10). The installed original
// schedules are loaded unchanged; this position is not a Redux relocation.
func LordBritishBasementChests() [3]BasementChestSource {
	return [3]BasementChestSource{
		{NPCNumber: 23, Position: Position{X: 16, Y: 21}},
		{NPCNumber: 24, Position: Position{X: 17, Y: 22}},
		{NPCNumber: 25, Position: Position{X: 13, Y: 23}},
	}
}

// BasementChestIndex validates the source identity rather than its runtime
// sprite/position alone. Generated references are separately marked at runtime.
func BasementChestIndex(reference NPCReference, npcNumber int) (int, bool) {
	if reference.Location != LordBritishsCastle || reference.GetSpriteIndex() != indexes.Chest {
		return 0, false
	}
	for index, source := range LordBritishBasementChests() {
		if npcNumber != source.NPCNumber {
			continue
		}
		for slot := range reference.Schedule.X {
			if Coordinate(reference.Schedule.X[slot]) != source.Position.X ||
				Coordinate(reference.Schedule.Y[slot]) != source.Position.Y ||
				FloorNumber(reference.Schedule.Floor[slot]) != Basement {
				return 0, false
			}
		}

		return index, true
	}

	return 0, false
}
