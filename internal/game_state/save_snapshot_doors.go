package game_state

import (
	"errors"
	"fmt"
	"slices"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func (g *GameState) captureOpenDoor() *save_snapshot.OpenDoorSnapshot {
	door := g.MapState.TrackedOpenDoor()
	if door == nil {
		return nil
	}

	return &save_snapshot.OpenDoorSnapshot{
		Position: positionSnapshot(door.Position), RemainingTurns: door.RemainingTurns, ClosedTile: int(door.ClosedTile),
	}
}

// canonicalizeOpenDoorMutations substitutes the closed identity in a detached
// list. The timed floor replacement is reconstructed only after persisted map
// mutations on load. Occupancy-only openings never enter the mutation list.
func (g *GameState) canonicalizeOpenDoorMutations(mutations []save_snapshot.MapMutationSnapshot) []save_snapshot.MapMutationSnapshot {
	door := g.captureOpenDoor()
	if door == nil {
		return mutations
	}
	result := slices.Clone(mutations)
	for i := range result {
		if result[i].Position == door.Position && result[i].Kind == save_snapshot.MapMutationSetObject {
			if result[i].Value != int(indexes.BrickFloor) {
				panic("tracked open door must have a floor override")
			}
			result[i].Value = door.ClosedTile

			return result
		}
	}
	panic("tracked open door is missing its captured override")
}

// validateOpenDoorSnapshotCompatibility checks semantic map identity and the
// terrain that hydration will see after applying both persistent layers.
// In particular, zero turns is valid without an occupant at a stable boundary.
func validateOpenDoorSnapshotCompatibility(
	door *save_snapshot.OpenDoorSnapshot,
	world save_snapshot.PositionSnapshot,
	mutations []save_snapshot.MapMutationSnapshot,
	gameReferences *references.GameReferences,
) error {
	if door == nil {
		return nil
	}
	position := door.Position
	location := references.Location(position.Location)
	if location.GetMapType() != references.SmallMapType || position.Location != world.Location {
		return errors.New("openDoor.position: door must belong to the active small map")
	}
	if position.X < 0 || position.Y < 0 {
		return errors.New("openDoor.position: negative coordinates")
	}
	if err := validateSnapshotPosition(gameReferences, "openDoor.position", position); err != nil {
		return err
	}
	if door.RemainingTurns < 0 || door.RemainingTurns > map_state.DefaultOpenDoorTurns || !indexes.SpriteIndex(door.ClosedTile).IsUnlockedDoor() {
		return errors.New("openDoor: invalid countdown or closed variant")
	}
	source := gameReferences.LocationReferences.GetLocationReference(location)
	base := source.GetTileNumber(playerLocation(position).Position, references.FloorNumber(position.Floor))
	override := indexes.NoSprites
	for _, mutation := range mutations {
		if mutation.Position != position {
			continue
		}
		switch mutation.Kind {
		case save_snapshot.MapMutationSetTile:
			base = indexes.SpriteIndex(mutation.Value)
		case save_snapshot.MapMutationSetObject:
			if mutation.Value > 0 {
				override = indexes.SpriteIndex(mutation.Value)
			}
		}
	}
	if override > 0 {
		base = override
	}
	if int(base) != door.ClosedTile {
		return fmt.Errorf("openDoor.closedTile: persistent terrain %d does not match %d", base, door.ClosedTile)
	}

	return nil
}

// restoreOpenDoor runs after actor and map-mutation hydration. Occupancy is
// resolved dynamically from those final actors; no sprite-layer refresh is needed.
func (g *GameState) restoreOpenDoor(door *save_snapshot.OpenDoorSnapshot) error {
	if door == nil {
		return nil
	}
	if err := g.MapState.RestoreOpenDoor(map_state.TrackedOpenDoor{
		Position: playerLocation(door.Position), RemainingTurns: door.RemainingTurns, ClosedTile: indexes.SpriteIndex(door.ClosedTile),
	}); err != nil {
		return fmt.Errorf("hydrating open door: %w", err)
	}

	return nil
}
