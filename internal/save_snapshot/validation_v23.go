package save_snapshot

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func validateOpenDoor(snapshot SaveSnapshot, add func(string, string, string)) {
	door := snapshot.OpenDoor
	if door == nil {
		return
	}
	position := door.Position
	validatePosition("openDoor.position", position, add)
	inTowne := position.Location >= int(references.Moonglow) && position.Location <= int(references.SerpentsHold) &&
		position.Floor >= -1 && position.Floor <= 3
	// A door the party built on its own deed map opens and closes like a town door (ref:CRAFT-0010).
	onDeedMap := position.Location == int(references.DeedLocation) && position.Floor == 0
	if !inTowne && !onDeedMap || position.Location != snapshot.World.Position.Location || position.X > 31 || position.Y > 31 {
		add("openDoor.position", "inconsistent", "timed door must belong to the active small-map visit")
	}
	if door.RemainingTurns < 0 || door.RemainingTurns > 2 {
		add("openDoor.remainingTurns", "out_of_range", "door countdown must be 0 through 2")
	}
	if door.ClosedTile != int(indexes.RegularDoor) && door.ClosedTile != int(indexes.RegularDoorView) {
		add("openDoor.closedTile", "unsupported", "timed opening must retain its exact unlocked variant")
	}
	for _, mutation := range snapshot.MapMutations {
		if mutation.Position == position && mutation.Kind == MapMutationSetObject && mutation.Value != door.ClosedTile {
			add("openDoor.closedTile", "inconsistent", "persisted door override must contain its closed identity")
		}
	}
}
