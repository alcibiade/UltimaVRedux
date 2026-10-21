package game_state

import (
	"errors"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

var (
	// ErrDebugVehicleUnsupportedType indicates a vehicle outside the request's allowed set.
	ErrDebugVehicleUnsupportedType = ErrDockVehicleUnsupportedType
	// ErrDebugVehicleUnsupportedContext indicates a transient or non-world gameplay context.
	ErrDebugVehicleUnsupportedContext = errors.New("debug vehicle request is not supported in the current context")
	// ErrDebugVehicleInvalidLocation indicates invalid coordinates, floor, placement, or dock selection.
	ErrDebugVehicleInvalidLocation = ErrDockVehicleInvalidLocation
	// ErrDebugVehicleMapUnavailable indicates that the authoritative owning controller is absent.
	ErrDebugVehicleMapUnavailable = ErrDockVehicleMapUnavailable
	// ErrDebugVehiclePositionOccupied indicates another unit already owns the destination tile and floor.
	ErrDebugVehiclePositionOccupied = ErrDockVehiclePositionOccupied
	// ErrDebugVehicleMapFull indicates the owning map has no free unit slot.
	ErrDebugVehicleMapFull = ErrDockVehicleMapFull
)

// DebugVehicleSpawnPlacement selects an authoritative vehicle spawn destination.
type DebugVehicleSpawnPlacement int

const (
	// DebugVehicleSpawnAtAvatar places a vehicle at the Avatar's current stable map position.
	DebugVehicleSpawnAtAvatar DebugVehicleSpawnPlacement = iota
	// DebugVehicleSpawnAtDock places a vehicle at an authoritative overworld dock.
	DebugVehicleSpawnAtDock
)

// DebugVehicleSpawnRequest describes a direct frigate or skiff insertion.
// DockLocation is used only with DebugVehicleSpawnAtDock.
type DebugVehicleSpawnRequest struct {
	VehicleType  references.VehicleType
	Placement    DebugVehicleSpawnPlacement
	DockLocation references.Location
}

// DebugForceBoardRequest describes a direct replacement of the party vehicle.
type DebugForceBoardRequest struct {
	VehicleType references.VehicleType
}

// DebugVehicleMapSnapshot is a detached view of the map slot affected by a request.
type DebugVehicleMapSnapshot struct {
	Location     references.Location
	DockLocation references.Location
	Position     references.Position
	Floor        references.FloorNumber
	UnitCount    int
	Vehicle      *map_units.VehicleDetailsSnapshot
}

// DebugVehicleSpawnResult reports exact map state surrounding a vehicle insertion.
type DebugVehicleSpawnResult struct {
	Before DebugVehicleMapSnapshot
	After  DebugVehicleMapSnapshot
}

// DebugForceBoardResult reports exact party and map state surrounding a forced boarding.
type DebugForceBoardResult struct {
	PartyBefore PartyVehicleSnapshot
	PartyAfter  PartyVehicleSnapshot
	MapBefore   DebugVehicleMapSnapshot
	MapAfter    DebugVehicleMapSnapshot
}

// SpawnDebugVehicle inserts a new frigate or skiff at the Avatar or an authoritative overworld dock.
// It performs no implicit turn advancement.
func (g *GameState) SpawnDebugVehicle(request DebugVehicleSpawnRequest) (DebugVehicleSpawnResult, error) {
	var result DebugVehicleSpawnResult
	if request.VehicleType != references.FrigateVehicle && request.VehicleType != references.SkiffVehicle {
		return result, ErrDebugVehicleUnsupportedType
	}

	controller, location, position, floor, err := g.debugVehicleSpawnTarget(request)
	if err != nil {
		return result, err
	}

	units := controller.GetNpcs()
	if units == nil {
		return result, ErrDebugVehicleMapUnavailable
	}
	result.Before = debugVehicleMapSnapshot(units, location, position, floor)
	if request.Placement == DebugVehicleSpawnAtDock {
		result.Before.DockLocation = request.DockLocation
	}
	result.After = result.Before
	if units.GetMapUnitAtPositionAndFloorOrNil(position, floor) != nil {
		return result, ErrDebugVehiclePositionOccupied
	}
	if len(*units) >= map_units.MaximumNpcsPerMap {
		return result, ErrDebugVehicleMapFull
	}

	if request.Placement == DebugVehicleSpawnAtDock {
		if _, err = g.PlaceVehicleAtBritanniaDock(request.VehicleType, request.DockLocation, false); err != nil {
			return result, err
		}
	} else {
		vehicle := map_units.NewNPCFriendlyVehiceNewRef(request.VehicleType, position, floor)
		if units.InsertVehicle(*vehicle) == nil {
			return result, ErrDebugVehicleMapFull
		}
	}
	result.After = debugVehicleMapSnapshot(units, location, position, floor)
	if request.Placement == DebugVehicleSpawnAtDock {
		result.After.DockLocation = request.DockLocation
	}

	return result, nil
}

// ForceBoardDebugVehicle replaces the party vehicle without checking terrain.
// This test/debug contract intentionally permits otherwise invalid terrain, but only on stable
// large or small maps; combat, camping, dungeon, and stale combat-scene states are rejected.
func (g *GameState) ForceBoardDebugVehicle(request DebugForceBoardRequest) (DebugForceBoardResult, error) {
	var result DebugForceBoardResult
	if !isBoardableDebugVehicle(request.VehicleType) {
		return result, ErrDebugVehicleUnsupportedType
	}
	if err := g.validateDebugVehicleAvatarContext(); err != nil {
		return result, err
	}

	units := g.CurrentNPCAIController.GetNpcs()
	if units == nil {
		return result, ErrDebugVehicleMapUnavailable
	}
	location := g.MapState.PlayerLocation.Location
	position := g.MapState.PlayerLocation.Position
	floor := g.MapState.PlayerLocation.Floor
	result.PartyBefore = g.CurrentPartyVehicleSnapshot()
	result.PartyAfter = result.PartyBefore
	result.MapBefore = debugVehicleMapSnapshot(units, location, position, floor)
	result.MapAfter = result.MapBefore

	currentType := result.PartyBefore.VehicleType
	attachCurrentSkiff := currentType == references.SkiffVehicle && request.VehicleType == references.FrigateVehicle
	releaseCurrentVehicle := currentType != references.NoPartyVehicle && !attachCurrentSkiff
	if releaseCurrentVehicle {
		if units.GetMapUnitAtPositionAndFloorOrNil(position, floor) != nil {
			return result, ErrDebugVehiclePositionOccupied
		}
		if len(*units) >= map_units.MaximumNpcsPerMap {
			return result, ErrDebugVehicleMapFull
		}
	}

	replacement := map_units.NewNPCFriendlyVehiceNewRef(request.VehicleType, position, floor)
	if attachCurrentSkiff {
		replacement.GetVehicleDetails().IncrementSkiffQuantity()
	}
	if releaseCurrentVehicle {
		released := g.PartyVehicle
		released.SetPos(position)
		released.SetFloor(floor)
		// The validated world positions are within the original byte-addressable map range.
		released.NPCReference.Schedule.OverrideAllPositions(
			byte(position.X),
			byte(position.Y),
		)
		if !units.AddVehicle(released) {
			return result, ErrDebugVehicleMapFull
		}
	}

	g.PartyVehicle = *replacement
	g.isSailing = false
	result.PartyAfter = g.CurrentPartyVehicleSnapshot()
	result.MapAfter = debugVehicleMapSnapshot(units, location, position, floor)

	return result, nil
}

//nolint:ireturn // Each placement branch has distinct authoritative controller validation.
func (g *GameState) debugVehicleSpawnTarget(
	request DebugVehicleSpawnRequest,
) (ai.NPCAIController, references.Location, references.Position, references.FloorNumber, error) {
	switch request.Placement {
	case DebugVehicleSpawnAtAvatar:
		if err := g.validateDebugVehicleAvatarContext(); err != nil {
			return nil, 0, references.Position{}, 0, err
		}

		return g.CurrentNPCAIController,
			g.MapState.PlayerLocation.Location,
			g.MapState.PlayerLocation.Position,
			g.MapState.PlayerLocation.Floor,
			nil
	case DebugVehicleSpawnAtDock:
		if err := g.validateDebugVehicleAvatarContext(); err != nil {
			return nil, 0, references.Position{}, 0, err
		}
		units, position, err := g.britanniaDockVehicleTarget(request.DockLocation)
		if err != nil {
			return nil, 0, references.Position{}, 0, err
		}
		controller := g.LargeMapNPCAIController[references.OVERWORLD]
		if units != controller.GetNpcs() {
			panic("Britannia dock vehicle target returned mismatched controller units")
		}

		return controller, references.BritanniaUnderworld, position, references.MainFloor, nil
	default:
		return nil, 0, references.Position{}, 0, ErrDebugVehicleInvalidLocation
	}
}

func (g *GameState) validateDebugVehicleAvatarContext() error {
	if g.CombatScene != nil || g.EventQueueManager.HasActiveWorkflows() || g.CurrentNPCAIController == nil {
		return ErrDebugVehicleUnsupportedContext
	}

	location := g.MapState.PlayerLocation
	switch location.Location.GetMapType() {
	case references.LargeMapType:
		if location.Location != references.BritanniaUnderworld ||
			(location.Floor != references.FloorNumber(references.OVERWORLD) &&
				location.Floor != references.FloorNumber(references.UNDERWORLD)) ||
			!vehiclePositionInBounds(location.Position, references.XLargeMapTiles, references.YLargeMapTiles) {
			return ErrDebugVehicleInvalidLocation
		}
		if controller, ok := g.CurrentNPCAIController.(*ai.NPCAIControllerLargeMap); ok &&
			controller.World != references.World(location.Floor) {
			return ErrDebugVehicleMapUnavailable
		}
	case references.SmallMapType:
		if !vehiclePositionInBounds(location.Position, references.XSmallMapTiles, references.YSmallMapTiles) {
			return ErrDebugVehicleInvalidLocation
		}
		if g.GameReferences != nil && g.GameReferences.LocationReferences != nil {
			locationReference := g.GameReferences.LocationReferences.GetLocationReference(location.Location)
			if locationReference == nil {
				return ErrDebugVehicleInvalidLocation
			}
			minimumFloor, maximumFloor := locationReference.GetFloorMinMax()
			if location.Floor < minimumFloor || location.Floor > maximumFloor {
				return ErrDebugVehicleInvalidLocation
			}
		}
	case references.CombatMapType, references.DungeonMapType, references.CampingMapType:
		return ErrDebugVehicleUnsupportedContext
	default:
		panic("unknown map type")
	}

	if g.CurrentNPCAIController.GetNpcs() == nil {
		return ErrDebugVehicleMapUnavailable
	}

	return nil
}

func debugVehicleMapSnapshot(
	units *map_units.MapUnits,
	location references.Location,
	position references.Position,
	floor references.FloorNumber,
) DebugVehicleMapSnapshot {
	snapshot := DebugVehicleMapSnapshot{
		Location: location, DockLocation: 0, Position: position, Floor: floor,
		UnitCount: 0, Vehicle: nil,
	}
	if units == nil {
		return snapshot
	}

	snapshot.UnitCount = len(*units)
	mapUnit := units.GetMapUnitAtPositionAndFloorOrNil(position, floor)
	if mapUnit == nil {
		return snapshot
	}
	friendly, ok := (*mapUnit).(*map_units.NPCFriendly)
	if !ok || friendly.NPCReference.GetNPCType() != references.Vehicle {
		return snapshot
	}
	vehicle := friendly.GetVehicleDetails().Snapshot()
	snapshot.Vehicle = &vehicle

	return snapshot
}

func vehiclePositionInBounds(position references.Position, width, height references.Coordinate) bool {
	return position.X >= 0 && position.X < width && position.Y >= 0 && position.Y < height
}

func isBoardableDebugVehicle(vehicleType references.VehicleType) bool {
	switch vehicleType {
	case references.HorseVehicle, references.CarpetVehicle, references.SkiffVehicle, references.FrigateVehicle:
		return true
	case references.NoPartyVehicle, references.NPC:
		return false
	default:
		panic("unknown vehicle type")
	}
}
