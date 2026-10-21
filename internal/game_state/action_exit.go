package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// hasNearbyVehicleExit matches nearby landing check/test for the party's current floor.
// See ref:VEH-0003.
func (g *GameState) hasNearbyVehicleExit() bool {
	currentPosition := g.MapState.PlayerLocation.Position
	largeMap := g.MapState.PlayerLocation.Location.GetMapType() == references.LargeMapType
	directions := [...]references.Direction{
		references.Up,
		references.Down,
		references.Right,
		references.Left,
	}

	for _, direction := range directions {
		candidate := direction.GetNewPositionInDirection(&currentPosition)
		if largeMap {
			candidate = candidate.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
		} else if g.IsOutOfBounds(*candidate) {
			continue
		}

		if g.ObjectPresentAt(candidate) {
			mapUnit := g.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(
				*candidate,
				g.MapState.PlayerLocation.Floor,
			)
			if mapUnit != nil && isOriginalNearbyExitObject(*mapUnit) {
				return true
			}

			continue
		}

		if g.isEffectiveWalkingPassable(*candidate) {
			return true
		}
	}

	return false
}

func isOriginalNearbyExitObject(mapUnit map_units.MapUnit) bool {
	if friendly, ok := mapUnit.(*map_units.NPCFriendly); ok &&
		friendly.NPCReference.GetNPCType() == references.Vehicle {
		details := friendly.GetVehicleDetails()
		switch details.VehicleType {
		case references.CarpetVehicle, references.HorseVehicle, references.SkiffVehicle:
			return true
		case references.FrigateVehicle:
			return details.IsSailFurled()
		case references.NoPartyVehicle, references.NPC:
			return false
		}
	}

	sprite := mapUnit.GetKeySpriteIndex()

	return sprite&^3 == indexes.BasicAvatar&^3
}

func (g *GameState) isEffectiveWalkingPassable(position references.Position) bool {
	tile := references.Tile{Index: g.GetEffectiveTile(&position)}

	return tile.IsPassable(references.NoPartyVehicle)
}

func (g *GameState) isSkiffExitBridgeUnderlay() bool {
	underlay := g.GetEffectiveTile(&g.MapState.PlayerLocation.Position)

	return underlay == indexes.TrollBridgeHoriz || underlay == indexes.TrollBridgeVert
}

type exitVehicleResult uint8

const (
	exitVehicleSuccess exitVehicleResult = iota
	exitVehicleNoVehicle
	exitVehicleUnderSail
	exitVehicleNoLand
	exitVehicleBridgeUnderlay
	exitVehicleNoSkiffs
	exitVehicleNoMapSlot
)

