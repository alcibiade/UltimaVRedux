package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ExecuteJailSentence relocates surrender to Yew at (25,4), advances
// in 20-minute requests until hour 8, then installs floor zero and
// confiscates ordinary Keys. Minute offset is not reset. ref:CRIME-0003
func (gs *GameState) ExecuteJailSentence() int {
	if gs.MapState.PlayerLocation.Location.GetMapType() != references.SmallMapType {
		panic("jail sentence requires a small-map source")
	}
	if gs.GameReferences.LocationReferences == nil ||
		gs.GameReferences.LocationReferences.WorldLocations == nil {
		panic("jail sentence requires location references")
	}

	jailLocation := references.Yew
	jailPosition := references.Position{X: references.JailPositionX, Y: references.JailPositionY}
	jailFloor := references.JailFloorNumber
	worldPosition, worldFloor := gs.GameReferences.LocationReferences.WorldLocations.GetWorldLocation(jailLocation)
	sourceLocation := gs.MapState.SavePlayerLocation()

	// Original surrender relocates the party before advancing sentence time.
	// Keep the current terrain loaded until release while exposing Yew to
	// midnight systems. ref:CRIME-0003
	gs.MapState.PlayerLocation.Location = jailLocation
	gs.MapState.PlayerLocation.Position = jailPosition

	elapsedMinutes := 0
	for gs.DateTime.Hour != references.JailReleaseHour {
		var elapsed int
		if gs.HasActiveDurationEffect(DurationEffectTimeStop) {
			// REDUX DEVIATION: surrender handling repeatedly calls calendar advancement,
			// which never advances under An Tym and therefore hangs. Redux bypasses
			// the freeze for sentence time while preserving the active spell.
			// See ref:CRIME-0003.
			elapsed = gs.advanceElapsedGameplayMinutes(references.JailSentenceStepMinutes, false)
		} else {
			elapsed = gs.advanceGameplayMinutes(references.JailSentenceStepMinutes, false)
		}
		if elapsed <= 0 {
			panic("jail sentence time failed to advance")
		}
		elapsedMinutes += elapsed
	}

	gs.MapState.RestorePlayerLocation(sourceLocation)
	gs.LastLargeMapPosition = worldPosition
	gs.LastLargeMapFloor = worldFloor
	gs.PartyState.Inventory.Provisions.Keys.Set(0)
	gs.TransitionToLocation(jailLocation, jailFloor, jailPosition, TransitionFullLocation, false)
	gs.ClearCriminalStatus()

	logging.Info(logging.CategoryGameState,
		"Jailed: Location=%v, Pos=(%d,%d), Floor=%d, Release=%d:%02d, Elapsed=%d, Keys confiscated",
		jailLocation, jailPosition.X, jailPosition.Y, jailFloor,
		gs.DateTime.Hour, gs.DateTime.Minute, elapsedMinutes)

	return elapsedMinutes
}

// RefuseArrest activates the hostile-Guard encounter route and queues
// combat rather than the surrender sentence. ref:CRIME-0003
func (gs *GameState) RefuseArrest(source *map_units.NPCFriendly, completeSourceTurn bool) bool {
	// Show combat taunt message
	gs.SystemCallbacks.Message.AddRowStr("")
	gs.SystemCallbacks.Message.AddRowStr("\"Then defend thyself, rogue!\"")
	gs.SystemCallbacks.Message.AddRowStr("")

	if err := gs.QueueSmallMapCombatEncounter(source, SmallMapEncounterArrestRefusal); err != nil {
		logging.Error(logging.CategoryGameState, "Unable to queue arrest combat: %v", err)
		gs.SystemCallbacks.Message.AddRowStr("Combat could not begin.")

		return false
	}
	gs.pendingSmallMapEncounter.Consequences.CompleteSourceTurn = completeSourceTurn

	logging.Info(logging.CategoryGameState, "Player refused arrest - Guard combat queued")

	return true
}
