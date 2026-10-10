package references

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type VehicleType int

const (
	NoPartyVehicle VehicleType = iota
	CarpetVehicle
	HorseVehicle
	SkiffVehicle
	FrigateVehicle
	NPC
)

func (v VehicleType) GetUnBoardedSpriteByDirection(direction Direction) indexes.SpriteIndex {
	switch v {
	case CarpetVehicle:
		return indexes.Carpet2
	case HorseVehicle:
		if direction == Left || direction == Up {
			return indexes.HorseLeft
		}

		return indexes.HorseRight
	}

	return v.GetBoardedSpriteByDirection(direction, direction)
}

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
		default:
			return indexes.SkiffLeft
		}
	default:
		return indexes.AvatarKeyIndex
	}
}

func (v VehicleType) RequiresNewSprite(currentDirection, newDirection Direction) bool {
	switch v {
	case CarpetVehicle, HorseVehicle:
		if currentDirection == Up || currentDirection == Down || currentDirection == newDirection {
			return false
		}

		return true
	case SkiffVehicle:
		return currentDirection != newDirection
	case FrigateVehicle:
		return currentDirection != newDirection
	case NoPartyVehicle:
		return false
	}
	// Programming bug: all VehicleType values should be handled above
	panic("Unexpected: must have valid vehicle")
}

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
	default:
		return ""
	}
}

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
	default:
		panic(fmt.Sprintf("bad vehicle type %d", v))
	}
}

func (v VehicleType) GetExitString() string {
	if v == NoPartyVehicle {
		return "X-it what?"
	}

	return fmt.Sprintf("Xit- %s!", v.GetLowerCaseName())
}

func (v VehicleType) GetBoardString() string {
	if v == NoPartyVehicle {
		return "X-it what?"
	}

	return fmt.Sprintf("Board %s!", v.GetLowerCaseName())
}
