package game_state

import (
	"errors"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

var (
	// ErrDockVehicleUnsupportedType indicates a non-ship vehicle request.
	ErrDockVehicleUnsupportedType = errors.New("unsupported dock vehicle type")
	// ErrDockVehicleInvalidLocation indicates an unknown or invalid Britannia dock.
	ErrDockVehicleInvalidLocation = errors.New("invalid britannia dock vehicle location")
	// ErrDockVehicleMapUnavailable indicates that overworld authority is absent.
	ErrDockVehicleMapUnavailable = errors.New("britannia dock vehicle map is unavailable")
	// ErrDockVehiclePositionOccupied indicates that another unit owns the dock tile.
	ErrDockVehiclePositionOccupied = errors.New("britannia dock vehicle position is occupied")
	// ErrDockVehicleMapFull indicates that overworld authority has no free unit slot.
	ErrDockVehicleMapFull = errors.New("britannia dock vehicle map is full")
)

// PlaceVehicleAtBritanniaDock inserts a ship into authoritative overworld state.
// Pending deliveries already occupy their dock and are activated later in place.
func (g *GameState) PlaceVehicleAtBritanniaDock(
	vehicleType references.VehicleType,
	dockLocation references.Location,
	pendingDelivery bool,
) (*map_units.NPCFriendly, error) {
	if vehicleType != references.FrigateVehicle && vehicleType != references.SkiffVehicle {
		return nil, ErrDockVehicleUnsupportedType
	}

	units, position, err := g.britanniaDockVehicleTarget(dockLocation)
	if err != nil {
		return nil, err
	}
	if units.GetMapUnitAtPositionAndFloorOrNil(position, references.MainFloor) != nil {
		return nil, ErrDockVehiclePositionOccupied
	}
	if !units.HasCapacity() {
		return nil, ErrDockVehicleMapFull
	}

	vehicle := map_units.NewNPCFriendlyVehiceNewRef(vehicleType, position, references.MainFloor)
	if pendingDelivery {
		vehicle.GetVehicleDetails().MarkPendingDelivery()
	}
	stored := units.InsertVehicle(*vehicle)
	if stored == nil {
		return nil, ErrDockVehicleMapFull
	}
	// InsertVehicle positions through SetPos, which derives a facing from the
	// origin delta. Original outdoor rules materializes purchased vessels eastward;
	// applying it twice makes both persistent direction fields canonical.
	// See ref:MERCH-0008.
	stored.SetDirection(references.Right)
	stored.SetDirection(references.Right)

	return stored, nil
}

// ActivatePendingBritanniaDockVehicle activates the stored delivery when the
// player reaches Britannia's overworld. The vehicle is not reinserted.
func (g *GameState) ActivatePendingBritanniaDockVehicle() *map_units.NPCFriendly {
	if g.MapState.PlayerLocation.Location != references.BritanniaUnderworld ||
		g.MapState.PlayerLocation.Floor != references.MainFloor {
		return nil
	}

	controller := g.LargeMapNPCAIController[references.OVERWORLD]
	if controller == nil || controller.GetNpcs() == nil {
		return nil
	}

	return controller.GetNpcs().ActivatePendingVehicleDelivery()
}

func (g *GameState) britanniaDockVehicleTarget(
	dockLocation references.Location,
) (*map_units.MapUnits, references.Position, error) {
	if g.GameReferences == nil || g.GameReferences.DockReferences == nil ||
		!isCanonicalDockLocation(dockLocation) {
		return nil, references.Position{}, ErrDockVehicleInvalidLocation
	}

	position, found := g.GameReferences.DockReferences.GetDockPositionChecked(dockLocation)
	if !found || !vehiclePositionInBounds(position, references.XLargeMapTiles, references.YLargeMapTiles) {
		return nil, references.Position{}, ErrDockVehicleInvalidLocation
	}

	controller := g.LargeMapNPCAIController[references.OVERWORLD]
	if controller == nil || controller.GetNpcs() == nil {
		return nil, references.Position{}, ErrDockVehicleMapUnavailable
	}

	return controller.GetNpcs(), position, nil
}

func isCanonicalDockLocation(location references.Location) bool {
	for _, dockLocation := range references.GetListOfAllLocationsWithDocks() {
		if dockLocation == location {
			return true
		}
	}

	return false
}
