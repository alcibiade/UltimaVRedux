package save_snapshot

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Historical atlas domains remain frozen even as runtime journal validation grows.
func validateHistoricalAtlasAuthority(records []journal.AtlasLandmarkObservation, add func(string, string, string)) {
	if len(records) > 38 {
		add("atlasLandmarks", "out_of_range", "historical atlas has at most 38 landmarks")
	}
	for index, record := range records {
		if record.SiteKey == journal.AtlasGuardiansSiteKey || record.Kind == "guardians" ||
			record.Source == journal.AtlasSourceSextantReceipt || record.SurveySource == journal.AtlasSourceSextantReceipt {
			add(fmt.Sprintf("atlasLandmarks[%d]", index), "unsupported", "historical atlas cannot contain Guardians or sextant-receipt authority")
		}
	}
}

func validateOpenDoor(snapshot SaveSnapshot, add func(string, string, string)) {
	door := snapshot.OpenDoor
	if door == nil {
		return
	}
	if snapshot.Version < 23 {
		add("openDoor", "unsupported", "timed door continuation requires snapshot 23")
	}
	position := door.Position
	validatePosition("openDoor.position", position, add)
	if position.Location < int(references.Moonglow) || position.Location > int(references.SerpentsHold) ||
		position.Location != snapshot.World.Position.Location || position.X > 31 || position.Y > 31 ||
		position.Floor < -1 || position.Floor > 3 {
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
