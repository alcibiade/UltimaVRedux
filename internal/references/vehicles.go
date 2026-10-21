package references

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// VehicleType is what the party travels on; NPC marks a non-party vehicle.
type VehicleType int

// Vehicle kinds; NoPartyVehicle means the party is on foot.
const (
	NoPartyVehicle VehicleType = iota
	CarpetVehicle
	HorseVehicle
	SkiffVehicle
	FrigateVehicle
	NPC
)

// GetUnBoardedSpriteByDirection returns the sprite for the vehicle when no one is aboard.
func (v VehicleType) GetUnBoardedSpriteByDirection(direction Direction) indexes.SpriteIndex {
	switch v {
	case CarpetVehicle:
		return indexes.Carpet2
	case HorseVehicle:
		if direction == Left || direction == Up {
			return indexes.HorseLeft
		}

		return indexes.HorseRight
	case NoPartyVehicle, SkiffVehicle, FrigateVehicle, NPC:
	}

	return v.GetBoardedSpriteByDirection(direction, direction)
}

// GetBoardedSpriteByDirection returns the sprite for the vehicle while the party is aboard, given the previous and current direction.
func (v VehicleType) GetBoardedSpriteByDirection(previousDirection, direction Direction) indexes.SpriteIndex {
	switch v {
	case CarpetVehicle:
		// note previousDirection exists only for horses and carpet because they do not have vertical
		// tiles, so we keep the previous left/right direction
		if direction == Left || (previousDirection == Left && (direction == Up || direction == Down)) {
			return indexes.RidingMagicCarpetLeft
		}

		return indexes.RidingMagicCarpetRight
	case HorseVehicle:
		if direction == Left || (previousDirection == Left && (direction == Up || direction == Down)) {
			return indexes.RidingHorseLeft
		}

		return indexes.RidingHorseRight
	case FrigateVehicle:
		switch direction {
		case Left:
			return indexes.ShipNoSailsLeft
		case Right:
			return indexes.ShipNoSailsRight
		case Up:
			return indexes.ShipNoSailsUp
		case Down:
			return indexes.ShipNoSailsDown
		case NoneDirection:
			return indexes.ShipNoSailsRight
		default:
			return indexes.ShipNoSailsRight
		}
	case SkiffVehicle:
		switch direction {
		case Left:
			return indexes.SkiffLeft
		case Right:
			return indexes.SkiffRight
		case Up:
			return indexes.SkiffUp
		case Down:
			return indexes.SkiffDown
		case NoneDirection:
			return indexes.SkiffLeft
		default:
			return indexes.SkiffLeft
		}
	case NoPartyVehicle, NPC:
	}

	return indexes.AvatarKeyIndex
}

// GetMovementPrefix returns the movement verb printed before a direction, such as "Ride ".
func (v VehicleType) GetMovementPrefix() string {
	switch v {
	case CarpetVehicle:
		return "Fly "
	case HorseVehicle:
		return "Ride "
	case SkiffVehicle:
		return "Row "
	case FrigateVehicle:
		return "Row "
	case NoPartyVehicle:
		return ""
	case NPC:
	}

	return ""
}

// GetLowerCaseName returns the lower case vehicle name.
func (v VehicleType) GetLowerCaseName() string {
	switch v {
	case NoPartyVehicle:
		return "none"
	case FrigateVehicle:
		return "frigate"
	case CarpetVehicle:
		return "carpet"
	case SkiffVehicle:
		return "skiff"
	case HorseVehicle:
		return "horse"
	case NPC:
	}

	panic(fmt.Sprintf("bad vehicle type %d", v))
}
