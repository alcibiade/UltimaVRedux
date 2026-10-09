package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// validateAtlasLandmarkCompatibility augments structural validation with installed
// immutable identities. Hydration validates evidence; it never acquires knowledge.
func validateAtlasLandmarkCompatibility(records []journal.AtlasLandmarkObservation, refs *references.GameReferences) error {
	for index, observation := range records {
		if observation.SiteKey == journal.AtlasGuardiansSiteKey {
			world := refs.OverworldLargeMapReference
			if world == nil || world.GetSpriteIndex(232, 235) != indexes.CodexAngelLeft || world.GetSpriteIndex(234, 235) != indexes.CodexAngelRight {
				return fmt.Errorf("atlasLandmarks[%d]: Guardians artwork does not match installed references", index)
			}

			continue
		}
		if observation.SiteKey >= 101 {
			continue // Physical shrine anchors are structurally validated.
		}
		world := refs.LocationReferences.WorldLocations
		if world == nil {
			return fmt.Errorf("atlasLandmarks[%d]: exterior references unavailable", index)
		}
		anchor, exists := world.LargeMapLocationPositions[references.Location(observation.SiteKey)]
		if !exists || anchor.Floor != 0 || int(anchor.Position.X) != observation.Site.X || int(anchor.Position.Y) != observation.Site.Y {
			return fmt.Errorf("atlasLandmarks[%d].site does not match its exterior reference", index)
		}
	}

	return nil
}
