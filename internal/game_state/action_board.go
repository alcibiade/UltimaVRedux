package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ActionBoard - handles boarding vehicles
// Based on Commands.md Board section
func (g *GameState) ActionBoard() bool {
	g.SystemCallbacks.Message.ShowCommandPrompt("Board ")

	// Not allowed in dungeons
	if g.MapState.PlayerLocation.Location.GetMapType() == references.DungeonMapType {
		g.SystemCallbacks.Message.AddRowStr("Not here!")

		return false
	}

	// Look for vehicle at current position
	vehicle := g.CurrentNPCAIController.GetNpcs().GetVehicleAtPositionAndFloorOrNil(
		g.MapState.PlayerLocation.Position,
		g.MapState.PlayerLocation.Floor,
	)

	if vehicle == nil {
		g.SystemCallbacks.Message.AppendToCurrentRowStr("What?")

		return false
	}
	vehicleType := vehicle.GetVehicleDetails().VehicleType

	// Vehicle boarding checks the horse's active NPC talk link before on-foot state.
	// See ref:VEH-0002.
	if vehicleType == references.HorseVehicle && vehicle.GetVehicleDetails().IsOwnedByNPC {
		g.SystemCallbacks.Message.AppendToCurrentRowStr("\"Nay!\"")
		g.AdvanceOrdinarySurfaceCommandMinutes(1)

		return true
	}

	// A skiff or carpet at the ship's position is carried aboard instead of
	// requiring the party to disembark first.
	partyVehicleType := g.PartyVehicle.GetVehicleDetails().VehicleType
	frigateBelow := vehicle.GetVehicleDetails().VehicleType == references.FrigateVehicle
	canTransferAboard := frigateBelow &&
		(partyVehicleType == references.SkiffVehicle || partyVehicleType == references.CarpetVehicle)

	if !canTransferAboard {
		// Check if already on a vehicle (must be on foot to board)
		if partyVehicleType != references.NoPartyVehicle {
			g.SystemCallbacks.Message.AddRowStr("On foot")

			return false
		}
	}

	// Print boarding message based on vehicle type
	switch vehicleType {
	case references.HorseVehicle:
		g.SystemCallbacks.Message.AppendToCurrentRowStr("horse")

	case references.CarpetVehicle:
		g.SystemCallbacks.Message.AppendToCurrentRowStr("carpet")

	case references.SkiffVehicle:
		g.SystemCallbacks.Message.AppendToCurrentRowStr("skiff")

	case references.FrigateVehicle:
		g.SystemCallbacks.Message.AppendToCurrentRowStr("Ship")

	default:
		g.SystemCallbacks.Message.AppendToCurrentRowStr("What?")

		return false
	}

	// Attempt to board the vehicle
	result := g.BoardVehicle(*vehicle)

	switch result {
	case BoardVehicleResultSuccess:
		// Success message already printed above
		// REDUX DEVIATION: vehicle boarding has no direct boarding sound.
		// Redux exposes the successful transition as an authorable cue.
		// See ref:VEH-0002.
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundVehicleBoard)
		g.AdvanceOrdinarySurfaceCommandMinutes(1)

		return true

	case BoardVehicleResultNoVehicle:
		g.SystemCallbacks.Message.AppendToCurrentRowStr("What?")

		return false

	case BoardVehicleResultNoSkiffs:
		// Ship boarded but warning about no skiffs
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundVehicleBoard)
		g.SystemCallbacks.Message.AddRowStr("WARNING: NO SKIFFS ON BOARD!")
		g.AdvanceOrdinarySurfaceCommandMinutes(1)

		return true

	case BoardVehicleResultSkiffCapacity:
		g.SystemCallbacks.Message.AddRowStr("No room for another skiff.")

		return false
	}

	return false
}