func (g *GameState) exitVehicle() (*map_units.NPCFriendly, exitVehicleResult) {
	exitingVehicleType := g.PartyVehicle.GetVehicleDetails().VehicleType
	if exitingVehicleType < references.CarpetVehicle || exitingVehicleType > references.FrigateVehicle {
		return nil, exitVehicleNoVehicle
	}
	if exitingVehicleType == references.FrigateVehicle &&
		!g.PartyVehicle.GetVehicleDetails().IsSailFurled() {
		return nil, exitVehicleUnderSail
	}

	nearbyExit := g.hasNearbyVehicleExit()
	nextPartyVehicle := map_units.NewNPCFriendlyVehiceNoVehicle()
	abandonedVehicle := g.PartyVehicle
	consumeCarpet := false

	switch exitingVehicleType {
	case references.CarpetVehicle:
		if !nearbyExit && !g.isEffectiveWalkingPassable(g.MapState.PlayerLocation.Position) {
			return nil, exitVehicleNoLand
		}
	case references.HorseVehicle:
	case references.SkiffVehicle:
		if !nearbyExit {
			return nil, exitVehicleNoLand
		}
		if g.isSkiffExitBridgeUnderlay() {
			return nil, exitVehicleBridgeUnderlay
		}
	case references.FrigateVehicle:
		if nearbyExit {
			break
		}
		switch {
		case abandonedVehicle.GetVehicleDetails().HasAtLeastOneSkiff():
			abandonedVehicle.GetVehicleDetails().DecrementSkiffQuantity()
			nextPartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
				references.SkiffVehicle,
				g.MapState.PlayerLocation.Position,
				g.MapState.PlayerLocation.Floor,
			)
		case g.PartyState.Inventory.QuestItems.HasSome(references.Carpet):
			consumeCarpet = true
			nextPartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
				references.CarpetVehicle,
				g.MapState.PlayerLocation.Position,
				g.MapState.PlayerLocation.Floor,
			)
		default:
			return nil, exitVehicleNoSkiffs
		}
	case references.NoPartyVehicle, references.NPC:
		return nil, exitVehicleNoVehicle
	}

	units := g.CurrentNPCAIController.GetNpcs()
	if !units.HasCapacity() {
		return nil, exitVehicleNoMapSlot
	}

	vehicleState := abandonedVehicle.GetVehicleDetails().Snapshot()
	abandonedVehicle.SetPos(g.MapState.PlayerLocation.Position)
	abandonedVehicle.NPCReference.Schedule.OverrideAllPositions(
		byte(g.MapState.PlayerLocation.Position.X),
		byte(g.MapState.PlayerLocation.Position.Y),
	)
	abandonedVehicle.SetFloor(g.MapState.PlayerLocation.Floor)
	stored := units.InsertVehicle(abandonedVehicle)
	if stored == nil {
		panic("preflighted vehicle exit placement failed")
	}
	stored.GetVehicleDetails().Restore(vehicleState)
	g.registerAtlasVessel(stored.GetVehicleDetails())

	if consumeCarpet && !g.PartyState.Inventory.QuestItems.DecrementByOne(references.Carpet) {
		panic("preflighted frigate carpet exit could not consume carpet")
	}
	g.PartyVehicle = nextPartyVehicle
	g.registerAtlasVessel(g.PartyVehicle.GetVehicleDetails())

	return stored, exitVehicleSuccess
}

// ActionExit handles exiting or dismounting from vehicles.
func (g *GameState) ActionExit() bool {
	g.SystemCallbacks.Message.ShowCommandPrompt("X-it ")

	exitedVehicle, result := g.exitVehicle()
	if result != exitVehicleSuccess {
		switch result {
		case exitVehicleUnderSail:
			g.SystemCallbacks.Message.AddRowStr("Under sail!")
		case exitVehicleNoLand:
			g.SystemCallbacks.Message.AddRowStr("No land nearby!")
		case exitVehicleBridgeUnderlay, exitVehicleNoMapSlot:
			g.SystemCallbacks.Message.AddRowStr("Not here!")
		case exitVehicleNoSkiffs:
			g.SystemCallbacks.Message.AddRowStr("No skiffs on board!")
		case exitVehicleNoVehicle:
			g.SystemCallbacks.Message.AppendToCurrentRowStr("what?")
		case exitVehicleSuccess:
			panic("successful vehicle exit returned no vehicle")
		}

		return false
	}

	switch exitedVehicle.GetVehicleDetails().VehicleType {
	case references.HorseVehicle:
		g.SystemCallbacks.Message.AppendToCurrentRowStr("horse!")
	case references.CarpetVehicle:
		g.SystemCallbacks.Message.AppendToCurrentRowStr("carpet!")
	case references.SkiffVehicle:
		g.SystemCallbacks.Message.AppendToCurrentRowStr("skiff!")
	case references.FrigateVehicle:
		g.SystemCallbacks.Message.AppendToCurrentRowStr("ship!")
	case references.NoPartyVehicle, references.NPC:
		panic("vehicle exit stored an invalid vehicle type")
	}
	// REDUX DEVIATION: vehicle exit has no direct disembark sound.
	// Redux exposes the successful vehicle exit as a separate authorable moment.
	// See ref:VEH-0003.
	g.SystemCallbacks.Audio.PlaySoundEffect(SoundVehicleExit)
	g.AdvanceOrdinarySurfaceCommandMinutes(1)

	return true
}
